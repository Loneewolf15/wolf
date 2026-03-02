package stdlib

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// WolfEnv implements the Wolf\Env standard library.
type WolfEnv struct{}

// Get returns an environment variable, or empty string.
func (WolfEnv) Get(key string) string {
	return os.Getenv(key)
}

// GetDefault returns an environment variable, or a default value.
func (WolfEnv) GetDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// Set sets an environment variable.
func (WolfEnv) Set(key, value string) error {
	return os.Setenv(key, value)
}

// Unset removes an environment variable.
func (WolfEnv) Unset(key string) error {
	return os.Unsetenv(key)
}

// All returns all environment variables as a map.
func (WolfEnv) All() map[string]string {
	result := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// Has checks if an environment variable is set.
func (WolfEnv) Has(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// LoadFile loads environment variables from a .env file.
func (WolfEnv) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("wolf: failed to load env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

// Require returns an env var or panics if not set.
func (WolfEnv) Require(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("wolf: required environment variable %s is not set", key))
	}
	return val
}
