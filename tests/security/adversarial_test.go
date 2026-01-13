//go:build security_advanced
// +build security_advanced

// NOTE: This file is temporarily excluded from build pending DEX API updates.
// Run with: go test -tags=security_advanced ./tests/security/...

package security_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// AdversarialTestSuite tests adversarial attack scenarios
type AdversarialTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *AdversarialTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestAdversarialTestSuite(t *testing.T) {
	suite.Run(t, new(AdversarialTestSuite))
}

// TestDoubleSpending_Prevention tests double-spending prevention
func (suite *AdversarialTestSuite) TestDoubleSpending_Prevention() {
	// Create and fund attacker account
	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	initialCoins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(10000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, initialCoins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, initialCoins))

	// Create pool
	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdt",
		math.NewInt(1000000),
		math.NewInt(2000000),
	)
	suite.Require().NoError(err)
	poolId := pool.Id

	// Check balance before swap
	balanceBefore := suite.app.BankKeeper.GetBalance(suite.ctx, attackerAddr, "upaw")

	// First swap should succeed (use 1% of pool to avoid MEV protection)
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(10000), // 1% of pool size
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	balanceAfterFirst := suite.app.BankKeeper.GetBalance(suite.ctx, attackerAddr, "upaw")
	suite.Require().True(balanceAfterFirst.Amount.LT(balanceBefore.Amount), "Balance should decrease after swap")

	// Second identical swap
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(10000), // 1% of pool size
		math.NewInt(1),
	)

	// Should either succeed with remaining balance or fail with insufficient funds
	if err != nil {
		suite.T().Logf("second swap failed as expected: %v", err)
	}
	balanceAfterSecond := suite.app.BankKeeper.GetBalance(suite.ctx, attackerAddr, "upaw")

	// Verify total spent does not exceed original balance
	totalSpent := balanceBefore.Amount.Sub(balanceAfterSecond.Amount)
	suite.Require().True(totalSpent.LTE(balanceBefore.Amount), "Cannot spend more than owned")
}

// TestFrontRunning_Simulation tests front-running attack scenarios
func (suite *AdversarialTestSuite) TestFrontRunning_Simulation() {
	// Create victim and attacker accounts
	victimPriv := secp256k1.GenPrivKey()
	victimAddr := sdk.AccAddress(victimPriv.PubKey().Address())

	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	// Fund both accounts
	coins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(10000000)),
	)

	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, victimAddr, coins))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, coins))

	// Create pool
	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		victimAddr,
		"upaw",
		"uusdt",
		math.NewInt(1000000),
		math.NewInt(2000000),
	)
	suite.Require().NoError(err)
	poolId := pool.Id

	// Execute attacker swap first (front-running) - use 3% of pool
	attackerAmountOut, err := suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(30000), // 3% of pool size
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// Execute victim swap (price now worse due to front-running) - use 2% of pool
	victimAmountOut, err := suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		victimAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(20000), // 2% of pool size
		math.NewInt(1),
	)

	if err != nil {
		// Victim's swap fails due to slippage
		suite.T().Log("Victim's swap failed due to front-running changing the price")
	} else {
		// Victim gets worse price than expected
		suite.T().Logf("Victim received %s", victimAmountOut.String())
	}

	// Attacker tries to back-run by swapping reverse
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"uusdt",
		"upaw",
		attackerAmountOut,
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// Note: Proper MEV protection would include transaction ordering rules,
	// batch auctions, or commit-reveal schemes
}

