//go:build linux

package watcher

import (
	"sync"

	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)


type safeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.m.Lock()
	defer s.m.Unlock()
	return s.b.String()
}

func TestParseInotifyEvents_TruncatedBuffer(t *testing.T) {
	cases := []struct {
		name string
		buf  []byte
		n    int
	}{
		{"empty", []byte{}, 0},
		{"header only, no name", make([]byte, 16), 16},
		{"header claims huge name length", func() []byte {
			b := make([]byte, 16)
			binary.LittleEndian.PutUint32(b[12:16], 1<<20) // len field says 1MB name, buffer doesn't have it
			return b
		}(), 16},
		{"one byte short of a full event", make([]byte, 15), 15},
		{"negative-looking length via overflow", func() []byte {
			b := make([]byte, 16)
			binary.LittleEndian.PutUint32(b[12:16], 0xFFFFFFFF)
			return b
		}(), 16},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked on input %q: %v", c.name, r)
				}
			}()
			_ = parseInotifyEvents(c.buf, c.n) // should return whatever it can parse, or empty — never panic
		})
	}
}

func TestWatcher_IgnoredCleansMap(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "subdir")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	w := New(root, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Watch(ctx, func(path string) {})
	time.Sleep(50 * time.Millisecond) // let initial WalkDir + watch registration settle

	before := w.wdCount()

	if err := os.RemoveAll(sub); err != nil {
		t.Fatal(err)
	}

	// Poll rather than sleep-and-hope — IN_IGNORED delivery isn't instant
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if w.wdCount() < before {
			return // pass — map shrank
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("wdPaths did not shrink after directory deletion: before=%d, still=%d", before, w.wdCount())
}

func TestWatcher_ENOSPCWarns(t *testing.T) {
	orig := sysInotifyAddWatch
	defer func() { sysInotifyAddWatch = orig }()

	sysInotifyAddWatch = func(fd int, path string, mask uint32) (int, error) {
		return -1, syscall.ENOSPC
	}

	var buf safeBuffer
	logOutput = &buf // redirect for capture
	defer func() { logOutput = os.Stderr }()

	dir := t.TempDir()
	w := New(dir, time.Second)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go w.Watch(ctx, func(p string) {})
	time.Sleep(50 * time.Millisecond)

	if !strings.Contains(buf.String(), "inotify watch limit reached") {
		t.Fatalf("expected ENOSPC warning, got: %s", buf.String())
	}
}

func TestWatcher_CloseDuringBlockedRead(t *testing.T) {
	for i := 0; i < 200; i++ { // repeat to surface races statistically
		dir := t.TempDir()
		w := New(dir, time.Second)

		done := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			w.Watch(ctx, func(p string) {}) // blocks on Read() until ctx cancelled
			close(done)
		}()

		time.Sleep(2 * time.Millisecond) // let the read actually block
		
		cancel() // Cancel the context to unblock the watcher loop

		select {
		case <-done:
			// good — Watch() returned cleanly
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: Watch() did not return after Close(), likely hung on old fd", i)
		}
	}
}
