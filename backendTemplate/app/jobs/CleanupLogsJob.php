<?php

/**
 * Cleanup Logs Job
 * 
 * Removes logs older than 30 days
 */
require_once APPROOT . '/jobs/BaseJob.php';

class CleanupLogsJob extends BaseJob
{
    public function execute(): string
    {
        $logger = new Logger();
        $deleted = $logger->cleanOldLogs(30);

        $message = "Cleaned {$deleted} old log files";
        $this->log($message);

        return $message;
    }
}
