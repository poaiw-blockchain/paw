package keeper_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app/ibcutil"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computemodule "github.com/paw-chain/paw/x/compute"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestHandleJobResultReturnsAcknowledgement(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	authorizeComputeChannel(t, k, ctx, "channel-0")

	result := types.JobResult{
		ResultData:      []byte(`{"result":"ok"}`),
		ResultHash:      "hash-123",
		ComputeTime:     1234,
		AttestationSigs: [][]byte{[]byte("sig1")},
		Timestamp:       ctx.BlockTime().Unix(),
	}

	packetData := types.JobResultPacketData{
		Nonce:     1,
		Type:      types.JobResultType,
		Timestamp: ctx.BlockTime().Unix(),
		JobID:     "job-ack-1",
		Result:    result,
		Provider:  "provider-ack",
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ibcModule := computemodule.NewIBCModule(k, nil)
	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	t.Logf("job result ack payload: %s", string(ack.Acknowledgement()))
	t.Logf("job result ack success flag: %v", ack.Success())
	require.True(t, ack.Success(), "ack: %s", string(ack.Acknowledgement()))

	var channelAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &channelAck))
	require.True(t, channelAck.Success())

	var ackResp types.JobResultAcknowledgement
	require.NoError(t, json.Unmarshal(channelAck.GetResult(), &ackResp))

	sigHash := sha256.Sum256([]byte("sig1"))
	expectedAttestationHash := hex.EncodeToString(sigHash[:])

	require.Equal(t, packetData.JobID, ackResp.JobID)
	require.Equal(t, "completed", ackResp.Status)
	require.Equal(t, uint32(100), ackResp.Progress)
	require.Equal(t, result.ResultHash, ackResp.ResultHash)
	require.Equal(t, packetData.Provider, ackResp.Provider)
	require.Equal(t, expectedAttestationHash, ackResp.AttestationHash)
	require.Empty(t, ackResp.ProofHash)

	job := k.GetCrossChainJob(ctx, packetData.JobID)
	require.NotNil(t, job)
	require.Equal(t, "completed", job.Status)
	require.Equal(t, ackResp.Progress, job.Progress)
	require.Equal(t, expectedAttestationHash, job.AttestationHash)
}

func authorizeComputeChannel(t testing.TB, k *keeper.Keeper, ctx sdk.Context, channelID string) {
	t.Helper()
	require.NoError(t, ibcutil.AuthorizeChannel(ctx, k, types.PortID, channelID))
}

func TestHandleSubmitJobPersistsStateAndAck(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	authorizeComputeChannel(t, k, ctx, "channel-0")

	requester := sdk.AccAddress("requester1_________")
	provider := sdk.AccAddress("provider1__________")

	packetData := types.SubmitJobPacketData{
		Nonce:     1,
		Type:      types.SubmitJobType,
		Timestamp: ctx.BlockTime().Unix(),
		JobID:     "job-submit-1",
		JobType:   "docker",
		JobData:   []byte{0x1},
		Requirements: types.JobRequirements{
			CPUCores:    1,
			MemoryMB:    512,
			StorageGB:   1,
			MaxDuration: 600,
		},
		Provider:    provider.String(),
		Requester:   requester.String(),
		EscrowProof: []byte("proof"),
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		2,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ibcModule := computemodule.NewIBCModule(k, nil)
	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	require.True(t, ack.Success(), "ack: %s", string(ack.Acknowledgement()))

	var channelAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &channelAck))
	require.True(t, channelAck.Success())

	var ackResp types.SubmitJobAcknowledgement
	require.NoError(t, json.Unmarshal(channelAck.GetResult(), &ackResp))

	require.Equal(t, packetData.JobID, ackResp.JobID)
	require.Equal(t, "running", ackResp.Status)
	require.Equal(t, packetData.Requirements.MaxDuration, ackResp.EstimatedTime)
	require.True(t, ackResp.Progress > 0)

	job := k.GetCrossChainJob(ctx, packetData.JobID)
	require.NotNil(t, job)
	require.Equal(t, ackResp.Status, job.Status)
	require.Equal(t, ackResp.Progress, job.Progress)
	require.Equal(t, packetData.Requester, job.Requester)
	require.Equal(t, packetData.Provider, job.Provider)
}

