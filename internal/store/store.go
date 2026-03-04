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
	data *Skiplist
	mu   sync.RWMutex
	wal  *WAL
}

func NewStore(nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	data := NewSkiplist()

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
			data.Set(v.Key, v.Value)
		}
		if v.Type == RecordTypeDelete {
			data.Delete(v.Key)
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

	s.data.Set(key, value)
	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data.Get(key)
	return val, exists
}
