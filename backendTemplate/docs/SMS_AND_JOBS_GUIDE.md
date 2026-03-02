# SMS & Scheduled Jobs - Quick Guide

## SMS Service

### Configuration
Edit `app/config/sms_config.php`:
```php
'provider' => 'twilio',  // or 'africas_talking'
'providers' => [
    'twilio' => [
        'account_sid' => 'your_sid',
        'auth_token' => 'your_token',
        'from_number' => '+1234567890'
    ]
]
```

### Usage

**Send OTP:**
```php
$sms = new SMSService();
$result = $sms->sendOTP('+234812345678', 'verification');
```

**Verify OTP:**
```php
if ($sms->verifyOTP('+234812345678', '123456', 'verification')) {
    // OTP valid
}
```

**Send Custom SMS:**
```php
$sms->send('+234812345678', 'Your custom message');
```

---

## Scheduled Jobs

### Setup Cron
```bash
# Add to crontab (runs every minute)
* * * * * php /path/to/api/cron/cron.php >> /var/log/cron.log 2>&1
```

### Create Custom Job
1. Create file: `app/jobs/YourJob.php`
```php
<?php
require_once APPROOT . '/jobs/BaseJob.php';

class YourJob extends BaseJob
{
    public function execute(): string
    {
        // Your job logic here
        return "Job completed";
    }
}
```

2. Add to database:
```sql
INSERT INTO scheduled_jobs (name, job_class, schedule, enabled) 
VALUES ('your_job', 'YourJob', 'daily', 1);
```

### Admin Endpoints
```bash
# List all jobs
GET /v1/admin/jobs

# Run specific job manually
POST /v1/admin/jobs/{id}/run

# Enable/disable job
PUT /v1/admin/jobs/{id}/toggle

# View job history
GET /v1/admin/jobs/{id}/history
```

### Schedules
- `minutely` - Every minute
- `hourly` - Every hour
- `daily` - Every day
- `weekly` - Every week
- `monthly` - Every month

---

## Database Migrations

```bash
# SMS tables
mysql -u root -p your_db < database/sms_tables.sql

# Jobs tables
mysql -u root -p your_db < database/scheduled_jobs_tables.sql
```

---

## Pre-built Jobs

1. **CleanupLogsJob** - Cleans logs >30 days
2. **CleanupOTPJob** - Removes expired OTPs
3. **CleanupUploadsJob** - Cleanup orphaned files

All run automatically when enabled! ✅
