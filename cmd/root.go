package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile      string
	serverURL    string
	apiKey       string
	deviceID     string
	watchDirs    []string
	scanInterval int

	rootCmd = &cobra.Command{
		Use:   "immich-sync",
		Short: "A client for uploading images to Immich",
		Long:  "A client for uploading images to Immich",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/immich-sync/config.yaml)")
	viper.SetDefault("watch", []string{})
	viper.SetDefault("deviceid", "defaultdeviceid")
	viper.SetDefault("server", "")
	viper.SetDefault("apikey", "")
	viper.SetDefault("schedule", 15)
	viper.SetDefault("concurrent-uploads", 5)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.config/immich-sync")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	if !viper.IsSet("server") || !viper.IsSet("apikey") {
		log.Fatal("Server and apikey need to be set in config file!")
	}
	watchDirs = viper.GetStringSlice("watch")
	deviceID = viper.GetString("deviceid")
	serverURL = viper.GetString("server")
	apiKey = viper.GetString("apikey")
	scanInterval = viper.GetInt("schedule")
	concurrentUploads = viper.GetInt("concurrent-uploads")
}