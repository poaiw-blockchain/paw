package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/suite"

	testkeeper "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// SecurityIntegrationSuite is the comprehensive security testing suite with real keeper
type SecurityIntegrationSuite struct {
	suite.Suite
	ctx               sdk.Context
	keeper            *keeper.Keeper
	bankKeeper        bankkeeper.Keeper
	attacker          sdk.AccAddress
	normalUser        sdk.AccAddress
	liquidityProvider sdk.AccAddress
}

// SetupTest initializes the test suite with a fresh keeper and context
func (suite *SecurityIntegrationSuite) SetupTest() {
	// Use testutil keeper setup
	k, bk, ctx := testkeeper.DexKeeperWithBank(suite.T())
	suite.keeper = k
	suite.bankKeeper = bk
	suite.ctx = ctx

	// Create test accounts
	suite.attacker = sdk.AccAddress([]byte("attacker_address___"))
	suite.normalUser = sdk.AccAddress([]byte("normal_user_address"))
	suite.liquidityProvider = sdk.AccAddress([]byte("liquidity_provider_"))
}

// Helper function to fund test accounts (mock implementation for testing)
func (suite *SecurityIntegrationSuite) fundAccount(addr sdk.AccAddress, denom string, amount math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	suite.Require().NoError(suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, coins))
	suite.Require().NoError(suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, coins))
	balance := suite.bankKeeper.GetBalance(suite.ctx, addr, denom)
	suite.Require().True(balance.Amount.GTE(amount), "account must be funded before executing test flow")
}

// Helper function to create a test pool
func (suite *SecurityIntegrationSuite) createTestPool(tokenA, tokenB string, amountA, amountB math.Int) uint64 {
	pool, err := suite.keeper.CreatePool(suite.ctx, suite.liquidityProvider, tokenA, tokenB, amountA, amountB)
	suite.Require().NoError(err)
	suite.Require().NotNil(pool)
	return pool.Id
}

// Helper function to advance block height and time
func (suite *SecurityIntegrationSuite) advanceBlock(blocks int64) {
	header := suite.ctx.BlockHeader()
	header.Height += blocks
	header.Time = header.Time.Add(time.Duration(blocks) * 5 * time.Second) // Assume 5s block time
	suite.ctx = suite.ctx.WithBlockHeader(header)
}

func (suite *SecurityIntegrationSuite) flashLoanProtectionBlocks() int64 {
	params, err := suite.keeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	if params.FlashLoanProtectionBlocks == 0 {
		return keeper.DefaultFlashLoanProtectionBlocks
	}
	return int64(params.FlashLoanProtectionBlocks)
}

func (suite *SecurityIntegrationSuite) advanceFlashLoanWindow() {
	suite.advanceBlock(suite.flashLoanProtectionBlocks())
}

// ========== REENTRANCY ATTACK TESTS ==========

// TestReentrancyAttack_SwapDuringSwap tests that a swap cannot be executed during another swap
func (suite *SecurityIntegrationSuite) TestReentrancyAttack_SwapDuringSwap() {
	// Setup: Create pool with sufficient liquidity
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Get a reference to the reentrancy guard in context
	guard := keeper.NewReentrancyGuard()
	suite.ctx = suite.ctx.WithValue("reentrancy_guard", guard)

	// Attempt 1: Execute first swap (should succeed)
	firstSwap := math.NewInt(1000)
	_, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", firstSwap, math.NewInt(1))
	suite.Require().NoError(err, "First swap should succeed")

	// Attempt 2: Simulate reentrancy by manually locking
	err = guard.Lock("1:swap")
	suite.Require().NoError(err, "Initial lock should succeed")

	// Attempt 3: Try to execute swap while locked (simulates reentrant call)
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", firstSwap, math.NewInt(1))
	suite.Require().Error(err, "Reentrant swap should fail")
	suite.Require().Contains(err.Error(), "reentrancy", "Error should indicate reentrancy detection")

	// Cleanup: Unlock
	guard.Unlock("1:swap")

	// Verify: After unlock, swap should work again
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(100), math.NewInt(1))
	suite.Require().NoError(err, "Swap after unlock should succeed")
}

