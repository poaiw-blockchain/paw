package chaos

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// EnhancedPartitionTestSuite provides comprehensive network partition testing
type EnhancedPartitionTestSuite struct {
	suite.Suite
	nodes   []*Node
	network *NetworkSimulator
}

func TestEnhancedPartitionTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping enhanced partition tests in short mode")
	}
	suite.Run(t, new(EnhancedPartitionTestSuite))
}

func (suite *EnhancedPartitionTestSuite) SetupTest() {
	nodeCount := 12
	suite.nodes = make([]*Node, nodeCount)
	suite.network = NewNetworkSimulator()

	for i := 0; i < nodeCount; i++ {
		node := NewNode(fmt.Sprintf("node-%d", i), suite.network)
		suite.nodes[i] = node
		suite.network.AddNode(node)
	}

	suite.network.ConnectAll()
}

func (suite *EnhancedPartitionTestSuite) TearDownTest() {
	suite.network.Shutdown()
}

// TestThreeWayPartition tests network split into three partitions
func (suite *EnhancedPartitionTestSuite) TestThreeWayPartition() {
	suite.T().Log("Testing three-way network partition")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Split network: 5-4-3 nodes
	partition1 := suite.nodes[0:5]  // Majority
	partition2 := suite.nodes[5:9]  // Minority 1
	partition3 := suite.nodes[9:12] // Minority 2

	suite.network.CreateThreeWayPartition(partition1, partition2, partition3)

	suite.T().Log("Network partitioned into 3 groups: 5, 4, 3 nodes")

	// Run for 10 seconds
	time.Sleep(10 * time.Second)

	// Verify partition isolation
	suite.verifyPartitionIsolation(partition1, partition2)
	suite.verifyPartitionIsolation(partition1, partition3)
	suite.verifyPartitionIsolation(partition2, partition3)

	// Heal partition
	suite.network.HealPartition()
	suite.T().Log("Partition healed")

	time.Sleep(5 * time.Second)

	// Verify network convergence
	suite.verifyNetworkConvergence(suite.nodes)
}

// TestAsymmetricPartition tests one-way network connectivity
func (suite *EnhancedPartitionTestSuite) TestAsymmetricPartition() {
	suite.T().Log("Testing asymmetric network partition")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create asymmetric link: node0 can send to node1, but not receive
	node0 := suite.nodes[0]
	node1 := suite.nodes[1]

	suite.network.CreateAsymmetricLink(node0, node1)

	suite.T().Log("Created asymmetric link: node0 -> node1 (one way)")

	// Verify asymmetry
	time.Sleep(2 * time.Second)

	// node0 can send to node1
	tx1 := &Transaction{ID: "tx-forward", Data: []byte("test")}
	node0.SubmitTransaction(ctx, tx1)

	time.Sleep(1 * time.Second)
	suite.True(node1.HasTransaction(tx1.ID), "node1 should receive from node0")

	// node1 cannot send to node0
	tx2 := &Transaction{ID: "tx-backward", Data: []byte("test")}
	node1.SubmitTransaction(ctx, tx2)

	time.Sleep(1 * time.Second)
	suite.False(node0.HasTransaction(tx2.ID), "node0 should not receive from node1")

	// Repair link
	suite.network.RepairAsymmetricLink(node0, node1)
	suite.T().Log("Repaired asymmetric link")

	time.Sleep(2 * time.Second)

	// Now both directions should work
	tx3 := &Transaction{ID: "tx-bidirectional", Data: []byte("test")}
	node1.SubmitTransaction(ctx, tx3)

	time.Sleep(1 * time.Second)
	suite.True(node0.HasTransaction(tx3.ID), "node0 should now receive from node1")
}

// TestCascadingPartitions tests sequential network fragmentation
func (suite *EnhancedPartitionTestSuite) TestCascadingPartitions() {
	suite.T().Log("Testing cascading network partitions")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Isolate nodes one by one
	for i := len(suite.nodes) - 1; i >= len(suite.nodes)/2; i-- {
		suite.T().Logf("Isolating node %d", i)
		suite.network.IsolateNode(suite.nodes[i])

		time.Sleep(2 * time.Second)

		// Verify remaining nodes maintain consensus
		remaining := suite.nodes[:i]
		if len(remaining) > 1 {
			consensus := suite.checkPartitionConsensus(remaining)
			suite.True(consensus, fmt.Sprintf("Remaining %d nodes should maintain consensus", len(remaining)))
		}
	}

	// Reconnect all nodes
	suite.T().Log("Reconnecting all nodes")
	for i := len(suite.nodes)/2; i < len(suite.nodes); i++ {
		suite.network.ReconnectNode(suite.nodes[i])
		time.Sleep(1 * time.Second)
	}

	time.Sleep(5 * time.Second)

	// Verify full network convergence
	suite.verifyNetworkConvergence(suite.nodes)
}

