package cmd

import (
	"fmt"
	"log"

	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
}

var DownloadAlbumCmd = &cobra.Command{
	Use:   "download",
	Short: "<album name> <output path> - Download the album to the specified path",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			log.Fatalln("Service daemon not running.")
		}
		defer rpcClient.Close()
		_, err = rpcClient.SendMessage(socketrpc.CmdDownloadAlbum, args[0]+"//"+args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Done")
	},
}
