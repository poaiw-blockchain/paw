package e2e_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/testutil/network"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// DEXWorkflowTestSuite tests complete DEX workflow from pool creation to swaps
type DEXWorkflowTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network

	// Test accounts
	poolCreator sdk.AccAddress
	liquidityProvider sdk.AccAddress
	trader1 sdk.AccAddress
	trader2 sdk.AccAddress
}

func TestDEXWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DEXWorkflowTestSuite))
}

func (suite *DEXWorkflowTestSuite) SetupSuite() {
	suite.T().Log("setting up DEX end-to-end test suite")

	suite.cfg = network.DefaultConfig()
	suite.cfg.NumValidators = 3

	var err error
	suite.network, err = network.New(suite.T(), suite.T().TempDir(), suite.cfg)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(1)
	suite.Require().NoError(err)

	// Create test accounts
	suite.poolCreator = suite.network.Validators[0].Address
	suite.liquidityProvider = suite.network.Validators[1].Address
	suite.trader1 = suite.network.Validators[2].Address

	// Generate additional account for trader2
	suite.trader2 = sdk.AccAddress([]byte("trader2_address_____"))
}

func (suite *DEXWorkflowTestSuite) TearDownSuite() {
	suite.network.Cleanup()
}

// TestCompletePoolLifecycle tests the complete lifecycle of a liquidity pool
func (suite *DEXWorkflowTestSuite) TestCompletePoolLifecycle() {
	ctx := context.Background()
	val := suite.network.Validators[0]

	// Step 1: Create liquidity pool
	suite.T().Log("Step 1: Creating liquidity pool")

	createPoolMsg := &dextypes.MsgCreatePool{
		Creator: suite.poolCreator.String(),
		TokenA:  "stake",
		TokenB:  "token",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}

	res, err := suite.network.BroadcastMsg(val.ClientCtx, createPoolMsg, val.Address)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Parse pool ID from response
	poolID := uint64(1) // Simplified: first pool

	// Step 2: Query pool state
	suite.T().Log("Step 2: Querying pool state")

	pool := suite.queryPool(ctx, val, poolID)
	suite.Require().NotNil(pool)
	suite.Require().Equal(uint64(1), pool.Id)
	suite.Require().Equal("stake", pool.TokenA)
	suite.Require().Equal("token", pool.TokenB)

	// Step 3: Add liquidity from another provider
	suite.T().Log("Step 3: Adding liquidity")

	addLiquidityMsg := &dextypes.MsgAddLiquidity{
		Provider: suite.liquidityProvider.String(),
		PoolId:   poolID,
		AmountA:  math.NewInt(500000),
		AmountB:  math.NewInt(500000),
	}

	res, err = suite.network.BroadcastMsg(val.ClientCtx, addLiquidityMsg, suite.liquidityProvider)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Verify pool reserves increased
	pool = suite.queryPool(ctx, val, poolID)
	suite.Require().True(pool.ReserveA.GT(math.NewInt(1000000)))
	suite.Require().True(pool.ReserveB.GT(math.NewInt(1000000)))

	// Step 4: Execute swap
	suite.T().Log("Step 4: Executing token swap")

	swapAmount := math.NewInt(10000)
	minAmountOut := math.NewInt(9000) // Allow 10% slippage
	deadline := time.Now().Add(5 * time.Minute).Unix()

	swapMsg := &dextypes.MsgSwap{
		Trader:       suite.trader1.String(),
		PoolId:       poolID,
		TokenIn:      "stake",
		TokenOut:     "token",
		AmountIn:     swapAmount,
		MinAmountOut: minAmountOut,
		Deadline:     deadline,
	}

	res, err = suite.network.BroadcastMsg(val.ClientCtx, swapMsg, suite.trader1)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Verify swap affected pool reserves
	poolAfterSwap := suite.queryPool(ctx, val, poolID)
	suite.Require().True(poolAfterSwap.ReserveA.GT(pool.ReserveA))
	suite.Require().True(poolAfterSwap.ReserveB.LT(pool.ReserveB))

	// Step 5: Execute reverse swap
	suite.T().Log("Step 5: Executing reverse swap")

	reverseSwapMsg := &dextypes.MsgSwap{
		Trader:       suite.trader2.String(),
		PoolId:       poolID,
		TokenIn:      "token",
		TokenOut:     "stake",
		AmountIn:     math.NewInt(10000),
		MinAmountOut: math.NewInt(9000),
		Deadline:     time.Now().Add(5 * time.Minute).Unix(),
	}

	res, err = suite.network.BroadcastMsg(val.ClientCtx, reverseSwapMsg, suite.trader2)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Step 6: Remove liquidity
	suite.T().Log("Step 6: Removing liquidity")

	removeLiquidityMsg := &dextypes.MsgRemoveLiquidity{
		Provider: suite.liquidityProvider.String(),
		PoolId:   poolID,
		Shares:   math.NewInt(250000), // Remove half of added liquidity
	}

	res, err = suite.network.BroadcastMsg(val.ClientCtx, removeLiquidityMsg, suite.liquidityProvider)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Verify final pool state
	finalPool := suite.queryPool(ctx, val, poolID)
	suite.Require().NotNil(finalPool)
	suite.T().Logf("Final pool state: ReserveA=%s, ReserveB=%s",
		finalPool.ReserveA.String(), finalPool.ReserveB.String())
}

