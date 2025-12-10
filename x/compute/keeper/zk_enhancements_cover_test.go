package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestGetClientIDPrefersMetadataThenPeer(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-client-id", "meta-id"))
	require.Equal(t, "meta-id", getClientID(ctx))

	pctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}})
	require.Contains(t, getClientID(pctx), "127.0.0.1")
}

func TestVerifyProofWithCacheUsesCachedEntry(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proof := &types.ZKProof{
		CircuitId: "cached-circuit",
		Proof:     []byte{0x1, 0x2},
	}
	proofHash := k.calculateProofHash(proof)
	require.NoError(t, k.CacheProofVerification(sdkCtx, proofHash, true, 10, 77))

	verified, err := k.VerifyProofWithCache(sdkCtx, proof, 77)
	require.NoError(t, err)
	require.True(t, verified)
}

func TestVerifyProofWithCachePerformsVerification(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	originalSetup := Groth16SetupFunc()
	originalVerify := Groth16VerifyFunc()
	SetGroth16Setup(func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
		return groth16.NewProvingKey(ecc.BN254), groth16.NewVerifyingKey(ecc.BN254), nil
	})
	verifyCalled := false
	SetGroth16Verify(func(groth16.Proof, groth16.VerifyingKey, witness.Witness, ...backend.VerifierOption) error {
		verifyCalled = true
		return fmt.Errorf("forced verification failure")
	})
	defer func() {
		SetGroth16Setup(originalSetup)
		SetGroth16Verify(originalVerify)
	}()

	vk := groth16.NewVerifyingKey(ecc.BN254)
	var vkBuf bytes.Buffer
	_, _ = vk.WriteTo(&vkBuf)

	params := k.getDefaultCircuitParams(sdkCtx, "compute-verification-v1")
	params.VerifyingKey.VkData = vkBuf.Bytes()
	params.VerificationDepositAmount = 0
	require.NoError(t, k.SetCircuitParams(sdkCtx, *params))

	// Build a minimal valid proof encoding
	var proofBuf bytes.Buffer
	emptyProof := groth16.NewProof(ecc.BN254)
	_, _ = emptyProof.WriteTo(&proofBuf)

	resultData := make([]byte, 60)
	binary.BigEndian.PutUint64(resultData, 123)
	copy(resultData[8:40], bytes.Repeat([]byte{0x01}, 32))
	copy(resultData[40:60], []byte("test_provider_addr_1"))

	proof := &types.ZKProof{
		CircuitId:    "compute-verification-v1",
		Proof:        proofBuf.Bytes(),
		PublicInputs: resultData,
		ProofSystem:  "groth16",
	}

	verified, err := k.VerifyProofWithCache(sdkCtx, proof, 123)
	require.True(t, verifyCalled, "verification should be invoked when cache miss")
	require.Error(t, err)
	require.False(t, verified)
	require.Contains(t, err.Error(), "forced verification failure")
}

func TestVerifyTrustedSetupSuccess(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	setup := &TrustedSetup{
		CircuitID:         "circuit-id",
		Contributors:      []string{"a", "b", "c"},
		ContributionCount: 3,
		Finalized:         true,
	}
	setup.SetupHash = k.calculateSetupHash(setup)

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey)
	data := map[string]interface{}{
		"setup_hash":         setup.SetupHash,
		"finalized":          true,
		"contribution_count": float64(setup.ContributionCount),
		"contributors":       setup.Contributors,
	}
	bz, _ := json.Marshal(data)
	store.Set([]byte("trusted_setup_"+setup.CircuitID), bz)

	require.NoError(t, k.VerifyTrustedSetup(ctx, setup.CircuitID))
}