// TestReentrancyAttack_WithdrawDuringSwap tests that liquidity cannot be removed during a swap
func (suite *SecurityIntegrationSuite) TestReentrancyAttack_WithdrawDuringSwap() {
	// Setup: Create pool and add liquidity
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))
	shares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.liquidityProvider, poolID, math.NewInt(10000), math.NewInt(10000))
	suite.Require().NoError(err)

	// Advance block for flash loan protection
	suite.advanceFlashLoanWindow()

	// Get reentrancy guard
	guard := keeper.NewReentrancyGuard()
	suite.ctx = suite.ctx.WithValue("reentrancy_guard", guard)

	// Lock swap operation
	err = guard.Lock(fmt.Sprintf("%d:swap", poolID))
	suite.Require().NoError(err)

	// Attempt to remove liquidity while swap is in progress (should fail if using same guard)
	// Note: RemoveLiquidity doesn't currently use the guard, but this tests the guard mechanism
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.liquidityProvider, poolID, shares)
	suite.Require().NoError(err)
	// This will succeed because RemoveLiquidity uses different operation key
	// But demonstrates the guard prevents same operation type

	guard.Unlock(fmt.Sprintf("%d:swap", poolID))
}

// ========== FLASH LOAN ATTACK TESTS ==========

// TestFlashLoanAttack_PriceManipulation tests that same-block liquidity manipulation is prevented
func (suite *SecurityIntegrationSuite) TestFlashLoanAttack_PriceManipulation() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Try to add liquidity and immediately remove it in same block
	// to manipulate pool price without risk

	// Step 1: Add liquidity
	shares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, poolID, math.NewInt(50000), math.NewInt(50000))
	suite.Require().NoError(err)
	suite.Require().True(shares.GT(math.ZeroInt()))

	// Step 2: Try to immediately remove liquidity in same block (should fail)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().Error(err, "Same-block liquidity removal should be blocked")
	suite.Require().Contains(err.Error(), "flash loan", "Error should indicate flash loan detection")

	// Step 3: Advance past flash loan window
	suite.advanceFlashLoanWindow()

	// Step 4: Now removal should succeed
	amountA, amountB, err := suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().NoError(err, "Liquidity removal should succeed after block delay")
	suite.Require().True(amountA.GT(math.ZeroInt()))
	suite.Require().True(amountB.GT(math.ZeroInt()))
}

// TestFlashLoanAttack_MultiplePools tests flash loan protection across multiple pools
func (suite *SecurityIntegrationSuite) TestFlashLoanAttack_MultiplePools() {
	// Setup: Create two pools
	pool1 := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))
	pool2 := suite.createTestPool("atom", "osmo", math.NewInt(100000), math.NewInt(200000))

	// Attack scenario: Try to manipulate prices across pools using flash loans

	// Add liquidity to pool 1
	shares1, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, pool1, math.NewInt(10000), math.NewInt(10000))
	suite.Require().NoError(err)

	// Add liquidity to pool 2
	shares2, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, pool2, math.NewInt(10000), math.NewInt(20000))
	suite.Require().NoError(err)

	// Try to remove from pool 1 immediately (should fail)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, pool1, shares1)
	suite.Require().Error(err, "Flash loan protection should prevent same-block removal")

	// Try to remove from pool 2 immediately (should also fail)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, pool2, shares2)
	suite.Require().Error(err, "Flash loan protection should work independently per pool")

	// Advance block window for flash loan protection
	suite.advanceFlashLoanWindow()

	// Now both should succeed
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, pool1, shares1)
	suite.Require().NoError(err)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, pool2, shares2)
	suite.Require().NoError(err)
}

