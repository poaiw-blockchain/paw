package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestOnTimeoutPricePacketEmitsEvent(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	packet := channeltypes.Packet{
		Data:          []byte(`{"asset":"PAW/USD"}`),
		Sequence:      12,
		SourcePort:    types.PortID,
		SourceChannel: "channel-2",
	}

	require.NoError(t, k.OnTimeoutPricePacket(ctx, packet))

	found := false
	for _, evt := range ctx.EventManager().Events() {
		if evt.Type == "oracle_cross_chain_price_timeout" {
			found = true
			break
		}
	}
	require.True(t, found, "expected oracle_cross_chain_price_timeout event for oracle packet")
}
