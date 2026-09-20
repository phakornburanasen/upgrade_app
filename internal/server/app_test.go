package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanAppsAndListFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Agent_TNLX", "Agent_TNLX.exe"), "exe")
	mustWrite(t, filepath.Join(root, "Agent_TNLX", "config", "app.json"), "{}")
	mustWrite(t, filepath.Join(root, "SSO_Check", "SSO_Check.exe"), "exe")
	mustWrite(t, filepath.Join(root, "plain.txt"), "skip")

	app, err := New(Config{SourcePath: root, LogPath: filepath.Join(t.TempDir(), "server.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	apps, err := app.scanApps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("got %d apps, want 2", len(apps))
	}

	files, total, err := app.listFiles("Agent_TNLX")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].Path != "Agent_TNLX.exe" || files[1].Path != "config/app.json" {
		t.Fatalf("unexpected files: %#v", files)
	}
	if total != 5 {
		t.Fatalf("total got %d want 5", total)
	}
}

func TestListFilesResolvesAppNameCase(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Car", "car.exe"), "exe")
	app, err := New(Config{SourcePath: root, LogPath: filepath.Join(t.TempDir(), "server.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	files, _, err := app.listFiles("car")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "car.exe" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
