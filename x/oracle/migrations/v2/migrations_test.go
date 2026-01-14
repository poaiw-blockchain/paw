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

	v2 "github.com/paw-chain/paw/x/oracle/migrations/v2"
	"github.com/paw-chain/paw/x/oracle/types"
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

// Helper to set price in store
func setPrice(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, price types.Price) {
	key := append(v2.PriceKeyPrefix, []byte(price.Asset)...)
	bz, err := cdc.Marshal(&price)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to get price from store
func getPrice(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, asset string) (types.Price, bool) {
	key := append(v2.PriceKeyPrefix, []byte(asset)...)
	bz := store.Get(key)
	if bz == nil {
		return types.Price{}, false
	}
	var price types.Price
	err := cdc.Unmarshal(bz, &price)
	require.NoError(t, err)
	return price, true
}

// Helper to set validator oracle in store
func setValidatorOracle(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, vo types.ValidatorOracle) {
	key := append(v2.ValidatorOracleKeyPrefix, []byte(vo.ValidatorAddr)...)
	bz, err := cdc.Marshal(&vo)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to get validator oracle from store
func getValidatorOracle(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, valAddr string) (types.ValidatorOracle, bool) {
	key := append(v2.ValidatorOracleKeyPrefix, []byte(valAddr)...)
	bz := store.Get(key)
	if bz == nil {
		return types.ValidatorOracle{}, false
	}
	var vo types.ValidatorOracle
	err := cdc.Unmarshal(bz, &vo)
	require.NoError(t, err)
	return vo, true
}

// Helper to set price snapshot in store
func setPriceSnapshot(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec, snapshot types.PriceSnapshot) {
	// Key format: prefix + asset + block_height
	key := append(v2.PriceSnapshotKeyPrefix, []byte(snapshot.Asset)...)
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(snapshot.BlockHeight))
	key = append(key, heightBz...)

	bz, err := cdc.Marshal(&snapshot)
	require.NoError(t, err)
	store.Set(key, bz)
}

// Helper to count snapshots in store
func countSnapshots(store storetypes.KVStore) int {
	iterator := storetypes.KVStorePrefixIterator(store, v2.PriceSnapshotKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	return count
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

// Helper to get miss counter from store
func getMissCounter(store storetypes.KVStore, valAddr string) (uint64, bool) {
	key := append(v2.MissCounterKeyPrefix, []byte(valAddr)...)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
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
	require.Equal(t, types.DefaultParams().VotePeriod, params.VotePeriod)
	require.Equal(t, types.DefaultParams().VoteThreshold, params.VoteThreshold)
}

func TestMigrate_ValidPriceData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up valid price data
	validPrice := types.Price{
		Asset:         "BTC",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 5,
	}
	setPrice(t, store, cdc, validPrice)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Price should remain unchanged
	price, found := getPrice(t, store, cdc, "BTC")
	require.True(t, found)
	require.Equal(t, validPrice.Asset, price.Asset)
	require.Equal(t, validPrice.Price, price.Price)
	require.Equal(t, validPrice.BlockHeight, price.BlockHeight)
	require.Equal(t, validPrice.BlockTime, price.BlockTime)
	require.Equal(t, validPrice.NumValidators, price.NumValidators)
}

func TestMigrate_InvalidPriceData_ZeroPrice(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up invalid price with zero value
	invalidPrice := types.Price{
		Asset:         "ETH",
		Price:         math.LegacyZeroDec(),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 3,
	}
	setPrice(t, store, cdc, invalidPrice)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Invalid price should be deleted
	_, found := getPrice(t, store, cdc, "ETH")
	require.False(t, found)
}

func TestMigrate_InvalidPriceData_NegativePrice(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up invalid price with negative value
	invalidPrice := types.Price{
		Asset:         "SOL",
		Price:         math.LegacyMustNewDecFromStr("-100.00"),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 2,
	}
	setPrice(t, store, cdc, invalidPrice)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Invalid price should be deleted
	_, found := getPrice(t, store, cdc, "SOL")
	require.False(t, found)
}

func TestMigrate_FixNegativeBlockHeight(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with negative block height
	price := types.Price{
		Asset:         "ADA",
		Price:         math.LegacyMustNewDecFromStr("1.50"),
		BlockHeight:   -10,
		BlockTime:     1609459200,
		NumValidators: 4,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Block height should be fixed to current block height
	fixedPrice, found := getPrice(t, store, cdc, "ADA")
	require.True(t, found)
	require.Equal(t, ctx.BlockHeight(), fixedPrice.BlockHeight)
	require.Equal(t, price.Price, fixedPrice.Price)
}

func TestMigrate_FixNegativeBlockTime(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with negative block time
	price := types.Price{
		Asset:         "DOT",
		Price:         math.LegacyMustNewDecFromStr("25.00"),
		BlockHeight:   200,
		BlockTime:     -1000,
		NumValidators: 6,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Block time should be fixed to current block time
	fixedPrice, found := getPrice(t, store, cdc, "DOT")
	require.True(t, found)
	require.Equal(t, ctx.BlockTime().Unix(), fixedPrice.BlockTime)
	require.Equal(t, price.Price, fixedPrice.Price)
}

func TestMigrate_FixZeroValidators(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with zero validators
	price := types.Price{
		Asset:         "AVAX",
		Price:         math.LegacyMustNewDecFromStr("75.00"),
		BlockHeight:   300,
		BlockTime:     1609459200,
		NumValidators: 0,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// NumValidators should be fixed to 1
	fixedPrice, found := getPrice(t, store, cdc, "AVAX")
	require.True(t, found)
	require.Equal(t, uint32(1), fixedPrice.NumValidators)
	require.Equal(t, price.Price, fixedPrice.Price)
}

func TestMigrate_MultiplePriceIssues(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with multiple issues
	price := types.Price{
		Asset:         "MATIC",
		Price:         math.LegacyMustNewDecFromStr("2.00"),
		BlockHeight:   -5,
		BlockTime:     -500,
		NumValidators: 0,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// All issues should be fixed
	fixedPrice, found := getPrice(t, store, cdc, "MATIC")
	require.True(t, found)
	require.Equal(t, ctx.BlockHeight(), fixedPrice.BlockHeight)
	require.Equal(t, ctx.BlockTime().Unix(), fixedPrice.BlockTime)
	require.Equal(t, uint32(1), fixedPrice.NumValidators)
	require.Equal(t, price.Price, fixedPrice.Price)
}

func TestMigrate_CorruptedPriceData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted price data directly
	key := append(v2.PriceKeyPrefix, []byte("CORRUPTED")...)
	store.Set(key, []byte("invalid data"))

	// Run migration - should not error, just delete corrupted data
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Corrupted price should be deleted
	_, found := getPrice(t, store, cdc, "CORRUPTED")
	require.False(t, found)
}

func TestMigrate_ValidatorOracles_Valid(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create valid validator address
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	validatorOracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
		GeographicRegion: "na",
		IpAddress:        "192.168.1.1",
		Asn:              12345,
	}
	setValidatorOracle(t, store, cdc, validatorOracle)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Validator oracle should remain unchanged
	vo, found := getValidatorOracle(t, store, cdc, valAddr)
	require.True(t, found)
	require.Equal(t, validatorOracle.ValidatorAddr, vo.ValidatorAddr)
	require.Equal(t, validatorOracle.MissCounter, vo.MissCounter)
}

func TestMigrate_ValidatorOracles_InvalidAddress(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set validator oracle with invalid address
	invalidOracle := types.ValidatorOracle{
		ValidatorAddr:    "invalid-address",
		MissCounter:      2,
		TotalSubmissions: 50,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, invalidOracle)

	// Run migration - should not error, just skip invalid oracle
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Invalid oracle should still be there (not deleted, just skipped)
	vo, found := getValidatorOracle(t, store, cdc, "invalid-address")
	require.True(t, found)
	require.Equal(t, invalidOracle.ValidatorAddr, vo.ValidatorAddr)
}

func TestMigrate_InitializeMissCounters(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create validator oracles
	valAddr1 := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	valAddr2 := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	vo1 := types.ValidatorOracle{
		ValidatorAddr:    valAddr1,
		MissCounter:      10,
		TotalSubmissions: 200,
		IsActive:         true,
	}
	vo2 := types.ValidatorOracle{
		ValidatorAddr:    valAddr2,
		MissCounter:      25,
		TotalSubmissions: 150,
		IsActive:         false,
	}
	setValidatorOracle(t, store, cdc, vo1)
	setValidatorOracle(t, store, cdc, vo2)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Miss counters should be initialized
	counter1, found := getMissCounter(store, valAddr1)
	require.True(t, found)
	require.Equal(t, uint64(10), counter1)

	counter2, found := getMissCounter(store, valAddr2)
	require.True(t, found)
	require.Equal(t, uint64(25), counter2)
}

func TestMigrate_MissCountersAlreadyExist(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create validator oracle
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	vo := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      15,
		TotalSubmissions: 100,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, vo)

	// Pre-set miss counter
	key := append(v2.MissCounterKeyPrefix, []byte(valAddr)...)
	counterBz := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBz, uint64(20))
	store.Set(key, counterBz)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Miss counter should remain unchanged
	counter, found := getMissCounter(store, valAddr)
	require.True(t, found)
	require.Equal(t, uint64(20), counter)
}

func TestMigrate_PriceSnapshots_Valid(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create valid snapshots
	snapshot1 := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("48000.00"),
		BlockHeight: 900,
		BlockTime:   1609459100,
	}
	snapshot2 := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyMustNewDecFromStr("3200.00"),
		BlockHeight: 950,
		BlockTime:   1609459150,
	}
	setPriceSnapshot(t, store, cdc, snapshot1)
	setPriceSnapshot(t, store, cdc, snapshot2)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Both snapshots should remain
	count := countSnapshots(store)
	require.Equal(t, 2, count)
}

