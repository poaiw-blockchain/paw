package keeper

import (
	"context"
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

// mockBankKeeper is a simple mock implementation for testing
type mockBankKeeper struct {
	balances map[string]sdk.Coins
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	// Simple implementation - just track balances
	fromKey := fromAddr.String()
	toKey := toAddr.String()

	// Initialize if needed
	if m.balances[fromKey] == nil {
		m.balances[fromKey] = sdk.NewCoins()
	}
	if m.balances[toKey] == nil {
		m.balances[toKey] = sdk.NewCoins()
	}

	// Add coins to recipient
	m.balances[toKey] = m.balances[toKey].Add(amt...)

	return nil
}

func (m *mockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	key := addr.String()
	if m.balances[key] == nil {
		return sdk.NewCoin(denom, math.ZeroInt())
	}
	return sdk.NewCoin(denom, m.balances[key].AmountOf(denom))
}

func (m *mockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	key := addr.String()
	if m.balances[key] == nil {
		return sdk.NewCoins()
	}
	return m.balances[key]
}

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

	// Create a mock bank keeper
	mockBankKeeper := &mockBankKeeper{
		balances: make(map[string]sdk.Coins),
	}

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		mockBankKeeper,
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
