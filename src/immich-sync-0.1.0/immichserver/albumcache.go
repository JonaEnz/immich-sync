package immichserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/JonaEnz/immich-sync/oapi"
	"github.com/google/uuid"
)

type ImmichAlbumCache struct {
	cache map[string]*oapi.AlbumResponseDto
}

func NewImmichAlbumCache() ImmichAlbumCache {
	return ImmichAlbumCache{
		cache: make(map[string]*oapi.AlbumResponseDto),
	}
}

func (a *ImmichAlbumCache) FillCache(server *ImmichServer) error {
	assetUUID := oapi.OptUUID{}
	assetUUID.Reset()
	result, err := server.oapiClient.GetAllAlbums(context.Background(), oapi.GetAllAlbumsParams{AssetId: assetUUID})
	if err != nil {
		return err
	}
	for _, album := range result {
		a.cache[album.ID] = &album
	}
	return nil
}

func (a *ImmichAlbumCache) GetAlbumUUIDByName(server *ImmichServer, name string) (uuid.UUID, error) {
	for _, a := range a.cache {
		if a.AlbumName == name {
			u, err := uuid.Parse(a.ID)
			if err != nil {
				return uuid.UUID{}, err
			}
			return u, nil
		}
	}
	a.FillCache(server)
	for _, a := range a.cache {
		if a.AlbumName == name {
			u, err := uuid.Parse(a.ID)
			if err != nil {
				return uuid.UUID{}, err
			}
			return u, nil
		}
	}
	return uuid.UUID{}, fmt.Errorf("an album with name %s does not exist.", name)
}

func (a *ImmichAlbumCache) updateAlbum(server *ImmichServer, albumUUID uuid.UUID) error {
	resp, err := server.oapiClient.GetAlbumInfo(context.Background(), oapi.GetAlbumInfoParams{
		ID:            albumUUID,
		WithoutAssets: oapi.NewOptBool(false),
	})
	if err != nil {
		return err
	}
	a.cache[resp.ID] = resp
	return nil
}

func (a *ImmichAlbumCache) Album(server *ImmichServer, albumUUID uuid.UUID) (*oapi.AlbumResponseDto, error) {
	tryGet := func(server *ImmichServer, albumUUID uuid.UUID) (*oapi.AlbumResponseDto, error) {
		if album, ok := a.cache[albumUUID.String()]; ok {
			if album.AssetCount != len(album.Assets) {
				a.updateAlbum(server, albumUUID)
				album = a.cache[albumUUID.String()]
			}
			return album, nil
		}
		return nil, errors.New("album not found")
	}
	album, err := tryGet(server, albumUUID)
	if err == nil {
		return album, nil
	}
	a.FillCache(server)
	return tryGet(server, albumUUID)
}