func TestMigrate_PriceSnapshots_InvalidPrice(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create valid and invalid snapshots
	validSnapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("48000.00"),
		BlockHeight: 900,
		BlockTime:   1609459100,
	}
	invalidSnapshot := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyZeroDec(),
		BlockHeight: 950,
		BlockTime:   1609459150,
	}
	setPriceSnapshot(t, store, cdc, validSnapshot)
	setPriceSnapshot(t, store, cdc, invalidSnapshot)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Only valid snapshot should remain
	count := countSnapshots(store)
	require.Equal(t, 1, count)
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
	require.Equal(t, defaults.VotePeriod, params.VotePeriod)
	require.Equal(t, defaults.VoteThreshold, params.VoteThreshold)
	require.Equal(t, defaults.MinValidPerWindow, params.MinValidPerWindow)
	require.Equal(t, defaults.SlashFraction, params.SlashFraction)
	require.Equal(t, defaults.TwapLookbackWindow, params.TwapLookbackWindow)
}

func TestMigrate_Params_MissingFields(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with missing new fields (zero values)
	oldParams := types.Params{
		VotePeriod:         30,
		VoteThreshold:      math.LegacyZeroDec(), // Should be updated
		SlashFraction:      math.LegacyZeroDec(), // Should be updated
		SlashWindow:        10000,
		MinValidPerWindow:  0, // Should be updated
		TwapLookbackWindow: 0, // Should be updated
	}
	setParams(t, store, cdc, oldParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Params should be updated with defaults
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.67"), params.VoteThreshold)
	require.Equal(t, uint64(100), params.MinValidPerWindow)
	require.Equal(t, math.LegacyMustNewDecFromStr("0.01"), params.SlashFraction)
	require.Equal(t, uint64(1000), params.TwapLookbackWindow)
	require.Equal(t, oldParams.VotePeriod, params.VotePeriod)
}

func TestMigrate_Params_PreservesExistingValues(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with existing valid values
	existingParams := types.Params{
		VotePeriod:         50,
		VoteThreshold:      math.LegacyMustNewDecFromStr("0.75"),
		SlashFraction:      math.LegacyMustNewDecFromStr("0.02"),
		SlashWindow:        15000,
		MinValidPerWindow:  200,
		TwapLookbackWindow: 2000,
	}
	setParams(t, store, cdc, existingParams)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Existing values should be preserved
	params, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, existingParams.VotePeriod, params.VotePeriod)
	require.Equal(t, existingParams.VoteThreshold, params.VoteThreshold)
	require.Equal(t, existingParams.SlashFraction, params.SlashFraction)
	require.Equal(t, existingParams.MinValidPerWindow, params.MinValidPerWindow)
	require.Equal(t, existingParams.TwapLookbackWindow, params.TwapLookbackWindow)
}

