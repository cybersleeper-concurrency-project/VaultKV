package config

import (
	"encoding/json"
	"os"
	"strings"
)

type Config struct {
	Server struct {
		Port    string `json:"port"`
		Timeout string `json:"timeout"`
	} `json:"server"`
	Cluster struct {
		Nodes          []string `json:"nodes"`
		MaxConcurrency int      `json:"max_concurrency"`
	} `json:"cluster"`
	HTTPClient struct {
		MaxIdleConns        int    `json:"max_idle_conns"`
		MaxIdleConnsPerHost int    `json:"max_idle_conns_per_host"`
		IdleConnTimeout     string `json:"idle_conn_timeout"`
	} `json:"http_client"`
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

	return cfg, nil
}
