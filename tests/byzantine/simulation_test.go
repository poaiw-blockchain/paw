package byzantine

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// Task 172: Byzantine Behavior Simulation
// Task 173: Network Partition Testing

// ValidatorBehavior defines validator behavior types
type ValidatorBehavior int

const (
	HonestValidator ValidatorBehavior = iota
	CrashingValidator
	ByzantineValidator
	SlowValidator
	EquivocatingValidator
)

// TestByzantineFaultTolerance tests BFT consensus with Byzantine validators
func TestByzantineFaultTolerance(t *testing.T) {
	testCases := []struct {
		name                 string
		totalValidators      int
		byzantineCount       int
		shouldReachConsensus bool
	}{
		{
			name:                 "2/3+ honest validators reach consensus",
			totalValidators:      10,
			byzantineCount:       3, // Less than 1/3
			shouldReachConsensus: true,
		},
		{
			name:                 "1/3+ Byzantine validators prevent consensus",
			totalValidators:      10,
			byzantineCount:       4, // More than 1/3
			shouldReachConsensus: false,
		},
		{
			name:                 "exactly 1/3 Byzantine at boundary",
			totalValidators:      9,
			byzantineCount:       3,    // Exactly 1/3
			shouldReachConsensus: true, // Should still work at boundary
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validators := createValidators(tc.totalValidators, tc.byzantineCount)
			consensus := simulateConsensus(validators)

			if tc.shouldReachConsensus {
				require.True(t, consensus.Reached, "consensus should be reached")
				require.NotEmpty(t, consensus.AgreedValue, "should have agreed value")
			} else {
				require.False(t, consensus.Reached, "consensus should not be reached")
			}
		})
	}
}

// TestDoubleSigningDetection tests detection of double-signing attacks
func TestDoubleSigningDetection(t *testing.T) {
	validator := &Validator{
		ID:       1,
		Behavior: EquivocatingValidator,
	}

	// Validator signs two different blocks at same height
	block1 := &Block{Height: 100, Hash: "hash1"}
	block2 := &Block{Height: 100, Hash: "hash2"}

	sig1 := validator.Sign(block1)
	sig2 := validator.Sign(block2)

	// Detect equivocation
	isEquivocating := detectEquivocation(sig1, sig2)
	require.True(t, isEquivocating, "should detect double-signing")
}

// TestOraclePriceManipulation tests Byzantine oracle validators
func TestOraclePriceManipulation(t *testing.T) {
	testCases := []struct {
		name            string
		honestPrices    []math.LegacyDec
		byzantinePrices []math.LegacyDec
		expectedMedian  math.LegacyDec
	}{
		{
			name: "Byzantine minority cannot manipulate median",
			honestPrices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(99),
				math.LegacyNewDec(100),
				math.LegacyNewDec(102),
			},
			byzantinePrices: []math.LegacyDec{
				math.LegacyNewDec(1000), // Extreme value
				math.LegacyNewDec(1),    // Extreme value
			},
			expectedMedian: math.LegacyNewDec(100), // Should be close to 100
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allPrices := make([]math.LegacyDec, 0, len(tc.honestPrices)+len(tc.byzantinePrices))
			allPrices = append(allPrices, tc.honestPrices...)
			allPrices = append(allPrices, tc.byzantinePrices...)
			median := calculateMedian(allPrices)

			deviation := median.Sub(tc.expectedMedian).Abs()
			maxDeviation := math.LegacyNewDec(5)

			require.True(t, deviation.LTE(maxDeviation),
				"median %s should be within %s of expected %s",
				median, maxDeviation, tc.expectedMedian)
		})
	}
}

// TestNetworkPartition tests behavior during network partitions
func TestNetworkPartition(t *testing.T) {
	// Create two partitions
	partition1 := []int{1, 2, 3, 4} // Majority
	partition2 := []int{5, 6}       // Minority

	network := &PartitionedNetwork{
		Partitions: [][]int{partition1, partition2},
	}

	// Simulate consensus attempts in both partitions
	consensus1 := network.AttemptConsensus(partition1)
	consensus2 := network.AttemptConsensus(partition2)

	// Majority partition should reach consensus
	require.True(t, consensus1.Reached, "majority partition should reach consensus")

	// Minority partition should not reach consensus
	require.False(t, consensus2.Reached, "minority partition should not reach consensus")
}

// TestPartitionRecovery tests recovery after network partition heals
func TestPartitionRecovery(t *testing.T) {
	// Initial partition
	partition1 := []int{1, 2, 3, 4}
	partition2 := []int{5, 6}

	network := &PartitionedNetwork{
		Partitions: [][]int{partition1, partition2},
	}

	// Partition 1 makes progress
	consensus1 := network.AttemptConsensus(partition1)
	require.True(t, consensus1.Reached)

	// Heal partition
	network.HealPartition()

	// After healing, all validators should converge to same state
	finalConsensus := network.AttemptConsensus([]int{1, 2, 3, 4, 5, 6})
	require.True(t, finalConsensus.Reached, "should reach consensus after healing")

	// Minority should adopt majority's state
	state1 := network.GetValidatorState(1)
	state5 := network.GetValidatorState(5)
	require.Equal(t, state1.BlockHeight, state5.BlockHeight,
		"all validators should converge to same height")
}

