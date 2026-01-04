-- PAW Explorer Database Schema (PostgreSQL)

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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blocks_time ON blocks (time DESC);
CREATE INDEX IF NOT EXISTS idx_blocks_proposer ON blocks (proposer_address);

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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tx_hash ON transactions (hash);
CREATE INDEX IF NOT EXISTS idx_tx_block ON transactions (block_height);
CREATE INDEX IF NOT EXISTS idx_tx_sender ON transactions (sender);
CREATE INDEX IF NOT EXISTS idx_tx_time ON transactions (time DESC);
CREATE INDEX IF NOT EXISTS idx_tx_type ON transactions (type);

-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    address VARCHAR(64) PRIMARY KEY,
    balance JSONB NOT NULL DEFAULT '[]',
    tx_count BIGINT NOT NULL DEFAULT 0,
    first_seen_height BIGINT,
    last_seen_height BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_tx_count ON accounts (tx_count DESC);

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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_validators_voting_power ON validators (voting_power DESC);
CREATE INDEX IF NOT EXISTS idx_validators_status ON validators (status);

-- Validator uptime tracking
CREATE TABLE IF NOT EXISTS validator_uptime (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL REFERENCES validators(address) ON DELETE CASCADE,
    height BIGINT NOT NULL,
    signed BOOLEAN NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    UNIQUE (validator_address, height)
);

CREATE INDEX IF NOT EXISTS idx_uptime_validator ON validator_uptime (validator_address);
CREATE INDEX IF NOT EXISTS idx_uptime_height ON validator_uptime (height);

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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pools_tokens ON dex_pools (token_a, token_b);
CREATE INDEX IF NOT EXISTS idx_pools_tvl ON dex_pools (tvl DESC);

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
    time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_swaps_pool ON dex_swaps (pool_id);
CREATE INDEX IF NOT EXISTS idx_swaps_sender ON dex_swaps (sender);
CREATE INDEX IF NOT EXISTS idx_swaps_time ON dex_swaps (time DESC);

