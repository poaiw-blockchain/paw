package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestNextOutboundNonce(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	channel := "channel-99"
	sender := "sender-outbound"

	first := k.NextOutboundNonce(sdkCtx, channel, sender)
	second := k.NextOutboundNonce(sdkCtx, channel, sender)

	require.Equal(t, uint64(1), first)
	require.Equal(t, uint64(2), second)
}
