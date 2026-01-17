package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/paw-chain/paw/app/ibcutil"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Hooks setter should set once, panic on duplicate, and allow callbacks.
func TestOracleSetHooks(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	h := &dummyOracleHooks{}
	k.SetHooks(h)

	require.Equal(t, h, k.GetHooks())
	require.Panics(t, func() { k.SetHooks(h) })

	// Invoke a hook to verify wiring
	err := k.GetHooks().AfterPriceAggregated(ctx, "ATOM", math.LegacyNewDec(100), 12345)
	require.NoError(t, err)
	require.True(t, h.afterPriceAggregated)
}

// Channel capability getter should return not-found when none stored.
func TestOracleGetChannelCapability_NotFound(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	cap, found := k.GetChannelCapability(ctx, types.PortID, "channel-0")
	require.False(t, found)
	require.Nil(t, cap)
}

// Authorized channels roundtrip via params.
func TestOracleAuthorizedChannels(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	channels := []types.AuthorizedChannel{
		{PortId: types.PortID, ChannelId: "channel-1"},
		{PortId: types.PortID, ChannelId: "channel-2"},
	}

	err := k.SetAuthorizedChannels(ctx, []ibcutil.AuthorizedChannel{
		{PortId: channels[0].PortId, ChannelId: channels[0].ChannelId},
		{PortId: channels[1].PortId, ChannelId: channels[1].ChannelId},
	})
	require.NoError(t, err)

	got, err := k.GetAuthorizedChannels(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, channels[0].PortId, got[0].PortId)
	require.Equal(t, channels[0].ChannelId, got[0].ChannelId)
}

// OnAcknowledgementPricePacket should emit failure event on error ack.
func TestOracleOnAcknowledgementPricePacket_Error(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	packet := channeltypes.Packet{
		Data: []byte(`{"asset":"ATOM"}`),
	}
	ack := channeltypes.NewResultAcknowledgement([]byte(`{"error":"timeout"}`))

	err := k.OnAcknowledgementPricePacket(ctx, packet, ack)
	require.NoError(t, err)

	found := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == "oracle_cross_chain_price_failed" {
			found = true
			requireEventAttr(t, ev, types.AttributeKeyAsset, "ATOM")
			requireEventAttr(t, ev, types.AttributeKeyError, "timeout")
		}
	}
	require.True(t, found, "expected price packet failure event")
}

// dummyOracleHooks implements types.OracleHooks for testing.
type dummyOracleHooks struct {
	afterPriceAggregated bool
}

func (d *dummyOracleHooks) AfterPriceAggregated(ctx context.Context, asset string, price math.LegacyDec, height int64) error {
	d.afterPriceAggregated = true
	return nil
}

func (d *dummyOracleHooks) AfterPriceSubmitted(ctx context.Context, validator string, asset string, price math.LegacyDec) error {
	return nil
}

func (d *dummyOracleHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	return nil
}

func (d *dummyOracleHooks) AfterValidatorSlashed(ctx context.Context, validator string, reason string) error {
	return nil
}

// Helper to assert event attributes.
func requireEventAttr(t *testing.T, ev sdk.Event, key, expected string) {
	t.Helper()
	for _, attr := range ev.Attributes {
		if string(attr.Key) == key && string(attr.Value) == expected {
			return
		}
	}
	require.Failf(t, "attribute not found", "wanted key=%s value=%s", key, expected)
}
