package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/wolflang/wolf/internal/config"
)

// openDB opens a *sql.DB connection using the Wolf project config.
func openDB(cfg config.DBConfig) (*sql.DB, error) {
	dsn, driverName, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("wolf migrate: open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("wolf migrate: cannot connect to database (%s@%s:%d/%s): %w",
			cfg.User, cfg.Host, cfg.Port, cfg.Name, err)
	}

	return db, nil
}

// buildDSN constructs a driver-specific DSN from config.
func buildDSN(cfg config.DBConfig) (dsn, driverName string, err error) {
	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		// user:pass@tcp(host:port)/dbname?parseTime=true
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
		driverName = "mysql"
	case "postgres":
		// host=... port=... user=... password=... dbname=... sslmode=disable
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
		driverName = "postgres"
	case "mssql":
		// MSSQL not yet implemented with real driver — provide a helpful error
		err = fmt.Errorf("wolf migrate: MSSQL migrations are not yet supported; use mysql or postgres")
	default:
		err = fmt.Errorf("wolf migrate: unknown driver %q; must be mysql|postgres|mssql", cfg.Driver)
	}
	return
}

// ensureMigrationsTable creates the wolf_migrations tracking table if it doesn't exist.
func ensureMigrationsTable(db *sql.DB, driver string) error {
	var createSQL string
	switch strings.ToLower(driver) {
	case "postgres":
		createSQL = `
CREATE TABLE IF NOT EXISTS wolf_migrations (
    id          SERIAL PRIMARY KEY,
    filename    VARCHAR(255) NOT NULL UNIQUE,
    applied_at  TIMESTAMP NOT NULL DEFAULT NOW()
);`
	default: // mysql
		createSQL = `
CREATE TABLE IF NOT EXISTS wolf_migrations (
    id          INT AUTO_INCREMENT PRIMARY KEY,
    filename    VARCHAR(255) NOT NULL UNIQUE,
    applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	}

	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("wolf migrate: cannot create wolf_migrations table: %w", err)
	}
	return nil
}

// loadApplied returns a set of filenames that have already been applied.
func loadApplied(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT filename, applied_at FROM wolf_migrations ORDER BY applied_at")
	if err != nil {
		return nil, fmt.Errorf("wolf migrate: query wolf_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]string)
	for rows.Next() {
		var filename, appliedAt string
		if err := rows.Scan(&filename, &appliedAt); err != nil {
			return nil, err
		}
		applied[filename] = appliedAt
	}
	return applied, rows.Err()
}

// markApplied inserts a filename into wolf_migrations.
func markApplied(db *sql.DB, filename string) error {
	_, err := db.Exec("INSERT INTO wolf_migrations (filename) VALUES (?)", filename)
	return err
}

// markAppliedPostgres inserts a filename into wolf_migrations (Postgres uses $1 placeholders).
func markAppliedPostgres(db *sql.DB, filename string) error {
	_, err := db.Exec("INSERT INTO wolf_migrations (filename) VALUES ($1)", filename)
	return err
}

// markRolledBack removes a filename from wolf_migrations.
func markRolledBack(db *sql.DB, filename string) error {
	_, err := db.Exec("DELETE FROM wolf_migrations WHERE filename = ?", filename)
	return err
}

// markRolledBackPostgres removes a filename from wolf_migrations (Postgres).
func markRolledBackPostgres(db *sql.DB, filename string) error {
	_, err := db.Exec("DELETE FROM wolf_migrations WHERE filename = $1", filename)
	return err
}
