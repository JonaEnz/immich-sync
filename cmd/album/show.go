package cmd

import (
	"fmt"
	"log"

	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
}

var ShowAlbumCmd = &cobra.Command{
	Use:   "show",
	Short: "Show album info with given name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			log.Fatalln("Service daemon not running.")
		}
		defer rpcClient.Close()
		response, err := rpcClient.SendMessage(socketrpc.CmdShowAlbum, args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(response)
	},
}
