package app_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

type AppTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *AppTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

// TestNewApp validates app initialization
func TestNewApp(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	app := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)

	require.NotNil(t, app)
	require.NotNil(t, app.AccountKeeper)
	require.NotNil(t, app.BankKeeper)
	require.NotNil(t, app.StakingKeeper)
	require.NotNil(t, app.DEXKeeper)
	require.NotNil(t, app.ComputeKeeper)
	require.NotNil(t, app.OracleKeeper)
}

// TestAppModules validates all required modules are registered
func TestAppModules(t *testing.T) {
	t.Skip("TODO: Add public accessor method for ModuleManager in PAWApp")
	// db := dbm.NewMemDB()
	// app := app.NewPAWApp(
	// 	log.NewNopLogger(),
	// 	db,
	// 	nil,
	// 	true,
	// 	simtestutil.EmptyAppOptions{},
	// )
	//
	// // Check standard Cosmos SDK modules
	// requiredModules := []string{
	// 	authtypes.ModuleName,
	// 	banktypes.ModuleName,
	// 	"staking",
	// 	"distribution",
	// 	"slashing",
	// 	"gov",
	// 	"params",
	// 	"upgrade",
	// 	"evidence",
	// 	"feegrant",
	// 	"authz",
	// 	"consensus",
	// 	// PAW custom modules
	// 	dextypes.ModuleName,
	// 	computetypes.ModuleName,
	// 	oracletypes.ModuleName,
	// }
	//
	// // TODO: Access module manager through public method
}

// TestExportAppStateAndValidators validates genesis export
func TestExportAppStateAndValidators(t *testing.T) {
	pawApp, _ := keepertest.SetupTestApp(t)

	// Export genesis
	exported, err := pawApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)
	require.NotNil(t, exported.AppState)
	require.NotNil(t, exported.Validators)
	require.Greater(t, len(exported.Validators), 0, "should have at least one validator")
}

// TestDexModuleIntegration tests DEX module integration
func (suite *AppTestSuite) TestDexModuleIntegration() {
	// Create test account
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Fund account
	coins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(10000000)),
	)
	require.NoError(suite.T(), suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	require.NoError(suite.T(), suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	// Create pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	poolID, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		msgCreatePool.Creator,
		msgCreatePool.TokenA,
		msgCreatePool.TokenB,
		msgCreatePool.AmountA,
		msgCreatePool.AmountB,
	)
	require.NoError(suite.T(), err)
	require.Greater(suite.T(), poolID, uint64(0))

	// Verify pool exists
	pool := suite.app.DEXKeeper.GetPool(suite.ctx, poolID)
	require.NotNil(suite.T(), pool)
	require.Equal(suite.T(), "upaw", pool.TokenA)
	require.Equal(suite.T(), "uusdt", pool.TokenB)
}

// TestComputeModuleIntegration tests Compute module integration
func (suite *AppTestSuite) TestComputeModuleIntegration() {
	// TODO: Implement RegisterProvider and GetProvider methods in compute keeper
	suite.T().Skip("Waiting for compute keeper methods implementation")
	// Create test provider account
	priv := secp256k1.GenPrivKey()
	providerAddr := sdk.AccAddress(priv.PubKey().Address())

	// Fund provider for stake
	stakeCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	require.NoError(suite.T(), suite.app.BankKeeper.MintCoins(suite.ctx, computetypes.ModuleName, stakeCoins))
	require.NoError(suite.T(), suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, computetypes.ModuleName, providerAddr, stakeCoins))

	// TODO: Register provider when method is implemented
	// msgRegister := &computetypes.MsgRegisterProvider{
	// 	Provider: providerAddr.String(),
	// 	Endpoint: "https://api.provider.io",
	// 	Stake:    math.NewInt(100000),
	// }
}

// TestOracleModuleIntegration tests Oracle module integration
func (suite *AppTestSuite) TestOracleModuleIntegration() {
	// TODO: Implement RegisterOracle, SubmitPrice, and GetPrice methods in oracle keeper
	suite.T().Skip("Waiting for oracle keeper methods implementation")
	// Create test oracle account
	priv := secp256k1.GenPrivKey()
	oracleAddr := sdk.AccAddress(priv.PubKey().Address())

	// TODO: Implement oracle functionality
	_ = oracleAddr
}

// TestModuleAccountsExist validates module accounts are created
func TestModuleAccountsExist(t *testing.T) {
	testApp, ctx := keepertest.SetupTestApp(t)

	// Check module accounts exist
	moduleAccounts := []string{
		authtypes.FeeCollectorName,
		"distribution",
		"bonded_tokens_pool",
		"not_bonded_tokens_pool",
		"gov",
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	}

	for _, moduleName := range moduleAccounts {
		acc := testApp.AccountKeeper.GetModuleAccount(ctx, moduleName)
		require.NotNil(t, acc, "module account %s should exist", moduleName)
	}
}
