package cf_k8s_api

import (
	"encoding/json"
	"os"
)

type Config struct {
	ServerURL string `json:serverURL`
}
var defaults = Config{
	ServerURL: "https://api.example.org",
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