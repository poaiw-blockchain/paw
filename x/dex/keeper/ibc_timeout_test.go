package keeper_test

import (
	"bytes"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestOnTimeoutSwapPacketRefundsUser(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	user := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20))
	refund := sdk.NewCoins(sdk.NewInt64Coin("upaw", 5000))

	// Prefund the DEX module account so refunds are possible.
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, refund))

	packet := channeltypes.Packet{
		Data:          []byte(fmt.Sprintf(`{"swap_id":"swap-123","user":"%s","amount":"%s"}`, user.String(), refund.String())),
		Sequence:      7,
		SourcePort:    types.PortID,
		SourceChannel: "channel-0",
	}

	before := k.BankKeeper().GetBalance(ctx, user, "upaw")

	require.NoError(t, k.OnTimeoutSwapPacket(ctx, packet))

	after := k.BankKeeper().GetBalance(ctx, user, "upaw")
	require.True(t, after.Sub(before).Amount.Equal(refund[0].Amount))

	events := ctx.EventManager().Events()
	found := false
	for _, evt := range events {
		if evt.Type == "cross_chain_swap_timeout_refund" {
			found = true
			break
		}
	}
	require.True(t, found, "expected refund event")
}
