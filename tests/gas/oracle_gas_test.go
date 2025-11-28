package gas

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Gas limits for Oracle operations
const (
	GasRegisterOracleMin      = 40000
	GasRegisterOracleMax      = 100000
	GasSubmitPriceMin         = 30000
	GasSubmitPriceMax         = 100000
	GasAggregateVotesMax      = 3000000  // Variable, depends on validator count
	GasOutlierDetectionMin    = 200000
	GasOutlierDetectionMax    = 1000000
	GasTWAPCalculationMin     = 50000
	GasTWAPCalculationMax     = 200000
	GasVolatilityCalcMin      = 100000
	GasVolatilityCalcMax      = 500000
	GasSlashOracleMin         = 80000
	GasSlashOracleMax         = 200000
)

func TestOracleGas_RegisterOracle(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	ctx = ctx.WithGasMeter(sdk.NewGasMeter(200000))

	oracle := sdk.AccAddress("oracle1_____________")

	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasRegisterOracleMax),
		"RegisterOracle should use <%d gas, used %d", GasRegisterOracleMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasRegisterOracleMin),
		"RegisterOracle should use >%d gas, used %d", GasRegisterOracleMin, gasUsed)

	t.Logf("RegisterOracle gas usage: %d", gasUsed)
}

func TestOracleGas_SubmitPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register oracle first
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	tests := []struct {
		name   string
		asset  string
		price  math.LegacyDec
		maxGas uint64
	}{
		{"BTC price", "BTC", math.LegacyNewDec(50000), 70000},
		{"ETH price", "ETH", math.LegacyNewDec(3000), 70000},
		{"ATOM price", "ATOM", math.LegacyNewDec(10), 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(200000))

			err := k.SubmitPrice(ctx, oracle.String(), tt.asset, tt.price)
			require.NoError(t, err)

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"SubmitPrice should use <%d gas, used %d", tt.maxGas, gasUsed)
			require.Greater(t, gasUsed, uint64(GasSubmitPriceMin),
				"SubmitPrice should use >%d gas, used %d", GasSubmitPriceMin, gasUsed)

			t.Logf("SubmitPrice (%s): %d gas", tt.name, gasUsed)
		})
	}
}

func TestOracleGas_AggregateVotes(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name           string
		numOracles     int
		maxGas         uint64
		perOracleGas   uint64
	}{
		{"7 oracles", 7, 300000, 43000},
		{"21 oracles", 21, 700000, 34000},
		{"50 oracles", 50, 1600000, 32000},
		{"100 oracles", 100, 3000000, 30000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register oracles
			oracles := make([]sdk.AccAddress, tt.numOracles)
			for i := 0; i < tt.numOracles; i++ {
				oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
				oracles[i] = oracle
				err := k.RegisterOracle(ctx, oracle.String())
				require.NoError(t, err)
			}

			// Submit prices from all oracles
			asset := "BTC"
			basePrice := math.LegacyNewDec(50000)

			for i, oracle := range oracles {
				// Add slight variance to prices
				variance := math.LegacyNewDec(int64(i * 10))
				price := basePrice.Add(variance)

				err := k.SubmitPrice(ctx, oracle.String(), asset, price)
				require.NoError(t, err)
			}

			// Aggregate votes (expensive operation)
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(tt.maxGas + 500000))

			aggregatedPrice, err := k.AggregateVotes(ctx, asset)
			require.NoError(t, err)
			require.True(t, aggregatedPrice.GT(math.LegacyZeroDec()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"AggregateVotes gas exceeds limit for %d oracles: used %d, max %d",
				tt.numOracles, gasUsed, tt.maxGas)

			// Gas should scale approximately linearly with oracle count
			gasPerOracle := gasUsed / uint64(tt.numOracles)
			require.Less(t, gasPerOracle, tt.perOracleGas,
				"Per-oracle gas cost too high: %d gas/oracle (limit: %d)",
				gasPerOracle, tt.perOracleGas)

			t.Logf("AggregateVotes (%d oracles): %d gas total, %d gas/oracle, price: %s",
				tt.numOracles, gasUsed, gasPerOracle, aggregatedPrice.String())
		})
	}
}

