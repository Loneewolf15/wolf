<?php

/**
 * Cleanup Exports Job
 * 
 * Removes expired exports (older than 7 days)
 */
require_once APPROOT . '/jobs/BaseJob.php';

class CleanupExportsJob extends BaseJob
{
    public function execute(): string
    {
        $exportService = new ExportService();
        $deleted = $exportService->cleanExpiredExports();

        $message = "Cleaned {$deleted} expired exports";
        $this->log($message);

        return $message;
    }
}
