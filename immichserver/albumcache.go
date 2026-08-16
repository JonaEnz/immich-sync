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
		a.cache[album.ID.String()] = &album
	}
	return nil
}

func (a *ImmichAlbumCache) GetAlbumUUIDByName(server *ImmichServer, name string) (uuid.UUID, error) {
	for _, a := range a.cache {
		if a.AlbumName == name {
			return a.ID, nil
		}
	}
	a.FillCache(server)
	for _, a := range a.cache {
		if a.AlbumName == name {
			return a.ID, nil
		}
	}
	return uuid.UUID{}, fmt.Errorf("an album with name %s does not exist.", name)
}

func (a *ImmichAlbumCache) Album(server *ImmichServer, albumUUID uuid.UUID) (*oapi.AlbumResponseDto, error) {
	tryGet := func(server *ImmichServer, albumUUID uuid.UUID) (*oapi.AlbumResponseDto, error) {
		if album, ok := a.cache[albumUUID.String()]; ok {
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
