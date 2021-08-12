package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ServerURL string `json:"serverURL"`
	ServerPort int `json:"serverPort"`
}

func LoadConfigFromPath(path string, config *Config) error {
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer configFile.Close()
	decoder := json.NewDecoder(configFile)
	return decoder.Decode(config)
}