<?php

/**
 * Logger Service
 * 
 * Structured logging service for API requests, responses, and errors
 */
class Logger
{
    private $logDir;
    private $requestId;

    // Log levels
    const DEBUG = 'DEBUG';
    const INFO = 'INFO';
    const WARN = 'WARN';
    const ERROR = 'ERROR';
    const CRITICAL = 'CRITICAL';

    public function __construct()
    {
        $this->logDir = APPROOT . '/logs';
        $this->ensureLogDirectory();
        $this->requestId = $this->generateRequestId();
    }

    /**
     * Log a request
     */
    public function logRequest(array $data = []): void
    {
        $logData = [
            'type' => 'request',
            'method' => $_SERVER['REQUEST_METHOD'] ?? 'UNKNOWN',
            'uri' => $_SERVER['REQUEST_URI'] ?? '',
            'ip' => $this->getClientIp(),
            'user_agent' => $_SERVER['HTTP_USER_AGENT'] ?? '',
            'user_id' => $data['user_id'] ?? null,
            'request_data' => $data['request_data'] ?? null
        ];

        $this->log(self::INFO, 'API Request', $logData);
    }

    /**
     * Log a response
     */
    public function logResponse(int $httpCode, $responseData = null, float $executionTime = null): void
    {
        $logData = [
            'type' => 'response',
            'http_code' => $httpCode,
            'execution_time_ms' => $executionTime ? round($executionTime * 1000, 2) : null,
            'response_size' => $responseData ? strlen(json_encode($responseData)) : 0
        ];

        $level = $httpCode >= 500 ? self::ERROR : ($httpCode >= 400 ? self::WARN : self::INFO);
        $this->log($level, 'API Response', $logData);
    }

    /**
     * Log an error
     */
    public function logError(string $message, Exception $exception = null, array $context = []): void
    {
        $logData = [
            'type' => 'error',
            'message' => $message,
            'context' => $context
        ];

        if ($exception) {
            $logData['exception'] = [
                'class' => get_class($exception),
                'message' => $exception->getMessage(),
                'code' => $exception->getCode(),
                'file' => $exception->getFile(),
                'line' => $exception->getLine(),
                'trace' => $exception->getTraceAsString()
            ];
        }

        $this->log(self::ERROR, 'Error occurred', $logData);
    }

    /**
     * Log a slow query
     */
    public function logSlowQuery(string $query, float $executionTime, array $params = []): void
    {
        $logData = [
            'type' => 'slow_query',
            'query' => $query,
            'execution_time_ms' => round($executionTime * 1000, 2),
            'params' => $params
        ];

        $this->log(self::WARN, 'Slow query detected', $logData);
    }

    /**
     * Core logging method
     */
    public function log(string $level, string $message, array $data = []): void
    {
        $logEntry = [
            'timestamp' => date('Y-m-d H:i:s'),
            'request_id' => $this->requestId,
            'level' => $level,
            'message' => $message,
            'data' => $data,
            'server' => [
                'hostname' => gethostname(),
                'ip' => $_SERVER['SERVER_ADDR'] ?? 'unknown'
            ]
        ];

        // Write to daily log file
        $this->writeLog($logEntry);

        // Also write to error log if ERROR or CRITICAL
        if (in_array($level, [self::ERROR, self::CRITICAL])) {
            error_log(json_encode($logEntry));
        }
    }

    /**
     * Write log to file
     */
    private function writeLog(array $logEntry): void
    {
        $filename = $this->getLogFilename($logEntry['level']);
        $logLine = json_encode($logEntry) . PHP_EOL;

        file_put_contents($filename, $logLine, FILE_APPEND | LOCK_EX);
    }

    /**
     * Get log filename based on level and date
     */
    private function getLogFilename(string $level): string
    {
        $date = date('Y-m-d');

        // Separate files per level
        switch ($level) {
            case self::ERROR:
            case self::CRITICAL:
                return $this->logDir . "/error-{$date}.log";
            case self::WARN:
                return $this->logDir . "/warn-{$date}.log";
            default:
                return $this->logDir . "/app-{$date}.log";
        }
    }