func TestHandleJobStatusQueryUsesStoredProgress(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	authorizeComputeChannel(t, k, ctx, "channel-0")

	requester := sdk.AccAddress("requester1_________")
	provider := sdk.AccAddress("provider1__________")

	job := keeper.CrossChainComputeJob{
		JobID:       "job-status-1",
		Status:      "running",
		Progress:    70,
		Provider:    provider.String(),
		Requester:   requester.String(),
		SubmittedAt: ctx.BlockTime(),
	}
	k.UpsertCrossChainJob(ctx, &job)

	packetData := types.JobStatusPacketData{
		Nonce:     1,
		Type:      types.JobStatusType,
		Timestamp: ctx.BlockTime().Unix(),
		JobID:     job.JobID,
		Requester: job.Requester,
	}

	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		3,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 100),
		0,
	)

	ibcModule := computemodule.NewIBCModule(k, nil)
	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	require.True(t, ack.Success(), "ack: %s", string(ack.Acknowledgement()))

	var channelAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &channelAck))
	require.True(t, channelAck.Success())

	var ackResp types.JobStatusAcknowledgement
	require.NoError(t, json.Unmarshal(channelAck.GetResult(), &ackResp))

	require.Equal(t, job.JobID, ackResp.JobID)
	require.Equal(t, job.Status, ackResp.Status)
	require.Equal(t, job.Progress, ackResp.Progress)
}

func TestComputeOnAcknowledgementPacketRejectsOversizedPayload(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ibcModule := computemodule.NewIBCModule(k, nil)

	packet := channeltypes.NewPacket(
		nil,
		1,
		types.PortID,
		"channel-0",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(0, 10),
		0,
	)

	// Create ack larger than 256KB limit (512KB)
	oversizedAck := bytes.Repeat([]byte{0x1}, 512*1024)
	err := ibcModule.OnAcknowledgementPacket(ctx, packet, oversizedAck, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidAck)
}

func TestComputeOnRecvPacketRejectsUnauthorizedChannel(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ibcModule := computemodule.NewIBCModule(k, nil)

	packetData := types.SubmitJobPacketData{
		Type:    types.SubmitJobType,
		Nonce:   1,
		JobID:   "job-unauthorized",
		JobType: "docker",
		JobData: []byte("{}"),
		Requirements: types.JobRequirements{
			CPUCores:    1,
			MemoryMB:    512,
			StorageGB:   1,
			MaxDuration: 300,
		},
		Provider:  sdk.AccAddress("provider1__________").String(),
		Requester: sdk.AccAddress("requester1_________").String(),
	}
	packetBytes, err := packetData.GetBytes()
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		types.PortID,
		"channel-99",
		types.PortID,
		"channel-1",
		clienttypes.NewHeight(1, 10),
		0,
	)

	ack := ibcModule.OnRecvPacket(ctx, packet, nil)
	var chAck channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ack.Acknowledgement(), &chAck))
	require.False(t, chAck.Success())
	require.Contains(t, chAck.GetError(), fmt.Sprintf("ABCI code: %d", types.ErrUnauthorizedChannel.ABCICode()))
}

func TestVerifyAttestationsFailsWithoutPublicKeys(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	attestations := [][]byte{[]byte("sig1"), []byte("sig2")}
	message := make([]byte, 32)

	err := k.VerifyAttestationsForTest(ctx, attestations, nil, message)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no public keys provided")
}

func TestJobResultAcknowledgementPersistsState(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())

	jobID := "job-ack-success"
	initial := keeper.CrossChainComputeJob{
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
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())

	jobID := "job-ack-failure"
	job := keeper.CrossChainComputeJob{
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
	k, ctx := keepertest.ComputeKeeper(t)
	_, err := k.GetValidatorPublicKeysForTest(ctx, "non-existent-chain")
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrVerificationFailed)
}

func TestVerifyAttestations_NoPublicKeys(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	messageHash := sha256.Sum256([]byte("result-hash"))

	err := k.VerifyAttestationsForTest(ctx, [][]byte{[]byte("sig")}, [][]byte{}, messageHash[:])
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSignature)
}
