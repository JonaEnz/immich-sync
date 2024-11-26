package cmd

import (
	"log"
	"time"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var (
	server            *immichserver.ImmichServer
	concurrentUploads int
)

func scanAll(imageDirs []*immichserver.ImageDirectory) {
	for _, dir := range imageDirs {
		log.Printf("Scanning directory %s...\n", dir.Path())
		read, err := dir.Read()
		if err != nil {
			log.Println(err)
			continue
		} else {
			log.Printf("Found %d new/updated files in %s.\n", read, dir.Path())
		}
		dir.Upload(server, concurrentUploads)
	}
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon mode, opens a unix socket for communication",
	Run: func(cmd *cobra.Command, args []string) {
		server = immichserver.NewImmichServer(apiKey, serverURL, deviceID)
		imageDirs := make([]*immichserver.ImageDirectory, len(watchDirs))
		for i := range watchDirs {
			idir := immichserver.NewImageDirectory(watchDirs[i])
			imageDirs[i] = &idir
		}
		rpcServer := socketrpc.NewRPCServer()
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
		go func() {
			for {
				scanAll(imageDirs)
				time.Sleep(time.Minute * time.Duration(scanInterval))
			}
		}()
		rpcServer.WaitForExit()
	},
}
