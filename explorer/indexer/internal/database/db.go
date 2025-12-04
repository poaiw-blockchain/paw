package database

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

//go:embed schema.sql
var schemaFile embed.FS

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// Config holds database configuration
type Config struct {
	URL            string
	MaxConnections int
	MaxIdle        int
	ConnMaxLife    time.Duration
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(cfg.ConnMaxLife)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("Successfully connected to database")

	return &DB{db}, nil
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	schema, err := schemaFile.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Info().Msg("Database schema initialized successfully")
	return nil
}

// GetLastIndexedHeight returns the last indexed block height
func (db *DB) GetLastIndexedHeight() (int64, error) {
	var height int64
	err := db.QueryRow("SELECT value FROM indexer_state WHERE key = 'last_indexed_height'").Scan(&height)
	if err != nil {
		return 0, err
	}
	return height, nil
}

// UpdateLastIndexedHeight updates the last indexed block height
func (db *DB) UpdateLastIndexedHeight(height int64) error {
	_, err := db.Exec(
		"INSERT INTO indexer_state (key, value, updated_at) VALUES ('last_indexed_height', $1, NOW()) ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()",
		height,
	)
	return err
}

// BeginTx starts a new transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// InsertBlock inserts a new block
func (db *DB) InsertBlock(tx *sql.Tx, block Block) error {
	_, err := tx.Exec(`
		INSERT INTO blocks (height, hash, proposer_address, time, tx_count, gas_used, gas_wanted, evidence_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (height) DO UPDATE SET
			hash = EXCLUDED.hash,
			proposer_address = EXCLUDED.proposer_address,
			time = EXCLUDED.time,
			tx_count = EXCLUDED.tx_count,
			gas_used = EXCLUDED.gas_used,
			gas_wanted = EXCLUDED.gas_wanted,
			evidence_count = EXCLUDED.evidence_count
	`, block.Height, block.Hash, block.ProposerAddress, block.Time, block.TxCount, block.GasUsed, block.GasWanted, block.EvidenceCount)
	return err
}

// InsertTransaction inserts a new transaction
func (db *DB) InsertTransaction(tx *sql.Tx, transaction Transaction) error {
	_, err := tx.Exec(`
		INSERT INTO transactions (hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted, fee_amount, fee_denom, memo, raw_log, time, messages, events)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (hash) DO UPDATE SET
			block_height = EXCLUDED.block_height,
			tx_index = EXCLUDED.tx_index,
			type = EXCLUDED.type,
			sender = EXCLUDED.sender,
			status = EXCLUDED.status,
			code = EXCLUDED.code,
			gas_used = EXCLUDED.gas_used,
			gas_wanted = EXCLUDED.gas_wanted,
			fee_amount = EXCLUDED.fee_amount,
			fee_denom = EXCLUDED.fee_denom,
			memo = EXCLUDED.memo,
			raw_log = EXCLUDED.raw_log,
			time = EXCLUDED.time,
			messages = EXCLUDED.messages,
			events = EXCLUDED.events
	`, transaction.Hash, transaction.BlockHeight, transaction.TxIndex, transaction.Type, transaction.Sender,
		transaction.Status, transaction.Code, transaction.GasUsed, transaction.GasWanted, transaction.FeeAmount,
		transaction.FeeDenom, transaction.Memo, transaction.RawLog, transaction.Time, transaction.Messages, transaction.Events)
	return err
}

// UpsertAccount updates or inserts account information
func (db *DB) UpsertAccount(tx *sql.Tx, address string, height int64) error {
	_, err := tx.Exec(`
		INSERT INTO accounts (address, tx_count, first_seen_height, last_seen_height, updated_at)
		VALUES ($1, 1, $2, $2, NOW())
		ON CONFLICT (address) DO UPDATE SET
			tx_count = accounts.tx_count + 1,
			last_seen_height = $2,
			updated_at = NOW()
	`, address, height)
	return err
}

