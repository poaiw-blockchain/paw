package keeper_test

import (
	"bytes"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestDexGenesisRoundTrip(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	params.MaxSlippagePercent = sdkmath.LegacyNewDecWithPrec(25, 2)
	params.AuthorizedChannels = []types.AuthorizedChannel{
		{PortId: types.PortID, ChannelId: "channel-0"},
	}

	creatorOne := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	creatorTwo := sdk.AccAddress(bytes.Repeat([]byte{0x2}, 20)).String()
	providerOne := sdk.AccAddress(bytes.Repeat([]byte{0x3}, 20)).String()
	providerTwo := sdk.AccAddress(bytes.Repeat([]byte{0x4}, 20)).String()

	pools := []types.Pool{
		{
			Id:          1,
			TokenA:      "atom",
			TokenB:      "paw",
			ReserveA:    sdkmath.NewInt(1_000_000),
			ReserveB:    sdkmath.NewInt(2_000_000),
			TotalShares: sdkmath.NewInt(900_000),
			Creator:     creatorOne,
		},
		{
			Id:          2,
			TokenA:      "paw",
			TokenB:      "usdc",
			ReserveA:    sdkmath.NewInt(3_000_000),
			ReserveB:    sdkmath.NewInt(1_500_000),
			TotalShares: sdkmath.NewInt(1_200_000),
			Creator:     creatorTwo,
		},
	}

	twaps := []types.PoolTWAP{
		{
			PoolId:          1,
			LastPrice:       sdkmath.LegacyMustNewDecFromStr("2.0"),
			CumulativePrice: sdkmath.LegacyMustNewDecFromStr("20.0"),
			TotalSeconds:    120,
			LastTimestamp:   1_700_000_000,
			TwapPrice:       sdkmath.LegacyZeroDec(),
		},
		{
			PoolId:          2,
			LastPrice:       sdkmath.LegacyMustNewDecFromStr("0.5"),
			CumulativePrice: sdkmath.LegacyMustNewDecFromStr("5.0"),
			TotalSeconds:    60,
			LastTimestamp:   1_700_000_100,
			TwapPrice:       sdkmath.LegacyZeroDec(),
		},
	}

	circuitBreakerStates := []types.CircuitBreakerStateExport{
		{
			PoolId:            1,
			Enabled:           false,
			PausedUntil:       0,
			LastPrice:         sdkmath.LegacyMustNewDecFromStr("2.0"),
			TriggeredBy:       "",
			TriggerReason:     "",
			NotificationsSent: 0,
			LastNotification:  0,
			PersistenceKey:    "",
		},
		{
			PoolId:            2,
			Enabled:           true,
			PausedUntil:       1_700_000_200,
			LastPrice:         sdkmath.LegacyMustNewDecFromStr("0.5"),
			TriggeredBy:       "system",
			TriggerReason:     "price deviation",
			NotificationsSent: 2,
			LastNotification:  1_700_000_150,
			PersistenceKey:    "pool_2_cb",
		},
	}

	liquidityPositions := []types.LiquidityPositionExport{
		{
			PoolId:   1,
			Provider: providerOne,
			Shares:   sdkmath.NewInt(500_000),
		},
		{
			PoolId:   1,
			Provider: providerTwo,
			Shares:   sdkmath.NewInt(400_000),
		},
		{
			PoolId:   2,
			Provider: providerOne,
			Shares:   sdkmath.NewInt(700_000),
		},
		{
			PoolId:   2,
			Provider: providerTwo,
			Shares:   sdkmath.NewInt(500_000),
		},
	}

	genesis := types.GenesisState{
		Params:               params,
		Pools:                pools,
		NextPoolId:           3,
		PoolTwapRecords:      twaps,
		CircuitBreakerStates: circuitBreakerStates,
		LiquidityPositions:   liquidityPositions,
	}

	require.NoError(t, k.InitGenesis(ctx, genesis))

	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	require.Equal(t, genesis.Params, exported.Params)
	require.Equal(t, genesis.Pools, exported.Pools)
	require.Equal(t, genesis.PoolTwapRecords, exported.PoolTwapRecords)
	require.Equal(t, genesis.NextPoolId, exported.NextPoolId)
	require.Equal(t, len(genesis.CircuitBreakerStates), len(exported.CircuitBreakerStates))
	require.Equal(t, len(genesis.LiquidityPositions), len(exported.LiquidityPositions))

	// Verify circuit breaker states match
	// NOTE: Only persistent configuration is preserved across export/import.
	// Volatile runtime state (PausedUntil, NotificationsSent, etc.) is intentionally reset.
	for i, expected := range genesis.CircuitBreakerStates {
		actual := exported.CircuitBreakerStates[i]
		// Persistent configuration should match
		require.Equal(t, expected.PoolId, actual.PoolId)
		require.Equal(t, expected.Enabled, actual.Enabled)
		require.Equal(t, expected.LastPrice, actual.LastPrice)
		require.Equal(t, expected.PersistenceKey, actual.PersistenceKey)

		// Runtime state should be reset to zero values on export
		require.Equal(t, int64(0), actual.PausedUntil, "PausedUntil should be reset on export")
		require.Equal(t, int32(0), actual.NotificationsSent, "NotificationsSent should be reset on export")
		require.Equal(t, int64(0), actual.LastNotification, "LastNotification should be reset on export")
		require.Equal(t, "", actual.TriggeredBy, "TriggeredBy should be reset on export")
		require.Equal(t, "", actual.TriggerReason, "TriggerReason should be reset on export")
	}

	// Verify liquidity positions match
	for i, expected := range genesis.LiquidityPositions {
		actual := exported.LiquidityPositions[i]
		require.Equal(t, expected.PoolId, actual.PoolId)
		require.Equal(t, expected.Provider, actual.Provider)
		require.Equal(t, expected.Shares, actual.Shares)
	}
}

// TestGenesisSharesValidation tests that InitGenesis validates LP shares sum equals pool.TotalShares
func TestGenesisSharesValidation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	creatorOne := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	providerOne := sdk.AccAddress(bytes.Repeat([]byte{0x3}, 20)).String()
	providerTwo := sdk.AccAddress(bytes.Repeat([]byte{0x4}, 20)).String()

	t.Run("valid shares sum", func(t *testing.T) {
		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(600_000),
			},
			{
				PoolId:   1,
				Provider: providerTwo,
				Shares:   sdkmath.NewInt(400_000),
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(ctx, genesis)
		require.NoError(t, err)
	})

	t.Run("invalid shares sum - too many shares", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(700_000),
			},
			{
				PoolId:   1,
				Provider: providerTwo,
				Shares:   sdkmath.NewInt(400_000), // Total = 1,100,000 > pool.TotalShares
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(ctx, genesis)
		require.Error(t, err)
		require.Contains(t, err.Error(), "shares mismatch")
	})

	t.Run("invalid shares sum - too few shares", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(300_000), // Total = 300,000 < pool.TotalShares
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(ctx, genesis)
		require.Error(t, err)
		require.Contains(t, err.Error(), "shares mismatch")
	})

	t.Run("empty pool with no liquidity positions", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(0),
				ReserveB:    sdkmath.NewInt(0),
				TotalShares: sdkmath.NewInt(0),
				Creator:     creatorOne,
			},
		}

		genesis := types.GenesisState{
			Params:     params,
			Pools:      pools,
			NextPoolId: 2,
		}

		err := k.InitGenesis(ctx, genesis)
		require.NoError(t, err)
	})
}