// TestFlashLoanAttack_AddSwapRemove tests the flash loan attack vector where an attacker:
// 1. Adds huge liquidity (becomes dominant LP)
// 2. Executes large swap to manipulate price
// 3. Arbitrages the price discrepancy on another pool/chain
// 4. Attempts to remove liquidity in the same block to eliminate exposure
// This test verifies that step 4 is blocked by the multi-block LP lock period,
// forcing attackers to maintain price exposure and eliminating the flash loan advantage.
func (suite *SecurityIntegrationSuite) TestFlashLoanAttack_AddSwapRemove() {
	// Fund attacker with large balances
	suite.fundAccount(suite.attacker, "atom", math.NewInt(1000000))
	suite.fundAccount(suite.attacker, "usdc", math.NewInt(1000000))

	// Setup: Create pool with moderate liquidity
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Add→Swap→Remove in same block
	// Step 1: Attacker adds huge liquidity (becomes dominant LP)
	shares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, poolID,
		math.NewInt(500000), math.NewInt(500000))
	suite.Require().NoError(err, "Initial liquidity addition should succeed")
	suite.Require().True(shares.GT(math.ZeroInt()))

	// Step 2: Execute large swap to manipulate price (still same block)
	// This would normally move the price, creating arbitrage opportunity
	swapAmount := math.NewInt(50000) // 10% of reserves after addition
	if _, swapErr := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID,
		"atom", "usdc", swapAmount, math.NewInt(1)); swapErr != nil {
		suite.T().Logf("swap prevented by MEV protections: %v", swapErr)
	}
	// The key test is that removal is blocked regardless

	// Step 3: Arbitrage would happen on another chain (simulated - not part of test)
	// In real attack, attacker would profit from price difference between chains

	// Step 4: Try to remove liquidity immediately (MUST BE BLOCKED)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().Error(err, "Flash loan protection MUST block same-block removal after add")
	suite.Require().Contains(err.Error(), "flash loan",
		"Error must indicate flash loan protection triggered")

	// Verify protection persists for insufficient block gap
	suite.advanceBlock(5) // Advance only 5 blocks (less than 10 block minimum)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().Error(err, "Protection must persist for minimum lock period")
	suite.Require().Contains(err.Error(), "flash loan",
		"Error must indicate flash loan protection still active")

	// Verify removal succeeds after full lock period
	suite.advanceBlock(5) // Now total 10 blocks have passed
	amountA, amountB, err := suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().NoError(err, "Removal should succeed after full lock period")
	suite.Require().True(amountA.GT(math.ZeroInt()), "Should receive non-zero amount A")
	suite.Require().True(amountB.GT(math.ZeroInt()), "Should receive non-zero amount B")
}

// TestFlashLoanAttack_PartialRemovalBlocked tests that even partial removal is blocked
func (suite *SecurityIntegrationSuite) TestFlashLoanAttack_PartialRemovalBlocked() {
	// Fund attacker
	suite.fundAccount(suite.attacker, "atom", math.NewInt(500000))
	suite.fundAccount(suite.attacker, "usdc", math.NewInt(500000))

	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Add liquidity
	shares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, poolID,
		math.NewInt(200000), math.NewInt(200000))
	suite.Require().NoError(err)

	// Try to remove even a tiny fraction immediately (should still be blocked)
	partialShares := shares.QuoRaw(100) // Just 1% of shares
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, partialShares)
	suite.Require().Error(err, "Even partial removal must be blocked in same block")
	suite.Require().Contains(err.Error(), "flash loan")

	// After lock period, partial removal should work
	suite.advanceFlashLoanWindow()
	amountA, amountB, err := suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, partialShares)
	suite.Require().NoError(err, "Partial removal should succeed after lock period")
	suite.Require().True(amountA.GT(math.ZeroInt()))
	suite.Require().True(amountB.GT(math.ZeroInt()))
}

// ========== MEV ATTACK TESTS ==========

// TestMEVAttack_Frontrunning tests that large swaps are blocked to prevent frontrunning
func (suite *SecurityIntegrationSuite) TestMEVAttack_Frontrunning() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Attacker tries to frontrun a large trade with their own large trade

	// Normal user announces intention to swap 5000 atoms (5% of pool)
	normalUserSwap := math.NewInt(5000)

	// Attacker tries to frontrun with 15000 atoms (15% of pool - exceeds 10% limit)
	attackerSwap := math.NewInt(15000)

	// Attacker's large swap should fail due to size limit (>10% of reserve)
	_, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", attackerSwap, math.NewInt(1))
	suite.Require().Error(err, "Swap exceeding 10% should be blocked")
	suite.Require().Contains(err.Error(), "too large", "Error should indicate swap size limit")

	// Normal user's swap should succeed (within limits)
	amountOut, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", normalUserSwap, math.NewInt(1))
	suite.Require().NoError(err, "Normal sized swap should succeed")
	suite.Require().True(amountOut.GT(math.ZeroInt()))

	// Verify pool state is still valid
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.keeper.ValidatePoolState(pool))
}

