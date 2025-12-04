-- PAW Chain Explorer Database Schema
-- Production-ready PostgreSQL schema with optimizations
-- Version: 1.0.0

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For full-text search
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For GIN indexes
CREATE EXTENSION IF NOT EXISTS "btree_gist"; -- For GIST indexes

-- Drop existing tables (for clean setup)
DROP TABLE IF EXISTS compute_verifications CASCADE;
DROP TABLE IF EXISTS compute_results CASCADE;
DROP TABLE IF EXISTS compute_requests CASCADE;
DROP TABLE IF EXISTS oracle_slashes CASCADE;
DROP TABLE IF EXISTS oracle_submissions CASCADE;
DROP TABLE IF EXISTS oracle_prices CASCADE;
DROP TABLE IF EXISTS dex_trades CASCADE;
DROP TABLE IF EXISTS dex_liquidity CASCADE;
DROP TABLE IF EXISTS dex_pools CASCADE;
DROP TABLE IF EXISTS validator_uptime CASCADE;
DROP TABLE IF EXISTS validator_rewards CASCADE;
DROP TABLE IF EXISTS validators CASCADE;
DROP TABLE IF EXISTS account_balances CASCADE;
DROP TABLE IF EXISTS account_tokens CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS blocks CASCADE;
DROP TABLE IF EXISTS network_stats CASCADE;
DROP TABLE IF EXISTS search_index CASCADE;

-- ============================================================================
-- CORE BLOCKCHAIN TABLES
-- ============================================================================

-- Blocks table: stores all blockchain blocks
CREATE TABLE blocks (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT NOT NULL UNIQUE,
    hash VARCHAR(64) NOT NULL UNIQUE,
    chain_id VARCHAR(64) NOT NULL,
    proposer_address VARCHAR(128) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_count INTEGER DEFAULT 0,
    gas_used BIGINT DEFAULT 0,
    gas_wanted BIGINT DEFAULT 0,
    evidence_count INTEGER DEFAULT 0,
    evidence JSONB,
    signatures JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_blocks_height ON blocks(height DESC);
CREATE INDEX idx_blocks_timestamp ON blocks(timestamp DESC);
CREATE INDEX idx_blocks_proposer ON blocks(proposer_address);
CREATE INDEX idx_blocks_chain_id ON blocks(chain_id);
CREATE INDEX idx_blocks_created_at ON blocks(created_at DESC);

-- Transactions table: stores all transactions
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    block_height BIGINT NOT NULL REFERENCES blocks(height) ON DELETE CASCADE,
    tx_index INTEGER NOT NULL,
    type VARCHAR(256),
    sender VARCHAR(128),
    status VARCHAR(20) NOT NULL, -- 'success', 'failed'
    code INTEGER DEFAULT 0,
    gas_used BIGINT DEFAULT 0,
    gas_wanted BIGINT DEFAULT 0,
    fee_amount VARCHAR(256),
    fee_denom VARCHAR(64),
    memo TEXT,
    raw_log TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    messages JSONB,
    events JSONB,
    signatures JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(block_height, tx_index)
);

CREATE INDEX idx_transactions_hash ON transactions(hash);
CREATE INDEX idx_transactions_block_height ON transactions(block_height DESC);
CREATE INDEX idx_transactions_sender ON transactions(sender);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);
CREATE INDEX idx_transactions_sender_timestamp ON transactions(sender, timestamp DESC);
CREATE INDEX idx_transactions_type_timestamp ON transactions(type, timestamp DESC);
CREATE INDEX idx_transactions_messages_gin ON transactions USING gin(messages jsonb_path_ops);

-- Events table: stores transaction events
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(64) NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE,
    block_height BIGINT NOT NULL,
    event_index INTEGER NOT NULL,
    type VARCHAR(256) NOT NULL,
    module VARCHAR(64),
    attributes JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tx_hash, event_index)
);

CREATE INDEX idx_events_tx_hash ON events(tx_hash);
CREATE INDEX idx_events_block_height ON events(block_height DESC);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_module ON events(module);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_attributes_gin ON events USING gin(attributes jsonb_path_ops);

-- ============================================================================
-- ACCOUNTS AND BALANCES
-- ============================================================================

