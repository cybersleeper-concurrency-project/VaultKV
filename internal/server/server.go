package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"vault-kv/internal/store"
)

type Response struct {
	Message string `json:"message,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
}

type KeyValueRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Server struct {
	store store.Engine
}

func NewServer(s store.Engine) *Server {
	return &Server{store: s}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Request processed", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) HandleSet(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody KeyValueRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.Set(reqBody.Key, reqBody.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	slog.Debug("Handle Set Request", "key", reqBody.Key, "value", reqBody.Value)

	response := Response{
		Message: "Success!",
		Key:     reqBody.Key,
		Value:   reqBody.Value,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) HandleGet(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	value, exists := s.store.Get(key)

	w.Header().Set("Content-Type", "application/json")
	var response Response

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		slog.Warn("Key not found", "key", key)
		response = Response{
			Message: "Key not found!",
			Key:     key,
		}
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	slog.Debug("Handle Get", "key", key, "value", value)
	response = Response{
		Message: "Success!",
		Key:     key,
		Value:   value,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