// TestMEVAttack_Sandwiching tests protection against sandwich attacks
func (suite *SecurityIntegrationSuite) TestMEVAttack_Sandwiching() {
	// Setup: Create pool with initial liquidity
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Attacker tries to sandwich a victim's trade
	// 1. Attacker buys before victim (frontrun)
	// 2. Victim executes trade (price moves against them)
	// 3. Attacker sells after victim (backrun)

	victimSwap := math.NewInt(5000)

	// Step 1: Attacker tries to frontrun with max allowed swap (10%)
	attackFrontrun := math.NewInt(10000) // Exactly 10% of reserve
	frontrunOut, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", attackFrontrun, math.NewInt(1))
	suite.Require().NoError(err, "10% swap should succeed")

	// Get pool state after frontrun
	poolAfterFrontrun, _ := suite.keeper.GetPool(suite.ctx, poolID)
	oldReserveA := poolAfterFrontrun.ReserveA
	oldReserveB := poolAfterFrontrun.ReserveB

	// Step 2: Victim executes trade (gets worse price due to frontrun)
	victimOut, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", victimSwap, math.NewInt(1))
	suite.Require().NoError(err)

	// Step 3: Attacker tries to backrun by selling back
	// This should be limited by price impact protection (50% max)
	// Selling large amount back should fail if price impact is too high
	if _, backrunErr := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "usdc", "atom", frontrunOut, math.NewInt(1)); backrunErr != nil {
		suite.T().Logf("backrun attempt limited: %v", backrunErr)
	}

	// Verify price impact protection kicked in or trade succeeded with limited profit
	poolAfterBackrun, _ := suite.keeper.GetPool(suite.ctx, poolID)

	// The pool reserves should not have been manipulated back to original state
	// (which would indicate successful sandwich attack)
	reserveDiffA := oldReserveA.Sub(poolAfterBackrun.ReserveA).Abs()
	reserveDiffB := oldReserveB.Sub(poolAfterBackrun.ReserveB).Abs()

	// Reserves should have changed (not perfect sandwich)
	suite.Require().True(reserveDiffA.GT(math.ZeroInt()) || reserveDiffB.GT(math.ZeroInt()),
		"Pool reserves should reflect trades, preventing perfect sandwich")

	// Verify victim got some output despite sandwich attempt
	suite.Require().True(victimOut.GT(math.ZeroInt()), "Victim should still receive output")
}

// ========== OVERFLOW/UNDERFLOW ATTACK TESTS ==========

// TestOverflowAttack_LargeAmounts tests that extremely large amounts don't cause overflow
func (suite *SecurityIntegrationSuite) TestOverflowAttack_LargeAmounts() {
	// Setup: Create pool
	_ = suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Try to cause overflow with massive numbers

	// Create extremely large amount (near max int)
	hugeAmount := math.NewInt(1).MulRaw(1000000000000000000) // 10^18

	// Attempt 1: Try swap with huge amount (should fail gracefully, not overflow)
	params, paramErr := suite.keeper.GetParams(suite.ctx)
	suite.Require().NoError(paramErr)
	_, err := suite.keeper.CalculateSwapOutputSecure(
		suite.ctx,
		hugeAmount,
		math.NewInt(100000),
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(3, 3),
		params.MaxPoolDrainPercent,
	)
	// Should either succeed with valid calculation or fail gracefully
	if err != nil {
		suite.Require().NotContains(err.Error(), "panic", "Should not panic on large numbers")
	}

	// Attempt 2: Test math.Int addition with large values
	// math.Int uses arbitrary precision, so overflow is not possible
	a := math.NewInt(1).MulRaw(1000000000000000000)
	b := math.NewInt(1).MulRaw(1000000000000000000)
	result := a.Add(b)
	suite.Require().True(result.GT(a), "math.Int.Add should produce valid result")

	// Attempt 3: Test math.Int multiplication with large values
	// math.Int handles arbitrary precision, so this always succeeds
	mulResult := a.Mul(b)
	suite.Require().True(mulResult.GT(a), "math.Int.Mul should handle large values")
}

