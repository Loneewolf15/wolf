<?php

/**
 * Audit Log Model
 * 
 * Tracks all user actions and entity changes
 */
class AuditLog
{
    private $db;

    public function __construct()
    {
        $this->db = new Database();
    }

    /**
     * Create audit log entry
     * 
     * @param array $data Audit log data
     * @return int|false Audit log ID or false
     */
    public function log(array $data)
    {
        $this->db->query("INSERT INTO audit_logs (
            user_id, user_email, user_ip,
            action, entity_type, entity_id,
            before_values, after_values, changes,
            request_id, endpoint, http_method,
            metadata, description
        ) VALUES (
            :user_id, :user_email, :user_ip,
            :action, :entity_type, :entity_id,
            :before_values, :after_values, :changes,
            :request_id, :endpoint, :http_method,
            :metadata, :description
        )");

        // Bind values
        $this->db->bind(':user_id', $data['user_id'] ?? null);
        $this->db->bind(':user_email', $data['user_email'] ?? null);
        $this->db->bind(':user_ip', $data['user_ip'] ?? null);
        $this->db->bind(':action', $data['action']);
        $this->db->bind(':entity_type', $data['entity_type']);
        $this->db->bind(':entity_id', $data['entity_id'] ?? null);
        $this->db->bind(':before_values', isset($data['before_values']) ? json_encode($data['before_values']) : null);
        $this->db->bind(':after_values', isset($data['after_values']) ? json_encode($data['after_values']) : null);
        $this->db->bind(':changes', isset($data['changes']) ? json_encode($data['changes']) : null);
        $this->db->bind(':request_id', $data['request_id'] ?? null);
        $this->db->bind(':endpoint', $data['endpoint'] ?? null);
        $this->db->bind(':http_method', $data['http_method'] ?? null);
        $this->db->bind(':metadata', isset($data['metadata']) ? json_encode($data['metadata']) : null);
        $this->db->bind(':description', $data['description'] ?? null);

        if ($this->db->execute()) {
            return $this->db->lastInsertId();
        }

        return false;
    }

    /**
     * Get audit logs with filters
     */
    public function getLogs(array $filters = [], int $limit = 100, int $offset = 0): array
    {
        $sql = "SELECT * FROM audit_logs WHERE 1=1";
        $bindings = [];

        // Apply filters
        if (isset($filters['user_id'])) {
            $sql .= " AND user_id = :user_id";
            $bindings[':user_id'] = $filters['user_id'];
        }

        if (isset($filters['action'])) {
            $sql .= " AND action = :action";
            $bindings[':action'] = $filters['action'];
        }

        if (isset($filters['entity_type'])) {
            $sql .= " AND entity_type = :entity_type";
            $bindings[':entity_type'] = $filters['entity_type'];
        }

        if (isset($filters['entity_id'])) {
            $sql .= " AND entity_id = :entity_id";
            $bindings[':entity_id'] = $filters['entity_id'];
        }

        if (isset($filters['date_from'])) {
            $sql .= " AND created_at >= :date_from";
            $bindings[':date_from'] = $filters['date_from'];
        }

        if (isset($filters['date_to'])) {
            $sql .= " AND created_at <= :date_to";
            $bindings[':date_to'] = $filters['date_to'];
        }

        $sql .= " ORDER BY created_at DESC LIMIT :limit OFFSET :offset";

        $this->db->query($sql);

        foreach ($bindings as $key => $value) {
            $this->db->bind($key, $value);
        }

        $this->db->bind(':limit', $limit, PDO::PARAM_INT);
        $this->db->bind(':offset', $offset, PDO::PARAM_INT);

        $results = $this->db->resultSet();

        // Decode JSON fields
        foreach ($results as $result) {
            $result->before_values = json_decode($result->before_values);
            $result->after_values = json_decode($result->after_values);
            $result->changes = json_decode($result->changes);
            $result->metadata = json_decode($result->metadata);
        }

        return $results;
    }

    /**
     * Get audit log by ID
     */
    public function findById(int $id): ?object
    {
        $this->db->query("SELECT * FROM audit_logs WHERE id = :id");
        $this->db->bind(':id', $id);

        $result = $this->db->single();

        if ($result) {
            $result->before_values = json_decode($result->before_values);
            $result->after_values = json_decode($result->after_values);
            $result->changes = json_decode($result->changes);
            $result->metadata = json_decode($result->metadata);
        }

        return $result ?: null;
    }

    /**
     * Get entity history
     */
    public function getEntityHistory(string $entityType, int $entityId, int $limit = 50): array
    {
        $this->db->query("SELECT * FROM audit_logs 
            WHERE entity_type = :entity_type 
            AND entity_id = :entity_id 
            ORDER BY created_at DESC 
            LIMIT :limit");

        $this->db->bind(':entity_type', $entityType);
        $this->db->bind(':entity_id', $entityId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        $results = $this->db->resultSet();

        foreach ($results as $result) {
            $result->before_values = json_decode($result->before_values);
            $result->after_values = json_decode($result->after_values);
            $result->changes = json_decode($result->changes);
            $result->metadata = json_decode($result->metadata);
        }

        return $results;
    }

    /**
     * Get user activity
     */
    public function getUserActivity(int $userId, int $limit = 100): array
    {
        $this->db->query("SELECT * FROM audit_logs 
            WHERE user_id = :user_id 
            ORDER BY created_at DESC 
            LIMIT :limit");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        return $this->db->resultSet();
    }

    /**
     * Count logs
     */
    public function count(array $filters = []): int
    {
        $sql = "SELECT COUNT(*) as total FROM audit_logs WHERE 1=1";
        $bindings = [];

        if (isset($filters['user_id'])) {
            $sql .= " AND user_id = :user_id";
            $bindings[':user_id'] = $filters['user_id'];
        }

        if (isset($filters['action'])) {
            $sql .= " AND action = :action";
            $bindings[':action'] = $filters['action'];
        }

        if (isset($filters['entity_type'])) {
            $sql .= " AND entity_type = :entity_type";
            $bindings[':entity_type'] = $filters['entity_type'];
        }

        $this->db->query($sql);

        foreach ($bindings as $key => $value) {
            $this->db->bind($key, $value);
        }

        $result = $this->db->single();
        return (int)$result->total;
    }

    /**
     * Get audit statistics
     */
    public function getStatistics(string $dateFrom = null, string $dateTo = null): array
    {
        $sql = "SELECT 
            action,
            entity_type,
            COUNT(*) as count
            FROM audit_logs
            WHERE 1=1";

        $bindings = [];

        if ($dateFrom) {
            $sql .= " AND created_at >= :date_from";
            $bindings[':date_from'] = $dateFrom;
        }

        if ($dateTo) {
            $sql .= " AND created_at <= :date_to";
            $bindings[':date_to'] = $dateTo;
        }

        $sql .= " GROUP BY action, entity_type";

        $this->db->query($sql);

        foreach ($bindings as $key => $value) {
            $this->db->bind($key, $value);
        }

        return $this->db->resultSet();
    }
}
