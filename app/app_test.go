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
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

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


// TestExportAppStateAndValidators validates genesis export
func TestExportAppStateAndValidators(t *testing.T) {
	pawApp, ctx := keepertest.SetupTestApp(t)

	// Ensure transfer params are initialized to avoid panics during export
	pawApp.TransferKeeper.SetParams(ctx, ibctransfertypes.DefaultParams())
	// Seed crisis constant fee
	err := pawApp.CrisisKeeper.ConstantFee.Set(ctx, sdk.NewCoin("upaw", math.NewInt(1000)))
	require.NoError(t, err)
	// Seed mint state
	require.NoError(t, pawApp.MintKeeper.Minter.Set(ctx, minttypes.DefaultInitialMinter()))
	require.NoError(t, pawApp.MintKeeper.Params.Set(ctx, minttypes.DefaultParams()))
	// Seed gov params
	require.NoError(t, pawApp.GovKeeper.Params.Set(ctx, govv1.DefaultParams()))
	govmodule.InitGenesis(ctx, pawApp.AccountKeeper, pawApp.BankKeeper, pawApp.GovKeeper, govv1.DefaultGenesisState())
	// Seed distribution state
	pawApp.DistrKeeper.InitGenesis(ctx, *distrtypes.DefaultGenesisState())
	// Seed IBC client params
	pawApp.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.DefaultParams())
	pawApp.IBCKeeper.ClientKeeper.SetNextClientSequence(ctx, 0)
	pawApp.IBCKeeper.ConnectionKeeper.SetNextConnectionSequence(ctx, 0)
	pawApp.IBCKeeper.ChannelKeeper.SetNextChannelSequence(ctx, 0)
	pawApp.IBCKeeper.ConnectionKeeper.SetParams(ctx, ibcconnectiontypes.DefaultParams())
	pawApp.IBCKeeper.ChannelKeeper.SetParams(ctx, ibcchanneltypes.DefaultParams())

	// Export genesis
	exported, err := pawApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)
	require.NotNil(t, exported)
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

	creatorAddr, err := sdk.AccAddressFromBech32(msgCreatePool.Creator)
	require.NoError(suite.T(), err)

	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx,
		creatorAddr,
		msgCreatePool.TokenA,
		msgCreatePool.TokenB,
		msgCreatePool.AmountA,
		msgCreatePool.AmountB,
	)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), pool)
	require.Greater(suite.T(), pool.Id, uint64(0))

	// Verify pool exists
	pool, err = suite.app.DEXKeeper.GetPool(suite.ctx, pool.Id)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), pool)
	require.Equal(suite.T(), "upaw", pool.TokenA)
	require.Equal(suite.T(), "uusdt", pool.TokenB)
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
