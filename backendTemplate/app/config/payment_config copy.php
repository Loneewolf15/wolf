<?php

/**
 * Payment Gateway Configuration
 * 
 * Configuration for all supported payment providers
 */

return [
    // Default payment provider
    'default_provider' => getenv('DEFAULT_PAYMENT_PROVIDER') ?: 'paystack',

    // Paystack Configuration
    'paystack' => [
        'secret_key' => getenv('PAYSTACK_SECRET_KEY') ?: 'sk_test_your_paystack_secret_key',
        'public_key' => getenv('PAYSTACK_PUBLIC_KEY') ?: 'pk_test_your_paystack_public_key',
        'base_url' => 'https://api.paystack.co',
        'callback_url' => URLROOT . 'api/v1/payments/callback/paystack'
    ],

    // Monnify Configuration
    'monnify' => [
        'api_key' => getenv('MONNIFY_API_KEY') ?: 'MK_TEST_YOUR_API_KEY',
        'secret_key' => getenv('MONNIFY_SECRET_KEY') ?: 'YOUR_SECRET_KEY',
        'contract_code' => getenv('MONNIFY_CONTRACT_CODE') ?: 'YOUR_CONTRACT_CODE',
        'base_url' => getenv('MONNIFY_ENV') === 'production'
            ? 'https://api.monnify.com'
            : 'https://sandbox.monnify.com',
        'callback_url' => URLROOT . 'api/v1/payments/callback/monnify'
    ],

    // Supported currencies
    'supported_currencies' => ['NGN', 'USD', 'GHS', 'ZAR', 'KES'],

    // Transaction limits (in NGN)
    'limits' => [
        'min_amount' => 100,          // Minimum 100 Naira
        'max_amount' => 5000000,      // Maximum 5 Million Naira
        'daily_limit' => 10000000     // Daily limit 10 Million Naira
    ],

    // Rate limiting configuration
    'rate_limits' => [
        'initialize' => [
            'requests' => 60,
            'period' => 3600,          // 1 hour
            'identifier' => 'user'
        ],
        'verify' => [
            'requests' => 300,
            'period' => 3600,
            'identifier' => 'user'
        ],
        'webhook' => [
            'requests' => 1000,
            'period' => 3600,
            'identifier' => 'ip'
        ],
        'refund' => [
            'requests' => 20,
            'period' => 3600,
            'identifier' => 'user'
        ]
    ]
];
