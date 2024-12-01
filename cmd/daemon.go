package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var (
	server            *immichserver.ImmichServer
	concurrentUploads int
)

func scanAll(imageDirs []*immichserver.ImageDirectory) {
	for _, dir := range imageDirs {
		log.Printf("Scanning directory %s...\n", dir.Path())
		read, err := dir.Read()
		if err != nil {
			log.Println(err)
			continue
		} else {
			log.Printf("Found %d new/updated files in %s.\n", read, dir.Path())
		}
		dir.Upload(server, concurrentUploads)
	}
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon mode, opens a unix socket for communication",
	Run: func(cmd *cobra.Command, args []string) {
		server = immichserver.NewImmichServer(apiKey, serverURL, deviceID)
		server.ImageDirs = make([]*immichserver.ImageDirectory, len(watchDirs))
		for i := range watchDirs {
			idir := immichserver.NewImageDirectory(watchDirs[i])
			server.ImageDirs[i] = &idir
		}
		rpcServer := socketrpc.NewRPCServer()
		rpcServer.RegisterCallback(socketrpc.CmdScanAll, func(s string) (byte, string) {
			scanAll(server.ImageDirs)
			return socketrpc.ErrOk, ""
		})
		rpcServer.RegisterCallback(socketrpc.CmdStatus, func(s string) (byte, string) {
			result := ""
			for _, d := range server.ImageDirs {
				result += d.String() + "\n"
			}
			return socketrpc.ErrOk, result[:len(result)-1]
		})
		rpcServer.RegisterCallback(socketrpc.CmdAddDir, addDir)
		rpcServer.RegisterCallback(socketrpc.CmdRmDir, rmDir)
		rpcServer.RegisterCallback(socketrpc.CmdUploadFile, uploadFile)
		rpcServer.RegisterCallback(socketrpc.CmdCreateAlbum, createAlbum)
		rpcServer.RegisterCallback(socketrpc.CmdAddAlbum, addToAlbum)
		rpcServer.RegisterCallback(socketrpc.CmdDownloadAlbum, downloadAlbum)
		rpcServer.Start()
		go func() {
			for {
				scanAll(server.ImageDirs)
				time.Sleep(time.Minute * time.Duration(scanInterval))
			}
		}()
		rpcServer.WaitForExit()
	},
}

func addDir(path string) (byte, string) {
	stat, err := os.Stat(path)
	if err != nil {
		return socketrpc.ErrFileNotFound, err.Error()
	}
	if !stat.IsDir() {
		return socketrpc.ErrWrongArgs, fmt.Sprintf("'%s' is not a directory", path)
	}
	iDir := immichserver.NewImageDirectory(path)
	server.ImageDirs = append(server.ImageDirs, &iDir)
	updateConfig()
	return socketrpc.ErrOk, ""
}

func rmDir(path string) (byte, string) {
	stat, err := os.Stat(path)
	if err != nil {
		return socketrpc.ErrFileNotFound, err.Error()
	}
	if !stat.IsDir() {
		return socketrpc.ErrWrongArgs, fmt.Sprintf("'%s' is not a directory", path)
	}
	for i := range server.ImageDirs {
		if server.ImageDirs[i].Path() == path {
			server.ImageDirs = append(server.ImageDirs[:i], server.ImageDirs[i+1:]...)
			updateConfig()
			return socketrpc.ErrOk, ""
		}
	}
	return socketrpc.ErrGeneric, fmt.Sprintf("'%s' is not watched by immich-sync and could not be removed.", path)
}

func createAlbum(albumName string) (byte, string) {
	_, err := server.CreateNewAlbum(albumName)
	if err != nil {
		return socketrpc.ErrGeneric, err.Error()
	}
	return socketrpc.ErrOk, ""
}

func addToAlbum(args string) (byte, string) {
	splitArgs := strings.Split(args, ":")
	if len(splitArgs) != 2 {
		return socketrpc.ErrWrongArgs, ""
	}
	path, albumName := splitArgs[0], splitArgs[1]
	imageUUID, err := server.GetImageUUIDByPath(path)
	if err != nil {
		return socketrpc.ErrGeneric, err.Error()
	}
	albumUUID, err := server.GetAlbumUUIDByName(albumName)
	if err != nil {
		return socketrpc.ErrGeneric, err.Error()
	}
	err = server.AddToAlbum([]uuid.UUID{imageUUID}, albumUUID)
	if err != nil {
		return socketrpc.ErrGeneric, err.Error()
	}
	return socketrpc.ErrOk, ""
}

func downloadAlbum(args string) (byte, string) {
	splitArgs := strings.Split(args, ":")
	if len(splitArgs) != 2 {
		return socketrpc.ErrWrongArgs, ""
	}
	albumName, path := splitArgs[0], splitArgs[1]

	// Download album
	albumUUID, err := server.GetAlbumUUIDByName(albumName)
	if err != nil {
		return socketrpc.ErrGeneric, err.Error()
	}
	album, _ := server.Album(albumUUID)
	for _, asset := range album.Assets {
		if asset.IsTrashed || asset.IsArchived {
			continue
		}
		imageUUID, err := uuid.Parse(asset.ID)
		if err != nil {
			log.Println(err)
			continue
		}
		err = server.Download(path, imageUUID)
		if err != nil {
			if os.IsNotExist(err) {
				return socketrpc.ErrFileNotFound, ""
			}
			return socketrpc.ErrGeneric, err.Error()
		}

	}
	return socketrpc.ErrOk, ""
}

func uploadFile(arg string) (byte, string) {
	paths := strings.Split(arg, ":")
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			return socketrpc.ErrFileNotFound, fmt.Sprintf("File '%s' does not exist / could not be accessed", path)
		}
		if stat.IsDir() {
			return socketrpc.ErrWrongArgs, fmt.Sprintf("'%s' is a directory, this is not currently supported", path)
		}
	}
	success, failed := 0, 0
	for _, path := range paths {
		log.Printf("Uploading %s\n", path)
		_, err := server.Upload(path, nil)
		if err != nil {
			failed += 1
		} else {
			success += 1
		}
	}
	answer := fmt.Sprintf("Uploaded %d files, %d failed", success, failed)
	if failed > 0 {
		return socketrpc.ErrGeneric, answer
	}
	return socketrpc.ErrOk, answer
}

func updateConfig() {
	paths := []string{}
	for i := range server.ImageDirs {
		paths = append(paths, server.ImageDirs[i].Path())
	}
	viper.Set("watch", paths)
	viper.WriteConfig()
}