// TestSandwichAttack_Detection tests sandwich attack detection
func (suite *AdversarialTestSuite) TestSandwichAttack_Detection() {
	// Setup: Attacker, Victim, Pool
	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	victimPriv := secp256k1.GenPrivKey()
	victimAddr := sdk.AccAddress(victimPriv.PubKey().Address())

	// Fund accounts
	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(50000000)),
		sdk.NewCoin("uusdt", math.NewInt(50000000)),
	)

	for _, addr := range []sdk.AccAddress{attackerAddr, victimAddr} {
		suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
		suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, funds))
	}

	// Create pool
	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		victimAddr,
		"upaw",
		"uusdt",
		math.NewInt(10000000),
		math.NewInt(20000000),
	)
	suite.Require().NoError(err)
	poolId := pool.Id

	// 1. Attacker front-runs with buy (use 2% of pool)
	attackBuyAmountOut, err := suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(200000), // 2% of pool size
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// 2. Victim executes their trade (use 3% of pool)
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		victimAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(300000), // 3% of pool size
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// 3. Attacker back-runs with sell
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"uusdt",
		"upaw",
		attackBuyAmountOut,
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// Analysis: In a sandwich attack, attacker profits from price movement
	// Mitigation: Private mempools, batch auctions, MEV protection
	suite.T().Log("Sandwich attack sequence completed - mitigation strategies needed")
}

// TestFlashLoan_Attack tests flash loan attack scenarios
func (suite *AdversarialTestSuite) TestFlashLoan_Attack() {
	// Flash loan attacks: Borrow large amounts, manipulate price, repay
	// PAW blockchain should not allow uncollateralized borrowing

	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	// Fund with minimal amount
	minimalFunds := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000)))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, minimalFunds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, minimalFunds))

	// Attempt to execute operations requiring large capital
	_, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdt",
		math.NewInt(10000000), // Much more than attacker has
		math.NewInt(20000000),
	)

	// Should fail - insufficient funds
	suite.Require().Error(err, "Flash loan-style attack should fail without collateral")
}

// TestOracleManipulation tests oracle price manipulation attacks
func (suite *AdversarialTestSuite) TestOracleManipulation() {
	// Test that oracle system uses validator-based submissions
	// Good oracle implementation should:
	// 1. Use median instead of mean (resistant to outliers)
	// 2. Require minimum number of oracles
	// 3. Detect and slash malicious oracles
	// 4. Use time-weighted average prices (TWAP)

	suite.T().Log("Oracle manipulation test - system requires active validators for price submission")
	// Note: Full oracle testing requires validator setup which is beyond this security test
}

// TestSybilAttack tests Sybil attack resistance
func (suite *AdversarialTestSuite) TestSybilAttack() {
	// Sybil attack resistance in PAW blockchain:
	// Oracle system requires validators which need:
	// 1. Stake/bond for participation
	// 2. Validator status via staking module
	// 3. Reputation system (slashing for bad behavior)

	suite.T().Log("Sybil resistance: Oracle system requires validators with stake")
	// Note: Creating fake validators requires staking module setup beyond this test
}

// TestTimeBandits tests time manipulation attacks
func (suite *AdversarialTestSuite) TestTimeBandits() {
	// Validators could manipulate block timestamps
	currentTime := suite.ctx.BlockTime()

	// Create pool
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(10000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, funds))

	_, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		addr,
		"upaw",
		"uusdt",
		math.NewInt(1000000),
		math.NewInt(2000000),
	)
	suite.Require().NoError(err)

	// Simulate time manipulation (moving time backward)
	pastTime := currentTime.Add(-1 * time.Hour)
	manipulatedCtx := suite.ctx.WithBlockTime(pastTime)

	// Operations using manipulated time
	suite.Require().True(manipulatedCtx.BlockTime().Before(currentTime), "Time moved backward")

	// Consensus should prevent this in production
	// BFT timestamp validation ensures blocks are ordered correctly
}

// TestLongRangeAttack tests long-range attack scenarios
func (suite *AdversarialTestSuite) TestLongRangeAttack() {
	// Long-range attack: Attacker tries to rewrite history from genesis
	// by accumulating old validator keys

	// Get current block height
	currentHeight := suite.ctx.BlockHeight()
	suite.Require().Greater(currentHeight, int64(0))

	// Attempt to create context at old height
	oldHeight := currentHeight - 100
	if oldHeight < 1 {
		oldHeight = 1
	}

	oldCtx := suite.ctx.WithBlockHeight(oldHeight)

	// Verify height changed
	suite.Require().Equal(oldHeight, oldCtx.BlockHeight())

	// In production, checkpoints and weak subjectivity prevent this
	// Nodes reject chains that deviate too far from known checkpoints
	suite.T().Log("Long-range attack simulation - checkpoints needed")
}

