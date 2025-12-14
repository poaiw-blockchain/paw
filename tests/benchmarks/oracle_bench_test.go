package benchmarks

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// BenchmarkSubmitPrice benchmarks submitting oracle prices
func BenchmarkSubmitPrice(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Register test oracle
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Advance block height by 11 to avoid rate limits (max 10 submissions per 100 blocks)
		// Start from block 1000 to ensure we're not in the initial window
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		ctx = sdkCtx.WithBlockHeight(1000 + int64(i)*11)

		price := math.LegacyNewDec(int64(100 + i%50))
		keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", price)
	}
}

// BenchmarkGetPrice benchmarks querying oracle prices
func BenchmarkGetPrice(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Register oracle and submit price
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)
	keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", math.LegacyNewDec(100))

	// Aggregate price to make it available
	_ = k.AggregatePrices(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetPrice(ctx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAggregateMedian benchmarks calculating median price from multiple validators
func BenchmarkAggregateMedian(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators and submit prices
	validators := []string{
		sdk.ValAddress("validator1_____________").String(),
		sdk.ValAddress("validator2_____________").String(),
		sdk.ValAddress("validator3_____________").String(),
		sdk.ValAddress("validator4_____________").String(),
		sdk.ValAddress("validator5_____________").String(),
	}

	for _, val := range validators {
		keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, val)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Advance block height by 11 to avoid rate limits (max 10 submissions per 100 blocks)
		// Start from block 1000 to ensure we're not in the initial window
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		ctx = sdkCtx.WithBlockHeight(1000 + int64(i)*11)

		// Submit prices from all validators
		for j, val := range validators {
			price := math.LegacyNewDec(int64(95 + j*2)) // Prices: 95, 97, 99, 101, 103
			keepertest.SubmitTestPrice(b, k, ctx, val, "PAW/USD", price)
		}
		b.StartTimer()

		// Aggregate prices (calculates median)
		err := k.AggregatePrices(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTWAPCalculation benchmarks TWAP calculation over price history
func BenchmarkTWAPCalculation(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		price := math.LegacyNewDec(100 + i%10)
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateTWAP(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPriceUpdate benchmarks updating aggregated oracle prices
func BenchmarkPriceUpdate(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Advance block height by 11 to avoid rate limits (max 10 submissions per 100 blocks)
		// Start from block 1000 to ensure we're not in the initial window
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		ctx = sdkCtx.WithBlockHeight(1000 + int64(i)*11)

		price := math.LegacyNewDec(int64(100 + i%50))
		keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", price)
		b.StartTimer()

		err := k.AggregatePrices(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiSourceAggregation benchmarks aggregating prices from multiple sources
func BenchmarkMultiSourceAggregation(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Register 10 validators
	validators := make([]string, 10)
	for i := 0; i < 10; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validators[i])
	}

	// Setup: Multiple assets
	assets := []string{"PAW/USD", "PAW/BTC", "PAW/ETH", "ATOM/USD", "OSMO/USD"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Advance block height by 60 to avoid rate limits
		// (10 validators × 5 assets = 50 submissions, need 50 * 11 = 550 blocks spacing)
		// Start from block 10000 to ensure we're not in the initial window
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		ctx = sdkCtx.WithBlockHeight(10000 + int64(i)*600)

		// Submit prices from all validators for all assets
		for _, val := range validators {
			for j, asset := range assets {
				price := math.LegacyNewDec(int64(100 + j*10 + i%20))
				keepertest.SubmitTestPrice(b, k, ctx, val, asset, price)
			}
		}
		b.StartTimer()

		// Aggregate all assets
		err := k.AggregatePrices(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVolumeWeightedTWAP benchmarks volume-weighted TWAP calculation
func BenchmarkVolumeWeightedTWAP(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		price := math.LegacyNewDec(100 + i%10)
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateVolumeWeightedTWAP(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExponentialTWAP benchmarks exponentially weighted TWAP calculation
func BenchmarkExponentialTWAP(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		price := math.LegacyNewDec(100 + i%10)
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateExponentialTWAP(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTrimmedTWAP benchmarks outlier-resistant trimmed TWAP calculation
func BenchmarkTrimmedTWAP(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history with some outliers
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		// Add some outliers
		price := math.LegacyNewDec(100 + i%5)
		if i%20 == 0 {
			price = math.LegacyNewDec(200) // Outlier
		} else if i%20 == 10 {
			price = math.LegacyNewDec(50) // Outlier
		}

		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateTrimmedTWAP(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkKalmanFilterTWAP benchmarks Kalman filter-based TWAP calculation
func BenchmarkKalmanFilterTWAP(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		price := math.LegacyNewDec(100 + i%10)
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateKalmanTWAP(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiMethodTWAP benchmarks calculating TWAP using all methods
func BenchmarkMultiMethodTWAP(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		price := math.LegacyNewDec(100 + i%10)
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		if err := k.SetPriceSnapshot(newCtx, snapshot); err != nil {
			b.Fatal(err)
		}
	}

	// Use context at block 100
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	newCtx := sdkCtx.WithBlockHeight(100).WithBlockTime(time.Unix(baseTime+600, 0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateTWAPMultiMethod(newCtx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPriceSnapshotStorage benchmarks storing price snapshots
func BenchmarkPriceSnapshotStorage(b *testing.B) {
	k, _, ctx := keepertest.OracleKeeper(b)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now().Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       math.LegacyNewDec(100 + int64(i%50)),
			BlockHeight: int64(i) + 1,
			BlockTime:   baseTime + int64(i)*6,
		}
		err := k.SetPriceSnapshot(sdkCtx, snapshot)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOutlierDetection benchmarks outlier detection in price aggregation
func BenchmarkOutlierDetection(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators
	validators := make([]string, 10)
	for i := 0; i < 10; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validators[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Advance block height by 110 to avoid rate limits (10 validators submitting)
		// Start from block 10000 to ensure we're not in the initial window
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		baseHeight := 10000 + int64(i)*110

		// Submit prices with some outliers (but not extreme enough to trigger slashing)
		// Each validator submits at a different block height to avoid flash loan resistance
		for j, val := range validators {
			var price math.LegacyDec
			if j == 0 {
				price = math.LegacyNewDec(110) // Mild outlier
			} else if j == 9 {
				price = math.LegacyNewDec(90) // Mild outlier
			} else {
				price = math.LegacyNewDec(100 + int64(j))
			}
			// Each validator gets a unique block height
			blockCtx := sdkCtx.WithBlockHeight(baseHeight + int64(j))
			keepertest.SubmitTestPrice(b, k, blockCtx, val, "PAW/USD", price)
		}
		b.StartTimer()

		// Aggregation includes outlier detection
		ctx = sdkCtx.WithBlockHeight(baseHeight + 10)
		err := k.AggregatePrices(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidatorPriceIteration benchmarks iterating over validator prices
func BenchmarkValidatorPriceIteration(b *testing.B) {
	k, sk, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators and submit prices
	// Advance block height for each submission to avoid flash loan resistance
	validators := make([]string, 50)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for i := 0; i < 50; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracleWithKeeper(b, sk, ctx, validators[i])

		// Advance block height by 11 for each submission to avoid rate limits
		blockCtx := sdkCtx.WithBlockHeight(1000 + int64(i)*11)
		keepertest.SubmitTestPrice(b, k, blockCtx, validators[i], "PAW/USD", math.LegacyNewDec(100))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetValidatorPricesByAsset(ctx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}