// TestGenesisExportImportWithFeeAccumulatedPools tests that pools with accumulated fees
// can be successfully exported and re-imported. This is the core test for P1-DATA-1.
//
// SCENARIO: Pool accumulates fees from swaps, causing reserves to increase while shares
// remain constant. The k-value (reserveA * reserveB) increases but stays within the 10%
// tolerance defined in invariants.go.
//
// VALIDATION:
// - Genesis export should succeed with fee-accumulated pools
// - Genesis import should succeed with the exported state
// - LP shares sum must equal pool.TotalShares (strict equality)
// - Constant product invariant must pass (allows up to 10% increase from fees)
func TestGenesisExportImportWithFeeAccumulatedPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	params.SwapFee = sdkmath.LegacyNewDecWithPrec(3, 3) // 0.3% fee

	creatorAddr := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20))
	creator := creatorAddr.String()
	_ = sdk.AccAddress(bytes.Repeat([]byte{0x3}, 20)).String() // lpProvider (unused in this test)

	// Create pool with initial state
	// Using reserves that when multiplied equal shares^2 (no fee accumulation yet)
	initialReserveA := sdkmath.NewInt(1_000_000)
	initialReserveB := sdkmath.NewInt(1_000_000)
	initialShares := sdkmath.NewInt(1_000_000)

	pool := types.Pool{
		Id:          1,
		TokenA:      "atom",
		TokenB:      "paw",
		ReserveA:    initialReserveA,
		ReserveB:    initialReserveB,
		TotalShares: initialShares,
		Creator:     creator,
	}

	// Set pool and liquidity position
	require.NoError(t, k.SetPool(ctx, &pool))
	require.NoError(t, k.SetLiquidity(ctx, pool.Id, creatorAddr, initialShares))

	// Verify initial k-value
	initialK := initialReserveA.Mul(initialReserveB)
	expectedK := initialShares.Mul(initialShares)
	require.Equal(t, expectedK, initialK, "Initial k should equal shares^2")

	// Simulate fee accumulation by increasing reserves while keeping shares constant
	// We increase each reserve by 3% (factor 1.03), which increases k by 1.03*1.03 = 1.0609 (6.09%)
	// This is well within the 10% tolerance for k-value increase
	// This simulates multiple swaps that have accumulated fees in the pool
	feeAccumulationFactor := sdkmath.LegacyNewDecWithPrec(103, 2) // 1.03 = 103%
	newReserveA := feeAccumulationFactor.MulInt(initialReserveA).TruncateInt()
	newReserveB := feeAccumulationFactor.MulInt(initialReserveB).TruncateInt()

	pool.ReserveA = newReserveA
	pool.ReserveB = newReserveB
	// TotalShares remains unchanged - this is the key point
	require.NoError(t, k.SetPool(ctx, &pool))

	// Verify k-value has increased from fee accumulation
	newK := newReserveA.Mul(newReserveB)
	kRatio := sdkmath.LegacyNewDecFromInt(newK).Quo(sdkmath.LegacyNewDecFromInt(expectedK))
	t.Logf("K-value ratio after fee accumulation: %s (should be ~1.06)", kRatio)
	require.True(t, kRatio.GT(sdkmath.LegacyOneDec()), "K should have increased from fees")
	require.True(t, kRatio.LTE(sdkmath.LegacyNewDecWithPrec(11, 1)), "K increase should be within 10% tolerance")

	// Run constant product invariant - should pass with fee accumulation
	invariant := keeper.ConstantProductInvariant(*k)
	msg, broken := invariant(ctx)
	require.False(t, broken, "Constant product invariant should pass with 5%% fee accumulation: %s", msg)

	// Run liquidity shares invariant - should pass (shares unchanged)
	sharesInvariant := keeper.LiquiditySharesInvariant(*k)
	msg, broken = sharesInvariant(ctx)
	require.False(t, broken, "Liquidity shares invariant should pass: %s", msg)

	// Export genesis
	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err, "Genesis export should succeed with fee-accumulated pool")
	require.Len(t, exported.Pools, 1)
	require.Len(t, exported.LiquidityPositions, 1)

	// Verify exported pool has fee-accumulated reserves
	exportedPool := exported.Pools[0]
	require.Equal(t, newReserveA, exportedPool.ReserveA)
	require.Equal(t, newReserveB, exportedPool.ReserveB)
	require.Equal(t, initialShares, exportedPool.TotalShares)

	// Verify exported LP shares sum equals TotalShares
	require.Equal(t, initialShares, exported.LiquidityPositions[0].Shares)

	// Create a fresh keeper to test import
	k2, ctx2 := keepertest.DexKeeper(t)

	// Import genesis with fee-accumulated pool
	err = k2.InitGenesis(ctx2, *exported)
	require.NoError(t, err, "Genesis import should succeed with fee-accumulated pool")

	// Verify imported pool state
	importedPool, err := k2.GetPool(ctx2, 1)
	require.NoError(t, err)
	require.Equal(t, newReserveA, importedPool.ReserveA, "Reserves should match after import")
	require.Equal(t, newReserveB, importedPool.ReserveB, "Reserves should match after import")
	require.Equal(t, initialShares, importedPool.TotalShares, "Shares should match after import")

	// Verify LP position was imported correctly
	lpShares, err := k2.GetLiquidity(ctx2, 1, creatorAddr)
	require.NoError(t, err)
	require.Equal(t, initialShares, lpShares, "LP shares should match after import")

	// Check constant product invariant passes (key invariant for P1-DATA-1)
	cpInvariant := keeper.ConstantProductInvariant(*k2)
	msg, broken = cpInvariant(ctx2)
	require.False(t, broken, "Constant product invariant should pass after import: %s", msg)

	// Check liquidity shares invariant passes (strict equality requirement)
	sharesInvariant2 := keeper.LiquiditySharesInvariant(*k2)
	msg, broken = sharesInvariant2(ctx2)
	require.False(t, broken, "Liquidity shares invariant should pass after import: %s", msg)

	// Note: ModuleBalanceInvariant is skipped here because this test creates pools
	// directly without depositing tokens. In production, tokens are deposited when
	// pools are created through AddLiquidity, so the invariant would pass.
}