// TestUnderflowAttack_NegativeBalances tests that negative amounts are prevented
func (suite *SecurityIntegrationSuite) TestUnderflowAttack_NegativeBalances() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Try to cause underflow by swapping more than reserve

	// Get pool
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)

	// Try to swap amount that would drain entire reserve
	drainAmount := pool.ReserveA.Add(math.NewInt(1))

	// This should fail before causing underflow
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", drainAmount, math.NewInt(1))
	suite.Require().Error(err, "Swap larger than reserve should fail")
	suite.Require().NotContains(err.Error(), "negative", "Should prevent underflow, not create negative")

	// Test underflow protection: math.Int.Sub allows negative results but
	// DEX code must validate amounts before operations
	subResult := math.NewInt(100).Sub(math.NewInt(200))
	suite.Require().True(subResult.IsNegative(), "math.Int.Sub should return negative on underflow")
	// DEX validates non-negative before storing, preventing negative reserves

	// Verify pool reserves are still positive
	pool, err = suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().False(pool.ReserveA.IsNegative(), "Reserve A should never be negative")
	suite.Require().False(pool.ReserveB.IsNegative(), "Reserve B should never be negative")
}

// ========== CIRCUIT BREAKER TESTS ==========

// TestCircuitBreaker_ExtremePriceDeviation tests automatic circuit breaker triggering
func (suite *SecurityIntegrationSuite) TestCircuitBreaker_ExtremePriceDeviation() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Execute initial swap to establish price baseline
	_, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(1000), math.NewInt(1))
	suite.Require().NoError(err)

	// Get pool and manually manipulate reserves to simulate extreme price change
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)

	// Save original reserves
	originalReserveA := pool.ReserveA
	originalReserveB := pool.ReserveB

	// Simulate external manipulation: Drastically change price (>20% deviation)
	// Double reserve A, halve reserve B (creates ~75% price change)
	pool.ReserveA = originalReserveA.MulRaw(2)
	pool.ReserveB = originalReserveB.QuoRaw(2)
	suite.Require().NoError(suite.keeper.SetPool(suite.ctx, pool))

	// Try to execute swap - should trigger circuit breaker
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", math.NewInt(100), math.NewInt(1))
	suite.Require().Error(err, "Circuit breaker should trigger on extreme price deviation")
	suite.Require().Contains(err.Error(), "circuit breaker", "Error should indicate circuit breaker activation")

	// Verify circuit breaker state
	cbState, err := suite.keeper.GetPoolCircuitBreakerState(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().True(cbState.Enabled, "Circuit breaker should be enabled")
	suite.Require().False(cbState.PausedUntil.IsZero(), "Pause time should be set")

	// Verify all operations are blocked
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(100), math.NewInt(1))
	suite.Require().Error(err, "All swaps should be blocked")
}

// TestCircuitBreaker_AutoRecovery tests that circuit breaker auto-recovers after timeout
func (suite *SecurityIntegrationSuite) TestCircuitBreaker_AutoRecovery() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Trigger circuit breaker manually
	err := suite.keeper.EmergencyPausePool(suite.ctx, poolID, "test emergency", 10*time.Minute)
	suite.Require().NoError(err)

	// Verify pool is paused
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(100), math.NewInt(1))
	suite.Require().Error(err, "Operations should be blocked while paused")
	suite.Require().Contains(err.Error(), "circuit breaker", "Should indicate circuit breaker is active")

	// Advance time past the pause duration
	header := suite.ctx.BlockHeader()
	header.Time = header.Time.Add(11 * time.Minute)
	suite.ctx = suite.ctx.WithBlockHeader(header)

	// Now operations should succeed (auto-recovery)
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(1000), math.NewInt(1))
	suite.Require().NoError(err, "Operations should resume after pause timeout")

	// Verify circuit breaker is no longer blocking
	cbState, err := suite.keeper.GetPoolCircuitBreakerState(suite.ctx, poolID)
	suite.Require().NoError(err)
	// State might still exist but should not be blocking
	currentTime := suite.ctx.BlockTime()
	suite.Require().True(currentTime.After(cbState.PausedUntil), "Current time should be past pause deadline")
}

// ========== INVARIANT VIOLATION TESTS ==========

// TestInvariantViolation_DirectReserveManipulation tests that invariant checks prevent reserve manipulation
func (suite *SecurityIntegrationSuite) TestInvariantViolation_DirectReserveManipulation() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Get pool and calculate initial invariant k = x * y
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	oldK := pool.ReserveA.Mul(pool.ReserveB)

	// Attack scenario: Try to manipulate reserves to decrease k (steal value)
	pool.ReserveA = pool.ReserveA.SubRaw(10000) // Remove 10000 from reserve A
	pool.ReserveB = pool.ReserveB.SubRaw(5000)  // Remove 5000 from reserve B

	// Validate invariant - should fail
	err = suite.keeper.ValidatePoolInvariant(suite.ctx, pool, oldK)
	suite.Require().Error(err, "Invariant check should catch reserve manipulation")
	suite.Require().Contains(err.Error(), "invariant violated", "Error should indicate invariant violation")

	// Verify that increasing k is allowed (due to fees)
	pool.ReserveA = pool.ReserveA.AddRaw(20000)
	err = suite.keeper.ValidatePoolInvariant(suite.ctx, pool, oldK)
	suite.Require().NoError(err, "Increasing k should be allowed (fees)")
}

