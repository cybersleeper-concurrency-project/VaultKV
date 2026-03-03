package store

import (
	"math/rand/v2"
	"sync"
)

const (
	// probability for each level increment on the new node
	probability = 0.25
	maxLevel    = 16
)

type Node struct {
	key   string
	value string
	next  []*Node
	level int
}

type Skiplist struct {
	BeginNode *Node
	mu        sync.RWMutex
}

func NewNode() *Node {
	return &Node{
		next: make([]*Node, maxLevel+1),
	}
}

func NewSkiplist() *Skiplist {
	beginNode := NewNode()

	return &Skiplist{
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
	curNode.key = k
	curNode.value = v
	curNode.level = randomLevel()

	for i := range curNode.level + 1 {
		curNode.next[i] = befNode[i].next[i]
		befNode[i].next[i] = curNode
	}
}

func (s *Skiplist) getUpdatePath(k string) [maxLevel + 1]*Node {
	// Store the last visited node for each level which key is
	// STRICTLY less than k
	var lastNodes [maxLevel + 1]*Node
	curNode := s.BeginNode

	for i := maxLevel; i >= 0; i-- {
		for curNode.next[i] != nil && curNode.next[i].key < k {
			curNode = curNode.next[i]
		}
		lastNodes[i] = curNode
	}
	return lastNodes
}

func (s *Skiplist) Set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].next[0]

	if candidate != nil && candidate.key == k {
		candidate.value = v
	} else {
		s.insert(lastNodes, k, v)
	}
}

func (s *Skiplist) Get(k string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].next[0]

	if candidate != nil && candidate.key == k {
		return candidate.value, true
	}
	return "", false
}
