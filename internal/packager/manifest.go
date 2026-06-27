package packager

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest represents the parsed wolf.json file.
type Manifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// LockFile represents the parsed wolf.lock file.
type LockFile struct {
	Locked map[string]string `json:"locked"` // Module URL -> SHA
}

// ParseManifest parses the wolf.json file.
func ParseManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}

	return &m, nil
}

// WriteManifest writes the wolf.json file.
func WriteManifest(path string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// ParseLockFile parses the wolf.lock file. Returns an empty LockFile if not found.
func ParseLockFile(path string) (*LockFile, error) {
	lock := &LockFile{
		Locked: make(map[string]string),
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return lock, nil
		}
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}

	if err := json.Unmarshal(b, lock); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if lock.Locked == nil {
		lock.Locked = make(map[string]string)
	}

	return lock, nil
}

// WriteLockFile writes the wolf.lock file.
func WriteLockFile(path string, lock *LockFile) error {
	b, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
