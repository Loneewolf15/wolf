<?php

/**
 * Rate Limiting Middleware
 * 
 * Automatically applies rate limiting to all API endpoints
 * Can be used at the application level or per-controller
 */
class RateLimitMiddleware
{
    private $rateLimiter;
    private $excludedEndpoints = [
        'health',  // Health checks should not be rate limited
        'index'    // API index/welcome page
    ];

    public function __construct()
    {
        $this->rateLimiter = new RateLimiter();
    }

    /**
     * Apply rate limiting to the current request
     * 
     * @param string $controller Controller name
     * @param string $method Method name
     * @param mixed $identifier Optional user ID or email
     */
    public function handle($controller, $method, $identifier = null)
    {
        // Skip rate limiting for excluded endpoints
        if (in_array($method, $this->excludedEndpoints)) {
            return true;
        }

        // Map controller methods to rate limit endpoints
        $endpoint = $this->mapEndpoint($controller, $method);

        // Apply rate limit
        return $this->rateLimiter->apply($endpoint, $identifier);
    }

    /**
     * Map controller and method to rate limit endpoint name
     */
    private function mapEndpoint($controller, $method)
    {
        // Normalize controller name
        $controller = strtolower($controller);

        // Map specific methods to rate limit configs
        $endpointMap = [
            'users' => [
                'loginfunc' => 'login',
                'registeruser' => 'register',
                'forgotpassword' => 'forgot_password',
                'updatelocation' => 'update_location',
                'getuser' => 'get_user',
            ]
        ];

        if (isset($endpointMap[$controller][$method])) {
            return $endpointMap[$controller][$method];
        }

        // Default rate limit
        return 'default';
    }

    /**
     * Get rate limiter instance for custom usage
     */
    public function getRateLimiter()
    {
        return $this->rateLimiter;
    }

    /**
     * Add custom rate limit for an endpoint
     */
    public function setCustomLimit($endpoint, $requests, $period, $identifier = 'ip')
    {
        $this->rateLimiter->setLimit($endpoint, $requests, $period, $identifier);
    }

    /**
     * Exclude an endpoint from rate limiting
     */
    public function excludeEndpoint($endpoint)
    {
        if (!in_array($endpoint, $this->excludedEndpoints)) {
            $this->excludedEndpoints[] = $endpoint;
        }
    }
}
