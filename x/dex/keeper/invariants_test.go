package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestConstantProductInvariant tests the constant product invariant k = x * y
func TestConstantProductInvariant(t *testing.T) {
	tests := []struct {
		name           string
		setupPool      func(k *keeper.Keeper, ctx sdk.Context) uint64
		expectedBroken bool
		description    string
	}{
		{
			name: "valid pool with 1:1 ratio",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				// Create pool with equal reserves
				reserveA := math.NewInt(1_000_000)
				reserveB := math.NewInt(1_000_000)
				shares := math.NewInt(1_000_000)

				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    reserveA,
					ReserveB:    reserveB,
					TotalShares: shares,
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: false,
			description:    "Pool with proper k value should pass",
		},
		{
			name: "pool with slight fee accumulation (within 10%)",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				// Create pool where k has increased slightly from fees
				// For ratio = k / shares^2 = 1.05, we need k = 1.05 * shares^2
				// With shares = 1M, we need reserveA * reserveB = 1.05 * 1M^2 = 1.05T
				// sqrt(1.05T) ~= 1.0247M for each reserve
				shares := math.NewInt(1_000_000)
				reserveA := math.NewInt(1_024_700)
				reserveB := math.NewInt(1_024_700)

				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    reserveA,
					ReserveB:    reserveB,
					TotalShares: shares,
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: false,
			description:    "Pool with fee accumulation within 10% should pass",
		},
		{
			name: "pool with excessive fee accumulation (above 10%)",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				// Create pool where k has increased more than 10%
				// This should trigger invariant violation
				shares := math.NewInt(1_000_000)
				reserveA := math.NewInt(1_200_000)
				reserveB := math.NewInt(1_200_000)

				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    reserveA,
					ReserveB:    reserveB,
					TotalShares: shares,
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: true,
			description:    "Pool with excessive k increase should fail",
		},
		{
			name: "pool with decreased k (fund extraction attempt)",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				// Create pool where k has decreased - this should ALWAYS fail
				// This simulates precision manipulation or fund extraction
				shares := math.NewInt(1_000_000)
				reserveA := math.NewInt(990_000) // 1% decrease
				reserveB := math.NewInt(990_000)

				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    reserveA,
					ReserveB:    reserveB,
					TotalShares: shares,
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: true,
			description:    "Pool with decreased k should fail (prevents fund extraction)",
		},
		{
			name: "pool with significant k decrease (old 50% tolerance would allow)",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				// This test verifies the fix - old tolerance was 0.5 (50%)
				// New tolerance is 0.999 (99.9%), so this should fail
				shares := math.NewInt(1_000_000)
				reserveA := math.NewInt(750_000) // 25% decrease
				reserveB := math.NewInt(750_000)

				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    reserveA,
					ReserveB:    reserveB,
					TotalShares: shares,
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: true,
			description:    "Pool with 25% k decrease should fail (old bug allowed 50%)",
		},
		{
			name: "empty pool",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) uint64 {
				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    math.ZeroInt(),
					ReserveB:    math.ZeroInt(),
					TotalShares: math.ZeroInt(),
				}

				require.NoError(t, k.SetPool(ctx, &pool))
				return pool.Id
			},
			expectedBroken: false,
			description:    "Empty pool should be skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.DexKeeper(t)

			// Setup pool
			_ = tt.setupPool(k, ctx)

			// Run invariant - dereference k since invariant takes value receiver
			invariant := keeper.ConstantProductInvariant(*k)
			msg, broken := invariant(ctx)

			// Check result
			require.Equal(t, tt.expectedBroken, broken,
				"Test: %s\nDescription: %s\nInvariant message: %s",
				tt.name, tt.description, msg)

			if broken {
				require.NotEmpty(t, msg, "Broken invariant should have a message")
			}
		})
	}
}

// TestAllInvariants tests that all invariants run together
func TestAllInvariants(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool using the normal flow which handles all state correctly
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "tokenA", "tokenB", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Run all invariants - dereference k since invariant takes value receiver
	invariant := keeper.AllInvariants(*k)
	msg, broken := invariant(ctx)

	require.False(t, broken, "All invariants should pass for valid pool: %s", msg)
}

// TestPoolReservesInvariant tests the pool reserves invariant
func TestPoolReservesInvariant(t *testing.T) {
	tests := []struct {
		name           string
		setupPool      func(k *keeper.Keeper, ctx sdk.Context)
		expectedBroken bool
	}{
		{
			name: "valid pool",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) {
				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    math.NewInt(1_000_000),
					ReserveB:    math.NewInt(1_000_000),
					TotalShares: math.NewInt(1_000_000),
				}
				require.NoError(t, k.SetPool(ctx, &pool))
			},
			expectedBroken: false,
		},
		{
			name: "zero reserve A",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) {
				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenA",
					TokenB:      "tokenB",
					ReserveA:    math.ZeroInt(),
					ReserveB:    math.NewInt(1_000_000),
					TotalShares: math.NewInt(1_000_000),
				}
				require.NoError(t, k.SetPool(ctx, &pool))
			},
			expectedBroken: true,
		},
		{
			name: "incorrect token ordering",
			setupPool: func(k *keeper.Keeper, ctx sdk.Context) {
				pool := types.Pool{
					Id:          1,
					TokenA:      "tokenB", // Wrong order
					TokenB:      "tokenA",
					ReserveA:    math.NewInt(1_000_000),
					ReserveB:    math.NewInt(1_000_000),
					TotalShares: math.NewInt(1_000_000),
				}
				require.NoError(t, k.SetPool(ctx, &pool))
			},
			expectedBroken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.DexKeeper(t)
			tt.setupPool(k, ctx)

			invariant := keeper.PoolReservesInvariant(*k)
			_, broken := invariant(ctx)

			require.Equal(t, tt.expectedBroken, broken)
		})
	}
}

// TestModuleBalanceInvariant tests the module balance invariant
func TestModuleBalanceInvariant(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool using normal flow - this properly transfers tokens to module
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "tokenA", "tokenB", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Test that invariant passes with properly created pool
	invariant := keeper.ModuleBalanceInvariant(*k)
	_, broken := invariant(ctx)
	require.False(t, broken, "Should pass when pool is created correctly")

	// Create an inconsistent pool state by directly setting pool with higher reserves
	// than what the module account actually has
	pool.ReserveA = math.NewInt(10_000_000) // Much higher than actual balance
	pool.ReserveB = math.NewInt(10_000_000)
	require.NoError(t, k.SetPool(ctx, pool))

	// Now invariant should fail
	_, broken = invariant(ctx)
	require.True(t, broken, "Should fail when reserves exceed actual module balance")
}
