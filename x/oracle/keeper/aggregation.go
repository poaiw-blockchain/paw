package keeper

import (
	"fmt"
	"math"
	"sort"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// AggregatePrice aggregates validator price submissions for an asset
// It removes outliers, calculates the median, and updates the price feed
func (k Keeper) AggregatePrice(ctx sdk.Context, asset string) error {
	// Get all valid submissions
	submissions := k.GetValidSubmissions(ctx, asset)

	// Check minimum submissions
	params := k.GetParams(ctx)
	if len(submissions) < int(params.MinValidators) {
		return fmt.Errorf(
			"insufficient submissions: got %d, need %d",
			len(submissions),
			params.MinValidators,
		)
	}

	// Extract prices for aggregation
	prices := make([]sdkmath.LegacyDec, len(submissions))
	validatorAddrs := make([]string, len(submissions))

	for i, sub := range submissions {
		prices[i] = sub.Price
		validatorAddrs[i] = sub.Validator
	}

	// Remove outliers using standard deviation
	filteredPrices, filteredValidators := k.removeOutliers(ctx, prices, validatorAddrs)

	// Check if we still have enough submissions after filtering
	if len(filteredPrices) < int(params.MinValidators) {
		k.Logger(ctx).Warn(
			"Not enough submissions after outlier removal",
			"asset", asset,
			"original", len(prices),
			"filtered", len(filteredPrices),
			"required", params.MinValidators,
		)
		return fmt.Errorf(
			"insufficient valid submissions after outlier removal: got %d, need %d",
			len(filteredPrices),
			params.MinValidators,
		)
	}

	// Calculate median
	medianPrice := calculateMedian(filteredPrices)

	// Get timestamp - use block time if available, otherwise current time
	timestamp := ctx.BlockTime().Unix()
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}

	// Create aggregated price
	aggregatedPrice := types.NewAggregatedPrice(
		asset,
		medianPrice,
		filteredValidators,
		timestamp,
		ctx.BlockHeight(),
	)

	// Update price feed
	priceFeed := types.PriceFeed{
		Asset:      asset,
		Price:      medianPrice,
		Timestamp:  aggregatedPrice.Timestamp,
		Validators: filteredValidators,
	}

	if err := k.SetPriceFeed(ctx, priceFeed); err != nil {
		return fmt.Errorf("failed to set price feed: %w", err)
	}

	// Track accuracy for all validators who submitted
	k.trackValidatorAccuracy(ctx, asset, submissions, medianPrice)

	// Emit aggregation event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_aggregated",
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("median_price", medianPrice.String()),
			sdk.NewAttribute("submissions_total", fmt.Sprintf("%d", len(submissions))),
			sdk.NewAttribute("submissions_used", fmt.Sprintf("%d", len(filteredPrices))),
			sdk.NewAttribute("timestamp", fmt.Sprintf("%d", aggregatedPrice.Timestamp)),
		),
	)

	k.Logger(ctx).Info(
		"Price aggregated",
		"asset", asset,
		"median_price", medianPrice.String(),
		"submissions", len(filteredPrices),
	)

	return nil
}

// removeOutliers removes prices that are beyond 2 standard deviations from mean
func (k Keeper) removeOutliers(ctx sdk.Context, prices []sdkmath.LegacyDec, validators []string) ([]sdkmath.LegacyDec, []string) {
	if len(prices) <= 3 {
		// Don't remove outliers if we have 3 or fewer submissions
		return prices, validators
	}

	// Calculate mean
	mean := calculateMean(prices)

	// Calculate standard deviation
	stdDev := calculateStdDev(prices, mean)

	// If standard deviation is very small, all prices are similar, keep all
	threshold := sdkmath.LegacyNewDecWithPrec(1, 6) // 0.000001
	if stdDev.LT(threshold) {
		return prices, validators
	}

	// Filter outliers (beyond 2 standard deviations)
	var filteredPrices []sdkmath.LegacyDec
	var filteredValidators []string

	twoStdDev := stdDev.MulInt64(2)

	for i, price := range prices {
		deviation := price.Sub(mean).Abs()
		if deviation.LTE(twoStdDev) {
			filteredPrices = append(filteredPrices, price)
			filteredValidators = append(filteredValidators, validators[i])
		} else {
			k.Logger(ctx).Info(
				"Removed outlier price",
				"validator", validators[i],
				"price", price.String(),
				"mean", mean.String(),
				"deviation", deviation.String(),
				"threshold", twoStdDev.String(),
			)
		}
	}

	return filteredPrices, filteredValidators
}

// calculateMedian calculates the median of a slice of LegacyDec values
func calculateMedian(prices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	// Sort prices
	sortedPrices := make([]sdkmath.LegacyDec, len(prices))
	copy(sortedPrices, prices)

	sort.Slice(sortedPrices, func(i, j int) bool {
		return sortedPrices[i].LT(sortedPrices[j])
	})

	n := len(sortedPrices)
	if n%2 == 0 {
		// Even number of elements: average the two middle values
		mid1 := sortedPrices[n/2-1]
		mid2 := sortedPrices[n/2]
		return mid1.Add(mid2).QuoInt64(2)
	}

	// Odd number of elements: return middle value
	return sortedPrices[n/2]
}

