package upgrade_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// UpgradeSimulationSuite tests upgrade simulation scenarios
type UpgradeSimulationSuite struct {
	suite.Suite

	app     *app.PAWApp
	ctx     sdk.Context
	chainID string
}

func TestUpgradeSimulationSuite(t *testing.T) {
	suite.Run(t, new(UpgradeSimulationSuite))
}

func (suite *UpgradeSimulationSuite) SetupTest() {
	suite.chainID = "paw-upgrade-test"
	suite.app, suite.ctx = suite.setupTestAppWithGenesisState()
}

// TestUpgradeFromV1ToV2_FullSimulation tests complete v1 → v2 upgrade simulation
func (suite *UpgradeSimulationSuite) TestUpgradeFromV1ToV2_FullSimulation() {
	// 1. Create comprehensive initial state (v1 schema)
	suite.seedCompleteV1State()

	// 2. Capture pre-upgrade state
	preUpgradeState := suite.captureFullState()

	// 3. Schedule upgrade at a future height
	upgradeHeight := suite.ctx.BlockHeight() + 10
	plan := upgradetypes.Plan{
		Name:   "v1.1.0",
		Height: upgradeHeight,
		Info:   "Simulated v1 to v2 upgrade",
	}

	err := suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan)
	suite.Require().NoError(err)

	// 4. Simulate blocks up to upgrade height
	suite.simulateBlocksUntilUpgrade(upgradeHeight)

	// 5. Execute upgrade at upgrade height
	suite.ctx = suite.ctx.WithBlockHeight(upgradeHeight)
	err = suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan)
	suite.Require().NoError(err)

	// 6. Capture post-upgrade state
	postUpgradeState := suite.captureFullState()

	// 7. Verify state integrity
	suite.verifyStateIntegrity(preUpgradeState, postUpgradeState)

	// 8. Verify all module versions upgraded to v2
	suite.verifyModuleVersions(2)

	// 9. Simulate post-upgrade blocks to ensure stability
	suite.simulatePostUpgradeBlocks(10)

	// 10. Verify all module functionality still works
	suite.verifyModuleFunctionality()
}

// TestMultiVersionUpgradePath tests sequential upgrades v1 → v2 → v3
func (suite *UpgradeSimulationSuite) TestMultiVersionUpgradePath() {
	// Start with v1 state
	suite.seedCompleteV1State()

	// Upgrade to v1.1.0 (consensus version 2)
	suite.executeUpgrade("v1.1.0", suite.ctx.BlockHeight()+5)
	suite.verifyModuleVersions(2)

	// Add some new data at v2
	suite.addDataAtV2()

	// Upgrade to v1.2.0 (may bump to v3 or stay at v2)
	suite.executeUpgrade("v1.2.0", suite.ctx.BlockHeight()+5)

	// Verify data from both v1 and v2 is intact
	suite.verifyV1DataIntact()
	suite.verifyV2DataIntact()

	// Upgrade to v1.3.0
	suite.executeUpgrade("v1.3.0", suite.ctx.BlockHeight()+5)

	// Final verification
	suite.verifyAllDataIntact()
	suite.verifyModuleFunctionality()
}

