package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"stream-to-iptv/pkg/utils"
)

type Stream struct {
	Name      string   `json:"channel"`
	Media     string   `json:"media"`
	Logo      string   `json:"logo"`
	Groups    []string `json:"groups"`
	ProgramId string   `json:"program_id"`
}

func GetStreamConfig() ([]Stream, error) {
	var streams []Stream
	configFile := utils.GetConfigPath()

	// Open the JSON file
	jsonFile, err := os.Open(configFile)
	if err != nil {
		return streams, fmt.Errorf("failed to open config file: %v", err)
	}
	defer jsonFile.Close()

	// Read the JSON file
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return streams, fmt.Errorf("failed to read config file: %v", err)
	}

	// Unmarshal the JSON data
	if err := json.Unmarshal(byteValue, &streams); err != nil {
		return streams, fmt.Errorf("failed to unmarshal config data: %v", err)
	}

	return streams, nil
}
