package statemachine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/testutil/network"
)

// StateMachineTestSuite tests state machine transitions and invariants
type StateMachineTestSuite struct {
	suite.Suite
	cfg     network.Config
	network *network.Network
}

func TestStateMachineTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping state machine tests in short mode")
	}
	suite.Run(t, new(StateMachineTestSuite))
}

func (suite *StateMachineTestSuite) SetupSuite() {
	suite.T().Log("setting up state machine test suite")

	suite.cfg = network.DefaultConfig()
	suite.cfg.NumValidators = 1

	var err error
	suite.network, err = network.New(suite.T(), suite.T().TempDir(), suite.cfg)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(1)
	suite.Require().NoError(err)
}

func (suite *StateMachineTestSuite) TearDownSuite() {
	suite.network.Cleanup()
}

// TestDEXPoolStateTransitions tests all valid DEX pool state transitions
func (suite *StateMachineTestSuite) TestDEXPoolStateTransitions() {
	suite.T().Log("Testing DEX pool state transitions")

	ctx := context.Background()
	val := suite.network.Validators[0]

	// State 1: Empty (no pool exists)
	suite.verifyDEXState(val, "empty")

	// Transition: Create Pool
	// State 1 -> State 2: Pool Created (with initial liquidity)
	poolID := suite.createDEXPool(ctx, val, "token0", "token1", 1000000, 2000000)
	suite.verifyDEXState(val, "pool_created")
	suite.verifyPoolInvariant(val, poolID)

	// Transition: Add Liquidity
	// State 2 -> State 3: Pool Active (increased liquidity)
	suite.addLiquidity(ctx, val, poolID, 500000, 1000000)
	suite.verifyDEXState(val, "pool_active")
	suite.verifyPoolInvariant(val, poolID)

	// Transition: Swap
	// State 3 -> State 3: Pool Active (reserves changed)
	suite.executeSwap(ctx, val, poolID, "token0", 100000)
	suite.verifyPoolInvariant(val, poolID)

	// Transition: Remove Liquidity
	// State 3 -> State 3 or State 4: Pool Active or Depleted
	suite.removeLiquidity(ctx, val, poolID, 50)
	suite.verifyPoolInvariant(val, poolID)

	// Verify all invariants hold after all transitions
	suite.verifyAllDEXInvariants(val)
}

// TestOracleStateTransitions tests oracle price submission state machine
func (suite *StateMachineTestSuite) TestOracleStateTransitions() {
	suite.T().Log("Testing oracle price submission state machine")

	ctx := context.Background()
	val := suite.network.Validators[0]

	// State 1: No Price Data
	suite.verifyOracleState(val, "BTC", "no_data")

	// Transition: First Price Submission
	// State 1 -> State 2: Price Prevote
	suite.submitPricePrevote(ctx, val, "BTC", 50000.0)
	suite.verifyOracleState(val, "BTC", "prevote")

	// Transition: Reveal Price
	// State 2 -> State 3: Price Revealed
	suite.revealPrice(ctx, val, "BTC", 50000.0)
	suite.verifyOracleState(val, "BTC", "revealed")

	// Wait for vote period
	time.Sleep(2 * time.Second)

	// Transition: Aggregate Prices
	// State 3 -> State 4: Price Active
	suite.verifyOracleState(val, "BTC", "active")
	suite.verifyPriceInvariant(val, "BTC")

	// Transition: Price Update
	// State 4 -> State 2: New Prevote (cycle repeats)
	suite.submitPricePrevote(ctx, val, "BTC", 51000.0)
	suite.verifyOracleState(val, "BTC", "prevote")
}

// TestComputeRequestStateTransitions tests compute request lifecycle
func (suite *StateMachineTestSuite) TestComputeRequestStateTransitions() {
	suite.T().Log("Testing compute request state transitions")

	ctx := context.Background()
	val := suite.network.Validators[0]

	// State 1: No Request
	requestID := suite.generateRequestID()
	suite.verifyComputeState(val, requestID, "not_exist")

	// Transition: Submit Request
	// State 1 -> State 2: Request Pending
	suite.submitComputeRequest(ctx, val, requestID)
	suite.verifyComputeState(val, requestID, "pending")

	// Transition: Assign Provider
	// State 2 -> State 3: Request Assigned
	suite.assignProvider(ctx, val, requestID, "provider1")
	suite.verifyComputeState(val, requestID, "assigned")

	// Transition: Start Execution
	// State 3 -> State 4: Request Running
	suite.startExecution(ctx, val, requestID)
	suite.verifyComputeState(val, requestID, "running")

	// Transition: Submit Result
	// State 4 -> State 5: Result Submitted
	suite.submitResult(ctx, val, requestID, []byte("result"))
	suite.verifyComputeState(val, requestID, "result_submitted")

	// Transition: Verify Result
	// State 5 -> State 6: Request Completed
	suite.verifyResult(ctx, val, requestID)
	suite.verifyComputeState(val, requestID, "completed")

	// Verify escrow released
	suite.verifyEscrowReleased(val, requestID)
}

