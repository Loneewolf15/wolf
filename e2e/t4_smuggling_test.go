package e2e_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

const smugglingPort = 19080

// startSmugglingServer compiles and starts 55_smuggling_defense.wolf.
func startSmugglingServer(t *testing.T) {
	t.Helper()
	compileAndStartServer(t, "testdata/55_smuggling_defense.wolf", smugglingPort)
}

// --------------------------------------------------------------------------
// T4: HTTP Smuggling Defenses (RFC 9112)
//
// These tests verify that the HTTP parser correctly rejects structurally
// ambiguous requests that could be used for HTTP Request Smuggling, bypassing
// WAFs, or poisoning frontend caches.
// --------------------------------------------------------------------------

func TestT4_Smuggling(t *testing.T) {
	requireHTTPTest(t)
	startSmugglingServer(t)

	testCases := []struct {
		name         string
		payload      string
		expectStatus string
	}{
		{
			name:         "Rule 1 (CL+TE coexistence)",
			payload:      "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n",
			expectStatus: "400",
		},
		{
			name:         "Rule 3 (Bad TE value 'xchunked')",
			payload:      "POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: xchunked\r\n\r\n",
			expectStatus: "400",
		},
		{
			name:         "Rule 3 (Obfuscated TE 'chunked, trailers')",
			payload:      "POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked, trailers\r\n\r\n",
			expectStatus: "400",
		},
		{
			name:         "Rule 4 (Duplicate CL)",
			payload:      "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nContent-Length: 6\r\n\r\n",
			expectStatus: "400",
		},
		{
			name:         "Rule 5 (Bare CR in headers)",
			payload:      "GET / HTTP/1.1\r\nHost: localhost\r\nX-Bad: this has a bare\rCR\r\n\r\n",
			expectStatus: "400",
		},
		{
			name:         "Baseline Valid TE (Should not be 400)",
			payload:      "POST / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
			expectStatus: "200",
		},
	}

	addr := fmt.Sprintf("127.0.0.1:%d", smugglingPort)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))

			// Write the raw payload
			if _, err := fmt.Fprint(conn, tc.payload); err != nil {
				t.Fatalf("write payload: %v", err)
			}

			// Read response
			resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			defer resp.Body.Close()

			statusStr := fmt.Sprintf("%d", resp.StatusCode)
			if statusStr != tc.expectStatus {
				t.Errorf("expected status %s, got response:\n%s", tc.expectStatus, resp.Status)
			} else {
				t.Logf("PASS: Received expected %s -> %s", tc.expectStatus, resp.Status)
			}
		})
	}
}