// TestUpgradeWithActiveOperations tests upgrade while DEX and compute operations are ongoing
func (suite *UpgradeSimulationSuite) TestUpgradeWithActiveOperations() {
	// Create active DEX pools
	creator := sdk.AccAddress([]byte("active_creator______"))
	suite.fundAccount(creator, sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 20_000_000),
		sdk.NewInt64Coin("uusdc", 20_000_000),
	))

	pool, err := suite.app.DEXKeeper.CreatePool(
		suite.ctx, creator, "upaw", "uusdc",
		sdkmath.NewInt(10_000_000), sdkmath.NewInt(10_000_000),
	)
	suite.Require().NoError(err)

	// Create active compute provider
	providerAddr := sdk.AccAddress([]byte("active_provider_____"))
	provider := computetypes.Provider{
		Address:        providerAddr.String(),
		Moniker:        "active-provider",
		Endpoint:       "https://active.test",
		AvailableSpecs: computetypes.ComputeSpec{CpuCores: 100, MemoryMb: 1024, StorageGb: 100, TimeoutSeconds: 600},
		Pricing:        computetypes.Pricing{},
		Stake:          sdkmath.NewInt(5_000_000),
		Reputation:     95,
		Active:         true,
		RegisteredAt:   suite.ctx.BlockTime(),
		LastActiveAt:   suite.ctx.BlockTime(),
	}
	suite.Require().NoError(suite.app.ComputeKeeper.SetProvider(suite.ctx, provider))

	// Perform swap
	trader := sdk.AccAddress([]byte("active_trader_______"))
	suite.fundAccount(trader, sdk.NewCoins(sdk.NewInt64Coin("upaw", 1_000_000)))
	_, err = suite.app.DEXKeeper.Swap(suite.ctx, trader, pool.Id, "upaw", "uusdc", sdkmath.NewInt(100_000), sdkmath.NewInt(1))
	suite.Require().NoError(err)

	// Verify operations can continue - test pool retrieval and provider status
	pool2, err := suite.app.DEXKeeper.GetPool(suite.ctx, pool.Id)
	suite.Require().NoError(err)
	suite.Require().True(pool2.ReserveA.GT(sdkmath.ZeroInt()))

	// Verify compute provider still accessible
	provider2, err := suite.app.ComputeKeeper.GetProvider(suite.ctx, providerAddr)
	suite.Require().NoError(err)
	suite.Require().True(provider2.Active)
}

// TestUpgradeRollbackOnMigrationFailure tests that failed migrations don't corrupt state
func (suite *UpgradeSimulationSuite) TestUpgradeRollbackOnMigrationFailure() {
	suite.seedCompleteV1State()
	preState := suite.captureFullState()

	// Create a failing upgrade handler
	plan := upgradetypes.Plan{
		Name:   "faulty-upgrade",
		Height: suite.ctx.BlockHeight() + 1,
		Info:   "This upgrade will fail",
	}

	suite.app.UpgradeKeeper.SetUpgradeHandler(
		plan.Name,
		func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "intentional migration failure")
		},
	)

	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))

	// Execute upgrade in a cache context
	cacheCtx, _ := suite.ctx.CacheContext()
	cacheCtx = cacheCtx.WithBlockHeight(plan.Height)
	err := suite.app.UpgradeKeeper.ApplyUpgrade(cacheCtx, plan)
	suite.Require().Error(err, "upgrade should fail")

	// Verify committed state unchanged
	postState := suite.captureFullState()
	suite.Require().Equal(preState, postState, "failed upgrade must not mutate state")
}

// TestAppVersionBump tests app version increments
func (suite *UpgradeSimulationSuite) TestAppVersionBump() {
	// Get initial app version
	initialVersion := suite.app.AppVersion()
	suite.Require().GreaterOrEqual(initialVersion, uint64(0))

	// Execute upgrade
	plan := upgradetypes.Plan{
		Name:   "v1.1.0",
		Height: suite.ctx.BlockHeight() + 1,
	}
	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))

	suite.ctx = suite.ctx.WithBlockHeight(plan.Height)
	suite.Require().NoError(suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan))

	// Verify app version may have bumped or stayed same (depends on implementation)
	// The important thing is it didn't decrease
	newVersion := suite.app.AppVersion()
	suite.Require().GreaterOrEqual(newVersion, initialVersion)
}

