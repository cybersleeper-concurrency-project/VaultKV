package store

import (
	"math/rand/v2"
	"sync"
)

const (
	// probability for each level increment on the new node
	probability      = 0.25
	maxLevel         = 16
	tombstone        = "0:^_#TOMBSTONE#_^:0"
	skiplistCapacity = 100
)

type Node struct {
	Key   string
	Value string
	Next  []*Node
	level int
}

type Skiplist struct {
	BeginNode *Node
	Size      int
	mu        sync.RWMutex
}

func NewNode() *Node {
	return &Node{
		Next: make([]*Node, maxLevel+1),
	}
}

func NewSkiplist() *Skiplist {
	beginNode := NewNode()

	return &Skiplist{
		Size:      0,
		BeginNode: beginNode,
	}
}

func randomLevel() int {
	lvl := 0
	for rand.Float32() < probability && lvl < maxLevel {
		lvl++
	}
	return lvl
}

func (s *Skiplist) insert(befNode [maxLevel + 1]*Node, k, v string) {
	curNode := NewNode()
	curNode.Key = k
	curNode.Value = v
	curNode.level = randomLevel()

	for i := range curNode.level + 1 {
		curNode.Next[i] = befNode[i].Next[i]
		befNode[i].Next[i] = curNode
	}

	s.Size++
}

func (s *Skiplist) getUpdatePath(k string) [maxLevel + 1]*Node {
	// Store the last visited node for each level which key is
	// STRICTLY less than k
	var lastNodes [maxLevel + 1]*Node
	curNode := s.BeginNode

	for i := maxLevel; i >= 0; i-- {
		for curNode.Next[i] != nil && curNode.Next[i].Key < k {
			curNode = curNode.Next[i]
		}
		lastNodes[i] = curNode
	}
	return lastNodes
}

func (s *Skiplist) Set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].Next[0]

	if candidate != nil && candidate.Key == k {
		candidate.Value = v
	} else {
		s.insert(lastNodes, k, v)
	}
}

func (s *Skiplist) Get(k string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].Next[0]

	if candidate != nil && candidate.Key == k && candidate.Value != tombstone {
		return candidate.Value, true
	}
	return "", false
}

func (s *Skiplist) Delete(k string) {
	s.Set(k, tombstone)
}
