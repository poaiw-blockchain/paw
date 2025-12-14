package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// TestDatabaseConfig holds test database configuration
var testDBConfig = Config{
	URL:            "postgres://postgres:postgres@localhost:5432/paw_explorer_test?sslmode=disable",
	MaxConnections: 10,
	MaxIdle:        5,
	ConnMaxLife:    time.Hour,
}

// setupTestDB creates a test database connection and initializes schema
func setupTestDB(t *testing.T) *Database {
	db, err := New(testDBConfig)
	require.NoError(t, err, "Failed to create test database connection")

	// Initialize schema
	err = db.InitSchema()
	require.NoError(t, err, "Failed to initialize schema")

	// Clean all tables
	cleanTestDB(t, db)

	return db
}

// cleanTestDB truncates all tables
func cleanTestDB(t *testing.T, db *Database) {
	tables := []string{
		"dex_trades", "dex_liquidity", "dex_pools", "dex_swaps",
		"dex_pool_price_history", "dex_pool_statistics", "dex_user_positions",
		"dex_analytics_cache", "oracle_prices", "compute_requests",
		"validator_uptime", "validators", "events", "transactions",
		"blocks", "accounts", "failed_blocks", "indexing_checkpoints",
		"indexing_metrics", "indexing_progress", "indexer_state",
	}

	for _, table := range tables {
		_, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// teardownTestDB closes the test database connection
func teardownTestDB(t *testing.T, db *Database) {
	err := db.Close()
	assert.NoError(t, err, "Failed to close test database connection")
}

// ============================================================================
// BLOCK CRUD TESTS
// ============================================================================

func TestInsertBlock(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, err := db.BeginTx()
	require.NoError(t, err)
	defer tx.Rollback()

	block := Block{
		Height:          1000,
		Hash:            "ABC123DEF456",
		ProposerAddress: "pawvaloper1abc123",
		Time:            time.Now().UTC(),
		TxCount:         5,
		GasUsed:         100000,
		GasWanted:       120000,
		EvidenceCount:   0,
	}

	err = db.InsertBlock(tx, block)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify insertion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height = $1", block.Height).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertBlockConflict(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	block := Block{
		Height:          2000,
		Hash:            "HASH1",
		ProposerAddress: "pawvaloper1test",
		Time:            time.Now().UTC(),
		TxCount:         3,
		GasUsed:         50000,
		GasWanted:       60000,
		EvidenceCount:   0,
	}

	// Insert first time
	tx1, _ := db.BeginTx()
	err := db.InsertBlock(tx1, block)
	require.NoError(t, err)
	tx1.Commit()

	// Insert again with updated data
	block.Hash = "HASH2_UPDATED"
	block.TxCount = 10

	tx2, _ := db.BeginTx()
	err = db.InsertBlock(tx2, block)
	require.NoError(t, err)
	tx2.Commit()

	// Verify update occurred
	var hash string
	var txCount int
	err = db.QueryRow("SELECT hash, tx_count FROM blocks WHERE height = $1", block.Height).Scan(&hash, &txCount)
	require.NoError(t, err)
	assert.Equal(t, "HASH2_UPDATED", hash)
	assert.Equal(t, 10, txCount)
}

func TestGetBlockByHeight(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Insert test block
	tx, _ := db.BeginTx()
	block := Block{
		Height:          3000,
		Hash:            "TESTHASH",
		ProposerAddress: "pawvaloper1test",
		Time:            time.Now().UTC(),
		TxCount:         7,
		GasUsed:         75000,
		GasWanted:       80000,
		EvidenceCount:   1,
	}
	db.InsertBlock(tx, block)
	tx.Commit()

	// Retrieve block
	retrieved, err := db.GetBlockByHeight(3000)
	require.NoError(t, err)
	assert.Equal(t, block.Height, retrieved.Height)
	assert.Equal(t, block.Hash, retrieved.Hash)
	assert.Equal(t, block.ProposerAddress, retrieved.ProposerAddress)
	assert.Equal(t, block.TxCount, retrieved.TxCount)
}

func TestGetBlocks_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Insert multiple blocks
	tx, _ := db.BeginTx()
	for i := 1; i <= 50; i++ {
		block := Block{
			Height:          int64(i),
			Hash:            "HASH" + string(rune(i)),
			ProposerAddress: "pawvaloper1test",
			Time:            time.Now().UTC(),
			TxCount:         i % 10,
			GasUsed:         int64(i * 1000),
			GasWanted:       int64(i * 1200),
			EvidenceCount:   0,
		}
		db.InsertBlock(tx, block)
	}
	tx.Commit()

	// Test pagination
	blocks, total, err := db.GetBlocks(0, 10)
	require.NoError(t, err)
	assert.Equal(t, 10, len(blocks))
	assert.Equal(t, 50, total)

	// Test second page
	blocks, total, err = db.GetBlocks(10, 10)
	require.NoError(t, err)
	assert.Equal(t, 10, len(blocks))
	assert.Equal(t, 50, total)
}

