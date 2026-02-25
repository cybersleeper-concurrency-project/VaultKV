package main

import (
	"fmt"
	"log"

	"vault-kv/internal/store"
)

func main() {
	s, err := store.NewStore("vault.wal")
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}

	if err := s.Set("direct", "write"); err != nil {
		log.Fatalf("Failed to write to store: %v", err)
	}

	val, ok := s.Get("direct")
	fmt.Printf("Get direct: %s (ok=%v)\n", val, ok)
}
