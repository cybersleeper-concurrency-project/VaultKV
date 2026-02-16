package cluster

import (
	"net/http"
	"time"
	"vault-kv/internal/config"
)

type ClusterBuilder struct {
	nodes           []string
	httpClient      config.HttpClient
	replicas        int
	serverTimeout   time.Duration
	idleConnTimeout time.Duration
}

type Cluster struct {
	Nodes  []string
	Client *http.Client
	ring   *ConsistentHash
}

func NewClusterBuilder() *ClusterBuilder {
	return &ClusterBuilder{}
}

func (c *ClusterBuilder) SetNodes(nodes []string) *ClusterBuilder {
	c.nodes = nodes
	return c
}

func (c *ClusterBuilder) SetHttpClient(httpClient config.HttpClient) *ClusterBuilder {
	c.httpClient = httpClient
	return c
}

func (c *ClusterBuilder) SetReplicas(replicas int) *ClusterBuilder {
	c.replicas = replicas
	return c
}

func (c *ClusterBuilder) SetServerTimeout(serverTimeout time.Duration) *ClusterBuilder {
	c.serverTimeout = serverTimeout
	return c
}

func (c *ClusterBuilder) SetIdleConnTimeout(idleTimeout time.Duration) *ClusterBuilder {
	c.idleConnTimeout = idleTimeout
	return c
}

func (c *ClusterBuilder) Build() Cluster {
	cluster := Cluster{}

	if len(c.nodes) > 0 {
		ring := NewConsistentHash(c.replicas)
		for _, n := range c.nodes {
			ring.AddNode(n)
		}
		cluster.ring = ring
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        c.httpClient.MaxIdleConns,
			MaxIdleConnsPerHost: c.httpClient.MaxIdleConnsPerHost,
			MaxConnsPerHost:     c.httpClient.MaxConnsPerHost,
			IdleConnTimeout:     c.idleConnTimeout,
		},
		Timeout: c.serverTimeout,
	}

	cluster.Client = client
	return cluster
}

func (c *Cluster) GetNode(key string) string {
	return c.ring.GetNode(key)
}

func (c *Cluster) GetClient() *http.Client {
	return c.Client
}
