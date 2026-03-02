<?php

/**
 * File Upload Configuration for Divine API
 * 
 * CUSTOMIZE ALL UPLOAD SETTINGS HERE
 * 
 * 1. Enable/disable file types
 * 2. Set max file sizes per type  
 * 3. Choose storage mode (local/cloud/hybrid)
 * 4. Enable/disable image processing (0/1)
 * 5. Configure security settings
 */

return [
    // ==========================================
    // STORAGE SETTINGS
    // ==========================================

    // Storage mode: 'local', 'cloud', or 'hybrid'
    'storage_mode' => 'local',  // CHANGE: 'cloud' for production

    // Cloud provider (if using cloud/hybrid): 'cloudinary', 's3', 'azure'
    'cloud_provider' => 'cloudinary',

    // Cloud credentials
    'cloud_config' => [
        'cloudinary' => [
            'cloud_name' => 'your_cloud_name',
            'api_key' => 'your_api_key',
            'api_secret' => 'your_api_secret'
        ],
        's3' => [
            'bucket' => 'your_bucket_name',
            'region' => 'us-east-1',
            'access_key' => 'your_access_key',
            'secret_key' => 'your_secret_key'
        ]
    ],

    // ==========================================
    // FILE TYPES & SIZES (USER-DEFINED)
    // ==========================================

    'allowed_types' => [
        // Images
        'images' => [
            'enabled' => true,  // CHANGE: false to disable
            'extensions' => ['jpg', 'jpeg', 'png', 'gif', 'webp'],
            'mime_types' => ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
            'max_size' => 5 * 1024 * 1024,  // CHANGE: 5MB default
        ],

        // Videos
        'videos' => [
            'enabled' => true,  // CHANGE: false to disable
            'extensions' => ['mp4', 'mov', 'avi', 'webm'],
            'mime_types' => ['video/mp4', 'video/quicktime', 'video/x-msvideo', 'video/webm'],
            'max_size' => 50 * 1024 * 1024,  // CHANGE: 50MB default
        ],

        // Documents
        'documents' => [
            'enabled' => true,  // CHANGE: false to disable
            'extensions' => ['pdf', 'doc', 'docx', 'txt', 'xlsx', 'csv', 'ppt', 'pptx'],
            'mime_types' => [
                'application/pdf',
                'application/msword',
                'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
                'text/plain',
                'application/vnd.ms-excel',
                'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
                'application/csv',
                'text/csv'
            ],
            'max_size' => 10 * 1024 * 1024,  // CHANGE: 10MB default
        ],

        // Audio files
        'audio' => [
            'enabled' => false,  // CHANGE: true to enable
            'extensions' => ['mp3', 'wav', 'ogg', 'm4a'],
            'mime_types' => ['audio/mpeg', 'audio/wav', 'audio/ogg', 'audio/mp4'],
            'max_size' => 10 * 1024 * 1024,  // CHANGE: 10MB
        ]
    ],

    // ==========================================
    // IMAGE PROCESSING (OPTIONAL)
    // ==========================================

    'image_processing' => [
        'enabled' => 1,  // CHANGE: 0 to disable all processing

        // Image resizing
        'resize' => [
            'enabled' => 1,  // CHANGE: 0 to disable resizing
            'sizes' => [
                'thumbnail' => ['width' => 150, 'height' => 150],
                'small' => ['width' => 400, 'height' => 400],
                'medium' => ['width' => 800, 'height' => 800]
            ]
        ],

        // Image compression
        'compression' => [
            'enabled' => 1,  // CHANGE: 0 to disable compression
            'quality' => 85  // CHANGE: 0-100 (85 recommended)
        ],

        // Format conversion (HEIC to JPG, etc.)
        'format_conversion' => [
            'enabled' => 0,  // CHANGE: 1 to enable
            'target_format' => 'jpg'
        ]
    ],

    // ==========================================
    // USER QUOTAS
    // ==========================================

    'quotas' => [
        'max_files_per_user' => 100,  // CHANGE: Maximum files per user
        'max_storage_per_user' => 100 * 1024 * 1024,  // CHANGE: 100MB total storage
        'max_uploads_per_day' => 50,  // CHANGE: Daily upload limit
    ],

    // ==========================================
    // UPLOAD PATHS
    // ==========================================

    'upload_paths' => [
        'local_base' => 'public/assets/uploads',  // CHANGE: Base upload directory
        'structure' => 'user_id',  // Options: 'user_id', 'date', 'type', 'flat'
    ],

    // ==========================================
    // SECURITY SETTINGS
    // ==========================================

    'security' => [
        'check_magic_bytes' => true,  // Verify actual file type (recommended)
        'sanitize_filename' => true,   // Remove dangerous characters
        'generate_unique_names' => true,  // Prevent overwrites
        'block_executable' => true,  // Block .exe, .sh, .bat files
    ],

    // ==========================================
    // RATE LIMITING
    // ==========================================

    'rate_limits' => [
        'uploads_per_hour' => 20,  // CHANGE: Max uploads per hour
        'uploads_per_minute' => 5,  // CHANGE: Max uploads per minute
    ]
];
