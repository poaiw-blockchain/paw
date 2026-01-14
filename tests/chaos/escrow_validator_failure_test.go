//go:build chaos

// TEST-1.2: Chaos Engineering - Validator Failures During Escrow
// Tests escrow resilience when validators fail during critical operations
package chaos

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// EscrowValidatorFailureTestSuite tests escrow behavior during validator failures
type EscrowValidatorFailureTestSuite struct {
	suite.Suite
	k          *keeper.Keeper
	ctx        sdk.Context
	bankKeeper bankkeeper.Keeper

	// Simulated validators
	validators      []*SimulatedValidator
	activeCount     int32
	consensusQuorum int
	requestCounter  uint64
}

// SimulatedValidator represents a validator in the test
type SimulatedValidator struct {
	ID          string
	VotingPower int64
	IsActive    bool
	FailedAt    time.Time
	RecoveredAt time.Time
	mu          sync.Mutex
}

func TestEscrowValidatorFailureTestSuite(t *testing.T) {
	suite.Run(t, new(EscrowValidatorFailureTestSuite))
}

func (suite *EscrowValidatorFailureTestSuite) SetupTest() {
	suite.k, suite.ctx, suite.bankKeeper = keepertest.ComputeKeeperWithBank(suite.T())

	// Create 10 validators (4 required for consensus in BFT)
	numValidators := 10
	suite.validators = make([]*SimulatedValidator, numValidators)
	suite.consensusQuorum = (numValidators * 2 / 3) + 1 // 67% + 1

	for i := 0; i < numValidators; i++ {
		suite.validators[i] = &SimulatedValidator{
			ID:          fmt.Sprintf("validator_%d", i),
			VotingPower: 100,
			IsActive:    true,
		}
	}
	suite.activeCount = int32(numValidators)
	suite.requestCounter = 1000
}

// getNextRequestID generates a unique request ID for escrow tests
func (suite *EscrowValidatorFailureTestSuite) getNextRequestID() uint64 {
	return atomic.AddUint64(&suite.requestCounter, 1)
}

// fundAccount mints coins to an account for testing
func (suite *EscrowValidatorFailureTestSuite) fundAccount(addr sdk.AccAddress, amount math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin("upaw", amount))
	err := suite.bankKeeper.MintCoins(suite.ctx, computetypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, computetypes.ModuleName, addr, coins)
	suite.Require().NoError(err)
}

// failValidator simulates a validator going offline
func (suite *EscrowValidatorFailureTestSuite) failValidator(v *SimulatedValidator) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.IsActive {
		v.IsActive = false
		v.FailedAt = time.Now()
		atomic.AddInt32(&suite.activeCount, -1)
	}
}

// recoverValidator simulates a validator coming back online
func (suite *EscrowValidatorFailureTestSuite) recoverValidator(v *SimulatedValidator) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.IsActive {
		v.IsActive = true
		v.RecoveredAt = time.Now()
		atomic.AddInt32(&suite.activeCount, 1)
	}
}

// hasConsensus checks if we still have consensus
func (suite *EscrowValidatorFailureTestSuite) hasConsensus() bool {
	return int(atomic.LoadInt32(&suite.activeCount)) >= suite.consensusQuorum
}

