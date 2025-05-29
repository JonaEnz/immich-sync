package immichserver

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ImageDirectory struct {
	path         string
	subdir       bool
	album        *uuid.UUID
	contentCache map[string]FileStat
	lastScan     time.Time
}

type ImageDirectoryConfig struct {
	Path  string `json:"path"`
	Album string `json:"album"`
}

type FileStat struct {
	info     os.FileInfo
	hashSha1 []byte
	uuid     uuid.UUID
}

func (f *FileStat) HashHexString() string {
	return fmt.Sprintf("%x", f.hashSha1)
}

func NewImageDirectory(path string, subdir bool) ImageDirectory {
	return ImageDirectory{
		path:         path,
		album:        nil,
		subdir:       subdir,
		contentCache: make(map[string]FileStat),
		lastScan:     time.Time{},
	}
}

func (i *ImageDirectory) Path() string {
	return i.path
}

func (i *ImageDirectory) AlbumUUID() string {
	if i.album == nil {
		return ""
	}
	return (*i.album).String()
}

func (i *ImageDirectory) SetAlbum(albumUUID *uuid.UUID) {
	i.album = albumUUID
}

func (i *ImageDirectory) Count() int {
	return len(i.contentCache)
}

func (i *ImageDirectory) String() string {
	return fmt.Sprintf("%s: %d images, last scanned %s", i.path, i.Count(), i.lastScan.Format("Mon Jan 2 15:04:05 MST 2006"))
}

func (i *ImageDirectory) Read() (int, error) {
	updated := 0
	err := filepath.WalkDir(i.path, func(path string, d fs.DirEntry, err error) error {
		if d.Type().IsRegular() {
			if ok, _ := i.addOrUpdateCache(path); ok {
				updated += 1
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	i.lastScan = time.Now()
	return updated, nil
}

func (i *ImageDirectory) addOrUpdateCache(filePath string) (bool, error) {
	cacheEntry, ok := i.contentCache[filePath]
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return false, err
	}

	if ok && fileInfo.ModTime().Unix() <= cacheEntry.info.ModTime().Unix() {
		return false, nil // Cache still current
	}

	h := sha1.New()
	if _, err = io.Copy(h, f); err != nil {
		return false, err
	}

	i.contentCache[filePath] = FileStat{
		info:     fileInfo,
		hashSha1: h.Sum(nil),
	}
	log.Printf("%s %x\n", fileInfo.Name(), i.contentCache[filePath].hashSha1)
	return true, nil
}

func (i *ImageDirectory) Upload(server *ImmichServer, concurrentUploads int) {
	sem := make(chan int, concurrentUploads)
	mu := sync.Mutex{}
	for imagePath, entry := range i.contentCache {
		h := entry.HashHexString()
		sem <- 1
		go func(imagePath, h string, mu *sync.Mutex) {
			rawUUID, err := server.Upload(imagePath, &h)
			if err != nil {
				log.Printf("Failed to upload image at '%s' to server: %s\n", imagePath, err.Error())
				<-sem
				return
			}
			u, err := uuid.Parse(rawUUID)
			if err != nil {
				<-sem
				return
			}
			entry.uuid = u
			mu.Lock()
			i.contentCache[imagePath] = entry
			mu.Unlock()
			if i.album != nil {
				err = server.AddToAlbum([]uuid.UUID{u}, *i.album)
			}
			if err != nil {
				log.Printf("Uploaded image at '%s' to server, but could not add to album '%s': %s\n", imagePath, (*i.album).String(), err.Error())
				<-sem
				return
			}
			<-sem
		}(imagePath, h, &mu)
	}
}
