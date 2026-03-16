# File Upload Service - Documentation & Use Cases

## Overview

The Divine API File Upload Service provides a flexible, configurable, and secure way to handle file uploads with support for multiple file types, automatic image processing, and cloud storage integration.

---

## Features

✅ **Multiple File Types**: Images, videos, documents, audio  
✅ **Configurable**: All settings in one config file  
✅ **Image Processing**: Automatic resize, compress, thumbnails  
✅ **Storage Options**: Local, cloud (Cloudinary/S3), or hybrid  
✅ **Security**: File validation, magic bytes checking, quotas  
✅ **Rate Limiting**: Prevent abuse  
✅ **RESTful API**: Standard endpoints for upload, list, delete  

---

## Quick Start

### 1. Configure Upload Settings

Edit `app/config/upload_config.php`:

```php
return [
    'storage_mode' => 'local',  // 'local', 'cloud', or 'hybrid'
    
    'allowed_types' => [
        'images' => [
            'enabled' => true,
            'max_size' => 5 * 1024 * 1024,  // 5MB
        ],
        'videos' => [
            'enabled' => true,
            'max_size' => 50 * 1024 * 1024,  // 50MB
        ],
    ],
    
    'image_processing' => [
        'enabled' => 1,  // Set to 0 to disable
        'resize' => ['enabled' => 1],
        'compression' => ['quality' => 85]
    ]
];
```

### 2. Apply Database Migration

```bash
mysql -u root -p your_database < database/uploads_table.sql
```

### 3. Test Upload

```bash
curl -X POST http://localhost/v1/uploads \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@image.jpg"
```

---

## API Endpoints

### POST `/v1/uploads` - Upload File(s)

**Upload single file:**

```bash
curl -X POST http://api.yourdomain.com/v1/uploads \
  -H "Authorization: Bearer YOUR_JWT" \
  -F "file=@photo.jpg" \
  -F "type=images"
```

**Upload multiple files:**

```bash
curl -X POST http://api.yourdomain.com/v1/uploads \
  -H "Authorization: Bearer YOUR_JWT" \
  -F "files[]=@photo1.jpg" \
  -F "files[]=@photo2.png"
```

**Response:**

```json
{
    "status": true,
    "message": "File uploaded successfully",
    "data": {
        "id": 123,
        "url": "http://yourdomain.com/assets/uploads/users/1/photo_1706150400.jpg",
        "file_path": "users/1/photo_1706150400.jpg",
        "file_type": "image",
        "size": 245678,
        "variants": {
            "thumbnail": "http://yourdomain.com/.../photo_1706150400_thumbnail.jpg",
            "small": "http://yourdomain.com/.../photo_1706150400_small.jpg"
        }
    }
}
```

### GET `/v1/uploads` - List User's Uploads

```bash
curl -X GET "http://api.yourdomain.com/v1/uploads?page=1&limit=20&type=images" \
  -H "Authorization: Bearer YOUR_JWT"
```

**Query Parameters:**
- `page` - Page number (default: 1)
- `limit` - Items per page (max: 100, default: 20)
- `type` - Filter by type: images, videos, documents

**Response:**

```json
{
    "status": true,
    "message": "Uploads retrieved",
    "data": {
        "uploads": [...],
        "pagination": {
            "page": 1,
            "limit": 20,
            "total": 45,
            "pages": 3
        }
    }
}
```

### GET `/v1/uploads/{id}` - Get File Details

```bash
curl -X GET http://api.yourdomain.com/v1/uploads/123 \
  -H "Authorization: Bearer YOUR_JWT"
```

### DELETE `/v1/uploads/{id}` - Delete File

```bash
curl -X DELETE http://api.yourdomain.com/v1/uploads/123 \
  -H "Authorization: Bearer YOUR_JWT"
```

### GET `/v1/uploads/quota` - Check Storage Quota

```bash
curl -X GET http://api.yourdomain.com/v1/uploads/quota \
  -H "Authorization: Bearer YOUR_JWT"
```

**Response:**

```json
{
    "status": true,
    "data": {
        "usage": {
            "file_count": 25,
            "total_bytes": 15728640,
            "total_mb": 15,
            "image_count": 20,
            "video_count": 3,
            "document_count": 2
        },
        "limits": {
            "max_files": 100,
            "max_storage_mb": 100
        },
        "percentage_used": 15
    }
}
```

---

## Integration Examples

### Use Case 1: User Profile Picture

