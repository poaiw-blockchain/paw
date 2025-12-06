package keeper_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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
		if evt.Type == "dex_cross_chain_swap_timeout" {
			found = true
			break
		}
	}
	require.True(t, found, "expected refund event")
}

func TestOnTimeoutPacketQueryPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-5"
	sequence := uint64(100)

	// Create a pending query operation
	store := ctx.KVStore(k.GetStoreKey())
	queryKey := []byte(fmt.Sprintf("pending_query_%d", sequence))
	store.Set(queryKey, []byte("cosmos-hub:upaw:uatom"))

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"query_pools","target_chain":"cosmos-hub"}`),
		Sequence:      sequence,
		SourcePort:    types.PortID,
		SourceChannel: channelID,
	}

	// Verify pending query exists
	require.NotNil(t, store.Get(queryKey))

	// Process timeout
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify pending query was removed
	require.Nil(t, store.Get(queryKey))
}

func TestOnTimeoutPacketExecuteSwapRefundsTokens(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	sender := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20))
	amountIn := sdk.NewInt64Coin("upaw", 10000)
	channelID := "channel-3"
	sequence := uint64(55)

	// Create a pending remote swap
	store := ctx.KVStore(k.GetStoreKey())
	swapKey := []byte(fmt.Sprintf("pending_remote_swap_%d", sequence))
	pendingSwap := map[string]interface{}{
		"sender":     sender.String(),
		"amount_in":  amountIn.Amount.String(),
		"token_in":   amountIn.Denom,
		"channel_id": channelID,
	}
	swapData, err := json.Marshal(pendingSwap)
	require.NoError(t, err)
	store.Set(swapKey, swapData)

	// Create pending operation tracking using correct format
	opKeyPrefix := []byte(fmt.Sprintf("pending_op/%s/", channelID))
	seqBz := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBz, sequence)
	opKey := append(opKeyPrefix, seqBz...)
	opData := map[string]interface{}{
		"channel_id":  channelID,
		"sequence":    sequence,
		"packet_type": "execute_swap",
	}
	opBz, err := json.Marshal(opData)
	require.NoError(t, err)
	store.Set(opKey, opBz)

	// Prefund escrow address for refund (this is where tokens are held during IBC swap)
	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	coins := sdk.NewCoins(amountIn)
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, escrowAddr, coins))

	packet := channeltypes.Packet{
		Data:          []byte(fmt.Sprintf(`{"type":"execute_swap","sender":"%s"}`, sender.String())),
		Sequence:      sequence,
		SourcePort:    types.PortID,
		SourceChannel: channelID,
	}

	// Check balance before timeout
	balanceBefore := k.BankKeeper().GetBalance(ctx, sender, "upaw")

	// Process timeout - should refund tokens
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify tokens were refunded
	balanceAfter := k.BankKeeper().GetBalance(ctx, sender, "upaw")
	require.Equal(t, balanceBefore.Amount.Add(amountIn.Amount), balanceAfter.Amount)

	// Verify pending swap was cleared
	require.Nil(t, store.Get(swapKey))

	// Verify pending operation was cleared
	require.Nil(t, store.Get(opKey))

	// Verify refund event was emitted
	events := ctx.EventManager().Events()
	hasRefundEvent := false
	for _, evt := range events {
		if evt.Type == "remote_swap_refunded" {
			hasRefundEvent = true
			break
		}
	}
	require.True(t, hasRefundEvent, "expected refund event to be emitted")
}

func TestOnTimeoutPacketExecuteSwapMissingPendingSwap(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-7"
	sequence := uint64(99)

	// No pending swap exists - this simulates already processed or invalid swap

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"execute_swap","sender":"paw1addr"}`),
		Sequence:      sequence,
		SourcePort:    types.PortID,
		SourceChannel: channelID,
	}

	// Should not error even with missing pending swap
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))
}

func TestOnTimeoutPacketInvalidPacketType(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"unknown_packet_type"}`),
		Sequence:      1,
		SourcePort:    types.PortID,
		SourceChannel: "channel-0",
	}

	// Should return error for unknown packet type
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown packet type")
}

func TestOnTimeoutPacketMalformedJSON(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	packet := channeltypes.Packet{
		Data:          []byte(`{invalid json`),
		Sequence:      1,
		SourcePort:    types.PortID,
		SourceChannel: "channel-0",
	}

	// Should return error for malformed JSON
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal packet data")
}

func TestOnTimeoutPacketMissingPacketType(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	packet := channeltypes.Packet{
		Data:          []byte(`{"sender":"paw1addr"}`),
		Sequence:      1,
		SourcePort:    types.PortID,
		SourceChannel: "channel-0",
	}

	// Should return error for missing packet type
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing packet type")
}
