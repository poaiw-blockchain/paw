package keeper

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestVerifyZKProofPositivePathWithStubs(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Now())

	originalVerify := Groth16VerifyFunc()
	SetGroth16Verify(func(groth16.Proof, groth16.VerifyingKey, witness.Witness, ...backend.VerifierOption) error {
		return nil
	})
	t.Cleanup(func() { SetGroth16Verify(originalVerify) })

	vk := groth16.NewVerifyingKey(ecc.BN254)
	var vkBuf bytes.Buffer
	_, _ = vk.WriteTo(&vkBuf)
	params := k.getDefaultCircuitParams(sdkCtx, "compute-verification-v1")
	params.VerifyingKey.VkData = vkBuf.Bytes()
	params.VerificationDepositAmount = 0
	require.NoError(t, k.SetCircuitParams(sdkCtx, *params))

	proof := groth16.NewProof(ecc.BN254)
	var proofBuf bytes.Buffer
	_, _ = proof.WriteTo(&proofBuf)

	// Build result data: requestID (8) + resultHash(32) + provider(20)
	resultData := make([]byte, 60)
	binary.BigEndian.PutUint64(resultData[:8], 1)
	copy(resultData[8:40], bytes.Repeat([]byte{0x01}, 32))
	copy(resultData[40:60], []byte("prov_addr_1234567890"))

	require.NoError(t, k.VerifyZKProof(sdkCtx, proofBuf.Bytes(), resultData))
}
