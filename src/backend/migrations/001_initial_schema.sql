-- Migration: 001_initial_schema
-- Description: Create initial database schema for ISP Health Checker
-- Created: 2024-01-01

BEGIN;

-- ============================================================================
-- Users Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- ============================================================================
-- API Keys Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- ============================================================================
-- Runs Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS runs (
    id SERIAL PRIMARY KEY,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    target VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    score FLOAT NOT NULL,
    summary TEXT,
    report JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Probes Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS probes (
    id SERIAL PRIMARY KEY,
    run_db_id INTEGER REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    latency_ms FLOAT,
    details JSONB,
    error TEXT
);

-- ============================================================================
-- Indexes
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_runs_user_id ON runs(user_id);
CREATE INDEX IF NOT EXISTS idx_runs_timestamp ON runs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_runs_target ON runs(target);
CREATE INDEX IF NOT EXISTS idx_runs_run_id ON runs(run_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
CREATE INDEX IF NOT EXISTS idx_probes_run_db_id ON probes(run_db_id);

-- ============================================================================
-- Default Admin User (for development only - password: admin123)
-- In production, use environment variables or secure key management
-- ============================================================================
INSERT INTO users (username, email, password_hash, is_active)
VALUES (
    'admin',
    'admin@localhost',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.S5/N0OpfVBr5Ny',  -- bcrypt hash of 'admin123'
    TRUE
) ON CONFLICT (username) DO NOTHING;

-- Create a default API key for the admin user
INSERT INTO api_keys (key, user_id, name, is_active)
SELECT
    'isp-checker-dev-key-12345',
    id,
    'Default Development Key',
    TRUE
FROM users WHERE username = 'admin'
ON CONFLICT (key) DO NOTHING;

COMMIT;
