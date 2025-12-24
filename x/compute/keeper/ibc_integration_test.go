package keeper_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computemodule "github.com/paw-chain/paw/x/compute"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// =============================================================================
// IBC Module Integration Tests
// These tests require the full IBCModule wrapper and use external test package
// to avoid import cycles with testutil/keeper.
// =============================================================================

func TestHandleJobResultReturnsAcknowledgement(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	k.AuthorizeComputeChannelForTest(ctx, "channel-0")

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

func TestHandleSubmitJobPersistsStateAndAck(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithBlockTime(time.Now())
	k.AuthorizeComputeChannelForTest(ctx, "channel-0")

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
	k.AuthorizeComputeChannelForTest(ctx, "channel-0")

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
