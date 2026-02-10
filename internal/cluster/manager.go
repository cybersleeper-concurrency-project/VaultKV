package cluster

import (
	"net/http"
	"time"
)

var Nodes = []string{
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
}

var Client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 5 * time.Second,
}