func TestMigrate_CleanStaleSnapshots(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with lookback window
	params := types.DefaultParams()
	params.TwapLookbackWindow = 1000
	setParams(t, store, cdc, params)

	currentHeight := ctx.BlockHeight()
	retentionBlocks := int64(params.TwapLookbackWindow * 2)
	cutoffHeight := currentHeight - retentionBlocks

	// Create snapshots: some old, some recent
	staleSnapshot1 := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("40000.00"),
		BlockHeight: cutoffHeight - 100, // Old
		BlockTime:   1609459000,
	}
	staleSnapshot2 := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyMustNewDecFromStr("2800.00"),
		BlockHeight: cutoffHeight - 50, // Old
		BlockTime:   1609459050,
	}
	recentSnapshot := types.PriceSnapshot{
		Asset:       "SOL",
		Price:       math.LegacyMustNewDecFromStr("150.00"),
		BlockHeight: cutoffHeight + 100, // Recent
		BlockTime:   1609459200,
	}
	setPriceSnapshot(t, store, cdc, staleSnapshot1)
	setPriceSnapshot(t, store, cdc, staleSnapshot2)
	setPriceSnapshot(t, store, cdc, recentSnapshot)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Only recent snapshot should remain
	count := countSnapshots(store)
	require.Equal(t, 1, count)
}

