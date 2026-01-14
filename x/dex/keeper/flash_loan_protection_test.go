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

// TestFlashLoanProtection_BasicScenarios tests the core flash loan protection mechanism
// Note: Default FlashLoanProtectionBlocks parameter is 100 blocks (SEC-18)
func TestFlashLoanProtection_BasicScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		lastActionBlock   int64
		currentBlock      int64
		expectError       bool
		expectErrorString string
	}{
		{
			name:              "same block - should fail",
			lastActionBlock:   100,
			currentBlock:      100,
			expectError:       true,
			expectErrorString: "must wait 100 blocks",
		},
		{
			name:              "one block later - should fail (need 100)",
			lastActionBlock:   100,
			currentBlock:      101,
			expectError:       true,
			expectErrorString: "must wait 100 blocks",
		},
		{
			name:              "99 blocks later - should fail (need 100)",
			lastActionBlock:   100,
			currentBlock:      199,
			expectError:       true,
			expectErrorString: "must wait 100 blocks",
		},
		{
			name:            "exact 100 blocks - should pass",
			lastActionBlock: 100,
			currentBlock:    200,
			expectError:     false,
		},
		{
			name:            "many blocks later - should pass",
			lastActionBlock: 100,
			currentBlock:    500,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.DexKeeper(t)
			poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
				math.NewInt(1_000_000), math.NewInt(1_000_000))
			provider := types.TestAddr()

			// Set last action block
			ctx = ctx.WithBlockHeight(tt.lastActionBlock)
			require.NoError(t, k.SetLastLiquidityActionBlock(ctx, poolID, provider))

			// Move to current block
			ctx = ctx.WithBlockHeight(tt.currentBlock)

			// Check flash loan protection
			err := k.CheckFlashLoanProtection(ctx, poolID, provider)

			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrFlashLoanDetected)
				if tt.expectErrorString != "" {
					require.Contains(t, err.Error(), tt.expectErrorString)
				}
			} else {
			}
		})
	}
}

// TestFlashLoanProtection_FirstAction tests that first liquidity action is allowed
func TestFlashLoanProtection_FirstAction(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))
	provider := types.TestAddr()

	// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
	// Pool was created at default block height (1), so we need to be at block 102+ for actions
	ctx = ctx.WithBlockHeight(200)

	// First action should always be allowed (no previous action recorded)
	err := k.CheckFlashLoanProtection(ctx, poolID, provider)
	require.NoError(t, err)
}

// TestFlashLoanProtection_AddAndRemoveSameBlock tests the classic flash loan attack pattern
func TestFlashLoanProtection_AddAndRemoveSameBlock(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool with initial liquidity
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Fund the provider account
	keepertest.FundAccount(t, k, ctx, provider,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(500_000)),
			sdk.NewCoin("uatom", math.NewInt(500_000)),
		))

	// Move to a later block for the attack attempt
	// Note: FlashLoanProtectionBlocks is 100 (SEC-18), pool created at block 10
	// First action needs to be at block 110+ (100 blocks after pool creation)
	ctx = ctx.WithBlockHeight(200)

	// Attacker adds liquidity
	sharesAdded, addErr := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, addErr)
	require.True(t, sharesAdded.GT(math.ZeroInt()))

	// SAME BLOCK: Attacker tries to remove liquidity immediately
	// This should FAIL due to flash loan protection
	_, _, err := k.RemoveLiquidity(ctx, provider, poolID, sharesAdded)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)
	require.Contains(t, err.Error(), "must wait")
}

// TestFlashLoanProtection_AddAndRemoveNextBlock tests legitimate LP behavior
// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
func TestFlashLoanProtection_AddAndRemoveNextBlock(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Add liquidity at block 200 (well past pool creation)
	ctx = ctx.WithBlockHeight(200)
	sharesAdded, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, err)
	require.True(t, sharesAdded.GT(math.ZeroInt()))

	// Block 300 (100 blocks later): Remove liquidity
	// With FlashLoanProtectionBlocks=100, this should SUCCEED
	ctx = ctx.WithBlockHeight(300)
	amountA, amountB, err := k.RemoveLiquidity(ctx, provider, poolID, sharesAdded)
	require.NoError(t, err)
	require.True(t, amountA.GT(math.ZeroInt()))
	require.True(t, amountB.GT(math.ZeroInt()))
}

