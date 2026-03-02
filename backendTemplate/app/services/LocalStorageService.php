<?php

/**
 * Local Storage Service
 * 
 * Handles file storage on local filesystem
 * Stores filepath AND full URL with URLROOT in database
 */
class LocalStorageService
{
    private $config;
    private $baseDir;

    public function __construct()
    {
        $this->loadConfig();
        $this->baseDir = APPROOT . '/../' . $this->config['upload_paths']['local_base'];
        $this->ensureDirectoryExists($this->baseDir);
    }

    /**
     * Load upload configuration
     */
    private function loadConfig(): void
    {
        $configPath = APPROOT . '/config/upload_config.php';
        $this->config = file_exists($configPath) ? require $configPath : [];
    }

    /**
     * Store file locally and return file info with full URL
     * 
     * @param array $file PHP $_FILES array element
     * @param int $userId User ID for organization
     * @param array $options Additional options
     * @return array File information with filepath AND full URL
     */
    public function store(array $file, int $userId, array $options = []): array
    {
        try {
            // Generate storage path based on structure
            $structure = $this->config['upload_paths']['structure'] ?? 'user_id';
            $relativePath = $this->generatePath($userId, $file, $structure);

            // Create directory if needed
            $fullDir = $this->baseDir . '/' . dirname($relativePath);
            $this->ensureDirectoryExists($fullDir);

            // Full file path
            $fullPath = $this->baseDir . '/' . $relativePath;

            // Move uploaded file
            if (!move_uploaded_file($file['tmp_name'], $fullPath)) {
                throw new Exception('Failed to move uploaded file');
            }

            // Generate full URL (URLROOT + relative path)
            $fileUrl = $this->generateUrl($relativePath);

            return [
                'success' => true,
                'file_path' => $relativePath,  // Relative path for filesystem
                'file_url' => $fileUrl,  // Full URL with URLROOT
                'full_path' => $fullPath,  // Absolute path (for processing)
                'size' => filesize($fullPath)
            ];
        } catch (Exception $e) {
            error_log('Local storage error: ' . $e->getMessage());
            return [
                'success' => false,
                'error' => $e->getMessage()
            ];
        }
    }

    /**
     * Delete file from local storage
     */
    public function delete(string $relativePath): bool
    {
        try {
            $fullPath = $this->baseDir . '/' . $relativePath;

            if (file_exists($fullPath)) {
                return unlink($fullPath);
            }

            return false;
        } catch (Exception $e) {
            error_log('File deletion error: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Generate full URL with URLROOT
     * 
     * Example: uploads/users/123/file.jpg → http://yourdomain.com/uploads/users/123/file.jpg
     */
    public function generateUrl(string $relativePath): string
    {
        // Remove 'public/' prefix if exists
        $path = str_replace('public/', '', $relativePath);

        // Concatenate URLROOT with path
        $url = rtrim(URLROOT, '/') . '/' . ltrim($path, '/');

        return $url;
    }

    /**
     * Get file URL from relative path
     */
    public function getUrl(string $relativePath): string
    {
        return $this->generateUrl($relativePath);
    }

    /**
     * Generate storage path based on structure setting
     */
    private function generatePath(int $userId, array $file, string $structure): string
    {
        $filename = $this->generateUniqueFilename($file);

        switch ($structure) {
            case 'user_id':
                return "users/{$userId}/{$filename}";

            case 'date':
                $date = date('Y/m/d');
                return "{$date}/{$filename}";

            case 'type':
                $type = $this->detectFileType($file);
                return "{$type}/{$filename}";

            case 'flat':
            default:
                return $filename;
        }
    }

    /**
     * Generate unique filename
     */
    private function generateUniqueFilename(array $file): string
    {
        $extension = strtolower(pathinfo($file['name'], PATHINFO_EXTENSION));

        // Sanitize original filename
        $originalName = pathinfo($file['name'], PATHINFO_FILENAME);
        $sanitized = preg_replace('/[^a-zA-Z0-9_-]/', '_', $originalName);
        $sanitized = substr($sanitized, 0, 50);  // Limit length

        // Generate unique ID
        $uniqueId = time() . '_' . bin2hex(random_bytes(8));

        return "{$sanitized}_{$uniqueId}.{$extension}";
    }

    /**
     * Detect file type category
     */
    private function detectFileType(array $file): string
    {
        $mimeType = $file['type'];

        if (strpos($mimeType, 'image/') === 0) {
            return 'images';
        } elseif (strpos($mimeType, 'video/') === 0) {
            return 'videos';
        } elseif (strpos($mimeType, 'audio/') === 0) {
            return 'audio';
        } else {
            return 'documents';
        }
    }

    /**
     * Ensure directory exists
     */
    private function ensureDirectoryExists(string $dir): void
    {
        if (!is_dir($dir)) {
            if (!mkdir($dir, 0755, true)) {
                throw new Exception("Failed to create directory: {$dir}");
            }
        }
    }

    /**
     * Get file info
     */
    public function getFileInfo(string $relativePath): ?array
    {
        $fullPath = $this->baseDir . '/' . $relativePath;

        if (!file_exists($fullPath)) {
            return null;
        }

        return [
            'exists' => true,
            'size' => filesize($fullPath),
            'mime_type' => mime_content_type($fullPath),
            'url' => $this->generateUrl($relativePath),
            'modified_at' => date('Y-m-d H:i:s', filemtime($fullPath))
        ];
    }
}
