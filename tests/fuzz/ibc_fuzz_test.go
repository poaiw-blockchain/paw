package fuzz

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// IBCPacket represents a simplified IBC packet for fuzzing
type IBCPacket struct {
	Sequence      uint64
	SourcePort    string
	SourceChannel string
	DestPort      string
	DestChannel   string
	Data          []byte
	TimeoutHeight uint64
	TimeoutStamp  uint64
}

// FuzzIBCPacketValidation tests IBC packet validation logic
func FuzzIBCPacketValidation(f *testing.F) {
	// Seed corpus with valid and edge case packets
	seeds := [][]byte{
		encodeIBCPacket(1, "transfer", "channel-0", "transfer", "channel-1", []byte("data"), 1000, 1234567890),
		encodeIBCPacket(0, "transfer", "channel-2", "oracle", "channel-1", []byte("data"), 1000, 1234567890),            // Zero sequence
		encodeIBCPacket(1, "", "channel-1", "dex", "channel-2", []byte("data"), 1000, 1234567890),                       // Empty port
		encodeIBCPacket(1, "transfer", "channel-3", "bridge", "channel-1", []byte{}, 1000, 1234567890),                  // Empty data
		encodeIBCPacket(1, "transfer", "channel-4", "transfer", "channel-6", []byte("data"), 0, 0),                      // Zero timeout
		encodeIBCPacket(^uint64(0), "transfer", "channel-0", "transfer", "channel-5", []byte("data"), 1000, 1234567890), // Max sequence
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 50 {
			return
		}

		packet := parseIBCPacket(data)
		if packet == nil {
			return
		}

		// Test packet validation
		err := validateIBCPacket(packet)

		// Validation invariants
		if packet.Sequence == 0 {
			require.Error(t, err, "Zero sequence should be invalid")
			return
		}

		if packet.SourcePort == "" || packet.DestPort == "" {
			require.Error(t, err, "Empty ports should be invalid")
			return
		}

		if packet.SourceChannel == "" || packet.DestChannel == "" {
			require.Error(t, err, "Empty channels should be invalid")
			return
		}

		if packet.TimeoutHeight == 0 && packet.TimeoutStamp == 0 {
			require.Error(t, err, "Must have at least one timeout mechanism")
			return
		}

		if err == nil {
			// Valid packet should have non-empty data
			require.NotEmpty(t, packet.Data, "Valid packet should have data")

			// Test packet hash determinism
			hash1 := hashIBCPacket(packet)
			hash2 := hashIBCPacket(packet)
			require.Equal(t, hash1, hash2, "Packet hash must be deterministic")
		}
	})
}

// FuzzIBCSequenceOrdering tests sequence number ordering and gaps
func FuzzIBCSequenceOrdering(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 80 {
			return
		}

		// Parse multiple sequence numbers
		sequences := make([]uint64, 10)
		for i := 0; i < 10; i++ {
			sequences[i] = binary.BigEndian.Uint64(data[i*8 : (i+1)*8])
		}

		tracker := newSequenceTracker()

		for _, seq := range sequences {
			if seq == 0 {
				continue
			}

			err := tracker.processSequence(seq)

			// Invariant: Sequences must be monotonically increasing
			if seq <= tracker.lastSequence && tracker.lastSequence > 0 {
				require.Error(t, err, "Out-of-order sequence should be rejected")
			}

			// Invariant: No duplicate sequences
			if tracker.hasSequence(seq) && seq != sequences[0] {
				require.Error(t, err, "Duplicate sequence should be rejected")
			}
		}

		// Test gap detection
		gaps := tracker.detectGaps(sequences[len(sequences)-1])
		require.NotNil(t, gaps, "Gap detection should return a slice even when empty")

		// Invariant: Gap count should be reasonable
		maxExpectedGaps := sequences[len(sequences)-1] - uint64(len(sequences))
		require.True(t, uint64(len(gaps)) <= maxExpectedGaps,
			"Gap count should not exceed theoretical maximum")
	})
}

