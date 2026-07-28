//go:build linux

package watcher

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Watcher struct {
	Root     string
	Interval time.Duration

	fd      int
	wdPaths map[int32]string
	mu      sync.Mutex
}

func New(root string, interval time.Duration) *Watcher {
	return &Watcher{
		Root:     root,
		Interval: interval,
		wdPaths:  make(map[int32]string),
	}
}

func (w *Watcher) Watch(ctx context.Context, onChange func(string)) {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wolf dev: inotify init failed: %v\n", err)
		return
	}
	w.fd = fd

	// Wrap in os.File to hook into Go's runtime poller.
	// This makes concurrent Close() safe against fd reuse races.
	f := os.NewFile(uintptr(fd), "inotify")
	defer f.Close()

	// Recursively add watches to all initial directories
	filepath.WalkDir(w.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "wolf_out" || name == ".wolf_modules" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			w.addWatch(path)
		}
		return nil
	})

	go func() {
		<-ctx.Done()
		f.Close() // Safely unblocks f.Read()
	}()

	var buf [4096]byte
	for {
		n, err := f.Read(buf[:])
		if err != nil {
			return // Context cancelled or fd closed
		}

		if n < 16 {
			continue
		}

		offset := 0
		var changedFile string
		for offset <= n-16 {
			wd := int32(binary.LittleEndian.Uint32(buf[offset : offset+4]))
			mask := binary.LittleEndian.Uint32(buf[offset+4 : offset+8])
			length := binary.LittleEndian.Uint32(buf[offset+12 : offset+16])

			var name string
			if length > 0 {
				// Prevent slice out-of-bounds panic on malformed/truncated events
				if offset+16+int(length) > n {
					break
				}
				nameBytes := buf[offset+16 : offset+16+int(length)]
				name = strings.TrimRight(string(nameBytes), "\x00")
			}

			offset += 16 + int(length)

			// Clean up memory when watches are implicitly removed
			if (mask & syscall.IN_IGNORED) != 0 {
				w.mu.Lock()
				delete(w.wdPaths, wd)
				w.mu.Unlock()
				continue
			}

			w.mu.Lock()
			dirPath, ok := w.wdPaths[wd]
			w.mu.Unlock()

			if !ok {
				continue
			}

			fullPath := filepath.Join(dirPath, name)

			// Dynamically watch new directories
			if (mask & syscall.IN_ISDIR) != 0 {
				if (mask & syscall.IN_CREATE) != 0 {
					w.addWatch(fullPath)
				}
				continue
			}

			// Filter to Wolf files
			if !strings.HasSuffix(name, ".wolf") && name != "wolf.json" {
				continue
			}

			// Check for relevant modifications
			if (mask & (syscall.IN_MODIFY | syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_MOVED_TO)) != 0 {
				changedFile = fullPath
			}
		}

		// Fire callback once per batch of events (debouncing handled by caller)
		if changedFile != "" {
			onChange(changedFile)
		}
	}
}

func (w *Watcher) addWatch(path string) {
	flags := uint32(syscall.IN_MODIFY | syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_MOVED_TO)
	wd, err := syscall.InotifyAddWatch(w.fd, path, flags)
	if err == nil {
		w.mu.Lock()
		w.wdPaths[int32(wd)] = path
		w.mu.Unlock()
	} else if err == syscall.ENOSPC {
		// Log a warning if the user hits the fs.inotify.max_user_watches limit
		fmt.Fprintf(os.Stderr, "wolf dev: inotify watch limit reached. Cannot watch %s\n", path)
	}
}
