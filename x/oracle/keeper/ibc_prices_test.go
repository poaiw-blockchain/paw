package keeper_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func setupOracleKeeper(t *testing.T) (*keeper.Keeper, sdk.Context) {
	t.Helper()
	k, _, ctx := keepertest.OracleKeeper(t)
	return k, ctx
}

func TestGetPriceMetadataFallback(t *testing.T) {
	k, ctx := setupOracleKeeper(t)

	vol, conf := k.GetPriceMetadata(ctx, "UNKNOWN/PAIR")
	require.True(t, vol.Equal(math.ZeroInt()))
	require.True(t, conf.IsPositive())
}

func TestGetPriceMetadataFromSnapshots(t *testing.T) {
	k, ctx := setupOracleKeeper(t)
	baseTime := ctx.BlockTime()

	price := types.Price{
		Asset:         "ATOM/USD",
		Price:         math.LegacyMustNewDecFromStr("100.0"),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     baseTime.Unix(),
		NumValidators: 4,
	}
	require.NoError(t, k.SetPrice(ctx, price))

	snapshots := []types.PriceSnapshot{
		{Asset: price.Asset, Price: price.Price, BlockHeight: price.BlockHeight, BlockTime: baseTime.Unix()},
		{Asset: price.Asset, Price: math.LegacyMustNewDecFromStr("101.5"), BlockHeight: price.BlockHeight + 1, BlockTime: baseTime.Add(5 * time.Minute).Unix()},
		{Asset: price.Asset, Price: math.LegacyMustNewDecFromStr("99.8"), BlockHeight: price.BlockHeight + 2, BlockTime: baseTime.Add(10 * time.Minute).Unix()},
	}

	for _, snapshot := range snapshots {
		require.NoError(t, k.SetPriceSnapshot(ctx, snapshot))
	}

	ctx = ctx.WithBlockTime(baseTime.Add(15 * time.Minute))

	volume, confidence := k.GetPriceMetadata(ctx, price.Asset)
	require.True(t, volume.GT(math.ZeroInt()))
	require.True(t, confidence.GT(math.LegacyZeroDec()))
	require.True(t, confidence.LTE(math.LegacyOneDec()))
}

func TestBuildPriceDataUsesLiveState(t *testing.T) {
	k, ctx := setupOracleKeeper(t)
	baseTime := ctx.BlockTime()

	price := types.Price{
		Asset:         "PAW/USD",
		Price:         math.LegacyMustNewDecFromStr("12.3400"),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     baseTime.Unix(),
		NumValidators: 3,
	}
	require.NoError(t, k.SetPrice(ctx, price))

	snapshots := []types.PriceSnapshot{
		{Asset: price.Asset, Price: price.Price, BlockHeight: price.BlockHeight, BlockTime: baseTime.Unix()},
		{Asset: price.Asset, Price: math.LegacyMustNewDecFromStr("12.5"), BlockHeight: price.BlockHeight + 1, BlockTime: baseTime.Add(10 * time.Minute).Unix()},
	}

	for _, snapshot := range snapshots {
		require.NoError(t, k.SetPriceSnapshot(ctx, snapshot))
	}

	ctx = ctx.WithBlockTime(baseTime.Add(20 * time.Minute))

	priceData, err := k.BuildPriceData(ctx, price.Asset)
	require.NoError(t, err)
	require.Equal(t, price.Asset, priceData.Symbol)
	require.Equal(t, price.Price, priceData.Price)
	require.Equal(t, price.NumValidators, priceData.OracleCount)
	require.True(t, priceData.Volume24h.GT(math.ZeroInt()))
	require.True(t, priceData.Confidence.GT(math.LegacyZeroDec()))
}

func TestBuildPriceDataMissingPrice(t *testing.T) {
	k, ctx := setupOracleKeeper(t)

	_, err := k.BuildPriceData(ctx, "MISSING")
	require.ErrorIs(t, err, types.ErrOracleDataUnavailable)
}

func TestOracleOnAcknowledgementPacketRejectsOversizedPayload(t *testing.T) {
	k, ctx := setupOracleKeeper(t)
	ibcModule := oracle.NewIBCModule(*k, nil)

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
	require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
}

func TestOracleOnRecvPacketRejectsUnauthorizedChannel(t *testing.T) {
	k, ctx := setupOracleKeeper(t)
	ibcModule := oracle.NewIBCModule(*k, nil)

	packetData := types.QueryPricePacketData{
		Type:   types.QueryPriceType,
		Nonce:  1,
		Symbol: "PAW/USD",
		Sender: sdk.AccAddress("oracle_req_address").String(),
	}
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-77",
		types.PortID,
		"channel-0",
		clienttypes.NewHeight(1, 10),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.False(t, chAck.Success())
	require.Contains(t, chAck.GetError(), fmt.Sprintf("ABCI code: %d", types.ErrUnauthorizedChannel.ABCICode()))
}
