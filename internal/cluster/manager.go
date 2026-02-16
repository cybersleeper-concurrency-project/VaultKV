package cluster

import (
	"hash/crc32"
	"slices"
	"sort"
	"strconv"

	"vault-kv/internal/config"
)

func Init(cfg *config.Config) Cluster {
	clusterBuilder := NewClusterBuilder()
	clusterBuilder.SetReplicas(50)
	clusterBuilder.SetNodes(cfg.Cluster.Nodes)
	clusterBuilder.SetHttpClient(cfg.HTTPClient)
	clusterBuilder.SetServerTimeout(cfg.Server.Timeout)

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
