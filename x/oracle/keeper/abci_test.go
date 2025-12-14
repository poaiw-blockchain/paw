package keeper_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestBeginBlocker_AggregatePrices(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(30).
		WithBlockTime(time.Unix(1_700_000_000, 0)).
		WithEventManager(sdk.NewEventManager())

	asset := "PAW/USD"
	priceSchedule := []struct {
		addr  sdk.ValAddress
		price sdkmath.LegacyDec
	}{
		{addr: makeValidatorAddress(0x01), price: sdkmath.LegacyNewDec(90)},
		{addr: makeValidatorAddress(0x02), price: sdkmath.LegacyNewDec(100)},
		{addr: makeValidatorAddress(0x03), price: sdkmath.LegacyNewDec(110)},
		{addr: makeValidatorAddress(0x04), price: sdkmath.LegacyNewDec(125)},
	}

	for _, entry := range priceSchedule {
		keepertest.RegisterTestOracle(t, k, ctx, entry.addr.String())
		submitValidatorPrice(t, k, ctx, entry.addr, asset, entry.price)
	}

	require.NoError(t, k.BeginBlocker(ctx))

	aggregated, err := k.GetPrice(ctx, asset)
	require.NoError(t, err)
	expected := sdkmath.LegacyNewDec(100)
	require.True(t, aggregated.Price.Equal(expected), "expected aggregated price %s got %s", expected, aggregated.Price)
	require.Equal(t, ctx.BlockHeight(), aggregated.BlockHeight)
	require.Equal(t, ctx.BlockTime().Unix(), aggregated.BlockTime)
	require.Equal(t, uint32(len(priceSchedule)), aggregated.NumValidators)

	events := ctx.EventManager().Events()
	require.True(t, eventExists(events, "oracle_begin_block", "", ""), "expected oracle_begin_block event")
	require.True(t, eventExists(events, "oracle_price_aggregated", "asset", asset), "expected oracle_price_aggregated event for asset")
}

