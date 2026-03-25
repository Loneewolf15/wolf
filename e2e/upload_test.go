package e2e_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wolflang/wolf/internal/compiler"
)

func TestFileUpload(t *testing.T) {
	wolfFile := "testdata/_server_upload.wolf"
	source, err := os.ReadFile(wolfFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	c := compiler.New()
	c.OutDir = "testdata/wolf_out_upload"
	defer os.RemoveAll(c.OutDir)

	result, err := c.Build(string(source), wolfFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Start server binary
	cmd := exec.Command(result.OutputPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		_ = cmd.Process.Signal(syscall.SIGINT)
		_ = cmd.Wait()
	}()

	// Wait for "Wolf HTTP Server running" on stdout
	ready := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Wolf HTTP Server running") {
				ready <- true
				return
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(30 * time.Second):
		t.Fatal("Server did not start in time")
	}

	// Build multipart/form-data request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("avatar", "test_photo.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	fileContent := []byte("fake-jpeg-bytes-1234567890")
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}
	writer.Close()

	url := "http://127.0.0.1:9292/upload"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result2 map[string]interface{}
	if err := json.Unmarshal(respBody, &result2); err != nil {
		t.Fatalf("Response is not valid JSON: %s — error: %v", string(respBody), err)
	}

	if received, ok := result2["received"].(bool); !ok || !received {
		t.Errorf("Expected received=true, got: %v", result2)
	}
	if name, ok := result2["name"].(string); !ok || name != "test_photo.jpg" {
		t.Errorf("Expected name=test_photo.jpg, got: %v", result2["name"])
	}
	if size, ok := result2["size"].(float64); !ok || int(size) != len(fileContent) {
		t.Errorf("Expected size=%d, got: %v", len(fileContent), result2["size"])
	}

	t.Logf("Upload test passed: %s", string(respBody))
}
