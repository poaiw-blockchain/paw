package keeper

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

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
