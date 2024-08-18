package utils

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var tmpDir string
var configFilePath string

func GetStreamPath(base, channel string) (path string) {
	return fmt.Sprintf("%s/%s", base, channel)
}

func GetStreamFileName(channel string) (playlist string) {
	return fmt.Sprintf("%s.m3u8", channel)
}

func GetBaseFolder() string {
	if tmpDir != "" {
		return tmpDir
	}

	var err error
	tmpDir, err = os.MkdirTemp("", "stream-to-iptv-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	logrus.Infof("created temp dir: %s", tmpDir)

	return tmpDir
}

func CleanTempDir() {
	if tmpDir != "" {
		os.RemoveAll(tmpDir)
	}
	tmpDir = ""
}

func GetPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8068"
}

func GetConfigPath() string {
	if configFilePath != "" {
		return configFilePath
	}

	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile
	}

	// Define a command-line flag for the config file
	configFile := flag.String("config", "config.json", "Path to the JSON config file")
	flag.Parse()
	configFilePath = *configFile

	if configFilePath == "" {
		configFilePath = "config.json"
	}

	return configFilePath
}
