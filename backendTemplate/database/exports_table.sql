-- ============================================
-- Data Export Tables Migration
-- ============================================

USE market_plaza_pid2025;

-- Exports table
CREATE TABLE IF NOT EXISTS exports (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    
    -- Export details
    export_type VARCHAR(50) NOT NULL COMMENT 'user_data, payments, transactions, custom',
    format VARCHAR(10) NOT NULL COMMENT 'csv, json',
    
    -- Status
    status ENUM('pending', 'processing', 'completed', 'failed') DEFAULT 'pending',
    
    -- File details
    filename VARCHAR(255) NULL,
    file_path VARCHAR(500) NULL,
    file_size BIGINT NULL COMMENT 'Size in bytes',
    
    -- Metadata
    filters JSON NULL COMMENT 'Export filters/parameters',
    total_records INT DEFAULT 0,
    
    -- Progress tracking
    progress INT DEFAULT 0 COMMENT 'Percentage 0-100',
    error_message TEXT NULL,
    
    -- Timestamps
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL COMMENT 'Auto-delete after 7 days',
    
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_type (export_type),
    INDEX idx_expires (expires_at),
    
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SELECT 'Exports table created successfully!' AS message;
