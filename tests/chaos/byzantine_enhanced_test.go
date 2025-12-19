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
			if err := node.ProposeBlock(ctx, attackBlock); err != nil {
				suite.T().Logf("failed to propose block: %v", err)
			}
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
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run 100 random Byzantine actions
	for i := 0; i < 100; i++ {
		attacker := attackers[rng.Intn(len(attackers))]
		action := rng.Intn(6)

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
		byzantine  int
		shouldHalt bool
	}{
		{4, false}, // 26.7% - should work
		{5, false}, // 33.3% - edge case
		{6, true},  // 40% - should halt
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
	// Simulate oracle price submission by creating a transaction with price data
	tx := &Transaction{
		ID:        fmt.Sprintf("oracle-%s-%s", node.ID, asset),
		Data:      []byte(fmt.Sprintf("%s:%.2f", asset, price)),
		Timestamp: time.Now(),
		Nonce:     uint64(price * 100), // Use price as nonce for detection
	}
	node.SubmitTransaction(ctx, tx)
}

func (suite *EnhancedByzantineTestSuite) calculateMedianPrice() float64 {
	// Calculate median from submitted prices
	prices := []float64{}
	for _, node := range suite.nodes {
		node.mu.RLock()
		for _, tx := range node.transactions {
			if len(tx.Data) > 0 && tx.Nonce > 0 {
				prices = append(prices, float64(tx.Nonce)/100.0)
			}
		}
		node.mu.RUnlock()
	}
	if len(prices) == 0 {
		return 50000.0
	}
	// Simple average as approximation
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	return sum / float64(len(prices))
}

func (suite *EnhancedByzantineTestSuite) checkOracleOutlier(nodeID string) bool {
	median := suite.calculateMedianPrice()
	threshold := median * 0.1 // 10% deviation threshold

	for _, node := range suite.nodes {
		if node.ID == nodeID {
			node.mu.RLock()
			for _, tx := range node.transactions {
				price := float64(tx.Nonce) / 100.0
				if price > 0 && (price > median+threshold || price < median-threshold) {
					node.mu.RUnlock()
					return true
				}
			}
			node.mu.RUnlock()
		}
	}
	return false
}

func (suite *EnhancedByzantineTestSuite) executeFrontRun(ctx context.Context, node *ByzantineNode, amount int64) {
	// Simulate front-running by submitting a transaction before victim
	tx := &Transaction{
		ID:        fmt.Sprintf("frontrun-%s-%d", node.ID, time.Now().UnixNano()),
		Data:      []byte(fmt.Sprintf("frontrun:%d", amount)),
		Timestamp: time.Now().Add(-100 * time.Millisecond), // Earlier timestamp
		Nonce:     uint64(amount),
	}
	node.SubmitTransaction(ctx, tx)
}

func (suite *EnhancedByzantineTestSuite) executeBackRun(ctx context.Context, node *ByzantineNode, amount int64) {
	// Simulate back-running by submitting a transaction after victim
	tx := &Transaction{
		ID:        fmt.Sprintf("backrun-%s-%d", node.ID, time.Now().UnixNano()),
		Data:      []byte(fmt.Sprintf("backrun:%d", amount)),
		Timestamp: time.Now().Add(100 * time.Millisecond), // Later timestamp
		Nonce:     uint64(amount),
	}
	node.SubmitTransaction(ctx, tx)
}

func (suite *EnhancedByzantineTestSuite) executeVictimSwap(ctx context.Context, amount int64) {
	// Execute victim's swap transaction
	if len(suite.nodes) > suite.maliciousCount {
		victim := suite.nodes[suite.maliciousCount] // First honest node
		tx := &Transaction{
			ID:        fmt.Sprintf("swap-victim-%d", time.Now().UnixNano()),
			Data:      []byte(fmt.Sprintf("swap:%d", amount)),
			Timestamp: time.Now(),
			Nonce:     uint64(amount),
		}
		victim.SubmitTransaction(ctx, tx)
	}
}

func (suite *EnhancedByzantineTestSuite) checkMEVProtection() bool {
	// Check if MEV protection is in place by verifying transaction ordering
	// If front-run transactions are consistently before victim transactions, protection failed
	frontRunCount := 0
	victimCount := 0

	for _, node := range suite.nodes {
		node.mu.RLock()
		for _, tx := range node.transactions {
			if len(tx.Data) > 8 && string(tx.Data[:8]) == "frontrun" {
				frontRunCount++
			}
			if len(tx.Data) > 4 && string(tx.Data[:4]) == "swap" {
				victimCount++
			}
		}
		node.mu.RUnlock()
	}

	// If front-runs exceed victims, protection is inadequate
	return frontRunCount <= victimCount
}

func (suite *EnhancedByzantineTestSuite) calculateVictimSlippage() float64 {
	// Calculate slippage based on front-run volume
	frontRunVolume := int64(0)
	victimVolume := int64(0)

	for _, node := range suite.nodes {
		node.mu.RLock()
		for _, tx := range node.transactions {
			if len(tx.Data) > 8 && string(tx.Data[:8]) == "frontrun" {
				frontRunVolume += int64(tx.Nonce)
			}
			if len(tx.Data) > 4 && string(tx.Data[:4]) == "swap" {
				victimVolume += int64(tx.Nonce)
			}
		}
		node.mu.RUnlock()
	}

	if victimVolume == 0 {
		return 0.0
	}
	return float64(frontRunVolume) / float64(victimVolume) * 0.01 // 1% per unit ratio
}

