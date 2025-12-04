-- DEX Enhancements Migration
-- Adds missing tables for production-grade DEX analytics
-- Version: 1.1.0

-- ============================================================================
-- DEX POOL PRICE HISTORY (OHLCV data for charting)
-- ============================================================================

CREATE TABLE dex_pool_price_history (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    block_height BIGINT NOT NULL,

    -- OHLCV candlestick data
    open VARCHAR(64) NOT NULL,
    high VARCHAR(64) NOT NULL,
    low VARCHAR(64) NOT NULL,
    close VARCHAR(64) NOT NULL,
    volume VARCHAR(256) NOT NULL,

    -- Additional metrics
    liquidity_a VARCHAR(256) NOT NULL,
    liquidity_b VARCHAR(256) NOT NULL,
    price_a_to_b VARCHAR(64) NOT NULL,
    price_b_to_a VARCHAR(64) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Unique constraint to prevent duplicate data points
    UNIQUE(pool_id, timestamp)
);

CREATE INDEX idx_dex_price_history_pool_id ON dex_pool_price_history(pool_id);
CREATE INDEX idx_dex_price_history_timestamp ON dex_pool_price_history(timestamp DESC);
CREATE INDEX idx_dex_price_history_pool_timestamp ON dex_pool_price_history(pool_id, timestamp DESC);
CREATE INDEX idx_dex_price_history_block_height ON dex_pool_price_history(block_height DESC);

COMMENT ON TABLE dex_pool_price_history IS 'OHLCV price history for DEX pool charting';

-- ============================================================================
-- DEX POOL STATISTICS (Aggregated metrics by time period)
-- ============================================================================

CREATE TABLE dex_pool_statistics (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    period VARCHAR(16) NOT NULL, -- '1h', '24h', '7d', '30d'
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Volume metrics
    volume_token_a VARCHAR(256) DEFAULT '0',
    volume_token_b VARCHAR(256) DEFAULT '0',
    volume_usd VARCHAR(256) DEFAULT '0',
    trade_count INTEGER DEFAULT 0,

    -- Liquidity metrics
    avg_liquidity_a VARCHAR(256) DEFAULT '0',
    avg_liquidity_b VARCHAR(256) DEFAULT '0',
    min_liquidity VARCHAR(256) DEFAULT '0',
    max_liquidity VARCHAR(256) DEFAULT '0',

    -- Fee metrics
    fees_collected_a VARCHAR(256) DEFAULT '0',
    fees_collected_b VARCHAR(256) DEFAULT '0',
    fees_usd VARCHAR(256) DEFAULT '0',

    -- Performance metrics
    avg_price VARCHAR(64) DEFAULT '0',
    high_price VARCHAR(64) DEFAULT '0',
    low_price VARCHAR(64) DEFAULT '0',
    price_change_percent VARCHAR(16) DEFAULT '0',

    -- APR calculation
    apr VARCHAR(16) DEFAULT '0',

    -- User metrics
    unique_traders INTEGER DEFAULT 0,
    unique_liquidity_providers INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(pool_id, period, period_start)
);

CREATE INDEX idx_dex_statistics_pool_id ON dex_pool_statistics(pool_id);
CREATE INDEX idx_dex_statistics_period ON dex_pool_statistics(period);
CREATE INDEX idx_dex_statistics_period_start ON dex_pool_statistics(period_start DESC);
CREATE INDEX idx_dex_statistics_pool_period ON dex_pool_statistics(pool_id, period, period_start DESC);
CREATE INDEX idx_dex_statistics_volume ON dex_pool_statistics(volume_usd DESC);
CREATE INDEX idx_dex_statistics_apr ON dex_pool_statistics(apr DESC);

COMMENT ON TABLE dex_pool_statistics IS 'Aggregated pool statistics by time period';

-- ============================================================================
-- DEX USER POSITIONS (Track LP positions per user)
-- ============================================================================

