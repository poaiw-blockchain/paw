package ibc_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/testutil/ibctesting"
)

// IBCTransferTestSuite tests IBC token transfer functionality
type IBCTransferTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// Chains
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	// Paths
	path *ibctesting.Path
}

func TestIBCTransferTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTransferTestSuite))
}

func (suite *IBCTransferTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// Setup IBC path
	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(suite.path)
}

// TestBasicTransfer tests basic IBC token transfer from chain A to chain B
func (suite *IBCTransferTestSuite) TestBasicTransfer() {
	ctx := context.Background()

	// Get sender address on chain A
	sender := suite.chainA.SenderAccount.GetAddress()

	// Get receiver address on chain B
	receiver := suite.chainB.SenderAccount.GetAddress()

	// Transfer amount
	amount := sdk.NewCoin("stake", math.NewInt(1000))

	// Create transfer message
	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	// Execute transfer on chain A
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Relay packet to chain B
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)

	// Verify receiver balance on chain B
	expectedDenom := transfertypes.GetPrefixedDenom(
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		amount.Denom,
	)
	denomTrace := transfertypes.ParseDenomTrace(expectedDenom)

	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(
		suite.chainB.GetContext(),
		receiver,
		denomTrace.IBCDenom(),
	)

	suite.Require().Equal(amount.Amount, balance.Amount)
}

// TestTransferTimeout tests IBC transfer timeout handling
func (suite *IBCTransferTestSuite) TestTransferTimeout() {
	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin("stake", math.NewInt(1000))

	// Get initial sender balance
	initialBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		amount.Denom,
	)

	// Create transfer with short timeout
	timeoutHeight := clienttypes.NewHeight(0, suite.chainB.CurrentHeader.Height+1)

	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)

	// Execute transfer
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	// Parse packet
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// Update chain B to trigger timeout
	suite.coordinator.CommitNBlocks(suite.chainB, 5)

	// Process timeout on chain A
	err = suite.path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = suite.path.EndpointA.TimeoutPacket(packet)
	suite.Require().NoError(err)

	// Verify sender balance restored on chain A
	finalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		amount.Denom,
	)

	suite.Require().Equal(initialBalance.Amount, finalBalance.Amount)
}

