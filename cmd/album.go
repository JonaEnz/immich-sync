package cmd

import (
	album "github.com/JonaEnz/immich-sync/cmd/album"
	"github.com/spf13/cobra"
)

func init() {
	albumCmd.AddCommand(album.AddAlbumCmd)
	albumCmd.AddCommand(album.ShowAlbumCmd)
	albumCmd.AddCommand(album.CreateAlbumCmd)
	rootCmd.AddCommand(albumCmd)
}

var albumCmd = &cobra.Command{
	Use:   "album",
	Short: "Create and show albums",
}