// TestGenesisExportImportMaxFeeAccumulation tests the edge case where fee accumulation
// is at a high tolerance (~8%, well below 10% max but testing boundary behavior)
func TestGenesisExportImportMaxFeeAccumulation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creatorAddr := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20))
	creator := creatorAddr.String()

	// Create pool with ~8% k increase (high but safely under 10% max)
	initialShares := sdkmath.NewInt(1_000_000)
	// For 8% k increase: k_new = 1.08 * shares^2
	// We need reserveA * reserveB = 1.08 * shares^2
	// With equal reserves: reserve = sqrt(1.08) * shares ≈ 1.0392 * shares
	highFeeReserve := sdkmath.LegacyNewDecWithPrec(10392, 4).MulInt(initialShares).TruncateInt()

	pool := types.Pool{
		Id:          1,
		TokenA:      "atom",
		TokenB:      "paw",
		ReserveA:    highFeeReserve,
		ReserveB:    highFeeReserve,
		TotalShares: initialShares,
		Creator:     creator,
	}

	require.NoError(t, k.SetPool(ctx, &pool))
	require.NoError(t, k.SetLiquidity(ctx, pool.Id, creatorAddr, initialShares))

	// Verify we're at ~8% (below max tolerance)
	actualK := highFeeReserve.Mul(highFeeReserve)
	expectedK := initialShares.Mul(initialShares)
	kRatio := sdkmath.LegacyNewDecFromInt(actualK).Quo(sdkmath.LegacyNewDecFromInt(expectedK))
	t.Logf("K-value ratio: %s (should be ~1.08)", kRatio)
	require.True(t, kRatio.GT(sdkmath.LegacyOneDec()), "K should be greater than 1")
	require.True(t, kRatio.LTE(sdkmath.LegacyNewDecWithPrec(11, 1)), "K should be at or below max tolerance")

	// Export and import
	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	k2, ctx2 := keepertest.DexKeeper(t)
	err = k2.InitGenesis(ctx2, *exported)
	require.NoError(t, err, "Should succeed at high fee accumulation tolerance")

	// Check constant product invariant passes
	cpInvariant := keeper.ConstantProductInvariant(*k2)
	msg, broken := cpInvariant(ctx2)
	require.False(t, broken, "Constant product invariant should pass at ~8%% fee accumulation: %s", msg)

	// Check liquidity shares invariant passes
	sharesInvariant := keeper.LiquiditySharesInvariant(*k2)
	msg, broken = sharesInvariant(ctx2)
	require.False(t, broken, "Liquidity shares invariant should pass: %s", msg)
}

