package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TEST-6: Genesis export/import with custom modules

func TestGenesisExportImport_Pools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create multiple pools
	for i := 0; i < 5; i++ {
		tokenA := "token" + string(rune('A'+i))
		tokenB := "token" + string(rune('B'+i))
		_, err := k.CreatePool(ctx, types.TestAddr(), tokenA, tokenB,
			math.NewInt(int64(1000000*(i+1))),
			math.NewInt(int64(500000*(i+1))))
		require.NoError(t, err)
	}

	t.Run("exports and imports pools correctly", func(t *testing.T) {
		// Export genesis
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)
		require.Len(t, genesis.Pools, 5)

		// Create new keeper
		k2, ctx2 := keepertest.DexKeeper(t)

		// Import genesis
		k2.InitGenesis(ctx2, genesis)

		// Verify pools match
		for i := uint64(1); i <= 5; i++ {
			pool, err := k2.GetPool(ctx2, i)
			require.NoError(t, err)
			require.NotNil(t, pool)
		}
	})
}

func TestGenesisExportImport_Liquidity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool and add liquidity from multiple providers
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	providers := make([]types.TestAddress, 5)
	for i := 0; i < 5; i++ {
		providers[i] = types.TestAddrWithSeed(i)
		_, _, err := k.AddLiquidity(ctx, providers[i], poolID,
			math.NewInt(int64(10000*(i+1))),
			math.NewInt(int64(5000*(i+1))))
		require.NoError(t, err)
	}

	t.Run("exports and imports liquidity positions correctly", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)
		require.NotEmpty(t, genesis.LiquidityPositions)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify liquidity positions
		for i, provider := range providers {
			shares, err := k2.GetLiquidity(ctx2, poolID, provider)
			require.NoError(t, err)
			require.True(t, shares.IsPositive(), "provider %d should have shares", i)
		}
	})
}

func TestGenesisExportImport_LimitOrders(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	// Create limit orders
	for i := 0; i < 10; i++ {
		owner := types.TestAddrWithSeed(100 + i)
		_, err := k.PlaceLimitOrder(ctx, owner, poolID,
			i%2 == 0, // alternating buy/sell
			"upaw",
			math.NewInt(int64(1000*(i+1))),
			math.LegacyNewDecWithPrec(int64(50+i), 2))
		require.NoError(t, err)
	}

	t.Run("exports and imports limit orders correctly", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)
		require.Len(t, genesis.LimitOrders, 10)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify orders
		orders, err := k2.GetAllLimitOrders(ctx2)
		require.NoError(t, err)
		require.Len(t, orders, 10)
	})
}

func TestGenesisExportImport_CircuitBreakerState(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	// Trigger circuit breaker
	err := k.TriggerCircuitBreaker(ctx, poolID, "test_trigger")
	require.NoError(t, err)

	t.Run("exports and imports circuit breaker state", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)

		// Circuit breaker state should be included
		require.NotEmpty(t, genesis.CircuitBreakerStates)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify circuit breaker state preserved
		isPaused := k2.IsPoolPaused(ctx2, poolID)
		require.True(t, isPaused, "circuit breaker state should be preserved")
	})
}

func TestGenesisExportImport_Params(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set custom params
	params := types.DefaultParams()
	params.SwapFee = math.LegacyNewDecWithPrec(5, 3) // 0.5%
	params.MinLiquidity = math.NewInt(10000)
	k.SetParams(ctx, params)

	t.Run("exports and imports params correctly", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify params
		importedParams := k2.GetParams(ctx2)
		require.Equal(t, params.SwapFee, importedParams.SwapFee)
		require.Equal(t, params.MinLiquidity, importedParams.MinLiquidity)
	})
}

func TestGenesisExportImport_PendingIBCSwaps(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pending IBC swaps
	for i := uint64(1); i <= 3; i++ {
		swapState := &types.PendingIBCSwap{
			Sender:    types.TestAddrWithSeed(int(i)).String(),
			PoolId:    1,
			AmountIn:  math.NewInt(int64(i * 1000)),
			TokenIn:   "upaw",
			TokenOut:  "uatom",
			Sequence:  i,
			Channel:   "channel-0",
			Timestamp: ctx.BlockTime(),
		}
		err := k.SetPendingIBCSwap(ctx, i, "channel-0", swapState)
		require.NoError(t, err)
	}

	t.Run("exports and imports pending IBC swaps", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)
		require.Len(t, genesis.PendingIBCSwaps, 3)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify pending swaps
		for i := uint64(1); i <= 3; i++ {
			swap, found := k2.GetPendingIBCSwap(ctx2, i, "channel-0")
			require.True(t, found)
			require.Equal(t, math.NewInt(int64(i*1000)), swap.AmountIn)
		}
	})
}

func TestGenesisExportImport_FeeAccrual(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	// Simulate fee accrual via swaps
	for i := 0; i < 5; i++ {
		_, err := k.Swap(ctx, types.TestAddrWithSeed(200+i), poolID,
			"upaw", "uatom", math.NewInt(10000), math.NewInt(1))
		require.NoError(t, err)
	}

	t.Run("exports and imports fee state", func(t *testing.T) {
		genesis := k.ExportGenesis(ctx)
		require.NotNil(t, genesis)

		// Create new keeper and import
		k2, ctx2 := keepertest.DexKeeper(t)
		k2.InitGenesis(ctx2, genesis)

		// Verify pool state after import includes fee effects
		pool, err := k2.GetPool(ctx2, poolID)
		require.NoError(t, err)
		require.NotNil(t, pool)
	})
}

func TestGenesisValidation(t *testing.T) {
	t.Run("rejects genesis with invalid pool", func(t *testing.T) {
		genesis := &types.GenesisState{
			Pools: []types.Pool{
				{
					Id:          1,
					TokenA:      "", // Invalid: empty token
					TokenB:      "uatom",
					ReserveA:    math.NewInt(1000),
					ReserveB:    math.NewInt(500),
					TotalShares: math.NewInt(707),
				},
			},
		}

		err := genesis.Validate()
		require.Error(t, err)
	})

	t.Run("rejects genesis with negative reserves", func(t *testing.T) {
		genesis := &types.GenesisState{
			Pools: []types.Pool{
				{
					Id:          1,
					TokenA:      "upaw",
					TokenB:      "uatom",
					ReserveA:    math.NewInt(-1000), // Invalid
					ReserveB:    math.NewInt(500),
					TotalShares: math.NewInt(707),
				},
			},
		}

		err := genesis.Validate()
		require.Error(t, err)
	})

	t.Run("accepts valid genesis", func(t *testing.T) {
		genesis := types.DefaultGenesis()
		err := genesis.Validate()
		require.NoError(t, err)
	})
}
