package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app/ibcutil"
	"github.com/paw-chain/paw/x/compute/types"
)

// =============================================================================
// Acknowledgement Handlers Tests (from ibc_ack_handlers_test.go)
// =============================================================================

func TestHandleDiscoverProvidersAckStoresProviders(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// mark pending discovery
	k.storePendingDiscovery(sdkCtx, "channel-0", 1, "chain-X")

	packet := channeltypes.Packet{
		SourceChannel: "channel-0",
		Sequence:      1,
	}
	ack := types.DiscoverProvidersAcknowledgement{
		Success: true,
		Providers: []types.ProviderInfo{
			{
				ProviderID:   "p1",
				Address:      "addr1",
				Capabilities: []string{"gpu"},
				PricePerUnit: math.LegacyNewDec(1),
				Reputation:   math.LegacyNewDec(9),
			},
		},
	}

	err := k.handleDiscoverProvidersAck(sdkCtx, packet, ack)
	require.NoError(t, err)

	// provider cached
	cached := k.getCachedProviders(sdkCtx, []string{"gpu"}, math.LegacyNewDec(10))
	require.Len(t, cached, 1)
	require.Equal(t, "p1", cached[0].ProviderID)
}

func TestHandleSubmitJobAckUpdatesJob(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{
		JobID:       "job-ack-1",
		TargetChain: "chain-A",
		Status:      "pending",
		Progress:    0,
		SubmittedAt: time.Now(),
	}
	k.storeJob(sdkCtx, job.JobID, job)
	k.storePendingJobSubmission(sdkCtx, "channel-0", 5, job.JobID)

	packet := channeltypes.Packet{
		SourceChannel: "channel-0",
		Sequence:      5,
	}
	ack := types.SubmitJobAcknowledgement{
		Success:  true,
		JobID:    job.JobID,
		Status:   "submitted",
		Progress: 30,
	}

	err := k.handleSubmitJobAck(sdkCtx, packet, types.SubmitJobPacketData{JobID: job.JobID}, ack)
	require.NoError(t, err)

	updated := k.getJob(sdkCtx, job.JobID)
	require.NotNil(t, updated)
	require.Equal(t, "submitted", updated.Status)
	require.Equal(t, progressForStatus("submitted", 30), updated.Progress)
	require.Equal(t, "", k.getPendingJobSubmission(sdkCtx, packet.Sequence))
}

func TestHandleJobStatusAckUpdatesJob(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{
		JobID:    "job-status-1",
		Status:   "running",
		Progress: 70,
	}
	k.storeJob(sdkCtx, job.JobID, job)

	ack := types.JobStatusAcknowledgement{
		Success:  true,
		JobID:    job.JobID,
		Status:   "completed",
		Progress: 100,
	}

	err := k.handleJobStatusAck(sdkCtx, ack)
	require.NoError(t, err)

	updated := k.getJob(sdkCtx, job.JobID)
	require.NotNil(t, updated)
	require.Equal(t, "completed", updated.Status)
	require.Equal(t, uint32(100), updated.Progress)
}

func TestOnAcknowledgementPacketSubmitJobErrorUpdatesState(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{
		JobID:    "job-err-1",
		Status:   "pending",
		Progress: 0,
	}
	k.storeJob(sdkCtx, job.JobID, job)
	k.storePendingJobSubmission(sdkCtx, "channel-0", 7, job.JobID)

	packetData := types.SubmitJobPacketData{
		Type:    types.SubmitJobType,
		JobID:   job.JobID,
		JobType: "docker",
	}
	dataBz, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.Packet{
		Data:          dataBz,
		SourceChannel: "channel-0",
		Sequence:      7,
	}

	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("remote failure"))

	err = k.OnAcknowledgementPacket(sdkCtx, packet, ack)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid request")

	updated := k.getJob(sdkCtx, job.JobID)
	require.NotNil(t, updated)
	require.Equal(t, "failed", updated.Status)
	require.Equal(t, "", k.getPendingJobSubmission(sdkCtx, packet.Sequence))
}

func TestOnAcknowledgementPacketDecodeError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	packet := channeltypes.Packet{
		Data:          []byte("{invalid-json"),
		SourceChannel: "channel-X",
		Sequence:      1,
	}

	ack := channeltypes.NewResultAcknowledgement([]byte(`{"ok":true}`))

	err := k.OnAcknowledgementPacket(sdkCtx, packet, ack)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode packet data")
}

