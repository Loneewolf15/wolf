package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	start := time.Now()
	numWorkers := 100
	requestsPerWorker := 500

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 100,
				},
			}
			for j := 0; j < requestsPerWorker; j++ {
				resp, err := client.Get("http://127.0.0.1:19070/ping")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == 200 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := successCount + errorCount
	rps := float64(totalRequests) / duration.Seconds()

	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Successes: %d\n", successCount)
	fmt.Printf("Errors: %d\n", errorCount)
	fmt.Printf("RPS: %.2f\n", rps)
}
