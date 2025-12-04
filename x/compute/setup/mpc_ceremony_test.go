package setup

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
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
	binary.Write(h, binary.BigEndian, round)
	return h.Sum(nil), nil
}

func (deterministicBeacon) VerifyRandomness(round uint64, randomness []byte) bool {
	expected, _ := deterministicBeacon{}.GetRandomness(round)
	return bytes.Equal(expected, randomness)
}

func randomTestBytes(size int) []byte {
	buf := make([]byte, size)
	rand.Read(buf)
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
