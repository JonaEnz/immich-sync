package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/JonaEnz/immich-sync/immichserver"
)

var (
	apiKey            string
	server            *immichserver.ImmichServer
	concurrentUploads int
)

func main() {
	serverURL := flag.String("url", "http://192.168.0.136:2283/api", "Immich server url with trailing /api")
	apiKey = *flag.String("api-key", "y2gDkeRqPpiTcM0CpQpTc58hxTutkltzBOHLYYw70", "api key")
	concurrentUploads = *flag.Int("concurrentUploads", 5, "Number of concurrent uploads")
	daemon := flag.Bool("d", false, "Start as daemon")
	scanMins := flag.Int("scan-minutes", 15, "Minutes delay between scans (requires -d)")
	flag.Parse()

	server = immichserver.NewImmichServer(apiKey, *serverURL)

	i := NewImageDirectory("/home/jona/Pictures/Screenshots")
	imageDirs := []*ImageDirectory{&i}

	if *daemon && *scanMins >= 1 {
		for {
			doScan(imageDirs)
			time.Sleep(time.Minute * time.Duration(*scanMins))
		}
	} else {
		doScan(imageDirs)
	}
}

func doScan(imageDirs []*ImageDirectory) {
	sem := make(chan int, concurrentUploads)
	for _, dir := range imageDirs {
		fmt.Printf("Scanning directory %s...\n", dir.path)
		read, err := dir.Read()
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			fmt.Printf("Found %d new/updated files in %s.\n", read, dir.path)
		}
		for imagePath, entry := range dir.contentCache {
			h := entry.HashHexString()
			sem <- 1
			go func(imagePath, h string) {
				err := server.Upload(imagePath, &h)
				if err != nil {
					fmt.Println(err)
				}
				<-sem
			}(imagePath, h)
		}
	}
}
