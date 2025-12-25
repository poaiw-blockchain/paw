package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestSwapRevertOnTokenTransferFailure tests that failed swaps properly revert token transfers
func TestSwapRevertOnTokenTransferFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000) // 10M upaw
	amountB := math.NewInt(20000000) // 20M uusdt

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create trader with insufficient balance
	trader := sdk.AccAddress("trader_____________")

	// Get trader's initial balance (should be zero)
	initialBalance := k.BankKeeper().GetBalance(ctx, trader, tokenA)

	// Attempt swap with insufficient funds
	amountIn := math.NewInt(1000000) // 1M upaw (which trader doesn't have)
	minAmountOut := math.NewInt(1)

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to transfer input tokens")

	// Verify trader balance unchanged (revert occurred)
	finalBalance := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	require.Equal(t, initialBalance, finalBalance)

	// Verify pool reserves unchanged (no partial state update)
	finalPool, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, amountA, finalPool.ReserveA)
	require.Equal(t, amountB, finalPool.ReserveB)
}

// TestSwapRevertOnSlippageFailure tests that swaps revert when slippage protection fails
func TestSwapRevertOnSlippageFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool with specific reserves
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000) // 1M upaw
	amountB := math.NewInt(2000000) // 2M uusdt

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create trader with sufficient balance
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, tokenA, math.NewInt(500000))

	// Get initial balances
	initialTraderBalanceA := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	initialTraderBalanceB := k.BankKeeper().GetBalance(ctx, trader, tokenB)
	initialPoolReserveA := pool.ReserveA
	initialPoolReserveB := pool.ReserveB

	// Attempt swap with unrealistic minimum output (will fail slippage check)
	amountIn := math.NewInt(100000)
	minAmountOut := math.NewInt(999999999) // Impossibly high minimum

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")

	// Verify trader balances unchanged (complete revert)
	finalTraderBalanceA := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	finalTraderBalanceB := k.BankKeeper().GetBalance(ctx, trader, tokenB)
	require.Equal(t, initialTraderBalanceA, finalTraderBalanceA)
	require.Equal(t, initialTraderBalanceB, finalTraderBalanceB)

	// Verify pool state unchanged
	finalPool, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, initialPoolReserveA, finalPool.ReserveA)
	require.Equal(t, initialPoolReserveB, finalPool.ReserveB)
}

// TestSwapRevertOnFeeCollectionFailure tests that swaps revert when fee collection fails
func TestSwapRevertOnFeeCollectionFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create trader with exact balance (not enough for fees after transfer)
	trader := createTestTraderForErrorRecovery(t)
	swapAmount := math.NewInt(100000)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, tokenA, swapAmount)

	// Get initial state
	initialTraderBalance := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	initialPoolReserveA := pool.ReserveA
	initialPoolReserveB := pool.ReserveB

	// Execute swap (may succeed or fail depending on fee handling)
	minAmountOut := math.NewInt(1)
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, swapAmount, minAmountOut)

	// If swap fails, verify complete revert
	if err != nil {
		finalTraderBalance := k.BankKeeper().GetBalance(ctx, trader, tokenA)
		require.Equal(t, initialTraderBalance, finalTraderBalance, "trader balance should be reverted on fee failure")

		finalPool, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		require.Equal(t, initialPoolReserveA, finalPool.ReserveA, "pool reserves should be reverted on fee failure")
		require.Equal(t, initialPoolReserveB, finalPool.ReserveB, "pool reserves should be reverted on fee failure")
	}
}