// FuzzIBCTimeoutValidation tests timeout height and timestamp validation
func FuzzIBCTimeoutValidation(f *testing.F) {
	seeds := [][]byte{
		encodeTimeout(1000, 1234567890, 500, 1234567800),  // Valid
		encodeTimeout(1000, 1234567890, 1500, 1234567800), // Expired by height
		encodeTimeout(1000, 1234567890, 500, 1234569999),  // Expired by time
		encodeTimeout(0, 1234567890, 500, 1234567800),     // Height-only timeout
		encodeTimeout(1000, 0, 500, 1234567800),           // Timestamp-only timeout
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 32 {
			return
		}

		timeoutHeight := binary.BigEndian.Uint64(data[0:8])
		timeoutStamp := binary.BigEndian.Uint64(data[8:16])
		currentHeight := binary.BigEndian.Uint64(data[16:24])
		currentTime := binary.BigEndian.Uint64(data[24:32])

		isExpired := checkIBCTimeout(timeoutHeight, timeoutStamp, currentHeight, currentTime)

		// Invariant: If both timeouts are zero, packet is invalid
		if timeoutHeight == 0 && timeoutStamp == 0 {
			return
		}

		// Invariant: Timeout logic consistency
		if timeoutHeight > 0 && currentHeight >= timeoutHeight {
			require.True(t, isExpired, "Packet should be expired when current height >= timeout height")
		}

		if timeoutStamp > 0 && currentTime >= timeoutStamp {
			require.True(t, isExpired, "Packet should be expired when current time >= timeout time")
		}

		// Test timeout precedence (either timeout can trigger)
		expiredByHeight := timeoutHeight > 0 && currentHeight >= timeoutHeight
		expiredByTime := timeoutStamp > 0 && currentTime >= timeoutStamp

		if expiredByHeight || expiredByTime {
			require.True(t, isExpired, "Packet should be expired if either timeout condition is met")
		}
	})
}

// FuzzIBCChannelHandshake tests channel handshake state transitions
func FuzzIBCChannelHandshake(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}

		initialState := ChannelState(data[0] % 6) // 6 possible states
		event := HandshakeEvent(data[1] % 4)      // 4 possible events

		channel := &Channel{
			State:        initialState,
			Sequence:     binary.BigEndian.Uint64(data[2:10]) % 1000,
			Counterparty: "channel-" + string(data[10:15]),
			Version:      "ics20-1",
		}

		// Test state transition
		newState, err := transitionChannelState(channel, event)

		// Invariants based on IBC spec
		switch initialState {
		case ChannelStateInit:
			if event == EventTryOpen {
				require.NoError(t, err)
				require.Equal(t, ChannelStateTryOpen, newState)
			} else if event == EventOpen {
				require.Error(t, err, "Cannot go directly from INIT to OPEN")
			}

		case ChannelStateTryOpen:
			if event == EventAck {
				require.NoError(t, err)
				require.Equal(t, ChannelStateOpen, newState)
			}

		case ChannelStateOpen:
			if event == EventClose {
				require.NoError(t, err)
				require.Equal(t, ChannelStateClosed, newState)
			} else if event == EventTryOpen {
				require.Error(t, err, "Cannot reopen an open channel")
			}

		case ChannelStateClosed:
			// No transitions from closed state
			require.Error(t, err, "No transitions allowed from CLOSED state")
		}
	})
}

