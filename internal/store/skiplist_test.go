package store

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestSkiplist_BasicGetSet(t *testing.T) {
	sl := NewSkiplist()

	// Test Set and Get for a single item
	sl.Set("key1", "value1")
	val, ok := sl.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("Expected to get 'value1', got '%s' (ok: %v)", val, ok)
	}

	// Test Get for a non-existent item
	val, ok = sl.Get("key2")
	if ok || val != "" {
		t.Errorf("Expected to not find 'key2', got '%s' (ok: %v)", val, ok)
	}

	// Test Overwrite
	sl.Set("key1", "value1_updated")
	val, ok = sl.Get("key1")
	if !ok || val != "value1_updated" {
		t.Errorf("Expected to get 'value1_updated', got '%s' (ok: %v)", val, ok)
	}
}

func TestSkiplist_Ordering(t *testing.T) {
	sl := NewSkiplist()

	// Insert keys out of alphabetical order
	sl.Set("c", "3")
	sl.Set("a", "1")
	sl.Set("d", "4")
	sl.Set("b", "2")

	// We must verify they are actually sorted by traversing Level 0
	expectedKeys := []string{"a", "b", "c", "d"}
	expectedVals := []string{"1", "2", "3", "4"}

	current := sl.beginNode.next[0]
	count := 0

	for current != nil {
		if count >= len(expectedKeys) {
			t.Fatalf("Found more nodes than expected. Extra key: %s", current.key)
		}

		if current.key != expectedKeys[count] {
			t.Errorf("Expected key %s at position %d, got %s", expectedKeys[count], count, current.key)
		}
		if current.value != expectedVals[count] {
			t.Errorf("Expected value %s at position %d, got %s", expectedVals[count], count, current.value)
		}

		current = current.next[0]
		count++
	}

	if count != len(expectedKeys) {
		t.Errorf("Expected traversal of %d nodes, but only found %d", len(expectedKeys), count)
	}
}

func TestSkiplist_Concurrency(t *testing.T) {
	sl := NewSkiplist()
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent Writes
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(gID int) {
			defer wg.Done()
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-%d", gID, j)
				val := fmt.Sprintf("val-%d", j)
				sl.Set(key, val)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent Reads
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(gID int) {
			defer wg.Done()
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-%d", gID, j)
				expectedVal := fmt.Sprintf("val-%d", j)

				val, ok := sl.Get(key)
				if !ok || val != expectedVal {
					t.Errorf("Concurrent Read Failed for %s. Expected %s, got %s (ok: %v)", key, expectedVal, val, ok)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestSkiplist_Concurrency_Overlapping(t *testing.T) {
	sl := NewSkiplist()
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Adding both Writers and Readers to the wait group at once
	wg.Add(numGoroutines * 2)

	for i := range numGoroutines {
		go func(gID int) {
			defer wg.Done()
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-%d", gID, j)
				sl.Set(key, "value")
			}
		}(i)

		go func(gID int) {
			defer wg.Done()
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-%d", gID, j)

				// We don't assert the exact value here because the writer
				// might not have inserted it yet.
				// We are strictly testing that reading DURING a write
				// doesn't cause a panic or a fatal memory race!
				sl.Get(key)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkSkiplist_Set(b *testing.B) {
	sl := NewSkiplist()

	for i := 0; b.Loop(); i++ {
		sl.Set(strconv.Itoa(i), "value")
	}
}

func BenchmarkSkiplist_Get(b *testing.B) {
	sl := NewSkiplist()

	// Pre-generate the exact 10,000 keys we will use, this will prevent noises
	// like garbage collection and strconv for our benchmark
	const numItems = 10000
	keys := make([]string, numItems)

	for i := range numItems {
		keys[i] = strconv.Itoa(i)
		sl.Set(keys[i], "value")
	}

	// Loop resets the benchmark timer the first time it is called in a benchmark
	for i := 0; b.Loop(); i++ {
		sl.Get(keys[i%10000])
	}
}
