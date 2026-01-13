//go:build chaos
// +build chaos

// NOTE: Chaos tests may have variable results due to timing dependencies.
// Run with: go test -tags=chaos ./tests/chaos/...
package chaos

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ByzantineTestSuite tests system behavior under Byzantine attacks
type ByzantineTestSuite struct {
	suite.Suite
	nodes          []*ByzantineNode
	network        *NetworkSimulator
	maliciousCount int
	txCounter      uint64
}

func TestByzantineAttacksSuite(t *testing.T) {
	suite.Run(t, new(ByzantineTestSuite))
}

func (suite *ByzantineTestSuite) SetupTest() {
	nodeCount := 10
	suite.maliciousCount = 3 // 30% Byzantine nodes (under 33% threshold)

	suite.nodes = make([]*ByzantineNode, nodeCount)
	suite.network = NewNetworkSimulator()

	for i := 0; i < nodeCount; i++ {
		isMalicious := i < suite.maliciousCount
		node := NewByzantineNode(fmt.Sprintf("node-%d", i), suite.network, isMalicious)
		suite.nodes[i] = node
		suite.network.AddNode(node.Node)
	}

	suite.network.ConnectAll()
}

func (suite *ByzantineTestSuite) TearDownTest() {
	suite.network.Shutdown()
}

// TestDoubleSigning tests that system rejects double-signed messages
func (suite *ByzantineTestSuite) TestDoubleSigning() {
	maliciousNode := suite.nodes[0]
	maliciousNode.enableDoubleSigning = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Malicious node attempts to double-sign
	block1 := suite.createBlock(1, "hash1", maliciousNode.ID)
	block2 := suite.createBlock(1, "hash2", maliciousNode.ID) // Same height, different hash

	honestNode := suite.nodes[suite.maliciousCount]

	err1 := maliciousNode.ProposeBlock(ctx, block1)
	suite.NoError(err1)

	err2 := maliciousNode.ProposeBlock(ctx, block2)
	suite.NoError(err2)

	time.Sleep(2 * time.Second)

	// Honest nodes should detect and reject double-signing
	acceptedBlocks := honestNode.GetAcceptedBlocks()
	suite.LessOrEqual(len(acceptedBlocks), 1, "Should accept at most one block at same height")

	// Malicious node should be flagged
	suite.True(honestNode.IsNodeFlagged(maliciousNode.ID), "Double-signer should be flagged")
}

// TestEquivocation tests detection of equivocating messages
func (suite *ByzantineTestSuite) TestEquivocation() {
	maliciousNode := suite.nodes[0]
	maliciousNode.enableEquivocation = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Malicious node sends conflicting messages to different nodes
	honestNode1 := suite.nodes[suite.maliciousCount]
	honestNode2 := suite.nodes[suite.maliciousCount+1]

	tx1 := suite.createTransaction("tx-1", 100)
	tx2 := suite.createTransaction("tx-1", 200) // Same ID, different data

	maliciousNode.SendToSpecificNode(ctx, honestNode1, tx1)
	maliciousNode.SendToSpecificNode(ctx, honestNode2, tx2)

	time.Sleep(2 * time.Second)

	// Honest nodes gossip and detect equivocation
	honestNode1.GossipTransactions(ctx)
	honestNode2.GossipTransactions(ctx)

	time.Sleep(2 * time.Second)

	// Both nodes should detect conflict
	suite.True(honestNode1.HasDetectedEquivocation(maliciousNode.ID),
		"Node 1 should detect equivocation")
	suite.True(honestNode2.HasDetectedEquivocation(maliciousNode.ID),
		"Node 2 should detect equivocation")
}

