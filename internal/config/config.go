package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server     Server     `json:"server"`
	Cluster    Cluster    `json:"cluster"`
	HTTPClient HttpClient `json:"http_client"`
}

type Server struct {
	Port    string `json:"port"`
	Timeout string `json:"timeout"`
}

type Cluster struct {
	Nodes          []string `json:"nodes"`
	MaxConcurrency int      `json:"max_concurrency"`
	Replicas       int      `json:"replicas"`
}
type HttpClient struct {
	MaxIdleConns        int    `json:"max_idle_conns"`
	MaxIdleConnsPerHost int    `json:"max_idle_conns_per_host"`
	MaxConnsPerHost     int    `json:"max_conns_per_host"`
	IdleConnTimeout     string `json:"idle_conn_timeout"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	// Environment variable overrides
	if port := os.Getenv("VAULT_SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if timeout := os.Getenv("VAULT_SERVER_TIMEOUT"); timeout != "" {
		cfg.Server.Timeout = timeout
	}
	if nodes := os.Getenv("VAULT_CLUSTER_NODES"); nodes != "" {
		cfg.Cluster.Nodes = strings.Split(nodes, ",")
	}
	if val := os.Getenv("VAULT_CLUSTER_MAX_CONCURRENCY"); val != "" {
		maxConcurrency, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		cfg.Cluster.MaxConcurrency = maxConcurrency
	}

	return cfg, nil
}
