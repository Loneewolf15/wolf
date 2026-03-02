<?php

/**
 * Scheduled Jobs Service
 * 
 * Run and manage scheduled jobs (cron-like functionality)
 */
class ScheduledJobsService
{
    private $db;

    public function __construct()
    {
        $this->db = new Database();
    }

    /**
     * Run all due jobs
     */
    public function runDueJobs(): array
    {
        $jobs = $this->getDueJobs();
        $results = [];

        foreach ($jobs as $job) {
            $result = $this->runJob($job);
            $results[] = [
                'job' => $job->name,
                'status' => $result['status'],
                'duration' => $result['duration'] ?? null
            ];
        }

        return $results;
    }

    /**
     * Run a specific job
     */
    public function runJob(object $job): array
    {
        $executionId = $this->startExecution($job->id);
        $startTime = microtime(true);

        try {
            // Load job class
            $jobClass = $job->job_class;
            $jobFile = APPROOT . '/jobs/' . $jobClass . '.php';

            if (!file_exists($jobFile)) {
                throw new Exception("Job class file not found: {$jobFile}");
            }

            require_once $jobFile;

            if (!class_exists($jobClass)) {
                throw new Exception("Job class not found: {$jobClass}");
            }

            // Execute job
            $jobInstance = new $jobClass();
            $output = $jobInstance->execute();

            // Mark as completed
            $duration = microtime(true) - $startTime;
            $this->completeExecution($executionId, 'completed', $output, $duration);
            $this->updateJobRun($job->id);

            return [
                'status' => 'completed',
                'output' => $output,
                'duration' => round($duration, 2)
            ];
        } catch (Exception $e) {
            $duration = microtime(true) - $startTime;
            $this->completeExecution($executionId, 'failed', null, $duration, $e->getMessage());

            return [
                'status' => 'failed',
                'error' => $e->getMessage(),
                'duration' => round($duration, 2)
            ];
        }
    }

    /**
     * Get jobs that are due to run
     */
    private function getDueJobs(): array
    {
        $this->db->query("SELECT * FROM scheduled_jobs 
            WHERE enabled = 1 
            AND (next_run_at IS NULL OR next_run_at <= NOW())
            ORDER BY next_run_at ASC");

        return $this->db->resultSet();
    }

    /**
     * Start job execution
     */
    private function startExecution(int $jobId): int
    {
        $this->db->query("INSERT INTO job_executions (job_id, status) 
            VALUES (:job_id, 'running')");
        $this->db->bind(':job_id', $jobId);
        $this->db->execute();

        return $this->db->lastInsertId();
    }

    /**
     * Complete job execution
     */
    private function completeExecution(int $executionId, string $status, ?string $output, float $duration, ?string $error = null): void
    {
        $this->db->query("UPDATE job_executions 
            SET status = :status,
                completed_at = NOW(),
                duration_seconds = :duration,
                output = :output,
                error_message = :error
            WHERE id = :id");

        $this->db->bind(':id', $executionId);
        $this->db->bind(':status', $status);
        $this->db->bind(':duration', $duration);
        $this->db->bind(':output', $output);
        $this->db->bind(':error', $error);
        $this->db->execute();
    }

    /**
     * Update job last/next run times
     */
    private function updateJobRun(int $jobId): void
    {
        // Get job schedule
        $this->db->query("SELECT * FROM scheduled_jobs WHERE id = :id");
        $this->db->bind(':id', $jobId);
        $job = $this->db->single();

        // Calculate next run
        $nextRun = $this->calculateNextRun($job->schedule);

        $this->db->query("UPDATE scheduled_jobs 
            SET last_run_at = NOW(),
                next_run_at = :next_run,
                run_count = run_count + 1
            WHERE id = :id");

        $this->db->bind(':id', $jobId);
        $this->db->bind(':next_run', $nextRun);
        $this->db->execute();
    }

    /**
     * Calculate next run time
     */
    private function calculateNextRun(string $schedule): string
    {
        return match ($schedule) {
            'minutely' => date('Y-m-d H:i:s', strtotime('+1 minute')),
            'hourly' => date('Y-m-d H:i:s', strtotime('+1 hour')),
            'daily' => date('Y-m-d H:i:s', strtotime('+1 day')),
            'weekly' => date('Y-m-d H:i:s', strtotime('+1 week')),
            'monthly' => date('Y-m-d H:i:s', strtotime('+1 month')),
            default => date('Y-m-d H:i:s', strtotime('+1 hour'))
        };
    }

    /**
     * Get all jobs
     */
    public function getAllJobs(): array
    {
        $this->db->query("SELECT * FROM scheduled_jobs ORDER BY name ASC");
        return $this->db->resultSet();
    }

    /**
     * Get job by ID
     */
    public function getJob(int $id): ?object
    {
        $this->db->query("SELECT * FROM scheduled_jobs WHERE id = :id");
        $this->db->bind(':id', $id);
        return $this->db->single() ?: null;
    }

    /**
     * Enable/disable job
     */
    public function toggleJob(int $id, bool $enabled): bool
    {
        $this->db->query("UPDATE scheduled_jobs SET enabled = :enabled WHERE id = :id");
        $this->db->bind(':id', $id);
        $this->db->bind(':enabled', $enabled ? 1 : 0);
        return $this->db->execute();
    }

    /**
     * Get job execution history
     */
    public function getJobHistory(int $jobId, int $limit = 50): array
    {
        $this->db->query("SELECT * FROM job_executions 
            WHERE job_id = :job_id 
            ORDER BY started_at DESC 
            LIMIT :limit");

        $this->db->bind(':job_id', $jobId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        return $this->db->resultSet();
    }
}
