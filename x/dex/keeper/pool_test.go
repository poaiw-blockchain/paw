package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Helper functions for pool tests

func createTestTrader(t *testing.T) sdk.AccAddress {
	return sdk.AccAddress([]byte("test_trader_address"))
}

func createTestTraderWithIndex(t *testing.T, index int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("test_trader_"))
	addr[19] = byte(index)
	return sdk.AccAddress(addr)
}

// TestCreatePool_Valid tests successful pool creation
func TestCreatePool_Valid(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	require.NotNil(t, pool)
	require.Greater(t, pool.Id, uint64(0))
	require.Equal(t, tokenA, pool.TokenA)
	require.Equal(t, tokenB, pool.TokenB)
	require.Equal(t, amountA, pool.ReserveA)
	require.Equal(t, amountB, pool.ReserveB)
	require.True(t, pool.TotalShares.IsPositive())
	require.Equal(t, creator.String(), pool.Creator)
}

// TestCreatePool_DuplicateTokenPair tests rejection of duplicate pools
func TestCreatePool_DuplicateTokenPair(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	// Create pool first time
	_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Try to create again with same token pair
	_, err = k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

// TestCreatePool_SameToken tests rejection of pools with same token
func TestCreatePool_SameToken(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "upaw" // Same as tokenA
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "identical tokens")
}

// TestCreatePool_ZeroAmountA tests rejection of zero initial liquidity for token A
func TestCreatePool_ZeroAmountA(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(0)
	amountB := math.NewInt(2000000)

	_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be positive")
}

// TestCreatePool_ZeroAmountB tests rejection of zero initial liquidity for token B
func TestCreatePool_ZeroAmountB(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(0)

	_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be positive")
}

// TestCreatePool_BelowMinimumLiquidity tests rejection when initial liquidity too low
func TestCreatePool_BelowMinimumLiquidity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	// Very small amounts that won't meet minimum liquidity
	amountA := math.NewInt(1)
	amountB := math.NewInt(1)

	_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too low")
}

// TestCreatePool_TokenOrdering tests that tokens are ordered consistently
func TestCreatePool_TokenOrdering(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create pool with tokens in reverse order
	tokenA := "uusdt"
	tokenB := "upaw"
	amountA := math.NewInt(2000000)
	amountB := math.NewInt(1000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Verify tokens are sorted (upaw < uusdt alphabetically)
	require.Equal(t, "upaw", pool.TokenA)
	require.Equal(t, "uusdt", pool.TokenB)
	// Amounts should be swapped to match
	require.Equal(t, amountB, pool.ReserveA)
	require.Equal(t, amountA, pool.ReserveB)
}

// TestGetPool tests pool retrieval
func TestGetPool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	// Pool doesn't exist initially
	_, err := k.GetPool(ctx, 1)
	require.Error(t, err)

	// Create pool
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Retrieve pool
	retrieved, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, pool.Id, retrieved.Id)
	require.Equal(t, pool.TokenA, retrieved.TokenA)
	require.Equal(t, pool.TokenB, retrieved.TokenB)
	require.Equal(t, pool.ReserveA, retrieved.ReserveA)
	require.Equal(t, pool.ReserveB, retrieved.ReserveB)
}

// TestGetPoolByTokens tests pool retrieval by token pair
func TestGetPoolByTokens(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	// Create pool
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Retrieve by token pair (in same order)
	retrieved, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
	require.NoError(t, err)
	require.Equal(t, pool.Id, retrieved.Id)

	// Retrieve by token pair (in reverse order - should still work)
	retrieved, err = k.GetPoolByTokens(ctx, tokenB, tokenA)
	require.NoError(t, err)
	require.Equal(t, pool.Id, retrieved.Id)
}

// TestIteratePools tests pool iteration
func TestIteratePools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create multiple pools
	numPools := 5
	for i := 0; i < numPools; i++ {
		tokenA := "upaw"
		tokenB := "token" + string(rune('A'+i))
		amountA := math.NewInt(1000000)
		amountB := math.NewInt(2000000)

		_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err := k.IteratePools(ctx, func(pool types.Pool) bool {
		count++
		require.Greater(t, pool.Id, uint64(0))
		require.NotEmpty(t, pool.TokenA)
		require.NotEmpty(t, pool.TokenB)
		require.True(t, pool.ReserveA.IsPositive())
		require.True(t, pool.ReserveB.IsPositive())
		return false // continue iteration
	})
	require.NoError(t, err)
	require.Equal(t, numPools, count)
}

