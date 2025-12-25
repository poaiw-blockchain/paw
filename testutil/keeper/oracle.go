package keeper

import (
	"fmt"
	"sync"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdkstd "github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// lastStakingKeeper is a thread-safe storage for the most recently created
// staking keeper. Tests that don't use parallel subtests can use EnsureBondedValidator
// without passing the keeper explicitly. For parallel tests, use EnsureBondedValidatorWithKeeper.
var lastStakingKeeper struct {
	mu     sync.Mutex
	keeper *stakingkeeper.Keeper
}

// EnsureBondedValidatorWithKeeper seeds a bonded validator into the given staking keeper.
// This is the thread-safe version for parallel tests.
func EnsureBondedValidatorWithKeeper(ctx sdk.Context, sk *stakingkeeper.Keeper, valAddr sdk.ValAddress) error {
	if sk == nil {
		return fmt.Errorf("staking keeper not initialized")
	}

	if val, err := sk.GetValidator(ctx, valAddr); err == nil && val.IsBonded() {
		return nil
	}

	pubKey := ed25519.GenPrivKey().PubKey()
	validatorObj, err := stakingtypes.NewValidator(valAddr.String(), pubKey, stakingtypes.Description{Moniker: "oracle-gas"})
	if err != nil {
		return err
	}
	validatorObj.Status = stakingtypes.Bonded
	validatorObj.Tokens = math.NewInt(1_000_000)
	validatorObj.DelegatorShares = math.LegacyNewDecFromInt(validatorObj.Tokens)

	if err := sk.SetValidator(ctx, validatorObj); err != nil {
		return err
	}

	// Set validator by consensus address so JailValidator can find it
	if err := sk.SetValidatorByConsAddr(ctx, validatorObj); err != nil {
		return err
	}

	return sk.SetNewValidatorByPowerIndex(ctx, validatorObj)
}

// EnsureBondedValidator seeds a bonded validator into the oracle staking keeper for tests.
// Deprecated: Use EnsureBondedValidatorWithKeeper for parallel tests.
func EnsureBondedValidator(ctx sdk.Context, valAddr sdk.ValAddress) error {
	lastStakingKeeper.mu.Lock()
	sk := lastStakingKeeper.keeper
	lastStakingKeeper.mu.Unlock()
	return EnsureBondedValidatorWithKeeper(ctx, sk, valAddr)
}

// OracleKeeper creates a test keeper for the Oracle module with mock dependencies.
// Returns the oracle keeper, staking keeper (for EnsureBondedValidatorWithKeeper), and context.
func OracleKeeper(t testing.TB) (*keeper.Keeper, *stakingkeeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	stakingStoreKey := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	slashingStoreKey := storetypes.NewKVStoreKey(slashingtypes.StoreKey)
	capStoreKey := storetypes.NewKVStoreKey(capabilitytypes.StoreKey)
	capMemStoreKey := storetypes.NewMemoryStoreKey(capabilitytypes.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(stakingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(slashingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capMemStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	sdkstd.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create account keeper (required by bank keeper)
	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		types.ModuleName:               nil,
	}

	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		maccPerms,
		address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authority.String(),
	)

	// Create bank keeper
	blockedAddrs := map[string]bool{
		authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String():    true,
		authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String(): true,
	}

	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(bankStoreKey),
		accountKeeper,
		blockedAddrs,
		authority.String(),
		log.NewNopLogger(),
	)

	// Create staking keeper
	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(stakingStoreKey),
		accountKeeper,
		bankKeeper,
		authority.String(),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	// Create slashing keeper
	slashingKeeper := slashingkeeper.NewKeeper(
		cdc,
		codec.NewLegacyAmino(),
		runtime.NewKVStoreService(slashingStoreKey),
		stakingKeeper,
		authority.String(),
	)

	capKeeper := capabilitykeeper.NewKeeper(cdc, capStoreKey, capMemStoreKey)
	scopedOracleKeeper := capKeeper.ScopeToModule(types.ModuleName)
	scopedPortKeeper := capKeeper.ScopeToModule(porttypes.SubModuleName)
	portKeeper := portkeeper.NewKeeper(scopedPortKeeper)
	// Store for backward compatibility with non-parallel tests using EnsureBondedValidator
	lastStakingKeeper.mu.Lock()
	lastStakingKeeper.keeper = stakingKeeper
	lastStakingKeeper.mu.Unlock()

	var ibcKeeper *ibckeeper.Keeper

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		bankKeeper,
		stakingKeeper,
		slashingKeeper,
		ibcKeeper,
		&portKeeper,
		authority.String(),
		scopedOracleKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, stakingKeeper.SetParams(ctx, stakingtypes.DefaultParams()))
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	return k, stakingKeeper, ctx
}

// RegisterTestOracleWithKeeper registers a test oracle validator using the given staking keeper.
// This is the thread-safe version for parallel tests.
func RegisterTestOracleWithKeeper(t testing.TB, sk *stakingkeeper.Keeper, ctx sdk.Context, validator string) {
	valAddr, err := sdk.ValAddressFromBech32(validator)
	require.NoError(t, err)

	require.NotNil(t, sk)

	pubKey := ed25519.GenPrivKey().PubKey()
	validatorObj, err := stakingtypes.NewValidator(valAddr.String(), pubKey, stakingtypes.Description{})
	require.NoError(t, err)
	validatorObj.Status = stakingtypes.Bonded
	validatorObj.Tokens = math.NewInt(1_000_000)
	validatorObj.DelegatorShares = math.LegacyNewDecFromInt(validatorObj.Tokens)

	require.NoError(t, sk.SetValidator(ctx, validatorObj))
	require.NoError(t, sk.SetNewValidatorByPowerIndex(ctx, validatorObj))
}

// RegisterTestOracle registers a test oracle validator.
// Deprecated: Use RegisterTestOracleWithKeeper for parallel tests.
func RegisterTestOracle(t testing.TB, k *keeper.Keeper, ctx sdk.Context, validator string) {
	lastStakingKeeper.mu.Lock()
	sk := lastStakingKeeper.keeper
	lastStakingKeeper.mu.Unlock()
	RegisterTestOracleWithKeeper(t, sk, ctx, validator)
}

// SubmitTestPrice submits a test price feed
func SubmitTestPrice(t testing.TB, k *keeper.Keeper, ctx sdk.Context, oracle, asset string, price math.LegacyDec) {
	// Convert validator address to account address for feeder
	valAddr, err := sdk.ValAddressFromBech32(oracle)
	require.NoError(t, err)
	feederAddr := sdk.AccAddress(valAddr)

	msg := &types.MsgSubmitPrice{
		Validator: oracle,
		Feeder:    feederAddr.String(),
		Asset:     asset,
		Price:     price,
	}
	_, err = keeper.NewMsgServerImpl(*k).SubmitPrice(ctx, msg)
	require.NoError(t, err)
}
