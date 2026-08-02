package e2e

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
	"wolf/internal/config"
)

func TestCGIAdapter(t *testing.T) {
	// 1. Create a dummy test file
	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "cgi_app.wolf")
	sourceCode := `
func handle($req: int, $res: int) {
	wolf_http_res_status($res, 200)
	wolf_http_res_write($res, "Hello CGI World!")
}
wolf_http_serve(8080, handle)
`
	if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	// 2. Setup config for CGI
	c := compiler.New()
	c.Config = &config.WolfConfig{}
	c.Config.Target.CGI = true

	result, err := c.Build(string(sourceCode), sourcePath)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}
	binPath := result.OutputPath

	// Helper to run CGI
	runCGI := func(env map[string]string, stdin string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath)
		cmd.Env = os.Environ() // inherit path, etc
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr
		
		err := cmd.Run()
		return out.String(), err
	}

	// Test 1: Standard GET request
	t.Run("StandardGET", func(t *testing.T) {
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "GET",
			"PATH_INFO":      "/hello",
		}, "")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
		if !strings.Contains(out, "Hello CGI World!") {
			t.Errorf("expected body, got: %s", out)
		}
	})

	// Test 2: Missing CONTENT_LENGTH for POST shouldn't crash
	t.Run("MissingContentLength", func(t *testing.T) {
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "POST",
			"PATH_INFO":      "/upload",
		}, "body data")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
	})

	// Test 3: Oversized body truncated correctly (or doesn't crash)
	t.Run("OversizedBody", func(t *testing.T) {
		// CGI adapter allocates CONTENT_LENGTH bytes.
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "POST",
			"CONTENT_LENGTH": "100000",
		}, "small")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
	})
	
	// Test 4: Malformed REQUEST_METHOD
	t.Run("MalformedMethod", func(t *testing.T) {
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "BLAHBLAH",
		}, "")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
	})
}
