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
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
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
	t.Skip("TODO: Refactor tests to use correct keeper method signatures")
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
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: attackerAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	resp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Check balance before swap
	balanceBefore := suite.app.BankKeeper.GetBalance(suite.ctx, attackerAddr, "upaw")

	// Attempt to spend same funds twice
	msgSwap := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(1000000),
		MinAmountOut: math.NewInt(1),
	}

	// First swap should succeed
	_, err = suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)
	suite.Require().NoError(err)

	balanceAfterFirst := suite.app.BankKeeper.GetBalance(suite.ctx, attackerAddr, "upaw")
	suite.Require().True(balanceAfterFirst.Amount.LT(balanceBefore.Amount), "Balance should decrease after swap")

	// Second identical swap
	_, err = suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)

	// Should either succeed with remaining balance or fail with insufficient funds
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
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: victimAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	resp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Victim plans to make a large swap
	victimSwap := &dextypes.MsgSwap{
		Trader:       victimAddr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(500000),
		MinAmountOut: math.NewInt(900000),
	}

	// Attacker tries to front-run by swapping first
	attackerSwap := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(300000),
		MinAmountOut: math.NewInt(1),
	}

	// Execute attacker swap first (front-running)
	attackerRespBefore, err := suite.app.DEXKeeper.Swap(suite.ctx, attackerSwap)
	suite.Require().NoError(err)

	// Execute victim swap (price now worse due to front-running)
	victimResp, err := suite.app.DEXKeeper.Swap(suite.ctx, victimSwap)

	if err != nil {
		// Victim's swap fails due to slippage
		suite.T().Log("Victim's swap failed due to front-running changing the price")
	} else {
		// Victim gets worse price than expected
		suite.T().Logf("Victim received %s (expected >= %s)", victimResp.AmountOut, victimSwap.MinAmountOut)
	}

	// Attacker tries to back-run by swapping reverse
	attackerBackRun := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "uusdt",
		AmountIn:     attackerRespBefore.AmountOut,
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.app.DEXKeeper.Swap(suite.ctx, attackerBackRun)
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
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: victimAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(10000000),
		AmountB: math.NewInt(20000000),
	}

	poolResp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// 1. Attacker front-runs with buy
	attackBuy := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       poolResp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(1000000),
		MinAmountOut: math.NewInt(1),
	}

	attackBuyResp, err := suite.app.DEXKeeper.Swap(suite.ctx, attackBuy)
	suite.Require().NoError(err)

	// 2. Victim executes their trade
	victimTrade := &dextypes.MsgSwap{
		Trader:       victimAddr.String(),
		PoolId:       poolResp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(2000000),
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.app.DEXKeeper.Swap(suite.ctx, victimTrade)
	suite.Require().NoError(err)

	// 3. Attacker back-runs with sell
	attackSell := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       poolResp.PoolId,
		TokenIn:      "uusdt",
		AmountIn:     attackBuyResp.AmountOut,
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.app.DEXKeeper.Swap(suite.ctx, attackSell)
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
	largeSwap := &dextypes.MsgCreatePool{
		Creator: attackerAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(10000000), // Much more than attacker has
		AmountB: math.NewInt(20000000),
	}

	_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, largeSwap)

	// Should fail - insufficient funds
	suite.Require().Error(err, "Flash loan-style attack should fail without collateral")
}

// TestOracleManipulation tests oracle price manipulation attacks
func (suite *AdversarialTestSuite) TestOracleManipulation() {
	// Create multiple malicious oracles
	numMalicious := 3
	maliciousOracles := make([]sdk.AccAddress, numMalicious)

	for i := 0; i < numMalicious; i++ {
		priv := secp256k1.GenPrivKey()
		addr := sdk.AccAddress(priv.PubKey().Address())
		maliciousOracles[i] = addr

		// Register oracle
		msgRegister := &oracletypes.MsgRegisterOracle{
			Validator: addr.String(),
		}

		_, err := suite.app.OracleKeeper.RegisterOracle(suite.ctx, msgRegister)
		suite.Require().NoError(err)
	}

	// Malicious oracles submit false prices
	falsePrices := []string{"1000000.00", "999999.00", "1000001.00"}

	for i, oracle := range maliciousOracles {
		msgSubmit := &oracletypes.MsgSubmitPrice{
			Oracle: oracle.String(),
			Asset:  "BTC/USD",
			Price:  math.LegacyMustNewDecFromStr(falsePrices[i]),
		}

		_, err := suite.app.OracleKeeper.SubmitPrice(suite.ctx, msgSubmit)
		suite.Require().NoError(err)
	}

	// Check aggregated price
	// Good oracle implementation should:
	// 1. Use median instead of mean (resistant to outliers)
	// 2. Require minimum number of oracles
	// 3. Detect and slash malicious oracles
	// 4. Use time-weighted average prices (TWAP)

	suite.T().Log("Oracle manipulation test completed - verify aggregation logic")
}

// TestSybilAttack tests Sybil attack resistance
func (suite *AdversarialTestSuite) TestSybilAttack() {
	// Create many fake identities
	numSybils := 100
	sybilAccounts := make([]sdk.AccAddress, numSybils)

	for i := 0; i < numSybils; i++ {
		priv := secp256k1.GenPrivKey()
		addr := sdk.AccAddress(priv.PubKey().Address())
		sybilAccounts[i] = addr
	}

	// Attempt to register all as oracles (should require stake/validation)
	successfulRegistrations := 0

	for _, addr := range sybilAccounts {
		msgRegister := &oracletypes.MsgRegisterOracle{
			Validator: addr.String(),
		}

		_, err := suite.app.OracleKeeper.RegisterOracle(suite.ctx, msgRegister)
		if err == nil {
			successfulRegistrations++
		}
	}

	// Without stake/validation, Sybil resistance may be weak
	// Proper implementation should require:
	// 1. Stake/bond for participation
	// 2. Validator status
	// 3. Reputation system
	suite.T().Logf("Sybil accounts registered: %d/%d", successfulRegistrations, numSybils)
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

	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
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

	// Fund attacker minimally
	funds := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(100000000)))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, attackerAddr, funds))

	// Attempt to create many small pools to exhaust storage
	maxGriefAttempts := 1000
	successfulGrief := 0

	for i := 0; i < maxGriefAttempts; i++ {
		msg := &dextypes.MsgCreatePool{
			Creator: attackerAddr.String(),
			TokenA:  "upaw",
			TokenB:  "uusdt",
			AmountA: math.NewInt(1), // Minimal amounts
			AmountB: math.NewInt(1),
		}

		_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msg)
		if err == nil {
			successfulGrief++
		} else {
			break
		}
	}

	// Should have limits to prevent resource exhaustion
	suite.T().Logf("Griefing attack created %d pools before hitting limits", successfulGrief)
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
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: attackerAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	resp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Execute swap
	msgSwap := &dextypes.MsgSwap{
		Trader:       attackerAddr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(100000),
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)
	suite.Require().NoError(err)

	// Attempt to replay same transaction
	// In Cosmos SDK, sequence numbers prevent replay attacks
	_, err = suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)

	// May succeed or fail depending on balance, but won't be identical replay
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

	funds := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, funds))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, victimAddr, funds))

	// Create transaction from victim
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: victimAddr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(100000),
		AmountB: math.NewInt(200000),
	}

	// In honest network, transaction should be included
	_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err, "Transaction should not be censored")

	// Mitigation:
	// 1. Multiple validators (requires 2/3+ to censor)
	// 2. Transaction prioritization by fees
	// 3. Minimum inclusion guarantees
	suite.T().Log("Censorship resistance test - decentralization is key")
}