// ============================================================================
// TRANSACTION CRUD TESTS
// ============================================================================

func TestInsertTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, _ := db.BeginTx()

	// Insert block first
	block := Block{
		Height:          100,
		Hash:            "BLOCKHASH",
		ProposerAddress: "pawvaloper1test",
		Time:            time.Now().UTC(),
		TxCount:         1,
		GasUsed:         10000,
		GasWanted:       12000,
		EvidenceCount:   0,
	}
	db.InsertBlock(tx, block)

	// Insert transaction
	transaction := Transaction{
		Hash:        "TXHASH123",
		BlockHeight: 100,
		TxIndex:     0,
		Type:        "/cosmos.bank.v1beta1.MsgSend",
		Sender:      "paw1sender",
		Status:      "success",
		Code:        0,
		GasUsed:     10000,
		GasWanted:   12000,
		FeeAmount:   "1000",
		FeeDenom:    "upaw",
		Memo:        "test memo",
		RawLog:      "[]",
		Time:        time.Now().UTC(),
		Messages:    json.RawMessage(`[{"type":"send"}]`),
		Events:      json.RawMessage(`[{"type":"transfer"}]`),
	}

	err := db.InsertTransaction(tx, transaction)
	require.NoError(t, err)

	tx.Commit()

	// Verify
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM transactions WHERE hash = $1", transaction.Hash).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetTransactionByHash(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, _ := db.BeginTx()

	block := Block{Height: 200, Hash: "BH", ProposerAddress: "p", Time: time.Now().UTC(), TxCount: 1, GasUsed: 1000, GasWanted: 1200, EvidenceCount: 0}
	db.InsertBlock(tx, block)

	transaction := Transaction{
		Hash:        "FINDME",
		BlockHeight: 200,
		TxIndex:     0,
		Type:        "/cosmos.bank.v1beta1.MsgSend",
		Sender:      "paw1sender",
		Status:      "success",
		Code:        0,
		GasUsed:     1000,
		GasWanted:   1200,
		FeeAmount:   "100",
		FeeDenom:    "upaw",
		Memo:        "",
		RawLog:      "[]",
		Time:        time.Now().UTC(),
		Messages:    json.RawMessage(`[]`),
		Events:      json.RawMessage(`[]`),
	}
	db.InsertTransaction(tx, transaction)
	tx.Commit()

	// Retrieve
	retrieved, err := db.GetTransactionByHash("FINDME")
	require.NoError(t, err)
	assert.Equal(t, "FINDME", retrieved.Hash)
	assert.Equal(t, int64(200), retrieved.BlockHeight)
	assert.Equal(t, "paw1sender", retrieved.Sender)
}

func TestGetTransactionsByHeight(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, _ := db.BeginTx()

	block := Block{Height: 300, Hash: "BH300", ProposerAddress: "p", Time: time.Now().UTC(), TxCount: 3, GasUsed: 3000, GasWanted: 3600, EvidenceCount: 0}
	db.InsertBlock(tx, block)

	for i := 0; i < 3; i++ {
		transaction := Transaction{
			Hash:        "TX" + string(rune(i)),
			BlockHeight: 300,
			TxIndex:     i,
			Type:        "/cosmos.bank.v1beta1.MsgSend",
			Sender:      "paw1sender",
			Status:      "success",
			Code:        0,
			GasUsed:     1000,
			GasWanted:   1200,
			FeeAmount:   "100",
			FeeDenom:    "upaw",
			Memo:        "",
			RawLog:      "[]",
			Time:        time.Now().UTC(),
			Messages:    json.RawMessage(`[]`),
			Events:      json.RawMessage(`[]`),
		}
		db.InsertTransaction(tx, transaction)
	}
	tx.Commit()

	txs, err := db.GetTransactionsByHeight(300)
	require.NoError(t, err)
	assert.Equal(t, 3, len(txs))
}

// ============================================================================
// DEX QUERIES TESTS
// ============================================================================