func TestOnAcknowledgementPacketJobResultSuccess(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	jobID := "job-success-1"
	packetData := types.JobResultPacketData{
		Type:      types.JobResultType,
		Nonce:     1,
		Timestamp: time.Now().Unix(),
		JobID:     jobID,
		Provider:  "prov-1",
		Result: types.JobResult{
			ResultData: []byte{0x1, 0x2},
			ResultHash: "hash123",
		},
	}
	dataBz, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.Packet{
		Data:               dataBz,
		SourceChannel:      "channel-0",
		DestinationChannel: "channel-1",
		Sequence:           9,
	}

	ackPayload, err := json.Marshal(types.JobResultAcknowledgement{
		Nonce:      1,
		Success:    true,
		JobID:      jobID,
		Status:     "completed",
		Progress:   80,
		Provider:   "prov-1",
		ResultHash: "hash123",
		ProofHash:  "proof-hash",
	})
	require.NoError(t, err)
	ack := channeltypes.NewResultAcknowledgement(ackPayload)

	require.NoError(t, k.OnAcknowledgementPacket(sdkCtx, packet, ack))

	job := k.getJob(sdkCtx, jobID)
	require.NotNil(t, job)
	require.Equal(t, "completed", job.Status)
	require.Equal(t, uint32(100), job.Progress)
	require.Equal(t, "hash123", job.Result.ResultHash)
	require.Equal(t, "proof-hash", job.ProofHash)
	require.True(t, job.Verified)
}

func TestOnAcknowledgementPacketJobResultError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	jobID := "job-error-1"
	job := &CrossChainComputeJob{JobID: jobID, Status: "running", Progress: 50}
	k.storeJob(sdkCtx, jobID, job)

	packetData := types.JobResultPacketData{
		Type:      types.JobResultType,
		Nonce:     2,
		Timestamp: time.Now().Unix(),
		JobID:     jobID,
		Provider:  "prov-err",
		Result: types.JobResult{
			ResultData: []byte{0x1},
			ResultHash: "hasherr",
		},
	}
	dataBz, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.Packet{
		Data:          dataBz,
		SourceChannel: "channel-0",
		Sequence:      10,
	}

	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("remote validation failed"))

	err = k.OnAcknowledgementPacket(sdkCtx, packet, ack)
	require.Error(t, err)
	require.Contains(t, err.Error(), "acknowledgement failed")

	updated := k.getJob(sdkCtx, jobID)
	require.NotNil(t, updated)
	require.Equal(t, "failed", updated.Status)
	require.Equal(t, uint32(0), updated.Progress)
}

func TestOnRecvPacketJobResultValidationError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"job_result","nonce":0,"job_id":"","result":{"result_data":null}}`),
		SourceChannel: "channel-0",
		Sequence:      1,
	}

	ack, err := k.OnRecvPacket(sdkCtx, packet, 1)
	require.NoError(t, err)
	require.False(t, ack.Success())
	require.NotEmpty(t, ack.GetError())
}

func TestOnRecvPacketUnknownType(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"unknown","nonce":1}`),
		SourceChannel: "channel-0",
		Sequence:      2,
	}

	ack, err := k.OnRecvPacket(sdkCtx, packet, 2)
	require.NoError(t, err)
	require.False(t, ack.Success())
	require.NotEmpty(t, ack.GetError())
}

func TestOnRecvPacketJobResultVerificationFailureStillAcknowledges(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	jobID := "job-verif-fail"
	resultData := types.JobResultPacketData{
		Type:     types.JobResultType,
		Nonce:    3,
		JobID:    jobID,
		Provider: "prov-x",
		Result: types.JobResult{
			ResultData: []byte("payload"),
			ResultHash: "bad-hash", // mismatch to trigger verifyJobResult error
		},
	}
	bz, err := json.Marshal(resultData)
	require.NoError(t, err)

	packet := channeltypes.Packet{
		Data:          bz,
		SourceChannel: "channel-0",
		Sequence:      3,
	}

	ack, err := k.OnRecvPacket(sdkCtx, packet, 3)
	require.NoError(t, err)
	require.True(t, ack.Success(), "verification failures are logged but should not abort acknowledgement")

	var ackResp types.JobResultAcknowledgement
	require.NoError(t, json.Unmarshal(ack.GetResult(), &ackResp))
	require.Equal(t, uint64(3), ackResp.Nonce)
	require.Equal(t, jobID, ackResp.JobID)
	require.Equal(t, "completed", ackResp.Status)

	job := k.getJob(sdkCtx, jobID)
	require.NotNil(t, job)
	require.Equal(t, "completed", job.Status)
	require.True(t, job.Verified)
	require.Equal(t, "bad-hash", job.Result.ResultHash)
}

// =============================================================================
// Compute Module Tests (from ibc_compute_test.go)
// =============================================================================

func authorizeComputeChannel(t testing.TB, k *Keeper, ctx sdk.Context, channelID string) {
	t.Helper()
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))
}

// Tests moved to ibc_integration_test.go (external tests to avoid import cycle):
// - TestHandleJobResultReturnsAcknowledgement
// - TestHandleSubmitJobPersistsStateAndAck
// - TestHandleJobStatusQueryUsesStoredProgress
// - TestComputeOnAcknowledgementPacketRejectsOversizedPayload
// - TestComputeOnRecvPacketRejectsUnauthorizedChannel

func TestVerifyAttestationsFailsWithoutPublicKeys(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	attestations := [][]byte{[]byte("sig1"), []byte("sig2")}
	message := make([]byte, 32)

	err := k.VerifyAttestationsForTest(ctx, attestations, nil, message)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no public keys provided")
}