// TestIntermittentPartition tests rapidly changing network topology
func (suite *EnhancedPartitionTestSuite) TestIntermittentPartition() {
	suite.T().Log("Testing intermittent network partition")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	partition1 := suite.nodes[0:6]
	partition2 := suite.nodes[6:12]

	// Rapidly partition and heal 20 times
	for i := 0; i < 20; i++ {
		// Partition
		suite.network.CreatePartition(partition1, partition2)
		time.Sleep(500 * time.Millisecond)

		// Heal
		suite.network.HealPartition()
		time.Sleep(500 * time.Millisecond)
	}

	suite.T().Log("Completed 20 partition/heal cycles")

	// Let network stabilize
	time.Sleep(5 * time.Second)

	// Verify network convergence despite rapid changes
	suite.verifyNetworkConvergence(suite.nodes)
}

// TestPartialPartition tests partial network connectivity
func (suite *EnhancedPartitionTestSuite) TestPartialPartition() {
	suite.T().Log("Testing partial network partition")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create partial partition: some nodes connected, some not
	// Topology: nodes 0-5 fully connected, nodes 6-11 fully connected
	// Only nodes 5 and 6 bridge the two groups

	for i := 0; i < 5; i++ {
		for j := 6; j < 11; j++ {
			suite.network.DisconnectNodes(suite.nodes[i], suite.nodes[j])
		}
	}

	suite.T().Log("Created partial partition with bridge nodes")

	time.Sleep(5 * time.Second)

	// Submit transaction in group 1
	tx := &Transaction{ID: "cross-partition-tx", Data: []byte("test")}
	suite.nodes[0].SubmitTransaction(ctx, tx)

	time.Sleep(3 * time.Second)

	// Verify transaction propagates through bridge
	bridgeNode := suite.nodes[5]
	suite.True(bridgeNode.HasTransaction(tx.ID), "Bridge node should receive transaction")

	// Transaction should eventually reach group 2 through bridge
	time.Sleep(2 * time.Second)
	suite.True(suite.nodes[10].HasTransaction(tx.ID), "Transaction should propagate through bridge")
}

// TestIslandFormation tests isolated network islands
func (suite *EnhancedPartitionTestSuite) TestIslandFormation() {
	suite.T().Log("Testing network island formation")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create 3 islands: 4-4-4 nodes, completely isolated
	island1 := suite.nodes[0:4]
	island2 := suite.nodes[4:8]
	island3 := suite.nodes[8:12]

	// Disconnect all inter-island links
	for _, n1 := range island1 {
		for _, n2 := range island2 {
			suite.network.DisconnectNodes(n1, n2)
		}
		for _, n3 := range island3 {
			suite.network.DisconnectNodes(n1, n3)
		}
	}

	for _, n2 := range island2 {
		for _, n3 := range island3 {
			suite.network.DisconnectNodes(n2, n3)
		}
	}

	suite.T().Log("Created 3 isolated network islands")

	// Each island should develop independent state
	tx1 := &Transaction{ID: "island1-tx", Data: []byte("island1")}
	tx2 := &Transaction{ID: "island2-tx", Data: []byte("island2")}
	tx3 := &Transaction{ID: "island3-tx", Data: []byte("island3")}

	island1[0].SubmitTransaction(ctx, tx1)
	island2[0].SubmitTransaction(ctx, tx2)
	island3[0].SubmitTransaction(ctx, tx3)

	time.Sleep(3 * time.Second)

	// Verify islands are independent
	suite.assertTransactionConfinement(island1, tx1.ID, tx2.ID, tx3.ID)
	suite.assertTransactionConfinement(island2, tx2.ID, tx1.ID, tx3.ID)
	suite.assertTransactionConfinement(island3, tx3.ID, tx1.ID, tx2.ID)

	// Reconnect islands
	suite.network.ConnectAll()
	suite.T().Log("Reconnected all islands")

	time.Sleep(5 * time.Second)

	// Verify islands merge correctly
	suite.verifyNetworkConvergence(suite.nodes)
}

