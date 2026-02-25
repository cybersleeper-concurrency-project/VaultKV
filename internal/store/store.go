package store

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
)

var validNodeID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Engine interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

type Store struct {
	data map[string]string
	mu   sync.RWMutex
	wal  *WAL
}

func NewStore(nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	data := make(map[string]string)

	filename := "vault_" + nodeID + ".wal"

	wal, err := NewWAL(filename)
	if err != nil {
		return nil, fmt.Errorf("initializing WAL: %w", err)
	}

	entries, err := wal.ReadAll()
	if err != nil {
		wal.Close()
		return nil, fmt.Errorf("reading WAL entries: %w", err)
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
		wal:  wal,
	}, nil
}

func (s *Store) Set(key, value string) error {
	if s.wal == nil {
		return errors.New("WAL is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.wal.Append(&LogEntry{
		Type:  RecordTypePut,
		Key:   key,
		Value: value,
	})
	if err != nil {
		return err
	}

	s.data[key] = value
	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data[key]
	return val, exists
}
