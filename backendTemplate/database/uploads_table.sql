-- ============================================
-- File Uploads Database Migration
-- ============================================
-- Creates uploads table for tracking uploaded files
-- Run with: mysql -u root -p market_plaza_pid2025 < database/uploads_table.sql
-- ============================================

USE market_plaza_pid2025;

-- Create uploads table
CREATE TABLE IF NOT EXISTS uploads (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    
    -- File information
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL COMMENT 'Relative filepath',
    file_url VARCHAR(500) NOT NULL COMMENT 'Full URL with URLROOT',
    
    -- File metadata
    file_type ENUM('image', 'video', 'document', 'audio', 'other') NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size INT UNSIGNED NOT NULL COMMENT 'File size in bytes',
    
    -- Storage information
    storage_type ENUM('local', 'cloud') NOT NULL DEFAULT 'local',
    cloud_provider VARCHAR(50) NULL COMMENT 'cloudinary, s3, azure, etc.',
    cloud_id VARCHAR(255) NULL COMMENT 'Cloud provider file ID',
    
    -- Image variants (thumbnails, resized versions)
    variants JSON NULL COMMENT 'Stores thumbnail URLs and other variants',
    
    -- Additional metadata
    metadata JSON NULL COMMENT 'Additional file metadata',
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Indexes for performance
    INDEX idx_user_id (user_id),
    INDEX idx_file_type (file_type),
    INDEX idx_storage_type (storage_type),
    INDEX idx_created_at (created_at),
    INDEX idx_user_type (user_id, file_type),
    INDEX idx_user_created (user_id, created_at),
    
    -- Foreign key constraint
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Verify table creation
-- ============================================
DESCRIBE uploads;

SELECT 'Uploads table created successfully!' AS message;

-- Show indexes
SHOW INDEX FROM uploads;

-- ============================================
-- Sample queries for testing
-- ============================================

-- Get all uploads for a user:
-- SELECT * FROM uploads WHERE user_id = 1 ORDER BY created_at DESC;

-- Get user's total storage usage:
-- SELECT user_id, COUNT(*) as file_count, SUM(file_size) as total_bytes
-- FROM uploads WHERE user_id = 1;

-- Get user's images only:
-- SELECT * FROM uploads WHERE user_id = 1 AND file_type = 'image';

-- Get upload statistics:
-- SELECT file_type, COUNT(*) as count, SUM(file_size) as total_size
-- FROM uploads GROUP BY file_type;
