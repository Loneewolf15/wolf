<?php

/**
 * Payment Gateway Factory
 * 
 * Creates payment gateway instances based on provider name.
 * Provides a single entry point for gateway creation.
 */
class GatewayFactory
{
    /**
     * Create a payment gateway instance
     * 
     * @param string $provider Provider name ('paystack', 'monnify')
     * @return PaymentGatewayInterface
     * @throws Exception If provider is not supported
     */
    public static function create(string $provider): PaymentGatewayInterface
    {
        $provider = strtolower(trim($provider));

        switch ($provider) {
            case 'paystack':
                return new PaystackService();

            case 'monnify':
                return new MonnifyService();

            default:
                throw new Exception("Unsupported payment provider: {$provider}. Supported providers: " . implode(', ', self::getSupportedProviders()));
        }
    }

    /**
     * Get list of supported payment providers
     * 
     * @return array List of provider names
     */
    public static function getSupportedProviders(): array
    {
        return ['paystack', 'monnify'];
    }

    /**
     * Check if a provider is supported
     * 
     * @param string $provider Provider name
     * @return bool
     */
    public static function isSupported(string $provider): bool
    {
        return in_array(strtolower($provider), self::getSupportedProviders());
    }

    /**
     * Get default payment provider from config
     * 
     * @return string Default provider name
     */
    public static function getDefaultProvider(): string
    {
        $configPath = APPROOT . '/config/payment_config.php';

        if (file_exists($configPath)) {
            $config = require $configPath;
            return $config['default_provider'] ?? 'paystack';
        }

        return 'paystack';
    }
}
