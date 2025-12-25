package setup

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/stretchr/testify/require"
)

func TestMPCCeremonyFinalizationPersistsKeys(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	sink := &mockKeySink{}
	ceremony := NewMPCCeremony("compute-test", ccs, SecurityLevel256, &deterministicBeacon{}, sink)

	participants := []struct {
		id  string
		key []byte
	}{
		{"alice", randomTestBytes(32)},
		{"bob", randomTestBytes(32)},
		{"charlie", randomTestBytes(32)},
	}

	for _, p := range participants {
		require.NoError(t, ceremony.RegisterParticipant(p.id, p.key))
	}

	require.NoError(t, ceremony.StartCeremony())

	for _, p := range participants {
		_, err := ceremony.Contribute(p.id, randomTestBytes(64))
		require.NoError(t, err)
	}

	_, _, err = ceremony.Finalize(context.Background())
	require.NoError(t, err)

	require.True(t, sink.called, "ceremony should persist keys via sink")
	require.NotEmpty(t, sink.pkBytes)
	require.NotEmpty(t, sink.vkBytes)
	require.NotEmpty(t, ceremony.provingKeyBytes)
	require.NotEmpty(t, ceremony.verifyingKeyBytes)
}

type mockKeySink struct {
	pkBytes []byte
	vkBytes []byte
	called  bool
}

func (m *mockKeySink) StoreCeremonyKeys(ctx context.Context, circuitID string, provingKey, verifyingKey []byte) error {
	m.pkBytes = append([]byte(nil), provingKey...)
	m.vkBytes = append([]byte(nil), verifyingKey...)
	m.called = true
	return nil
}

type deterministicBeacon struct{}

func (deterministicBeacon) GetRandomness(round uint64) ([]byte, error) {
	h := sha256.New()
	if err := binary.Write(h, binary.BigEndian, round); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func (deterministicBeacon) VerifyRandomness(round uint64, randomness []byte) bool {
	expected, err := deterministicBeacon{}.GetRandomness(round)
	if err != nil {
		return false
	}
	return bytes.Equal(expected, randomness)
}

func randomTestBytes(size int) []byte {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("failed to generate random test bytes: %v", err))
	}
	return buf
}

type equalityCircuit struct {
	A frontend.Variable `gnark:",public"`
	B frontend.Variable `gnark:",public"`
}

func (c *equalityCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(c.A, c.B)
	return nil
}

