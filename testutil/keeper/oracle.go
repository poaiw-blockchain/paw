package keeper

import (
	"testing"

	"cosmossdk.io/log"
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

	"github.com/paw/x/oracle/keeper"
	"github.com/paw/x/oracle/types"
)

// OracleKeeper creates a test keeper for the Oracle module with mock dependencies
func OracleKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
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
		log.NewNopLogger(),
		authority.String(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize module genesis
	require.NoError(t, k.InitGenesis(ctx, types.DefaultGenesis()))

	return k, ctx
}

// RegisterTestOracle registers a test oracle validator
func RegisterTestOracle(t testing.TB, k keeper.Keeper, ctx sdk.Context, validator string) {
	msgRegister := &types.MsgRegisterOracle{
		Validator: validator,
	}

	_, err := k.RegisterOracle(ctx, msgRegister)
	require.NoError(t, err)
}

// SubmitTestPrice submits a test price feed
func SubmitTestPrice(t testing.TB, k keeper.Keeper, ctx sdk.Context, oracle, asset string, price sdk.Dec) {
	msgSubmit := &types.MsgSubmitPrice{
		Oracle: oracle,
		Asset:  asset,
		Price:  price,
	}

	_, err := k.SubmitPrice(ctx, msgSubmit)
	require.NoError(t, err)
}
