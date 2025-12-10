package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestVerifyJobResultThresholdFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Seed three pubkeys but only one signature to trigger threshold error
	pubKeys := [][]byte{
		[]byte{0x01},
		[]byte{0x02},
		[]byte{0x03},
	}
	bz, err := json.Marshal(pubKeys)
	require.NoError(t, err)
	sdkCtx.KVStore(k.storeKey).Set([]byte("validator_keys_chain-thresh"), bz)

	job := &CrossChainComputeJob{TargetChain: "chain-thresh"}
	data := []byte("payload")
	hash := sha256.Sum256(data)
	result := &JobResult{
		ResultData:      data,
		ResultHash:      hex.EncodeToString(hash[:]),
		AttestationSigs: [][]byte{{0xAA}},
	}

	err = k.verifyJobResult(sdkCtx, job, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient attestations")
}

func TestVerifyJobResultInvalidSignature(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Real secp256k1 pubkeys
	pk1 := secp256k1.GenPrivKey()
	pk2 := secp256k1.GenPrivKey()
	pubKeys := [][]byte{pk1.PubKey().Bytes(), pk2.PubKey().Bytes()}

	bz, err := json.Marshal(pubKeys)
	require.NoError(t, err)
	sdkCtx.KVStore(k.storeKey).Set([]byte("validator_keys_chain-invalid-sig"), bz)

	job := &CrossChainComputeJob{TargetChain: "chain-invalid-sig"}
	data := []byte("payload")
	hash := sha256.Sum256(data)
	result := &JobResult{
		ResultData:      data,
		ResultHash:      hex.EncodeToString(hash[:]),
		AttestationSigs: [][]byte{{0x0}, {0x1}}, // malformed signatures
	}

	err = k.verifyJobResult(sdkCtx, job, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid signature")
}

func TestVerifyJobResultHashMismatchCover(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{}
	data := []byte("payload")
	wrongHash := sha256.Sum256([]byte("other"))
	result := &JobResult{ResultData: data, ResultHash: hex.EncodeToString(wrongHash[:])}

	err := k.verifyJobResult(sdkCtx, job, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "result hash mismatch")
}