func TestInsertPriceHistory(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()
	priceHistory := &DEXPriceHistory{
		PoolID:      "pool1",
		Timestamp:   time.Now().UTC().Truncate(time.Hour),
		BlockHeight: 1000,
		Open:        "1.00",
		High:        "1.10",
		Low:         "0.95",
		Close:       "1.05",
		Volume:      "10000",
		LiquidityA:  "50000",
		LiquidityB:  "50000",
		PriceAToB:   "1.00",
		PriceBToA:   "1.00",
	}

	err := db.InsertPriceHistory(ctx, priceHistory)
	require.NoError(t, err)
	assert.NotZero(t, priceHistory.ID)
}

func TestGetPoolPriceHistory(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	// Insert multiple price history entries
	for i := 0; i < 24; i++ {
		ph := &DEXPriceHistory{
			PoolID:      "pool1",
			Timestamp:   now.Add(time.Duration(i) * time.Hour),
			BlockHeight: int64(1000 + i),
			Open:        "1.00",
			High:        "1.10",
			Low:         "0.95",
			Close:       "1.05",
			Volume:      "1000",
			LiquidityA:  "50000",
			LiquidityB:  "50000",
			PriceAToB:   "1.00",
			PriceBToA:   "1.00",
		}
		db.InsertPriceHistory(ctx, ph)
	}

	// Retrieve price history
	start := now
	end := now.Add(24 * time.Hour)
	history, err := db.GetPoolPriceHistory(ctx, "pool1", start, end, "1h")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 24)
}

func TestUpsertUserPosition(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()
	position := &DEXUserPosition{
		Address:        "paw1user",
		PoolID:         "pool1",
		Shares:         "1000",
		InitialAmountA: "500",
		InitialAmountB: "500",
		CurrentAmountA: "510",
		CurrentAmountB: "510",
		EntryPrice:     "1.00",
		EntryHeight:    1000,
		EntryTimestamp: time.Now().UTC(),
		EntryTxHash:    "TX1",
		FeesEarnedA:    "10",
		FeesEarnedB:    "10",
		FeesEarnedUSD:  "20",
		ImpermanentLoss: "0",
		TotalReturnPercent: "2.0",
		Status:         "active",
	}

	err := db.UpsertUserPosition(ctx, position)
	require.NoError(t, err)
	assert.NotZero(t, position.ID)

	// Update position
	position.CurrentAmountA = "520"
	position.FeesEarnedA = "20"

	err = db.UpsertUserPosition(ctx, position)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := db.GetUserPosition(ctx, "paw1user", "pool1")
	require.NoError(t, err)
	assert.Equal(t, "520", retrieved.CurrentAmountA)
	assert.Equal(t, "20", retrieved.FeesEarnedA)
}

func TestGetUserDEXPositions(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()

	// Insert multiple positions for same user
	for i := 1; i <= 3; i++ {
		position := &DEXUserPosition{
			Address:        "paw1user",
			PoolID:         "pool" + string(rune(i)),
			Shares:         "1000",
			InitialAmountA: "500",
			InitialAmountB: "500",
			CurrentAmountA: "510",
			CurrentAmountB: "510",
			EntryPrice:     "1.00",
			EntryHeight:    int64(1000 + i),
			EntryTimestamp: time.Now().UTC(),
			EntryTxHash:    "TX" + string(rune(i)),
			FeesEarnedA:    "10",
			FeesEarnedB:    "10",
			FeesEarnedUSD:  "20",
			ImpermanentLoss: "0",
			TotalReturnPercent: "2.0",
			Status:         "active",
		}
		db.UpsertUserPosition(ctx, position)
	}

	positions, err := db.GetUserDEXPositions(ctx, "paw1user", "active")
	require.NoError(t, err)
	assert.Equal(t, 3, len(positions))
}

func TestGetUserDEXAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()

	// Insert position
	position := &DEXUserPosition{
		Address:        "paw1analytics",
		PoolID:         "pool1",
		Shares:         "1000",
		InitialAmountA: "500",
		InitialAmountB: "500",
		CurrentAmountA: "510",
		CurrentAmountB: "510",
		EntryPrice:     "1.00",
		EntryHeight:    1000,
		EntryTimestamp: time.Now().UTC(),
		EntryTxHash:    "TX1",
		FeesEarnedA:    "10",
		FeesEarnedB:    "10",
		FeesEarnedUSD:  "100.50",
		ImpermanentLoss: "0",
		TotalReturnPercent: "5.5",
		Status:         "active",
	}
	db.UpsertUserPosition(ctx, position)

	analytics, err := db.GetUserDEXAnalytics(ctx, "paw1analytics")
	require.NoError(t, err)
	assert.NotNil(t, analytics)
	assert.Equal(t, 1, analytics["active_positions"])
}

// ============================================================================
// COMPLEX QUERY TESTS
// ============================================================================

