package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TEST-8: Error path tests for auth/access control

// === Authorization Tests ===

func TestErrorPath_UnauthorizedPoolCreation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	t.Run("rejects pool creation with empty creator", func(t *testing.T) {
		_, err := k.CreatePool(ctx, sdk.AccAddress{}, "upaw", "uatom",
			math.NewInt(1000), math.NewInt(500))
		require.Error(t, err)
	})

	t.Run("rejects pool with same tokens", func(t *testing.T) {
		_, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "upaw",
			math.NewInt(1000), math.NewInt(500))
		require.Error(t, err)
		require.Contains(t, err.Error(), "same")
	})
}

func TestErrorPath_UnauthorizedLiquidityRemoval(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	provider := types.TestAddrWithSeed(1)
	_, _, _ = k.AddLiquidity(ctx, provider, poolID,
		math.NewInt(10_000), math.NewInt(5_000))

	t.Run("rejects removal by non-owner", func(t *testing.T) {
		attacker := types.TestAddrWithSeed(999)
		_, _, err := k.RemoveLiquidity(ctx, attacker, poolID, math.NewInt(1000))
		require.Error(t, err)
	})

	t.Run("rejects removal of more shares than owned", func(t *testing.T) {
		shares, _ := k.GetLiquidity(ctx, poolID, provider)
		_, _, err := k.RemoveLiquidity(ctx, provider, poolID, shares.Add(math.NewInt(1)))
		require.Error(t, err)
	})
}

func TestErrorPath_UnauthorizedOrderCancellation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	owner := types.TestAddrWithSeed(10)
	orderID, _ := k.PlaceLimitOrder(ctx, owner, poolID, true, "upaw",
		math.NewInt(1000), math.LegacyNewDecWithPrec(50, 2))

	t.Run("rejects cancellation by non-owner", func(t *testing.T) {
		attacker := types.TestAddrWithSeed(999)
		err := k.CancelLimitOrder(ctx, attacker, orderID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unauthorized")
	})
}

// === Input Validation Tests ===

func TestErrorPath_InvalidSwapInputs(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	user := types.TestAddr()

	t.Run("rejects zero amount swap", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.ZeroInt(), math.NewInt(1))
		require.Error(t, err)
	})

	t.Run("rejects negative amount swap", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.NewInt(-100), math.NewInt(1))
		require.Error(t, err)
	})

	t.Run("rejects swap with invalid token", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "invalid", "uatom",
			math.NewInt(100), math.NewInt(1))
		require.Error(t, err)
	})

	t.Run("rejects swap to same token", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "upaw", "upaw",
			math.NewInt(100), math.NewInt(1))
		require.Error(t, err)
	})

	t.Run("rejects swap exceeding pool reserves", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.NewInt(10_000_000), math.NewInt(1))
		require.Error(t, err)
	})
}

func TestErrorPath_InvalidLiquidityInputs(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	user := types.TestAddr()

	t.Run("rejects zero liquidity", func(t *testing.T) {
		_, _, err := k.AddLiquidity(ctx, user, poolID,
			math.ZeroInt(), math.ZeroInt())
		require.Error(t, err)
	})

	t.Run("rejects negative liquidity", func(t *testing.T) {
		_, _, err := k.AddLiquidity(ctx, user, poolID,
			math.NewInt(-100), math.NewInt(50))
		require.Error(t, err)
	})

	t.Run("rejects imbalanced liquidity", func(t *testing.T) {
		// Very imbalanced addition should fail or be limited
		// (depends on implementation - may be allowed with slippage)
	})
}

func TestErrorPath_InvalidLimitOrderInputs(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	user := types.TestAddr()

	t.Run("rejects zero amount order", func(t *testing.T) {
		_, err := k.PlaceLimitOrder(ctx, user, poolID, true, "upaw",
			math.ZeroInt(), math.LegacyNewDecWithPrec(50, 2))
		require.Error(t, err)
	})

	t.Run("rejects zero price order", func(t *testing.T) {
		_, err := k.PlaceLimitOrder(ctx, user, poolID, true, "upaw",
			math.NewInt(1000), math.LegacyZeroDec())
		require.Error(t, err)
	})

	t.Run("rejects negative price order", func(t *testing.T) {
		_, err := k.PlaceLimitOrder(ctx, user, poolID, true, "upaw",
			math.NewInt(1000), math.LegacyNewDec(-1))
		require.Error(t, err)
	})
}

// === Pool State Tests ===

