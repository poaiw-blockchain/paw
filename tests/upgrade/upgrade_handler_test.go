package upgrade_test

import (
	"context"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	testkeeper "github.com/paw-chain/paw/testutil/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// UpgradeTestSuite tests blockchain upgrade mechanisms
type UpgradeTestSuite struct {
	suite.Suite

	app *app.PAWApp
	ctx sdk.Context
}

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (suite *UpgradeTestSuite) SetupTest() {
	suite.app, suite.ctx = testkeeper.SetupTestApp(suite.T())
}

// TestUpgradeFromV1ToV2 validates that module migrations repair legacy state and bump versions.
func (suite *UpgradeTestSuite) TestUpgradeFromV1ToV2() {
	poolID := suite.seedLegacyDexState()
	providerAddr := suite.seedLegacyComputeProvider()

	legacyVM := suite.app.ModuleManager().GetVersionMap()
	legacyVM[dextypes.ModuleName] = 1
	legacyVM[computetypes.ModuleName] = 1
	legacyVM[oracletypes.ModuleName] = 1
	suite.Require().NoError(suite.app.UpgradeKeeper.SetModuleVersionMap(suite.ctx, legacyVM))

	plan := upgradetypes.Plan{
		Name:   "security-hardening-v2",
		Height: suite.ctx.BlockHeight() + 1,
		Info:   "exercise dex/compute/oracle migrations and repair corrupted indexes",
	}

	suite.app.UpgradeKeeper.SetUpgradeHandler(
		plan.Name,
		func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			ctx := sdk.UnwrapSDKContext(goCtx)
			return suite.app.ModuleManager().RunMigrations(ctx, suite.app.Configurator(), vm)
		},
	)

	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))
	suite.ctx = suite.ctx.WithBlockHeight(plan.Height).WithBlockTime(time.Now())

	suite.Require().NoError(suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan))

	suite.assertDexStateRecovered(poolID)
	suite.assertComputeReputationRepaired(providerAddr)
	suite.assertModuleVersionsAreCurrent()
}

// TestUpgradeRepairsComputeIndexes ensures migrations rebuild missing compute indexes idempotently.
func (suite *UpgradeTestSuite) TestUpgradeRepairsComputeIndexes() {
	request := suite.createCorruptedRequest()

	vm := suite.app.ModuleManager().GetVersionMap()
	vm[computetypes.ModuleName] = 1
	suite.Require().NoError(suite.app.UpgradeKeeper.SetModuleVersionMap(suite.ctx, vm))

	plan := upgradetypes.Plan{
		Name:   "compute-index-repair",
		Height: suite.ctx.BlockHeight() + 1,
	}

	suite.app.UpgradeKeeper.SetUpgradeHandler(
		plan.Name,
		func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			ctx := sdk.UnwrapSDKContext(goCtx)
			return suite.app.ModuleManager().RunMigrations(ctx, suite.app.Configurator(), vm)
		},
	)

	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))
	suite.ctx = suite.ctx.WithBlockHeight(plan.Height)
	suite.Require().NoError(suite.app.UpgradeKeeper.ApplyUpgrade(suite.ctx, plan))

	var seen []uint64
	err := suite.app.ComputeKeeper.IterateRequestsByStatus(suite.ctx, computetypes.REQUEST_STATUS_COMPLETED, func(req computetypes.Request) (bool, error) {
		seen = append(seen, req.Id)
		return false, nil
	})
	suite.Require().NoError(err)
	suite.Require().Contains(seen, request.Id, "migration should rebuild status index for legacy request")
}

// TestUpgradeRollback uses cache context to verify failed migrations do not mutate persisted state.
func (suite *UpgradeTestSuite) TestUpgradeRollback() {
	preState := suite.captureState()

	plan := upgradetypes.Plan{
		Name:   "faulty-upgrade",
		Height: suite.ctx.BlockHeight() + 1,
	}

	suite.app.UpgradeKeeper.SetUpgradeHandler(
		plan.Name,
		func(context.Context, upgradetypes.Plan, module.VersionMap) (module.VersionMap, error) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "simulated upgrade failure")
		},
	)

	suite.Require().NoError(suite.app.UpgradeKeeper.ScheduleUpgrade(suite.ctx, plan))

	cacheCtx, _ := suite.ctx.CacheContext()
	cacheCtx = cacheCtx.WithBlockHeight(plan.Height)
	suite.Require().Error(suite.app.UpgradeKeeper.ApplyUpgrade(cacheCtx, plan))

	postState := suite.captureState()
	suite.Require().Equal(preState, postState, "failed upgrade must not mutate committed state")
}

// Helper functions

