<?php

/**
 * API Keys Controller
 * 
 * Manage API keys for integrations
 */
class ApiKeys extends Controller
{
    private $apiKeyService;

    public function __construct()
    {
        $this->apiKeyService = new APIKeyService();
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
     * POST /v1/api-keys - Create new API key
     * 
     * Body:
     * - name: Key name (required)
     * - scopes: Array of scopes (required)
     * - description: Optional description
     * - rate_limit_minute: Requests per minute (optional, default: 60)
     * - rate_limit_hour: Requests per hour (optional, default: 1000)
     * - rate_limit_day: Requests per day (optional, default: 10000)
     * - expires_at: Expiry date (optional, format: Y-m-d H:i:s)
     */
    public function create()
    {
        $user = $this->RouteProtection();

        $name = $this->getData('name');
        $scopes = $this->getData('scopes');

        if (empty($name)) {
            return $this->sendResponse(false, 'API key name is required', [], 400);
        }

        if (empty($scopes) || !is_array($scopes)) {
            return $this->sendResponse(false, 'Scopes array is required', [], 400);
        }

        $data = [
            'name' => $name,
            'scopes' => $scopes,
            'description' => $this->getData('description'),
            'rate_limit_minute' => (int)($this->getData('rate_limit_minute') ?? 60),
            'rate_limit_hour' => (int)($this->getData('rate_limit_hour') ?? 1000),
            'rate_limit_day' => (int)($this->getData('rate_limit_day') ?? 10000),
            'expires_at' => $this->getData('expires_at')
        ];

        $newKey = $this->apiKeyService->generateKey($user->user_id, $data);

        if ($newKey) {
            return $this->sendResponse(true, 'API key created successfully', [
                'api_key' => $newKey->api_key,
                'id' => $newKey->id,
                'prefix' => $newKey->key_prefix,
                'name' => $newKey->name,
                'scopes' => $newKey->scopes,
                'warning' => 'Store this API key securely. It will not be shown again!'
            ], 201);
        }

        return $this->sendResponse(false, 'Failed to create API key', [], 500);
    }

    /**
     * GET /v1/api-keys - List user's API keys
     */
    public function index()
    {
        $user = $this->RouteProtection();

        $keys = $this->apiKeyService->getUserKeys($user->user_id);

        return $this->sendResponse(true, 'API keys retrieved', [
            'total' => count($keys),
            'keys' => $keys
        ]);
    }

    /**
     * DELETE /v1/api-keys/{id} - Revoke API key
     */
    public function delete($id)
    {
        $user = $this->RouteProtection();

        $success = $this->apiKeyService->revokeKey((int)$id, $user->user_id);

        if ($success) {
            return $this->sendResponse(true, 'API key revoked successfully');
        }

        return $this->sendResponse(false, 'Failed to revoke API key', [], 404);
    }

    /**
     * POST /v1/api-keys/{id}/rotate - Rotate API key
     */
    public function rotate($id)
    {
        $user = $this->RouteProtection();

        $newKey = $this->apiKeyService->rotateKey((int)$id, $user->user_id);

        if ($newKey) {
            return $this->sendResponse(true, 'API key rotated successfully', [
                'api_key' => $newKey->api_key,
                'id' => $newKey->id,
                'prefix' => $newKey->key_prefix,
                'warning' => 'Old key has been revoked. Store this new key securely!'
            ]);
        }

        return $this->sendResponse(false, 'Failed to rotate API key', [], 404);
    }

    /**
     * GET /v1/api-keys/{id}/stats - Get API key usage statistics
     */
    public function stats($id)
    {
        $user = $this->RouteProtection();

        // TODO: Verify key belongs to user

        $stats = $this->apiKeyService->getKeyStats((int)$id);

        return $this->sendResponse(true, 'API key statistics retrieved', $stats);
    }

    /**
     * GET /v1/api-keys/scopes - Get available scopes
     */
    public function scopes()
    {
        $user = $this->RouteProtection();

        return $this->sendResponse(true, 'Available scopes', [
            'scopes' => APIKeyService::SCOPES
        ]);
    }
}
