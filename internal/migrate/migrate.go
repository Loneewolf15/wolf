package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"wolf/internal/config"
)

// Migrator manages applying and rolling back database migrations.
type Migrator struct {
	cfg           config.DBConfig
	migrationsDir string
	out           io.Writer // where status output is written
}

// New creates a Migrator from the project's database config.
// migrationsDir defaults to "./migrations" if empty.
func New(cfg config.DBConfig, migrationsDir string) *Migrator {
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	return &Migrator{
		cfg:           cfg,
		migrationsDir: migrationsDir,
		out:           os.Stdout,
	}
}

// Up applies up to `n` pending migrations. If n == 0, all pending are applied.
func (m *Migrator) Up(n int) error {
	db, err := openDB(m.cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db, m.cfg.Driver); err != nil {
		return err
	}

	migrations, err := Scan(m.migrationsDir)
	if err != nil {
		return err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return err
	}

	// Execute each pending migration inside its own transaction.
	// If any statement fails, the tx is rolled back so the schema is never half-applied.
	count := 0
	for _, mg := range migrations {
		if _, done := applied[mg.Filename]; done {
			continue
		}
		if n > 0 && count >= n {
			break
		}

		if mg.UpSQL == "" {
			fmt.Fprintf(m.out, "  ⚠ skipping %s — no Up section found\n", mg.Filename)
			continue
		}

		fmt.Fprintf(m.out, "  ▶ Applying %s...\n", mg.Filename)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("wolf migrate: begin tx for %s: %w", mg.Filename, err)
		}

		var txErr error
		statements := splitStatements(mg.UpSQL)
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, txErr = tx.Exec(stmt); txErr != nil {
				break
			}
		}

		if txErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("wolf migrate: failed running Up for %s (rolled back): %w", mg.Filename, txErr)
		}

		// Mark applied inside the same transaction — atomic with the DDL.
		if strings.ToLower(m.cfg.Driver) == "postgres" {
			_, txErr = tx.Exec("INSERT INTO wolf_migrations (filename) VALUES ($1)", mg.Filename)
		} else {
			_, txErr = tx.Exec("INSERT INTO wolf_migrations (filename) VALUES (?)", mg.Filename)
		}
		if txErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("wolf migrate: failed to mark %s as applied (rolled back): %w", mg.Filename, txErr)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("wolf migrate: commit for %s: %w", mg.Filename, err)
		}

		fmt.Fprintf(m.out, "  ✓ Applied %s\n", mg.Filename)
		count++
	}

	if count == 0 {
		fmt.Fprintln(m.out, "  Nothing to migrate — already up to date.")
	} else {
		fmt.Fprintf(m.out, "\nwolf migrate: %d migration(s) applied ✓\n", count)
	}

	return nil
}

// Down rolls back the last `n` applied migrations. If n == 0, rolls back 1.
func (m *Migrator) Down(n int) error {
	if n <= 0 {
		n = 1
	}

	db, err := openDB(m.cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db, m.cfg.Driver); err != nil {
		return err
	}

	migrations, err := Scan(m.migrationsDir)
	if err != nil {
		return err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return err
	}

	// Roll back each migration inside its own transaction.
	count := 0
	for i := len(migrations) - 1; i >= 0; i-- {
		mg := migrations[i]
		if _, done := applied[mg.Filename]; !done {
			continue
		}
		if count >= n {
			break
		}

		if mg.DownSQL == "" {
			fmt.Fprintf(m.out, "  ⚠ skipping rollback of %s — no Down section found\n", mg.Filename)
			count++ // still count as "processed" so we stop at n
			continue
		}

		fmt.Fprintf(m.out, "  ◀ Rolling back %s...\n", mg.Filename)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("wolf migrate: begin tx for rollback of %s: %w", mg.Filename, err)
		}

		var txErr error
		statements := splitStatements(mg.DownSQL)
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, txErr = tx.Exec(stmt); txErr != nil {
				break
			}
		}

		if txErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("wolf migrate: failed running Down for %s (rolled back): %w", mg.Filename, txErr)
		}

		// Remove from tracking inside the same transaction.
		if strings.ToLower(m.cfg.Driver) == "postgres" {
			_, txErr = tx.Exec("DELETE FROM wolf_migrations WHERE filename = $1", mg.Filename)
		} else {
			_, txErr = tx.Exec("DELETE FROM wolf_migrations WHERE filename = ?", mg.Filename)
		}
		if txErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("wolf migrate: failed to remove %s from applied list (rolled back): %w", mg.Filename, txErr)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("wolf migrate: commit for rollback of %s: %w", mg.Filename, err)
		}

		fmt.Fprintf(m.out, "  ✓ Rolled back %s\n", mg.Filename)
		count++
	}

	if count == 0 {
		fmt.Fprintln(m.out, "  Nothing to roll back.")
	} else {
		fmt.Fprintf(m.out, "\nwolf migrate: %d migration(s) rolled back ✓\n", count)
	}

	return nil
}

