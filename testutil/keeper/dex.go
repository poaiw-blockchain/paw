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
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// DexKeeper creates a test keeper for the DEX module with mock dependencies
func DexKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		nil, // TODO: Add mock bank keeper
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize module genesis
	k.InitGenesis(ctx, *types.DefaultGenesis())

	return k, ctx
}

// CreateTestPool creates a test liquidity pool with given tokens
// TODO: Implement when message handlers are ready
func CreateTestPool(t testing.TB, k keeper.Keeper, ctx sdk.Context, tokenA, tokenB string, amountA, amountB math.Int) uint64 {
	// msgCreate := &types.MsgCreatePool{
	// 	Creator: "paw1test",
	// 	TokenA:  tokenA,
	// 	TokenB:  tokenB,
	// 	AmountA: amountA,
	// 	AmountB: amountB,
	// }
	// resp, err := k.CreatePool(ctx, msgCreate)
	// require.NoError(t, err)
	// require.NotNil(t, resp)
	// return resp.PoolId
	t.Skip("CreatePool message handler not implemented yet")
	return 0
}