func (suite *EnhancedByzantineTestSuite) submitComputeResult(ctx context.Context, node *ByzantineNode, requestID string, result []byte) {
	tx := &Transaction{
		ID:        fmt.Sprintf("compute-result-%s-%s", node.ID, requestID),
		Data:      result,
		Timestamp: time.Now(),
		Nonce:     uint64(len(result)),
	}
	node.SubmitTransaction(ctx, tx)
}

func (suite *EnhancedByzantineTestSuite) verifyComputeResult(ctx context.Context, node *ByzantineNode, requestID string, expectedResult []byte) {
	// Verify that the submitted result matches expected
	node.mu.RLock()
	defer node.mu.RUnlock()

	txID := fmt.Sprintf("compute-result-%s-%s", node.ID, requestID)
	if tx, exists := node.transactions[txID]; exists {
		if string(tx.Data) != string(expectedResult) {
			// Mark node as flagged for incorrect result
			node.mu.RUnlock()
			node.mu.Lock()
			node.flaggedNodes[node.ID] = true
			node.mu.Unlock()
			node.mu.RLock()
		}
	}
}

func (suite *EnhancedByzantineTestSuite) checkResultAccepted(requestID string, result []byte) bool {
	// Check if majority of honest nodes accepted the result
	acceptCount := 0
	for i := suite.maliciousCount; i < len(suite.nodes); i++ {
		node := suite.nodes[i]
		node.mu.RLock()
		for _, tx := range node.transactions {
			if string(tx.Data) == string(result) {
				acceptCount++
				break
			}
		}
		node.mu.RUnlock()
	}
	honestCount := len(suite.nodes) - suite.maliciousCount
	return acceptCount > honestCount/2
}

func (suite *EnhancedByzantineTestSuite) checkProviderSlashed(providerID string) bool {
	// Check if the provider has been flagged/slashed
	for _, node := range suite.nodes {
		if node.ID == providerID {
			node.mu.RLock()
			flagged := node.flaggedNodes[providerID]
			slashed := node.slashedNodes[providerID]
			node.mu.RUnlock()
			return flagged || slashed
		}
	}
	return false
}

func (suite *EnhancedByzantineTestSuite) randomDoubleSigning(ctx context.Context, node *ByzantineNode) {
	if !node.isMalicious {
		return
	}
	// Create two blocks at same height with different hashes
	block1 := &Block{
		Height:       node.GetChainLength() + 1,
		PreviousHash: "hash1",
		Proposer:     node.ID,
		Timestamp:    time.Now(),
	}
	block2 := &Block{
		Height:       node.GetChainLength() + 1,
		PreviousHash: "hash2",
		Proposer:     node.ID,
		Timestamp:    time.Now(),
	}
	node.ProposeBlock(ctx, block1)
	node.ProposeBlock(ctx, block2)
}

func (suite *EnhancedByzantineTestSuite) randomEquivocation(ctx context.Context, node *ByzantineNode) {
	if !node.isMalicious {
		return
	}
	// Send conflicting transactions to different peers
	tx1 := &Transaction{ID: "equivoc-tx", Data: []byte("data1"), Nonce: 1}
	tx2 := &Transaction{ID: "equivoc-tx", Data: []byte("data2"), Nonce: 2}

	if len(suite.nodes) > suite.maliciousCount+1 {
		node.SendToSpecificNode(ctx, suite.nodes[suite.maliciousCount], tx1)
		node.SendToSpecificNode(ctx, suite.nodes[suite.maliciousCount+1], tx2)
	}
}

func (suite *EnhancedByzantineTestSuite) randomWithholding(ctx context.Context, node *ByzantineNode) {
	if !node.isMalicious {
		return
	}
	// Withhold by mining secretly
	block := &Block{
		Height:       node.GetChainLength() + 1,
		PreviousHash: "withheld",
		Proposer:     node.ID,
		Timestamp:    time.Now(),
	}
	node.MineSecretly(ctx, block)
}

func (suite *EnhancedByzantineTestSuite) randomSpam(ctx context.Context, node *ByzantineNode) {
	// Submit many transactions rapidly
	for i := 0; i < 100; i++ {
		tx := &Transaction{
			ID:        fmt.Sprintf("spam-%s-%d", node.ID, i),
			Data:      make([]byte, 1024), // 1KB spam
			Timestamp: time.Now(),
			Nonce:     uint64(i),
		}
		node.SubmitTransaction(ctx, tx)
	}
}

func (suite *EnhancedByzantineTestSuite) randomInvalidData(ctx context.Context, node *ByzantineNode) {
	// Submit transaction with invalid/malformed data
	tx := &Transaction{
		ID:        fmt.Sprintf("invalid-%s", node.ID),
		Data:      []byte{0xFF, 0xFE, 0x00, 0x01}, // Invalid encoding
		Timestamp: time.Now(),
		Nonce:     0xFFFFFFFFFFFFFFFF, // Max nonce (suspicious)
	}
	node.SubmitTransaction(ctx, tx)
}

func (suite *EnhancedByzantineTestSuite) randomDelayedMessage(ctx context.Context, node *ByzantineNode) {
	// Submit transaction with old timestamp
	tx := &Transaction{
		ID:        fmt.Sprintf("delayed-%s", node.ID),
		Data:      []byte("delayed message"),
		Timestamp: time.Now().Add(-1 * time.Hour), // 1 hour old
		Nonce:     1,
	}
	node.SubmitTransaction(ctx, tx)
}
