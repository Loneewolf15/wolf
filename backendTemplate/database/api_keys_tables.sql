-- ============================================
-- API Keys Database Migration
-- ============================================

USE market_plaza_pid2025;

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    
    -- Key details
    key_name VARCHAR(100) NOT NULL,
    api_key VARCHAR(64) NOT NULL UNIQUE COMMENT 'The actual API key',
    key_prefix VARCHAR(10) NOT NULL COMMENT 'First 8 chars for display (pk_xxxxxx)',
    
    -- Permissions & scopes
    scopes TEXT NULL COMMENT 'JSON array of allowed scopes',
    permissions JSON NULL COMMENT 'Specific permissions',
    
    -- Rate limiting
    rate_limit_per_minute INT DEFAULT 60,
    rate_limit_per_hour INT DEFAULT 1000,
    rate_limit_per_day INT DEFAULT 10000,
    
    -- Status & expiry
    is_active TINYINT(1) DEFAULT 1,
    expires_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    
    -- Metadata
    description TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_api_key (api_key),
    INDEX idx_prefix (key_prefix),
    INDEX idx_active (is_active),
    INDEX idx_expires (expires_at),
    
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Key usage logs
CREATE TABLE IF NOT EXISTS api_key_usage (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    api_key_id BIGINT UNSIGNED NOT NULL,
    
    -- Request details
    endpoint VARCHAR(255) NOT NULL,
    http_method VARCHAR(10) NOT NULL,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    
    -- Response
    status_code INT NULL,
    response_time_ms DECIMAL(10, 2) NULL,
    
    -- Timestamp
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_api_key_id (api_key_id),
    INDEX idx_created_at (created_at),
    INDEX idx_endpoint (endpoint),
    
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Key rate limit tracking
CREATE TABLE IF NOT EXISTS api_key_rate_limits (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    api_key_id BIGINT UNSIGNED NOT NULL,
    
    -- Time windows
    minute_window TIMESTAMP NOT NULL,
    hour_window TIMESTAMP NOT NULL,
    day_window DATE NOT NULL,
    
    -- Counters
    requests_this_minute INT DEFAULT 0,
    requests_this_hour INT DEFAULT 0,
    requests_this_day INT DEFAULT 0,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_windows (api_key_id, minute_window, hour_window, day_window),
    INDEX idx_api_key_id (api_key_id),
    
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SELECT 'API Keys tables created successfully!' AS message;

-- Sample scopes (documentation)
-- ['users:read', 'users:write', 'payments:read', 'payments:write', 'uploads:read', 'uploads:write']
