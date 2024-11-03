package main

import (
	"context"
	"crypto/sha1"
	"flag"
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

var apiKey string

func main() {
	serverURL := flag.String("url", "http://192.168.0.136:2283/api", "Immich server url with trailing /api")
	apiKey = *flag.String("api-key", "y2gDkeRqPpiTcM0CpQpTc58hxTutkltzBOHLYYw70", "api key")
	client, _ := oapi.NewClient(*serverURL, &APIKeySecuritySource{})

	albums, err := client.GetAllAlbums(context.Background(), oapi.GetAllAlbumsParams{})
	if err != nil {
		fmt.Println(err)
	}
	for _, a := range albums {
		fmt.Println(a.AlbumName)
	}
	err = upload(client, "/home/jona/Pictures/plasma6-sun.jpeg")
	if err != nil {
		fmt.Println(err)
	}
}

func upload(c *oapi.Client, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	h := sha1.New()
	if _, err = io.Copy(h, file); err != nil {
		return err
	}
	assetSha1 := fmt.Sprintf("%x", h.Sum(nil))

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
	_, err = c.UploadAsset(context.Background(), &oapi.AssetMediaCreateDtoMultipart{
		AssetData: http.MultipartFile{
			Name:   file.Name(),
			File:   file,
			Size:   fileInfo.Size(),
			Header: mimetype,
		},
		DeviceAssetId:  assetUUID,
		DeviceId:       "apitest",
		FileCreatedAt:  time.Now(),
		FileModifiedAt: time.Now(),
	},
		oapi.UploadAssetParams{
			XImmichChecksum: oapi.NewOptString(assetSha1),
		})
	if err != nil {
		return err
	}
	return nil
}

type APIKeySecuritySource struct{}

func (a *APIKeySecuritySource) APIKey(ctx context.Context, operationName string) (oapi.APIKey, error) {
	return oapi.APIKey{
		APIKey: apiKey,
	}, nil
}

// Bearer provides bearer security value.
func (a *APIKeySecuritySource) Bearer(ctx context.Context, operationName string) (oapi.Bearer, error) {
	return oapi.Bearer{}, ogenerrors.ErrSkipClientSecurity
}

// Cookie provides cookie security value.
func (a *APIKeySecuritySource) Cookie(ctx context.Context, operationName string) (oapi.Cookie, error) {
	return oapi.Cookie{}, ogenerrors.ErrSkipClientSecurity
}