func TestOracleGas_OutlierDetection(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name       string
		numOracles int
		maxGas     uint64
	}{
		{"10 oracles", 10, 400000},
		{"25 oracles", 25, 600000},
		{"50 oracles", 50, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register oracles and submit prices
			oracles := make([]sdk.AccAddress, tt.numOracles)
			asset := "ETH"
			basePrice := math.LegacyNewDec(3000)

			for i := 0; i < tt.numOracles; i++ {
				oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
				oracles[i] = oracle
				err := k.RegisterOracle(ctx, oracle.String())
				require.NoError(t, err)

				// Most oracles report similar prices
				variance := math.LegacyNewDec(int64(i * 5))
				price := basePrice.Add(variance)

				// Add one outlier
				if i == tt.numOracles-1 {
					price = basePrice.Mul(math.LegacyNewDec(2)) // 2x price (outlier)
				}

				err = k.SubmitPrice(ctx, oracle.String(), asset, price)
				require.NoError(t, err)
			}

			// Detect outliers (expensive statistical computation)
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(tt.maxGas + 200000))

			outliers, err := k.DetectOutliers(ctx, asset)
			require.NoError(t, err)
			require.NotEmpty(t, outliers, "Should detect at least one outlier")

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"OutlierDetection too expensive: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasOutlierDetectionMin),
				"OutlierDetection gas too low: used %d, min %d", gasUsed, GasOutlierDetectionMin)

			t.Logf("OutlierDetection (%d oracles): %d gas, found %d outliers",
				tt.numOracles, gasUsed, len(outliers))
		})
	}
}

func TestOracleGas_TWAPCalculation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Setup: Register oracle and submit prices at different times
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	asset := "BTC"

	// Submit prices over time
	prices := []math.LegacyDec{
		math.LegacyNewDec(50000),
		math.LegacyNewDec(50500),
		math.LegacyNewDec(51000),
		math.LegacyNewDec(50800),
		math.LegacyNewDec(50200),
	}

	for i, price := range prices {
		// Advance block height to simulate time passing
		ctx = ctx.WithBlockHeight(int64(i + 1))

		err := k.SubmitPrice(ctx, oracle.String(), asset, price)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		lookback  uint64
		maxGas    uint64
	}{
		{"5 block TWAP", 5, 100000},
		{"10 block TWAP", 10, 150000},
		{"20 block TWAP", 20, 200000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(300000))

			twap, err := k.CalculateTWAP(ctx, asset, tt.lookback)
			require.NoError(t, err)
			require.True(t, twap.GT(math.LegacyZeroDec()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"TWAP calculation too expensive: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasTWAPCalculationMin),
				"TWAP calculation gas too low: used %d, min %d", gasUsed, GasTWAPCalculationMin)

			t.Logf("TWAP (%d blocks): %d gas, TWAP: %s", tt.lookback, gasUsed, twap.String())
		})
	}
}

func TestOracleGas_VolatilityCalculation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Setup: Register oracle and submit prices
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	asset := "ETH"

	// Submit prices with some volatility
	prices := []math.LegacyDec{
		math.LegacyNewDec(3000),
		math.LegacyNewDec(3100),
		math.LegacyNewDec(2950),
		math.LegacyNewDec(3050),
		math.LegacyNewDec(3200),
		math.LegacyNewDec(2900),
		math.LegacyNewDec(3150),
		math.LegacyNewDec(3000),
		math.LegacyNewDec(3250),
		math.LegacyNewDec(2850),
	}

	for i, price := range prices {
		ctx = ctx.WithBlockHeight(int64(i + 1))
		err := k.SubmitPrice(ctx, oracle.String(), asset, price)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		window    uint64
		maxGas    uint64
	}{
		{"5 block window", 5, 250000},
		{"10 block window", 10, 500000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(600000))

			volatility, err := k.CalculateVolatility(ctx, asset, tt.window)
			require.NoError(t, err)
			require.True(t, volatility.GT(math.LegacyZeroDec()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"Volatility calculation too expensive: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasVolatilityCalcMin),
				"Volatility calculation gas too low: used %d, min %d", gasUsed, GasVolatilityCalcMin)

			t.Logf("Volatility (%d blocks): %d gas, volatility: %s%%",
				tt.window, gasUsed, volatility.Mul(math.LegacyNewDec(100)).String())
		})
	}
}

func TestOracleGas_SlashOracle(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register oracle
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	ctx = ctx.WithGasMeter(sdk.NewGasMeter(300000))

	// Slash oracle for misbehavior
	slashAmount := math.LegacyNewDec(100) // 1% slash
	err = k.SlashOracle(ctx, oracle.String(), slashAmount, "outlier_submission")
	require.NoError(t, err)

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasSlashOracleMax),
		"SlashOracle should use <%d gas, used %d", GasSlashOracleMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasSlashOracleMin),
		"SlashOracle should use >%d gas, used %d", GasSlashOracleMin, gasUsed)

	t.Logf("SlashOracle gas usage: %d", gasUsed)
}

