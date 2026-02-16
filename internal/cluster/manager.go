package cluster

import (
	"hash/crc32"
	"log"
	"slices"
	"sort"
	"strconv"
	"time"

	"vault-kv/internal/config"
)

func Init(cfg *config.Config) Cluster {
	clusterBuilder := NewClusterBuilder()

	idleTimeout, err := time.ParseDuration(cfg.HTTPClient.IdleConnTimeout)
	if err != nil {
		log.Printf("Invalid IdleConnTimeout, defaulting to 90s: %v", err)
		idleTimeout = 90 * time.Second
	}
	clusterBuilder.SetIdleConnTimeout(idleTimeout)

	if cfg.Cluster.Replicas == 0 {
		cfg.Cluster.Replicas = 50
	}
	clusterBuilder.SetReplicas(cfg.Cluster.Replicas)

	serverTimeout := 5 * time.Second
	if cfg.Server.Timeout != "" {
		if t, err := time.ParseDuration(cfg.Server.Timeout); err == nil {
			serverTimeout = t
		}
	}
	clusterBuilder.SetServerTimeout(serverTimeout)

	clusterBuilder.SetNodes(cfg.Cluster.Nodes)
	clusterBuilder.SetHttpClient(cfg.HTTPClient)

	return clusterBuilder.Build()
}

func (c *ConsistentHash) AddNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for idx := range c.replicas {
		strIdx := strconv.Itoa(idx)
		vKey := node + "#" + strIdx

		hash := crc32.ChecksumIEEE([]byte(vKey))
		c.keys = append(c.keys, hash)
		c.hashMap[hash] = node
	}

	slices.Sort(c.keys)
}

func (c *ConsistentHash) GetNode(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.keys) == 0 {
		return ""
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	nodeIdx := sort.Search(len(c.keys), func(i int) bool {
		return c.keys[i] >= hash
	})

	if nodeIdx == len(c.keys) {
		nodeIdx = 0
	}

	return c.hashMap[c.keys[nodeIdx]]
}
