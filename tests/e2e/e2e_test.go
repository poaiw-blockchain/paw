//go:build integration
// +build integration

package e2e_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

	creatorAddr, err := sdk.AccAddressFromBech32(msgCreatePool.Creator)
	suite.Require().NoError(err)

	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		creatorAddr,
		msgCreatePool.TokenA,
		msgCreatePool.TokenB,
		msgCreatePool.AmountA,
		msgCreatePool.AmountB,
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(pool)
	poolId := pool.Id

	// Step 2: Add liquidity
	msgAddLiq := &dextypes.MsgAddLiquidity{
		Provider: suite.dexUser.String(),
		PoolId:   poolId,
		AmountA:  math.NewInt(1000000),
		AmountB:  math.NewInt(2000000),
	}

	liquidityProvider, err := sdk.AccAddressFromBech32(msgAddLiq.Provider)
	suite.Require().NoError(err)

	liquidityTokens, err := suite.app.DEXKeeper.AddLiquidity(
		suite.ctx,
		liquidityProvider,
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

	swapper, err := sdk.AccAddressFromBech32(msgSwap.Trader)
	suite.Require().NoError(err)

	amountOut, err := suite.app.DEXKeeper.ExecuteSwapSecure(
		suite.ctx,
		swapper,
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

	providerAcc := sdk.MustAccAddressFromBech32(msgRemoveLiq.Provider)
	amountA, amountB, err := suite.app.DEXKeeper.RemoveLiquidity(
		suite.ctx,
		providerAcc,
		msgRemoveLiq.PoolId,
		msgRemoveLiq.Shares,
	)
	suite.Require().NoError(err)
	suite.Require().True(amountA.GT(math.ZeroInt()))
	suite.Require().True(amountB.GT(math.ZeroInt()))
}

// TestValidatorGovernanceLifecycle ensures staking params can be updated via gov authority.
func (suite *E2ETestSuite) TestValidatorGovernanceLifecycle() {
	stakingServer := keeper.NewMsgServerImpl(suite.app.StakingKeeper)
	var err error
	params := stakingtypes.DefaultParams()
	params.UnbondingTime = time.Hour
	params.BondDenom = "upaw"
	govAuthority := suite.app.GovKeeper.GetGovernanceAccount(suite.ctx).GetAddress().String()
	_, err = stakingServer.UpdateParams(suite.ctx, &stakingtypes.MsgUpdateParams{
		Authority: govAuthority,
		Params:    params,
	})
	suite.Require().NoError(err)

	updatedParams, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(time.Hour, updatedParams.UnbondingTime)
}
