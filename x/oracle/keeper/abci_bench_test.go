package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// BenchmarkEndBlocker benchmarks the EndBlocker performance with varying validator and asset counts
func BenchmarkEndBlocker(b *testing.B) {
	testCases := []struct {
		name           string
		numValidators  int
		numAssets      int
		numOutlierDays int
	}{
		{"10vals_5assets", 10, 5, 30},
		{"50vals_10assets", 50, 10, 30},
		{"100vals_20assets", 100, 20, 30},
		{"200vals_30assets", 200, 30, 30},
		{"500vals_50assets", 500, 50, 30}, // Stress test for large networks
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			k, _, ctx := keepertest.OracleKeeper(b)
			ctx = ctx.WithBlockHeight(10000).WithEventManager(sdk.NewEventManager())

			// Setup: create validators, assets, and outlier history
			setupBenchmarkData(b, k, ctx, tc.numValidators, tc.numAssets, tc.numOutlierDays)

			// Reset timer to exclude setup time
			b.ResetTimer()

			// Run EndBlocker b.N times
			for i := 0; i < b.N; i++ {
				// Increment block height for each iteration
				ctx = ctx.WithBlockHeight(10000 + int64(i))
				err := k.EndBlocker(sdk.WrapSDKContext(ctx))
				require.NoError(b, err)
			}

			b.ReportMetric(float64(tc.numValidators), "validators")
			b.ReportMetric(float64(tc.numAssets), "assets")
			b.ReportMetric(float64(tc.numValidators*tc.numAssets), "pairs")
		})
	}
}

// BenchmarkCleanupOldOutlierHistoryGlobal benchmarks just the outlier cleanup function
func BenchmarkCleanupOldOutlierHistoryGlobal(b *testing.B) {
	testCases := []struct {
		name           string
		numValidators  int
		numAssets      int
		numOutlierDays int
	}{
		{"small_10v_5a", 10, 5, 30},
		{"medium_50v_10a", 50, 10, 30},
		{"large_100v_20a", 100, 20, 30},
		{"xlarge_200v_30a", 200, 30, 30},
		{"stress_500v_50a", 500, 50, 30},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			k, _, ctx := keepertest.OracleKeeper(b)

			// Setup baseline data
			currentHeight := int64(10000)
			ctx = ctx.WithBlockHeight(currentHeight)
			setupBenchmarkData(b, k, ctx, tc.numValidators, tc.numAssets, tc.numOutlierDays)

			b.ResetTimer()

			// Benchmark the cleanup function across multiple blocks
			for i := 0; i < b.N; i++ {
				blockHeight := currentHeight + int64(i)
				ctx = ctx.WithBlockHeight(blockHeight)
				err := k.CleanupOldOutlierHistoryGlobal(sdk.WrapSDKContext(ctx))
				require.NoError(b, err)
			}

			totalPairs := tc.numValidators * tc.numAssets
			b.ReportMetric(float64(totalPairs), "pairs")
		})
	}
}

// BenchmarkCleanupOldOutlierHistory_SinglePair benchmarks cleanup for a single validator-asset pair
func BenchmarkCleanupOldOutlierHistory_SinglePair(b *testing.B) {
	k, _, ctx := keepertest.OracleKeeper(b)
	ctx = ctx.WithBlockHeight(10000)

	validator := makeValidatorAddress(0x01)
	asset := "BTC/USD"

	// Create 1000 outlier history entries spanning 30 days
	for day := 0; day < 30; day++ {
		for entry := 0; entry < 33; entry++ { // ~33 entries per day = 1000 total
			height := int64(10000 - (day * 100) - entry)
			recordOutlierEntry(b, k, ctx, validator.String(), asset, height, 1) // SeverityLow
		}
	}

	minHeight := int64(7000) // Cleanup anything older than block 7000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := k.CleanupOldOutlierHistory(sdk.WrapSDKContext(ctx), validator.String(), asset, minHeight)
		require.NoError(b, err)
	}
}

