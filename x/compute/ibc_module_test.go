package compute_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// TEST-MED-1: IBC Channel Lifecycle Tests for Compute Module

// setupComputeIBCModule creates a compute keeper, context, and IBC module for testing
func setupComputeIBCModule(t *testing.T) (*compute.IBCModule, sdk.Context) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	ibcModule := compute.NewIBCModule(*k, cdc)
	return &ibcModule, ctx
}

// setupComputeIBCModuleWithKeeper creates a compute keeper, context, and IBC module for testing (with keeper returned)
func setupComputeIBCModuleWithKeeper(t *testing.T) (*compute.IBCModule, *keeper.Keeper, sdk.Context) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	ibcModule := compute.NewIBCModule(*k, cdc)
	return &ibcModule, k, ctx
}

func TestOnChanOpenInit_Success(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	// Compute requires ORDERED channels
	version, err := ibcModule.OnChanOpenInit(
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
	ibcModule, ctx := setupComputeIBCModule(t)

	// Compute requires ORDERED channels, UNORDERED should fail
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
		types.IBCVersion,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "ORDERED")
}

func TestOnChanOpenInit_InvalidVersion(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

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
		"wrong-version",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenInit_InvalidPort(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	_, err := ibcModule.OnChanOpenInit(
		ctx,
		channeltypes.ORDERED,
		[]string{"connection-0"},
		"wrong-port",
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
	ibcModule, ctx := setupComputeIBCModule(t)

	version, err := ibcModule.OnChanOpenTry(
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

	require.NoError(t, err)
	require.Equal(t, types.IBCVersion, version)
}

func TestOnChanOpenTry_InvalidCounterpartyVersion(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	_, err := ibcModule.OnChanOpenTry(
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
		"invalid",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenAck_Success(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	err := ibcModule.OnChanOpenAck(
		ctx,
		types.PortID,
		"channel-0",
		"channel-1",
		types.IBCVersion,
	)

	require.NoError(t, err)
}

func TestOnChanOpenAck_InvalidVersion(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	err := ibcModule.OnChanOpenAck(
		ctx,
		types.PortID,
		"channel-0",
		"channel-1",
		"invalid",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestOnChanOpenConfirm_Success(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	err := ibcModule.OnChanOpenConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

func TestOnChanCloseInit_DisallowUserInitiated(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	err := ibcModule.OnChanCloseInit(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "user cannot close channel")
}

func TestOnChanCloseConfirm_WithPendingOperations(t *testing.T) {
	ibcModule, k, ctx := setupComputeIBCModuleWithKeeper(t)

	channelID := "channel-0"

	// Simulate pending compute job operations with escrow
	keeper.TrackPendingOperationForTest(k, ctx, keeper.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   1,
		PacketType: types.SubmitJobType,
		JobID:      "job-001",
	})
	keeper.TrackPendingOperationForTest(k, ctx, keeper.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   2,
		PacketType: types.DiscoverProvidersType,
	})

	// Close channel - should refund escrow and cleanup
	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err)

	// Verify cleanup event
	events := ctx.EventManager().Events()
	hasCloseEvent := false
	for _, event := range events {
		if event.Type == types.EventTypeChannelClose {
			hasCloseEvent = true
			// Verify pending operations count
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyPendingOperations {
					require.NotEqual(t, "0", attr.Value)
				}
			}
			break
		}
	}
	require.True(t, hasCloseEvent)

	// Verify operations were cleaned up
	remainingOps := k.GetPendingOperations(ctx, channelID)
	require.Empty(t, remainingOps)
}

func TestOnChanCloseConfirm_NoPendingOperations(t *testing.T) {
	ibcModule, ctx := setupComputeIBCModule(t)

	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

func TestOnChanCloseConfirm_PartialRefundFailure(t *testing.T) {
	ibcModule, k, ctx := setupComputeIBCModuleWithKeeper(t)

	channelID := "channel-0"

	// Create operations with potentially problematic data
	keeper.TrackPendingOperationForTest(k, ctx, keeper.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   1,
		PacketType: types.SubmitJobType,
		JobID:      "job-invalid",
	})
	keeper.TrackPendingOperationForTest(k, ctx, keeper.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   2,
		PacketType: types.JobResultType,
		JobID:      "nonexistent-job",
	})

	// Should not fail even if some refunds fail
	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		channelID,
	)

	require.NoError(t, err, "channel close should succeed despite refund failures")
}

// Table-driven tests
func TestComputeChannelLifecycle_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		order       channeltypes.Order
		version     string
		portID      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid ordered channel",
			order:       channeltypes.ORDERED,
			version:     types.IBCVersion,
			portID:      types.PortID,
			expectError: false,
		},
		{
			name:        "invalid unordered channel",
			order:       channeltypes.UNORDERED,
			version:     types.IBCVersion,
			portID:      types.PortID,
			expectError: true,
			errorMsg:    "ORDERED",
		},
		{
			name:        "invalid version",
			order:       channeltypes.ORDERED,
			version:     "v0.0.0",
			portID:      types.PortID,
			expectError: true,
			errorMsg:    "version",
		},
		{
			name:        "invalid port",
			order:       channeltypes.ORDERED,
			version:     types.IBCVersion,
			portID:      "wrong-port",
			expectError: true,
			errorMsg:    "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ibcModule, ctx := setupComputeIBCModule(t)

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