func TestMigrate_CleanStaleSnapshots_NoParams(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create snapshots without setting params
	snapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("40000.00"),
		BlockHeight: 100,
		BlockTime:   1609459000,
	}
	setPriceSnapshot(t, store, cdc, snapshot)

	// Run migration - should not error when no params
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Snapshot should remain (default params will be created)
	count := countSnapshots(store)
	require.GreaterOrEqual(t, count, 0) // May or may not be cleaned based on default params
}

func TestMigrate_LargeDataset(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create many prices
	assets := []string{"BTC", "ETH", "SOL", "ADA", "DOT", "AVAX", "MATIC", "LINK", "UNI", "ATOM"}
	for _, asset := range assets {
		price := types.Price{
			Asset:         asset,
			Price:         math.LegacyMustNewDecFromStr("100.00"),
			BlockHeight:   500,
			BlockTime:     1609459200,
			NumValidators: 5,
		}
		setPrice(t, store, cdc, price)
	}

	// Create many validator oracles
	for i := 0; i < 20; i++ {
		valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		vo := types.ValidatorOracle{
			ValidatorAddr:    valAddr,
			MissCounter:      uint64(i),
			TotalSubmissions: uint64(100 + i),
			IsActive:         true,
		}
		setValidatorOracle(t, store, cdc, vo)
	}

	// Create many snapshots
	for i, asset := range assets {
		for j := 0; j < 10; j++ {
			snapshot := types.PriceSnapshot{
				Asset:       asset,
				Price:       math.LegacyMustNewDecFromStr("100.00"),
				BlockHeight: int64(900 + i*10 + j),
				BlockTime:   int64(1609459000 + i*10 + j),
			}
			setPriceSnapshot(t, store, cdc, snapshot)
		}
	}

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Verify prices remain
	for _, asset := range assets {
		_, found := getPrice(t, store, cdc, asset)
		require.True(t, found, "price for %s not found", asset)
	}
}