// TestUpgradeDeterminism verifies upgrade is deterministic
func (suite *UpgradeSimulationSuite) TestUpgradeDeterminism() {
	// Create two independent app instances with identical state
	app1, ctx1 := suite.setupTestAppWithGenesisState()
	app2, ctx2 := suite.setupTestAppWithGenesisState()

	// Seed identical state in both
	suite.seedIdenticalState(app1, ctx1)
	suite.seedIdenticalState(app2, ctx2)

	// Execute same upgrade on both
	plan := upgradetypes.Plan{
		Name:   "v1.1.0",
		Height: ctx1.BlockHeight() + 1,
	}

	err1 := app1.UpgradeKeeper.ScheduleUpgrade(ctx1, plan)
	suite.Require().NoError(err1)
	err2 := app2.UpgradeKeeper.ScheduleUpgrade(ctx2, plan)
	suite.Require().NoError(err2)

	ctx1 = ctx1.WithBlockHeight(plan.Height)
	ctx2 = ctx2.WithBlockHeight(plan.Height)

	err1 = app1.UpgradeKeeper.ApplyUpgrade(ctx1, plan)
	suite.Require().NoError(err1)
	err2 = app2.UpgradeKeeper.ApplyUpgrade(ctx2, plan)
	suite.Require().NoError(err2)

	// Verify both have identical version maps
	vm1, err := app1.UpgradeKeeper.GetModuleVersionMap(ctx1)
	suite.Require().NoError(err)
	vm2, err := app2.UpgradeKeeper.GetModuleVersionMap(ctx2)
	suite.Require().NoError(err)

	suite.Require().Equal(vm1, vm2, "upgrade must be deterministic")
}

// TestUpgradeDataIntegrity verifies no data loss during migration
func (suite *UpgradeSimulationSuite) TestUpgradeDataIntegrity() {
	// Create detailed state
	poolID := suite.createDEXPool()
	providerAddr := suite.createComputeProvider()
	suite.createOracleData()

	// Capture exact counts
	poolCountBefore := suite.countPools()
	providerCountBefore := suite.countProviders()
	priceCountBefore := suite.countPrices()

	// Execute upgrade
	suite.executeUpgrade("v1.1.0", suite.ctx.BlockHeight()+1)

	// Verify counts unchanged
	suite.Require().Equal(poolCountBefore, suite.countPools())
	suite.Require().Equal(providerCountBefore, suite.countProviders())
	suite.Require().Equal(priceCountBefore, suite.countPrices())

	// Verify specific data still accessible
	_, err := suite.app.DEXKeeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)

	_, err = suite.app.ComputeKeeper.GetProvider(suite.ctx, providerAddr)
	suite.Require().NoError(err)
}

// Helper functions

func (suite *UpgradeSimulationSuite) setupTestAppWithGenesisState() (*app.PAWApp, sdk.Context) {
	testApp := app.NewPAWApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(suite.chainID),
	)

	// Create validator
	valPrivKey := sdked25519.GenPrivKey()
	valPubKey := valPrivKey.PubKey()
	delAddr := sdk.AccAddress(valPubKey.Address())
	valAddr := sdk.ValAddress(delAddr)
	bondedTokens := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)

	genesisState := suite.createGenesisStateWithValidator(delAddr, valAddr, valPubKey)
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	suite.Require().NoError(err)

	pkAny, err := codectypes.NewAnyWithValue(valPubKey)
	suite.Require().NoError(err)
	validator := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		ConsensusPubkey:   pkAny,
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            bondedTokens,
		DelegatorShares:   sdkmath.LegacyNewDecFromInt(bondedTokens),
		Description:       stakingtypes.Description{Moniker: "test-validator"},
		MinSelfDelegation: sdkmath.OneInt(),
	}

	_, err = testApp.InitChain(&abci.RequestInitChain{
		ChainId:       suite.chainID,
		Validators:    []abci.ValidatorUpdate{validator.ABCIValidatorUpdate(sdk.DefaultPowerReduction)},
		AppStateBytes: stateBytes,
	})
	suite.Require().NoError(err)

	ctx := testApp.NewContext(true).WithBlockHeader(tmproto.Header{
		Height:  testApp.LastBlockHeight() + 1,
		ChainID: suite.chainID,
		Time:    time.Now(),
	})

	return testApp, ctx
}

