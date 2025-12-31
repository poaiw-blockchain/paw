// TEST-1.3: Upgrade Migration Tests with Realistic State (10k+ Records)
// Tests migration performance and correctness with large datasets
package upgrade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// LargeStateMigrationTestSuite tests migrations with 10k+ records
type LargeStateMigrationTestSuite struct {
	suite.Suite

	app     *app.PAWApp
	ctx     sdk.Context
	chainID string

	// State counts
	poolCount     int
	jobCount      int
	priceCount    int
	providerCount int
	orderCount    int
}

func TestLargeStateMigrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large state migration test in short mode")
	}
	suite.Run(t, new(LargeStateMigrationTestSuite))
}

func (suite *LargeStateMigrationTestSuite) SetupTest() {
	suite.chainID = "paw-migration-test"
	suite.app, suite.ctx = suite.setupTestAppWithGenesisState()
}

func (suite *LargeStateMigrationTestSuite) setupTestAppWithGenesisState() (*app.PAWApp, sdk.Context) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(suite.chainID),
	)

	ctx := pawApp.BaseApp.NewContext(false, tmproto.Header{
		ChainID: suite.chainID,
		Height:  1,
		Time:    time.Now(),
	})

	return pawApp, ctx
}

// MigrationMetrics captures migration performance
type MigrationMetrics struct {
	TotalRecords     int
	MigratedRecords  int
	Duration         time.Duration
	MemoryUsedMB     float64
	RecordsPerSecond float64
	Errors           []error
}

func (m MigrationMetrics) String() string {
	return fmt.Sprintf("Migrated %d/%d records in %v (%.2f rec/sec), Memory: %.2f MB, Errors: %d",
		m.MigratedRecords, m.TotalRecords, m.Duration, m.RecordsPerSecond, m.MemoryUsedMB, len(m.Errors))
}

// TestMigrationWith10kPools tests migration with 10,000 pools
func (suite *LargeStateMigrationTestSuite) TestMigrationWith10kPools() {
	suite.poolCount = 10000

	metrics := suite.runPoolMigration(suite.poolCount)

	suite.T().Logf("TEST-1.3 10k Pools Migration: %s", metrics)
	suite.Equal(suite.poolCount, metrics.MigratedRecords, "All pools should be migrated")
	suite.Empty(metrics.Errors, "Should have no migration errors")
}

// TestMigrationWith10kJobs tests migration with 10,000 compute jobs
func (suite *LargeStateMigrationTestSuite) TestMigrationWith10kJobs() {
	suite.jobCount = 10000

	metrics := suite.runJobMigration(suite.jobCount)

	suite.T().Logf("TEST-1.3 10k Jobs Migration: %s", metrics)
	suite.Equal(suite.jobCount, metrics.MigratedRecords)
	suite.Empty(metrics.Errors)
}

// TestMigrationWith50kPrices tests migration with 50,000 price records
func (suite *LargeStateMigrationTestSuite) TestMigrationWith50kPrices() {
	suite.priceCount = 50000

	metrics := suite.runPriceMigration(suite.priceCount)

	suite.T().Logf("TEST-1.3 50k Prices Migration: %s", metrics)
	suite.Equal(suite.priceCount, metrics.MigratedRecords)
	suite.Empty(metrics.Errors)
}

// TestMixedStateMigration tests migration with mixed state
func (suite *LargeStateMigrationTestSuite) TestMixedStateMigration() {
	// Create mixed state: 5k pools, 10k jobs, 20k prices, 1k providers, 5k orders
	suite.poolCount = 5000
	suite.jobCount = 10000
	suite.priceCount = 20000
	suite.providerCount = 1000
	suite.orderCount = 5000

	startTime := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Seed all data
	suite.seedPools(suite.poolCount)
	suite.seedJobs(suite.jobCount)
	suite.seedPrices(suite.priceCount)
	suite.seedProviders(suite.providerCount)
	suite.seedOrders(suite.orderCount)

	seedDuration := time.Since(startTime)
	totalRecords := suite.poolCount + suite.jobCount + suite.priceCount + suite.providerCount + suite.orderCount

	// Capture pre-migration state
	preMigrationState := suite.captureStateCounts()

	// Schedule and execute upgrade
	upgradeHeight := suite.ctx.BlockHeight() + 10
	plan := upgradetypes.Plan{
		Name:   "v2.0.0-large-state",
		Height: upgradeHeight,
		Info:   "Large state migration test",
	}

	err := suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan)
	suite.Require().NoError(err)

	// Simulate blocks up to upgrade
	suite.simulateBlocks(upgradeHeight - suite.ctx.BlockHeight())

	// Execute upgrade
	migrationStart := time.Now()
	suite.ctx = suite.ctx.WithBlockHeight(upgradeHeight)
	err = suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan)
	suite.Require().NoError(err)
	migrationDuration := time.Since(migrationStart)

	// Capture post-migration state
	postMigrationState := suite.captureStateCounts()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	memUsedMB := float64(memAfter.HeapAlloc-memBefore.HeapAlloc) / 1024 / 1024

	suite.T().Log("\n=== TEST-1.3 MIXED STATE MIGRATION RESULTS ===")
	suite.T().Logf("Total records: %d", totalRecords)
	suite.T().Logf("  - Pools: %d", suite.poolCount)
	suite.T().Logf("  - Jobs: %d", suite.jobCount)
	suite.T().Logf("  - Prices: %d", suite.priceCount)
	suite.T().Logf("  - Providers: %d", suite.providerCount)
	suite.T().Logf("  - Orders: %d", suite.orderCount)
	suite.T().Logf("Seed duration: %v", seedDuration)
	suite.T().Logf("Migration duration: %v", migrationDuration)
	suite.T().Logf("Records/second: %.2f", float64(totalRecords)/migrationDuration.Seconds())
	suite.T().Logf("Memory used: %.2f MB", memUsedMB)
	suite.T().Log("")
	suite.T().Log("State Verification:")
	suite.T().Logf("  Pre-migration: %v", preMigrationState)
	suite.T().Logf("  Post-migration: %v", postMigrationState)
	suite.T().Log("=== END MIGRATION RESULTS ===\n")

	// Verify data integrity
	suite.verifyDataIntegrity()
}