// FuzzIBCPacketRelay tests packet relaying and acknowledgement
func FuzzIBCPacketRelay(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 100 {
			return
		}

		packet := parseIBCPacket(data)
		if packet == nil {
			return
		}

		relay := newPacketRelay()

		// Test sending packet
		err := relay.sendPacket(packet)
		if err != nil {
			// Should have specific error reason
			require.NotEmpty(t, err.Error())
			return
		}

		// Invariant: Packet should be in pending state
		require.True(t, relay.isPending(packet.Sequence),
			"Sent packet should be pending")

		// Test acknowledgement
		ackData := data[50:100]
		ackSuccess := data[0]%2 == 0

		err = relay.acknowledgePacket(packet.Sequence, ackData, ackSuccess)
		if !ackSuccess {
			require.Error(t, err, "Failed acknowledgements should surface errors")
			return
		}
		require.NoError(t, err, "Acknowledging valid packet should succeed")

		// Invariant: Acknowledged packet should not be pending
		require.False(t, relay.isPending(packet.Sequence),
			"Acknowledged packet should not be pending")

		// Test duplicate acknowledgement
		err2 := relay.acknowledgePacket(packet.Sequence, ackData, ackSuccess)
		require.Error(t, err2, "Duplicate acknowledgement should fail")

		// Test timeout after acknowledgement
		err3 := relay.timeoutPacket(packet.Sequence)
		require.Error(t, err3, "Cannot timeout acknowledged packet")
	})
}

// FuzzIBCConnectionVerification tests connection proof verification
func FuzzIBCConnectionVerification(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 200 {
			return
		}

		// Parse connection proof components
		connectionID := string(data[0:20])
		clientID := string(data[20:40])
		consensusState := data[40:72] // 32 bytes
		proofHeight := binary.BigEndian.Uint64(data[72:80])
		proofBytes := data[80:200]

		// Test proof verification
		valid := verifyConnectionProof(connectionID, clientID, consensusState, proofHeight, proofBytes)

		// If proof is claimed valid, it should meet requirements
		if valid {
			require.NotEmpty(t, connectionID, "Valid proof needs connection ID")
			require.NotEmpty(t, clientID, "Valid proof needs client ID")
			require.NotEmpty(t, consensusState, "Valid proof needs consensus state")
			require.True(t, proofHeight > 0, "Valid proof needs positive height")
		}

		// Test proof tampering detection
		tamperedProof := make([]byte, len(proofBytes))
		copy(tamperedProof, proofBytes)
		if len(tamperedProof) > 0 {
			tamperedProof[0] ^= 0xFF // Flip bits
		}

		validTampered := verifyConnectionProof(connectionID, clientID, consensusState, proofHeight, tamperedProof)

		// Invariant: Tampered proof should not be more valid than original
		if !valid {
			require.False(t, validTampered, "Tampered proof should not become valid")
		}
	})
}

// Helper types and functions

type ChannelState uint8

const (
	ChannelStateUninitialized ChannelState = iota
	ChannelStateInit
	ChannelStateTryOpen
	ChannelStateOpen
	ChannelStateClosed
	ChannelStateFlush
)

type HandshakeEvent uint8

const (
	EventInit HandshakeEvent = iota
	EventTryOpen
	EventAck
	EventOpen
	EventClose
)

type Channel struct {
	State        ChannelState
	Sequence     uint64
	Counterparty string
	Version      string
}

type SequenceTracker struct {
	sequences    map[uint64]bool
	lastSequence uint64
}

type PacketRelay struct {
	pendingPackets      map[uint64]*IBCPacket
	acknowledgedPackets map[uint64][]byte
	timedOutPackets     map[uint64]bool
}

func newSequenceTracker() *SequenceTracker {
	return &SequenceTracker{
		sequences: make(map[uint64]bool),
	}
}

func (st *SequenceTracker) processSequence(seq uint64) error {
	if seq <= st.lastSequence && st.lastSequence > 0 {
		return &IBCError{"sequence out of order"}
	}

	if st.sequences[seq] {
		return &IBCError{"duplicate sequence"}
	}

	st.sequences[seq] = true
	st.lastSequence = seq
	return nil
}

func (st *SequenceTracker) hasSequence(seq uint64) bool {
	return st.sequences[seq]
}

func (st *SequenceTracker) detectGaps(maxSeq uint64) []uint64 {
	gaps := make([]uint64, 0)

	for i := uint64(1); i <= maxSeq; i++ {
		if !st.sequences[i] {
			gaps = append(gaps, i)
		}
	}

	return gaps
}

