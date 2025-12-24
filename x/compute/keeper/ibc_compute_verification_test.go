package keeper

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGroth16ProofDeserializeErrors(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		var proof Groth16ProofBN254
		err := proof.Deserialize([]byte{0x01, 0x02})
		require.Error(t, err)
	})

	t.Run("insufficient B component", func(t *testing.T) {
		var proof Groth16ProofBN254
		data := make([]byte, 150) // enough for A, not for full B
		err := proof.Deserialize(data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient data for B component")
	})
}

func TestVerifyIBCZKProofInputValidation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	err := k.verifyIBCZKProof(sdkCtx, nil, []byte("inputs"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty ZK proof")

	err = k.verifyIBCZKProof(sdkCtx, []byte{0x1}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty public inputs")

	// malformed proof bytes should propagate deserialization error
	err = k.verifyIBCZKProof(sdkCtx, []byte{0xFF, 0xEE}, []byte("inputs"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to deserialize proof")
}

func TestVerifyGroth16PairingFailsWithoutVerifyingKey(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Mark circuits initialized but don't add result circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = make(map[string]*circuitKeys) // Empty state - no keys
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	k.circuitManager = NewCircuitManager(k)

	proof := &Groth16ProofBN254{}
	err := k.verifyGroth16Pairing(sdkCtx, []byte{0x1, 0x2}, proof, bn254.G1Affine{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "verifying key unavailable")
}

func TestVerifyGroth16PairingDeserializationError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup circuit state with a stub verifying key for result circuit
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = map[string]*circuitKeys{
		resultCircuitDef.id: {
			vk: groth16.NewVerifyingKey(ecc.BN254),
		},
	}
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	k.circuitManager = NewCircuitManager(k)

	proof := &Groth16ProofBN254{}
	err := k.verifyGroth16Pairing(sdkCtx, []byte{0x01}, proof, bn254.G1Affine{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to deserialize proof")
}

func TestVerifyIBCZKProofDeserializeError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	err := k.verifyIBCZKProof(sdkCtx, []byte{0x01}, []byte{0x02})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to deserialize proof")
}

func TestVerifyAttestationsThresholdsAndLengths(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	message := sha256.Sum256([]byte("attestation-msg"))

	// Build validator keys and signatures
	privKeys := []*secp256k1.PrivKey{
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
	}

	publicKeys := make([][]byte, len(privKeys))
	attestations := make([][]byte, len(privKeys))
	for i, pk := range privKeys {
		publicKeys[i] = pk.PubKey().Bytes()
		sig, err := pk.Sign(message[:])
		require.NoError(t, err)
		attestations[i] = sig
	}

	require.NoError(t, k.verifyAttestations(sdkCtx, attestations, publicKeys, message[:]))

	// Insufficient attestations (only one provided)
	err := k.verifyAttestations(sdkCtx, attestations[:1], publicKeys, message[:])
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient attestations")

	// Invalid message length
	err = k.verifyAttestations(sdkCtx, attestations, publicKeys, []byte{0x01, 0x02})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid message length")

	// Invalid public key length
	badKeys := [][]byte{{0x01, 0x02}}
	err = k.verifyAttestations(sdkCtx, attestations, badKeys, message[:])
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient valid signatures")

	// Signature verification failure
	badSigs := [][]byte{attestations[0], {0x01}} // second signature malformed
	err = k.verifyAttestations(sdkCtx, badSigs, publicKeys, message[:])
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid signature")
}

func TestGetValidatorPublicKeysReturnsErrorWhenMissing(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := k.getValidatorPublicKeys(sdkCtx, "unknown-chain")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no validator public keys available")
}

func TestGroth16ProofDeserializeRoundTrip(t *testing.T) {
	// Construct a minimal but valid proof encoding to hit the success path
	var proof Groth16ProofBN254

	var a bn254.G1Affine
	a.ScalarMultiplicationBase(big.NewInt(1))
	aBytes := a.Marshal()

	var b bn254.G2Affine
	b.ScalarMultiplicationBase(big.NewInt(1))
	bBytes := b.Marshal()

	var c bn254.G1Affine
	c.ScalarMultiplicationBase(big.NewInt(2))
	cBytes := c.Marshal()

	buf := bytes.NewBuffer(nil)
	buf.Write(aBytes)
	buf.Write(bBytes)
	buf.Write(cBytes)

	require.NoError(t, proof.Deserialize(buf.Bytes()))
	require.NoError(t, proof.Validate())
}
