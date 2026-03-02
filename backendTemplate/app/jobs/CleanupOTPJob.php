<?php

/**
 * Cleanup OTP Job
 * 
 * Removes expired OTP codes
 */
require_once APPROOT . '/jobs/BaseJob.php';

class CleanupOTPJob extends BaseJob
{
    public function execute(): string
    {
        $this->db->query("DELETE FROM otp_codes WHERE expires_at < NOW()");
        $this->db->execute();
        $count = $this->db->rowCount();

        $message = "Deleted {$count} expired OTP codes";
        $this->log($message);

        return $message;
    }
}
