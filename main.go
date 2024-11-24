package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
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
	useRPCServer := flag.Bool("rpc", true, "Start RPC socket server")
	scanMins := flag.Int("scan-minutes", 15, "Minutes delay between scans (requires -d)")
	flag.Parse()

	server = immichserver.NewImmichServer(apiKey, *serverURL)

	i := NewImageDirectory("/home/jona/Pictures/Screenshots")
	imageDirs := []*ImageDirectory{&i}

	var rpcServer socketrpc.RPCServer
	if *daemon {
		if *useRPCServer {
			rpcServer = socketrpc.NewRPCServer()
			rpcServer.RegisterCallback(socketrpc.CmdScanAll, func(s string) (byte, string) {
				scanAll(imageDirs)
				return socketrpc.ErrOk, ""
			})
			rpcServer.RegisterCallback(socketrpc.CmdStatus, func(s string) (byte, string) {
				result := ""
				for _, d := range imageDirs {
					result += d.String() + "\n"
				}
				return socketrpc.ErrOk, result[:len(result)-1]
			})
			rpcServer.Start()
		}
		if *scanMins >= 1 {
			for {
				scanAll(imageDirs)
				time.Sleep(time.Minute * time.Duration(*scanMins))
			}
		} else if *useRPCServer {
			rpcServer.WaitForExit()
		}

	}

	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		switch os.Args[1] {
		case "scan":
			rpcClient, err := socketrpc.NewRPCClient()
			if err != nil {
				scanAll(imageDirs) // No daemon, scan yourself
				return
			}
			defer rpcClient.Close()
			rpcClient.SendMessage(socketrpc.CmdScanAll, "")
		case "status":
			rpcClient, err := socketrpc.NewRPCClient()
			if err != nil {
				fmt.Println("Immich-Sync is not running.")
				return
			}
			defer rpcClient.Close()
			answer, err := rpcClient.SendMessage(socketrpc.CmdStatus, "")
			fmt.Println("Immich-Sync status:")
			for _, l := range strings.Split(answer, "\n") {
				fmt.Println(l)
			}
		}
	}
}

func startDaemon() {
}

func scanAll(imageDirs []*ImageDirectory) {
	sem := make(chan int, concurrentUploads)
	for _, dir := range imageDirs {
		log.Printf("Scanning directory %s...\n", dir.path)
		read, err := dir.Read()
		if err != nil {
			log.Println(err)
			continue
		} else {
			log.Printf("Found %d new/updated files in %s.\n", read, dir.path)
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
