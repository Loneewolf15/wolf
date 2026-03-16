-- ============================================
-- Scheduled Jobs Database Migration
-- ============================================

USE market_plaza_pid2025;

-- Scheduled jobs table
CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NULL,
    job_class VARCHAR(255) NOT NULL COMMENT 'Class name to execute',
    schedule VARCHAR(100) NOT NULL COMMENT 'Cron expression or interval',
    enabled TINYINT(1) DEFAULT 1,
    last_run_at TIMESTAMP NULL,
    next_run_at TIMESTAMP NULL,
    run_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_name (name),
    INDEX idx_enabled (enabled),
    INDEX idx_next_run (next_run_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Job execution history
CREATE TABLE IF NOT EXISTS job_executions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_id BIGINT UNSIGNED NOT NULL,
    status ENUM('running', 'completed', 'failed') NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    duration_seconds DECIMAL(10, 2) NULL,
    output TEXT NULL,
    error_message TEXT NULL,
    
    INDEX idx_job_id (job_id),
    INDEX idx_status (status),
    INDEX idx_started_at (started_at),
    
    FOREIGN KEY (job_id) REFERENCES scheduled_jobs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insert default jobs
INSERT INTO scheduled_jobs (name, description, job_class, schedule, enabled) VALUES
('cleanup_old_logs', 'Clean logs older than 30 days', 'CleanupLogsJob', 'daily', 1),
('cleanup_expired_otps', 'Clean expired OTP codes', 'CleanupOTPJob', 'hourly', 1),
('cleanup_old_uploads', 'Clean old temporary uploads', 'CleanupUploadsJob', 'daily', 1);

SELECT 'Scheduled jobs tables created successfully!' AS message;