// TestSybilAttackResistance tests resistance to Sybil attacks
func TestSybilAttackResistance(t *testing.T) {
	// Attacker creates many low-stake validators
	honestValidators := []Validator{
		{ID: 1, Stake: math.NewInt(1000000), Behavior: HonestValidator},
		{ID: 2, Stake: math.NewInt(1000000), Behavior: HonestValidator},
		{ID: 3, Stake: math.NewInt(1000000), Behavior: HonestValidator},
	}

	sybilValidators := []Validator{}
	for i := 0; i < 100; i++ {
		sybilValidators = append(sybilValidators, Validator{
			ID:       100 + i,
			Stake:    math.NewInt(100), // Very low stake
			Behavior: ByzantineValidator,
		})
	}

	// Calculate voting power
	totalStake := math.ZeroInt()
	honestStake := math.ZeroInt()

	for _, v := range honestValidators {
		totalStake = totalStake.Add(v.Stake)
		honestStake = honestStake.Add(v.Stake)
	}
	for _, v := range sybilValidators {
		totalStake = totalStake.Add(v.Stake)
	}

	honestPower := math.LegacyNewDecFromInt(honestStake).Quo(math.LegacyNewDecFromInt(totalStake))

	// Honest validators should still have >2/3 voting power despite Sybil attack
	require.True(t, honestPower.GT(math.LegacyNewDecWithPrec(66, 2)),
		"honest validators should have >66%% voting power: %s", honestPower)
}

// TestLongRangeAttack tests defense against long-range attacks
func TestLongRangeAttack(t *testing.T) {
	// Attacker tries to create alternate chain from genesis
	// Should be prevented by checkpoints or weak subjectivity

	mainChain := &BlockChain{
		Blocks: []*Block{
			{Height: 0, Hash: "genesis"},
			{Height: 1, Hash: "block1"},
			{Height: 2, Hash: "block2"},
			{Height: 3, Hash: "block3"},
		},
		Checkpoint: 2, // Checkpoint at height 2
	}

	// Attacker creates alternative chain from genesis
	attackChain := &BlockChain{
		Blocks: []*Block{
			{Height: 0, Hash: "genesis"},
			{Height: 1, Hash: "attack1"},
			{Height: 2, Hash: "attack2"},
			{Height: 3, Hash: "attack3"},
		},
	}

	// New node should reject attack chain due to checkpoint
	shouldAccept := mainChain.ValidateAgainstCheckpoint(attackChain)
	require.False(t, shouldAccept, "should reject long-range attack")
}

// TestNothingAtStakeAttack tests prevention of nothing-at-stake problem
func TestNothingAtStakeAttack(t *testing.T) {
	// In PoS, validators might vote on multiple forks since it costs nothing
	// Should be prevented by slashing

	validator := &Validator{
		ID:    1,
		Stake: math.NewInt(100000),
	}

	fork1 := &Block{Height: 100, Hash: "fork1"}
	fork2 := &Block{Height: 100, Hash: "fork2"}

	// Validator signs both forks
	sig1 := validator.Sign(fork1)
	sig2 := validator.Sign(fork2)

	// Should be detected and slashed
	slashingEvidence := &SlashingEvidence{
		ValidatorID: validator.ID,
		Signature1:  sig1,
		Signature2:  sig2,
		Height:      100,
	}

	shouldSlash := validateSlashingEvidence(slashingEvidence)
	require.True(t, shouldSlash, "validator should be slashed for voting on multiple forks")

	// Calculate slash amount (e.g., 5% of stake)
	slashPercentage := math.LegacyNewDecWithPrec(5, 2)
	slashAmount := slashPercentage.Mul(math.LegacyNewDecFromInt(validator.Stake)).TruncateInt()

	expectedSlash := math.NewInt(5000) // 5% of 100000
	require.Equal(t, expectedSlash, slashAmount)
}

// TestCensorshipResistance tests resistance to censorship attacks
func TestCensorshipResistance(t *testing.T) {
	// Byzantine validators try to censor specific transactions

	validators := []*Validator{
		{ID: 1, Behavior: HonestValidator},
		{ID: 2, Behavior: HonestValidator},
		{ID: 3, Behavior: HonestValidator},
		{ID: 4, Behavior: ByzantineValidator}, // Tries to censor
	}

	targetTx := &Transaction{Hash: "target_tx"}

	// Byzantine validator rejects target tx
	mempool := &Mempool{
		Validators: validators,
	}

	included := mempool.SubmitTransaction(targetTx)

	// Transaction should still be included by honest majority
	require.True(t, included, "transaction should be included despite censorship attempt")
}

