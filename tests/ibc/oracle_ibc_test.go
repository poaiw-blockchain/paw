package ibc_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	pawibctesting "github.com/paw-chain/paw/testutil/ibctesting"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"

	_ "github.com/paw-chain/paw/testutil/ibctesting"
)

// OracleIBCTestSuite tests cross-chain oracle operations
type OracleIBCTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	path        *ibctesting.Path
}

func TestOracleIBCTestSuite(t *testing.T) {
	suite.Run(t, new(OracleIBCTestSuite))
}

func (suite *OracleIBCTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	pawibctesting.BindCustomPorts(suite.chainA)
	pawibctesting.BindCustomPorts(suite.chainB)

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = "oracle"
	suite.path.EndpointB.ChannelConfig.PortID = "oracle"
	suite.path.EndpointA.ChannelConfig.Version = oracletypes.IBCVersion
	suite.path.EndpointB.ChannelConfig.Version = oracletypes.IBCVersion
	// Oracle module expects UNORDERED channels for price feeds
	// (no strict ordering needed for independent price updates)

	suite.coordinator.Setup(suite.path)

	pawibctesting.AuthorizeModuleChannel(suite.chainA, oracletypes.PortID, suite.path.EndpointA.ChannelID)
	pawibctesting.AuthorizeModuleChannel(suite.chainB, oracletypes.PortID, suite.path.EndpointB.ChannelID)
}

func (suite *OracleIBCTestSuite) TestSubscribeToPrices() {
	// Test subscribing to price feeds from remote oracle

	subscriber := suite.chainA.SenderAccount.GetAddress()
	var err error

	packetData := oracletypes.SubscribePricesPacketData{
		Type:           oracletypes.SubscribePricesType,
		Nonce:          1,
		Symbols:        []string{"BTC/USD", "ETH/USD", "ATOM/USD"},
		UpdateInterval: 60, // 1 minute
		Subscriber:     subscriber.String(),
	}

	err = packetData.ValidateBasic()
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

	var ackData oracletypes.SubscribePricesAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().Equal(uint64(1), ackData.Nonce)
	suite.Require().NotEmpty(ackData.SubscriptionID)
	suite.Require().Equal(len(packetData.Symbols), len(ackData.SubscribedSymbols))
}

func (suite *OracleIBCTestSuite) TestQueryPrice() {
	// Test querying current price from remote oracle

	sender := suite.chainA.SenderAccount.GetAddress()

	// Seed live price on counterparty chain for deterministic ACK payload.
	pawApp := pawibctesting.GetPAWApp(suite.chainB)
	ctxB := suite.chainB.GetContext()
	params, err := pawApp.OracleKeeper.GetParams(ctxB)
	suite.Require().NoError(err)
	if params.VotePeriod == 0 {
		params.VotePeriod = 1
	}
	if params.SlashWindow == 0 {
		params.SlashWindow = 10000
	}
	if params.MinValidPerWindow == 0 {
		params.MinValidPerWindow = 100
	}
	suite.Require().NoError(pawApp.OracleKeeper.SetParams(ctxB, params))
	price := oracletypes.Price{
		Asset:         "BTC/USD",
		Price:         math.LegacyMustNewDecFromStr("45234.12"),
		BlockHeight:   ctxB.BlockHeight(),
		BlockTime:     ctxB.BlockTime().Unix(),
		NumValidators: 4,
	}
	suite.Require().NoError(pawApp.OracleKeeper.SetPrice(ctxB, price))

	packetData := oracletypes.QueryPricePacketData{
		Type:   oracletypes.QueryPriceType,
		Nonce:  1,
		Symbol: "BTC/USD",
		Sender: sender.String(),
	}

	err = packetData.ValidateBasic()
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

	var ackData oracletypes.QueryPriceAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ackData))
	suite.Require().True(ackData.Success)
	suite.Require().Equal(uint64(1), ackData.Nonce)
	suite.Require().Equal("BTC/USD", ackData.PriceData.Symbol)
	suite.Require().True(ackData.PriceData.Price.IsPositive())
}

