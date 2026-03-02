# Request Logging Service - Documentation

## Overview

Comprehensive logging system for tracking API requests, responses, errors, and slow queries with structured JSON logging, automatic request ID generation, and admin management endpoints.

---

## Features

✅ **Structured Logging**: JSON format for easy parsing  
✅ **Request IDs**: Unique ID for each request  
✅ **Log Levels**: DEBUG, INFO, WARN, ERROR, CRITICAL  
✅ **Automatic Logging**: Middleware logs all requests/responses  
✅ **Slow Query Detection**: Logs queries >1s  
✅ **Log Rotation**: Daily log files  
✅ **Admin Endpoints**: View, search, and manage logs  
✅ **Sensitive Data Redaction**: Auto-hides passwords, tokens  

---

## Quick Start

### 1. The Logger Service

```php
// Create logger instance
$logger = new Logger();

// Log different levels
$logger->log(Logger::INFO, 'User logged in', ['user_id' => 123]);
$logger->log(Logger::WARN, 'High memory usage', ['usage_mb' => 245]);
$logger->log(Logger::ERROR, 'Database connection failed');

// Specialized logging methods
$logger->logRequest(['user_id' => 123]);
$logger->logResponse(200, $data, 0.5);
$logger->logError('Payment failed', $exception);
$logger->logSlowQuery($sql, 2.5, $params);
```

### 2. Automatic Request Logging (Recommended)

Add the trait to your base `Controller` class:

```php
// In app/libraries/Controller.php
class Controller
{
    use RequestLoggingTrait;
    
    public function __construct()
    {
        // Initialize automatic logging
        $this->initializeRequestLogging();
    }
}
```

Now all requests are automatically logged! 🎉

---

## Log Structure

### Log Entry Format

```json
{
    "timestamp": "2026-01-24 12:00:00",
    "request_id": "req_678abc123_a1b2c3d4",
    "level": "INFO",
    "message": "API Request",
    "data": {
        "type": "request",
        "method": "POST",
        "uri": "/v1/uploads",
        "ip": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "user_id": 123
    },
    "server": {
        "hostname": "api-server-1",
        "ip": "10.0.0.5"
    }
}
```

### Log Files Organization

```
app/logs/
├── app-2026-01-24.log          # General logs (INFO, DEBUG)
├── warn-2026-01-24.log         # Warning logs
├── error-2026-01-24.log        # Error logs
├── app-2026-01-23.log          # Previous day
└── ...
```

---

## Admin Endpoints

### 1. GET `/v1/admin/logs` - View Logs

```bash
# Get today's logs
curl http://api.yourdomain.com/v1/admin/logs \
  -H "Authorization: Bearer ADMIN_JWT"

# Get logs for specific date
curl "http://api.yourdomain.com/v1/admin/logs?date=2026-01-24&level=error&limit=50" \
  -H "Authorization: Bearer ADMIN_JWT"
```

**Query Parameters:**
- `date` - YYYY-MM-DD (default: today)
- `level` - info, warn, error (default: all)
- `limit` - 1-1000 (default: 100)

**Response:**
```json
{
    "status": true,
    "data": {
        "date": "2026-01-24",
        "level": "error",
        "total": 15,
        "logs": [...]
    }
}
```

### 2. GET `/v1/admin/logs/stats` - Log Statistics

```bash
curl "http://api.yourdomain.com/v1/admin/logs/stats?date=2026-01-24" \
  -H "Authorization: Bearer ADMIN_JWT"
```

**Response:**
```json
{
    "status": true,
    "data": {
        "date": "2026-01-24",
        "total_logs": 1250,
        "by_level": {
            "info": 1100,
            "warn": 125,
            "error": 25
        },
        "file_sizes": {
            "app": 524288,
            "warn": 15360,
            "error": 8192
        }
    }
}
```

### 3. GET `/v1/admin/logs/errors` - Recent Errors

```bash
curl "http://api.yourdomain.com/v1/admin/logs/errors?limit=20" \
  -H "Authorization: Bearer ADMIN_JWT"
```

### 4. GET `/v1/admin/logs/search` - Search Logs

```bash
curl "http://api.yourdomain.com/v1/admin/logs/search?q=payment&date=2026-01-24" \
  -H "Authorization: Bearer ADMIN_JWT"
```

### 5. POST `/v1/admin/logs/cleanup` - Clean Old Logs

```bash
curl -X POST "http://api.yourdomain.com/v1/admin/logs/cleanup" \
  -H "Authorization: Bearer ADMIN_JWT" \
  -d "days_to_keep=30"
```

---

## Usage Examples

### Example 1: Log User Action

```php
class Users extends Controller
{
    public function updateProfile()
    {
        $user = $this->RouteProtection();
        $logger = $this->getLogger();
        
        // Update profile
        $this->userModel->update($user->user_id, $data);
        
        // Log the action
        $logger->log(Logger::INFO, 'Profile updated', [
            'user_id' => $user->user_id,
            'fields_changed' => array_keys($data)
        ]);
    }
}
```

### Example 2: Log Error with Exception

```php
try {
    $result = $this->paymentService->processPayment($data);
} catch (Exception $e) {
    $logger = $this->getLogger();
    $logger->logError('Payment processing failed', $e, [
        'user_id' => $user->user_id,
        'amount' => $data['amount']
    ]);
    
    return $this->sendResponse(false, 'Payment failed', [], 500);
}
```

### Example 3: Track Slow Database Queries

```php
$startTime = microtime(true);
$this->db->query($sql);
$this->db->execute();
$executionTime = microtime(true) - $startTime;

if ($executionTime > 1.0) {
    $logger = $this->getLogger();
    $logger->logSlowQuery($sql, $executionTime, $params);
}
```

