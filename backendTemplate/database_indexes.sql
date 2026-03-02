-- ============================================
-- Database Indexing Script for Users Table
-- ============================================
-- This script creates optimized indexes for the users table
-- to improve query performance by 10-1000x
--
-- Run this script with:
-- mysql -u root -p market_plaza_pid2025 < database_indexes.sql
--
-- Or execute from MySQL shell:
-- source /path/to/database_indexes.sql;
-- ============================================

USE market_plaza_pid2025;

-- Check if indexes already exist before creating
-- This prevents errors on re-runs

-- 1. Email Index (Most critical - used in login and registration)
-- Speeds up: WHERE email = ?
SELECT 'Creating index on email...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 2. Phone Index (Alternative login method)
-- Speeds up: WHERE phone = ?
SELECT 'Creating index on phone...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- 3. Referral Code Index (Referral system lookups)
-- Speeds up: WHERE referral_code = ?
SELECT 'Creating index on referral_code...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);

-- 4. Google ID Index (Social authentication)
-- Speeds up: WHERE google_id = ?
SELECT 'Creating index on google_id...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);

-- 5. Facebook ID Index (Social authentication)
-- Speeds up: WHERE facebook_id = ?
SELECT 'Creating index on facebook_id...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_facebook_id ON users(facebook_id);

-- 6. Apple ID Index (Social authentication)
-- Speeds up: WHERE apple_id = ?
SELECT 'Creating index on apple_id...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_apple_id ON users(apple_id);

-- 7. Composite Index for Active and Verified Users
-- Speeds up: WHERE is_active = 1 AND is_verified = 1
SELECT 'Creating composite index on is_active and is_verified...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_active_verified ON users(is_active, is_verified);

-- 8. Access Token Index (JWT validation - partial index due to size)
-- Speeds up: WHERE access_token = ?
-- Using prefix index (255 chars) for performance
SELECT 'Creating index on access_token (prefix)...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_access_token ON users(access_token(255));

-- 9. Created At Index (For user analytics and reporting)
-- Speeds up: WHERE created_at BETWEEN ? AND ?
SELECT 'Creating index on created_at...' AS '';
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- ============================================
-- Show all indexes on users table
-- ============================================
SELECT '\nCurrent indexes on users table:' AS '';
SHOW INDEX FROM users;

-- ============================================
-- Performance Analysis
-- ============================================
-- Run these queries to verify index usage:
--
-- 1. Check email lookup performance:
-- EXPLAIN SELECT * FROM users WHERE email = 'test@example.com';
-- Should show: type=ref, key=idx_users_email
--
-- 2. Check social login performance:
-- EXPLAIN SELECT * FROM users WHERE google_id = '12345';
-- Should show: type=ref, key=idx_users_google_id
--
-- 3. Check referral code lookup:
-- EXPLAIN SELECT * FROM users WHERE referral_code = 'ABC123';
-- Should show: type=ref, key=idx_users_referral_code
-- ============================================

SELECT '\nIndexes created successfully!' AS '';
SELECT 'Run EXPLAIN on your queries to verify index usage.' AS '';
