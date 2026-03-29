package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPClient(t *testing.T) {
	// 1. Start a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"hello":"world"}`)
		}
	}))
	defer ts.Close()

	// 2. Prepare the Wolf script (replace localhost with the mock server URL)
	// Since the URL in the .wolf file is hardcoded to localhost:9292,
	// we'll create a temporary .wolf file with the actual mock server URL.
	tmpWolf := filepath.Join("testdata", "30_http_client_tmp.wolf")
	script := fmt.Sprintf(`$res = wolf_http_get("%s/test")
print($res)`, ts.URL)
	err := os.WriteFile(tmpWolf, []byte(script), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpWolf)

	// 3. Compile and Run
	// Note: We need to use the wolf compiler binary if it exists, or use 'go run'
	cmd := exec.Command("go", "run", "../cmd/wolf/main.go", "run", tmpWolf)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Wolf run failed: %v\nOutput: %s", err, string(out))
	}

	output := string(out)
	t.Logf("Wolf output: %s", output)

	if !strings.Contains(output, `"status":200`) {
		t.Errorf("Expected status 200 in output, got: %s", output)
	}
	if !strings.Contains(output, `{\"hello\":\"world\"}`) {
		t.Errorf("Expected body in output, got: %s", output)
	}
}