// BenchmarkExtractValidatorAssetPair benchmarks the key parsing helper
func BenchmarkExtractValidatorAssetPair(b *testing.B) {
	// Create a realistic outlier history key
	validator := makeValidatorAddress(0x01).String()
	asset := "BTC/USD"
	height := int64(12345)

	key := append([]byte(nil), keeper.OutlierHistoryKeyPrefix...)
	key = append(key, []byte(validator)...)
	key = append(key, byte(0x00))
	key = append(key, []byte(asset)...)
	key = append(key, byte(0x00))
	key = append(key, sdk.Uint64ToBigEndian(uint64(height))...)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := extractValidatorAssetPairBenchHelper(key)
		if result == "" {
			b.Fatal("extractValidatorAssetPair returned empty string")
		}
	}
}

// setupBenchmarkData creates validators, assets, price data, and outlier history
func setupBenchmarkData(tb testing.TB, k *keeper.Keeper, ctx sdk.Context, numValidators, numAssets, numDays int) {
	tb.Helper()

	// Create validators
	validators := make([]sdk.ValAddress, numValidators)
	for i := 0; i < numValidators; i++ {
		valAddr := makeValidatorAddress(byte(i % 256))
		validators[i] = valAddr
		keepertest.RegisterTestOracle(tb, k, ctx, valAddr.String())
	}

	// Create assets with prices
	assets := make([]string, numAssets)
	for i := 0; i < numAssets; i++ {
		asset := fmt.Sprintf("ASSET%d/USD", i)
		assets[i] = asset

		// Set aggregated price
		price := types.Price{
			Asset:         asset,
			Price:         sdkmath.LegacyNewDec(100 + int64(i)),
			BlockHeight:   ctx.BlockHeight(),
			BlockTime:     ctx.BlockTime().Unix(),
			NumValidators: uint32(numValidators),
		}
		require.NoError(tb, k.SetPrice(sdk.WrapSDKContext(ctx), price))

		// Create validator price submissions
		for j := 0; j < numValidators; j++ {
			vp := types.ValidatorPrice{
				ValidatorAddr: validators[j].String(),
				Asset:         asset,
				Price:         sdkmath.LegacyNewDec(100 + int64(i) + int64(j)),
				BlockHeight:   ctx.BlockHeight(),
				VotingPower:   1,
			}
			require.NoError(tb, k.SetValidatorPrice(sdk.WrapSDKContext(ctx), vp))
		}
	}

	// Create outlier history for realistic scenario
	// Assume ~10% of submissions are outliers, spread across the retention window
	outlierRate := 0.1
	entriesPerDay := int(float64(numValidators*numAssets) * outlierRate / float64(numDays))
	if entriesPerDay < 1 {
		entriesPerDay = 1
	}

	currentHeight := ctx.BlockHeight()
	for day := 0; day < numDays; day++ {
		for entry := 0; entry < entriesPerDay; entry++ {
			// Distribute outliers across validators and assets
			valIdx := (day*entriesPerDay + entry) % numValidators
			assetIdx := (day*entriesPerDay + entry) % numAssets

			height := currentHeight - int64(day*100) - int64(entry)
			if height < 1 {
				height = 1
			}

			severity := 1 // SeverityLow
			if entry%10 == 0 {
				severity = 3 // SeverityHigh
			}

			recordOutlierEntry(tb, k, ctx, validators[valIdx].String(), assets[assetIdx], height, severity)
		}
	}
}

// recordOutlierEntry records an outlier history entry directly in storage
func recordOutlierEntry(tb testing.TB, k *keeper.Keeper, ctx sdk.Context, validator, asset string, height int64, severity int) {
	tb.Helper()

	// Use the keeper's internal method to record outlier history
	// We need to access the store directly since recordOutlierHistory is not exported
	store := ctx.KVStore(k.GetStoreKey())

	key := append([]byte(nil), keeper.OutlierHistoryKeyPrefix...)
	key = append(key, []byte(validator)...)
	key = append(key, byte(0x00))
	key = append(key, []byte(asset)...)
	key = append(key, byte(0x00))
	key = append(key, sdk.Uint64ToBigEndian(uint64(height))...)

	value := fmt.Sprintf("%d:%d", severity, height)
	store.Set(key, []byte(value))
}

// extractValidatorAssetPairBenchHelper wraps the function for benchmarking
// This is needed because the actual function is in the keeper package
func extractValidatorAssetPairBenchHelper(key []byte) string {
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
