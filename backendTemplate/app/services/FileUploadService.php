<?php

/**
 * File Upload Service
 * 
 * Core service for handling file uploads with validation and processing
 * Reads configuration from upload_config.php
 */
class FileUploadService
{
    private $config;
    private $storage;
    private $database;

    public function __construct()
    {
        $this->loadConfig();
        $this->initializeStorage();
        $this->database = new Database();
    }

    /**
     * Load upload configuration
     */
    private function loadConfig(): void
    {
        $configPath = APPROOT . '/config/upload_config.php';

        if (!file_exists($configPath)) {
            throw new Exception('Upload configuration file not found');
        }

        $this->config = require $configPath;
    }

    /**
     * Initialize storage service based on config
     */
    private function initializeStorage(): void
    {
        $mode = $this->config['storage_mode'] ?? 'local';

        switch ($mode) {
            case 'local':
                $this->storage = new LocalStorageService();
                break;

            case 'cloud':
                $this->storage = new CloudStorageService();
                break;

            case 'hybrid':
                // For hybrid, will decide per file type
                $this->storage = null;
                break;

            default:
                throw new Exception('Invalid storage mode');
        }
    }

    /**
     * Upload single file
     * 
     * @param array $file PHP $_FILES array element
     * @param int $userId User ID
     * @param array $options Additional options (type, resize, etc.)
     * @return array Upload result with file URL
     */
    public function upload(array $file, int $userId, array $options = []): array
    {
        try {
            // Validate file
            $validation = $this->validateFile($file, $options);
            if (!$validation['valid']) {
                return [
                    'success' => false,
                    'error' => $validation['error']
                ];
            }

            $fileType = $validation['type'];

            // Check user quota
            $quotaCheck = $this->checkUserQuota($userId, $file['size']);
            if (!$quotaCheck['allowed']) {
                return [
                    'success' => false,
                    'error' => $quotaCheck['error']
                ];
            }

            // Store file (local or cloud)
            $storageResult = $this->storage->store($file, $userId, $options);

            if (!$storageResult['success']) {
                return $storageResult;
            }

            // Process image if enabled and file is image
            $variants = [];
            if ($fileType === 'images' && $this->shouldProcessImage()) {
                $variants = $this->processImage($storageResult['full_path'], $userId);
            }

            return [
                'success' => true,
                'data' => [
                    'file_path' => $storageResult['file_path'],  // Relative path
                    'file_url' => $storageResult['file_url'],    // Full URL with URLROOT
                    'file_type' => $fileType,
                    'mime_type' => $file['type'],
                    'size' => $storageResult['size'],
                    'original_name' => $file['name'],
                    'variants' => $variants
                ]
            ];
        } catch (Exception $e) {
            error_log('Upload error: ' . $e->getMessage());
            return [
                'success' => false,
                'error' => 'Upload failed: ' . $e->getMessage()
            ];
        }
    }

    /**
     * Upload multiple files
     */
    public function uploadMultiple(array $files, int $userId, array $options = []): array
    {
        $results = [];
        $normalized = $this->normalizeFilesArray($files);

        foreach ($normalized as $file) {
            $results[] = $this->upload($file, $userId, $options);
        }

        return $results;
    }

    /**
     * Validate file against configuration
     */
    private function validateFile(array $file, array $options): array
    {
        // Check for upload errors
        if ($file['error'] !== UPLOAD_ERR_OK) {
            return [
                'valid' => false,
                'error' => $this->getUploadErrorMessage($file['error'])
            ];
        }

        // Determine file type
        $fileType = $options['type'] ?? $this->detectFileType($file);

        // Check if file type is enabled
        if (!isset($this->config['allowed_types'][$fileType])) {
            return [
                'valid' => false,
                'error' => 'Unknown file type'
            ];
        }

        $typeConfig = $this->config['allowed_types'][$fileType];

        if (!$typeConfig['enabled']) {
            return [
                'valid' => false,
                'error' => ucfirst($fileType) . ' uploads are disabled'
            ];
        }

        // Validate file extension
        $extension = strtolower(pathinfo($file['name'], PATHINFO_EXTENSION));
        if (!in_array($extension, $typeConfig['extensions'])) {
            return [
                'valid' => false,
                'error' => 'File extension not allowed: ' . $extension
            ];
        }

        // Validate MIME type
        if (!in_array($file['type'], $typeConfig['mime_types'])) {
            return [
                'valid' => false,
                'error' => 'File type not allowed: ' . $file['type']
            ];
        }

        // Validate file size
        if ($file['size'] > $typeConfig['max_size']) {
            $maxSizeMB = round($typeConfig['max_size'] / 1024 / 1024, 2);
            return [
                'valid' => false,
                'error' => "File too large. Maximum size: {$maxSizeMB}MB"
            ];
        }

        // Check magic bytes if enabled
        if ($this->config['security']['check_magic_bytes'] ?? false) {
            if (!$this->validateMagicBytes($file, $typeConfig['mime_types'])) {
                return [
                    'valid' => false,
                    'error' => 'File content does not match extension'
                ];
            }
        }

        // Block executable files
        if ($this->config['security']['block_executable'] ?? true) {
            $dangerous = ['exe', 'sh', 'bat', 'cmd', 'com', 'pif', 'scr'];
            if (in_array($extension, $dangerous)) {
                return [
                    'valid' => false,
                    'error' => 'Executable files are not allowed'
                ];
            }
        }

        return [
            'valid' => true,
            'type' => $fileType
        ];
    }

