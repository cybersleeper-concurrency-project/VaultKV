package main

import (
	"fmt"
	"log"
	"net/http"

	"vault-kv/internal/proxy"
)

func main() {
	http.HandleFunc("/", proxy.HandleProxy)

	port := ":8080"
	fmt.Printf("VaultKV router running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