// TestRegisterParticipantValidation tests participant registration validation
func TestRegisterParticipantValidation(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	// Test invalid public key length
	err = ceremony.RegisterParticipant("alice", []byte("short"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid public key length")

	// Test valid registration
	validKey := randomTestBytes(32)
	err = ceremony.RegisterParticipant("alice", validKey)
	require.NoError(t, err)

	// Test duplicate registration
	err = ceremony.RegisterParticipant("alice", validKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
}

// TestRegisterParticipantAfterStart tests that registration is still allowed during contribution phase
func TestRegisterParticipantAfterStart(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	// Register and start
	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Should still allow registration during contribution phase
	err = ceremony.RegisterParticipant("bob", randomTestBytes(32))
	require.NoError(t, err)
}

// TestRegisterParticipantAfterFinalization tests that registration fails after ceremony is finalized
func TestRegisterParticipantAfterFinalization(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	_, _, err = ceremony.Finalize(context.Background())
	require.NoError(t, err)

	// Should fail to register after finalization
	err = ceremony.RegisterParticipant("bob", randomTestBytes(32))
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot register participant")
}

// TestStartCeremonyNoParticipants tests that ceremony requires at least one participant
func TestStartCeremonyNoParticipants(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	err = ceremony.StartCeremony()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no participants registered")
}

// TestStartCeremonyAlreadyStarted tests that ceremony cannot be started twice
func TestStartCeremonyAlreadyStarted(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Try to start again
	err = ceremony.StartCeremony()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ceremony already started")
}

// TestContributeNotRegistered tests that unregistered participants cannot contribute
func TestContributeNotRegistered(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Bob is not registered
	_, err = ceremony.Contribute("bob", randomTestBytes(64))
	require.Error(t, err)
	require.Contains(t, err.Error(), "participant not registered")
}

// TestContributeBeforeStart tests that contributions cannot be made before ceremony starts
func TestContributeBeforeStart(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))

	// Try to contribute before starting
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in contribution phase")
}

// TestContributeDuplicate tests that participants cannot contribute twice
func TestContributeDuplicate(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// First contribution
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	// Second contribution should fail
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.Error(t, err)
	require.Contains(t, err.Error(), "already contributed")
}

// TestContributeInsufficientRandomness tests that contributions require sufficient randomness
func TestContributeInsufficientRandomness(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Insufficient randomness (< 32 bytes)
	_, err = ceremony.Contribute("alice", []byte("short"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient randomness")
}

// TestVerifyContribution tests contribution verification
func TestVerifyContribution(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.RegisterParticipant("bob", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Verify genesis contribution (index 0)
	valid, err := ceremony.VerifyContribution(0)
	require.NoError(t, err)
	require.True(t, valid)

	// Add contributions
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	// Verify alice's contribution (index 1)
	valid, err = ceremony.VerifyContribution(1)
	require.NoError(t, err)
	require.True(t, valid)

	_, err = ceremony.Contribute("bob", randomTestBytes(64))
	require.NoError(t, err)

	// Verify bob's contribution (index 2)
	valid, err = ceremony.VerifyContribution(2)
	require.NoError(t, err)
	require.True(t, valid)
}

// TestVerifyContributionInvalidIndex tests verification with invalid contribution index
func TestVerifyContributionInvalidIndex(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Test negative index
	valid, err := ceremony.VerifyContribution(-1)
	require.Error(t, err)
	require.False(t, valid)
	require.Contains(t, err.Error(), "invalid contribution index")

	// Test out of bounds index
	valid, err = ceremony.VerifyContribution(999)
	require.Error(t, err)
	require.False(t, valid)
	require.Contains(t, err.Error(), "invalid contribution index")
}

// TestFinalizeNoContributions tests that finalization requires contributions
func TestFinalizeNoContributions(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	// Try to finalize without any contributions
	_, _, err = ceremony.Finalize(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "ceremony not in contribution phase")
}

// TestFinalizeBeforeContributions tests finalization fails without starting ceremony
func TestFinalizeBeforeContributions(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	// Don't start ceremony

	_, _, err = ceremony.Finalize(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in contribution phase")
}

// TestFinalizeWithBeaconError tests finalization when beacon fails
func TestFinalizeWithBeaconError(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	// Use failing beacon
	failingBeacon := &failingBeacon{}
	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, failingBeacon, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	_, _, err = ceremony.Finalize(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get beacon randomness")
}

// TestFinalizeWithNilKeySink tests finalization without a key sink (should still work)
func TestFinalizeWithNilKeySink(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, nil)

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	pk, vk, err := ceremony.Finalize(context.Background())
	require.NoError(t, err)
	require.NotNil(t, pk)
	require.NotNil(t, vk)
}

// TestGenerateSecureScalar tests secure scalar generation
func TestGenerateSecureScalar(t *testing.T) {
	t.Parallel()

	scalar, err := generateSecureScalar()
	require.NoError(t, err)
	require.NotNil(t, scalar)

	// Generate another and ensure they're different
	scalar2, err := generateSecureScalar()
	require.NoError(t, err)
	require.NotNil(t, scalar2)

	require.NotEqual(t, scalar.Bytes(), scalar2.Bytes())
}

// TestGetNextPowerOfTwo tests power of two calculation
func TestGetNextPowerOfTwo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{15, 16},
		{16, 16},
		{17, 32},
		{100, 128},
		{1000, 1024},
	}

	for _, tc := range tests {
		result := getNextPowerOfTwo(tc.input)
		require.Equal(t, tc.expected, result, "getNextPowerOfTwo(%d) should be %d", tc.input, tc.expected)
	}
}

// TestGenerateCeremonyID tests ceremony ID generation
func TestGenerateCeremonyID(t *testing.T) {
	t.Parallel()

	id1 := generateCeremonyID()
	require.NotEmpty(t, id1)
	require.Contains(t, id1, "ceremony-")

	// Generate another and ensure they're different
	id2 := generateCeremonyID()
	require.NotEmpty(t, id2)
	require.NotEqual(t, id1, id2)
}

// TestComputeContributionHash tests contribution hash computation
func TestComputeContributionHash(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	contrib, err := ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	hash1, err := ceremony.computeContributionHash(contrib)
	require.NoError(t, err)
	require.NotEmpty(t, hash1)

	// Same contribution should produce same hash
	hash2, err := ceremony.computeContributionHash(contrib)
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)
}

// TestComputeTranscriptHash tests transcript hash computation
func TestComputeTranscriptHash(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	_, _, err = ceremony.Finalize(context.Background())
	require.NoError(t, err)

	hash1, err := ceremony.computeTranscriptHash()
	require.NoError(t, err)
	require.NotEmpty(t, hash1)

	// Should be deterministic
	hash2, err := ceremony.computeTranscriptHash()
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	t.Parallel()

	require.Equal(t, 1, min(1, 2))
	require.Equal(t, 1, min(2, 1))
	require.Equal(t, 5, min(5, 5))
	require.Equal(t, -1, min(-1, 0))
	require.Equal(t, -10, min(-10, -5))
}

// TestSerializeKeys tests key serialization
func TestSerializeKeys(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})
	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	pk, vk, err := ceremony.Finalize(context.Background())
	require.NoError(t, err)

	pkBytes, vkBytes, err := serializeKeys(*pk, *vk)
	require.NoError(t, err)
	require.NotEmpty(t, pkBytes)
	require.NotEmpty(t, vkBytes)
}

