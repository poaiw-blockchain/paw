package keeper

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestVerifyProofSucceedsAndRefundsDeposit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	verifier := NewZKVerifier(k)

	originalVerify := Groth16VerifyFunc()
	SetGroth16Verify(func(groth16.Proof, groth16.VerifyingKey, witness.Witness, ...backend.VerifierOption) error {
		return nil
	})
	t.Cleanup(func() {
		SetGroth16Verify(originalVerify)
	})

	vk := groth16.NewVerifyingKey(ecc.BN254)
	var vkBuf bytes.Buffer
	_, err := vk.WriteTo(&vkBuf)
	require.NoError(t, err)

	params := k.getDefaultCircuitParams(sdkCtx, "compute-verification-v1")
	params.VerifyingKey.VkData = vkBuf.Bytes()
	params.VerificationDepositAmount = 1_000 // small deposit for test
	require.NoError(t, k.SetCircuitParams(sdkCtx, *params))

	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	resultHash := sha256.Sum256([]byte("result"))
	providerHash := sha256.Sum256(provider.Bytes())
	publicInputs := serializePublicInputs(1, resultHash[:], providerHash[:])

	proofObj := groth16.NewProof(ecc.BN254)
	var proofBuf bytes.Buffer
	_, err = proofObj.WriteTo(&proofBuf)
	require.NoError(t, err)

	zkProof := &types.ZKProof{
		Proof:        proofBuf.Bytes(),
		ProofSystem:  "groth16",
		CircuitId:    params.CircuitId,
		PublicInputs: publicInputs,
	}

	balanceBefore := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")

	valid, err := verifier.VerifyProof(sdkCtx, zkProof, 1, resultHash[:], provider)
	require.NoError(t, err)
	require.True(t, valid)

	balanceAfter := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")
	require.Equal(t, balanceBefore, balanceAfter, "deposit should be refunded on valid proof")
}
