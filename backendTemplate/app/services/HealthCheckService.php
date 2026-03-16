<?php

/**
 * Health Check Service
 * 
 * Performs system health checks for monitoring and load balancers
 */
class HealthCheckService
{
    private $db;
    private $cache;

    public function __construct()
    {
        $this->db = new Database();
        $this->cache = new Cache();
    }

    /**
     * Perform comprehensive health check
     * 
     * @return array Health check results
     */
    public function performHealthCheck(): array
    {
        $checks = [
            'database' => $this->checkDatabase(),
            'cache' => $this->checkCache(),
            'disk_space' => $this->checkDiskSpace(),
            'uploads_directory' => $this->checkUploadsDirectory(),
        ];

        // Overall health status
        $healthy = true;
        foreach ($checks as $check) {
            if (!$check['healthy']) {
                $healthy = false;
                break;
            }
        }

        return [
            'healthy' => $healthy,
            'timestamp' => date('Y-m-d H:i:s'),
            'checks' => $checks
        ];
    }

    /**
     * Get system status information
     * 
     * @return array System status details
     */
    public function getSystemStatus(): array
    {
        // Detect environment
        $environment = 'production'; // default
        if (defined('APP_ENV')) {
            $environment = APP_ENV;
        } elseif (getenv('APP_ENV')) {
            $environment = getenv('APP_ENV');
        }

        return [
            'api' => [
                'name' => 'Divine API',
                'version' => '1.0.0',
                'environment' => $environment
            ],
            'server' => [
                'php_version' => phpversion(),
                'server_software' => $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown',
                'server_name' => $_SERVER['SERVER_NAME'] ?? 'Unknown',
                'server_time' => date('Y-m-d H:i:s'),
                'timezone' => date_default_timezone_get()
            ],
            'resources' => $this->getResourceUsage(),
            'uptime' => $this->getUptime(),
            'health' => $this->performHealthCheck()
        ];
    }

    /**
     * Check database connectivity
     */
    private function checkDatabase(): array
    {
        $startTime = microtime(true);

        try {
            $this->db->query("SELECT 1");
            $this->db->execute();

            $responseTime = round((microtime(true) - $startTime) * 1000, 2);

            return [
                'healthy' => true,
                'message' => 'Database connection successful',
                'response_time_ms' => $responseTime
            ];
        } catch (Exception $e) {
            return [
                'healthy' => false,
                'message' => 'Database connection failed',
                'error' => $e->getMessage()
            ];
        }
    }

    /**
     * Check cache availability
     */
    private function checkCache(): array
    {
        try {
            $testKey = 'health_check_' . time();
            $testValue = 'test';

            // Write test
            $this->cache->set($testKey, $testValue, 10);

            // Read test
            $retrieved = $this->cache->get($testKey);

            // Cleanup
            $this->cache->del($testKey);

            if ($retrieved === $testValue) {
                return [
                    'healthy' => true,
                    'message' => 'Cache is working',
                    'type' => 'file-based'
                ];
            } else {
                return [
                    'healthy' => false,
                    'message' => 'Cache read/write failed'
                ];
            }
        } catch (Exception $e) {
            return [
                'healthy' => false,
                'message' => 'Cache unavailable',
                'error' => $e->getMessage()
            ];
        }
    }

    /**
     * Check disk space
     */
    private function checkDiskSpace(): array
    {
        try {
            $freeSpace = disk_free_space(APPROOT);
            $totalSpace = disk_total_space(APPROOT);

            $freeSpaceMB = round($freeSpace / 1024 / 1024, 2);
            $totalSpaceMB = round($totalSpace / 1024 / 1024, 2);
            $usedPercent = round((($totalSpace - $freeSpace) / $totalSpace) * 100, 2);

            // Alert if less than 1GB free or over 90% used
            $healthy = $freeSpace > (1024 * 1024 * 1024) && $usedPercent < 90;

            return [
                'healthy' => $healthy,
                'message' => $healthy ? 'Sufficient disk space' : 'Low disk space warning',
                'free_mb' => $freeSpaceMB,
                'total_mb' => $totalSpaceMB,
                'used_percent' => $usedPercent
            ];
        } catch (Exception $e) {
            return [
                'healthy' => false,
                'message' => 'Unable to check disk space',
                'error' => $e->getMessage()
            ];
        }
    }

    /**
     * Check uploads directory
     */
    private function checkUploadsDirectory(): array
    {
        $uploadDir = APPROOT . '/../public/assets/uploads';

        if (!file_exists($uploadDir)) {
            return [
                'healthy' => false,
                'message' => 'Uploads directory does not exist',
                'path' => $uploadDir
            ];
        }

        if (!is_writable($uploadDir)) {
            return [
                'healthy' => false,
                'message' => 'Uploads directory is not writable',
                'path' => $uploadDir
            ];
        }

        return [
            'healthy' => true,
            'message' => 'Uploads directory accessible',
            'path' => $uploadDir,
            'writable' => true
        ];
    }

    /**
     * Get resource usage
     */
    private function getResourceUsage(): array
    {
        return [
            'memory' => [
                'current_mb' => round(memory_get_usage() / 1024 / 1024, 2),
                'peak_mb' => round(memory_get_peak_usage() / 1024 / 1024, 2),
                'limit' => ini_get('memory_limit')
            ],
            'cpu' => [
                'load_average' => $this->getLoadAverage()
            ]
        ];
    }

    /**
     * Get load average (Unix systems)
     */
    private function getLoadAverage(): ?array
    {
        if (function_exists('sys_getloadavg')) {
            $load = sys_getloadavg();
            return [
                '1min' => round($load[0], 2),
                '5min' => round($load[1], 2),
                '15min' => round($load[2], 2)
            ];
        }

        return null;
    }

    /**
     * Get server uptime (Unix systems)
     */
    private function getUptime(): ?string
    {
        if (file_exists('/proc/uptime')) {
            $uptime = file_get_contents('/proc/uptime');
            $uptimeSeconds = (int)explode(' ', $uptime)[0];

            $days = floor($uptimeSeconds / 86400);
            $hours = floor(($uptimeSeconds % 86400) / 3600);
            $minutes = floor(($uptimeSeconds % 3600) / 60);

            return "{$days}d {$hours}h {$minutes}m";
        }

        return null;
    }

    /**
     * Quick health check (for load balancers)
     * Just checks critical services
     */
    public function quickCheck(): bool
    {
        try {
            // Quick database check
            $this->db->query("SELECT 1");
            $this->db->execute();

            return true;
        } catch (Exception $e) {
            return false;
        }
    }
}
