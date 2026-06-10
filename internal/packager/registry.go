package packager

import (
	"fmt"
	"os"
	"path/filepath"
)

// Install dependencies listed in wolf.mod.
func Install(projectRoot string) error {
	modPath := filepath.Join(projectRoot, "wolf.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return fmt.Errorf("no wolf.mod found in %s", projectRoot)
	}

	mod, err := ParseModFile(modPath)
	if err != nil {
		return fmt.Errorf("failed to parse wolf.mod: %w", err)
	}

	if len(mod.Require) == 0 {
		fmt.Println("wolf install: no dependencies to install")
		return nil
	}

	lockPath := filepath.Join(projectRoot, "wolf.lock")
	lock, _ := ParseLockFile(lockPath)
	if lock == nil {
		lock = &LockFile{Locked: make(map[string]string)}
	}

	fetcher := NewFetcher(projectRoot)

	// Fetch dependencies
	changed := false
	for pkg, req := range mod.Require {
		lockedSHA, isLocked := lock.Locked[pkg]
		
		// If it's already locked and downloaded, we could verify the SHA, but for MVP we skip if it exists
		pkgDir := filepath.Join(fetcher.ModulesDir, filepath.Base(pkg))
		if isLocked {
			if _, err := os.Stat(pkgDir); err == nil {
				fmt.Printf("wolf install: %s is already up to date\n", pkg)
				continue
			}
		}

		// Not locked or missing directory: fetch it
		sha, err := fetcher.Fetch(pkg, req.Version)
		if err != nil {
			return err
		}

		if isLocked && lockedSHA != sha {
			fmt.Printf("wolf install: warning: SHA changed for %s (was %s, now %s)\n", pkg, lockedSHA, sha)
		}

		lock.Locked[pkg] = sha
		changed = true
	}

	if changed {
		if err := WriteLockFile(lockPath, lock); err != nil {
			return fmt.Errorf("failed to write wolf.lock: %w", err)
		}
		fmt.Println("wolf install: wrote wolf.lock")
	}

	fmt.Println("wolf install: success ✓")
	return nil
}
