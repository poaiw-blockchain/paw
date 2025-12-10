package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestCalculateSetupHashDeterministic(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	setup := &TrustedSetup{
		CircuitID:         "circuit-A",
		SetupHash:         "",
		Contributors:      []string{"a", "b", "c"},
		ContributionCount: 3,
	}

	hash1 := k.calculateSetupHash(setup)
	hash2 := k.calculateSetupHash(setup)
	require.Equal(t, hash1, hash2)
}

func TestVerifyProofBatchUsesCache(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proof1 := &types.ZKProof{CircuitId: "circuit-1", Proof: []byte{0x1}, PublicInputs: []byte{0x2}}
	proof2 := &types.ZKProof{CircuitId: "circuit-2", Proof: []byte{0x3}, PublicInputs: []byte{0x4}}

	hash1 := k.calculateProofHash(proof1)
	hash2 := k.calculateProofHash(proof2)

	require.NoError(t, k.CacheProofVerification(sdkCtx, hash1, true, 10, 101))
	require.NoError(t, k.CacheProofVerification(sdkCtx, hash2, false, 20, 202))

	batch := &BatchProofVerification{
		Proofs:      []*types.ZKProof{proof1, proof2},
		RequestIDs:  []uint64{101, 202},
		BatchID:     "batch-1",
		SubmittedAt: time.Now(),
	}

	results, err := k.VerifyProofBatch(sdkCtx, batch)
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, results)
}

func TestVerifyProofBatchValidatesInputLengths(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := k.VerifyProofBatch(sdkCtx, &BatchProofVerification{
		Proofs:     []*types.ZKProof{},
		RequestIDs: []uint64{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty proof batch")

	_, err = k.VerifyProofBatch(sdkCtx, &BatchProofVerification{
		Proofs:     []*types.ZKProof{{}},
		RequestIDs: []uint64{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "proof count mismatch")
}

func TestRotateKeysEmitsEvent(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	require.NoError(t, k.rotateKeys(sdkCtx, "circuit-rotate"))
	events := sdkCtx.EventManager().Events()

	require.NotEmpty(t, events)
	found := false
	for _, ev := range events {
		if ev.Type == "keys_rotated" {
			found = true
			break
		}
	}
	require.True(t, found)
}
