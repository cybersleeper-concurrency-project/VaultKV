package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

const BASE_URL_SET = "http://localhost:8080/set"
const BASE_URL_GET = "http://localhost:8080/get"
const ITER_COUNT = 20

// SET, GET, MIX
const COMMAND_TYPE = "MIX"

const INIT_SET = true

const LOAD_COUNT = 1000
const CONCURRENCY_LIMIT = 50

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

	if INIT_SET {
		fmt.Printf("Initializing store set\n")
		successCnt := 0
		for i := range LOAD_COUNT {
			body := fmt.Sprintf(`{"key": "user:%d", "value": "value:%d"}`, i, i)
			resp, err := client.Post(BASE_URL_SET, "application/json", strings.NewReader(body))
			if err == nil && 200 <= resp.StatusCode && resp.StatusCode < 300 {
				successCnt++
			}
		}
		fmt.Printf("Done init store cnt: %d\n", successCnt)
	}

	for tt := range ITER_COUNT {
		start := time.Now()

		errCnt := 0
		for i := range LOAD_COUNT {
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()

				var resp *http.Response
				var err error
				command := COMMAND_TYPE
				if strings.EqualFold(command, "MIX") {
					rnd := rand.IntN(2)
					if rnd == 0 {
						command = "SET"
					} else {
						command = "GET"
					}
				}
				if strings.EqualFold(command, "SET") {
					body := fmt.Sprintf(`{"key": "user:%d", "value": "value:%d"}`, i, i)
					resp, err = client.Post(BASE_URL_SET, "application/json", strings.NewReader(body))
				} else {
					resp, err = client.Get(fmt.Sprintf(`%s?key=user:%d`, BASE_URL_GET, i))
				}
				if err != nil {
					errCnt++
					// fmt.Printf("Error: %v\n", err)
					return
				}
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					errCnt++
					// fmt.Printf("Error Status: %d\n", resp.StatusCode)
					return
				}

				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
				// fmt.Printf("Request %d sent\n", i)
			})
		}
		wg.Wait()

		duration := time.Since(start).Milliseconds()
		rps := float64(LOAD_COUNT) / (float64(duration) / 1000)

		fmt.Printf("Test #%d\n", tt+1)
		fmt.Printf("Duration: %dms\n", duration)
		fmt.Printf("RPS: %.2f\n", rps)
		fmt.Printf("Err count: %d\n", errCnt)
		fmt.Printf("Err rate: %.2f%%\n", float64(errCnt)/LOAD_COUNT*100)
		fmt.Printf("==================================\n")

		avgTime += int(duration)
		avgErr += errCnt

		time.Sleep(5 * time.Second)
	}

	avgTimeFinal := avgTime / ITER_COUNT
	avgRps := float64(LOAD_COUNT) / (float64(avgTimeFinal) / 1000)

	fmt.Printf("Avg time: %dms\n", avgTimeFinal)
	fmt.Printf("Avg RPS: %.2f\n", avgRps)
	fmt.Printf("Avg err rate: %.2f%%\n", float64(avgErr)/ITER_COUNT/LOAD_COUNT*100)
	fmt.Printf("==================================\n")

}
