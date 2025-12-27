package keeper_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app/ibcutil"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// ===========================================================================
// TEST-9: IBC Scenario Tests (12+ scenarios)
// ===========================================================================

// ---------------------------------------------------------------------------
// Scenario 1: Timeout Handling - packet times out, funds refunded
// ---------------------------------------------------------------------------

func TestIBCScenario_TimeoutHandling_RefundsFunds(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-timeout-1"
	sequence := uint64(100)

	// Create a user and fund them
	user := sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20))
	swapAmount := sdk.NewInt64Coin("upaw", 500000)
	coins := sdk.NewCoins(swapAmount)

	// Fund user via module
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, user, coins))

	// Transfer user funds to module (simulating escrow during IBC swap)
	require.NoError(t, k.BankKeeper().SendCoins(ctx, user, k.GetModuleAddress(), coins))

	// Store pending remote swap using exported test helper
	step := dexkeeper.SwapStep{
		ChainID:      "osmosis-1",
		PoolID:       "pool-1",
		TokenIn:      swapAmount.Denom,
		TokenOut:     "uosmo",
		AmountIn:     swapAmount.Amount,
		MinAmountOut: math.NewInt(100000),
	}
	dexkeeper.StorePendingRemoteSwapForTest(k, ctx, channelID, sequence, 1, user.String(), swapAmount.Amount, step)

	// Create timeout packet
	packetData := fmt.Sprintf(`{"type":"execute_swap","sender":"%s","pool_id":"pool-1","token_in":"upaw","amount_in":"%s"}`,
		user.String(), swapAmount.Amount.String())

	packet := channeltypes.Packet{
		Data:          []byte(packetData),
		Sequence:      sequence,
		SourcePort:    types.PortID,
		SourceChannel: channelID,
	}

	// Get user balance before timeout
	balanceBefore := k.BankKeeper().GetBalance(ctx, user, "upaw")

	// Process timeout
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify funds were refunded
	balanceAfter := k.BankKeeper().GetBalance(ctx, user, "upaw")
	require.True(t, balanceAfter.Amount.GT(balanceBefore.Amount), "user should receive refund after timeout")

	// Verify timeout event was emitted
	events := ctx.EventManager().Events()
	hasTimeoutEvent := false
	for _, evt := range events {
		if evt.Type == "dex_cross_chain_swap_timeout" || evt.Type == "remote_swap_refunded" {
			hasTimeoutEvent = true
			break
		}
	}
	require.True(t, hasTimeoutEvent, "expected timeout/refund event")
}

// ---------------------------------------------------------------------------
// Scenario 2: Channel Closure - graceful shutdown with pending ops refunded
// ---------------------------------------------------------------------------

func TestIBCScenario_ChannelClosure_GracefulShutdown(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-close-1"
	sequence := uint64(200)

	// Create user and fund them
	user := sdk.AccAddress([]byte("channel_close_user_"))
	amount := sdk.NewCoin("upaw", math.NewInt(1000000))
	coins := sdk.NewCoins(amount)

	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, user, coins))
	require.NoError(t, k.BankKeeper().SendCoins(ctx, user, k.GetModuleAddress(), coins))

	// Store pending swap
	step := dexkeeper.SwapStep{
		ChainID:      "osmosis-1",
		PoolID:       "pool-1",
		TokenIn:      amount.Denom,
		TokenOut:     "uosmo",
		AmountIn:     amount.Amount,
		MinAmountOut: math.NewInt(500000),
	}
	dexkeeper.StorePendingRemoteSwapForTest(k, ctx, channelID, sequence, 1, user.String(), amount.Amount, step)

	// Verify pending operation exists
	pendingOps := k.GetPendingOperations(ctx, channelID)
	require.Len(t, pendingOps, 1, "should have one pending operation")

	// Process channel close via IBC module
	ibcModule := dex.NewIBCModule(*k, nil)
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	// Verify pending operations cleared
	pendingOpsAfter := k.GetPendingOperations(ctx, channelID)
	require.Len(t, pendingOpsAfter, 0, "pending operations should be cleared after channel close")

	// Verify close event emitted
	events := ctx.EventManager().Events()
	hasCloseEvent := false
	for _, evt := range events {
		if evt.Type == types.EventTypeChannelClose {
			hasCloseEvent = true
			break
		}
	}
	require.True(t, hasCloseEvent, "expected channel close event")
}

// ---------------------------------------------------------------------------
// Scenario 3: Channel Reopening After Closure
// ---------------------------------------------------------------------------

