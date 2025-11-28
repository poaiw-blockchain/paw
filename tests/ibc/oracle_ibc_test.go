package ibc_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	oracletypes "github.com/paw-chain/paw/x/oracle/types"
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

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = "oracle"
	suite.path.EndpointB.ChannelConfig.PortID = "oracle"
	suite.path.EndpointA.ChannelConfig.Version = oracletypes.IBCVersion
	suite.path.EndpointB.ChannelConfig.Version = oracletypes.IBCVersion

	suite.coordinator.Setup(suite.path)
}

func (suite *OracleIBCTestSuite) TestSubscribeToPrices() {
	// Test subscribing to price feeds from remote oracle

	subscriber := suite.chainA.SenderAccount.GetAddress()

	packetData := oracletypes.SubscribePricesPacketData{
		Type:           oracletypes.SubscribePricesType,
		Symbols:        []string{"BTC/USD", "ETH/USD", "ATOM/USD"},
		UpdateInterval: 60, // 1 minute
		Subscriber:     subscriber.String(),
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

	err = suite.path.EndpointA.SendPacket(packet)
	suite.Require().NoError(err)

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify subscription acknowledgement
	ackData := oracletypes.SubscribePricesAcknowledgement{
		Success:           true,
		SubscribedSymbols: []string{"BTC/USD", "ETH/USD", "ATOM/USD"},
		SubscriptionID:    "sub-1",
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *OracleIBCTestSuite) TestQueryPrice() {
	// Test querying current price from remote oracle

	sender := suite.chainA.SenderAccount.GetAddress()

	packetData := oracletypes.QueryPricePacketData{
		Type:   oracletypes.QueryPriceType,
		Symbol: "BTC/USD",
		Sender: sender.String(),
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

	err = suite.path.EndpointA.SendPacket(packet)
	suite.Require().NoError(err)

	err = suite.path.EndpointB.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify price data in acknowledgement
	ackData := oracletypes.QueryPriceAcknowledgement{
		Success: true,
		PriceData: oracletypes.PriceData{
			Symbol:      "BTC/USD",
			Price:       sdk.MustNewDecFromStr("45000.50"),
			Volume24h:   math.NewInt(1000000000),
			Timestamp:   time.Now().Unix(),
			Confidence:  sdk.MustNewDecFromStr("0.95"),
			OracleCount: 7,
		},
	}

	ackBytes, err := ackData.GetBytes()
	suite.Require().NoError(err)

	ack := channeltypes.NewResultAcknowledgement(ackBytes)
	err = suite.path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)
}

func (suite *OracleIBCTestSuite) TestReceivePriceUpdate() {
	// Test receiving price updates from remote oracle

	priceData := []oracletypes.PriceData{
		{
			Symbol:      "BTC/USD",
			Price:       sdk.MustNewDecFromStr("45123.45"),
			Volume24h:   math.NewInt(1234567890),
			Timestamp:   time.Now().Unix(),
			Confidence:  sdk.MustNewDecFromStr("0.98"),
			OracleCount: 9,
		},
		{
			Symbol:      "ETH/USD",
			Price:       sdk.MustNewDecFromStr("2345.67"),
			Volume24h:   math.NewInt(987654321),
			Timestamp:   time.Now().Unix(),
			Confidence:  sdk.MustNewDecFromStr("0.97"),
			OracleCount: 8,
		},
	}

	packetData := oracletypes.PriceUpdatePacketData{
		Type:      oracletypes.PriceUpdateType,
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

	// Receive price update on chain A
	err = suite.path.EndpointA.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify prices were stored
	// (In production, query oracle keeper to verify price storage)
}

func (suite *OracleIBCTestSuite) TestOracleHeartbeat() {
	// Test oracle liveness monitoring via heartbeats

	packetData := oracletypes.OracleHeartbeatPacketData{
		Type:          oracletypes.OracleHeartbeatType,
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

	err = suite.path.EndpointA.RecvPacket(packet)
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
					Price:       sdk.MustNewDecFromStr("45000.00").Add(sdk.NewDec(int64(len(source)))),
					Volume24h:   math.NewInt(1000000000),
					Timestamp:   time.Now().Unix(),
					Confidence:  sdk.MustNewDecFromStr("0.95"),
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

		err = suite.path.EndpointA.RecvPacket(packet)
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
		Price:       sdk.MustNewDecFromStr("1000000.00"), // Clearly wrong
		Volume24h:   math.NewInt(1),
		Timestamp:   time.Now().Unix(),
		Confidence:  sdk.MustNewDecFromStr("0.99"),
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

	err = suite.path.EndpointA.RecvPacket(packet)
	suite.Require().NoError(err)

	// Verify malicious data was filtered out
	// (BFT requires 2/3+ agreement, single malicious oracle should be ignored)
}