// TestPartitionDuringHighLoad tests partitions under transaction load
func (suite *EnhancedPartitionTestSuite) TestPartitionDuringHighLoad() {
	suite.T().Log("Testing partition during high transaction load")

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Start high transaction load
	stopLoad := make(chan bool)
	go suite.generateTransactionLoad(ctx, stopLoad)

	time.Sleep(5 * time.Second)

	// Create partition during load
	partition1 := suite.nodes[0:6]
	partition2 := suite.nodes[6:12]

	suite.network.CreatePartition(partition1, partition2)
	suite.T().Log("Created partition during high load")

	time.Sleep(10 * time.Second)

	// Heal partition while load continues
	suite.network.HealPartition()
	suite.T().Log("Healed partition during high load")

	time.Sleep(10 * time.Second)

	// Stop load
	stopLoad <- true
	time.Sleep(3 * time.Second)

	// Verify network recovered
	suite.verifyNetworkConvergence(suite.nodes)
}

// TestBrainSplit tests classic "split-brain" scenario
func (suite *EnhancedPartitionTestSuite) TestBrainSplit() {
	suite.T().Log("Testing split-brain scenario")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Split exactly in half: 6-6
	brain1 := suite.nodes[0:6]
	brain2 := suite.nodes[6:12]

	suite.network.CreatePartition(brain1, brain2)

	suite.T().Log("Created split-brain: 6-6 partition")

	// Both sides think they're the majority (edge case)
	time.Sleep(10 * time.Second)

	// Heal partition
	suite.network.HealPartition()

	time.Sleep(5 * time.Second)

	// Verify resolution - one side should win
	suite.verifyNetworkConvergence(suite.nodes)
}

// Helper methods

func (suite *EnhancedPartitionTestSuite) verifyPartitionIsolation(partition1, partition2 []*Node) {
	suite.T().Logf("Verifying isolation between partitions of size %d and %d", len(partition1), len(partition2))

	for _, n1 := range partition1 {
		for _, n2 := range partition2 {
			connected := suite.network.IsConnected(n1.ID, n2.ID)
			suite.False(connected, fmt.Sprintf("Nodes %s and %s should be disconnected", n1.ID, n2.ID))
		}
	}
}

func (suite *EnhancedPartitionTestSuite) verifyNetworkConvergence(nodes []*Node) {
	suite.T().Logf("Verifying network convergence for %d nodes", len(nodes))

	if len(nodes) < 2 {
		return
	}

	// Check all nodes have similar state
	firstHash := nodes[0].GetStateHash()
	for i, node := range nodes[1:] {
		hash := node.GetStateHash()
		suite.Equal(firstHash, hash, fmt.Sprintf("Node %d state should match node 0", i+1))
	}

	suite.T().Log("Network convergence verified")
}

func (suite *EnhancedPartitionTestSuite) checkPartitionConsensus(nodes []*Node) bool {
	if len(nodes) < 2 {
		return true
	}

	firstHash := nodes[0].GetStateHash()
	for _, node := range nodes[1:] {
		if node.GetStateHash() != firstHash {
			return false
		}
	}
	return true
}

func (suite *EnhancedPartitionTestSuite) assertTransactionConfinement(island []*Node, presentTx string, absentTx1 string, absentTx2 string) {
	for _, node := range island {
		suite.True(node.HasTransaction(presentTx), "Island should have own transaction")
		suite.False(node.HasTransaction(absentTx1), "Island should not have other island's transaction")
		suite.False(node.HasTransaction(absentTx2), "Island should not have other island's transaction")
	}
}

func (suite *EnhancedPartitionTestSuite) generateTransactionLoad(ctx context.Context, stop chan bool) {
	counter := 0
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		default:
			// Submit transaction from random node
			node := suite.nodes[counter%len(suite.nodes)]
			tx := &Transaction{
				ID:   fmt.Sprintf("load-tx-%d", counter),
				Data: []byte(fmt.Sprintf("data-%d", counter)),
			}
			node.SubmitTransaction(ctx, tx)
			counter++
			time.Sleep(50 * time.Millisecond)
		}
	}
}
