package store

import (
	"math/rand/v2"
)

const (
	// Probability for each level increment on the new node
	Probability = 0.25
	MaxLevel    = 16
)

type Node struct {
	Key   string
	Value string
	Next  []*Node
	Level int
}

type Skiplist struct {
	BeginNode *Node
}

func NewNode() (*Node, error) {
	return &Node{
		Next: make([]*Node, MaxLevel+1),
	}, nil
}

func NewSkiplist() (*Skiplist, error) {
	beginNode, _ := NewNode()
	for i := range MaxLevel + 1 {
		beginNode.Next[i] = nil
	}

	return &Skiplist{
		BeginNode: beginNode,
	}, nil
}

func randomLevel() int {
	lvl := 0
	for rand.Float32() < Probability && lvl < MaxLevel {
		lvl++
	}
	return lvl
}

func (s *Skiplist) insert(befNode [MaxLevel + 1]*Node, k, v string) error {
	nxtNode := befNode[0].Next[0]
	curNode, _ := NewNode()
	curNode.Key = k
	curNode.Value = v
	curNode.Level = randomLevel()

	for i := range curNode.Level + 1 {
		befNode[i].Next[i] = curNode
	}
	curLvl := 0
	for curLvl <= curNode.Level {
		for curLvl > nxtNode.Level {
			nxtNode = nxtNode.Next[curLvl-1]
		}
		curNode.Next[curLvl] = nxtNode
		curLvl++
	}
	return nil
}

func (s *Skiplist) getUpdatePath(k string) [MaxLevel + 1]*Node {
	// Store the last visited node for each level which key is
	// STRICTLY less than k
	var lastNodes [MaxLevel + 1]*Node
	curNode := s.BeginNode

	for i := MaxLevel; i >= 0; i-- {
		for curNode.Next[i] != nil && curNode.Next[i].Key < k {
			curNode = curNode.Next[i]
		}
		lastNodes[i] = curNode
	}
	return lastNodes
}

func (s *Skiplist) Set(k, v string) {
	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].Next[0]

	if candidate != nil && candidate.Key == k {
		candidate.Value = v
	}
	s.insert(lastNodes, k, v)
}

func (s *Skiplist) Get(k string) string {
	lastNodes := s.getUpdatePath(k)
	candidate := lastNodes[0].Next[0]

	if candidate != nil && candidate.Key == k {
		return candidate.Value
	}
	return ""
}
