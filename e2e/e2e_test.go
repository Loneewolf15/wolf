package e2e_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wolflang/wolf/internal/compiler"
)

// isHTTPTest returns true for tests that require a live network or bind a
// server port (30_* = HTTP client, 31_* = WebSocket server).
// These tests are skipped unless WOLF_HTTP_TEST=1 is set locally,
// and are always skipped in CI (where the network is unavailable).
func isHTTPTest(name string) bool {
	return strings.HasPrefix(name, "30_") || strings.HasPrefix(name, "31_")
}

func TestEndToEnd(t *testing.T) {
	testdata := "testdata"
	files, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if !file.Type().IsRegular() || !strings.HasSuffix(file.Name(), ".wolf") || strings.HasPrefix(file.Name(), "_") {
			continue
		}

		name := file.Name()
		t.Run(name, func(t *testing.T) {
			if isHTTPTest(name) {
				if os.Getenv("CI") != "" {
					t.Skip("skipping HTTP/WS e2e in CI (network/port dependency)")
				}
				if os.Getenv("WOLF_HTTP_TEST") != "1" {
					t.Skip("skipping HTTP/WS e2e locally (set WOLF_HTTP_TEST=1 to run)")
				}
			}

			wolfFile := filepath.Join(testdata, name)
			outFile := filepath.Join(testdata, strings.TrimSuffix(name, ".wolf")+".out")

			// Check if expected output exists
			expectedOut, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("Expected output file %s is missing: %v", outFile, err)
			}

			// Read Wolf source
			source, err := os.ReadFile(wolfFile)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", wolfFile, err)
			}

			t.Logf("Compiling %s...", name)
			// Compile
			startBuild := time.Now()
			c := compiler.New()
			// Set OutDir exactly 3 levels deep from repo root: e2e/testdata/wolf_out_X
			c.OutDir = filepath.Join(testdata, "wolf_out_"+name)
			defer os.RemoveAll(c.OutDir)

			result, err := c.Build(string(source), wolfFile)
			if err != nil {
				// Print errors
				t.Fatalf("Compilation failed:\n%v\n%s", err, strings.Join(result.Errors, "\n"))
			}
			t.Logf("Compiled %s to %s (Took: %v)", name, result.OutputPath, time.Since(startBuild))

			// Use a timeout context so server tests that never exit can't
			// orphan a process and hold a port across test runs.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, result.OutputPath)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Env = append(os.Environ(), "TZ=UTC")

			t.Logf("Running %s...", name)
			startRun := time.Now()
			err = cmd.Run()
			t.Logf("Finished %s (Took: %v)", name, time.Since(startRun))
			if err != nil {
				t.Fatalf("Program execution failed: %v\nStderr: %s", err, stderr.String())
			}

			// Compare output
			actual := strings.TrimSpace(stdout.String())
			expected := strings.TrimSpace(string(expectedOut))

			if actual != expected {
				t.Errorf("Output mismatch.\nExpected:\n%s\nGot:\n%s", expected, actual)
			}
		})
	}
}
