package cmd

import (
	"fmt"
	"log"

	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
}

var AddAlbumCmd = &cobra.Command{
	Use:   "add",
	Short: "<image path> <album name> - Adds the image to the album if it exists",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			log.Fatalln("Service daemon not running.")
		}
		defer rpcClient.Close()
		_, err = rpcClient.SendMessage(socketrpc.CmdAddAlbum, args[0]+":"+args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Done")
	},
}
