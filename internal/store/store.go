package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

const memTableSizeThreshold = 4 * 1024 * 1024 // 4MB

var validNodeID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Engine interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

type Store struct {
	nodeId string
	data   *Skiplist
	mu     sync.RWMutex
	wal    *WAL
	sst    *SSTable
}

func NewStore(nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	sstFilename := nodeID + ".sst"

	data := NewSkiplist()

	sst, err := NewSSTable(sstFilename)
	if err != nil {
		return nil, fmt.Errorf("initializing SSTable: %w", err)
	}

	pattern := fmt.Sprintf("vault_%s_*.wal", nodeID)
	walFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for old WALs: %w", err)
	}

	sort.Strings(walFiles)

	// Note: Currently if there exist several frozen WALs, they all gonna be
	// dumped into a single active MemTable. Ex: if there are three 4MB WALs,
	// the MemT will be 12MB. As per 30 April 2026 this is perfectly fine as
	// it will just trigger the IsFull() == true then get flushed as a massive
	// 12MB block. But still, we need to be cautious and maintain carefully

	for _, file := range walFiles {
		oldWal, err := NewWAL(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open old WAL %s: %w", file, err)
		}

		entries, err := oldWal.ReadAll()
		if err != nil {
			oldWal.Close()
			return nil, fmt.Errorf("corrupted WAL detected in %s: %w", file, err)
		}

		for _, v := range entries {
			if v.Value == tombstone {
				data.Delete(v.Key)
			} else {
				data.Set(v.Key, v.Value)
			}
		}
		oldWal.Close()
	}

	newWalName := fmt.Sprintf("vault_%s_%d.wal", nodeID, time.Now().UnixNano())

	wal, err := NewWAL(newWalName)
	if err != nil {
		return nil, fmt.Errorf("initializing WAL: %w", err)
	}

	return &Store{
		nodeId: nodeID,
		data:   data,
		wal:    wal,
		sst:    sst,
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

	if s.data.IsFull() {
		if err := s.FlushMemTable(); err != nil {
			return err
		}
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
		Key:   key,
		Value: tombstone,
	})
	if err != nil {
		return err
	}

	s.data.Delete(key)
	if s.data.IsFull() {
		if err := s.FlushMemTable(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) FlushMemTable() error {
	frozenData := s.data
	frozenWal := s.wal

	newWalName := fmt.Sprintf("vault_%s_%d.wal", s.nodeId, time.Now().UnixNano())

	s.data = NewSkiplist()
	var err error
	s.wal, err = NewWAL(newWalName)
	if err != nil {
		return fmt.Errorf("failed to create new WAL: %w", err)
	}

	go func(dataToFlush *Skiplist, oldWal *WAL) {
		err := s.sst.Flush(dataToFlush)
		if err != nil {
			fmt.Printf("Background flush failed: %v\n", err)
			return
		}

		oldWal.Delete()
	}(frozenData, frozenWal)

	return nil
}