func TestOracleGas_MedianCalculation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name       string
		numOracles int
		maxGas     uint64
	}{
		{"7 oracles", 7, 80000},
		{"21 oracles", 21, 150000},
		{"50 oracles", 50, 300000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register oracles and submit prices
			asset := "ATOM"
			basePrice := math.LegacyNewDec(10)

			for i := 0; i < tt.numOracles; i++ {
				oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
				err := k.RegisterOracle(ctx, oracle.String())
				require.NoError(t, err)

				price := basePrice.Add(math.LegacyNewDec(int64(i)))
				err = k.SubmitPrice(ctx, oracle.String(), asset, price)
				require.NoError(t, err)
			}

			ctx = ctx.WithGasMeter(sdk.NewGasMeter(500000))

			// Calculate median price
			median, err := k.CalculateMedianPrice(ctx, asset)
			require.NoError(t, err)
			require.True(t, median.GT(math.LegacyZeroDec()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"Median calculation too expensive: used %d, max %d", gasUsed, tt.maxGas)

			gasPerOracle := gasUsed / uint64(tt.numOracles)
			t.Logf("Median (%d oracles): %d gas total, %d gas/oracle, median: %s",
				tt.numOracles, gasUsed, gasPerOracle, median.String())
		})
	}
}

func TestOracleGas_BatchPriceSubmission(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	tests := []struct {
		name       string
		numAssets  int
		maxGas     uint64
	}{
		{"5 assets", 5, 200000},
		{"10 assets", 10, 350000},
		{"20 assets", 20, 650000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(tt.maxGas + 100000))

			// Submit prices for multiple assets in batch
			for i := 0; i < tt.numAssets; i++ {
				asset := fmt.Sprintf("ASSET%d", i)
				price := math.LegacyNewDec(int64(1000 + i*100))

				err := k.SubmitPrice(ctx, oracle.String(), asset, price)
				require.NoError(t, err)
			}

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"Batch submission too expensive: used %d, max %d", gasUsed, tt.maxGas)

			gasPerAsset := gasUsed / uint64(tt.numAssets)
			t.Logf("Batch (%d assets): %d gas total, %d gas/asset", tt.numAssets, gasUsed, gasPerAsset)

			// Gas per asset should be relatively consistent
			require.Less(t, gasPerAsset, uint64(40000),
				"Gas per asset too high: %d", gasPerAsset)
		})
	}
}

func TestOracleGas_PriceAgeCheck(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register oracle and submit price
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	require.NoError(t, err)

	asset := "BTC"
	price := math.LegacyNewDec(50000)
	err = k.SubmitPrice(ctx, oracle.String(), asset, price)
	require.NoError(t, err)

	ctx = ctx.WithGasMeter(sdk.NewGasMeter(50000))

	// Check if price is fresh
	isFresh, err := k.IsPriceFresh(ctx, asset, 100) // 100 blocks
	require.NoError(t, err)
	require.True(t, isFresh)

	gasUsed := ctx.GasMeter().GasConsumed()

	// Age check should be cheap (just reading state)
	require.Less(t, gasUsed, uint64(10000),
		"Price age check should be cheap: used %d gas", gasUsed)

	t.Logf("Price age check gas: %d", gasUsed)
}

func TestOracleGas_GasRegression(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Baseline gas values
	baselines := map[string]uint64{
		"RegisterOracle": 60000,
		"SubmitPrice":    50000,
		"AggregateVotes": 250000, // For 7 oracles
	}

	tolerance := uint64(15000)

	t.Run("RegisterOracle baseline", func(t *testing.T) {
		ctx = ctx.WithGasMeter(sdk.NewGasMeter(200000))

		oracle := sdk.AccAddress("oracle1_____________")
		err := k.RegisterOracle(ctx, oracle.String())
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["RegisterOracle"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"RegisterOracle gas changed from baseline %d to %d", baseline, gasUsed)
	})

	t.Run("SubmitPrice baseline", func(t *testing.T) {
		oracle := sdk.AccAddress("oracle2_____________")
		err := k.RegisterOracle(ctx, oracle.String())
		require.NoError(t, err)

		ctx = ctx.WithGasMeter(sdk.NewGasMeter(200000))

		err = k.SubmitPrice(ctx, oracle.String(), "BTC", math.LegacyNewDec(50000))
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["SubmitPrice"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"SubmitPrice gas changed from baseline %d to %d", baseline, gasUsed)
	})
}
