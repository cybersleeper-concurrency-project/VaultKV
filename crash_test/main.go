package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const BASE_URL_SET = "http://localhost:8080/set"
const BASE_URL_GET = "http://localhost:8080/get"
const ITER_COUNT = 20
const COMMAND_TYPE = "SET"

const LOAD_COUNT = 1000
const CONCURRENCY_LIMIT = 100

func main() {
	t := &http.Transport{
		MaxIdleConns:        CONCURRENCY_LIMIT,
		MaxIdleConnsPerHost: CONCURRENCY_LIMIT,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}

	avgTime, avgErr := 0, 0

	var wg sync.WaitGroup
	sem := make(chan struct{}, CONCURRENCY_LIMIT)

	for tt := range ITER_COUNT {
		start := time.Now()

		errCnt := 0
		for i := range LOAD_COUNT {
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()

				var resp *http.Response
				var err error
				if strings.EqualFold(COMMAND_TYPE, "SET") {
					body := fmt.Sprintf(`{"key": "user:%d", "value": "CrashTest"}`, i)
					resp, err = client.Post(BASE_URL_SET, "application/json", strings.NewReader(body))
				} else {
					resp, err = client.Get(fmt.Sprintf(`%s?key=%d`, BASE_URL_GET, i))
				}
				if err != nil {
					errCnt++
					// fmt.Printf("Error: %v\n", err)
					return
				}

				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
				// fmt.Printf("Request %d sent\n", i)
			})
		}
		wg.Wait()

		duration := time.Since(start).Milliseconds()

		fmt.Printf("Test #%d\n", tt+1)
		fmt.Printf("Duration: %dms\n", duration)
		fmt.Printf("Err count: %d\n", errCnt)
		fmt.Printf("Err rate: %.2f%%\n", float64(errCnt)/LOAD_COUNT*100)
		fmt.Printf("==================================\n")

		avgTime += int(duration)
		avgErr += errCnt

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Avg time: %d\n", avgTime/ITER_COUNT)
	fmt.Printf("Avg rrr rate: %.2f%%\n", float64(avgErr)/ITER_COUNT/LOAD_COUNT*100)
	fmt.Printf("==================================\n")

}
