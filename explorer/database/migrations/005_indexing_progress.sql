-- Migration: Add indexing progress tracking tables
-- Version: 005
-- Description: Tables for tracking historical indexing progress and failed blocks

-- ============================================================================
-- INDEXING PROGRESS TRACKING
-- ============================================================================

-- Table to track overall indexing progress
CREATE TABLE IF NOT EXISTS indexing_progress (
    id INTEGER PRIMARY KEY DEFAULT 1,
    last_indexed_height BIGINT NOT NULL DEFAULT 0,
    total_blocks_indexed BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'idle',
    -- Status values: 'idle', 'indexing', 'complete', 'paused', 'failed'
    start_height BIGINT,
    target_height BIGINT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    error_message TEXT,
    -- Ensure only one row exists
    CONSTRAINT single_progress_row CHECK (id = 1)
);

-- Create indexes for indexing_progress
CREATE INDEX idx_indexing_progress_status ON indexing_progress(status);
CREATE INDEX idx_indexing_progress_updated_at ON indexing_progress(updated_at DESC);

-- Initialize with default row
INSERT INTO indexing_progress (id, last_indexed_height, total_blocks_indexed, status)
VALUES (1, 0, 0, 'idle')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- FAILED BLOCKS TRACKING
-- ============================================================================

-- Table to track blocks that failed to index
CREATE TABLE IF NOT EXISTS failed_blocks (
    height BIGINT PRIMARY KEY,
    error_message TEXT NOT NULL,
    error_type VARCHAR(64),
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    first_failed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_error_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB
);

-- Create indexes for failed_blocks
CREATE INDEX idx_failed_blocks_retry_count ON failed_blocks(retry_count);
CREATE INDEX idx_failed_blocks_last_retry_at ON failed_blocks(last_retry_at);
CREATE INDEX idx_failed_blocks_first_failed_at ON failed_blocks(first_failed_at DESC);
CREATE INDEX idx_failed_blocks_resolved ON failed_blocks(resolved) WHERE NOT resolved;
CREATE INDEX idx_failed_blocks_error_type ON failed_blocks(error_type);

-- ============================================================================
-- INDEXING PERFORMANCE METRICS
-- ============================================================================