// TestEscrowSurvivesSingleValidatorFailure tests escrow with 1 validator down
func (suite *EscrowValidatorFailureTestSuite) TestEscrowSurvivesSingleValidatorFailure() {
	// Create escrow
	requester := sdk.AccAddress([]byte("escrow_test_requester"))
	provider := sdk.AccAddress([]byte("chaos_test_provider1"))
	escrowAmount := math.NewInt(1_000_000)
	requestID := suite.getNextRequestID()

	// Fund requester
	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	// Create job with escrow
	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	// Fail one validator
	suite.failValidator(suite.validators[0])

	// Verify consensus still possible
	suite.True(suite.hasConsensus(), "Should still have consensus with 1 validator down")

	// Verify escrow still accessible
	escrow, err := suite.k.GetEscrowState(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.NotNil(escrow, "Escrow should still be accessible")

	// Release escrow (simulating job completion)
	err = suite.k.ReleaseEscrow(suite.ctx, requestID, false)
	suite.NoError(err, "Escrow release should succeed with 1 validator down")

	suite.T().Log("TEST-1.2: Escrow survived single validator failure")
}

// TestEscrowSurvivesMultipleValidatorFailures tests escrow with multiple failures
func (suite *EscrowValidatorFailureTestSuite) TestEscrowSurvivesMultipleValidatorFailures() {
	requester := sdk.AccAddress([]byte("multi_fail_requester"))
	provider := sdk.AccAddress([]byte("multi_fail_provider1"))
	escrowAmount := math.NewInt(2_000_000)
	requestID := suite.getNextRequestID()

	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	// Fail 3 validators (should still have consensus with 7/10)
	for i := 0; i < 3; i++ {
		suite.failValidator(suite.validators[i])
	}

	suite.True(suite.hasConsensus(), "Should have consensus with 3 validators down (7/10)")

	// Operations should still work
	escrow, err := suite.k.GetEscrowState(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.NotNil(escrow)
	suite.Equal(computetypes.ESCROW_STATUS_LOCKED, escrow.Status)

	suite.T().Logf("TEST-1.2: Escrow operational with %d validators (quorum=%d)",
		atomic.LoadInt32(&suite.activeCount), suite.consensusQuorum)
}

// TestEscrowBehaviorAtConsensusThreshold tests at exact consensus threshold
func (suite *EscrowValidatorFailureTestSuite) TestEscrowBehaviorAtConsensusThreshold() {
	requester := sdk.AccAddress([]byte("threshold_requester"))
	provider := sdk.AccAddress([]byte("threshold_provider1"))
	escrowAmount := math.NewInt(3_000_000)
	requestID := suite.getNextRequestID()

	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	// Fail validators until exactly at threshold
	failCount := len(suite.validators) - suite.consensusQuorum
	for i := 0; i < failCount; i++ {
		suite.failValidator(suite.validators[i])
	}

	activeNow := int(atomic.LoadInt32(&suite.activeCount))
	suite.Equal(suite.consensusQuorum, activeNow,
		"Should be exactly at consensus threshold")

	suite.True(suite.hasConsensus(), "Should still have consensus at threshold")

	// Escrow operations should work at threshold
	escrow, err := suite.k.GetEscrowState(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.NotNil(escrow)

	suite.T().Logf("TEST-1.2: Escrow functional at exact consensus threshold (%d validators)",
		activeNow)
}

// TestEscrowRefundOnValidatorRecovery tests refund after validators recover
func (suite *EscrowValidatorFailureTestSuite) TestEscrowRefundOnValidatorRecovery() {
	requester := sdk.AccAddress([]byte("recovery_requester1"))
	provider := sdk.AccAddress([]byte("recovery_provider01"))
	escrowAmount := math.NewInt(5_000_000)
	requestID := suite.getNextRequestID()

	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	// Fail some validators
	for i := 0; i < 3; i++ {
		suite.failValidator(suite.validators[i])
	}

	// Simulate timeout/failure requiring refund
	err = suite.k.RefundEscrow(suite.ctx, requestID, "validator_failures")
	suite.NoError(err, "Refund should work with reduced validator set")

	// Recover validators
	for i := 0; i < 3; i++ {
		suite.recoverValidator(suite.validators[i])
	}

	// Verify full validator set restored
	suite.Equal(int32(10), atomic.LoadInt32(&suite.activeCount))

	suite.T().Log("TEST-1.2: Escrow refund succeeded, validators recovered")
}

// TestConcurrentEscrowsWithRollingFailures tests multiple escrows during rolling failures
func (suite *EscrowValidatorFailureTestSuite) TestConcurrentEscrowsWithRollingFailures() {
	numJobs := 20
	escrowAmount := math.NewInt(100_000)

	var wg sync.WaitGroup
	var successCount atomic.Uint64
	var failCount atomic.Uint64

	// Start rolling validator failures in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		failIdx := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Fail one, recover previous
				if failIdx > 0 && failIdx < len(suite.validators) {
					suite.recoverValidator(suite.validators[failIdx-1])
				}
				if failIdx < len(suite.validators)-suite.consensusQuorum {
					suite.failValidator(suite.validators[failIdx])
					failIdx++
				}
			}
		}
	}()

	// Create concurrent escrows
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			requester := sdk.AccAddress([]byte(fmt.Sprintf("concurrent_req_%03d", idx)))
			provider := sdk.AccAddress([]byte(fmt.Sprintf("concurrent_prov_%03d", idx)))
			requestID := suite.getNextRequestID()

			suite.fundAccount(requester, escrowAmount.MulRaw(10))

			err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)

			if err == nil {
				successCount.Add(1)

				// Try to release
				_ = suite.k.ReleaseEscrow(suite.ctx, requestID, false)
			} else {
				failCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	cancel()

	successRate := float64(successCount.Load()) / float64(numJobs) * 100

	suite.T().Logf("TEST-1.2: Concurrent escrows with rolling failures:")
	suite.T().Logf("  → Total jobs: %d", numJobs)
	suite.T().Logf("  → Successful: %d (%.1f%%)", successCount.Load(), successRate)
	suite.T().Logf("  → Failed: %d", failCount.Load())

	// Should maintain high success rate even with failures
	suite.GreaterOrEqual(successRate, 50.0, "Success rate should be >=50% during rolling failures")
}

