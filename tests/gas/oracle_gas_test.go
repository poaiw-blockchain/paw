package gas

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// Gas limits for Oracle operations
const (
	GasRegisterOracleMin  = 40000
	GasRegisterOracleMax  = 100000
	GasSubmitPriceMin     = 30000
	GasSubmitPriceMax     = 100000
	GasAggregatePricesMax = 3000000 // Variable, depends on validator count
	GasTWAPCalculationMin = 50000
	GasTWAPCalculationMax = 200000
)

func TestOracleGas_RegisterOracle(t *testing.T) {
	rawKeeper, _, ctx := keepertest.OracleKeeper(t)
	k := NewOracleGasKeeper(rawKeeper)

	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

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
	rawKeeper, _, ctx := keepertest.OracleKeeper(t)
	k := NewOracleGasKeeper(rawKeeper)

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
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

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

func TestOracleGas_AggregatePrices(t *testing.T) {
	rawKeeper, _, ctx := keepertest.OracleKeeper(t)
	k := NewOracleGasKeeper(rawKeeper)

	tests := []struct {
		name       string
		numOracles int
		maxGas     uint64
	}{
		{"7 oracles", 7, 300000},
		{"21 oracles", 21, 700000},
		{"50 oracles", 50, 1600000},
		{"100 oracles", 100, GasAggregatePricesMax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(tt.maxGas + 500000))

			// Register oracles
			oracles := make([]sdk.AccAddress, tt.numOracles)
			for i := 0; i < tt.numOracles; i++ {
				oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
				oracles[i] = oracle
				err := k.RegisterOracle(ctx, oracle.String())
				require.NoError(t, err)
			}

			asset := "BTC"
			basePrice := math.LegacyNewDec(50000)

			for i, oracle := range oracles {
				variance := math.LegacyNewDec(int64(i * 10))
				price := basePrice.Add(variance)
				err := k.SubmitPrice(ctx, oracle.String(), asset, price)
				require.NoError(t, err)
			}

			err := k.AggregatePrices(ctx)
			require.NoError(t, err)

			price, err := k.GetPrice(ctx, asset)
			require.NoError(t, err)
			require.True(t, price.Price.GT(math.LegacyZeroDec()))

			gasUsed := ctx.GasMeter().GasConsumed()
			require.Less(t, gasUsed, tt.maxGas,
				"AggregatePrices gas exceeds limit for %d oracles: used %d, max %d",
				tt.numOracles, gasUsed, tt.maxGas)

			t.Logf("AggregatePrices (%d oracles): %d gas, price: %s",
				tt.numOracles, gasUsed, price.Price.String())
		})
	}
}

func TestOracleGas_TWAPCalculation(t *testing.T) {
	rawKeeper, _, ctx := keepertest.OracleKeeper(t)
	k := NewOracleGasKeeper(rawKeeper)

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
		name     string
		lookback uint64
		maxGas   uint64
	}{
		{"5 block TWAP", 5, 100000},
		{"10 block TWAP", 10, 150000},
		{"20 block TWAP", 20, 200000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300000))

			twap, err := k.CalculateTWAP(ctx, asset)
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

func TestOracleGas_GasRegression(t *testing.T) {
	rawKeeper, _, ctx := keepertest.OracleKeeper(t)
	k := NewOracleGasKeeper(rawKeeper)

	// Baseline gas values
	baselines := map[string]uint64{
		"RegisterOracle":  60000,
		"SubmitPrice":     50000,
		"AggregatePrices": 250000, // For 7 oracles
	}

	tolerance := uint64(15000)

	t.Run("RegisterOracle baseline", func(t *testing.T) {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

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

		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

		err = k.SubmitPrice(ctx, oracle.String(), "BTC", math.LegacyNewDec(50000))
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["SubmitPrice"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"SubmitPrice gas changed from baseline %d to %d", baseline, gasUsed)
	})

	t.Run("AggregatePrices baseline", func(t *testing.T) {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

		asset := "BTC"
		basePrice := math.LegacyNewDec(50000)

		for i := 0; i < 7; i++ {
			oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i+10))
			err := k.RegisterOracle(ctx, oracle.String())
			require.NoError(t, err)
			err = k.SubmitPrice(ctx, oracle.String(), asset, basePrice.Add(math.LegacyNewDec(int64(i))))
			require.NoError(t, err)
		}

		err := k.AggregatePrices(ctx)
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["AggregatePrices"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"AggregatePrices gas changed from baseline %d to %d", baseline, gasUsed)
	})
}
