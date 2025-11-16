package app_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
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
	db := dbm.NewMemDB()
	suite.app = app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-testnet-1"),
	)

	suite.ctx = suite.app.BaseApp.NewContext(false).WithChainID("paw-testnet-1")
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
	db := dbm.NewMemDB()
	app := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)

	// Check standard Cosmos SDK modules
	requiredModules := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		"staking",
		"distribution",
		"slashing",
		"gov",
		"params",
		"upgrade",
		"evidence",
		"feegrant",
		"authz",
		"consensus",
		// PAW custom modules
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
	}

	moduleManager := app.mm
	require.NotNil(t, moduleManager)

	for _, moduleName := range requiredModules {
		// Module should be registered in the module manager
		require.Contains(t, moduleManager.Modules, moduleName,
			"module %s should be registered", moduleName)
	}
}

// TestExportAppStateAndValidators validates genesis export
func TestExportAppStateAndValidators(t *testing.T) {
	db := dbm.NewMemDB()
	app := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)

	// Initialize chain
	genesisState := app.NewDefaultGenesisState("paw-testnet-1")
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	_, err = app.InitChain(
		&abci.RequestInitChain{
			ChainId:       "paw-testnet-1",
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	require.NoError(t, err)
	app.Commit()

	// Export genesis
	exported, err := app.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)
	require.NotNil(t, exported.AppState)
	require.NotNil(t, exported.Validators)
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
	pool, found := suite.app.DEXKeeper.GetPool(suite.ctx, poolID)
	require.True(suite.T(), found)
	require.Equal(suite.T(), "upaw", pool.TokenA)
	require.Equal(suite.T(), "uusdt", pool.TokenB)
}

// TestComputeModuleIntegration tests Compute module integration
func (suite *AppTestSuite) TestComputeModuleIntegration() {
	suite.T().Skip("TODO: Implement RegisterProvider and GetProvider methods in compute keeper")
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
	suite.T().Skip("TODO: Implement RegisterOracle, SubmitPrice, and GetPrice methods in oracle keeper")
	// Create test oracle account
	priv := secp256k1.GenPrivKey()
	oracleAddr := sdk.AccAddress(priv.PubKey().Address())

	// TODO: Implement oracle functionality
	_ = oracleAddr
}

// TestModuleAccountsExist validates module accounts are created
func TestModuleAccountsExist(t *testing.T) {
	db := dbm.NewMemDB()
	app := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)

	ctx := app.BaseApp.NewContext(false)

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
		acc := app.AccountKeeper.GetModuleAccount(ctx, moduleName)
		require.NotNil(t, acc, "module account %s should exist", moduleName)
	}
}
