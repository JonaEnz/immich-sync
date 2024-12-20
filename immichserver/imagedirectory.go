package immichserver

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
)

type ImageDirectory struct {
	path         string
	contentCache map[string]FileStat
	lastScan     time.Time
}

type FileStat struct {
	info     os.FileInfo
	hashSha1 []byte
	uuid     uuid.UUID
}

func (f *FileStat) HashHexString() string {
	return fmt.Sprintf("%x", f.hashSha1)
}

func NewImageDirectory(path string) ImageDirectory {
	return ImageDirectory{
		path:         path,
		contentCache: make(map[string]FileStat),
		lastScan:     time.Time{},
	}
}

func (i *ImageDirectory) Path() string {
	return i.path
}

func (i *ImageDirectory) Count() int {
	return len(i.contentCache)
}

func (i *ImageDirectory) String() string {
	return fmt.Sprintf("%s: %d images, last scanned %s", i.path, i.Count(), i.lastScan.Format("Mon Jan 2 15:04:05 MST 2006"))
}

func (i *ImageDirectory) Read() (int, error) {
	p, err := os.ReadDir(i.path)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, entry := range p {
		if entry.Type().IsRegular() {
			if ok, _ := i.addOrUpdateCache(path.Join(i.path, entry.Name())); ok {
				updated += 1
			}
		}
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
	for imagePath, entry := range i.contentCache {
		h := entry.HashHexString()
		sem <- 1
		go func(imagePath, h string) {
			rawUUID, err := server.Upload(imagePath, &h)
			if err != nil {
				fmt.Println(err)
				<-sem
				return
			}
			u, err := uuid.Parse(rawUUID)
			if err != nil {
				<-sem
				return
			}
			entry.uuid = u
			i.contentCache[imagePath] = entry
			<-sem
		}(imagePath, h)
	}
}
