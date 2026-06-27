package packager

import (
	"fmt"
	"os"
	"path/filepath"
)

// Install dependencies listed in wolf.json recursively.
func Install(projectRoot string) error {
	manifestPath := filepath.Join(projectRoot, "wolf.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("no wolf.json found in %s", projectRoot)
	}

	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse wolf.json: %w", err)
	}

	if len(manifest.Dependencies) == 0 {
		fmt.Println("wolf install: no dependencies to install")
		return nil
	}

	lockPath := filepath.Join(projectRoot, "wolf.lock")
	lock, _ := ParseLockFile(lockPath)
	if lock == nil {
		lock = &LockFile{Locked: make(map[string]string)}
	}

	fetcher := NewFetcher(projectRoot)
	changed := false
	visited := make(map[string]bool)

	// Recursive fetch function
	var fetchDeps func(deps map[string]string) error
	fetchDeps = func(deps map[string]string) error {
		for pkgName, semverConstraint := range deps {
			if visited[pkgName] {
				continue
			}
			visited[pkgName] = true

			lockedEntry, isLocked := lock.Locked[pkgName]
			pkgDir := filepath.Join(fetcher.ModulesDir, pkgName)

			// Simple caching logic:
			// If it's in the lock file, and the directory exists, we assume it's good.
			// (A true production version would verify the SHA inside .wolf_modules).
			if isLocked {
				if _, err := os.Stat(pkgDir); err == nil {
					// Recurse
					subManifestPath := filepath.Join(pkgDir, "wolf.json")
					if subManifest, err := ParseManifest(subManifestPath); err == nil {
						if err := fetchDeps(subManifest.Dependencies); err != nil {
							return err
						}
					}
					continue
				}
			}

			// Resolve package to get Repository URL and available versions
			regResp, err := ResolvePackage(pkgName)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s: %w", pkgName, err)
			}

			repoURL := regResp.Repository
			availableTags := regResp.Versions

			// If it's a direct URL bypass, it won't have versions from the registry.
			// We need to fetch them directly via git ls-remote.
			if len(availableTags) == 0 {
				tags, err := fetcher.GetTags(repoURL)
				if err != nil {
					return fmt.Errorf("failed to get tags for %s: %w", repoURL, err)
				}
				availableTags = tags
			}

			// Evaluate SemVer constraint
			resolvedVersion := HighestCompatible(semverConstraint, availableTags)
			if resolvedVersion == "" {
				return fmt.Errorf("no compatible version found for %s (constraint: %s, available: %v)", pkgName, semverConstraint, availableTags)
			}

			// Not locked or missing directory: fetch it
			sha, err := fetcher.Fetch(pkgName, repoURL, resolvedVersion)
			if err != nil {
				return err
			}

			if isLocked && lockedEntry != sha {
				fmt.Printf("wolf install: warning: SHA changed for %s\n", pkgName)
			}

			lock.Locked[pkgName] = sha
			changed = true

			// Recurse into the newly downloaded package
			subManifestPath := filepath.Join(pkgDir, "wolf.json")
			if subManifest, err := ParseManifest(subManifestPath); err == nil {
				if err := fetchDeps(subManifest.Dependencies); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := fetchDeps(manifest.Dependencies); err != nil {
		return err
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
