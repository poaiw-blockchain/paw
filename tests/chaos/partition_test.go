package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// NetworkPartitionTestSuite tests system behavior under network partitions
type NetworkPartitionTestSuite struct {
	suite.Suite
	nodes      []*Node
	network    *NetworkSimulator
	txCounter  uint64
	blockCount uint64
}

func TestNetworkPartitionSuite(t *testing.T) {
	suite.Run(t, new(NetworkPartitionTestSuite))
}

func (suite *NetworkPartitionTestSuite) SetupTest() {
	suite.nodes = make([]*Node, 7) // 7 nodes for Byzantine fault tolerance
	suite.network = NewNetworkSimulator()

	for i := 0; i < len(suite.nodes); i++ {
		suite.nodes[i] = NewNode(fmt.Sprintf("node-%d", i), suite.network)
		suite.network.AddNode(suite.nodes[i])
	}

	// Connect all nodes in fully connected mesh
	suite.network.ConnectAll()
}

func (suite *NetworkPartitionTestSuite) TearDownTest() {
	suite.network.Shutdown()
}

// TestMajorityPartition tests that majority partition can make progress
func (suite *NetworkPartitionTestSuite) TestMajorityPartition() {
	// Create 2/3 - 1/3 partition (5 nodes vs 2 nodes)
	majorityNodes := suite.nodes[:5]
	minorityNodes := suite.nodes[5:]

	suite.network.CreatePartition(majorityNodes, minorityNodes)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Majority partition should make progress
	initialBlocks := atomic.LoadUint64(&suite.blockCount)

	err := suite.submitTransactions(ctx, majorityNodes, 100)
	suite.NoError(err, "Majority partition should process transactions")

	time.Sleep(2 * time.Second)

	finalBlocks := atomic.LoadUint64(&suite.blockCount)
	suite.Greater(finalBlocks, initialBlocks, "Majority should produce blocks")

	// Minority partition should not make progress
	minorityBlocks := suite.getNodeBlockHeight(minorityNodes[0])
	suite.Less(minorityBlocks, finalBlocks, "Minority should lag behind")

	// Heal partition
	suite.network.HealPartition()
	time.Sleep(5 * time.Second)

	// After healing, minority should catch up
	minorityBlocksAfter := suite.getNodeBlockHeight(minorityNodes[0])
	suite.Greater(minorityBlocksAfter, minorityBlocks, "Minority should catch up")
}

// TestMinorityIsolation tests that isolated nodes rejoin correctly
func (suite *NetworkPartitionTestSuite) TestMinorityIsolation() {
	isolatedNode := suite.nodes[0]
	restNodes := suite.nodes[1:]

	suite.network.IsolateNode(isolatedNode)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Submit transactions to connected nodes
	err := suite.submitTransactions(ctx, restNodes, 50)
	suite.NoError(err)

	time.Sleep(2 * time.Second)

	connectedHeight := suite.getNodeBlockHeight(restNodes[0])
	isolatedHeight := suite.getNodeBlockHeight(isolatedNode)

	suite.Greater(connectedHeight, isolatedHeight, "Isolated node should fall behind")

	// Reconnect isolated node
	suite.network.ReconnectNode(isolatedNode)
	time.Sleep(10 * time.Second)

	// Node should sync
	finalIsolatedHeight := suite.getNodeBlockHeight(isolatedNode)
	suite.GreaterOrEqual(finalIsolatedHeight, connectedHeight-2, "Node should sync after reconnection")
}

