package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Load finds and parses wolf.config starting from projectRoot, then overlays
// environment variables. Returns a fully populated *WolfConfig.
// If no wolf.config is found, the defaults are returned unchanged — this is
// intentional so `wolf run hello.wolf` always works with zero config.
func Load(projectRoot string) (*WolfConfig, error) {
	cfg := Defaults()

	// 1. Find wolf.config (walk up from projectRoot)
	configPath := findConfigFile(projectRoot)
	if configPath != "" {
		if err := parseINI(configPath, cfg); err != nil {
			return nil, fmt.Errorf("wolf.config: %w", err)
		}
	}

	// 2. Overlay environment variables (env always wins over file)
	overlayEnv(cfg)

	// 3. Validate
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("wolf.config validation: %w", err)
	}

	// 4. Auto-detect worker count if not set
	if cfg.Server.Workers == 0 {
		cfg.Server.Workers = runtime.NumCPU()
	}

	return cfg, nil
}

// findConfigFile walks up from dir looking for wolf.config.
// Returns the path if found, empty string if not found.
func findConfigFile(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		candidate := filepath.Join(dir, "wolf.config")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return ""
}

// parseINI reads an INI-style wolf.config file.
// Format:
//
//	[section]
//	key = value   # inline comments allowed
//	key = value
//
// Section names are case-insensitive. Keys are case-insensitive.
// Values are trimmed of surrounding whitespace and optional inline comments.
func parseINI(path string, cfg *WolfConfig) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	var section string
	lineNum := 0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()

		// Strip inline comment and trim
		line := stripComment(raw)
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Section header: [database]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}

		// Key = value
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("line %d: expected 'key = value', got: %q", lineNum, raw)
		}

		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])

		if err := applyKey(cfg, section, key, val, lineNum); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// stripComment removes everything from the first unquoted # character.
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

// applyKey sets a single config field identified by (section, key) to val.
func applyKey(cfg *WolfConfig, section, key, val string, line int) error {
	errorf := func(f string, a ...any) error {
		return fmt.Errorf("line %d [%s] %s: %s", line, section, key, fmt.Sprintf(f, a...))
	}

	switch section {
	case "app":
		switch key {
		case "name":
			cfg.App.Name = val
		case "env":
			cfg.App.Env = val
		case "debug":
			b, err := parseBool(val)
			if err != nil {
				return errorf("expected true/false, got %q", val)
			}
			cfg.App.Debug = b
		case "version":
			cfg.App.Version = val
		case "key":
			cfg.App.Key = val
		default:
			return errorf("unknown key")
		}

	case "server":
		switch key {
		case "host":
			cfg.Server.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.Port = n
		case "read_timeout":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.ReadTimeoutSec = n
		case "write_timeout":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.WriteTimeoutSec = n
		case "max_request_size":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.MaxRequestSize = n
		case "max_concurrent":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.MaxConcurrent = n
		case "workers":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Server.Workers = n
		default:
			return errorf("unknown key")
		}

	case "database", "db":
		switch key {
		case "host":
			cfg.DB.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.Port = n
		case "name":
			cfg.DB.Name = val
		case "user":
			cfg.DB.User = val
		case "password":
			cfg.DB.Password = val
		case "pool_size":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.PoolSize = n
		case "pool_min_idle":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.PoolMinIdle = n
		case "pool_timeout":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.PoolTimeout = n
		case "max_retries":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.MaxRetries = n
		case "read_host":
			cfg.DB.ReadHost = val
		case "read_port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.DB.ReadPort = n
		default:
			return errorf("unknown key")
		}

	case "redis":
		switch key {
		case "host":
			cfg.Redis.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Redis.Port = n
		case "password":
			cfg.Redis.Password = val
		case "db":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.Redis.DB = n
		default:
			return errorf("unknown key")
		}

	case "jwt":
		switch key {
		case "secret":
			cfg.JWT.Secret = val
		case "expiry_minutes", "expiry":
			n, err := strconv.Atoi(val)
			if err != nil {
				return errorf("expected integer, got %q", val)
			}
			cfg.JWT.ExpiryMin = n
		case "algorithm":
			cfg.JWT.Algorithm = val
		default:
			return errorf("unknown key")
		}

	case "log":
		switch key {
		case "level":
			cfg.Log.Level = val
		case "output":
			cfg.Log.Output = val
		case "file":
			cfg.Log.File = val
		default:
			return errorf("unknown key")
		}

	case "build":
		switch key {
		case "out_dir":
			cfg.Build.OutDir = val
		case "optimise", "optimize":
			b, err := parseBool(val)
			if err != nil {
				return errorf("expected true/false, got %q", val)
			}
			cfg.Build.Optimise = b
		case "keep_ll":
			b, err := parseBool(val)
			if err != nil {
				return errorf("expected true/false, got %q", val)
			}
			cfg.Build.KeepLL = b
		case "strict_mode":
			b, err := parseBool(val)
			if err != nil {
				return errorf("expected true/false, got %q", val)
			}
			cfg.Build.StrictMode = b
		default:
			return errorf("unknown key")
		}

	case "":
		return fmt.Errorf("line %d: key %q is outside any section", line, key)

	default:
		// Unknown section — warn but don't error (forward compatibility)
		// In strict mode this could become an error later.
	}

	return nil
}

