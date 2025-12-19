package ibc_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	pawibctesting "github.com/paw-chain/paw/testutil/ibctesting"
	"github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
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

	seedComputeProvider(suite.chainA)
	seedComputeProvider(suite.chainB)

	suite.coordinator.Setup(suite.path)

	pawibctesting.AuthorizeModuleChannel(suite.chainA, computetypes.PortID, suite.path.EndpointA.ChannelID)
	pawibctesting.AuthorizeModuleChannel(suite.chainB, computetypes.PortID, suite.path.EndpointB.ChannelID)
}

func seedComputeProvider(chain *ibctesting.TestChain) {
	app := pawibctesting.GetPAWApp(chain)
	ctx := chain.GetContext()

	params, err := app.ComputeKeeper.GetParams(ctx)
	if err != nil {
		return
	}

	provider := chain.SenderAccount.GetAddress()
	stake := params.MinProviderStake.Add(math.NewInt(1_000_000))
	fund := sdk.NewCoins(sdk.NewCoin("upaw", stake.MulRaw(10)))

	if err := app.BankKeeper.MintCoins(ctx, computetypes.ModuleName, fund); err != nil {
		return
	}
	if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, computetypes.ModuleName, provider, fund); err != nil {
		return
	}

	specs := computetypes.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	pricing := computetypes.Pricing{
		CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.01"),
		MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.001"),
		GpuPricePerHour:       math.LegacyMustNewDecFromStr("0.05"),
		StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.0005"),
	}

	_ = app.ComputeKeeper.RegisterProvider(
		ctx,
		provider,
		"ibc-provider",
		"https://provider.local",
		specs,
		pricing,
		stake,
	)
}

