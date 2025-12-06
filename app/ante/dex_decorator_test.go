package ante_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app/ante"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

type DEXDecoratorTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	dexKeeper *dexkeeper.Keeper
	decorator ante.DEXDecorator
	encCfg    moduletestutil.TestEncodingConfig
	addr      sdk.AccAddress
}

func TestDEXDecoratorTestSuite(t *testing.T) {
	suite.Run(t, new(DEXDecoratorTestSuite))
}

func (suite *DEXDecoratorTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(dextypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	suite.encCfg = moduletestutil.MakeTestEncodingConfig()
	dextypes.RegisterInterfaces(suite.encCfg.InterfaceRegistry)

	// Create DEX keeper
	suite.dexKeeper = dexkeeper.NewKeeper(
		suite.encCfg.Codec,
		key,
		nil, // bankKeeper
		nil, // ibcKeeper
		nil, // portKeeper
		capabilitykeeper.ScopedKeeper{},
	)

	// Initialize default params
	params := dextypes.DefaultParams()
	err := suite.dexKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.decorator = ante.NewDEXDecorator(*suite.dexKeeper)
	suite.addr = sdk.AccAddress([]byte("creator1"))
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_ValidPool() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)

	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: params.MinLiquidity.Add(math.NewInt(1000)),
		AmountB: params.MinLiquidity.Add(math.NewInt(2000)),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	// Test in simulate mode - should skip validation
	_, err = suite.decorator.AnteHandle(suite.ctx, tx, true, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().NoError(err, "simulate mode should skip validation")
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_InvalidCreatorAddress() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)

	msg := &dextypes.MsgCreatePool{
		Creator: "invalid_address",
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: params.MinLiquidity,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid creator address")
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_EmptyTokenIdentifiers() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)

	// Empty TokenA
	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "",
		TokenB:  "uosmo",
		AmountA: params.MinLiquidity,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "token identifiers cannot be empty")

	// Empty TokenB
	msg.TokenA = "uatom"
	msg.TokenB = ""
	txBuilder = suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx = txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "token identifiers cannot be empty")
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_SameTokens() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)

	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uatom",
		AmountA: params.MinLiquidity,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "pool tokens must differ")
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_ZeroLiquidity() {
	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: math.ZeroInt(),
		AmountB: math.NewInt(1000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "initial liquidity amounts must be positive")
}

func (suite *DEXDecoratorTestSuite) TestValidateCreatePool_BelowMinLiquidity() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)
	belowMin := params.MinLiquidity.Sub(math.NewInt(1))

	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: belowMin,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "initial liquidity must be at least")
}