### Example 4: Custom Business Logic Logging

```php
class Orders extends Controller
{
    public function createOrder()
    {
        $logger = $this->getLogger();
        
        // Log order creation attempt
        $logger->log(Logger::INFO, 'Order creation started', [
            'user_id' => $user->user_id,
            'items_count' => count($items),
            'total_amount' => $total
        ]);
        
        $orderId = $this->orderModel->create($data);
        
        if ($orderId) {
            // Log success
            $logger->log(Logger::INFO, 'Order created successfully', [
                'order_id' => $orderId,
                'user_id' => $user->user_id
            ]);
        } else {
            // Log failure
            $logger->log(Logger::ERROR, 'Order creation failed', [
                'user_id' => $user->user_id,
                'reason' => 'Database error'
            ]);
        }
    }
}
```

---

## Automatic Features

### Request/Response Logging

With `RequestLoggingTrait` enabled, every request automatically logs:

**Request Log:**
- HTTP method
- URI
- Client IP
- User agent
- User ID (if authenticated)
- Request data (sanitized)

**Response Log:**
- HTTP status code
- Execution time
- Response size

### Slow Request Detection

Requests taking >1 second are automatically flagged:

```json
{
    "level": "WARN",
    "message": "Slow request detected",
    "data": {
        "uri": "/v1/search",
        "execution_time": 1.523
    }
}
```

### Sensitive Data Redaction

Passwords, tokens, and secrets are automatically redacted:

```php
// Input
['username' => 'john', 'password' => 'secret123']

// Logged as
['username' => 'john', 'password' => '***REDACTED***']
```

---

## Log Rotation & Cleanup

### Automatic Daily Rotation

Logs are automatically rotated daily:
- `app-2026-01-24.log`
- `app-2026-01-25.log`
- etc.

### Manual Cleanup

```php
$logger = new Logger();

// Keep last 30 days
$deleted = $logger->cleanOldLogs(30);

echo "Deleted {$deleted} old log files";
```

### Scheduled Cleanup (Cron)

```bash
# Add to crontab - clean logs older than 30 days daily at 2 AM
0 2 * * * cd /var/www/api && php -r "require 'app/bootstrap.php'; (new Logger())->cleanOldLogs(30);"
```

---

## Monitoring & Analysis

### Grep for Errors

```bash
# Find all errors today
grep -i "error" app/logs/error-$(date +%Y-%m-%d).log

# Find specific user's actions
grep "user_id.*123" app/logs/app-$(date +%Y-%m-%d).log
```

### Count Requests by Endpoint

```bash
grep "API Request" app/logs/app-2026-01-24.log | \
  jq -r '.data.uri' | \
  sort | uniq -c | sort -rn | head -10
```

### Find Slow Queries

```bash
grep "slow_query" app/logs/warn-2026-01-24.log | \
  jq '.data.execution_time_ms' | \
  sort -rn | head -10
```

---

## Configuration

### Change Log Directory

```php
// In Logger.php constructor
$this->logDir = '/var/log/divine-api';
```

### Adjust Slow Request Threshold

```php
// In RequestLoggingTrait.php
if ($executionTime > 2.0) {  // Change to 2 seconds
    $this->logger->log(Logger::WARN, 'Slow request detected', ...);
}
```

### Add Custom Log Level

```php
// In Logger.php
const INFO = 'INFO';
const CUSTOM = 'CUSTOM';  // Add new level
```

---

## Best Practices

### 1. Use Appropriate Log Levels

- **DEBUG**: Development/debugging info
- **INFO**: Normal operations (user login, order created)
- **WARN**: Concerning but not critical (slow query, high memory)
- **ERROR**: Errors that need attention (payment failed, API error)
- **CRITICAL**: System failures (database down, disk full)

### 2. Include Context

```php
// ❌ Bad
$logger->log(Logger::ERROR, 'Payment failed');

// ✅ Good
$logger->log(Logger::ERROR, 'Payment failed', [
    'user_id' => 123,
    'amount' => 5000,
    'provider' => 'paystack',
    'error_code' => 'insufficient_funds'
]);
```

### 3. Don't Log Sensitive Data

```php
// ❌ Bad
$logger->log(Logger::INFO, 'User registered', [
    'email' => 'user@example.com',
    'password' => '12345'  // Never log passwords!
]);

// ✅ Good  
$logger->log(Logger::INFO, 'User registered', [
    'email' => 'user@example.com'
    // Password excluded
]);
```

### 4. Regular Cleanup

Set up automated log cleanup to prevent disk space issues.

---

## Troubleshooting

### Logs Not Being Created

**Check:**
1. Directory exists: `app/logs/`
2. Directory is writable: `chmod 755 app/logs`
3. Logger is initialized: `$logger = new Logger();`

### Permission Denied

```bash
# Fix permissions
chmod 755 app/logs
chown www-data:www-data app/logs
```

### Disk Space Issues

```bash
# Check log sizes
du -sh app/logs/*

# Clean old logs
php -r "require 'app/bootstrap.php'; (new Logger())->cleanOldLogs(7);"
```

---

## Summary

**Request Logging Service provides:**
- ✅ Structured JSON logging
- ✅ Automatic request/response tracking
- ✅ Error and exception logging
- ✅ Slow query detection
- ✅ Admin viewing endpoints
- ✅ Sensitive data protection
- ✅ Daily log rotation
- ✅ Search capabilities

**Essential for:**
- Debugging production issues
- Monitoring API health
- Tracking user actions
- Performance optimization
- Security audits

Your API is now fully observable! 🚀