-- DEX pool price history table
CREATE TABLE IF NOT EXISTS dex_pool_price_history (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_height BIGINT NOT NULL,
    open VARCHAR(64) NOT NULL,
    high VARCHAR(64) NOT NULL,
    low VARCHAR(64) NOT NULL,
    close VARCHAR(64) NOT NULL,
    volume VARCHAR(64) NOT NULL,
    liquidity_a VARCHAR(64) NOT NULL,
    liquidity_b VARCHAR(64) NOT NULL,
    price_a_to_b VARCHAR(64) NOT NULL,
    price_b_to_a VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (pool_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_price_history_pool ON dex_pool_price_history (pool_id);
CREATE INDEX IF NOT EXISTS idx_price_history_time ON dex_pool_price_history (timestamp DESC);

-- DEX pool statistics table
CREATE TABLE IF NOT EXISTS dex_pool_statistics (
    id BIGSERIAL PRIMARY KEY,
    pool_id VARCHAR(64) NOT NULL,
    period VARCHAR(20) NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    volume_token_a VARCHAR(64) NOT NULL,
    volume_token_b VARCHAR(64) NOT NULL,
    volume_usd VARCHAR(64) NOT NULL,
    trade_count INTEGER NOT NULL DEFAULT 0,
    avg_liquidity_a VARCHAR(64) NOT NULL,
    avg_liquidity_b VARCHAR(64) NOT NULL,
    min_liquidity VARCHAR(64) NOT NULL,
    max_liquidity VARCHAR(64) NOT NULL,
    fees_collected_a VARCHAR(64) NOT NULL,
    fees_collected_b VARCHAR(64) NOT NULL,
    fees_usd VARCHAR(64) NOT NULL,
    avg_price VARCHAR(64) NOT NULL,
    high_price VARCHAR(64) NOT NULL,
    low_price VARCHAR(64) NOT NULL,
    price_change_percent VARCHAR(64) NOT NULL,
    apr VARCHAR(64) NOT NULL,
    unique_traders INTEGER NOT NULL DEFAULT 0,
    unique_liquidity_providers INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (pool_id, period, period_start)
);

CREATE INDEX IF NOT EXISTS idx_pool_stats_pool ON dex_pool_statistics (pool_id);
CREATE INDEX IF NOT EXISTS idx_pool_stats_period ON dex_pool_statistics (period, period_start DESC);

-- DEX user positions table
CREATE TABLE IF NOT EXISTS dex_user_positions (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(64) NOT NULL,
    pool_id VARCHAR(64) NOT NULL,
    shares VARCHAR(64) NOT NULL,
    initial_amount_a VARCHAR(64) NOT NULL,
    initial_amount_b VARCHAR(64) NOT NULL,
    current_amount_a VARCHAR(64) NOT NULL,
    current_amount_b VARCHAR(64) NOT NULL,
    entry_price VARCHAR(64) NOT NULL,
    entry_height BIGINT NOT NULL,
    entry_timestamp TIMESTAMP NOT NULL,
    entry_tx_hash VARCHAR(64) NOT NULL,
    exit_height BIGINT,
    exit_timestamp TIMESTAMP,
    exit_tx_hash VARCHAR(64),
    fees_earned_a VARCHAR(64) NOT NULL,
    fees_earned_b VARCHAR(64) NOT NULL,
    fees_earned_usd VARCHAR(64) NOT NULL,
    impermanent_loss VARCHAR(64) NOT NULL,
    total_return_percent VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (address, pool_id)
);

CREATE INDEX IF NOT EXISTS idx_user_positions_address ON dex_user_positions (address);
CREATE INDEX IF NOT EXISTS idx_user_positions_pool ON dex_user_positions (pool_id);
CREATE INDEX IF NOT EXISTS idx_user_positions_status ON dex_user_positions (status);

-- DEX analytics cache table
CREATE TABLE IF NOT EXISTS dex_analytics_cache (
    id BIGSERIAL PRIMARY KEY,
    cache_key VARCHAR(200) UNIQUE NOT NULL,
    cache_type VARCHAR(50) NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cache_expires ON dex_analytics_cache (expires_at);

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
    time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_liquidity_pool ON dex_liquidity_events (pool_id);
CREATE INDEX IF NOT EXISTS idx_liquidity_sender ON dex_liquidity_events (sender);
CREATE INDEX IF NOT EXISTS idx_liquidity_time ON dex_liquidity_events (time DESC);

-- Oracle prices table
CREATE TABLE IF NOT EXISTS oracle_prices (
    id BIGSERIAL PRIMARY KEY,
    asset VARCHAR(20) NOT NULL,
    price DECIMAL(30, 10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_height BIGINT NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'aggregate',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prices_asset ON oracle_prices (asset);
CREATE INDEX IF NOT EXISTS idx_prices_timestamp ON oracle_prices (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_prices_height ON oracle_prices (block_height DESC);

-- Oracle validator submissions table
CREATE TABLE IF NOT EXISTS oracle_submissions (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL,
    asset VARCHAR(20) NOT NULL,
    price DECIMAL(30, 10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_height BIGINT NOT NULL,
    included BOOLEAN NOT NULL DEFAULT false,
    deviation DECIMAL(10, 4)
);

CREATE INDEX IF NOT EXISTS idx_submissions_validator ON oracle_submissions (validator_address);
CREATE INDEX IF NOT EXISTS idx_submissions_asset ON oracle_submissions (asset);
CREATE INDEX IF NOT EXISTS idx_submissions_time ON oracle_submissions (timestamp DESC);

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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compute_requester ON compute_requests (requester);
CREATE INDEX IF NOT EXISTS idx_compute_provider ON compute_requests (provider);
CREATE INDEX IF NOT EXISTS idx_compute_status ON compute_requests (status);
CREATE INDEX IF NOT EXISTS idx_compute_created ON compute_requests (created_at DESC);

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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stats_date ON network_stats (date DESC);

-- Events table (generic event storage)
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(64) NOT NULL,
    block_height BIGINT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    attributes JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_tx ON events (tx_hash);
CREATE INDEX IF NOT EXISTS idx_events_type ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_module ON events (module);
CREATE INDEX IF NOT EXISTS idx_events_height ON events (block_height);

-- Account balances table
CREATE TABLE IF NOT EXISTS account_balances (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(64) NOT NULL,
    denom VARCHAR(64) NOT NULL,
    amount VARCHAR(100) NOT NULL,
    last_updated_height BIGINT NOT NULL,
    last_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (address, denom)
);

CREATE INDEX IF NOT EXISTS idx_account_balances_address ON account_balances (address);

-- Account tokens table
CREATE TABLE IF NOT EXISTS account_tokens (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(64) NOT NULL,
    token_denom VARCHAR(64) NOT NULL,
    token_name VARCHAR(100),
    token_symbol VARCHAR(50),
    amount VARCHAR(100) NOT NULL,
    ibc_trace JSONB,
    last_updated_height BIGINT NOT NULL,
    last_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (address, token_denom)
);

CREATE INDEX IF NOT EXISTS idx_account_tokens_address ON account_tokens (address);

-- Validator rewards table
CREATE TABLE IF NOT EXISTS validator_rewards (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL,
    amount VARCHAR(100) NOT NULL,
    denom VARCHAR(64) NOT NULL,
    height BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rewards_validator ON validator_rewards (validator_address);
CREATE INDEX IF NOT EXISTS idx_rewards_height ON validator_rewards (height DESC);

-- Oracle slashes table
CREATE TABLE IF NOT EXISTS oracle_slashes (
    id BIGSERIAL PRIMARY KEY,
    validator_address VARCHAR(64) NOT NULL,
    slash_amount VARCHAR(100) NOT NULL,
    reason VARCHAR(200) NOT NULL,
    height BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_oracle_slashes_validator ON oracle_slashes (validator_address);
CREATE INDEX IF NOT EXISTS idx_oracle_slashes_height ON oracle_slashes (height DESC);

-- Compute request results table
CREATE TABLE IF NOT EXISTS compute_results (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    result_hash VARCHAR(64) NOT NULL,
    output_data_hash VARCHAR(64),
    status VARCHAR(20) NOT NULL,
    height BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compute_results_request ON compute_results (request_id);

-- Compute verifications table
CREATE TABLE IF NOT EXISTS compute_verifications (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL,
    verifier VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL,
    score VARCHAR(64),
    height BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compute_verifications_request ON compute_verifications (request_id);

-- Compute providers table
CREATE TABLE IF NOT EXISTS compute_providers (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(64) UNIQUE NOT NULL,
    stake VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    reputation INTEGER NOT NULL DEFAULT 0,
    total_jobs BIGINT NOT NULL DEFAULT 0,
    completed_jobs BIGINT NOT NULL DEFAULT 0,
    failed_jobs BIGINT NOT NULL DEFAULT 0,
    uptime_30d DECIMAL(5, 2) NOT NULL DEFAULT 0,
    avg_completion_time DECIMAL(10, 2) NOT NULL DEFAULT 0,
    slash_count BIGINT NOT NULL DEFAULT 0,
    updated_height BIGINT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compute_providers_active ON compute_providers (active);

-- Indexing progress table
CREATE TABLE IF NOT EXISTS indexing_progress (
    id BIGSERIAL PRIMARY KEY,
    last_indexed_height BIGINT NOT NULL,
    total_blocks_indexed BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    start_height BIGINT,
    target_height BIGINT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing metrics table
CREATE TABLE IF NOT EXISTS indexing_metrics (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    start_height BIGINT,
    end_height BIGINT,
    blocks_processed BIGINT,
    duration_seconds DOUBLE PRECISION,
    blocks_per_second DOUBLE PRECISION,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing checkpoints table
CREATE TABLE IF NOT EXISTS indexing_checkpoints (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT NOT NULL,
    block_hash VARCHAR(64) NOT NULL,
    blocks_since_last_checkpoint INTEGER NOT NULL,
    time_since_last_checkpoint INTERVAL NOT NULL,
    avg_blocks_per_second DOUBLE PRECISION NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_indexing_checkpoints_height ON indexing_checkpoints (height DESC);

-- Failed blocks table
CREATE TABLE IF NOT EXISTS failed_blocks (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT NOT NULL,
    error_message TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP,
    resolved BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_failed_blocks_height ON failed_blocks (height DESC);

-- Indexer state (track indexing progress)
CREATE TABLE IF NOT EXISTS indexer_state (
    key VARCHAR(50) PRIMARY KEY,
    value BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Initialize indexer state
INSERT INTO indexer_state (key, value) VALUES ('last_indexed_height', 0) ON CONFLICT DO NOTHING;

-- Views

CREATE OR REPLACE VIEW active_validators AS
SELECT
    v.*,
    COUNT(DISTINCT vu.height) as signed_blocks_count
FROM validators v
LEFT JOIN validator_uptime vu ON v.address = vu.validator_address AND vu.signed = true
WHERE v.status = 'BOND_STATUS_BONDED'
GROUP BY v.address;

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

-- Sample seed data for staging visualization

INSERT INTO blocks (height, hash, proposer_address, time, tx_count, gas_used, gas_wanted, evidence_count)
VALUES (1, '0000000000000000000000000000000000000000000000000000000000000001', 'pawvalcons1sampleaddr', NOW() - INTERVAL '10 minutes', 2, 200000, 300000, 0)
ON CONFLICT (height) DO NOTHING;

INSERT INTO transactions (hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted, fee_amount, fee_denom, memo, raw_log, time, messages, events)
VALUES
('txhashsample1', 1, 0, 'bank/MsgSend', 'paw1senderaddress', 'success', 0, 80000, 120000, '100000', 'upaw', 'Sample send', '{"events":[]}', NOW() - INTERVAL '9 minutes', '[]', '[]'),
('txhashsample2', 1, 1, 'dex/MsgSwap', 'paw1dexuser', 'success', 0, 90000, 130000, '150000', 'upaw', 'Sample swap', '{"events":[]}', NOW() - INTERVAL '9 minutes', '[]', '[]')
ON CONFLICT (hash) DO NOTHING;

INSERT INTO accounts (address, balance, tx_count, first_seen_height, last_seen_height)
VALUES
('paw1senderaddress', '[{"denom":"upaw","amount":"5000000"}]', 1, 1, 1),
('paw1dexuser', '[{"denom":"upaw","amount":"7500000"}]', 1, 1, 1)
ON CONFLICT (address) DO NOTHING;

INSERT INTO validators (address, operator_address, consensus_pubkey, moniker, commission_rate, commission_max_rate, commission_max_change_rate, voting_power, status, tokens, delegator_shares)
VALUES (
    'pawval1sample',
    'pawvop1sample',
    'consensuspubkeysample',
    'Staging Validator One',
    0.0500000000,
    0.2000000000,
    0.0200000000,
    1000000,
    'BOND_STATUS_BONDED',
    1000000,
    1000000.0
)
ON CONFLICT (address) DO NOTHING;

INSERT INTO validator_uptime (validator_address, height, signed, timestamp)
VALUES ('pawval1sample', 1, true, NOW() - INTERVAL '10 minutes')
ON CONFLICT DO NOTHING;

INSERT INTO dex_pools (pool_id, token_a, token_b, reserve_a, reserve_b, lp_token_supply, swap_fee_rate, total_volume_24h, total_fees_24h, apr, tvl, created_height)
VALUES ('pool-1', 'upaw', 'uusdc', 100000.0, 50000.0, 10000.0, 0.0030, 250000.0, 750.0, 0.1200, 150000.0, 1)
ON CONFLICT (pool_id) DO NOTHING;

INSERT INTO dex_swaps (tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time)
VALUES ('txhashsample2', 'pool-1', 'paw1dexuser', 'upaw', 'uusdc', 1000.0, 500.0, 0.5, 3.0, NOW() - INTERVAL '9 minutes')
ON CONFLICT DO NOTHING;

INSERT INTO dex_liquidity_events (tx_hash, pool_id, sender, event_type, amount_a, amount_b, lp_tokens, time)
VALUES ('txhashsample1', 'pool-1', 'paw1senderaddress', 'add', 5000.0, 2500.0, 300.0, NOW() - INTERVAL '9 minutes')
ON CONFLICT DO NOTHING;

INSERT INTO oracle_prices (asset, price, timestamp, block_height)
VALUES ('PAW/USD', 1.23, NOW() - INTERVAL '5 minutes', 1)
ON CONFLICT DO NOTHING;

INSERT INTO oracle_submissions (validator_address, asset, price, timestamp, block_height, included, deviation)
VALUES ('pawval1sample', 'PAW/USD', 1.23, NOW() - INTERVAL '5 minutes', 1, true, 0.0)
ON CONFLICT DO NOTHING;

INSERT INTO compute_requests (request_id, requester, provider, status, task_type, payment_amount, payment_denom, escrow_amount, result_hash, verification_status, created_height, completed_height)
VALUES ('req-1', 'paw1senderaddress', 'pawval1sample', 'completed', 'zk-proof', 1000.0, 'upaw', 500.0, 'resulthashsample', 'verified', 1, 1)
ON CONFLICT DO NOTHING;

INSERT INTO network_stats (date, total_txs, unique_accounts, total_volume, dex_tvl, active_validators, avg_block_time)
VALUES (CURRENT_DATE, 2, 2, 250000.0, 150000.0, 1, 6.5)
ON CONFLICT DO NOTHING;

INSERT INTO events (tx_hash, block_height, event_type, module, attributes)
VALUES ('txhashsample2', 1, 'swap_executed', 'dex', '{"token_in":"upaw","token_out":"uusdc"}')
ON CONFLICT DO NOTHING;
