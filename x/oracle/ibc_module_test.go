package oracle_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle"
	"github.com/paw-chain/paw/x/oracle/types"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// TEST-MED-1: IBC Channel Lifecycle Tests for Oracle Module

func TestOnChanOpenInit_Success(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	version, err := ibcModule.OnChanOpenInit(
		ctx,
		channeltypes.UNORDERED,
		[]string{"connection-0"},
		types.PortID,
		"channel-0",
		&capabilitytypes.Capability{},
		channeltypes.Counterparty{
			PortId:    types.PortID,
			ChannelId: "channel-1",
		},
		types.IBCVersion,
	)

	require.NoError(t, err)
	require.Equal(t, types.IBCVersion, version)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
	hasChannelOpenEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelOpen {
			hasChannelOpenEvent = true
			break
		}
	}
	require.True(t, hasChannelOpenEvent)
}

func TestOnChanOpenInit_InvalidOrdering(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	_, err := ibcModule.OnChanOpenInit(
		ctx,
		channeltypes.ORDERED,
		[]string{"connection-0"},
		types.PortID,
		"channel-0",
		&capabilitytypes.Capability{},
		channeltypes.Counterparty{
			PortId:    types.PortID,
			ChannelId: "channel-1",
		},
		types.IBCVersion,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "UNORDERED")
}

func TestOnChanOpenTry_Success(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	version, err := ibcModule.OnChanOpenTry(
		ctx,
		channeltypes.UNORDERED,
		[]string{"connection-0"},
		types.PortID,
		"channel-0",
		&capabilitytypes.Capability{},
		channeltypes.Counterparty{
			PortId:    types.PortID,
			ChannelId: "channel-1",
		},
		types.IBCVersion,
	)

	require.NoError(t, err)
	require.Equal(t, types.IBCVersion, version)
}

func TestOnChanOpenAck_Success(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanOpenAck(
		ctx,
		types.PortID,
		"channel-0",
		"channel-1",
		types.IBCVersion,
	)

	require.NoError(t, err)
}

func TestOnChanOpenConfirm_Success(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanOpenConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

func TestOnChanCloseInit_DisallowUserInitiated(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanCloseInit(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "user cannot close channel")
}

func TestOnChanCloseConfirm_WithPendingOperations(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	channelID := "channel-0"

	// Simulate pending operations
	pendingOps := []types.PendingOperation{
		{
			ChannelID:  channelID,
			Sequence:   1,
			PacketType: types.QueryPriceType,
			Sender:     sdk.AccAddress([]byte("test_sender________")).String(),
		},
		{
			ChannelID:  channelID,
			Sequence:   2,
			PacketType: types.SubscribePricesType,
			Sender:     sdk.AccAddress([]byte("test_subscriber____")).String(),
		},
	}

	for _, op := range pendingOps {
		k.SetPendingOperation(ctx, op)
	}

	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err)

	// Verify cleanup event was emitted
	events := ctx.EventManager().Events()
	hasCloseEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelClose {
			hasCloseEvent = true
			break
		}
	}
	require.True(t, hasCloseEvent)
}

func TestOnChanCloseConfirm_NoPendingOperations(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	cdc := k.Codec()
	ibcModule := oracle.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

// Table-driven tests
func TestOracleChannelLifecycle_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		order       channeltypes.Order
		version     string
		portID      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid unordered channel",
			order:       channeltypes.UNORDERED,
			version:     types.IBCVersion,
			portID:      types.PortID,
			expectError: false,
		},
		{
			name:        "invalid ordered channel",
			order:       channeltypes.ORDERED,
			version:     types.IBCVersion,
			portID:      types.PortID,
			expectError: true,
			errorMsg:    "UNORDERED",
		},
		{
			name:        "invalid version",
			order:       channeltypes.UNORDERED,
			version:     "invalid",
			portID:      types.PortID,
			expectError: true,
			errorMsg:    "version",
		},
		{
			name:        "invalid port",
			order:       channeltypes.UNORDERED,
			version:     types.IBCVersion,
			portID:      "wrong-port",
			expectError: true,
			errorMsg:    "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.OracleKeeper(t)
			cdc := k.Codec()
			ibcModule := oracle.NewIBCModule(*k, cdc)

			_, err := ibcModule.OnChanOpenInit(
				ctx,
				tt.order,
				[]string{"connection-0"},
				tt.portID,
				"channel-0",
				&capabilitytypes.Capability{},
				channeltypes.Counterparty{
					PortId:    types.PortID,
					ChannelId: "channel-1",
				},
				tt.version,
			)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
