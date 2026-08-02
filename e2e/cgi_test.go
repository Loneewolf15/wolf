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
	$method = wolf_http_req_method($req)
	$path = wolf_http_req_path($req)
	$body = wolf_http_req_body($req)
	$out = "Method: " + $method + ", Path: " + $path
	if ($body != "") {
		$out = $out + ", Body: " + $body
	}
	wolf_http_res_write($res, $out)
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
		if !strings.Contains(out, "Method: GET, Path: /hello") {
			t.Errorf("expected echoed context, got: %s", out)
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
		// CGI adapter allocates NO body if CONTENT_LENGTH is missing
		if !strings.Contains(out, "Method: POST, Path: /upload") || strings.Contains(out, "Body:") {
			t.Errorf("expected no body due to missing content length, got: %s", out)
		}
	})

	// Test 3: Oversized body truncated correctly (or doesn't crash)
	t.Run("OversizedBody", func(t *testing.T) {
		// CGI adapter allocates CONTENT_LENGTH bytes.
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "POST",
			"CONTENT_LENGTH": "100000",
			"PATH_INFO":      "/large",
		}, "short data")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
		if !strings.Contains(out, "Method: POST, Path: /large") || strings.Contains(out, "Body:") {
			t.Errorf("expected empty body (exceeds WOLF_MAX_REQUEST_SIZE), got: %s", out)
		}
	})
	
	// Test 4: Malformed method shouldn't crash and should pass through accurately
	t.Run("MalformedMethod", func(t *testing.T) {
		out, err := runCGI(map[string]string{
			"REQUEST_METHOD": "BLAHBLAH",
			"PATH_INFO":      "/test",
		}, "")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
		if !strings.Contains(out, "Status: 200") {
			t.Errorf("expected Status: 200, got: %s", out)
		}
		if !strings.Contains(out, "Method: BLAHBLAH, Path: /test") {
			t.Errorf("expected malformed method to be passed through, got: %s", out)
		}
	})
}
