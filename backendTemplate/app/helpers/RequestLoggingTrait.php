<?php

/**
 * Request Logging Middleware
 * 
 * Automatically logs all API requests and responses
 * Add to base Controller class
 */
trait RequestLoggingTrait
{
    private $logger;
    private $requestStartTime;

    /**
     * Initialize request logging
     */
    protected function initializeRequestLogging(): void
    {
        $this->logger = new Logger();
        $this->requestStartTime = microtime(true);

        // Log incoming request
        $this->logIncomingRequest();

        // Register shutdown function to log response
        register_shutdown_function([$this, 'logOutgoingResponse']);
    }

    /**
     * Log incoming request
     */
    private function logIncomingRequest(): void
    {
        // Get request data
        $requestData = $this->getData();

        // Remove sensitive data from logs
        $sanitized = $this->sanitizeLogData($requestData);

        $this->logger->logRequest([
            'request_data' => $sanitized,
            'user_id' => isset($this->user) ? $this->user->user_id : null
        ]);
    }

    /**
     * Log outgoing response
     */
    public function logOutgoingResponse(): void
    {
        if (!isset($this->logger)) {
            return;
        }

        $executionTime = microtime(true) - $this->requestStartTime;
        $httpCode = http_response_code();

        $this->logger->logResponse($httpCode, null, $executionTime);

        // Log slow requests (> 1 second)
        if ($executionTime > 1.0) {
            $this->logger->log(
                Logger::WARN,
                'Slow request detected',
                [
                    'uri' => $_SERVER['REQUEST_URI'] ?? '',
                    'execution_time' => round($executionTime, 3)
                ]
            );
        }
    }

    /**
     * Sanitize log data (remove passwords, tokens, etc.)
     */
    private function sanitizeLogData($data): array
    {
        if (!is_array($data)) {
            return [];
        }

        $sensitive = ['password', 'token', 'secret', 'api_key', 'credit_card'];

        foreach ($data as $key => $value) {
            foreach ($sensitive as $keyword) {
                if (stripos($key, $keyword) !== false) {
                    $data[$key] = '***REDACTED***';
                }
            }

            // Recursively sanitize nested arrays
            if (is_array($value)) {
                $data[$key] = $this->sanitizeLogData($value);
            }
        }

        return $data;
    }

    /**
     * Get logger instance
     */
    protected function getLogger(): Logger
    {
        if (!isset($this->logger)) {
            $this->logger = new Logger();
        }

        return $this->logger;
    }
}
