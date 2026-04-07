package store

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

// BEGIN AI SECTION

// setupStore creates a completely isolated new Store instance using a temporary file path
func setupStore(t *testing.T, prefix string) (*Store, func()) {
	// Create a unique nodeID for the temporary file
	// Use t.TempDir() to guarantee cleanup of files when the test is done

	// // Create a unique nodeID for the temporary file
	nodeID := prefix + "_testnode"

	// Since NewStore automatically hardcodes the prefix "vault_", we must ensure we test in isolation
	// We'll temporarily override the directory by running the test from inside the tempdir,
	// OR we can just allow the store to create it in the local test folder if we clean it up later.
	// For testing, it's safer to build a specific init function, but we will wrap the NewStore call.

	s, err := NewStore(nodeID)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// Move the WAL file to our temp directory manually to avoid cluttering the project root
	// Since NewStore hardcodes the local path, we have to clean up the local directory
	cleanup := func() {
		os.Remove("vault_" + nodeID + ".wal")
	}

	return s, cleanup
}

func TestStore_SetGetDelete(t *testing.T) {
	s, cleanup := setupStore(t, "basic")
	defer cleanup()

	// Test Set
	if err := s.Set("hero", "batman"); err != nil {
		t.Fatalf("Expected nil err on Set, got: %v", err)
	}

	// Test Get
	val, ok := s.Get("hero")
	if !ok || val != "batman" {
		t.Errorf("Expected batman, got %s (ok: %v)", val, ok)
	}

	// Test Delete
	if err := s.Delete("hero"); err != nil {
		t.Fatalf("Expected nil err on Delete, got: %v", err)
	}

	val, ok = s.Get("hero")
	if ok || val != "" {
		t.Errorf("Expected hero to be deleted, got %s (ok: %v)", val, ok)
	}
}

func TestStore_RecoveryFromWAL(t *testing.T) {
	nodeID := "recovery_testnode"

	// 1. Initialize a generic store and write some data
	s1, err := NewStore(nodeID)
	if err != nil {
		t.Fatalf("failed to init first store: %v", err)
	}
	s1.Set("persisted_key", "survives_crash")
	s1.Set("deleted_key", "will_be_gone")
	s1.Delete("deleted_key")
	s1.wal.Close() // Simulate a crash / shutdown

	// 2. Start a brand new Store instance pointing to the exact same file
	s2, err := NewStore(nodeID)
	if err != nil {
		t.Fatalf("failed to init recovered store: %v", err)
	}
	defer func() {
		s2.wal.Close()
		os.Remove("vault_" + nodeID + ".wal")
	}()

	// 3. Verify the MemTable was accurately rebuilt from the WAL entries (Puts and Tombstones)
	val, ok := s2.Get("persisted_key")
	if !ok || val != "survives_crash" {
		t.Errorf("Expected 'survives_crash' from recovered store, got %s (ok: %v)", val, ok)
	}

	val, ok = s2.Get("deleted_key")
	if ok || val != "" {
		t.Errorf("Expected 'deleted_key' to still be a Tombstone after recovery, got %s (ok: %v)", val, ok)
	}
}

func TestStore_ConcurrentSetGet(t *testing.T) {
	s, cleanup := setupStore(t, "concurrent")
	defer cleanup()

	var wg sync.WaitGroup
	workers := 20

	// Blast the Store with concurrent WAL Appends + MemTable Sets
	for i := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("k-%d", id)
			val := fmt.Sprintf("v-%d", id)

			if err := s.Set(key, val); err != nil {
				t.Errorf("Concurrent set failed for worker %d: %v", id, err)
			}

			readVal, ok := s.Get(key)
			if !ok || readVal != val {
				t.Errorf("Concurrent get failed for worker %d. Expected %s, got %s (ok: %v)", id, val, readVal, ok)
			}
		}(i)
	}

	wg.Wait()
}

// END AI SECTION
