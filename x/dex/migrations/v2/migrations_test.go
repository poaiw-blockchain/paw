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

	v2 "github.com/paw-chain/paw/x/dex/migrations/v2"
	"github.com/paw-chain/paw/x/dex/types"
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

// Helper to set pool in store
func setPool(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, pool types.Pool) {
	key := append(v2.PoolKeyPrefix, getPoolIDBytes(pool.Id)...)
	bz, err := cdc.Marshal(&pool)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to get pool from store
func getPool(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, poolID uint64) (types.Pool, bool) {
	key := append(v2.PoolKeyPrefix, getPoolIDBytes(poolID)...)
	bz := store.Get(key)
	if bz == nil {
		return types.Pool{}, false
	}
	var pool types.Pool
	err := cdc.Unmarshal(bz, &pool)
	require.NoError(t, err)
	return pool, true
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

// Helper to get pool counter
func getPoolCounter(store storetypes.KVStore) uint64 {
	bz := store.Get(v2.PoolCounterKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// Helper to set pool counter
func setPoolCounter(store storetypes.KVStore, counter uint64) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, counter)
	store.Set(v2.PoolCounterKey, bz)
}

// Helper to get pool by tokens index
func getPoolByTokensIndex(store storetypes.KVStore, tokenA, tokenB string) (uint64, bool) {
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}
	key := append(v2.PoolByTokensKeyPrefix, []byte(tokenA)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(tokenB)...)

	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
}

// Helper to count pools
func countPools(store storetypes.KVStore) int {
	iterator := storetypes.KVStorePrefixIterator(store, v2.PoolKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	return count
}

// Helper to set liquidity position
func setLiquidityPosition(store storetypes.KVStore, poolID uint64, addr sdk.AccAddress, shares math.Int) {
	key := append(v2.LiquidityKeyPrefix, getPoolIDBytes(poolID)...)
	key = append(key, addr.Bytes()...)
	bz, _ := shares.Marshal()
	store.Set(key, bz)
}

// Helper to check circuit breaker exists and optionally verify it can be unmarshaled
func hasCircuitBreaker(store storetypes.KVStore, poolID uint64) bool {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, poolID)
	key := append(v2.CircuitBreakerKeyPrefix, bz...)
	return store.Has(key)
}

// Helper to get circuit breaker state
func getCircuitBreakerState(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, poolID uint64) (types.CircuitBreakerState, bool) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, poolID)
	key := append(v2.CircuitBreakerKeyPrefix, bz...)
	data := store.Get(key)
	if data == nil {
		return types.CircuitBreakerState{}, false
	}
	var state types.CircuitBreakerState
	err := cdc.Unmarshal(data, &state)
	require.NoError(t, err)
	return state, true
}

func getPoolIDBytes(id uint64) []byte {
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
	require.Equal(t, types.DefaultParams().SwapFee, params.SwapFee)
}

func TestMigrate_ValidPoolData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up valid pool
	validPool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, validPool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Pool should remain unchanged
	pool, found := getPool(t, store, cdc, 1)
	require.True(t, found)
	require.Equal(t, validPool.Id, pool.Id)
	require.Equal(t, validPool.TokenA, pool.TokenA)
	require.Equal(t, validPool.TokenB, pool.TokenB)
	require.True(t, validPool.ReserveA.Equal(pool.ReserveA))
	require.True(t, validPool.ReserveB.Equal(pool.ReserveB))
}

