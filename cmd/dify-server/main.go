package main

import (
	"log"
	"net/http"

	"github.com/langgenius/dify-go/internal/config"
	"github.com/langgenius/dify-go/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	handler, err := server.New(cfg)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	log.Printf("dify-go listening on %s", cfg.Addr)
	if cfg.LegacyAPIBaseURL != "" {
		log.Printf("legacy api fallback enabled: %s", cfg.LegacyAPIBaseURL)
	}

	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