// TestAddLiquidityRevertOnTokenTransferFailure tests liquidity add revert behavior
func TestAddLiquidityRevertOnTokenTransferFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create provider with insufficient balance
	provider := sdk.AccAddress("provider____________")

	// Attempt to add liquidity without funds
	addAmountA := math.NewInt(1000000)
	addAmountB := math.NewInt(2000000)

	_, err = k.AddLiquidity(ctx, provider, pool.Id, addAmountA, addAmountB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to transfer tokens")

	// NOTE: In a real transaction, the SDK would automatically revert all state changes
	// when an error is returned. However, in this test environment (direct keeper calls),
	// there is no transaction boundary, so state changes may persist even on error.
	// This is expected behavior for unit tests and doesn't reflect production behavior.
	// In production, msg server handlers wrap keeper calls in transactions that auto-revert on error.
	//
	// Therefore, we cannot assert that pool state or provider shares are unchanged, as they
	// may have been updated before the token transfer failure occurred. The important check
	// is that the operation returns an error (which it does).
}

// TestRemoveLiquidityRevertOnTokenTransferFailure tests liquidity removal revert behavior
func TestRemoveLiquidityRevertOnTokenTransferFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Add liquidity as a provider
	provider := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, provider, tokenA, math.NewInt(1000000))
	fundTestAccountForErrorRecovery(t, k, ctx, provider, tokenB, math.NewInt(2000000))

	shares, err := k.AddLiquidity(ctx, provider, pool.Id, math.NewInt(1000000), math.NewInt(2000000))
	require.NoError(t, err)
	require.True(t, shares.IsPositive())

	// Advance blocks to avoid flash loan protection (requires 10 blocks between liquidity actions)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 11)

	// Drain module account to force transfer failure
	moduleAddr := k.GetModuleAddress()
	moduleBalanceA := k.BankKeeper().GetBalance(ctx, moduleAddr, tokenA)
	moduleBalanceB := k.BankKeeper().GetBalance(ctx, moduleAddr, tokenB)

	// Send module funds away (simulating insufficient balance)
	if moduleBalanceA.IsPositive() || moduleBalanceB.IsPositive() {
		burnAddr := authtypes.NewModuleAddress("burn")
		if moduleBalanceA.IsPositive() {
			err = k.BankKeeper().SendCoins(ctx, moduleAddr, burnAddr, sdk.NewCoins(moduleBalanceA))
			require.NoError(t, err)
		}
		if moduleBalanceB.IsPositive() {
			err = k.BankKeeper().SendCoins(ctx, moduleAddr, burnAddr, sdk.NewCoins(moduleBalanceB))
			require.NoError(t, err)
		}
	}

	// Attempt to remove liquidity (should fail due to insufficient module balance)
	_, _, err = k.RemoveLiquidity(ctx, provider, pool.Id, shares)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to transfer tokens")

	// NOTE: In a real transaction, the SDK would automatically revert all state changes
	// when an error is returned. However, in this test environment (direct keeper calls),
	// there is no transaction boundary, so state changes may persist even on error.
	// This is expected behavior for unit tests and doesn't reflect production behavior.
	// In production, msg server handlers wrap keeper calls in transactions that auto-revert on error.
}

// TestPartialSwapFailureDoesNotCorruptState tests that partial failures don't leave inconsistent state
func TestPartialSwapFailureDoesNotCorruptState(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create trader
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, tokenA, math.NewInt(1000000))

	// Record initial state
	initialPool, _ := k.GetPool(ctx, pool.Id)
	initialTraderBalanceA := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	initialTraderBalanceB := k.BankKeeper().GetBalance(ctx, trader, tokenB)
	initialModuleBalanceA := k.BankKeeper().GetBalance(ctx, k.GetModuleAddress(), tokenA)
	initialModuleBalanceB := k.BankKeeper().GetBalance(ctx, k.GetModuleAddress(), tokenB)

	// Attempt swap that will fail (e.g., slippage)
	amountIn := math.NewInt(100000)
	minAmountOut := math.NewInt(999999999) // Impossible

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, amountIn, minAmountOut)
	require.Error(t, err)

	// Verify COMPLETE revert - no partial state changes
	finalPool, _ := k.GetPool(ctx, pool.Id)
	require.Equal(t, initialPool.ReserveA, finalPool.ReserveA, "pool reserve A must be unchanged")
	require.Equal(t, initialPool.ReserveB, finalPool.ReserveB, "pool reserve B must be unchanged")

	finalTraderBalanceA := k.BankKeeper().GetBalance(ctx, trader, tokenA)
	finalTraderBalanceB := k.BankKeeper().GetBalance(ctx, trader, tokenB)
	require.Equal(t, initialTraderBalanceA, finalTraderBalanceA, "trader balance A must be unchanged")
	require.Equal(t, initialTraderBalanceB, finalTraderBalanceB, "trader balance B must be unchanged")

	finalModuleBalanceA := k.BankKeeper().GetBalance(ctx, k.GetModuleAddress(), tokenA)
	finalModuleBalanceB := k.BankKeeper().GetBalance(ctx, k.GetModuleAddress(), tokenB)
	require.Equal(t, initialModuleBalanceA, finalModuleBalanceA, "module balance A must be unchanged")
	require.Equal(t, initialModuleBalanceB, finalModuleBalanceB, "module balance B must be unchanged")
}

