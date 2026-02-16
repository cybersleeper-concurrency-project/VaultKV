package cluster

import (
	"log"
	"net/http"
	"time"
	"vault-kv/internal/config"
)

type ClusterBuilder struct {
	Nodes         []string
	HttpClient    config.HttpClient
	Replicas      int
	ServerTimeout string
}

type Cluster struct {
	Nodes  []string
	Client *http.Client
	Ring   *ConsistentHash
}

func NewClusterBuilder() *ClusterBuilder {
	return &ClusterBuilder{}
}

func (c *ClusterBuilder) SetNodes(nodes []string) *ClusterBuilder {
	c.Nodes = nodes
	return c
}

func (c *ClusterBuilder) SetHttpClient(httpClient config.HttpClient) *ClusterBuilder {
	c.HttpClient = httpClient
	return c
}

func (c *ClusterBuilder) SetReplicas(replicas int) *ClusterBuilder {
	c.Replicas = replicas
	return c
}

func (c *ClusterBuilder) SetServerTimeout(serverTimeout string) *ClusterBuilder {
	c.ServerTimeout = serverTimeout
	return c
}

func (c *ClusterBuilder) Build() Cluster {
	cluster := Cluster{}

	if len(c.Nodes) > 0 {
		ring := NewConsistentHash(c.Replicas)
		for _, n := range c.Nodes {
			ring.AddNode(n)
		}
		cluster.Ring = ring
	}

	timeout, err := time.ParseDuration(c.HttpClient.IdleConnTimeout)
	if err != nil {
		log.Printf("Invalid IdleConnTimeout, defaulting to 90s: %v", err)
		timeout = 90 * time.Second
	}

	Client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        c.HttpClient.MaxIdleConns,
			MaxIdleConnsPerHost: c.HttpClient.MaxIdleConnsPerHost,
			MaxConnsPerHost:     c.HttpClient.MaxConnsPerHost,
			IdleConnTimeout:     timeout,
		},
		Timeout: 5 * time.Second,
	}

	if c.ServerTimeout != "" {
		if t, err := time.ParseDuration(c.ServerTimeout); err == nil {
			Client.Timeout = t
		}
	}

	cluster.Client = Client

	return cluster
}

func (c *Cluster) GetNode(key string) string {
	return c.Ring.GetNode(key)
}