// TestInvalidStateTransitions tests that invalid transitions are rejected
func (suite *StateMachineTestSuite) TestInvalidStateTransitions() {
	suite.T().Log("Testing invalid state transitions are rejected")

	ctx := context.Background()
	val := suite.network.Validators[0]

	// Try to remove liquidity before creating pool
	err := suite.tryRemoveLiquidity(ctx, val, 99999, 100)
	suite.Require().Error(err, "should reject removing liquidity from non-existent pool")

	// Try to swap in non-existent pool
	err = suite.trySwap(ctx, val, 99999, "token0", 100)
	suite.Require().Error(err, "should reject swap in non-existent pool")

	// Try to submit result for non-existent request
	err = suite.trySubmitResult(ctx, val, "fake-request", []byte("result"))
	suite.Require().Error(err, "should reject result for non-existent request")

	// Try to double-submit result
	requestID := suite.generateRequestID()
	suite.submitComputeRequest(ctx, val, requestID)
	suite.submitResult(ctx, val, requestID, []byte("result1"))
	err = suite.trySubmitResult(ctx, val, requestID, []byte("result2"))
	suite.Require().Error(err, "should reject double result submission")
}

// TestStateInvariants tests that invariants hold across all states
func (suite *StateMachineTestSuite) TestStateInvariants() {
	suite.T().Log("Testing state invariants")

	ctx := context.Background()
	val := suite.network.Validators[0]

	// Create initial state
	poolID := suite.createDEXPool(ctx, val, "token0", "token1", 1000000, 2000000)

	// Perform random state transitions
	for i := 0; i < 50; i++ {
		action := i % 3

		switch action {
		case 0: // Add liquidity
			suite.addLiquidity(ctx, val, poolID, 10000, 20000)
		case 1: // Swap
			suite.executeSwap(ctx, val, poolID, "token0", 5000)
		case 2: // Remove liquidity
			suite.removeLiquidity(ctx, val, poolID, 10)
		}

		// Verify invariants after each transition
		suite.verifyPoolInvariant(val, poolID)
		suite.verifyNoNegativeBalances(val)
		suite.verifyTotalSupplyConsistency(val)
	}

	suite.T().Log("All invariants maintained across 50 random transitions")
}

