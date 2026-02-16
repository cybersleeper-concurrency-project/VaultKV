package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"vault-kv/internal/cluster"
)

type KeyCheck struct {
	Key string `json:"key"`
}

type ProxyHandler struct {
	Cluster cluster.ClusterInterface
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	targetNode := h.Cluster.GetNode(key)
	targetNodeUrl := targetNode + r.URL.RequestURI()

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

	resp, err := h.Cluster.GetClient().Do(proxyReq)
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
