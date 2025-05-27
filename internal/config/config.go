package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	con := Config{}
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return con, err
	}
	gatorDir := homeDirectory + "/.gatorconfig.json"

	data, err := os.ReadFile(gatorDir)
	if err != nil {
		return con, err
	}

	err = json.Unmarshal(data, &con)
	if err != nil {
		return con, err
	}

	return con, nil
}
