<?php

/**
 * Data Export Service
 * 
 * Export user data to CSV and JSON formats
 */
class ExportService
{
    private $db;
    private $exportDir;

    public function __construct()
    {
        $this->db = new Database();
        $this->exportDir = APPROOT . '/../public/assets/exports';
        $this->ensureExportDirectory();
    }

    /**
     * Request new export
     */
    public function requestExport(int $userId, string $type, string $format = 'csv', array $filters = []): ?int
    {
        // Validate format
        if (!in_array($format, ['csv', 'json'])) {
            return null;
        }

        // Create export record
        $this->db->query("INSERT INTO exports 
            (user_id, export_type, format, filters, expires_at)
            VALUES (:user_id, :type, :format, :filters, DATE_ADD(NOW(), INTERVAL 7 DAY))");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':type', $type);
        $this->db->bind(':format', $format);
        $this->db->bind(':filters', json_encode($filters));

        if ($this->db->execute()) {
            $exportId = $this->db->lastInsertId();

            // Queue for background processing
            $this->queueExport($exportId);

            return $exportId;
        }

        return null;
    }

    /**
     * Process export (called by background job)
     */
    public function processExport(int $exportId): bool
    {
        // Get export details
        $export = $this->getExport($exportId);

        if (!$export || $export->status !== 'pending') {
            return false;
        }

        // Update status to processing
        $this->updateExportStatus($exportId, 'processing');

        try {
            // Get data based on type
            $data = $this->getData($export);

            if (empty($data)) {
                throw new Exception('No data to export');
            }

            // Generate file
            $filename = $this->generateFilename($export);
            $filepath = $this->exportDir . '/' . $filename;

            if ($export->format === 'csv') {
                $this->exportToCSV($data, $filepath);
            } else {
                $this->exportToJSON($data, $filepath);
            }

            // Update export record
            $filesize = filesize($filepath);

            $this->db->query("UPDATE exports SET 
                status = 'completed',
                filename = :filename,
                file_path = :filepath,
                file_size = :filesize,
                total_records = :records,
                progress = 100,
                completed_at = NOW()
                WHERE id = :id");

            $this->db->bind(':id', $exportId);
            $this->db->bind(':filename', $filename);
            $this->db->bind(':filepath', $filepath);
            $this->db->bind(':filesize', $filesize);
            $this->db->bind(':records', count($data));
            $this->db->execute();

            // Send notification email
            $this->notifyExportReady($export->user_id, $exportId);

            return true;
        } catch (Exception $e) {
            // Mark as failed
            $this->db->query("UPDATE exports SET 
                status = 'failed',
                error_message = :error,
                completed_at = NOW()
                WHERE id = :id");

            $this->db->bind(':id', $exportId);
            $this->db->bind(':error', $e->getMessage());
            $this->db->execute();

            return false;
        }
    }

    /**
     * Get data based on export type
     */
    private function getData(object $export): array
    {
        $filters = json_decode($export->filters, true) ?? [];

        switch ($export->export_type) {
            case 'user_data':
                return $this->getUserData($export->user_id);

            case 'payments':
                return $this->getPayments($export->user_id, $filters);

            case 'transactions':
                return $this->getTransactions($export->user_id, $filters);

            case 'uploads':
                return $this->getUploads($export->user_id, $filters);

            default:
                return [];
        }
    }

    /**
     * Export user data (GDPR compliance)
     */
    private function getUserData(int $userId): array
    {
        $this->db->query("SELECT * FROM users WHERE user_id = :user_id");
        $this->db->bind(':user_id', $userId);
        $user = $this->db->single();

        if (!$user) {
            return [];
        }

        // Remove sensitive data
        unset($user->password);

        return [(array)$user];
    }

    /**
     * Export payments
     */
    private function getPayments(int $userId, array $filters): array
    {
        $sql = "SELECT * FROM payments WHERE user_id = :user_id";
        $bindings = [':user_id' => $userId];

        if (isset($filters['date_from'])) {
            $sql .= " AND created_at >= :date_from";
            $bindings[':date_from'] = $filters['date_from'];
        }

        if (isset($filters['date_to'])) {
            $sql .= " AND created_at <= :date_to";
            $bindings[':date_to'] = $filters['date_to'];
        }

        $sql .= " ORDER BY created_at DESC";

        $this->db->query($sql);
        foreach ($bindings as $key => $value) {
            $this->db->bind($key, $value);
        }

        $results = $this->db->resultSet();

        return array_map(function ($row) {
            return (array)$row;
        }, $results);
    }

    /**
     * Export transactions (generic)
     */
    private function getTransactions(int $userId, array $filters): array
    {
        // Implement based on your transaction table structure
        return $this->getPayments($userId, $filters);
    }

    /**
     * Export uploads
     */
    private function getUploads(int $userId, array $filters): array
    {
        $this->db->query("SELECT * FROM uploads 
            WHERE user_id = :user_id 
            ORDER BY created_at DESC");
        $this->db->bind(':user_id', $userId);

        $results = $this->db->resultSet();

        return array_map(function ($row) {
            return (array)$row;
        }, $results);
    }

    /**
     * Export to CSV
     */
    private function exportToCSV(array $data, string $filepath): void
    {
        $file = fopen($filepath, 'w');

        if (empty($data)) {
            fclose($file);
            return;
        }

        // Write header
        fputcsv($file, array_keys($data[0]));

        // Write data
        foreach ($data as $row) {
            fputcsv($file, $row);
        }

        fclose($file);
    }

    /**
     * Export to JSON
     */
    private function exportToJSON(array $data, string $filepath): void
    {
        $json = json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
        file_put_contents($filepath, $json);
    }

    /**
     * Queue export for background processing
     */
    private function queueExport(int $exportId): void
    {
        $queue = new JobQueue();
        $queue->push('ProcessExport', ['export_id' => $exportId]);
    }

    /**
     * Get export by ID
     */
    public function getExport(int $exportId): ?object
    {
        $this->db->query("SELECT * FROM exports WHERE id = :id");
        $this->db->bind(':id', $exportId);
        return $this->db->single() ?: null;
    }

    /**
     * Get user exports
     */
    public function getUserExports(int $userId, int $limit = 20): array
    {
        $this->db->query("SELECT * FROM exports 
            WHERE user_id = :user_id 
            ORDER BY requested_at DESC 
            LIMIT :limit");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        return $this->db->resultSet();
    }

    /**
     * Update export status
     */
    private function updateExportStatus(int $exportId, string $status): void
    {
        $this->db->query("UPDATE exports SET 
            status = :status,
            started_at = CASE WHEN :status = 'processing' THEN NOW() ELSE started_at END
            WHERE id = :id");

        $this->db->bind(':id', $exportId);
        $this->db->bind(':status', $status);
        $this->db->execute();
    }

    /**
     * Clean expired exports
     */
    public function cleanExpiredExports(): int
    {
        // Get expired exports
        $this->db->query("SELECT * FROM exports WHERE expires_at < NOW()");
        $expired = $this->db->resultSet();

        $deleted = 0;

        foreach ($expired as $export) {
            // Delete file
            if ($export->file_path && file_exists($export->file_path)) {
                unlink($export->file_path);
            }

            // Delete record
            $this->db->query("DELETE FROM exports WHERE id = :id");
            $this->db->bind(':id', $export->id);
            if ($this->db->execute()) {
                $deleted++;
            }
        }

        return $deleted;
    }

    /**
     * Notify user when export is ready
     */
    private function notifyExportReady(int $userId, int $exportId): void
    {
        // Get user email
        $this->db->query("SELECT email, name FROM users WHERE user_id = :user_id");
        $this->db->bind(':user_id', $userId);
        $user = $this->db->single();

        if ($user) {
            $emailService = new EmailService();
            $downloadUrl = URLROOT . "/v1/exports/{$exportId}/download";

            $emailService->send(
                $user->email,
                'Your Data Export is Ready',
                "Hi {$user->name},\n\nYour data export is ready for download.\n\nDownload link: {$downloadUrl}\n\nThis link expires in 7 days.\n\nThank you!"
            );
        }
    }

    /**
     * Generate unique filename
     */
    private function generateFilename(object $export): string
    {
        $timestamp = date('Ymd_His');
        $extension = $export->format;
        return "export_{$export->export_type}_{$timestamp}_{$export->id}.{$extension}";
    }

    /**
     * Ensure export directory exists
     */
    private function ensureExportDirectory(): void
    {
        if (!is_dir($this->exportDir)) {
            mkdir($this->exportDir, 0755, true);
        }
    }
}
