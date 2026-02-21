package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"

	"vault-kv/internal/config"
	"vault-kv/internal/logger"
	"vault-kv/internal/server"
	"vault-kv/internal/store"
)

func main() {
	cfg, err := config.LoadConfig("NODE")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.LogLevel)

	port := flag.String("port", "8080", "server port")
	flag.Parse()

	kvStore := store.NewStore()
	srv := server.NewServer(kvStore)
	mux := http.NewServeMux()

	mux.Handle("/set", server.LoggingMiddleware(http.HandlerFunc(srv.HandleSet)))
	mux.Handle("/get", server.LoggingMiddleware(http.HandlerFunc(srv.HandleGet)))

	addr := ":" + *port
	slog.Info("VaultKV Node running", "addr", addr, "log_level", cfg.LogLevel)
	log.Fatal(http.ListenAndServe(addr, mux))
}
