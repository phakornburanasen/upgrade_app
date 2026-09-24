package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	root, err := appRoot()
	if err != nil {
		messageBox("TNLX Update Agent", err.Error())
		os.Exit(1)
	}

	agentExe := filepath.Join(root, "bin", "client-agent.exe")
	if _, err := os.Stat(agentExe); err != nil {
		messageBox("TNLX Update Agent", fmt.Sprintf("Cannot find %s", agentExe))
		os.Exit(1)
	}

	configPath := filepath.Join(root, "config", "agent.enc")
	if _, err := os.Stat(configPath); err != nil {
		configPath = filepath.Join(root, "config", "agent.json")
	}

	cmd := exec.Command(agentExe, "-config", configPath)
	cmd.Dir = root
	if err := cmd.Start(); err != nil {
		messageBox("TNLX Update Agent", fmt.Sprintf("Start failed: %v", err))
		os.Exit(1)
	}
}

func appRoot() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exePath)
	if exists(filepath.Join(dir, "bin", "client-agent.exe")) {
		return dir, nil
	}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		parent := filepath.Dir(dir)
		if exists(filepath.Join(parent, "bin", "client-agent.exe")) {
			return parent, nil
		}
	}
	return dir, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