func TestJobResultAcknowledgementPersistsState(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithBlockTime(time.Now())

	jobID := "job-ack-success"
	initial := CrossChainComputeJob{
		JobID:       jobID,
		Status:      "running",
		Progress:    70,
		Provider:    "provider-ack",
		SubmittedAt: ctx.BlockTime(),
	}
	k.UpsertCrossChainJob(ctx, &initial)

	packetData := types.JobResultPacketData{
		Type:     types.JobResultType,
		JobID:    jobID,
		Provider: initial.Provider,
		Result: types.JobResult{
			ResultHash: "ignored",
		},
	}
	packetBytes, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		4,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ackResp := types.JobResultAcknowledgement{
		Success:         true,
		JobID:           jobID,
		Status:          "completed",
		Progress:        90,
		Provider:        packetData.Provider,
		ResultHash:      "result-hash",
		ProofHash:       "proof-hash",
		AttestationHash: "attestation-hash",
	}
	ackBytes, err := ackResp.GetBytes()
	require.NoError(t, err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)

	err = k.OnAcknowledgementPacket(ctx, packet, ack)
	require.NoError(t, err)

	stored := k.GetCrossChainJob(ctx, jobID)
	require.NotNil(t, stored)
	require.Equal(t, "completed", stored.Status)
	require.Equal(t, uint32(100), stored.Progress)
	require.Equal(t, "proof-hash", stored.ProofHash)
	require.Equal(t, "attestation-hash", stored.AttestationHash)
	require.NotNil(t, stored.Result)
	require.Equal(t, "result-hash", stored.Result.ResultHash)
	require.False(t, stored.Result.CompletedAt.IsZero())
}

func TestJobResultAcknowledgementFailureMarksJobFailed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithBlockTime(time.Now())

	jobID := "job-ack-failure"
	job := CrossChainComputeJob{
		JobID:       jobID,
		Status:      "running",
		Progress:    70,
		Provider:    "provider-fail",
		SubmittedAt: ctx.BlockTime(),
	}
	k.UpsertCrossChainJob(ctx, &job)

	packetData := types.JobResultPacketData{
		Type:     types.JobResultType,
		JobID:    jobID,
		Provider: job.Provider,
	}
	packetBytes, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		5,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ack := channeltypes.NewErrorAcknowledgement(errors.New("remote job result rejected"))

	require.False(t, ack.Success())
	t.Logf("job result error ack payload: %s", string(ack.Acknowledgement()))
	t.Logf("job result error message: %s", ack.GetError())

	err = k.OnAcknowledgementPacket(ctx, packet, ack)
	require.Error(t, err)

	stored := k.GetCrossChainJob(ctx, jobID)
	require.NotNil(t, stored)
	require.Equal(t, "failed", stored.Status)
	require.Equal(t, uint32(0), stored.Progress)
}

func TestGetValidatorPublicKeys_NoKeys(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	_, err := k.GetValidatorPublicKeysForTest(ctx, "non-existent-chain")
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrVerificationFailed)
}

func TestVerifyAttestations_NoPublicKeys(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	messageHash := sha256.Sum256([]byte("result-hash"))

	err := k.VerifyAttestationsForTest(ctx, [][]byte{[]byte("sig")}, [][]byte{}, messageHash[:])
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSignature)
}

// =============================================================================
// Helpers Tests (from ibc_compute_helpers_test.go and ibc_helpers_test.go)
// =============================================================================

func TestMax32AndHashBytes(t *testing.T) {
	require.Equal(t, uint32(5), max32(5, 3))
	require.Equal(t, uint32(7), max32(4, 7))

	require.Equal(t, "", hashBytes(nil))
	require.NotEmpty(t, hashBytes([]byte("data")))
}

func TestBuildMerkleProofPath(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	path := k.buildMerkleProofPath(sdkCtx, []byte("escrow_key"))
	require.Len(t, path, 3)
	require.NotEmpty(t, path[0])
	require.NotEmpty(t, path[1])
	require.NotEmpty(t, path[2])
}

func TestPendingDiscoveryStorage(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.storePendingDiscovery(sdkCtx, "channel-1", 7, "chain-X")
	store := sdkCtx.KVStore(k.storeKey)
	key := []byte("pending_discovery_7")
	require.Equal(t, []byte("chain-X"), store.Get(key))

	k.removePendingDiscovery(sdkCtx, "channel-1", 7)
	require.Nil(t, store.Get(key))
}

func TestGroth16ProofValidate_FailsOnInfinity(t *testing.T) {
	proof := &Groth16ProofBN254{}
	err := proof.Validate()
	require.Error(t, err)
}

func TestSendComputeIBCPacket_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ibc keeper not configured")
}

