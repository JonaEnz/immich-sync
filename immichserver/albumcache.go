package immichserver

import (
	"context"
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
	result, err := server.oapiClient.GetAllAlbums(context.Background(), oapi.GetAllAlbumsParams{})
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
