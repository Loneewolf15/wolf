// Package e2e_test — Tier 3 regression tests.
//
// These tests cover the security and correctness fixes made in the
// strcpy/sprintf/keep-alive pass. Each test is named after the issue it
// would have caught BEFORE the fix was applied.
//
// Run with: WOLF_HTTP_TEST=1 go test ./e2e/... -run TestT3
package e2e_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const regressionPort = 19070

// startRegressionServer compiles and starts 66_regression_server.wolf once
// per test that needs it. The server listens on regressionPort.
func startRegressionServer(t *testing.T) {
	t.Helper()
	compileAndStartServer(t, "testdata/66_regression_server.wolf", regressionPort)
}

// --------------------------------------------------------------------------
// T3-01: Keep-alive request limit
//
// Before fix: wolf_http_engine.c had no keep_alive_count — a single client
// could hold a connection open for unlimited requests, hogging a ctx slot.
// After fix: the engine closes the connection after WOLF_KEEPALIVE_MAX_REQUESTS
// (1000) successful responses.
//
// Strategy: use a raw TCP connection to send 1001 HTTP/1.1 GET requests on
// the same socket, reading each response. Request 1001 must either:
//   (a) receive a response with "Connection: close", or
//   (b) get EOF when attempting to read the response
//
// We test up to 1005 to give a small margin for off-by-one in the limit.
// --------------------------------------------------------------------------

func TestT3_01_KeepAliveLimit(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	addr := fmt.Sprintf("127.0.0.1:%d", regressionPort)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(60 * time.Second)) //nolint

	req := "GET /ping HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n"
	br := bufio.NewReader(conn)

	closedAt := -1
	const limit = 1005 // check up to 5 past the nominal limit

	for i := 1; i <= limit; i++ {
		if _, err := fmt.Fprint(conn, req); err != nil {
			// Server closed the write side — normal after limit
			closedAt = i
			break
		}

		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			// EOF or connection reset after the limit
			closedAt = i
			break
		}
		io.Copy(io.Discard, resp.Body) //nolint
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected 200, got %d", i, resp.StatusCode)
		}

		// Check for Connection: close header — server signalling graceful close
		if strings.EqualFold(resp.Header.Get("Connection"), "close") {
			closedAt = i
			break
		}
	}

	if closedAt == -1 {
		t.Errorf("server kept connection alive for all %d requests — keep-alive limit not enforced", limit)
		return
	}

	// The limit is 1000; allow the server to close anywhere in [999, 1005]
	// to account for the response being sent before close is processed.
	if closedAt < 999 || closedAt > 1005 {
		t.Errorf("connection closed at request %d; expected closure around request 1000", closedAt)
	} else {
		t.Logf("T3-01: connection correctly closed at request %d", closedAt)
	}
}

// --------------------------------------------------------------------------
// T3-02: Concurrent stress — 500 simultaneous connections
//
// Before fix: arena pool could be exhausted silently; overflow arenas were
// unbounded. Also: strcpy race in body append could corrupt responses.
// After fix: memcpy-based append; overflow arenas bounded by OOM panic.
//
// We fire 500 goroutines simultaneously and require all succeed.
// --------------------------------------------------------------------------

func TestT3_02_StressConcurrent500(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	const N = 500
	var success, failure atomic.Int64
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: N,
			DisableKeepAlives:   false,
		},
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/ping", regressionPort)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(url)
			if err != nil {
				failure.Add(1)
				return
			}
			io.Copy(io.Discard, resp.Body) //nolint
			resp.Body.Close()
			if resp.StatusCode == 200 {
				success.Add(1)
			} else {
				failure.Add(1)
			}
		}()
	}
	wg.Wait()

	t.Logf("T3-02: %d/%d concurrent requests succeeded", success.Load(), N)
	if failure.Load() > 0 {
		t.Errorf("T3-02: %d/%d requests failed under 500-connection stress", failure.Load(), N)
	}
}

// --------------------------------------------------------------------------
// T3-03: Metrics /dash endpoint — JSON integrity
//
// Before fix: strcpy + O(n²) strcat with no overflow guard.
// After fix: pointer-advance with explicit bounds check, valid JSON always.
// --------------------------------------------------------------------------

