package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const BASE_URL_SET = "http://localhost:8080/set"

func main() {
	var wg sync.WaitGroup

	for i := range 1000 {
		wg.Go(func() {
			body := fmt.Sprintf(`{"key": "user:%d", "value": "CrashTest"}`, i)

			resp, err := http.Post(BASE_URL_SET, "application/json", strings.NewReader(body))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			resp.Body.Close()
			fmt.Printf("Request %d sent\n", i)
		})
	}
	wg.Wait()
	fmt.Println("Done!")
}
