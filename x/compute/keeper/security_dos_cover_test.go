package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestZKProofSizeGuardForIBCPacket(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	huge := make([]byte, 2*1024*1024) // exceeds default maxProofSize 1MB fallback
	err := k.verifyIBCZKProof(sdkCtx, huge, []byte{0x1})
	require.Error(t, err)
}

func TestVerifyProofWithCacheSizeGuard(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proof := &types.ZKProof{Proof: make([]byte, 2*1024*1024), PublicInputs: []byte{0x1}}
	verified, err := k.VerifyProofWithCache(sdkCtx, proof, 1)
	require.False(t, verified)
	require.Error(t, err)
}
