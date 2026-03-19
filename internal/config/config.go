// Package config defines the WolfConfig struct and all default values.
// Every subsystem that needs configuration imports this package —
// no subsystem reads wolf.config directly.
package config

// WolfConfig is the single source of truth for all Wolf project settings.
// It is populated once by the loader and passed through the compiler pipeline.
// Fields use concrete types (int, string, bool) so consumers never need
// to parse strings themselves.
type WolfConfig struct {
	// ---- Project ----
	App    AppConfig
	Server ServerConfig
	DB     DBConfig
	Redis  RedisConfig
	JWT    JWTConfig
	Log    LogConfig
	Build  BuildConfig
}

// AppConfig holds general application identity.
type AppConfig struct {
	Name    string // APP_NAME
	Env     string // APP_ENV: "development" | "staging" | "production"
	Debug   bool   // APP_DEBUG
	Version string // APP_VERSION
	Key     string // APP_KEY — secret key for encryption/signing
}

// ServerConfig controls the HTTP server behaviour.
type ServerConfig struct {
	Host            string // SERVER_HOST (default "0.0.0.0")
	Port            int    // SERVER_PORT (default 2006)
	ReadTimeoutSec  int    // SERVER_READ_TIMEOUT  (default 30)
	WriteTimeoutSec int    // SERVER_WRITE_TIMEOUT (default 30)
	MaxRequestSize  int    // SERVER_MAX_REQUEST_SIZE bytes (default 65536)
	MaxConcurrent   int    // SERVER_MAX_CONCURRENT (default 1024)
	Workers         int    // SERVER_WORKERS: OS threads for http_worker (default = CPU count)
}

// DBConfig controls MySQL connection pooling and credentials.
// This is the section most relevant to the connection pool work.
type DBConfig struct {
	Driver   string // DB_DRIVER (mysql | postgres | mssql)
	Host     string // DB_HOST
	Port     int    // DB_PORT (default 3306)
	Name     string // DB_NAME
	User     string // DB_USER
	Password string // DB_PASSWORD

	// Pool tuning — these are compiled into wolf_runtime as #define values.
	PoolSize    int // DB_POOL_SIZE    — total connections in the pool (default 10)
	PoolMinIdle int // DB_POOL_MIN_IDLE — minimum warm idle connections (default 2)
	PoolTimeout int // DB_POOL_TIMEOUT  — seconds to wait for a free slot (default 30)
	MaxRetries  int // DB_MAX_RETRIES   — reconnect attempts on stale conn (default 3)

	// Future: read replica
	ReadHost string // DB_READ_HOST — empty means all reads go to primary
	ReadPort int    // DB_READ_PORT
}

// RedisConfig controls the Redis connection.
type RedisConfig struct {
	Host     string // REDIS_HOST (default "127.0.0.1")
	Port     int    // REDIS_PORT (default 6379)
	Password string // REDIS_PASSWORD
	DB       int    // REDIS_DB (default 0)
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret    string // JWT_SECRET
	ExpiryMin int    // JWT_EXPIRY_MINUTES (default 60)
	Algorithm string // JWT_ALGORITHM (default "HS256")
}

// LogConfig controls runtime log output.
type LogConfig struct {
	Level  string // LOG_LEVEL: "debug"|"info"|"warning"|"error" (default "info")
	Output string // LOG_OUTPUT: "stdout"|"file" (default "stdout")
	File   string // LOG_FILE path when Output == "file"
}

// BuildConfig controls the compiler and output.
type BuildConfig struct {
	OutDir     string // BUILD_OUT_DIR (default "wolf_out")
	Optimise   bool   // BUILD_OPTIMISE: -O2 vs -O0 (default true in production)
	KeepLL     bool   // BUILD_KEEP_LL: retain .ll file for debugging (default false)
	StrictMode bool   // BUILD_STRICT_MODE
}

// Defaults returns a WolfConfig with every field set to its production default.
// The loader overlays only the fields the user explicitly sets in wolf.config,
// so callers always get a fully populated struct.
func Defaults() *WolfConfig {
	return &WolfConfig{
		App: AppConfig{
			Name:    "wolf-app",
			Env:     "development",
			Debug:   false,
			Version: "1.0.0",
			Key:     "",
		},
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            2006,
			ReadTimeoutSec:  30,
			WriteTimeoutSec: 30,
			MaxRequestSize:  65536,
			MaxConcurrent:   1024,
			Workers:         0, // 0 = auto-detect CPU count at runtime
		},
		DB: DBConfig{
			Host:        "localhost",
			Port:        3306,
			Name:        "",
			User:        "",
			Password:    "",
			PoolSize:    10,
			PoolMinIdle: 2,
			PoolTimeout: 30,
			MaxRetries:  3,
			ReadHost:    "",
			ReadPort:    3306,
		},
		Redis: RedisConfig{
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		JWT: JWTConfig{
			Secret:    "",
			ExpiryMin: 60,
			Algorithm: "HS256",
		},
		Log: LogConfig{
			Level:  "info",
			Output: "stdout",
			File:   "",
		},
		Build: BuildConfig{
			OutDir:     "wolf_out",
			Optimise:   true,
			KeepLL:     false,
			StrictMode: false,
		},
	}
}
