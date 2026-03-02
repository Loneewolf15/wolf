-- ============================================
-- Audit Trail Database Migration
-- ============================================
-- Creates audit_logs table for tracking user actions
-- Run with: mysql -u root -p market_plaza_pid2025 < database/audit_logs_table.sql
-- ============================================

USE market_plaza_pid2025;

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    
    -- User information
    user_id BIGINT UNSIGNED NULL,
    user_email VARCHAR(255) NULL,
    user_ip VARCHAR(45) NULL,
    
    -- Action details
    action VARCHAR(50) NOT NULL COMMENT 'create, update, delete, login, etc.',
    entity_type VARCHAR(50) NOT NULL COMMENT 'user, payment, upload, listing, etc.',
    entity_id BIGINT UNSIGNED NULL COMMENT 'ID of affected entity',
    
    -- Change tracking
    before_values JSON NULL COMMENT 'Entity state before change',
    after_values JSON NULL COMMENT 'Entity state after change',
    changes JSON NULL COMMENT 'Summary of what changed',
    
    -- Request context
    request_id VARCHAR(100) NULL COMMENT 'Corresponds to Logger request_id',
    endpoint VARCHAR(255) NULL COMMENT 'API endpoint called',
    http_method VARCHAR(10) NULL COMMENT 'GET, POST, PUT, DELETE',
    
    -- Additional metadata
    metadata JSON NULL COMMENT 'Additional context',
    description TEXT NULL COMMENT 'Human-readable description',
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for performance
    INDEX idx_user_id (user_id),
    INDEX idx_action (action),
    INDEX idx_entity_type (entity_type),
    INDEX idx_entity_id (entity_id),
    INDEX idx_created_at (created_at),
    INDEX idx_user_action (user_id, action),
    INDEX idx_entity (entity_type, entity_id),
    INDEX idx_request_id (request_id),
    
    -- Foreign key
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Verify table creation
-- ============================================
DESCRIBE audit_logs;

SELECT 'Audit logs table created successfully!' AS message;

-- Show indexes
SHOW INDEX FROM audit_logs;

-- ============================================
-- Sample queries
-- ============================================

-- Get all actions for a user:
-- SELECT * FROM audit_logs WHERE user_id = 1 ORDER BY created_at DESC;

-- Get all changes to a specific entity:
-- SELECT * FROM audit_logs WHERE entity_type = 'payment' AND entity_id = 123;

-- Get all delete actions:
-- SELECT * FROM audit_logs WHERE action = 'delete' ORDER BY created_at DESC;

-- Count actions by type:
-- SELECT action, COUNT(*) as count FROM audit_logs GROUP BY action;

-- Get recent sensitive actions:
-- SELECT * FROM audit_logs 
-- WHERE action IN ('delete', 'update') 
-- AND entity_type IN ('user', 'payment')
-- ORDER BY created_at DESC LIMIT 50;
