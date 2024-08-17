package utils

import (
	"flag"
	"fmt"
)

func GetStreamPath(base, channel string) (path string) {
	return fmt.Sprintf("%s/%s", base, channel)
}

func GetStreamFileName(channel string) (playlist string) {
	return fmt.Sprintf("%s.m3u8", channel)
}

func GetBaseFolder() string {
	return "./media"
}

func GetPort() string {
	return "8068"
}

func GetConfigPath() string {
	// Define a command-line flag for the config file
	configFile := flag.String("config", "config.json", "Path to the JSON config file")
	flag.Parse()
	return *configFile
}