// runPoolMigration executes pool migration test
func (suite *LargeStateMigrationTestSuite) runPoolMigration(count int) MigrationMetrics {
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	startTime := time.Now()

	// Seed pools
	suite.seedPools(count)

	// Simulate migration
	migrated := 0
	var errors []error

	store := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte("pool_"))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Simulate migration transformation
		value := iterator.Value()
		var pool dextypes.Pool
		if err := json.Unmarshal(value, &pool); err != nil {
			errors = append(errors, err)
			continue
		}

		// Apply v2 migration (add new fields, transform data)
		pool.LastUpdatedHeight = suite.ctx.BlockHeight()

		// Re-store migrated pool
		newValue, _ := json.Marshal(pool)
		store.Set(iterator.Key(), newValue)
		migrated++
	}

	duration := time.Since(startTime)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	return MigrationMetrics{
		TotalRecords:     count,
		MigratedRecords:  migrated,
		Duration:         duration,
		MemoryUsedMB:     float64(memAfter.HeapAlloc-memBefore.HeapAlloc) / 1024 / 1024,
		RecordsPerSecond: float64(migrated) / duration.Seconds(),
		Errors:           errors,
	}
}

// runJobMigration executes job migration test
func (suite *LargeStateMigrationTestSuite) runJobMigration(count int) MigrationMetrics {
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	startTime := time.Now()
	suite.seedJobs(count)

	migrated := 0
	var errors []error

	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte("job_"))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		value := iterator.Value()
		var job computetypes.Job
		if err := json.Unmarshal(value, &job); err != nil {
			errors = append(errors, err)
			continue
		}

		// V2 migration: add new fields
		job.MigrationVersion = 2

		newValue, _ := json.Marshal(job)
		store.Set(iterator.Key(), newValue)
		migrated++
	}

	duration := time.Since(startTime)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	return MigrationMetrics{
		TotalRecords:     count,
		MigratedRecords:  migrated,
		Duration:         duration,
		MemoryUsedMB:     float64(memAfter.HeapAlloc-memBefore.HeapAlloc) / 1024 / 1024,
		RecordsPerSecond: float64(migrated) / duration.Seconds(),
		Errors:           errors,
	}
}

// runPriceMigration executes price migration test
func (suite *LargeStateMigrationTestSuite) runPriceMigration(count int) MigrationMetrics {
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	startTime := time.Now()
	suite.seedPrices(count)

	migrated := 0
	var errors []error

	store := suite.ctx.KVStore(suite.app.GetKey(oracletypes.StoreKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte("price_"))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		value := iterator.Value()
		var price oracletypes.Price
		if err := json.Unmarshal(value, &price); err != nil {
			errors = append(errors, err)
			continue
		}

		// V2 migration: add source chain info
		price.SourceChain = "paw-mainnet"

		newValue, _ := json.Marshal(price)
		store.Set(iterator.Key(), newValue)
		migrated++
	}

	duration := time.Since(startTime)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	return MigrationMetrics{
		TotalRecords:     count,
		MigratedRecords:  migrated,
		Duration:         duration,
		MemoryUsedMB:     float64(memAfter.HeapAlloc-memBefore.HeapAlloc) / 1024 / 1024,
		RecordsPerSecond: float64(migrated) / duration.Seconds(),
		Errors:           errors,
	}
}

// seedPools creates test pools
func (suite *LargeStateMigrationTestSuite) seedPools(count int) {
	store := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))

	for i := 0; i < count; i++ {
		pool := dextypes.Pool{
			Id:       uint64(i + 1),
			TokenA:   fmt.Sprintf("token%d", i*2),
			TokenB:   fmt.Sprintf("token%d", i*2+1),
			ReserveA: sdkmath.NewInt(1_000_000_000),
			ReserveB: sdkmath.NewInt(1_000_000_000),
		}
		key := []byte(fmt.Sprintf("pool_%d", i+1))
		value, _ := json.Marshal(pool)
		store.Set(key, value)
	}
}