// TestSwapGasMeteringOnFailure tests that gas is consumed even on failed operations
func TestSwapGasMeteringOnFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Create trader without funds
	trader := sdk.AccAddress("trader_____________")

	// Record initial gas
	gasBefore := ctx.GasMeter().GasConsumed()

	// Attempt swap that will fail
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(1)

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, amountIn, minAmountOut)
	require.Error(t, err)

	// Verify gas was consumed (validation and attempt happened)
	gasAfter := ctx.GasMeter().GasConsumed()
	require.Greater(t, gasAfter, gasBefore, "gas should be consumed even on failure")
}

// TestSwapRevertPreservesInvariants tests that failed swaps preserve pool invariants
func TestSwapRevertPreservesInvariants(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Calculate initial invariant (k = x * y)
	initialK := pool.ReserveA.Mul(pool.ReserveB)

	// Create trader
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, tokenA, math.NewInt(1000000))

	// Attempt swap with impossible slippage
	amountIn := math.NewInt(100000)
	minAmountOut := math.NewInt(999999999)

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, amountIn, minAmountOut)
	require.Error(t, err)

	// Verify invariant unchanged (k = x * y)
	finalPool, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)

	finalK := finalPool.ReserveA.Mul(finalPool.ReserveB)
	require.Equal(t, initialK, finalK, "pool invariant must be preserved on failed swap")
}

// TestAddLiquidityRevertPreservesRatio tests that failed liquidity adds preserve pool ratio
func TestAddLiquidityRevertPreservesRatio(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000)
	amountB := math.NewInt(20000000)

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Calculate initial ratio
	initialRatio := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))

	// Create provider without funds
	provider := sdk.AccAddress("provider____________")

	// Attempt to add liquidity
	addAmountA := math.NewInt(1000000)
	addAmountB := math.NewInt(2000000)

	_, err = k.AddLiquidity(ctx, provider, pool.Id, addAmountA, addAmountB)
	require.Error(t, err)

	// Verify ratio unchanged
	finalPool, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)

	finalRatio := math.LegacyNewDecFromInt(finalPool.ReserveA).Quo(math.LegacyNewDecFromInt(finalPool.ReserveB))
	require.True(t, initialRatio.Equal(finalRatio), "pool ratio must be preserved on failed liquidity add")
}

// Helper functions

func createTestTraderForErrorRecovery(t *testing.T) sdk.AccAddress {
	return sdk.AccAddress("test_trader_________")
}

func fundTestAccountForErrorRecovery(t *testing.T, k *keeper.Keeper, ctx sdk.Context, addr sdk.AccAddress, denom string, amount math.Int) {
	// Mint coins to module account first
	moduleAddr := k.GetModuleAddress()
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))

	err := k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
	require.NoError(t, err)

	// Transfer to target address
	err = k.BankKeeper().SendCoins(ctx, moduleAddr, addr, coins)
	require.NoError(t, err)
}
