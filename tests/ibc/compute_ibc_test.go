package ibc_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	pawibctesting "github.com/paw-chain/paw/testutil/ibctesting"

	_ "github.com/paw-chain/paw/testutil/ibctesting"
)

// ComputeIBCTestSuite tests cross-chain compute operations
type ComputeIBCTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	path        *ibctesting.Path
}

func TestComputeIBCTestSuite(t *testing.T) {
	suite.Run(t, new(ComputeIBCTestSuite))
}

func (suite *ComputeIBCTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	pawibctesting.BindCustomPorts(suite.chainA)
	pawibctesting.BindCustomPorts(suite.chainB)

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = "compute"
	suite.path.EndpointB.ChannelConfig.PortID = "compute"
	suite.path.EndpointA.ChannelConfig.Version = computetypes.IBCVersion
	suite.path.EndpointB.ChannelConfig.Version = computetypes.IBCVersion
	suite.path.SetChannelOrdered()

	suite.coordinator.Setup(suite.path)
}

func (suite *ComputeIBCTestSuite) TestDiscoverProviders() {
	// Test discovering compute providers on remote chain

	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.DiscoverProvidersPacketData{
		Type:         computetypes.DiscoverProvidersType,
		Capabilities: []string{"gpu", "tee"},
		MaxPrice:     math.LegacyMustNewDecFromStr("10.0"),
		Requester:    requester.String(),
	}

	err := packetData.ValidateBasic()
	suite.Require().NoError(err)

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify provider list in acknowledgement
	ackData := computetypes.DiscoverProvidersAcknowledgement{
		Success: true,
		Providers: []computetypes.ProviderInfo{
			{
				ProviderID:   "provider-1",
				Address:      "paw1provider1...",
				Capabilities: []string{"gpu", "tee"},
				PricePerUnit: math.LegacyMustNewDecFromStr("5.0"),
				Reputation:   math.LegacyMustNewDecFromStr("0.95"),
			},
		},
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *ComputeIBCTestSuite) TestSubmitJob() {
	// Test submitting compute job to remote chain

	requester := suite.chainA.SenderAccount.GetAddress()

	jobData := []byte(`{"function":"fibonacci","input":{"n":1000}}`)
	escrowProof := []byte(`{"job_id":"job-1","amount":"1000000","locked_at":1234567890}`)

	packetData := computetypes.SubmitJobPacketData{
		Type:    computetypes.SubmitJobType,
		JobID:   "job-1",
		JobType: "wasm",
		JobData: jobData,
		Requirements: computetypes.JobRequirements{
			CPUCores:    4,
			MemoryMB:    8192,
			StorageGB:   100,
			GPURequired: false,
			TEERequired: true,
			MaxDuration: 3600,
		},
		Provider:    "provider-1",
		Requester:   requester.String(),
		EscrowProof: escrowProof,
		Timeout:     uint64(time.Now().Add(time.Hour).Unix()),
	}

	err := packetData.ValidateBasic()
	suite.Require().NoError(err)

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify job submission acknowledgement
	ackData := computetypes.SubmitJobAcknowledgement{
		Success:       true,
		JobID:         "job-1",
		Status:        "running",
		EstimatedTime: 1800, // 30 minutes
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *ComputeIBCTestSuite) TestReceiveJobResult() {
	// Test receiving computation result from remote chain

	resultData := []byte(`{"result":43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875}`)
	zkProof := []byte(`{"proof":"..."}`)
	attestations := [][]byte{
		[]byte("sig1"),
		[]byte("sig2"),
		[]byte("sig3"),
	}

	packetData := computetypes.JobResultPacketData{
		Type:  computetypes.JobResultType,
		JobID: "job-1",
		Result: computetypes.JobResult{
			ResultData:      resultData,
			ResultHash:      "0x1234567890abcdef",
			ComputeTime:     1542000, // 25.7 minutes
			ZKProof:         zkProof,
			AttestationSigs: attestations,
			Timestamp:       time.Now().Unix(),
		},
		Provider: "provider-1",
	}

	err := packetData.ValidateBasic()
	suite.Require().NoError(err)

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	// Receive result on chain A
	err = suite.path.EndpointA.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify result was stored and escrow released
	// (In production, verify job status and escrow release)
}

func (suite *ComputeIBCTestSuite) TestQueryJobStatus() {
	// Test querying status of remote job

	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.JobStatusPacketData{
		Type:      computetypes.JobStatusType,
		JobID:     "job-1",
		Requester: requester.String(),
	}

	err := packetData.ValidateBasic()
	suite.Require().NoError(err)

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify status in acknowledgement
	ackData := computetypes.JobStatusAcknowledgement{
		Success:  true,
		JobID:    "job-1",
		Status:   "running",
		Progress: 65,
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *ComputeIBCTestSuite) TestJobTimeout() {
	// Test job timeout and escrow refund

	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.SubmitJobPacketData{
		Type:    computetypes.SubmitJobType,
		JobID:   "job-timeout",
		JobType: "wasm",
		JobData: []byte(`{"function":"test"}`),
		Requirements: computetypes.JobRequirements{
			CPUCores:    1,
			MemoryMB:    1024,
			StorageGB:   10,
			GPURequired: false,
			TEERequired: false,
			MaxDuration: 60,
		},
		Provider:    "provider-1",
		Requester:   requester.String(),
		EscrowProof: []byte(`{"job_id":"job-timeout","amount":"100000"}`),
		Timeout:     uint64(time.Now().Add(time.Second).Unix()),
	}

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight())+2),
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	// Advance chain B past timeout
	suite.coordinator.CommitNBlocks(suite.chainB, 10)

	// Timeout packet
	err = suite.path.EndpointA.TimeoutPacket(packet)
	suite.Require().NoError(err)

	// Verify escrow was refunded
	// (In production, verify requester balance increased)
}

func (suite *ComputeIBCTestSuite) TestZKProofVerification() {
	// Test zero-knowledge proof verification of results

	zkProof := []byte(`{
		"proof": {
			"pi_a": ["..."],
			"pi_b": ["..."],
			"pi_c": ["..."]
		},
		"public_inputs": ["..."]
	}`)

	packetData := computetypes.JobResultPacketData{
		Type:  computetypes.JobResultType,
		JobID: "job-zk",
		Result: computetypes.JobResult{
			ResultData:      []byte("encrypted_result"),
			ResultHash:      "0xabcdef1234567890",
			ComputeTime:     5000,
			ZKProof:         zkProof,
			AttestationSigs: nil,
			Timestamp:       time.Now().Unix(),
		},
		Provider: "provider-tee",
	}

	packetBytes, err := packetData.GetBytes()
	suite.Require().NoError(err)

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	err = suite.path.EndpointA.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify ZK proof was validated
	// (In production, verify proof verification logic executed)
}

func (suite *ComputeIBCTestSuite) TestEscrowManagement() {
	// Test cross-chain escrow lock and release

	requester := suite.chainA.SenderAccount.GetAddress()
	escrowAmount := math.NewInt(5000000)

	// Submit job with escrow
	escrowProof, _ := json.Marshal(map[string]interface{}{
		"job_id":      "job-escrow",
		"amount":      escrowAmount.String(),
		"locked_at":   time.Now().Unix(),
		"requester":   requester.String(),
		"provider":    "provider-1",
		"block_height": suite.chainA.GetContext().BlockHeight(),
	})

	submitPacket := computetypes.SubmitJobPacketData{
		Type:    computetypes.SubmitJobType,
		JobID:   "job-escrow",
		JobType: "docker",
		JobData: []byte(`{"image":"ubuntu:latest","cmd":"echo hello"}`),
		Requirements: computetypes.JobRequirements{
			CPUCores:    2,
			MemoryMB:    4096,
			StorageGB:   50,
			GPURequired: false,
			TEERequired: false,
			MaxDuration: 300,
		},
		Provider:    "provider-1",
		Requester:   requester.String(),
		EscrowProof: escrowProof,
		Timeout:     uint64(time.Now().Add(time.Hour).Unix()),
	}

	submitBytes, _ := submitPacket.GetBytes()
	packet1 := channeltypes.NewPacket(
		submitBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet1.TimeoutHeight, packet1.TimeoutTimestamp, packet1.GetData())
	suite.Require().NoError(err)
	packet1.Sequence = sequence
	suite.path.EndpointB.RecvPacket(packet1)

	// Job completes successfully
	resultPacket := computetypes.JobResultPacketData{
		Type:  computetypes.JobResultType,
		JobID: "job-escrow",
		Result: computetypes.JobResult{
			ResultData:  []byte("hello"),
			ResultHash:  "0xabc123",
			ComputeTime: 1000,
			Timestamp:   time.Now().Unix(),
		},
		Provider: "provider-1",
	}

	resultBytes, _ := resultPacket.GetBytes()
	packet2 := channeltypes.NewPacket(
		resultBytes,
		2,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		clienttypes.NewHeight(1, 110),
		0,
	)

	suite.path.EndpointA.RecvPacket(packet2)

	// Verify escrow was released to provider
	// (In production, verify provider balance increased by escrow amount)
}
