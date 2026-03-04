// Package stdlib provides Wolf's standard library runtime implementations.
// These are Go implementations that Wolf programs compile against.
package stdlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Database implements the Wolf Database library class.
// It wraps Go's database/sql with the Wolf PDO-style API:
//
//	$this->db->query("SQL")
//	$this->db->bind(":param", $value)
//	$this->db->execute()
//	$this->db->resultSet() / $this->db->single() / $this->db->rowCount()
type Database struct {
	db       *sql.DB
	stmt     *sql.Stmt
	sql      string
	bindings map[string]interface{}
	result   *sql.Rows
	lastErr  error
	rowCount int64
}

// DBConfig holds database connection configuration.
type DBConfig struct {
	Driver  string `json:"driver"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
	Name    string `json:"name"`
	SSLMode string `json:"ssl_mode,omitempty"`
}

// DSN returns the driver-specific connection string.
func (c *DBConfig) DSN() string {
	switch c.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.User, c.Pass, c.Host, c.Port, c.Name)
	case "postgres":
		sslMode := c.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Pass, c.Name, sslMode)
	case "sqlite3":
		return c.Name
	default:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Pass, c.Host, c.Port, c.Name)
	}
}

// LoadDBConfig reads database config from wolf.config.json.
func LoadDBConfig(path string) (*DBConfig, error) {
	type configFile struct {
		Database DBConfig `json:"database"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mysql"
	}
	if cfg.Database.Port == 0 {
		switch cfg.Database.Driver {
		case "mysql":
			cfg.Database.Port = 3306
		case "postgres":
			cfg.Database.Port = 5432
		}
	}

	return &cfg.Database, nil
}

// NewDatabase creates a new Database instance.
func NewDatabase() *Database {
	return &Database{
		bindings: make(map[string]interface{}),
	}
}

// NewDatabaseFromDB creates a Database wrapping an existing *sql.DB.
func NewDatabaseFromDB(db *sql.DB) *Database {
	return &Database{
		db:       db,
		bindings: make(map[string]interface{}),
	}
}

// Connect opens the database connection.
func (d *Database) Connect(config *DBConfig) error {
	db, err := sql.Open(config.Driver, config.DSN())
	if err != nil {
		return fmt.Errorf("wolf: database connection failed: %w", err)
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("wolf: database ping failed: %w", err)
	}
	d.db = db
	return nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// DB returns the underlying *sql.DB.
func (d *Database) DB() *sql.DB {
	return d.db
}

// ========== Query Builder API (Wolf PDO style) ==========

// Query sets the SQL query string.
func (d *Database) Query(sqlStr string) {
	d.sql = sqlStr
	d.bindings = make(map[string]interface{})
	d.result = nil
	d.lastErr = nil
	d.rowCount = 0
}

// Bind binds a named parameter.
func (d *Database) Bind(param string, value interface{}) {
	d.bindings[param] = value
}

// Execute runs the prepared query.
func (d *Database) Execute() error {
	if d.db == nil {
		return fmt.Errorf("wolf: database not connected")
	}

	query, args := d.resolveBindings()

	upperSQL := strings.TrimSpace(strings.ToUpper(d.sql))
	if strings.HasPrefix(upperSQL, "SELECT") {
		rows, err := d.db.Query(query, args...)
		if err != nil {
			d.lastErr = err
			return err
		}
		d.result = rows
	} else {
		result, err := d.db.Exec(query, args...)
		if err != nil {
			d.lastErr = err
			return err
		}
		d.rowCount, _ = result.RowsAffected()
	}

	return nil
}

// ResultSet returns all rows as a slice of maps.
func (d *Database) ResultSet() ([]map[string]interface{}, error) {
	if d.result == nil {
		return nil, fmt.Errorf("wolf: no result set available")
	}
	defer d.result.Close()

	columns, err := d.result.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for d.result.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := d.result.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	d.rowCount = int64(len(results))
	return results, d.result.Err()
}

// Single returns the first row as a map.
func (d *Database) Single() (map[string]interface{}, error) {
	results, err := d.ResultSet()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// RowCount returns the number of affected/returned rows.
func (d *Database) RowCount() int64 {
	return d.rowCount
}

// LastError returns the last error.
func (d *Database) LastError() error {
	return d.lastErr
}

// ========== Convenience Methods ==========

// FetchAll runs a SELECT and returns all rows.
func (d *Database) FetchAll(sqlStr string, bindings map[string]interface{}) ([]map[string]interface{}, error) {
	d.Query(sqlStr)
	for k, v := range bindings {
		d.Bind(k, v)
	}
	if err := d.Execute(); err != nil {
		return nil, err
	}
	return d.ResultSet()
}

// FetchOne runs a SELECT and returns the first row.
func (d *Database) FetchOne(sqlStr string, bindings map[string]interface{}) (map[string]interface{}, error) {
	d.Query(sqlStr)
	for k, v := range bindings {
		d.Bind(k, v)
	}
	if err := d.Execute(); err != nil {
		return nil, err
	}
	return d.Single()
}

// ========== Internal ==========

// resolveBindings converts named :param bindings to positional ? args.
func (d *Database) resolveBindings() (string, []interface{}) {
	query := d.sql
	var args []interface{}

	type binding struct {
		param string
		value interface{}
	}
	var sorted []binding
	for k, v := range d.bindings {
		sorted = append(sorted, binding{k, v})
	}
	// Sort descending by length to avoid partial replacements
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if len(sorted[j].param) > len(sorted[i].param) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, b := range sorted {
		param := b.param
		if !strings.HasPrefix(param, ":") {
			param = ":" + param
		}
		if strings.Contains(query, param) {
			query = strings.Replace(query, param, "?", 1)
			args = append(args, b.value)
		}
	}

	return query, args
}