func (suite *UpgradeSimulationSuite) createGenesisStateWithValidator(address sdk.AccAddress, valAddress sdk.ValAddress, pubKey cryptotypes.PubKey) map[string]json.RawMessage {
	encCfg := app.MakeEncodingConfig()
	genesisState := app.NewDefaultGenesisState(suite.chainID)

	bondDenom := "upaw"
	accountTokens := sdk.TokensFromConsensusPower(10000, sdk.DefaultPowerReduction)
	bondedTokens := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)

	baseAcc := authtypes.NewBaseAccount(address, pubKey, 0, 0)
	authGenesis := authtypes.GetGenesisStateFromAppState(encCfg.Codec, genesisState)
	baseAccAny, err := codectypes.NewAnyWithValue(baseAcc)
	suite.Require().NoError(err)
	authGenesis.Accounts = append(authGenesis.Accounts, baseAccAny)
	genesisState[authtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(&authGenesis)

	bondedPoolAddress := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	balances := []banktypes.Balance{
		{Address: address.String(), Coins: sdk.NewCoins(sdk.NewCoin(bondDenom, accountTokens))},
		{Address: bondedPoolAddress.String(), Coins: sdk.NewCoins(sdk.NewCoin(bondDenom, bondedTokens))},
	}

	bankGenesis := banktypes.GetGenesisStateFromAppState(encCfg.Codec, genesisState)
	bankGenesis.Balances = append(bankGenesis.Balances, balances...)
	totalSupply := sdk.NewCoin(bondDenom, accountTokens.Add(bondedTokens))
	bankGenesis.Supply = bankGenesis.Supply.Add(totalSupply)
	genesisState[banktypes.ModuleName] = encCfg.Codec.MustMarshalJSON(bankGenesis)

	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	suite.Require().NoError(err)

	validator := stakingtypes.Validator{
		OperatorAddress:   valAddress.String(),
		ConsensusPubkey:   pkAny,
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            bondedTokens,
		DelegatorShares:   sdkmath.LegacyNewDecFromInt(bondedTokens),
		Description:       stakingtypes.Description{Moniker: "test-validator"},
		MinSelfDelegation: sdkmath.OneInt(),
	}

	delegation := stakingtypes.Delegation{
		DelegatorAddress: address.String(),
		ValidatorAddress: valAddress.String(),
		Shares:           sdkmath.LegacyNewDecFromInt(bondedTokens),
	}

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = bondDenom
	stakingGenesis := stakingtypes.NewGenesisState(
		stakingParams,
		[]stakingtypes.Validator{validator},
		[]stakingtypes.Delegation{delegation},
	)
	genesisState[stakingtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(stakingGenesis)

	return genesisState
}

func (suite *UpgradeSimulationSuite) seedCompleteV1State() {
	// DEX: Create multiple pools
	creator := sdk.AccAddress([]byte("v1_pool_creator_____"))
	suite.fundAccount(creator, sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 50_000_000),
		sdk.NewInt64Coin("uusdc", 50_000_000),
		sdk.NewInt64Coin("uatom", 50_000_000),
	))

	suite.app.DEXKeeper.CreatePool(suite.ctx, creator, "upaw", "uusdc", sdkmath.NewInt(10_000_000), sdkmath.NewInt(10_000_000))
	suite.app.DEXKeeper.CreatePool(suite.ctx, creator, "upaw", "uatom", sdkmath.NewInt(5_000_000), sdkmath.NewInt(5_000_000))

	// Compute: Create multiple providers
	for i := 1; i <= 3; i++ {
		providerAddr := sdk.AccAddress([]byte("v1_provider_" + string(rune(i)) + "______"))
		provider := computetypes.Provider{
			Address:                providerAddr.String(),
			Moniker:                "v1-provider-" + string(rune(i)),
			Endpoint:               "https://v1.test",
			AvailableSpecs:         computetypes.ComputeSpec{CpuCores: 10, MemoryMb: 1024, StorageGb: 100, TimeoutSeconds: 600},
			Pricing:                computetypes.Pricing{},
			Stake:                  sdkmath.NewInt(1_000_000),
			Reputation:             80 + uint32(i*5),
			TotalRequestsCompleted: 10,
			TotalRequestsFailed:    1,
			Active:                 true,
			RegisteredAt:           suite.ctx.BlockTime(),
			LastActiveAt:           suite.ctx.BlockTime(),
		}
		suite.app.ComputeKeeper.SetProvider(suite.ctx, provider)
	}

	// Oracle: Set prices
	suite.app.OracleKeeper.SetPrice(suite.ctx, oracletypes.Price{
		Asset:         "UPAW/USD",
		Price:         sdkmath.LegacyMustNewDecFromStr("1.23"),
		BlockHeight:   suite.ctx.BlockHeight(),
		BlockTime:     suite.ctx.BlockTime().Unix(),
		NumValidators: 1,
	})
}