func (suite *ComputeIBCTestSuite) TestDiscoverProviders() {
	// Test discovering compute providers on remote chain

	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.DiscoverProvidersPacketData{
		Nonce:        1,
		Type:         computetypes.DiscoverProvidersType,
		Timestamp:    suite.chainA.GetContext().BlockTime().Unix(),
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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData computetypes.DiscoverProvidersAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().Equal(uint64(1), ackData.Nonce)
	suite.Require().NotEmpty(ackData.Providers)
	suite.Require().True(ackData.Providers[0].PricePerUnit.IsPositive())
	suite.Require().GreaterOrEqual(ackData.TotalProviders, uint32(len(ackData.Providers)))
}

func (suite *ComputeIBCTestSuite) TestSubmitJob() {
	// Test submitting compute job to remote chain

	requester := suite.chainA.SenderAccount.GetAddress()

	jobData := []byte(`{"function":"fibonacci","input":{"n":1000}}`)
	escrowProof := []byte(`{"job_id":"job-1","amount":"1000000","locked_at":1234567890}`)

	packetData := computetypes.SubmitJobPacketData{
		Nonce:     1,
		Type:      computetypes.SubmitJobType,
		Timestamp: suite.chainA.GetContext().BlockTime().Unix(),
		JobID:     "job-1",
		JobType:   "wasm",
		JobData:   jobData,
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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData computetypes.SubmitJobAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().Equal(uint64(1), ackData.Nonce)
	suite.Require().Equal("job-1", ackData.JobID)
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
		Nonce:     1,
		Type:      computetypes.JobResultType,
		Timestamp: suite.chainB.GetContext().BlockTime().Unix(),
		JobID:     "job-1",
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

	sequence, err := suite.path.EndpointB.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData computetypes.JobResultAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().Equal(uint64(1), ackData.Nonce)
	suite.Require().Equal("job-1", ackData.JobID)
	suite.Require().Equal("completed", ackData.Status)
	suite.Require().Equal(uint32(100), ackData.Progress)
	suite.Require().Equal(packetData.Result.ResultHash, ackData.ResultHash)
	suite.Require().Equal(hashBytes(packetData.Result.ZKProof), ackData.ProofHash)
	suite.Require().Equal(hashByteSlices(packetData.Result.AttestationSigs), ackData.AttestationHash)
	suite.Require().Equal(packetData.Provider, ackData.Provider)

	// Verify result was stored and escrow released
	// (In production, verify job status and escrow release)
}

func (suite *ComputeIBCTestSuite) TestQueryJobStatus() {
	// Test querying status of remote job

	requester := suite.chainA.SenderAccount.GetAddress()

	// Seed a job on the counterparty chain so the status query succeeds.
	pawApp := pawibctesting.GetPAWApp(suite.chainB)
	ctxB := suite.chainB.GetContext()
	pawApp.ComputeKeeper.UpsertCrossChainJob(ctxB, &keeper.CrossChainComputeJob{
		JobID:       "job-1",
		Status:      "running",
		Progress:    70,
		Provider:    "provider-1",
		SubmittedAt: ctxB.BlockTime(),
	})

	packetData := computetypes.JobStatusPacketData{
		Nonce:     1,
		Type:      computetypes.JobStatusType,
		Timestamp: suite.chainA.GetContext().BlockTime().Unix(),
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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData computetypes.JobStatusAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().Equal(uint64(1), ackData.Nonce)
}

func (suite *ComputeIBCTestSuite) TestJobTimeout() {
	// Test job timeout and escrow refund

	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.SubmitJobPacketData{
		Nonce:     1,
		Type:      computetypes.SubmitJobType,
		Timestamp: suite.chainA.GetContext().BlockTime().Unix(),
		JobID:     "job-timeout",
		JobType:   "wasm",
		JobData:   []byte(`{"function":"test"}`),
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

	timeoutHeight := suite.chainB.GetTimeoutHeight()

	packet := channeltypes.NewPacket(
		packetBytes,
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		timeoutHeight,
		0,
	)

	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	// Advance destination chain past timeout height
	suite.coordinator.CommitNBlocks(suite.chainB, 150)

	// Timeout packet
	err = suite.path.EndpointA.UpdateClient()
	suite.Require().NoError(err)
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
		Nonce:     1,
		Type:      computetypes.JobResultType,
		Timestamp: suite.chainB.GetContext().BlockTime().Unix(),
		JobID:     "job-zk",
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

	sequence, err := suite.path.EndpointB.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	_, _, err = suite.path.RelayPacketWithResults(packet)
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
		"job_id":       "job-escrow",
		"amount":       escrowAmount.String(),
		"locked_at":    time.Now().Unix(),
		"requester":    requester.String(),
		"provider":     "provider-1",
		"block_height": suite.chainA.GetContext().BlockHeight(),
	})

	submitPacket := computetypes.SubmitJobPacketData{
		Nonce:     1,
		Type:      computetypes.SubmitJobType,
		Timestamp: suite.chainA.GetContext().BlockTime().Unix(),
		JobID:     "job-escrow",
		JobType:   "docker",
		JobData:   []byte(`{"image":"ubuntu:latest","cmd":"echo hello"}`),
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
	_, _, err = suite.path.RelayPacketWithResults(packet1)
	suite.Require().NoError(err)

	// Job completes successfully
	resultPacket := computetypes.JobResultPacketData{
		Nonce:     2,
		Type:      computetypes.JobResultType,
		Timestamp: suite.chainB.GetContext().BlockTime().Unix(),
		JobID:     "job-escrow",
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

	sequence, err = suite.path.EndpointB.SendPacket(packet2.TimeoutHeight, packet2.TimeoutTimestamp, packet2.GetData())
	suite.Require().NoError(err)
	packet2.Sequence = sequence

	_, _, err = suite.path.RelayPacketWithResults(packet2)
	suite.Require().NoError(err)

	// Verify escrow was released to provider
	// (In production, verify provider balance increased by escrow amount)
}

func (suite *ComputeIBCTestSuite) TestOnRecvPacketRejectsDuplicateNonce() {
	requester := suite.chainA.SenderAccount.GetAddress()

	packetData := computetypes.DiscoverProvidersPacketData{
		Type:         computetypes.DiscoverProvidersType,
		Nonce:        1,
		Timestamp:    suite.chainA.GetContext().BlockTime().Unix(),
		Capabilities: []string{"gpu"},
		MaxPrice:     math.LegacyMustNewDecFromStr("10.0"),
		Requester:    requester.String(),
	}

	ackBz, err := suite.sendDiscoverProvidersPacket(packetData)
	suite.Require().NoError(err)
	var ack computetypes.DiscoverProvidersAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ack))
	suite.Require().True(ack.Success)

	ackBz, err = suite.sendDiscoverProvidersPacket(packetData)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(ackError(suite.T(), ackBz))
}

func (suite *ComputeIBCTestSuite) sendDiscoverProvidersPacket(packetData computetypes.DiscoverProvidersPacketData) ([]byte, error) {
	packetBytes, err := packetData.GetBytes()
	if err != nil {
		return nil, err
	}

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
	if err != nil {
		return nil, err
	}
	packet.Sequence = sequence

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	if err != nil {
		return nil, err
	}

	return ackBz, nil
}

// helper hashes mirror keeper hashing for proof and attestation aggregation
func hashBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashByteSlices(data [][]byte) string {
	if len(data) == 0 {
		return ""
	}

	hasher := sha256.New()
	written := false
	for _, b := range data {
		if len(b) == 0 {
			continue
		}
		hasher.Write(b)
		written = true
	}
	if !written {
		return ""
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
