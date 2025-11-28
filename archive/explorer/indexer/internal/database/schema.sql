-- PAW Explorer Database Schema

-- Blocks table
CREATE TABLE IF NOT EXISTS blocks (
    height BIGINT PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    proposer_address VARCHAR(64) NOT NULL,
    time TIMESTAMP NOT NULL,
    tx_count INTEGER NOT NULL DEFAULT 0,
    gas_used BIGINT NOT NULL DEFAULT 0,
    gas_wanted BIGINT NOT NULL DEFAULT 0,
    evidence_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_blocks_time (time DESC),
    INDEX idx_blocks_proposer (proposer_address)
);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    block_height BIGINT NOT NULL REFERENCES blocks(height) ON DELETE CASCADE,
    tx_index INTEGER NOT NULL,
    type VARCHAR(100) NOT NULL,
    sender VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL,
    code INTEGER NOT NULL DEFAULT 0,
    gas_used BIGINT NOT NULL DEFAULT 0,
    gas_wanted BIGINT NOT NULL DEFAULT 0,
    fee_amount VARCHAR(100),
    fee_denom VARCHAR(20),
    memo TEXT,
    raw_log TEXT,
    time TIMESTAMP NOT NULL,
    messages JSONB NOT NULL,
    events JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tx_hash (hash),
    INDEX idx_tx_block (block_height),
    INDEX idx_tx_sender (sender),
    INDEX idx_tx_time (time DESC),
    INDEX idx_tx_type (type)
);

-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    address VARCHAR(64) PRIMARY KEY,
    balance JSONB NOT NULL DEFAULT '[]',
    tx_count BIGINT NOT NULL DEFAULT 0,
    first_seen_height BIGINT,
    last_seen_height BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_accounts_tx_count (tx_count DESC)
);

-- Validators table
CREATE TABLE IF NOT EXISTS validators (
    address VARCHAR(64) PRIMARY KEY,
    operator_address VARCHAR(64) UNIQUE NOT NULL,
    consensus_pubkey TEXT NOT NULL,
    moniker VARCHAR(100) NOT NULL,
    identity VARCHAR(100),
    website VARCHAR(200),
    security_contact VARCHAR(200),
    details TEXT,
    commission_rate DECIMAL(20, 10) NOT NULL,
    commission_max_rate DECIMAL(20, 10) NOT NULL,
    commission_max_change_rate DECIMAL(20, 10) NOT NULL,
    voting_power BIGINT NOT NULL DEFAULT 0,
    jailed BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL,
    tokens BIGINT NOT NULL DEFAULT 0,
    delegator_shares DECIMAL(30, 10) NOT NULL DEFAULT 0,
    uptime_percentage DECIMAL(5, 2) NOT NULL DEFAULT 100.00,
    missed_blocks BIGINT NOT NULL DEFAULT 0,
    total_blocks BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_validators_voting_power (voting_power DESC),
    INDEX idx_validators_status (status)
);

-- Validator uptime tracking
CREATE TABLE IF NOT EXISTS validator_uptime (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL REFERENCES validators(address) ON DELETE CASCADE,
    height BIGINT NOT NULL,
    signed BOOLEAN NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    INDEX idx_uptime_validator (validator_address),
    INDEX idx_uptime_height (height),
    UNIQUE (validator_address, height)
);

-- DEX Pools table
CREATE TABLE IF NOT EXISTS dex_pools (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) UNIQUE NOT NULL,
    token_a VARCHAR(20) NOT NULL,
    token_b VARCHAR(20) NOT NULL,
    reserve_a DECIMAL(40, 10) NOT NULL DEFAULT 0,
    reserve_b DECIMAL(40, 10) NOT NULL DEFAULT 0,
    lp_token_supply DECIMAL(40, 10) NOT NULL DEFAULT 0,
    swap_fee_rate DECIMAL(5, 4) NOT NULL,
    total_volume_24h DECIMAL(40, 10) NOT NULL DEFAULT 0,
    total_fees_24h DECIMAL(40, 10) NOT NULL DEFAULT 0,
    apr DECIMAL(10, 4),
    tvl DECIMAL(40, 10) NOT NULL DEFAULT 0,
    created_height BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_pools_tokens (token_a, token_b),
    INDEX idx_pools_tvl (tvl DESC)
);

-- DEX Swaps table
CREATE TABLE IF NOT EXISTS dex_swaps (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(64) NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE,
    pool_id VARCHAR(64) NOT NULL,
    sender VARCHAR(64) NOT NULL,
    token_in VARCHAR(20) NOT NULL,
    token_out VARCHAR(20) NOT NULL,
    amount_in DECIMAL(40, 10) NOT NULL,
    amount_out DECIMAL(40, 10) NOT NULL,
    price DECIMAL(40, 10) NOT NULL,
    fee DECIMAL(40, 10) NOT NULL,
    time TIMESTAMP NOT NULL,
    INDEX idx_swaps_pool (pool_id),
    INDEX idx_swaps_sender (sender),
    INDEX idx_swaps_time (time DESC)
);

