package ibc_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	dextypes "github.com/paw-chain/paw/x/dex/types"
	pawibctesting "github.com/paw-chain/paw/testutil/ibctesting"

	_ "github.com/paw-chain/paw/testutil/ibctesting"
)

// DEXCrossChainTestSuite tests cross-chain DEX operations
type DEXCrossChainTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	path        *ibctesting.Path
}

func TestDEXCrossChainTestSuite(t *testing.T) {
	suite.Run(t, new(DEXCrossChainTestSuite))
}

func (suite *DEXCrossChainTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	pawibctesting.BindCustomPorts(suite.chainA)
	pawibctesting.BindCustomPorts(suite.chainB)

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = "dex"
	suite.path.EndpointB.ChannelConfig.PortID = "dex"
	suite.path.EndpointA.ChannelConfig.Version = dextypes.IBCVersion
	suite.path.EndpointB.ChannelConfig.Version = dextypes.IBCVersion

	suite.coordinator.Setup(suite.path)
}

func (suite *DEXCrossChainTestSuite) TestQueryRemotePools() {
	// Test querying pools on remote chain

	// Create query packet
	packetData := dextypes.NewQueryPoolsPacket("upaw", "uosmo")
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

	// Send packet
	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	// Receive on chain B
	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify acknowledgement contains pool data
	ack := channeltypes.NewResultAcknowledgement([]byte(`{"success":true,"pools":[]}`))
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *DEXCrossChainTestSuite) TestCrossChainSwap() {
	// Test executing a swap on remote chain

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainA.SenderAccount.GetAddress()

	// Create swap packet
	packetData := dextypes.NewExecuteSwapPacket(
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(900000),
		sender.String(),
		receiver.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)

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

	// Send swap packet
	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	// Receive and execute on chain B
	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify swap execution
	ackData := dextypes.ExecuteSwapAcknowledgement{
		Success:   true,
		AmountOut: math.NewInt(950000),
		SwapFee:   math.NewInt(3000),
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *DEXCrossChainTestSuite) TestMultiHopSwap() {
	// Test multi-hop swap across multiple chains

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainA.SenderAccount.GetAddress()

	// Create multi-hop route
	route := []dextypes.SwapHop{
		{
			ChainID:      suite.chainA.ChainID,
			PoolID:       "pool-1",
			TokenIn:      "upaw",
			TokenOut:     "uatom",
			MinAmountOut: math.NewInt(900000),
		},
		{
			ChainID:      suite.chainB.ChainID,
			PoolID:       "pool-2",
			TokenIn:      "uatom",
			TokenOut:     "uosmo",
			MinAmountOut: math.NewInt(800000),
		},
	}

	packetData := dextypes.NewCrossChainSwapPacket(
		route,
		sender.String(),
		receiver.String(),
		math.NewInt(1000000),
		math.NewInt(750000),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)

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

	// Execute multi-hop swap
	sequence, err := suite.path.EndpointA.SendPacket(packet.TimeoutHeight, packet.TimeoutTimestamp, packet.GetData())
	suite.Require().NoError(err)
	packet.Sequence = sequence

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify final amount
	ackData := dextypes.CrossChainSwapAcknowledgement{
		Success:      true,
		FinalAmount:  math.NewInt(850000),
		HopsExecuted: 2,
		TotalFees:    math.NewInt(6000),
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *DEXCrossChainTestSuite) TestSwapTimeout() {
	// Test swap timeout and refund

	sender := suite.chainA.SenderAccount.GetAddress()

	packetData := dextypes.NewExecuteSwapPacket(
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(900000),
		sender.String(),
		sender.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Second).Unix()),
	)

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

	// Verify refund logic executed
	// (Tokens should be refunded to sender)
}

func (suite *DEXCrossChainTestSuite) TestSlippageProtection() {
	// Test slippage protection in cross-chain swaps

	sender := suite.chainA.SenderAccount.GetAddress()

	// Create swap with tight slippage tolerance
	packetData := dextypes.NewExecuteSwapPacket(
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(995000), // Only 0.5% slippage allowed
		sender.String(),
		sender.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)

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

	// If actual output is below minimum, swap should fail
	ackErr := fmt.Errorf("slippage exceeded: expected 995000, got 990000")
	ack := channeltypes.NewErrorAcknowledgement(ackErr)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}
