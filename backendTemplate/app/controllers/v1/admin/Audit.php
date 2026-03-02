<?php

/**
 * Audit Controller (Admin)
 * 
 * Endpoints for viewing audit trail
 */
class Audit extends Controller
{
    private $auditModel;

    public function __construct()
    {
        $this->auditModel = $this->model('AuditLog');
    }

    /**
     * GET /v1/admin/audit - View audit trail
     *
     * Query params:
     * - user_id: Filter by user
     * - action: Filter by action (create, update, delete, etc.)
     * - entity_type: Filter by entity type
     * - entity_id: Filter by entity ID
     * - date_from: Start date (YYYY-MM-DD)
     * - date_to: End date (YYYY-MM-DD)
     * - page: Page number
     * - limit: Items per page
     */
    public function index()
    {
        // Require authentication
        $user = $this->RouteProtection();

        // TODO: Check if user is admin
        // if (!$this->isAdmin($user)) {
        //     return $this->sendResponse(false, 'Admin access required', [], 403);
        // }

        // Build filters
        $filters = [];

        if ($this->getData('user_id')) {
            $filters['user_id'] = (int)$this->getData('user_id');
        }

        if ($this->getData('action')) {
            $filters['action'] = $this->getData('action');
        }

        if ($this->getData('entity_type')) {
            $filters['entity_type'] = $this->getData('entity_type');
        }

        if ($this->getData('entity_id')) {
            $filters['entity_id'] = (int)$this->getData('entity_id');
        }

        if ($this->getData('date_from')) {
            $filters['date_from'] = $this->getData('date_from') . ' 00:00:00';
        }

        if ($this->getData('date_to')) {
            $filters['date_to'] = $this->getData('date_to') . ' 23:59:59';
        }

        // Pagination
        $page = (int)($this->getData('page') ?? 1);
        $limit = min((int)($this->getData('limit') ?? 50), 100);
        $offset = ($page - 1) * $limit;

        // Get logs
        $logs = $this->auditModel->getLogs($filters, $limit, $offset);
        $total = $this->auditModel->count($filters);

        return $this->sendResponse(true, 'Audit trail retrieved', [
            'logs' => $logs,
            'pagination' => [
                'page' => $page,
                'limit' => $limit,
                'total' => $total,
                'pages' => ceil($total / $limit)
            ],
            'filters' => $filters
        ]);
    }

    /**
     * GET /v1/admin/audit/{id} - Get specific audit log
     */
    public function get($id)
    {
        $user = $this->RouteProtection();

        $log = $this->auditModel->findById((int)$id);

        if (!$log) {
            return $this->sendResponse(false, 'Audit log not found', [], 404);
        }

        return $this->sendResponse(true, 'Audit log retrieved', $log);
    }

    /**
     * GET /v1/admin/audit/entity/{entityType}/{entityId} - Get entity history
     */
    public function entity($entityType, $entityId)
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 50), 100);

        $history = $this->auditModel->getEntityHistory($entityType, (int)$entityId, $limit);

        return $this->sendResponse(true, 'Entity history retrieved', [
            'entity_type' => $entityType,
            'entity_id' => $entityId,
            'total' => count($history),
            'history' => $history
        ]);
    }

    /**
     * GET /v1/admin/audit/user/{userId} - Get user activity
     */
    public function user($userId)
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 100), 500);

        $activity = $this->auditModel->getUserActivity((int)$userId, $limit);

        return $this->sendResponse(true, 'User activity retrieved', [
            'user_id' => $userId,
            'total' => count($activity),
            'activity' => $activity
        ]);
    }

    /**
     * GET /v1/admin/audit/stats - Get audit statistics
     */
    public function stats()
    {
        $user = $this->RouteProtection();

        $dateFrom = $this->getData('date_from');
        $dateTo = $this->getData('date_to');

        $stats = $this->auditModel->getStatistics($dateFrom, $dateTo);

        // Group by action
        $byAction = [];
        $byEntityType = [];

        foreach ($stats as $stat) {
            if (!isset($byAction[$stat->action])) {
                $byAction[$stat->action] = 0;
            }
            $byAction[$stat->action] += $stat->count;

            if (!isset($byEntityType[$stat->entity_type])) {
                $byEntityType[$stat->entity_type] = 0;
            }
            $byEntityType[$stat->entity_type] += $stat->count;
        }

        return $this->sendResponse(true, 'Audit statistics retrieved', [
            'period' => [
                'from' => $dateFrom,
                'to' => $dateTo
            ],
            'by_action' => $byAction,
            'by_entity_type' => $byEntityType,
            'detailed' => $stats
        ]);
    }
}