// overlayEnv reads well-known environment variables and overrides cfg fields.
// Env vars always beat wolf.config — this is the standard 12-factor pattern.
// Only variables that are actually set in the environment are applied.
func overlayEnv(cfg *WolfConfig) {
	strEnv := func(key string, dest *string) {
		if v, ok := os.LookupEnv(key); ok {
			*dest = v
		}
	}
	intEnv := func(key string, dest *int) {
		if v, ok := os.LookupEnv(key); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	boolEnv := func(key string, dest *bool) {
		if v, ok := os.LookupEnv(key); ok {
			if b, err := parseBool(v); err == nil {
				*dest = b
			}
		}
	}

	// App
	strEnv("APP_NAME", &cfg.App.Name)
	strEnv("APP_ENV", &cfg.App.Env)
	boolEnv("APP_DEBUG", &cfg.App.Debug)
	strEnv("APP_VERSION", &cfg.App.Version)
	strEnv("APP_KEY", &cfg.App.Key)

	// Server
	strEnv("SERVER_HOST", &cfg.Server.Host)
	intEnv("SERVER_PORT", &cfg.Server.Port)
	intEnv("SERVER_READ_TIMEOUT", &cfg.Server.ReadTimeoutSec)
	intEnv("SERVER_WRITE_TIMEOUT", &cfg.Server.WriteTimeoutSec)
	intEnv("SERVER_MAX_REQUEST_SIZE", &cfg.Server.MaxRequestSize)
	intEnv("SERVER_MAX_CONCURRENT", &cfg.Server.MaxConcurrent)
	intEnv("SERVER_WORKERS", &cfg.Server.Workers)

	// Database
	strEnv("DB_HOST", &cfg.DB.Host)
	intEnv("DB_PORT", &cfg.DB.Port)
	strEnv("DB_NAME", &cfg.DB.Name)
	strEnv("DB_USER", &cfg.DB.User)
	strEnv("DB_PASSWORD", &cfg.DB.Password)
	intEnv("DB_POOL_SIZE", &cfg.DB.PoolSize)
	intEnv("DB_POOL_MIN_IDLE", &cfg.DB.PoolMinIdle)
	intEnv("DB_POOL_TIMEOUT", &cfg.DB.PoolTimeout)
	intEnv("DB_MAX_RETRIES", &cfg.DB.MaxRetries)
	strEnv("DB_READ_HOST", &cfg.DB.ReadHost)
	intEnv("DB_READ_PORT", &cfg.DB.ReadPort)

	// Redis
	strEnv("REDIS_HOST", &cfg.Redis.Host)
	intEnv("REDIS_PORT", &cfg.Redis.Port)
	strEnv("REDIS_PASSWORD", &cfg.Redis.Password)
	intEnv("REDIS_DB", &cfg.Redis.DB)

	// JWT
	strEnv("JWT_SECRET", &cfg.JWT.Secret)
	intEnv("JWT_EXPIRY_MINUTES", &cfg.JWT.ExpiryMin)
	strEnv("JWT_ALGORITHM", &cfg.JWT.Algorithm)

	// Log
	strEnv("LOG_LEVEL", &cfg.Log.Level)
	strEnv("LOG_OUTPUT", &cfg.Log.Output)
	strEnv("LOG_FILE", &cfg.Log.File)

	// Build
	strEnv("BUILD_OUT_DIR", &cfg.Build.OutDir)
	boolEnv("BUILD_OPTIMISE", &cfg.Build.Optimise)
	boolEnv("BUILD_KEEP_LL", &cfg.Build.KeepLL)
	boolEnv("BUILD_STRICT_MODE", &cfg.Build.StrictMode)
}

// validate checks that required fields are coherent.
// It does NOT require credentials to be set — a project may not use the DB.
func validate(cfg *WolfConfig) error {
	validEnvs := map[string]bool{"development": true, "staging": true, "production": true}
	if !validEnvs[cfg.App.Env] {
		return fmt.Errorf("[app] env must be development|staging|production, got %q", cfg.App.Env)
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("[server] port must be 1–65535, got %d", cfg.Server.Port)
	}

	if cfg.DB.PoolSize < 1 {
		return fmt.Errorf("[database] pool_size must be >= 1, got %d", cfg.DB.PoolSize)
	}
	if cfg.DB.PoolMinIdle < 0 {
		return fmt.Errorf("[database] pool_min_idle must be >= 0, got %d", cfg.DB.PoolMinIdle)
	}
	if cfg.DB.PoolMinIdle > cfg.DB.PoolSize {
		return fmt.Errorf("[database] pool_min_idle (%d) cannot exceed pool_size (%d)",
			cfg.DB.PoolMinIdle, cfg.DB.PoolSize)
	}
	if cfg.DB.PoolTimeout < 1 {
		return fmt.Errorf("[database] pool_timeout must be >= 1 second, got %d", cfg.DB.PoolTimeout)
	}

	validAlgos := map[string]bool{"HS256": true, "HS384": true, "HS512": true}
	if cfg.JWT.Algorithm != "" && !validAlgos[cfg.JWT.Algorithm] {
		return fmt.Errorf("[jwt] algorithm must be HS256|HS384|HS512, got %q", cfg.JWT.Algorithm)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warning": true, "error": true}
	if !validLevels[cfg.Log.Level] {
		return fmt.Errorf("[log] level must be debug|info|warning|error, got %q", cfg.Log.Level)
	}

	return nil
}

// parseBool accepts true/false/1/0/yes/no (case-insensitive).
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("cannot parse %q as bool", s)
}