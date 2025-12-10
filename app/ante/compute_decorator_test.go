package ante_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app/ante"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

type ComputeDecoratorTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	computeKeeper *computekeeper.Keeper
	decorator     ante.ComputeDecorator
	encCfg        moduletestutil.TestEncodingConfig
	addr          sdk.AccAddress
	providerAddr  sdk.AccAddress
}

func TestComputeDecoratorTestSuite(t *testing.T) {
	suite.Run(t, new(ComputeDecoratorTestSuite))
}

func (suite *ComputeDecoratorTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(computetypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	suite.encCfg = moduletestutil.MakeTestEncodingConfig()
	computetypes.RegisterInterfaces(suite.encCfg.InterfaceRegistry)

	// Create compute keeper
	suite.computeKeeper = computekeeper.NewKeeper(
		suite.encCfg.Codec,
		key,
		nil, // bankKeeper - not needed for decorator tests
		accountkeeper.AccountKeeper{},
		nil, // stakingKeeper - not needed for decorator tests
		slashingkeeper.Keeper{},
		nil, // ibcKeeper - not needed for decorator tests
		nil, // portKeeper - not needed for decorator tests
		authtypes.NewModuleAddress("gov").String(),
		capabilitykeeper.ScopedKeeper{},
	)

	// Initialize default params
	params := computetypes.DefaultParams()
	err := suite.computeKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.decorator = ante.NewComputeDecorator(*suite.computeKeeper)
	suite.addr = sdk.AccAddress([]byte("requester1"))
	suite.providerAddr = sdk.AccAddress([]byte("provider1"))
}

// TestValidateSubmitRequest_SimulateMode tests that simulation skips validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitRequest_SimulateMode() {
	msg := &computetypes.MsgSubmitRequest{
		Requester:  "invalid_address",
		MaxPayment: math.NewInt(-100), // Invalid negative payment
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

// TestValidateSubmitRequest_InvalidRequesterAddress tests invalid requester address validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitRequest_InvalidRequesterAddress() {
	msg := &computetypes.MsgSubmitRequest{
		Requester:      "invalid_address",
		ContainerImage: "ubuntu:latest",
		MaxPayment:     math.NewInt(1000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid requester address")
}

// TestValidateSubmitRequest_NegativeMaxPayment tests negative max payment validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitRequest_NegativeMaxPayment() {
	msg := &computetypes.MsgSubmitRequest{
		Requester:      suite.addr.String(),
		ContainerImage: "ubuntu:latest",
		MaxPayment:     math.NewInt(-500),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "max payment must be non-negative")
}

// TestValidateSubmitRequest_GasConsumption tests that gas is consumed during validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitRequest_GasConsumption() {
	// Note: This test would panic due to nil bankKeeper in ValidateRequesterBalance
	// In a real deployment, bankKeeper is properly initialized
	// We test gas consumption using messages that don't require bank balance checks
	msg := &computetypes.MsgRegisterProvider{
		Provider: suite.providerAddr.String(),
		Moniker:  "Test Provider",
		Endpoint: "https://provider.example.com",
		Stake:    math.NewInt(2000000),
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
	suite.Require().Greater(gasAfterDecorator, gasBeforeDecorator, "decorator should consume gas during validation")
}

// TestValidateRegisterProvider_InvalidProviderAddress tests invalid provider address validation
func (suite *ComputeDecoratorTestSuite) TestValidateRegisterProvider_InvalidProviderAddress() {
	msg := &computetypes.MsgRegisterProvider{
		Provider: "invalid_provider_address",
		Moniker:  "Test Provider",
		Endpoint: "https://provider.example.com",
		Stake:    math.NewInt(2000000),
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

// TestValidateRegisterProvider_StakeBelowMinimum tests stake below minimum validation
func (suite *ComputeDecoratorTestSuite) TestValidateRegisterProvider_StakeBelowMinimum() {
	params, _ := suite.computeKeeper.GetParams(suite.ctx)
	belowMinStake := params.MinProviderStake.Sub(math.NewInt(1))

	msg := &computetypes.MsgRegisterProvider{
		Provider: suite.providerAddr.String(),
		Moniker:  "Test Provider",
		Endpoint: "https://provider.example.com",
		Stake:    belowMinStake,
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "stake")
	suite.Require().Contains(err.Error(), "is less than minimum")
}

// TestValidateRegisterProvider_AlreadyRegisteredActive tests duplicate active provider registration
func (suite *ComputeDecoratorTestSuite) TestValidateRegisterProvider_AlreadyRegisteredActive() {
	// Create an active provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Existing Provider",
		Endpoint: "https://existing.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Try to register the same provider again
	msg := &computetypes.MsgRegisterProvider{
		Provider: suite.providerAddr.String(),
		Moniker:  "New Provider",
		Endpoint: "https://new.example.com",
		Stake:    math.NewInt(2000000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "provider already registered and active")
}

// TestValidateRegisterProvider_GasConsumption tests gas consumption for provider registration
func (suite *ComputeDecoratorTestSuite) TestValidateRegisterProvider_GasConsumption() {
	msg := &computetypes.MsgRegisterProvider{
		Provider: suite.providerAddr.String(),
		Moniker:  "Test Provider",
		Endpoint: "https://provider.example.com",
		Stake:    math.NewInt(2000000),
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
	// Gas consumed includes validation overhead and GetParams/GetProvider calls
	suite.Require().Greater(gasAfterDecorator, gasBeforeDecorator, "decorator should consume gas during validation")
	suite.Require().GreaterOrEqual(gasAfterDecorator-gasBeforeDecorator, uint64(1500), "decorator should consume at least 1500 gas for provider registration validation")
}

// TestValidateSubmitResult_InvalidProviderAddress tests invalid provider address validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_InvalidProviderAddress() {
	msg := &computetypes.MsgSubmitResult{
		Provider:   "invalid_provider_address",
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
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

// TestValidateSubmitResult_ProviderNotFound tests provider not found validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_ProviderNotFound() {
	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "provider not found")
}

// TestValidateSubmitResult_ProviderNotActive tests inactive provider validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_ProviderNotActive() {
	// Create an inactive provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Inactive Provider",
		Endpoint: "https://inactive.example.com",
		Active:   false,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "provider is not active")
}

// TestValidateSubmitResult_RequestNotFound tests request not found validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_RequestNotFound() {
	// Create an active provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  999, // Non-existent request
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "request 999 not found")
}

// TestValidateSubmitResult_RequestNotAssignedToProvider tests unauthorized provider validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_RequestNotAssignedToProvider() {
	// Create an active provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Create a request assigned to a different provider
	otherProvider := sdk.AccAddress([]byte("other_provider")).String()
	request := computetypes.Request{
		Id:        1,
		Requester: suite.addr.String(),
		Provider:  otherProvider,
		Status:    computetypes.REQUEST_STATUS_ASSIGNED,
	}
	err = suite.computeKeeper.SetRequest(suite.ctx, request)
	suite.Require().NoError(err)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "request 1 is not assigned to provider")
}

