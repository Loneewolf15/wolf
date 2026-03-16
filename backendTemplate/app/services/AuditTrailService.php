<?php

/**
 * Audit Trail Service
 * 
 * Simplifies audit logging with common actions
 */
class AuditTrailService
{
    private $auditModel;
    private $logger;

    public function __construct()
    {
        $this->auditModel = new AuditLog();
        $this->logger = new Logger();
    }

    /**
     * Log a create action
     */
    public function logCreate($user, string $entityType, int $entityId, array $data, string $description = null): void
    {
        $this->log([
            'user' => $user,
            'action' => 'create',
            'entity_type' => $entityType,
            'entity_id' => $entityId,
            'after_values' => $data,
            'description' => $description ?? ucfirst($entityType) . ' created'
        ]);
    }

    /**
     * Log an update action
     */
    public function logUpdate($user, string $entityType, int $entityId, array $beforeData, array $afterData, string $description = null): void
    {
        // Calculate what changed
        $changes = $this->calculateChanges($beforeData, $afterData);

        $this->log([
            'user' => $user,
            'action' => 'update',
            'entity_type' => $entityType,
            'entity_id' => $entityId,
            'before_values' => $beforeData,
            'after_values' => $afterData,
            'changes' => $changes,
            'description' => $description ?? ucfirst($entityType) . ' updated'
        ]);
    }

    /**
     * Log a delete action
     */
    public function logDelete($user, string $entityType, int $entityId, array $data, string $description = null): void
    {
        $this->log([
            'user' => $user,
            'action' => 'delete',
            'entity_type' => $entityType,
            'entity_id' => $entityId,
            'before_values' => $data,
            'description' => $description ?? ucfirst($entityType) . ' deleted'
        ]);
    }

    /**
     * Log a login action
     */
    public function logLogin($user, bool $success = true, array $metadata = []): void
    {
        $this->log([
            'user' => $user,
            'action' => $success ? 'login_success' : 'login_failed',
            'entity_type' => 'authentication',
            'metadata' => $metadata,
            'description' => $success ? 'User logged in' : 'Login attempt failed'
        ]);
    }

    /**
     * Log a logout action
     */
    public function logLogout($user): void
    {
        $this->log([
            'user' => $user,
            'action' => 'logout',
            'entity_type' => 'authentication',
            'description' => 'User logged out'
        ]);
    }

    /**
     * Log a password change
     */
    public function logPasswordChange($user): void
    {
        $this->log([
            'user' => $user,
            'action' => 'password_changed',
            'entity_type' => 'user',
            'entity_id' => is_object($user) ? $user->user_id : $user,
            'description' => 'Password changed'
        ]);
    }

    /**
     * Log a custom action
     */
    public function logAction($user, string $action, string $entityType, int $entityId = null, array $metadata = [], string $description = null): void
    {
        $this->log([
            'user' => $user,
            'action' => $action,
            'entity_type' => $entityType,
            'entity_id' => $entityId,
            'metadata' => $metadata,
            'description' => $description
        ]);
    }

    /**
     * Core logging method
     */
    private function log(array $data): void
    {
        // Extract user information
        $user = $data['user'] ?? null;
        $userId = null;
        $userEmail = null;

        if (is_object($user)) {
            $userId = $user->user_id ?? null;
            $userEmail = $user->email ?? null;
        } elseif (is_numeric($user)) {
            $userId = $user;
        }

        // Get request context
        $requestId = $this->logger->getRequestId();
        $endpoint = $_SERVER['REQUEST_URI'] ?? null;
        $httpMethod = $_SERVER['REQUEST_METHOD'] ?? null;
        $userIp = $this->getClientIp();

        // Prepare audit log data
        $logData = [
            'user_id' => $userId,
            'user_email' => $userEmail,
            'user_ip' => $userIp,
            'action' => $data['action'],
            'entity_type' => $data['entity_type'],
            'entity_id' => $data['entity_id'] ?? null,
            'before_values' => $data['before_values'] ?? null,
            'after_values' => $data['after_values'] ?? null,
            'changes' => $data['changes'] ?? null,
            'request_id' => $requestId,
            'endpoint' => $endpoint,
            'http_method' => $httpMethod,
            'metadata' => $data['metadata'] ?? null,
            'description' => $data['description'] ?? null
        ];

        // Save to database
        $this->auditModel->log($logData);
    }

    /**
     * Calculate what changed between before and after
     */
    private function calculateChanges(array $before, array $after): array
    {
        $changes = [];

        // Find changed fields
        foreach ($after as $key => $value) {
            if (!isset($before[$key]) || $before[$key] !== $value) {
                $changes[$key] = [
                    'old' => $before[$key] ?? null,
                    'new' => $value
                ];
            }
        }

        // Find removed fields
        foreach ($before as $key => $value) {
            if (!isset($after[$key])) {
                $changes[$key] = [
                    'old' => $value,
                    'new' => null
                ];
            }
        }

        return $changes;
    }

    /**
     * Get client IP
     */
    private function getClientIp(): string
    {
        $headers = [
            'HTTP_CF_CONNECTING_IP',
            'HTTP_X_FORWARDED_FOR',
            'HTTP_X_REAL_IP',
            'REMOTE_ADDR'
        ];

        foreach ($headers as $header) {
            if (isset($_SERVER[$header]) && !empty($_SERVER[$header])) {
                $ip = $_SERVER[$header];
                if (strpos($ip, ',') !== false) {
                    $ip = trim(explode(',', $ip)[0]);
                }
                return $ip;
            }
        }

        return 'unknown';
    }
}
