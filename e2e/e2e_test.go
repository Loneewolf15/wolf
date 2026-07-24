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

	"wolf/internal/compiler"
	"wolf/internal/packager"
	"wolf/internal/tester"
)

// isHTTPTest returns true for tests that require a live network or bind a
// server port (30_* = HTTP client, 31_* = WebSocket server, 37_http_client = outbound HTTP).
// These tests are skipped unless WOLF_HTTP_TEST=1 is set locally,
// and are always skipped in CI (where the network is unavailable).
func isHTTPTest(name string) bool {
	return strings.HasPrefix(name, "30_") ||
		strings.HasPrefix(name, "31_") ||
		name == "37_http_client.wolf" ||
		name == "58_sockets.wolf" ||
		// T2/T3 HTTP server programs — bind ports, never exit on their own;
		// driven by TestT2_* / TestT3_* in t2_http_test.go.
		strings.HasPrefix(name, "61_") ||
		strings.HasPrefix(name, "62_") ||
		strings.HasPrefix(name, "63_") ||
		strings.HasPrefix(name, "64_") ||
		strings.HasPrefix(name, "65_") ||
		strings.HasPrefix(name, "66_")
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
		if len(file.Name()) == 0 || file.Name()[0] < '0' || file.Name()[0] > '9' {
			continue
		}

		name := file.Name()
		t.Run(name, func(t *testing.T) {
			// Tests 45 and 46 mutate shared state (testdata/wolf.mod, os.Stdout)
			// and must NOT run in parallel. All other tests use unique OutDirs.
			if name != "45_package_install.wolf" && name != "46_test_runner.wolf" {
				t.Parallel()
			}
			if isHTTPTest(name) {
				if os.Getenv("CI") != "" {
					t.Skip("skipping HTTP/WS e2e in CI (network/port dependency)")
				}
				if os.Getenv("WOLF_HTTP_TEST") != "1" {
					t.Skip("skipping HTTP/WS e2e locally (set WOLF_HTTP_TEST=1 to run)")
				}
			}

			if name == "45_package_install.wolf" {
				// Setup wolf.json dynamically
				absTestdata, _ := filepath.Abs(testdata)
				tmpl, _ := os.ReadFile(filepath.Join(testdata, "wolf.json.template"))
				modContent := strings.ReplaceAll(string(tmpl), "{PWD}", absTestdata)
				os.WriteFile(filepath.Join(testdata, "wolf.json"), []byte(modContent), 0644)

				// Ensure dummy_pkg_repo is a git repo so clone works
				dummyRepo := filepath.Join(testdata, "dummy_pkg_repo")
				exec.Command("git", "-C", dummyRepo, "init", "-b", "main").Run()
				exec.Command("git", "-C", dummyRepo, "config", "user.name", "Test").Run()
				exec.Command("git", "-C", dummyRepo, "config", "user.email", "test@test").Run()
				exec.Command("git", "-C", dummyRepo, "add", ".").Run()
				exec.Command("git", "-C", dummyRepo, "commit", "-m", "Init").Run()

				// Run packager
				if err := packager.Install(testdata); err != nil {
					t.Fatalf("Failed to run wolf install: %v", err)
				}
				defer os.RemoveAll(filepath.Join(testdata, ".wolf_modules"))
				defer os.Remove(filepath.Join(testdata, "wolf.json"))
				defer os.Remove(filepath.Join(testdata, "wolf.lock"))
			}

			if name == "46_test_runner.wolf" {
				// We don't want to compile 46_test_runner.wolf directly, it's just a dummy to trigger this
				// But we need to capture stdout.
				// Actually, e2e_test.go always compiles and runs the file. We can just let it compile the dummy file.
				// But we want to run tester.Run(). Let's capture stdout here.

				// Create a temporary directory for testing
				testDir := filepath.Join(testdata, "temp_test_dir")
				os.MkdirAll(testDir, 0755)
				defer os.RemoveAll(testDir)

				// Write a dummy test file
				os.WriteFile(filepath.Join(testDir, "dummy_test.wolf"), []byte(`
func test_math() {
    assert(1 + 1 == 2, "Math is broken")
}
func test_fail() {
    assert(1 == 2, "Expected failure")
}
`), 0644)

				// Capture stdout
				oldStdout := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w

				err := tester.Run(testDir)

				w.Close()
				os.Stdout = oldStdout

				var buf bytes.Buffer
				buf.ReadFrom(r)

				out := buf.String()

				// Strip time, compiler debug output, and linker commands
				lines := strings.Split(out, "\n")
				var cleanLines []string
				for _, line := range lines {
					if !strings.HasPrefix(line, "wolf test: Running tests") &&
						!strings.HasPrefix(line, "Total Time:") &&
						!strings.HasPrefix(line, ">> ") &&
						!strings.HasPrefix(line, "Warning: ") &&
						!strings.HasPrefix(line, "wolf: linker command: ") {
						cleanLines = append(cleanLines, line)
					}
				}
				cleanOut := strings.TrimSpace(strings.Join(cleanLines, "\n"))

				expectedOut, _ := os.ReadFile(filepath.Join(testdata, "46_test_runner.out"))
				expectedStr := strings.TrimSpace(string(expectedOut))

				if cleanOut != expectedStr {
					t.Errorf("Test runner output mismatch.\nExpected:\n%s\nGot:\n%s", expectedStr, cleanOut)
				}

				if err == nil {
					t.Errorf("Expected tester.Run() to return an error because test_fail failed")
				}

				return // skip the normal build/run phase
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
			c.ProjectRoot = testdata
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

func TestInternalSuites(t *testing.T) {
	err := tester.Run("../src/compiler")
	if err != nil {
		t.Fatalf("Compiler tests failed: %v", err)
	}
	err = tester.Run("../tests")
	if err != nil {
		t.Fatalf("Integration tests failed: %v", err)
	}
}
