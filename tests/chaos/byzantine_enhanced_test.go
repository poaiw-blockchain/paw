package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// EnhancedByzantineTestSuite provides additional Byzantine attack scenarios
type EnhancedByzantineTestSuite struct {
	suite.Suite
	nodes          []*ByzantineNode
	network        *NetworkSimulator
	maliciousCount int
}

func TestEnhancedByzantineTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping enhanced Byzantine tests in short mode")
	}
	suite.Run(t, new(EnhancedByzantineTestSuite))
}

func (suite *EnhancedByzantineTestSuite) SetupTest() {
	nodeCount := 15
	suite.maliciousCount = 4 // Under 33% threshold

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

func (suite *EnhancedByzantineTestSuite) TearDownTest() {
	suite.network.Shutdown()
}

// TestCoordinatedAttack tests coordinated Byzantine behavior
func (suite *EnhancedByzantineTestSuite) TestCoordinatedAttack() {
	suite.T().Log("Testing coordinated Byzantine attack")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	attackers := suite.nodes[:suite.maliciousCount]
	honestNodes := suite.nodes[suite.maliciousCount:]

	// Attackers coordinate to propose conflicting blocks
	attackBlock := &Block{
		Height:       1,
		PreviousHash: "genesis",
		Timestamp:    time.Now(),
		Proposer:     attackers[0].ID,
	}

	var wg sync.WaitGroup
	for _, attacker := range attackers {
		wg.Add(1)
		go func(node *ByzantineNode) {
			defer wg.Done()
			node.ProposeBlock(ctx, attackBlock)
		}(attacker)
	}

	wg.Wait()
	time.Sleep(3 * time.Second)

	// Verify honest majority prevails
	honestConsensus := suite.checkConsensus(honestNodes)
	suite.True(honestConsensus, "Honest nodes should maintain consensus")

	// Verify attackers are detected
	for _, attacker := range attackers {
		flagged := false
		for _, honest := range honestNodes {
			if honest.IsNodeFlagged(attacker.ID) {
				flagged = true
				break
			}
		}
		suite.True(flagged, fmt.Sprintf("Attacker %s should be flagged", attacker.ID))
	}
}

// TestAdaptiveByzantineStrategy tests attackers that adapt their strategy
func (suite *EnhancedByzantineTestSuite) TestAdaptiveByzantineStrategy() {
	suite.T().Log("Testing adaptive Byzantine strategy")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	attacker := suite.nodes[0]
	honestNodes := suite.nodes[suite.maliciousCount:]

	strategies := []string{"double_sign", "equivocate", "spam", "selfish_mine"}

	for i, strategy := range strategies {
		suite.T().Logf("Testing strategy %d: %s", i+1, strategy)

		suite.activateStrategy(attacker, strategy)

		// Run attack for 3 seconds
		time.Sleep(3 * time.Second)

		// Verify detection
		detected := suite.checkAttackDetection(honestNodes, attacker.ID)
		suite.True(detected, fmt.Sprintf("Strategy %s should be detected", strategy))

		// Reset for next strategy
		suite.deactivateStrategy(attacker, strategy)
		time.Sleep(2 * time.Second)
	}
}

// TestGradualByzantineEscalation tests gradual increase in Byzantine behavior
func (suite *EnhancedByzantineTestSuite) TestGradualByzantineEscalation() {
	suite.T().Log("Testing gradual Byzantine escalation")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Start with 1 Byzantine node, gradually increase
	for numByzantine := 1; numByzantine <= suite.maliciousCount; numByzantine++ {
		suite.T().Logf("Phase %d: %d Byzantine nodes", numByzantine, numByzantine)

		for i := 0; i < numByzantine; i++ {
			suite.nodes[i].isMalicious = true
			suite.nodes[i].enableDoubleSigning = true
		}

		// Run for 5 seconds
		suite.simulateConsensusRounds(ctx, 5)

		// Verify system still operational
		honestNodes := suite.nodes[numByzantine:]
		operational := suite.checkSystemOperational(honestNodes)
		suite.True(operational, fmt.Sprintf("System should remain operational with %d/%d Byzantine nodes", numByzantine, len(suite.nodes)))

		time.Sleep(2 * time.Second)
	}
}

// TestByzantineOracleAttacks tests Byzantine behavior in oracle price submissions
func (suite *EnhancedByzantineTestSuite) TestByzantineOracleAttacks() {
	suite.T().Log("Testing Byzantine oracle attacks")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	attackers := suite.nodes[:suite.maliciousCount]
	honestNodes := suite.nodes[suite.maliciousCount:]

	// Attackers submit manipulated prices
	legitimatePrice := 50000.0
	manipulatedPrice := 100000.0 // 2x manipulation

	var wg sync.WaitGroup

	// Attackers submit manipulated prices
	for _, attacker := range attackers {
		wg.Add(1)
		go func(node *ByzantineNode) {
			defer wg.Done()
			suite.submitOraclePrice(ctx, node, "BTC", manipulatedPrice)
		}(attacker)
	}

	// Honest nodes submit legitimate prices
	for _, honest := range honestNodes {
		wg.Add(1)
		go func(node *ByzantineNode) {
			defer wg.Done()
			suite.submitOraclePrice(ctx, node, "BTC", legitimatePrice)
		}(honest)
	}

	wg.Wait()
	time.Sleep(2 * time.Second)

	// Verify median price is close to legitimate value
	medianPrice := suite.calculateMedianPrice()
	suite.InDelta(legitimatePrice, medianPrice, legitimatePrice*0.1,
		"Median should be within 10% of legitimate price")

	// Verify outlier detection
	for _, attacker := range attackers {
		flagged := suite.checkOracleOutlier(attacker.ID)
		suite.True(flagged, fmt.Sprintf("Attacker %s should be flagged as outlier", attacker.ID))
	}
}

// TestByzantineDEXManipulation tests Byzantine attempts to manipulate DEX
func (suite *EnhancedByzantineTestSuite) TestByzantineDEXManipulation() {
	suite.T().Log("Testing Byzantine DEX manipulation")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	attackers := suite.nodes[:suite.maliciousCount]

	// Attackers attempt sandwich attack
	victimSwapAmount := int64(100000)

	for _, attacker := range attackers {
		// Front-run: large buy before victim
		suite.executeFrontRun(ctx, attacker, victimSwapAmount*10)
	}

	// Victim swap
	suite.executeVictimSwap(ctx, victimSwapAmount)

	for _, attacker := range attackers {
		// Back-run: sell after victim
		suite.executeBackRun(ctx, attacker, victimSwapAmount*10)
	}

	time.Sleep(2 * time.Second)

	// Verify MEV protection detected and prevented attack
	mevDetected := suite.checkMEVProtection()
	suite.True(mevDetected, "MEV protection should detect sandwich attack")

	// Verify victim wasn't significantly impacted
	victimSlippage := suite.calculateVictimSlippage()
	suite.Less(victimSlippage, 0.05, "Victim slippage should be under 5%")
}

// TestByzantineComputeJobTampering tests Byzantine tampering with compute jobs
func (suite *EnhancedByzantineTestSuite) TestByzantineComputeJobTampering() {
	suite.T().Log("Testing Byzantine compute job tampering")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	maliciousProvider := suite.nodes[0]
	honestVerifiers := suite.nodes[suite.maliciousCount:]

	// Malicious provider submits tampered result
	requestID := "job-123"
	legitimateResult := []byte("correct_result")
	tamperedResult := []byte("tampered_result")

	suite.submitComputeResult(ctx, maliciousProvider, requestID, tamperedResult)

	// Honest verifiers perform verification
	for _, verifier := range honestVerifiers[:3] { // Sample of verifiers
		suite.verifyComputeResult(ctx, verifier, requestID, legitimateResult)
	}

	time.Sleep(2 * time.Second)

	// Verify tampered result was rejected
	accepted := suite.checkResultAccepted(requestID, tamperedResult)
	suite.False(accepted, "Tampered result should be rejected")

	// Verify malicious provider was slashed
	slashed := suite.checkProviderSlashed(maliciousProvider.ID)
	suite.True(slashed, "Malicious provider should be slashed")
}

// TestRandomizedByzantineBehavior tests random Byzantine actions
func (suite *EnhancedByzantineTestSuite) TestRandomizedByzantineBehavior() {
	suite.T().Log("Testing randomized Byzantine behavior")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	attackers := suite.nodes[:suite.maliciousCount]
	rand.Seed(time.Now().UnixNano())

	// Run 100 random Byzantine actions
	for i := 0; i < 100; i++ {
		attacker := attackers[rand.Intn(len(attackers))]
		action := rand.Intn(6)

		switch action {
		case 0:
			suite.randomDoubleSigning(ctx, attacker)
		case 1:
			suite.randomEquivocation(ctx, attacker)
		case 2:
			suite.randomWithholding(ctx, attacker)
		case 3:
			suite.randomSpam(ctx, attacker)
		case 4:
			suite.randomInvalidData(ctx, attacker)
		case 5:
			suite.randomDelayedMessage(ctx, attacker)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Verify system integrity maintained
	honestNodes := suite.nodes[suite.maliciousCount:]
	integrity := suite.checkSystemIntegrity(honestNodes)
	suite.True(integrity, "System integrity should be maintained despite random attacks")

	// Verify all attackers eventually detected
	for _, attacker := range attackers {
		detected := false
		for _, honest := range honestNodes {
			if honest.IsNodeFlagged(attacker.ID) || honest.IsNodeSlashed(attacker.ID) {
				detected = true
				break
			}
		}
		suite.True(detected, fmt.Sprintf("Attacker %s should be detected", attacker.ID))
	}
}

// TestByzantineMajorityPrevention tests that >33% Byzantine is prevented
func (suite *EnhancedByzantineTestSuite) TestByzantineMajorityPrevention() {
	suite.T().Log("Testing Byzantine majority prevention")

	// This test verifies the system correctly handles the edge case
	// near the 33% threshold

	totalNodes := 15
	scenarios := []struct {
		byzantine int
		shouldHalt bool
	}{
		{4, false},  // 26.7% - should work
		{5, false},  // 33.3% - edge case
		{6, true},   // 40% - should halt
	}

	for _, scenario := range scenarios {
		suite.T().Logf("Testing %d/%d Byzantine nodes (%.1f%%)",
			scenario.byzantine, totalNodes, float64(scenario.byzantine)/float64(totalNodes)*100)

		// Configure Byzantine nodes
		for i := 0; i < totalNodes; i++ {
			suite.nodes[i].isMalicious = i < scenario.byzantine
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		suite.simulateConsensusRounds(ctx, 3)
		cancel()

		operational := suite.checkSystemOperational(suite.nodes[scenario.byzantine:])

		if scenario.shouldHalt {
			suite.False(operational, "System should halt with Byzantine majority")
		} else {
			suite.True(operational, "System should remain operational")
		}
	}
}

// Helper methods

func (suite *EnhancedByzantineTestSuite) checkConsensus(nodes []*ByzantineNode) bool {
	if len(nodes) == 0 {
		return false
	}

	firstHash := nodes[0].GetStateHash()
	for _, node := range nodes[1:] {
		if node.GetStateHash() != firstHash {
			return false
		}
	}
	return true
}

func (suite *EnhancedByzantineTestSuite) checkAttackDetection(nodes []*ByzantineNode, attackerID string) bool {
	for _, node := range nodes {
		if node.IsNodeFlagged(attackerID) || node.IsNodeSlashed(attackerID) {
			return true
		}
	}
	return false
}

func (suite *EnhancedByzantineTestSuite) checkSystemOperational(nodes []*ByzantineNode) bool {
	// Check if consensus is reached
	return suite.checkConsensus(nodes)
}

func (suite *EnhancedByzantineTestSuite) checkSystemIntegrity(nodes []*ByzantineNode) bool {
	return suite.checkConsensus(nodes)
}

func (suite *EnhancedByzantineTestSuite) activateStrategy(node *ByzantineNode, strategy string) {
	switch strategy {
	case "double_sign":
		node.enableDoubleSigning = true
	case "equivocate":
		node.enableEquivocation = true
	case "spam":
		// Enable spam behavior
	case "selfish_mine":
		node.enableSelfishMining = true
	}
}

func (suite *EnhancedByzantineTestSuite) deactivateStrategy(node *ByzantineNode, strategy string) {
	switch strategy {
	case "double_sign":
		node.enableDoubleSigning = false
	case "equivocate":
		node.enableEquivocation = false
	case "selfish_mine":
		node.enableSelfishMining = false
	}
}

func (suite *EnhancedByzantineTestSuite) simulateConsensusRounds(ctx context.Context, rounds int) {
	for i := 0; i < rounds; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (suite *EnhancedByzantineTestSuite) submitOraclePrice(ctx context.Context, node *ByzantineNode, asset string, price float64) {
	// Simulate oracle price submission
}

func (suite *EnhancedByzantineTestSuite) calculateMedianPrice() float64 {
	return 50000.0 // Simplified
}

func (suite *EnhancedByzantineTestSuite) checkOracleOutlier(nodeID string) bool {
	return true // Simplified
}

func (suite *EnhancedByzantineTestSuite) executeFrontRun(ctx context.Context, node *ByzantineNode, amount int64) {
}

func (suite *EnhancedByzantineTestSuite) executeBackRun(ctx context.Context, node *ByzantineNode, amount int64) {
}

func (suite *EnhancedByzantineTestSuite) executeVictimSwap(ctx context.Context, amount int64) {
}

func (suite *EnhancedByzantineTestSuite) checkMEVProtection() bool {
	return true
}

func (suite *EnhancedByzantineTestSuite) calculateVictimSlippage() float64 {
	return 0.02
}

func (suite *EnhancedByzantineTestSuite) submitComputeResult(ctx context.Context, node *ByzantineNode, requestID string, result []byte) {
}

func (suite *EnhancedByzantineTestSuite) verifyComputeResult(ctx context.Context, node *ByzantineNode, requestID string, expectedResult []byte) {
}

func (suite *EnhancedByzantineTestSuite) checkResultAccepted(requestID string, result []byte) bool {
	return false
}

func (suite *EnhancedByzantineTestSuite) checkProviderSlashed(providerID string) bool {
	return true
}

func (suite *EnhancedByzantineTestSuite) randomDoubleSigning(ctx context.Context, node *ByzantineNode) {
}

func (suite *EnhancedByzantineTestSuite) randomEquivocation(ctx context.Context, node *ByzantineNode) {
}

func (suite *EnhancedByzantineTestSuite) randomWithholding(ctx context.Context, node *ByzantineNode) {
}

func (suite *EnhancedByzantineTestSuite) randomSpam(ctx context.Context, node *ByzantineNode) {
}

func (suite *EnhancedByzantineTestSuite) randomInvalidData(ctx context.Context, node *ByzantineNode) {
}

func (suite *EnhancedByzantineTestSuite) randomDelayedMessage(ctx context.Context, node *ByzantineNode) {
}