func TestSendComputeIBCPacket_SucceedsWithCapability(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.ibcKeeper = &ibckeeper.Keeper{}

	capPath := host.ChannelCapabilityPath(types.PortID, "channel-0")
	cap, err := k.scopedKeeper.NewCapability(sdkCtx, capPath)
	require.NoError(t, err)
	if err := k.scopedKeeper.ClaimCapability(sdkCtx, cap, capPath); err != nil {
		require.ErrorIs(t, err, capabilitytypes.ErrOwnerClaimed)
	}

	originalSend := sendPacketFn
	sendPacketFn = func(
		_ *Keeper,
		_ sdk.Context,
		_ *capabilitytypes.Capability,
		_ string,
		_ string,
		_ uint64,
		_ []byte,
	) (uint64, error) {
		return 99, nil
	}
	t.Cleanup(func() { sendPacketFn = originalSend })

	seq, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.NoError(t, err)
	require.Equal(t, uint64(99), seq)
}

func TestSendComputeIBCPacket_ChannelCapabilityMissing(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// provide a non-nil IBC keeper to bypass the initial nil check
	k.ibcKeeper = &ibckeeper.Keeper{}

	_, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel capability not found")
}

func TestSendComputeIBCPacket_SendPacketFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Seed a channel capability so SendPacket would be attempted
	capPath := host.ChannelCapabilityPath(types.PortID, "channel-0")
	_, err := k.scopedKeeper.NewCapability(sdkCtx, capPath)
	require.NoError(t, err)

	k.ibcKeeper = &ibckeeper.Keeper{}

	originalSend := sendPacketFn
	sendPacketFn = func(
		_ *Keeper,
		_ sdk.Context,
		_ *capabilitytypes.Capability,
		_ string,
		_ string,
		_ uint64,
		_ []byte,
	) (uint64, error) {
		return 0, fmt.Errorf("send failure")
	}
	defer func() { sendPacketFn = originalSend }()

	_, err = k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "send failure")
}

func TestDiscoverRemoteProviders_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	providers, err := k.DiscoverRemoteProviders(ctx, []string{AkashChainID}, []string{"gpu"}, math.LegacyNewDec(1))
	require.NoError(t, err)
	require.Len(t, providers, 0)
}

func TestSubmitCrossChainJob_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	requester := sdk.AccAddress([]byte("job_requester"))
	specs := JobRequirements{CPUCores: 1, MemoryMB: 512, StorageGB: 5}
	err := k.bankKeeper.MintCoins(sdk.UnwrapSDKContext(ctx), types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)))
	require.NoError(t, err)
	err = k.bankKeeper.SendCoinsFromModuleToAccount(sdk.UnwrapSDKContext(ctx), types.ModuleName, requester, sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)))
	require.NoError(t, err)

	_, err = k.SubmitCrossChainJob(ctx, "docker", []byte{0x1}, specs, AkashChainID, "provider1", requester, sdk.NewInt64Coin("upaw", 1000))
	require.Error(t, err)
}

func TestQueryCrossChainJobStatus_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	// store job
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.storeJob(sdkCtx, "job-123", &CrossChainComputeJob{
		JobID:       "job-123",
		TargetChain: AkashChainID,
		Status:      "pending",
		Requester:   sdk.AccAddress([]byte("job_req")).String(),
	})
	job, err := k.QueryCrossChainJobStatus(ctx, "job-123")
	require.NoError(t, err)
	require.Equal(t, "job-123", job.JobID)
}

func TestCreateEscrowProof(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("proof_requester"))
	amount := sdk.NewInt64Coin("upaw", 100)
	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))

	escrow := &CrossChainEscrow{
		JobID:     "job-proof-1",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	proof, err := k.createEscrowProof(sdkCtx, escrow)
	require.NoError(t, err)
	require.NotEmpty(t, proof)
}

func TestVerifyGroth16PairingFailsWithoutCircuit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	proof := &Groth16ProofBN254{}
	err := k.verifyGroth16Pairing(sdkCtx, []byte{}, proof, bn254.G1Affine{})
	require.Error(t, err)
}

func TestGetComputeChannel(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	channel, err := k.getComputeChannel(sdkCtx, AkashChainID)
	require.NoError(t, err)
	require.Equal(t, "channel-akash", channel)

	store := sdkCtx.KVStore(k.storeKey)
	store.Set([]byte("compute_channel_custom"), []byte("channel-custom"))
	channel, err = k.getComputeChannel(sdkCtx, "custom")
	require.NoError(t, err)
	require.Equal(t, "channel-custom", channel)

	store.Set([]byte("compute_channel_"+RenderChainID), []byte("channel-override"))
	channel, err = k.getComputeChannel(sdkCtx, RenderChainID)
	require.NoError(t, err)
	require.Equal(t, "channel-override", channel)
}

func TestEscrowLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_requester"))
	provider := sdk.AccAddress([]byte("escrow_provider"))
	amount := sdk.NewInt64Coin("upaw", 1000)

	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))

	// Lock funds
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))
	k.storeEscrow(sdkCtx, "job-1", &CrossChainEscrow{
		JobID:     "job-1",
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	initialProviderBal := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")

	// Release to provider
	require.NoError(t, k.releaseEscrow(sdkCtx, "job-1"))
	escrow := k.getEscrow(sdkCtx, "job-1")
	require.Equal(t, "released", escrow.Status)
	require.NotNil(t, escrow.ReleasedAt)
	finalProviderBal := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")
	require.Equal(t, initialProviderBal.Amount.Add(amount.Amount), finalProviderBal.Amount)
}

func TestReleaseEscrowFailsWithInvalidProvider(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	escrow := &CrossChainEscrow{
		JobID:     "job-invalid-provider",
		Requester: sdk.AccAddress([]byte("req")).String(),
		Provider:  "not-a-bech32", // force address decode failure
		Amount:    math.NewInt(500),
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	err := k.releaseEscrow(sdkCtx, escrow.JobID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bech32")

	stored := k.getEscrow(sdkCtx, escrow.JobID)
	require.NotNil(t, stored)
	require.Equal(t, "locked", stored.Status)
	require.Nil(t, stored.ReleasedAt)
}

func TestReleaseEscrowFailsWithInsufficientFunds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Escrow stored but module account not funded for transfer
	escrow := &CrossChainEscrow{
		JobID:     "job-no-funds",
		Requester: sdk.AccAddress([]byte("req-no-funds")).String(),
		Provider:  sdk.AccAddress([]byte("prov-no-funds")).String(),
		Amount:    math.NewInt(10_000),
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	err := k.releaseEscrow(sdkCtx, escrow.JobID)
	require.Error(t, err)

	stored := k.getEscrow(sdkCtx, escrow.JobID)
	require.NotNil(t, stored)
	require.Equal(t, "locked", stored.Status)
}

func TestPendingJobTracking(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.storePendingJobSubmission(sdkCtx, "channel-0", 1, "job-123")
	require.Equal(t, "job-123", k.getPendingJobSubmission(sdkCtx, 1))
	k.removePendingJobSubmission(sdkCtx, "channel-0", 1)
	require.Equal(t, "", k.getPendingJobSubmission(sdkCtx, 1))
}

func TestCachedProviderStorage(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	provider := &RemoteComputeProvider{
		ChainID:      "chain-X",
		ProviderID:   "p1",
		Address:      "addr1",
		Capabilities: []string{"gpu"},
		PricePerUnit: math.LegacyNewDec(5),
		Reputation:   math.LegacyNewDec(9),
		Active:       true,
		LastSeen:     time.Now(),
	}
	k.storeProvider(sdkCtx, provider)

	result := k.getCachedProviders(sdkCtx, []string{"gpu"}, math.LegacyNewDec(10))
	require.Len(t, result, 1)
	require.Equal(t, "p1", result[0].ProviderID)
}

func TestProgressForStatus(t *testing.T) {
	require.Equal(t, uint32(10), progressForStatus("pending", 0))
	require.Equal(t, uint32(100), progressForStatus("completed", 50))
	require.Equal(t, uint32(0), progressForStatus("failed", 70))
	require.Equal(t, uint32(25), progressForStatus("accepted", 10))
}

// =============================================================================
// Validation Tests (from ibc_compute_validate_test.go)
// =============================================================================

func TestJobResultPacketDataValidateBasic(t *testing.T) {
	valid := types.JobResultPacketData{
		Type:      types.JobResultType,
		Nonce:     1,
		Timestamp: 1,
		JobID:     "job-1",
		Provider:  "prov-1",
		Result: types.JobResult{
			ResultData: []byte{0x1},
			ResultHash: "abc",
		},
	}

	tt := []struct {
		name    string
		mutate  func(p *types.JobResultPacketData)
		wantErr string
	}{
		{"bad type", func(p *types.JobResultPacketData) { p.Type = "bad" }, "invalid packet type"},
		{"zero nonce", func(p *types.JobResultPacketData) { p.Nonce = 0 }, "nonce"},
		{"zero timestamp", func(p *types.JobResultPacketData) { p.Timestamp = 0 }, "timestamp"},
		{"empty job id", func(p *types.JobResultPacketData) { p.JobID = "" }, "job ID"},
		{"empty result data", func(p *types.JobResultPacketData) { p.Result.ResultData = nil }, "result data"},
		{"empty result hash", func(p *types.JobResultPacketData) { p.Result.ResultHash = "" }, "result hash"},
		{"empty provider", func(p *types.JobResultPacketData) { p.Provider = "" }, "provider"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.mutate(&p)
			err := p.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}

	require.NoError(t, valid.ValidateBasic())
}

// =============================================================================
// Verification Tests (from ibc_compute_verification_test.go)
// =============================================================================

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

// =============================================================================
// Packet Tracking Tests (from ibc_packet_tracking_test.go and ibc_packet_tracking_cover_test.go)
// =============================================================================

func TestGetPacketNonceKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-0", 1)
		require.Equal(t, key1, key2)
	})

	t.Run("different channels produce different keys", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-1", 1)
		require.NotEqual(t, key1, key2)
	})

	t.Run("different sequences produce different keys", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-0", 2)
		require.NotEqual(t, key1, key2)
	})
}

