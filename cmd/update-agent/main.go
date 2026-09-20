package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"upgrade_app/internal/agent"
)

func main() {
	configPath := flag.String("config", "config/agent.json", "path to agent config")
	openUI := flag.Bool("open", true, "open the update UI in the default browser")
	flag.Parse()

	cfg, err := agent.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app := agent.New(cfg)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	uiURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if *openUI {
			_ = openBrowser(uiURL)
		}
		log.Printf("update agent is not listening on %s: %v", addr, err)
		log.Printf("opened existing UI if another agent is already running: %s", uiURL)
		return
	}

	if *openUI {
		go func() {
			time.Sleep(700 * time.Millisecond)
			if err := openBrowser(uiURL); err != nil {
				log.Printf("open browser: %v", err)
			}
		}()
	}
	log.Printf("update agent listening on http://%s, ui=%s, server=%s, install=%s", addr, uiURL, cfg.ServerURL, cfg.InstallPath)
	if err := http.Serve(ln, app.Router()); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