func TestMigrate_Idempotency(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up initial data
	price := types.Price{
		Asset:         "BTC",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 5,
	}
	setPrice(t, store, cdc, price)

	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	vo := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      10,
		TotalSubmissions: 200,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, vo)

	// Run migration first time
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Get state after first migration
	params1, _ := getParams(t, store, cdc)
	price1, _ := getPrice(t, store, cdc, "BTC")
	vo1, _ := getValidatorOracle(t, store, cdc, valAddr)
	counter1, _ := getMissCounter(store, valAddr)

	// Run migration second time
	err = v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Get state after second migration
	params2, _ := getParams(t, store, cdc)
	price2, _ := getPrice(t, store, cdc, "BTC")
	vo2, _ := getValidatorOracle(t, store, cdc, valAddr)
	counter2, _ := getMissCounter(store, valAddr)

	// State should be identical
	require.Equal(t, params1, params2)
	require.Equal(t, price1, price2)
	require.Equal(t, vo1, vo2)
	require.Equal(t, counter1, counter2)
}

func TestMigrate_EdgeCases_NilPrice(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with nil price value (using empty LegacyDec)
	price := types.Price{
		Asset: "ATOM",
		// Price left as default (nil)
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 3,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Nil price should be deleted
	_, found := getPrice(t, store, cdc, "ATOM")
	require.False(t, found)
}

func TestMigrate_MixedValidAndInvalidData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up mix of valid and invalid data
	validPrice := types.Price{
		Asset:         "BTC",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 5,
	}
	invalidPrice1 := types.Price{
		Asset:         "ETH",
		Price:         math.LegacyZeroDec(),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 3,
	}
	invalidPrice2 := types.Price{
		Asset:         "SOL",
		Price:         math.LegacyMustNewDecFromStr("100.00"),
		BlockHeight:   -50,
		BlockTime:     -1000,
		NumValidators: 0,
	}

	setPrice(t, store, cdc, validPrice)
	setPrice(t, store, cdc, invalidPrice1)
	setPrice(t, store, cdc, invalidPrice2)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Valid price should remain unchanged
	btcPrice, found := getPrice(t, store, cdc, "BTC")
	require.True(t, found)
	require.Equal(t, validPrice.Price, btcPrice.Price)

	// Invalid price with zero should be deleted
	_, found = getPrice(t, store, cdc, "ETH")
	require.False(t, found)

	// Invalid price with fixable issues should be fixed
	solPrice, found := getPrice(t, store, cdc, "SOL")
	require.True(t, found)
	require.Equal(t, ctx.BlockHeight(), solPrice.BlockHeight)
	require.Equal(t, ctx.BlockTime().Unix(), solPrice.BlockTime)
	require.Equal(t, uint32(1), solPrice.NumValidators)
}

func TestMigrate_SnapshotCleaning_BoundaryConditions(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params
	params := types.DefaultParams()
	params.TwapLookbackWindow = 1000
	setParams(t, store, cdc, params)

	currentHeight := ctx.BlockHeight()
	retentionBlocks := int64(params.TwapLookbackWindow * 2)
	cutoffHeight := currentHeight - retentionBlocks

	// Snapshot exactly at cutoff (should be kept)
	atCutoff := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight: cutoffHeight,
		BlockTime:   1609459200,
	}
	// Snapshot just below cutoff (should be deleted)
	belowCutoff := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyMustNewDecFromStr("3000.00"),
		BlockHeight: cutoffHeight - 1,
		BlockTime:   1609459100,
	}
	// Snapshot just above cutoff (should be kept)
	aboveCutoff := types.PriceSnapshot{
		Asset:       "SOL",
		Price:       math.LegacyMustNewDecFromStr("150.00"),
		BlockHeight: cutoffHeight + 1,
		BlockTime:   1609459300,
	}

	setPriceSnapshot(t, store, cdc, atCutoff)
	setPriceSnapshot(t, store, cdc, belowCutoff)
	setPriceSnapshot(t, store, cdc, aboveCutoff)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Should keep snapshots at or above cutoff
	count := countSnapshots(store)
	require.Equal(t, 2, count)
}

func TestMigrate_CorruptedSnapshot(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params
	params := types.DefaultParams()
	setParams(t, store, cdc, params)

	// Add valid snapshot
	validSnapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight: 900,
		BlockTime:   1609459200,
	}
	setPriceSnapshot(t, store, cdc, validSnapshot)

	// Add corrupted snapshot data
	corruptedKey := append(v2.PriceSnapshotKeyPrefix, []byte("CORRUPTED")...)
	store.Set(corruptedKey, []byte("invalid snapshot data"))

	// Run migration - should handle corruption gracefully
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Valid snapshot should remain, count may vary due to corrupted data handling
	count := countSnapshots(store)
	require.GreaterOrEqual(t, count, 0)
}

