<?php

/**
 * Paystack Payment Service
 * Handles payment operations using Paystack API
 */
class PaystackService implements PaymentGatewayInterface
{
    private $secretKey;
    private $publicKey;
    private $baseUrl;

    public function __construct()
    {
        // In production, these should be environment variables
        $this->secretKey = 'sk_test_your_paystack_secret_key'; // Replace with actual key
        $this->publicKey = 'pk_test_your_paystack_public_key'; // Replace with actual key
        $this->baseUrl = 'https://api.paystack.co';
    }

    // Initialize payment transaction
    public function initializePayment(array $paymentData): array
    {
        try {
            $url = $this->baseUrl . '/transaction/initialize';

            $data = [
                'email' => $paymentData['email'],
                'amount' => $paymentData['amount'] * 100, // Paystack expects amount in kobo
                'reference' => $paymentData['reference'],
                'currency' => $paymentData['currency'] ?? 'NGN',
                'callback_url' => $paymentData['callback_url'] ?? null,
                'metadata' => [
                    'user_id' => $paymentData['user_id'],
                    'subscription_id' => $paymentData['subscription_id'],
                    'plan_id' => $paymentData['plan_id'],
                    'custom_fields' => [
                        [
                            'display_name' => 'Subscription Plan',
                            'variable_name' => 'plan_name',
                            'value' => $paymentData['plan_name'] ?? ''
                        ]
                    ]
                ]
            ];

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'data' => $response['data'],
                    'authorization_url' => $response['data']['authorization_url'],
                    'access_code' => $response['data']['access_code'],
                    'reference' => $response['data']['reference']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Payment initialization failed',
                'error' => $response
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Payment initialization error: ' . $e->getMessage()
            ];
        }
    }

    // Verify payment transaction
    public function verifyPayment(string $reference): array
    {
        try {
            $url = $this->baseUrl . '/transaction/verify/' . $reference;

            $response = $this->makeRequest('GET', $url);

            if ($response && $response['status'] === true) {
                $data = $response['data'];

                return [
                    'success' => true,
                    'verified' => $data['status'] === 'success',
                    'data' => $data,
                    'amount' => $data['amount'] / 100, // Convert from kobo to naira
                    'currency' => $data['currency'],
                    'reference' => $data['reference'],
                    'status' => $data['status'],
                    'gateway_response' => $data['gateway_response'],
                    'paid_at' => $data['paid_at'],
                    'customer' => $data['customer'],
                    'metadata' => $data['metadata']
                ];
            }

            return [
                'success' => false,
                'verified' => false,
                'message' => $response['message'] ?? 'Payment verification failed',
                'error' => $response
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'verified' => false,
                'message' => 'Payment verification error: ' . $e->getMessage()
            ];
        }
    }

    // Create customer
    public function createCustomer($customerData)
    {
        try {
            $url = $this->baseUrl . '/customer';

            $data = [
                'email' => $customerData['email'],
                'first_name' => $customerData['first_name'],
                'last_name' => $customerData['last_name'],
                'phone' => $customerData['phone'] ?? null,
                'metadata' => [
                    'user_id' => $customerData['user_id']
                ]
            ];

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'customer_code' => $response['data']['customer_code'],
                    'data' => $response['data']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Customer creation failed'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Customer creation error: ' . $e->getMessage()
            ];
        }
    }

    // Create subscription plan on Paystack
    public function createSubscriptionPlan($planData)
    {
        try {
            $url = $this->baseUrl . '/plan';

            $data = [
                'name' => $planData['name'],
                'amount' => $planData['amount'] * 100, // Convert to kobo
                'interval' => $planData['interval'], // monthly, yearly
                'currency' => $planData['currency'] ?? 'NGN',
                'description' => $planData['description'] ?? null,
                'send_invoices' => true,
                'send_sms' => false,
                'hosted_page' => false
            ];

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'plan_code' => $response['data']['plan_code'],
                    'data' => $response['data']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Plan creation failed'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Plan creation error: ' . $e->getMessage()
            ];
        }
    }

    // Create subscription
    public function createSubscription($subscriptionData)
    {
        try {
            $url = $this->baseUrl . '/subscription';

            $data = [
                'customer' => $subscriptionData['customer_code'],
                'plan' => $subscriptionData['plan_code'],
                'authorization' => $subscriptionData['authorization_code'] ?? null,
                'start_date' => $subscriptionData['start_date'] ?? null
            ];

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'subscription_code' => $response['data']['subscription_code'],
                    'data' => $response['data']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Subscription creation failed'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Subscription creation error: ' . $e->getMessage()
            ];
        }
    }

    // Cancel subscription
    public function cancelSubscription($subscription_code)
    {
        try {
            $url = $this->baseUrl . '/subscription/disable';

            $data = [
                'code' => $subscription_code,
                'token' => $subscription_code
            ];

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'message' => 'Subscription cancelled successfully'
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Subscription cancellation failed'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Subscription cancellation error: ' . $e->getMessage()
            ];
        }
    }

    // Process refund
    public function processRefund(string $transactionId, ?float $amount = null): array
    {
        try {
            $url = $this->baseUrl . '/refund';

            $data = [
                'transaction' => $transactionId
            ];

            if ($amount) {
                $data['amount'] = $amount * 100; // Convert to kobo
            }

            $response = $this->makeRequest('POST', $url, $data);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'data' => $response['data']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Refund processing failed'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Refund processing error: ' . $e->getMessage()
            ];
        }
    }

    // Get transaction details
    public function getTransaction($transactionId)
    {
        try {
            $url = $this->baseUrl . '/transaction/' . $transactionId;

            $response = $this->makeRequest('GET', $url);

            if ($response && $response['status'] === true) {
                return [
                    'success' => true,
                    'data' => $response['data']
                ];
            }

            return [
                'success' => false,
                'message' => $response['message'] ?? 'Transaction not found'
            ];
        } catch (Exception $e) {
            return [
                'success' => false,
                'message' => 'Transaction retrieval error: ' . $e->getMessage()
            ];
        }
    }

    // Verify webhook signature
    public function verifyWebhookSignature(string $payload, string $signature): bool
    {
        try {
            $computedSignature = hash_hmac('sha512', $payload, $this->secretKey);

            return hash_equals($signature, $computedSignature);
        } catch (Exception $e) {
            return false;
        }
    }

    // Make HTTP request to Paystack API
    private function makeRequest($method, $url, $data = null)
    {
        try {
            $ch = curl_init();

            curl_setopt_array($ch, [
                CURLOPT_URL => $url,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_CUSTOMREQUEST => $method,
                CURLOPT_HTTPHEADER => [
                    'Authorization: Bearer ' . $this->secretKey,
                    'Content-Type: application/json',
                    'Cache-Control: no-cache'
                ],
                CURLOPT_TIMEOUT => 30,
                CURLOPT_SSL_VERIFYPEER => true
            ]);

            if ($data && ($method === 'POST' || $method === 'PUT')) {
                curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
            }

            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            $error = curl_error($ch);

            curl_close($ch);

            if ($error) {
                throw new Exception('cURL Error: ' . $error);
            }

            if ($httpCode >= 400) {
                $errorResponse = json_decode($response, true);
                throw new Exception('HTTP Error ' . $httpCode . ': ' . ($errorResponse['message'] ?? 'Unknown error'));
            }

            return json_decode($response, true);
        } catch (Exception $e) {
            error_log('Paystack API Error: ' . $e->getMessage());
            throw $e;
        }
    }

    // Generate payment reference
    public function generateReference($prefix = 'SEL')
    {
        return $prefix . '_' . time() . '_' . rand(100000, 999999);
    }

    // Get public key for frontend
    public function getPublicKey()
    {
        return $this->publicKey;
    }

    // Validate webhook event
    public function validateWebhookEvent($event)
    {
        $validEvents = [
            'charge.success',
            'charge.failed',
            'subscription.create',
            'subscription.disable',
            'invoice.create',
            'invoice.payment_failed'
        ];

        return in_array($event, $validEvents);
    }

    // Get provider name (required by interface)
    public function getProviderName(): string
    {
        return 'paystack';
    }

    // Get provider configuration (required by interface)
    public function getConfig(): array
    {
        return [
            'name' => 'Paystack',
            'public_key' => $this->publicKey,
            'base_url' => $this->baseUrl,
            'supported_currencies' => ['NGN', 'USD', 'GHS', 'ZAR', 'KES'],
            'supported_methods' => ['CARD', 'BANK', 'BANK_TRANSFER', 'USSD', 'QR']
        ];
    }
}
