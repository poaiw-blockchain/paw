package ibc

import (
	"fmt"
	"sync"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/stretchr/testify/require"
)

// TestPacketValidatorIntegration tests the full packet validation flow.
func TestPacketValidatorIntegration(t *testing.T) {
	ctx := createTestContext()

	// Track nonce state
	nonceTracker := &statefulNonceValidator{
		nonces: make(map[string]uint64),
	}

	// Track channel authorization
	authTracker := &statefulChannelAuthorizer{
		authorizedChannels: make(map[string]bool),
	}

	pv := NewPacketValidator(nonceTracker, authTracker)

	// Authorize a channel
	authTracker.AuthorizeChannel("port-1", "channel-1")

	// First packet should succeed
	packet1 := channeltypes.Packet{
		SourcePort:    "port-1",
		SourceChannel: "channel-1",
		Sequence:      1,
	}

	err := pv.ValidateIncomingPacket(ctx, packet1, mockPacketData{valid: true}, 1, 12345, "sender1")
	require.NoError(t, err)

	// Second packet with same nonce should fail
	err = pv.ValidateIncomingPacket(ctx, packet1, mockPacketData{valid: true}, 1, 12345, "sender1")
	require.Error(t, err)

	// Second packet with higher nonce should succeed
	packet2 := packet1
	packet2.Sequence = 2

	err = pv.ValidateIncomingPacket(ctx, packet2, mockPacketData{valid: true}, 2, 12345, "sender1")
	require.NoError(t, err)

	// Packet from unauthorized channel should fail
	packet3 := channeltypes.Packet{
		SourcePort:    "port-2",
		SourceChannel: "channel-2",
		Sequence:      1,
	}

	err = pv.ValidateIncomingPacket(ctx, packet3, mockPacketData{valid: true}, 1, 12345, "sender1")
	require.Error(t, err)
}

// statefulNonceValidator implements NonceValidator with state.
type statefulNonceValidator struct {
	mu     sync.Mutex
	nonces map[string]uint64
}

func (s *statefulNonceValidator) ValidateIncomingPacketNonce(ctx sdk.Context, sourceChannel, sender string, nonce uint64, timestamp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", sourceChannel, sender)
	stored := s.nonces[key]

	if nonce <= stored {
		return fmt.Errorf("replay attack: nonce %d <= stored %d", nonce, stored)
	}

	s.nonces[key] = nonce
	return nil
}

// statefulChannelAuthorizer implements ChannelAuthorizer with state.
type statefulChannelAuthorizer struct {
	mu                 sync.Mutex
	authorizedChannels map[string]bool
}

func (s *statefulChannelAuthorizer) AuthorizeChannel(port, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%s", port, channel)
	s.authorizedChannels[key] = true
}

func (s *statefulChannelAuthorizer) IsAuthorizedChannel(ctx sdk.Context, sourcePort, sourceChannel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", sourcePort, sourceChannel)
	if !s.authorizedChannels[key] {
		return fmt.Errorf("channel not authorized")
	}
	return nil
}

// TestChannelOpenValidatorIntegration tests the full channel opening flow.
func TestChannelOpenValidatorIntegration(t *testing.T) {
	ctx := createTestContext()

	claimer := &statefulCapabilityClaimer{
		claimed: make(map[string]bool),
	}

	cov := NewChannelOpenValidator("v1", "test-port", channeltypes.ORDERED, claimer)

	// Test Init
	err := cov.ValidateChannelOpenInit(ctx, channeltypes.ORDERED, "test-port", "channel-0", nil, "v1")
	require.NoError(t, err)

	// Verify capability was claimed
	expectedPath := host.ChannelCapabilityPath("test-port", "channel-0")
	require.True(t, claimer.IsClaimed(expectedPath), "expected path: %s", expectedPath)

	// Test Try
	err = cov.ValidateChannelOpenTry(ctx, channeltypes.ORDERED, "test-port", "channel-1", nil, "v1")
	require.NoError(t, err)

	// Test Ack
	err = cov.ValidateChannelOpenAck("v1")
	require.NoError(t, err)

	// Test Ack with wrong version
	err = cov.ValidateChannelOpenAck("v2")
	require.Error(t, err)
}