```php
class Users extends Controller
{
    private $uploadService;
    
    public function __construct()
    {
        $this->uploadService = new FileUploadService();
    }
    
    public function updateProfilePicture()
    {
        $user = $this->RouteProtection();
        
        if (empty($_FILES['profile_picture'])) {
            return $this->sendResponse(false, 'No file uploaded', [], 400);
        }
        
        // Upload image
        $result = $this->uploadService->upload(
            $_FILES['profile_picture'],
            $user->user_id,
            ['type' => 'images']
        );
        
        if (!$result['success']) {
            return $this->sendResponse(false, $result['error'], [], 400);
        }
        
        // Update user profile with full URL
        $this->userModel->update($user->user_id, [
            'profile_picture' => $result['data']['file_url'],
            'profile_thumbnail' => $result['data']['variants']['thumbnail'] ?? null
        ]);
        
        return $this->sendResponse(true, 'Profile picture updated', [
            'url' => $result['data']['file_url'],
            'thumbnail' => $result['data']['variants']['thumbnail']
        ]);
    }
}
```

### Use Case 2: Product Listings with Multiple Images

```php
class Listings extends Controller
{
    public function createListing()
    {
        $user = $this->RouteProtection();
        
        // Create listing
        $listingId = $this->listingModel->create([
            'title' => $this->getData('title'),
            'price' => $this->getData('price'),
            'user_id' => $user->user_id
        ]);
        
        // Upload product images
        $imageUrls = [];
        
        if (!empty($_FILES['images'])) {
            $results = $this->uploadService->uploadMultiple(
                $_FILES['images'],
                $user->user_id,
                ['type' => 'images']
            );
            
            foreach ($results as $result) {
                if ($result['success']) {
                    $imageUrls[] = $result['data']['file_url'];
                }
            }
        }
        
        // Save image URLs to listing
        $this->listingModel->update($listingId, [
            'images' => json_encode($imageUrls)
        ]);
        
        return $this->sendResponse(true, 'Listing created', [
            'id' => $listingId,
            'images' => $imageUrls
        ]);
    }
}
```

### Use Case 3: Document Upload (KYC/Verification)

```php
class Verification extends Controller
{
    public function uploadDocument()
    {
        $user = $this->RouteProtection();
        
        $documentType = $this->getData('document_type');  // 'id_card', 'passport', etc.
        
        $result = $this->uploadService->upload(
            $_FILES['document'],
            $user->user_id,
            ['type' => 'documents']
        );
        
        if ($result['success']) {
            // Save document reference
            $this->verificationModel->create([
                'user_id' => $user->user_id,
                'document_type' => $documentType,
                'document_url' => $result['data']['file_url'],
                'status' => 'pending'
            ]);
            
            return $this->sendResponse(true, 'Document uploaded for verification');
        }
        
        return $this->sendResponse(false, $result['error'], [], 400);
    }
}
```

### Use Case 4: Chat/Message Attachments

```php
class Messages extends Controller
{
    public function sendMessage()
    {
        $user = $this->RouteProtection();
        
        $messageData = [
            'sender_id' => $user->user_id,
            'receiver_id' => $this->getData('receiver_id'),
            'message' => $this->getData('message')
        ];
        
        // Check for attachments
        if (!empty($_FILES['attachment'])) {
            $result = $this->uploadService->upload(
                $_FILES['attachment'],
                $user->user_id,
                ['type' => $this->getData('file_type')]  // User specifies type
            );
            
            if ($result['success']) {
                $messageData['attachment_url'] = $result['data']['file_url'];
                $messageData['attachment_type'] = $result['data']['file_type'];
            }
        }
        
        $messageId = $this->messageModel->create($messageData);
        
        return $this->sendResponse(true, 'Message sent', ['id' => $messageId]);
    }
}
```

---

## Configuration Guide

### File Type Control

**Enable/Disable File Types:**

```php
'allowed_types' => [
    'images' => ['enabled' => true],   // Allow images
    'videos' => ['enabled' => false],  // Disable videos
    'documents' => ['enabled' => true],
    'audio' => ['enabled' => false]
]
```

### Size Limits Per Type

```php
'allowed_types' => [
    'images' => [
        'max_size' => 5 * 1024 * 1024,   // 5MB
    ],
    'videos' => [
        'max_size' => 100 * 1024 * 1024,  // 100MB
    ]
]
```

### Image Processing

**Enable resizing and compression:**

```php
'image_processing' => [
    'enabled' => 1,  // Master switch
    'resize' => [
        'enabled' => 1,
        'sizes' => [
            'thumbnail' => ['width' => 150, 'height' => 150],
            'small' => ['width' => 400, 'height' => 400],
            'medium' => ['width' => 800, 'height' => 800]
        ]
    ],
    'compression' => [
        'enabled' => 1,
        'quality' => 85  // 0-100
    ]
]
```