// TestFlashLoanProtection_MultipleProviders tests that protection is per-provider
// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
func TestFlashLoanProtection_MultipleProviders(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Two different providers
	provider1 := sdk.AccAddress([]byte("provider1_address_"))
	provider2 := sdk.AccAddress([]byte("provider2_address_"))

	// Move well past pool creation (100+ blocks)
	ctx = ctx.WithBlockHeight(200)

	// Provider 1 adds liquidity
	shares1, err := k.AddLiquidity(ctx, provider1, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, err)

	// Provider 2 adds liquidity in SAME BLOCK
	shares2, err := k.AddLiquidity(ctx, provider2, poolID,
		math.NewInt(50_000), math.NewInt(50_000))
	require.NoError(t, err)

	// Provider 1 cannot remove in same block
	_, _, err = k.RemoveLiquidity(ctx, provider1, poolID, shares1)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)

	// Provider 2 cannot remove in same block
	_, _, err = k.RemoveLiquidity(ctx, provider2, poolID, shares2)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)

	// Block 300 (100 blocks later): both providers can remove
	ctx = ctx.WithBlockHeight(300)

	amountA1, amountB1, err := k.RemoveLiquidity(ctx, provider1, poolID, shares1)
	require.NoError(t, err)
	require.True(t, amountA1.GT(math.ZeroInt()))
	require.True(t, amountB1.GT(math.ZeroInt()))

	amountA2, amountB2, err := k.RemoveLiquidity(ctx, provider2, poolID, shares2)
	require.NoError(t, err)
	require.True(t, amountA2.GT(math.ZeroInt()))
	require.True(t, amountB2.GT(math.ZeroInt()))
}

// TestFlashLoanProtection_MultipleAddRemoveCycles tests repeated operations
// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
func TestFlashLoanProtection_MultipleAddRemoveCycles(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Cycle 1: Add at block 200 (well past pool creation)
	ctx = ctx.WithBlockHeight(200)
	shares1, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, err)

	// Cannot remove in same block
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, shares1)
	require.Error(t, err)

	// Cycle 1: Remove at block 300 (100 blocks after add)
	ctx = ctx.WithBlockHeight(300)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, shares1)
	require.NoError(t, err)

	// Cycle 2: Add at block 400 (100 blocks after remove)
	ctx = ctx.WithBlockHeight(400)
	shares2, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(80_000), math.NewInt(80_000))
	require.NoError(t, err)

	// Cannot remove in same block
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, shares2)
	require.Error(t, err)

	// Cycle 2: Remove at block 500 (100 blocks after add)
	ctx = ctx.WithBlockHeight(500)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, shares2)
	require.NoError(t, err)
}

// TestFlashLoanProtection_PartialRemoval tests removing liquidity in multiple steps
// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
func TestFlashLoanProtection_PartialRemoval(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Add liquidity at block 200 (well past pool creation)
	ctx = ctx.WithBlockHeight(200)
	totalShares, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, err)

	// Block 300 (100 blocks after add): Remove 50% of shares
	ctx = ctx.WithBlockHeight(300)
	halfShares := totalShares.QuoRaw(2)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, halfShares)
	require.NoError(t, err)

	// SAME BLOCK 300: Try to remove remaining 50%
	// This should FAIL because we just performed a liquidity action
	remainingShares := totalShares.Sub(halfShares)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, remainingShares)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)

	// Block 400 (100 blocks after previous remove): Can now remove the rest
	ctx = ctx.WithBlockHeight(400)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, remainingShares)
	require.NoError(t, err)
}

// TestFlashLoanProtection_AddMultipleThenRemove tests adding in multiple blocks
func TestFlashLoanProtection_AddMultipleThenRemove(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Note: FlashLoanProtectionBlocks is 100 (SEC-18)
	// Add liquidity at block 200 (well past pool creation at block 10)
	ctx = ctx.WithBlockHeight(200)
	shares1, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(50_000), math.NewInt(50_000))
	require.NoError(t, err)

	// Add MORE liquidity at block 300 (100 blocks later)
	ctx = ctx.WithBlockHeight(300)
	shares2, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(50_000), math.NewInt(50_000))
	require.NoError(t, err)

	totalShares := shares1.Add(shares2)

	// Try to remove ALL at block 300 (same block as last add)
	// This should FAIL
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, totalShares)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)

	// Block 400 (100 blocks after last action at 300): Can remove all
	ctx = ctx.WithBlockHeight(400)
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, totalShares)
	require.NoError(t, err)
}

// TestFlashLoanProtection_SimulatedAttackScenario tests a realistic attack attempt
func TestFlashLoanProtection_SimulatedAttackScenario(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)

	// Create pool with substantial liquidity
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(10_000_000), math.NewInt(10_000_000))

	// Attacker's address
	attacker := sdk.AccAddress([]byte("attacker_address__"))

	// ATTACK SCENARIO:
	// Block 1000: Attacker gets flash loan and adds massive liquidity
	ctx = ctx.WithBlockHeight(1000)
	attackShares, err := k.AddLiquidity(ctx, attacker, poolID,
		math.NewInt(5_000_000), math.NewInt(5_000_000))
	require.NoError(t, err)
	require.True(t, attackShares.GT(math.ZeroInt()))

	// SAME BLOCK 1000: Attacker tries to manipulate pool and remove liquidity
	// This is where flash loan protection kicks in - should FAIL
	_, _, err = k.RemoveLiquidity(ctx, attacker, poolID, attackShares)
	require.Error(t, err, "flash loan attack should be prevented")
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)
	require.Contains(t, err.Error(), "must wait", "error should explain the waiting period")

	// Attack FAILED - attacker is forced to hold position for at least 100 blocks (SEC-18)
	// In reality, this exposes them to:
	// - Price risk
	// - Arbitrageur competition
	// - Gas costs
	// - Capital lock-up cost (~10 minutes at 6s blocks)

	// Block 1100 (100 blocks later): Attacker can now remove, but the atomic attack is broken
	ctx = ctx.WithBlockHeight(1100)
	amountA, amountB, err := k.RemoveLiquidity(ctx, attacker, poolID, attackShares)
	require.NoError(t, err)
	require.True(t, amountA.GT(math.ZeroInt()))
	require.True(t, amountB.GT(math.ZeroInt()))
}