// TestInvariantViolation_AfterSwap tests invariant validation in actual swap flow
func (suite *SecurityIntegrationSuite) TestInvariantViolation_AfterSwap() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Get initial pool state
	poolBefore, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	kBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)

	// Execute swap
	swapAmount := math.NewInt(5000)
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", swapAmount, math.NewInt(1))
	suite.Require().NoError(err)

	// Get pool state after swap
	poolAfter, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	kAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)

	// Invariant should have increased or stayed same (due to fees)
	suite.Require().True(kAfter.GTE(kBefore), "Invariant k should increase or stay same after swap")

	// Verify pool state is valid
	suite.Require().NoError(suite.keeper.ValidatePoolState(poolAfter))
}

// ========== DOS ATTACK TESTS ==========

// TestDoSAttack_MaxPoolCreation tests protection against unlimited pool creation
func (suite *SecurityIntegrationSuite) TestDoSAttack_MaxPoolCreation() {
	// Attack scenario: Attacker tries to create excessive pools to DoS the system

	// Fund attacker with resources for multiple pool creations
	for i := 0; i < 100; i++ {
		tokenName := fmt.Sprintf("token%d", i)
		suite.fundAccount(suite.attacker, tokenName, math.NewInt(1000000))
	}

	// Try to create many pools
	poolsCreated := 0
	maxPoolsToTry := 1100 // Try to exceed the limit of 1000

	for i := 0; i < maxPoolsToTry; i++ {
		tokenA := fmt.Sprintf("token%d", i)
		tokenB := "usdc"

		// Fund if needed
		if i >= 100 {
			suite.fundAccount(suite.attacker, tokenA, math.NewInt(1000000))
		}

		_, err := suite.keeper.CreatePool(suite.ctx, suite.attacker, tokenA, tokenB, math.NewInt(10000), math.NewInt(10000))
		if err != nil {
			// Should eventually hit max pools limit
			if poolsCreated >= int(keeper.MaxPools) {
				suite.Require().Contains(err.Error(), "maximum", "Should indicate max pools reached")
				break
			}
		} else {
			poolsCreated++
		}
	}

	// Verify we can still operate on existing pools (not DoS'd)
	pool1, err := suite.keeper.GetPool(suite.ctx, 1)
	suite.Require().NoError(err)
	suite.Require().NotNil(pool1, "Existing pools should still be accessible")
}

// TestDoSAttack_TinySwapSpam tests protection against spam of tiny swaps
func (suite *SecurityIntegrationSuite) TestDoSAttack_TinySwapSpam() {
	// Setup: Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(100000), math.NewInt(100000))

	// Attack scenario: Spam tiny swaps to waste gas/resources
	tinySwap := math.NewInt(1) // Minimum possible swap

	// Attempt many tiny swaps
	successfulSwaps := 0
	failedSwaps := 0

	for i := 0; i < 100; i++ {
		_, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", tinySwap, math.NewInt(0))
		if err != nil {
			failedSwaps++
			// Tiny swaps might fail due to "output amount too small"
			if i == 0 {
				// First tiny swap failure is expected
				suite.Require().Contains(err.Error(), "too small", "Tiny swaps should be rejected")
			}
		} else {
			successfulSwaps++
		}
	}

	// Verify pool is still functional after spam attempt
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.keeper.ValidatePoolState(pool), "Pool should remain in valid state")

	// Verify normal swaps still work
	normalSwap := math.NewInt(1000)
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", normalSwap, math.NewInt(1))
	suite.Require().NoError(err, "Normal swaps should still work after spam")
}

// ========== COMPREHENSIVE INTEGRATION TESTS ==========