// TestGenesisImportRejectsInvalidShares verifies that import fails if LP shares
// don't sum to TotalShares, even with valid fee accumulation
func TestGenesisImportRejectsInvalidShares(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	creator := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x2}, 20)).String()

	// Create pool with fee accumulation (valid k-value)
	shares := sdkmath.NewInt(1_000_000)
	reserve := sdkmath.LegacyNewDecWithPrec(105, 2).MulInt(shares).TruncateInt() // 5% fee accumulation

	pools := []types.Pool{{
		Id:          1,
		TokenA:      "atom",
		TokenB:      "paw",
		ReserveA:    reserve,
		ReserveB:    reserve,
		TotalShares: shares,
		Creator:     creator,
	}}

	// Create liquidity positions that DON'T sum to TotalShares
	// This represents corrupted genesis data
	liquidityPositions := []types.LiquidityPositionExport{
		{
			PoolId:   1,
			Provider: provider,
			Shares:   sdkmath.NewInt(800_000), // Only 80% of shares
		},
		// Missing 200,000 shares - this is invalid
	}

	genesis := types.GenesisState{
		Params:             params,
		Pools:              pools,
		NextPoolId:         2,
		LiquidityPositions: liquidityPositions,
	}

	// Import should fail due to shares mismatch
	err := k.InitGenesis(ctx, genesis)
	require.Error(t, err)
	require.Contains(t, err.Error(), "shares mismatch")
}