-- DEX Liquidity events table
CREATE TABLE IF NOT EXISTS dex_liquidity_events (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(64) NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE,
    pool_id VARCHAR(64) NOT NULL,
    sender VARCHAR(64) NOT NULL,
    event_type VARCHAR(20) NOT NULL, -- 'add' or 'remove'
    amount_a DECIMAL(40, 10) NOT NULL,
    amount_b DECIMAL(40, 10) NOT NULL,
    lp_tokens DECIMAL(40, 10) NOT NULL,
    time TIMESTAMP NOT NULL,
    INDEX idx_liquidity_pool (pool_id),
    INDEX idx_liquidity_sender (sender),
    INDEX idx_liquidity_time (time DESC)
);

-- Oracle prices table
CREATE TABLE IF NOT EXISTS oracle_prices (
    id BIGSERIAL PRIMARY KEY,
    asset VARCHAR(20) NOT NULL,
    price DECIMAL(30, 10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_height BIGINT NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'aggregate',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_prices_asset (asset),
    INDEX idx_prices_timestamp (timestamp DESC),
    INDEX idx_prices_height (block_height DESC)
);

-- Oracle validator submissions table
CREATE TABLE IF NOT EXISTS oracle_submissions (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL,
    asset VARCHAR(20) NOT NULL,
    price DECIMAL(30, 10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_height BIGINT NOT NULL,
    included BOOLEAN NOT NULL DEFAULT false,
    deviation DECIMAL(10, 4),
    INDEX idx_submissions_validator (validator_address),
    INDEX idx_submissions_asset (asset),
    INDEX idx_submissions_time (timestamp DESC)
);

-- Compute requests table
CREATE TABLE IF NOT EXISTS compute_requests (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) UNIQUE NOT NULL,
    requester VARCHAR(64) NOT NULL,
    provider VARCHAR(64),
    status VARCHAR(20) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    payment_amount DECIMAL(40, 10) NOT NULL,
    payment_denom VARCHAR(20) NOT NULL,
    escrow_amount DECIMAL(40, 10),
    result_hash VARCHAR(64),
    verification_status VARCHAR(20),
    created_height BIGINT NOT NULL,
    completed_height BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_compute_requester (requester),
    INDEX idx_compute_provider (provider),
    INDEX idx_compute_status (status),
    INDEX idx_compute_created (created_at DESC)
);

-- Network statistics table (for analytics)
CREATE TABLE IF NOT EXISTS network_stats (
    id BIGSERIAL PRIMARY KEY,
    date DATE UNIQUE NOT NULL,
    total_txs BIGINT NOT NULL DEFAULT 0,
    unique_accounts BIGINT NOT NULL DEFAULT 0,
    total_volume DECIMAL(40, 10) NOT NULL DEFAULT 0,
    dex_tvl DECIMAL(40, 10) NOT NULL DEFAULT 0,
    active_validators INTEGER NOT NULL DEFAULT 0,
    avg_block_time DECIMAL(10, 4),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_stats_date (date DESC)
);

-- Events table (generic event storage)
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    attributes JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_events_tx (tx_hash),
    INDEX idx_events_type (event_type),
    INDEX idx_events_module (module),
    INDEX idx_events_height (block_height)
);

-- Indexer state (track indexing progress)
CREATE TABLE IF NOT EXISTS indexer_state (
    key VARCHAR(50) PRIMARY KEY,
    value BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Initialize indexer state
INSERT INTO indexer_state (key, value) VALUES ('last_indexed_height', 0) ON CONFLICT DO NOTHING;

-- Create views for common queries

-- Active validators view
CREATE OR REPLACE VIEW active_validators AS
SELECT
    v.*,
    COUNT(DISTINCT vu.height) as signed_blocks_count
FROM validators v
LEFT JOIN validator_uptime vu ON v.address = vu.validator_address AND vu.signed = true
WHERE v.status = 'BOND_STATUS_BONDED'
GROUP BY v.address;

-- Pool statistics view (24h)
CREATE OR REPLACE VIEW pool_stats_24h AS
SELECT
    p.pool_id,
    p.token_a,
    p.token_b,
    p.tvl,
    COUNT(DISTINCT s.id) as swap_count_24h,
    SUM(s.amount_in) as volume_24h,
    SUM(s.fee) as fees_24h,
    p.apr
FROM dex_pools p
LEFT JOIN dex_swaps s ON p.pool_id = s.pool_id AND s.time > NOW() - INTERVAL '24 hours'
GROUP BY p.pool_id, p.token_a, p.token_b, p.tvl, p.apr;

-- Daily transaction statistics
CREATE OR REPLACE VIEW daily_tx_stats AS
SELECT
    DATE(time) as date,
    COUNT(*) as tx_count,
    COUNT(DISTINCT sender) as unique_senders,
    SUM(gas_used) as total_gas_used,
    AVG(gas_used) as avg_gas_used
FROM transactions
GROUP BY DATE(time)
ORDER BY date DESC;