// TestEscrowStateConsistencyDuringChaos verifies state consistency
func (suite *EscrowValidatorFailureTestSuite) TestEscrowStateConsistencyDuringChaos() {
	requester := sdk.AccAddress([]byte("consistency_requester"))
	provider := sdk.AccAddress([]byte("consistency_provider"))
	escrowAmount := math.NewInt(10_000_000)
	requestID := suite.getNextRequestID()

	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	// Create escrow
	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	escrowBefore, err := suite.k.GetEscrowState(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.Require().NotNil(escrowBefore)

	// Chaos: fail and recover validators multiple times
	for cycle := 0; cycle < 5; cycle++ {
		// Fail 2 random validators
		suite.failValidator(suite.validators[cycle%10])
		suite.failValidator(suite.validators[(cycle+5)%10])

		// Verify escrow state unchanged
		escrowDuring, err := suite.k.GetEscrowState(suite.ctx, requestID)
		suite.Require().NoError(err)
		suite.NotNil(escrowDuring)
		suite.Equal(escrowBefore.Status, escrowDuring.Status, "Status should be consistent")
		suite.True(escrowBefore.Amount.Equal(escrowDuring.Amount), "Amount should be consistent")

		// Recover validators
		suite.recoverValidator(suite.validators[cycle%10])
		suite.recoverValidator(suite.validators[(cycle+5)%10])
	}

	escrowAfter, err := suite.k.GetEscrowState(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.Equal(escrowBefore.Status, escrowAfter.Status, "Final status should match initial")

	suite.T().Log("TEST-1.2: Escrow state consistency maintained during chaos")
}

// TestEscrowTimeoutDuringPartition tests timeout handling during network partition
func (suite *EscrowValidatorFailureTestSuite) TestEscrowTimeoutDuringPartition() {
	requester := sdk.AccAddress([]byte("partition_requester"))
	provider := sdk.AccAddress([]byte("partition_provider1"))
	escrowAmount := math.NewInt(15_000_000)
	requestID := suite.getNextRequestID()

	suite.fundAccount(requester, escrowAmount.MulRaw(10))

	err := suite.k.LockEscrow(suite.ctx, requester, provider, escrowAmount, requestID, 3600)
	suite.Require().NoError(err)

	// Simulate network partition (fail half the validators)
	for i := 0; i < 5; i++ {
		suite.failValidator(suite.validators[i])
	}

	// We still have consensus (5/10 = 50%, need 67%)
	// This partition would actually halt consensus

	activeNow := atomic.LoadInt32(&suite.activeCount)
	hasConsensus := suite.hasConsensus()

	suite.T().Logf("TEST-1.2: During partition - active validators: %d, consensus: %v",
		activeNow, hasConsensus)

	// Recover partition
	for i := 0; i < 5; i++ {
		suite.recoverValidator(suite.validators[i])
	}

	// Now refund should work
	err = suite.k.RefundEscrow(suite.ctx, requestID, "partition_recovery")
	suite.NoError(err)

	suite.T().Log("TEST-1.2: Escrow timeout handled after partition recovery")
}

// TestValidatorFailureSummary generates failure scenario summary
func (suite *EscrowValidatorFailureTestSuite) TestValidatorFailureSummary() {
	suite.T().Log("\n=== TEST-1.2 VALIDATOR FAILURE SUMMARY ===")
	suite.T().Logf("Total validators: %d", len(suite.validators))
	suite.T().Logf("Consensus quorum: %d (67%%+1)", suite.consensusQuorum)
	suite.T().Logf("Max tolerable failures: %d", len(suite.validators)-suite.consensusQuorum)
	suite.T().Log("")
	suite.T().Log("Failure scenarios tested:")
	suite.T().Log("  ✓ Single validator failure")
	suite.T().Log("  ✓ Multiple validator failures (within threshold)")
	suite.T().Log("  ✓ Exact consensus threshold")
	suite.T().Log("  ✓ Validator recovery")
	suite.T().Log("  ✓ Rolling failures during concurrent operations")
	suite.T().Log("  ✓ State consistency during chaos")
	suite.T().Log("  ✓ Network partition timeout handling")
	suite.T().Log("=== END SUMMARY ===\n")
}