-- Accounts table: stores account information
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(128) NOT NULL UNIQUE,
    first_seen_height BIGINT NOT NULL,
    last_seen_height BIGINT NOT NULL,
    tx_count INTEGER DEFAULT 0,
    total_received VARCHAR(256) DEFAULT '0',
    total_sent VARCHAR(256) DEFAULT '0',
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_accounts_address ON accounts(address);
CREATE INDEX idx_accounts_tx_count ON accounts(tx_count DESC);
CREATE INDEX idx_accounts_first_seen ON accounts(first_seen_at DESC);
CREATE INDEX idx_accounts_last_seen ON accounts(last_seen_at DESC);

-- Account balances table: stores current balances
CREATE TABLE account_balances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(128) NOT NULL,
    denom VARCHAR(64) NOT NULL,
    amount VARCHAR(256) NOT NULL,
    last_updated_height BIGINT NOT NULL,
    last_updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(address, denom)
);

CREATE INDEX idx_account_balances_address ON account_balances(address);
CREATE INDEX idx_account_balances_denom ON account_balances(denom);
CREATE INDEX idx_account_balances_amount ON account_balances(amount DESC);

-- Account tokens table: IBC and custom tokens
CREATE TABLE account_tokens (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(128) NOT NULL,
    token_denom VARCHAR(256) NOT NULL,
    token_name VARCHAR(128),
    token_symbol VARCHAR(32),
    amount VARCHAR(256) NOT NULL,
    ibc_trace JSONB,
    last_updated_height BIGINT NOT NULL,
    last_updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(address, token_denom)
);

CREATE INDEX idx_account_tokens_address ON account_tokens(address);
CREATE INDEX idx_account_tokens_denom ON account_tokens(token_denom);
CREATE INDEX idx_account_tokens_symbol ON account_tokens(token_symbol);

-- ============================================================================
-- VALIDATORS
-- ============================================================================

