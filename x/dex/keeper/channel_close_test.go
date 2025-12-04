package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestDexOnChanCloseConfirmRefundsPendingSwap(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-99"
	sequence := uint64(55)

	user := sdk.AccAddress([]byte("dex_swap_user_____"))
	amount := sdk.NewCoin("upaw", sdkmath.NewInt(2500000))

	coins := sdk.NewCoins(amount)
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, user, coins))
	require.NoError(t, k.BankKeeper().SendCoins(ctx, user, k.GetModuleAddress(), coins))

	step := dexkeeper.SwapStep{
		ChainID:      "osmosis-1",
		PoolID:       "1",
		TokenIn:      amount.Denom,
		TokenOut:     "uosmo",
		AmountIn:     amount.Amount,
		MinAmountOut: sdkmath.NewInt(1000000),
	}
	dexkeeper.StorePendingRemoteSwapForTest(k, ctx, channelID, sequence, 1, user.String(), amount.Amount, step)
	require.Len(t, k.GetPendingOperations(ctx, channelID), 1, "expected pending swap operation")

	orig := k.BankKeeper().GetBalance(ctx, user, amount.Denom)

	ibcModule := dex.NewIBCModule(*k, nil)
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	after := k.BankKeeper().GetBalance(ctx, user, amount.Denom)
	require.Equal(t, orig.Amount, after.Amount)

	require.Len(t, k.GetPendingOperations(ctx, channelID), 0)

	foundClose := false
	for _, evt := range ctx.EventManager().Events() {
		switch evt.Type {
		case types.EventTypeChannelClose:
			foundClose = true
		}
	}
	require.True(t, foundClose, "expected channel close event")
}