    /**
     * Generate unique request ID
     */
    private function generateRequestId(): string
    {
        return uniqid('req_', true) . '_' . bin2hex(random_bytes(4));
    }

    /**
     * Get current request ID
     */
    public function getRequestId(): string
    {
        return $this->requestId;
    }

    /**
     * Get client IP address
     */
    private function getClientIp(): string
    {
        $headers = [
            'HTTP_CF_CONNECTING_IP',  // Cloudflare
            'HTTP_X_FORWARDED_FOR',   // Proxy/Load balancer
            'HTTP_X_REAL_IP',         // Nginx
            'REMOTE_ADDR'             // Direct connection
        ];

        foreach ($headers as $header) {
            if (isset($_SERVER[$header]) && !empty($_SERVER[$header])) {
                $ip = $_SERVER[$header];
                // Handle comma-separated IPs (proxies)
                if (strpos($ip, ',') !== false) {
                    $ip = trim(explode(',', $ip)[0]);
                }
                return $ip;
            }
        }

        return 'unknown';
    }

    /**
     * Ensure log directory exists
     */
    private function ensureLogDirectory(): void
    {
        if (!is_dir($this->logDir)) {
            mkdir($this->logDir, 0755, true);
        }
    }

    /**
     * Read logs from file
     */
    public function readLogs(string $date = null, string $level = null, int $limit = 100): array
    {
        $date = $date ?? date('Y-m-d');

        // Determine which file to read
        if ($level === 'error') {
            $filename = $this->logDir . "/error-{$date}.log";
        } elseif ($level === 'warn') {
            $filename = $this->logDir . "/warn-{$date}.log";
        } else {
            $filename = $this->logDir . "/app-{$date}.log";
        }

        if (!file_exists($filename)) {
            return [];
        }

        // Read file
        $lines = file($filename, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);

        // Parse JSON lines
        $logs = [];
        foreach (array_reverse($lines) as $line) {
            if (count($logs) >= $limit) {
                break;
            }

            $log = json_decode($line, true);
            if ($log) {
                $logs[] = $log;
            }
        }

        return $logs;
    }

    /**
     * Clean old logs (retention policy)
     */
    public function cleanOldLogs(int $daysToKeep = 30): int
    {
        $deleted = 0;
        $cutoffDate = strtotime("-{$daysToKeep} days");

        $files = glob($this->logDir . '/*.log');

        foreach ($files as $file) {
            if (filemtime($file) < $cutoffDate) {
                if (unlink($file)) {
                    $deleted++;
                }
            }
        }

        return $deleted;
    }

    /**
     * Get log statistics
     */
    public function getLogStats(string $date = null): array
    {
        $date = $date ?? date('Y-m-d');
        $files = [
            'app' => $this->logDir . "/app-{$date}.log",
            'warn' => $this->logDir . "/warn-{$date}.log",
            'error' => $this->logDir . "/error-{$date}.log"
        ];

        $stats = [
            'date' => $date,
            'total_logs' => 0,
            'by_level' => [
                'info' => 0,
                'warn' => 0,
                'error' => 0
            ],
            'by_type' => [],
            'file_sizes' => []
        ];

        foreach ($files as $type => $file) {
            if (file_exists($file)) {
                $lines = file($file, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
                $count = count($lines);
                $stats['total_logs'] += $count;

                // Count by level
                if ($type === 'error') {
                    $stats['by_level']['error'] = $count;
                } elseif ($type === 'warn') {
                    $stats['by_level']['warn'] = $count;
                } else {
                    $stats['by_level']['info'] = $count;
                }

                $stats['file_sizes'][$type] = filesize($file);
            }
        }

        return $stats;
    }
}
