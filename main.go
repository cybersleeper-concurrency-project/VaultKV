package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Store struct {
	data map[string]string
	mu   sync.RWMutex
}

type Response struct {
	Message string `json:"message,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
}

type KeyValueRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s took %dms", r.Method, r.URL.Path, time.Since(start).Milliseconds())

	})
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data[key]
	return val, exists
}

func (s *Store) handleSet(w http.ResponseWriter, r *http.Request) {
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

	s.Set(reqBody.Key, reqBody.Value)

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

func (s *Store) handleGet(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	value, exists := s.Get(key)

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

func main() {
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	store := NewStore()
	mux := http.NewServeMux()

	mux.Handle("/set", LoggingMiddleware(http.HandlerFunc(store.handleSet)))
	mux.Handle("/get", LoggingMiddleware(http.HandlerFunc(store.handleGet)))

	addr := ":" + *port
	fmt.Printf("VaultKV Node running on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