**Disable all processing:**

```php
'image_processing' => ['enabled' => 0]
```

### User Quotas

```php
'quotas' => [
    'max_files_per_user' => 100,  // Max 100 files
    'max_storage_per_user' => 100 * 1024 * 1024,  // 100MB total
    'max_uploads_per_day' => 50
]
```

### Storage Modes

**Local storage (default):**

```php
'storage_mode' => 'local',
'upload_paths' => [
    'local_base' => 'public/assets/uploads',
    'structure' => 'user_id'  // Organize by user ID
]
```

**Cloud storage (future):**

```php
'storage_mode' => 'cloud',
'cloud_provider' => 'cloudinary',
'cloud_config' => [
    'cloudinary' => [
        'cloud_name' => 'your_cloud_name',
        'api_key' => 'your_key',
        'api_secret' => 'your_secret'
    ]
]
```

---

## Security Features

### 1. File Validation

- Extension checking (jpg, png, pdf, etc.)
- MIME type validation
- File size limits
- Magic bytes verification (actual file content)

### 2. User Quotas

- Maximum files per user
- Maximum storage per user
- Daily upload limits

### 3. Rate Limiting

- 20 uploads per hour (configurable)
- 5 uploads per minute

### 4. Access Control

- JWT authentication required
- Users can only delete their own files
- Ownership verification on all operations

---

## Troubleshooting

### Error: "File too large"

**Solution:** Adjust max size in config:

```php
'allowed_types' => [
    'images' => ['max_size' => 10 * 1024 * 1024]  // Increase to 10MB
]
```

### Error: "File type not allowed"

**Solution:** Enable the file type or add extension:

```php
'allowed_types' => [
    'images' => [
        'enabled' => true,
        'extensions' => ['jpg', 'jpeg', 'png', 'webp']  // Add webp
    ]
]
```

### Error: "Storage quota exceeded"

**Solution:** Increase user quota:

```php
'quotas' => [
    'max_storage_per_user' => 200 * 1024 * 1024  // Increase to 200MB
]
```

### Image Processing Not Working

**Check:**
1. GD library installed: `php -m | grep gd`
2. Processing enabled in config: `'enabled' => 1`
3. File permissions on upload directory: `chmod 755 public/assets/uploads`

---

## Production Checklist

- [ ] Set appropriate file size limits
- [ ] Configure user quotas
- [ ] Enable magic bytes checking
- [ ] Set up proper file permissions (755 for directories, 644 for files)
- [ ] Configure rate limiting
- [ ] Enable image compression
- [ ] Set up backup strategy for uploads
- [ ] Configure cloud storage (optional)
- [ ] Test upload/delete workflows
- [ ] Monitor disk space usage

---

## Database Schema

```sql
CREATE TABLE uploads (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    filename VARCHAR(255),
    original_name VARCHAR(255),
    file_path VARCHAR(500),      -- Relative path
    file_url VARCHAR(500),        -- Full URL with domain
    file_type ENUM('image', 'video', 'document', 'audio'),
    mime_type VARCHAR(100),
    file_size INT UNSIGNED,
    storage_type ENUM('local', 'cloud'),
    variants JSON,                -- Thumbnail URLs
    created_at TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
```

---

## Advanced Usage

### Disable Processing for Specific Upload

```php
$result = $this->uploadService->upload(
    $_FILES['file'],
    $userId,
    ['resize' => 0]  // Skip resizing even if enabled in config
);
```

### Force File Type

```php
$result = $this->uploadService->upload(
    $_FILES['file'],
    $userId,
    ['type' => 'documents']  // Force validation as document
);
```

### Retrieve User's Total Storage

```php
$uploadModel = $this->model('Upload');
$quota = $uploadModel->getUserQuota($userId);

echo "Used: " . $quota['total_mb'] . "MB";
echo "Files: " . $quota['file_count'];
```

---

## Support

For issues or questions:
1. Check configuration in `app/config/upload_config.php`
2. Review error logs
3. Verify file permissions
4. Test with small files first

---

## Summary

**The File Upload Service provides:**
- ✅ Flexible configuration
- ✅ Multiple file type support
- ✅ Automatic image processing
- ✅ Security and validation
- ✅ Easy integration
- ✅ Production-ready features

**Perfect for:** Profile pictures, product images, document verification, chat attachments, media galleries, and more!
