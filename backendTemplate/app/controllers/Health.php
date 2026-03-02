<?php

/**
 * Health Check Controller
 * 
 * Provides health check and status endpoints for monitoring
 */
class Health extends Controller
{
    private $healthService;

    public function __construct()
    {
        $this->healthService = new HealthCheckService();
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200)
    {
        http_response_code($httpCode);
        echo json_encode([
            'status' => $status,
            'message' => $message,
            'data' => $data,
            'timestamp' => time()
        ]);
        exit;
    }

    /**
     * GET /health - Quick health check
     * 
     * For load balancers and simple monitoring
     * Returns 200 if healthy, 503 if unhealthy
     */
    public function index()
    {
        $isHealthy = $this->healthService->quickCheck();

        if ($isHealthy) {
            http_response_code(200);
            return $this->sendResponse(true, 'Service is healthy', [
                'status' => 'UP',
                'timestamp' => date('Y-m-d H:i:s')
            ]);
        } else {
            http_response_code(503);
            return $this->sendResponse(false, 'Service is unhealthy', [
                'status' => 'DOWN',
                'timestamp' => date('Y-m-d H:i:s')
            ], 503);
        }
    }

    /**
     * GET /health/detailed - Detailed health check
     * 
     * Comprehensive health information
     * Requires authentication (optional - can be public)
     */
    public function detailed()
    {
        // Optional: Uncomment to require authentication
        // $this->RouteProtection();

        $health = $this->healthService->performHealthCheck();

        $httpCode = $health['healthy'] ? 200 : 503;
        http_response_code($httpCode);

        return $this->sendResponse(
            $health['healthy'],
            $health['healthy'] ? 'All systems operational' : 'Some systems are unhealthy',
            $health,
            $httpCode
        );
    }

    /**
     * GET /health/status - System status
     * 
     * Detailed system information including version, resources, etc.
     * Useful for debugging and monitoring dashboards
     */
    public function status()
    {
        // Optional: Uncomment to require admin authentication
        // $user = $this->RouteProtection();
        // if (!$this->isAdmin($user)) {
        //     return $this->sendResponse(false, 'Admin access required', [], 403);
        // }

        $status = $this->healthService->getSystemStatus();

        return $this->sendResponse(true, 'System status retrieved', $status);
    }

    /**
     * GET /health/ping - Simple ping endpoint
     * 
     * Extremely lightweight - just confirms server is responding
     */
    public function ping()
    {
        return $this->sendResponse(true, 'pong', [
            'timestamp' => microtime(true)
        ]);
    }

    /**
     * GET /health/ready - Readiness check
     * 
     * For Kubernetes/container orchestration
     * Checks if service is ready to accept traffic
     */
    public function ready()
    {
        $health = $this->healthService->performHealthCheck();

        if ($health['healthy']) {
            http_response_code(200);
            return $this->sendResponse(true, 'Service is ready', [
                'ready' => true
            ]);
        } else {
            http_response_code(503);
            return $this->sendResponse(false, 'Service not ready', [
                'ready' => false,
                'checks' => $health['checks']
            ], 503);
        }
    }

    /**
     * GET /health/live - Liveness check
     * 
     * For Kubernetes/container orchestration
     * Simple check that process is alive
     */
    public function live()
    {
        // If we got here, the process is alive
        return $this->sendResponse(true, 'Service is alive', [
            'alive' => true,
            'timestamp' => date('Y-m-d H:i:s')
        ]);
    }
}
