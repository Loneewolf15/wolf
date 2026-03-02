<?php

/**
 * Exports Controller
 * 
 * Manage data exports
 */
class Exports extends Controller
{
    private $exportService;

    public function __construct()
    {
        $this->exportService = new ExportService();
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
     * POST /v1/exports - Request new export
     * 
     * Body:
     * - type: Export type (user_data, payments, transactions, uploads)
     * - format: csv or json (optional, default: csv)
     * - filters: Object with filters (optional)
     *   - date_from: Start date
     *   - date_to: End date
     */
    public function create()
    {
        $user = $this->RouteProtection();

        $type = $this->getData('type');
        $format = $this->getData('format') ?? 'csv';
        $filters = $this->getData('filters') ?? [];

        if (empty($type)) {
            return $this->sendResponse(false, 'Export type is required', [], 400);
        }

        $validTypes = ['user_data', 'payments', 'transactions', 'uploads'];
        if (!in_array($type, $validTypes)) {
            return $this->sendResponse(false, 'Invalid export type', [
                'valid_types' => $validTypes
            ], 400);
        }

        // Rate limiting: Max 5 exports per hour per user
        $recentExports = $this->exportService->getUserExports($user->user_id, 100);
        $lastHour = array_filter($recentExports, function ($export) {
            return strtotime($export->requested_at) > strtotime('-1 hour');
        });

        if (count($lastHour) >= 5) {
            return $this->sendResponse(false, 'Export rate limit exceeded. Max 5 exports per hour.', [], 429);
        }

        $exportId = $this->exportService->requestExport($user->user_id, $type, $format, $filters);

        if ($exportId) {
            return $this->sendResponse(true, 'Export requested successfully', [
                'export_id' => $exportId,
                'status' => 'pending',
                'message' => 'Your export is being processed. You will receive an email when ready.'
            ], 201);
        }

        return $this->sendResponse(false, 'Failed to create export', [], 500);
    }

    /**
     * GET /v1/exports - List user's exports
     */
    public function index()
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 20), 100);

        $exports = $this->exportService->getUserExports($user->user_id, $limit);

        return $this->sendResponse(true, 'Exports retrieved', [
            'total' => count($exports),
            'exports' => $exports
        ]);
    }

    /**
     * GET /v1/exports/{id} - Get export details
     */
    public function get($id)
    {
        $user = $this->RouteProtection();

        $export = $this->exportService->getExport((int)$id);

        if (!$export) {
            return $this->sendResponse(false, 'Export not found', [], 404);
        }

        // Check ownership
        if ($export->user_id != $user->user_id) {
            return $this->sendResponse(false, 'Unauthorized', [], 403);
        }

        return $this->sendResponse(true, 'Export details', $export);
    }

    /**
     * GET /v1/exports/{id}/download - Download export file
     */
    public function download($id)
    {
        $user = $this->RouteProtection();

        $export = $this->exportService->getExport((int)$id);

        if (!$export) {
            return $this->sendResponse(false, 'Export not found', [], 404);
        }

        // Check ownership
        if ($export->user_id != $user->user_id) {
            return $this->sendResponse(false, 'Unauthorized', [], 403);
        }

        // Check status
        if ($export->status !== 'completed') {
            return $this->sendResponse(false, 'Export not ready yet', [
                'status' => $export->status,
                'progress' => $export->progress
            ], 400);
        }

        // Check if file exists
        if (!file_exists($export->file_path)) {
            return $this->sendResponse(false, 'Export file not found', [], 404);
        }

        // Download file
        header('Content-Type: application/octet-stream');
        header('Content-Disposition: attachment; filename="' . $export->filename . '"');
        header('Content-Length: ' . filesize($export->file_path));
        header('Cache-Control: no-cache, must-revalidate');
        header('Expires: 0');

        readfile($export->file_path);
        exit;
    }

    /**
     * GET /v1/exports/types - Get available export types
     */
    public function types()
    {
        $user = $this->RouteProtection();

        return $this->sendResponse(true, 'Available export types', [
            'types' => [
                'user_data' => 'Your personal data (GDPR compliant)',
                'payments' => 'Payment history',
                'transactions' => 'Transaction history',
                'uploads' => 'Uploaded files metadata'
            ],
            'formats' => ['csv', 'json']
        ]);
    }
}
