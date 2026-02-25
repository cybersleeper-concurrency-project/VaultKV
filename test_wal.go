package main

import (
	"fmt"
	"log"

	"vault-kv/internal/store"
)

func main() {
	s := store.NewStore("vault.wal")
	err := s.Set("direct", "write")
	if err != nil {
		log.Fatalf("Failed to write to store: %v", err)
	}

	val, ok := s.Get("direct")
	fmt.Printf("Get direct: %s (ok=%v)\n", val, ok)
}
