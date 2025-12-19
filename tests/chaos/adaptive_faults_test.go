package chaos

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type AdaptiveChaosTestSuite struct {
	suite.Suite
	network *NetworkSimulator
	nodes   []*Node
}

func TestAdaptiveChaosTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping adaptive chaos tests in short mode")
	}
	suite.Run(t, new(AdaptiveChaosTestSuite))
}

func (suite *AdaptiveChaosTestSuite) SetupTest() {
	suite.network = NewNetworkSimulator()
	suite.nodes = make([]*Node, 6)

	for i := 0; i < len(suite.nodes); i++ {
		node := NewNode(fmt.Sprintf("adaptive-node-%d", i), suite.network)
		suite.nodes[i] = node
		suite.network.AddNode(node)
	}

	suite.network.ConnectAll()
}

func (suite *AdaptiveChaosTestSuite) TearDownTest() {
	suite.network.Shutdown()
}

func (suite *AdaptiveChaosTestSuite) TestRollingLatencySpikes() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Apply heavy latency spike across the network
	suite.network.SetLatency(400 * time.Millisecond)
	tx := suite.newTransaction("latency-spike")
	suite.Require().NoError(suite.nodes[0].SubmitTransaction(ctx, tx))

	// Eventually some nodes should receive the tx even with spike
	suite.Eventually(func() bool {
		return suite.nodesWithTransaction(tx.ID) >= 2
	}, 3*time.Second, 100*time.Millisecond)

	// Normalize latency and ensure full propagation
	suite.network.SetLatency(15 * time.Millisecond)
	suite.Eventually(func() bool {
		return suite.allNodesHaveTransaction(tx.ID)
	}, 6*time.Second, 100*time.Millisecond, "transaction should reach all nodes after latency normalizes")
}

func (suite *AdaptiveChaosTestSuite) TestPacketLossRecovery() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Simulate extreme packet loss
	suite.network.SetPacketLoss(0.85)
	lossyTx := suite.newTransaction("lossy")
	suite.Require().NoError(suite.nodes[1].SubmitTransaction(ctx, lossyTx))

	time.Sleep(1 * time.Second)
	suite.False(suite.allNodesHaveTransaction(lossyTx.ID), "extreme packet loss should prevent propagation")

	// Restore reliable network and send a second tx
	suite.network.SetPacketLoss(0.0)
	reliableTx := suite.newTransaction("reliable")
	suite.Require().NoError(suite.nodes[2].SubmitTransaction(ctx, reliableTx))

	suite.Eventually(func() bool {
		return suite.allNodesHaveTransaction(reliableTx.ID)
	}, 6*time.Second, 100*time.Millisecond, "reliable network should propagate tx")

	// Re-broadcast the lossy tx to ensure eventual consistency
	suite.Require().NoError(suite.nodes[1].SubmitTransaction(ctx, lossyTx))
	suite.Eventually(func() bool {
		return suite.allNodesHaveTransaction(lossyTx.ID)
	}, 6*time.Second, 100*time.Millisecond, "retry should deliver previously dropped tx")
}

func (suite *AdaptiveChaosTestSuite) TestCompoundFaultRecovery() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Combine packet loss, node crash, and isolation
	suite.network.SetPacketLoss(0.6)
	suite.network.CrashNode(suite.nodes[0])
	suite.network.IsolateNode(suite.nodes[5])

	faultTx := suite.newTransaction("compound")
	suite.Require().NoError(suite.nodes[3].SubmitTransaction(ctx, faultTx))

	time.Sleep(1 * time.Second)
	suite.False(suite.nodes[4].HasTransaction(faultTx.ID), "compound faults should delay propagation")

	// Heal all faults
	suite.network.SetPacketLoss(0.0)
	suite.network.RecoverNode(suite.nodes[0])
	suite.network.ReconnectNode(suite.nodes[5])
	suite.network.HealPartition()

	// Resubmit to guarantee dissemination after recovery
	suite.Require().NoError(suite.nodes[3].SubmitTransaction(ctx, faultTx))
	suite.Eventually(func() bool {
		return suite.allNodesHaveTransaction(faultTx.ID)
	}, 10*time.Second, 200*time.Millisecond, "network should converge after combined fault recovery")
}

func (suite *AdaptiveChaosTestSuite) allNodesHaveTransaction(txID string) bool {
	for _, node := range suite.nodes {
		if !node.HasTransaction(txID) {
			return false
		}
	}
	return true
}

func (suite *AdaptiveChaosTestSuite) nodesWithTransaction(txID string) int {
	count := 0
	for _, node := range suite.nodes {
		if node.HasTransaction(txID) {
			count++
		}
	}
	return count
}

func (suite *AdaptiveChaosTestSuite) newTransaction(prefix string) *Transaction {
	return &Transaction{
		ID:        fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()),
		Data:      []byte(prefix),
		Timestamp: time.Now(),
	}
}
