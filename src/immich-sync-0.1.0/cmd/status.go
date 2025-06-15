package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the service daemon",
	Run: func(cmd *cobra.Command, args []string) {
		rpcClient, err := socketrpc.NewRPCClient()
		if err != nil {
			log.Fatalln("Service daemon not running.")
		}
		defer rpcClient.Close()
		answer, err := rpcClient.SendMessage(socketrpc.CmdStatus, "")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Immich-Sync status:")
		for _, l := range strings.Split(answer, "\n") {
			fmt.Println(l)
		}
	},
}