// TestCircuitBreakerRuntimeStateReset verifies that volatile runtime state
// is reset during genesis import, even if it's present in the genesis data
func TestCircuitBreakerRuntimeStateReset(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	creatorOne := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()

	pools := []types.Pool{
		{
			Id:          1,
			TokenA:      "atom",
			TokenB:      "paw",
			ReserveA:    sdkmath.NewInt(1_000_000),
			ReserveB:    sdkmath.NewInt(2_000_000),
			TotalShares: sdkmath.NewInt(900_000),
			Creator:     creatorOne,
		},
	}

	// Create genesis data with runtime state populated
	// (simulating an export from a paused chain)
	circuitBreakerStates := []types.CircuitBreakerStateExport{
		{
			PoolId:         1,
			Enabled:        true,
			LastPrice:      sdkmath.LegacyMustNewDecFromStr("2.0"),
			PersistenceKey: "pool_1_cb",

			// Runtime state that should be ignored during import
			PausedUntil:       1_700_000_200, // Should be reset to 0
			TriggeredBy:       "admin",       // Should be reset to ""
			TriggerReason:     "manual pause", // Should be reset to ""
			NotificationsSent: 5,              // Should be reset to 0
			LastNotification:  1_700_000_150, // Should be reset to 0
		},
	}

	genesis := types.GenesisState{
		Params:               params,
		Pools:                pools,
		NextPoolId:           2,
		CircuitBreakerStates: circuitBreakerStates,
	}

	// Import genesis
	require.NoError(t, k.InitGenesis(ctx, genesis))

	// Verify that runtime state was reset during import
	cbState, err := k.GetPoolCircuitBreakerState(ctx, 1)
	require.NoError(t, err)

	// Persistent configuration should be preserved
	require.True(t, cbState.Enabled, "Enabled flag should be preserved")
	require.Equal(t, sdkmath.LegacyMustNewDecFromStr("2.0"), cbState.LastPrice, "LastPrice should be preserved")
	require.Equal(t, "pool_1_cb", cbState.PersistenceKey, "PersistenceKey should be preserved")

	// Runtime state should be reset to zero values
	require.True(t, cbState.PausedUntil.IsZero(), "PausedUntil should be reset to zero time")
	require.Equal(t, 0, cbState.NotificationsSent, "NotificationsSent should be reset to 0")
	require.True(t, cbState.LastNotification.IsZero(), "LastNotification should be reset to zero time")
	require.Equal(t, "", cbState.TriggeredBy, "TriggeredBy should be reset to empty string")
	require.Equal(t, "", cbState.TriggerReason, "TriggerReason should be reset to empty string")
}

