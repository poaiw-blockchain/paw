package keeper

import (
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
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// DexKeeper creates a test keeper for the DEX module with mock dependencies
func DexKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	k, _, ctx := buildDexKeeper(t)
	return k, ctx
}

// DexKeeperWithBank returns the dex keeper and bank keeper for tests needing explicit funding.
func DexKeeperWithBank(t testing.TB) (*keeper.Keeper, bankkeeper.Keeper, sdk.Context) {
	return buildDexKeeper(t)
}

func buildDexKeeper(t testing.TB) (*keeper.Keeper, bankkeeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	capStoreKey := storetypes.NewKVStoreKey(capabilitytypes.StoreKey)
	capMemStoreKey := storetypes.NewMemoryStoreKey(capabilitytypes.MemStoreKey)

	db := dbm.NewMemDB()
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(capMemStoreKey, storetypes.StoreTypeMemory, nil)
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	banktypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create account keeper (required by bank keeper)
	maccPerms := map[string][]string{
		types.ModuleName: {authtypes.Minter, authtypes.Burner},
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

	// Create bank keeper
	blockedAddrs := map[string]bool{}

	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(bankStoreKey),
		accountKeeper,
		blockedAddrs,
		authority.String(),
		log.NewNopLogger(),
	)

	capKeeper := capabilitykeeper.NewKeeper(cdc, capStoreKey, capMemStoreKey)
	scopedDexKeeper := capKeeper.ScopeToModule(types.ModuleName)
	scopedPortKeeper := capKeeper.ScopeToModule(porttypes.SubModuleName)
	portKeeper := portkeeper.NewKeeper(scopedPortKeeper)
	var ibcKeeper *ibckeeper.Keeper

	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, ibcKeeper, &portKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(), scopedDexKeeper)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Setup module account for mint/burn
	moduleAccount := accountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)).(*authtypes.ModuleAccount)
	accountKeeper.SetModuleAccount(ctx, moduleAccount)

	// Prefund common test accounts with ample liquidity across all denoms used in tests
	mintCoins := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 5_000_000_000),
		sdk.NewInt64Coin("uusdt", 5_000_000_000),
		sdk.NewInt64Coin("uatom", 5_000_000_000),
		sdk.NewInt64Coin("atom", 5_000_000_000),
		sdk.NewInt64Coin("usdc", 5_000_000_000),
		sdk.NewInt64Coin("osmo", 5_000_000_000),
		sdk.NewInt64Coin("tokenA", 5_000_000_000),
		sdk.NewInt64Coin("tokenB", 5_000_000_000),
		sdk.NewInt64Coin("tokenC", 5_000_000_000),
		sdk.NewInt64Coin("tokenD", 5_000_000_000),
		sdk.NewInt64Coin("tokenE", 5_000_000_000),
	)

	fundAddrs := []sdk.AccAddress{
		types.TestAddr(),
		sdk.AccAddress([]byte("test_trader_address")),
		sdk.AccAddress([]byte("creator1___________")),
		sdk.AccAddress([]byte("trader1____________")),
		sdk.AccAddress([]byte("trader2____________")),
		sdk.AccAddress([]byte("trader3____________")),
		sdk.AccAddress([]byte("provider1__________")),
		sdk.AccAddress([]byte("attacker_address___")),
		sdk.AccAddress([]byte("normal_user_address")),
		sdk.AccAddress([]byte("liquidity_provider_")),
		sdk.AccAddress([]byte("regular_user_1_____")),
		sdk.AccAddress([]byte("regular_user_2_____")),
		sdk.AccAddress([]byte("regular_user_3_____")),
	}
	for i := 0; i < 10; i++ {
		addr := make([]byte, 20)
		copy(addr, []byte("test_trader_"))
		addr[19] = byte(i)
		fundAddrs = append(fundAddrs, sdk.AccAddress(addr))
	}

	for _, addr := range fundAddrs {
		require.NoError(t, bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins))
		require.NoError(t, bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, mintCoins))
	}

	// initialize params and defaults
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	k.SetNextPoolId(ctx, 1)

	return k, bankKeeper, ctx
}

// CreateTestPool creates a test liquidity pool with given tokens
func CreateTestPool(t testing.TB, k *keeper.Keeper, ctx sdk.Context, tokenA, tokenB string, amountA, amountB math.Int) uint64 {
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	require.NotNil(t, pool)
	require.Greater(t, pool.Id, uint64(0))
	return pool.Id
}

// FundAccount mints tokens and sends them to an account for testing purposes
func FundAccount(t testing.TB, k *keeper.Keeper, ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) {
	t.Helper()
	bk := k.BankKeeper()
	require.NoError(t, bk.MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins))
}