// TestFlappingNetwork tests behavior under intermittent connectivity
func (suite *NetworkPartitionTestSuite) TestFlappingNetwork() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	stopFlapping := make(chan struct{})

	// Start flapping network connections
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.simulateNetworkFlapping(ctx, stopFlapping, 100*time.Millisecond)
	}()

	// Submit continuous transactions during flapping
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopFlapping:
				return
			default:
				_ = suite.submitSingleTransaction(suite.nodes[rand.Intn(len(suite.nodes))])
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	time.Sleep(30 * time.Second)
	close(stopFlapping)
	wg.Wait()

	// Verify all nodes eventually converge
	suite.network.StabilizeNetwork(5 * time.Second)

	heights := make([]uint64, len(suite.nodes))
	for i, node := range suite.nodes {
		heights[i] = suite.getNodeBlockHeight(node)
	}

	maxHeight := maxUint64Slice(heights)
	minHeight := minUint64Slice(heights)

	suite.LessOrEqual(maxHeight-minHeight, uint64(3), "Nodes should converge within 3 blocks")
}

// TestSplitBrain tests split-brain scenarios
func (suite *NetworkPartitionTestSuite) TestSplitBrain() {
	// Create exact 50-50 split (3 vs 3, with 1 isolated arbiter)
	partition1 := suite.nodes[:3]
	partition2 := suite.nodes[3:6]
	arbiter := suite.nodes[6]

	suite.network.CreateThreeWayPartition(partition1, partition2, []*Node{arbiter})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Neither partition should make progress (no 2/3 majority)
	initialHeight := suite.getNodeBlockHeight(partition1[0])

	_ = suite.submitTransactions(ctx, partition1, 10)
	time.Sleep(3 * time.Second)

	height1 := suite.getNodeBlockHeight(partition1[0])
	suite.LessOrEqual(height1-initialHeight, uint64(2), "Split brain should not make significant progress")

	// Arbiter joins partition1, giving it majority
	suite.network.ConnectNodes(arbiter, partition1...)
	time.Sleep(5 * time.Second)

	_ = suite.submitTransactions(ctx, partition1, 20)
	time.Sleep(3 * time.Second)

	height1After := suite.getNodeBlockHeight(partition1[0])
	suite.Greater(height1After, height1, "Partition with majority should make progress")
}

// TestAsymmetricPartition tests unidirectional network failures
func (suite *NetworkPartitionTestSuite) TestAsymmetricPartition() {
	nodeA := suite.nodes[0]
	nodeB := suite.nodes[1]

	// A can send to B, but B cannot send to A
	suite.network.CreateAsymmetricLink(nodeA, nodeB)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Send transaction from A
	tx := suite.createTransaction(nodeA, "asymmetric-test")
	err := nodeA.SubmitTransaction(ctx, tx)
	suite.NoError(err)

	time.Sleep(2 * time.Second)

	// B should receive transaction from A
	suite.True(nodeB.HasTransaction(tx.ID), "NodeB should receive transaction from NodeA")

	// Send transaction from B
	tx2 := suite.createTransaction(nodeB, "reverse-test")
	err = nodeB.SubmitTransaction(ctx, tx2)
	suite.NoError(err)

	time.Sleep(2 * time.Second)

	// A should not receive transaction from B
	suite.False(nodeA.HasTransaction(tx2.ID), "NodeA should not receive transaction from NodeB")

	// Repair asymmetric link
	suite.network.RepairAsymmetricLink(nodeA, nodeB)
	time.Sleep(3 * time.Second)

	// Now A should receive B's transaction
	suite.True(nodeA.HasTransaction(tx2.ID), "After repair, NodeA should receive transaction")
}

