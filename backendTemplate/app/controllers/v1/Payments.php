<?php

/**
 * Payments Controller
 * 
 * Handles payment operations for multiple providers (Paystack, Monnify)
 */
class Payments extends Controller
{
    private $paymentModel;
    private $config;

    public function __construct()
    {
        parent::__construct();
        $this->paymentModel = $this->model('Payment');
        $this->config = $this->loadConfig();
    }

    /**
     * Load payment configuration
     */
    private function loadConfig(): array
    {
        $configPath = APPROOT . '/config/payment_config.php';
        return file_exists($configPath) ? require $configPath : [];
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
     * API index
     */
    public function index()
    {
        $this->sendResponse(true, 'Payments API v1', [
            'endpoints' => [
                'POST /v1/payments/initialize' => 'Initialize payment',
                'GET /v1/payments/verify/{reference}' => 'Verify payment',
                'POST /v1/payments/webhook/{provider}' => 'Payment webhook',
                'GET /v1/payments/transaction/{id}' => 'Get transaction',
                'GET /v1/payments/transactions' => 'Get user transactions',
                'POST /v1/payments/refund' => 'Process refund'
            ],
            'supported_providers' => GatewayFactory::getSupportedProviders()
        ]);
    }

    /**
     * POST /v1/payments/initialize
     * Initialize payment transaction
     */
    public function initialize()
    {
        try {
            // Apply rate limiting
            $this->applyRateLimit('payment_initialize');

            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                return $this->sendResponse(false, 'Invalid request method', [], 405);
            }

            // Authenticate user
            $user = $this->RouteProtection();
            $data = $this->getData();

            // Validate required fields
            $required = ['provider', 'amount', 'currency'];
            foreach ($required as $field) {
                if (empty($data[$field])) {
                    return $this->sendResponse(false, ucfirst($field) . ' is required', [], 400);
                }
            }

            // Validate provider
            if (!GatewayFactory::isSupported($data['provider'])) {
                return $this->sendResponse(false, 'Unsupported payment provider', [
                    'supported_providers' => GatewayFactory::getSupportedProviders()
                ], 400);
            }

            // Validate amount
            $amount = (float)$data['amount'];
            $minAmount = $this->config['limits']['min_amount'] ?? 100;
            $maxAmount = $this->config['limits']['max_amount'] ?? 5000000;

            if ($amount < $minAmount || $amount > $maxAmount) {
                return $this->sendResponse(false, "Amount must be between {$minAmount} and {$maxAmount}", [], 400);
            }

            // Validate currency
            $supportedCurrencies = $this->config['supported_currencies'] ?? ['NGN'];
            if (!in_array($data['currency'], $supportedCurrencies)) {
                return $this->sendResponse(false, 'Unsupported currency', [
                    'supported_currencies' => $supportedCurrencies
                ], 400);
            }

            // Generate unique reference
            $reference = $this->generateReference($data['provider']);

            // Ensure reference is unique
            while ($this->paymentModel->referenceExists($reference)) {
                $reference = $this->generateReference($data['provider']);
            }

            // Create gateway instance
            $gateway = GatewayFactory::create($data['provider']);

            // Prepare payment data
            $paymentData = [
                'email' => $user->email,
                'amount' => $amount,
                'currency' => $data['currency'],
                'reference' => $reference,
                'customer_name' => $user->name ?? 'Customer',
                'user_id' => $user->user_id,
                'callback_url' => $data['callback_url'] ?? ($this->config[$data['provider']]['callback_url'] ?? URLROOT . 'payment/callback'),
                'description' => $data['description'] ?? 'Payment for services',
                'metadata' => $data['metadata'] ?? []
            ];

            // Initialize payment with gateway
            $result = $gateway->initializePayment($paymentData);

            if ($result['success']) {
                // Log transaction to database
                $this->paymentModel->createTransaction([
                    'user_id' => $user->user_id,
                    'provider' => $data['provider'],
                    'reference' => $reference,
                    'amount' => $amount,
                    'currency' => $data['currency'],
                    'status' => 'pending',
                    'metadata' => json_encode($paymentData['metadata'])
                ]);

                return $this->sendResponse(true, 'Payment initialized successfully', [
                    'authorization_url' => $result['data']['authorization_url'] ?? null,
                    'access_code' => $result['data']['access_code'] ?? null,
                    'reference' => $reference,
                    'provider' => $data['provider']
                ]);
            }

            return $this->sendResponse(false, $result['message'] ?? 'Payment initialization failed', [], 400);
        } catch (Exception $e) {
            error_log('Payment initialization error: ' . $e->getMessage());
            return $this->sendResponse(false, 'An error occurred while initializing payment', [], 500);
        }
    }

