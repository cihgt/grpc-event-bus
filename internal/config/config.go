package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port int `json:"port"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка во время чтения конфига: %v", err)
	}
	err = json.Unmarshal(raw, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