func TestHasPacketBeenProcessed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns false for unprocessed packet", func(t *testing.T) {
		processed := k.HasPacketBeenProcessed(sdkCtx, "channel-0", 1)
		require.False(t, processed)
	})
}

func TestMarkPacketAsProcessed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("marks packet as processed", func(t *testing.T) {
		err := k.MarkPacketAsProcessed(sdkCtx, "channel-0", 1)
		require.NoError(t, err)
		processed := k.HasPacketBeenProcessed(sdkCtx, "channel-0", 1)
		require.True(t, processed)
	})
}

func TestValidatePacketOrdering(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("first packet validates", func(t *testing.T) {
		packet := channeltypes.Packet{
			Sequence:           1,
			SourcePort:         "compute",
			SourceChannel:      "channel-0",
			DestinationPort:    "compute",
			DestinationChannel: "channel-0",
		}
		err := k.ValidatePacketOrdering(sdkCtx, packet)
		require.NoError(t, err)
	})
}

func TestGetLastProcessedSequence(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns 0 for new channel", func(t *testing.T) {
		seq := k.GetLastProcessedSequence(sdkCtx, "channel-new")
		require.Equal(t, uint64(0), seq)
	})
}

func TestSetLastProcessedSequence(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("sets and gets sequence", func(t *testing.T) {
		k.SetLastProcessedSequence(sdkCtx, "channel-0", 5)
		seq := k.GetLastProcessedSequence(sdkCtx, "channel-0")
		require.Equal(t, uint64(5), seq)
	})
}

func TestCleanupOldPacketNonces(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("cleanup with no nonces", func(t *testing.T) {
		err := k.CleanupOldPacketNonces(sdkCtx, 100)
		require.NoError(t, err)
	})
}

func TestGetJobStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns error for non-existent job", func(t *testing.T) {
		status, err := k.GetJobStatus(sdkCtx, "non-existent-job")
		require.Error(t, err)
		require.Nil(t, status)
	})
}

func TestCleanupOldPacketNoncesPrunesAndEmitsEvent(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	store := sdkCtx.KVStore(k.storeKey)
	keyOld := []byte("pending_packet_channel-1_1")
	keyNew := []byte("pending_packet_channel-1_2")

	valOld := make([]byte, 8)
	binary.BigEndian.PutUint64(valOld, uint64(sdkCtx.BlockHeight()-10))
	valNew := make([]byte, 8)
	binary.BigEndian.PutUint64(valNew, uint64(sdkCtx.BlockHeight()))

	store.Set(keyOld, valOld)
	store.Set(keyNew, valNew)

	require.NoError(t, k.CleanupOldPacketNonces(sdkCtx, 5))

	require.Nil(t, store.Get(keyOld))
	require.NotNil(t, store.Get(keyNew))
	events := sdkCtx.EventManager().Events()
	require.NotEmpty(t, events)
}

func TestRefundEscrowOnTimeoutMissingEscrow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	err := k.RefundEscrowOnTimeout(sdkCtx, "unknown-job", "timeout")
	require.Error(t, err)
	require.Contains(t, err.Error(), "escrow not found")
}

