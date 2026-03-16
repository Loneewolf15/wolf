<?php

/**
 * API Key Middleware
 * 
 * Add to Controller for API key authentication
 * Similar to RouteProtection but for API keys
 */
trait APIKeyAuthTrait
{
    private $apiKeyService;
    private $currentApiKey;

    /**
     * Protect route with API key authentication
     * Similar to RouteProtection() but for API keys
     */
    protected function APIKeyProtection(string $requiredScope = null): ?object
    {
        if (!isset($this->apiKeyService)) {
            $this->apiKeyService = new APIKeyService();
        }

        // Get API key from header
        $apiKey = $this->getAPIKeyFromRequest();

        if (!$apiKey) {
            http_response_code(401);
            echo json_encode([
                'status' => false,
                'message' => 'API key required. Please provide X-API-Key header.'
            ]);
            exit;
        }

        // Validate API key
        $keyData = $this->apiKeyService->validateKey($apiKey);

        if (!$keyData) {
            http_response_code(401);
            echo json_encode([
                'status' => false,
                'message' => 'Invalid or expired API key'
            ]);
            exit;
        }

        // Check scope if required
        if ($requiredScope && !$this->apiKeyService->hasScope($keyData, $requiredScope)) {
            http_response_code(403);
            echo json_encode([
                'status' => false,
                'message' => "API key does not have required scope: {$requiredScope}"
            ]);
            exit;
        }

        // Check rate limit
        if (!$this->apiKeyService->checkRateLimit($keyData->id, $keyData)) {
            http_response_code(429);
            echo json_encode([
                'status' => false,
                'message' => 'Rate limit exceeded'
            ]);
            exit;
        }

        // Log usage (at end of request)
        register_shutdown_function([$this, 'logAPIKeyUsage'], $keyData->id);

        $this->currentApiKey = $keyData;

        return $keyData;
    }

    /**
     * Get API key from request headers
     */
    private function getAPIKeyFromRequest(): ?string
    {
        // Check X-API-Key header
        $headers = getallheaders();

        if (isset($headers['X-API-Key'])) {
            return $headers['X-API-Key'];
        }

        if (isset($headers['x-api-key'])) {
            return $headers['x-api-key'];
        }

        // Check query parameter (less secure, use only for testing)
        if (isset($_GET['api_key'])) {
            return $_GET['api_key'];
        }

        return null;
    }

    /**
     * Log API key usage
     */
    public function logAPIKeyUsage(int $keyId): void
    {
        $startTime = $_SERVER['REQUEST_TIME_FLOAT'] ?? microtime(true);
        $responseTime = (microtime(true) - $startTime) * 1000;

        $this->apiKeyService->logUsage($keyId, [
            'endpoint' => $_SERVER['REQUEST_URI'] ?? '',
            'method' => $_SERVER['REQUEST_METHOD'] ?? 'GET',
            'ip' => $_SERVER['REMOTE_ADDR'] ?? null,
            'user_agent' => $_SERVER['HTTP_USER_AGENT'] ?? null,
            'status_code' => http_response_code(),
            'response_time' => $responseTime
        ]);
    }

    /**
     * Get current API key
     */
    protected function getCurrentAPIKey(): ?object
    {
        return $this->currentApiKey ?? null;
    }
}