// InsertEvent inserts a new event
func (db *DB) InsertEvent(tx *sql.Tx, event Event) error {
	_, err := tx.Exec(`
		INSERT INTO events (tx_hash, block_height, event_type, module, attributes)
		VALUES ($1, $2, $3, $4, $5)
	`, event.TxHash, event.BlockHeight, event.EventType, event.Module, event.Attributes)
	return err
}

// InsertDEXSwap inserts a DEX swap event
func (db *DB) InsertDEXSwap(tx *sql.Tx, swap DEXSwap) error {
	_, err := tx.Exec(`
		INSERT INTO dex_swaps (tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, swap.TxHash, swap.PoolID, swap.Sender, swap.TokenIn, swap.TokenOut, swap.AmountIn, swap.AmountOut, swap.Price, swap.Fee, swap.Time)
	return err
}

// UpsertDEXPool updates or inserts a DEX pool
func (db *DB) UpsertDEXPool(tx *sql.Tx, pool DEXPool) error {
	_, err := tx.Exec(`
		INSERT INTO dex_pools (pool_id, token_a, token_b, reserve_a, reserve_b, lp_token_supply, swap_fee_rate, tvl, created_height, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (pool_id) DO UPDATE SET
			reserve_a = EXCLUDED.reserve_a,
			reserve_b = EXCLUDED.reserve_b,
			lp_token_supply = EXCLUDED.lp_token_supply,
			tvl = EXCLUDED.tvl,
			updated_at = NOW()
	`, pool.PoolID, pool.TokenA, pool.TokenB, pool.ReserveA, pool.ReserveB, pool.LPTokenSupply, pool.SwapFeeRate, pool.TVL, pool.CreatedHeight)
	return err
}

// InsertOraclePrice inserts an oracle price update
func (db *DB) InsertOraclePrice(tx *sql.Tx, price OraclePrice) error {
	_, err := tx.Exec(`
		INSERT INTO oracle_prices (asset, price, timestamp, block_height, source)
		VALUES ($1, $2, $3, $4, $5)
	`, price.Asset, price.Price, price.Timestamp, price.BlockHeight, price.Source)
	return err
}

// UpsertValidator updates or inserts validator information
func (db *DB) UpsertValidator(tx *sql.Tx, validator Validator) error {
	_, err := tx.Exec(`
		INSERT INTO validators (address, operator_address, consensus_pubkey, moniker, commission_rate,
			commission_max_rate, commission_max_change_rate, voting_power, jailed, status, tokens,
			delegator_shares, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (address) DO UPDATE SET
			operator_address = EXCLUDED.operator_address,
			consensus_pubkey = EXCLUDED.consensus_pubkey,
			moniker = EXCLUDED.moniker,
			commission_rate = EXCLUDED.commission_rate,
			commission_max_rate = EXCLUDED.commission_max_rate,
			commission_max_change_rate = EXCLUDED.commission_max_change_rate,
			voting_power = EXCLUDED.voting_power,
			jailed = EXCLUDED.jailed,
			status = EXCLUDED.status,
			tokens = EXCLUDED.tokens,
			delegator_shares = EXCLUDED.delegator_shares,
			updated_at = NOW()
	`, validator.Address, validator.OperatorAddress, validator.ConsensusPubkey, validator.Moniker,
		validator.CommissionRate, validator.CommissionMaxRate, validator.CommissionMaxChangeRate,
		validator.VotingPower, validator.Jailed, validator.Status, validator.Tokens, validator.DelegatorShares)
	return err
}

// RecordValidatorUptime records validator uptime for a block
func (db *DB) RecordValidatorUptime(tx *sql.Tx, validatorAddress string, height int64, signed bool, timestamp time.Time) error {
	_, err := tx.Exec(`
		INSERT INTO validator_uptime (validator_address, height, signed, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (validator_address, height) DO UPDATE SET
			signed = EXCLUDED.signed,
			timestamp = EXCLUDED.timestamp
	`, validatorAddress, height, signed, timestamp)
	return err
}

// SaveIndexingProgress saves the indexing progress
func (db *DB) SaveIndexingProgress(height int64, status string) error {
	_, err := db.Exec(
		"SELECT update_indexing_progress($1, $2)",
		height, status,
	)
	return err
}

// GetIndexingProgress retrieves the current indexing progress
func (db *DB) GetIndexingProgress() (*IndexingProgress, error) {
	var progress IndexingProgress
	err := db.QueryRow(`
		SELECT
			last_indexed_height,
			total_blocks_indexed,
			status,
			start_height,
			target_height,
			started_at,
			completed_at,
			updated_at
		FROM indexing_progress
		WHERE id = 1
	`).Scan(
		&progress.LastIndexedHeight,
		&progress.TotalBlocksIndexed,
		&progress.Status,
		&progress.StartHeight,
		&progress.TargetHeight,
		&progress.StartedAt,
		&progress.CompletedAt,
		&progress.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

// SaveFailedBlock records a failed block
func (db *DB) SaveFailedBlock(height int64, errorMsg string) error {
	_, err := db.Exec(
		"SELECT record_failed_block($1, $2, $3)",
		height, errorMsg, "indexing_error",
	)
	return err
}

// GetFailedBlocks retrieves all unresolved failed blocks
func (db *DB) GetFailedBlocks(maxRetries int) ([]FailedBlock, error) {
	rows, err := db.Query(`
		SELECT height, error_message, retry_count, last_retry_at
		FROM get_unresolved_failed_blocks($1)
	`, maxRetries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failedBlocks []FailedBlock
	for rows.Next() {
		var fb FailedBlock
		if err := rows.Scan(&fb.Height, &fb.ErrorMessage, &fb.RetryCount, &fb.LastRetryAt); err != nil {
			return nil, err
		}
		failedBlocks = append(failedBlocks, fb)
	}

	return failedBlocks, rows.Err()
}

// ResolveFailedBlock marks a failed block as resolved
func (db *DB) ResolveFailedBlock(height int64) error {
	_, err := db.Exec("SELECT resolve_failed_block($1)", height)
	return err
}

// RecordIndexingMetric records an indexing performance metric
func (db *DB) RecordIndexingMetric(metric IndexingMetric) error {
	_, err := db.Exec(`
		SELECT record_indexing_metric($1, $2, $3, $4, $5, $6, $7)
	`,
		metric.Name,
		metric.Value,
		metric.StartHeight,
		metric.EndHeight,
		metric.BlocksProcessed,
		metric.DurationSeconds,
		metric.BlocksPerSecond,
	)
	return err
}

// CreateIndexingCheckpoint creates a checkpoint
func (db *DB) CreateIndexingCheckpoint(checkpoint IndexingCheckpoint) error {
	_, err := db.Exec(`
		SELECT create_indexing_checkpoint($1, $2, $3, $4, $5, $6)
	`,
		checkpoint.Height,
		checkpoint.BlockHash,
		checkpoint.BlocksSinceLastCheckpoint,
		checkpoint.TimeSinceLastCheckpoint,
		checkpoint.AvgBlocksPerSecond,
		checkpoint.Status,
	)
	return err
}

// GetIndexingStatistics returns comprehensive indexing statistics
func (db *DB) GetIndexingStatistics() (*IndexingStatistics, error) {
	var stats IndexingStatistics
	err := db.QueryRow(`
		SELECT * FROM get_indexing_statistics()
	`).Scan(
		&stats.TotalBlocksIndexed,
		&stats.LastIndexedHeight,
		&stats.CurrentStatus,
		&stats.FailedBlocksCount,
		&stats.UnresolvedFailedBlocks,
		&stats.AvgBlocksPerSecond,
		&stats.EstimatedCompletionTime,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	log.Info().Msg("Closing database connection")
	return db.DB.Close()
}
