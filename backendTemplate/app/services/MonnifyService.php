<?php

/**
 * Monnify Payment Service
 * 
 * Implements Monnify payment gateway integration
 * API Documentation: https://docs.monnify.com
 */
class MonnifyService implements PaymentGatewayInterface
{
    private $apiKey;
    private $secretKey;
    private $contractCode;
    private $baseUrl;
    private $accessToken;
    private $tokenExpiry;

    public function __construct()
    {
        // Load configuration
        $config = $this->loadConfig();

        $this->apiKey = $config['api_key'];
        $this->secretKey = $config['secret_key'];
        $this->contractCode = $config['contract_code'];
        $this->baseUrl = $config['base_url'];
        $this->accessToken = null;
        $this->tokenExpiry = null;
    }

    /**
     * Load Monnify configuration
     */
    private function loadConfig(): array
    {
        $configPath = APPROOT . '/config/payment_config.php';

        if (file_exists($configPath)) {
            $config = require $configPath;
            return $config['monnify'] ?? [];
        }

        // Fallback to hardcoded values
        return [
            'api_key' => getenv('MONNIFY_API_KEY') ?: 'MK_TEST_YOUR_API_KEY',
            'secret_key' => getenv('MONNIFY_SECRET_KEY') ?: 'YOUR_SECRET_KEY',
            'contract_code' => getenv('MONNIFY_CONTRACT_CODE') ?: 'YOUR_CONTRACT_CODE',
            'base_url' => getenv('MONNIFY_BASE_URL') ?: 'https://sandbox.monnify.com'
        ];
    }

