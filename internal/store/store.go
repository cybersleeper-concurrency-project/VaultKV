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
	sst  *SSTable
}

func NewStore(nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	walFilename := "vault_" + nodeID + ".wal"
	sstFilename := nodeID + ".sst"

	data := NewSkiplist()

	sst, err := NewSSTable(sstFilename)
	if err != nil {
		return nil, fmt.Errorf("initializing SSTable: %w", err)
	}

	wal, err := NewWAL(walFilename)
	if err != nil {
		return nil, fmt.Errorf("initializing WAL: %w", err)
	}

	entries, err := wal.ReadAll()
	if err != nil {
		wal.Close()
		return nil, fmt.Errorf("reading WAL entries: %w", err)
	}

	for _, v := range entries {
		if v.Value == tombstone {
			data.Delete(v.Key)
		} else {
			data.Set(v.Key, v.Value)

			if data.Size >= skiplistCapacity {
				err := sst.Flush(data)
				if err != nil {
					return nil, err
				}

				data = NewSkiplist()
			}
		}
	}

	return &Store{
		data: data,
		wal:  wal,
		sst:  sst,
	}, nil
}

func (s *Store) Set(key, value string) error {
	if s.wal == nil {
		return errors.New("WAL is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.wal.Append(&LogEntry{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return err
	}

	s.data.Set(key, value)

	if s.data.Size >= skiplistCapacity {
		err := s.sst.Flush(s.data)
		if err != nil {
			return err
		}

		s.data = NewSkiplist()
	}

	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data.Get(key)
	return val, exists
}

func (s *Store) Delete(key string) error {
	if s.wal == nil {
		return errors.New("WAL is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.wal.Append(&LogEntry{
		Key: key,
	})
	if err != nil {
		return err
	}

	s.data.Delete(key)
	return nil
}