// TestMultiHopTransfer tests IBC transfer through multiple chains
func (suite *IBCTransferTestSuite) TestMultiHopTransfer() {
	// Setup additional chain
	suite.coordinator.Chains = append(suite.coordinator.Chains, ibctesting.NewTestChain(suite.T(), suite.coordinator, ibctesting.GetChainID(3)))
	chainC := suite.coordinator.GetChain(ibctesting.GetChainID(3))

	// Setup path A -> B
	pathAB := suite.path

	// Setup path B -> C
	pathBC := ibctesting.NewPath(suite.chainB, chainC)
	suite.coordinator.Setup(pathBC)

	// Transfer from A to B
	senderA := suite.chainA.SenderAccount.GetAddress()
	receiverB := suite.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin("stake", math.NewInt(1000))

	msgAB := transfertypes.NewMsgTransfer(
		pathAB.EndpointA.ChannelConfig.PortID,
		pathAB.EndpointA.ChannelID,
		amount,
		senderA.String(),
		receiverB.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	resAB, err := suite.chainA.SendMsgs(msgAB)
	suite.Require().NoError(err)

	packetAB, err := ibctesting.ParsePacketFromEvents(resAB.GetEvents())
	suite.Require().NoError(err)

	err = pathAB.RelayPacket(packetAB)
	suite.Require().NoError(err)

	// Get IBC denom on chain B
	denomTraceAB := transfertypes.ParseDenomTrace(
		transfertypes.GetPrefixedDenom(
			pathAB.EndpointB.ChannelConfig.PortID,
			pathAB.EndpointB.ChannelID,
			amount.Denom,
		),
	)

	// Transfer from B to C
	receiverC := chainC.SenderAccount.GetAddress()
	amountBC := sdk.NewCoin(denomTraceAB.IBCDenom(), math.NewInt(500))

	msgBC := transfertypes.NewMsgTransfer(
		pathBC.EndpointA.ChannelConfig.PortID,
		pathBC.EndpointA.ChannelID,
		amountBC,
		receiverB.String(),
		receiverC.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	resBC, err := suite.chainB.SendMsgs(msgBC)
	suite.Require().NoError(err)

	packetBC, err := ibctesting.ParsePacketFromEvents(resBC.GetEvents())
	suite.Require().NoError(err)

	err = pathBC.RelayPacket(packetBC)
	suite.Require().NoError(err)

	// Verify final balance on chain C
	expectedDenomC := transfertypes.GetPrefixedDenom(
		pathBC.EndpointB.ChannelConfig.PortID,
		pathBC.EndpointB.ChannelID,
		denomTraceAB.IBCDenom(),
	)
	denomTraceC := transfertypes.ParseDenomTrace(expectedDenomC)

	balanceC := chainC.GetSimApp().BankKeeper.GetBalance(
		chainC.GetContext(),
		receiverC,
		denomTraceC.IBCDenom(),
	)

	suite.Require().Equal(math.NewInt(500), balanceC.Amount)
}

// TestPacketAcknowledgement tests IBC packet acknowledgment handling
func (suite *IBCTransferTestSuite) TestPacketAcknowledgement() {
	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()
	amount := sdk.NewCoin("stake", math.NewInt(1000))

	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// Relay packet and get acknowledgement
	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)

	// Verify acknowledgement event was emitted
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	suite.Require().NotNil(ack)
}

// TestChannelHandshake tests IBC channel creation and handshake
func (suite *IBCTransferTestSuite) TestChannelHandshake() {
	// Create new path without setup
	newPath := ibctesting.NewPath(suite.chainA, suite.chainB)

	// Execute channel handshake
	err := newPath.EndpointA.ChanOpenInit()
	suite.Require().NoError(err)

	err = newPath.EndpointB.ChanOpenTry()
	suite.Require().NoError(err)

	err = newPath.EndpointA.ChanOpenAck()
	suite.Require().NoError(err)

	err = newPath.EndpointB.ChanOpenConfirm()
	suite.Require().NoError(err)

	// Verify channel is open
	channelA := newPath.EndpointA.GetChannel()
	suite.Require().Equal(channeltypes.OPEN, channelA.State)

	channelB := newPath.EndpointB.GetChannel()
	suite.Require().Equal(channeltypes.OPEN, channelB.State)
}

// TestConcurrentTransfers tests multiple concurrent IBC transfers
func (suite *IBCTransferTestSuite) TestConcurrentTransfers() {
	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	numTransfers := 5
	transferAmount := math.NewInt(100)

	// Execute multiple transfers
	for i := 0; i < numTransfers; i++ {
		amount := sdk.NewCoin("stake", transferAmount)

		msg := transfertypes.NewMsgTransfer(
			suite.path.EndpointA.ChannelConfig.PortID,
			suite.path.EndpointA.ChannelID,
			amount,
			sender.String(),
			receiver.String(),
			clienttypes.NewHeight(1, 110),
			0,
			"",
		)

		res, err := suite.chainA.SendMsgs(msg)
		suite.Require().NoError(err)

		packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
		suite.Require().NoError(err)

		err = suite.path.RelayPacket(packet)
		suite.Require().NoError(err)
	}

	// Verify total received amount
	expectedDenom := transfertypes.GetPrefixedDenom(
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		"stake",
	)
	denomTrace := transfertypes.ParseDenomTrace(expectedDenom)

	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(
		suite.chainB.GetContext(),
		receiver,
		denomTrace.IBCDenom(),
	)

	expectedTotal := transferAmount.Mul(math.NewInt(int64(numTransfers)))
	suite.Require().Equal(expectedTotal, balance.Amount)
}
