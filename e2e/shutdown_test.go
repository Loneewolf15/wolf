package e2e_test

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wolflang/wolf/internal/compiler"
)

func TestGracefulShutdown(t *testing.T) {
	wolfFile := "testdata/_server_shutdown.wolf"
	source, err := os.ReadFile(wolfFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	c := compiler.New()
	c.OutDir = "testdata/wolf_out_shutdown"
	defer os.RemoveAll(c.OutDir)

	result, err := c.Build(string(source), wolfFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	cmd := exec.Command(result.OutputPath)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait for server to start
	scanner := bufio.NewScanner(stdout)
	stopChan := make(chan bool)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Wolf HTTP Server running") {
				stopChan <- true
				return
			}
		}
	}()

	select {
	case <-stopChan:
		// Started
	case <-time.After(120 * time.Second):
		t.Errorf("Server failed to start in time")
		cmd.Process.Kill()
		return
	}

	// Send SIGINT
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	// Capture stderr
	stderrReader := io.TeeReader(stderr, os.Stderr)
	stderrScanner := bufio.NewScanner(stderrReader)
	shutdownFound := false
	poolDestroyed := false

	done := make(chan bool)
	go func() {
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			if strings.Contains(line, "Shutdown signal received") {
				shutdownFound = true
			}
			if strings.Contains(line, "Pool destroyed") {
				poolDestroyed = true
			}
			if strings.Contains(line, "Shutdown complete") {
				done <- true
				return
			}
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Errorf("Server failed to shut down in time")
	}

	if err := cmd.Wait(); err != nil {
		// exec.Command returns an error if exit code is non-zero.
		// Our runtime exits with exit(0), so err should be nil.
		t.Errorf("Server exited with error: %v", err)
	}

	if !shutdownFound {
		t.Errorf("Shutdown signal message not found in stderr")
	}
	if !poolDestroyed {
		t.Errorf("Pool destroyed message not found in stderr")
	}
}
