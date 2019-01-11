package config

import (
	"encoding/json"
	"os"

	"github.com/burntsushi/toml"
	"golang.org/x/oauth2"
)

// Config describes the configuration for the draupnir client
type Config struct {
	Domain   string
	Token    oauth2.Token
	Database string
}

// Load parses the client config file
func Load() (Config, error) {
	config := Config{Domain: "set-me-to-a-real-domain"}
	file, err := os.Open(configFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			err = Store(config)
			return config, err
		}
		return config, err
	}
	_, err = toml.DecodeReader(file, &config)
	if err != nil {
		// Older versions of .draupnir were JSON formatted
		// TODO: remove this in a future major version
		file.Seek(0, 0)
		err = json.NewDecoder(file).Decode(&config)
	}
	return config, err
}

// Store serialises the given config struct as TOML and saves it to disk
func Store(config Config) error {
	file, err := os.Create(configFilePath())
	if err != nil {
		return err
	}
	err = toml.NewEncoder(file).Encode(config)
	return err
}

func configFilePath() string {
	return os.Getenv("HOME") + "/.draupnir"
}
