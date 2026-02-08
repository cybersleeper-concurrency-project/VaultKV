package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const BASE_URL_SET = "http://localhost:8080/set"
const BASE_URL_GET = "http://localhost:8080/get"
const LOAD_COUNT = 500
const ITER_COUNT = 5
const COMMAND_TYPE = "SET"

func main() {
	var wg sync.WaitGroup
	for tt := range ITER_COUNT {
		start := time.Now()

		errCnt := 0
		for i := range LOAD_COUNT {
			wg.Go(func() {
				var resp *http.Response
				var err error
				if strings.EqualFold(COMMAND_TYPE, "SET") {
					body := fmt.Sprintf(`{"key": "user:%d", "value": "CrashTest"}`, i)
					resp, err = http.Post(BASE_URL_SET, "application/json", strings.NewReader(body))
				} else {
					resp, err = http.Get(fmt.Sprintf(`%s?key=%d`, BASE_URL_GET, i))
				}
				if err != nil {
					errCnt++
					// fmt.Printf("Error: %v\n", err)
					return
				}
				resp.Body.Close()
				// fmt.Printf("Request %d sent\n", i)
			})
		}
		wg.Wait()
		fmt.Printf("Test #%d\n", tt+1)
		fmt.Printf("Duration: %dms\n", time.Since(start).Milliseconds())
		fmt.Printf("Err count: %d\n", errCnt)
		fmt.Printf("Err rate: %.2f%%\n", float64(errCnt)/LOAD_COUNT*100)
		fmt.Printf("==================================\n")
	}
}
