package server

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SourcePath string `json:"source_path"`
	LogPath    string `json:"log_path"`
}

func DefaultConfig() Config {
	return Config{
		Host:       "0.0.0.0",
		Port:       45000,
		SourcePath: `Z:\`,
		LogPath:    `logs\update-server.log`,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 45000
	}
	if cfg.SourcePath == "" {
		cfg.SourcePath = `Z:\`
	}
	if cfg.LogPath == "" {
		cfg.LogPath = `logs\update-server.log`
	}
	return cfg, nil
}
