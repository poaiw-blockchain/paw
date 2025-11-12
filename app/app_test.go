package app_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
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
	app *app.App
	ctx sdk.Context
}

func (suite *AppTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	suite.app = app.New(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-testnet-1"),
	)

	suite.ctx = suite.app.BaseApp.NewContext(false, cmtproto.Header{ChainID: "paw-testnet-1"})
}

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}

// TestNewApp validates app initialization
func TestNewApp(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	app := app.New(
		logger,
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		simtestutil.EmptyAppOptions{},
	)

	require.NotNil(t, app)
	require.NotNil(t, app.AccountKeeper)
	require.NotNil(t, app.BankKeeper)
	require.NotNil(t, app.StakingKeeper)
	require.NotNil(t, app.DexKeeper)
	require.NotNil(t, app.ComputeKeeper)
	require.NotNil(t, app.OracleKeeper)
}

// TestAppModules validates all required modules are registered
func TestAppModules(t *testing.T) {
	db := dbm.NewMemDB()
	app := app.New(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
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

	moduleManager := app.ModuleManager
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
	app := app.New(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		simtestutil.EmptyAppOptions{},
	)

	// Initialize chain
	genesisState := app.DefaultGenesis()
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	app.InitChain(
		abci.RequestInitChain{
			ChainId:       "paw-testnet-1",
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
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
		sdk.NewCoin("upaw", sdk.NewInt(10000000)),
		sdk.NewCoin("uusdt", sdk.NewInt(10000000)),
	)
	require.NoError(suite.T(), suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	require.NoError(suite.T(), suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	// Create pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: sdk.NewInt(1000000),
		AmountB: sdk.NewInt(2000000),
	}

	resp, err := suite.app.DexKeeper.CreatePool(suite.ctx, msgCreatePool)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Greater(suite.T(), resp.PoolId, uint64(0))

	// Verify pool exists
	pool, found := suite.app.DexKeeper.GetPool(suite.ctx, resp.PoolId)
	require.True(suite.T(), found)
	require.Equal(suite.T(), "upaw", pool.TokenA)
	require.Equal(suite.T(), "uusdt", pool.TokenB)
}

// TestComputeModuleIntegration tests Compute module integration
func (suite *AppTestSuite) TestComputeModuleIntegration() {
	// Create test provider account
	priv := secp256k1.GenPrivKey()
	providerAddr := sdk.AccAddress(priv.PubKey().Address())

	// Fund provider for stake
	stakeCoins := sdk.NewCoins(sdk.NewCoin("upaw", sdk.NewInt(1000000)))
	require.NoError(suite.T(), suite.app.BankKeeper.MintCoins(suite.ctx, computetypes.ModuleName, stakeCoins))
	require.NoError(suite.T(), suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, computetypes.ModuleName, providerAddr, stakeCoins))

	// Register provider
	msgRegister := &computetypes.MsgRegisterProvider{
		Provider: providerAddr.String(),
		Endpoint: "https://api.provider.io",
		Stake:    sdk.NewInt(100000),
	}

	_, err := suite.app.ComputeKeeper.RegisterProvider(suite.ctx, msgRegister)
	require.NoError(suite.T(), err)

	// Verify provider registered
	provider, found := suite.app.ComputeKeeper.GetProvider(suite.ctx, providerAddr.String())
	require.True(suite.T(), found)
	require.Equal(suite.T(), "https://api.provider.io", provider.Endpoint)
}

// TestOracleModuleIntegration tests Oracle module integration
func (suite *AppTestSuite) TestOracleModuleIntegration() {
	// Create test oracle account
	priv := secp256k1.GenPrivKey()
	oracleAddr := sdk.AccAddress(priv.PubKey().Address())

	// Register oracle (typically validator)
	msgRegister := &oracletypes.MsgRegisterOracle{
		Validator: oracleAddr.String(),
	}

	_, err := suite.app.OracleKeeper.RegisterOracle(suite.ctx, msgRegister)
	require.NoError(suite.T(), err)

	// Submit price feed
	msgSubmit := &oracletypes.MsgSubmitPrice{
		Oracle: oracleAddr.String(),
		Asset:  "BTC/USD",
		Price:  sdk.MustNewDecFromStr("45000.00"),
	}

	_, err = suite.app.OracleKeeper.SubmitPrice(suite.ctx, msgSubmit)
	require.NoError(suite.T(), err)

	// Verify price submitted
	price, found := suite.app.OracleKeeper.GetPrice(suite.ctx, "BTC/USD", oracleAddr.String())
	require.True(suite.T(), found)
	require.Equal(suite.T(), sdk.MustNewDecFromStr("45000.00"), price.Price)
}

// TestModuleAccountsExist validates module accounts are created
func TestModuleAccountsExist(t *testing.T) {
	db := dbm.NewMemDB()
	app := app.New(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		simtestutil.EmptyAppOptions{},
	)

	ctx := app.BaseApp.NewContext(false, cmtproto.Header{})

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
