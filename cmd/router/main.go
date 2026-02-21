package main

import (
	"log"
	"net/http"

	"vault-kv/internal/cluster"
	"vault-kv/internal/config"
	"vault-kv/internal/logger"
	"vault-kv/internal/proxy"

	"log/slog"
)

func main() {
	cfg, err := config.LoadConfig("ROUTER")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.LogLevel)
	c := cluster.Init(cfg)

	ph := &proxy.ProxyHandler{
		Cluster: &c,
	}

	http.Handle("/", ph)

	port := cfg.ServerPort
	if port == "" {
		port = ":8080"
	}

	slog.Info("VaultKV router running", "port", port, "log_level", cfg.LogLevel)
	log.Fatal(http.ListenAndServe(port, nil))
}