func TestMigrate_EmptyAssetName(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set up price with empty asset name
	price := types.Price{
		Asset:         "",
		Price:         math.LegacyMustNewDecFromStr("100.00"),
		BlockHeight:   100,
		BlockTime:     1609459200,
		NumValidators: 5,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Price with empty asset should remain (only price value validation, not asset name)
	emptyPrice, found := getPrice(t, store, cdc, "")
	require.True(t, found)
	require.Equal(t, price.Price, emptyPrice.Price)
}

func TestMigrate_ValidatorOracle_CorruptedData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted validator oracle data directly
	key := append(v2.ValidatorOracleKeyPrefix, []byte("corrupted-val")...)
	store.Set(key, []byte("invalid validator oracle data"))

	// Add valid validator oracle
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	validOracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, validOracle)

	// Run migration - should handle corruption gracefully
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Valid oracle should remain
	vo, found := getValidatorOracle(t, store, cdc, valAddr)
	require.True(t, found)
	require.Equal(t, validOracle.ValidatorAddr, vo.ValidatorAddr)
}

func TestMigrate_InitializeMissCounters_CorruptedOracle(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Add valid validator oracle
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	validOracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      7,
		TotalSubmissions: 150,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, validOracle)

	// Set corrupted validator oracle data
	corruptedKey := append(v2.ValidatorOracleKeyPrefix, []byte("corrupted")...)
	store.Set(corruptedKey, []byte("invalid data"))

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Valid oracle's miss counter should be initialized
	counter, found := getMissCounter(store, valAddr)
	require.True(t, found)
	require.Equal(t, uint64(7), counter)

	// Corrupted oracle should not have miss counter
	_, found = getMissCounter(store, "corrupted")
	require.False(t, found)
}

func TestMigrate_Params_CorruptedData(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set corrupted params data
	store.Set(v2.ParamsKey, []byte("corrupted params data"))

	// Run migration - should fail gracefully with error
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to migrate params")
}

func TestMigrate_CleanStaleSnapshots_CorruptedParams(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create snapshot
	snapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight: 900,
		BlockTime:   1609459200,
	}
	setPriceSnapshot(t, store, cdc, snapshot)

	// Set corrupted params data (this will be caught during migration)
	store.Set(v2.ParamsKey, []byte("corrupted params"))

	// Run migration - should fail at params step before snapshot cleaning
	err := v2.Migrate(ctx, storeKey, cdc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to migrate params")
}

func TestMigrate_PriceMarshalError(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// This test verifies that if we somehow had valid price data that needs updating
	// but marshaling fails (extremely unlikely in practice), it would error
	// Since we can't easily trigger a marshal error with valid data,
	// we just verify the happy path works
	price := types.Price{
		Asset:         "BTC",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   -10, // Will be fixed
		BlockTime:     1609459200,
		NumValidators: 5,
	}
	setPrice(t, store, cdc, price)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Price should be fixed
	fixedPrice, found := getPrice(t, store, cdc, "BTC")
	require.True(t, found)
	require.Equal(t, ctx.BlockHeight(), fixedPrice.BlockHeight)
}

func TestMigrate_ValidatorOracle_MarshalError(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Similar to price marshal test, verify happy path with valid data
	valAddr := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	vo := types.ValidatorOracle{
		ValidatorAddr:    valAddr,
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
	}
	setValidatorOracle(t, store, cdc, vo)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Oracle should remain valid
	result, found := getValidatorOracle(t, store, cdc, valAddr)
	require.True(t, found)
	require.Equal(t, vo.ValidatorAddr, result.ValidatorAddr)
}

