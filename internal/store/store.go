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

type flushTask struct {
	data    *Skiplist
	wal     *WAL
	sstName string
}

type Store struct {
	dir         string
	nodeId      string
	data        *Skiplist
	frozenMemTs []*Skiplist
	sstables    []*SSTable
	flushChan   chan *flushTask
	mu          sync.RWMutex
	wal         *WAL
	OnFlushErr  func(error) // Callback for background flush errors
}

func NewStore(dir, nodeID string) (*Store, error) {
	if !validNodeID.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid nodeID: %q", nodeID)
	}

	data := NewSkiplist()

	// Load existing SSTs
	sstPattern := filepath.Join(dir, fmt.Sprintf("vault_%s_*.sst", nodeID))
	sstFiles, err := filepath.Glob(sstPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for old SSTs: %w", err)
	}

	existingSstables := make([]*SSTable, len(sstFiles))

	sort.Strings(sstFiles)

	for i, file := range sstFiles {
		curSst, err := NewSSTable(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open old SST %s: %w", file, err)
		}

		if err := curSst.LoadIndexBlock(); err != nil {
			return nil, fmt.Errorf("corrupted SST detected in %s: %w", file, err)
		}

		existingSstables[i] = curSst
	}

	// Load data from the existing WALs
	walPattern := filepath.Join(dir, fmt.Sprintf("vault_%s_*.wal", nodeID))
	walFiles, err := filepath.Glob(walPattern)
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

	storeObj := &Store{
		dir:         dir,
		nodeId:      nodeID,
		data:        data,
		frozenMemTs: make([]*Skiplist, 0),
		sstables:    existingSstables,
		flushChan:   make(chan *flushTask, 10),
		wal:         wal,
	}

	go storeObj.flushWorker()

	return storeObj, nil
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

	// 1. Check Active MemTable
	if val, exists := s.data.Get(key); exists {
		return val, true
	}

	// 2. Check Frozen MemTables (newest to oldest)
	for i := len(s.frozenMemTs) - 1; i >= 0; i-- {
		if val, exists := s.frozenMemTs[i].Get(key); exists {
			return val, true
		}
	}

	// 3. SSTables on disk (newest to oldest)
	pattern := filepath.Join(s.dir, fmt.Sprintf("vault_%s_*.sst", s.nodeId))
	sstFiles, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Errorf("failed to scan for SST: %w", err)
		return "", false
	}

	sort.Strings(sstFiles)

	for i := len(s.sstables) - 1; i >= 0; i-- {
		// TODO: binser
	}

	return "", false
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

	// Freeze the memtable so it can still be queried during the flush
	s.frozenMemTs = append(s.frozenMemTs, frozenData)
	s.data = NewSkiplist()
	s.wal = newWal

	// Push to the background worker
	s.flushChan <- &flushTask{
		data:    frozenData,
		wal:     frozenWal,
		sstName: newSstName,
	}

	return nil
}

func (s *Store) flushWorker() {
	for task := range s.flushChan {
		handleErr := func(err error) {
			if s.OnFlushErr != nil {
				s.OnFlushErr(err)
			} else {
				fmt.Printf("Background flush error: %v\n", err)
			}
		}

		newSst, err := NewSSTable(filepath.Join(s.dir, task.sstName))
		if err != nil {
			handleErr(fmt.Errorf("failed to create new SSTable: %w", err))
			continue
		}

		err = newSst.Flush(task.data)
		newSst.Close()
		if err != nil {
			handleErr(fmt.Errorf("failed to flush SSTable: %w", err))
			continue
		}

		if err := task.wal.Delete(); err != nil {
			handleErr(fmt.Errorf("failed to delete obsolete WAL: %w", err))
		}

		// Success, then remove the flushed Skiplist from the front of frozenMemTs
		// We must lock here as we are about to "write" (deleting a Frozen MemT)
		s.mu.Lock()
		s.frozenMemTs[0] = nil // Avoid memory leak, tell GC to sweep it
		s.frozenMemTs = s.frozenMemTs[1:]
		s.sstables = append(s.sstables, newSst)
		s.mu.Unlock()
	}
}
