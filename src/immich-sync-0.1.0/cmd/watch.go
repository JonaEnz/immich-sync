package cmd

import (
	cmd "github.com/JonaEnz/immich-sync/cmd/watch"
	"github.com/spf13/cobra"
)

func init() {
	watchCmd.AddCommand(cmd.AddWatchCmd)
	watchCmd.AddCommand(cmd.RmWatchCmd)
	rootCmd.AddCommand(watchCmd)
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Adds or remove directories from scan",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
