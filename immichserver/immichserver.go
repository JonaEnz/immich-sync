package immichserver

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"os"
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
	}
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
	return response.Response.ID, nil
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
