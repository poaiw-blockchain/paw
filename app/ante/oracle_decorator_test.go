package ante_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app/ante"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

type OracleDecoratorTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	oracleKeeper  *oraclekeeper.Keeper
	decorator     ante.OracleDecorator
	encCfg        moduletestutil.TestEncodingConfig
	validatorAddr sdk.ValAddress
	feederAddr    sdk.AccAddress
}

func TestOracleDecoratorTestSuite(t *testing.T) {
	suite.Run(t, new(OracleDecoratorTestSuite))
}

func (suite *OracleDecoratorTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	suite.encCfg = moduletestutil.MakeTestEncodingConfig()
	oracletypes.RegisterInterfaces(suite.encCfg.InterfaceRegistry)

	// Create mock keepers (minimal setup)
	suite.oracleKeeper = oraclekeeper.NewKeeper(
		suite.encCfg.Codec,
		key,
		nil, // bankKeeper - not needed for decorator tests
		nil, // stakingKeeper - will be mocked
		slashingkeeper.Keeper{},
		nil, // ibcKeeper - not needed for decorator tests
		nil, // portKeeper - not needed for decorator tests
		authtypes.NewModuleAddress("gov").String(),
		capabilitykeeper.ScopedKeeper{},
	)

	suite.decorator = ante.NewOracleDecorator(*suite.oracleKeeper)

	// Setup test addresses
	suite.validatorAddr = sdk.ValAddress([]byte("validator1"))
	suite.feederAddr = sdk.AccAddress([]byte("feeder1"))
}

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_ValidPrice() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
	}

	// Create mock transaction
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

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_InvalidValidatorAddress() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: "invalid_address",
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid validator address")
}

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_InvalidFeederAddress() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    "invalid_feeder",
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid feeder address")
}

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_EmptyAsset() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    suite.feederAddr.String(),
		Asset:     "",
		Price:     math.LegacyNewDec(50000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "asset cannot be empty")
}

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_ZeroPrice() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyZeroDec(),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "price must be positive")
}

func (suite *OracleDecoratorTestSuite) TestValidateSubmitPrice_NegativePrice() {
	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyNewDec(-100),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "price must be positive")
}

func (suite *OracleDecoratorTestSuite) TestValidateDelegateFeedConsent_InvalidValidatorAddress() {
	msg := &oracletypes.MsgDelegateFeedConsent{
		Validator: "invalid_address",
		Delegate:  suite.feederAddr.String(),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid validator address")
}

func (suite *OracleDecoratorTestSuite) TestValidateDelegateFeedConsent_InvalidDelegateAddress() {
	msg := &oracletypes.MsgDelegateFeedConsent{
		Validator: suite.validatorAddr.String(),
		Delegate:  "invalid_delegate",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid delegate address")
}

func (suite *OracleDecoratorTestSuite) TestAnteHandle_SimulateMode() {
	// Even with invalid data, simulate mode should pass
	msg := &oracletypes.MsgSubmitPrice{
		Validator: "invalid",
		Feeder:    "invalid",
		Asset:     "",
		Price:     math.LegacyZeroDec(),
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

func (suite *OracleDecoratorTestSuite) TestAnteHandle_NonOracleMessage() {
	// Non-oracle messages should pass through without validation
	msg := &banktypes.MsgSend{
		FromAddress: suite.feederAddr.String(),
		ToAddress:   suite.validatorAddr.String(),
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
	suite.Require().True(called, "next handler should be called for non-oracle messages")
}

func (suite *OracleDecoratorTestSuite) TestAnteHandle_MultipleMessages() {
	// Test transaction with multiple messages
	msg1 := &banktypes.MsgSend{
		FromAddress: suite.feederAddr.String(),
		ToAddress:   suite.validatorAddr.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
	}

	msg2 := &oracletypes.MsgSubmitPrice{
		Validator: "invalid",
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg1, msg2)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err, "should fail on invalid oracle message even with valid non-oracle message")
	suite.Require().Contains(err.Error(), "invalid validator address")
}

// Benchmark tests
func BenchmarkOracleDecorator_ValidateSubmitPrice(b *testing.B) {
	suite := new(OracleDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	msg := &oracletypes.MsgSubmitPrice{
		Validator: suite.validatorAddr.String(),
		Feeder:    suite.feederAddr.String(),
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
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
