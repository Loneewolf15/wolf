#!/usr/bin/env php
<?php
/**
 * Cron Runner Script
 * 
 * Add to crontab: * * * * * php /path/to/api/cron.php
 */

// Bootstrap the application
require_once __DIR__ . '/app/bootstrap.php';

// Run due jobs
$jobService = new ScheduledJobsService();
$results = $jobService->runDueJobs();

// Output results
echo date('Y-m-d H:i:s') . " - Ran " . count($results) . " jobs\n";

foreach ($results as $result) {
    echo "  - {$result['job']}: {$result['status']}";
    if (isset($result['duration'])) {
        echo " ({$result['duration']}s)";
    }
    echo "\n";
}
