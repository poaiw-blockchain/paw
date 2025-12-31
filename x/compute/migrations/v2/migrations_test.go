package v2_test

import (
	"encoding/binary"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	v2 "github.com/paw-chain/paw/x/compute/migrations/v2"
	"github.com/paw-chain/paw/x/compute/types"
)

// setupTest creates a minimal test environment for migration testing
func setupTest(t *testing.T) (storetypes.StoreKey, codec.BinaryCodec, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Height: 1000,
		Time:   time.Unix(1609459200, 0), // 2021-01-01
	}, false, log.NewNopLogger())

	return storeKey, cdc, ctx
}

// Helper to set request in store
func setRequest(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, request types.Request) {
	key := append(v2.RequestKeyPrefix, getRequestIDBytes(request.Id)...)
	bz, err := cdc.Marshal(&request)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to get request from store
func getRequest(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, requestID uint64) (types.Request, bool) {
	key := append(v2.RequestKeyPrefix, getRequestIDBytes(requestID)...)
	bz := store.Get(key)
	if bz == nil {
		return types.Request{}, false
	}
	var request types.Request
	err := cdc.Unmarshal(bz, &request)
	require.NoError(t, err)
	return request, true
}

// Helper to set provider in store
func setProvider(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, provider types.Provider) {
	addr, err := sdk.AccAddressFromBech32(provider.Address)
	require.NoError(t, err)
	key := append(v2.ProviderKeyPrefix, addr.Bytes()...)
	bz, err := cdc.Marshal(&provider)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to get provider from store
func getProvider(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, addr sdk.AccAddress) (types.Provider, bool) {
	key := append(v2.ProviderKeyPrefix, addr.Bytes()...)
	bz := store.Get(key)
	if bz == nil {
		return types.Provider{}, false
	}
	var provider types.Provider
	err := cdc.Unmarshal(bz, &provider)
	require.NoError(t, err)
	return provider, true
}

// Helper to set params in store
func setParams(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, params types.Params) {
	bz, err := cdc.Marshal(&params)
	require.NoError(t, err)
	store.Set(v2.ParamsKey, bz)
}

// Helper to get params from store
func getParams(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec) (types.Params, bool) {
	bz := store.Get(v2.ParamsKey)
	if bz == nil {
		return types.Params{}, false
	}
	var params types.Params
	err := cdc.Unmarshal(bz, &params)
	require.NoError(t, err)
	return params, true
}

// Helper to get request counter
func getRequestCounter(store storetypes.KVStore) uint64 {
	bz := store.Get(v2.NextRequestIDKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// Helper to set request counter
func setRequestCounter(store storetypes.KVStore, counter uint64) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, counter)
	store.Set(v2.NextRequestIDKey, bz)
}

// Helper to check if active provider index exists
func hasActiveProviderIndex(store storetypes.KVStore, addr sdk.AccAddress) bool {
	key := append(v2.ActiveProvidersPrefix, addr.Bytes()...)
	return store.Has(key)
}

// Helper to count requests
func countRequests(store storetypes.KVStore) int {
	iterator := storetypes.KVStorePrefixIterator(store, v2.RequestKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	return count
}

// Helper to set escrow state in store
func setEscrowState(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, escrow types.EscrowState) {
	key := append(v2.EscrowStateKeyPrefix, getRequestIDBytes(escrow.RequestId)...)
	bz, err := cdc.Marshal(&escrow)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to check escrow timeout index exists
func hasEscrowTimeoutIndex(store storetypes.KVStore, requestID uint64) bool {
	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)
	key := append(v2.EscrowTimeoutReversePrefix, idBz...)
	return store.Has(key)
}

func getRequestIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

func TestMigrate_EmptyState(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)

	// Run migration on empty state
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Should have default params
	store := ctx.KVStore(storeKey)
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, types.DefaultParams().MinProviderStake, params.MinProviderStake)
}

func TestMigrate_ValidRequestData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up valid request
	validRequest := types.Request{
		Id:       1,
		Provider: providerAddr.String(),
		Status:   types.REQUEST_STATUS_PENDING,
	}
	setRequest(t, store, cdc, validRequest)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Request should remain unchanged
	request, found := getRequest(t, store, cdc, 1)
	require.True(t, found)
	require.Equal(t, validRequest.Id, request.Id)
	require.Equal(t, validRequest.Provider, request.Provider)
	require.Equal(t, validRequest.Status, request.Status)
}