// Fresh drops all tables (via rolling back everything) then re-runs all Up migrations.
func (m *Migrator) Fresh() error {
	db, err := openDB(m.cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db, m.cfg.Driver); err != nil {
		return err
	}

	migrations, err := Scan(m.migrationsDir)
	if err != nil {
		return err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return err
	}

	// Rollback all applied in reverse order
	fmt.Fprintln(m.out, "wolf migrate: dropping all tables...")
	for i := len(migrations) - 1; i >= 0; i-- {
		mg := migrations[i]
		if _, done := applied[mg.Filename]; !done {
			continue
		}
		if mg.DownSQL == "" {
			fmt.Fprintf(m.out, "  ⚠ no Down section for %s — skipping\n", mg.Filename)
			continue
		}
		statements := splitStatements(mg.DownSQL)
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				db.Exec(stmt) // best effort during fresh
			}
		}
		if strings.ToLower(m.cfg.Driver) == "postgres" {
			markRolledBackPostgres(db, mg.Filename)
		} else {
			markRolledBack(db, mg.Filename)
		}
	}

	fmt.Fprintln(m.out, "wolf migrate: re-running all migrations...")
	return m.Up(0)
}

// Status prints the status of every migration file.
func (m *Migrator) Status() error {
	db, err := openDB(m.cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db, m.cfg.Driver); err != nil {
		return err
	}

	migrations, err := Scan(m.migrationsDir)
	if err != nil {
		return err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Fprintln(m.out, "  No migration files found in ./migrations/")
		return nil
	}

	fmt.Fprintf(m.out, "\n  %-45s  %s\n", "Migration", "Status")
	fmt.Fprintf(m.out, "  %s\n", strings.Repeat("─", 70))

	for _, mg := range migrations {
		if ts, done := applied[mg.Filename]; done {
			fmt.Fprintf(m.out, "  ✓ %-43s  Applied (%s)\n", mg.Filename, ts)
		} else {
			fmt.Fprintf(m.out, "  ○ %-43s  Pending\n", mg.Filename)
		}
	}
	fmt.Fprintln(m.out)
	return nil
}

// Make creates a new migration file in the migrations directory.
func (m *Migrator) Make(description string) (string, error) {
	if description == "" {
		return "", fmt.Errorf("wolf migrate: description is required")
	}

	if err := os.MkdirAll(m.migrationsDir, 0755); err != nil {
		return "", fmt.Errorf("wolf migrate: cannot create migrations dir: %w", err)
	}

	filename := MakeFilename(description)
	path := filepath.Join(m.migrationsDir, filename)

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("wolf migrate: %s already exists", path)
	}

	content := MakeTemplate(description)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("wolf migrate: cannot write %s: %w", path, err)
	}

	return path, nil
}

// splitStatements splits a SQL string into individual executable statements.
// Handles semicolons as statement delimiters.
func splitStatements(sql string) []string {
	var stmts []string
	for _, stmt := range strings.Split(sql, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" && !strings.HasPrefix(stmt, "--") {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