func TestRefundEscrowOnTimeoutSucceeds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_refund_req"))
	amount := sdk.NewInt64Coin("upaw", 200)

	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))

	k.storeEscrow(sdkCtx, "job-refund", &CrossChainEscrow{
		JobID:     "job-refund",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	balBefore := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount
	require.NoError(t, k.RefundEscrowOnTimeout(sdkCtx, "job-refund", "timeout"))
	balAfter := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount

	require.True(t, balAfter.GTE(balBefore))
}

func TestRefundEscrowOnTimeoutFailsWithInsufficientModuleFunds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_refund_req2"))
	amount := sdk.NewInt64Coin("upaw", 500)

	// Do NOT mint or lock escrow funds, leaving module account empty
	k.storeEscrow(sdkCtx, "job-refund-fail", &CrossChainEscrow{
		JobID:     "job-refund-fail",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	err := k.RefundEscrowOnTimeout(sdkCtx, "job-refund-fail", "timeout")
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
}

func TestGetJobStatusMissingAndExisting(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	status, err := k.GetJobStatus(sdkCtx, "missing-job")
	require.Error(t, err)
	require.Nil(t, status)

	jobStatus := JobStatus{
		JobID:        "job-status-1",
		Status:       "completed",
		Requester:    "req1",
		Provider:     "prov1",
		Progress:     90,
		UpdatedAt:    sdkCtx.BlockTime().Unix(),
		ErrorMessage: "",
	}
	require.NoError(t, k.storeJobStatus(sdkCtx, jobStatus))

	found, err := k.GetJobStatus(sdkCtx, jobStatus.JobID)
	require.NoError(t, err)
	require.Equal(t, jobStatus.JobID, found.JobID)
	require.Equal(t, jobStatus.Status, found.Status)
	require.Equal(t, jobStatus.Progress, found.Progress)
}

// =============================================================================
// Timeout Tests (from ibc_timeout_test.go)
// =============================================================================

func TestOnTimeoutPacketRefundsEscrow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-timeout-1"
	requester := sdk.AccAddress(bytes.Repeat([]byte{0x21}, 20))
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20))
	escrowAmount := math.NewInt(2_500_000)
	channelID := "channel-7"
	sequence := uint64(42)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	job := CrossChainComputeJob{
		JobID:        jobID,
		Requester:    requester.String(),
		Provider:     provider.String(),
		Status:       "pending",
		Progress:     10,
		SubmittedAt:  ctx.BlockTime(),
		Requirements: JobRequirements{},
		EscrowAmount: escrowAmount,
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("job_%s", jobID)), job)

	escrow := CrossChainEscrow{
		JobID:     jobID,
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    escrowAmount,
		Status:    "locked",
		LockedAt:  ctx.BlockTime(),
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("escrow_%s", jobID)), escrow)
	store.Set([]byte(fmt.Sprintf("pending_job_%d", sequence)), []byte(jobID))
	recordPendingOperation(t, store, channelID, sequence, ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: PacketTypeSubmitJob,
		JobID:      jobID,
	})

	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowAmount))
	require.NoError(t, getBankKeeper(t, k).MintCoins(ctx, types.ModuleName, coins))

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	before := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.True(t, before.IsZero())

	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	after := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.Equal(t, escrowAmount, after.Amount)

	var stored CrossChainComputeJob
	require.NoError(t, json.Unmarshal(store.Get([]byte(fmt.Sprintf("job_%s", jobID))), &stored))
	require.Equal(t, "timeout", stored.Status)

	events := ctx.EventManager().Events()
	hasTimeout := false
	hasRefund := false
	for _, evt := range events {
		switch evt.Type {
		case "job_submission_timeout":
			hasTimeout = true
		case "escrow_refunded":
			hasRefund = true
		}
	}

	require.True(t, hasTimeout, "expected job timeout event")
	require.True(t, hasRefund, "expected escrow refund event")
}

func TestOnTimeoutPacketSubmitJobMissingJob(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-9"
	sequence := uint64(77)

	// No job or escrow exists - simulates already processed or invalid submission

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should not error even with missing job
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))
}

func TestOnTimeoutPacketDiscoverProviders(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-2"
	sequence := uint64(88)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	// Record pending discovery operation
	recordPendingOperation(t, store, channelID, sequence, ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: PacketTypeDiscoverProviders,
		JobID:      "",
	})

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s","requirements":{"cpu":4,"memory":8000}}`, PacketTypeDiscoverProviders)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Verify pending operation exists
	opKey := getPendingOperationKey(channelID, sequence)
	require.NotNil(t, store.Get(opKey))

	// Process timeout - should remove pending discovery
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify pending operation was removed
	require.Nil(t, store.Get(opKey))
}

func TestOnTimeoutPacketJobStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-status-query-1"
	channelID := "channel-4"
	sequence := uint64(123)

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s","job_id":"%s"}`, PacketTypeJobStatus, jobID)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Status query timeout is non-critical - should succeed without error
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// No state changes expected for status query timeout
	// This is a read-only operation, so timeout just means we don't get the status
}

func TestOnTimeoutPacketJobSubmissionRefundStateConsistency(t *testing.T) {
	// This test verifies that state remains consistent after a job submission timeout
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-consistency-check"
	requester := sdk.AccAddress(bytes.Repeat([]byte{0x77}, 20))
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x88}, 20))
	escrowAmount := math.NewInt(5_000_000)
	channelID := "channel-11"
	sequence := uint64(200)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	// Create job with specific status
	job := CrossChainComputeJob{
		JobID:        jobID,
		Requester:    requester.String(),
		Provider:     provider.String(),
		Status:       "pending",
		Progress:     0,
		SubmittedAt:  ctx.BlockTime(),
		Requirements: JobRequirements{},
		EscrowAmount: escrowAmount,
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("job_%s", jobID)), job)

	// Create locked escrow
	escrow := CrossChainEscrow{
		JobID:     jobID,
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    escrowAmount,
		Status:    "locked",
		LockedAt:  ctx.BlockTime(),
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("escrow_%s", jobID)), escrow)

	// Track pending submission
	store.Set([]byte(fmt.Sprintf("pending_job_%d", sequence)), []byte(jobID))
	recordPendingOperation(t, store, channelID, sequence, ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: PacketTypeSubmitJob,
		JobID:      jobID,
	})

	// Fund module for refund
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowAmount))
	require.NoError(t, getBankKeeper(t, k).MintCoins(ctx, types.ModuleName, coins))

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Process timeout
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify state consistency:

	// 1. Job status should be "timeout"
	var storedJob CrossChainComputeJob
	require.NoError(t, json.Unmarshal(store.Get([]byte(fmt.Sprintf("job_%s", jobID))), &storedJob))
	require.Equal(t, "timeout", storedJob.Status)

	// 2. Escrow should be refunded to requester
	balance := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.Equal(t, escrowAmount, balance.Amount)

	// 3. Module account should be drained
	moduleBalance := getBankKeeper(t, k).GetBalance(ctx, sdk.AccAddress(types.ModuleName), "upaw")
	require.True(t, moduleBalance.IsZero())

	// 4. Pending job tracking should be removed
	require.Nil(t, store.Get([]byte(fmt.Sprintf("pending_job_%d", sequence))))

	// 5. Pending operation should be removed
	opKey := getPendingOperationKey(channelID, sequence)
	require.Nil(t, store.Get(opKey))

	// 6. Events should be emitted
	events := ctx.EventManager().Events()
	hasTimeoutEvent := false
	hasRefundEvent := false
	for _, evt := range events {
		switch evt.Type {
		case "job_submission_timeout":
			hasTimeoutEvent = true
		case "escrow_refunded":
			hasRefundEvent = true
		}
	}
	require.True(t, hasTimeoutEvent, "expected job_submission_timeout event")
	require.True(t, hasRefundEvent, "expected escrow_refunded event")
}

