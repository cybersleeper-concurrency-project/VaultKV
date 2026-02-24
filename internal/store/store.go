package store

import (
	"log/slog"
	"sync"
)

type Engine interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

type Store struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewStore() *Store {
	data := make(map[string]string)

	wal, err := NewWAL("vault.wal")
	if err != nil {
		slog.Warn("Error when initializing the WAL", "err", err)
		return &Store{
			data: data,
		}
	}

	entries, err := wal.ReadAll()
	if err != nil {
		slog.Warn("Error when reading WAL entries", "err", err)
		return &Store{
			data: data,
		}
	}

	for _, v := range entries {
		if v.Type == RecordTypePut {
			data[v.Key] = v.Value
		}
		if v.Type == RecordTypeDelete {
			delete(data, v.Key)
		}
	}

	return &Store{
		data: data,
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
