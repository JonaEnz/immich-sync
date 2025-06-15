package cmd

import (
	"fmt"
	"log"

	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
}

var AddWatchCmd = &cobra.Command{
	Use:   "add",
	Short: "",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			log.Fatalln("Service daemon not running.")
		}
		defer rpcClient.Close()
		_, err = rpcClient.SendMessage(socketrpc.CmdAddDir, args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Done")
	},
}