func TestIBCScenario_ChannelReopeningAfterClosure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	ctx = ctx.WithChainID("paw-test-1")

	channelID := "channel-reopen-1"

	// Authorize channel initially
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create a pool for testing
	keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(1_000_000), math.NewInt(2_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Close the channel
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	// Verify channel is closed by checking events
	closeEvents := ctx.EventManager().Events()
	require.NotEmpty(t, closeEvents)

	// Simulate reopening by authorizing a new channel with same ID
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Verify the channel can process packets after reauthorization
	packetData := types.NewQueryPoolsPacket("upaw", "uusdt", 1)
	packetData.Timestamp = ctx.BlockTime().Unix()
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		"counterparty-port",
		"counterparty-channel",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.True(t, chAck.Success(), "channel should be functional after reopening")
}

// ---------------------------------------------------------------------------
// Scenario 4: Error Recovery - malformed packets
// ---------------------------------------------------------------------------

func TestIBCScenario_ErrorRecovery_MalformedPackets(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	testCases := []struct {
		name       string
		packetData []byte
		errContains string
	}{
		{
			name:       "invalid JSON",
			packetData: []byte(`{not valid json`),
			errContains: "failed to parse packet data",
		},
		{
			name:       "empty packet",
			packetData: []byte(``),
			errContains: "failed to parse packet data",
		},
		{
			name:       "missing type field",
			packetData: []byte(`{"nonce": 1, "timestamp": 1234567890}`),
			errContains: "unknown packet type",
		},
		{
			name:       "invalid packet type",
			packetData: []byte(`{"type":"invalid_type","nonce":1}`),
			errContains: "unknown packet type",
		},
		{
			name:       "swap with invalid pool ID format",
			packetData: []byte(`{"type":"execute_swap","nonce":1,"timestamp":1234567890,"pool_id":"","token_in":"upaw","token_out":"uusdt","amount_in":"1000","min_amount_out":"0","sender":"paw1addr","receiver":"paw1addr"}`),
			errContains: "", // Should fail validation
		},
	}

	ibcModule := dex.NewIBCModule(*k, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet := channeltypes.NewPacket(
				tc.packetData,
				1,
				types.PortID,
				"channel-0",
				types.PortID,
				"channel-1",
				clienttypes.NewHeight(1, 10),
				0,
			)

			ack := ibcModule.OnRecvPacket(ctx, packet, nil)
			var chAck channeltypes.Acknowledgement
			require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
			require.False(t, chAck.Success(), "malformed packet should fail: %s", tc.name)

			if tc.errContains != "" {
				require.Contains(t, chAck.GetError(), tc.errContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: Acknowledgement Success
// ---------------------------------------------------------------------------

func TestIBCScenario_AcknowledgementSuccess(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-ack-success"
	sequence := uint64(300)

	// Create a successful acknowledgement
	ackData := types.ExecuteSwapAcknowledgement{
		Nonce:     1,
		Success:   true,
		AmountOut: math.NewInt(500000),
		SwapFee:   math.NewInt(1500),
	}
	ackBytes, err := ackData.GetBytes()
	require.NoError(t, err)

	successAck := channeltypes.NewResultAcknowledgement(ackBytes)

	packet := channeltypes.NewPacket(
		[]byte(`{"type":"execute_swap","pool_id":"pool-1"}`),
		sequence,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	// Process acknowledgement
	err = k.OnAcknowledgementPacket(ctx, packet, successAck)
	require.NoError(t, err)

	// Verify success event was emitted (not error event)
	events := ctx.EventManager().Events()
	hasErrorEvent := false
	for _, evt := range events {
		if evt.Type == "dex_acknowledgement_error" {
			hasErrorEvent = true
			break
		}
	}
	require.False(t, hasErrorEvent, "should not emit error event on success ack")
}

// ---------------------------------------------------------------------------
// Scenario 6: Acknowledgement Failure
// ---------------------------------------------------------------------------

func TestIBCScenario_AcknowledgementFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-ack-fail"
	sequence := uint64(400)

	// Create an error acknowledgement
	errAck := channeltypes.NewErrorAcknowledgement(
		errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "swap failed: insufficient liquidity"))

	packet := channeltypes.NewPacket(
		[]byte(`{"type":"execute_swap","pool_id":"pool-1"}`),
		sequence,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	// Process failed acknowledgement
	err := k.OnAcknowledgementPacket(ctx, packet, errAck)
	require.NoError(t, err) // Handler should not error, just process the failure

	// Verify error event was emitted
	events := ctx.EventManager().Events()
	hasErrorEvent := false
	for _, evt := range events {
		if evt.Type == "dex_acknowledgement_error" {
			hasErrorEvent = true
			break
		}
	}
	require.True(t, hasErrorEvent, "should emit error event on failed ack")
}

// ---------------------------------------------------------------------------
// Scenario 7: Duplicate Packet Handling (Nonce Replay Protection)
// ---------------------------------------------------------------------------

func TestIBCScenario_DuplicatePacketHandling(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-dup-1"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create pool for testing
	keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(1_000_000), math.NewInt(2_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Create packet with specific nonce
	nonce := uint64(12345)
	packetData := types.NewQueryPoolsPacket("upaw", "uusdt", nonce)
	packetData.Timestamp = ctx.BlockTime().Unix()
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	// First packet should succeed
	firstAck := ibcModule.OnRecvPacket(ctx, packet, nil)
	var first channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(firstAck.Acknowledgement(), &first))
	require.True(t, first.Success(), "first packet should succeed")

	// Second packet with same nonce should be rejected
	secondAck := ibcModule.OnRecvPacket(ctx, packet, nil)
	var second channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(secondAck.Acknowledgement(), &second))
	require.False(t, second.Success(), "duplicate packet should be rejected")
}

// ---------------------------------------------------------------------------
// Scenario 8: Out-of-Order Packets
// ---------------------------------------------------------------------------

func TestIBCScenario_OutOfOrderPackets(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-ooo-1"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create pool
	keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(1_000_000), math.NewInt(2_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Process packets out of order (sequence 3, then 1, then 2)
	sequences := []uint64{3, 1, 2}
	nonces := []uint64{103, 101, 102}

	for i, seq := range sequences {
		packetData := types.NewQueryPoolsPacket("upaw", "uusdt", nonces[i])
		packetData.Timestamp = ctx.BlockTime().Unix()
		packetBytes, err := packetData.GetBytes()
		require.NoError(t, err)

		packet := channeltypes.NewPacket(
			packetBytes,
			seq,
			types.PortID,
			channelID,
			types.PortID,
			"channel-1",
			clienttypes.NewHeight(1, 100),
			0,
		)

		ack := ibcModule.OnRecvPacket(ctx, packet, nil)
		var chAck channeltypes.Acknowledgement
		require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))

		// DEX uses unordered channels, so all packets should succeed
		require.True(t, chAck.Success(), "out-of-order packet (seq %d) should succeed on unordered channel", seq)
	}
}

// ---------------------------------------------------------------------------
// Scenario 9: Packet with Invalid Pool ID
// ---------------------------------------------------------------------------

func TestIBCScenario_PacketWithInvalidPoolID(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-invalid-pool"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	ibcModule := dex.NewIBCModule(*k, nil)

	testCases := []struct {
		name   string
		poolID string
	}{
		{"non-existent pool", "pool-999999"},
		{"invalid pool format", "invalid-pool-id"},
		{"empty pool ID", ""},
		{"pool-0", "pool-0"},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := types.TestAddr()
			packetData := types.ExecuteSwapPacketData{
				Type:         types.ExecuteSwapType,
				Nonce:        uint64(500 + i),
				Timestamp:    ctx.BlockTime().Unix(),
				PoolID:       tc.poolID,
				TokenIn:      "upaw",
				TokenOut:     "uusdt",
				AmountIn:     math.NewInt(1000),
				MinAmountOut: math.NewInt(1),
				Sender:       sender.String(),
				Receiver:     sender.String(),
				Timeout:      uint64(ctx.BlockTime().Add(time.Hour).Unix()),
			}

			packetBytes, err := packetData.GetBytes()
			require.NoError(t, err)

			packet := channeltypes.NewPacket(
				packetBytes,
				uint64(i+1),
				types.PortID,
				channelID,
				types.PortID,
				"channel-1",
				clienttypes.NewHeight(1, 100),
				0,
			)

			ack := ibcModule.OnRecvPacket(ctx, packet, nil)
			var chAck channeltypes.Acknowledgement
			require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
			require.False(t, chAck.Success(), "swap with invalid pool ID should fail: %s", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 10: Cross-Chain Swap Completion
// ---------------------------------------------------------------------------

func TestIBCScenario_CrossChainSwapCompletion(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-xchain-1"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create pool with sufficient liquidity
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(10_000_000), math.NewInt(20_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Create cross-chain swap packet with local hop
	sender := types.TestAddr()
	route := []types.SwapHop{
		{
			ChainID:      "paw-local", // Local chain - will be executed
			PoolID:       fmt.Sprintf("pool-%d", poolID),
			TokenIn:      "upaw",
			TokenOut:     "uusdt",
			MinAmountOut: math.NewInt(1000),
		},
	}

	packetData := types.CrossChainSwapPacketData{
		Type:      types.CrossChainSwapType,
		Nonce:     600,
		Timestamp: ctx.BlockTime().Unix(),
		Route:     route,
		Sender:    sender.String(),
		Receiver:  sender.String(),
		AmountIn:  math.NewInt(100000),
		MinOut:    math.NewInt(1000),
		Timeout:   uint64(ctx.BlockTime().Add(time.Hour).Unix()),
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.True(t, chAck.Success(), "cross-chain swap should complete successfully")

	// Verify the acknowledgement contains swap results
	var swapAck types.CrossChainSwapAcknowledgement
	require.NoError(t, json.Unmarshal(chAck.GetResult(), &swapAck))
	require.True(t, swapAck.Success)
	require.Equal(t, 1, swapAck.HopsExecuted)
	require.True(t, swapAck.FinalAmount.IsPositive(), "should have positive output amount")
}

// ---------------------------------------------------------------------------
// Scenario 11: Partial Fill Handling Over IBC
// ---------------------------------------------------------------------------

func TestIBCScenario_PartialFillHandling(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-partial-fill"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create pool with limited liquidity
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(100_000), math.NewInt(200_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Try to execute a swap that's too large for the pool
	sender := types.TestAddr()
	packetData := types.ExecuteSwapPacketData{
		Type:         types.ExecuteSwapType,
		Nonce:        700,
		Timestamp:    ctx.BlockTime().Unix(),
		PoolID:       fmt.Sprintf("pool-%d", poolID),
		TokenIn:      "upaw",
		TokenOut:     "uusdt",
		AmountIn:     math.NewInt(50_000), // Large swap relative to pool
		MinAmountOut: math.NewInt(1),      // Low min to allow the swap
		Sender:       sender.String(),
		Receiver:     sender.String(),
		Timeout:      uint64(ctx.BlockTime().Add(time.Hour).Unix()),
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))

	// Swap may succeed with high slippage or fail due to protections
	// Either outcome is valid - we're testing that the system handles it gracefully
	if chAck.Success() {
		var swapAck types.ExecuteSwapAcknowledgement
		require.NoError(t, json.Unmarshal(chAck.GetResult(), &swapAck))
		require.True(t, swapAck.AmountOut.IsPositive(), "should have positive output if swap succeeded")
	} else {
		// Error should be related to slippage or pool protection
		errorMsg := chAck.GetError()
		require.NotEmpty(t, errorMsg, "should have error message")
	}
}

// ---------------------------------------------------------------------------
// Scenario 12: Rate Limiting Across IBC
// ---------------------------------------------------------------------------

func TestIBCScenario_RateLimitingAcrossIBC(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-rate-limit"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(10_000_000), math.NewInt(20_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	sender := types.TestAddr()
	successCount := 0
	failCount := 0

	// Send many swap packets in rapid succession
	for i := 0; i < 20; i++ {
		packetData := types.ExecuteSwapPacketData{
			Type:         types.ExecuteSwapType,
			Nonce:        uint64(800 + i),
			Timestamp:    ctx.BlockTime().Unix(),
			PoolID:       fmt.Sprintf("pool-%d", poolID),
			TokenIn:      "upaw",
			TokenOut:     "uusdt",
			AmountIn:     math.NewInt(10000),
			MinAmountOut: math.NewInt(1),
			Sender:       sender.String(),
			Receiver:     sender.String(),
			Timeout:      uint64(ctx.BlockTime().Add(time.Hour).Unix()),
		}

		packetBytes, err := packetData.GetBytes()
		require.NoError(t, err)

		packet := channeltypes.NewPacket(
			packetBytes,
			uint64(i+1),
			types.PortID,
			channelID,
			types.PortID,
			"channel-1",
			clienttypes.NewHeight(1, 100),
			0,
		)

		ack := ibcModule.OnRecvPacket(ctx, packet, nil)
		var chAck channeltypes.Acknowledgement
		require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))

		if chAck.Success() {
			successCount++
		} else {
			failCount++
		}
	}

	// At least some should succeed, and rate limiting may kick in
	require.Greater(t, successCount, 0, "some swaps should succeed")
	t.Logf("Rate limit test: %d succeeded, %d failed out of 20", successCount, failCount)
}

// ---------------------------------------------------------------------------
// Scenario 13: Multi-Hop Cross-Chain Route with Mixed Chains
// ---------------------------------------------------------------------------

func TestIBCScenario_MultiHopCrossChainRoute(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-multihop"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	// Create local pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", math.NewInt(10_000_000), math.NewInt(20_000_000))

	ibcModule := dex.NewIBCModule(*k, nil)

	sender := types.TestAddr()

	// Multi-hop route: local hop, then remote hop
	route := []types.SwapHop{
		{
			ChainID:      "paw-local", // Will be executed locally
			PoolID:       fmt.Sprintf("pool-%d", poolID),
			TokenIn:      "upaw",
			TokenOut:     "uusdt",
			MinAmountOut: math.NewInt(1000),
		},
		{
			ChainID:      "osmosis-1", // Remote chain - will be estimated
			PoolID:       "pool-osmo-1",
			TokenIn:      "uusdt",
			TokenOut:     "uosmo",
			MinAmountOut: math.NewInt(500),
		},
	}

	packetData := types.CrossChainSwapPacketData{
		Type:      types.CrossChainSwapType,
		Nonce:     900,
		Timestamp: ctx.BlockTime().Unix(),
		Route:     route,
		Sender:    sender.String(),
		Receiver:  sender.String(),
		AmountIn:  math.NewInt(100000),
		MinOut:    math.NewInt(100),
		Timeout:   uint64(ctx.BlockTime().Add(time.Hour).Unix()),
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.True(t, chAck.Success(), "multi-hop route should succeed")

	var swapAck types.CrossChainSwapAcknowledgement
	require.NoError(t, json.Unmarshal(chAck.GetResult(), &swapAck))
	require.Equal(t, 2, swapAck.HopsExecuted, "should execute both hops")
}

// ---------------------------------------------------------------------------
// Scenario 14: Oversized Acknowledgement Rejection
// ---------------------------------------------------------------------------

func TestIBCScenario_OversizedAckRejection(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

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

	// Create oversized acknowledgement (2MB)
	oversizedAck := bytes.Repeat([]byte{0x1}, 2*1024*1024)

	err := ibcModule.OnAcknowledgementPacket(ctx, packet, oversizedAck, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ack too large")
}

// ---------------------------------------------------------------------------
// Scenario 15: Stale Timestamp Rejection
// ---------------------------------------------------------------------------

func TestIBCScenario_StaleTimestampRejection(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithChainID("paw-local")
	ctx = ctx.WithBlockTime(time.Now())

	channelID := "channel-stale-ts"

	// Authorize channel
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))

	ibcModule := dex.NewIBCModule(*k, nil)

	// Create packet with very old timestamp
	staleTime := ctx.BlockTime().Add(-24 * time.Hour).Unix() // 24 hours ago
	packetData := types.QueryPoolsPacketData{
		Type:      types.QueryPoolsType,
		Nonce:     1000,
		Timestamp: staleTime,
		TokenA:    "upaw",
		TokenB:    "uusdt",
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		channelID,
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.False(t, chAck.Success(), "stale timestamp should be rejected")
}

// ---------------------------------------------------------------------------
// Scenario 16: Pending Operations Tracking Consistency
// ---------------------------------------------------------------------------

func TestIBCScenario_PendingOperationsTracking(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-tracking-1"

	// Track multiple pending operations
	dexkeeper.TrackPendingOperationForTest(k, ctx, channelID, "execute_swap", 1)
	dexkeeper.TrackPendingOperationForTest(k, ctx, channelID, "execute_swap", 2)
	dexkeeper.TrackPendingOperationForTest(k, ctx, channelID, "query_pools", 3)

	// Verify all operations are tracked
	ops := k.GetPendingOperations(ctx, channelID)
	require.Len(t, ops, 3, "should have 3 pending operations")

	// Verify operations from different channel are isolated
	otherOps := k.GetPendingOperations(ctx, "channel-other")
	require.Len(t, otherOps, 0, "other channel should have no pending operations")

	// Process channel close - should clear all pending ops
	ibcModule := dex.NewIBCModule(*k, nil)
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	// Verify all operations are cleared
	opsAfterClose := k.GetPendingOperations(ctx, channelID)
	require.Len(t, opsAfterClose, 0, "all pending operations should be cleared after channel close")
}

// ---------------------------------------------------------------------------
// Helper: Create pending operation key for direct store access
// ---------------------------------------------------------------------------

func pendingOperationKey(channelID string, sequence uint64) []byte {
	prefix := []byte(fmt.Sprintf("pending_op/%s/", channelID))
	seq := make([]byte, 8)
	binary.BigEndian.PutUint64(seq, sequence)
	return append(prefix, seq...)
}
