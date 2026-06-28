package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Publish executes a decentralized release of a Wolf package.
// It verifies the wolf.json version, creates a Git tag, and pushes it to the origin.
func Publish(projectRoot string) error {
	manifestPath := filepath.Join(projectRoot, "wolf.json")
	m, err := ParseManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no wolf.json found in %s. Run 'wolf init' first", projectRoot)
		}
		return fmt.Errorf("failed to parse wolf.json: %w", err)
	}

	if m.Version == "" {
		return fmt.Errorf("invalid version in wolf.json")
	}

	tagName := "v" + m.Version

	fmt.Printf("wolf publish: preparing release for %s @ %s...\n", m.Name, tagName)

	// Check if working directory is clean
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = projectRoot
	statusOut, _ := statusCmd.Output()
	if len(statusOut) > 0 {
		return fmt.Errorf("git working directory is not clean. Commit your changes before publishing")
	}

	// Create git tag
	tagCmd := exec.Command("git", "tag", "-a", tagName, "-m", "Release "+tagName)
	tagCmd.Dir = projectRoot
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("failed to create git tag %s. Does it already exist?", tagName)
	}

	// Push tags to origin
	fmt.Printf("wolf publish: pushing tag %s to origin...\n", tagName)
	pushCmd := exec.Command("git", "push", "origin", tagName)
	pushCmd.Dir = projectRoot
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push tag to origin. Make sure you have a remote origin configured")
	}

	fmt.Printf("wolf publish: successfully published %s!\n", tagName)
	fmt.Println("\nTo make this package discoverable in the central vanity index,")
	fmt.Println("submit a Pull Request to the registry index: https://github.com/wolflang/registry")

	return nil
}
