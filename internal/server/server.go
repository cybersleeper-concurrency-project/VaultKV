package server

import (
	"encoding/json"
	"log"
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
	store *store.Store
}

func NewServer(s *store.Store) *Server {
	return &Server{store: s}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s took %dms", r.Method, r.URL.Path, time.Since(start).Milliseconds())
	})
}

func (s *Server) HandleSet(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody KeyValueRequest
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.store.Set(reqBody.Key, reqBody.Value)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	log.Printf("key: %s | value: %s", reqBody.Key, reqBody.Value)

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
		log.Printf("key: %s not found", key)
		response = Response{
			Message: "Key not found!",
			Key:     key,
		}
	} else {
		w.WriteHeader(http.StatusOK)
		log.Printf("key: %s | value: %s", key, value)
		response = Response{
			Message: "Success!",
			Key:     key,
			Value:   value,
		}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
