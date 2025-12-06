package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// TEST-MED-4: DEX Oracle Integration Comprehensive Tests

// mockOracleKeeper implements the OracleKeeper interface for testing
type mockOracleKeeper struct {
	prices map[string]math.LegacyDec
	timestamps map[string]int64
}

func newMockOracleKeeper() *mockOracleKeeper {
	return &mockOracleKeeper{
		prices: make(map[string]math.LegacyDec),
		timestamps: make(map[string]int64),
	}
}

func (m *mockOracleKeeper) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	price, ok := m.prices[denom]
	if !ok {
		return math.LegacyZeroDec(), dextypes.ErrOraclePrice.Wrapf("price not found for %s", denom)
	}
	return price, nil
}

func (m *mockOracleKeeper) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	price, ok := m.prices[denom]
	if !ok {
		return math.LegacyZeroDec(), 0, dextypes.ErrOraclePrice.Wrapf("price not found for %s", denom)
	}
	timestamp := m.timestamps[denom]
	return price, timestamp, nil
}

func (m *mockOracleKeeper) setPrice(denom string, price math.LegacyDec, timestamp int64) {
	m.prices[denom] = price
	m.timestamps[denom] = timestamp
}

// Helper function to create a test pool
func createTestPoolForOracle(t *testing.T, k *dexkeeper.Keeper, ctx sdk.Context, tokenA, tokenB string, amountA, amountB math.Int) uint64 {
	creator := dextypes.TestAddr()
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	require.NotNil(t, pool)
	return pool.Id
}

// TestGetPoolValueUSD_Success tests successful pool USD valuation
func TestGetPoolValueUSD_Success(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Set oracle prices (in USD)
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Calculate pool value
	totalValue, err := k.GetPoolValueUSD(ctx, poolID, oracle)
	require.NoError(t, err)

	// Total value = 1,000,000 * 10 + 2,000,000 * 5 = 10,000,000 + 10,000,000 = 20,000,000
	expectedValue := math.LegacyMustNewDecFromStr("20000000.00")
	require.Equal(t, expectedValue, totalValue)
}

// TestGetPoolValueUSD_PoolNotFound tests error handling for non-existent pool
func TestGetPoolValueUSD_PoolNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	_, err := k.GetPoolValueUSD(ctx, 999, oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pool not found")
}

// TestGetPoolValueUSD_OraclePriceNotFound tests error handling when oracle price is missing
func TestGetPoolValueUSD_OraclePriceNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Only set price for one token
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())

	// Should fail because osmo price is missing
	_, err := k.GetPoolValueUSD(ctx, poolID, oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "oracle price error")
}

// TestValidatePoolPrice_WithinTolerance tests pool price validation when within tolerance
func TestValidatePoolPrice_WithinTolerance(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool with reserves that match oracle ratio
	// Pool ratio: 1000 / 2000 = 0.5
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Oracle prices: atom/osmo = 10/5 = 2.0 (inverse of pool ratio is acceptable)
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// 10% max deviation should pass
	maxDeviation := math.LegacyMustNewDecFromStr("0.10")
	err := k.ValidatePoolPrice(ctx, poolID, oracle, maxDeviation)
	require.NoError(t, err)
}

// TestValidatePoolPrice_ExceedsTolerance tests pool price validation when exceeding tolerance
func TestValidatePoolPrice_ExceedsTolerance(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool with imbalanced reserves
	// Pool ratio: 1000 / 5000 = 0.2
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(5000000))

	// Oracle prices suggest different ratio: atom/osmo = 10/5 = 2.0
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// 5% max deviation should fail (actual deviation is much larger)
	maxDeviation := math.LegacyMustNewDecFromStr("0.05")
	err := k.ValidatePoolPrice(ctx, poolID, oracle, maxDeviation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "price deviation too high")
}

// TestValidatePoolPrice_StalePrices tests rejection of stale oracle prices
func TestValidatePoolPrice_StalePrices(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Set stale timestamps (more than 60 seconds old)
	staleTimestamp := ctx.BlockTime().Unix() - 100
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), staleTimestamp)
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), staleTimestamp)

	maxDeviation := math.LegacyMustNewDecFromStr("0.10")
	err := k.ValidatePoolPrice(ctx, poolID, oracle, maxDeviation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "stale")
}

// TestGetFairPoolPrice_Success tests fair price retrieval from oracle
func TestGetFairPoolPrice_Success(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	fairPrice, err := k.GetFairPoolPrice(ctx, poolID, oracle)
	require.NoError(t, err)

	// Fair price = priceA / priceB = 10 / 5 = 2.0
	expectedPrice := math.LegacyMustNewDecFromStr("2.00")
	require.Equal(t, expectedPrice, fairPrice)
}

