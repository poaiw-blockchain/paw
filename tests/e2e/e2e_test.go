package e2e_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// E2ETestSuite is a comprehensive end-to-end test suite
type E2ETestSuite struct {
	suite.Suite

	app *app.PAWApp
	ctx sdk.Context

	// Test accounts
	dexUser  sdk.AccAddress
	trader   sdk.AccAddress
	provider sdk.AccAddress
	oracle   sdk.AccAddress
}

func (suite *E2ETestSuite) SetupSuite() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())

	// Create test accounts
	suite.dexUser = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	suite.trader = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	suite.provider = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	suite.oracle = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Fund accounts
	suite.fundAccount(suite.dexUser, sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(100000000)),
		sdk.NewCoin("uusdt", math.NewInt(100000000)),
	))
	suite.fundAccount(suite.trader, sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(50000000)),
	))
	suite.fundAccount(suite.provider, sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
	))
}

func (suite *E2ETestSuite) fundAccount(addr sdk.AccAddress, coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins)
	suite.Require().NoError(err)
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// TestDEXFullWorkflow tests complete DEX lifecycle
func (suite *E2ETestSuite) TestDEXFullWorkflow() {
	// Step 1: Create liquidity pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: suite.dexUser.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(10000000), // 10M PAW
		AmountB: math.NewInt(20000000), // 20M USDT
	}

	poolId, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		msgCreatePool.Creator,
		msgCreatePool.TokenA,
		msgCreatePool.TokenB,
		msgCreatePool.AmountA,
		msgCreatePool.AmountB,
	)
	suite.Require().NoError(err)
	suite.Require().Greater(poolId, uint64(0))

	// Step 2: Add liquidity
	msgAddLiq := &dextypes.MsgAddLiquidity{
		Provider: suite.dexUser.String(),
		PoolId:   poolId,
		AmountA:  math.NewInt(1000000),
		AmountB:  math.NewInt(2000000),
	}

	liquidityTokens, err := suite.app.DEXKeeper.AddLiquidity(
		suite.ctx,
		msgAddLiq.Provider,
		msgAddLiq.PoolId,
		msgAddLiq.AmountA,
		msgAddLiq.AmountB,
	)
	suite.Require().NoError(err)
	suite.Require().True(liquidityTokens.GT(math.ZeroInt()))

	// Step 3: Execute swap (use smaller amount to keep price impact under 5%)
	msgSwap := &dextypes.MsgSwap{
		Trader:       suite.trader.String(),
		PoolId:       poolId,
		TokenIn:      "upaw",
		TokenOut:     "uusdt",
		AmountIn:     math.NewInt(500000), // Reduced to 500k to keep price impact low
		MinAmountOut: math.NewInt(900000), // Adjusted accordingly
	}

	amountOut, err := suite.app.DEXKeeper.Swap(
		suite.ctx,
		msgSwap.Trader,
		msgSwap.PoolId,
		msgSwap.TokenIn,
		msgSwap.TokenOut,
		msgSwap.AmountIn,
		msgSwap.MinAmountOut,
	)
	suite.Require().NoError(err)
	suite.Require().True(amountOut.GT(math.ZeroInt()))

	// Step 4: Remove liquidity
	msgRemoveLiq := &dextypes.MsgRemoveLiquidity{
		Provider: suite.dexUser.String(),
		PoolId:   poolId,
		Shares:   math.NewInt(500000),
	}

	amountA, amountB, err := suite.app.DEXKeeper.RemoveLiquidity(
		suite.ctx,
		msgRemoveLiq.Provider,
		msgRemoveLiq.PoolId,
		msgRemoveLiq.Shares,
	)
	suite.Require().NoError(err)
	suite.Require().True(amountA.GT(math.ZeroInt()))
	suite.Require().True(amountB.GT(math.ZeroInt()))
}

