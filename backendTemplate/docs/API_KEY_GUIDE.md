# API Key Management - Quick Guide

## Overview

Secure API key system for third-party integrations with scopes, rate limiting, and usage tracking.

---

## Quick Start

### 1. Apply Database Migration

```bash
mysql -u root -p your_database < database/api_keys_tables.sql
```

### 2. Create API Key

```bash
POST /v1/api-keys
{
    "name": "My Integration",
    "scopes": ["users:read", "payments:read"],
    "description": "Third-party integration",
    "rate_limit_minute": 60
}
```

**Response** (save the api_key!):
```json
{
    "status": true,
    "data": {
        "api_key": "pk_live_xxxxxxxxxxxxxxxxxxxx",
        "id": 1,
        "prefix": "pk_live_",
        "name": "My Integration",
        "scopes": ["users:read", "payments:read"],
        "warning": "Store this API key securely. It will not be shown again!"
    }
}
```

### 3. Use API Key

Add to request headers:
```bash
curl "http://api.yourdomain.com/v1/users" \
  -H "X-API-Key: pk_live_xxxxxxxxxxxxxxxxxxxx"
```

---

## Protecting Endpoints with API Keys

### Method 1: Using APIKeyAuthTrait

```php
class MyPublicAPI extends Controller
{
    use APIKeyAuthTrait;
    
    public function getData()
    {
        // Require valid API key with specific scope
        $apiKey = $this->APIKeyProtection('users:read');
        
        // Your logic here
        return $this->sendResponse(true, 'Data retrieved', $data);
    }
}
```

### Method 2: Mixed Authentication (JWT or API Key)

```php
public function getData()
{
    // Try JWT first
    try {
        $user = $this->RouteProtection();
        $userId = $user->user_id;
    } catch (Exception $e) {
        // Fall back to API key
        $apiKey = $this->APIKeyProtection('users:read');
        $userId = $apiKey->user_id;
    }
    
    return $this->sendResponse(true, 'Data', $data);
}
```

---

## Available Scopes

- `users:read` - Read user data
- `users:write` - Create/update users
- `payments:read` - Read payments
- `payments:write` - Create payments
- `uploads:read` - Read uploads
- `uploads:write` - Upload files
- `search:read` - Use search
- `admin:read` - Admin read access
- `admin:write` - Admin write access

**Wildcards:**
- `users:*` - All user permissions
- `*` - All permissions (use with caution!)

---

## API Endpoints

### Create API Key
```bash
POST /v1/api-keys
```

### List API Keys
```bash
GET /v1/api-keys
```

### Get Available Scopes
```bash
GET /v1/api-keys/scopes
```

### Revoke API Key
```bash
DELETE /v1/api-keys/{id}
```

### Rotate API Key
```bash
POST /v1/api-keys/{id}/rotate
```

### Get Usage Stats
```bash
GET /v1/api-keys/{id}/stats
```

---

## Rate Limiting

Each API key has configurable rate limits:
- Per minute (default: 60 requests)
- Per hour (default: 1000 requests)
- Per day (default: 10000 requests)

Exceed limits returns `429 Too Many Requests`.

---

## Security Best Practices

1. **Store keys securely** - Never commit to Git
2. **Use scopes** - Grant minimal permissions
3. **Rotate keys** - Periodically rotate for security
4. **Monitor usage** - Check stats regularly
5. **Set expiry** - Use temporary keys when possible

---

## Example Integration

```php
// Third-party app using your API
$apiKey = 'pk_live_xxxxxxxxxxxxxxxxxxxx';

$ch = curl_init('https://api.yourdomain.com/v1/users');
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'X-API-Key: ' . $apiKey,
    'Content-Type: application/json'
]);
$response = curl_exec($ch);
```

Your API is now integration-ready! 🔑
