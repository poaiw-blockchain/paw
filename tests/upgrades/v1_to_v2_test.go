//go:build integration
// +build integration

package upgrades_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// V1ToV2TestSuite tests the v1 to v2 upgrade path
type V1ToV2TestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func TestV1ToV2Suite(t *testing.T) {
	suite.Run(t, new(V1ToV2TestSuite))
}

func (s *V1ToV2TestSuite) SetupTest() {
	s.app = setupTestApp(s.T())
	s.ctx = s.app.NewContextLegacy(false, tmproto.Header{
		Height: 1,
		Time:   time.Now(),
	})
}

// TestBasicV1ToV2Migration tests the basic v1 to v2 migration
func (s *V1ToV2TestSuite) TestBasicV1ToV2Migration() {
	// Create initial v1 state
	s.createV1State()

	// Capture state before upgrade
	stateBefore := s.captureState()

	// Execute upgrade
	plan := upgradetypes.Plan{
		Name:   "v1-to-v2",
		Height: 100,
		Info:   "Test v1 to v2 upgrade",
	}

	err := s.app.UpgradeKeeper.ScheduleUpgrade(s.ctx, plan)
	s.Require().NoError(err)

	s.ctx = s.ctx.WithBlockHeight(plan.Height)
	fromVM := s.app.ModuleManager().GetVersionMap()

	// Execute migrations
	toVM, err := s.app.ModuleManager().RunMigrations(s.ctx, s.app.Configurator(), fromVM)
	s.Require().NoError(err)
	s.Require().NotNil(toVM)

	// Verify module versions upgraded
	s.Require().Equal(uint64(2), toVM["compute"])
	s.Require().Equal(uint64(2), toVM["dex"])
	s.Require().Equal(uint64(2), toVM["oracle"])

	// Verify state preservation
	stateAfter := s.captureState()
	s.verifyStatePreserved(stateBefore, stateAfter)
}

// TestComputeModuleMigration tests compute module specific migration
func (s *V1ToV2TestSuite) TestComputeModuleMigration() {
	// Create compute state
	providerAddr := sdk.AccAddress([]byte("compute_provider_____"))
	provider := computetypes.Provider{
		Address:        providerAddr.String(),
		Moniker:        "test-provider",
		Endpoint:       "https://provider.test",
		AvailableSpecs: computetypes.ComputeSpec{CpuCores: 100, MemoryMb: 1024, StorageGb: 50, TimeoutSeconds: 600},
		Pricing: computetypes.Pricing{
			CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.001"),
			MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
		},
		Stake:                  math.NewInt(1_000_000),
		Reputation:             0, // Will be fixed by migration
		TotalRequestsCompleted: 100,
		TotalRequestsFailed:    10,
		Active:                 true,
		RegisteredAt:           time.Now(),
		LastActiveAt:           time.Now(),
	}

	err := s.app.ComputeKeeper.SetProvider(s.ctx, provider)
	s.Require().NoError(err)

	// Run migration
	plan := upgradetypes.Plan{Name: "v1-to-v2", Height: 100}
	err = s.app.UpgradeKeeper.ScheduleUpgrade(s.ctx, plan)
	s.Require().NoError(err)

	s.ctx = s.ctx.WithBlockHeight(plan.Height)
	fromVM := s.app.ModuleManager().GetVersionMap()
	_, err = s.app.ModuleManager().RunMigrations(s.ctx, s.app.Configurator(), fromVM)
	s.Require().NoError(err)

	// Verify provider reputation was calculated
	migratedProvider, err := s.app.ComputeKeeper.GetProvider(s.ctx, providerAddr)
	s.Require().NoError(err)
	s.Require().Greater(migratedProvider.Reputation, uint32(0))

	// Should be (100 / 110) * 100 = 90
	s.Require().InDelta(90, migratedProvider.Reputation, 2)
}