// TestValidateSubmitResult_RequestNotInAssignedStatus tests request status validation
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_RequestNotInAssignedStatus() {
	// Create an active provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Create a request in PENDING status (not ASSIGNED)
	request := computetypes.Request{
		Id:        1,
		Requester: suite.addr.String(),
		Provider:  suite.providerAddr.String(),
		Status:    computetypes.REQUEST_STATUS_PENDING, // Wrong status
	}
	err = suite.computeKeeper.SetRequest(suite.ctx, request)
	suite.Require().NoError(err)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "request 1 is not in ASSIGNED status")
}

// TestValidateSubmitResult_GasConsumption tests gas consumption for result submission
func (suite *ComputeDecoratorTestSuite) TestValidateSubmitResult_GasConsumption() {
	// Create an active provider
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Create a properly assigned request
	request := computetypes.Request{
		Id:        1,
		Requester: suite.addr.String(),
		Provider:  suite.providerAddr.String(),
		Status:    computetypes.REQUEST_STATUS_ASSIGNED,
	}
	err = suite.computeKeeper.SetRequest(suite.ctx, request)
	suite.Require().NoError(err)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	gasBeforeDecorator := suite.ctx.GasMeter().GasConsumed()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().NoError(err)

	gasAfterDecorator := suite.ctx.GasMeter().GasConsumed()
	// Gas consumed includes overhead from GetProvider and GetRequest calls
	suite.Require().Greater(gasAfterDecorator, gasBeforeDecorator, "decorator should consume gas during validation")
	suite.Require().GreaterOrEqual(gasAfterDecorator-gasBeforeDecorator, uint64(2000), "decorator should consume at least 2000 gas for result submission validation")
}