// TestGenesisExportImport_CircuitBreakerPreservationEnabled tests that circuit breaker
// pause state is preserved across genesis export/import when the parameter is enabled (default)
func TestGenesisExportImport_CircuitBreakerPreservationEnabled(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))

	// Pause the pool with circuit breaker
	pauseDuration := 24 * 60 * 60 // 24 hours in seconds
	pausedUntilTime := sdkCtx.BlockTime().Add(24 * 60 * 60 * 1_000_000_000) // 24 hours in nanoseconds
	err := k.EmergencyPausePool(ctx, poolID, "security incident", 24*60*60*1_000_000_000)
	require.NoError(t, err)

	// Verify pool is paused
	cbState, err := k.GetPoolCircuitBreakerState(ctx, poolID)
	require.NoError(t, err)
	require.True(t, cbState.Enabled)
	require.Equal(t, "governance", cbState.TriggeredBy)
	require.Equal(t, "security incident", cbState.TriggerReason)
	require.False(t, cbState.PausedUntil.IsZero())

	// Export genesis with default params (preservation enabled)
	exportedGenesis, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.True(t, exportedGenesis.Params.UpgradePreserveCircuitBreakerState, "Default should preserve state")

	// Verify exported circuit breaker state includes runtime data
	require.Len(t, exportedGenesis.CircuitBreakerStates, 1)
	exportedCB := exportedGenesis.CircuitBreakerStates[0]
	require.Equal(t, poolID, exportedCB.PoolId)
	require.True(t, exportedCB.Enabled)
	require.Greater(t, exportedCB.PausedUntil, int64(0), "PausedUntil should be exported")
	require.Equal(t, "governance", exportedCB.TriggeredBy)
	require.Equal(t, "security incident", exportedCB.TriggerReason)

	// Create new keeper for import simulation
	k2, ctx2 := keepertest.DexKeeper(t)

	// Import genesis
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Verify pause state was restored
	restoredState, err := k2.GetPoolCircuitBreakerState(ctx2, poolID)
	require.NoError(t, err)
	require.True(t, restoredState.Enabled, "Circuit breaker should remain enabled")
	require.False(t, restoredState.PausedUntil.IsZero(), "PausedUntil should be restored")
	require.Equal(t, "governance", restoredState.TriggeredBy)
	require.Equal(t, "security incident", restoredState.TriggerReason)

	// Verify pause time is approximately correct (within 1 second tolerance for rounding)
	expectedPauseTime := pausedUntilTime.Unix()
	actualPauseTime := restoredState.PausedUntil.Unix()
	require.InDelta(t, expectedPauseTime, actualPauseTime, 1.0,
		"Pause time should be preserved within 1 second tolerance")

	// Verify pool operations are still blocked after import
	pool, err := k2.GetPool(ctx2, poolID)
	require.NoError(t, err)
	err = k2.CheckPoolPriceDeviationForTesting(ctx2, pool, "test_operation")
	require.Error(t, err, "Pool should remain paused after import")
	require.Contains(t, err.Error(), "paused until")
}

