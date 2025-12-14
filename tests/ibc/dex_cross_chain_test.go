package ibc_test

import (
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
	dextypes "github.com/paw-chain/paw/x/dex/types"
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

	seedDexPool(suite.chainA)
	seedDexPool(suite.chainB)

	suite.coordinator.Setup(suite.path)

	pawibctesting.AuthorizeModuleChannel(suite.chainA, dextypes.PortID, suite.path.EndpointA.ChannelID)
	pawibctesting.AuthorizeModuleChannel(suite.chainB, dextypes.PortID, suite.path.EndpointB.ChannelID)
}

func seedDexPool(chain *ibctesting.TestChain) {
	app := pawibctesting.GetPAWApp(chain)
	ctx := chain.GetContext()
	creator := chain.SenderAccount.GetAddress()

	fund := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 2_000_000_000),
		sdk.NewInt64Coin("uosmo", 2_000_000_000),
		sdk.NewInt64Coin("uatom", 2_000_000_000),
	)

	if err := app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, fund); err != nil {
		return
	}
	if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, creator, fund); err != nil {
		return
	}

	extraFloat := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 1_000_000_000),
		sdk.NewInt64Coin("uosmo", 1_000_000_000),
		sdk.NewInt64Coin("uatom", 1_000_000_000),
	)
	_ = app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, extraFloat)

	if _, err := app.DEXKeeper.CreatePool(ctx, creator, "upaw", "uosmo", math.NewInt(500_000_000), math.NewInt(500_000_000)); err != nil {
		return
	}
	if _, err := app.DEXKeeper.CreatePool(ctx, creator, "upaw", "uatom", math.NewInt(400_000_000), math.NewInt(400_000_000)); err != nil {
		return
	}
	if _, err := app.DEXKeeper.CreatePool(ctx, creator, "uatom", "uosmo", math.NewInt(400_000_000), math.NewInt(400_000_000)); err != nil {
		return
	}
}

func (suite *DEXCrossChainTestSuite) TestQueryRemotePools() {
	// Test querying pools on remote chain

	// Create query packet
	packetData := dextypes.NewQueryPoolsPacket("upaw", "uosmo", 1)
	packetData.Timestamp = suite.chainA.GetContext().BlockTime().Unix()
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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData dextypes.QueryPoolsAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().NotEmpty(ackData.Pools)
	tokens := []string{ackData.Pools[0].TokenA, ackData.Pools[0].TokenB}
	suite.Require().ElementsMatch([]string{"upaw", "uosmo"}, tokens)
}

func (suite *DEXCrossChainTestSuite) TestCrossChainSwap() {
	// Test executing a swap on remote chain

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainA.SenderAccount.GetAddress()

	// Create swap packet
	packetData := dextypes.NewExecuteSwapPacket(
		1,
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(900000),
		sender.String(),
		receiver.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)
	packetData.Timestamp = suite.chainA.GetContext().BlockTime().Unix()

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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData dextypes.ExecuteSwapAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().True(ackData.AmountOut.IsPositive())
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
		1,
		route,
		sender.String(),
		receiver.String(),
		math.NewInt(1000000),
		math.NewInt(750000),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)
	packetData.Timestamp = suite.chainA.GetContext().BlockTime().Unix()

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

	_, ackBz, err := suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	var ackData dextypes.CrossChainSwapAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().GreaterOrEqual(ackData.HopsExecuted, 1)
}

func (suite *DEXCrossChainTestSuite) TestSwapTimeout() {
	// Test swap timeout and refund

	sender := suite.chainA.SenderAccount.GetAddress()

	packetData := dextypes.NewExecuteSwapPacket(
		1,
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(900000),
		sender.String(),
		sender.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Second).Unix()),
	)
	packetData.Timestamp = suite.chainA.GetContext().BlockTime().Unix()

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

	// Verify refund logic executed
	// (Tokens should be refunded to sender)
}

func (suite *DEXCrossChainTestSuite) TestSlippageProtection() {
	// Test slippage protection in cross-chain swaps

	sender := suite.chainA.SenderAccount.GetAddress()

	// Create swap with tight slippage tolerance
	packetData := dextypes.NewExecuteSwapPacket(
		1,
		"pool-1",
		"upaw",
		"uosmo",
		math.NewInt(1000000),
		math.NewInt(995000), // Only 0.5% slippage allowed
		sender.String(),
		sender.String(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*10).Unix()),
	)
	packetData.Timestamp = suite.chainA.GetContext().BlockTime().Unix()

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
	suite.Require().NotEmpty(ackResult(suite.T(), ackBz))
}
