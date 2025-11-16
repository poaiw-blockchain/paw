package e2e_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
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
	suite.app = suite.setupApp()
	suite.ctx = suite.app.BaseApp.NewContext(false).WithChainID("paw-e2e-1")

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

func (suite *E2ETestSuite) setupApp() *app.PAWApp {
	db := dbm.NewMemDB()
	return app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-e2e-1"),
	)
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

	createResp, err := suite.app.DexKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)
	suite.Require().NotNil(createResp)
	poolId := createResp.PoolId

	// Step 2: Add liquidity
	msgAddLiq := &dextypes.MsgAddLiquidity{
		Provider: suite.dexUser.String(),
		PoolId:   poolId,
		AmountA:  math.NewInt(1000000),
		AmountB:  math.NewInt(2000000),
	}

	addLiqResp, err := suite.app.DexKeeper.AddLiquidity(suite.ctx, msgAddLiq)
	suite.Require().NoError(err)
	suite.Require().True(addLiqResp.LiquidityTokens.GT(math.ZeroInt()))

	// Step 3: Execute swap
	msgSwap := &dextypes.MsgSwap{
		Trader:       suite.trader.String(),
		PoolId:       poolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(1000000),
		MinAmountOut: math.NewInt(1800000),
	}

	swapResp, err := suite.app.DexKeeper.Swap(suite.ctx, msgSwap)
	suite.Require().NoError(err)
	suite.Require().True(swapResp.AmountOut.GT(math.ZeroInt()))

	// Step 4: Remove liquidity
	msgRemoveLiq := &dextypes.MsgRemoveLiquidity{
		Provider:        suite.dexUser.String(),
		PoolId:          poolId,
		LiquidityTokens: math.NewInt(500000),
		MinAmountA:      math.NewInt(400000),
		MinAmountB:      math.NewInt(800000),
	}

	removeLiqResp, err := suite.app.DexKeeper.RemoveLiquidity(suite.ctx, msgRemoveLiq)
	suite.Require().NoError(err)
	suite.Require().True(removeLiqResp.AmountA.GT(math.ZeroInt()))
	suite.Require().True(removeLiqResp.AmountB.GT(math.ZeroInt()))
}

// TestComputeWorkflow tests compute request and response flow
func (suite *E2ETestSuite) TestComputeWorkflow() {
	// Step 1: Register compute provider
	msgRegister := &computetypes.MsgRegisterProvider{
		Provider: suite.provider.String(),
		Endpoint: "https://api.compute-provider.io/v1",
		Stake:    math.NewInt(1000000),
	}

	_, err := suite.app.ComputeKeeper.RegisterProvider(suite.ctx, msgRegister)
	suite.Require().NoError(err)

	// Step 2: Submit compute request
	msgRequest := &computetypes.MsgRequestCompute{
		Requester: suite.dexUser.String(),
		ApiUrl:    "https://api.openai.com/v1/chat/completions",
		MaxFee:    math.NewInt(10000),
	}

	requestResp, err := suite.app.ComputeKeeper.RequestCompute(suite.ctx, msgRequest)
	suite.Require().NoError(err)
	suite.Require().Greater(requestResp.RequestId, uint64(0))

	// Step 3: Provider submits result
	msgResult := &computetypes.MsgSubmitResult{
		Provider:  suite.provider.String(),
		RequestId: requestResp.RequestId,
		Result:    `{"choices": [{"message": {"content": "Hello from PAW AI"}}]}`,
	}

	resultResp, err := suite.app.ComputeKeeper.SubmitResult(suite.ctx, msgResult)
	suite.Require().NoError(err)
	suite.Require().NotNil(resultResp)

	// Verify request completed
	request, found := suite.app.ComputeKeeper.GetRequest(suite.ctx, requestResp.RequestId)
	suite.Require().True(found)
	suite.Require().Equal(computetypes.RequestStatus_COMPLETED, request.Status)
}

// TestOracleWorkflow tests oracle price feed workflow
func (suite *E2ETestSuite) TestOracleWorkflow() {
	// Step 1: Register oracle
	msgRegister := &oracletypes.MsgRegisterOracle{
		Validator: suite.oracle.String(),
	}

	_, err := suite.app.OracleKeeper.RegisterOracle(suite.ctx, msgRegister)
	suite.Require().NoError(err)

	// Step 2: Submit price feeds
	assets := []struct {
		name  string
		price string
	}{
		{"BTC/USD", "45000.00"},
		{"ETH/USD", "2500.00"},
		{"PAW/USD", "0.50"},
	}

	for _, asset := range assets {
		msgPrice := &oracletypes.MsgSubmitPrice{
			Oracle: suite.oracle.String(),
			Asset:  asset.name,
			Price:  sdk.MustNewDecFromStr(asset.price),
		}

		_, err := suite.app.OracleKeeper.SubmitPrice(suite.ctx, msgPrice)
		suite.Require().NoError(err)
	}

	// Step 3: Verify prices are retrievable
	for _, asset := range assets {
		price, found := suite.app.OracleKeeper.GetPrice(suite.ctx, asset.name, suite.oracle.String())
		suite.Require().True(found)
		suite.Require().Equal(sdk.MustNewDecFromStr(asset.price), price.Price)
	}
}

// TestCrossModuleInteraction tests interaction between modules
func (suite *E2ETestSuite) TestCrossModuleInteraction() {
	// Setup: Create DEX pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: suite.dexUser.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(5000000),
		AmountB: math.NewInt(10000000),
	}

	poolResp, err := suite.app.DexKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Setup: Register oracle and submit PAW price
	msgRegisterOracle := &oracletypes.MsgRegisterOracle{
		Validator: suite.oracle.String(),
	}
	_, err = suite.app.OracleKeeper.RegisterOracle(suite.ctx, msgRegisterOracle)
	suite.Require().NoError(err)

	msgPrice := &oracletypes.MsgSubmitPrice{
		Oracle: suite.oracle.String(),
		Asset:  "PAW/USDT",
		Price:  sdk.MustNewDecFromStr("2.00"), // Should match pool ratio
	}
	_, err = suite.app.OracleKeeper.SubmitPrice(suite.ctx, msgPrice)
	suite.Require().NoError(err)

	// Verify pool price aligns with oracle price
	pool, found := suite.app.DexKeeper.GetPool(suite.ctx, poolResp.PoolId)
	suite.Require().True(found)

	// Pool ratio: ReserveB / ReserveA = 10M / 5M = 2.0
	poolRatio := sdk.NewDecFromInt(pool.ReserveB).Quo(sdk.NewDecFromInt(pool.ReserveA))
	oraclePrice := sdk.MustNewDecFromStr("2.00")

	suite.Require().True(poolRatio.Sub(oraclePrice).Abs().LT(sdk.MustNewDecFromStr("0.01")),
		"Pool ratio should align with oracle price")
}
