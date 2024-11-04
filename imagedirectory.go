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

func (i *ImageDirectory) Read() error {
	p, err := os.ReadDir(i.path)
	if err != nil {
		return err
	}
	for _, entry := range p {
		if entry.Type().IsRegular() {
			i.addOrUpdateCache(path.Join(i.path, entry.Name()))
		}
	}
	return nil
}

func (i *ImageDirectory) addOrUpdateCache(filePath string) error {
	cacheEntry, ok := i.contentCache[filePath]
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return err
	}

	if ok && fileInfo.ModTime().Unix() <= cacheEntry.info.ModTime().Unix() {
		return nil // Cache still current
	}

	h := sha1.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}

	i.contentCache[filePath] = FileStat{
		info:     fileInfo,
		hashSha1: h.Sum(nil),
	}
	return nil
}
