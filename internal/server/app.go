package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type App struct {
	cfg    Config
	logger *log.Logger
	logf   *os.File
	mu     sync.Mutex
}

type Application struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type FileEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func New(cfg Config) (*App, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg, logger: log.New(f, "", 0), logf: f}, nil
}

func (a *App) Close() error {
	if a.logf != nil {
		return a.logf.Close()
	}
	return nil
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", a.health)
	mux.HandleFunc("/api/apps", a.apps)
	mux.HandleFunc("/api/apps/", a.appRoutes)
	mux.HandleFunc("/api/update/report", a.report)
	mux.HandleFunc("/api/update/logs", a.logs)
	return a.logging(mux)
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	info, err := os.Stat(a.cfg.SourcePath)
	sourceExists := err == nil && info.IsDir()
	sourceError := ""
	if err != nil {
		sourceError = err.Error()
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"status":        "ok",
		"source_path":   a.cfg.SourcePath,
		"source_exists": sourceExists,
		"source_error":  sourceError,
	}})
}

func (a *App) apps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	apps, err := a.scanApps()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "APP_SCAN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "apps": apps, "data": map[string]any{"apps": apps}})
}

func (a *App) appRoutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/apps/")
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 || parts[1] != "files" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	app, err := url.PathUnescape(parts[0])
	if err != nil || ValidateAppName(app) != nil {
		writeError(w, http.StatusBadRequest, "INVALID_APPLICATION", "invalid application")
		return
	}
	app, _ = NormalizeAppName(app)
	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		files, total, err := a.listFiles(app)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			writeError(w, status, "FILE_LIST_FAILED", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"application": app, "files": files, "total_bytes": total}})
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	rel, err := url.PathUnescape(parts[2])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "invalid file path")
		return
	}
	a.downloadFile(w, r, app, rel)
}

func (a *App) report(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON")
		return
	}
	delete(payload, "token")
	delete(payload, "api_token")
	a.writeLog("client_report", payload)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"accepted": true}})
}

func (a *App) logs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	http.ServeFile(w, r, a.cfg.LogPath)
}

func (a *App) scanApps() ([]Application, error) {
	entries, err := os.ReadDir(a.cfg.SourcePath)
	if err != nil {
		return nil, err
	}
	apps := make([]Application, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if ValidateAppName(name) != nil {
			continue
		}
		apps = append(apps, Application{Name: name, Path: name})
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	return apps, nil
}

func (a *App) listFiles(app string) ([]FileEntry, int64, error) {
	root, resolvedApp, err := a.resolveAppRoot(app)
	if err != nil {
		return nil, 0, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("%s is not a directory", resolvedApp)
	}
	files := []FileEntry{}
	var total int64
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files = append(files, FileEntry{Path: rel, Size: info.Size()})
		total += info.Size()
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, total, err
}

func (a *App) downloadFile(w http.ResponseWriter, r *http.Request, app, rel string) {
	clean, err := CleanRelativePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", err.Error())
		return
	}
	_, resolvedApp, err := a.resolveAppRoot(app)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeError(w, status, "APPLICATION_NOT_FOUND", err.Error())
		return
	}
	full, err := safeJoin(a.cfg.SourcePath, resolvedApp, clean)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", err.Error())
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeError(w, status, "FILE_NOT_FOUND", err.Error())
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "requested path is a directory")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(clean)))
	w.Header().Set("X-File-Path", clean)
	http.ServeFile(w, r, full)
}

func (a *App) resolveAppRoot(app string) (string, string, error) {
	app, err := NormalizeAppName(app)
	if err != nil {
		return "", "", err
	}
	root, err := safeJoin(a.cfg.SourcePath, app)
	if err == nil {
		if info, statErr := os.Stat(root); statErr == nil && info.IsDir() {
			return root, app, nil
		}
	}
	entries, readErr := os.ReadDir(a.cfg.SourcePath)
	if readErr != nil {
		return root, app, readErr
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.EqualFold(entry.Name(), app) {
			resolvedRoot, joinErr := safeJoin(a.cfg.SourcePath, entry.Name())
			if joinErr != nil {
				return "", "", joinErr
			}
			return resolvedRoot, entry.Name(), nil
		}
	}
	return root, app, os.ErrNotExist
}

func (a *App) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		a.writeLog("request", map[string]any{
			"method":     r.Method,
			"path":       r.URL.Path,
			"remote_ip":  r.RemoteAddr,
			"durationms": time.Since(start).Milliseconds(),
		})
	})
}

func (a *App) writeLog(event string, fields map[string]any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	row := map[string]any{"time": time.Now().Format(time.RFC3339), "event": event}
	for k, v := range fields {
		row[k] = v
	}
	b, _ := json.Marshal(row)
	a.logger.Println(string(b))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"success": false, "error": map[string]string{"code": code, "message": message}})
}
