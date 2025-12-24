package keeper

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
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
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// setupKeeperForTest creates a test keeper for internal package tests.
// This mirrors the testutil/keeper/compute.go setup for use in _test.go files
// within the keeper package that need to test unexported functions.
func setupKeeperForTest(t *testing.T) (*Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	stakingStoreKey := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	slashingStoreKey := storetypes.NewKVStoreKey(slashingtypes.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	capStoreKey := storetypes.NewKVStoreKey(capabilitytypes.StoreKey)
	capMemStoreKey := storetypes.NewMemoryStoreKey(capabilitytypes.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(stakingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(slashingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capMemStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	banktypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		types.ModuleName:               {authtypes.Minter, authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		maccPerms,
		address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authority.String(),
	)

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

	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(stakingStoreKey),
		accountKeeper,
		bankKeeper,
		authority.String(),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		address.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	slashingKeeper := slashingkeeper.NewKeeper(
		cdc,
		codec.NewLegacyAmino(),
		runtime.NewKVStoreService(slashingStoreKey),
		stakingKeeper,
		authority.String(),
	)

	capKeeper := capabilitykeeper.NewKeeper(cdc, capStoreKey, capMemStoreKey)
	scopedKeeper := capKeeper.ScopeToModule(types.ModuleName)
	portKeeper := portkeeper.NewKeeper(scopedKeeper)
	capKeeper.Seal()

	k := NewKeeper(
		cdc,
		storeKey,
		bankKeeper,
		accountKeeper,
		stakingKeeper,
		slashingKeeper,
		nil,
		&portKeeper,
		authority.String(),
		scopedKeeper,
	)

	header := cmtproto.Header{
		Time: time.Now().UTC(),
	}
	sdkCtx := sdk.NewContext(stateStore, header, false, log.NewNopLogger())
	sdkCtx = sdkCtx.WithContext(context.Background())
	sdkCtx = sdkCtx.WithBlockTime(time.Now().UTC())

	moduleAccount := accountKeeper.NewAccount(sdkCtx, authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)).(*authtypes.ModuleAccount)
	accountKeeper.SetModuleAccount(sdkCtx, moduleAccount)

	// Fund common test accounts
	fundCoins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1_000_000_000))
	fundAddrs := []sdk.AccAddress{
		sdk.AccAddress([]byte("test_provider_addr_")),
		sdk.AccAddress([]byte("test_requester_addr")),
		sdk.AccAddress([]byte("other_user_address_")),
		sdk.AccAddress([]byte("test_address")),
		sdk.AccAddress([]byte("validator1")),
		sdk.AccAddress([]byte("random_address")),
	}
	for _, addr := range fundAddrs {
		if len(addr) > 0 {
			require.NoError(t, bankKeeper.MintCoins(sdkCtx, types.ModuleName, fundCoins))
			require.NoError(t, bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, addr, fundCoins))
		}
	}

	ctx := sdkCtx
	return k, ctx
}