func TestT3_03_MetricsDash(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	url := fmt.Sprintf("http://127.0.0.1:%d/dash", regressionPort)
	resp, err := http.Get(url) //nolint
	if err != nil {
		t.Fatalf("GET /dash: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GET /dash: expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /dash body: %v", err)
	}

	// Must be valid JSON array
	var arr []interface{}
	if err := json.Unmarshal(body, &arr); err != nil {
		t.Errorf("GET /dash: body is not a valid JSON array: %v\nbody: %s", err, body)
	} else {
		t.Logf("T3-03: /dash returned %d metrics", len(arr))
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("GET /dash: expected Content-Type application/json, got %q", ct)
	}
}

// --------------------------------------------------------------------------
// T3-04: Body append integrity — multi-chunk write
//
// Before fix: wolf_http_res_write with existing body used strcpy+strcat,
// which is undefined behaviour if the source overlaps or the null-terminator
// is at the wrong position on the second call.
// After fix: memcpy with tracked lengths.
//
// Wolf server calls wolf_http_res_write 3× with "alpha", "beta", "gamma".
// The concatenated body must equal exactly "alphabetagamma".
// --------------------------------------------------------------------------

func TestT3_04_BodyAppendIntegrity(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	url := fmt.Sprintf("http://127.0.0.1:%d/multi-write", regressionPort)
	resp, err := http.Get(url) //nolint
	if err != nil {
		t.Fatalf("GET /multi-write: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GET /multi-write: expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	const want = "alphabetagamma"
	got := string(body)
	if got != want {
		t.Errorf("body append: want %q, got %q", want, got)
	} else {
		t.Logf("T3-04: body append correct: %q", got)
	}
}

// --------------------------------------------------------------------------
// T3-05: Many response headers — buffer bounds
//
// Before fix: response header buffer is 4096 bytes with no overflow check;
// 20+ large headers could silently truncate the HTTP response.
// After fix: snprintf with remaining-bytes tracking.
//
// Wolf server sets 20 custom headers. We verify all 20 arrive intact.
// --------------------------------------------------------------------------

func TestT3_05_ManyResponseHeaders(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	url := fmt.Sprintf("http://127.0.0.1:%d/many-headers", regressionPort)
	resp, err := http.Get(url) //nolint
	if err != nil {
		t.Fatalf("GET /many-headers: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GET /many-headers: expected 200, got %d", resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body) //nolint

	missing := []string{}
	for i := 1; i <= 20; i++ {
		key := fmt.Sprintf("X-H%02d", i)
		want := fmt.Sprintf("value-%02d", i)
		got := resp.Header.Get(key)
		if got != want {
			missing = append(missing, fmt.Sprintf("%s: want %q got %q", key, want, got))
		}
	}

	if len(missing) > 0 {
		t.Errorf("T3-05: %d headers missing or truncated:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	} else {
		t.Logf("T3-05: all 20 response headers intact")
	}
}

// --------------------------------------------------------------------------
// T3-06: Keep-alive connection reuse — regression
//
// Complementary to T3-01: verifies that keep-alive works correctly for
// the normal case (< 1000 requests) and doesn't close prematurely.
// --------------------------------------------------------------------------

func TestT3_06_KeepAliveReuse(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	// A single http.Client with keep-alives enabled should reuse the connection.
	// We send 10 requests and verify all succeed — if keep-alive were broken
	// (premature close) we'd see connection errors starting from request 2.
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 1,
		},
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/ping", regressionPort)

	for i := 1; i <= 10; i++ {
		resp, err := client.Get(url)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body) //nolint
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected 200, got %d", i, resp.StatusCode)
		}
	}
	t.Logf("T3-06: 10 keep-alive requests all succeeded")
}

// requireHTTPTest skips the test unless WOLF_HTTP_TEST=1 is set.
func requireHTTPTest(t *testing.T) {
	t.Helper()
	if os.Getenv("WOLF_HTTP_TEST") != "1" {
		t.Skip("set WOLF_HTTP_TEST=1 to run regression HTTP tests")
	}
}
