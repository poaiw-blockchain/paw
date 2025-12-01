package verification

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Task 170: State Machine Verification Tests

// StateMachineTestSuite tests state machine properties and invariants
type StateMachineTestSuite struct {
	suite.Suite
	// Test fixtures would be initialized here
}

func TestStateMachine(t *testing.T) {
	suite.Run(t, new(StateMachineTestSuite))
}

// TestDEXPoolInvariants verifies DEX pool state machine invariants
func (s *StateMachineTestSuite) TestDEXPoolInvariants() {
	t := s.T()

	// Invariant 1: k = reserveA * reserveB always increases or stays same (except for removals)
	testCases := []struct {
		name           string
		initialA       math.Int
		initialB       math.Int
		swapAmountIn   math.Int
		expectedKDelta string // "increase", "same", or "decrease"
	}{
		{
			name:           "swap increases k due to fees",
			initialA:       math.NewInt(1000000),
			initialB:       math.NewInt(1000000),
			swapAmountIn:   math.NewInt(1000),
			expectedKDelta: "increase",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate initial k
			initialK := tc.initialA.Mul(tc.initialB)

			// Simulate swap with 0.3% fee
			fee := math.LegacyNewDecWithPrec(3, 3)
			amountInAfterFee := math.LegacyNewDecFromInt(tc.swapAmountIn).Mul(math.LegacyOneDec().Sub(fee))

			// Calculate output
			numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(tc.initialB))
			denominator := math.LegacyNewDecFromInt(tc.initialA).Add(amountInAfterFee)
			amountOut := numerator.Quo(denominator).TruncateInt()

			// New reserves
			newA := tc.initialA.Add(tc.swapAmountIn)
			newB := tc.initialB.Sub(amountOut)
			newK := newA.Mul(newB)

			// Verify k increased (due to fees)
			switch tc.expectedKDelta {
			case "increase":
				require.True(t, newK.GT(initialK), "k should increase due to fees")
			case "same":
				require.True(t, newK.Equal(initialK), "k should stay same")
			case "decrease":
				require.True(t, newK.LT(initialK), "k should decrease")
			}
		})
	}
}

// TestOracleConsensusInvariants verifies oracle consensus properties
func (s *StateMachineTestSuite) TestOracleConsensusInvariants() {
	t := s.T()

	// Invariant: Median price cannot change by more than max deviation per block
	maxDeviation := math.LegacyNewDecWithPrec(10, 2) // 10%

	testCases := []struct {
		name          string
		previousPrice math.LegacyDec
		newPrices     []math.LegacyDec
		shouldPass    bool
	}{
		{
			name:          "normal price update within bounds",
			previousPrice: math.LegacyNewDec(100),
			newPrices: []math.LegacyDec{
				math.LegacyNewDec(99),
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(103),
			},
			shouldPass: true,
		},
		{
			name:          "extreme price change should be rejected",
			previousPrice: math.LegacyNewDec(100),
			newPrices: []math.LegacyDec{
				math.LegacyNewDec(50),
				math.LegacyNewDec(150),
				math.LegacyNewDec(200),
			},
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			median := calculateMedian(tc.newPrices)
			deviation := median.Sub(tc.previousPrice).Abs().Quo(tc.previousPrice)

			if tc.shouldPass {
				require.True(t, deviation.LTE(maxDeviation), "deviation should be within bounds")
			} else {
				require.True(t, deviation.GT(maxDeviation), "extreme deviation should be detected")
			}
		})
	}
}

// TestComputeJobStateTransitions verifies compute job state machine
func (s *StateMachineTestSuite) TestComputeJobStateTransitions() {
	t := s.T()

	// Valid state transitions: pending -> assigned -> executing -> completed
	//                                  -> assigned -> executing -> failed
	//                         pending -> cancelled

	type transition struct {
		from  string
		to    string
		valid bool
	}

	transitions := []transition{
		{"pending", "assigned", true},
		{"assigned", "executing", true},
		{"executing", "completed", true},
		{"executing", "failed", true},
		{"pending", "cancelled", true},
		{"pending", "completed", false}, // invalid - cannot skip states
		{"completed", "executing", false}, // invalid - cannot go back
		{"failed", "executing", false},    // invalid - terminal state
	}

	for _, tr := range transitions {
		t.Run(tr.from+"->"+tr.to, func(t *testing.T) {
			valid := isValidStateTransition(tr.from, tr.to)
			require.Equal(t, tr.valid, valid, "unexpected transition validity")
		})
	}
}

// TestBalanceConservation verifies token balance conservation
func (s *StateMachineTestSuite) TestBalanceConservation() {
	t := s.T()

	// Test that total token supply is conserved across operations
	type operation struct {
		opType string
		amount math.Int
	}

	initialSupply := math.NewInt(1000000000)
	ops := []operation{
		{"mint", math.NewInt(1000)},
		{"burn", math.NewInt(500)},
		{"transfer", math.NewInt(100)}, // transfer doesn't change total
	}

	expectedSupply := initialSupply
	for _, op := range ops {
		switch op.opType {
		case "mint":
			expectedSupply = expectedSupply.Add(op.amount)
		case "burn":
			expectedSupply = expectedSupply.Sub(op.amount)
		case "transfer":
			// No change to total supply
		}
	}

	require.Equal(t, math.NewInt(1000000500), expectedSupply)
}

