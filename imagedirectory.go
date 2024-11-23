package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
)

type ImageDirectory struct {
	path         string
	contentCache map[string]FileStat
}

type FileStat struct {
	info     os.FileInfo
	hashSha1 []byte
}

func (f *FileStat) HashHexString() string {
	return fmt.Sprintf("%x", f.hashSha1)
}

func NewImageDirectory(path string) ImageDirectory {
	return ImageDirectory{
		path:         path,
		contentCache: make(map[string]FileStat),
	}
}

func (i *ImageDirectory) Path() string {
	return i.path
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
	fmt.Printf("%s %x\n", fileInfo.Name(), i.contentCache[filePath].hashSha1)
	return true, nil
}
