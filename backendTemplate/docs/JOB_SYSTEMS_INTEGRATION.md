# Job Systems Integration Guide

## Two Complementary Systems

Your API has **two job systems** that work together:

### 1. **JobQueue** (Async/Background Jobs)
- **Location:** `app/libraries/JobQueue.php`
- **Purpose:** Process tasks asynchronously in background
- **Use for:** Email sending, image processing, API calls, heavy operations
- **Pattern:** "Do this task NOW, but in the background"

### 2. **ScheduledJobsService** (Cron/Recurring Jobs)
- **Location:** `app/services/ScheduledJobsService.php`
- **Purpose:** Run tasks on schedule (hourly, daily, weekly)
- **Use for:** Cleanup, reports, maintenance, automated tasks
- **Pattern:** "Do this task EVERY DAY at 2 AM"

---

## How They Work Together

**Scheduled jobs** can **queue async jobs** for processing:

```
Scheduled Job (runs daily) 
    ↓
Pushes tasks to JobQueue
    ↓
Worker processes them asynchronously
```

---

## Example 1: Daily Email Reports

**Scheduled Job** (runs daily at 6 AM):
```php
<?php
// app/jobs/DailyReportJob.php
require_once APPROOT . '/jobs/BaseJob.php';

class DailyReportJob extends BaseJob
{
    public function execute(): string
    {
        $queue = new JobQueue();
        
        // Get all users who requested daily reports
        $this->db->query("SELECT user_id, email FROM users WHERE daily_report = 1");
        $users = $this->db->resultSet();
        
        // Queue email job for each user
        foreach ($users as $user) {
            $queue->push('SendDailyReport', [
                'user_id' => $user->user_id,
                'email' => $user->email,
                'date' => date('Y-m-d')
            ]);
        }
        
        $this->log("Queued " . count($users) . " daily reports");
        
        return "Queued " . count($users) . " reports for processing";
    }
}
```

**Worker** processes the queued jobs:
```php
// worker.php (running continuously)
$queue = new JobQueue();

while (true) {
    $job = $queue->pop();
    
    if ($job) {
        switch ($job['type']) {
            case 'SendDailyReport':
                $emailService = new EmailService();
                $emailService->sendDailyReport(
                    $job['data']['email'],
                    $job['data']['user_id']
                );
                $queue->delete($job['id']);
                break;
        }
    } else {
        sleep(5); // Wait before checking again
    }
}
```

---

## Example 2: Bulk Image Processing

**Scheduled Job** (runs hourly):
```php
<?php
// app/jobs/ProcessPendingImagesJob.php
require_once APPROOT . '/jobs/BaseJob.php';

class ProcessPendingImagesJob extends BaseJob
{
    public function execute(): string
    {
        $queue = new JobQueue();
        
        // Find uploads that need processing
        $this->db->query("SELECT id, file_path 
            FROM uploads 
            WHERE processed = 0 
            AND file_type = 'image'
            LIMIT 100");
        
        $images = $this->db->resultSet();
        
        foreach ($images as $image) {
            // Queue each image for processing
            $queue->push('ProcessImage', [
                'upload_id' => $image->id,
                'file_path' => $image->file_path
            ]);
        }
        
        return "Queued " . count($images) . " images for processing";
    }
}
```

**Worker** handles the heavy processing:
```php
// In worker.php
case 'ProcessImage':
    $processor = new ImageProcessor();
    $processor->resize($job['data']['file_path'], 800, 800);
    $processor->compress($job['data']['file_path'], 85);
    
    // Mark as processed
    $db->query("UPDATE uploads SET processed = 1 WHERE id = :id");
    $db->bind(':id', $job['data']['upload_id']);
    $db->execute();
    
    $queue->delete($job['id']);
    break;
```

---

## Example 3: Payment Reminder System

**Scheduled Job** (runs daily):
```php
<?php
// app/jobs/PaymentRemindersJob.php
require_once APPROOT . '/jobs/BaseJob.php';

class PaymentRemindersJob extends BaseJob
{
    public function execute(): string
    {
        $queue = new JobQueue();
        $sms = new SMSService();
        
        // Find overdue payments
        $this->db->query("SELECT * FROM payments 
            WHERE status = 'pending' 
            AND due_date < CURDATE() 
            AND reminder_sent = 0");
        
        $overdue = $this->db->resultSet();
        
        foreach ($overdue as $payment) {
            // Queue SMS job
            $queue->push('SendPaymentReminder', [
                'payment_id' => $payment->id,
                'phone' => $payment->phone,
                'amount' => $payment->amount
            ]);
        }
        
        return "Queued " . count($overdue) . " payment reminders";
    }
}
```

---

## Example 4: Log Cleanup with Async Processing