func TestMigrate_ValidProviderData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up valid provider
	validProvider := types.Provider{
		Address:                 providerAddr.String(),
		Active:                  true,
		Reputation:              80,
		TotalRequestsCompleted:  100,
		TotalRequestsFailed:     20,
	}
	setProvider(t, store, cdc, validProvider)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Provider should remain unchanged
	provider, found := getProvider(t, store, cdc, providerAddr)
	require.True(t, found)
	require.Equal(t, validProvider.Address, provider.Address)
	require.Equal(t, validProvider.Reputation, provider.Reputation)
	require.True(t, provider.Active)
}

func TestMigrate_FixZeroReputation(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up provider with zero reputation
	invalidProvider := types.Provider{
		Address:                 providerAddr.String(),
		Active:                  true,
		Reputation:              0,
		TotalRequestsCompleted:  50,
		TotalRequestsFailed:     10,
	}
	setProvider(t, store, cdc, invalidProvider)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Zero reputation should be fixed to 1
	provider, found := getProvider(t, store, cdc, providerAddr)
	require.True(t, found)
	require.GreaterOrEqual(t, provider.Reputation, uint32(1))
}

func TestMigrate_FixReputationAbove100(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up provider with reputation above 100
	invalidProvider := types.Provider{
		Address:                 providerAddr.String(),
		Active:                  true,
		Reputation:              150,
		TotalRequestsCompleted:  100,
		TotalRequestsFailed:     0,
	}
	setProvider(t, store, cdc, invalidProvider)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Reputation above 100 should be fixed to 100
	provider, found := getProvider(t, store, cdc, providerAddr)
	require.True(t, found)
	require.LessOrEqual(t, provider.Reputation, uint32(100))
}

func TestMigrate_RecalculateWrongReputation(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up provider with clearly wrong reputation
	// 80% success rate (80/100) but reputation is 50
	invalidProvider := types.Provider{
		Address:                 providerAddr.String(),
		Active:                  true,
		Reputation:              50, // Should be ~80
		TotalRequestsCompleted:  80,
		TotalRequestsFailed:     20,
	}
	setProvider(t, store, cdc, invalidProvider)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Reputation should be recalculated (~80)
	provider, found := getProvider(t, store, cdc, providerAddr)
	require.True(t, found)
	require.Equal(t, uint32(80), provider.Reputation)
}

func TestMigrate_RebuildProviderIndexes_Active(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	activeAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	inactiveAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up active and inactive providers
	activeProvider := types.Provider{
		Address:    activeAddr.String(),
		Active:     true,
		Reputation: 80,
	}
	inactiveProvider := types.Provider{
		Address:    inactiveAddr.String(),
		Active:     false,
		Reputation: 60,
	}
	setProvider(t, store, cdc, activeProvider)
	setProvider(t, store, cdc, inactiveProvider)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Active provider should be in active index
	require.True(t, hasActiveProviderIndex(store, activeAddr))
	// Inactive provider should not be in active index
	require.False(t, hasActiveProviderIndex(store, inactiveAddr))
}

func TestMigrate_Params_NoExistingParams(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)

	// Run migration without existing params
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Should create default params
	store := ctx.KVStore(storeKey)
	params, found := getParams(t, store, cdc)
	require.True(t, found)

	defaults := types.DefaultParams()
	require.True(t, defaults.MinProviderStake.Equal(params.MinProviderStake))
}