-- Table to track indexing performance over time
CREATE TABLE IF NOT EXISTS indexing_metrics (
    id BIGSERIAL PRIMARY KEY,
    metric_name VARCHAR(64) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    start_height BIGINT,
    end_height BIGINT,
    blocks_processed BIGINT,
    duration_seconds DOUBLE PRECISION,
    blocks_per_second DOUBLE PRECISION,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Create indexes for indexing_metrics
CREATE INDEX idx_indexing_metrics_metric_name ON indexing_metrics(metric_name);
CREATE INDEX idx_indexing_metrics_timestamp ON indexing_metrics(timestamp DESC);
CREATE INDEX idx_indexing_metrics_name_timestamp ON indexing_metrics(metric_name, timestamp DESC);

-- ============================================================================
-- INDEXING CHECKPOINTS
-- ============================================================================

-- Table to store periodic checkpoints for resumable indexing
CREATE TABLE IF NOT EXISTS indexing_checkpoints (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT NOT NULL,
    block_hash VARCHAR(64),
    blocks_since_last_checkpoint INTEGER NOT NULL,
    time_since_last_checkpoint INTERVAL,
    avg_blocks_per_second DOUBLE PRECISION,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for indexing_checkpoints
CREATE INDEX idx_indexing_checkpoints_height ON indexing_checkpoints(height DESC);
CREATE INDEX idx_indexing_checkpoints_created_at ON indexing_checkpoints(created_at DESC);
CREATE INDEX idx_indexing_checkpoints_status ON indexing_checkpoints(status);

-- ============================================================================
-- STORED PROCEDURES AND FUNCTIONS
-- ============================================================================

-- Function to update indexing progress
CREATE OR REPLACE FUNCTION update_indexing_progress(
    p_height BIGINT,
    p_status VARCHAR(32)
)
RETURNS void AS $$
BEGIN
    UPDATE indexing_progress
    SET
        last_indexed_height = p_height,
        total_blocks_indexed = total_blocks_indexed + 1,
        status = p_status,
        updated_at = NOW(),
        completed_at = CASE
            WHEN p_status = 'complete' THEN NOW()
            ELSE completed_at
        END
    WHERE id = 1;

    -- If no row exists, insert one
    IF NOT FOUND THEN
        INSERT INTO indexing_progress (
            id, last_indexed_height, total_blocks_indexed, status
        ) VALUES (
            1, p_height, 1, p_status
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record failed block
CREATE OR REPLACE FUNCTION record_failed_block(
    p_height BIGINT,
    p_error_message TEXT,
    p_error_type VARCHAR(64) DEFAULT NULL
)
RETURNS void AS $$
BEGIN
    INSERT INTO failed_blocks (
        height,
        error_message,
        error_type,
        retry_count,
        first_failed_at,
        last_error_at
    ) VALUES (
        p_height,
        p_error_message,
        p_error_type,
        0,
        NOW(),
        NOW()
    )
    ON CONFLICT (height) DO UPDATE
    SET
        error_message = p_error_message,
        error_type = COALESCE(p_error_type, failed_blocks.error_type),
        retry_count = failed_blocks.retry_count + 1,
        last_retry_at = NOW(),
        last_error_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to mark failed block as resolved
CREATE OR REPLACE FUNCTION resolve_failed_block(p_height BIGINT)
RETURNS void AS $$
BEGIN
    UPDATE failed_blocks
    SET
        resolved = TRUE,
        resolved_at = NOW()
    WHERE height = p_height;
END;
$$ LANGUAGE plpgsql;

-- Function to create indexing checkpoint
CREATE OR REPLACE FUNCTION create_indexing_checkpoint(
    p_height BIGINT,
    p_block_hash VARCHAR(64),
    p_blocks_since_last INTEGER,
    p_time_since_last INTERVAL,
    p_blocks_per_second DOUBLE PRECISION,
    p_status VARCHAR(32)
)
RETURNS void AS $$
BEGIN
    INSERT INTO indexing_checkpoints (
        height,
        block_hash,
        blocks_since_last_checkpoint,
        time_since_last_checkpoint,
        avg_blocks_per_second,
        status
    ) VALUES (
        p_height,
        p_block_hash,
        p_blocks_since_last,
        p_time_since_last,
        p_blocks_per_second,
        p_status
    );
END;
$$ LANGUAGE plpgsql;

-- Function to record indexing metric
CREATE OR REPLACE FUNCTION record_indexing_metric(
    p_metric_name VARCHAR(64),
    p_metric_value DOUBLE PRECISION,
    p_start_height BIGINT DEFAULT NULL,
    p_end_height BIGINT DEFAULT NULL,
    p_blocks_processed BIGINT DEFAULT NULL,
    p_duration_seconds DOUBLE PRECISION DEFAULT NULL,
    p_blocks_per_second DOUBLE PRECISION DEFAULT NULL
)
RETURNS void AS $$
BEGIN
    INSERT INTO indexing_metrics (
        metric_name,
        metric_value,
        start_height,
        end_height,
        blocks_processed,
        duration_seconds,
        blocks_per_second
    ) VALUES (
        p_metric_name,
        p_metric_value,
        p_start_height,
        p_end_height,
        p_blocks_processed,
        p_duration_seconds,
        p_blocks_per_second
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get unresolved failed blocks
CREATE OR REPLACE FUNCTION get_unresolved_failed_blocks(
    p_max_retry_count INTEGER DEFAULT 5
)
RETURNS TABLE (
    height BIGINT,
    error_message TEXT,
    retry_count INTEGER,
    last_retry_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        fb.height,
        fb.error_message,
        fb.retry_count,
        fb.last_retry_at
    FROM failed_blocks fb
    WHERE fb.resolved = FALSE
      AND fb.retry_count < p_max_retry_count
    ORDER BY fb.retry_count ASC, fb.first_failed_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to clean old checkpoints (keep last 1000)
CREATE OR REPLACE FUNCTION cleanup_old_checkpoints()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH to_delete AS (
        SELECT id
        FROM indexing_checkpoints
        ORDER BY created_at DESC
        OFFSET 1000
    )
    DELETE FROM indexing_checkpoints
    WHERE id IN (SELECT id FROM to_delete);

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get indexing statistics
CREATE OR REPLACE FUNCTION get_indexing_statistics()
RETURNS TABLE (
    total_blocks_indexed BIGINT,
    last_indexed_height BIGINT,
    current_status VARCHAR(32),
    failed_blocks_count BIGINT,
    unresolved_failed_blocks BIGINT,
    avg_blocks_per_second DOUBLE PRECISION,
    estimated_completion_time TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_chain_height BIGINT;
    v_blocks_remaining BIGINT;
    v_avg_bps DOUBLE PRECISION;
    v_eta_seconds DOUBLE PRECISION;
BEGIN
    -- Get current chain height from latest block
    SELECT COALESCE(MAX(height), 0) INTO v_chain_height FROM blocks;

    RETURN QUERY
    SELECT
        ip.total_blocks_indexed,
        ip.last_indexed_height,
        ip.status,
        (SELECT COUNT(*) FROM failed_blocks)::BIGINT,
        (SELECT COUNT(*) FROM failed_blocks WHERE resolved = FALSE)::BIGINT,
        (
            SELECT AVG(blocks_per_second)
            FROM indexing_metrics
            WHERE metric_name = 'batch_performance'
              AND timestamp > NOW() - INTERVAL '1 hour'
        ),
        CASE
            WHEN ip.status = 'indexing' AND v_chain_height > ip.last_indexed_height THEN
                NOW() + (
                    ((v_chain_height - ip.last_indexed_height)::DOUBLE PRECISION /
                    NULLIF((
                        SELECT AVG(blocks_per_second)
                        FROM indexing_metrics
                        WHERE metric_name = 'batch_performance'
                          AND timestamp > NOW() - INTERVAL '1 hour'
                    ), 0)) * INTERVAL '1 second'
                )
            ELSE NULL
        END
    FROM indexing_progress ip
    WHERE ip.id = 1;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE indexing_progress IS 'Tracks overall historical block indexing progress';
COMMENT ON TABLE failed_blocks IS 'Records blocks that failed to index for retry';
COMMENT ON TABLE indexing_metrics IS 'Stores performance metrics for indexing operations';
COMMENT ON TABLE indexing_checkpoints IS 'Periodic checkpoints for resumable indexing';

COMMENT ON FUNCTION update_indexing_progress IS 'Updates the main indexing progress record';
COMMENT ON FUNCTION record_failed_block IS 'Records a block that failed to index';
COMMENT ON FUNCTION resolve_failed_block IS 'Marks a failed block as successfully resolved';
COMMENT ON FUNCTION create_indexing_checkpoint IS 'Creates a checkpoint for resumable indexing';
COMMENT ON FUNCTION record_indexing_metric IS 'Records an indexing performance metric';
COMMENT ON FUNCTION get_unresolved_failed_blocks IS 'Returns failed blocks that need retry';
COMMENT ON FUNCTION cleanup_old_checkpoints IS 'Removes old checkpoint records to save space';
COMMENT ON FUNCTION get_indexing_statistics IS 'Returns comprehensive indexing statistics';

-- ============================================================================
-- GRANT PERMISSIONS (adjust as needed for your setup)
-- ============================================================================

-- GRANT SELECT, INSERT, UPDATE ON indexing_progress TO explorer_indexer;
-- GRANT SELECT, INSERT, UPDATE ON failed_blocks TO explorer_indexer;
-- GRANT SELECT, INSERT ON indexing_metrics TO explorer_indexer;
-- GRANT SELECT, INSERT ON indexing_checkpoints TO explorer_indexer;
-- GRANT EXECUTE ON FUNCTION update_indexing_progress TO explorer_indexer;
-- GRANT EXECUTE ON FUNCTION record_failed_block TO explorer_indexer;
-- GRANT EXECUTE ON FUNCTION resolve_failed_block TO explorer_indexer;

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Ensure indexing_progress table has initial row
INSERT INTO indexing_progress (id, last_indexed_height, total_blocks_indexed, status)
VALUES (1, 0, 0, 'idle')
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- MAINTENANCE SCHEDULE (optional, for production use)
-- ============================================================================

-- Schedule periodic cleanup of old checkpoints
-- This should be run via cron or pg_cron extension
-- Example: SELECT cleanup_old_checkpoints();

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 005: Indexing progress tracking tables created successfully';
    RAISE NOTICE 'Tables: indexing_progress, failed_blocks, indexing_metrics, indexing_checkpoints';
    RAISE NOTICE 'Functions: 8 stored procedures for indexing management';
END $$;