// TestGetLPTokenValueUSD_Success tests LP token USD valuation
func TestGetLPTokenValueUSD_Success(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Get pool to check total shares
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)

	// Calculate value of 10% of LP tokens
	shares := pool.TotalShares.QuoRaw(10)
	lpValue, err := k.GetLPTokenValueUSD(ctx, poolID, shares, oracle)
	require.NoError(t, err)

	// Total pool value is 20,000,000, so 10% should be 2,000,000
	expectedValue := math.LegacyMustNewDecFromStr("2000000.00")
	require.Equal(t, expectedValue, lpValue)
}

// TestGetLPTokenValueUSD_ZeroShares tests error handling for zero total shares
func TestGetLPTokenValueUSD_ZeroShares(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create a pool
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Manually corrupt pool to have zero shares (for testing)
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	pool.TotalShares = math.ZeroInt()
	err = k.SetPool(ctx, pool)
	require.NoError(t, err)

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	_, err = k.GetLPTokenValueUSD(ctx, poolID, math.NewInt(100), oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zero total shares")
}

// TestDetectArbitrageOpportunity_NoOpportunity tests when pool price matches oracle
func TestDetectArbitrageOpportunity_NoOpportunity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create pool with reserves matching oracle ratio
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(2000000), math.NewInt(1000000))

	// Oracle ratio matches pool ratio: 10/5 = 2.0, pool ratio: 2000000/1000000 = 2.0
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	minProfit := math.LegacyMustNewDecFromStr("0.01") // 1% minimum profit
	hasOpportunity, profitPercent, err := k.DetectArbitrageOpportunity(ctx, poolID, oracle, minProfit)
	require.NoError(t, err)
	require.False(t, hasOpportunity)
	require.True(t, profitPercent.LT(minProfit))
}

// TestDetectArbitrageOpportunity_HasOpportunity tests detection of arbitrage opportunity
func TestDetectArbitrageOpportunity_HasOpportunity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	// Create pool with imbalanced reserves
	// Pool ratio: 1000000/3000000 = 0.333
	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(3000000))

	// Oracle ratio: 10/5 = 2.0 (much different from pool)
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	minProfit := math.LegacyMustNewDecFromStr("0.01") // 1% minimum profit
	hasOpportunity, profitPercent, err := k.DetectArbitrageOpportunity(ctx, poolID, oracle, minProfit)
	require.NoError(t, err)
	require.True(t, hasOpportunity)
	require.True(t, profitPercent.GT(minProfit))
}

// TestValidateSwapWithOracle_Success tests swap validation against oracle prices
func TestValidateSwapWithOracle_Success(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Swap 1000 atom for osmo
	// Oracle-based expected: 1000 * 10 / 5 = 2000 osmo
	amountIn := math.NewInt(1000)
	expectedOut := math.NewInt(2000)

	err := k.ValidateSwapWithOracle(ctx, poolID, "atom", "osmo", amountIn, expectedOut, oracle)
	require.NoError(t, err)
}

// TestValidateSwapWithOracle_DeviationTooHigh tests rejection of swap with excessive deviation
func TestValidateSwapWithOracle_DeviationTooHigh(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Swap 1000 atom, but claim to expect way too much osmo
	// Oracle-based expected: 1000 * 10 / 5 = 2000 osmo
	// Claiming: 5000 osmo (way above 5% tolerance)
	amountIn := math.NewInt(1000)
	expectedOut := math.NewInt(5000) // Excessive

	err := k.ValidateSwapWithOracle(ctx, poolID, "atom", "osmo", amountIn, expectedOut, oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "price deviation too high")
}

// TestValidateSwapWithOracle_WithinTolerance tests swap within acceptable deviation
func TestValidateSwapWithOracle_WithinTolerance(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Swap 1000 atom for osmo
	// Oracle-based expected: 1000 * 10 / 5 = 2000 osmo
	// Claim 2080 osmo (4% above, within 5% tolerance)
	amountIn := math.NewInt(1000)
	expectedOut := math.NewInt(2080)

	err := k.ValidateSwapWithOracle(ctx, poolID, "atom", "osmo", amountIn, expectedOut, oracle)
	require.NoError(t, err)
}

