package server

import (
	"errors"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var appNamePattern = regexp.MustCompile(`^[A-Za-z0-9._ -]+$`)

func ValidateAppName(app string) error {
	_, err := NormalizeAppName(app)
	return err
}

func NormalizeAppName(app string) (string, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return "", errors.New("application is required")
	}
	if app == "." || app == ".." || strings.Contains(app, "..") {
		return "", errors.New("invalid application name")
	}
	if strings.ContainsAny(app, `/\:`) || strings.HasPrefix(app, `\\`) || filepath.IsAbs(app) {
		return "", errors.New("application must not be a path")
	}
	if !appNamePattern.MatchString(app) {
		return "", errors.New("application contains unsupported characters")
	}
	return app, nil
}

func CleanRelativePath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", errors.New("relative path is required")
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	if strings.Contains(rel, "\x00") || strings.Contains(rel, ":") {
		return "", errors.New("invalid relative path")
	}
	if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "//") || filepath.IsAbs(rel) {
		return "", errors.New("absolute paths are not allowed")
	}
	clean := path.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("path traversal is not allowed")
	}
	return clean, nil
}

func safeJoin(base, app string, rel ...string) (string, error) {
	app, err := NormalizeAppName(app)
	if err != nil {
		return "", err
	}
	parts := []string{base, app}
	for _, p := range rel {
		clean, err := CleanRelativePath(p)
		if err != nil {
			return "", err
		}
		parts = append(parts, filepath.FromSlash(clean))
	}
	full := filepath.Join(parts...)
	root := filepath.Join(base, app)
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if fullAbs != rootAbs && !strings.HasPrefix(strings.ToLower(fullAbs), strings.ToLower(rootAbs)+string(filepath.Separator)) {
		return "", errors.New("resolved path escaped application root")
	}
	return full, nil
}
