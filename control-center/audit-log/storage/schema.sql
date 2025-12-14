-- Audit log main table
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    event_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(255),
    user_email VARCHAR(255) NOT NULL,
    user_role VARCHAR(50),
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    resource_id VARCHAR(255),
    changes JSONB,
    previous_value JSONB,
    new_value JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    result VARCHAR(20) NOT NULL,
    error_message TEXT,
    severity VARCHAR(20) NOT NULL,
    metadata JSONB,
    hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_event_type ON audit_log(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_email ON audit_log(user_email);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource_id ON audit_log(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_result ON audit_log(result);
CREATE INDEX IF NOT EXISTS idx_audit_log_severity ON audit_log(severity);
CREATE INDEX IF NOT EXISTS idx_audit_log_session_id ON audit_log(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_hash ON audit_log(hash);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_audit_log_user_timestamp ON audit_log(user_email, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_event_timestamp ON audit_log(event_type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource_timestamp ON audit_log(resource, timestamp DESC);

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_audit_log_search ON audit_log USING gin(to_tsvector('english',
    coalesce(action, '') || ' ' ||
    coalesce(resource, '') || ' ' ||
    coalesce(error_message, '') || ' ' ||
    coalesce(metadata::text, '')
));

-- Partitioning by month for better performance (example for current year)
-- CREATE TABLE audit_log_2025_01 PARTITION OF audit_log
--     FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Archive table for old entries (older than retention period)
CREATE TABLE IF NOT EXISTS audit_log_archive (
    LIKE audit_log INCLUDING ALL
);

-- Integrity verification table
CREATE TABLE IF NOT EXISTS audit_integrity_checks (
    id SERIAL PRIMARY KEY,
    start_id UUID NOT NULL,
    end_id UUID NOT NULL,
    entries_checked BIGINT NOT NULL,
    verified BOOLEAN NOT NULL,
    errors TEXT[],
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Statistics materialized view (refreshed periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS audit_log_stats AS
SELECT
    COUNT(*) as total_events,
    COUNT(DISTINCT user_email) as unique_users,
    COUNT(CASE WHEN result = 'success' THEN 1 END) as success_count,
    COUNT(CASE WHEN result = 'failure' THEN 1 END) as failure_count,
    COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_count,
    DATE_TRUNC('hour', timestamp) as hour_bucket
FROM audit_log
GROUP BY hour_bucket;

CREATE INDEX IF NOT EXISTS idx_audit_stats_hour ON audit_log_stats(hour_bucket DESC);

-- Function to automatically update previous_hash
CREATE OR REPLACE FUNCTION update_audit_log_hash_chain()
RETURNS TRIGGER AS $$
DECLARE
    last_hash VARCHAR(64);
BEGIN
    -- Get the hash of the most recent entry
    SELECT hash INTO last_hash
    FROM audit_log
    ORDER BY timestamp DESC, created_at DESC
    LIMIT 1;

    -- Set the previous_hash for this new entry
    NEW.previous_hash := last_hash;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain hash chain
CREATE TRIGGER audit_log_hash_chain_trigger
    BEFORE INSERT ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION update_audit_log_hash_chain();

-- Function to archive old entries
CREATE OR REPLACE FUNCTION archive_old_audit_logs(retention_days INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    -- Move old entries to archive
    WITH archived AS (
        DELETE FROM audit_log
        WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL
        RETURNING *
    )
    INSERT INTO audit_log_archive
    SELECT * FROM archived;

    GET DIAGNOSTICS archived_count = ROW_COUNT;

    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Function to refresh statistics
CREATE OR REPLACE FUNCTION refresh_audit_stats()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY audit_log_stats;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON audit_log TO audit_reader;
-- GRANT INSERT ON audit_log TO audit_writer;
-- GRANT ALL ON audit_log TO audit_admin;

-- Comments for documentation
COMMENT ON TABLE audit_log IS 'Immutable audit trail for all administrative actions';
COMMENT ON COLUMN audit_log.hash IS 'SHA-256 hash of entry content for integrity verification';
COMMENT ON COLUMN audit_log.previous_hash IS 'Hash of previous entry, forming a hash chain';
COMMENT ON COLUMN audit_log.changes IS 'Detailed changes in JSON format';
COMMENT ON COLUMN audit_log.metadata IS 'Additional context-specific metadata';
