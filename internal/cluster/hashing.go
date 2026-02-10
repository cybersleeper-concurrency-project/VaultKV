package cluster

import "hash/fnv"

func HashKey(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(Nodes)
}
