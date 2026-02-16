package cluster

import (
	"sync"
)

type ConsistentHash struct {
	replicas int
	keys     []uint32
	hashMap  map[uint32]string
	mu       sync.RWMutex
}

func NewConsistentHash(replicas int) *ConsistentHash {
	return &ConsistentHash{
		replicas: replicas,
		hashMap:  make(map[uint32]string),
	}
}
