package immichserver

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/JonaEnz/immich-sync/oapi"
	"github.com/google/uuid"
	"github.com/ogen-go/ogen/http"
	"github.com/ogen-go/ogen/ogenerrors"
)

type ImmichServer struct {
	apiURL     string
	apiKey     string
	deviceID   string
	oapiClient *oapi.Client
	ImageDirs  []*ImageDirectory
	albumCache ImmichAlbumCache
}

func NewImmichServer(apiKey, serverURL, deviceID string) *ImmichServer {
	client, _ := oapi.NewClient(serverURL, &ImmichServerSecuritySource{
		key: oapi.APIKey{
			APIKey: apiKey,
		},
	})

	return &ImmichServer{
		apiURL:     serverURL,
		apiKey:     apiKey,
		deviceID:   deviceID,
		oapiClient: client,
		albumCache: NewImmichAlbumCache(),
	}
}

func (i *ImmichServer) GetAlbumByUUIDOrName(uuidOrName string) (uuid.UUID, error) {
	u, err := uuid.Parse(uuidOrName)
	if err == nil {
		return u, nil
	}
	return i.albumCache.GetAlbumUUIDByName(i, uuidOrName)
}

func (i *ImmichServer) Album(albumUUID uuid.UUID) (*oapi.AlbumResponseDto, error) {
	return i.albumCache.Album(i, albumUUID)
}

func (i *ImmichServer) CreateNewAlbum(name string) (uuid.UUID, error) {
	i.albumCache.FillCache(i)

	for _, album := range i.albumCache.cache {
		if album.AlbumName == name {
			return uuid.UUID{}, errors.New("an album with this name already exists")
		}
	}

	response, err := i.oapiClient.CreateAlbum(context.Background(), &oapi.CreateAlbumDto{
		AlbumName:   name,
		AlbumUsers:  make([]oapi.AlbumUserCreateDto, 0),
		AssetIds:    make([]uuid.UUID, 0),
		Description: oapi.OptString{},
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	albumUUID, err := uuid.Parse(response.ID)
	if err != nil {
		return uuid.UUID{}, err
	}
	i.albumCache.cache[name] = response
	return albumUUID, nil
}

func (i *ImmichServer) AddToAlbum(imageUUIDs []uuid.UUID, albumUUID uuid.UUID) error {
	response, err := i.oapiClient.AddAssetsToAlbum(context.Background(), &oapi.BulkIdsDto{Ids: imageUUIDs}, oapi.AddAssetsToAlbumParams{
		ID: albumUUID,
	})
	if err != nil {
		return err
	}
	for _, r := range response {
		if !r.Success {
			return fmt.Errorf("Image '%s' failed with error '%s'", r.ID, r.Error.Value)
		}
	}
	return nil
}

func (i *ImmichServer) GetUserUUID() (uuid.UUID, error) {
	resp, err := i.oapiClient.GetMyUser(context.Background())
	if err != nil {
		return uuid.UUID{}, err
	}
	return uuid.Parse(resp.ID)
}

func (i *ImmichServer) GetSyncAfter(t time.Time) (*oapi.AssetDeltaSyncResponseDto, error) {
	userUUID, err := i.GetUserUUID()
	if err != nil {
		return nil, err
	}
	return i.oapiClient.GetDeltaSync(context.Background(), &oapi.AssetDeltaSyncDto{
		UpdatedAfter: t,
		UserIds:      []uuid.UUID{userUUID},
	})
}

func (i *ImmichServer) DoFullSync(t time.Time) (*[]oapi.AssetResponseDto, error) {
	userUUID, err := i.GetUserUUID()
	if err != nil {
		return nil, err
	}

	ouuid := oapi.OptUUID{}
	ouuid.Reset()
	assets := make([]oapi.AssetResponseDto, 0)

	getMore := true
	for getMore {
		newAssets, err := i.oapiClient.GetFullSyncForUser(context.Background(), &oapi.AssetFullSyncDto{
			LastId:       ouuid,
			Limit:        100,
			UpdatedUntil: t,
			UserId:       oapi.NewOptUUID(userUUID),
		})
		if err != nil {
			return nil, err
		}
		assets = append(assets, newAssets...)
		getMore = len(newAssets) < 100
		if len(newAssets) == 0 {
			break
		}
		u, _ := uuid.Parse(newAssets[len(newAssets)-1].ID)
		ouuid.SetTo(u)
	}
	return &assets, nil
}

func (i *ImmichServer) GetImageUUIDByPath(path string) (uuid.UUID, error) {
	for j := range i.ImageDirs {
		if cache, ok := i.ImageDirs[j].contentCache[path]; ok {
			return cache.uuid, nil
		}
	}

	return uuid.UUID{}, errors.New("path is not in the watched directories")
}

func (i *ImmichServer) Upload(path string, assetSha1 *string) (string, error) {
	var r io.Reader
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	r = file

	if assetSha1 == nil {
		h := sha1.New()
		var buf bytes.Buffer
		tee := io.TeeReader(file, &buf)
		if _, err = io.Copy(h, tee); err != nil {
			return "", err
		}
		r = &buf
		sha1String := fmt.Sprintf("%x", h.Sum(nil))
		assetSha1 = &sha1String
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}
	mimename, _, err := mime.ParseMediaType(filepath.Ext(path))
	if err != nil {
		return "", err
	}
	mimetype := textproto.MIMEHeader{}
	mimetype.Set("Content-Type", mimename)
	response, err := i.oapiClient.UploadAsset(context.Background(), &oapi.AssetMediaCreateDtoMultipart{
		AssetData: http.MultipartFile{
			Name:   file.Name(),
			File:   r,
			Size:   fileInfo.Size(),
			Header: mimetype,
		},
		DeviceAssetId:  i.deviceID + *assetSha1,
		DeviceId:       i.deviceID,
		FileCreatedAt:  time.Now(),
		FileModifiedAt: time.Now(),
	},
		oapi.UploadAssetParams{
			XImmichChecksum: oapi.NewOptString(*assetSha1),
		})
	if err != nil {
		return "", err
	}
	if r, ok := response.(*oapi.UploadAssetCreated); ok {
		return r.ID, nil
	}
	if r, ok := response.(*oapi.UploadAssetOK); ok {
		return r.ID, nil
	}

	return "", err
}

func (i *ImmichServer) Download(filePath string, imageUUID uuid.UUID) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	response, err := i.oapiClient.DownloadAsset(context.Background(), oapi.DownloadAssetParams{ID: imageUUID})
	if err != nil {
		return err
	}
	if stat.IsDir() {
		filePath = path.Join(filePath, imageUUID.String())
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, response.Data)
	if err != nil {
		return err
	}
	return nil
}

type ImmichServerSecuritySource struct {
	key oapi.APIKey
}

func (s *ImmichServerSecuritySource) APIKey(ctx context.Context, operationName string) (oapi.APIKey, error) {
	return s.key, nil
}

// Bearer provides bearer security value.
func (s *ImmichServerSecuritySource) Bearer(ctx context.Context, operationName string) (oapi.Bearer, error) {
	return oapi.Bearer{}, ogenerrors.ErrSkipClientSecurity
}

// Cookie provides cookie security value.
func (s *ImmichServerSecuritySource) Cookie(ctx context.Context, operationName string) (oapi.Cookie, error) {
	return oapi.Cookie{}, ogenerrors.ErrSkipClientSecurity
}