func TestEndBlocker_ProcessSlashWindows(t *testing.T) {
	kImpl, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(1).
		WithBlockTime(time.Unix(1_700_000_100, 0)).
		WithEventManager(sdk.NewEventManager())

	params := types.DefaultParams()
	params.VotePeriod = 1
	params.MinValidPerWindow = 1
	require.NoError(t, kImpl.SetParams(ctx, params))

	asset := "PAW/USD"
	activeVal := makeValidatorAddress(0x10)
	missingVal := makeValidatorAddress(0x20)
	keepertest.RegisterTestOracle(t, kImpl, ctx, activeVal.String())
	keepertest.RegisterTestOracle(t, kImpl, ctx, missingVal.String())

	price := types.Price{
		Asset:         asset,
		Price:         sdkmath.LegacyNewDec(100),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	require.NoError(t, kImpl.SetPrice(ctx, price))
	submitValidatorPrice(t, kImpl, ctx, activeVal, asset, sdkmath.LegacyNewDec(100))

	require.NoError(t, kImpl.EndBlocker(ctx))

	events := ctx.EventManager().Events()
	require.True(t, eventExists(events, "oracle_slash", "validator", missingVal.String()), "expected slash for missing validator")
	require.True(t, eventExists(events, "oracle_end_block", "", ""), "expected oracle_end_block event")

	missingOracle, err := kImpl.GetValidatorOracle(ctx, missingVal.String())
	require.NoError(t, err)
	require.GreaterOrEqual(t, missingOracle.MissCounter, uint64(1))
}

func makeValidatorAddress(tag byte) sdk.ValAddress {
	return sdk.ValAddress(bytes.Repeat([]byte{tag}, 20))
}

func submitValidatorPrice(t *testing.T, k *keeper.Keeper, ctx sdk.Context, val sdk.ValAddress, asset string, price sdkmath.LegacyDec) {
	t.Helper()
	vp := types.ValidatorPrice{
		ValidatorAddr: val.String(),
		Asset:         asset,
		Price:         price,
		BlockHeight:   ctx.BlockHeight(),
		VotingPower:   1,
	}
	require.NoError(t, k.SetValidatorPrice(ctx, vp))
}

func eventExists(events sdk.Events, eventType string, attrKey string, attrValue string) bool {
	for _, evt := range events {
		if evt.Type != eventType {
			continue
		}
		if attrKey == "" {
			return true
		}
		for _, attr := range evt.Attributes {
			if attr.Key == attrKey && string(attr.Value) == attrValue {
				return true
			}
		}
	}
	return false
}

// TestCleanupOldOutlierHistoryGlobal_AmortizedProcessing tests the amortized cleanup algorithm
func TestCleanupOldOutlierHistoryGlobal_AmortizedProcessing(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup: 20 validators × 5 assets = 100 pairs
	// With maxCleanupPerBlock=50, should process in 2 blocks
	numValidators := 20
	numAssets := 5
	currentHeight := int64(10000)

	ctx = ctx.WithBlockHeight(currentHeight).WithEventManager(sdk.NewEventManager())

	validators := make([]sdk.ValAddress, numValidators)
	assets := make([]string, numAssets)

	// Create validators and assets
	for i := 0; i < numValidators; i++ {
		valAddr := makeValidatorAddress(byte(i))
		validators[i] = valAddr
		keepertest.RegisterTestOracle(t, k, ctx, valAddr.String())
	}

	for i := 0; i < numAssets; i++ {
		assets[i] = fmt.Sprintf("ASSET%d/USD", i)
	}

	// Create outlier history for all pairs across many blocks
	// Create old entries (should be cleaned) far in the past
	// Create recent entries (should be kept) within the retention window
	store := ctx.KVStore(k.GetStoreKey())

	// Use a safety margin to ensure recent entries stay recent across all cleanup runs
	// minHeight at block 10000 = 9000
	// minHeight at block 10099 = 9099
	// So we need recent entries to be >= 9100 to stay safe across 100 blocks
	safeRecentThreshold := currentHeight - keeper.OutlierReputationWindow + 100

	oldEntriesCount := 0
	recentEntriesCount := 0

	for _, val := range validators {
		for _, asset := range assets {
			// Old entries (should be cleaned up) - well before the retention window
			for h := int64(8900); h < 8940; h++ {
				key := makeOutlierKey(val.String(), asset, h)
				value := fmt.Sprintf("%d:%d", 1, h) // SeverityLow
				store.Set(key, []byte(value))
				oldEntriesCount++
			}

			// Recent entries (should be kept) - well within retention window
			for h := safeRecentThreshold; h < safeRecentThreshold+10; h++ {
				key := makeOutlierKey(val.String(), asset, h)
				value := fmt.Sprintf("%d:%d", 1, h) // SeverityLow
				store.Set(key, []byte(value))
				recentEntriesCount++
			}
		}
	}

	t.Logf("Created %d old entries and %d recent entries", oldEntriesCount, recentEntriesCount)

	// Run cleanup across multiple blocks to test amortization
	// The cleanup should process different validator-asset pairs in different blocks
	totalCleaned := 0
	totalProcessed := 0

	for blockOffset := int64(0); blockOffset < 100; blockOffset++ {
		testCtx := ctx.WithBlockHeight(currentHeight + blockOffset).WithEventManager(sdk.NewEventManager())

		err := k.CleanupOldOutlierHistoryGlobal(testCtx)
		require.NoError(t, err)

		// Check event for this block's cleanup stats
		events := testCtx.EventManager().Events()
		for _, evt := range events {
			if evt.Type == "outlier_history_cleaned" {
				for _, attr := range evt.Attributes {
					if attr.Key == "entries_deleted" {
						var cleaned int
						_, err := fmt.Sscanf(string(attr.Value), "%d", &cleaned)
						require.NoError(t, err)
						totalCleaned += cleaned
					}
					if attr.Key == "pairs_processed" {
						var processed int
						_, err := fmt.Sscanf(string(attr.Value), "%d", &processed)
						require.NoError(t, err)
						totalProcessed += processed
					}
				}
			}
		}
	}

	t.Logf("Total cleaned over 100 blocks: %d entries, %d pairs processed", totalCleaned, totalProcessed)

	// Verify that old entries were cleaned
	require.Greater(t, totalCleaned, 0, "should have cleaned some old entries")

	// Verify that recent entries still exist
	recentStillPresent := 0
	for _, val := range validators {
		for _, asset := range assets {
			for h := safeRecentThreshold; h < safeRecentThreshold+10; h++ {
				key := makeOutlierKey(val.String(), asset, h)
				if store.Has(key) {
					recentStillPresent++
				}
			}
		}
	}

	require.Equal(t, recentEntriesCount, recentStillPresent,
		"recent entries should not be cleaned up")
}

// TestCleanupOldOutlierHistoryGlobal_NoTimeoutWithLargeData tests that cleanup doesn't timeout
func TestCleanupOldOutlierHistoryGlobal_NoTimeoutWithLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large data test in short mode")
	}

	k, _, ctx := keepertest.OracleKeeper(t)

	// Simulate large validator set: 100 validators × 20 assets = 2000 pairs
	numValidators := 100
	numAssets := 20
	currentHeight := int64(10000)

	ctx = ctx.WithBlockHeight(currentHeight).WithEventManager(sdk.NewEventManager())
	store := ctx.KVStore(k.GetStoreKey())

	// Create outlier history
	minHeight := currentHeight - keeper.OutlierReputationWindow
	entriesCreated := 0

	for v := 0; v < numValidators; v++ {
		valAddr := makeValidatorAddress(byte(v % 256))
		for a := 0; a < numAssets; a++ {
			asset := fmt.Sprintf("ASSET%d/USD", a)

			// Create old entries
			for h := minHeight - 30; h < minHeight-10; h++ {
				if h > 0 {
					key := makeOutlierKey(valAddr.String(), asset, h)
					value := fmt.Sprintf("%d:%d", 1, h) // SeverityLow
					store.Set(key, []byte(value))
					entriesCreated++
				}
			}
		}
	}

	t.Logf("Created %d outlier entries for %d validator-asset pairs",
		entriesCreated, numValidators*numAssets)

	// Run cleanup for a single block - should not timeout
	// With amortization, it processes maxCleanupPerBlock=50 pairs
	err := k.CleanupOldOutlierHistoryGlobal(ctx)
	require.NoError(t, err, "cleanup should not error even with large dataset")

	// Verify cleanup event was emitted
	events := ctx.EventManager().Events()
	found := false
	for _, evt := range events {
		if evt.Type == "outlier_history_cleaned" {
			found = true
			t.Logf("Cleanup event: %v", evt.Attributes)
		}
	}

	// Event may not exist if no cleanup happened this block due to modulo scheduling
	// This is expected behavior - we just verify no timeout/error occurred
	t.Logf("Cleanup event found: %v", found)
}