// TestDEXModuleMigration tests DEX module specific migration
func (s *V1ToV2TestSuite) TestDEXModuleMigration() {
	// Create DEX state with negative values that need fixing
	creator := sdk.AccAddress([]byte("pool_creator_________"))

	// Fund creator
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 10_000_000),
		sdk.NewInt64Coin("uusdc", 10_000_000),
	)
	s.Require().NoError(s.app.BankKeeper.MintCoins(s.ctx, dextypes.ModuleName, coins))
	s.Require().NoError(s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, dextypes.ModuleName, creator, coins))

	// Create pool
	pool, err := s.app.DEXKeeper.CreatePool(s.ctx, creator, "upaw", "uusdc",
		math.NewInt(1_000_000), math.NewInt(1_000_000))
	s.Require().NoError(err)

	// Verify pool created
	s.Require().Equal(uint64(1), pool.Id)

	// Run migration
	plan := upgradetypes.Plan{Name: "v1-to-v2", Height: 100}
	err = s.app.UpgradeKeeper.ScheduleUpgrade(s.ctx, plan)
	s.Require().NoError(err)

	s.ctx = s.ctx.WithBlockHeight(plan.Height)
	fromVM := s.app.ModuleManager().GetVersionMap()
	_, err = s.app.ModuleManager().RunMigrations(s.ctx, s.app.Configurator(), fromVM)
	s.Require().NoError(err)

	// Verify pool still exists and has valid state
	migratedPool, err := s.app.DEXKeeper.GetPool(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(pool.Id, migratedPool.Id)
	s.Require().True(migratedPool.ReserveA.IsPositive())
	s.Require().True(migratedPool.ReserveB.IsPositive())
	s.Require().True(migratedPool.TotalShares.IsPositive())

	// Verify token ordering (should be lexicographic)
	s.Require().LessOrEqual(migratedPool.TokenA, migratedPool.TokenB)
}

// TestOracleModuleMigration tests Oracle module specific migration
func (s *V1ToV2TestSuite) TestOracleModuleMigration() {
	// Create oracle state
	price := oracletypes.Price{
		Asset:         "UPAW/USD",
		Price:         math.LegacyMustNewDecFromStr("1.50"),
		BlockHeight:   s.ctx.BlockHeight(),
		BlockTime:     s.ctx.BlockTime().Unix(),
		NumValidators: 1,
	}

	err := s.app.OracleKeeper.SetPrice(s.ctx, price)
	s.Require().NoError(err)

	// Create validator oracle
	valAddr := sdk.ValAddress([]byte("validator_for_oracle"))
	validatorOracle := oracletypes.ValidatorOracle{
		ValidatorAddr:    valAddr.String(),
		MissCounter:      0,
		TotalSubmissions: 10,
		IsActive:         true,
		GeographicRegion: "us-east",
	}

	err = s.app.OracleKeeper.SetValidatorOracle(s.ctx, validatorOracle)
	s.Require().NoError(err)

	// Run migration
	plan := upgradetypes.Plan{Name: "v1-to-v2", Height: 100}
	err = s.app.UpgradeKeeper.ScheduleUpgrade(s.ctx, plan)
	s.Require().NoError(err)

	s.ctx = s.ctx.WithBlockHeight(plan.Height)
	fromVM := s.app.ModuleManager().GetVersionMap()
	_, err = s.app.ModuleManager().RunMigrations(s.ctx, s.app.Configurator(), fromVM)
	s.Require().NoError(err)

	// Verify price still exists
	migratedPrice, err := s.app.OracleKeeper.GetPrice(s.ctx, "UPAW/USD")
	s.Require().NoError(err)
	s.Require().Equal(price.Asset, migratedPrice.Asset)
	s.Require().True(migratedPrice.Price.IsPositive())

	// Verify validator oracle still exists
	migratedValidator, err := s.app.OracleKeeper.GetValidatorOracle(s.ctx, valAddr)
	s.Require().NoError(err)
	s.Require().Equal(validatorOracle.ValidatorAddr, migratedValidator.ValidatorAddr)
	s.Require().True(migratedValidator.IsActive)
}

