package packager

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Fetcher handles downloading and verifying packages.
type Fetcher struct {
	ModulesDir string
}

func NewFetcher(projectRoot string) *Fetcher {
	return &Fetcher{
		ModulesDir: filepath.Join(projectRoot, ".wolf_modules"),
	}
}

func (f *Fetcher) Fetch(pkgURL, version string) (string, error) {
	if err := f.validateURL(pkgURL); err != nil {
		return "", fmt.Errorf("invalid package URL %q: %w", pkgURL, err)
	}

	// Create .wolf_modules if it doesn't exist
	if err := os.MkdirAll(f.ModulesDir, 0755); err != nil {
		return "", err
	}

	// Determine target directory
	pkgDir := filepath.Join(f.ModulesDir, filepath.Base(pkgURL))

	// If it already exists, remove it (force clean install)
	if _, err := os.Stat(pkgDir); err == nil {
		os.RemoveAll(pkgDir)
	}

	// Normalize URL for git clone
	gitURL := pkgURL
	if !strings.HasPrefix(gitURL, "https://") && !strings.HasPrefix(gitURL, "git@") && !strings.HasPrefix(gitURL, "file://") {
		gitURL = "https://" + gitURL
	}

	fmt.Printf("wolf install: cloning %s @ %s...\n", pkgURL, version)

	// Execute git clone
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", version, gitURL, pkgDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed for %s: %w\n%s", pkgURL, err, out.String())
	}

	// Get the commit SHA
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaCmd.Dir = pkgDir
	shaOut, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA for %s: %w", pkgURL, err)
	}

	sha := strings.TrimSpace(string(shaOut))

	// Remove .git to prevent nested git repos
	os.RemoveAll(filepath.Join(pkgDir, ".git"))

	return sha, nil
}

func (f *Fetcher) validateURL(url string) error {
	if strings.Contains(url, "--") || strings.Contains(url, ";") || strings.Contains(url, "&") {
		return fmt.Errorf("URL contains invalid characters")
	}
	return nil
}
