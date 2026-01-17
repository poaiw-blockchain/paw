package keeper_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// seedPendingRemoteSwap writes a pending swap record exactly as storePendingRemoteSwap would.
func seedPendingRemoteSwap(t *testing.T, k *keeper.Keeper, ctx sdk.Context, seq uint64, sender sdk.AccAddress, amountIn math.Int, tokenIn, channelID string) {
	t.Helper()
	store := ctx.KVStore(k.GetStoreKey())
	data := map[string]interface{}{
		"swap_seq":       seq,
		"transfer_seq":   seq - 1,
		"sender":         sender.String(),
		"amount_in":      amountIn.String(),
		"chain_id":       "osmosis-1",
		"pool_id":        "remote-pool-1",
		"token_in":       tokenIn,
		"token_out":      "uosmo",
		"min_amount_out": amountIn.QuoRaw(2).String(),
		"channel_id":     channelID,
	}
	bz, err := json.Marshal(data)
	require.NoError(t, err)
	store.Set([]byte(fmt.Sprintf("pending_remote_swap_%d", seq)), bz)
}

func TestOnAcknowledgementPacket_RemoteSwapSuccess(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithEventManager(sdk.NewEventManager())

	seq := uint64(11)
	channelID := "channel-0"
	packetData, _ := json.Marshal(map[string]interface{}{"type": keeper.PacketTypeExecuteSwap})
	packet := channeltypes.Packet{
		Data:          packetData,
		SourceChannel: channelID,
		Sequence:      seq,
	}

	sender := dextypes.TestAddr()
	amountIn := math.NewInt(500_000)
	seedPendingRemoteSwap(t, k, ctx, seq, sender, amountIn, "upaw", channelID)

	ackStruct := keeper.ExecuteSwapPacketAck{
		Success:   true,
		AmountOut: math.NewInt(480_000),
		SwapFee:   math.NewInt(1_000),
	}
	ackBytes, _ := json.Marshal(ackStruct)
	ack := channeltypes.NewResultAcknowledgement(ackBytes)

	err := k.OnAcknowledgementPacket(ctx, packet, ack)
	require.NoError(t, err)

	// Pending swap should be cleared
	store := ctx.KVStore(k.GetStoreKey())
	require.Nil(t, store.Get([]byte(fmt.Sprintf("pending_remote_swap_%d", seq))))

	// Result should be persisted for observability
	resultKey := []byte(fmt.Sprintf("remote_swap_result_%d", seq))
	result := store.Get(resultKey)
	require.NotNil(t, result)
	var parsed map[string]string
	require.NoError(t, json.Unmarshal(result, &parsed))
	require.Equal(t, "480000", parsed["amount_out"])
	require.Equal(t, "1000", parsed["swap_fee"])

	// Event emitted
	events := ctx.EventManager().Events()
	found := false
	for _, ev := range events {
		if ev.Type == "remote_swap_completed" {
			found = true
			break
		}
	}
	require.True(t, found, "expected remote_swap_completed event")
}

func TestOnAcknowledgementPacket_RemoteSwapFailure(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithEventManager(sdk.NewEventManager())

	seq := uint64(12)
	channelID := "channel-1"
	packetData, _ := json.Marshal(map[string]interface{}{"type": keeper.PacketTypeExecuteSwap})
	packet := channeltypes.Packet{
		Data:          packetData,
		SourceChannel: channelID,
		Sequence:      seq,
	}

	sender := dextypes.TestAddr()
	amountIn := math.NewInt(600_000)
	seedPendingRemoteSwap(t, k, ctx, seq, sender, amountIn, "upaw", channelID)

	// Application-level failure encoded in acknowledgement payload (Success=false)
	ackPayload := keeper.ExecuteSwapPacketAck{
		Success:   false,
		AmountOut: math.ZeroInt(),
		Error:     "remote slippage exceeded",
	}
	ackBz, _ := json.Marshal(ackPayload)
	ack := channeltypes.NewResultAcknowledgement(ackBz)

	err := k.OnAcknowledgementPacket(ctx, packet, ack)
	require.NoError(t, err)

	store := ctx.KVStore(k.GetStoreKey())
	require.Nil(t, store.Get([]byte(fmt.Sprintf("pending_remote_swap_%d", seq))), "pending swap should be cleared on application-level failure ack")

	events := ctx.EventManager().Events()
	foundFail := false
	for _, ev := range events {
		if ev.Type == "remote_swap_failed" {
			foundFail = true
			break
		}
	}
	require.True(t, foundFail, "expected remote_swap_failed event")
}

