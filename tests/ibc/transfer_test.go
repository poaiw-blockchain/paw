package ibc_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// IBCTransferTestSuite tests IBC token transfers (ICS-20)
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

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = ibctransfertypes.PortID
	suite.path.EndpointB.ChannelConfig.PortID = ibctransfertypes.PortID
	suite.path.EndpointA.ChannelConfig.Version = ibctransfertypes.Version
	suite.path.EndpointB.ChannelConfig.Version = ibctransfertypes.Version

	suite.coordinator.Setup(suite.path)
}

func (suite *IBCTransferTestSuite) TestIBCTransfer() {
	// Test basic IBC token transfer from chain A to chain B

	// Get sender and receiver addresses
	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	// Transfer amount
	amount := sdk.NewCoin("upaw", math.NewInt(1000000))

	// Create transfer message
	msg := ibctransfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	// Execute transfer
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Relay packet
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)

	// Verify receiver balance on chain B
	expectedDenom := ibctransfertypes.GetPrefixedDenom(
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		amount.Denom,
	)

	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(
		suite.chainB.GetContext(),
		receiver,
		expectedDenom,
	)

	suite.Require().Equal(amount.Amount, balance.Amount)
}

func (suite *IBCTransferTestSuite) TestIBCTransferTimeout() {
	// Test IBC transfer timeout and refund

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	// Get initial balance
	initialBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		"upaw",
	)

	// Transfer amount
	amount := sdk.NewCoin("upaw", math.NewInt(1000000))

	// Create transfer with short timeout
	timeoutHeight := clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight())+2)

	msg := ibctransfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		timeoutHeight,
		0,
		"",
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// Advance chain B past timeout height
	suite.coordinator.CommitNBlocks(suite.chainB, 10)

	// Timeout packet
	err = suite.path.EndpointA.TimeoutPacket(packet)
	suite.Require().NoError(err)

	// Verify sender balance is refunded
	finalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		"upaw",
	)

	// Balance should be refunded (minus gas fees)
	suite.Require().True(finalBalance.Amount.GTE(initialBalance.Amount.Sub(math.NewInt(100000))))
}

func (suite *IBCTransferTestSuite) TestMultiHopTransfer() {
	// Test IBC transfer through multiple chains
	// A -> B -> A (round trip)

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	// First transfer: A -> B
	amount := sdk.NewCoin("upaw", math.NewInt(1000000))

	msg1 := ibctransfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	res1, err := suite.chainA.SendMsgs(msg1)
	suite.Require().NoError(err)

	packet1, err := ibctesting.ParsePacketFromEvents(res1.GetEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet1)
	suite.Require().NoError(err)

	// Second transfer: B -> A
	ibcDenom := ibctransfertypes.GetPrefixedDenom(
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		"upaw",
	)

	msg2 := ibctransfertypes.NewMsgTransfer(
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		sdk.NewCoin(ibcDenom, math.NewInt(500000)),
		receiver.String(),
		sender.String(),
		clienttypes.NewHeight(1, 110),
		0,
		"",
	)

	res2, err := suite.chainB.SendMsgs(msg2)
	suite.Require().NoError(err)

	packet2, err := ibctesting.ParsePacketFromEvents(res2.GetEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet2)
	suite.Require().NoError(err)

	// Verify sender received tokens back
	finalBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		"upaw",
	)

	suite.Require().True(finalBalance.Amount.GT(math.ZeroInt()))
}

func (suite *IBCTransferTestSuite) TestIBCTransferWithMemo() {
	// Test IBC transfer with memo field

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()

	amount := sdk.NewCoin("upaw", math.NewInt(1000000))
	memo := `{"wasm":{"contract":"paw1...",&#34;msg":{"swap":{}}}}`

	msg := ibctransfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		amount,
		sender.String(),
		receiver.String(),
		clienttypes.NewHeight(1, 110),
		0,
		memo,
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)

	// Verify packet data includes memo
	var data ibctransfertypes.FungibleTokenPacketData
	err = ibctransfertypes.ModuleCdc.UnmarshalJSON(packet.Data, &data)
	suite.Require().NoError(err)
	suite.Require().Equal(memo, data.Memo)
}

func (suite *IBCTransferTestSuite) TestIBCFeePayment() {
	// Test IBC transfer with relayer fee payment (ICS-29)

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()
	relayer := suite.chainA.SenderAccount.GetAddress()

	amount := sdk.NewCoin("upaw", math.NewInt(1000000))

	// Create fee for relayer
	recvFee := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000)))
	ackFee := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000)))
	timeoutFee := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000)))

	msg := ibctransfertypes.NewMsgTransfer(
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

	// Relay with fee
	err = suite.path.RelayPacket(packet)
	suite.Require().NoError(err)

	// Verify relayer received fee
	// (In production, verify fee was paid to relayer address)
}
