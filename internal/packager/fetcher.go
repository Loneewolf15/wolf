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

func (f *Fetcher) GetTags(repoURL string) ([]string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "--refs", repoURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags for %s: %w", repoURL, err)
	}

	lines := strings.Split(string(out), "\n")
	var tags []string
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			tag := strings.TrimPrefix(ref, "refs/tags/")
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (f *Fetcher) Fetch(pkgName, repoURL, version string) (string, error) {
	if err := f.validateURL(repoURL); err != nil {
		return "", fmt.Errorf("invalid repository URL %q: %w", repoURL, err)
	}

	// Create .wolf_modules if it doesn't exist
	if err := os.MkdirAll(f.ModulesDir, 0755); err != nil {
		return "", err
	}

	// Determine target directory using pkgName, NOT repoURL
	pkgDir := filepath.Join(f.ModulesDir, filepath.FromSlash(pkgName))

	// If it already exists, remove it (force clean install)
	if _, err := os.Stat(pkgDir); err == nil {
		os.RemoveAll(pkgDir)
	}

	// Normalize URL for git clone
	gitURL := repoURL
	if strings.HasPrefix(gitURL, "/") || strings.HasPrefix(gitURL, "./") || strings.HasPrefix(gitURL, "../") {
		// Local path; use absolute file:// URI
		absPath, _ := filepath.Abs(gitURL)
		gitURL = "file://" + absPath
	} else if !strings.HasPrefix(gitURL, "https://") && !strings.HasPrefix(gitURL, "git@") && !strings.HasPrefix(gitURL, "file://") {
		gitURL = "https://" + gitURL
	}

	fmt.Printf("wolf install: cloning %s (%s) @ %s...\n", pkgName, gitURL, version)

	// Execute git clone
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", version, gitURL, pkgDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed for %s: %w\n%s", pkgName, err, out.String())
	}

	// Get the commit SHA
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaCmd.Dir = pkgDir
	shaOut, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA for %s: %w", pkgName, err)
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