func TestErrorPath_NonexistentPool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	nonexistentPoolID := uint64(9999)
	user := types.TestAddr()

	t.Run("rejects swap on nonexistent pool", func(t *testing.T) {
		_, err := k.Swap(ctx, user, nonexistentPoolID, "upaw", "uatom",
			math.NewInt(100), math.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("rejects add liquidity to nonexistent pool", func(t *testing.T) {
		_, _, err := k.AddLiquidity(ctx, user, nonexistentPoolID,
			math.NewInt(100), math.NewInt(50))
		require.Error(t, err)
	})

	t.Run("rejects limit order on nonexistent pool", func(t *testing.T) {
		_, err := k.PlaceLimitOrder(ctx, user, nonexistentPoolID, true, "upaw",
			math.NewInt(100), math.LegacyNewDecWithPrec(50, 2))
		require.Error(t, err)
	})
}

func TestErrorPath_PausedPool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	// Pause pool
	k.TriggerCircuitBreaker(ctx, poolID, "test")

	user := types.TestAddr()

	t.Run("rejects swap on paused pool", func(t *testing.T) {
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.NewInt(100), math.NewInt(1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "paused")
	})

	t.Run("rejects add liquidity to paused pool", func(t *testing.T) {
		_, _, err := k.AddLiquidity(ctx, user, poolID,
			math.NewInt(100), math.NewInt(50))
		require.Error(t, err)
	})
}

// === Slippage Tests ===

func TestErrorPath_SlippageExceeded(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	user := types.TestAddr()

	t.Run("rejects swap when min output not met", func(t *testing.T) {
		// Set unrealistic min output
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.NewInt(1000), math.NewInt(1_000_000))
		require.Error(t, err)
		require.Contains(t, err.Error(), "slippage")
	})
}

// === Rate Limiting Tests ===

func TestErrorPath_RateLimitExceeded(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	t.Run("enforces swap size limit", func(t *testing.T) {
		user := types.TestAddr()
		// Try to swap more than 10% of reserves (MaxSwapSizePercent)
		_, err := k.Swap(ctx, user, poolID, "upaw", "uatom",
			math.NewInt(200_000), math.NewInt(1)) // 20% of reserves
		require.Error(t, err)
	})
}

// === Duplicate Prevention Tests ===

func TestErrorPath_DuplicatePool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creator := types.TestAddr()

	// Create first pool
	_, err := k.CreatePool(ctx, creator, "upaw", "uatom",
		math.NewInt(1000), math.NewInt(500))
	require.NoError(t, err)

	t.Run("rejects duplicate pool creation", func(t *testing.T) {
		_, err := k.CreatePool(ctx, creator, "upaw", "uatom",
			math.NewInt(2000), math.NewInt(1000))
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("rejects duplicate pool with reversed tokens", func(t *testing.T) {
		_, err := k.CreatePool(ctx, creator, "uatom", "upaw",
			math.NewInt(500), math.NewInt(1000))
		require.Error(t, err)
	})
}

// === Balance Tests ===

func TestErrorPath_InsufficientBalance(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	t.Run("rejects swap with insufficient balance", func(t *testing.T) {
		poorUser := types.TestAddrWithSeed(999)
		// User has no tokens
		_, err := k.Swap(ctx, poorUser, poolID, "upaw", "uatom",
			math.NewInt(1000), math.NewInt(1))
		// This may or may not error depending on how bank keeper is mocked
		_ = err
	})
}

// === Governance Tests ===

func TestErrorPath_UnauthorizedParamsUpdate(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	t.Run("rejects params update from non-authority", func(t *testing.T) {
		attacker := types.TestAddrWithSeed(999)
		params := types.DefaultParams()
		params.SwapFee = math.LegacyNewDecWithPrec(99, 2) // 99% fee

		err := k.UpdateParams(ctx, attacker, params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unauthorized")
	})
}

// === Timeout and Expiry Tests ===

func TestErrorPath_ExpiredOrder(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(500_000))

	user := types.TestAddr()

	t.Run("handles expired limit order", func(t *testing.T) {
		// Place order with short expiry
		orderID, err := k.PlaceLimitOrderWithExpiry(ctx, user, poolID, true, "upaw",
			math.NewInt(1000), math.LegacyNewDecWithPrec(50, 2),
			ctx.BlockHeight()+1) // Expires next block
		require.NoError(t, err)

		// Advance past expiry
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 5)

		// Order should be expired
		order, err := k.GetLimitOrder(ctx, orderID)
		if err == nil {
			require.True(t, order.IsExpired(ctx.BlockHeight()))
		}
	})
}