    /**
     * GET /v1/payments/verify/{reference}
     * Verify payment transaction
     */
    public function verify($reference = null)
    {
        try {
            // Apply rate limiting
            $this->applyRateLimit('payment_verify');

            // Authenticate user
            $user = $this->RouteProtection();

            if (!$reference) {
                return $this->sendResponse(false, 'Payment reference is required', [], 400);
            }

            // Get transaction from database
            $transaction = $this->paymentModel->findByReference($reference);

            if (!$transaction) {
                return $this->sendResponse(false, 'Transaction not found', [], 404);
            }

            // Verify user owns this transaction
            if ($transaction->user_id != $user->user_id) {
                return $this->sendResponse(false, 'Unauthorized access to transaction', [], 403);
            }

            // If already verified, return cached result
            if ($transaction->status === 'successful') {
                return $this->sendResponse(true, 'Payment already verified', [
                    'verified' => true,
                    'amount' => $transaction->amount,
                    'currency' => $transaction->currency,
                    'status' => $transaction->status,
                    'provider' => $transaction->provider,
                    'verified_at' => $transaction->verified_at
                ]);
            }

            // Create gateway instance
            $gateway = GatewayFactory::create($transaction->provider);

            // Verify payment with provider
            $result = $gateway->verifyPayment($reference);

            if ($result['success'] && ($result['verified'] ?? false)) {
                // Update transaction status
                $this->paymentModel->updateTransaction($transaction->id, [
                    'status' => 'successful',
                    'verified_at' => date('Y-m-d H:i:s'),
                    'gateway_response' => json_encode($result['data'] ?? [])
                ]);

                return $this->sendResponse(true, 'Payment verified successfully', [
                    'verified' => true,
                    'amount' => $result['amount'] ?? $transaction->amount,
                    'currency' => $result['currency'] ?? $transaction->currency,
                    'status' => $result['status'] ?? 'successful',
                    'provider' => $transaction->provider,
                    'reference' => $reference
                ]);
            }

            // Payment not successful
            if (isset($result['status']) && in_array($result['status'], ['failed', 'abandoned'])) {
                $this->paymentModel->updateTransaction($transaction->id, [
                    'status' => 'failed',
                    'gateway_response' => json_encode($result['data'] ?? [])
                ]);
            }

            return $this->sendResponse(false, $result['message'] ?? 'Payment verification failed', [
                'verified' => false,
                'reference' => $reference
            ], 400);
        } catch (Exception $e) {
            error_log('Payment verification error: ' . $e->getMessage());
            return $this->sendResponse(false, 'An error occurred while verifying payment', [], 500);
        }
    }

    /**
     * POST /v1/payments/webhook/{provider}
     * Handle payment webhooks from providers
     */
    public function webhook($provider = null)
    {
        try {
            // Webhook doesn't need user auth - uses signature verification
            // Apply rate limiting by IP
            $this->applyRateLimit('payment_webhook');

            if (!$provider || !GatewayFactory::isSupported($provider)) {
                http_response_code(400);
                echo json_encode(['error' => 'Invalid provider']);
                exit;
            }

            // Create gateway instance
            $gateway = GatewayFactory::create($provider);

            // Get webhook payload
            $payload = file_get_contents('php://input');

            // Get signature from headers
            $signature = '';
            if ($provider === 'paystack') {
                $signature = $_SERVER['HTTP_X_PAYSTACK_SIGNATURE'] ?? '';
            } elseif ($provider === 'monnify') {
                $signature = $_SERVER['HTTP_MONNIFY_SIGNATURE'] ?? '';
            }

            // Verify webhook signature
            if (!$gateway->verifyWebhookSignature($payload, $signature)) {
                error_log("Webhook signature verification failed for provider: {$provider}");
                http_response_code(401);
                echo json_encode(['error' => 'Invalid signature']);
                exit;
            }

            // Parse webhook event
            $event = json_decode($payload, true);

            if (!$event) {
                http_response_code(400);
                echo json_encode(['error' => 'Invalid JSON']);
                exit;
            }

            // Process event based on type
            $eventType = $event['event'] ?? $event['eventType'] ?? null;

            if (in_array($eventType, ['charge.success', 'SUCCESSFUL_TRANSACTION'])) {
                // Extract reference from event
                $reference = $event['data']['reference'] ?? $event['eventData']['paymentReference'] ?? null;

                if ($reference) {
                    $transaction = $this->paymentModel->findByReference($reference);

                    if ($transaction && $transaction->status !== 'successful') {
                        // Update transaction status
                        $this->paymentModel->updateTransaction($transaction->id, [
                            'status' => 'successful',
                            'verified_at' => date('Y-m-d H:i:s'),
                            'gateway_response' => $payload
                        ]);

                        error_log("Webhook: Payment {$reference} marked as successful");
                    }
                }
            }

            // Return success response
            http_response_code(200);
            echo json_encode(['status' => 'success']);
            exit;
        } catch (Exception $e) {
            error_log('Webhook processing error: ' . $e->getMessage());
            http_response_code(500);
            echo json_encode(['error' => 'Internal server error']);
            exit;
        }
    }