// calculateMean calculates the arithmetic mean of a slice of LegacyDec values
func calculateMean(prices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	sum := sdkmath.LegacyZeroDec()
	for _, price := range prices {
		sum = sum.Add(price)
	}

	return sum.QuoInt64(int64(len(prices)))
}

// calculateStdDev calculates the standard deviation of prices
func calculateStdDev(prices []sdkmath.LegacyDec, mean sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	sumSquaredDiff := sdkmath.LegacyZeroDec()

	for _, price := range prices {
		diff := price.Sub(mean)
		squaredDiff := diff.Mul(diff)
		sumSquaredDiff = sumSquaredDiff.Add(squaredDiff)
	}

	variance := sumSquaredDiff.QuoInt64(int64(len(prices)))

	// Calculate square root
	// Since LegacyDec doesn't have a built-in sqrt, we approximate it
	return approximateSqrt(variance)
}

// approximateSqrt approximates the square root of a LegacyDec using Newton's method
func approximateSqrt(x sdkmath.LegacyDec) sdkmath.LegacyDec {
	if x.IsZero() {
		return sdkmath.LegacyZeroDec()
	}

	if x.IsNegative() {
		return sdkmath.LegacyZeroDec()
	}

	// Convert to float64 for sqrt calculation
	xFloat, err := x.Float64()
	if err != nil {
		return sdkmath.LegacyZeroDec()
	}

	sqrtFloat := math.Sqrt(xFloat)

	// Convert back to LegacyDec
	sqrtDec, err := sdkmath.LegacyNewDecFromStr(fmt.Sprintf("%.18f", sqrtFloat))
	if err != nil {
		return sdkmath.LegacyZeroDec()
	}

	return sqrtDec
}

// trackValidatorAccuracy tracks each validator's accuracy against the median
func (k Keeper) trackValidatorAccuracy(ctx sdk.Context, asset string, submissions []types.ValidatorPriceSubmission, medianPrice sdkmath.LegacyDec) {
	// Threshold for considering a submission accurate (10%)
	accuracyThreshold := int64(10)

	for _, submission := range submissions {
		accuracy := k.GetValidatorAccuracy(ctx, submission.Validator)

		// Check if submission is within threshold
		deviation := types.PriceDeviation(submission.Price, medianPrice)
		isAccurate := deviation.LTE(sdkmath.LegacyNewDec(accuracyThreshold))

		if isAccurate {
			accuracy.RecordAccurate()
		} else {
			accuracy.RecordInaccurate()

			// Emit event for inaccurate submission
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"inaccurate_price_submission",
					sdk.NewAttribute("validator", submission.Validator),
					sdk.NewAttribute("asset", asset),
					sdk.NewAttribute("submitted_price", submission.Price.String()),
					sdk.NewAttribute("median_price", medianPrice.String()),
					sdk.NewAttribute("deviation_percent", deviation.String()),
				),
			)
		}

		// Save updated accuracy
		_ = k.SetValidatorAccuracy(ctx, accuracy)
	}
}

// GetPriceWithConfidence returns a price along with a confidence metric
// based on the number of validators and their voting power
func (k Keeper) GetPriceWithConfidence(ctx sdk.Context, asset string) (sdkmath.LegacyDec, sdkmath.LegacyDec, error) {
	priceFeed, found := k.GetPriceFeed(ctx, asset)
	if !found {
		return sdkmath.LegacyDec{}, sdkmath.LegacyDec{}, fmt.Errorf("price feed not found for asset: %s", asset)
	}

	// Check if stale
	if k.IsPriceFeedStale(ctx, priceFeed) {
		return sdkmath.LegacyDec{}, sdkmath.LegacyDec{}, fmt.Errorf("price feed is stale for asset: %s", asset)
	}

	// Calculate confidence based on number of validators
	// More validators = higher confidence (up to 1.0)
	params := k.GetParams(ctx)
	validatorCount := sdkmath.LegacyNewDec(int64(len(priceFeed.Validators)))
	minValidators := sdkmath.LegacyNewDec(int64(params.MinValidators))

	// Confidence ranges from 0.5 (at minimum) to 1.0 (at 2x minimum or more)
	confidence := sdkmath.LegacyNewDecWithPrec(5, 1) // 0.5
	if validatorCount.GT(minValidators) {
		additionalConfidence := validatorCount.Sub(minValidators).Quo(minValidators)
		maxAdditional := sdkmath.LegacyNewDecWithPrec(5, 1) // 0.5
		if additionalConfidence.GT(maxAdditional) {
			additionalConfidence = maxAdditional
		}
		confidence = confidence.Add(additionalConfidence)
	}

	return priceFeed.Price, confidence, nil
}
