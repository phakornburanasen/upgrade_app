package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"upgrade_app/internal/agentui"
)

type App struct {
	cfg    Config
	client *http.Client
	mu     sync.Mutex
	jobs   map[string]*Job
	locks  map[string]string
}

type RemoteApp struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
}

type FileEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type Job struct {
	ID             string          `json:"id"`
	Status         string          `json:"status"`
	Applications   []string        `json:"applications"`
	CurrentApp     string          `json:"current_app,omitempty"`
	TotalFiles     int             `json:"total_files"`
	CompletedFiles int             `json:"completed_files"`
	TotalBytes     int64           `json:"total_bytes"`
	CompletedBytes int64           `json:"completed_bytes"`
	Message        string          `json:"message"`
	Error          string          `json:"error,omitempty"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	AppResults     []AppJobResult  `json:"app_results"`
	activeAppLocks map[string]bool `json:"-"`
}

type AppJobResult struct {
	Application string `json:"application"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

func New(cfg Config) *App {
	cfg.ServerURL = strings.TrimRight(cfg.ServerURL, "/")
	return &App{cfg: cfg, client: &http.Client{Timeout: 30 * time.Minute}, jobs: map[string]*Job{}, locks: map[string]string{}}
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.index)
	mux.HandleFunc("/static/", a.static)
	mux.HandleFunc("/api/local/status", a.localStatus)
	mux.HandleFunc("/api/apps", a.apps)
	mux.HandleFunc("/api/apps/", a.appRoutes)
	mux.HandleFunc("/api/update", a.update)
	mux.HandleFunc("/api/update/status/", a.updateStatus)
	return mux
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := agentui.Files.ReadFile("index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "UI_ERROR", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

func (a *App) static(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/")
	clean := path.Clean(name)
	if clean == "." || strings.HasPrefix(clean, "../") {
		http.NotFound(w, r)
		return
	}
	b, err := agentui.Files.ReadFile("static/" + clean)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ctype := mime.TypeByExtension(filepath.Ext(clean)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

func (a *App) localStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	host, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"hostname":     host,
		"server_url":   a.cfg.ServerURL,
		"install_path": a.cfg.InstallPath,
	}})
}

func (a *App) apps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	apps, err := a.fetchApps()
	if err != nil {
		writeError(w, http.StatusBadGateway, "SERVER_UNAVAILABLE", err.Error())
		return
	}
	for i := range apps {
		apps[i].Installed = dirExists(filepath.Join(a.cfg.InstallPath, apps[i].Name))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"apps": apps}})
}

func (a *App) appRoutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/apps/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "files" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	appName, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(appName) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_APPLICATION", "invalid application")
		return
	}
	files, total, err := a.fetchFiles(appName)
	if err != nil {
		writeError(w, http.StatusBadGateway, "FILE_CHECK_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"application": appName,
		"file_count":  len(files),
		"total_bytes": total,
		"files":       files,
	}})
}