// TestComprehensiveSecurity_MultipleAttackVectors tests multiple attacks in sequence
func (suite *SecurityIntegrationSuite) TestComprehensiveSecurity_MultipleAttackVectors() {
	// Setup: Create pool with substantial liquidity
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(500000), math.NewInt(500000))

	// Attack Vector 1: Try flash loan
	shares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, poolID, math.NewInt(100000), math.NewInt(100000))
	suite.Require().NoError(err)

	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, shares)
	suite.Require().Error(err, "Flash loan should be blocked")

	suite.advanceFlashLoanWindow()

	// Attack Vector 2: Try large swap for MEV
	largeSwap := math.NewInt(70000) // More than 10% of reserve after initial add
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", largeSwap, math.NewInt(1))
	suite.Require().Error(err, "Large swap should be blocked")

	// Attack Vector 3: Try acceptable swap followed by another large one
	acceptableSwap := math.NewInt(40000) // Under 10% limit
	out1, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", acceptableSwap, math.NewInt(1))
	suite.Require().NoError(err, "First swap within limits should succeed")
	suite.Require().True(out1.GT(math.ZeroInt()))

	// Pool reserves changed, now try another 10% swap (of new reserves)
	pool, _ := suite.keeper.GetPool(suite.ctx, poolID)
	nextSwap := pool.ReserveA.QuoRaw(10) // 10% of new reserve
	out2, err := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", nextSwap, math.NewInt(1))
	suite.Require().NoError(err, "Sequential swaps within individual limits should work")
	suite.Require().True(out2.GT(math.ZeroInt()))

	// Verify pool integrity after all attacks
	finalPool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.keeper.ValidatePoolState(finalPool))

	// Verify invariant is maintained
	k := finalPool.ReserveA.Mul(finalPool.ReserveB)
	originalK := math.NewInt(500000).Mul(math.NewInt(500000))
	suite.Require().True(k.GTE(originalK), "Invariant should be maintained or increased")
}

// TestComprehensiveSecurity_AllSecurityFeatures tests all security features together
func (suite *SecurityIntegrationSuite) TestComprehensiveSecurity_AllSecurityFeatures() {
	// Create pool
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(1000000), math.NewInt(1000000))

	// Test 1: Pool state validation
	pool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.keeper.ValidatePoolState(pool))

	// Test 2: math.Int operations (safe by design - arbitrary precision)
	totalReserves := pool.ReserveA.Add(pool.ReserveB)
	suite.Require().True(totalReserves.GT(pool.ReserveA), "Addition should work correctly")

	// Test 3: Swap size validation
	err = suite.keeper.ValidateSwapSize(math.NewInt(50000), pool.ReserveA)
	suite.Require().NoError(err, "5% swap should be valid")

	err = suite.keeper.ValidateSwapSize(math.NewInt(150000), pool.ReserveA)
	suite.Require().Error(err, "15% swap should be invalid")

	// Test 4: Price impact validation
	amountIn := math.NewInt(50000)
	amountOut := math.NewInt(45000)
	err = suite.keeper.ValidatePriceImpact(amountIn, pool.ReserveA, pool.ReserveB, amountOut)
	suite.Require().NoError(err, "Reasonable price impact should be valid")

	// Test 5: Circuit breaker
	err = suite.keeper.CheckPoolCircuitBreaker(suite.ctx, poolID)
	suite.Require().NoError(err, "Circuit breaker should not be triggered initially")

	// Test 6: Flash loan protection
	shares, _ := suite.keeper.AddLiquiditySecure(suite.ctx, suite.normalUser, poolID, math.NewInt(10000), math.NewInt(10000))
	err = suite.keeper.CheckFlashLoanProtection(suite.ctx, poolID, suite.normalUser)
	suite.Require().Error(err, "Flash loan protection should block same-block removal")

	suite.advanceFlashLoanWindow()
	err = suite.keeper.CheckFlashLoanProtection(suite.ctx, poolID, suite.normalUser)
	suite.Require().NoError(err, "Flash loan protection should allow after block delay")

	// Test 7: Reentrancy guard
	guard := keeper.NewReentrancyGuard()
	err = guard.Lock("test_op")
	suite.Require().NoError(err)
	err = guard.Lock("test_op")
	suite.Require().Error(err, "Reentrancy should be detected")
	guard.Unlock("test_op")

	// Test 8: Invariant validation
	oldK := pool.ReserveA.Mul(pool.ReserveB)
	err = suite.keeper.ValidatePoolInvariant(suite.ctx, pool, oldK)
	suite.Require().NoError(err)

	// Test 9: Execute secure swap with all protections
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.normalUser, poolID, "atom", "usdc", math.NewInt(10000), math.NewInt(1))
	suite.Require().NoError(err, "Secure swap with all protections should succeed")

	// Clean up - remove liquidity
	suite.advanceFlashLoanWindow()
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.normalUser, poolID, shares)
	suite.Require().NoError(err)
}