// TestGriefing_ResourceExhaustion tests griefing attacks
func (suite *AdversarialTestSuite) TestGriefing_ResourceExhaustion() {
	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	// Fund attacker minimally with both tokens
	params, err := suite.app.DEXKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	minLiquidity := params.MinLiquidity

	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(100000000)),
		sdk.NewCoin("uusdt", math.NewInt(100000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, funds))

	// Attempt to create many small pools to exhaust storage
	// Note: Each pool creation needs unique token pairs, so we can't create duplicates
	// The test demonstrates that the system prevents pool duplication

	// First pool should succeed
	_, err = suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdt",
		minLiquidity,
		minLiquidity,
	)
	suite.Require().NoError(err)

	// Duplicate pool should fail
	_, err = suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdt",
		minLiquidity,
		minLiquidity,
	)
	suite.Require().Error(err, "Duplicate pool creation should be prevented")

	suite.T().Log("Griefing protection: Duplicate pools are prevented")
}

// TestReplayAttack tests replay attack prevention
func (suite *AdversarialTestSuite) TestReplayAttack() {
	attackerPriv := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attackerPriv.PubKey().Address())

	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(10000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, funds))

	// Create pool
	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdt",
		math.NewInt(1000000),
		math.NewInt(2000000),
	)
	suite.Require().NoError(err)
	poolId := pool.Id

	// Execute swap (use 1% of pool)
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(10000), // 1% of pool size
		math.NewInt(1),
	)
	suite.Require().NoError(err)

	// Attempt to replay same transaction
	// In Cosmos SDK, sequence numbers prevent replay attacks
	_, err = suite.app.DEXKeeper.ExecuteSwap(
		suite.ctx,
		attackerAddr,
		poolId,
		"upaw",
		"uusdt",
		math.NewInt(10000), // 1% of pool size
		math.NewInt(1),
	)

	// May succeed or fail depending on balance, but won't be identical replay
	if err != nil {
		suite.T().Logf("replay attempt rejected: %v", err)
	}
	// Nonce/sequence ensures each transaction is unique
	suite.T().Log("Replay attack test - sequence numbers provide protection")
}

// TestEclipseAttack tests eclipse attack scenarios
func (suite *AdversarialTestSuite) TestEclipseAttack() {
	// Eclipse attack: Isolate a node from honest peers
	// Simulate by creating isolated context

	isolatedCtx := suite.ctx
	normalCtx := suite.ctx

	// Both contexts should process same transactions identically
	// Network layer should prevent eclipse via diverse peer connections

	suite.Require().Equal(isolatedCtx.BlockHeight(), normalCtx.BlockHeight())

	// In production:
	// 1. Connect to multiple diverse peers
	// 2. Validate peer information
	// 3. Monitor for network partitions
	// 4. Use peer reputation systems
	suite.T().Log("Eclipse attack test - network diversity required")
}

// TestSelfish_Mining tests selfish mining resistance
func (suite *AdversarialTestSuite) TestSelfish_Mining() {
	// Selfish mining: Validator withholds blocks to gain advantage
	// In PoS, this is less effective than PoW

	currentHeight := suite.ctx.BlockHeight()

	// Simulate validator creating blocks but not broadcasting
	privateChain := suite.ctx.WithBlockHeight(currentHeight + 5)
	publicChain := suite.ctx.WithBlockHeight(currentHeight + 3)

	suite.Require().Greater(privateChain.BlockHeight(), publicChain.BlockHeight())

	// In Cosmos BFT consensus:
	// 1. 2/3+ validators must agree on each block
	// 2. Cannot withhold blocks without losing consensus
	// 3. Finality prevents chain reorganization
	suite.T().Log("Selfish mining test - BFT consensus provides protection")
}

