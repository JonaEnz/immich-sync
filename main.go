package main

import (
	"flag"
	"fmt"

	"github.com/JonaEnz/immich-sync/immichserver"
)

var apiKey string

func main() {
	serverURL := flag.String("url", "http://192.168.0.136:2283/api", "Immich server url with trailing /api")
	apiKey = *flag.String("api-key", "y2gDkeRqPpiTcM0CpQpTc58hxTutkltzBOHLYYw70", "api key")
	concurrentUploads := flag.Int("concurrentUploads", 5, "Number of concurrent uploads")

	server := immichserver.NewImmichServer(apiKey, *serverURL)

	imageDir := NewImageDirectory("/home/jona/Pictures/Screenshots")
	err := imageDir.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	sem := make(chan int, *concurrentUploads)

	for imagePath, entry := range imageDir.contentCache {
		fmt.Printf("%s %x\n", entry.info.Name(), entry.hashSha1)
		h := entry.HashHexString()
		sem <- 1
		go func(imagePath, h string) {
			err = server.Upload(imagePath, &h)
			if err != nil {
				fmt.Println(err)
			}
			<-sem
		}(imagePath, h)
	}

	// albums, err := client.GetAllAlbums(context.Background(), oapi.GetAllAlbumsParams{})
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// for _, a := range albums {
	// 	fmt.Println(a.AlbumName)
	// }
}