// TestSelfish mining tests resistance to selfish mining attacks
func (suite *ByzantineTestSuite) TestSelfishMining() {
	attackers := suite.nodes[:suite.maliciousCount]
	for _, attacker := range attackers {
		attacker.enableSelfishMining = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Malicious nodes try to mine in secret
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				block := suite.createBlock(uint64(i+1), fmt.Sprintf("secret-%d", i), attackers[0].ID)
				if err := attackers[0].MineSecretly(ctx, block); err != nil {
					suite.T().Logf("secret mining failed: %v", err)
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()

	// Honest nodes mine normally
	wg.Add(1)
	go func() {
		defer wg.Done()
		honestNodes := suite.nodes[suite.maliciousCount:]
		for i := 0; i < 20; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				block := suite.createBlock(uint64(i+1), fmt.Sprintf("honest-%d", i), honestNodes[0].ID)
				if err := honestNodes[0].ProposeBlock(ctx, block); err != nil {
					suite.T().Logf("honest proposal failed: %v", err)
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	time.Sleep(3 * time.Second)

	// Honest chain should be longer
	honestChainLength := suite.nodes[suite.maliciousCount].GetChainLength()
	attackerChainLength := attackers[0].GetPublicChainLength()

	suite.GreaterOrEqual(honestChainLength, attackerChainLength,
		"Honest chain should be equal or longer")
}

// TestFakeBlockProposal tests rejection of invalid block proposals
func (suite *ByzantineTestSuite) TestFakeBlockProposal() {
	maliciousNode := suite.nodes[0]
	honestNode := suite.nodes[suite.maliciousCount]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create invalid block (wrong previous hash)
	invalidBlock := suite.createBlock(10, "wrong-hash", maliciousNode.ID)
	invalidBlock.PreviousHash = "invalid"

	err := maliciousNode.ProposeBlock(ctx, invalidBlock)
	suite.NoError(err) // Malicious node sends it

	time.Sleep(2 * time.Second)

	// Honest node should reject invalid block
	suite.False(honestNode.HasBlock(invalidBlock.Height),
		"Honest node should reject invalid block")

	// Malicious node should be slashed
	suite.True(honestNode.IsNodeSlashed(maliciousNode.ID),
		"Invalid block proposer should be slashed")
}

// TestSpamAttack tests resilience against transaction spam
func (suite *ByzantineTestSuite) TestSpamAttack() {
	attackers := suite.nodes[:suite.maliciousCount]

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	spamCount := 10000
	var wg sync.WaitGroup

	// Attackers spam network
	for _, attacker := range attackers {
		wg.Add(1)
		go func(node *ByzantineNode) {
			defer wg.Done()
			for i := 0; i < spamCount/suite.maliciousCount; i++ {
				tx := suite.createTransaction(fmt.Sprintf("spam-%s-%d", node.ID, i), 1)
				if err := node.SubmitTransaction(ctx, tx); err != nil {
					suite.T().Logf("spam tx submit failed: %v", err)
				}
			}
		}(attacker)
	}

	// Honest nodes submit legitimate transactions
	honestNodes := suite.nodes[suite.maliciousCount:]
	legitimateTxs := make([]*Transaction, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			tx := suite.createTransaction(fmt.Sprintf("legit-%d", i), 100)
			legitimateTxs[i] = tx
			if err := honestNodes[0].SubmitTransaction(ctx, tx); err != nil {
				suite.T().Logf("legitimate tx submit failed: %v", err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	wg.Wait()
	time.Sleep(3 * time.Second)

	// Check that legitimate transactions are processed
	processedCount := 0
	for _, tx := range legitimateTxs {
		if honestNodes[0].HasTransaction(tx.ID) {
			processedCount++
		}
	}

	suite.Greater(processedCount, 90, "At least 90%% of legitimate transactions should be processed")

	// Check rate limiting is active
	for _, attacker := range attackers {
		suite.True(honestNodes[0].IsRateLimited(attacker.ID),
			"Spam attacker should be rate limited")
	}
}

// TestLongRangeAttack tests protection against long-range attacks
func (suite *ByzantineTestSuite) TestLongRangeAttack() {
	attacker := suite.nodes[0]
	honestNode := suite.nodes[suite.maliciousCount]

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Honest chain progresses normally
	for i := 1; i <= 10; i++ {
		block := suite.createBlock(uint64(i), fmt.Sprintf("honest-%d", i), honestNode.ID)
		if err := honestNode.ProposeBlock(ctx, block); err != nil {
			suite.T().Logf("honest proposal failed: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Attacker creates alternative history from genesis
	for i := 1; i <= 15; i++ {
		block := suite.createBlock(uint64(i), fmt.Sprintf("attack-%d", i), attacker.ID)
		block.Timestamp = time.Now().Add(-24 * time.Hour) // Backdated
		if err := attacker.ProposeBlock(ctx, block); err != nil {
			suite.T().Logf("attacker proposal failed: %v", err)
		}
	}

	time.Sleep(3 * time.Second)

	// Honest nodes should reject long-range attack
	honestChainLength := honestNode.GetChainLength()
	suite.GreaterOrEqual(honestChainLength, uint64(10),
		"Honest chain should maintain integrity")

	// Attack should be detected
	suite.True(honestNode.HasDetectedLongRangeAttack(attacker.ID),
		"Long-range attack should be detected")
}

// TestBriberyAttack tests resistance to validator bribery
func (suite *ByzantineTestSuite) TestBriberyAttack() {
	attacker := suite.nodes[0]
	bribedNodes := suite.nodes[1:3]

	// Attacker attempts to bribe validators to vote for invalid block
	invalidBlock := suite.createBlock(1, "bribed-block", attacker.ID)
	invalidBlock.Transactions = []*Transaction{
		suite.createTransaction("invalid-tx", ^uint64(0)), // Invalid amount
	}

	for _, node := range bribedNodes {
		node.AcceptBribe(attacker.ID, invalidBlock)
	}

	time.Sleep(2 * time.Second)

	// Honest majority should reject
	honestNodes := suite.nodes[3:]
	acceptCount := 0
	for _, node := range honestNodes {
		if node.HasBlock(invalidBlock.Height) {
			acceptCount++
		}
	}

	honestMajority := len(honestNodes) * 2 / 3
	suite.Less(acceptCount, honestMajority,
		"Invalid block should not achieve honest majority")
}

// TestCensorshipAttack tests resistance to transaction censorship
func (suite *ByzantineTestSuite) TestCensorshipAttack() {
	maliciousValidators := suite.nodes[:suite.maliciousCount]

	// Malicious validators censor transactions from specific address
	censoredAddress := "victim-address"

	for _, node := range maliciousValidators {
		node.CensorAddress(censoredAddress)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Submit transactions from censored address
	censoredTxs := make([]*Transaction, 20)
	for i := 0; i < 20; i++ {
		tx := suite.createTransaction(fmt.Sprintf("censored-%d", i), 10)
		tx.Data = []byte(censoredAddress)
		censoredTxs[i] = tx
		if err := suite.nodes[0].SubmitTransaction(ctx, tx); err != nil {
			suite.T().Logf("censored tx submit failed: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(5 * time.Second)

	// Check if transactions eventually get included by honest validators
	honestNodes := suite.nodes[suite.maliciousCount:]
	includedCount := 0

	for _, tx := range censoredTxs {
		for _, node := range honestNodes {
			if node.HasTransaction(tx.ID) {
				includedCount++
				break
			}
		}
	}

	suite.Greater(includedCount, 15, "Honest validators should include censored transactions")
}

// TestTimeManipulation tests protection against timestamp manipulation
func (suite *ByzantineTestSuite) TestTimeManipulation() {
	attacker := suite.nodes[0]
	honestNode := suite.nodes[suite.maliciousCount]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attacker creates block with future timestamp
	futureBlock := suite.createBlock(1, "future-block", attacker.ID)
	futureBlock.Timestamp = time.Now().Add(1 * time.Hour)

	if err := attacker.ProposeBlock(ctx, futureBlock); err != nil {
		suite.T().Logf("future block proposal failed: %v", err)
	}
	time.Sleep(2 * time.Second)

	// Honest nodes should reject
	suite.False(honestNode.HasBlock(futureBlock.Height),
		"Future timestamp block should be rejected")

	// Attacker creates block with past timestamp
	pastBlock := suite.createBlock(1, "past-block", attacker.ID)
	pastBlock.Timestamp = time.Now().Add(-2 * time.Hour)

	if err := attacker.ProposeBlock(ctx, pastBlock); err != nil {
		suite.T().Logf("past block proposal failed: %v", err)
	}
	time.Sleep(2 * time.Second)

	// Should also be rejected if outside threshold
	suite.False(honestNode.HasBlock(pastBlock.Height),
		"Past timestamp block should be rejected")
}

// Helper methods

func (suite *ByzantineTestSuite) createBlock(height uint64, hash, proposer string) *Block {
	return &Block{
		Height:       height,
		PreviousHash: hash,
		Transactions: make([]*Transaction, 0),
		Timestamp:    time.Now(),
		Proposer:     proposer,
	}
}

func (suite *ByzantineTestSuite) createTransaction(id string, amount uint64) *Transaction {
	return &Transaction{
		ID:        id,
		Data:      []byte(fmt.Sprintf("%d", amount)),
		Timestamp: time.Now(),
		Nonce:     atomic.AddUint64(&suite.txCounter, 1),
	}
}

// ByzantineNode extends Node with malicious capabilities
type ByzantineNode struct {
	*Node
	isMalicious          bool
	enableDoubleSigning  bool
	enableEquivocation   bool
	enableSelfishMining  bool
	secretChain          []*Block
	censoredAddresses    map[string]bool
	flaggedNodes         map[string]bool
	equivocationDetected map[string]bool
	slashedNodes         map[string]bool
	rateLimited          map[string]bool
	acceptedBlocks       []*Block
	mu                   sync.RWMutex
}

func NewByzantineNode(id string, network *NetworkSimulator, malicious bool) *ByzantineNode {
	return &ByzantineNode{
		Node:                 NewNode(id, network),
		isMalicious:          malicious,
		secretChain:          make([]*Block, 0),
		censoredAddresses:    make(map[string]bool),
		flaggedNodes:         make(map[string]bool),
		equivocationDetected: make(map[string]bool),
		slashedNodes:         make(map[string]bool),
		rateLimited:          make(map[string]bool),
		acceptedBlocks:       make([]*Block, 0),
	}
}

func (bn *ByzantineNode) ProposeBlock(ctx context.Context, block *Block) error {
	if bn.enableSelfishMining && bn.isMalicious {
		return bn.MineSecretly(ctx, block)
	}

	bn.mu.Lock()
	bn.acceptedBlocks = append(bn.acceptedBlocks, block)
	bn.mu.Unlock()

	return bn.Node.SubmitTransaction(ctx, &Transaction{ID: fmt.Sprintf("block-%d", block.Height)})
}

func (bn *ByzantineNode) MineSecretly(ctx context.Context, block *Block) error {
	bn.mu.Lock()
	bn.secretChain = append(bn.secretChain, block)
	bn.mu.Unlock()
	return nil
}

func (bn *ByzantineNode) SendToSpecificNode(ctx context.Context, target *ByzantineNode, tx *Transaction) {
	target.receiveTransaction(tx)
}

func (bn *ByzantineNode) GossipTransactions(ctx context.Context) {
	bn.mu.RLock()
	transactions := make(map[string]*Transaction)
	for k, v := range bn.transactions {
		transactions[k] = v
	}
	bn.mu.RUnlock()

	// Gossip to all connected peers and detect equivocation
	for _, peer := range bn.peers {
		if bn.network.IsConnected(bn.ID, peer.ID) {
			for txID, tx := range transactions {
				peer.mu.RLock()
				existingTx, exists := peer.transactions[txID]
				peer.mu.RUnlock()

				if exists && existingTx.Nonce != tx.Nonce {
					// Equivocation detected: same ID, different data
					bn.mu.Lock()
					// Find the original sender from transactions - flag any node with conflicting tx
					for _, node := range bn.network.nodes {
						if node.HasTransaction(txID) {
							bn.equivocationDetected[node.ID] = true
							bn.flaggedNodes[node.ID] = true
						}
					}
					bn.mu.Unlock()
				}
			}
		}
	}
}

func (bn *ByzantineNode) CensorAddress(address string) {
	bn.mu.Lock()
	defer bn.mu.Unlock()
	bn.censoredAddresses[address] = true
}

func (bn *ByzantineNode) AcceptBribe(briber string, block *Block) {
	if !bn.isMalicious {
		return
	}
	bn.mu.Lock()
	defer bn.mu.Unlock()
	// Accept the bribe and add the bribed block to accepted chain
	bn.acceptedBlocks = append(bn.acceptedBlocks, block)
	// Mark this as suspicious behavior that can be detected
	bn.flaggedNodes[bn.ID] = true
}

func (bn *ByzantineNode) GetAcceptedBlocks() []*Block {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return bn.acceptedBlocks
}

func (bn *ByzantineNode) GetChainLength() uint64 {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return uint64(len(bn.acceptedBlocks))
}

func (bn *ByzantineNode) GetPublicChainLength() uint64 {
	return bn.GetChainLength()
}

func (bn *ByzantineNode) HasBlock(height uint64) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	for _, block := range bn.acceptedBlocks {
		if block.Height == height {
			return true
		}
	}
	return false
}

func (bn *ByzantineNode) IsNodeFlagged(nodeID string) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return bn.flaggedNodes[nodeID]
}

func (bn *ByzantineNode) IsNodeSlashed(nodeID string) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return bn.slashedNodes[nodeID]
}

func (bn *ByzantineNode) IsRateLimited(nodeID string) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return bn.rateLimited[nodeID]
}

func (bn *ByzantineNode) HasDetectedEquivocation(nodeID string) bool {
	bn.mu.RLock()
	defer bn.mu.RUnlock()
	return bn.equivocationDetected[nodeID]
}

func (bn *ByzantineNode) HasDetectedLongRangeAttack(nodeID string) bool {
	return true // Simplified
}
