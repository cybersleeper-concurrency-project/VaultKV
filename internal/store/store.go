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
	dir        string
	nodeId     string
	data       *Skiplist
	mu         sync.RWMutex
	wal        *WAL
	OnFlushErr func(error) // Callback for background flush errors
}

func NewStore(dir, nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	data := NewSkiplist()

	pattern := filepath.Join(dir, fmt.Sprintf("vault_%s_*.wal", nodeID))
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

	wal, err := NewWAL(filepath.Join(dir, newWalName))
	if err != nil {
		return nil, fmt.Errorf("initializing WAL: %w", err)
	}

	return &Store{
		dir:    dir,
		nodeId: nodeID,
		data:   data,
		wal:    wal,
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
		if err := s.flushMemTable(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// TODO: query in this exact order:
	// 1. Active MemTable
	// 2. Frozen MemTables in order from newest to oldest
	// 3. SSTables on disk in order from newest to oldest
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
		if err := s.flushMemTable(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) flushMemTable() error {

	timestamp := time.Now().UnixNano()
	newWalName := fmt.Sprintf("vault_%s_%d.wal", s.nodeId, timestamp)
	newSstName := fmt.Sprintf("vault_%s_%d.sst", s.nodeId, timestamp)

	newWal, err := NewWAL(filepath.Join(s.dir, newWalName))
	if err != nil {
		return fmt.Errorf("failed to create new WAL: %w", err)
	}

	frozenData := s.data
	frozenWal := s.wal

	s.data = NewSkiplist()
	s.wal = newWal

	// TODO: Add flush tracking (e.g., sync.WaitGroup) to signal completion and allow
	// graceful shutdown. Also implement a recovery path (e.g., retry queue or persistent state)
	// so that frozenData is not permanently lost if the background flush fails.
	go func(dataToFlush *Skiplist, oldWal *WAL, sstFilename string) {
		handleErr := func(err error) {
			if s.OnFlushErr != nil {
				s.OnFlushErr(err)
			} else {
				fmt.Printf("Background flush error: %v\n", err)
			}
		}

		newSst, err := NewSSTable(filepath.Join(s.dir, sstFilename))
		if err != nil {
			handleErr(fmt.Errorf("failed to create new SSTable: %w", err))
			return
		}

		err = newSst.Flush(dataToFlush)
		// Close the file descriptor so we don't leak memory (since we haven't
		// built the logic to query from it yet)
		newSst.Close()
		if err != nil {
			handleErr(fmt.Errorf("failed to flush SSTable: %w", err))
			return
		}

		if err := oldWal.Delete(); err != nil {
			handleErr(fmt.Errorf("failed to delete obsolete WAL: %w", err))
		}
	}(frozenData, frozenWal, newSstName)

	return nil
}