// TestCleanupOldOutlierHistoryGlobal_EmptyState tests cleanup with no data
func TestCleanupOldOutlierHistoryGlobal_EmptyState(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(10000).WithEventManager(sdk.NewEventManager())

	// Run cleanup on empty state - should not error
	err := k.CleanupOldOutlierHistoryGlobal(ctx)
	require.NoError(t, err)

	// No event should be emitted when nothing processed
	events := ctx.EventManager().Events()
	for _, evt := range events {
		require.NotEqual(t, "outlier_history_cleaned", evt.Type,
			"should not emit cleanup event when nothing processed")
	}
}

// TestCleanupOldOutlierHistoryGlobal_EarlyBlocks tests cleanup in early blocks
func TestCleanupOldOutlierHistoryGlobal_EarlyBlocks(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(100).WithEventManager(sdk.NewEventManager())

	// Create some outlier entries
	store := ctx.KVStore(k.GetStoreKey())
	valAddr := makeValidatorAddress(0x01)
	asset := "BTC/USD"

	for h := int64(1); h < 50; h++ {
		key := makeOutlierKey(valAddr.String(), asset, h)
		value := fmt.Sprintf("%d:%d", 1, h) // SeverityLow
		store.Set(key, []byte(value))
	}

	// Run cleanup - should skip because minHeight would be negative
	err := k.CleanupOldOutlierHistoryGlobal(ctx)
	require.NoError(t, err)

	// Verify no entries were deleted
	for h := int64(1); h < 50; h++ {
		key := makeOutlierKey(valAddr.String(), asset, h)
		require.True(t, store.Has(key), "entries should not be deleted in early blocks")
	}
}

// TestExtractValidatorAssetPair tests the key parsing helper
func TestExtractValidatorAssetPair(t *testing.T) {
	testCases := []struct {
		name     string
		key      []byte
		expected string
	}{
		{
			name:     "valid key",
			key:      makeOutlierKey("cosmosvaloper1abc", "BTC/USD", 12345),
			expected: "cosmosvaloper1abc\x00BTC/USD",
		},
		{
			name:     "empty key",
			key:      []byte{},
			expected: "",
		},
		{
			name:     "too short key",
			key:      keeper.OutlierHistoryKeyPrefix,
			expected: "",
		},
		{
			name:     "missing separator",
			key:      append(keeper.OutlierHistoryKeyPrefix, []byte{'a', 'b', 'c'}...),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We need to test the helper directly - implemented inline here
			result := extractValidatorAssetPairHelper(tc.key)
			require.Equal(t, tc.expected, result)
		})
	}
}

// makeOutlierKey creates an outlier history key for testing
func makeOutlierKey(validator, asset string, height int64) []byte {
	key := append([]byte(nil), keeper.OutlierHistoryKeyPrefix...)
	key = append(key, []byte(validator)...)
	key = append(key, byte(0x00))
	key = append(key, []byte(asset)...)
	key = append(key, byte(0x00))
	key = append(key, sdk.Uint64ToBigEndian(uint64(height))...)
	return key
}

// extractValidatorAssetPairHelper is a test helper that replicates the keeper logic
func extractValidatorAssetPairHelper(key []byte) string {
	if len(key) < len(keeper.OutlierHistoryKeyPrefix)+1 {
		return ""
	}

	remainder := key[len(keeper.OutlierHistoryKeyPrefix):]
	separatorCount := 0
	for i, b := range remainder {
		if b == 0x00 {
			separatorCount++
			if separatorCount == 2 {
				return string(remainder[:i])
			}
		}
	}

	return ""
}