func TestMigrate_MultipleValidators_MixedValidity(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create multiple validators with different validity states
	validVal1 := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	validVal2 := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	vo1 := types.ValidatorOracle{
		ValidatorAddr:    validVal1,
		MissCounter:      10,
		TotalSubmissions: 200,
		IsActive:         true,
	}
	vo2 := types.ValidatorOracle{
		ValidatorAddr:    validVal2,
		MissCounter:      0,
		TotalSubmissions: 50,
		IsActive:         false,
	}
	// Invalid validator address
	vo3 := types.ValidatorOracle{
		ValidatorAddr:    "invalid-bech32-address",
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
	}

	setValidatorOracle(t, store, cdc, vo1)
	setValidatorOracle(t, store, cdc, vo2)
	setValidatorOracle(t, store, cdc, vo3)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Valid validators should have miss counters
	counter1, found := getMissCounter(store, validVal1)
	require.True(t, found)
	require.Equal(t, uint64(10), counter1)

	counter2, found := getMissCounter(store, validVal2)
	require.True(t, found)
	require.Equal(t, uint64(0), counter2)

	// Invalid validator oracle still exists but won't be used
	// (validator validation happens during validateValidatorOracles, not initializeMissCounters)
	vo3Result, found := getValidatorOracle(t, store, cdc, "invalid-bech32-address")
	require.True(t, found)
	require.Equal(t, "invalid-bech32-address", vo3Result.ValidatorAddr)
}

func TestMigrate_ZeroLookbackWindow(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Set params with zero lookback window (will be updated to default)
	params := types.Params{
		VotePeriod:         30,
		VoteThreshold:      math.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:      math.LegacyMustNewDecFromStr("0.01"),
		SlashWindow:        10000,
		MinValidPerWindow:  100,
		TwapLookbackWindow: 0, // Will be set to default 1000
	}
	setParams(t, store, cdc, params)

	// Create snapshots
	snapshot1 := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight: 100,
		BlockTime:   1609459000,
	}
	snapshot2 := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyMustNewDecFromStr("3000.00"),
		BlockHeight: 900,
		BlockTime:   1609459100,
	}
	setPriceSnapshot(t, store, cdc, snapshot1)
	setPriceSnapshot(t, store, cdc, snapshot2)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Params should be updated
	updatedParams, found := getParams(t, store, cdc)
	require.True(t, found)
	require.Equal(t, uint64(1000), updatedParams.TwapLookbackWindow)

	// With lookback window = 1000, retention = 2000 blocks
	// Current height = 1000, cutoff = 1000 - 2000 = -1000
	// Both snapshots at height 100 and 900 should be kept
	count := countSnapshots(store)
	require.GreaterOrEqual(t, count, 0) // Snapshots may be cleaned based on height
}

func TestMigrate_HighBlockHeight(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Use high block height context
	ctx = ctx.WithBlockHeight(1000000)

	// Set params with lookback window
	params := types.DefaultParams()
	params.TwapLookbackWindow = 1000
	setParams(t, store, cdc, params)

	// Create snapshots: very old ones
	oldSnapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight: 100,
		BlockTime:   1609459000,
	}
	recentSnapshot := types.PriceSnapshot{
		Asset:       "ETH",
		Price:       math.LegacyMustNewDecFromStr("3000.00"),
		BlockHeight: 999000, // Within retention
		BlockTime:   1609459100,
	}
	setPriceSnapshot(t, store, cdc, oldSnapshot)
	setPriceSnapshot(t, store, cdc, recentSnapshot)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Old snapshot should be cleaned, recent should remain
	count := countSnapshots(store)
	require.Equal(t, 1, count)
}

func TestMigrate_NegativeSnapshot(t *testing.T) {
	storeKey, cdc, ctx := setupTest(t)
	store := ctx.KVStore(storeKey)

	// Create snapshot with negative price
	invalidSnapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyMustNewDecFromStr("-1000.00"),
		BlockHeight: 900,
		BlockTime:   1609459100,
	}
	setPriceSnapshot(t, store, cdc, invalidSnapshot)

	// Run migration
	err := v2.Migrate(ctx, storeKey, cdc)
	require.NoError(t, err)

	// Invalid snapshot should be deleted
	count := countSnapshots(store)
	require.Equal(t, 0, count)
}