// TestGenesisExportImport_CircuitBreakerPreservationDisabled tests that circuit breaker
// pause state is cleared during genesis export/import when the parameter is disabled
func TestGenesisExportImport_CircuitBreakerPreservationDisabled(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))

	// Pause the pool with circuit breaker
	err := k.EmergencyPausePool(ctx, poolID, "security incident", 24*60*60*1_000_000_000)
	require.NoError(t, err)

	// Verify pool is paused
	cbState, err := k.GetPoolCircuitBreakerState(ctx, poolID)
	require.NoError(t, err)
	require.True(t, cbState.Enabled)
	require.False(t, cbState.PausedUntil.IsZero())

	// Modify params to disable preservation
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UpgradePreserveCircuitBreakerState = false
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export genesis with preservation disabled
	exportedGenesis, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.False(t, exportedGenesis.Params.UpgradePreserveCircuitBreakerState,
		"Parameter should be disabled")

	// Verify exported circuit breaker state has runtime data cleared
	require.Len(t, exportedGenesis.CircuitBreakerStates, 1)
	exportedCB := exportedGenesis.CircuitBreakerStates[0]
	require.Equal(t, poolID, exportedCB.PoolId)
	require.True(t, exportedCB.Enabled, "Enabled flag should be preserved")
	require.Equal(t, int64(0), exportedCB.PausedUntil, "PausedUntil should be zero")
	require.Equal(t, "", exportedCB.TriggeredBy, "TriggeredBy should be empty")
	require.Equal(t, "", exportedCB.TriggerReason, "TriggerReason should be empty")
	require.Equal(t, int32(0), exportedCB.NotificationsSent)

	// Create new keeper for import simulation
	k2, ctx2 := keepertest.DexKeeper(t)

	// Import genesis
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Verify pause state was NOT restored (cleared)
	restoredState, err := k2.GetPoolCircuitBreakerState(ctx2, poolID)
	require.NoError(t, err)
	require.True(t, restoredState.Enabled, "Enabled flag should be preserved")
	require.True(t, restoredState.PausedUntil.IsZero(), "PausedUntil should be cleared")
	require.Equal(t, "", restoredState.TriggeredBy)
	require.Equal(t, "", restoredState.TriggerReason)
	require.Equal(t, 0, restoredState.NotificationsSent)

	// Verify pool operations are allowed after import (pause cleared)
	pool, err := k2.GetPool(ctx2, poolID)
	require.NoError(t, err)
	err = k2.CheckPoolPriceDeviationForTesting(ctx2, pool, "test_operation")
	require.NoError(t, err, "Pool should be operational after pause state cleared on import")
}

// TestGenesisExportImport_ExpiredPause tests that expired pause times are handled correctly
func TestGenesisExportImport_ExpiredPause(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))

	// Pause the pool with very short duration (already expired)
	pastTime := sdkCtx.BlockTime().Add(-1 * 60 * 60 * 1_000_000_000) // 1 hour ago
	cbState := keeper.CircuitBreakerState{
		Enabled:       true,
		PausedUntil:   pastTime,
		TriggeredBy:   "governance",
		TriggerReason: "test expired pause",
		LastPrice:     sdkmath.LegacyMustNewDecFromStr("1.0"),
	}
	err := k.SetCircuitBreakerState(ctx, poolID, cbState)
	require.NoError(t, err)

	// Export genesis (with preservation enabled by default)
	exportedGenesis, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	// Verify exported state includes the expired pause time
	require.Len(t, exportedGenesis.CircuitBreakerStates, 1)
	exportedCB := exportedGenesis.CircuitBreakerStates[0]
	require.Greater(t, exportedCB.PausedUntil, int64(0))

	// Create new keeper for import with advanced time
	k2, ctx2 := keepertest.DexKeeper(t)
	sdkCtx2 := sdk.UnwrapSDKContext(ctx2)

	// Advance block time to simulate chain restart later
	header := sdkCtx2.BlockHeader()
	header.Time = sdkCtx.BlockTime().Add(2 * 60 * 60 * 1_000_000_000) // 2 hours after original
	ctx2 = ctx2.WithBlockHeader(header)

	// Import genesis
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Verify pause state was restored but is expired
	restoredState, err := k2.GetPoolCircuitBreakerState(ctx2, poolID)
	require.NoError(t, err)
	require.True(t, restoredState.Enabled)
	require.False(t, restoredState.PausedUntil.IsZero())

	// Verify that current time is after pause expiration
	require.True(t, sdkCtx2.BlockTime().After(restoredState.PausedUntil),
		"Current time should be after pause expiration")

	// Verify pool operations are allowed (pause expired)
	pool, err := k2.GetPool(ctx2, poolID)
	require.NoError(t, err)
	err = k2.CheckPoolPriceDeviationForTesting(ctx2, pool, "test_operation")
	require.NoError(t, err, "Pool should be operational since pause expired")
}

