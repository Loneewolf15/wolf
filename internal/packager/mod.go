package packager

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Requirement represents a required module version.
type Requirement struct {
	Version string
}

// ModFile represents the parsed wolf.mod file.
type ModFile struct {
	Module  string
	Wolf    string
	Require map[string]Requirement
}

// LockFile represents the parsed wolf.lock file.
type LockFile struct {
	Locked map[string]string // Module URL -> SHA
}

// ParseModFile parses the wolf.mod file.
func ParseModFile(path string) (*ModFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	mod := &ModFile{
		Require: make(map[string]Requirement),
	}

	scanner := bufio.NewScanner(f)
	var section string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		line := stripComment(raw)
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: expected 'key = value', got: %q", lineNum, raw)
		}

		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])

		// strip quotes
		if strings.HasPrefix(key, `"`) && strings.HasSuffix(key, `"`) {
			key = key[1 : len(key)-1]
		}
		if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
			val = val[1 : len(val)-1]
		}

		if section == "" {
			if key == "module" {
				mod.Module = val
			} else if key == "wolf" {
				mod.Wolf = val
			}
		} else if section == "require" {
			mod.Require[key] = Requirement{Version: val}
		}
	}

	return mod, scanner.Err()
}

// ParseLockFile parses the wolf.lock file. Returns an empty LockFile if not found.
func ParseLockFile(path string) (*LockFile, error) {
	lock := &LockFile{
		Locked: make(map[string]string),
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return lock, nil
		}
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" || (strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")) {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if strings.HasPrefix(key, `"`) && strings.HasSuffix(key, `"`) {
			key = key[1 : len(key)-1]
		}
		if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
			val = val[1 : len(val)-1]
		}
		lock.Locked[key] = val
	}

	return lock, nil
}

// WriteLockFile writes the wolf.lock file.
func WriteLockFile(path string, lock *LockFile) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "[locked]")
	for pkg, sha := range lock.Locked {
		fmt.Fprintf(f, `"%s" = "%s"`+"\n", pkg, sha)
	}
	return nil
}

func stripComment(s string) string {
	inQuote := false
	for i, ch := range s {
		if ch == '"' {
			inQuote = !inQuote
		}
		if ch == '#' && !inQuote {
			return s[:i]
		}
	}
	return s
}
