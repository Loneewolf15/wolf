package e2e_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wolflang/wolf/internal/compiler"
)

func TestEndToEnd(t *testing.T) {
	testdata := "testdata"
	files, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".wolf") {
			continue
		}

		name := file.Name()
		t.Run(name, func(t *testing.T) {
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

			// Compile
			c := compiler.New()
			// Set OutDir exactly 3 levels deep from repo root: e2e/testdata/wolf_out_X
			c.OutDir = filepath.Join(testdata, "wolf_out_"+name)
			defer os.RemoveAll(c.OutDir)
			
			result, err := c.Build(string(source), wolfFile)
			if err != nil {
				// Print errors
				t.Fatalf("Compilation failed:\n%v\n%s", err, strings.Join(result.Errors, "\n"))
			}

			// Run it
			cmd := exec.Command(result.OutputPath)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err = cmd.Run()
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