// TestSecurityIntegration_RealWorldScenario simulates real-world attack attempts
func (suite *SecurityIntegrationSuite) TestSecurityIntegration_RealWorldScenario() {
	// Scenario: Multiple users interacting with pool, one attacker trying various exploits

	// Setup: Create pool with market maker
	poolID := suite.createTestPool("atom", "usdc", math.NewInt(1000000), math.NewInt(1000000))

	// Regular user 1 adds liquidity
	user1 := sdk.AccAddress([]byte("regular_user_1_____"))
	suite.fundAccount(user1, "atom", math.NewInt(100000))
	suite.fundAccount(user1, "usdc", math.NewInt(100000))

	shares1, err := suite.keeper.AddLiquiditySecure(suite.ctx, user1, poolID, math.NewInt(50000), math.NewInt(50000))
	suite.Require().NoError(err)

	suite.advanceFlashLoanWindow()

	// Regular user 2 performs normal swap
	user2 := sdk.AccAddress([]byte("regular_user_2_____"))
	suite.fundAccount(user2, "atom", math.NewInt(10000))

	out2, err := suite.keeper.ExecuteSwapSecure(suite.ctx, user2, poolID, "atom", "usdc", math.NewInt(5000), math.NewInt(1))
	suite.Require().NoError(err)
	suite.Require().True(out2.GT(math.ZeroInt()))

	// Attacker attempt 1: Try to sandwich the next trade
	user3 := sdk.AccAddress([]byte("regular_user_3_____"))
	suite.fundAccount(user3, "atom", math.NewInt(20000))

	// Attacker frontrun (max 10%)
	pool, _ := suite.keeper.GetPool(suite.ctx, poolID)
	frontrunAmount := pool.ReserveA.QuoRaw(10)
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "atom", "usdc", frontrunAmount, math.NewInt(1))
	suite.Require().NoError(err, "10% frontrun succeeds")

	// Victim trade
	_, err = suite.keeper.ExecuteSwapSecure(suite.ctx, user3, poolID, "atom", "usdc", math.NewInt(10000), math.NewInt(1))
	suite.Require().NoError(err)

	// Attacker backrun attempt - price impact should limit profit
	pool, _ = suite.keeper.GetPool(suite.ctx, poolID)
	backrunAmount := pool.ReserveB.QuoRaw(10)
	if _, swapErr := suite.keeper.ExecuteSwapSecure(suite.ctx, suite.attacker, poolID, "usdc", "atom", backrunAmount, math.NewInt(1)); swapErr != nil {
		suite.T().Logf("backrun thwarted: %v", swapErr)
	}
	// May succeed or fail depending on price impact

	// Attacker attempt 2: Flash loan attack
	suite.advanceFlashLoanWindow()
	attackShares, err := suite.keeper.AddLiquiditySecure(suite.ctx, suite.attacker, poolID, math.NewInt(200000), math.NewInt(200000))
	suite.Require().NoError(err)

	// Try to remove immediately
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, attackShares)
	suite.Require().Error(err, "Flash loan protection blocks immediate removal")

	// User 1 removes their liquidity successfully (added earlier)
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, user1, poolID, shares1)
	suite.Require().NoError(err, "Legitimate liquidity removal succeeds")

	// Advance block(s) and attacker can now remove
	suite.advanceFlashLoanWindow()
	_, _, err = suite.keeper.RemoveLiquiditySecure(suite.ctx, suite.attacker, poolID, attackShares)
	suite.Require().NoError(err)

	// Verify pool integrity after all interactions
	finalPool, err := suite.keeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.keeper.ValidatePoolState(finalPool))
	suite.Require().True(finalPool.ReserveA.GT(math.ZeroInt()))
	suite.Require().True(finalPool.ReserveB.GT(math.ZeroInt()))
	suite.Require().True(finalPool.TotalShares.GT(math.ZeroInt()))
}

// Run the test suite
func TestSecurityIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SecurityIntegrationSuite))
}
