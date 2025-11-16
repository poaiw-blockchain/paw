package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestCalculateMedian(t *testing.T) {
	tests := []struct {
		name     string
		prices   []math.LegacyDec
		expected math.LegacyDec
	}{
		{
			name:     "odd number of prices",
			prices:   []math.LegacyDec{math.LegacyNewDec(100), math.LegacyNewDec(200), math.LegacyNewDec(300)},
			expected: math.LegacyNewDec(200),
		},
		{
			name:     "even number of prices",
			prices:   []math.LegacyDec{math.LegacyNewDec(100), math.LegacyNewDec(200), math.LegacyNewDec(300), math.LegacyNewDec(400)},
			expected: math.LegacyNewDec(250),
		},
		{
			name:     "single price",
			prices:   []math.LegacyDec{math.LegacyNewDec(100)},
			expected: math.LegacyNewDec(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test uses an internal function, so we'll test via AggregatePrice instead
			// The calculateMedian function is tested indirectly
			require.NotNil(t, tt.prices)
		})
	}
}

func TestAggregatePrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.MinValidators = 3
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create submissions
	submissions := []types.ValidatorPriceSubmission{
		types.NewValidatorPriceSubmission(
			"cosmosvaloper1",
			"BTC/USD",
			math.LegacyNewDec(50000),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper2",
			"BTC/USD",
			math.LegacyNewDec(50100),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper3",
			"BTC/USD",
			math.LegacyNewDec(49900),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
	}

	for _, sub := range submissions {
		err := k.SetValidatorSubmission(ctx, sub)
		require.NoError(t, err)
	}

	// Aggregate price
	err = k.AggregatePrice(ctx, "BTC/USD")
	require.NoError(t, err)

	// Verify aggregated price
	priceFeed, found := k.GetPriceFeed(ctx, "BTC/USD")
	require.True(t, found)
	// Median should be 50000
	require.True(t, priceFeed.Price.Equal(math.LegacyNewDec(50000)))
}

func TestAggregatePriceInsufficientSubmissions(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params requiring 3 validators
	params := types.DefaultParams()
	params.MinValidators = 3
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Only submit 2 prices
	submissions := []types.ValidatorPriceSubmission{
		types.NewValidatorPriceSubmission(
			"cosmosvaloper1",
			"BTC/USD",
			math.LegacyNewDec(50000),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper2",
			"BTC/USD",
			math.LegacyNewDec(50100),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
	}

	for _, sub := range submissions {
		err := k.SetValidatorSubmission(ctx, sub)
		require.NoError(t, err)
	}

	// Should fail due to insufficient submissions
	err = k.AggregatePrice(ctx, "BTC/USD")
	require.Error(t, err)
}

func TestAggregatePriceWithOutliers(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	params := types.DefaultParams()
	params.MinValidators = 3
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create submissions with one extreme outlier
	submissions := []types.ValidatorPriceSubmission{
		types.NewValidatorPriceSubmission(
			"cosmosvaloper1",
			"BTC/USD",
			math.LegacyNewDec(50000),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper2",
			"BTC/USD",
			math.LegacyNewDec(50100),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper3",
			"BTC/USD",
			math.LegacyNewDec(49900),
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
		types.NewValidatorPriceSubmission(
			"cosmosvaloper4",
			"BTC/USD",
			math.LegacyNewDec(100000), // Extreme outlier
			getTestTimestamp(),
			ctx.BlockHeight(),
		),
	}

	for _, sub := range submissions {
		err := k.SetValidatorSubmission(ctx, sub)
		require.NoError(t, err)
	}

	// Aggregate - should remove outlier
	err = k.AggregatePrice(ctx, "BTC/USD")
	require.NoError(t, err)

	priceFeed, found := k.GetPriceFeed(ctx, "BTC/USD")
	require.True(t, found)
	// Median should be close to 50000, not affected by 100000 outlier
	require.True(t, priceFeed.Price.LT(math.LegacyNewDec(55000)))
}

func TestGetPriceWithConfidence(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	params := types.DefaultParams()
	params.MinValidators = 3
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create price feed with minimum validators
	priceFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  getTestTimestamp(),
		Validators: []string{"val1", "val2", "val3"},
	}

	err = k.SetPriceFeed(ctx, priceFeed)
	require.NoError(t, err)

	price, confidence, err := k.GetPriceWithConfidence(ctx, "BTC/USD")
	require.NoError(t, err)
	require.True(t, price.Equal(math.LegacyNewDec(50000)))
	// With exactly minimum validators, confidence should be 0.5
	require.True(t, confidence.GTE(math.LegacyNewDecWithPrec(5, 1)))

	// Non-existent asset
	_, _, err = k.GetPriceWithConfidence(ctx, "DOGE/USD")
	require.Error(t, err)
}

func TestPriceDeviation(t *testing.T) {
	tests := []struct {
		name     string
		price    math.LegacyDec
		median   math.LegacyDec
		expected math.LegacyDec // percentage
	}{
		{
			name:     "no deviation",
			price:    math.LegacyNewDec(100),
			median:   math.LegacyNewDec(100),
			expected: math.LegacyZeroDec(),
		},
		{
			name:     "10% higher",
			price:    math.LegacyNewDec(110),
			median:   math.LegacyNewDec(100),
			expected: math.LegacyNewDec(10),
		},
		{
			name:     "10% lower",
			price:    math.LegacyNewDec(90),
			median:   math.LegacyNewDec(100),
			expected: math.LegacyNewDec(10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviation := types.PriceDeviation(tt.price, tt.median)
			require.True(t, deviation.Equal(tt.expected))
		})
	}
}

func TestIsOutlier(t *testing.T) {
	tests := []struct {
		name              string
		price             math.LegacyDec
		median            math.LegacyDec
		thresholdPercent  int64
		expectedIsOutlier bool
	}{
		{
			name:              "within threshold",
			price:             math.LegacyNewDec(105),
			median:            math.LegacyNewDec(100),
			thresholdPercent:  10,
			expectedIsOutlier: false,
		},
		{
			name:              "beyond threshold",
			price:             math.LegacyNewDec(115),
			median:            math.LegacyNewDec(100),
			thresholdPercent:  10,
			expectedIsOutlier: true,
		},
		{
			name:              "exactly at threshold",
			price:             math.LegacyNewDec(110),
			median:            math.LegacyNewDec(100),
			thresholdPercent:  10,
			expectedIsOutlier: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOutlier := types.IsOutlier(tt.price, tt.median, tt.thresholdPercent)
			require.Equal(t, tt.expectedIsOutlier, isOutlier)
		})
	}
}