**Scheduled Job** (runs daily):
```php
<?php
// app/jobs/CleanupLogsJob.php (enhanced version)
require_once APPROOT . '/jobs/BaseJob.php';

class CleanupLogsJob extends BaseJob
{
    public function execute(): string
    {
        $queue = new JobQueue();
        $logDir = APPROOT . '/logs';
        
        // Find old log files
        $files = glob($logDir . '/*.log');
        $cutoff = strtotime('-30 days');
        $toClean = [];
        
        foreach ($files as $file) {
            if (filemtime($file) < $cutoff) {
                $toClean[] = $file;
            }
        }
        
        // If many files, queue for async processing
        if (count($toClean) > 100) {
            $queue->push('BulkFileCleanup', [
                'files' => $toClean,
                'type' => 'logs'
            ]);
            
            return "Queued " . count($toClean) . " files for async cleanup";
        }
        
        // If few files, delete directly
        $deleted = 0;
        foreach ($toClean as $file) {
            if (unlink($file)) {
                $deleted++;
            }
        }
        
        return "Deleted {$deleted} old log files";
    }
}
```

---

## Complete Worker Script

```php
<?php
// worker.php - Processes async jobs from queue
require_once __DIR__ . '/app/bootstrap.php';

$queue = new JobQueue();
$emailService = new EmailService();
$smsService = new SMSService();

echo "Worker started at " . date('Y-m-d H:i:s') . "\n";

while (true) {
    try {
        $job = $queue->pop();
        
        if ($job) {
            echo "[" . date('H:i:s') . "] Processing: {$job['type']}\n";
            
            switch ($job['type']) {
                case 'SendDailyReport':
                    $emailService->sendDailyReport(
                        $job['data']['email'],
                        $job['data']['user_id']
                    );
                    $queue->delete($job['id']);
                    break;
                    
                case 'ProcessImage':
                    $processor = new ImageProcessor();
                    $processor->resize($job['data']['file_path'], 800, 800);
                    $queue->delete($job['id']);
                    break;
                    
                case 'SendPaymentReminder':
                    $smsService->send(
                        $job['data']['phone'],
                        "Payment reminder: ₦{$job['data']['amount']} is overdue"
                    );
                    $queue->delete($job['id']);
                    break;
                    
                case 'BulkFileCleanup':
                    foreach ($job['data']['files'] as $file) {
                        @unlink($file);
                    }
                    $queue->delete($job['id']);
                    break;
                    
                default:
                    echo "Unknown job type: {$job['type']}\n";
                    $queue->delete($job['id']);
            }
            
            echo "[" . date('H:i:s') . "] Completed: {$job['type']}\n";
        } else {
            sleep(5); // No jobs, wait 5 seconds
        }
        
    } catch (Exception $e) {
        echo "Error: " . $e->getMessage() . "\n";
        if (isset($job)) {
            $queue->release($job['id'], 300); // Retry in 5 minutes
        }
        sleep(5);
    }
}
```

---

## Setup Instructions

### 1. Run Scheduled Jobs (Cron)
```bash
# Add to crontab - runs every minute
* * * * * php /path/to/api/cron/cron.php >> /var/log/cron.log 2>&1
```

### 2. Run Worker (Async Processing)
```bash
# Run as background service
nohup php /path/to/api/app/workers/worker.php >> /var/log/worker.log 2>&1 &

# Or use supervisor (recommended for production)
```

### 3. Supervisor Config (Production)
```ini
[program:api-worker]
command=php /var/www/api/app/workers/worker.php
directory=/var/www/api
autostart=true
autorestart=true
user=www-data
stdout_logfile=/var/log/worker.log
stderr_logfile=/var/log/worker-error.log
```

---

## Decision Tree

**Use Scheduled Jobs when:**
- ✅ Task runs on schedule (hourly, daily, weekly)
- ✅ Automated maintenance (cleanup, backups)
- ✅ Periodic reports
- ✅ Recurring notifications

**Use JobQueue when:**
- ✅ User-triggered background tasks
- ✅ Email sending (don't make user wait)
- ✅ Image/video processing
- ✅ API calls to third parties
- ✅ Any slow operation

**Use BOTH when:**
- ✅ Scheduled job finds work, queues it for processing
- ✅ Example: Daily job finds 1000 emails to send → queues them → worker sends them
- ✅ Heavy workloads need async processing

---

## Best Practices

1. **Scheduled jobs should be fast**
   - Don't do heavy work in scheduled jobs
   - Queue heavy work for async processing

2. **Worker should handle failures gracefully**
   - Use try-catch
   - Use $queue->release() for retries
   - Max 3 attempts (built into JobQueue)

3. **Monitor both systems**
   - Check cron logs: `/var/log/cron.log`
   - Check worker logs: `/var/log/worker.log`
   - Admin endpoints: `/v1/admin/jobs`

4. **Database performance**
   - JobQueue uses transactions for reliability
   - Scheduled jobs tracks execution history

---

## Summary

| Feature | Scheduled Jobs | JobQueue |
|---------|---------------|----------|
| **Purpose** | Recurring tasks | Async processing |
| **Runs** | On schedule | Continuously |
| **Trigger** | Time-based | Event-based |
| **Examples** | Daily cleanup | Send email |
| **File** | `cron/cron.php` | `app/workers/worker.php` |

**Together:** Scheduled jobs can queue async jobs for heavy processing! 🚀