-- Validators table: stores validator information
CREATE TABLE validators (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(128) NOT NULL UNIQUE,
    consensus_address VARCHAR(128),
    consensus_pubkey VARCHAR(256),
    operator_address VARCHAR(128),
    moniker VARCHAR(256),
    identity VARCHAR(128),
    website VARCHAR(512),
    security_contact VARCHAR(256),
    details TEXT,
    voting_power BIGINT DEFAULT 0,
    commission_rate VARCHAR(64),
    commission_max_rate VARCHAR(64),
    commission_max_change_rate VARCHAR(64),
    min_self_delegation VARCHAR(64),
    jailed BOOLEAN DEFAULT FALSE,
    status VARCHAR(32), -- 'bonded', 'unbonding', 'unbonded'
    tokens VARCHAR(256) DEFAULT '0',
    delegator_shares VARCHAR(256) DEFAULT '0',
    unbonding_height BIGINT,
    unbonding_time TIMESTAMP WITH TIME ZONE,
    updated_height BIGINT NOT NULL,
    updated_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_validators_address ON validators(address);
CREATE INDEX idx_validators_consensus ON validators(consensus_address);
CREATE INDEX idx_validators_operator ON validators(operator_address);
CREATE INDEX idx_validators_moniker ON validators(moniker);
CREATE INDEX idx_validators_voting_power ON validators(voting_power DESC);
CREATE INDEX idx_validators_status ON validators(status);
CREATE INDEX idx_validators_jailed ON validators(jailed);

-- Validator rewards table: tracks validator rewards
CREATE TABLE validator_rewards (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(128) NOT NULL,
    height BIGINT NOT NULL,
    amount VARCHAR(256) NOT NULL,
    denom VARCHAR(64) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_validator_rewards_address ON validator_rewards(validator_address);
CREATE INDEX idx_validator_rewards_height ON validator_rewards(height DESC);
CREATE INDEX idx_validator_rewards_timestamp ON validator_rewards(timestamp DESC);

-- Validator uptime table: tracks validator signing activity
CREATE TABLE validator_uptime (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(128) NOT NULL,
    height BIGINT NOT NULL,
    signed BOOLEAN NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(validator_address, height)
);

CREATE INDEX idx_validator_uptime_address ON validator_uptime(validator_address);
CREATE INDEX idx_validator_uptime_height ON validator_uptime(height DESC);
CREATE INDEX idx_validator_uptime_signed ON validator_uptime(signed);

-- ============================================================================
-- DEX MODULE TABLES
-- ============================================================================

-- DEX Pools table: stores liquidity pools
CREATE TABLE dex_pools (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL UNIQUE,
    token_a VARCHAR(64) NOT NULL,
    token_b VARCHAR(64) NOT NULL,
    reserve_a VARCHAR(256) DEFAULT '0',
    reserve_b VARCHAR(256) DEFAULT '0',
    total_shares VARCHAR(256) DEFAULT '0',
    creator VARCHAR(128),
    swap_fee VARCHAR(64),
    protocol_fee VARCHAR(64),
    volume_24h VARCHAR(256) DEFAULT '0',
    volume_7d VARCHAR(256) DEFAULT '0',
    volume_30d VARCHAR(256) DEFAULT '0',
    tvl VARCHAR(256) DEFAULT '0',
    apr VARCHAR(64) DEFAULT '0',
    block_height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dex_pools_pool_id ON dex_pools(pool_id);
CREATE INDEX idx_dex_pools_tokens ON dex_pools(token_a, token_b);
CREATE INDEX idx_dex_pools_creator ON dex_pools(creator);
CREATE INDEX idx_dex_pools_tvl ON dex_pools(tvl DESC);
CREATE INDEX idx_dex_pools_volume_24h ON dex_pools(volume_24h DESC);

-- DEX Liquidity table: tracks liquidity additions/removals
CREATE TABLE dex_liquidity (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    provider VARCHAR(128) NOT NULL,
    action VARCHAR(16) NOT NULL, -- 'add', 'remove'
    amount_a VARCHAR(256) NOT NULL,
    amount_b VARCHAR(256) NOT NULL,
    shares VARCHAR(256) NOT NULL,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dex_liquidity_pool_id ON dex_liquidity(pool_id);
CREATE INDEX idx_dex_liquidity_provider ON dex_liquidity(provider);
CREATE INDEX idx_dex_liquidity_action ON dex_liquidity(action);
CREATE INDEX idx_dex_liquidity_timestamp ON dex_liquidity(timestamp DESC);
CREATE INDEX idx_dex_liquidity_tx_hash ON dex_liquidity(tx_hash);

-- DEX Trades table: stores all swaps/trades
CREATE TABLE dex_trades (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    trader VARCHAR(128) NOT NULL,
    token_in VARCHAR(64) NOT NULL,
    token_out VARCHAR(64) NOT NULL,
    amount_in VARCHAR(256) NOT NULL,
    amount_out VARCHAR(256) NOT NULL,
    price VARCHAR(64) NOT NULL,
    fee VARCHAR(256) NOT NULL,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_dex_trades_pool_id ON dex_trades(pool_id);
CREATE INDEX idx_dex_trades_trader ON dex_trades(trader);
CREATE INDEX idx_dex_trades_tokens ON dex_trades(token_in, token_out);
CREATE INDEX idx_dex_trades_timestamp ON dex_trades(timestamp DESC);
CREATE INDEX idx_dex_trades_tx_hash ON dex_trades(tx_hash);
CREATE INDEX idx_dex_trades_pool_timestamp ON dex_trades(pool_id, timestamp DESC);

-- ============================================================================
-- ORACLE MODULE TABLES
-- ============================================================================

-- Oracle Prices table: stores aggregated oracle prices
CREATE TABLE oracle_prices (
    id BIGSERIAL PRIMARY KEY,
    asset VARCHAR(64) NOT NULL,
    price VARCHAR(64) NOT NULL,
    median VARCHAR(64),
    average VARCHAR(64),
    std_deviation VARCHAR(64),
    num_validators INTEGER,
    num_submissions INTEGER,
    confidence_score VARCHAR(64),
    block_height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_oracle_prices_asset ON oracle_prices(asset);
CREATE INDEX idx_oracle_prices_timestamp ON oracle_prices(timestamp DESC);
CREATE INDEX idx_oracle_prices_block_height ON oracle_prices(block_height DESC);
CREATE INDEX idx_oracle_prices_asset_timestamp ON oracle_prices(asset, timestamp DESC);

-- Oracle Submissions table: individual price submissions
CREATE TABLE oracle_submissions (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(128) NOT NULL,
    feeder_address VARCHAR(128),
    asset VARCHAR(64) NOT NULL,
    price VARCHAR(64) NOT NULL,
    power VARCHAR(64),
    outlier BOOLEAN DEFAULT FALSE,
    deviation VARCHAR(64),
    tx_hash VARCHAR(64),
    block_height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_oracle_submissions_validator ON oracle_submissions(validator_address);
CREATE INDEX idx_oracle_submissions_asset ON oracle_submissions(asset);
CREATE INDEX idx_oracle_submissions_timestamp ON oracle_submissions(timestamp DESC);
CREATE INDEX idx_oracle_submissions_outlier ON oracle_submissions(outlier);
CREATE INDEX idx_oracle_submissions_tx_hash ON oracle_submissions(tx_hash);

-- Oracle Slashes table: records oracle slashing events
CREATE TABLE oracle_slashes (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(128) NOT NULL,
    reason VARCHAR(256) NOT NULL,
    slash_fraction VARCHAR(64),
    slash_amount VARCHAR(256),
    asset VARCHAR(64),
    severity VARCHAR(32), -- 'low', 'medium', 'high'
    block_height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_oracle_slashes_validator ON oracle_slashes(validator_address);
CREATE INDEX idx_oracle_slashes_timestamp ON oracle_slashes(timestamp DESC);
CREATE INDEX idx_oracle_slashes_severity ON oracle_slashes(severity);
CREATE INDEX idx_oracle_slashes_asset ON oracle_slashes(asset);

-- ============================================================================
-- COMPUTE MODULE TABLES
-- ============================================================================

-- Compute Requests table: stores computation requests
CREATE TABLE compute_requests (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    requester VARCHAR(128) NOT NULL,
    program_hash VARCHAR(64) NOT NULL,
    input_data_hash VARCHAR(64),
    reward VARCHAR(256) NOT NULL,
    timeout_height BIGINT,
    status VARCHAR(32) NOT NULL, -- 'pending', 'assigned', 'submitted', 'verified', 'completed', 'failed'
    provider VARCHAR(128),
    result_hash VARCHAR(64),
    verification_score VARCHAR(64),
    verified BOOLEAN DEFAULT FALSE,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_compute_requests_request_id ON compute_requests(request_id);
CREATE INDEX idx_compute_requests_requester ON compute_requests(requester);
CREATE INDEX idx_compute_requests_provider ON compute_requests(provider);
CREATE INDEX idx_compute_requests_status ON compute_requests(status);
CREATE INDEX idx_compute_requests_program_hash ON compute_requests(program_hash);
CREATE INDEX idx_compute_requests_created_at ON compute_requests(created_at DESC);
CREATE INDEX idx_compute_requests_tx_hash ON compute_requests(tx_hash);

-- Compute Results table: stores computation results
CREATE TABLE compute_results (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES compute_requests(request_id) ON DELETE CASCADE,
    provider VARCHAR(128) NOT NULL,
    result_hash VARCHAR(64) NOT NULL,
    result_data TEXT,
    execution_time INTEGER, -- milliseconds
    gas_used BIGINT,
    status VARCHAR(32) NOT NULL,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_compute_results_request_id ON compute_results(request_id);
CREATE INDEX idx_compute_results_provider ON compute_results(provider);
CREATE INDEX idx_compute_results_status ON compute_results(status);
CREATE INDEX idx_compute_results_submitted_at ON compute_results(submitted_at DESC);
CREATE INDEX idx_compute_results_tx_hash ON compute_results(tx_hash);

-- Compute Verifications table: stores verification proofs
CREATE TABLE compute_verifications (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL REFERENCES compute_requests(request_id) ON DELETE CASCADE,
    verifier VARCHAR(128),
    verification_score VARCHAR(64) NOT NULL,
    verified BOOLEAN NOT NULL,
    proof_data JSONB,
    signature VARCHAR(256),
    merkle_root VARCHAR(64),
    tx_hash VARCHAR(64),
    block_height BIGINT NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_compute_verifications_request_id ON compute_verifications(request_id);
CREATE INDEX idx_compute_verifications_verified ON compute_verifications(verified);
CREATE INDEX idx_compute_verifications_verifier ON compute_verifications(verifier);
CREATE INDEX idx_compute_verifications_verified_at ON compute_verifications(verified_at DESC);

-- ============================================================================
-- NETWORK STATISTICS
-- ============================================================================

-- Network Stats table: aggregated network statistics
CREATE TABLE network_stats (
    id BIGSERIAL PRIMARY KEY,
    metric_name VARCHAR(128) NOT NULL,
    metric_value VARCHAR(256) NOT NULL,
    metric_type VARCHAR(32) NOT NULL, -- 'counter', 'gauge', 'histogram'
    category VARCHAR(64), -- 'blockchain', 'dex', 'oracle', 'compute', 'validators'
    period VARCHAR(32), -- '1h', '24h', '7d', '30d', 'all'
    block_height BIGINT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(metric_name, period, timestamp)
);

CREATE INDEX idx_network_stats_metric_name ON network_stats(metric_name);
CREATE INDEX idx_network_stats_category ON network_stats(category);
CREATE INDEX idx_network_stats_period ON network_stats(period);
CREATE INDEX idx_network_stats_timestamp ON network_stats(timestamp DESC);

-- ============================================================================
-- SEARCH INDEX
-- ============================================================================

-- Search Index table: unified search across all entities
CREATE TABLE search_index (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(32) NOT NULL, -- 'block', 'transaction', 'address', 'validator', 'pool'
    entity_id VARCHAR(256) NOT NULL,
    search_text TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(entity_type, entity_id)
);

CREATE INDEX idx_search_index_entity_type ON search_index(entity_type);
CREATE INDEX idx_search_index_entity_id ON search_index(entity_id);
CREATE INDEX idx_search_index_text ON search_index USING gin(to_tsvector('english', search_text));
CREATE INDEX idx_search_index_metadata ON search_index USING gin(metadata jsonb_path_ops);

-- ============================================================================
-- MATERIALIZED VIEWS FOR PERFORMANCE
-- ============================================================================

-- Latest blocks view
CREATE MATERIALIZED VIEW mv_latest_blocks AS
SELECT
    height,
    hash,
    proposer_address,
    timestamp,
    tx_count,
    gas_used,
    gas_wanted
FROM blocks
ORDER BY height DESC
LIMIT 100;

CREATE UNIQUE INDEX idx_mv_latest_blocks_height ON mv_latest_blocks(height DESC);

-- Latest transactions view
CREATE MATERIALIZED VIEW mv_latest_transactions AS
SELECT
    t.hash,
    t.block_height,
    t.type,
    t.sender,
    t.status,
    t.timestamp,
    t.gas_used,
    t.fee_amount,
    t.fee_denom
FROM transactions t
ORDER BY t.timestamp DESC
LIMIT 1000;

CREATE UNIQUE INDEX idx_mv_latest_transactions_hash ON mv_latest_transactions(hash);

-- Top validators view
CREATE MATERIALIZED VIEW mv_top_validators AS
SELECT
    address,
    moniker,
    voting_power,
    commission_rate,
    status,
    jailed,
    updated_time
FROM validators
ORDER BY voting_power DESC
LIMIT 200;

CREATE UNIQUE INDEX idx_mv_top_validators_address ON mv_top_validators(address);

-- Top DEX pools view
CREATE MATERIALIZED VIEW mv_top_dex_pools AS
SELECT
    pool_id,
    token_a,
    token_b,
    tvl,
    volume_24h,
    apr
FROM dex_pools
ORDER BY tvl DESC
LIMIT 100;

CREATE UNIQUE INDEX idx_mv_top_dex_pools_pool_id ON mv_top_dex_pools(pool_id);

-- Network statistics view (24h aggregates)
CREATE MATERIALIZED VIEW mv_network_stats_24h AS
SELECT
    COUNT(DISTINCT blocks.height) as block_count,
    COUNT(DISTINCT transactions.hash) as tx_count,
    COUNT(DISTINCT accounts.address) as active_accounts,
    SUM(blocks.gas_used) as total_gas_used,
    AVG(blocks.tx_count) as avg_tx_per_block,
    MAX(blocks.height) as latest_height,
    (SELECT COUNT(*) FROM validators WHERE status = 'bonded') as active_validators,
    (SELECT COUNT(*) FROM dex_pools) as dex_pool_count,
    (SELECT COUNT(*) FROM dex_trades WHERE timestamp > NOW() - INTERVAL '24 hours') as dex_trades_24h,
    NOW() as computed_at
FROM blocks
LEFT JOIN transactions ON blocks.height = transactions.block_height
LEFT JOIN accounts ON accounts.last_seen_height = blocks.height
WHERE blocks.timestamp > NOW() - INTERVAL '24 hours';

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to update account last seen
CREATE OR REPLACE FUNCTION update_account_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE accounts
    SET
        last_seen_height = NEW.block_height,
        last_seen_at = NEW.timestamp,
        tx_count = tx_count + 1,
        updated_at = NOW()
    WHERE address = NEW.sender;

    IF NOT FOUND THEN
        INSERT INTO accounts (address, first_seen_height, last_seen_height, first_seen_at, last_seen_at, tx_count)
        VALUES (NEW.sender, NEW.block_height, NEW.block_height, NEW.timestamp, NEW.timestamp, 1);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_account_last_seen
AFTER INSERT ON transactions
FOR EACH ROW
WHEN (NEW.sender IS NOT NULL)
EXECUTE FUNCTION update_account_last_seen();

-- Function to update DEX pool TVL
CREATE OR REPLACE FUNCTION update_dex_pool_tvl()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE dex_pools
    SET
        tvl = (reserve_a::numeric * 2)::varchar, -- Simplified TVL calculation
        updated_at = NOW()
    WHERE pool_id = NEW.pool_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_dex_pool_tvl
AFTER INSERT ON dex_trades
FOR EACH ROW
EXECUTE FUNCTION update_dex_pool_tvl();

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_latest_blocks;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_latest_transactions;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_top_validators;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_top_dex_pools;
    REFRESH MATERIALIZED VIEW mv_network_stats_24h;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PARTITIONING (for scalability)
-- ============================================================================

-- Partition transactions table by month
CREATE TABLE IF NOT EXISTS transactions_2024_01 PARTITION OF transactions
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE IF NOT EXISTS transactions_2024_02 PARTITION OF transactions
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Add more partitions as needed...

-- ============================================================================
-- INITIAL DATA AND INDEXES
-- ============================================================================

-- Insert initial network stats
INSERT INTO network_stats (metric_name, metric_value, metric_type, category, period, timestamp)
VALUES
    ('total_blocks', '0', 'counter', 'blockchain', 'all', NOW()),
    ('total_transactions', '0', 'counter', 'blockchain', 'all', NOW()),
    ('total_addresses', '0', 'counter', 'blockchain', 'all', NOW()),
    ('active_validators', '0', 'gauge', 'validators', 'all', NOW()),
    ('dex_total_volume', '0', 'counter', 'dex', 'all', NOW()),
    ('oracle_price_feeds', '0', 'counter', 'oracle', 'all', NOW()),
    ('compute_requests', '0', 'counter', 'compute', 'all', NOW());

-- Grant permissions (adjust as needed)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO explorer_api;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO explorer_api;

-- Comments for documentation
COMMENT ON TABLE blocks IS 'Stores all blockchain blocks with metadata';
COMMENT ON TABLE transactions IS 'Stores all transactions with execution results';
COMMENT ON TABLE events IS 'Stores transaction events for detailed tracking';
COMMENT ON TABLE validators IS 'Stores validator information and status';
COMMENT ON TABLE dex_pools IS 'Stores DEX liquidity pools';
COMMENT ON TABLE dex_trades IS 'Stores DEX swap/trade history';
COMMENT ON TABLE oracle_prices IS 'Stores aggregated oracle price data';
COMMENT ON TABLE compute_requests IS 'Stores compute marketplace requests';

-- ============================================================================
-- PERFORMANCE TUNING
-- ============================================================================

-- Analyze tables for query optimization
ANALYZE blocks;
ANALYZE transactions;
ANALYZE events;
ANALYZE accounts;
ANALYZE validators;
ANALYZE dex_pools;
ANALYZE dex_trades;
ANALYZE oracle_prices;
ANALYZE compute_requests;

-- Vacuum tables
VACUUM ANALYZE blocks;
VACUUM ANALYZE transactions;

-- ============================================================================
-- MAINTENANCE PROCEDURES
-- ============================================================================

-- Procedure to clean old search index entries
CREATE OR REPLACE FUNCTION cleanup_old_search_index()
RETURNS void AS $$
BEGIN
    DELETE FROM search_index
    WHERE updated_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Procedure to archive old transactions (optional)
CREATE OR REPLACE FUNCTION archive_old_transactions(days_old INTEGER)
RETURNS INTEGER AS $$
DECLARE
    rows_archived INTEGER;
BEGIN
    -- Archive to separate table or delete
    WITH archived AS (
        DELETE FROM transactions
        WHERE timestamp < NOW() - (days_old || ' days')::INTERVAL
        RETURNING *
    )
    SELECT COUNT(*) INTO rows_archived FROM archived;

    RETURN rows_archived;
END;
$$ LANGUAGE plpgsql;

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    id SERIAL PRIMARY KEY,
    version VARCHAR(32) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT
);

INSERT INTO schema_version (version, description)
VALUES ('1.0.0', 'Initial schema with complete explorer database structure');

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'PAW Chain Explorer database schema created successfully!';
    RAISE NOTICE 'Schema version: 1.0.0';
    RAISE NOTICE 'Total tables: 30+';
    RAISE NOTICE 'Total indexes: 100+';
    RAISE NOTICE 'Materialized views: 5';
END $$;
