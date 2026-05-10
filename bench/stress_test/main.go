// Wolf HTTP Stress Harness
// Tests: arena overflow leak, O(1) free-list correctness, closure lifetime under load.
// Usage: go run main.go [--url http://...] [--requests N] [--concurrency N]
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:8084/", "Target URL")
	total := flag.Int("requests", 100_000, "Total number of requests")
	concurrency := flag.Int("concurrency", 200, "Concurrent workers")
	flag.Parse()

	fmt.Printf("╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║       Wolf HTTP Engine — Stress Test Harness         ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  URL          : %-36s ║\n", *url)
	fmt.Printf("║  Requests     : %-36d ║\n", *total)
	fmt.Printf("║  Concurrency  : %-36d ║\n", *concurrency)
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n\n")

	// Wait for server to be ready (up to 10s)
	fmt.Print("⏳ Waiting for server to come up...")
	host := "127.0.0.1:8084"
	for i := 0; i < 100; i++ {
		conn, err := net.DialTimeout("tcp", host, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			fmt.Println(" ✓ ready")
			break
		}
		if i == 99 {
			fmt.Println("\n✗ Server did not come up within 10s — aborting")
			os.Exit(1)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Shared transport — persistent connections, no per-request TLS overhead
	transport := &http.Transport{
		MaxIdleConns:        *concurrency + 50,
		MaxIdleConnsPerHost: *concurrency + 50,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	// Counters
	var (
		done       atomic.Int64
		ok200      atomic.Int64
		errConn    atomic.Int64
		errTimeout atomic.Int64
		err5xx     atomic.Int64
		errBody    atomic.Int64 // wrong body / corrupted
		latSum     atomic.Int64 // nanoseconds
		latMax     atomic.Int64
	)

	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup
	start := time.Now()

	// Progress ticker
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			d := done.Load()
			elapsed := time.Since(start).Seconds()
			rps := float64(d) / elapsed
			fmt.Printf("  → %6d/%d done  |  %.0f RPS  |  5xx: %d  conn-err: %d  body-err: %d\n",
				d, *total, rps, err5xx.Load(), errConn.Load(), errBody.Load())
		}
	}()

	for i := 0; i < *total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			t0 := time.Now()
			resp, err := client.Get(*url)
			latNs := time.Since(t0).Nanoseconds()
			latSum.Add(latNs)

			// Track max latency atomically
			for {
				old := latMax.Load()
				if latNs <= old {
					break
				}
				if latMax.CompareAndSwap(old, latNs) {
					break
				}
			}

			done.Add(1)

			if err != nil {
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
					errTimeout.Add(1)
				} else {
					errConn.Add(1)
				}
				return
			}
			defer resp.Body.Close()

			body, readErr := io.ReadAll(resp.Body)

			if resp.StatusCode >= 500 {
				err5xx.Add(1)
				return
			}
			if resp.StatusCode == 200 {
				ok200.Add(1)
			}

			// Validate body is non-empty and contains expected string
			if readErr != nil || len(body) == 0 || !strings.Contains(string(body), "Wolf") {
				errBody.Add(1)
			}
		}()
	}

	wg.Wait()
	ticker.Stop()
	elapsed := time.Since(start)

	totalDone := done.Load()
	avgLatMs := float64(latSum.Load()) / float64(totalDone) / 1e6
	maxLatMs := float64(latMax.Load()) / 1e6
	rps := float64(totalDone) / elapsed.Seconds()

	// Failure rate
	failures := errConn.Load() + errTimeout.Load() + err5xx.Load() + errBody.Load()
	failPct := float64(failures) / float64(totalDone) * 100.0

	// Result
	fmt.Println()
	fmt.Printf("╔══════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                  STRESS TEST RESULTS                ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Total requests  : %-33d ║\n", totalDone)
	fmt.Printf("║  Duration        : %-33s ║\n", elapsed.Round(time.Millisecond))
	fmt.Printf("║  Throughput      : %-30.0f RPS ║\n", rps)
	fmt.Printf("╠══════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  ✓  200 OK       : %-33d ║\n", ok200.Load())
	fmt.Printf("║  ✗  5xx errors   : %-33d ║\n", err5xx.Load())
	fmt.Printf("║  ✗  conn errors  : %-33d ║\n", errConn.Load())
	fmt.Printf("║  ✗  timeouts     : %-33d ║\n", errTimeout.Load())
	fmt.Printf("║  ✗  body corrupt : %-33d ║\n", errBody.Load())
	fmt.Printf("╠══════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Avg latency     : %-30.2f ms ║\n", avgLatMs)
	fmt.Printf("║  Max latency     : %-30.2f ms ║\n", maxLatMs)
	fmt.Printf("╠══════════════════════════════════════════════════════╣\n")

	if failures == 0 {
		fmt.Printf("║  VERDICT: ✅ PASS — 0 failures (%.2f%% error rate)  ║\n", failPct)
	} else {
		fmt.Printf("║  VERDICT: ❌ FAIL — %d failures (%.2f%% error rate)%-5s║\n", failures, failPct, "")
	}
	fmt.Printf("╚══════════════════════════════════════════════════════╝\n")

	if failures > 0 {
		os.Exit(1)
	}
}