CREATE TABLE dex_user_positions (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(128) NOT NULL,
    pool_id VARCHAR(64) NOT NULL,

    -- Position details
    shares VARCHAR(256) NOT NULL,
    initial_amount_a VARCHAR(256) NOT NULL,
    initial_amount_b VARCHAR(256) NOT NULL,
    current_amount_a VARCHAR(256) NOT NULL,
    current_amount_b VARCHAR(256) NOT NULL,

    -- Entry information
    entry_price VARCHAR(64) NOT NULL,
    entry_height BIGINT NOT NULL,
    entry_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    entry_tx_hash VARCHAR(64) NOT NULL,

    -- Exit information (NULL if position still active)
    exit_price VARCHAR(64),
    exit_height BIGINT,
    exit_timestamp TIMESTAMP WITH TIME ZONE,
    exit_tx_hash VARCHAR(64),

    -- Performance metrics
    fees_earned_a VARCHAR(256) DEFAULT '0',
    fees_earned_b VARCHAR(256) DEFAULT '0',
    fees_earned_usd VARCHAR(256) DEFAULT '0',
    impermanent_loss VARCHAR(64) DEFAULT '0',
    total_return_percent VARCHAR(16) DEFAULT '0',

    -- Status
    status VARCHAR(16) NOT NULL DEFAULT 'active', -- 'active', 'closed'

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(address, pool_id, entry_tx_hash)
);

CREATE INDEX idx_dex_user_positions_address ON dex_user_positions(address);
CREATE INDEX idx_dex_user_positions_pool_id ON dex_user_positions(pool_id);
CREATE INDEX idx_dex_user_positions_status ON dex_user_positions(status);
CREATE INDEX idx_dex_user_positions_entry_timestamp ON dex_user_positions(entry_timestamp DESC);
CREATE INDEX idx_dex_user_positions_exit_timestamp ON dex_user_positions(exit_timestamp DESC);
CREATE INDEX idx_dex_user_positions_address_status ON dex_user_positions(address, status);
CREATE INDEX idx_dex_user_positions_pool_status ON dex_user_positions(pool_id, status);

COMMENT ON TABLE dex_user_positions IS 'Individual LP positions with P&L tracking';

-- ============================================================================
-- DEX ANALYTICS CACHE (Performance optimization)
-- ============================================================================