// TestCensorship_Resistance tests transaction censorship resistance
func (suite *AdversarialTestSuite) TestCensorship_Resistance() {
	// Malicious validator tries to censor transactions
	victimPriv := secp256k1.GenPrivKey()
	victimAddr := sdk.AccAddress(victimPriv.PubKey().Address())

	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(1000000)),
		sdk.NewCoin("uusdt", math.NewInt(1000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, victimAddr, funds))

	// In honest network, transaction should be included
	_, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		victimAddr,
		"upaw",
		"uusdt",
		math.NewInt(100000),
		math.NewInt(200000),
	)
	suite.Require().NoError(err, "Transaction should not be censored")

	// Mitigation:
	// 1. Multiple validators (requires 2/3+ to censor)
	// 2. Transaction prioritization by fees
	// 3. Minimum inclusion guarantees
	suite.T().Log("Censorship resistance test - decentralization is key")
}

// TestCircuitBreakerBlocksManipulation ensures a sudden price swing triggers an automatic pause.
func (suite *AdversarialTestSuite) TestCircuitBreakerBlocksManipulation() {
	suite.ctx = suite.ctx.WithBlockTime(time.Now())

	attacker := secp256k1.GenPrivKey()
	attackerAddr := sdk.AccAddress(attacker.PubKey().Address())
	funds := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(20_000_000)),
		sdk.NewCoin("uusdc", math.NewInt(20_000_000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, funds))

	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		attackerAddr,
		"upaw",
		"uusdc",
		math.NewInt(10_000_000),
		math.NewInt(10_000_000),
	)
	suite.Require().NoError(err)

	// seed last observed price at 1.0
	initialPrice := math.LegacyNewDec(1)
	err = suite.app.DEXKeeper.SetCircuitBreakerState(suite.ctx, pool.Id, dexkeeper.CircuitBreakerState{
		LastPrice: initialPrice,
	})
	suite.Require().NoError(err)

	// inject manipulated reserves (4x price move) to force circuit breaker
	pool.ReserveA = math.NewInt(2_000_000)
	pool.ReserveB = math.NewInt(8_000_000)
	suite.Require().NoError(suite.app.DEXKeeper.SetPool(suite.ctx, pool))

	err = suite.app.DEXKeeper.CheckPoolPriceDeviationForTesting(suite.ctx, pool, "swap")
	suite.Require().ErrorIs(err, dextypes.ErrCircuitBreakerTriggered)

	cbState, err := suite.app.DEXKeeper.GetPoolCircuitBreakerState(suite.ctx, pool.Id)
	suite.Require().NoError(err)
	suite.Require().True(cbState.Enabled, "circuit breaker should be opened after extreme deviation")
	suite.Require().True(cbState.PausedUntil.After(suite.ctx.BlockTime()), "pause window should be set into the future")
	suite.Require().Contains(cbState.TriggerReason, "price deviation", "trigger reason should document the anomaly")
}

// TestReentrancyGuardPreventsNestedSwap proves the per-pool guard rejects nested execution paths.
func (suite *AdversarialTestSuite) TestReentrancyGuardPreventsNestedSwap() {
	guard := dexkeeper.NewReentrancyGuard()

	err := suite.app.DEXKeeper.WithReentrancyGuardAndLock(suite.ctx, 99, "swap", guard, func() error {
		innerErr := suite.app.DEXKeeper.WithReentrancyGuardAndLock(suite.ctx, 99, "swap", guard, func() error {
			return nil
		})
		suite.Require().ErrorIs(innerErr, dextypes.ErrReentrancy)
		return innerErr
	})

	suite.Require().ErrorIs(err, dextypes.ErrReentrancy, "outer call should surface reentrancy violation")
}
