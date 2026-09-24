package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEncryptedConfig(t *testing.T) {
	plain := []byte(`{
		"host": "127.0.0.1",
		"port": 45123,
		"server_url": "http://10.0.32.71:45000/",
		"install_path": "C:\\tnl_appl",
		"temp_path": "C:\\tnl_appl\\_update",
		"backup_path": "C:\\tnl_appl\\_backup"
	}`)
	encrypted, err := EncryptConfigBytes(plain)
	if err != nil {
		t.Fatalf("EncryptConfigBytes: %v", err)
	}
	path := filepath.Join(t.TempDir(), "agent.enc")
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Port != 45123 {
		t.Fatalf("Port = %d, want 45123", cfg.Port)
	}
	if cfg.ServerURL != "http://10.0.32.71:45000" {
		t.Fatalf("ServerURL = %q", cfg.ServerURL)
	}
}