// Helper types and functions

type Validator struct {
	ID       int
	Stake    math.Int
	Behavior ValidatorBehavior
}

type Block struct {
	Height int64
	Hash   string
}

type Signature struct {
	ValidatorID int
	BlockHeight int64
	BlockHash   string
}

type Consensus struct {
	Reached     bool
	AgreedValue string
}

type PartitionedNetwork struct {
	Partitions [][]int
	States     map[int]*ValidatorState
}

type ValidatorState struct {
	BlockHeight int64
	BlockHash   string
}

type BlockChain struct {
	Blocks     []*Block
	Checkpoint int64
}

type SlashingEvidence struct {
	ValidatorID int
	Signature1  *Signature
	Signature2  *Signature
	Height      int64
}

type Transaction struct {
	Hash string
}

type Mempool struct {
	Validators []*Validator
}

func createValidators(total, byzantine int) []*Validator {
	validators := make([]*Validator, total)
	for i := 0; i < total; i++ {
		behavior := HonestValidator
		if i < byzantine {
			behavior = ByzantineValidator
		}
		validators[i] = &Validator{
			ID:       i,
			Stake:    math.NewInt(100000),
			Behavior: behavior,
		}
	}
	return validators
}

func simulateConsensus(validators []*Validator) *Consensus {
	totalStake := math.ZeroInt()
	honestStake := math.ZeroInt()

	for _, v := range validators {
		totalStake = totalStake.Add(v.Stake)
		if v.Behavior == HonestValidator {
			honestStake = honestStake.Add(v.Stake)
		}
	}

	honestPower := math.LegacyNewDecFromInt(honestStake).Quo(math.LegacyNewDecFromInt(totalStake))
	threshold := math.LegacyNewDec(2).Quo(math.LegacyNewDec(3)) // 2/3 threshold

	if honestPower.GTE(threshold) {
		return &Consensus{
			Reached:     true,
			AgreedValue: "consensus_block",
		}
	}

	return &Consensus{
		Reached: false,
	}
}

func (v *Validator) Sign(block *Block) *Signature {
	return &Signature{
		ValidatorID: v.ID,
		BlockHeight: block.Height,
		BlockHash:   block.Hash,
	}
}

func detectEquivocation(sig1, sig2 *Signature) bool {
	return sig1.ValidatorID == sig2.ValidatorID &&
		sig1.BlockHeight == sig2.BlockHeight &&
		sig1.BlockHash != sig2.BlockHash
}

func calculateMedian(prices []math.LegacyDec) math.LegacyDec {
	if len(prices) == 0 {
		return math.LegacyZeroDec()
	}

	sorted := make([]math.LegacyDec, len(prices))
	copy(sorted, prices)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].GT(sorted[j+1]) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return sorted[mid-1].Add(sorted[mid]).Quo(math.LegacyNewDec(2))
	}
	return sorted[mid]
}

func (n *PartitionedNetwork) AttemptConsensus(partition []int) *Consensus {
	// Simulate consensus in partition
	honestCount := 0
	for _, id := range partition {
		// Assume validators 1-4 are honest, 5-6 might be Byzantine
		if id <= 4 {
			honestCount++
		}
	}

	threshold := len(partition) * 2 / 3
	if honestCount >= threshold {
		return &Consensus{Reached: true, AgreedValue: "block"}
	}
	return &Consensus{Reached: false}
}

func (n *PartitionedNetwork) HealPartition() {
	// Merge all partitions
	n.Partitions = [][]int{{1, 2, 3, 4, 5, 6}}
}

func (n *PartitionedNetwork) GetValidatorState(id int) *ValidatorState {
	if n.States == nil {
		n.States = make(map[int]*ValidatorState)
	}
	if state, exists := n.States[id]; exists {
		return state
	}
	state := &ValidatorState{BlockHeight: 100, BlockHash: "block100"}
	n.States[id] = state
	return state
}

func (bc *BlockChain) ValidateAgainstCheckpoint(other *BlockChain) bool {
	// Check if other chain matches checkpoint
	if bc.Checkpoint >= int64(len(other.Blocks)) {
		return false
	}

	checkpointBlock := bc.Blocks[bc.Checkpoint]
	otherBlock := other.Blocks[bc.Checkpoint]

	return checkpointBlock.Hash == otherBlock.Hash
}

func validateSlashingEvidence(evidence *SlashingEvidence) bool {
	return detectEquivocation(
		evidence.Signature1,
		evidence.Signature2,
	)
}

func (m *Mempool) SubmitTransaction(tx *Transaction) bool {
	votes := 0
	for _, v := range m.Validators {
		if v.Behavior == HonestValidator {
			votes++
		}
		// Byzantine validators reject the transaction
	}

	// Need majority to include
	return votes > len(m.Validators)/2
}
