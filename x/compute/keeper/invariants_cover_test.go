package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"bytes"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestEscrowBalanceInvariantMismatch(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	req := types.Request{
		Id:             1,
		Status:         types.REQUEST_STATUS_PENDING,
		EscrowedAmount: sdkmath.NewInt(100),
	}
	require.NoError(t, k.SetRequest(sdkCtx, req))

	_, broken := EscrowBalanceInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestProviderStakeInvariantInsufficientStake(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)
	params.MinProviderStake = sdkmath.NewInt(200)
	require.NoError(t, k.SetParams(sdkCtx, params))

	addr := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20))
	provider := types.Provider{
		Address: addr.String(),
		Stake:   sdkmath.NewInt(50),
		Active:  true,
	}
	require.NoError(t, k.SetProvider(sdkCtx, provider))

	_, broken := ProviderStakeInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestRequestStatusInvariantMissingData(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	req := types.Request{
		Id:             2,
		Status:         types.REQUEST_STATUS_PROCESSING,
		EscrowedAmount: sdkmath.ZeroInt(),
		Provider:       "",
	}
	require.NoError(t, k.SetRequest(sdkCtx, req))

	_, broken := RequestStatusInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestNonceUniquenessInvariantDuplicate(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	store := sdkCtx.KVStore(k.storeKey)
	prov := sdk.AccAddress(bytes.Repeat([]byte{0xAB}, 20))
	key := NonceKey(prov, 1)
	store.Set(key, []byte{0x1})
	// Different key, same provider+nonce prefix to simulate corrupted duplicate entry
	store.Set(append(key, 0xFF), []byte{0x2})

	_, broken := NonceUniquenessInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestDisputeIndexInvariantMissingEntry(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Store dispute without indexes to trigger invariant failures
	dispute := types.Dispute{
		Id:        1,
		RequestId: 99,
		Status:    types.DISPUTE_STATUS_VOTING,
	}
	require.NoError(t, k.setDispute(sdkCtx, dispute))
	// Remove expected indexes
	store := sdkCtx.KVStore(k.storeKey)
	store.Delete(DisputeByRequestKey(dispute.RequestId, dispute.Id))
	store.Delete(DisputeByStatusKey(uint32(dispute.Status), dispute.Id))

	_, broken := DisputeIndexInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestAppealIndexInvariantMissingEntry(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	appeal := types.Appeal{
		Id:       2,
		SlashId:  77, // missing slash record
		Status:   types.APPEAL_STATUS_PENDING,
		Provider: "prov",
	}
	require.NoError(t, k.setAppeal(sdkCtx, appeal))
	// Remove the status index to trigger invariant
	store := sdkCtx.KVStore(k.storeKey)
	store.Delete(AppealByStatusKey(uint32(appeal.Status), appeal.Id))

	_, broken := AppealIndexInvariant(*k)(sdkCtx)
	require.True(t, broken)
}

func TestVerifyZKProofShortResultData(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	err := k.VerifyZKProof(sdkCtx, []byte{0x1, 0x2}, []byte{0x1, 0x2})
	require.Error(t, err)
}

func TestVerifyResultZKProofInvalidJSON(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	result := types.Result{
		VerificationProof: []byte("{not-json"),
	}
	ok, err := k.verifyResultZKProof(sdkCtx, result, types.Request{})
	require.False(t, ok)
	require.Error(t, err)
}

func TestVerifyResultZKProofInvalidHash(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	zkProof := types.ZKProof{}
	proofBytes, err := k.cdc.Marshal(&zkProof)
	require.NoError(t, err)

	result := types.Result{
		OutputHash:        "zzzz", // invalid hex
		VerificationProof: proofBytes,
	}
	ok, err := k.verifyResultZKProof(sdkCtx, result, types.Request{})
	require.False(t, ok)
	require.Error(t, err)
}

func TestVerifyJobResultHashMismatch(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{
		TargetChain: AkashChainID,
	}
	result := &JobResult{
		ResultData: []byte("data"),
	}
	hash := sha256.Sum256([]byte("other"))
	result.ResultHash = hex.EncodeToString(hash[:])

	err := k.verifyJobResult(sdkCtx, job, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "result hash mismatch")
}

func TestVerifyJobResultAttestationsMissingKeys(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	data := []byte("result")
	hash := sha256.Sum256(data)
	job := &CrossChainComputeJob{
		TargetChain: "chain-without-keys",
	}
	result := &JobResult{
		ResultData:      data,
		ResultHash:      hex.EncodeToString(hash[:]),
		AttestationSigs: [][]byte{{0x1, 0x2}},
	}

	err := k.verifyJobResult(sdkCtx, job, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validator public keys")
}