func newPacketRelay() *PacketRelay {
	return &PacketRelay{
		pendingPackets:      make(map[uint64]*IBCPacket),
		acknowledgedPackets: make(map[uint64][]byte),
		timedOutPackets:     make(map[uint64]bool),
	}
}

func (pr *PacketRelay) sendPacket(packet *IBCPacket) error {
	if packet.Sequence == 0 {
		return &IBCError{"invalid sequence"}
	}

	if pr.pendingPackets[packet.Sequence] != nil {
		return &IBCError{"packet already sent"}
	}

	pr.pendingPackets[packet.Sequence] = packet
	return nil
}

func (pr *PacketRelay) isPending(sequence uint64) bool {
	return pr.pendingPackets[sequence] != nil
}

func (pr *PacketRelay) acknowledgePacket(sequence uint64, ackData []byte, success bool) error {
	if pr.pendingPackets[sequence] == nil {
		return &IBCError{"packet not pending"}
	}

	if pr.acknowledgedPackets[sequence] != nil {
		return &IBCError{"packet already acknowledged"}
	}

	if !success {
		return &IBCError{"acknowledgement indicates failure"}
	}

	pr.acknowledgedPackets[sequence] = ackData
	delete(pr.pendingPackets, sequence)
	return nil
}

func (pr *PacketRelay) timeoutPacket(sequence uint64) error {
	if pr.acknowledgedPackets[sequence] != nil {
		return &IBCError{"cannot timeout acknowledged packet"}
	}

	if pr.pendingPackets[sequence] == nil {
		return &IBCError{"packet not pending"}
	}

	pr.timedOutPackets[sequence] = true
	delete(pr.pendingPackets, sequence)
	return nil
}

func validateIBCPacket(packet *IBCPacket) error {
	if packet.Sequence == 0 {
		return &IBCError{"sequence must be positive"}
	}

	if packet.SourcePort == "" || packet.DestPort == "" {
		return &IBCError{"ports cannot be empty"}
	}

	if packet.SourceChannel == "" || packet.DestChannel == "" {
		return &IBCError{"channels cannot be empty"}
	}

	if packet.TimeoutHeight == 0 && packet.TimeoutStamp == 0 {
		return &IBCError{"must specify at least one timeout"}
	}

	if len(packet.Data) == 0 {
		return &IBCError{"packet data cannot be empty"}
	}

	return nil
}

func hashIBCPacket(packet *IBCPacket) []byte {
	hasher := sha256.New()

	if err := binary.Write(hasher, binary.BigEndian, packet.Sequence); err != nil {
		panic(err)
	}
	hasher.Write([]byte(packet.SourcePort))
	hasher.Write([]byte(packet.SourceChannel))
	hasher.Write([]byte(packet.DestPort))
	hasher.Write([]byte(packet.DestChannel))
	hasher.Write(packet.Data)
	if err := binary.Write(hasher, binary.BigEndian, packet.TimeoutHeight); err != nil {
		panic(err)
	}
	if err := binary.Write(hasher, binary.BigEndian, packet.TimeoutStamp); err != nil {
		panic(err)
	}

	return hasher.Sum(nil)
}

func checkIBCTimeout(timeoutHeight, timeoutStamp, currentHeight, currentTime uint64) bool {
	if timeoutHeight > 0 && currentHeight >= timeoutHeight {
		return true
	}

	if timeoutStamp > 0 && currentTime >= timeoutStamp {
		return true
	}

	return false
}

