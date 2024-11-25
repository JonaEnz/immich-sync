package immichserver

import (
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

func (i *ImmichServer) Upload(path string, assetSha1 *string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	if assetSha1 == nil {
		h := sha1.New()
		if _, err = io.Copy(h, file); err != nil {
			return err
		}
		sha1String := fmt.Sprintf("%x", h.Sum(nil))
		assetSha1 = &sha1String
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	mimename, _, err := mime.ParseMediaType(filepath.Ext(path))
	if err != nil {
		return err
	}
	mimetype := textproto.MIMEHeader{}
	mimetype.Set("Content-Type", mimename)
	assetUUID := uuid.New().String()
	_, err = i.oapiClient.UploadAsset(context.Background(), &oapi.AssetMediaCreateDtoMultipart{
		AssetData: http.MultipartFile{
			Name:   file.Name(),
			File:   file,
			Size:   fileInfo.Size(),
			Header: mimetype,
		},
		DeviceAssetId:  assetUUID,
		DeviceId:       i.deviceID,
		FileCreatedAt:  time.Now(),
		FileModifiedAt: time.Now(),
	},
		oapi.UploadAssetParams{
			XImmichChecksum: oapi.NewOptString(*assetSha1),
		})
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
