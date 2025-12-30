package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestGetStore_WithSDKContext verifies getStore works with standard sdk.Context
func TestGetStore_WithSDKContext(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set and get params to verify store access works
	// This implicitly tests getStore() works correctly with sdk.Context
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	retrieved, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params.SwapFee, retrieved.SwapFee)
}

// TestGetStore_ViaPoolOperations verifies store access works via pool operations
func TestGetStore_ViaPoolOperations(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// GetPool should fail gracefully when pool doesn't exist
	_, err := k.GetPool(ctx, 1)
	require.Error(t, err)

	// After creating a pool, we should be able to retrieve it
	// This tests the full store read/write cycle through getStore()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", types.DefaultParams().MinLiquidity, types.DefaultParams().MinLiquidity)
	require.NoError(t, err)

	retrieved, err := k.GetPool(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, pool.Id, retrieved.Id)
	require.Equal(t, pool.TokenA, retrieved.TokenA)
	require.Equal(t, pool.TokenB, retrieved.TokenB)
}

// TestGetStoreKey verifies the store key getter for testing purposes
func TestGetStoreKey(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	storeKey := k.GetStoreKey()
	require.NotNil(t, storeKey)
	require.Equal(t, types.StoreKey, storeKey.Name())
}

// TestGetAuthority verifies the authority getter for testing purposes
func TestGetAuthority(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	authority := k.GetAuthority()
	require.NotEmpty(t, authority)
}
