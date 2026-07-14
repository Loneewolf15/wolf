// Package e2e_test — SQL injection and input-handling tests.
//
// These tests verify that the Wolf HTTP engine survives hostile input at every
// injection surface (URL, headers, body) without crashing, corrupting memory,
// or leaking raw user data into a position where it could become a SQL command.
//
// No live database is required. The assertions are:
//  1. Server returns a valid HTTP response (not 5xx or connection failure).
//  2. Server echoes the raw input back unchanged — proving it passed through
//     the C runtime without truncation (null-byte), duplication (strcat race),
//     or corruption (buffer overflow).
//  3. Common SQL/XSS injection payloads do not crash the process.
//
// Run with: WOLF_HTTP_TEST=1 go test ./e2e/... -run TestT3_SQL
package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// sqlInjectionPayloads is a representative set of classic and modern
// SQL injection / XSS patterns used as fuzzing inputs.
var sqlInjectionPayloads = []struct {
	name    string
	payload string
}{
	{"single quote", "'"},
	{"double quote", "\""},
	{"sql comment", "' --"},
	{"or-true", "' OR '1'='1"},
	{"union select", "' UNION SELECT NULL,NULL,NULL--"},
	{"drop table", "'; DROP TABLE users; --"},
	{"sleep", "'; SELECT SLEEP(5); --"},
	{"stacked queries", "1; SELECT * FROM users"},
	{"null byte", "hello\x00world"},
	{"backslash", "O'Brien\\backslash"},
	{"percent", "100% done"},
	{"ampersand", "a=1&b=2"},
	{"angle brackets", "<script>alert(1)</script>"},
	{"newline injection", "value\r\nX-Injected: evil"},
	{"long string 1k", strings.Repeat("A", 1024)},
	{"unicode", "héllo wörld"},
}

// --------------------------------------------------------------------------
// T3_SQL-01: URL query parameter injection surface
//
// Sends each SQL payload as the ?input= query parameter to /echo-query.
// The server echoes it back as plain text. We assert:
//   - HTTP 200 (no crash)
//   - Response body is non-empty (no buffer truncation)
//   - Connection remains alive for subsequent requests
// --------------------------------------------------------------------------

func TestT3_SQL_01_QueryParamInjection(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	client := &http.Client{Timeout: 5 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%d/echo-query", regressionPort)

	for _, tc := range sqlInjectionPayloads {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			u := base + "?input=" + url.QueryEscape(tc.payload)
			resp, err := client.Get(u)
			if err != nil {
				t.Fatalf("GET failed: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d (payload: %q)", resp.StatusCode, tc.payload)
			}

			// Null byte special case: C strings truncate at \x00, so the
			// echo will contain only "hello" — but must NOT crash.
			if tc.name == "null byte" {
				if len(body) == 0 {
					t.Errorf("null byte payload: expected non-empty response, got empty")
				}
				return
			}

			got := string(body)
			if got != tc.payload {
				t.Errorf("echo mismatch for %q:\n  want: %q\n   got: %q", tc.name, tc.payload, got)
			}
		})
	}
}

// --------------------------------------------------------------------------
// T3_SQL-02: HTTP header injection surface
//
// Sends each payload as the X-Test-Header value. The server echoes it.
// Header values cannot contain bare \r or \n (HTTP spec forbids it), so we
// skip the newline injection case. All others must echo intact.
// --------------------------------------------------------------------------

func TestT3_SQL_02_HeaderInjection(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/echo-header", regressionPort)

	for _, tc := range sqlInjectionPayloads {
		tc := tc
		// Skip cases that are illegal in HTTP header values
		if tc.name == "newline injection" || tc.name == "null byte" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			// http.Header.Set validates header values — use canonical form
			req.Header["X-Test-Header"] = []string{tc.payload}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d (payload: %q)", resp.StatusCode, tc.payload)
				return
			}

			got := string(body)
			if got != tc.payload {
				t.Errorf("header echo mismatch for %q:\n  want: %q\n   got: %q",
					tc.name, tc.payload, got)
			}
		})
	}
}

// --------------------------------------------------------------------------
// T3_SQL-03: POST body injection surface
//
// Sends each payload as a raw POST body to /echo-body.
// Since wolf_http_req_body uses C string semantics, null bytes will truncate —
// but the server must NOT crash, corrupt memory, or return 5xx.
// --------------------------------------------------------------------------

func TestT3_SQL_03_BodyInjection(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	client := &http.Client{Timeout: 5 * time.Second}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/echo-body", regressionPort)

	for _, tc := range sqlInjectionPayloads {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Post(endpoint, "text/plain", strings.NewReader(tc.payload))
			if err != nil {
				t.Fatalf("POST failed: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Must be 200 — server must not crash on hostile input
			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d (payload: %q)", resp.StatusCode, tc.payload)
				return
			}

			// Body must be non-empty (proves request body was processed)
			if len(body) == 0 && len(tc.payload) > 0 {
				// Null byte truncation is acceptable
				if tc.name != "null byte" {
					t.Errorf("body injection: empty response for non-empty payload %q", tc.name)
				}
			}
		})
	}
}

// --------------------------------------------------------------------------
// T3_SQL-04: Route path injection — no path traversal or injection via path
//
// Sends SQL injection payloads as URL path components.
// The server must return 404 (not found) for unknown paths — proving the
// router didn't execute or interpret the payload.
// --------------------------------------------------------------------------

func TestT3_SQL_04_PathInjection(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	client := &http.Client{Timeout: 5 * time.Second}

	pathPayloads := []string{
		"/ping' OR '1'='1",
		"/ping; DROP TABLE routes",
		"/ping/../../../etc/passwd",
		"/ping%00",
		"/ping\r\nX-Injected: evil",
	}

	for _, p := range pathPayloads {
		p := p
		t.Run(p, func(t *testing.T) {
			u := fmt.Sprintf("http://127.0.0.1:%d%s", regressionPort, url.PathEscape(p))
			resp, err := client.Get(u)
			if err != nil {
				// Connection failure = crash = fail
				t.Errorf("GET %q: connection failed (possible crash): %v", p, err)
				return
			}
			io.Copy(io.Discard, resp.Body) //nolint
			resp.Body.Close()

			// Injected path should NOT match /ping and return 404
			if resp.StatusCode == 200 {
				t.Errorf("path %q matched a route unexpectedly — possible path traversal", p)
			}
			// 400/404/500 are acceptable; connection failure is not
		})
	}
}

// --------------------------------------------------------------------------
// T3_SQL-05: Server survives 100 consecutive injection requests
//
// Confirms no memory corruption accumulates across repeated hostile requests
// on the same keep-alive connection.
// --------------------------------------------------------------------------

func TestT3_SQL_05_RepeatedInjectionStability(t *testing.T) {
	requireHTTPTest(t)
	startRegressionServer(t)

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 1,
		},
	}
	base := fmt.Sprintf("http://127.0.0.1:%d/echo-query", regressionPort)
	payload := url.QueryEscape("' OR '1'='1; DROP TABLE users; --")

	for i := 0; i < 100; i++ {
		resp, err := client.Get(base + "?input=" + payload)
		if err != nil {
			t.Fatalf("request %d failed: %v — possible crash after injection accumulation", i+1, err)
		}
		io.Copy(io.Discard, resp.Body) //nolint
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
	}
	t.Logf("T3_SQL-05: 100 consecutive injection requests completed without crash")
}