func (suite *UpgradeTestSuite) seedLegacyDexState() uint64 {
	// Create a pool with reversed token ordering and missing indexes/circuit breaker state.
	pool := &dextypes.Pool{
		Id:          1,
		TokenA:      "uusdc",
		TokenB:      "upaw",
		ReserveA:    sdkmath.NewInt(8_000_000),
		ReserveB:    sdkmath.NewInt(2_000_000),
		TotalShares: sdkmath.NewInt(10_000_000),
	}

	suite.Require().NoError(suite.app.DEXKeeper.SetPool(suite.ctx, pool))

	// Delete token pair index and circuit breaker state to mimic legacy stores.
	store := suite.ctx.KVStore(suite.app.GetKey(dextypes.StoreKey))
	store.Delete(dexkeeper.PoolByTokensKey(pool.TokenA, pool.TokenB))
	store.Delete(dexkeeper.CircuitBreakerKey(pool.Id))

	return pool.Id
}

func (suite *UpgradeTestSuite) seedLegacyComputeProvider() sdk.AccAddress {
	addr := sdk.AccAddress([]byte("compute_provider_____"))
	provider := computetypes.Provider{
		Address:                addr.String(),
		Moniker:                "legacy-provider",
		Endpoint:               "https://legacy.compute",
		AvailableSpecs:         computetypes.ComputeSpec{CpuCores: 4, MemoryMb: 8192, StorageGb: 500, TimeoutSeconds: 3600},
		Pricing:                computetypes.Pricing{},
		Stake:                  sdkmath.NewInt(5_000_000),
		Reputation:             0, // intentionally invalid so migration recalculates
		TotalRequestsCompleted: 12,
		TotalRequestsFailed:    3,
		Active:                 true,
	}

	suite.Require().NoError(suite.app.ComputeKeeper.SetProvider(suite.ctx, provider))
	return addr
}

func (suite *UpgradeTestSuite) createCorruptedRequest() computetypes.Request {
	req := computetypes.Request{
		Id:             42,
		Requester:      sdk.AccAddress([]byte("requester_addr_____")).String(),
		Provider:       sdk.AccAddress([]byte("provider_addr_____")).String(),
		Status:         computetypes.REQUEST_STATUS_COMPLETED,
		MaxPayment:     sdkmath.NewInt(10_000),
		EscrowedAmount: sdkmath.NewInt(10_000),
	}

	suite.Require().NoError(suite.app.ComputeKeeper.SetRequest(suite.ctx, req))
	return req
}

func (suite *UpgradeTestSuite) assertDexStateRecovered(poolID uint64) {
	pool, err := suite.app.DEXKeeper.GetPool(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().LessOrEqual(pool.TokenA, pool.TokenB, "tokens should be reordered lexicographically after migration")

	_, err = suite.app.DEXKeeper.GetPoolByTokens(suite.ctx, pool.TokenA, pool.TokenB)
	suite.Require().NoError(err, "token pair index should be rebuilt during migration")

	cbState, err := suite.app.DEXKeeper.GetPoolCircuitBreakerState(suite.ctx, poolID)
	suite.Require().NoError(err)
	suite.Require().False(cbState.Enabled, "circuit breaker should be initialized but not left active")
}

func (suite *UpgradeTestSuite) assertComputeReputationRepaired(providerAddr sdk.AccAddress) {
	provider, err := suite.app.ComputeKeeper.GetProvider(suite.ctx, providerAddr)
	suite.Require().NoError(err)
	expectedReputation := uint32(80) // 12 successful out of 15 => 80
	suite.Require().Equal(expectedReputation, provider.Reputation, "migration should recalc provider reputation")
}

func (suite *UpgradeTestSuite) assertModuleVersionsAreCurrent() {
	vm, err := suite.app.UpgradeKeeper.GetModuleVersionMap(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(2), vm[dextypes.ModuleName], "dex should be migrated to v2")
	suite.Require().Equal(uint64(2), vm[computetypes.ModuleName], "compute should be migrated to v2")
	suite.Require().Equal(uint64(2), vm[oracletypes.ModuleName], "oracle should be migrated to v2")
}

func (suite *UpgradeTestSuite) captureState() map[string]interface{} {
	state := make(map[string]interface{})

	// capture pool tokens and circuit breaker state
	_ = suite.app.DEXKeeper.IteratePools(suite.ctx, func(pool dextypes.Pool) bool {
		state["pool"] = pool
		cb, _ := suite.app.DEXKeeper.GetPoolCircuitBreakerState(suite.ctx, pool.Id)
		state["pool_cb"] = cb
		return false
	})

	// capture provider state
	providers := []computetypes.Provider{}
	store := suite.ctx.KVStore(suite.app.GetKey(computetypes.StoreKey))
	iter := storetypes.KVStorePrefixIterator(store, []byte{0x02})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var provider computetypes.Provider
		suite.Require().NoError(suite.app.AppCodec().Unmarshal(iter.Value(), &provider))
		providers = append(providers, provider)
	}
	state["providers"] = providers

	return state
}