// statefulCapabilityClaimer implements CapabilityClaimer with state.
type statefulCapabilityClaimer struct {
	mu      sync.Mutex
	claimed map[string]bool
}

func (s *statefulCapabilityClaimer) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.claimed[name] {
		return fmt.Errorf("capability already claimed: %s", name)
	}

	s.claimed[name] = true
	return nil
}

func (s *statefulCapabilityClaimer) IsClaimed(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claimed[name]
}

// TestChannelCloseIntegration tests the full channel close flow with pending operations.
func TestChannelCloseIntegration(t *testing.T) {
	ctx := createTestContext()

	handler := &statefulChannelCloseHandler{
		operations: make(map[string][]PendingOperation),
		refunded:   make(map[uint64]bool),
	}

	channelID := "channel-0"

	// Add some pending operations
	handler.AddOperation(channelID, PendingOperation{Sequence: 1, PacketType: "swap", Data: "data1"})
	handler.AddOperation(channelID, PendingOperation{Sequence: 2, PacketType: "query", Data: "data2"})
	handler.AddOperation(channelID, PendingOperation{Sequence: 3, PacketType: "update", Data: "data3"})

	// Close channel
	cleaned, err := HandleChannelClose(ctx, channelID, handler)
	require.NoError(t, err)
	require.Equal(t, 3, cleaned)

	// Verify all operations were refunded
	require.True(t, handler.IsRefunded(1))
	require.True(t, handler.IsRefunded(2))
	require.True(t, handler.IsRefunded(3))
}

// statefulChannelCloseHandler implements ChannelCloseHandler with state.
type statefulChannelCloseHandler struct {
	mu         sync.Mutex
	operations map[string][]PendingOperation
	refunded   map[uint64]bool
}

func (s *statefulChannelCloseHandler) AddOperation(channelID string, op PendingOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[channelID] = append(s.operations[channelID], op)
}

func (s *statefulChannelCloseHandler) GetPendingOperations(ctx sdk.Context, channelID string) []PendingOperation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.operations[channelID]
}

func (s *statefulChannelCloseHandler) RefundOnChannelClose(ctx sdk.Context, op PendingOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refunded[op.Sequence] = true
	return nil
}

func (s *statefulChannelCloseHandler) IsRefunded(sequence uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refunded[sequence]
}

// TestAcknowledgementIntegration tests acknowledgement creation and validation flow.
func TestAcknowledgementIntegration(t *testing.T) {
	ah := NewAcknowledgementHelper()

	// Create success ack
	successData := []byte("success data")
	successAck := CreateSuccessAck(successData)
	require.True(t, successAck.Success())

	// Marshal and validate
	ackBytes := channeltypes.SubModuleCdc.MustMarshalJSON(&successAck)
	validated, err := ah.ValidateAndUnmarshalAck(ackBytes)
	require.NoError(t, err)
	require.True(t, validated.Success())

	// Create error ack
	errorAck := CreateErrorAck(fmt.Errorf("test error"))
	require.False(t, errorAck.Success())

	// Marshal and validate
	errorBytes := channeltypes.SubModuleCdc.MustMarshalJSON(&errorAck)
	validatedError, err := ah.ValidateAndUnmarshalAck(errorBytes)
	require.NoError(t, err)
	require.False(t, validatedError.Success())
}

