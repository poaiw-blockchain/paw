package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DEX-specific database query functions for production-grade analytics

// ============================================================================
// PRICE HISTORY QUERIES
// ============================================================================

// DEXPriceHistory represents an OHLCV data point
type DEXPriceHistory struct {
	ID          int64     `json:"id"`
	PoolID      string    `json:"pool_id"`
	Timestamp   time.Time `json:"timestamp"`
	BlockHeight int64     `json:"block_height"`
	Open        string    `json:"open"`
	High        string    `json:"high"`
	Low         string    `json:"low"`
	Close       string    `json:"close"`
	Volume      string    `json:"volume"`
	LiquidityA  string    `json:"liquidity_a"`
	LiquidityB  string    `json:"liquidity_b"`
	PriceAToB   string    `json:"price_a_to_b"`
	PriceBToA   string    `json:"price_b_to_a"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetPoolPriceHistory retrieves OHLCV data for charting
func (db *Database) GetPoolPriceHistory(ctx context.Context, poolID string, start, end time.Time, interval string) ([]DEXPriceHistory, error) {
	query := `
		SELECT
			id, pool_id, timestamp, block_height,
			open, high, low, close, volume,
			liquidity_a, liquidity_b,
			price_a_to_b, price_b_to_a,
			created_at
		FROM dex_pool_price_history
		WHERE pool_id = $1
			AND timestamp >= $2
			AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := db.QueryContext(ctx, query, poolID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query price history: %w", err)
	}
	defer rows.Close()

	var history []DEXPriceHistory
	for rows.Next() {
		var h DEXPriceHistory
		err := rows.Scan(
			&h.ID, &h.PoolID, &h.Timestamp, &h.BlockHeight,
			&h.Open, &h.High, &h.Low, &h.Close, &h.Volume,
			&h.LiquidityA, &h.LiquidityB,
			&h.PriceAToB, &h.PriceBToA,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan price history: %w", err)
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// InsertPriceHistory inserts a new price history data point
func (db *Database) InsertPriceHistory(ctx context.Context, ph *DEXPriceHistory) error {
	query := `
		INSERT INTO dex_pool_price_history (
			pool_id, timestamp, block_height,
			open, high, low, close, volume,
			liquidity_a, liquidity_b,
			price_a_to_b, price_b_to_a
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (pool_id, timestamp) DO UPDATE SET
			high = GREATEST(EXCLUDED.high, dex_pool_price_history.high),
			low = LEAST(EXCLUDED.low, dex_pool_price_history.low),
			close = EXCLUDED.close,
			volume = CAST(CAST(dex_pool_price_history.volume AS NUMERIC) + CAST(EXCLUDED.volume AS NUMERIC) AS VARCHAR),
			liquidity_a = EXCLUDED.liquidity_a,
			liquidity_b = EXCLUDED.liquidity_b
		RETURNING id
	`

	return db.QueryRowContext(
		ctx, query,
		ph.PoolID, ph.Timestamp, ph.BlockHeight,
		ph.Open, ph.High, ph.Low, ph.Close, ph.Volume,
		ph.LiquidityA, ph.LiquidityB,
		ph.PriceAToB, ph.PriceBToA,
	).Scan(&ph.ID)
}

// ============================================================================
// POOL STATISTICS QUERIES
// ============================================================================

// DEXPoolStatistics represents aggregated pool metrics
type DEXPoolStatistics struct {
	ID                 int64     `json:"id"`
	PoolID             string    `json:"pool_id"`
	Period             string    `json:"period"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	VolumeTokenA       string    `json:"volume_token_a"`
	VolumeTokenB       string    `json:"volume_token_b"`
	VolumeUSD          string    `json:"volume_usd"`
	TradeCount         int       `json:"trade_count"`
	AvgLiquidityA      string    `json:"avg_liquidity_a"`
	AvgLiquidityB      string    `json:"avg_liquidity_b"`
	MinLiquidity       string    `json:"min_liquidity"`
	MaxLiquidity       string    `json:"max_liquidity"`
	FeesCollectedA     string    `json:"fees_collected_a"`
	FeesCollectedB     string    `json:"fees_collected_b"`
	FeesUSD            string    `json:"fees_usd"`
	AvgPrice           string    `json:"avg_price"`
	HighPrice          string    `json:"high_price"`
	LowPrice           string    `json:"low_price"`
	PriceChangePercent string    `json:"price_change_percent"`
	APR                string    `json:"apr"`
	UniqueTraders      int       `json:"unique_traders"`
	UniqueLPs          int       `json:"unique_liquidity_providers"`
	CreatedAt          time.Time `json:"created_at"`
}

// GetPoolStatistics retrieves aggregated statistics for a pool
func (db *Database) GetPoolStatistics(ctx context.Context, poolID, period string, start, end time.Time) ([]DEXPoolStatistics, error) {
	query := `
		SELECT
			id, pool_id, period, period_start, period_end,
			volume_token_a, volume_token_b, volume_usd, trade_count,
			avg_liquidity_a, avg_liquidity_b, min_liquidity, max_liquidity,
			fees_collected_a, fees_collected_b, fees_usd,
			avg_price, high_price, low_price, price_change_percent,
			apr, unique_traders, unique_liquidity_providers,
			created_at
		FROM dex_pool_statistics
		WHERE pool_id = $1
			AND period = $2
			AND period_start >= $3
			AND period_end <= $4
		ORDER BY period_start DESC
	`

	rows, err := db.QueryContext(ctx, query, poolID, period, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query pool statistics: %w", err)
	}
	defer rows.Close()

	var stats []DEXPoolStatistics
	for rows.Next() {
		var s DEXPoolStatistics
		err := rows.Scan(
			&s.ID, &s.PoolID, &s.Period, &s.PeriodStart, &s.PeriodEnd,
			&s.VolumeTokenA, &s.VolumeTokenB, &s.VolumeUSD, &s.TradeCount,
			&s.AvgLiquidityA, &s.AvgLiquidityB, &s.MinLiquidity, &s.MaxLiquidity,
			&s.FeesCollectedA, &s.FeesCollectedB, &s.FeesUSD,
			&s.AvgPrice, &s.HighPrice, &s.LowPrice, &s.PriceChangePercent,
			&s.APR, &s.UniqueTraders, &s.UniqueLPs,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pool statistics: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetPoolVolumeHistory retrieves volume data over time
func (db *Database) GetPoolVolumeHistory(ctx context.Context, poolID string, start, end time.Time, interval string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			period_start as timestamp,
			volume_usd,
			trade_count
		FROM dex_pool_statistics
		WHERE pool_id = $1
			AND period = $2
			AND period_start >= $3
			AND period_end <= $4
		ORDER BY period_start ASC
	`

	rows, err := db.QueryContext(ctx, query, poolID, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query volume history: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var volumeUSD string
		var tradeCount int

		err := rows.Scan(&timestamp, &volumeUSD, &tradeCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan volume history: %w", err)
		}

		results = append(results, map[string]interface{}{
			"timestamp":   timestamp,
			"volume_usd":  volumeUSD,
			"trade_count": tradeCount,
		})
	}

	return results, rows.Err()
}

// GetPoolLiquidityHistory retrieves TVL history
func (db *Database) GetPoolLiquidityHistory(ctx context.Context, poolID string, start, end time.Time) ([]map[string]interface{}, error) {
	query := `
		SELECT
			timestamp,
			liquidity_a,
			liquidity_b
		FROM dex_pool_price_history
		WHERE pool_id = $1
			AND timestamp >= $2
			AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := db.QueryContext(ctx, query, poolID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query liquidity history: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var liquidityA, liquidityB string

		err := rows.Scan(&timestamp, &liquidityA, &liquidityB)
		if err != nil {
			return nil, fmt.Errorf("failed to scan liquidity history: %w", err)
		}

		results = append(results, map[string]interface{}{
			"timestamp":   timestamp,
			"liquidity_a": liquidityA,
			"liquidity_b": liquidityB,
		})
	}

	return results, rows.Err()
}

// GetPoolFeeBreakdown retrieves fee collection details
func (db *Database) GetPoolFeeBreakdown(ctx context.Context, poolID string, start, end time.Time) (map[string]interface{}, error) {
	query := `
		SELECT
			SUM(CAST(fees_collected_a AS NUMERIC)) as total_fees_a,
			SUM(CAST(fees_collected_b AS NUMERIC)) as total_fees_b,
			SUM(CAST(fees_usd AS NUMERIC)) as total_fees_usd,
			AVG(CAST(apr AS NUMERIC)) as avg_apr
		FROM dex_pool_statistics
		WHERE pool_id = $1
			AND period = '24h'
			AND period_start >= $2
			AND period_end <= $3
	`

	var totalFeesA, totalFeesB, totalFeesUSD, avgAPR sql.NullString
	err := db.QueryRowContext(ctx, query, poolID, start, end).Scan(
		&totalFeesA, &totalFeesB, &totalFeesUSD, &avgAPR,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query fee breakdown: %w", err)
	}

	result := map[string]interface{}{
		"total_fees_a":   nullStringToString(totalFeesA),
		"total_fees_b":   nullStringToString(totalFeesB),
		"total_fees_usd": nullStringToString(totalFeesUSD),
		"avg_apr":        nullStringToString(avgAPR),
	}

	return result, nil
}

// GetPoolAPRHistory retrieves APR trend over time
func (db *Database) GetPoolAPRHistory(ctx context.Context, poolID string, start, end time.Time) ([]map[string]interface{}, error) {
	query := `
		SELECT
			period_start as timestamp,
			apr
		FROM dex_pool_statistics
		WHERE pool_id = $1
			AND period = '24h'
			AND period_start >= $2
			AND period_end <= $3
		ORDER BY period_start ASC
	`

	rows, err := db.QueryContext(ctx, query, poolID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query APR history: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var apr string

		err := rows.Scan(&timestamp, &apr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan APR history: %w", err)
		}

		results = append(results, map[string]interface{}{
			"timestamp": timestamp,
			"apr":       apr,
		})
	}

	return results, rows.Err()
}

// ============================================================================
// USER POSITION QUERIES
// ============================================================================

// DEXUserPosition represents a user's LP position
type DEXUserPosition struct {
	ID                 int64      `json:"id"`
	Address            string     `json:"address"`
	PoolID             string     `json:"pool_id"`
	Shares             string     `json:"shares"`
	InitialAmountA     string     `json:"initial_amount_a"`
	InitialAmountB     string     `json:"initial_amount_b"`
	CurrentAmountA     string     `json:"current_amount_a"`
	CurrentAmountB     string     `json:"current_amount_b"`
	EntryPrice         string     `json:"entry_price"`
	EntryHeight        int64      `json:"entry_height"`
	EntryTimestamp     time.Time  `json:"entry_timestamp"`
	EntryTxHash        string     `json:"entry_tx_hash"`
	ExitPrice          *string    `json:"exit_price,omitempty"`
	ExitHeight         *int64     `json:"exit_height,omitempty"`
	ExitTimestamp      *time.Time `json:"exit_timestamp,omitempty"`
	ExitTxHash         *string    `json:"exit_tx_hash,omitempty"`
	FeesEarnedA        string     `json:"fees_earned_a"`
	FeesEarnedB        string     `json:"fees_earned_b"`
	FeesEarnedUSD      string     `json:"fees_earned_usd"`
	ImpermanentLoss    string     `json:"impermanent_loss"`
	TotalReturnPercent string     `json:"total_return_percent"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// GetUserPosition retrieves a specific user position
func (db *Database) GetUserPosition(ctx context.Context, address, poolID string) (*DEXUserPosition, error) {
	query := `
		SELECT
			id, address, pool_id, shares,
			initial_amount_a, initial_amount_b,
			current_amount_a, current_amount_b,
			entry_price, entry_height, entry_timestamp, entry_tx_hash,
			exit_price, exit_height, exit_timestamp, exit_tx_hash,
			fees_earned_a, fees_earned_b, fees_earned_usd,
			impermanent_loss, total_return_percent,
			status, created_at, updated_at
		FROM dex_user_positions
		WHERE address = $1 AND pool_id = $2 AND status = 'active'
		LIMIT 1
	`

	var pos DEXUserPosition
	err := db.QueryRowContext(ctx, query, address, poolID).Scan(
		&pos.ID, &pos.Address, &pos.PoolID, &pos.Shares,
		&pos.InitialAmountA, &pos.InitialAmountB,
		&pos.CurrentAmountA, &pos.CurrentAmountB,
		&pos.EntryPrice, &pos.EntryHeight, &pos.EntryTimestamp, &pos.EntryTxHash,
		&pos.ExitPrice, &pos.ExitHeight, &pos.ExitTimestamp, &pos.ExitTxHash,
		&pos.FeesEarnedA, &pos.FeesEarnedB, &pos.FeesEarnedUSD,
		&pos.ImpermanentLoss, &pos.TotalReturnPercent,
		&pos.Status, &pos.CreatedAt, &pos.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user position: %w", err)
	}

	return &pos, nil
}

// UpsertUserPosition inserts or updates a user position
func (db *Database) UpsertUserPosition(ctx context.Context, pos *DEXUserPosition) error {
	query := `
		INSERT INTO dex_user_positions (
			address, pool_id, shares,
			initial_amount_a, initial_amount_b,
			current_amount_a, current_amount_b,
			entry_price, entry_height, entry_timestamp, entry_tx_hash,
			fees_earned_a, fees_earned_b, fees_earned_usd,
			impermanent_loss, total_return_percent,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (address, pool_id, entry_tx_hash) DO UPDATE SET
			current_amount_a = EXCLUDED.current_amount_a,
			current_amount_b = EXCLUDED.current_amount_b,
			fees_earned_a = EXCLUDED.fees_earned_a,
			fees_earned_b = EXCLUDED.fees_earned_b,
			fees_earned_usd = EXCLUDED.fees_earned_usd,
			impermanent_loss = EXCLUDED.impermanent_loss,
			total_return_percent = EXCLUDED.total_return_percent,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING id
	`

	return db.QueryRowContext(
		ctx, query,
		pos.Address, pos.PoolID, pos.Shares,
		pos.InitialAmountA, pos.InitialAmountB,
		pos.CurrentAmountA, pos.CurrentAmountB,
		pos.EntryPrice, pos.EntryHeight, pos.EntryTimestamp, pos.EntryTxHash,
		pos.FeesEarnedA, pos.FeesEarnedB, pos.FeesEarnedUSD,
		pos.ImpermanentLoss, pos.TotalReturnPercent,
		pos.Status,
	).Scan(&pos.ID)
}

// GetUserDEXPositions retrieves all positions for a user
func (db *Database) GetUserDEXPositions(ctx context.Context, address, status string) ([]DEXUserPosition, error) {
	query := `
		SELECT
			id, address, pool_id, shares,
			initial_amount_a, initial_amount_b,
			current_amount_a, current_amount_b,
			entry_price, entry_height, entry_timestamp, entry_tx_hash,
			exit_price, exit_height, exit_timestamp, exit_tx_hash,
			fees_earned_a, fees_earned_b, fees_earned_usd,
			impermanent_loss, total_return_percent,
			status, created_at, updated_at
		FROM dex_user_positions
		WHERE address = $1
	`

	args := []interface{}{address}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY entry_timestamp DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user positions: %w", err)
	}
	defer rows.Close()

	var positions []DEXUserPosition
	for rows.Next() {
		var pos DEXUserPosition
		err := rows.Scan(
			&pos.ID, &pos.Address, &pos.PoolID, &pos.Shares,
			&pos.InitialAmountA, &pos.InitialAmountB,
			&pos.CurrentAmountA, &pos.CurrentAmountB,
			&pos.EntryPrice, &pos.EntryHeight, &pos.EntryTimestamp, &pos.EntryTxHash,
			&pos.ExitPrice, &pos.ExitHeight, &pos.ExitTimestamp, &pos.ExitTxHash,
			&pos.FeesEarnedA, &pos.FeesEarnedB, &pos.FeesEarnedUSD,
			&pos.ImpermanentLoss, &pos.TotalReturnPercent,
			&pos.Status, &pos.CreatedAt, &pos.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user position: %w", err)
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// GetUserDEXHistory retrieves DEX activity for a user
func (db *Database) GetUserDEXHistory(ctx context.Context, address string, offset, limit int) ([]map[string]interface{}, int, error) {
	// Query both trades and liquidity events
	query := `
		SELECT 'trade' as type, tx_hash, pool_id, timestamp,
			token_in as token_a, amount_in as amount_a,
			token_out as token_b, amount_out as amount_b
		FROM dex_trades
		WHERE trader = $1
		UNION ALL
		SELECT action as type, tx_hash, pool_id, timestamp,
			'' as token_a, amount_a,
			'' as token_b, amount_b
		FROM dex_liquidity
		WHERE provider = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, query, address, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user DEX history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var typ, txHash, poolID, tokenA, amountA, tokenB, amountB string
		var timestamp time.Time

		err := rows.Scan(&typ, &txHash, &poolID, &timestamp, &tokenA, &amountA, &tokenB, &amountB)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user history: %w", err)
		}

		history = append(history, map[string]interface{}{
			"type":      typ,
			"tx_hash":   txHash,
			"pool_id":   poolID,
			"timestamp": timestamp,
			"token_a":   tokenA,
			"amount_a":  amountA,
			"token_b":   tokenB,
			"amount_b":  amountB,
		})
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM dex_trades WHERE trader = $1
			UNION ALL
			SELECT 1 FROM dex_liquidity WHERE provider = $1
		) AS combined
	`
	var total int
	err = db.QueryRowContext(ctx, countQuery, address).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user history: %w", err)
	}

	return history, total, rows.Err()
}

// GetUserDEXAnalytics retrieves comprehensive user analytics
func (db *Database) GetUserDEXAnalytics(ctx context.Context, address string) (map[string]interface{}, error) {
	// Aggregate all user DEX activity
	query := `
		WITH active_positions AS (
			SELECT
				COUNT(*) as position_count,
				SUM(CAST(fees_earned_usd AS NUMERIC)) as total_fees,
				AVG(CAST(total_return_percent AS NUMERIC)) as avg_return
			FROM dex_user_positions
			WHERE address = $1 AND status = 'active'
		),
		trade_stats AS (
			SELECT
				COUNT(*) as trade_count,
				COUNT(DISTINCT pool_id) as pools_traded
			FROM dex_trades
			WHERE trader = $1
		)
		SELECT
			COALESCE(ap.position_count, 0) as active_positions,
			COALESCE(ap.total_fees, 0) as total_fees_earned,
			COALESCE(ap.avg_return, 0) as avg_return_percent,
			COALESCE(ts.trade_count, 0) as total_trades,
			COALESCE(ts.pools_traded, 0) as unique_pools
		FROM active_positions ap
		CROSS JOIN trade_stats ts
	`

	var activePositions, totalTrades, uniquePools int
	var totalFees, avgReturn float64

	err := db.QueryRowContext(ctx, query, address).Scan(
		&activePositions, &totalFees, &avgReturn, &totalTrades, &uniquePools,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query user analytics: %w", err)
	}

	result := map[string]interface{}{
		"active_positions":    activePositions,
		"total_fees_earned":   fmt.Sprintf("%.6f", totalFees),
		"avg_return_percent":  fmt.Sprintf("%.2f", avgReturn),
		"total_trades":        totalTrades,
		"unique_pools_traded": uniquePools,
	}

	return result, nil
}

// ============================================================================
// ANALYTICS CACHE QUERIES
// ============================================================================

// GetCachedAnalytics retrieves cached analytics data
func (db *Database) GetCachedAnalytics(ctx context.Context, cacheKey string) (map[string]interface{}, error) {
	query := `
		SELECT data
		FROM dex_analytics_cache
		WHERE cache_key = $1
			AND expires_at > NOW()
	`

	var dataJSON []byte
	err := db.QueryRowContext(ctx, query, cacheKey).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query cache: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return data, nil
}

// SetCachedAnalytics stores analytics data in cache
func (db *Database) SetCachedAnalytics(ctx context.Context, cacheKey, cacheType string, data map[string]interface{}, ttl time.Duration) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT INTO dex_analytics_cache (cache_key, cache_type, data, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cache_key) DO UPDATE SET
			data = EXCLUDED.data,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
	`

	expiresAt := time.Now().Add(ttl)
	_, err = db.ExecContext(ctx, query, cacheKey, cacheType, dataJSON, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert cache: %w", err)
	}

	return nil
}

// ============================================================================
// DEX ANALYTICS SUMMARY
// ============================================================================

// GetDEXAnalyticsSummary retrieves overall DEX statistics
func (db *Database) GetDEXAnalyticsSummary(ctx context.Context) (map[string]interface{}, error) {
	query := `
		WITH pool_stats AS (
			SELECT
				COUNT(*) as total_pools,
				SUM(CAST(tvl AS NUMERIC)) as total_tvl,
				SUM(CAST(volume_24h AS NUMERIC)) as total_volume_24h
			FROM dex_pools
		),
		trade_stats AS (
			SELECT
				COUNT(*) as total_trades_24h,
				COUNT(DISTINCT trader) as unique_traders_24h
			FROM dex_trades
			WHERE timestamp > NOW() - INTERVAL '24 hours'
		),
		lp_stats AS (
			SELECT
				COUNT(DISTINCT address) as active_lps
			FROM dex_user_positions
			WHERE status = 'active'
		)
		SELECT
			ps.total_pools,
			ps.total_tvl,
			ps.total_volume_24h,
			ts.total_trades_24h,
			ts.unique_traders_24h,
			ls.active_lps
		FROM pool_stats ps
		CROSS JOIN trade_stats ts
		CROSS JOIN lp_stats ls
	`

	var totalPools, totalTrades, uniqueTraders, activeLPs int
	var totalTVL, totalVolume float64

	err := db.QueryRowContext(ctx, query).Scan(
		&totalPools, &totalTVL, &totalVolume,
		&totalTrades, &uniqueTraders, &activeLPs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DEX summary: %w", err)
	}

	result := map[string]interface{}{
		"total_pools":        totalPools,
		"total_tvl":          fmt.Sprintf("%.2f", totalTVL),
		"total_volume_24h":   fmt.Sprintf("%.2f", totalVolume),
		"total_trades_24h":   totalTrades,
		"unique_traders_24h": uniqueTraders,
		"active_lps":         activeLPs,
	}

	return result, nil
}

// GetTopTradingPairs retrieves top pools by volume
func (db *Database) GetTopTradingPairs(ctx context.Context, period string, limit int) ([]map[string]interface{}, error) {
	var volumeCol string
	switch period {
	case "24h":
		volumeCol = "volume_24h"
	case "7d":
		volumeCol = "volume_7d"
	case "30d":
		volumeCol = "volume_30d"
	default:
		volumeCol = "volume_24h"
	}

	query := fmt.Sprintf(`
		SELECT
			pool_id,
			token_a,
			token_b,
			%s as volume,
			tvl,
			apr
		FROM dex_pools
		ORDER BY CAST(%s AS NUMERIC) DESC
		LIMIT $1
	`, volumeCol, volumeCol)

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top pairs: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var poolID, tokenA, tokenB, volume, tvl, apr string

		err := rows.Scan(&poolID, &tokenA, &tokenB, &volume, &tvl, &apr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top pair: %w", err)
		}

		results = append(results, map[string]interface{}{
			"pool_id": poolID,
			"token_a": tokenA,
			"token_b": tokenB,
			"volume":  volume,
			"tvl":     tvl,
			"apr":     apr,
		})
	}

	return results, rows.Err()
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "0"
}
