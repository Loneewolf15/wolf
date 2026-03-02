package stdlib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- DBConfig Tests ---

func TestDBConfigDSN_MySQL(t *testing.T) {
	cfg := &DBConfig{
		Driver: "mysql", Host: "localhost", Port: 3306,
		User: "root", Pass: "secret", Name: "wolfdb",
	}
	dsn := cfg.DSN()
	expected := "root:secret@tcp(localhost:3306)/wolfdb?parseTime=true"
	if dsn != expected {
		t.Errorf("Expected '%s', got '%s'", expected, dsn)
	}
}

func TestDBConfigDSN_Postgres(t *testing.T) {
	cfg := &DBConfig{
		Driver: "postgres", Host: "localhost", Port: 5432,
		User: "admin", Pass: "pass", Name: "wolfdb",
	}
	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("Expected non-empty DSN")
	}
	// Should contain sslmode=disable by default
	if !containsStr(dsn, "sslmode=disable") {
		t.Error("Expected sslmode=disable in postgres DSN")
	}
}

func TestDBConfigDSN_SQLite(t *testing.T) {
	cfg := &DBConfig{Driver: "sqlite3", Name: "/tmp/wolf.db"}
	if cfg.DSN() != "/tmp/wolf.db" {
		t.Errorf("Expected '/tmp/wolf.db', got '%s'", cfg.DSN())
	}
}

func TestLoadDBConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wolf.config.json")

	cfgData := `{
		"database": {
			"driver": "mysql",
			"host": "127.0.0.1",
			"port": 3306,
			"user": "wolf",
			"pass": "secret",
			"name": "wolfdb"
		}
	}`
	os.WriteFile(configPath, []byte(cfgData), 0644)

	cfg, err := LoadDBConfig(configPath)
	if err != nil {
		t.Fatalf("LoadDBConfig failed: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Errorf("Expected 'mysql', got '%s'", cfg.Driver)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected '127.0.0.1', got '%s'", cfg.Host)
	}
}

func TestLoadDBConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wolf.config.json")

	cfgData := `{"database": {"host": "db.example.com", "name": "mydb"}}`
	os.WriteFile(configPath, []byte(cfgData), 0644)

	cfg, err := LoadDBConfig(configPath)
	if err != nil {
		t.Fatalf("LoadDBConfig failed: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Error("Expected default driver 'mysql'")
	}
	if cfg.Port != 3306 {
		t.Error("Expected default port 3306")
	}
}

func TestLoadDBConfigMissing(t *testing.T) {
	_, err := LoadDBConfig("/nonexistent/wolf.config.json")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

func TestLoadDBConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "wolf.config.json")
	os.WriteFile(configPath, []byte("{invalid json"), 0644)

	_, err := LoadDBConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// --- Database Object Tests ---

func TestNewDatabase(t *testing.T) {
	db := NewDatabase()
	if db == nil {
		t.Fatal("Expected non-nil Database")
	}
	if db.bindings == nil {
		t.Error("Expected initialized bindings map")
	}
}

func TestQuerySetsSQL(t *testing.T) {
	db := NewDatabase()
	db.Query("SELECT * FROM users")
	if db.sql != "SELECT * FROM users" {
		t.Errorf("Expected SQL to be set, got '%s'", db.sql)
	}
}

func TestBindAddsParam(t *testing.T) {
	db := NewDatabase()
	db.Query("SELECT * FROM users WHERE id = :id")
	db.Bind(":id", 42)
	if db.bindings[":id"] != 42 {
		t.Error("Expected binding for :id")
	}
}

func TestResolveBindings(t *testing.T) {
	db := NewDatabase()
	db.Query("SELECT * FROM users WHERE name = :name AND age = :age")
	db.Bind(":name", "Wolf")
	db.Bind(":age", 25)

	query, args := db.resolveBindings()
	if !containsStr(query, "?") {
		t.Error("Expected ? in resolved query")
	}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestExecuteWithoutConnection(t *testing.T) {
	db := NewDatabase()
	db.Query("SELECT 1")
	err := db.Execute()
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestRowCountDefault(t *testing.T) {
	db := NewDatabase()
	if db.RowCount() != 0 {
		t.Error("Expected 0 row count initially")
	}
}

func TestCloseNilDB(t *testing.T) {
	db := NewDatabase()
	err := db.Close()
	if err != nil {
		t.Error("Close on nil DB should not error")
	}
}

// --- Redis Tests (using MemoryRedis) ---

func TestRedisSetGet(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	mem := NewMemoryRedis()
	r.SetConn(mem)

	r.Set("greeting", "hello", 0)
	val := r.Get("greeting")
	if val == "" {
		t.Fatalf("Get failed to retrieve key")
	}
	if val != "hello" {
		t.Errorf("Expected 'hello', got '%s'", val)
	}
}

func TestRedisDel(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	mem := NewMemoryRedis()
	r.SetConn(mem)

	r.Set("key1", "val1", 0)
	count, _ := r.Del("key1")
	if count != 1 {
		t.Errorf("Expected 1 deleted, got %d", count)
	}

	val := r.Get("key1")
	if val != "" {
		t.Error("Expected empty string after delete")
	}
}

func TestRedisExists(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	mem := NewMemoryRedis()
	r.SetConn(mem)

	r.Set("exists-key", "val", 0)
	count, _ := r.Exists("exists-key")
	if count != 1 {
		t.Error("Expected key to exist")
	}
	count, _ = r.Exists("no-key")
	if count != 0 {
		t.Error("Expected key to not exist")
	}
}

func TestRedisHash(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	mem := NewMemoryRedis()
	r.SetConn(mem)

	r.HSet("user:1", "name", "Wolf")
	r.HSet("user:1", "age", 3)

	name, err := r.HGet("user:1", "name")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if name != "Wolf" {
		t.Errorf("Expected 'Wolf', got '%s'", name)
	}

	all, _ := r.HGetAll("user:1")
	if len(all) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(all))
	}
}

func TestRedisNotConnected(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	val := r.Get("key")
	if val != "" {
		t.Error("Expected empty string when not connected")
	}
}

func TestRedisClose(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	mem := NewMemoryRedis()
	r.SetConn(mem)
	if err := r.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewRedisDefaults(t *testing.T) {
	r := NewRedis(&RedisConfig{})
	if r.host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", r.host)
	}
	if r.port != 6379 {
		t.Errorf("Expected default port 6379, got %d", r.port)
	}
}

// --- Helpers ---

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Suppress unused import warnings
var _ = json.Marshal
var _ = fmt.Sprintf