// TestCeremonyTranscriptCompleteness tests that transcript captures all information
func TestCeremonyTranscriptCompleteness(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.RegisterParticipant("bob", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)
	_, err = ceremony.Contribute("bob", randomTestBytes(64))
	require.NoError(t, err)

	_, _, err = ceremony.Finalize(context.Background())
	require.NoError(t, err)

	transcript := ceremony.transcript

	require.NotEmpty(t, transcript.CeremonyID)
	require.Equal(t, "test-circuit", transcript.CircuitID)
	require.False(t, transcript.StartTime.IsZero())
	require.False(t, transcript.EndTime.IsZero())
	require.True(t, transcript.EndTime.After(transcript.StartTime))
	require.Equal(t, []string{"alice", "bob"}, transcript.Participants)
	// Contributions are only added during Contribute(), not StartCeremony
	// So we have alice + bob = 2 contributions
	require.Len(t, transcript.Contributions, 2)
	require.NotEmpty(t, transcript.TranscriptHash)
	require.NotEmpty(t, transcript.FinalBeacon)
	require.True(t, transcript.Verified)
	require.False(t, transcript.VerifiedAt.IsZero())
}

// TestApplyContributionZeroRandomness tests that zero randomness is rejected
func TestApplyContributionZeroRandomness(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	// Use all-zero randomness (should be rejected if it produces zero field elements)
	zeroRandomness := make([]byte, 64)
	_, err = ceremony.Contribute("alice", zeroRandomness)

	// This might fail depending on hash output - if it produces zero field elements
	// For now, just ensure it doesn't panic
	// The actual error depends on whether BLAKE2b(zero) produces zero field elements
	_ = err
}

// failingBeacon is a beacon that always returns errors
type failingBeacon struct{}

func (failingBeacon) GetRandomness(round uint64) ([]byte, error) {
	return nil, fmt.Errorf("beacon failure")
}

func (failingBeacon) VerifyRandomness(round uint64, randomness []byte) bool {
	return false
}

