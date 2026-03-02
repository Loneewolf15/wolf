<?php

/**
 * Upload Model
 * 
 * Handles database operations for file uploads
 */
class Upload
{
    private $db;

    public function __construct()
    {
        $this->db = new Database();
    }

    /**
     * Create new upload record
     * 
     * @param array $data Upload data (file_path, file_url, etc.)
     * @return int|false Upload ID or false on failure
     */
    public function create(array $data)
    {
        $this->db->query("INSERT INTO uploads (
            user_id, filename, original_name, file_path, file_url,
            file_type, mime_type, file_size, storage_type,
            cloud_provider, cloud_id, variants, metadata
        ) VALUES (
            :user_id, :filename, :original_name, :file_path, :file_url,
            :file_type, :mime_type, :file_size, :storage_type,
            :cloud_provider, :cloud_id, :variants, :metadata
        )");

        // Bind values
        $this->db->bind(':user_id', $data['user_id']);
        $this->db->bind(':filename', $data['filename']);
        $this->db->bind(':original_name', $data['original_name']);
        $this->db->bind(':file_path', $data['file_path']);
        $this->db->bind(':file_url', $data['file_url']);  // Full URL with URLROOT
        $this->db->bind(':file_type', $data['file_type']);
        $this->db->bind(':mime_type', $data['mime_type']);
        $this->db->bind(':file_size', $data['file_size']);
        $this->db->bind(':storage_type', $data['storage_type'] ?? 'local');
        $this->db->bind(':cloud_provider', $data['cloud_provider'] ?? null);
        $this->db->bind(':cloud_id', $data['cloud_id'] ?? null);
        $this->db->bind(':variants', isset($data['variants']) ? json_encode($data['variants']) : null);
        $this->db->bind(':metadata', isset($data['metadata']) ? json_encode($data['metadata']) : null);

        if ($this->db->execute()) {
            return $this->db->lastInsertId();
        }

        return false;
    }

    /**
     * Find upload by ID
     */
    public function findById(int $id): ?object
    {
        $this->db->query("SELECT * FROM uploads WHERE id = :id");
        $this->db->bind(':id', $id);

        $result = $this->db->single();

        if ($result) {
            // Decode JSON fields
            $result->variants = json_decode($result->variants);
            $result->metadata = json_decode($result->metadata);
        }

        return $result ?: null;
    }

    /**
     * Get user's uploads with pagination
     */
    public function findByUser(int $userId, int $limit = 20, int $offset = 0, string $fileType = null): array
    {
        $sql = "SELECT * FROM uploads WHERE user_id = :user_id";

        if ($fileType) {
            $sql .= " AND file_type = :file_type";
        }

        $sql .= " ORDER BY created_at DESC LIMIT :limit OFFSET :offset";

        $this->db->query($sql);
        $this->db->bind(':user_id', $userId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);
        $this->db->bind(':offset', $offset, PDO::PARAM_INT);

        if ($fileType) {
            $this->db->bind(':file_type', $fileType);
        }

        $results = $this->db->resultSet();

        // Decode JSON fields
        foreach ($results as $result) {
            $result->variants = json_decode($result->variants);
            $result->metadata = json_decode($result->metadata);
        }

        return $results;
    }

    /**
     * Delete upload record
     */
    public function delete(int $id): bool
    {
        $this->db->query("DELETE FROM uploads WHERE id = :id");
        $this->db->bind(':id', $id);

        return $this->db->execute();
    }

    /**
     * Get user's storage quota usage
     */
    public function getUserQuota(int $userId): array
    {
        $this->db->query("SELECT 
            COUNT(*) as file_count,
            COALESCE(SUM(file_size), 0) as total_bytes,
            COUNT(CASE WHEN file_type = 'image' THEN 1 END) as image_count,
            COUNT(CASE WHEN file_type = 'video' THEN 1 END) as video_count,
            COUNT(CASE WHEN file_type = 'document' THEN 1 END) as document_count
            FROM uploads 
            WHERE user_id = :user_id");
        $this->db->bind(':user_id', $userId);

        $result = $this->db->single();

        return [
            'file_count' => (int)$result->file_count,
            'total_bytes' => (int)$result->total_bytes,
            'total_mb' => round($result->total_bytes / 1024 / 1024, 2),
            'image_count' => (int)$result->image_count,
            'video_count' => (int)$result->video_count,
            'document_count' => (int)$result->document_count
        ];
    }

    /**
     * Check if upload belongs to user
     */
    public function belongsToUser(int $uploadId, int $userId): bool
    {
        $this->db->query("SELECT id FROM uploads WHERE id = :id AND user_id = :user_id");
        $this->db->bind(':id', $uploadId);
        $this->db->bind(':user_id', $userId);

        return $this->db->single() !== false;
    }

    /**
     * Get total uploads count for user
     */
    public function countByUser(int $userId, string $fileType = null): int
    {
        $sql = "SELECT COUNT(*) as total FROM uploads WHERE user_id = :user_id";

        if ($fileType) {
            $sql .= " AND file_type = :file_type";
        }

        $this->db->query($sql);
        $this->db->bind(':user_id', $userId);

        if ($fileType) {
            $this->db->bind(':file_type', $fileType);
        }

        $result = $this->db->single();
        return (int)$result->total;
    }
}