// TestMigrationIdempotency tests that running migration twice produces same result
func (s *V1ToV2TestSuite) TestMigrationIdempotency() {
	// Create initial state
	s.createV1State()

	// First migration
	plan1 := upgradetypes.Plan{Name: "v1-to-v2-first", Height: 100}
	err := s.app.UpgradeKeeper.ScheduleUpgrade(s.ctx, plan1)
	s.Require().NoError(err)

	ctx1 := s.ctx.WithBlockHeight(plan1.Height)
	fromVM1 := s.app.ModuleManager().GetVersionMap()
	toVM1, err := s.app.ModuleManager().RunMigrations(ctx1, s.app.Configurator(), fromVM1)
	s.Require().NoError(err)

	state1 := s.captureState()

	// Second migration (should be idempotent)
	plan2 := upgradetypes.Plan{Name: "v1-to-v2-second", Height: 200}
	err = s.app.UpgradeKeeper.ScheduleUpgrade(ctx1, plan2)
	s.Require().NoError(err)

	ctx2 := ctx1.WithBlockHeight(plan2.Height)
	// Reset version map to v1 for testing
	fromVM2 := map[string]uint64{
		"compute": 1,
		"dex":     1,
		"oracle":  1,
	}
	toVM2, err := s.app.ModuleManager().RunMigrations(ctx2, s.app.Configurator(), fromVM2)
	s.Require().NoError(err)

	// Version maps should be identical
	s.Require().Equal(toVM1, toVM2)

	state2 := s.captureState()

	// States should be equivalent
	s.Require().Equal(state1.computeProviderCount, state2.computeProviderCount)
	s.Require().Equal(state1.dexPoolCount, state2.dexPoolCount)
	s.Require().Equal(state1.oraclePriceCount, state2.oraclePriceCount)
}

// TestMigrationDeterminism tests deterministic behavior across multiple instances
func (s *V1ToV2TestSuite) TestMigrationDeterminism() {
	// Create two separate app instances
	app1 := setupTestApp(s.T())
	ctx1 := app1.NewContextLegacy(false, tmproto.Header{Height: 1, Time: time.Now()})

	app2 := setupTestApp(s.T())
	ctx2 := app2.NewContextLegacy(false, tmproto.Header{Height: 1, Time: time.Now()})

	// Create identical state in both apps
	s.createIdenticalState(app1, ctx1)
	s.createIdenticalState(app2, ctx2)

	// Run migration on both
	plan := upgradetypes.Plan{Name: "v1-to-v2", Height: 100}

	err := app1.UpgradeKeeper.ScheduleUpgrade(ctx1, plan)
	s.Require().NoError(err)
	err = app2.UpgradeKeeper.ScheduleUpgrade(ctx2, plan)
	s.Require().NoError(err)

	ctx1 = ctx1.WithBlockHeight(plan.Height)
	ctx2 = ctx2.WithBlockHeight(plan.Height)

	fromVM1 := app1.ModuleManager().GetVersionMap()
	fromVM2 := app2.ModuleManager().GetVersionMap()

	toVM1, err := app1.ModuleManager().RunMigrations(ctx1, app1.Configurator(), fromVM1)
	s.Require().NoError(err)

	toVM2, err := app2.ModuleManager().RunMigrations(ctx2, app2.Configurator(), fromVM2)
	s.Require().NoError(err)

	// Version maps should be identical
	s.Require().Equal(toVM1, toVM2)

	// Export and compare genesis
	genesis1, err := app1.ExportAppStateAndValidators(false, []string{}, []string{})
	s.Require().NoError(err)

	genesis2, err := app2.ExportAppStateAndValidators(false, []string{}, []string{})
	s.Require().NoError(err)

	// Heights should match
	s.Require().Equal(genesis1.Height, genesis2.Height)
}

// Helper methods

type stateSnapshot struct {
	computeProviderCount int
	computeRequestCount  int
	dexPoolCount         int
	dexLiquidityCount    int
	oraclePriceCount     int
	oracleValidatorCount int
}