func TestOnTimeoutPacket_RefundsEscrow(t *testing.T) {
	k, bk, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithEventManager(sdk.NewEventManager())

	seq := uint64(20)
	channelID := "channel-2"
	sender := dextypes.TestAddr()
	amountIn := math.NewInt(300_000)
	tokenIn := "upaw"

	// Fund escrow so refund can succeed
	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	require.NoError(t, bk.MintCoins(ctx, dextypes.ModuleName, sdk.NewCoins(sdk.NewCoin(tokenIn, amountIn))))
	require.NoError(t, bk.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, escrowAddr, sdk.NewCoins(sdk.NewCoin(tokenIn, amountIn))))

	seedPendingRemoteSwap(t, k, ctx, seq, sender, amountIn, tokenIn, channelID)

	packetData, _ := json.Marshal(map[string]interface{}{"type": keeper.PacketTypeExecuteSwap})
	packet := channeltypes.Packet{
		Data:          packetData,
		SourceChannel: channelID,
		Sequence:      seq,
	}

	startBal := bk.GetBalance(ctx, sender, tokenIn).Amount

	err := k.OnTimeoutPacket(ctx, packet)
	require.NoError(t, err)

	endBal := bk.GetBalance(ctx, sender, tokenIn).Amount
	require.True(t, endBal.Sub(startBal).Equal(amountIn), "expected full refund to sender")

	// Pending cleared
	store := ctx.KVStore(k.GetStoreKey())
	require.Nil(t, store.Get([]byte(fmt.Sprintf("pending_remote_swap_%d", seq))))

	events := ctx.EventManager().Events()
	found := false
	for _, ev := range events {
		if ev.Type == "remote_swap_refunded" {
			found = true
			break
		}
	}
	require.True(t, found, "expected remote_swap_refunded event")
}

// Remote execution attempts should refund when channel capability is missing (sendIBCPacket failure).
func TestExecuteCrossChainSwap_RemoteCapabilityMissingRefund(t *testing.T) {
	k, bk, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithChainID("paw-mvp-1")

	sender := dextypes.TestAddr()
	startUpaw := bk.GetBalance(ctx, sender, "upaw").Amount
	startUosmo := bk.GetBalance(ctx, sender, "uosmo").Amount

	// Seed local pool to produce uosmo for remote hop
	seedLocalPool(t, k, ctx, "local-remote", "upaw", "uosmo", math.NewInt(1_000_000), math.NewInt(900_000), math.LegacyMustNewDecFromStr("0.003"))
	fundPoolAddress(t, bk, ctx, "local-remote", "upaw", "uosmo", math.NewInt(1_000_000), math.NewInt(900_000))

	route := keeper.CrossChainSwapRoute{
		Steps: []keeper.SwapStep{
			{
				ChainID:      ctx.ChainID(), // local step to set currentAmount/currentToken
				PoolID:       "local-remote",
				TokenIn:      "upaw",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(100_000),
				MinAmountOut: math.NewInt(50_000),
			},
			{
				ChainID:      "osmosis-1", // remote to trigger executeRemoteSwap
				PoolID:       "remote-pool-1",
				TokenIn:      "uosmo",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(100_000),
				MinAmountOut: math.NewInt(50_000),
			},
		},
	}

	_, err := k.ExecuteCrossChainSwap(ctx, sender, route, math.LegacyMustNewDecFromStr("0.5"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel capability")

	endUpaw := bk.GetBalance(ctx, sender, "upaw").Amount
	endUosmo := bk.GetBalance(ctx, sender, "uosmo").Amount

	require.True(t, startUpaw.Sub(endUpaw).Equal(math.NewInt(100_000)), "only the local step input should be spent")
	require.True(t, endUosmo.GTE(startUosmo), "uosmo output from local step should remain with user after remote failure refund")

	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	escrowBal := bk.GetBalance(ctx, escrowAddr, "upaw").Amount
	require.True(t, escrowBal.IsZero(), "escrow should be drained after refund")
}

func TestExecuteCrossChainSwap_RemoteHappyPathWithMockIBC(t *testing.T) {
	k, bk, mockChan, ctx := keepertest.DexKeeperWithIBCMock(t)
	ctx = ctx.WithBlockTime(time.Now()).WithChainID("paw-mvp-1")

	sender := dextypes.TestAddr()
	start := bk.GetBalance(ctx, sender, "upaw").Amount

	seedLocalPool(t, k, ctx, "local-seed", "upaw", "uosmo", math.NewInt(1_000_000), math.NewInt(900_000), math.LegacyMustNewDecFromStr("0.003"))
	fundPoolAddress(t, bk, ctx, "local-seed", "upaw", "uosmo", math.NewInt(1_000_000), math.NewInt(900_000))

	route := keeper.CrossChainSwapRoute{
		Steps: []keeper.SwapStep{
			{
				ChainID:      ctx.ChainID(),
				PoolID:       "local-seed",
				TokenIn:      "upaw",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(100_000),
				MinAmountOut: math.NewInt(80_000),
			},
			{
				ChainID:      "osmosis-1",
				PoolID:       "remote-pool-1",
				TokenIn:      "uosmo",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(100_000),
				MinAmountOut: math.NewInt(50_000),
			},
		},
	}

	res, err := k.ExecuteCrossChainSwap(ctx, sender, route, math.LegacyMustNewDecFromStr("2.0"))
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, math.NewInt(100_000), res.AmountIn)
	require.Equal(t, 2, mockChan.Sent, "transfer + swap packets should be sent")

	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	escrowBal := bk.GetBalance(ctx, escrowAddr, "uosmo").Amount
	require.True(t, escrowBal.GT(math.ZeroInt()), "escrow should hold remote input denom until ACK")

	// User funds debited for input
	end := bk.GetBalance(ctx, sender, "upaw").Amount
	require.True(t, start.Sub(end).Equal(math.NewInt(100_000)))
}
