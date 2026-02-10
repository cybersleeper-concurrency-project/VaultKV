package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"vault-kv/internal/server"
	"vault-kv/internal/store"
)

func main() {
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	kvStore := store.NewStore()
	srv := server.NewServer(kvStore)
	mux := http.NewServeMux()

	mux.Handle("/set", server.LoggingMiddleware(http.HandlerFunc(srv.HandleSet)))
	mux.Handle("/get", server.LoggingMiddleware(http.HandlerFunc(srv.HandleGet)))

	addr := ":" + *port
	fmt.Printf("VaultKV Node running on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
