package cmd

import (
	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scans for new images, uses the daemon if it is running",
	Run: func(cmd *cobra.Command, args []string) {
		server = immichserver.NewImmichServer(apiKey, serverURL, deviceID)

		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			imageDirs := make([]*immichserver.ImageDirectory, len(watchDirs))
			for i := range watchDirs {
				idir := immichserver.NewImageDirectory(watchDirs[i].Path, false)
				if len(watchDirs[i].Album) > 0 {
					albumUUID, err := server.GetAlbumByUUIDOrName(watchDirs[0].Album)
					if err == nil {
						idir.SetAlbum(&albumUUID)
					}
				}
				imageDirs[i] = &idir
			}
			scanAll(imageDirs) // No daemon, scan yourself
			return
		}
		defer rpcClient.Close()
		rpcClient.SendMessage(socketrpc.CmdScanAll, "")
	},
}