func TestMigrate_FixNegativeReserves(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool with negative reserves
	invalidPool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(-1000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, invalidPool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Negative reserve should be fixed to zero
	pool, found := getPool(t, store, cdc, 1)
	require.True(t, found)
	require.True(t, pool.ReserveA.IsZero())
	require.True(t, pool.ReserveB.Equal(math.NewInt(500000)))
}

func TestMigrate_FixNegativeTotalShares(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool with negative total shares
	invalidPool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(-100),
	}
	setPool(t, store, cdc, invalidPool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Negative total shares should be fixed to zero
	pool, found := getPool(t, store, cdc, 1)
	require.True(t, found)
	require.True(t, pool.TotalShares.IsZero())
}

func TestMigrate_FixTokenOrdering(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool with incorrect token ordering
	invalidPool := types.Pool{
		Id:          1,
		TokenA:      "upaw", // Should be second lexicographically
		TokenB:      "uatom", // Should be first lexicographically
		ReserveA:    math.NewInt(500000),
		ReserveB:    math.NewInt(1000000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, invalidPool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Token ordering should be fixed
	pool, found := getPool(t, store, cdc, 1)
	require.True(t, found)
	require.Equal(t, "uatom", pool.TokenA)
	require.Equal(t, "upaw", pool.TokenB)
	// Reserves should be swapped with tokens
	require.True(t, pool.ReserveA.Equal(math.NewInt(1000000)))
	require.True(t, pool.ReserveB.Equal(math.NewInt(500000)))
}

func TestMigrate_RebuildPoolIndexes(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pools
	pool1 := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	pool2 := types.Pool{
		Id:          2,
		TokenA:      "ueth",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(2000000),
		ReserveB:    math.NewInt(1000000),
		TotalShares: math.NewInt(1500000),
	}
	setPool(t, store, cdc, pool1)
	setPool(t, store, cdc, pool2)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Indexes should be rebuilt
	poolID, found := getPoolByTokensIndex(store, "uatom", "upaw")
	require.True(t, found)
	require.Equal(t, uint64(1), poolID)

	poolID, found = getPoolByTokensIndex(store, "ueth", "upaw")
	require.True(t, found)
	require.Equal(t, uint64(2), poolID)
}

func TestMigrate_InitializeCircuitBreakers(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool
	pool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, pool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Circuit breaker should be initialized
	require.True(t, hasCircuitBreaker(store, 1))
}

func TestMigrate_CircuitBreakerAlreadyExists(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool
	pool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, pool)

	// Pre-set circuit breaker with protobuf encoding
	cbKey := append(v2.CircuitBreakerKeyPrefix, getPoolIDBytes(1)...)
	existingCB := &types.CircuitBreakerState{
		Enabled:     true,
		TriggeredBy: "manual",
		LastPrice:   math.LegacyMustNewDecFromStr("0.5"),
	}
	cbData, err := cdc.Marshal(existingCB)
	require.NoError(t, err)
	store.Set(cbKey, cbData)

	// Run migration
	err = v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Existing circuit breaker should not be overwritten
	cb, found := getCircuitBreakerState(t, store, cdc, 1)
	require.True(t, found)
	require.True(t, cb.Enabled)
	require.Equal(t, "manual", cb.TriggeredBy)
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
	require.Equal(t, defaults.SwapFee, params.SwapFee)
}

func TestMigrate_Params_UpdateZeroFields(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with zero new fields
	oldParams := types.Params{
		SwapFee:      math.LegacyZeroDec(),
		MinLiquidity: math.ZeroInt(),
	}
	setParams(t, store, cdc, oldParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Zero fields should be updated with defaults
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.003"), params.SwapFee)
	require.True(t, params.MinLiquidity.Equal(math.NewInt(1000)))
}

func TestMigrate_Params_PreserveExistingValues(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with existing valid values
	existingParams := types.Params{
		SwapFee:      math.LegacyMustNewDecFromStr("0.005"),
		MinLiquidity: math.NewInt(5000),
	}
	setParams(t, store, cdc, existingParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Existing values should be preserved
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, existingParams.SwapFee, params.SwapFee)
	require.True(t, existingParams.MinLiquidity.Equal(params.MinLiquidity))
}

func TestMigrate_ValidatePoolCounter(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pools
	pool1 := types.Pool{Id: 5, TokenA: "uatom", TokenB: "upaw", ReserveA: math.NewInt(100), ReserveB: math.NewInt(100), TotalShares: math.NewInt(100)}
	pool2 := types.Pool{Id: 10, TokenA: "ueth", TokenB: "upaw", ReserveA: math.NewInt(100), ReserveB: math.NewInt(100), TotalShares: math.NewInt(100)}
	setPool(t, store, cdc, pool1)
	setPool(t, store, cdc, pool2)

	// Set counter too low
	setPoolCounter(store, 5)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Counter should be updated to maxID + 1
	counter := getPoolCounter(store)
	require.Equal(t, uint64(11), counter)
}

func TestMigrate_PoolCounterAlreadyCorrect(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool
	pool := types.Pool{Id: 5, TokenA: "uatom", TokenB: "upaw", ReserveA: math.NewInt(100), ReserveB: math.NewInt(100), TotalShares: math.NewInt(100)}
	setPool(t, store, cdc, pool)

	// Set counter correctly
	setPoolCounter(store, 100)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Counter should remain unchanged
	counter := getPoolCounter(store)
	require.Equal(t, uint64(100), counter)
}

func TestMigrate_LiquidityPositions_FixNegative(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool
	pool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, pool)

	// Set up liquidity position with negative shares (invalid state)
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	setLiquidityPosition(store, 1, addr, math.NewInt(-100))

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Negative shares should be fixed to zero
	key := append(v2.LiquidityKeyPrefix, getPoolIDBytes(1)...)
	key = append(key, addr.Bytes()...)
	bz := store.Get(key)
	require.NotNil(t, bz)
	var shares math.Int
	err = shares.Unmarshal(bz)
	require.NoError(t, err)
	require.True(t, shares.IsZero())
}

func TestMigrate_MultiplePools(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create many pools with various issues
	pools := []types.Pool{
		{Id: 1, TokenA: "uatom", TokenB: "upaw", ReserveA: math.NewInt(1000), ReserveB: math.NewInt(500), TotalShares: math.NewInt(750)},
		{Id: 2, TokenA: "ueth", TokenB: "upaw", ReserveA: math.NewInt(-100), ReserveB: math.NewInt(200), TotalShares: math.NewInt(150)}, // Negative reserve
		{Id: 3, TokenA: "usol", TokenB: "uatom", ReserveA: math.NewInt(300), ReserveB: math.NewInt(400), TotalShares: math.NewInt(-50)}, // Negative shares
		{Id: 4, TokenA: "uusdc", TokenB: "uatom", ReserveA: math.NewInt(500), ReserveB: math.NewInt(600), TotalShares: math.NewInt(550)}, // Wrong order
	}
	for _, p := range pools {
		setPool(t, store, cdc, p)
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Verify all pools are fixed
	pool2, _ := getPool(t, store, cdc, 2)
	require.True(t, pool2.ReserveA.IsZero())

	pool3, _ := getPool(t, store, cdc, 3)
	require.True(t, pool3.TotalShares.IsZero())

	pool4, _ := getPool(t, store, cdc, 4)
	require.Equal(t, "uatom", pool4.TokenA)
	require.Equal(t, "uusdc", pool4.TokenB)

	// All pools should have circuit breakers
	for _, p := range pools {
		require.True(t, hasCircuitBreaker(store, p.Id))
	}
}

func TestMigrate_Idempotency(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool
	pool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(500000),
		TotalShares: math.NewInt(750000),
	}
	setPool(t, store, cdc, pool)

	// Run migration first time
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Get state after first migration
	pool1, _ := getPool(t, store, cdc, 1)
	params1, _ := getParams(t, store, cdc)
	counter1 := getPoolCounter(store)

	// Run migration second time
	err = v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// State should be identical
	pool2, _ := getPool(t, store, cdc, 1)
	params2, _ := getParams(t, store, cdc)
	counter2 := getPoolCounter(store)

	require.Equal(t, pool1, pool2)
	require.Equal(t, params1, params2)
	require.Equal(t, counter1, counter2)
}

func TestMigrate_CorruptedPoolData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted pool data
	key := append(v2.PoolKeyPrefix, getPoolIDBytes(1)...)
	store.Set(key, []byte("invalid pool data"))

	// Run migration - should error on corrupted data
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal pool")
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

	// Create many pools
	for i := uint64(1); i <= 100; i++ {
		pool := types.Pool{
			Id:          i,
			TokenA:      "uatom",
			TokenB:      "upaw",
			ReserveA:    math.NewInt(int64(i * 1000)),
			ReserveB:    math.NewInt(int64(i * 500)),
			TotalShares: math.NewInt(int64(i * 750)),
		}
		setPool(t, store, cdc, pool)
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// All pools should exist
	require.Equal(t, 100, countPools(store))

	// Counter should be correct
	counter := getPoolCounter(store)
	require.Equal(t, uint64(101), counter)
}

func TestMigrate_ZeroReserves(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up pool with zero reserves (valid but unusual)
	pool := types.Pool{
		Id:          1,
		TokenA:      "uatom",
		TokenB:      "upaw",
		ReserveA:    math.ZeroInt(),
		ReserveB:    math.ZeroInt(),
		TotalShares: math.ZeroInt(),
	}
	setPool(t, store, cdc, pool)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Pool should remain unchanged
	result, found := getPool(t, store, cdc, 1)
	require.True(t, found)
	require.True(t, result.ReserveA.IsZero())
	require.True(t, result.ReserveB.IsZero())
}