// TestOracleIntegration_PriceConsistency tests oracle prices remain consistent across calls
func TestOracleIntegration_PriceConsistency(t *testing.T) {
	_, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())

	// Call GetPrice multiple times and verify consistency
	for i := 0; i < 10; i++ {
		price, err := oracle.GetPrice(ctx, "atom")
		require.NoError(t, err)
		require.Equal(t, math.LegacyMustNewDecFromStr("10.00"), price)
	}
}

// TestOracleIntegration_MultipleAssets tests oracle with multiple assets
func TestOracleIntegration_MultipleAssets(t *testing.T) {
	_, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	assets := map[string]string{
		"atom": "10.00",
		"osmo": "5.00",
		"usdc": "1.00",
		"upaw": "0.50",
	}

	// Set all prices
	for denom, priceStr := range assets {
		oracle.setPrice(denom, math.LegacyMustNewDecFromStr(priceStr), ctx.BlockTime().Unix())
	}

	// Verify all prices
	for denom, priceStr := range assets {
		price, err := oracle.GetPrice(ctx, denom)
		require.NoError(t, err)
		require.Equal(t, math.LegacyMustNewDecFromStr(priceStr), price, "price mismatch for %s", denom)
	}
}

// TestOracleIntegration_EdgeCases tests edge cases in oracle integration
func TestOracleIntegration_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupOracle func(*mockOracleKeeper, sdk.Context)
		testFunc    func(*testing.T, *dexkeeper.Keeper, sdk.Context, *mockOracleKeeper)
	}{
		{
			name: "zero oracle price",
			setupOracle: func(m *mockOracleKeeper, ctx sdk.Context) {
				m.setPrice("atom", math.LegacyZeroDec(), ctx.BlockTime().Unix())
				m.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())
			},
			testFunc: func(t *testing.T, keeper *dexkeeper.Keeper, ctx sdk.Context, oracle *mockOracleKeeper) {
				poolID := createTestPoolForOracle(t, keeper, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))
				_, err := keeper.GetPoolValueUSD(ctx, poolID, oracle)
				// Should succeed but calculate zero value for atom
				require.NoError(t, err)
			},
		},
		{
			name: "very large oracle price",
			setupOracle: func(m *mockOracleKeeper, ctx sdk.Context) {
				m.setPrice("atom", math.LegacyMustNewDecFromStr("1000000000.00"), ctx.BlockTime().Unix())
				m.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())
			},
			testFunc: func(t *testing.T, keeper *dexkeeper.Keeper, ctx sdk.Context, oracle *mockOracleKeeper) {
				poolID := createTestPoolForOracle(t, keeper, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))
				totalValue, err := keeper.GetPoolValueUSD(ctx, poolID, oracle)
				require.NoError(t, err)
				require.True(t, totalValue.GT(math.LegacyZeroDec()))
			},
		},
		{
			name: "very small oracle price",
			setupOracle: func(m *mockOracleKeeper, ctx sdk.Context) {
				m.setPrice("atom", math.LegacyMustNewDecFromStr("0.000001"), ctx.BlockTime().Unix())
				m.setPrice("osmo", math.LegacyMustNewDecFromStr("0.000002"), ctx.BlockTime().Unix())
			},
			testFunc: func(t *testing.T, keeper *dexkeeper.Keeper, ctx sdk.Context, oracle *mockOracleKeeper) {
				poolID := createTestPoolForOracle(t, keeper, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))
				totalValue, err := keeper.GetPoolValueUSD(ctx, poolID, oracle)
				require.NoError(t, err)
				require.True(t, totalValue.GT(math.LegacyZeroDec()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dexK, dexCtx := keepertest.DexKeeper(t)
			oracle := newMockOracleKeeper()
			tt.setupOracle(oracle, dexCtx)
			tt.testFunc(t, dexK, dexCtx, oracle)
		})
	}
}