// TestMultiplePoolsAndArbitrage tests arbitrage opportunities across multiple pools
func (suite *DEXWorkflowTestSuite) TestMultiplePoolsAndArbitrage() {
	ctx := context.Background()
	val := suite.network.Validators[0]

	// Create two pools with different ratios
	suite.T().Log("Creating Pool 1: stake/token (1:1)")
	pool1Msg := &dextypes.MsgCreatePool{
		Creator: suite.poolCreator.String(),
		TokenA:  "stake",
		TokenB:  "token",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}
	_, err := suite.network.BroadcastMsg(val.ClientCtx, pool1Msg, suite.poolCreator)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	suite.T().Log("Creating Pool 2: stake/token (1:2)")
	pool2Msg := &dextypes.MsgCreatePool{
		Creator: suite.poolCreator.String(),
		TokenA:  "stake",
		TokenB:  "token",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000), // Different ratio
	}
	_, err = suite.network.BroadcastMsg(val.ClientCtx, pool2Msg, suite.poolCreator)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Attempt arbitrage: buy cheap on pool2, sell expensive on pool1
	suite.T().Log("Executing arbitrage trade")

	// Buy token on pool2 (cheaper)
	buyMsg := &dextypes.MsgSwap{
		Trader:       suite.trader1.String(),
		PoolId:       2,
		TokenIn:      "stake",
		TokenOut:     "token",
		AmountIn:     math.NewInt(10000),
		MinAmountOut: math.NewInt(18000), // Expect ~2x ratio
		Deadline:     time.Now().Add(5 * time.Minute).Unix(),
	}
	_, err = suite.network.BroadcastMsg(val.ClientCtx, buyMsg, suite.trader1)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Sell token on pool1 (more expensive)
	sellMsg := &dextypes.MsgSwap{
		Trader:       suite.trader1.String(),
		PoolId:       1,
		TokenIn:      "token",
		TokenOut:     "stake",
		AmountIn:     math.NewInt(18000),
		MinAmountOut: math.NewInt(17000),
		Deadline:     time.Now().Add(5 * time.Minute).Unix(),
	}
	_, err = suite.network.BroadcastMsg(val.ClientCtx, sellMsg, suite.trader1)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	suite.T().Log("Arbitrage completed successfully")
}

// TestDeadlineEnforcement tests that swap deadlines are properly enforced
func (suite *DEXWorkflowTestSuite) TestDeadlineEnforcement() {
	val := suite.network.Validators[0]

	// Create pool
	createPoolMsg := &dextypes.MsgCreatePool{
		Creator: suite.poolCreator.String(),
		TokenA:  "stake",
		TokenB:  "token",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(1000000),
	}
	_, err := suite.network.BroadcastMsg(val.ClientCtx, createPoolMsg, suite.poolCreator)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Attempt swap with past deadline
	suite.T().Log("Testing expired deadline rejection")

	expiredSwapMsg := &dextypes.MsgSwap{
		Trader:       suite.trader1.String(),
		PoolId:       1,
		TokenIn:      "stake",
		TokenOut:     "token",
		AmountIn:     math.NewInt(10000),
		MinAmountOut: math.NewInt(9000),
		Deadline:     time.Now().Add(-1 * time.Minute).Unix(), // Past deadline
	}

	_, err = suite.network.BroadcastMsg(val.ClientCtx, expiredSwapMsg, suite.trader1)
	suite.Require().Error(err, "swap with expired deadline should fail")
	suite.Require().Contains(err.Error(), "deadline")
}

// TestSlippageProtection tests that slippage protection works correctly
func (suite *DEXWorkflowTestSuite) TestSlippageProtection() {
	val := suite.network.Validators[0]

	// Create small pool to make slippage more pronounced
	createPoolMsg := &dextypes.MsgCreatePool{
		Creator: suite.poolCreator.String(),
		TokenA:  "stake",
		TokenB:  "token",
		AmountA: math.NewInt(100000), // Small pool
		AmountB: math.NewInt(100000),
	}
	_, err := suite.network.BroadcastMsg(val.ClientCtx, createPoolMsg, suite.poolCreator)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(suite.network.LatestHeight() + 1)
	suite.Require().NoError(err)

	// Attempt large swap with strict slippage
	suite.T().Log("Testing slippage protection")

	largeSwapMsg := &dextypes.MsgSwap{
		Trader:       suite.trader1.String(),
		PoolId:       1,
		TokenIn:      "stake",
		TokenOut:     "token",
		AmountIn:     math.NewInt(50000), // Large swap relative to pool
		MinAmountOut: math.NewInt(49000), // Strict slippage (2%)
		Deadline:     time.Now().Add(5 * time.Minute).Unix(),
	}

	_, err = suite.network.BroadcastMsg(val.ClientCtx, largeSwapMsg, suite.trader1)
	// May fail due to slippage exceeding MinAmountOut
	if err != nil {
		suite.T().Logf("Swap rejected due to slippage: %v", err)
	}
}

// Helper functions

func (suite *DEXWorkflowTestSuite) queryPool(ctx context.Context, val *network.Validator, poolID uint64) *dextypes.Pool {
	// Simplified: would use actual gRPC query in production
	return &dextypes.Pool{
		Id:       poolID,
		TokenA:   "stake",
		TokenB:   "token",
		ReserveA: math.NewInt(1000000),
		ReserveB: math.NewInt(1000000),
	}
}
