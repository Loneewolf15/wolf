<?php

/**
 * API Key Service
 * 
 * Manage API keys for third-party integrations
 */
class APIKeyService
{
    private $db;

    // Available scopes
    const SCOPES = [
        'users:read',
        'users:write',
        'payments:read',
        'payments:write',
        'uploads:read',
        'uploads:write',
        'search:read',
        'admin:read',
        'admin:write'
    ];

    public function __construct()
    {
        $this->db = new Database();
    }

    /**
     * Generate new API key
     */
    public function generateKey(int $userId, array $data): ?object
    {
        // Generate secure API key
        $apiKey = $this->createSecureKey();
        $keyPrefix = substr($apiKey, 0, 8);

        // Validate scopes
        $scopes = $data['scopes'] ?? ['users:read'];
        $scopes = $this->validateScopes($scopes);

        // Insert into database
        $this->db->query("INSERT INTO api_keys 
            (user_id, key_name, api_key, key_prefix, scopes, description, 
             rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day, expires_at)
            VALUES 
            (:user_id, :key_name, :api_key, :key_prefix, :scopes, :description,
             :rate_minute, :rate_hour, :rate_day, :expires_at)");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':key_name', $data['name']);
        $this->db->bind(':api_key', $apiKey);
        $this->db->bind(':key_prefix', $keyPrefix);
        $this->db->bind(':scopes', json_encode($scopes));
        $this->db->bind(':description', $data['description'] ?? null);
        $this->db->bind(':rate_minute', $data['rate_limit_minute'] ?? 60);
        $this->db->bind(':rate_hour', $data['rate_limit_hour'] ?? 1000);
        $this->db->bind(':rate_day', $data['rate_limit_day'] ?? 10000);
        $this->db->bind(':expires_at', $data['expires_at'] ?? null);

        if ($this->db->execute()) {
            $keyId = $this->db->lastInsertId();

            // Return key details (API key shown only once!)
            return (object)[
                'id' => $keyId,
                'api_key' => $apiKey,
                'key_prefix' => $keyPrefix,
                'name' => $data['name'],
                'scopes' => $scopes
            ];
        }

        return null;
    }

    /**
     * Validate API key
     */
    public function validateKey(string $apiKey): ?object
    {
        $this->db->query("SELECT * FROM api_keys 
            WHERE api_key = :api_key 
            AND is_active = 1 
            AND (expires_at IS NULL OR expires_at > NOW())");

        $this->db->bind(':api_key', $apiKey);

        $key = $this->db->single();

        if ($key) {
            // Update last used
            $this->updateLastUsed($key->id);

            // Parse scopes
            $key->scopes = json_decode($key->scopes, true);
            $key->permissions = json_decode($key->permissions, true);
        }

        return $key ?: null;
    }

    /**
     * Check if key has scope
     */
    public function hasScope(object $key, string $scope): bool
    {
        $scopes = is_array($key->scopes) ? $key->scopes : json_decode($key->scopes, true);

        // Check exact match
        if (in_array($scope, $scopes)) {
            return true;
        }

        // Check wildcard (e.g., 'users:*' allows 'users:read' and 'users:write')
        foreach ($scopes as $allowedScope) {
            if (str_ends_with($allowedScope, ':*')) {
                $prefix = str_replace(':*', '', $allowedScope);
                if (str_starts_with($scope, $prefix)) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Check rate limit
     */
    public function checkRateLimit(int $keyId, object $limits): bool
    {
        $now = new DateTime();
        $minuteWindow = $now->format('Y-m-d H:i:00');
        $hourWindow = $now->format('Y-m-d H:00:00');
        $dayWindow = $now->format('Y-m-d');

        // Get current usage
        $this->db->query("SELECT * FROM api_key_rate_limits 
            WHERE api_key_id = :key_id 
            AND minute_window = :minute 
            AND hour_window = :hour 
            AND day_window = :day");

        $this->db->bind(':key_id', $keyId);
        $this->db->bind(':minute', $minuteWindow);
        $this->db->bind(':hour', $hourWindow);
        $this->db->bind(':day', $dayWindow);

        $usage = $this->db->single();

        if ($usage) {
            // Check limits
            if ($usage->requests_this_minute >= $limits->rate_limit_per_minute) {
                return false;
            }
            if ($usage->requests_this_hour >= $limits->rate_limit_per_hour) {
                return false;
            }
            if ($usage->requests_this_day >= $limits->rate_limit_per_day) {
                return false;
            }

            // Increment counters
            $this->db->query("UPDATE api_key_rate_limits SET 
                requests_this_minute = requests_this_minute + 1,
                requests_this_hour = requests_this_hour + 1,
                requests_this_day = requests_this_day + 1
                WHERE id = :id");
            $this->db->bind(':id', $usage->id);
            $this->db->execute();
        } else {
            // Create new tracking record
            $this->db->query("INSERT INTO api_key_rate_limits 
                (api_key_id, minute_window, hour_window, day_window, 
                 requests_this_minute, requests_this_hour, requests_this_day)
                VALUES (:key_id, :minute, :hour, :day, 1, 1, 1)");

            $this->db->bind(':key_id', $keyId);
            $this->db->bind(':minute', $minuteWindow);
            $this->db->bind(':hour', $hourWindow);
            $this->db->bind(':day', $dayWindow);
            $this->db->execute();
        }

        return true;
    }

    /**
     * Log API key usage
     */
    public function logUsage(int $keyId, array $data): void
    {
        $this->db->query("INSERT INTO api_key_usage 
            (api_key_id, endpoint, http_method, ip_address, user_agent, status_code, response_time_ms)
            VALUES (:key_id, :endpoint, :method, :ip, :user_agent, :status, :time)");

        $this->db->bind(':key_id', $keyId);
        $this->db->bind(':endpoint', $data['endpoint']);
        $this->db->bind(':method', $data['method']);
        $this->db->bind(':ip', $data['ip'] ?? null);
        $this->db->bind(':user_agent', $data['user_agent'] ?? null);
        $this->db->bind(':status', $data['status_code'] ?? null);
        $this->db->bind(':time', $data['response_time'] ?? null);

        $this->db->execute();
    }

    /**
     * Get user's API keys
     */
    public function getUserKeys(int $userId): array
    {
        $this->db->query("SELECT id, key_name, key_prefix, scopes, is_active, 
            expires_at, last_used_at, created_at 
            FROM api_keys 
            WHERE user_id = :user_id 
            ORDER BY created_at DESC");

        $this->db->bind(':user_id', $userId);

        $keys = $this->db->resultSet();

        foreach ($keys as $key) {
            $key->scopes = json_decode($key->scopes, true);
        }

        return $keys;
    }

    /**
     * Revoke API key
     */
    public function revokeKey(int $keyId, int $userId): bool
    {
        $this->db->query("UPDATE api_keys SET is_active = 0 
            WHERE id = :id AND user_id = :user_id");

        $this->db->bind(':id', $keyId);
        $this->db->bind(':user_id', $userId);

        return $this->db->execute();
    }

    /**
     * Rotate API key (generate new, deactivate old)
     */
    public function rotateKey(int $keyId, int $userId): ?object
    {
        // Get old key details
        $this->db->query("SELECT * FROM api_keys WHERE id = :id AND user_id = :user_id");
        $this->db->bind(':id', $keyId);
        $this->db->bind(':user_id', $userId);
        $oldKey = $this->db->single();

        if (!$oldKey) {
            return null;
        }

        // Create new key with same settings
        $newKey = $this->generateKey($userId, [
            'name' => $oldKey->key_name,
            'scopes' => json_decode($oldKey->scopes, true),
            'description' => $oldKey->description,
            'rate_limit_minute' => $oldKey->rate_limit_per_minute,
            'rate_limit_hour' => $oldKey->rate_limit_per_hour,
            'rate_limit_day' => $oldKey->rate_limit_per_day
        ]);

        // Deactivate old key
        $this->revokeKey($keyId, $userId);

        return $newKey;
    }

    /**
     * Get key usage statistics
     */
    public function getKeyStats(int $keyId): array
    {
        $this->db->query("SELECT 
            COUNT(*) as total_requests,
            AVG(response_time_ms) as avg_response_time,
            COUNT(DISTINCT DATE(created_at)) as active_days
            FROM api_key_usage 
            WHERE api_key_id = :key_id");

        $this->db->bind(':key_id', $keyId);

        return (array)$this->db->single();
    }

    /**
     * Create secure API key
     */
    private function createSecureKey(): string
    {
        // Format: pk_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
        $prefix = 'pk_live_';
        $randomBytes = bin2hex(random_bytes(24));

        return $prefix . $randomBytes;
    }

    /**
     * Validate scopes
     */
    private function validateScopes(array $scopes): array
    {
        return array_filter($scopes, function ($scope) {
            return in_array($scope, self::SCOPES) || str_ends_with($scope, ':*');
        });
    }

    /**
     * Update last used timestamp
     */
    private function updateLastUsed(int $keyId): void
    {
        $this->db->query("UPDATE api_keys SET last_used_at = NOW() WHERE id = :id");
        $this->db->bind(':id', $keyId);
        $this->db->execute();
    }
}
