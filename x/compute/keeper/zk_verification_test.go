package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/circuits"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
)

func TestZKProofRejectsInvalidProof(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	zkVerifier := computekeeper.NewZKVerifier(k)

	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	require.NoError(t, zkVerifier.InitializeCircuit(ctx, circuitID))

	providerKey := secp256k1.GenPrivKey()
	provider := sdk.AccAddress(providerKey.PubKey().Address())

	requestID := uint64(42)
	computationData := []byte("test computation")
	timestamp := time.Now().Unix()

	resultHash, err := computekeeper.ComputeResultHash(
		requestID,
		provider,
		computationData,
		timestamp,
		0,
		1000,
		2048,
	)
	require.NoError(t, err)

	proof, err := zkVerifier.GenerateProof(
		ctx,
		requestID,
		resultHash,
		provider,
		computationData,
		timestamp,
		0,
		1000,
		2048,
	)
	require.NoError(t, err)
	require.NotNil(t, proof)

	require.NotEmpty(t, proof.Proof, "proof bytes should be present")
	proof.Proof[0] ^= 0xFF

	valid, err := zkVerifier.VerifyProof(ctx, proof, requestID, resultHash, provider)
	require.Error(t, err)
	require.False(t, valid)
}

func TestZKProofIBCPacketRejectsInvalidProof(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	require.NoError(t, k.InitializeCircuits(ctx))

	err := k.VerifyIBCZKProofForTest(ctx, []byte("invalid-proof-data"), []byte("bad-input"))
	require.Error(t, err)
}
