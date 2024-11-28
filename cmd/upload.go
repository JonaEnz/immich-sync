package cmd

import (
	"fmt"
	"strings"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(uploadCmd)
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Uploads image(s) to Immich",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server = immichserver.NewImmichServer(apiKey, serverURL, deviceID)

		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			fmt.Println("Failed to connect to daemon, is the service running?")
			return
		}
		defer rpcClient.Close()
		answer, err := rpcClient.SendMessage(socketrpc.CmdUploadFile, strings.Join(args, ":"))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Success: %s\n", answer)
	},
}