// TestGetAllPools tests getting all pools
func TestGetAllPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create multiple pools
	numPools := 3
	for i := 0; i < numPools; i++ {
		tokenA := "upaw"
		tokenB := "token" + string(rune('A'+i))
		amountA := math.NewInt(1000000)
		amountB := math.NewInt(2000000)

		_, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
		require.NoError(t, err)
	}

	// Get all pools
	pools, err := k.GetAllPools(ctx)
	require.NoError(t, err)
	require.Equal(t, numPools, len(pools))

	// Verify all pools are valid
	for _, pool := range pools {
		require.Greater(t, pool.Id, uint64(0))
		require.True(t, pool.ReserveA.IsPositive())
		require.True(t, pool.ReserveB.IsPositive())
	}
}

// TestPoolID_Increment tests that pool IDs increment correctly
func TestPoolID_Increment(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create first pool
	pool1, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(1000000), math.NewInt(2000000))
	require.NoError(t, err)

	// Create second pool
	pool2, err := k.CreatePool(ctx, creator, "upaw", "uatom", math.NewInt(1000000), math.NewInt(2000000))
	require.NoError(t, err)

	// Verify IDs increment
	require.Equal(t, pool1.Id+1, pool2.Id)
}

// TestPoolReserveOrdering tests that reserves match token ordering
func TestPoolReserveOrdering(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tests := []struct {
		name           string
		tokenA         string
		tokenB         string
		amountA        math.Int
		amountB        math.Int
		expectedTokenA string
		expectedTokenB string
	}{
		{
			name:           "already ordered",
			tokenA:         "uatom",
			tokenB:         "uusdt",
			amountA:        math.NewInt(1000000),
			amountB:        math.NewInt(2000000),
			expectedTokenA: "uatom",
			expectedTokenB: "uusdt",
		},
		{
			name:           "needs reordering",
			tokenA:         "uusdt",
			tokenB:         "uatom",
			amountA:        math.NewInt(2000000),
			amountB:        math.NewInt(1000000),
			expectedTokenA: "uatom",
			expectedTokenB: "uusdt",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := k.CreatePool(ctx, creator, tt.tokenA, tt.tokenB, tt.amountA, tt.amountB)
			require.NoError(t, err)
			require.Equal(t, tt.expectedTokenA, pool.TokenA)
			require.Equal(t, tt.expectedTokenB, pool.TokenB)

			// Verify reserves match the ordered tokens
			if tt.tokenA == tt.expectedTokenA {
				require.Equal(t, tt.amountA, pool.ReserveA)
				require.Equal(t, tt.amountB, pool.ReserveB)
			} else {
				require.Equal(t, tt.amountB, pool.ReserveA)
				require.Equal(t, tt.amountA, pool.ReserveB)
			}

			// Use different token pair for next test to avoid duplicate error
			_ = i
		})
	}
}

// TestPoolShares_Calculation tests initial shares calculation
func TestPoolShares_Calculation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000) // 10M
	amountB := math.NewInt(20000000) // 20M

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Initial shares should be sqrt(amountA * amountB)
	product := amountA.Mul(amountB)
	expectedShares, _ := math.LegacyNewDecFromInt(product).ApproxSqrt()
	expectedSharesInt := expectedShares.TruncateInt()

	require.Equal(t, expectedSharesInt, pool.TotalShares)
}

// TestSetPool tests pool update
func TestSetPool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	// Create pool
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Modify pool
	pool.ReserveA = math.NewInt(1500000)
	pool.ReserveB = math.NewInt(2500000)

	// Save changes
	err = k.SetPool(ctx, pool)
	require.NoError(t, err)

	// Verify changes persisted
	retrieved, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, pool.ReserveA, retrieved.ReserveA)
	require.Equal(t, pool.ReserveB, retrieved.ReserveB)
}

// TestPoolByTokensIndex tests token pair indexing
func TestPoolByTokensIndex(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	// Create pool
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Manually set index
	err = k.SetPoolByTokens(ctx, tokenA, tokenB, pool.Id)
	require.NoError(t, err)

	// Retrieve using index
	retrieved, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
	require.NoError(t, err)
	require.Equal(t, pool.Id, retrieved.Id)
}

// TestGetNextPoolID tests pool ID counter
func TestGetNextPoolID(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Get first ID
	id1, err := k.GetNextPoolID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	// Get second ID (should increment)
	id2, err := k.GetNextPoolID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	// Get third ID
	id3, err := k.GetNextPoolID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), id3)
}

// TestSetNextPoolId tests setting pool ID counter
func TestSetNextPoolId(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set counter to specific value
	k.SetNextPoolId(ctx, 100)

	// Get next ID
	id, err := k.GetNextPoolID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), id)

	// Next call should return 101
	id, err = k.GetNextPoolID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(101), id)
}
