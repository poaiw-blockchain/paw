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
	k, ctx := keepertest.OracleKeeper(b)

	// Register test oracle
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		price := math.LegacyNewDec(int64(100 + i%50))
		keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", price)
	}
}

// BenchmarkGetPrice benchmarks querying oracle prices
func BenchmarkGetPrice(b *testing.B) {
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Register oracle and submit price
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)
	keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", math.LegacyNewDec(100))

	// Aggregate price to make it available
	_ = k.AggregatePrices(sdk.WrapSDKContext(ctx))

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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators and submit prices
	validators := []string{
		sdk.ValAddress("validator1_____________").String(),
		sdk.ValAddress("validator2_____________").String(),
		sdk.ValAddress("validator3_____________").String(),
		sdk.ValAddress("validator4_____________").String(),
		sdk.ValAddress("validator5_____________").String(),
	}

	for _, val := range validators {
		keepertest.RegisterTestOracle(b, k, ctx, val)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Submit prices from all validators
		for j, val := range validators {
			price := math.LegacyNewDec(int64(95 + j*2)) // Prices: 95, 97, 99, 101, 103
			keepertest.SubmitTestPrice(b, k, ctx, val, "PAW/USD", price)
		}
		b.StartTimer()

		// Aggregate prices (calculates median)
		err := k.AggregatePrices(sdk.WrapSDKContext(ctx))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTWAPCalculation benchmarks TWAP calculation over price history
func BenchmarkTWAPCalculation(b *testing.B) {
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

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
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		price := math.LegacyNewDec(int64(100 + i%50))
		keepertest.SubmitTestPrice(b, k, ctx, validator, "PAW/USD", price)
		b.StartTimer()

		err := k.AggregatePrices(sdk.WrapSDKContext(ctx))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiSourceAggregation benchmarks aggregating prices from multiple sources
func BenchmarkMultiSourceAggregation(b *testing.B) {
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Register 10 validators
	validators := make([]string, 10)
	for i := 0; i < 10; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracle(b, k, ctx, validators[i])
	}

	// Setup: Multiple assets
	assets := []string{"PAW/USD", "PAW/BTC", "PAW/ETH", "ATOM/USD", "OSMO/USD"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Submit prices from all validators for all assets
		for _, val := range validators {
			for j, asset := range assets {
				price := math.LegacyNewDec(int64(100 + j*10 + i%20))
				keepertest.SubmitTestPrice(b, k, ctx, val, asset, price)
			}
		}
		b.StartTimer()

		// Aggregate all assets
		err := k.AggregatePrices(sdk.WrapSDKContext(ctx))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVolumeWeightedTWAP benchmarks volume-weighted TWAP calculation
func BenchmarkVolumeWeightedTWAP(b *testing.B) {
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

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
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

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
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history with some outliers
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

	// Create price snapshots over time
	baseTime := time.Now().Unix()
	for i := int64(0); i < 100; i++ {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		newCtx := sdkCtx.WithBlockHeight(i + 1).WithBlockTime(time.Unix(baseTime+i*6, 0))

		// Add some outliers
		price := math.LegacyNewDec(100)
		if i%20 == 0 {
			price = math.LegacyNewDec(200) // Outlier
		} else if i%20 == 10 {
			price = math.LegacyNewDec(50) // Outlier
		} else {
			price = math.LegacyNewDec(100 + i%5)
		}

		snapshot := types.PriceSnapshot{
			Asset:       "PAW/USD",
			Price:       price,
			BlockHeight: i + 1,
			BlockTime:   baseTime + i*6,
		}
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

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
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Create price history
	validator := sdk.ValAddress("validator1_____________").String()
	keepertest.RegisterTestOracle(b, k, ctx, validator)

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
		k.SetPriceSnapshot(newCtx, snapshot)
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
	k, ctx := keepertest.OracleKeeper(b)

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
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators
	validators := make([]string, 10)
	for i := 0; i < 10; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracle(b, k, ctx, validators[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Submit prices with some outliers
		for j, val := range validators {
			var price math.LegacyDec
			if j == 0 {
				price = math.LegacyNewDec(200) // Outlier
			} else if j == 9 {
				price = math.LegacyNewDec(50) // Outlier
			} else {
				price = math.LegacyNewDec(100 + int64(j))
			}
			keepertest.SubmitTestPrice(b, k, ctx, val, "PAW/USD", price)
		}
		b.StartTimer()

		// Aggregation includes outlier detection
		err := k.AggregatePrices(sdk.WrapSDKContext(ctx))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidatorPriceIteration benchmarks iterating over validator prices
func BenchmarkValidatorPriceIteration(b *testing.B) {
	k, ctx := keepertest.OracleKeeper(b)

	// Setup: Register multiple validators and submit prices
	validators := make([]string, 50)
	for i := 0; i < 50; i++ {
		valAddr := sdk.ValAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		validators[i] = valAddr.String()
		keepertest.RegisterTestOracle(b, k, ctx, validators[i])
		keepertest.SubmitTestPrice(b, k, ctx, validators[i], "PAW/USD", math.LegacyNewDec(100))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetValidatorPricesByAsset(ctx, "PAW/USD")
		if err != nil {
			b.Fatal(err)
		}
	}
}