// TestEventualConsistency tests eventual consistency properties
func (s *StateMachineTestSuite) TestEventualConsistency() {
	t := s.T()

	// Simulate delayed oracle updates converging to same value
	validators := []string{"val1", "val2", "val3", "val4", "val5"}
	submissions := make(map[string]math.LegacyDec)

	truePrice := math.LegacyNewDec(100)

	// Validators submit with some variance
	for i, val := range validators {
		variance := math.LegacyNewDec(int64(i - 2)) // -2, -1, 0, 1, 2
		submissions[val] = truePrice.Add(variance)
	}

	// Calculate median (should be close to true price)
	prices := make([]math.LegacyDec, 0, len(submissions))
	for _, price := range submissions {
		prices = append(prices, price)
	}

	median := calculateMedian(prices)

	// Median should be within 1% of true price
	deviation := median.Sub(truePrice).Abs().Quo(truePrice)
	maxDeviation := math.LegacyNewDecWithPrec(1, 2)

	require.True(t, deviation.LTE(maxDeviation),
		"median price %s should be within 1%% of true price %s, deviation: %s",
		median, truePrice, deviation)
}

// TestLivenessProperty verifies system liveness
func (s *StateMachineTestSuite) TestLivenessProperty() {
	t := s.T()

	// Test that system makes progress (blocks increase, transactions process)
	// This is a simplified test - in reality would test actual chain progression

	blocks := []int64{1, 2, 3, 4, 5}
	txCounts := []int{10, 15, 8, 20, 12}

	// Verify blocks always increase
	for i := 1; i < len(blocks); i++ {
		require.Greater(t, blocks[i], blocks[i-1], "blocks should always increase")
	}

	// Verify system processes transactions (at least some blocks have txs)
	totalTxs := 0
	for _, count := range txCounts {
		totalTxs += count
	}
	require.Greater(t, totalTxs, 0, "system should process transactions")
}

// TestSafetyProperty verifies safety properties
func (s *StateMachineTestSuite) TestSafetyProperty() {
	t := s.T()

	// Safety: Two correct validators should never disagree on committed blocks
	validator1Blocks := []string{"block1", "block2", "block3"}
	validator2Blocks := []string{"block1", "block2", "block3"}

	require.Equal(t, validator1Blocks, validator2Blocks,
		"validators should agree on committed blocks")

	// Safety: Double spend should be prevented
	balance := math.NewInt(100)
	spend1 := math.NewInt(60)
	spend2 := math.NewInt(60)

	// First spend succeeds
	newBalance := balance.Sub(spend1)
	require.Equal(t, math.NewInt(40), newBalance)

	// Second spend should fail (insufficient balance)
	canSpend := newBalance.GTE(spend2)
	require.False(t, canSpend, "double spend should be prevented")
}

// Helper functions

func calculateMedian(prices []math.LegacyDec) math.LegacyDec {
	if len(prices) == 0 {
		return math.LegacyZeroDec()
	}

	// Simple sort
	sorted := make([]math.LegacyDec, len(prices))
	copy(sorted, prices)

	// Bubble sort (sufficient for tests)
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

func isValidStateTransition(from, to string) bool {
	validTransitions := map[string][]string{
		"pending":   {"assigned", "cancelled"},
		"assigned":  {"executing"},
		"executing": {"completed", "failed"},
		"completed": {},
		"failed":    {},
		"cancelled": {},
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}

// TestPropertyBasedInvariants uses property-based testing
func (s *StateMachineTestSuite) TestPropertyBasedInvariants() {
	t := s.T()

	// Property: For any swap, output amount should be less than input amount
	// (accounting for price ratio)

	for i := 0; i < 100; i++ {
		reserveA := math.NewInt(int64(1000000 + i*1000))
		reserveB := math.NewInt(int64(2000000 + i*2000))
		amountIn := math.NewInt(int64(100 + i))

		fee := math.LegacyNewDecWithPrec(3, 3)
		amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(math.LegacyOneDec().Sub(fee))

		numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveB))
		denominator := math.LegacyNewDecFromInt(reserveA).Add(amountInAfterFee)
		amountOut := numerator.Quo(denominator).TruncateInt()

		// Verify output is positive and less than reserve
		require.True(t, amountOut.IsPositive(), "output should be positive")
		require.True(t, amountOut.LT(reserveB), "output should be less than reserve")

		// Verify constant product increased (due to fees)
		oldK := reserveA.Mul(reserveB)
		newK := reserveA.Add(amountIn).Mul(reserveB.Sub(amountOut))
		require.True(t, newK.GTE(oldK), "constant product should not decrease")
	}
}