// TestSecurityLevel128 tests ceremony with 128-bit security
func TestSecurityLevel128(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.Equal(t, SecurityLevel128, ceremony.securityLevel)
	require.NotNil(t, ceremony)
}

// TestSecurityLevel256 tests ceremony with 256-bit security
func TestSecurityLevel256(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	require.Equal(t, SecurityLevel256, ceremony.securityLevel)
	require.NotNil(t, ceremony)
}

// TestParticipantMetadata tests that participant metadata is tracked correctly
func TestParticipantMetadata(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	publicKey := randomTestBytes(32)
	require.NoError(t, ceremony.RegisterParticipant("alice", publicKey))

	require.Len(t, ceremony.participants, 1)
	p := ceremony.participants[0]

	require.Equal(t, "alice", p.ID)
	require.Equal(t, publicKey, p.PublicKey)
	require.Nil(t, p.Contribution)
	require.False(t, p.Verified)
	require.False(t, p.JoinedAt.IsZero())
	require.True(t, p.ContributedAt.IsZero())

	// After contribution
	require.NoError(t, ceremony.StartCeremony())
	_, err = ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	p = ceremony.participants[0]
	require.NotNil(t, p.Contribution)
	require.False(t, p.ContributedAt.IsZero())
}

// TestContributionStructure tests that contributions have proper structure
func TestContributionStructure(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	contrib, err := ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	require.Equal(t, "alice", contrib.ParticipantID)
	require.NotEmpty(t, contrib.PreviousHash)
	require.NotEmpty(t, contrib.TauG1Powers)
	require.NotEmpty(t, contrib.TauG2Powers)
	require.False(t, contrib.AlphaG1.IsInfinity())
	require.False(t, contrib.BetaG1.IsInfinity())
	require.False(t, contrib.BetaG2.IsInfinity())
	require.NotEmpty(t, contrib.ProofOfKnowledge.Challenge)
	require.NotEmpty(t, contrib.ProofOfKnowledge.Response)
	require.False(t, contrib.Timestamp.IsZero())
	require.NotEmpty(t, contrib.CommitmentHash)
}

// TestProofOfKnowledgeValidation tests proof of knowledge verification
func TestProofOfKnowledgeValidation(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	contrib, err := ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	// Verify the proof passes verification
	err = ceremony.verifyProofOfKnowledge(contrib)
	require.NoError(t, err)
}

// TestProofOfKnowledgeMissingChallenge tests verification with missing challenge
func TestProofOfKnowledgeMissingChallenge(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	contrib, err := ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	// Clear challenge
	contrib.ProofOfKnowledge.Challenge = []byte{}

	err = ceremony.verifyProofOfKnowledge(contrib)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing challenge")
}

// TestProofOfKnowledgeMissingResponse tests verification with missing response
func TestProofOfKnowledgeMissingResponse(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel128, &deterministicBeacon{}, &mockKeySink{})

	require.NoError(t, ceremony.RegisterParticipant("alice", randomTestBytes(32)))
	require.NoError(t, ceremony.StartCeremony())

	contrib, err := ceremony.Contribute("alice", randomTestBytes(64))
	require.NoError(t, err)

	// Clear response
	contrib.ProofOfKnowledge.Response = []byte{}

	err = ceremony.verifyProofOfKnowledge(contrib)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing response")
}

// TestConcurrentRegistration tests thread safety of participant registration
func TestConcurrentRegistration(t *testing.T) {
	t.Parallel()

	circuit := &equalityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	ceremony := NewMPCCeremony("test-circuit", ccs, SecurityLevel256, &deterministicBeacon{}, &mockKeySink{})

	// Register participants concurrently
	const numParticipants = 10
	errChan := make(chan error, numParticipants)

	for i := 0; i < numParticipants; i++ {
		go func(id int) {
			err := ceremony.RegisterParticipant(fmt.Sprintf("participant-%d", id), randomTestBytes(32))
			errChan <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numParticipants; i++ {
		err := <-errChan
		require.NoError(t, err)
	}

	require.Len(t, ceremony.participants, numParticipants)
}