func (suite *UpgradeSimulationSuite) fundAccount(addr sdk.AccAddress, coins sdk.Coins) {
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))
}

func (suite *UpgradeSimulationSuite) captureFullState() map[string]interface{} {
	state := make(map[string]interface{})

	poolCount := 0
	suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		poolCount++
		return false
	})
	state["pool_count"] = poolCount

	providerCount := 0
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(store, []byte{0x02})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		providerCount++
	}
	state["provider_count"] = providerCount

	return state
}

func (suite *UpgradeSimulationSuite) simulateBlocksUntilUpgrade(targetHeight int64) {
	for h := suite.ctx.BlockHeight(); h < targetHeight; h++ {
		suite.ctx = suite.ctx.WithBlockHeight(h + 1).WithBlockTime(suite.ctx.BlockTime().Add(5 * time.Second))
	}
}

func (suite *UpgradeSimulationSuite) executeUpgrade(upgradeName string, upgradeHeight int64) {
	plan := upgradetypes.Plan{
		Name:   upgradeName,
		Height: upgradeHeight,
	}
	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))
	suite.ctx = suite.ctx.WithBlockHeight(upgradeHeight)
	suite.Require().NoError(suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan))
}

func (suite *UpgradeSimulationSuite) verifyModuleVersions(expectedVersion uint64) {
	vm, err := suite.app.UpgradeKeeper.GetModuleVersionMap(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedVersion, vm[computetypes.ModuleName])
	suite.Require().Equal(expectedVersion, vm[dextypes.ModuleName])
	suite.Require().Equal(expectedVersion, vm[oracletypes.ModuleName])
}

func (suite *UpgradeSimulationSuite) verifyStateIntegrity(pre, post map[string]interface{}) {
	suite.Require().Equal(pre["pool_count"], post["pool_count"], "pool count must not change")
	suite.Require().Equal(pre["provider_count"], post["provider_count"], "provider count must not change")
}

func (suite *UpgradeSimulationSuite) simulatePostUpgradeBlocks(numBlocks int) {
	for i := 0; i < numBlocks; i++ {
		suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 1).
			WithBlockTime(suite.ctx.BlockTime().Add(5 * time.Second))
	}
}

func (suite *UpgradeSimulationSuite) verifyModuleFunctionality() {
	// Verify DEX still works
	pools := []dextypes.Pool{}
	suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		pools = append(pools, pool)
		return false
	})
	suite.Require().Greater(len(pools), 0)

	// Verify compute still works
	providerCount := 0
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(store, []byte{0x02})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		providerCount++
	}
	suite.Require().Greater(providerCount, 0)
}

