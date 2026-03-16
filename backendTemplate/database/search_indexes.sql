-- ============================================
-- Search Service - Full-Text Indexes Migration
-- ============================================
-- Adds FULLTEXT indexes for search functionality
-- Run with: mysql -u root -p market_plaza_pid2025 < database/search_indexes.sql
-- ============================================

USE market_plaza_pid2025;

-- Add FULLTEXT index to users table for searching
ALTER TABLE users 
ADD FULLTEXT INDEX ft_users_search (name, email, bio);

-- Add FULLTEXT index to listings table (if exists)
-- Uncomment and adjust based on your schema
-- ALTER TABLE listings 
-- ADD FULLTEXT INDEX ft_listings_search (title, description, tags);

-- Create search_logs table for tracking searches
CREATE TABLE IF NOT EXISTS search_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NULL,
    query VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NULL COMMENT 'users, listings, etc.',
    results_count INT DEFAULT 0,
    execution_time_ms DECIMAL(10, 2) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_query (query),
    INDEX idx_created_at (created_at),
    
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create search_suggestions table for autocomplete
CREATE TABLE IF NOT EXISTS search_suggestions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    suggestion VARCHAR(255) NOT NULL UNIQUE,
    search_count INT DEFAULT 1,
    last_searched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_suggestion (suggestion),
    INDEX idx_count (search_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SELECT 'Search indexes and tables created successfully!' AS message;

-- Test FULLTEXT search
-- SELECT *, MATCH(name, email, bio) AGAINST ('john' IN NATURAL LANGUAGE MODE) as relevance
-- FROM users
-- WHERE MATCH(name, email, bio) AGAINST ('john' IN NATURAL LANGUAGE MODE)
-- ORDER BY relevance DESC;
