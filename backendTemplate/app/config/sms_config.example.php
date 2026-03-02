<?php

/**
 * SMS Configuration - EXAMPLE TEMPLATE
 */

return [
    'provider' => 'twilio',

    'providers' => [
        'twilio' => [
            'account_sid' => 'your_account_sid',
            'auth_token' => 'your_auth_token',
            'from_number' => '+1234567890'
        ],
        'africas_talking' => [
            'username' => 'your_username',
            'api_key' => 'your_api_key',
            'from' => 'YourApp'
        ]
    ],

    'otp' => [
        'length' => 6,
        'expiry_minutes' => 10,
        'max_attempts' => 3
    ],
];
