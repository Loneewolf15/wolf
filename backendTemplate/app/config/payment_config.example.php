<?php

/**
 * Payment Gateway Configuration - EXAMPLE TEMPLATE
 * 
 * Copy this file to payment_config.php and update with your credentials.
 * This template is safe to commit to Git (no real credentials).
 */

return [
    // Default payment provider
    'default_provider' => 'paystack',

    // Paystack Configuration
    'paystack' => [
        // Get from: https://dashboard.paystack.com/#/settings/developer
        'secret_key' => 'sk_test_your_paystack_secret_key',
        'public_key' => 'pk_test_your_paystack_public_key',
        'base_url' => 'https://api.paystack.co',
        'callback_url' => URLROOT . 'api/v1/payments/callback/paystack'
    ],

    // Monnify Configuration
    'monnify' => [
        // Get from: Monnify Dashboard
        'api_key' => 'MK_TEST_YOUR_API_KEY',
        'secret_key' => 'YOUR_SECRET_KEY',
        'contract_code' => 'YOUR_CONTRACT_CODE',
        'base_url' => 'https://sandbox.monnify.com',  // Change to https://api.monnify.com for production
        'callback_url' => URLROOT . 'api/v1/payments/callback/monnify'
    ],

    // Supported currencies
    'supported_currencies' => ['NGN', 'USD', 'GHS', 'ZAR', 'KES'],

    // Transaction limits (in NGN)
    'limits' => [
        'min_amount' => 100,
        'max_amount' => 5000000,
        'daily_limit' => 10000000
    ],

    // Rate limiting configuration
    'rate_limits' => [
        'initialize' => [
            'requests' => 60,
            'period' => 3600,
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