// TestEventEmitterIntegration tests event emission in a realistic flow.
func TestEventEmitterIntegration(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	// Simulate channel opening flow
	ee.EmitChannelOpenEvent(ctx, "channel_open_init", "channel-0", "port-0", "counterparty-port", "counterparty-channel-0")
	ee.EmitChannelOpenAckEvent(ctx, "channel_open_ack", "channel-0", "port-0", "counterparty-channel-0")
	ee.EmitChannelOpenConfirmEvent(ctx, "channel_open_confirm", "channel-0", "port-0")

	// Simulate packet flow
	ee.EmitPacketReceiveEvent(ctx, "packet_receive", "swap", "channel-0", 1)
	ee.EmitPacketAckEvent(ctx, "packet_ack", "channel-0", 1, true)

	// Simulate timeout
	ee.EmitPacketTimeoutEvent(ctx, "packet_timeout", "channel-0", 2)

	// Simulate channel close
	ee.EmitChannelCloseEvent(ctx, "channel_close", "channel-0", "port-0", 5)

	// Verify all events were emitted
	events := ctx.EventManager().Events()
	require.Len(t, events, 7)
}

// TestConcurrentPacketValidation tests concurrent packet validation from multiple channels/senders.
func TestConcurrentPacketValidation(t *testing.T) {
	ctx := createTestContext()

	nonceTracker := &statefulNonceValidator{
		nonces: make(map[string]uint64),
	}

	authTracker := &statefulChannelAuthorizer{
		authorizedChannels: make(map[string]bool),
	}

	pv := NewPacketValidator(nonceTracker, authTracker)

	// Authorize multiple channels
	for i := 0; i < 10; i++ {
		authTracker.AuthorizeChannel(fmt.Sprintf("port-%d", i), fmt.Sprintf("channel-%d", i))
	}

	const goroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	// Concurrent packet validation
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			channelIdx := idx % 10
			packet := channeltypes.Packet{
				SourcePort:    fmt.Sprintf("port-%d", channelIdx),
				SourceChannel: fmt.Sprintf("channel-%d", channelIdx),
				Sequence:      uint64(idx),
			}

			err := pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, uint64(idx+1), 12345, fmt.Sprintf("sender-%d", idx))
			errors <- err
		}(i)
	}

	wg.Wait()
	close(errors)

	// Most should succeed (some might have ordering issues)
	successCount := 0
	for err := range errors {
		if err == nil {
			successCount++
		}
	}

	require.Greater(t, successCount, goroutines/2, "too many validations failed")
}

// TestChannelLifecycleIntegration tests the complete channel lifecycle.
func TestChannelLifecycleIntegration(t *testing.T) {
	ctx := createTestContext()

	// Setup validators
	claimer := &statefulCapabilityClaimer{
		claimed: make(map[string]bool),
	}
	cov := NewChannelOpenValidator("v1", "test-port", channeltypes.ORDERED, claimer)

	nonceTracker := &statefulNonceValidator{
		nonces: make(map[string]uint64),
	}
	authTracker := &statefulChannelAuthorizer{
		authorizedChannels: make(map[string]bool),
	}
	pv := NewPacketValidator(nonceTracker, authTracker)

	closeHandler := &statefulChannelCloseHandler{
		operations: make(map[string][]PendingOperation),
		refunded:   make(map[uint64]bool),
	}

	ee := NewEventEmitter()

	channelID := "channel-0"
	portID := "test-port"

	// 1. Open channel (Init)
	err := cov.ValidateChannelOpenInit(ctx, channeltypes.ORDERED, portID, channelID, nil, "v1")
	require.NoError(t, err)
	ee.EmitChannelOpenEvent(ctx, "open_init", channelID, portID, "cp-port", "cp-channel")

	// 2. Authorize channel for packet flow
	authTracker.AuthorizeChannel(portID, channelID)

	// 3. Send some packets
	for i := 1; i <= 5; i++ {
		packet := channeltypes.Packet{
			SourcePort:    portID,
			SourceChannel: channelID,
			Sequence:      uint64(i),
		}

		err := pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, uint64(i), 12345, "sender1")
		require.NoError(t, err)

		ee.EmitPacketReceiveEvent(ctx, "packet_receive", "test", channelID, uint64(i))

		// Simulate pending operation
		closeHandler.AddOperation(channelID, PendingOperation{
			Sequence:   uint64(i),
			PacketType: "test",
			Data:       fmt.Sprintf("data-%d", i),
		})
	}

	// 4. Close channel with cleanup
	cleaned, err := HandleChannelClose(ctx, channelID, closeHandler)
	require.NoError(t, err)
	require.Equal(t, 5, cleaned)

	ee.EmitChannelCloseEvent(ctx, "channel_close", channelID, portID, cleaned)

	// Verify events
	events := ctx.EventManager().Events()
	require.GreaterOrEqual(t, len(events), 7) // open + 5 packets + close
}

