package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

var albumFlag string

func init() {
	uploadCmd.PersistentFlags().StringVar(&albumFlag, "album", "", "Add uploaded image to album with this name")
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
		request := socketrpc.UploadFileRequest{
			Paths: args,
			Album: albumFlag,
		}
		jsonRequest, err := json.Marshal(request)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		answer, err := rpcClient.SendMessage(socketrpc.CmdUploadFile, string(jsonRequest))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Success: %s\n", answer)
	},
}