func TestOnTimeoutPacketInvalidPacketType(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{"type":"unknown_compute_packet"}`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for unknown packet type
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown packet type")
}

func TestOnTimeoutPacketMalformedJSON(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{invalid json payload`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for malformed JSON
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal packet data")
}

func TestOnTimeoutPacketMissingType(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{"job_id":"some-job"}`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for missing type field
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing packet type")
}

func TestOnTimeoutPacketSubmitJobRefundsAndUpdatesStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	job := &CrossChainComputeJob{
		JobID:    "job-timeout",
		Status:   "submitted",
		Progress: 20,
	}
	k.storeJob(sdkCtx, job.JobID, job)
	k.storePendingJobSubmission(sdkCtx, "channel-0", 11, job.JobID)

	// fund requester and lock escrow so refund path is exercised
	requester := sdk.AccAddress([]byte("req_timeout"))
	amount := sdk.NewCoin("upaw", math.NewInt(100))
	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))
	k.storeEscrow(sdkCtx, job.JobID, &CrossChainEscrow{
		JobID:     job.JobID,
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	packet := channeltypes.Packet{
		Data:             []byte(`{"type":"submit_job"}`),
		SourceChannel:    "channel-0",
		Sequence:         11,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	require.NoError(t, k.OnTimeoutPacket(sdkCtx, packet))

	updated := k.getJob(sdkCtx, job.JobID)
	require.NotNil(t, updated)
	require.Equal(t, "timeout", updated.Status)
	require.Equal(t, "", k.getPendingJobSubmission(sdkCtx, packet.Sequence))
}

func TestOnTimeoutPacketUnknownType(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"unexpected"}`),
		SourceChannel: "channel-0",
		Sequence:      1,
	}

	err := k.OnTimeoutPacket(sdkCtx, packet)
	require.Error(t, err)
}

func TestOnTimeoutPacketDiscoverProvidersCleansPending(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.storePendingDiscovery(sdkCtx, "channel-9", 55, "chain-Y")
	packet := channeltypes.Packet{
		Data:          []byte(`{"type":"discover_providers"}`),
		SourceChannel: "channel-9",
		Sequence:      55,
	}

	require.NoError(t, k.OnTimeoutPacket(sdkCtx, packet))
	require.Nil(t, sdkCtx.KVStore(k.storeKey).Get([]byte("pending_discovery_55")))
}

// =============================================================================
// Test Helper Functions (from ibc_timeout_test.go)
// =============================================================================

func getStoreKey(t *testing.T, k *Keeper) storetypes.StoreKey {
	t.Helper()
	val := reflect.ValueOf(k).Elem().FieldByName("storeKey")
	return reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem().Interface().(storetypes.StoreKey)
}

func getBankKeeper(t *testing.T, k *Keeper) bankkeeper.Keeper {
	t.Helper()
	val := reflect.ValueOf(k).Elem().FieldByName("bankKeeper")
	return reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem().Interface().(bankkeeper.Keeper)
}

func mustSetJSON(t *testing.T, store storetypes.KVStore, key []byte, value interface{}) {
	t.Helper()
	bz, err := json.Marshal(value)
	require.NoError(t, err)
	store.Set(key, bz)
}

func recordPendingOperation(t *testing.T, store storetypes.KVStore, channelID string, sequence uint64, op ChannelOperation) {
	t.Helper()
	bz, err := json.Marshal(op)
	require.NoError(t, err)
	prefix := []byte(fmt.Sprintf("compute_pending/%s/", channelID))
	seqBz := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBz, sequence)
	store.Set(append(prefix, seqBz...), bz)
}

func getPendingOperationKey(channelID string, sequence uint64) []byte {
	prefix := []byte(fmt.Sprintf("compute_pending/%s/", channelID))
	seqBz := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBz, sequence)
	return append(prefix, seqBz...)
}
