<?php

/**
 * Environment Variable Loader
 * 
 * Loads environment variables from .env file
 * Simple implementation without external dependencies
 */

class Env
{
    /**
     * Load environment variables from .env file
     */
    public static function load(string $path = null): void
    {
        if ($path === null) {
            $path = APPROOT . '/.env';
        }

        if (!file_exists($path)) {
            // .env file not found - use defaults or environment variables
            error_log("Warning: .env file not found at {$path}");
            return;
        }

        $lines = file($path, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);

        foreach ($lines as $line) {
            // Skip comments
            if (strpos(trim($line), '#') === 0) {
                continue;
            }

            // Parse KEY=VALUE
            if (strpos($line, '=') !== false) {
                list($key, $value) = explode('=', $line, 2);

                $key = trim($key);
                $value = trim($value);

                // Remove quotes from value
                $value = trim($value, '"\'');

                // Set environment variable if not already set
                if (!getenv($key)) {
                    putenv("{$key}={$value}");
                    $_ENV[$key] = $value;
                    $_SERVER[$key] = $value;
                }
            }
        }
    }

    /**
     * Get environment variable with default fallback
     */
    public static function get(string $key, $default = null)
    {
        $value = getenv($key);

        if ($value === false) {
            return $default;
        }

        // Convert boolean strings
        if (strtolower($value) === 'true') {
            return true;
        }

        if (strtolower($value) === 'false') {
            return false;
        }

        return $value;
    }
}
