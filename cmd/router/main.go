package main

import (
	"fmt"
	"log"
	"net/http"

	"vault-kv/internal/cluster"
	"vault-kv/internal/config"
	"vault-kv/internal/proxy"
)

func main() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	c := cluster.Init(cfg)

	ph := &proxy.ProxyHandler{
		Cluster: &c,
	}

	http.Handle("/", ph)

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	fmt.Printf("VaultKV router running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