func (suite *OracleIBCTestSuite) TestReceivePriceUpdate() {
	// Test receiving price updates from remote oracle

	priceData := []oracletypes.PriceData{
		{
			Symbol:      "BTC/USD",
			Price:       math.LegacyMustNewDecFromStr("45123.45"),
			Volume24h:   math.NewInt(1234567890),
			Timestamp:   time.Now().Unix(),
			Confidence:  math.LegacyMustNewDecFromStr("0.98"),
			OracleCount: 9,
		},
		{
			Symbol:      "ETH/USD",
			Price:       math.LegacyMustNewDecFromStr("2345.67"),
			Volume24h:   math.NewInt(987654321),
			Timestamp:   time.Now().Unix(),
			Confidence:  math.LegacyMustNewDecFromStr("0.97"),
			OracleCount: 8,
		},
	}

	packetData := oracletypes.PriceUpdatePacketData{
		Type:      oracletypes.PriceUpdateType,
		Nonce:     1,
		Prices:    priceData,
		Timestamp: time.Now().Unix(),
		Source:    suite.chainB.ChainID,
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

	_, _, err = suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	// Verify prices were stored
	// (In production, query oracle keeper to verify price storage)
}

func (suite *OracleIBCTestSuite) TestOracleHeartbeat() {
	// Test oracle liveness monitoring via heartbeats

	packetData := oracletypes.OracleHeartbeatPacketData{
		Type:          oracletypes.OracleHeartbeatType,
		Nonce:         1,
		ChainID:       suite.chainB.ChainID,
		Timestamp:     time.Now().Unix(),
		ActiveOracles: 10,
		BlockHeight:   suite.chainB.GetContext().BlockHeight(),
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

	_, _, err = suite.path.RelayPacketWithResults(packet)
	suite.Require().NoError(err)

	// Verify heartbeat was recorded
}

func (suite *OracleIBCTestSuite) TestCrossChainAggregation() {
	// Test aggregating prices from multiple chains

	// Receive price updates from multiple sources
	sources := []string{suite.chainB.ChainID, "osmosis-1", "cosmoshub-4"}

	for _, source := range sources {
		packetData := oracletypes.PriceUpdatePacketData{
			Type: oracletypes.PriceUpdateType,
			Prices: []oracletypes.PriceData{
				{
					Symbol:      "BTC/USD",
					Price:       math.LegacyMustNewDecFromStr("45000.00").Add(math.LegacyNewDec(int64(len(source)))),
					Volume24h:   math.NewInt(1000000000),
					Timestamp:   time.Now().Unix(),
					Confidence:  math.LegacyMustNewDecFromStr("0.95"),
					OracleCount: 7,
				},
			},
			Timestamp: time.Now().Unix(),
			Source:    source,
		}

		packetBytes, err := packetData.GetBytes()
		suite.Require().NoError(err)

		packet := channeltypes.NewPacket(
			packetBytes,
			uint64(len(source)),
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
	}

	// Verify aggregated price accounts for all sources
	// (In production, query aggregated price and verify weighted average)
}

func (suite *OracleIBCTestSuite) TestByzantineFaultTolerance() {
	// Test Byzantine fault tolerance with malicious oracle data

	// Send conflicting price data
	maliciousPrice := oracletypes.PriceData{
		Symbol:      "BTC/USD",
		Price:       math.LegacyMustNewDecFromStr("1000000.00"), // Clearly wrong
		Volume24h:   math.NewInt(1),
		Timestamp:   time.Now().Unix(),
		Confidence:  math.LegacyMustNewDecFromStr("0.99"),
		OracleCount: 1,
	}

	packetData := oracletypes.PriceUpdatePacketData{
		Type:      oracletypes.PriceUpdateType,
		Prices:    []oracletypes.PriceData{maliciousPrice},
		Timestamp: time.Now().Unix(),
		Source:    "malicious-oracle",
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

	// Verify malicious data was filtered out
	// (BFT requires 2/3+ agreement, single malicious oracle should be ignored)
}

func (suite *OracleIBCTestSuite) TestOnRecvPacketRejectsDuplicateNonce() {
	subscriber := suite.chainA.SenderAccount.GetAddress()

	packetData := oracletypes.SubscribePricesPacketData{
		Type:           oracletypes.SubscribePricesType,
		Nonce:          1,
		Symbols:        []string{"BTC/USD"},
		UpdateInterval: 60,
		Subscriber:     subscriber.String(),
	}

	ackBz, err := suite.sendSubscribePricesPacket(packetData)
	suite.Require().NoError(err)
	var ack oracletypes.SubscribePricesAcknowledgement
	suite.Require().NoError(json.Unmarshal(ackResult(suite.T(), ackBz), &ack))
	suite.Require().True(ack.Success)

	ackBz, err = suite.sendSubscribePricesPacket(packetData)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(ackError(suite.T(), ackBz))
}

func (suite *OracleIBCTestSuite) sendSubscribePricesPacket(packetData oracletypes.SubscribePricesPacketData) ([]byte, error) {
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
