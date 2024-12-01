package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/JonaEnz/immich-sync/immichserver"
	"github.com/JonaEnz/immich-sync/socketrpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var (
	server            *immichserver.ImmichServer
	imageDirs         []*immichserver.ImageDirectory
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
		imageDirs = make([]*immichserver.ImageDirectory, len(watchDirs))
		for i := range watchDirs {
			idir := immichserver.NewImageDirectory(watchDirs[i])
			imageDirs[i] = &idir
		}
		rpcServer := socketrpc.NewRPCServer()
		rpcServer.RegisterCallback(socketrpc.CmdScanAll, func(s string) (byte, string) {
			scanAll(imageDirs)
			return socketrpc.ErrOk, ""
		})
		rpcServer.RegisterCallback(socketrpc.CmdStatus, func(s string) (byte, string) {
			result := ""
			for _, d := range imageDirs {
				result += d.String() + "\n"
			}
			return socketrpc.ErrOk, result[:len(result)-1]
		})
		rpcServer.RegisterCallback(socketrpc.CmdAddDir, addDir)
		rpcServer.RegisterCallback(socketrpc.CmdRmDir, rmDir)
		rpcServer.RegisterCallback(socketrpc.CmdUploadFile, uploadFile)
		rpcServer.Start()
		go func() {
			for {
				scanAll(imageDirs)
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
	imageDirs = append(imageDirs, &iDir)
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
	for i := range imageDirs {
		if imageDirs[i].Path() == path {
			imageDirs = append(imageDirs[:i], imageDirs[i+1:]...)
			updateConfig()
			return socketrpc.ErrOk, ""
		}
	}
	return socketrpc.ErrGeneric, fmt.Sprintf("'%s' is not watched by immich-sync and could not be removed.", path)
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
	for i := range imageDirs {
		paths = append(paths, imageDirs[i].Path())
	}
	viper.Set("watch", paths)
	viper.WriteConfig()
}
