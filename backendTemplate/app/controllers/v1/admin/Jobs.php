<?php

/**
 * Jobs Controller (Admin)
 * 
 * Manage scheduled jobs
 */
class Jobs extends Controller
{
    private $jobService;

    public function __construct()
    {
        $this->jobService = new ScheduledJobsService();
    }

    /**
     * GET /v1/admin/jobs - List all jobs
     */
    public function index()
    {
        $user = $this->RouteProtection();

        $jobs = $this->jobService->getAllJobs();

        return $this->sendResponse(true, 'Jobs retrieved', [
            'total' => count($jobs),
            'jobs' => $jobs
        ]);
    }

    /**
     * POST /v1/admin/jobs/run - Run all due jobs manually
     */
    public function run()
    {
        $user = $this->RouteProtection();

        /*
     ### 5. SMS Service
- [x] Choose SMS provider (Twilio, Africa's Talking)
- [x] Create SMSService class
  - [x] Send single SMS
  - [x] Send bulk SMS (via loop)
  - [x] OTP generation & verification
  - [x] SMS templates
- [x] Configuration
  - [x] sms_config.php with credentials
  - [x] Template management
- [x] Implement use cases
  - [x] OTP for 2FA
  - [x] Password reset via SMS
  - [x] Transaction notifications (template)
  - [x] Marketing messages (template)
- [x] Rate limiting (prevent SMS spam)
- [x] Cost tracking & limits
- [x] Database tables (otp_codes, sms_logs)
- [x] Documentation created

### 6. Scheduled Jobs / Cron Service
- [x] Create Job scheduler infrastructure
  - [x] Job queue table (scheduled_jobs)
  - [x] Job types enum
  - [x] Job status tracking (job_executions)
- [x] Implement ScheduledJobsService
  - [x] Register jobs
  - [x] Execute jobs
  - [x] Retry failed jobs (manual)
  - [x] Job logging
- [x] Create common jobs
  - [x] Daily cleanup (temp files, old logs)
  - [x] Cleanup expired OTPs
  - [x] Upload cleanup
  - [x] BaseJob class template
- [x] Job management endpoints
  - [x] GET /v1/admin/jobs - List jobs
  - [x] POST /v1/admin/jobs/{id}/run - Trigger job
  - [x] GET /v1/admin/jobs/{id}/history - Job history
  - [x] PUT /v1/admin/jobs/{id}/toggle - Enable/disable
- [x] Cron setup instructions (cron.php)
- [x] Documentation created
        */

        $results = $this->jobService->runDueJobs();

        return $this->sendResponse(true, 'Jobs executed', [
            'executed' => count($results),
            'results' => $results
        ]);
    }

    /**
     * POST /v1/admin/jobs/{id}/run - Run specific job manually
     */
    public function runJob($id)
    {
        $user = $this->RouteProtection();

        $job = $this->jobService->getJob((int)$id);

        if (!$job) {
            return $this->sendResponse(false, 'Job not found', [], 404);
        }

        $result = $this->jobService->runJob($job);

        return $this->sendResponse(
            $result['status'] === 'completed',
            "Job {$job->name} executed",
            $result
        );
    }

    /**
     * PUT /v1/admin/jobs/{id}/toggle - Enable/disable job
     */
    public function toggle($id)
    {
        $user = $this->RouteProtection();

        $enabled = $this->getData('enabled') === '1' || $this->getData('enabled') === 'true';

        $this->jobService->toggleJob((int)$id, $enabled);

        return $this->sendResponse(true, 'Job updated', [
            'enabled' => $enabled
        ]);
    }

    /**
     * GET /v1/admin/jobs/{id}/history - Get job execution history
     */
    public function history($id)
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 50), 100);

        $history = $this->jobService->getJobHistory((int)$id, $limit);

        return $this->sendResponse(true, 'Job history retrieved', [
            'total' => count($history),
            'history' => $history
        ]);
    }
}
