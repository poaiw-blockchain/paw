package dex_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TEST-MED-1: IBC Channel Lifecycle Tests for DEX Module

// setupDexIBCModule creates a dex keeper, context, and IBC module for testing
func setupDexIBCModule(t *testing.T) (*dex.IBCModule, sdk.Context) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	ibcModule := dex.NewIBCModule(*k, cdc)
	return &ibcModule, ctx
}

func TestOnChanOpenInit_Success(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	// Valid channel opening
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
	require.True(t, hasChannelOpenEvent, "channel open event should be emitted")
}

func TestOnChanOpenInit_InvalidOrdering(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	// DEX requires UNORDERED channels, trying ORDERED should fail
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

func TestOnChanOpenInit_InvalidVersion(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	_, err := ibcModule.OnChanOpenInit(
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
		"invalid-version",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenInit_InvalidPort(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	_, err := ibcModule.OnChanOpenInit(
		ctx,
		channeltypes.UNORDERED,
		[]string{"connection-0"},
		"invalid-port",
		"channel-0",
		&capabilitytypes.Capability{},
		channeltypes.Counterparty{
			PortId:    types.PortID,
			ChannelId: "channel-1",
		},
		types.IBCVersion,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "port")
}

func TestOnChanOpenTry_Success(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

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

	// Verify event was emitted
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
}

func TestOnChanOpenTry_InvalidCounterpartyVersion(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	_, err := ibcModule.OnChanOpenTry(
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
		"invalid-version",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenAck_Success(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	err := ibcModule.OnChanOpenAck(
		ctx,
		types.PortID,
		"channel-0",
		"channel-1",
		types.IBCVersion,
	)

	require.NoError(t, err)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
	hasAckEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelOpenAck {
			hasAckEvent = true
			break
		}
	}
	require.True(t, hasAckEvent)
}

func TestOnChanOpenAck_InvalidVersion(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	err := ibcModule.OnChanOpenAck(
		ctx,
		types.PortID,
		"channel-0",
		"channel-1",
		"invalid-version",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenConfirm_Success(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	err := ibcModule.OnChanOpenConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
	hasConfirmEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelOpenConfirm {
			hasConfirmEvent = true
			break
		}
	}
	require.True(t, hasConfirmEvent)
}

func TestOnChanCloseInit_DisallowUserInitiated(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	err := ibcModule.OnChanCloseInit(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "user cannot close channel")
}

func TestOnChanCloseConfirm_WithPendingOperations(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	ibcModule := dex.NewIBCModule(*k, cdc)

	channelID := "channel-0"

	// Create a test pool to have operations against
	creator := types.TestAddr()
	_, err := k.CreatePool(ctx, creator, "tokenA", "tokenB", math.NewInt(1000000), math.NewInt(1000000))
	require.NoError(t, err)

	// Simulate pending operations
	keeper.TrackPendingOperationForTest(k, ctx, channelID, types.ExecuteSwapType, 1)
	keeper.TrackPendingOperationForTest(k, ctx, channelID, types.QueryPoolsType, 2)

	// Close channel - should cleanup pending operations
	err = ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err)

	// Verify event was emitted with cleanup count
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
	hasCloseEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelClose {
			hasCloseEvent = true
			// Check that pending operations count is recorded
			for _, attr := range event.Attributes {
				if attr.Key == "pending_operations" {
					require.NotEqual(t, "0", attr.Value)
				}
			}
			break
		}
	}
	require.True(t, hasCloseEvent)

	// Verify pending operations were cleaned up
	remainingOps := k.GetPendingOperations(ctx, channelID)
	require.Empty(t, remainingOps, "all pending operations should be refunded/cleaned")
}

func TestOnChanCloseConfirm_NoPendingOperations(t *testing.T) {
	ibcModule, ctx := setupDexIBCModule(t)

	channelID := "channel-0"

	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	require.NotEmpty(t, events)
}

func TestOnChanCloseConfirm_PartialRefundFailure(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	ibcModule := dex.NewIBCModule(*k, cdc)

	channelID := "channel-0"

	// Create operations with invalid data that might cause refund failures
	keeper.TrackPendingOperationForTest(k, ctx, channelID, types.ExecuteSwapType, 1)
	keeper.TrackPendingOperationForTest(k, ctx, channelID, types.QueryPoolsType, 2)

	// Should not fail even if some refunds fail
	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err, "channel close should succeed even with refund failures")
}

// Table-driven tests for comprehensive channel lifecycle coverage
func TestChannelLifecycle_TableDriven(t *testing.T) {
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
			version:     "v0.0.0",
			portID:      types.PortID,
			expectError: true,
			errorMsg:    "version",
		},
		{
			name:        "invalid port",
			order:       channeltypes.UNORDERED,
			version:     types.IBCVersion,
			portID:      "invalid-port",
			expectError: true,
			errorMsg:    "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ibcModule, ctx := setupDexIBCModule(t)

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
