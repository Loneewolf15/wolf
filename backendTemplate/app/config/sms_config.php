<?php

/**
 * SMS Configuration
 * 
 * Configure SMS providers and templates
 */

return [
    // Active provider: 'twilio' or 'africas_talking'
    'provider' => 'twilio',  // CHANGE: Set your preferred provider

    // Provider credentials
    'providers' => [
        'twilio' => [
            'account_sid' => 'your_twilio_account_sid',
            'auth_token' => 'your_twilio_auth_token',
            'from_number' => '+1234567890'  // Your Twilio number
        ],

        'africas_talking' => [
            'username' => 'your_at_username',
            'api_key' => 'your_at_api_key',
            'from' => 'YourAppName'  // Sender ID (11 chars max)
        ]
    ],

    // OTP settings
    'otp' => [
        'length' => 6,  // CHANGE: OTP code length
        'expiry_minutes' => 10,  // CHANGE: OTP validity period
        'max_attempts' => 3  // CHANGE: Max verification attempts
    ],

    // Rate limiting
    'rate_limits' => [
        'per_user_per_hour' => 5,  // CHANGE: Max SMS per user per hour
        'per_user_per_day' => 20   // CHANGE: Max SMS per user per day
    ],

    // SMS templates
    'templates' => [
        'otp' => 'Your {app_name} verification code is: {code}. Valid for {minutes} minutes.',
        'password_reset' => 'Your {app_name} password reset code is: {code}',
        'transaction' => 'Transaction alert: {amount} {currency} {action}. Ref: {reference}',
        'welcome' => 'Welcome to {app_name}! Your account has been created successfully.',
    ],

    // Cost tracking
    'cost_per_sms' => 0.05,  // CHANGE: Cost per SMS in your currency
    'monthly_budget' => 1000,  // CHANGE: Monthly SMS budget
];
