package keeper_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestExecuteSwapFromIBCInvalidPool(t *testing.T) {
	k, ctx := setupDexKeeper(t)
	moduleAddr := k.GetModuleAddress()
	require.NotNil(t, moduleAddr)

	amountOut, err := k.ExecuteSwapSecure(ctx, moduleAddr, 999, "upaw", "uosmo", math.NewInt(1), math.NewInt(1))
	require.Error(t, err)
	require.True(t, amountOut.IsZero())
}

func setupDexKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := keepertest.DexKeeper(t)
	return k, ctx
}

func authorizeDexChannel(t testing.TB, k *keeper.Keeper, ctx sdk.Context, channelID string) {
	t.Helper()
	require.NoError(t, k.AuthorizeChannel(ctx, types.PortID, channelID))
}

func TestHandleQueryPoolsReturnsLivePoolState(t *testing.T) {
	k, ctx := setupDexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	// Set block time for timestamp validation
	ctx = ctx.WithBlockTime(time.Now())
	authorizeDexChannel(t, k, ctx, "channel-0")

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(1_000_000), math.NewInt(2_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)
	req := types.QueryPoolsPacketData{
		Type:      types.QueryPoolsType,
		Nonce:     1,
		Timestamp: ctx.BlockTime().Unix(),
		TokenA:    "upaw",
		TokenB:    "uusdt",
	}

	packetBytes, err := req.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-0",
		"counterparty-port",
		"counterparty-channel",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)

	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))

	var resp types.QueryPoolsAcknowledgement
	require.NoError(t, json.Unmarshal(chAck.GetResult(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, uint64(1), resp.Nonce)
	require.NotEmpty(t, resp.Pools)

	var match *types.PoolInfo
	for _, p := range resp.Pools {
		if p.TokenA == "upaw" && p.TokenB == "uusdt" {
			pCopy := p
			match = &pCopy
			break
		}
	}
	require.NotNil(t, match, "expected pool for upaw/uutdt in ack")

	require.Equal(t, fmt.Sprintf("pool-%d", poolID), match.PoolID)
	require.Equal(t, "paw-local", match.ChainID)
	require.Equal(t, math.NewInt(1_000_000), match.ReserveA)
	require.Equal(t, math.NewInt(2_000_000), match.ReserveB)
	require.True(t, match.SwapFee.GTE(math.LegacyZeroDec()))
	require.True(t, match.TotalShares.IsPositive())
}

func TestOnAcknowledgementPacketEmitsErrorEvent(t *testing.T) {
	k, ctx := setupDexKeeper(t)

	packet := channeltypes.NewPacket(
		[]byte(`{"type":"execute_swap","pool_id":"pool-1"}`),
		1,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	errAck := channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid swap"))
	require.NoError(t, k.OnAcknowledgementPacket(ctx, packet, errAck))

	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
	found := false
	for _, evt := range events {
		if evt.Type == "dex_acknowledgement_error" {
			found = true
			break
		}
	}
	require.True(t, found, "expected dex_acknowledgement_error event")
}

func TestOnRecvPacketRejectsDuplicateNonce(t *testing.T) {
	k, ctx := setupDexKeeper(t)
	ibcModule := dex.NewIBCModule(*k, nil)

	ctx = ctx.WithChainID("paw-local")
	// Set block time for timestamp validation
	ctx = ctx.WithBlockTime(time.Now())
	authorizeDexChannel(t, k, ctx, "channel-0")
	keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(1_000_000), math.NewInt(2_000_000))

	packetData := types.NewQueryPoolsPacket("upaw", "uusdt", 1)
	packetData.Timestamp = ctx.BlockTime().Unix()
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	firstAck := ibcModule.OnRecvPacket(ctx, packet, nil)
	var first channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(firstAck.Acknowledgement(), &first))
	require.True(t, first.Success())

	secondAck := ibcModule.OnRecvPacket(ctx, packet, nil)
	var second channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(secondAck.Acknowledgement(), &second))
	require.False(t, second.Success())
}

func TestOnAcknowledgementPacketRejectsOversizedPayload(t *testing.T) {
	k, ctx := setupDexKeeper(t)
	ibcModule := dex.NewIBCModule(*k, nil)

	packet := channeltypes.NewPacket(
		nil,
		1,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(0, 10),
		0,
	)

	oversizedAck := bytes.Repeat([]byte{0x1}, 2*1024*1024)
	err := ibcModule.OnAcknowledgementPacket(ctx, packet, oversizedAck, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ack too large")
}

func TestOnRecvPacketRejectsUnauthorizedChannel(t *testing.T) {
	k, ctx := setupDexKeeper(t)
	ibcModule := dex.NewIBCModule(*k, nil)

	packetData := types.NewQueryPoolsPacket("upaw", "uusdt", 1)
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-99",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.False(t, chAck.Success())
	require.Contains(t, chAck.GetError(), fmt.Sprintf("ABCI code: %d", types.ErrUnauthorizedChannel.ABCICode()))
}