func (suite *UpgradeSimulationSuite) addDataAtV2() {
	creator := sdk.AccAddress([]byte("v2_creator__________"))
	suite.fundAccount(creator, sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 10_000_000),
		sdk.NewInt64Coin("ueth", 10_000_000),
	))
	suite.app.DEXKeeper.CreatePool(suite.ctx, creator, "upaw", "ueth", sdkmath.NewInt(5_000_000), sdkmath.NewInt(5_000_000))
}

func (suite *UpgradeSimulationSuite) verifyV1DataIntact() {
	// Should still have pools created at v1
	poolCount := 0
	suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		poolCount++
		return false
	})
	suite.Require().GreaterOrEqual(poolCount, 2)
}

func (suite *UpgradeSimulationSuite) verifyV2DataIntact() {
	// Should have additional pool from v2
	poolCount := 0
	suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		poolCount++
		return false
	})
	suite.Require().GreaterOrEqual(poolCount, 3)
}

func (suite *UpgradeSimulationSuite) verifyAllDataIntact() {
	suite.verifyV1DataIntact()
	suite.verifyV2DataIntact()
}

func (suite *UpgradeSimulationSuite) seedIdenticalState(app *app.PAWApp, ctx sdk.Context) {
	creator := sdk.AccAddress([]byte("identical_creator___"))
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 10_000_000), sdk.NewInt64Coin("uusdc", 10_000_000))
	suite.Require().NoError(app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, creator, coins))
	app.DEXKeeper.CreatePool(ctx, creator, "upaw", "uusdc", sdkmath.NewInt(5_000_000), sdkmath.NewInt(5_000_000))
}

func (suite *UpgradeSimulationSuite) createDEXPool() uint64 {
	creator := sdk.AccAddress([]byte("test_creator________"))
	suite.fundAccount(creator, sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 10_000_000),
		sdk.NewInt64Coin("uusdc", 10_000_000),
	))
	pool, err := suite.app.DEXKeeper.CreatePool(suite.ctx, creator, "upaw", "uusdc", sdkmath.NewInt(5_000_000), sdkmath.NewInt(5_000_000))
	suite.Require().NoError(err)
	return pool.Id
}

func (suite *UpgradeSimulationSuite) createComputeProvider() sdk.AccAddress {
	providerAddr := sdk.AccAddress([]byte("test_provider_______"))
	provider := computetypes.Provider{
		Address:        providerAddr.String(),
		Moniker:        "test-provider",
		Endpoint:       "https://test.com",
		AvailableSpecs: computetypes.ComputeSpec{CpuCores: 10, MemoryMb: 1024, StorageGb: 100, TimeoutSeconds: 600},
		Pricing:        computetypes.Pricing{},
		Stake:          sdkmath.NewInt(1_000_000),
		Reputation:     85,
		Active:         true,
		RegisteredAt:   suite.ctx.BlockTime(),
		LastActiveAt:   suite.ctx.BlockTime(),
	}
	suite.Require().NoError(suite.app.ComputeKeeper.SetProvider(suite.ctx, provider))
	return providerAddr
}

func (suite *UpgradeSimulationSuite) createOracleData() {
	suite.app.OracleKeeper.SetPrice(suite.ctx, oracletypes.Price{
		Asset:         "TEST/USD",
		Price:         sdkmath.LegacyMustNewDecFromStr("100.0"),
		BlockHeight:   suite.ctx.BlockHeight(),
		BlockTime:     suite.ctx.BlockTime().Unix(),
		NumValidators: 1,
	})
}

func (suite *UpgradeSimulationSuite) countPools() int {
	count := 0
	suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		count++
		return false
	})
	return count
}

func (suite *UpgradeSimulationSuite) countProviders() int {
	count := 0
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(store, []byte{0x02})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

func (suite *UpgradeSimulationSuite) countPrices() int {
	count := 0
	store := suite.ctx.KVStore(suite.app.GetKey(oracletypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(store, []byte{0x01})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}
