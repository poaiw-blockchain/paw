-- PAW Indexer - Staging Database Initialization
-- This script initializes the staging database schema

-- Create staging-specific extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create staging user if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'paw_staging') THEN
        CREATE USER paw_staging WITH PASSWORD 'staging_password_change_me';
    END IF;
END
$$;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE paw_indexer_staging TO paw_staging;

-- Create staging marker table
CREATE TABLE IF NOT EXISTS staging_metadata (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert staging environment marker
INSERT INTO staging_metadata (key, value) VALUES
    ('environment', 'staging'),
    ('initialized_at', CURRENT_TIMESTAMP::TEXT),
    ('warning', 'This is a staging database - not for production use')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = CURRENT_TIMESTAMP;

-- Note: Actual indexer tables will be created by the indexer service on startup
-- This script only creates staging-specific metadata