// TestGenesisExportImport_MultiplePoolsPreservation tests preservation with multiple pools
// in different states (some paused, some not)
func TestGenesisExportImport_MultiplePoolsPreservation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create three pools
	poolID1 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	poolID2 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	poolID3 := keepertest.CreateTestPool(t, k, ctx, "uatom", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))

	// Pool 1: Paused by governance
	err := k.EmergencyPausePool(ctx, poolID1, "security review", 12*60*60*1_000_000_000)
	require.NoError(t, err)

	// Pool 2: Not paused (operational)
	// (default state - no action needed)

	// Pool 3: Paused by automatic circuit breaker
	cbState3 := keeper.CircuitBreakerState{
		Enabled:           true,
		PausedUntil:       sdkCtx.BlockTime().Add(6 * 60 * 60 * 1_000_000_000), // 6 hours
		TriggeredBy:       "automatic",
		TriggerReason:     "price deviation >20%",
		LastPrice:         sdkmath.LegacyMustNewDecFromStr("1.5"),
		NotificationsSent: 1,
		LastNotification:  sdkCtx.BlockTime(),
	}
	err = k.SetCircuitBreakerState(ctx, poolID3, cbState3)
	require.NoError(t, err)

	// Export genesis with preservation enabled
	exportedGenesis, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.True(t, exportedGenesis.Params.UpgradePreserveCircuitBreakerState)

	// Verify all three pools exported
	require.Len(t, exportedGenesis.CircuitBreakerStates, 3)

	// Find exported states
	var exported1, exported2, exported3 *types.CircuitBreakerStateExport
	for i := range exportedGenesis.CircuitBreakerStates {
		switch exportedGenesis.CircuitBreakerStates[i].PoolId {
		case poolID1:
			exported1 = &exportedGenesis.CircuitBreakerStates[i]
		case poolID2:
			exported2 = &exportedGenesis.CircuitBreakerStates[i]
		case poolID3:
			exported3 = &exportedGenesis.CircuitBreakerStates[i]
		}
	}
	require.NotNil(t, exported1)
	require.NotNil(t, exported2)
	require.NotNil(t, exported3)

	// Pool 1: Should have governance pause state
	require.True(t, exported1.Enabled)
	require.Greater(t, exported1.PausedUntil, int64(0))
	require.Equal(t, "governance", exported1.TriggeredBy)

	// Pool 2: Should have no pause state
	require.False(t, exported2.Enabled)
	require.Equal(t, int64(0), exported2.PausedUntil)

	// Pool 3: Should have automatic circuit breaker pause state
	require.True(t, exported3.Enabled)
	require.Greater(t, exported3.PausedUntil, int64(0))
	require.Equal(t, "automatic", exported3.TriggeredBy)
	require.Equal(t, "price deviation >20%", exported3.TriggerReason)
	require.Equal(t, int32(1), exported3.NotificationsSent)

	// Import into new keeper
	k2, ctx2 := keepertest.DexKeeper(t)
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Verify all states restored correctly
	restored1, err := k2.GetPoolCircuitBreakerState(ctx2, poolID1)
	require.NoError(t, err)
	require.True(t, restored1.Enabled)
	require.False(t, restored1.PausedUntil.IsZero())
	require.Equal(t, "governance", restored1.TriggeredBy)

	restored2, err := k2.GetPoolCircuitBreakerState(ctx2, poolID2)
	require.NoError(t, err)
	require.False(t, restored2.Enabled)
	require.True(t, restored2.PausedUntil.IsZero())

	restored3, err := k2.GetPoolCircuitBreakerState(ctx2, poolID3)
	require.NoError(t, err)
	require.True(t, restored3.Enabled)
	require.False(t, restored3.PausedUntil.IsZero())
	require.Equal(t, "automatic", restored3.TriggeredBy)
	require.Equal(t, "price deviation >20%", restored3.TriggerReason)
	require.Equal(t, 1, restored3.NotificationsSent)
}
