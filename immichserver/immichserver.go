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
	apiURL       string
	apiKey       string
	deviceID     string
	oapiClient   *oapi.Client
	ImageDirs    []*ImageDirectory
	albumCache   ImmichAlbumCache
	versionCache ImmichServerVersion
}

type ImmichServerVersion struct {
	major int
	minor int
	patch int
}

func NewImmichServer(apiKey, serverURL, deviceID string) *ImmichServer {
	client, _ := oapi.NewClient(serverURL, &ImmichServerSecuritySource{
		key: oapi.APIKey{
			APIKey: apiKey,
		},
	})

	server := ImmichServer{
		apiURL:       serverURL,
		apiKey:       apiKey,
		deviceID:     deviceID,
		oapiClient:   client,
		albumCache:   NewImmichAlbumCache(),
		versionCache: ImmichServerVersion{},
	}
	return &server
}

func (i *ImmichServer) Version() (ImmichServerVersion, error) {
	response, err := i.oapiClient.GetServerVersion(context.Background())
	if err != nil {
		return ImmichServerVersion{}, err
	}
	i.versionCache = ImmichServerVersion{response.Major, response.Minor, response.Patch}
	return i.versionCache, nil
}

func (i *ImmichServer) MinVersionCheck(min ImmichServerVersion) error {
	version, err := i.Version()
	if err != nil {
		return fmt.Errorf("version error: server version could not be determined: %w", err)
	}
	if !version.IsMinimumVersion(min) {
		return fmt.Errorf("version error: server version for metadata copy required to be '%v', got '%v'", min, version)
	}
	return nil
}

func (v *ImmichServerVersion) IsMinimumVersion(min ImmichServerVersion) bool {
	if v.major > min.major {
		return true
	}
	if v.minor > min.minor {
		return true
	}
	return v.minor == min.minor && v.patch >= min.patch
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

func (i *ImmichServer) AlbumAssets(albumUUID uuid.UUID) ([]oapi.AssetResponseDto, error) {
	assets := make([]oapi.AssetResponseDto, 0)
	page := 0
	const size = 1000
	for {
		resp, err := i.oapiClient.SearchAssets(context.Background(), &oapi.MetadataSearchDto{
			AlbumIds: []uuid.UUID{albumUUID},
			Page:     oapi.NewOptInt(page),
			Size:     oapi.NewOptInt(size),
		}, oapi.SearchAssetsParams{})
		if err != nil {
			return nil, err
		}
		assets = append(assets, resp.Assets.Items...)
		if len(resp.Assets.Items) < size {
			break
		}
		page++
	}
	return assets, nil
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
		Description: oapi.OptNilString{},
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	i.albumCache.cache[name] = response
	return response.ID, nil
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
	return resp.ID, nil
}

func (i *ImmichServer) GetImageUUIDByPath(path string) (uuid.UUID, error) {
	for j := range i.ImageDirs {
		if cache, ok := i.ImageDirs[j].contentCache[path]; ok {
			return cache.uuid, nil
		}
	}

	return uuid.UUID{}, errors.New("path is not in the watched directories")
}

func (i *ImmichServer) Upload(path string, assetSha1 *string) (uuid.UUID, error) {
	var r io.Reader
	file, err := os.Open(path)
	if err != nil {
		return uuid.UUID{}, err
	}
	r = file

	if assetSha1 == nil {
		h := sha1.New()
		var buf bytes.Buffer
		tee := io.TeeReader(file, &buf)
		if _, err = io.Copy(h, tee); err != nil {
			return uuid.UUID{}, err
		}
		r = &buf
		sha1String := fmt.Sprintf("%x", h.Sum(nil))
		assetSha1 = &sha1String
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return uuid.UUID{}, err
	}
	mimename, _, err := mime.ParseMediaType(filepath.Ext(path))
	if err != nil {
		return uuid.UUID{}, err
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
		FileCreatedAt:  time.Now(),
		FileModifiedAt: time.Now(),
	},
		oapi.UploadAssetParams{
			XImmichChecksum: oapi.NewOptString(*assetSha1),
		})
	if err != nil {
		return uuid.UUID{}, err
	}
	if r, ok := response.(*oapi.UploadAssetCreated); ok {
		return r.ID, nil
	}
	if r, ok := response.(*oapi.UploadAssetOK); ok {
		return r.ID, nil
	}
	return uuid.UUID{}, errors.New("unexpected upload response")
}

func (i *ImmichServer) Delete(assetUUID uuid.UUID) error {
	return i.oapiClient.DeleteAssets(context.Background(), &oapi.AssetBulkDeleteDto{
		Force: oapi.NewOptBool(false),
		Ids:   []uuid.UUID{assetUUID},
	})
}

func (i *ImmichServer) CopyMetadata(oldAsset uuid.UUID, newAsset uuid.UUID) error {
	if err := i.MinVersionCheck(ImmichServerVersion{2, 2, 0}); err != nil {
		return err
	}
	return i.oapiClient.CopyAsset(context.Background(), &oapi.AssetCopyDto{
		Albums:      oapi.NewOptBool(true),
		Favorite:    oapi.NewOptBool(true),
		SharedLinks: oapi.NewOptBool(true),
		Sidecar:     oapi.NewOptBool(true),
		SourceId:    oldAsset,
		Stack:       oapi.NewOptBool(true),
		TargetId:    newAsset,
	})
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
