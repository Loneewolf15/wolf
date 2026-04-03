// Package migrate provides the Wolf database migration engine.
// Migration files live in ./migrations/ and follow the naming convention:
//
//	YYYYMMDD_HHMMSS_description.sql
//
// Each file must contain an "-- Up" section and optionally a "-- Down" section:
//
//	-- Up
//	CREATE TABLE ...;
//
//	-- Down
//	DROP TABLE IF EXISTS ...;
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a single versioned migration file.
type Migration struct {
	Filename  string // e.g. "20260402_180000_create_users.sql"
	Version   string // e.g. "20260402_180000"
	Name      string // e.g. "create_users"
	UpSQL     string // SQL to apply
	DownSQL   string // SQL to rollback (may be empty)
	AppliedAt *time.Time
}

// Scan reads the migrations directory and returns all Migration structs,
// sorted by filename (which is naturally chronological given the timestamp prefix).
func Scan(migrationsDir string) ([]Migration, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no migrations directory = no migrations
		}
		return nil, fmt.Errorf("wolf migrate: cannot read migrations dir: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		m, err := parseFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return nil, fmt.Errorf("wolf migrate: parse %s: %w", name, err)
		}
		migrations = append(migrations, m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Filename < migrations[j].Filename
	})

	return migrations, nil
}

// MakeFilename returns a new migration filename for the given description.
// Format: YYYYMMDD_HHMMSS_description.sql
func MakeFilename(description string) string {
	now := time.Now().UTC()
	ts := now.Format("20060102_150405")
	slug := toSlug(description)
	return fmt.Sprintf("%s_%s.sql", ts, slug)
}

// MakeTemplate returns the starter SQL content for a new migration file.
func MakeTemplate(description string) string {
	return fmt.Sprintf(`-- Migration: %s
-- Created: %s

-- Up


-- Down

`, description, time.Now().UTC().Format(time.RFC3339))
}

// parseFile reads a .sql migration file and splits it into Up/Down sections.
func parseFile(path string) (Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Migration{}, err
	}

	filename := filepath.Base(path)
	version, name := parseFilename(filename)

	content := string(data)
	upSQL, downSQL := splitSections(content)

	return Migration{
		Filename: filename,
		Version:  version,
		Name:     name,
		UpSQL:    strings.TrimSpace(upSQL),
		DownSQL:  strings.TrimSpace(downSQL),
	}, nil
}

// parseFilename splits "20260402_180000_create_users.sql" into version + name.
func parseFilename(filename string) (version, name string) {
	base := strings.TrimSuffix(filename, ".sql")
	// Version is the first two underscore-delimited segments: YYYYMMDD_HHMMSS
	parts := strings.SplitN(base, "_", 4)
	if len(parts) >= 3 {
		version = parts[0] + "_" + parts[1]
		if len(parts) >= 4 {
			name = parts[2] + "_" + parts[3]
		} else if len(parts) == 3 {
			name = parts[2]
		}
	} else {
		version = base
		name = base
	}
	return
}

// splitSections splits migration content into Up and Down parts.
// Sections are delimited by "-- Up" and "-- Down" markers (case-insensitive).
func splitSections(content string) (upSQL, downSQL string) {
	lines := strings.Split(content, "\n")
	const (
		sectionNone = iota
		sectionUp
		sectionDown
	)
	section := sectionNone

	var upLines, downLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)

		if upper == "-- UP" {
			section = sectionUp
			continue
		}
		if upper == "-- DOWN" {
			section = sectionDown
			continue
		}

		switch section {
		case sectionUp:
			upLines = append(upLines, line)
		case sectionDown:
			downLines = append(downLines, line)
		}
	}

	return strings.Join(upLines, "\n"), strings.Join(downLines, "\n")
}

// toSlug converts a description to a lowercase underscore slug.
func toSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, ch := range s {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch == ' ' || ch == '-':
			b.WriteByte('_')
		}
	}
	return b.String()
}