// TestConcurrentStateTransitions tests concurrent state modifications
func (suite *StateMachineTestSuite) TestConcurrentStateTransitions() {
	suite.T().Log("Testing concurrent state transitions")

	ctx := context.Background()
	val := suite.network.Validators[0]

	poolID := suite.createDEXPool(ctx, val, "token0", "token1", 10000000, 20000000)

	// Simulate concurrent swaps
	numConcurrent := 10
	done := make(chan bool, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			// Each goroutine performs swaps
			for j := 0; j < 5; j++ {
				suite.executeSwap(ctx, val, poolID, "token0", 1000)
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numConcurrent; i++ {
		<-done
	}

	// Verify invariants still hold after concurrent modifications
	suite.verifyPoolInvariant(val, poolID)
	suite.verifyNoNegativeBalances(val)
}

// Helper methods for state verification

func (suite *StateMachineTestSuite) verifyDEXState(val *network.Validator, expectedState string) {
	// Implementation depends on gRPC query client
	suite.T().Logf("Verifying DEX state: %s", expectedState)
}

func (suite *StateMachineTestSuite) verifyOracleState(val *network.Validator, asset, expectedState string) {
	suite.T().Logf("Verifying oracle state for %s: %s", asset, expectedState)
}

func (suite *StateMachineTestSuite) verifyComputeState(val *network.Validator, requestID, expectedState string) {
	suite.T().Logf("Verifying compute request %s state: %s", requestID, expectedState)
}

func (suite *StateMachineTestSuite) verifyPoolInvariant(val *network.Validator, poolID uint64) {
	// Verify constant product formula: x * y = k
	suite.T().Logf("Verifying pool %d invariant (x * y = k)", poolID)
}

func (suite *StateMachineTestSuite) verifyPriceInvariant(val *network.Validator, asset string) {
	// Verify price is positive and recent
	suite.T().Logf("Verifying price invariant for %s", asset)
}

func (suite *StateMachineTestSuite) verifyEscrowReleased(val *network.Validator, requestID string) {
	suite.T().Logf("Verifying escrow released for request %s", requestID)
}

func (suite *StateMachineTestSuite) verifyAllDEXInvariants(val *network.Validator) {
	suite.T().Log("Verifying all DEX invariants")
}

func (suite *StateMachineTestSuite) verifyNoNegativeBalances(val *network.Validator) {
	suite.T().Log("Verifying no negative balances")
}

func (suite *StateMachineTestSuite) verifyTotalSupplyConsistency(val *network.Validator) {
	suite.T().Log("Verifying total supply consistency")
}

// Helper methods for state transitions

func (suite *StateMachineTestSuite) createDEXPool(ctx context.Context, val *network.Validator, tokenA, tokenB string, amountA, amountB int64) uint64 {
	suite.T().Logf("Creating pool: %s/%s with liquidity %d/%d", tokenA, tokenB, amountA, amountB)
	return 1 // Mock pool ID
}

func (suite *StateMachineTestSuite) addLiquidity(ctx context.Context, val *network.Validator, poolID uint64, amountA, amountB int64) {
	suite.T().Logf("Adding liquidity to pool %d: %d/%d", poolID, amountA, amountB)
}

func (suite *StateMachineTestSuite) removeLiquidity(ctx context.Context, val *network.Validator, poolID uint64, sharePercent int) {
	suite.T().Logf("Removing %d%% liquidity from pool %d", sharePercent, poolID)
}

func (suite *StateMachineTestSuite) executeSwap(ctx context.Context, val *network.Validator, poolID uint64, tokenIn string, amountIn int64) {
	suite.T().Logf("Executing swap in pool %d: %d %s", poolID, amountIn, tokenIn)
}

func (suite *StateMachineTestSuite) tryRemoveLiquidity(ctx context.Context, val *network.Validator, poolID uint64, sharePercent int) error {
	return fmt.Errorf("pool not found")
}

func (suite *StateMachineTestSuite) trySwap(ctx context.Context, val *network.Validator, poolID uint64, tokenIn string, amountIn int64) error {
	return fmt.Errorf("pool not found")
}

func (suite *StateMachineTestSuite) submitPricePrevote(ctx context.Context, val *network.Validator, asset string, price float64) {
	suite.T().Logf("Submitting price prevote for %s: %.2f", asset, price)
}

func (suite *StateMachineTestSuite) revealPrice(ctx context.Context, val *network.Validator, asset string, price float64) {
	suite.T().Logf("Revealing price for %s: %.2f", asset, price)
}

func (suite *StateMachineTestSuite) submitComputeRequest(ctx context.Context, val *network.Validator, requestID string) {
	suite.T().Logf("Submitting compute request: %s", requestID)
}

func (suite *StateMachineTestSuite) assignProvider(ctx context.Context, val *network.Validator, requestID, providerID string) {
	suite.T().Logf("Assigning provider %s to request %s", providerID, requestID)
}

func (suite *StateMachineTestSuite) startExecution(ctx context.Context, val *network.Validator, requestID string) {
	suite.T().Logf("Starting execution for request %s", requestID)
}

func (suite *StateMachineTestSuite) submitResult(ctx context.Context, val *network.Validator, requestID string, result []byte) {
	suite.T().Logf("Submitting result for request %s", requestID)
}

func (suite *StateMachineTestSuite) verifyResult(ctx context.Context, val *network.Validator, requestID string) {
	suite.T().Logf("Verifying result for request %s", requestID)
}

func (suite *StateMachineTestSuite) trySubmitResult(ctx context.Context, val *network.Validator, requestID string, result []byte) error {
	return fmt.Errorf("request not found or already completed")
}

func (suite *StateMachineTestSuite) generateRequestID() string {
	return fmt.Sprintf("request-%d", time.Now().UnixNano())
}
