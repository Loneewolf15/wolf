//go:build !linux

package watcher

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// Watcher recursively monitors a directory for file changes
// using standard Go polling, adhering to zero-dependency rules.
type Watcher struct {
	Root     string
	Interval time.Duration
	modTimes map[string]time.Time
}

// New creates a new Watcher pointing to the project root
func New(root string, interval time.Duration) *Watcher {
	return &Watcher{
		Root:     root,
		Interval: interval,
		modTimes: make(map[string]time.Time),
	}
}

// Watch blocks and polls the filesystem. When a `.wolf` or `wolf.json` file changes,
// it invokes the onChange callback with the modified path.
func (w *Watcher) Watch(ctx context.Context, onChange func(string)) {
	// Initial scan to populate baseline mod times
	w.scan(false, nil)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(true, onChange)
		}
	}
}

// scan walks the root directory and compares modification times.
func (w *Watcher) scan(trigger bool, onChange func(string)) {
	filepath.WalkDir(w.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			// Skip internal/dependency directories to avoid massive polling overhead
			if name == ".git" || name == "wolf_out" || name == ".wolf_modules" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only watch wolf files and config
		if !strings.HasSuffix(d.Name(), ".wolf") && d.Name() != "wolf.json" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		mtime := info.ModTime()
		prevTime, exists := w.modTimes[path]

		if trigger && exists && mtime.After(prevTime) {
			w.modTimes[path] = mtime
			// Call callback on first file detected as changed
			onChange(path)
			// Stop scanning if we found a change (SkipAll requires Go 1.20+)
			return filepath.SkipAll
		}

		w.modTimes[path] = mtime
		return nil
	})
}
