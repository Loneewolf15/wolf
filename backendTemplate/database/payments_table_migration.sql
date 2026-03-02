-- ============================================
-- Payment Gateway Database Migration
-- ============================================
-- Creates payments table with indexes for performance
-- Run with: mysql -u root -p market_plaza_pid2025 < database/payments_table_migration.sql
-- ============================================

USE market_plaza_pid2025;

-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    provider ENUM('paystack', 'monnify') NOT NULL,
    reference VARCHAR(100) NOT NULL UNIQUE,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    status ENUM('pending', 'successful', 'failed', 'refunded') NOT NULL DEFAULT 'pending',
    metadata TEXT,
    gateway_response TEXT,
    verified_at TIMESTAMP NULL,
    refunded_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Indexes for performance
    INDEX idx_user_id (user_id),
    INDEX idx_provider (provider),
    INDEX idx_reference (reference),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_user_status (user_id, status),
    
    -- Foreign key constraint
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Verify table creation
-- ============================================
DESCRIBE payments;

SELECT 'Payments table created successfully!' AS message;

-- ============================================
-- Query for checking indexes
-- ============================================
-- SHOW INDEX FROM payments;

-- ============================================
-- Sample queries for testing
-- ============================================
-- Get all successful payments for a user:
-- SELECT * FROM payments WHERE user_id = 1 AND status = 'successful' ORDER BY created_at DESC;

-- Get total amount paid by user:
-- SELECT SUM(amount) as total FROM payments WHERE user_id = 1 AND status = 'successful';

-- Get payment statistics by provider:
-- SELECT provider, COUNT(*) as total, SUM(amount) as total_amount 
-- FROM payments WHERE status = 'successful'  GROUP BY provider;
