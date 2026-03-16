<?php
/**
 * RateLimiter Service
 * 
 * Implements Token Bucket algorithm for rate limiting
 * Supports per-endpoint, per-user, and per-IP rate limiting
 * Provides rate limit headers for API responses
 */
class RateLimiter
{
    private $cache;
    
    // Rate limit configurations per endpoint
    private $limits = [
        'login' => [
            'requests' => 60,
            'period' => 3600,        // 1 hour
            'identifier' => 'ip'     // Rate limit by IP
        ],
        'register' => [
            'requests' => 10,
            'period' => 3600,        // 1 hour
            'identifier' => 'ip'
        ],
        'forgot_password' => [
            'requests' => 5,
            'period' => 3600,        // 1 hour
            'identifier' => 'email'  // Rate limit by email
        ],
        'update_location' => [
            'requests' => 100,
            'period' => 3600,        // 1 hour
            'identifier' => 'user'   // Rate limit by user ID
        ],
        'get_user' => [
            'requests' => 1000,
            'period' => 3600,        // 1 hour
            'identifier' => 'user'
        ],
        'default' => [
            'requests' => 100,
            'period' => 60,          // 1 minute
            'identifier' => 'ip'
        ]
    ];

    public function __construct($cache = null)
    {
        $this->cache = $cache ?? new Cache();
    }

    /**
     * Check if request should be allowed
     * 
     * @param string $endpoint Endpoint name (e.g., 'login', 'register')
     * @param mixed $identifier User ID, email, or IP address
     * @return array ['allowed' => bool, 'headers' => array, 'retry_after' => int]
     */
    public function checkLimit($endpoint, $identifier = null)
    {
        // Get rate limit config for endpoint
        $config = $this->limits[$endpoint] ?? $this->limits['default'];
        
        // Auto-detect identifier if not provided
        if ($identifier === null) {
            $identifier = $this->getDefaultIdentifier($config['identifier']);
        }

        // Create cache key
        $key = $this->getCacheKey($endpoint, $identifier);
        
        // Get current bucket state
        $bucket = $this->getBucket($key, $config);
        
        // Calculate tokens
        $now = microtime(true);
        $timePassed = $now - $bucket['last_refill'];
        $tokensToAdd = ($timePassed / $config['period']) * $config['requests'];
        
        // Refill bucket
        $bucket['tokens'] = min(
            $config['requests'],
            $bucket['tokens'] + $tokensToAdd
        );
        $bucket['last_refill'] = $now;

        // Check if request can be allowed
        $allowed = $bucket['tokens'] >= 1;
        
        if ($allowed) {
            // Consume one token
            $bucket['tokens'] -= 1;
        }

        // Save bucket state
        $this->saveBucket($key, $bucket, $config['period']);

        // Calculate reset time
        $resetTime = time() + $config['period'];
        $retryAfter = $allowed ? 0 : ceil($config['period'] * (1 - $bucket['tokens']) / $config['requests']);

        return [
            'allowed' => $allowed,
            'headers' => $this->getHeaders($config['requests'], (int)$bucket['tokens'], $resetTime),
            'retry_after' => $retryAfter,
            'limit' => $config['requests'],
            'remaining' => max(0, (int)$bucket['tokens']),
            'reset' => $resetTime
        ];
    }

    /**
     * Apply rate limit to current request
     * Returns true if allowed, exits with 429 if blocked
     * 
     * @param string $endpoint Endpoint name
     * @param mixed $identifier Optional identifier
     * @return bool
     */
    public function apply($endpoint, $identifier = null)
    {
        $result = $this->checkLimit($endpoint, $identifier);
        
        // Always add headers
        $this->setHeaders($result['headers']);
        
        if (!$result['allowed']) {
            // Rate limit exceeded
            http_response_code(429);
            if ($result['retry_after'] > 0) {
                header("Retry-After: {$result['retry_after']}");
            }
            
            echo json_encode([
                'status' => false,
                'message' => 'Rate limit exceeded. Too many requests.',
                'error_code' => 'RATE_LIMIT_EXCEEDED',
                'retry_after' => $result['retry_after'],
                'limit' => $result['limit'],
                'reset' => date('Y-m-d H:i:s', $result['reset'])
            ]);
            exit;
        }
        
        return true;
    }

    /**
     * Get token bucket from cache
     */
    private function getBucket($key, $config)
    {
        $bucket = $this->cache->get($key);
        
        if ($bucket === false) {
            // Initialize new bucket
            return [
                'tokens' => $config['requests'],
                'last_refill' => microtime(true)
            ];
        }
        
        return $bucket;
    }

    /**
     * Save bucket state to cache
     */
    private function saveBucket($key, $bucket, $ttl)
    {
        $this->cache->set($key, $bucket, $ttl + 60); // TTL + 60s buffer
    }

    /**
     * Generate cache key for rate limit bucket
     */
    private function getCacheKey($endpoint, $identifier)
    {
        return "rate_limit:{$endpoint}:" . md5($identifier);
    }

    /**
     * Get default identifier based on type
     */
    private function getDefaultIdentifier($type)
    {
        switch ($type) {
            case 'ip':
                return $_SERVER['REMOTE_ADDR'] ?? 'unknown';
            case 'user':
                // Will need to be set by controller
                return 'anonymous';
            case 'email':
                return 'unspecified';
            default:
                return $_SERVER['REMOTE_ADDR'] ?? 'unknown';
        }
    }

    /**
     * Generate rate limit headers
     */
    private function getHeaders($limit, $remaining, $reset)
    {
        return [
            'X-RateLimit-Limit' => $limit,
            'X-RateLimit-Remaining' => max(0, $remaining),
            'X-RateLimit-Reset' => $reset
        ];
    }

    /**
     * Set headers in response
     */
    private function setHeaders($headers)
    {
        foreach ($headers as $name => $value) {
            header("{$name}: {$value}");
        }
    }

    /**
     * Reset rate limit for identifier (useful for testing or admin override)
     */
    public function reset($endpoint, $identifier)
    {
        $key = $this->getCacheKey($endpoint, $identifier);
        $this->cache->del($key);
    }

    /**
     * Get current rate limit status without consuming tokens
     */
    public function getStatus($endpoint, $identifier = null)
    {
        $config = $this->limits[$endpoint] ?? $this->limits['default'];
        
        if ($identifier === null) {
            $identifier = $this->getDefaultIdentifier($config['identifier']);
        }

        $key = $this->getCacheKey($endpoint, $identifier);
        $bucket = $this->getBucket($key, $config);
        
        // Calculate current tokens without modifying
        $now = microtime(true);
        $timePassed = $now - $bucket['last_refill'];
        $tokensToAdd = ($timePassed / $config['period']) * $config['requests'];
        $currentTokens = min($config['requests'], $bucket['tokens'] + $tokensToAdd);

        return [
            'limit' => $config['requests'],
            'remaining' => (int)$currentTokens,
            'reset' => time() + $config['period'],
            'identifier' => $identifier
        ];
    }

    /**
     * Update rate limit configuration for an endpoint
     */
    public function setLimit($endpoint, $requests, $period, $identifier = 'ip')
    {
        $this->limits[$endpoint] = [
            'requests' => $requests,
            'period' => $period,
            'identifier' => $identifier
        ];
    }
}