CREATE TABLE dex_analytics_cache (
    id BIGSERIAL PRIMARY KEY,
    cache_key VARCHAR(256) NOT NULL UNIQUE,
    cache_type VARCHAR(64) NOT NULL, -- 'pool_summary', 'user_analytics', 'top_pools', etc.
    data JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dex_analytics_cache_key ON dex_analytics_cache(cache_key);
CREATE INDEX idx_dex_analytics_cache_type ON dex_analytics_cache(cache_type);
CREATE INDEX idx_dex_analytics_cache_expires ON dex_analytics_cache(expires_at);

COMMENT ON TABLE dex_analytics_cache IS 'Cache for expensive DEX analytics queries';

-- ============================================================================
-- MATERIALIZED VIEWS FOR TOP POOLS (Fast access)
-- ============================================================================

CREATE MATERIALIZED VIEW mv_top_dex_pools_enhanced AS
SELECT
    p.pool_id,
    p.token_a,
    p.token_b,
    p.reserve_a,
    p.reserve_b,
    p.total_shares,
    p.swap_fee,
    p.tvl,
    p.apr,
    p.volume_24h,
    p.volume_7d,

    -- Recent statistics
    COALESCE(s24.trade_count, 0) as trades_24h,
    COALESCE(s24.unique_traders, 0) as traders_24h,
    COALESCE(s24.price_change_percent, '0') as price_change_24h,

    -- Liquidity provider count
    (SELECT COUNT(DISTINCT address) FROM dex_user_positions WHERE pool_id = p.pool_id AND status = 'active') as active_lps,

    p.created_at,
    p.updated_at
FROM dex_pools p
LEFT JOIN LATERAL (
    SELECT * FROM dex_pool_statistics
    WHERE pool_id = p.pool_id AND period = '24h'
    ORDER BY period_start DESC
    LIMIT 1
) s24 ON true
ORDER BY CAST(p.tvl AS NUMERIC) DESC;

CREATE UNIQUE INDEX idx_mv_top_dex_pools_enhanced_pool_id ON mv_top_dex_pools_enhanced(pool_id);
CREATE INDEX idx_mv_top_dex_pools_enhanced_tvl ON mv_top_dex_pools_enhanced(tvl DESC);
CREATE INDEX idx_mv_top_dex_pools_enhanced_volume ON mv_top_dex_pools_enhanced(volume_24h DESC);

-- ============================================================================
-- FUNCTIONS FOR DEX ANALYTICS
-- ============================================================================

-- Function to calculate impermanent loss
CREATE OR REPLACE FUNCTION calculate_impermanent_loss(
    entry_price VARCHAR,
    current_price VARCHAR
) RETURNS VARCHAR AS $$
DECLARE
    entry_p NUMERIC;
    current_p NUMERIC;
    price_ratio NUMERIC;
    il_ratio NUMERIC;
    il_percent VARCHAR;
BEGIN
    entry_p := CAST(entry_price AS NUMERIC);
    current_p := CAST(current_price AS NUMERIC);

    IF entry_p = 0 OR current_p = 0 THEN
        RETURN '0';
    END IF;

    price_ratio := current_p / entry_p;
    il_ratio := (2 * SQRT(price_ratio)) / (1 + price_ratio) - 1;
    il_percent := CAST((il_ratio * 100) AS VARCHAR);

    RETURN il_percent;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to update user position metrics
CREATE OR REPLACE FUNCTION update_user_position_metrics()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate impermanent loss if position is still active
    IF NEW.status = 'active' AND NEW.entry_price != '0' THEN
        -- Get current pool price
        DECLARE
            current_price VARCHAR;
        BEGIN
            SELECT
                CAST(CAST(reserve_b AS NUMERIC) / CAST(reserve_a AS NUMERIC) AS VARCHAR)
            INTO current_price
            FROM dex_pools
            WHERE pool_id = NEW.pool_id;

            IF current_price IS NOT NULL THEN
                NEW.impermanent_loss := calculate_impermanent_loss(NEW.entry_price, current_price);
            END IF;
        END;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_position_metrics
BEFORE INSERT OR UPDATE ON dex_user_positions
FOR EACH ROW
EXECUTE FUNCTION update_user_position_metrics();

-- Function to refresh all materialized views
CREATE OR REPLACE FUNCTION refresh_dex_materialized_views()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_top_dex_pools_enhanced;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_dex_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM dex_analytics_cache
    WHERE expires_at < NOW();

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS AND DOCUMENTATION
-- ============================================================================

COMMENT ON COLUMN dex_pool_price_history.open IS 'Opening price for the time period';
COMMENT ON COLUMN dex_pool_price_history.high IS 'Highest price during the period';
COMMENT ON COLUMN dex_pool_price_history.low IS 'Lowest price during the period';
COMMENT ON COLUMN dex_pool_price_history.close IS 'Closing price for the period';
COMMENT ON COLUMN dex_pool_price_history.volume IS 'Trading volume during the period';

COMMENT ON COLUMN dex_user_positions.impermanent_loss IS 'Impermanent loss percentage (negative = loss)';
COMMENT ON COLUMN dex_user_positions.total_return_percent IS 'Total return including fees (can offset IL)';

-- ============================================================================
-- INITIAL DATA AND SETUP
-- ============================================================================

-- Run initial materialized view refresh
SELECT refresh_dex_materialized_views();

-- Analyze new tables for query optimization
ANALYZE dex_pool_price_history;
ANALYZE dex_pool_statistics;
ANALYZE dex_user_positions;
ANALYZE dex_analytics_cache;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'DEX enhancements migration completed successfully!';
    RAISE NOTICE 'New tables: dex_pool_price_history, dex_pool_statistics, dex_user_positions, dex_analytics_cache';
    RAISE NOTICE 'Enhanced materialized view: mv_top_dex_pools_enhanced';
END $$;