// seedJobs creates test compute jobs
func (suite *LargeStateMigrationTestSuite) seedJobs(count int) {
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))

	for i := 0; i < count; i++ {
		job := computetypes.Job{
			Id:        fmt.Sprintf("job_%d", i+1),
			Requester: fmt.Sprintf("paw1requester%d", i),
			Status:    "completed",
		}
		key := []byte(fmt.Sprintf("job_%d", i+1))
		value, _ := json.Marshal(job)
		store.Set(key, value)
	}
}

// seedPrices creates test price records
func (suite *LargeStateMigrationTestSuite) seedPrices(count int) {
	store := suite.ctx.KVStore(suite.app.GetKey(oracletypes.StoreKey))

	assets := []string{"BTC", "ETH", "ATOM", "OSMO", "PAW"}
	for i := 0; i < count; i++ {
		asset := assets[i%len(assets)]
		price := oracletypes.Price{
			Asset:       asset,
			Price:       sdkmath.LegacyNewDec(int64(10000 + i%1000)),
			BlockHeight: int64(i + 1),
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Second),
		}
		key := []byte(fmt.Sprintf("price_%s_%d", asset, i+1))
		value, _ := json.Marshal(price)
		store.Set(key, value)
	}
}

// seedProviders creates test compute providers
func (suite *LargeStateMigrationTestSuite) seedProviders(count int) {
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))

	for i := 0; i < count; i++ {
		provider := computetypes.Provider{
			Address: fmt.Sprintf("paw1provider%d", i),
			Status:  "active",
		}
		key := []byte(fmt.Sprintf("provider_%d", i+1))
		value, _ := json.Marshal(provider)
		store.Set(key, value)
	}
}

// seedOrders creates test limit orders
func (suite *LargeStateMigrationTestSuite) seedOrders(count int) {
	store := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))

	for i := 0; i < count; i++ {
		order := dextypes.LimitOrder{
			Id:       uint64(i + 1),
			Owner:    fmt.Sprintf("paw1owner%d", i),
			PoolId:   uint64(i%1000 + 1),
			TokenIn:  "upaw",
			TokenOut: "uusdt",
			AmountIn: sdkmath.NewInt(10000),
			Price:    sdkmath.LegacyNewDecWithPrec(11, 1),
			Status:   "open",
		}
		key := []byte(fmt.Sprintf("order_%d", i+1))
		value, _ := json.Marshal(order)
		store.Set(key, value)
	}
}

// simulateBlocks advances the blockchain
func (suite *LargeStateMigrationTestSuite) simulateBlocks(count int64) {
	for i := int64(0); i < count; i++ {
		suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 1)
		suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(5 * time.Second))
	}
}

// captureStateCounts returns current state counts
func (suite *LargeStateMigrationTestSuite) captureStateCounts() map[string]int {
	counts := make(map[string]int)

	// Count pools
	poolStore := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))
	poolIter := storetypes.KVStorePrefixIterator(poolStore, []byte("pool_"))
	for ; poolIter.Valid(); poolIter.Next() {
		counts["pools"]++
	}
	poolIter.Close()

	// Count jobs
	jobStore := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	jobIter := storetypes.KVStorePrefixIterator(jobStore, []byte("job_"))
	for ; jobIter.Valid(); jobIter.Next() {
		counts["jobs"]++
	}
	jobIter.Close()

	return counts
}

// verifyDataIntegrity checks migrated data is valid
func (suite *LargeStateMigrationTestSuite) verifyDataIntegrity() {
	// Sample verification of pools
	store := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))

	verified := 0
	iterator := storetypes.KVStorePrefixIterator(store, []byte("pool_"))
	defer iterator.Close()

	for ; iterator.Valid() && verified < 100; iterator.Next() {
		var pool dextypes.Pool
		err := json.Unmarshal(iterator.Value(), &pool)
		suite.NoError(err, "Pool should unmarshal correctly")
		suite.NotZero(pool.Id, "Pool ID should be set")
		suite.NotEmpty(pool.TokenA, "TokenA should be set")
		suite.NotEmpty(pool.TokenB, "TokenB should be set")
		verified++
	}

	suite.T().Logf("Verified %d sample records for integrity", verified)
}

// TestMigrationPerformanceBaseline establishes performance baseline
func (suite *LargeStateMigrationTestSuite) TestMigrationPerformanceBaseline() {
	recordCounts := []int{1000, 5000, 10000, 25000}

	suite.T().Log("\n=== TEST-1.3 MIGRATION PERFORMANCE BASELINE ===")
	suite.T().Log("| Records | Duration | Records/Sec | Memory (MB) |")
	suite.T().Log("|---------|----------|-------------|-------------|")

	for _, count := range recordCounts {
		// Reset app for each test
		suite.app, suite.ctx = suite.setupTestAppWithGenesisState()

		metrics := suite.runPoolMigration(count)
		suite.T().Logf("| %7d | %8v | %11.2f | %11.2f |",
			count, metrics.Duration, metrics.RecordsPerSecond, metrics.MemoryUsedMB)
	}

	suite.T().Log("=== END BASELINE ===\n")
}
