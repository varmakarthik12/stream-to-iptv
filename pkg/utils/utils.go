package utils

import (
	"flag"
	"fmt"
	"os"
	"strconv"

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

func MaxSegmentsCount() string {
	if count := os.Getenv("MAX_SEGMENTS_COUNT"); count != "" {
		countInt, err := strconv.Atoi(count)
		if err != nil && countInt > 0 {
			return count
		}
	}
	return "10"
}

func MaxSegmentTime() string {
	if time := os.Getenv("MAX_SEGMENT_TIME"); time != "" {
		timeInt, err := strconv.Atoi(time)
		if err != nil && timeInt > 0 {
			return time
		}
	}
	return "15"
}

func GetEPGURL() string {
	if epgURL := os.Getenv("EPG_URL"); epgURL != "" {
		return epgURL
	}
	return "https://avkb.short.gy/epg.xml.gz"
}

func GetIpAddr() string {
	if ipAddr := os.Getenv("IP_ADDR"); ipAddr != "" {
		return ipAddr
	}
	return ""
}
