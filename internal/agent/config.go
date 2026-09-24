package agent

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type Config struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ServerURL   string `json:"server_url"`
	InstallPath string `json:"install_path"`
	TempPath    string `json:"temp_path"`
	BackupPath  string `json:"backup_path"`
}

func DefaultConfig() Config {
	return Config{
		Host:        "127.0.0.1",
		Port:        45100,
		ServerURL:   "http://127.0.0.1:45000",
		InstallPath: `C:\tnl_appl`,
		TempPath:    `C:\tnl_appl\_update`,
		BackupPath:  `C:\tnl_appl\_backup`,
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
	if IsEncryptedConfig(b) {
		b, err = DecryptConfigBytes(b)
		if err != nil {
			return cfg, err
		}
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 45100
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://127.0.0.1:45000"
	}
	cfg.ServerURL = strings.TrimRight(cfg.ServerURL, "/")
	if cfg.InstallPath == "" {
		cfg.InstallPath = `C:\tnl_appl`
	}
	if cfg.TempPath == "" {
		cfg.TempPath = `C:\tnl_appl\_update`
	}
	if cfg.BackupPath == "" {
		cfg.BackupPath = `C:\tnl_appl\_backup`
	}
	return cfg, nil
}