func transitionChannelState(channel *Channel, event HandshakeEvent) (ChannelState, error) {
	switch channel.State {
	case ChannelStateInit:
		if event == EventTryOpen {
			return ChannelStateTryOpen, nil
		}
		return channel.State, &IBCError{"invalid transition from INIT"}

	case ChannelStateTryOpen:
		if event == EventAck || event == EventOpen {
			return ChannelStateOpen, nil
		}
		return channel.State, &IBCError{"invalid transition from TRYOPEN"}

	case ChannelStateOpen:
		if event == EventClose {
			return ChannelStateClosed, nil
		}
		return channel.State, &IBCError{"invalid transition from OPEN"}

	case ChannelStateClosed:
		return channel.State, &IBCError{"no transitions from CLOSED"}

	default:
		return channel.State, &IBCError{"unknown state"}
	}
}

func verifyConnectionProof(connectionID, clientID string, consensusState []byte, proofHeight uint64, proofBytes []byte) bool {
	// Simplified proof verification (real implementation would use Merkle proofs)
	if connectionID == "" || clientID == "" {
		return false
	}

	if len(consensusState) != 32 {
		return false
	}

	if proofHeight == 0 {
		return false
	}

	if len(proofBytes) < 32 {
		return false
	}

	// Compute expected proof hash
	hasher := sha256.New()
	hasher.Write([]byte(connectionID))
	hasher.Write([]byte(clientID))
	hasher.Write(consensusState)
	if err := binary.Write(hasher, binary.BigEndian, proofHeight); err != nil {
		panic(err)
	}

	expectedHash := hasher.Sum(nil)

	// Check if proof contains expected hash
	return bytes.Contains(proofBytes, expectedHash[:16])
}

type IBCError struct {
	msg string
}

func (e *IBCError) Error() string {
	return e.msg
}

// Encoding helpers

func encodeIBCPacket(seq uint64, srcPort, srcChan, dstPort, dstChan string, data []byte, timeoutH, timeoutT uint64) []byte {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, seq); err != nil {
		panic(err)
	}
	buf.WriteString(srcPort)
	buf.WriteByte(0)
	buf.WriteString(srcChan)
	buf.WriteByte(0)
	buf.WriteString(dstPort)
	buf.WriteByte(0)
	buf.WriteString(dstChan)
	buf.WriteByte(0)
	buf.Write(data)
	if err := binary.Write(buf, binary.BigEndian, timeoutH); err != nil {
		panic(err)
	}
	if err := binary.Write(buf, binary.BigEndian, timeoutT); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func parseIBCPacket(data []byte) *IBCPacket {
	if len(data) < 50 {
		return nil
	}

	packet := &IBCPacket{}
	offset := 0

	if offset+8 > len(data) {
		return nil
	}
	packet.Sequence = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse null-terminated strings
	if offset >= len(data) {
		return nil
	}
	packet.SourcePort = readString(data[offset:])
	offset += len(packet.SourcePort) + 1

	if offset >= len(data) {
		return nil
	}
	packet.SourceChannel = readString(data[offset:])
	offset += len(packet.SourceChannel) + 1

	if offset >= len(data) {
		return nil
	}
	packet.DestPort = readString(data[offset:])
	offset += len(packet.DestPort) + 1

	if offset >= len(data) {
		return nil
	}
	packet.DestChannel = readString(data[offset:])
	offset += len(packet.DestChannel) + 1

	// Remaining data up to last 16 bytes is packet data
	if offset+16 > len(data) {
		return nil
	}

	packet.Data = data[offset : len(data)-16]
	offset = len(data) - 16

	packet.TimeoutHeight = binary.BigEndian.Uint64(data[offset : offset+8])
	packet.TimeoutStamp = binary.BigEndian.Uint64(data[offset+8 : offset+16])

	return packet
}

func readString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func encodeTimeout(timeoutH, timeoutT, currentH, currentT uint64) []byte {
	buf := make([]byte, 32)
	binary.BigEndian.PutUint64(buf[0:8], timeoutH)
	binary.BigEndian.PutUint64(buf[8:16], timeoutT)
	binary.BigEndian.PutUint64(buf[16:24], currentH)
	binary.BigEndian.PutUint64(buf[24:32], currentT)
	return buf
}