    /**
     * GET /v1/payments/transaction/{id}
     * Get transaction details
     */
    public function transaction($id = null)
    {
        try {
            $this->applyRateLimit('payment_verify');

            $user = $this->RouteProtection();

            if (!$id || !is_numeric($id)) {
                return $this->sendResponse(false, 'Valid transaction ID is required', [], 400);
            }

            $transaction = $this->paymentModel->findById((int)$id);

            if (!$transaction) {
                return $this->sendResponse(false, 'Transaction not found', [], 404);
            }

            // Verify user owns this transaction
            if ($transaction->user_id != $user->user_id) {
                return $this->sendResponse(false, 'Unauthorized access', [], 403);
            }

            return $this->sendResponse(true, 'Transaction retrieved successfully', [
                'transaction' => $transaction
            ]);
        } catch (Exception $e) {
            error_log('Transaction retrieval error: ' . $e->getMessage());
            return $this->sendResponse(false, 'An error occurred', [], 500);
        }
    }

    /**
     * GET /v1/payments/transactions
     * Get user transactions with pagination
     */
    public function transactions()
    {
        try {
            $this->applyRateLimit('payment_verify');

            $user = $this->RouteProtection();
            $data = $this->getData();

            $page = (int)($data['page'] ?? $_GET['page'] ?? 1);
            $perPage = (int)($data['per_page'] ?? $_GET['per_page'] ?? 20);
            $perPage = min($perPage, 100); // Max 100 per page

            $offset = ($page - 1) * $perPage;

            $transactions = $this->paymentModel->getUserTransactions($user->user_id, $perPage, $offset);
            $stats = $this->paymentModel->getTransactionStats($user->user_id);

            return $this->sendResponse(true, 'Transactions retrieved successfully', [
                'transactions' => $transactions,
                'stats' => $stats,
                'pagination' => [
                    'page' => $page,
                    'per_page' => $perPage,
                    'total' => $stats['total_transactions']
                ]
            ]);
        } catch (Exception $e) {
            error_log('Transactions retrieval error: ' . $e->getMessage());
            return $this->sendResponse(false, 'An error occurred', [], 500);
        }
    }

    /**
     * POST /v1/payments/refund
     * Process refund for a transaction
     */
    public function refund()
    {
        try {
            $this->applyRateLimit('payment_refund');

            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                return $this->sendResponse(false, 'Invalid request method', [], 405);
            }

            $user = $this->RouteProtection();
            $data = $this->getData();

            if (empty($data['transaction_id'])) {
                return $this->sendResponse(false, 'Transaction ID is required', [], 400);
            }

            $transaction = $this->paymentModel->findById((int)$data['transaction_id']);

            if (!$transaction) {
                return $this->sendResponse(false, 'Transaction not found', [], 404);
            }

            // Verify ownership
            if ($transaction->user_id != $user->user_id) {
                return $this->sendResponse(false, 'Unauthorized access', [], 403);
            }

            // Check if refundable
            if ($transaction->status !== 'successful') {
                return $this->sendResponse(false, 'Only successful transactions can be refunded', [], 400);
            }

            if ($transaction->status === 'refunded') {
                return $this->sendResponse(false, 'Transaction already refunded', [], 400);
            }

            // Create gateway instance
            $gateway = GatewayFactory::create($transaction->provider);

            // Process refund
            $amount = isset($data['amount']) ? (float)$data['amount'] : null;
            $result = $gateway->processRefund($transaction->reference, $amount);

            if ($result['success']) {
                // Update transaction
                $this->paymentModel->updateTransaction($transaction->id, [
                    'status' => 'refunded',
                    'refunded_at' => date('Y-m-d H:i:s'),
                    'gateway_response' => json_encode($result['data'] ?? [])
                ]);

                return $this->sendResponse(true, 'Refund processed successfully', [
                    'transaction_id' => $transaction->id,
                    'refund_amount' => $amount ?? $transaction->amount
                ]);
            }

            return $this->sendResponse(false, $result['message'] ?? 'Refund processing failed', [], 400);
        } catch (Exception $e) {
            error_log('Refund processing error: ' . $e->getMessage());
            return $this->sendResponse(false, 'An error occurred while processing refund', [], 500);
        }
    }

    /**
     * Generate unique payment reference
     */
    private function generateReference(string $provider): string
    {
        $prefix = strtoupper(substr($provider, 0, 3));
        return $prefix . '_' . time() . '_' . bin2hex(random_bytes(6));
    }
}