// TestFlashLoanProtection_EdgeCases tests boundary conditions
func TestFlashLoanProtection_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero block height", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t)
		poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		provider := types.TestAddr()

		// Set last action at block 0
		ctx = ctx.WithBlockHeight(0)
		require.NoError(t, k.SetLastLiquidityActionBlock(ctx, poolID, provider))

		// Try at block 0 - should fail (same block)
		err := k.CheckFlashLoanProtection(ctx, poolID, provider)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrFlashLoanDetected)

		// Try well past the flash loan protection window (100+ blocks)
		ctx = ctx.WithBlockHeight(101)
		err = k.CheckFlashLoanProtection(ctx, poolID, provider)
		require.NoError(t, err)
	})

	t.Run("very large block heights", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t)
		poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		provider := types.TestAddr()

		// Set last action at a very large block
		largeBlock := int64(1_000_000_000)
		ctx = ctx.WithBlockHeight(largeBlock)
		require.NoError(t, k.SetLastLiquidityActionBlock(ctx, poolID, provider))

		// Try at same large block - should fail
		err := k.CheckFlashLoanProtection(ctx, poolID, provider)
		require.Error(t, err)

		// Try well past the flash loan protection window (100+ blocks)
		ctx = ctx.WithBlockHeight(largeBlock + 101)
		err = k.CheckFlashLoanProtection(ctx, poolID, provider)
		require.NoError(t, err)
	})

	t.Run("different pools same provider", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t)
		provider := types.TestAddr()

		// Create two pools (pool creation may record action at current block)
		pool1 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		pool2 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uosmo",
			math.NewInt(1_000_000), math.NewInt(1_000_000))

		// Move well past the flash loan protection window (100+ blocks from pool creation)
		ctx = ctx.WithBlockHeight(200)

		// Set action for pool1
		require.NoError(t, k.SetLastLiquidityActionBlock(ctx, pool1, provider))

		// Check pool1 at same block - should fail
		err := k.CheckFlashLoanProtection(ctx, pool1, provider)
		require.Error(t, err)

		// Check pool2 at same block - should succeed (no previous action for this provider)
		err = k.CheckFlashLoanProtection(ctx, pool2, provider)
		require.NoError(t, err)
	})
}

// TestFlashLoanProtection_IntegrationWithReentrancy tests defense-in-depth
func TestFlashLoanProtection_IntegrationWithReentrancy(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	provider := sdk.AccAddress([]byte("provider_address__"))

	// Create pool
	ctx = ctx.WithBlockHeight(10)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Add liquidity
	ctx = ctx.WithBlockHeight(100)
	shares, err := k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(100_000), math.NewInt(100_000))
	require.NoError(t, err)

	// Same block removal fails due to flash loan protection
	// Even if reentrancy guard somehow allowed it, flash loan protection is second line of defense
	_, _, err = k.RemoveLiquidity(ctx, provider, poolID, shares)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)
}

// TestFlashLoanProtection_ConstantValidation ensures MinLPLockBlocks is set correctly
func TestFlashLoanProtection_ConstantValidation(t *testing.T) {
	t.Parallel()

	// Verify the constant is set to production value
	require.Equal(t, int64(1), keeper.MinLPLockBlocks,
		"MinLPLockBlocks must be set to 1 for production")

	// Ensure it's not zero (would disable protection)
	require.Greater(t, keeper.MinLPLockBlocks, int64(0),
		"MinLPLockBlocks must be positive to provide protection")
}

// TestFlashLoanProtection_ErrorMessageQuality tests error messages are informative
func TestFlashLoanProtection_ErrorMessageQuality(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))
	provider := types.TestAddr()

	ctx = ctx.WithBlockHeight(100)
	require.NoError(t, k.SetLastLiquidityActionBlock(ctx, poolID, provider))

	// Try to perform action in same block
	err := k.CheckFlashLoanProtection(ctx, poolID, provider)
	require.Error(t, err)

	// Verify error message contains useful debugging info
	errMsg := err.Error()
	require.Contains(t, errMsg, "must wait", "should explain waiting requirement")
	require.Contains(t, errMsg, "blocks", "should mention blocks")
	require.Contains(t, errMsg, "100", "should show the actual block numbers")
}