func TestMigrate_Params_UpdateZeroFields(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with zero new fields
	oldParams := types.Params{
		MinProviderStake:         math.ZeroInt(),
		MaxRequestTimeoutSeconds: 0,
	}
	setParams(t, store, cdc, oldParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Zero fields should be updated with defaults
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.True(t, params.MinProviderStake.Equal(math.NewInt(1000000)))
	require.Equal(t, uint64(3600), params.MaxRequestTimeoutSeconds)
}

func TestMigrate_Params_PreserveExistingValues(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with existing valid values
	existingParams := types.Params{
		MinProviderStake:         math.NewInt(5000000),
		MaxRequestTimeoutSeconds: 7200,
	}
	setParams(t, store, cdc, existingParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Existing values should be preserved
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.True(t, existingParams.MinProviderStake.Equal(params.MinProviderStake))
	require.Equal(t, existingParams.MaxRequestTimeoutSeconds, params.MaxRequestTimeoutSeconds)
}

func TestMigrate_ValidateRequestCounter(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up requests
	req1 := types.Request{Id: 5, Provider: providerAddr.String(), Status: types.REQUEST_STATUS_PENDING}
	req2 := types.Request{Id: 10, Provider: providerAddr.String(), Status: types.REQUEST_STATUS_COMPLETED}
	setRequest(t, store, cdc, req1)
	setRequest(t, store, cdc, req2)

	// Set counter too low
	setRequestCounter(store, 5)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Counter should be updated to maxID + 1
	counter := getRequestCounter(store)
	require.Equal(t, uint64(11), counter)
}

func TestMigrate_RequestCounterAlreadyCorrect(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up request
	req := types.Request{Id: 5, Provider: providerAddr.String(), Status: types.REQUEST_STATUS_PENDING}
	setRequest(t, store, cdc, req)

	// Set counter correctly
	setRequestCounter(store, 100)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Counter should remain unchanged
	counter := getRequestCounter(store)
	require.Equal(t, uint64(100), counter)
}

func TestMigrate_EscrowTimeoutIndexes_LockedEscrow(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up LOCKED escrow
	escrow := types.EscrowState{
		RequestId: 1,
		Status:    types.ESCROW_STATUS_LOCKED,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	setEscrowState(t, store, cdc, escrow)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// LOCKED escrow should have timeout index
	require.True(t, hasEscrowTimeoutIndex(store, 1))
}

func TestMigrate_EscrowTimeoutIndexes_ChallengedEscrow(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up CHALLENGED escrow
	escrow := types.EscrowState{
		RequestId: 2,
		Status:    types.ESCROW_STATUS_CHALLENGED,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	setEscrowState(t, store, cdc, escrow)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// CHALLENGED escrow should have timeout index
	require.True(t, hasEscrowTimeoutIndex(store, 2))
}

func TestMigrate_EscrowTimeoutIndexes_ReleasedEscrow(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up RELEASED escrow
	escrow := types.EscrowState{
		RequestId: 3,
		Status:    types.ESCROW_STATUS_RELEASED,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	setEscrowState(t, store, cdc, escrow)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// RELEASED escrow should NOT have timeout index
	require.False(t, hasEscrowTimeoutIndex(store, 3))
}

func TestMigrate_EscrowTimeoutIndexes_RefundedEscrow(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up REFUNDED escrow
	escrow := types.EscrowState{
		RequestId: 4,
		Status:    types.ESCROW_STATUS_REFUNDED,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	setEscrowState(t, store, cdc, escrow)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// REFUNDED escrow should NOT have timeout index
	require.False(t, hasEscrowTimeoutIndex(store, 4))
}

func TestMigrate_MultipleProviders(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create multiple providers with various states
	for i := 0; i < 10; i++ {
		addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
		provider := types.Provider{
			Address:                 addr.String(),
			Active:                  i%2 == 0,
			Reputation:              uint32(50 + i*5),
			TotalRequestsCompleted:  uint64(10 + i),
			TotalRequestsFailed:     uint64(i),
		}
		setProvider(t, store, cdc, provider)
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Count active provider indexes
	activeCount := 0
	iterator := storetypes.KVStorePrefixIterator(store, v2.ActiveProvidersPrefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		activeCount++
	}

	// Should have 5 active providers (i=0,2,4,6,8)
	require.Equal(t, 5, activeCount)
}

func TestMigrate_Idempotency(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set up provider
	provider := types.Provider{
		Address:                 providerAddr.String(),
		Active:                  true,
		Reputation:              80,
		TotalRequestsCompleted:  100,
		TotalRequestsFailed:     20,
	}
	setProvider(t, store, cdc, provider)

	// Set up request
	request := types.Request{
		Id:       1,
		Provider: providerAddr.String(),
		Status:   types.REQUEST_STATUS_PENDING,
	}
	setRequest(t, store, cdc, request)

	// Run migration first time
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Get state after first migration
	provider1, _ := getProvider(t, store, cdc, providerAddr)
	params1, _ := getParams(t, store, cdc)
	counter1 := getRequestCounter(store)

	// Run migration second time
	err = v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// State should be identical
	provider2, _ := getProvider(t, store, cdc, providerAddr)
	params2, _ := getParams(t, store, cdc)
	counter2 := getRequestCounter(store)

	require.Equal(t, provider1, provider2)
	require.Equal(t, params1, params2)
	require.Equal(t, counter1, counter2)
}

func TestMigrate_CorruptedRequestData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted request data
	key := append(v2.RequestKeyPrefix, getRequestIDBytes(1)...)
	store.Set(key, []byte("invalid request data"))

	// Run migration - should error on corrupted data
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal request")
}

func TestMigrate_CorruptedProviderData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Set corrupted provider data
	key := append(v2.ProviderKeyPrefix, addr.Bytes()...)
	store.Set(key, []byte("invalid provider data"))

	// Run migration - should error on corrupted data
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal provider")
}

func TestMigrate_CorruptedParams(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted params data
	store.Set(v2.ParamsKey, []byte("invalid params data"))

	// Run migration - should error on corrupted params
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to migrate params")
}

func TestMigrate_LargeDataset(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Create many requests
	for i := uint64(1); i <= 100; i++ {
		request := types.Request{
			Id:       i,
			Provider: providerAddr.String(),
			Status:   types.REQUEST_STATUS_PENDING,
		}
		setRequest(t, store, cdc, request)
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// All requests should exist
	require.Equal(t, 100, countRequests(store))

	// Counter should be correct
	counter := getRequestCounter(store)
	require.Equal(t, uint64(101), counter)
}

func TestMigrate_InvalidProviderAddress_Request(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up request with invalid provider address
	request := types.Request{
		Id:       1,
		Provider: "invalid-address",
		Status:   types.REQUEST_STATUS_PENDING,
	}
	setRequest(t, store, cdc, request)

	// Run migration - should not error, just skip invalid address
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Request should still exist
	_, found := getRequest(t, store, cdc, 1)
	require.True(t, found)
}

func TestMigrate_MixedEscrowStates(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create escrows with different statuses
	escrows := []types.EscrowState{
		{RequestId: 1, Status: types.ESCROW_STATUS_LOCKED, ExpiresAt: time.Now().Add(time.Hour)},
		{RequestId: 2, Status: types.ESCROW_STATUS_CHALLENGED, ExpiresAt: time.Now().Add(time.Hour)},
		{RequestId: 3, Status: types.ESCROW_STATUS_RELEASED, ExpiresAt: time.Now().Add(time.Hour)},
		{RequestId: 4, Status: types.ESCROW_STATUS_REFUNDED, ExpiresAt: time.Now().Add(time.Hour)},
	}
	for _, e := range escrows {
		setEscrowState(t, store, cdc, e)
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Only LOCKED and CHALLENGED should have timeout indexes
	require.True(t, hasEscrowTimeoutIndex(store, 1))
	require.True(t, hasEscrowTimeoutIndex(store, 2))
	require.False(t, hasEscrowTimeoutIndex(store, 3))
	require.False(t, hasEscrowTimeoutIndex(store, 4))
}
