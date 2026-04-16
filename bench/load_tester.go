package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/", "URL to test")
	concurrency := flag.Int("c", 100, "Number of concurrent workers")
	totalReqs := flag.Int("n", 100000, "Total number of requests")
	flag.Parse()

	reqsPerWorker := *totalReqs / *concurrency
	var wg sync.WaitGroup

	start := time.Now()
	var (
		mu       sync.Mutex
		failed   int
		success  int
		maxLat   time.Duration
		totalLat time.Duration
	)

	// Custom transport with connection pooling optimization
	tr := &http.Transport{
		MaxIdleConns:        *concurrency,
		MaxIdleConnsPerHost: *concurrency,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	fmt.Printf("Running load test: %d reqs across %d workers targeting %s\n", *totalReqs, *concurrency, *url)

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < reqsPerWorker; j++ {
				reqStart := time.Now()
				resp, err := client.Get(*url)
				lat := time.Since(reqStart)

				mu.Lock()
				totalLat += lat
				if lat > maxLat {
					maxLat = lat
				}
				if err != nil || resp.StatusCode != 200 {
					failed++
				} else {
					success++
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	dur := time.Since(start)

	fmt.Printf("Completed %d requests in %v\n", *totalReqs, dur)
	fmt.Printf("Success: %d, Failed: %d\n", success, failed)
	fmt.Printf("Req/s: %.2f\n", float64(*totalReqs)/dur.Seconds())
	fmt.Printf("Avg Latency: %v, Max Latency: %v\n", totalLat/time.Duration(*totalReqs), maxLat)
}