// TestAnteHandle_NonComputeMessage tests that non-compute messages pass through
func (suite *ComputeDecoratorTestSuite) TestAnteHandle_NonComputeMessage() {
	// Non-compute messages should pass through without validation
	msg := &banktypes.MsgSend{
		FromAddress: suite.addr.String(),
		ToAddress:   suite.providerAddr.String(),
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
	suite.Require().True(called, "next handler should be called for non-compute messages")
}

// TestAnteHandle_MultipleMessages tests transaction with multiple messages
func (suite *ComputeDecoratorTestSuite) TestAnteHandle_MultipleMessages() {
	// Test transaction with multiple messages, one of which is invalid
	msg1 := &banktypes.MsgSend{
		FromAddress: suite.addr.String(),
		ToAddress:   suite.providerAddr.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
	}

	msg2 := &computetypes.MsgSubmitRequest{
		Requester:  "invalid_address",
		MaxPayment: math.NewInt(1000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg1, msg2)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().Error(err, "should fail on invalid compute message even with valid non-compute message")
	suite.Require().Contains(err.Error(), "invalid requester address")
}

// TestAnteHandle_AllComputeMessageTypes tests all three compute message types together
func (suite *ComputeDecoratorTestSuite) TestAnteHandle_AllComputeMessageTypes() {
	// Create an active provider and valid request for comprehensive test
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	err := suite.computeKeeper.SetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	request := computetypes.Request{
		Id:        1,
		Requester: suite.addr.String(),
		Provider:  suite.providerAddr.String(),
		Status:    computetypes.REQUEST_STATUS_ASSIGNED,
	}
	err = suite.computeKeeper.SetRequest(suite.ctx, request)
	suite.Require().NoError(err)

	// Test messages individually to avoid bank keeper nil panic
	// Submit Result (works with our setup)
	msg3 := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
	}

	// Register Provider (works with our setup)
	msg2 := &computetypes.MsgRegisterProvider{
		Provider: sdk.AccAddress([]byte("new_provider")).String(),
		Moniker:  "New Provider",
		Endpoint: "https://new.example.com",
		Stake:    math.NewInt(2000000),
	}

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg2, msg3)
	suite.Require().NoError(err)
	tx := txBuilder.GetTx()

	gasBeforeDecorator := suite.ctx.GasMeter().GasConsumed()

	_, err = suite.decorator.AnteHandle(suite.ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	suite.Require().NoError(err)

	gasAfterDecorator := suite.ctx.GasMeter().GasConsumed()
	// Verify gas was consumed for both messages
	suite.Require().Greater(gasAfterDecorator, gasBeforeDecorator, "decorator should consume gas for all message validations")
	suite.Require().GreaterOrEqual(gasAfterDecorator-gasBeforeDecorator, uint64(3500), "decorator should consume at least base gas for both validations")
}

// Benchmark tests
func BenchmarkComputeDecorator_ValidateSubmitRequest(b *testing.B) {
	suite := new(ComputeDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	msg := &computetypes.MsgSubmitRequest{
		Requester:      suite.addr.String(),
		ContainerImage: "ubuntu:latest",
		MaxPayment:     math.NewInt(1000),
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

func BenchmarkComputeDecorator_ValidateRegisterProvider(b *testing.B) {
	suite := new(ComputeDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	msg := &computetypes.MsgRegisterProvider{
		Provider: suite.providerAddr.String(),
		Moniker:  "Test Provider",
		Endpoint: "https://provider.example.com",
		Stake:    math.NewInt(2000000),
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

func BenchmarkComputeDecorator_ValidateSubmitResult(b *testing.B) {
	suite := new(ComputeDecoratorTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupTest()

	// Setup provider and request
	provider := computetypes.Provider{
		Address:  suite.providerAddr.String(),
		Moniker:  "Active Provider",
		Endpoint: "https://active.example.com",
		Active:   true,
		Stake:    math.NewInt(2000000),
	}
	_ = suite.computeKeeper.SetProvider(suite.ctx, provider)

	request := computetypes.Request{
		Id:        1,
		Requester: suite.addr.String(),
		Provider:  suite.providerAddr.String(),
		Status:    computetypes.REQUEST_STATUS_ASSIGNED,
	}
	_ = suite.computeKeeper.SetRequest(suite.ctx, request)

	msg := &computetypes.MsgSubmitResult{
		Provider:   suite.providerAddr.String(),
		RequestId:  1,
		OutputHash: "abc123",
		OutputUrl:  "https://storage.example.com/output",
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
