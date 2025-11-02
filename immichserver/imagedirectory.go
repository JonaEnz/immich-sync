package immichserver

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/radovskyb/watcher"
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
	uploaded bool
	updated  bool
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

func (i *ImageDirectory) StartScan(server *ImmichServer, keepChangedFiles bool) {
	w := watcher.New()
	w.FilterOps(watcher.Create, watcher.Write)
	if err := w.AddRecursive(i.path); err != nil {
		log.Printf("Failed to start directory watcher for '%s': %s\n", i.path, err)
		return
	}
	go func() {
		for {
			select {
			case event := <-w.Event:
				switch event.Op {
				case watcher.Write:
					fallthrough
				case watcher.Create:
					if event.IsDir() {
						break
					}
					if ok, err := i.addOrUpdateCache(event.Path); !ok {
						log.Printf("Handling file event for '%s' failed: %s\n", event.Path, err)
						break
					}
					i.Upload(server, 1, keepChangedFiles)
				default:
					log.Printf("Unknown watcher event: %d\n", event.Op)
				}
			case err := <-w.Error:
				log.Println(fmt.Errorf("watcher error: %w", err))
			case <-w.Closed:
				log.Fatalln("Watcher closed")
			}
		}
	}()
	w.Start(1 * time.Second)
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
	cacheEntry, alreadyExists := i.contentCache[filePath]
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return false, err
	}

	if alreadyExists && fileInfo.ModTime().Unix() <= cacheEntry.info.ModTime().Unix() {
		return false, nil // Cache still current
	}

	h := sha1.New()
	if _, err = io.Copy(h, f); err != nil {
		return false, err
	}

	i.contentCache[filePath] = FileStat{
		info:     fileInfo,
		hashSha1: h.Sum(nil),
		uploaded: cacheEntry.uploaded,
		uuid:     cacheEntry.uuid,
		updated:  alreadyExists,
	}
	log.Printf("%s %x\n", fileInfo.Name(), i.contentCache[filePath].hashSha1)
	return true, nil
}

func (i *ImageDirectory) Upload(server *ImmichServer, concurrentUploads int, keepChangedFiles bool) {
	sem := make(chan int, concurrentUploads)
	mu := sync.Mutex{}
	copiedCache := i.contentCache
	for imagePath, entry := range copiedCache {
		if entry.uploaded && !entry.updated {
			continue
		}
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
			if entry.uploaded && entry.updated {
				if err = server.CopyMetadata(entry.uuid, u); err != nil {
					log.Printf("Failed to copy metadata to new asset: %v", err)
					if !strings.Contains(err.Error(), "version error:") {
						<-sem
						return
					}
				}
				if !keepChangedFiles {
					err = server.Delete(entry.uuid)
					if err != nil {
						log.Printf("Error deleting old version of image: %s\n", err)
					}

				}
			}
			entry.uuid = u
			entry.uploaded = true
			entry.updated = false
			mu.Lock()
			i.contentCache[imagePath] = entry
			mu.Unlock()
			if i.album != nil {
				err = server.AddToAlbum([]uuid.UUID{entry.uuid}, *i.album)
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
