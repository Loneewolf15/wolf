<?php

/**
 * File Uploads Controller (v1)
 * 
 * RESTful API for file uploads
 */
class Uploads extends Controller
{
    private $uploadService;
    private $uploadModel;

    public function __construct()
    {
        $this->uploadService = new FileUploadService();
        $this->uploadModel = $this->model('Upload');
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
     * POST /v1/uploads - Upload file(s)
     */
    public function index()
    {
        // Rate limit
        $this->applyRateLimit('file_upload', 20, 3600);  // 20 uploads/hour

        // Require authentication
        $user = $this->RouteProtection();

        // Check if files exist
        if (empty($_FILES)) {
            return $this->sendResponse(false, 'No files uploaded', [], 400);
        }

        // Get the first file array
        $fileKey = array_key_first($_FILES);
        $files = $_FILES[$fileKey];

        // Get options
        $options = [
            'type' => $this->getData('type') ?? null,  // Force file type (images, videos, etc.)
            'resize' => $this->getData('resize') ?? null  // Override resize setting
        ];

        // Handle multiple files or single file
        if (is_array($files['name'])) {
            $result = $this->uploadMultiple($files, $user->user_id, $options);
        } else {
            $result = $this->uploadSingle($files, $user->user_id, $options);
        }

        return $result;
    }

    /**
     * Upload single file
     */
    private function uploadSingle(array $file, int $userId, array $options): void
    {
        $result = $this->uploadService->upload($file, $userId, $options);

        if (!$result['success']) {
            return $this->sendResponse(false, $result['error'], [], 400);
        }

        // Save to database
        $uploadId = $this->uploadModel->create([
            'user_id' => $userId,
            'filename' => basename($result['data']['file_path']),
            'original_name' => $result['data']['original_name'],
            'file_path' => $result['data']['file_path'],
            'file_url' => $result['data']['file_url'],  // Full URL with URLROOT
            'file_type' => $result['data']['file_type'],
            'mime_type' => $result['data']['mime_type'],
            'file_size' => $result['data']['size'],
            'storage_type' => 'local',
            'variants' => $result['data']['variants']
        ]);

        if (!$uploadId) {
            return $this->sendResponse(false, 'Failed to save upload record', [], 500);
        }

        return $this->sendResponse(true, 'File uploaded successfully', [
            'id' => $uploadId,
            'url' => $result['data']['file_url'],
            'file_path' => $result['data']['file_path'],
            'file_type' => $result['data']['file_type'],
            'size' => $result['data']['size'],
            'variants' => $result['data']['variants']
        ], 201);
    }

    /**
     * Upload multiple files
     */
    private function uploadMultiple(array $files, int $userId, array $options): void
    {
        $results = $this->uploadService->uploadMultiple($files, $userId, $options);

        $uploaded = [];
        $failed = [];

        foreach ($results as $result) {
            if ($result['success']) {
                // Save to database
                $uploadId = $this->uploadModel->create([
                    'user_id' => $userId,
                    'filename' => basename($result['data']['file_path']),
                    'original_name' => $result['data']['original_name'],
                    'file_path' => $result['data']['file_path'],
                    'file_url' => $result['data']['file_url'],
                    'file_type' => $result['data']['file_type'],
                    'mime_type' => $result['data']['mime_type'],
                    'file_size' => $result['data']['size'],
                    'storage_type' => 'local',
                    'variants' => $result['data']['variants']
                ]);

                $uploaded[] = [
                    'id' => $uploadId,
                    'url' => $result['data']['file_url'],
                    'file_type' => $result['data']['file_type']
                ];
            } else {
                $failed[] = $result['error'];
            }
        }

        return $this->sendResponse(true, 'Upload process completed', [
            'uploaded' => $uploaded,
            'uploaded_count' => count($uploaded),
            'failed' => $failed,
            'failed_count' => count($failed)
        ], 201);
    }

    /**
     * GET /v1/uploads - List user's uploads
     */
    public function list()
    {
        $this->applyRateLimit('file_list', 100, 3600);
        $user = $this->RouteProtection();

        $page = (int)($this->getData('page') ?? 1);
        $limit = (int)($this->getData('limit') ?? 20);
        $fileType = $this->getData('type') ?? null;  // Filter by type

        $limit = min($limit, 100);  // Max 100 per page
        $offset = ($page - 1) * $limit;

        $uploads = $this->uploadModel->findByUser($user->user_id, $limit, $offset, $fileType);
        $total = $this->uploadModel->countByUser($user->user_id, $fileType);

        return $this->sendResponse(true, 'Uploads retrieved', [
            'uploads' => $uploads,
            'pagination' => [
                'page' => $page,
                'limit' => $limit,
                'total' => $total,
                'pages' => ceil($total / $limit)
            ]
        ]);
    }

    /**
     * GET /v1/uploads/{id} - Get upload details
     */
    public function get($id)
    {
        $this->applyRateLimit('file_get', 300, 3600);
        $user = $this->RouteProtection();

        $upload = $this->uploadModel->findById((int)$id);

        if (!$upload) {
            return $this->sendResponse(false, 'Upload not found', [], 404);
        }

        // Check ownership
        if ($upload->user_id != $user->user_id) {
            return $this->sendResponse(false, 'Unauthorized', [], 403);
        }

        return $this->sendResponse(true, 'Upload retrieved', $upload);
    }

    /**
     * DELETE /v1/uploads/{id} - Delete upload
     */
    public function delete($id)
    {
        $this->applyRateLimit('file_delete', 50, 3600);
        $user = $this->RouteProtection();

        $upload = $this->uploadModel->findById((int)$id);

        if (!$upload) {
            return $this->sendResponse(false, 'Upload not found', [], 404);
        }

        // Check ownership
        if ($upload->user_id != $user->user_id) {
            return $this->sendResponse(false, 'Unauthorized', [], 403);
        }

        // Delete physical file
        $storage = new LocalStorageService();
        $storage->delete($upload->file_path);

        // Delete variants if exist
        if ($upload->variants) {
            foreach ($upload->variants as $variantUrl) {
                // Extract path from URL and delete
                $variantPath = str_replace(URLROOT, '', $variantUrl);
                $storage->delete($variantPath);
            }
        }

        // Delete database record
        $this->uploadModel->delete((int)$id);

        return $this->sendResponse(true, 'Upload deleted successfully');
    }

    /**
     * GET /v1/uploads/quota - Get user's quota usage
     */
    public function quota()
    {
        $this->applyRateLimit('file_quota', 100, 3600);
        $user = $this->RouteProtection();

        $usage = $this->uploadModel->getUserQuota($user->user_id);

        // Get limits from config
        $config = require APPROOT . '/config/upload_config.php';
        $limits = $config['quotas'] ?? [];

        return $this->sendResponse(true, 'Quota retrieved', [
            'usage' => $usage,
            'limits' => [
                'max_files' => $limits['max_files_per_user'] ?? null,
                'max_storage_bytes' => $limits['max_storage_per_user'] ?? null,
                'max_storage_mb' => isset($limits['max_storage_per_user'])
                    ? round($limits['max_storage_per_user'] / 1024 / 1024, 2)
                    : null
            ],
            'percentage_used' => isset($limits['max_storage_per_user'])
                ? round(($usage['total_bytes'] / $limits['max_storage_per_user']) * 100, 2)
                : 0
        ]);
    }
}
