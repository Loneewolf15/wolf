<?php

/**
 * Logs Controller (Admin)
 * 
 * Endpoints for viewing and managing application logs
 */
class Logs extends Controller
{
    private $logger;

    public function __construct()
    {
        $this->logger = new Logger();
    }

    /**
     * GET /v1/admin/logs - View logs
     * 
     * Query params:
     * - date: YYYY-MM-DD (default: today)
     * - level: info, warn, error (default: all)
     * - limit: number of logs (default: 100, max: 1000)
     */
    public function index()
    {
        // Require authentication
        $user = $this->RouteProtection();

        // TODO: Check if user is admin
        // if (!$this->isAdmin($user)) {
        //     return $this->sendResponse(false, 'Admin access required', [], 403);
        // }

        // Get parameters
        $date = $this->getData('date') ?? date('Y-m-d');
        $level = $this->getData('level');
        $limit = min((int)($this->getData('limit') ?? 100), 1000);

        // Validate date format
        if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) {
            return $this->sendResponse(false, 'Invalid date format. Use YYYY-MM-DD', [], 400);
        }

        // Read logs
        $logs = $this->logger->readLogs($date, $level, $limit);

        return $this->sendResponse(true, 'Logs retrieved', [
            'date' => $date,
            'level' => $level,
            'total' => count($logs),
            'logs' => $logs
        ]);
    }

    /**
     * GET /v1/admin/logs/stats - Get log statistics
     */
    public function stats()
    {
        $user = $this->RouteProtection();

        $date = $this->getData('date') ?? date('Y-m-d');

        if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) {
            return $this->sendResponse(false, 'Invalid date format', [], 400);
        }

        $stats = $this->logger->getLogStats($date);

        return $this->sendResponse(true, 'Log statistics retrieved', $stats);
    }

    /**
     * GET /v1/admin/logs/errors - Get recent errors
     */
    public function errors()
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 50), 500);
        $date = $this->getData('date') ?? date('Y-m-d');

        $errors = $this->logger->readLogs($date, 'error', $limit);

        return $this->sendResponse(true, 'Error logs retrieved', [
            'date' => $date,
            'total' => count($errors),
            'errors' => $errors
        ]);
    }

    /**
     * POST /v1/admin/logs/cleanup - Clean old logs
     */
    public function cleanup()
    {
        $user = $this->RouteProtection();

        $daysToKeep = (int)($this->getData('days_to_keep') ?? 30);

        if ($daysToKeep < 7) {
            return $this->sendResponse(false, 'Must keep at least 7 days of logs', [], 400);
        }

        $deleted = $this->logger->cleanOldLogs($daysToKeep);

        return $this->sendResponse(true, 'Old logs cleaned', [
            'files_deleted' => $deleted,
            'retention_days' => $daysToKeep
        ]);
    }

    /**
     * GET /v1/admin/logs/search - Search logs
     */
    public function search()
    {
        $user = $this->RouteProtection();

        $query = $this->getData('q');
        $date = $this->getData('date') ?? date('Y-m-d');
        $limit = min((int)($this->getData('limit') ?? 100), 1000);

        if (empty($query)) {
            return $this->sendResponse(false, 'Search query required', [], 400);
        }

        // Read all logs for the date
        $logs = $this->logger->readLogs($date, null, 10000);

        // Filter logs containing search term
        $results = array_filter($logs, function ($log) use ($query) {
            $logString = json_encode($log);
            return stripos($logString, $query) !== false;
        });

        // Limit results
        $results = array_slice(array_values($results), 0, $limit);

        return $this->sendResponse(true, 'Search results', [
            'query' => $query,
            'date' => $date,
            'total' => count($results),
            'results' => $results
        ]);
    }
}
