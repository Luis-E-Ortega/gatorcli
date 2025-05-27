package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	// Initialize instance of Config struct
	config := Config{}

	// Run helper function to get Config file path
	gatorDir, err := getConfigFilePath()
	if err != nil {
		return config, err
	}

	// Read data from file given the full path
	data, err := os.ReadFile(gatorDir)
	if err != nil {
		return config, err
	}

	// Unmarshal into config struct instance we created
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func getConfigFilePath() (string, error) {
	const configFileName = "/.gatorconfig.json"
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use filepath to join paths for full portable path, usable across different systems
	fullPath := filepath.Join(homeDirectory, configFileName)

	return fullPath, nil
}

func write(cfg Config) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	err := write(*c)
	if err != nil {
		return err
	}
	return nil
}