    /**
     * Get access token (with caching)
     */
    private function getAccessToken(): ?string
    {
        // Check if token is still valid
        if ($this->accessToken && $this->tokenExpiry && time() < $this->tokenExpiry) {
            return $this->accessToken;
        }

        try {
            $url = $this->baseUrl . '/api/v1/auth/login';

            $credentials = base64_encode($this->apiKey . ':' . $this->secretKey);

            $ch = curl_init($url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_HTTPHEADER, [
                'Authorization: Basic ' . $credentials,
                'Content-Type: application/json'
            ]);

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);

            if ($httpCode === 200) {
                $data = json_decode($response, true);

                if ($data && $data['requestSuccessful']) {
                    $this->accessToken = $data['responseBody']['accessToken'];
                    // Token expires in 1 hour, cache for 55 minutes
                    $this->tokenExpiry = time() + (55 * 60);
                    return $this->accessToken;
                }
            }

            error_log('Monnify authentication failed: ' . $response);
            return null;
        } catch (Exception $e) {
            error_log('Monnify auth error: ' . $e->getMessage());
            return null;
        }
    }

    /**
     * Initialize payment transaction
     */
    public function initializePayment(array $data): array
    {
        try {
            $token = $this->getAccessToken();

            if (!$token) {
                return [
                    'success' => false,
                    'message' => 'Failed to authenticate with Monnify',
                    'provider' => 'monnify'
                ];
            }

            $url = $this->baseUrl . '/api/v1/merchant/transactions/init-transaction';

            $payload = [
                'amount' => $data['amount'],
                'customerName' => $data['customer_name'] ?? 'Customer',
                'customerEmail' => $data['email'],
                'paymentReference' => $data['reference'],
                'paymentDescription' => $data['description'] ?? 'Payment for services',
                'currencyCode' => $data['currency'] ?? 'NGN',
                'contractCode' => $this->contractCode,
                'redirectUrl' => $data['callback_url'] ?? '',
                'paymentMethods' => $data['payment_methods'] ?? ['CARD', 'ACCOUNT_TRANSFER']
            ];

            $ch = curl_init($url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($payload));
            curl_setopt($ch, CURLOPT_HTTPHEADER, [
                'Authorization: Bearer ' . $token,
                'Content-Type: application/json'
            ]);

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);

            $result = json_decode($response, true);

            if ($httpCode === 200 && $result && $result['requestSuccessful']) {
                $responseBody = $result['responseBody'];

                return [
                    'success' => true,
                    'data' => [
                        'authorization_url' => $responseBody['checkoutUrl'],
                        'access_code' => $responseBody['transactionReference'],
                        'reference' => $data['reference']
                    ],
                    'message' => 'Payment initialized successfully',
                    'provider' => 'monnify'
                ];
            }

            return [
                'success' => false,
                'message' => $result['responseMessage'] ?? 'Payment initialization failed',
                'provider' => 'monnify',
                'error' => $result
            ];
        } catch (Exception $e) {
            error_log('Monnify initialization error: ' . $e->getMessage());
            return [
                'success' => false,
                'message' => 'Payment initialization error: ' . $e->getMessage(),
                'provider' => 'monnify'
            ];
        }
    }

    /**
     * Verify payment transaction
     */
    public function verifyPayment(string $reference): array
    {
        try {
            $token = $this->getAccessToken();

            if (!$token) {
                return [
                    'success' => false,
                    'verified' => false,
                    'message' => 'Failed to authenticate with Monnify',
                    'provider' => 'monnify'
                ];
            }

            $url = $this->baseUrl . '/api/v2/transactions/' . urlencode($reference);

            $ch = curl_init($url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_HTTPHEADER, [
                'Authorization: Bearer ' . $token,
                'Content-Type: application/json'
            ]);

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);

            $result = json_decode($response, true);

            if ($httpCode === 200 && $result && $result['requestSuccessful']) {
                $transaction = $result['responseBody'];
                $isPaid = strtoupper($transaction['paymentStatus']) === 'PAID';

                return [
                    'success' => true,
                    'verified' => $isPaid,
                    'data' => $transaction,
                    'amount' => $transaction['amountPaid'] ?? $transaction['amount'],
                    'currency' => $transaction['currencyCode'],
                    'reference' => $transaction['paymentReference'],
                    'status' => $transaction['paymentStatus'],
                    'paid_at' => $transaction['paidOn'] ?? null,
                    'message' => $isPaid ? 'Payment verified successfully' : 'Payment not completed',
                    'provider' => 'monnify'
                ];
            }

            return [
                'success' => false,
                'verified' => false,
                'message' => $result['responseMessage'] ?? 'Payment verification failed',
                'provider' => 'monnify',
                'error' => $result
            ];
        } catch (Exception $e) {
            error_log('Monnify verification error: ' . $e->getMessage());
            return [
                'success' => false,
                'verified' => false,
                'message' => 'Payment verification error: ' . $e->getMessage(),
                'provider' => 'monnify'
            ];
        }
    }

    /**
     * Process refund
     */
    public function processRefund(string $transactionId, ?float $amount = null): array
    {
        try {
            $token = $this->getAccessToken();

            if (!$token) {
                return [
                    'success' => false,
                    'message' => 'Failed to authenticate with Monnify',
                    'provider' => 'monnify'
                ];
            }

            $url = $this->baseUrl . '/api/v1/merchant/refunds';

            $payload = [
                'transactionReference' => $transactionId,
                'refundAmount' => $amount,
                'refundReference' => 'REFUND_' . time() . '_' . bin2hex(random_bytes(4))
            ];

            $ch = curl_init($url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($payload));
            curl_setopt($ch, CURLOPT_HTTPHEADER, [
                'Authorization: Bearer ' . $token,
                'Content-Type: application/json'
            ]);

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);

            $result = json_decode($response, true);

            if ($httpCode === 200 && $result && $result['requestSuccessful']) {
                return [
                    'success' => true,
                    'data' => $result['responseBody'],
                    'message' => 'Refund processed successfully',
                    'provider' => 'monnify'
                ];
            }

            return [
                'success' => false,
                'message' => $result['responseMessage'] ?? 'Refund processing failed',
                'provider' => 'monnify',
                'error' => $result
            ];
        } catch (Exception $e) {
            error_log('Monnify refund error: ' . $e->getMessage());
            return [
                'success' => false,
                'message' => 'Refund error: ' . $e->getMessage(),
                'provider' => 'monnify'
            ];
        }
    }

    /**
     * Verify webhook signature
     */
    public function verifyWebhookSignature(string $payload, string $signature): bool
    {
        try {
            $computedHash = hash_hmac('sha512', $payload, $this->secretKey);
            return hash_equals($computedHash, $signature);
        } catch (Exception $e) {
            error_log('Monnify webhook verification error: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Get provider name
     */
    public function getProviderName(): string
    {
        return 'monnify';
    }

    /**
     * Get provider configuration
     */
    public function getConfig(): array
    {
        return [
            'name' => 'Monnify',
            'contract_code' => $this->contractCode,
            'base_url' => $this->baseUrl,
            'supported_currencies' => ['NGN'],
            'supported_methods' => ['CARD', 'ACCOUNT_TRANSFER', 'USSD']
        ];
    }
}
