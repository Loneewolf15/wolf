package compiler

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wolflang/wolf"
)

// ensureAssetsExtracted extracts the embedded runtime and third_party files
// into a temporary cache directory if they don't already exist.
// Returns the path to the cache directory.
func ensureAssetsExtracted() (string, error) {
	cacheDir := filepath.Join(os.TempDir(), "wolf_compiler_assets")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	// Extract runtime files
	runtimeDir := filepath.Join(cacheDir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return "", err
	}

	entries, err := fs.ReadDir(wolf.Assets, "runtime")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded runtime files: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(wolf.Assets, "runtime/"+entry.Name())
		if err != nil {
			return "", err
		}
		targetPath := filepath.Join(runtimeDir, entry.Name())
		if err := writeIfDifferent(targetPath, data); err != nil {
			return "", err
		}
	}

	// Extract third_party files
	thirdPartyDir := filepath.Join(cacheDir, "third_party")
	if err := os.MkdirAll(thirdPartyDir, 0755); err != nil {
		return "", err
	}

	if err := extractFS(wolf.Assets, "third_party/lib", filepath.Join(thirdPartyDir, "lib")); err != nil {
		return "", fmt.Errorf("failed to extract embedded third_party files: %w", err)
	}

	return cacheDir, nil
}

func extractFS(f fs.FS, srcDir string, destDir string) error {
	return fs.WalkDir(f, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, path[len(srcDir):])
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := fs.ReadFile(f, path)
		if err != nil {
			return err
		}
		return writeIfDifferent(targetPath, data)
	})
}

func writeIfDifferent(path string, data []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && len(existing) == len(data) {
		same := true
		for i := range data {
			if existing[i] != data[i] {
				same = false
				break
			}
		}
		if same {
			return nil // Already extracted and identical
		}
	}
	return os.WriteFile(path, data, 0644)
}
