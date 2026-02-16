package cluster

import "net/http"

type ClusterInterface interface {
	GetNode(key string) string
	GetClient() *http.Client
}
