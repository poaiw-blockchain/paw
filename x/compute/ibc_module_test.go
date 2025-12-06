package compute_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute"
	"github.com/paw-chain/paw/x/compute/types"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// TEST-MED-1: IBC Channel Lifecycle Tests for Compute Module

func TestOnChanOpenInit_Success(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	require.Contains(t, err.Error(), "expected ORDERED channel")
}

func TestOnChanOpenInit_InvalidVersion(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanOpenConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

func TestOnChanCloseInit_DisallowUserInitiated(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanCloseInit(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "user cannot close channel")
}

func TestOnChanCloseConfirm_WithPendingOperations(t *testing.T) {
	k, ctx, _ := keepertest.ComputeKeeperWithBank(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

	channelID := "channel-0"

	// Simulate pending compute job operations with escrow
	pendingOps := []types.PendingOperation{
		{
			ChannelID:  channelID,
			Sequence:   1,
			PacketType: types.SubmitJobType,
			Sender:     sdk.AccAddress([]byte("test_requester_addr")).String(),
			JobID:      "job-001",
			EscrowAmount: &sdk.Coin{
				Denom:  "upaw",
				Amount: math.NewInt(10000),
			},
		},
		{
			ChannelID:  channelID,
			Sequence:   2,
			PacketType: types.DiscoverProvidersType,
			Sender:     sdk.AccAddress([]byte("test_requester_addr")).String(),
		},
	}

	for _, op := range pendingOps {
		k.SetPendingOperation(ctx, op)
	}

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
	k, ctx := keepertest.ComputeKeeper(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

	err := ibcModule.OnChanCloseConfirm(
		ctx,
		types.PortID,
		"channel-0",
	)

	require.NoError(t, err)
}

func TestOnChanCloseConfirm_PartialRefundFailure(t *testing.T) {
	k, ctx, _ := keepertest.ComputeKeeperWithBank(t)
	cdc := k.Codec()
	ibcModule := compute.NewIBCModule(*k, cdc)

	channelID := "channel-0"

	// Create operations with potentially problematic data
	pendingOps := []types.PendingOperation{
		{
			ChannelID:  channelID,
			Sequence:   1,
			PacketType: types.SubmitJobType,
			Sender:     "", // Invalid sender
			JobID:      "job-invalid",
		},
		{
			ChannelID:  channelID,
			Sequence:   2,
			PacketType: types.JobResultType,
			Sender:     sdk.AccAddress([]byte("valid_sender_______")).String(),
			JobID:      "nonexistent-job",
		},
	}

	for _, op := range pendingOps {
		k.SetPendingOperation(ctx, op)
	}

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
			k, ctx := keepertest.ComputeKeeper(t)
			cdc := k.Codec()
			ibcModule := compute.NewIBCModule(*k, cdc)

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