func (a *App) update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var req struct {
		Applications []string `json:"applications"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON")
		return
	}
	apps := normalizeApps(req.Applications)
	if len(apps) == 0 {
		writeError(w, http.StatusBadRequest, "NO_APPLICATIONS", "select at least one application")
		return
	}
	job, err := a.createJob(apps)
	if err != nil {
		writeError(w, http.StatusConflict, "UPDATE_ALREADY_RUNNING", err.Error())
		return
	}
	go a.runJob(job.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"success": true, "data": map[string]any{"job_id": job.ID}})
}

func (a *App) updateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/update/status/")
	a.mu.Lock()
	job := a.cloneJobLocked(id)
	a.mu.Unlock()
	if job == nil {
		writeError(w, http.StatusNotFound, "JOB_NOT_FOUND", "job not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": job})
}

func (a *App) createJob(apps []string) (*Job, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, app := range apps {
		if existing := a.locks[app]; existing != "" {
			return nil, fmt.Errorf("%s is already updating in job %s", app, existing)
		}
	}
	id := time.Now().Format("20060102150405")
	job := &Job{ID: id, Status: "running", Applications: apps, StartedAt: time.Now(), Message: "queued", activeAppLocks: map[string]bool{}}
	for _, app := range apps {
		a.locks[app] = id
		job.activeAppLocks[app] = true
	}
	a.jobs[id] = job
	return job, nil
}

func (a *App) runJob(id string) {
	a.mu.Lock()
	job := a.jobs[id]
	apps := append([]string(nil), job.Applications...)
	a.mu.Unlock()

	for _, appName := range apps {
		a.setJob(id, func(j *Job) {
			j.CurrentApp = appName
			j.Message = "listing files"
			j.TotalFiles = 0
			j.CompletedFiles = 0
			j.TotalBytes = 0
			j.CompletedBytes = 0
		})
		err := a.updateApp(id, appName)
		if err != nil {
			a.addResult(id, AppJobResult{Application: appName, Status: "failed", Error: err.Error()})
		} else {
			a.addResult(id, AppJobResult{Application: appName, Status: "success", Message: "completed"})
		}
	}

	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	job = a.jobs[id]
	failed := false
	for _, result := range job.AppResults {
		if result.Status == "failed" {
			failed = true
			break
		}
	}
	if failed {
		job.Status = "failed"
		job.Message = "completed with errors"
	} else {
		job.Status = "success"
		job.Message = "completed"
	}
	job.FinishedAt = &now
	for app := range job.activeAppLocks {
		delete(a.locks, app)
	}
}

func (a *App) updateApp(jobID, appName string) error {
	files, total, err := a.fetchFiles(appName)
	if err != nil {
		return err
	}
	a.setJob(jobID, func(j *Job) {
		j.TotalFiles = len(files)
		j.TotalBytes = total
		j.Message = "downloading"
	})
	stage := filepath.Join(a.cfg.TempPath, appName, jobID)
	backup := filepath.Join(a.cfg.BackupPath, appName, time.Now().Format("20060102_150405"))
	dest := filepath.Join(a.cfg.InstallPath, appName)
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	if err := os.MkdirAll(stage, 0755); err != nil {
		return err
	}
	for _, file := range files {
		if err := a.downloadOne(jobID, appName, file, stage); err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
	}
	a.setJob(jobID, func(j *Job) { j.Message = "backing up" })
	hadDest := dirExists(dest)
	if hadDest {
		if err := copyDir(dest, backup, nil); err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
	}
	a.setJob(jobID, func(j *Job) { j.Message = "installing" })
	if err := copyDir(stage, dest, nil); err != nil {
		if hadDest {
			_ = os.RemoveAll(dest)
			_ = copyDir(backup, dest, nil)
		}
		_ = os.RemoveAll(stage)
		return err
	}
	_ = os.RemoveAll(stage)
	a.setJob(jobID, func(j *Job) { j.Message = "installed" })
	return nil
}

func (a *App) downloadOne(jobID, appName string, file FileEntry, stage string) error {
	clean, err := cleanRelativePath(file.Path)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/api/apps/%s/files/%s", a.cfg.ServerURL, url.PathEscape(appName), escapePath(clean))
	resp, err := a.client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download %s failed: %s %s", clean, resp.Status, strings.TrimSpace(string(body)))
	}
	target := filepath.Join(stage, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			a.setJob(jobID, func(j *Job) {
				j.CompletedBytes += int64(n)
			})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	a.setJob(jobID, func(j *Job) {
		j.CompletedFiles++
		j.Message = "downloaded " + clean
	})
	return nil
}

func (a *App) fetchApps() ([]RemoteApp, error) {
	resp, err := a.client.Get(a.cfg.ServerURL + "/api/apps")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}
	var decoded struct {
		Success bool        `json:"success"`
		Apps    []RemoteApp `json:"apps"`
		Data    struct {
			Apps []RemoteApp `json:"apps"`
		} `json:"data"`
		Error any `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	apps := decoded.Apps
	if len(apps) == 0 {
		apps = decoded.Data.Apps
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	return apps, nil
}

func (a *App) fetchFiles(appName string) ([]FileEntry, int64, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/apps/%s/files", a.cfg.ServerURL, url.PathEscape(appName)))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, 0, fmt.Errorf("file list failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Success bool `json:"success"`
		Data    struct {
			Files      []FileEntry `json:"files"`
			TotalBytes int64       `json:"total_bytes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, 0, err
	}
	return decoded.Data.Files, decoded.Data.TotalBytes, nil
}

func (a *App) setJob(id string, update func(*Job)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if job := a.jobs[id]; job != nil {
		update(job)
	}
}

func (a *App) addResult(id string, result AppJobResult) {
	a.setJob(id, func(j *Job) {
		j.AppResults = append(j.AppResults, result)
	})
	_ = a.report(result)
}

func (a *App) report(result AppJobResult) error {
	body, _ := json.Marshal(map[string]any{
		"time":        time.Now().Format(time.RFC3339),
		"application": result.Application,
		"status":      result.Status,
		"message":     result.Message,
		"error":       result.Error,
	})
	resp, err := a.client.Post(a.cfg.ServerURL+"/api/update/report", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (a *App) cloneJobLocked(id string) *Job {
	job := a.jobs[id]
	if job == nil {
		return nil
	}
	cp := *job
	cp.Applications = append([]string(nil), job.Applications...)
	cp.AppResults = append([]AppJobResult(nil), job.AppResults...)
	return &cp
}

func normalizeApps(apps []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" || seen[app] {
			continue
		}
		seen[app] = true
		out = append(out, app)
	}
	return out
}

func escapePath(rel string) string {
	parts := strings.Split(rel, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"success": false, "error": map[string]string{"code": code, "message": message}})
}
