# Audit Trail System - Documentation

## Overview

Complete audit trail system for tracking all user actions, entity changes, and storing before/after values for security, compliance, and debugging.

---

## Features

✅ **Action Tracking**: Create, update, delete, login, logout  
✅ **Change Detection**: Automatic before/after comparison  
✅ **Entity History**: Complete change history for any entity  
✅ **User Activity**: Track everything a user does  
✅ **Request Correlation**: Links to Logger request_id  
✅ **Admin Endpoints**: View, filter, and analyze audit logs  

---

## Quick Start

### 1. Apply Database Migration

```bash
mysql -u root -p your_database < database/audit_logs_table.sql
```

### 2. Use Audit Trail Service

```php
// In your controller
$audit = new AuditTrailService();
$user = $this->RouteProtection();

// Log a create action
$listingId = $this->listingModel->create($data);
$audit->logCreate($user, 'listing', $listingId, $data, 'New listing created');

// Log an update action
$before = $this->listingModel->findById($id);
$this->listingModel->update($id, $newData);
$after = $this->listingModel->findById($id);
$audit->logUpdate($user, 'listing', $id, (array)$before, (array)$after);

// Log a delete action
$listing = $this->listingModel->findById($id);
$this->listingModel->delete($id);
$audit->logDelete($user, 'listing', $id, (array)$listing);
```

---

## Usage Examples

### Example 1: Track User Profile Updates

```php
class Users extends Controller
{
    public function updateProfile()
    {
        $user = $this->RouteProtection();
        $audit = new AuditTrailService();
        
        // Get current state
        $before = $this->userModel->findById($user->user_id);
        
        // Update profile
        $newData = [
            'name' => $this->getData('name'),
            'bio' => $this->getData('bio')
        ];
        $this->userModel->update($user->user_id, $newData);
        
        // Get updated state
        $after = $this->userModel->findById($user->user_id);
        
        // Log the change
        $audit->logUpdate(
            $user,
            'user',
            $user->user_id,
            (array)$before,
            (array)$after,
            'Profile updated'
        );
        
        return $this->sendResponse(true, 'Profile updated');
    }
}
```

### Example 2: Track Payment Transactions

```php
class Payments extends Controller
{
    public function initializePayment()
    {
        $user = $this->RouteProtection();
        $audit = new AuditTrailService();
        
        // Create payment
        $paymentId = $this->paymentModel->create([
            'user_id' => $user->user_id,
            'amount' => $amount,
            'reference' => $reference
        ]);
        
        // Log payment creation
        $audit->logCreate(
            $user,
            'payment',
            $paymentId,
            [
                'amount' => $amount,
                'reference' => $reference,
                'provider' => 'paystack'
            ],
            "Payment of ₦{$amount} initialized"
        );
    }
}
```

### Example 3: Track Login Attempts

```php
class Auth extends Controller
{
    public function login()
    {
        $audit = new AuditTrailService();
        $email = $this->getData('email');
        
        $user = $this->userModel->findByEmail($email);
        
        if ($user && password_verify($password, $user->password)) {
            // Successful login
            $audit->logLogin($user, true, [
                'ip' => $_SERVER['REMOTE_ADDR'],
                'user_agent' => $_SERVER['HTTP_USER_AGENT']
            ]);
            
            return $this->sendResponse(true, 'Login successful');
        } else {
            // Failed login
            $audit->logLogin($email, false, [
                'reason' => 'Invalid credentials'
            ]);
            
            return $this->sendResponse(false, 'Invalid credentials', [], 401);
        }
    }
}
```

### Example 4: Track File Deletions

```php
class Uploads extends Controller
{
    public function delete($id)
    {
        $user = $this->RouteProtection();
        $audit = new AuditTrailService();
        
        $upload = $this->uploadModel->findById($id);
        
        // Check ownership
        if ($upload->user_id != $user->user_id) {
            return $this->sendResponse(false, 'Unauthorized', [], 403);
        }
        
        // Delete file
        $this->uploadModel->delete($id);
        
        // Log deletion
        $audit->logDelete(
            $user,
            'upload',
            $id,
            (array)$upload,
            "File '{$upload->filename}' deleted"
        );
        
        return $this->sendResponse(true, 'File deleted');
    }
}
```

---

## Admin Endpoints

### 1. GET `/v1/admin/audit` - View Audit Trail

```bash
# Get all audits
curl "http://api.yourdomain.com/v1/admin/audit" \
  -H "Authorization: Bearer ADMIN_JWT"

# Filter by user
curl "http://api.yourdomain.com/v1/admin/audit?user_id=123&page=1&limit=50" \
  -H "Authorization: Bearer ADMIN_JWT"

# Filter by action
curl "http://api.yourdomain.com/v1/admin/audit?action=delete&entity_type=upload" \
  -H "Authorization: Bearer ADMIN_JWT"

# Date range
curl "http://api.yourdomain.com/v1/admin/audit?date_from=2026-01-01&date_to=2026-01-31" \
  -H "Authorization: Bearer ADMIN_JWT"
```

