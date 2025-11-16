package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// getTestTimestamp returns a valid timestamp for testing
func getTestTimestamp() int64 {
	return time.Now().Unix()
}

func TestSetAndGetPriceFeed(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	priceFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  getTestTimestamp(),
		Validators: []string{"validator1", "validator2", "validator3"},
	}

	// Test setting price feed
	err := keeper.SetPriceFeed(ctx, priceFeed)
	require.NoError(t, err)

	// Test getting price feed
	retrieved, found := keeper.GetPriceFeed(ctx, "BTC/USD")
	require.True(t, found)
	require.Equal(t, priceFeed.Asset, retrieved.Asset)
	require.True(t, priceFeed.Price.Equal(retrieved.Price))
	require.Equal(t, priceFeed.Timestamp, retrieved.Timestamp)
	require.Equal(t, priceFeed.Validators, retrieved.Validators)
}

func TestGetAllPriceFeeds(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	priceFeeds := []types.PriceFeed{
		{
			Asset:      "BTC/USD",
			Price:      math.LegacyNewDec(50000),
			Timestamp:  getTestTimestamp(),
			Validators: []string{"validator1"},
		},
		{
			Asset:      "ETH/USD",
			Price:      math.LegacyNewDec(3000),
			Timestamp:  getTestTimestamp(),
			Validators: []string{"validator1"},
		},
	}

	for _, pf := range priceFeeds {
		err := keeper.SetPriceFeed(ctx, pf)
		require.NoError(t, err)
	}

	retrieved := keeper.GetAllPriceFeeds(ctx)
	require.Len(t, retrieved, 2)
}

func TestDeletePriceFeed(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	priceFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  getTestTimestamp(),
		Validators: []string{"validator1"},
	}

	err := keeper.SetPriceFeed(ctx, priceFeed)
	require.NoError(t, err)

	err = keeper.DeletePriceFeed(ctx, "BTC/USD")
	require.NoError(t, err)

	_, found := keeper.GetPriceFeed(ctx, "BTC/USD")
	require.False(t, found)
}

func TestIsPriceFeedStale(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	// Set params with 5 minute expiry
	params := types.DefaultParams()
	params.ExpiryDuration = 300 // 5 minutes
	err := keeper.SetParams(ctx, params)
	require.NoError(t, err)

	// Fresh price feed
	freshFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  getTestTimestamp(),
		Validators: []string{"validator1"},
	}

	require.False(t, keeper.IsPriceFeedStale(ctx, freshFeed))

	// Stale price feed (6 minutes old)
	staleFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  ctx.BlockTime().Add(-6 * time.Minute).Unix(),
		Validators: []string{"validator1"},
	}

	require.True(t, keeper.IsPriceFeedStale(ctx, staleFeed))
}

func TestGetValidPrice(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(t, err)

	// Set fresh price feed
	priceFeed := types.PriceFeed{
		Asset:      "BTC/USD",
		Price:      math.LegacyNewDec(50000),
		Timestamp:  getTestTimestamp(),
		Validators: []string{"validator1"},
	}

	err = keeper.SetPriceFeed(ctx, priceFeed)
	require.NoError(t, err)

	// Should get valid price
	price, found := keeper.GetValidPrice(ctx, "BTC/USD")
	require.True(t, found)
	require.True(t, price.Equal(math.LegacyNewDec(50000)))

	// Non-existent asset
	_, found = keeper.GetValidPrice(ctx, "DOGE/USD")
	require.False(t, found)
}

func TestValidatorSubmission(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	submission := types.NewValidatorPriceSubmission(
		"cosmosvaloper1xyz",
		"BTC/USD",
		math.LegacyNewDec(50000),
		getTestTimestamp(),
		ctx.BlockHeight(),
	)

	// Test setting submission
	err := keeper.SetValidatorSubmission(ctx, submission)
	require.NoError(t, err)

	// Test getting submission
	retrieved, found := keeper.GetValidatorSubmission(ctx, "BTC/USD", "cosmosvaloper1xyz")
	require.True(t, found)
	require.Equal(t, submission.Validator, retrieved.Validator)
	require.Equal(t, submission.Asset, retrieved.Asset)
	require.True(t, submission.Price.Equal(retrieved.Price))
}

func TestGetValidatorSubmissions(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

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
		err := keeper.SetValidatorSubmission(ctx, sub)
		require.NoError(t, err)
	}

	retrieved := keeper.GetValidatorSubmissions(ctx, "BTC/USD")
	require.Len(t, retrieved, 2)
}

func TestDeleteValidatorSubmission(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)

	submission := types.NewValidatorPriceSubmission(
		"cosmosvaloper1",
		"BTC/USD",
		math.LegacyNewDec(50000),
		getTestTimestamp(),
		ctx.BlockHeight(),
	)

	err := keeper.SetValidatorSubmission(ctx, submission)
	require.NoError(t, err)

	err = keeper.DeleteValidatorSubmission(ctx, "BTC/USD", "cosmosvaloper1")
	require.NoError(t, err)

	_, found := keeper.GetValidatorSubmission(ctx, "BTC/USD", "cosmosvaloper1")
	require.False(t, found)
}
