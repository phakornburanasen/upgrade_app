package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"upgrade_app/internal/server"
)

func main() {
	configPath := flag.String("config", "config/server.json", "path to server config")
	flag.Parse()

	cfg, err := server.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := server.New(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}
	defer app.Close()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("update server listening on %s, source=%s", addr, cfg.SourcePath)
	if err := http.ListenAndServe(addr, app.Router()); err != nil {
		log.Fatal(err)
	}
}