**Response:**
```json
{
    "status": true,
    "data": {
        "logs": [...],
        "pagination": {
            "page": 1,
            "limit": 50,
            "total": 245,
            "pages": 5
        }
    }
}
```

### 2. GET `/v1/admin/audit/entity/{type}/{id}` - Entity History

```bash
# Get complete history of a payment
curl "http://api.yourdomain.com/v1/admin/audit/entity/payment/123" \
  -H "Authorization: Bearer ADMIN_JWT"
```

### 3. GET `/v1/admin/audit/user/{userId}` - User Activity

```bash
# Get all actions by user
curl "http://api.yourdomain.com/v1/admin/audit/user/123?limit=100" \
  -H "Authorization: Bearer ADMIN_JWT"
```

### 4. GET `/v1/admin/audit/stats` - Statistics

```bash
# Get audit statistics
curl "http://api.yourdomain.com/v1/admin/audit/stats?date_from=2026-01-01" \
  -H "Authorization: Bearer ADMIN_JWT"
```

**Response:**
```json
{
    "status": true,
    "data": {
        "by_action": {
            "create": 1250,
            "update": 850,
            "delete": 125,
            "login_success": 3500
        },
        "by_entity_type": {
            "user": 450,
            "payment": 1200,
            "upload": 575
        }
    }
}
```

---

## Audit Trail Service Methods

### logCreate()
```php
$audit->logCreate($user, 'listing', $listingId, $data, 'Listing created');
```

### logUpdate()
```php
$audit->logUpdate($user, 'user', $userId, $beforeData, $afterData, 'Profile updated');
```

### logDelete()
```php
$audit->logDelete($user, 'upload', $uploadId, $data, 'File deleted');
```

### logLogin()
```php
$audit->logLogin($user, true, ['ip' => '192.168.1.1']);
```

### logLogout()
```php
$audit->logLogout($user);
```

### logPasswordChange()
```php
$audit->logPasswordChange($user);
```

### logAction() - Custom Actions
```php
$audit->logAction($user, 'export_data', 'user', $userId, ['format' => 'csv']);
```

---

## Database Schema

```sql
CREATE TABLE audit_logs (
    id BIGINT PRIMARY KEY,
    user_id BIGINT,
    action VARCHAR(50),          -- create, update, delete, etc.
    entity_type VARCHAR(50),     -- user, payment, upload, etc.
    entity_id BIGINT,
    before_values JSON,          -- State before change
    after_values JSON,           -- State after change
    changes JSON,                -- What changed
    request_id VARCHAR(100),     -- Links to Logger
    description TEXT,
    created_at TIMESTAMP
);
```

---

## Best Practices

### 1. Always Log Sensitive Actions

```php
// ✅ Good - Log deletes
$audit->logDelete($user, 'payment', $id, $data);

// ✅ Good - Log permission changes
$audit->logUpdate($user, 'user', $userId, $before, $after, 'Permissions changed');
```

### 2. Provide Descriptive Messages

```php
// ❌ Bad
$audit->logCreate($user, 'item', $id, $data);

// ✅ Good
$audit->logCreate($user, 'item', $id, $data, "Item '{$name}' created in category '{$category}'");
```

### 3. Don't Log Passwords

```php
// Sanitize before logging
$beforeData = (array)$user;
unset($beforeData['password']);  // Remove sensitive data

$audit->logUpdate($user, 'user', $userId, $beforeData, $afterData);
```

### 4. Link Actions to Requests

The service automatically links audit logs to Logger request_id for correlation.

---

## Compliance & Security

### Data Retention

Audit logs should be retained for compliance:

```php
// Keep audit logs longer than regular logs (90 days minimum)
// Set up scheduled cleanup if needed
```

### Access Control

Audit endpoints should be admin-only:

```php
public function index()
{
    $user = $this->RouteProtection();
    
    if (!$this->isAdmin($user)) {
        return $this->sendResponse(false, 'Admin access required', [], 403);
    }
    // ...
}
```

### GDPR Compliance

Support user data export/deletion:

```php
// Export user's audit trail
GET /v1/admin/audit/user/123

// Anonymize after user deletion
UPDATE audit_logs SET user_email = 'deleted@example.com' WHERE user_id = 123;
```

---

## Monitoring Queries

### Find Recent Deletions

```sql
SELECT * FROM audit_logs 
WHERE action = 'delete' 
AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
ORDER BY created_at DESC;
```

### Track Failed Logins

```sql
SELECT user_email, COUNT(*) as attempts 
FROM audit_logs 
WHERE action = 'login_failed' 
AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
GROUP BY user_email 
HAVING attempts > 5;
```

### Entity Change History

```sql
SELECT * FROM audit_logs 
WHERE entity_type = 'payment' 
AND entity_id = 123 
ORDER BY created_at ASC;
```

---

## Summary

**Audit Trail System provides:**
- ✅ Complete action tracking
- ✅ Before/after value storage
- ✅ Entity change history
- ✅ User activity logs
- ✅ Security compliance
- ✅ Admin management endpoints

**Essential for:**
- Security audits
- Compliance (GDPR, SOC2)
- Debugging issues
- User dispute resolution
- Fraud detection

Your API is now fully auditable! 🔒
