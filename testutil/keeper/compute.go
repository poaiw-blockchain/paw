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
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// ComputeKeeper creates a test keeper for the Compute module with mock dependencies
func ComputeKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		nil, // TODO: Add mock bank keeper
		authority.String(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize module genesis
	k.InitGenesis(ctx, *types.DefaultGenesis())

	return k, ctx
}

// RegisterTestProvider registers a test compute provider
func RegisterTestProvider(t testing.TB, k *keeper.Keeper, ctx sdk.Context, address, endpoint string, stake math.Int) {
	err := keeper.RegisterTestProvider(k, ctx, address, endpoint, stake)
	require.NoError(t, err)
}

// SubmitTestRequest submits a test compute request
func SubmitTestRequest(t testing.TB, k *keeper.Keeper, ctx sdk.Context, requester, apiUrl string) uint64 {
	requestId, err := keeper.SubmitTestRequest(k, ctx, requester, apiUrl, math.NewInt(1000))
	require.NoError(t, err)
	return requestId
}