// TestCascadingFailures tests cascading node failures
func (suite *NetworkPartitionTestSuite) TestCascadingFailures() {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Start with all nodes healthy
	initialHeight := suite.getNodeBlockHeight(suite.nodes[0])

	// Cascade failures: one node every 5 seconds
	for i := 0; i < 3; i++ {
		suite.network.CrashNode(suite.nodes[i])
		time.Sleep(5 * time.Second)

		remaining := suite.nodes[i+1:]
		_ = suite.submitTransactions(ctx, remaining, 5)

		// System should still make progress with 4+ nodes
		currentHeight := suite.getNodeBlockHeight(remaining[0])
		suite.Greater(currentHeight, initialHeight, "System should survive %d failures", i+1)
	}

	// Recover nodes one by one
	for i := 0; i < 3; i++ {
		suite.network.RecoverNode(suite.nodes[i])
		time.Sleep(5 * time.Second)
	}

	// Verify all nodes converge
	suite.network.StabilizeNetwork(10 * time.Second)

	heights := suite.getAllNodeHeights()
	maxHeight := maxUint64Slice(heights)
	minHeight := minUint64Slice(heights)

	suite.LessOrEqual(maxHeight-minHeight, uint64(5), "All nodes should eventually converge")
}

// TestPartitionWithStateChanges tests data consistency across partitions
func (suite *NetworkPartitionTestSuite) TestPartitionWithStateChanges() {
	partition1 := suite.nodes[:4]
	partition2 := suite.nodes[4:]

	suite.network.CreatePartition(partition1, partition2)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Both partitions process different transactions
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.submitTransactions(ctx, partition1, 30)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.submitTransactions(ctx, partition2, 20)
	}()

	wg.Wait()
	time.Sleep(2 * time.Second)

	// Heal partition
	suite.network.HealPartition()
	time.Sleep(15 * time.Second)

	// Verify state consistency
	stateHashes := make([]string, len(suite.nodes))
	for i, node := range suite.nodes {
		stateHashes[i] = node.GetStateHash()
	}

	// All nodes should converge to same state
	for i := 1; i < len(stateHashes); i++ {
		suite.Equal(stateHashes[0], stateHashes[i], "All nodes should have consistent state")
	}
}

// Helper methods

func (suite *NetworkPartitionTestSuite) submitTransactions(ctx context.Context, nodes []*Node, count int) error {
	for i := 0; i < count; i++ {
		node := nodes[rand.Intn(len(nodes))]
		tx := suite.createTransaction(node, fmt.Sprintf("tx-%d", atomic.AddUint64(&suite.txCounter, 1)))

		err := node.SubmitTransaction(ctx, tx)
		if err != nil {
			return err
		}

		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (suite *NetworkPartitionTestSuite) submitSingleTransaction(node *Node) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	tx := suite.createTransaction(node, fmt.Sprintf("tx-%d", atomic.AddUint64(&suite.txCounter, 1)))
	return node.SubmitTransaction(ctx, tx)
}

func (suite *NetworkPartitionTestSuite) createTransaction(node *Node, data string) *Transaction {
	return &Transaction{
		ID:        fmt.Sprintf("%s-%s-%d", node.ID, data, time.Now().UnixNano()),
		Data:      []byte(data),
		Timestamp: time.Now(),
	}
}

func (suite *NetworkPartitionTestSuite) getNodeBlockHeight(node *Node) uint64 {
	return atomic.LoadUint64(&node.blockHeight)
}

func (suite *NetworkPartitionTestSuite) getAllNodeHeights() []uint64 {
	heights := make([]uint64, len(suite.nodes))
	for i, node := range suite.nodes {
		heights[i] = suite.getNodeBlockHeight(node)
	}
	return heights
}

func (suite *NetworkPartitionTestSuite) simulateNetworkFlapping(ctx context.Context, stop chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	connected := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			if connected {
				// Disconnect random pairs
				n1, n2 := suite.nodes[rand.Intn(len(suite.nodes))], suite.nodes[rand.Intn(len(suite.nodes))]
				suite.network.DisconnectNodes(n1, n2)
			} else {
				// Reconnect random pairs
				n1, n2 := suite.nodes[rand.Intn(len(suite.nodes))], suite.nodes[rand.Intn(len(suite.nodes))]
				suite.network.ConnectNodes(n1, n2)
			}
			connected = !connected
		}
	}
}

func maxUint64Slice(vals []uint64) uint64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func minUint64Slice(vals []uint64) uint64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