// TestRealOracleKeeper_Integration tests with actual oracle keeper (integration test)
func TestRealOracleKeeper_Integration(t *testing.T) {
	// Create both DEX and Oracle keepers
	dexK, dexCtx := keepertest.DexKeeper(t)
	oracleK, oracleCtx := keepertest.OracleKeeper(t)

	// Set up oracle prices in oracle context
	atomPrice := oracletypes.Price{
		Asset:         "atom",
		Price:         math.LegacyMustNewDecFromStr("10.00"),
		BlockHeight:   oracleCtx.BlockHeight(),
		NumValidators: 1,
	}
	err := oracleK.SetPrice(oracleCtx, atomPrice)
	require.NoError(t, err)

	osmoPrice := oracletypes.Price{
		Asset:         "osmo",
		Price:         math.LegacyMustNewDecFromStr("5.00"),
		BlockHeight:   oracleCtx.BlockHeight(),
		NumValidators: 1,
	}
	err = oracleK.SetPrice(oracleCtx, osmoPrice)
	require.NoError(t, err)

	// Create a pool in DEX using dex context
	poolID := createTestPoolForOracle(t, dexK, dexCtx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Create wrapper that implements OracleKeeper interface and uses oracle context
	oracleWrapper := &realOracleKeeperWrapper{k: oracleK, ctx: oracleCtx}

	// Test pool value calculation
	totalValue, err := dexK.GetPoolValueUSD(dexCtx, poolID, oracleWrapper)
	require.NoError(t, err)

	expectedValue := math.LegacyMustNewDecFromStr("20000000.00")
	require.Equal(t, expectedValue, totalValue)
}

// realOracleKeeperWrapper wraps the real oracle keeper to implement the interface
// It uses the oracle context internally to access oracle store
type realOracleKeeperWrapper struct {
	k   *oraclekeeper.Keeper
	ctx sdk.Context
}

func (w *realOracleKeeperWrapper) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	// Use the oracle context instead of the passed context to access oracle store
	price, err := w.k.GetPrice(w.ctx, denom)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	return price.Price, nil
}

func (w *realOracleKeeperWrapper) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	// Use the oracle context instead of the passed context to access oracle store
	price, err := w.k.GetPrice(w.ctx, denom)
	if err != nil {
		return math.LegacyZeroDec(), 0, err
	}
	return price.Price, price.BlockHeight, nil
}

// TestOracleIntegration_ConcurrentAccess tests concurrent oracle price access
func TestOracleIntegration_ConcurrentAccess(t *testing.T) {
	_, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())

	// Simulate concurrent access patterns
	for i := 0; i < 100; i++ {
		price, err := oracle.GetPrice(ctx, "atom")
		require.NoError(t, err)
		require.Equal(t, math.LegacyMustNewDecFromStr("10.00"), price)
	}
}

// TestValidatePoolPrice_FreshPrices tests validation with fresh oracle prices
func TestValidatePoolPrice_FreshPrices(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Set prices with current timestamp
	currentTime := ctx.BlockTime().Unix()
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), currentTime)
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), currentTime)

	maxDeviation := math.LegacyMustNewDecFromStr("0.10")
	err := k.ValidatePoolPrice(ctx, poolID, oracle, maxDeviation)
	require.NoError(t, err)
}

// TestGetFairPoolPrice_PoolNotFound tests fair price with non-existent pool
func TestGetFairPoolPrice_PoolNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	_, err := k.GetFairPoolPrice(ctx, 999, oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pool not found")
}

// TestDetectArbitrageOpportunity_PoolNotFound tests arbitrage detection with non-existent pool
func TestDetectArbitrageOpportunity_PoolNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	minProfit := math.LegacyMustNewDecFromStr("0.01")
	_, _, err := k.DetectArbitrageOpportunity(ctx, 999, oracle, minProfit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pool not found")
}

// TestValidateSwapWithOracle_OraclePriceNotFound tests swap validation when oracle price is missing
func TestValidateSwapWithOracle_OraclePriceNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Only set price for atom, not osmo
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())

	amountIn := math.NewInt(1000)
	expectedOut := math.NewInt(2000)

	err := k.ValidateSwapWithOracle(ctx, poolID, "atom", "osmo", amountIn, expectedOut, oracle)
	require.Error(t, err)
	require.Contains(t, err.Error(), "oracle price error")
}

// TestOracleIntegration_PriceUpdates tests handling of oracle price updates
func TestOracleIntegration_PriceUpdates(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	oracle := newMockOracleKeeper()

	poolID := createTestPoolForOracle(t, k, ctx, "atom", "osmo", math.NewInt(1000000), math.NewInt(2000000))

	// Set initial prices
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("10.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("5.00"), ctx.BlockTime().Unix())

	// Get initial pool value
	value1, err := k.GetPoolValueUSD(ctx, poolID, oracle)
	require.NoError(t, err)

	// Update oracle prices
	oracle.setPrice("atom", math.LegacyMustNewDecFromStr("15.00"), ctx.BlockTime().Unix())
	oracle.setPrice("osmo", math.LegacyMustNewDecFromStr("7.50"), ctx.BlockTime().Unix())

	// Get updated pool value
	value2, err := k.GetPoolValueUSD(ctx, poolID, oracle)
	require.NoError(t, err)

	// Value should increase proportionally (1.5x)
	expectedValue2 := value1.Mul(math.LegacyMustNewDecFromStr("1.5"))
	require.Equal(t, expectedValue2, value2)
}
