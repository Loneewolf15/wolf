<?php

/**
 * File Upload Configuration - EXAMPLE TEMPLATE
 * 
 * Copy this to upload_config.php and customize
 */

return [
    'storage_mode' => 'local',
    'cloud_provider' => 'cloudinary',

    'allowed_types' => [
        'images' => [
            'enabled' => true,
            'extensions' => ['jpg', 'jpeg', 'png', 'gif', 'webp'],
            'mime_types' => ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
            'max_size' => 5 * 1024 * 1024,  // 5MB
        ],
        'videos' => [
            'enabled' => true,
            'extensions' => ['mp4', 'mov'],
            'mime_types' => ['video/mp4', 'video/quicktime'],
            'max_size' => 50 * 1024 * 1024,  // 50MB
        ],
        'documents' => [
            'enabled' => true,
            'extensions' => ['pdf', 'docx', 'txt'],
            'mime_types' => ['application/pdf', 'text/plain'],
            'max_size' => 10 * 1024 * 1024,  // 10MB
        ],
    ],

    'image_processing' => [
        'enabled' => 1,
        'resize' => ['enabled' => 1],
        'compression' => ['enabled' => 1, 'quality' => 85]
    ],

    'quotas' => [
        'max_files_per_user' => 100,
        'max_storage_per_user' => 100 * 1024 * 1024,
    ],

    'upload_paths' => [
        'local_base' => 'public/assets/uploads',
        'structure' => 'user_id',
    ],
];
