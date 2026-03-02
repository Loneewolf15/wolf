# Data Export Service - Quick Guide

## Overview

GDPR-compliant data export system with CSV/JSON formats, background processing, and automatic cleanup.

---

## Features

✅ **CSV & JSON Export**: Multiple format support  
✅ **Background Processing**: Via JobQueue  
✅ **Email Notifications**: When export is ready  
✅ **Auto Cleanup**: Expires after 7 days  
✅ **Rate Limiting**: 5 exports/hour per user  
✅ **GDPR Compliant**: User data export  

---

## Quick Start

### 1. Apply Database Migration

```bash
mysql -u root -p your_database < database/exports_table.sql
```

### 2. Add Scheduled Job

```sql
INSERT INTO scheduled_jobs (name, job_class, schedule, enabled) 
VALUES ('cleanup_exports', 'CleanupExportsJob', 'daily', 1);
```

### 3. Request Export

```bash
POST /v1/exports
{
    "type": "payments",
    "format": "csv",
    "filters": {
        "date_from": "2026-01-01",
        "date_to": "2026-01-31"
    }
}
```

---

## Export Types

1. **user_data** - Personal data (GDPR compliance)
2. **payments** - Payment history
3. **transactions** - Transaction records
4. **uploads** - File upload metadata

---

## API Endpoints

### Request Export
```bash
POST /v1/exports
Content-Type: application/json
Authorization: Bearer JWT_TOKEN

{
    "type": "payments",
    "format": "csv"
}
```

**Response:**
```json
{
    "status": true,
    "data": {
        "export_id": 123,
        "status": "pending",
        "message": "Your export is being processed..."
    }
}
```

### List Exports
```bash
GET /v1/exports
```

### Get Export Details
```bash
GET /v1/exports/{id}
```

### Download Export
```bash
GET /v1/exports/{id}/download
```

### Available Types
```bash
GET /v1/exports/types
```

---

## How It Works

```
1. User requests export → Creates pending export record
2. Request queued in JobQueue → Background processing
3. Worker processes export → Generates CSV/JSON file
4. Email sent to user → "Your export is ready!"
5. User downloads file → 7-day expiry
6. Scheduled job cleans up → Deletes expired exports
```

---

## Usage Examples

### Export User Data (GDPR)

```php
// Request user data export
POST /v1/exports
{
    "type": "user_data",
    "format": "json"
}
```

### Export Payments with Filters

```php
POST /v1/exports
{
    "type": "payments",
    "format": "csv",
    "filters": {
        "date_from": "2026-01-01",
        "date_to": "2026-12-31"
    }
}
```

### Check Export Status

```javascript
const exportId = 123;

async function checkStatus() {
    const response = await fetch(`/v1/exports/${exportId}`, {
        headers: { 'Authorization': 'Bearer ' + token }
    });
    
    const data = await response.json();
    
    if (data.status === 'completed') {
        // Download ready
        window.location = `/v1/exports/${exportId}/download`;
    } else {
        // Still processing
        console.log(`Progress: ${data.progress}%`);
    }
}
```

---

## Rate Limiting

- **5 exports per hour** per user
- Exceeding returns `429 Too Many Requests`

---

## File Storage

- **Location**: `public/assets/exports/`
- **Format**: `export_{type}_{timestamp}_{id}.{format}`
- **Example**: `export_payments_20260124_123456_1.csv`
- **Expiry**: 7 days automatic deletion

---

## Integration with Scheduled Jobs

Add to `scheduled_jobs` table:

```sql
INSERT INTO scheduled_jobs (name, description, job_class, schedule, enabled) 
VALUES (
    'cleanup_exports', 
    'Clean exports older than 7 days', 
    'CleanupExportsJob', 
    'daily', 
    1
);
```

---

## Email Notification

Users receive email when export is ready:

```
Subject: Your Data Export is Ready

Hi John,

Your data export is ready for download.

Download link: https://api.yourdomain.com/v1/exports/123/download

This link expires in 7 days.
```

---

## Security

- ✅ JWT authentication required
- ✅ Ownership verification (users can only download their own exports)
- ✅ Sensitive data removed (passwords excluded)
- ✅ Rate limiting prevents abuse
- ✅ Automatic cleanup prevents storage bloat

---

## Adding Custom Export Types

```php
// In ExportService.php, add to getData() method
case 'orders':
    return $this->getOrders($export->user_id, $filters);

// Add method
private function getOrders(int $userId, array $filters): array
{
    $this->db->query("SELECT * FROM orders 
        WHERE user_id = :user_id 
        ORDER BY created_at DESC");
    $this->db->bind(':user_id', $userId);
    
    $results = $this->db->resultSet();
    return array_map(fn($row) => (array)$row, $results);
}
```

---

## Monitoring

### Check Export Statistics

```sql
SELECT 
    export_type,
    format,
    COUNT(*) as total,
    AVG(total_records) as avg_records,
    SUM(file_size) as total_size
FROM exports
WHERE status = 'completed'
GROUP BY export_type, format;
```

### Failed Exports

```sql
SELECT * FROM exports 
WHERE status = 'failed' 
ORDER BY requested_at DESC 
LIMIT 10;
```

---

## Summary

**Data Export Service provides:**
- ✅ GDPR compliance (user data export)
- ✅ Multiple formats (CSV, JSON)
- ✅ Background processing (no blocking)
- ✅ Email notifications
- ✅ Automatic cleanup
- ✅ Rate limiting

Perfect for compliance and user data portability! 📊
