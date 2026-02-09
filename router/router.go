package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"
)

var nodes = []string{
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
}

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 5 * time.Second,
}

func hashKey(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(nodes)
}

type KeyCheck struct {
	Key string `json:"key"`
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	var key string
	var bodyLen int64

	switch r.Method {
	case http.MethodPost:
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		var check KeyCheck
		if err := json.Unmarshal(bodyBytes, &check); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		key = check.Key
		bodyLen = int64(len(bodyBytes))

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	case http.MethodGet:
		key = r.URL.Query().Get("key")
	}

	if key == "" {
		http.Error(w, "Key is missing", http.StatusBadRequest)
		return
	}

	targetNodeIdx := hashKey(key)
	targetNodeUrl := nodes[targetNodeIdx] + r.URL.RequestURI()

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetNodeUrl, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		proxyReq.ContentLength = bodyLen
	}

	for name, values := range r.Header {
		for _, v := range values {
			proxyReq.Header.Add(name, v)
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Node Down: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(name, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/", handleProxy)

	port := ":8080"
	fmt.Printf("VaultKV router running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