    /**
     * Check user storage quota
     */
    private function checkUserQuota(int $userId, int $fileSize): array
    {
        $quotas = $this->config['quotas'] ?? [];

        // Get current user storage usage (from database)
        $this->database->query("SELECT 
            COUNT(*) as file_count,
            COALESCE(SUM(file_size), 0) as total_size
            FROM uploads 
            WHERE user_id = :user_id");
        $this->database->bind(':user_id', $userId);
        $usage = $this->database->single();

        // Check file count limit
        if (isset($quotas['max_files_per_user'])) {
            if ($usage->file_count >= $quotas['max_files_per_user']) {
                return [
                    'allowed' => false,
                    'error' => 'Maximum file limit reached'
                ];
            }
        }

        // Check storage limit
        if (isset($quotas['max_storage_per_user'])) {
            if (($usage->total_size + $fileSize) > $quotas['max_storage_per_user']) {
                $maxMB = round($quotas['max_storage_per_user'] / 1024 / 1024, 2);
                return [
                    'allowed' => false,
                    'error' => "Storage quota exceeded. Maximum: {$maxMB}MB"
                ];
            }
        }

        return ['allowed' => true];
    }

    /**
     * Process image (resize, compress) if enabled
     */
    private function processImage(string $sourcePath, int $userId): array
    {
        $variants = [];

        try {
            $processor = new ImageProcessor();
            $config = $this->config['image_processing'];

            // Resize to multiple sizes
            if ($config['resize']['enabled']) {
                foreach ($config['resize']['sizes'] as $name => $dimensions) {
                    $variantPath = $processor->resize(
                        $sourcePath,
                        $dimensions['width'],
                        $dimensions['height'],
                        $name
                    );

                    // Generate URL for variant
                    $storage = new LocalStorageService();
                    $relativePath = str_replace(APPROOT . '/../public/', '', $variantPath);
                    $variants[$name] = $storage->generateUrl($relativePath);
                }
            }

            // Compress original
            if ($config['compression']['enabled']) {
                $processor->compress($sourcePath, $config['compression']['quality']);
            }
        } catch (Exception $e) {
            error_log('Image processing error: ' . $e->getMessage());
        }

        return $variants;
    }

    /**
     * Check if image processing is enabled
     */
    private function shouldProcessImage(): bool
    {
        return ($this->config['image_processing']['enabled'] ?? 0) === 1;
    }

    /**
     * Validate magic bytes (actual file content)
     */
    private function validateMagicBytes(array $file, array $allowedMimes): bool
    {
        $finfo = finfo_open(FILEINFO_MIME_TYPE);
        $actualMime = finfo_file($finfo, $file['tmp_name']);
        finfo_close($finfo);

        return in_array($actualMime, $allowedMimes);
    }

    /**
     * Detect file type from MIME
     */
    private function detectFileType(array $file): string
    {
        $mime = $file['type'];

        if (strpos($mime, 'image/') === 0) return 'images';
        if (strpos($mime, 'video/') === 0) return 'videos';
        if (strpos($mime, 'audio/') === 0) return 'audio';

        return 'documents';
    }

    /**
     * Normalize $_FILES array for multiple uploads
     */
    private function normalizeFilesArray(array $files): array
    {
        // Check if single file
        if (!is_array($files['name'])) {
            return [$files];
        }

        // Normalize multiple files
        $normalized = [];
        $fileCount = count($files['name']);

        for ($i = 0; $i < $fileCount; $i++) {
            $normalized[] = [
                'name' => $files['name'][$i],
                'type' => $files['type'][$i],
                'tmp_name' => $files['tmp_name'][$i],
                'error' => $files['error'][$i],
                'size' => $files['size'][$i]
            ];
        }

        return $normalized;
    }

    /**
     * Get upload error message
     */
    private function getUploadErrorMessage(int $errorCode): string
    {
        $errors = [
            UPLOAD_ERR_INI_SIZE => 'File exceeds upload_max_filesize',
            UPLOAD_ERR_FORM_SIZE => 'File exceeds MAX_FILE_SIZE',
            UPLOAD_ERR_PARTIAL => 'File only partially uploaded',
            UPLOAD_ERR_NO_FILE => 'No file uploaded',
            UPLOAD_ERR_NO_TMP_DIR => 'Missing temporary folder',
            UPLOAD_ERR_CANT_WRITE => 'Failed to write to disk',
            UPLOAD_ERR_EXTENSION => 'Upload stopped by extension'
        ];

        return $errors[$errorCode] ?? 'Unknown upload error';
    }
}
