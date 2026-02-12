package cluster

import (
	"log"
	"net/http"
	"time"

	"vault-kv/internal/config"
)

var (
	Nodes  []string
	Client *http.Client
)

func Init(cfg *config.Config) {
	Nodes = cfg.Cluster.Nodes

	timeout, err := time.ParseDuration(cfg.HTTPClient.IdleConnTimeout)
	if err != nil {
		log.Printf("Invalid IdleConnTimeout, defaulting to 90s: %v", err)
		timeout = 90 * time.Second
	}

	Client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HTTPClient.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.HTTPClient.MaxIdleConnsPerHost,
			IdleConnTimeout:     timeout,
		},
		Timeout: 5 * time.Second,
	}

	if cfg.Server.Timeout != "" {
		if t, err := time.ParseDuration(cfg.Server.Timeout); err == nil {
			Client.Timeout = t
		}
	}
}