// TestMultiChannelConcurrentOperations tests concurrent operations across multiple channels.
func TestMultiChannelConcurrentOperations(t *testing.T) {
	ctx := createTestContext()

	const numChannels = 10
	const opsPerChannel = 100

	// Setup
	nonceTracker := &statefulNonceValidator{
		nonces: make(map[string]uint64),
	}
	authTracker := &statefulChannelAuthorizer{
		authorizedChannels: make(map[string]bool),
	}

	// Authorize all channels
	for i := 0; i < numChannels; i++ {
		authTracker.AuthorizeChannel(fmt.Sprintf("port-%d", i), fmt.Sprintf("channel-%d", i))
	}

	pv := NewPacketValidator(nonceTracker, authTracker)

	var wg sync.WaitGroup

	// Each channel processes packets concurrently
	for ch := 0; ch < numChannels; ch++ {
		wg.Add(1)
		go func(channelIdx int) {
			defer wg.Done()

			for op := 1; op <= opsPerChannel; op++ {
				packet := channeltypes.Packet{
					SourcePort:    fmt.Sprintf("port-%d", channelIdx),
					SourceChannel: fmt.Sprintf("channel-%d", channelIdx),
					Sequence:      uint64(op),
				}

				_ = pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, uint64(op), 12345, "sender")
			}
		}(ch)
	}

	wg.Wait()

	// Verify nonce state is consistent
	// Each channel should have processed all operations
	for ch := 0; ch < numChannels; ch++ {
		key := fmt.Sprintf("channel-%d:sender", ch)
		require.Equal(t, uint64(opsPerChannel), nonceTracker.nonces[key])
	}
}

// TestPacketValidatorWithChannelClose tests packet validation during channel close.
func TestPacketValidatorWithChannelClose(t *testing.T) {
	ctx := createTestContext()

	nonceTracker := &statefulNonceValidator{
		nonces: make(map[string]uint64),
	}
	authTracker := &statefulChannelAuthorizer{
		authorizedChannels: make(map[string]bool),
	}
	pv := NewPacketValidator(nonceTracker, authTracker)

	closeHandler := &statefulChannelCloseHandler{
		operations: make(map[string][]PendingOperation),
		refunded:   make(map[uint64]bool),
	}

	channelID := "channel-0"
	portID := "port-0"

	// Authorize channel
	authTracker.AuthorizeChannel(portID, channelID)

	// Validate some packets
	for i := 1; i <= 10; i++ {
		packet := channeltypes.Packet{
			SourcePort:    portID,
			SourceChannel: channelID,
			Sequence:      uint64(i),
		}

		err := pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, uint64(i), 12345, "sender")
		require.NoError(t, err)

		closeHandler.AddOperation(channelID, PendingOperation{Sequence: uint64(i), PacketType: "test"})
	}

	// Close channel
	cleaned, err := HandleChannelClose(ctx, channelID, closeHandler)
	require.NoError(t, err)
	require.Equal(t, 10, cleaned)

	// After close, validation should still work (channel might be reopened)
	packet := channeltypes.Packet{
		SourcePort:    portID,
		SourceChannel: channelID,
		Sequence:      11,
	}

	err = pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, 11, 12345, "sender")
	require.NoError(t, err)
}