func TestGetPoolStatistics(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	// This would require actual pool statistics data to be inserted
	// Testing with empty result
	stats, err := db.GetPoolStatistics(ctx, "pool1", "24h", now.Add(-24*time.Hour), now)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestGetCachedAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()

	// Set cache
	data := map[string]interface{}{
		"total_volume": "10000",
		"total_tvl":    "50000",
	}
	err := db.SetCachedAnalytics(ctx, "test_key", "analytics", data, time.Minute)
	require.NoError(t, err)

	// Get cache
	retrieved, err := db.GetCachedAnalytics(ctx, "test_key")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "10000", retrieved["total_volume"])
}

func TestGetDEXAnalyticsSummary(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx := context.Background()

	// Test with empty database
	summary, err := db.GetDEXAnalyticsSummary(ctx)
	require.NoError(t, err)
	assert.NotNil(t, summary)
}

// ============================================================================
// TRANSACTION HANDLING TESTS
// ============================================================================

func TestTransactionCommit(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, err := db.BeginTx()
	require.NoError(t, err)

	block := Block{
		Height:          5000,
		Hash:            "TX_TEST",
		ProposerAddress: "p",
		Time:            time.Now().UTC(),
		TxCount:         1,
		GasUsed:         1000,
		GasWanted:       1200,
		EvidenceCount:   0,
	}
	err = db.InsertBlock(tx, block)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify data persisted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height = $1", 5000).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	tx, err := db.BeginTx()
	require.NoError(t, err)

	block := Block{
		Height:          6000,
		Hash:            "ROLLBACK_TEST",
		ProposerAddress: "p",
		Time:            time.Now().UTC(),
		TxCount:         1,
		GasUsed:         1000,
		GasWanted:       1200,
		EvidenceCount:   0,
	}
	err = db.InsertBlock(tx, block)
	require.NoError(t, err)

	err = tx.Rollback()
	require.NoError(t, err)

	// Verify data not persisted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height = $1", 6000).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ============================================================================
// CONNECTION POOL TESTS
// ============================================================================

func TestConnectionPoolBehavior(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Test concurrent connections
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func() {
			err := db.Ping()
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestQueryTimeout(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout (assuming query takes longer than 1ms)
	time.Sleep(2 * time.Millisecond)
	_, err := db.QueryContext(ctx, "SELECT pg_sleep(1)")
	assert.Error(t, err)
}

// ============================================================================
// BENCHMARK TESTS
// ============================================================================

func BenchmarkInsertBlock(b *testing.B) {
	db, _ := New(testDBConfig)
	defer db.Close()
	cleanTestDB(&testing.T{}, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, _ := db.BeginTx()
		block := Block{
			Height:          int64(i),
			Hash:            "HASH" + string(rune(i)),
			ProposerAddress: "p",
			Time:            time.Now().UTC(),
			TxCount:         1,
			GasUsed:         1000,
			GasWanted:       1200,
			EvidenceCount:   0,
		}
		db.InsertBlock(tx, block)
		tx.Commit()
	}
}

func BenchmarkGetBlockByHeight(b *testing.B) {
	db, _ := New(testDBConfig)
	defer db.Close()
	cleanTestDB(&testing.T{}, db)

	// Insert test data
	tx, _ := db.BeginTx()
	for i := 1; i <= 1000; i++ {
		block := Block{
			Height:          int64(i),
			Hash:            "HASH",
			ProposerAddress: "p",
			Time:            time.Now().UTC(),
			TxCount:         1,
			GasUsed:         1000,
			GasWanted:       1200,
			EvidenceCount:   0,
		}
		db.InsertBlock(tx, block)
	}
	tx.Commit()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetBlockByHeight(int64((i % 1000) + 1))
	}
}

func BenchmarkGetPoolPriceHistory(b *testing.B) {
	db, _ := New(testDBConfig)
	defer db.Close()
	cleanTestDB(&testing.T{}, db)

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	// Insert test data
	for i := 0; i < 100; i++ {
		ph := &DEXPriceHistory{
			PoolID:      "pool1",
			Timestamp:   now.Add(time.Duration(i) * time.Hour),
			BlockHeight: int64(1000 + i),
			Open:        "1.00",
			High:        "1.10",
			Low:         "0.95",
			Close:       "1.05",
			Volume:      "1000",
			LiquidityA:  "50000",
			LiquidityB:  "50000",
			PriceAToB:   "1.00",
			PriceBToA:   "1.00",
		}
		db.InsertPriceHistory(ctx, ph)
	}

	start := now
	end := now.Add(100 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetPoolPriceHistory(ctx, "pool1", start, end, "1h")
	}
}