func (suite *DEXDecoratorTestSuite) TestValidateSwap_InvalidTraderAddress() {
	msg := &dextypes.MsgSwap{
		Trader:       "invalid_address",
		PoolId:       1,
		TokenIn:      "uatom",
		TokenOut:     "uosmo",
		AmountIn:     math.NewInt(1000),
		MinAmountOut: math.NewInt(900),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid trader address")
}

func (suite *DEXDecoratorTestSuite) TestValidateSwap_EmptyTokens() {
	msg := &dextypes.MsgSwap{
		Trader:       suite.addr.String(),
		PoolId:       1,
		TokenIn:      "",
		TokenOut:     "uosmo",
		AmountIn:     math.NewInt(1000),
		MinAmountOut: math.NewInt(900),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "tokens cannot be empty")
}

func (suite *DEXDecoratorTestSuite) TestValidateSwap_ZeroAmount() {
	msg := &dextypes.MsgSwap{
		Trader:       suite.addr.String(),
		PoolId:       1,
		TokenIn:      "uatom",
		TokenOut:     "uosmo",
		AmountIn:     math.ZeroInt(),
		MinAmountOut: math.NewInt(900),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap amount must be positive")
}

func (suite *DEXDecoratorTestSuite) TestValidateSwap_NegativeMinAmountOut() {
	msg := &dextypes.MsgSwap{
		Trader:       suite.addr.String(),
		PoolId:       1,
		TokenIn:      "uatom",
		TokenOut:     "uosmo",
		AmountIn:     math.NewInt(1000),
		MinAmountOut: math.NewInt(-100),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "minimum amount out cannot be negative")
}

func (suite *DEXDecoratorTestSuite) TestValidateSwap_PoolNotFound() {
	msg := &dextypes.MsgSwap{
		Trader:       suite.addr.String(),
		PoolId:       999,
		TokenIn:      "uatom",
		TokenOut:     "uosmo",
		AmountIn:     math.NewInt(1000),
		MinAmountOut: math.NewInt(900),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "pool 999 not found")
}

func (suite *DEXDecoratorTestSuite) TestValidateAddLiquidity_InvalidProviderAddress() {
	msg := &dextypes.MsgAddLiquidity{
		Provider: "invalid_address",
		PoolId:   1,
		AmountA:  math.NewInt(1000),
		AmountB:  math.NewInt(2000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid provider address")
}

func (suite *DEXDecoratorTestSuite) TestValidateAddLiquidity_ZeroAmounts() {
	msg := &dextypes.MsgAddLiquidity{
		Provider: suite.addr.String(),
		PoolId:   1,
		AmountA:  math.ZeroInt(),
		AmountB:  math.NewInt(2000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "liquidity amounts must be positive")
}

func (suite *DEXDecoratorTestSuite) TestValidateRemoveLiquidity_InvalidProviderAddress() {
	msg := &dextypes.MsgRemoveLiquidity{
		Provider: "invalid_address",
		PoolId:   1,
		Shares:   math.NewInt(1000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid provider address")
}

func (suite *DEXDecoratorTestSuite) TestValidateRemoveLiquidity_ZeroShares() {
	msg := &dextypes.MsgRemoveLiquidity{
		Provider: suite.addr.String(),
		PoolId:   1,
		Shares:   math.ZeroInt(),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "shares to remove must be positive")
}

func (suite *DEXDecoratorTestSuite) TestAnteHandle_SimulateMode() {
	// Even with invalid data, simulate mode should pass
	msg := &dextypes.MsgSwap{
		Trader:       "invalid",
		PoolId:       999,
		TokenIn:      "",
		TokenOut:     "",
		AmountIn:     math.ZeroInt(),
		MinAmountOut: math.NewInt(-100),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, true, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().NoError(err, "simulate mode should skip all validation")
}

func (suite *DEXDecoratorTestSuite) TestAnteHandle_NonDEXMessage() {
	// Non-DEX messages should pass through without validation
	msg := &banktypes.MsgSend{
		FromAddress: suite.addr.String(),
		ToAddress:   sdk.AccAddress([]byte("addr2")).String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	called := false
	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		called = true
		return ctx, nil
	})
	suite.Require().NoError(err)
	suite.Require().True(called, "next handler should be called for non-DEX messages")
}

func (suite *DEXDecoratorTestSuite) TestAnteHandle_GasConsumption() {
	params, _ := suite.dexKeeper.GetParams(suite.ctx)

	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: params.MinLiquidity,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	gasBeforeDecorator := suite.ctx.GasMeter().GasConsumed()

	_, _ = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})

	gasAfterDecorator := suite.ctx.GasMeter().GasConsumed()
	suite.Require().Greater(gasAfterDecorator, gasBeforeDecorator, "decorator should consume gas")
}

// Benchmark tests
func BenchmarkDEXDecorator_ValidateCreatePool(b *testing.B) {
	suite := new(DEXDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	params, _ := suite.dexKeeper.GetParams(suite.ctx)
	msg := &dextypes.MsgCreatePool{
		Creator: suite.addr.String(),
		TokenA:  "uatom",
		TokenB:  "uosmo",
		AmountA: params.MinLiquidity,
		AmountB: params.MinLiquidity,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(b, err)
	tx := txBuilder.GetTx()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = suite.decorator.AnteHandle(suite.ctx, tx, true, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			return ctx, nil
		})
	}
}

func BenchmarkDEXDecorator_ValidateSwap(b *testing.B) {
	suite := new(DEXDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	msg := &dextypes.MsgSwap{
		Trader:       suite.addr.String(),
		PoolId:       1,
		TokenIn:      "uatom",
		TokenOut:     "uosmo",
		AmountIn:     math.NewInt(1000),
		MinAmountOut: math.NewInt(900),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(b, err)
	tx := txBuilder.GetTx()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = suite.decorator.AnteHandle(suite.ctx, tx, true, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			return ctx, nil
		})
	}
}