func (s *V1ToV2TestSuite) createV1State() {
	creator := sdk.AccAddress([]byte("test_creator_________"))

	// Fund accounts
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 20_000_000),
		sdk.NewInt64Coin("uusdc", 20_000_000),
	)
	s.Require().NoError(s.app.BankKeeper.MintCoins(s.ctx, dextypes.ModuleName, coins))
	s.Require().NoError(s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, dextypes.ModuleName, creator, coins))

	// Create DEX pool
	_, err := s.app.DEXKeeper.CreatePool(s.ctx, creator, "upaw", "uusdc",
		math.NewInt(5_000_000), math.NewInt(5_000_000))
	s.Require().NoError(err)

	// Create oracle price
	price := oracletypes.Price{
		Asset:         "UPAW/USD",
		Price:         math.LegacyMustNewDecFromStr("1.00"),
		BlockHeight:   s.ctx.BlockHeight(),
		BlockTime:     s.ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	s.Require().NoError(s.app.OracleKeeper.SetPrice(s.ctx, price))

	// Create compute provider
	providerAddr := sdk.AccAddress([]byte("compute_provider_____"))
	provider := computetypes.Provider{
		Address:        providerAddr.String(),
		Moniker:        "migration-test",
		Endpoint:       "https://provider.test",
		AvailableSpecs: computetypes.ComputeSpec{CpuCores: 100, MemoryMb: 1024, StorageGb: 50, TimeoutSeconds: 600},
		Pricing: computetypes.Pricing{
			CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.001"),
			MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
		},
		Stake:                  math.NewInt(1_000_000),
		Reputation:             80,
		TotalRequestsCompleted: 0,
		TotalRequestsFailed:    0,
		Active:                 true,
		RegisteredAt:           time.Now(),
		LastActiveAt:           time.Now(),
	}
	s.Require().NoError(s.app.ComputeKeeper.SetProvider(s.ctx, provider))
}

func (s *V1ToV2TestSuite) createIdenticalState(app *app.PAWApp, ctx sdk.Context) {
	creator := sdk.AccAddress([]byte("identical_creator____"))

	coins := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 10_000_000),
		sdk.NewInt64Coin("uusdc", 10_000_000),
	)
	s.Require().NoError(app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, coins))
	s.Require().NoError(app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, creator, coins))

	_, err := app.DEXKeeper.CreatePool(ctx, creator, "upaw", "uusdc",
		math.NewInt(1_000_000), math.NewInt(1_000_000))
	s.Require().NoError(err)
}

func (s *V1ToV2TestSuite) captureState() stateSnapshot {
	snapshot := stateSnapshot{}

	// Count compute providers
	computeStore := s.ctx.KVStore(s.app.GetKey(computetypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(computeStore, []byte{0x02}) // Provider prefix
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		snapshot.computeProviderCount++
	}

	// Count DEX pools
	dexStore := s.ctx.KVStore(s.app.GetKey(dextypes.StoreKey))
	poolIter := storetypes.KVStorePrefixIterator(dexStore, []byte{0x01}) // Pool prefix
	defer poolIter.Close()
	for ; poolIter.Valid(); poolIter.Next() {
		snapshot.dexPoolCount++
	}

	// Count oracle prices
	oracleStore := s.ctx.KVStore(s.app.GetKey(oracletypes.StoreKey))
	priceIter := storetypes.KVStorePrefixIterator(oracleStore, []byte{0x01}) // Price prefix
	defer priceIter.Close()
	for ; priceIter.Valid(); priceIter.Next() {
		snapshot.oraclePriceCount++
	}

	return snapshot
}

func (s *V1ToV2TestSuite) verifyStatePreserved(before, after stateSnapshot) {
	s.Require().Equal(before.computeProviderCount, after.computeProviderCount, "compute provider count should be preserved")
	s.Require().Equal(before.dexPoolCount, after.dexPoolCount, "DEX pool count should be preserved")
	s.Require().Equal(before.oraclePriceCount, after.oraclePriceCount, "oracle price count should be preserved")
}

func setupTestApp(t *testing.T) *app.PAWApp {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		emptyAppOptions{},
	)

	require.NotNil(t, pawApp)
	return pawApp
}

type emptyAppOptions struct{}

func (emptyAppOptions) Get(_ string) interface{} {
	return nil
}
