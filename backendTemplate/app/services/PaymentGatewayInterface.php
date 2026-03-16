<?php

/**
 * Payment Gateway Interface
 * 
 * Defines the contract that all payment gateway services must implement.
 * This ensures consistency across different payment providers.
 */
interface PaymentGatewayInterface
{
    /**
     * Initialize a payment transaction
     * 
     * @param array $data Payment data including amount, email, reference, etc.
     * @return array Standardized response with success, data, and message
     */
    public function initializePayment(array $data): array;

    /**
     * Verify a payment transaction
     * 
     * @param string $reference Unique transaction reference
     * @return array Response with verification status and transaction details
     */
    public function verifyPayment(string $reference): array;

    /**
     * Process a refund for a transaction
     * 
     * @param string $transactionId Transaction identifier
     * @param float|null $amount Amount to refund (null for full refund)
     * @return array Response with refund status
     */
    public function processRefund(string $transactionId, ?float $amount = null): array;

    /**
     * Verify webhook signature for security
     * 
     * @param string $payload Raw webhook payload
     * @param string $signature Signature from webhook header
     * @return bool True if signature is valid
     */
    public function verifyWebhookSignature(string $payload, string $signature): bool;

    /**
     * Get the payment provider name
     * 
     * @return string Provider name (e.g., 'paystack', 'monnify')
     */
    public function getProviderName(): string;

    /**
     * Get provider configuration
     * 
     * @return array Provider configuration details
     */
    public function getConfig(): array;
}
