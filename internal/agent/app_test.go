package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestRunJobDownloadsBacksUpAndMerges(t *testing.T) {
	root := t.TempDir()
	install := filepath.Join(root, "install")
	mustWrite(t, filepath.Join(install, "Agent_TNLX", "Agent_TNLX.exe"), "old")
	mustWrite(t, filepath.Join(install, "Agent_TNLX", "keep.txt"), "keep")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"apps": []RemoteApp{{Name: "Agent_TNLX", Path: "Agent_TNLX"}}}})
	})
	mux.HandleFunc("/api/apps/Agent_TNLX/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"files": []FileEntry{
			{Path: "Agent_TNLX.exe", Size: 3},
			{Path: "config/app.json", Size: 2},
		}, "total_bytes": 5}})
	})
	mux.HandleFunc("/api/apps/Agent_TNLX/files/Agent_TNLX.exe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new"))
	})
	mux.HandleFunc("/api/apps/Agent_TNLX/files/config/app.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
	})
	mux.HandleFunc("/api/update/report", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	app := New(Config{
		ServerURL:   ts.URL,
		InstallPath: install,
		TempPath:    filepath.Join(install, "_update"),
		BackupPath:  filepath.Join(install, "_backup"),
	})
	job, err := app.createJob([]string{"Agent_TNLX"})
	if err != nil {
		t.Fatal(err)
	}
	app.runJob(job.ID)

	app.mu.Lock()
	done := app.cloneJobLocked(job.ID)
	app.mu.Unlock()
	if done.Status != "success" {
		t.Fatalf("job status got %s error %#v", done.Status, done.AppResults)
	}
	assertFile(t, filepath.Join(install, "Agent_TNLX", "Agent_TNLX.exe"), "new")
	assertFile(t, filepath.Join(install, "Agent_TNLX", "config", "app.json"), "{}")
	assertFile(t, filepath.Join(install, "Agent_TNLX", "keep.txt"), "keep")

	backups, err := filepath.Glob(filepath.Join(install, "_backup", "Agent_TNLX", "*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup, got %v, err=%v", backups, err)
	}
	assertFile(t, filepath.Join(backups[0], "Agent_TNLX.exe"), "old")
}

func TestDuplicateJobLock(t *testing.T) {
	app := New(DefaultConfig())
	if _, err := app.createJob([]string{"Agent_TNLX"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.createJob([]string{"Agent_TNLX"}); err == nil {
		t.Fatal("expected duplicate lock error")
	}
}
