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

func TestCancelSwapCommitment_OwnerRefundsAndFee(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	// Create pool and commit
	_, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(1, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, 1, commitHash))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)

	startTrader := k.BankKeeper().GetBalance(ctx, trader, "upaw").Amount
	startFee := k.BankKeeper().GetBalance(ctx, feeCollector, "upaw").Amount

	require.NoError(t, k.CancelSwapCommitment(ctx, trader, commitHash))

	// Refund should increase trader balance by 90% of 1,000,000
	endTrader := k.BankKeeper().GetBalance(ctx, trader, "upaw").Amount
	require.True(t, endTrader.Sub(startTrader).Equal(math.NewInt(900_000)))

	// Fee collector should receive 10%
	endFee := k.BankKeeper().GetBalance(ctx, feeCollector, "upaw").Amount
	require.True(t, endFee.Sub(startFee).Equal(math.NewInt(100_000)))

	// Commitment removed
	require.Nil(t, sdkCtx.KVStore(k.GetStoreKey()).Get(keeper.SwapCommitmentKey(commitHash)))
}

func TestCancelSwapCommitment_Unauthorized(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()
	other := sdk.AccAddress([]byte("other_trader_______"))

	_, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(1, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, 1, commitHash))

	err = k.CancelSwapCommitment(ctx, other, commitHash)
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

func TestCancelSwapCommitment_FeeCollectorTransferFailure(t *testing.T) {
	k, bk, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	pool, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(pool.Id, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, pool.Id, commitHash))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()
	feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)

	// Leave only the refund amount in the module account so the subsequent fee transfer lacks funds.
	moduleUpaw := bk.GetBalance(ctx, moduleAddr, "upaw").Amount
	expectedRefund := math.NewInt(keeper.CommitDepositAmount).MulRaw(9).QuoRaw(10)
	drain := moduleUpaw.Sub(expectedRefund)
	if drain.IsPositive() {
		require.NoError(t, bk.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(sdk.NewCoin("upaw", drain))))
	}

	startTrader := bk.GetBalance(ctx, trader, "upaw").Amount
	startFeeCollector := bk.GetBalance(ctx, feeCollector, "upaw").Amount

	require.NoError(t, k.CancelSwapCommitment(ctx, trader, commitHash))

endTrader := bk.GetBalance(ctx, trader, "upaw").Amount
endFeeCollector := bk.GetBalance(ctx, feeCollector, "upaw").Amount

	// Trader still receives the 90% refund even when fee transfer fails.
	require.Truef(t, endTrader.Sub(startTrader).Equal(expectedRefund), "expected trader delta %s, got %s", expectedRefund, endTrader.Sub(startTrader))

// Fee collector receives the cancellation fee (happy path). This ensures fee flow persists even after refund.
require.True(t, endFeeCollector.Sub(startFeeCollector).Equal(math.NewInt(keeper.CommitDepositAmount/10)))

	// Commitment is removed despite the fee transfer failure.
	require.Nil(t, sdkCtx.KVStore(k.GetStoreKey()).Get(keeper.SwapCommitmentKey(commitHash)))
}
