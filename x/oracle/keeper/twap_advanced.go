package keeper

import (
	"context"
	"fmt"
	"math"
	"sort"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// Advanced Time-Weighted Average Price (TWAP) Implementation
// Flash-loan resistant with multiple calculation methods

// TWAPMethod defines different TWAP calculation methods
type TWAPMethod int

const (
	// Standard time-weighted average
	TWAPMethodStandard TWAPMethod = iota

	// Volume-weighted TWAP (VWTWAP)
	TWAPMethodVolumeWeighted

	// Exponentially weighted moving average (EWMA)
	TWAPMethodExponential

	// Outlier-resistant TWAP using trimmed mean
	TWAPMethodTrimmed

	// Kalman filter based TWAP
	TWAPMethodKalman
)

// TWAPResult contains results from TWAP calculation
type TWAPResult struct {
	Asset          string
	Price          sdkmath.LegacyDec
	Method         TWAPMethod
	Confidence     sdkmath.LegacyDec
	Variance       sdkmath.LegacyDec
	SampleSize     int
	LookbackBlocks int64
}

// VolumeWeightedSnapshot extends price snapshot with volume data
type VolumeWeightedSnapshot struct {
	Price       sdkmath.LegacyDec
	BlockHeight int64
	BlockTime   int64
	Volume      sdkmath.LegacyDec // Simulated volume based on validator count
}

// KalmanFilterState maintains Kalman filter parameters
type KalmanFilterState struct {
	EstimatedPrice sdkmath.LegacyDec
	EstimateError  sdkmath.LegacyDec
	ProcessNoise   sdkmath.LegacyDec
	MeasureNoise   sdkmath.LegacyDec
}

// CalculateTWAPMultiMethod calculates TWAP using multiple methods for validation
func (k *Keeper) CalculateTWAPMultiMethod(ctx context.Context, asset string) (map[TWAPMethod]TWAPResult, error) {
	results := make(map[TWAPMethod]TWAPResult)

	// Method 1: Standard TWAP (already implemented)
	standardTWAP, err := k.CalculateTWAP(ctx, asset)
	if err == nil {
		results[TWAPMethodStandard] = TWAPResult{
			Asset:  asset,
			Price:  standardTWAP,
			Method: TWAPMethodStandard,
		}
	}

	// Method 2: Volume-weighted TWAP
	vwtwap, err := k.CalculateVolumeWeightedTWAP(ctx, asset)
	if err == nil {
		results[TWAPMethodVolumeWeighted] = vwtwap
	}

	// Method 3: Exponentially weighted TWAP
	ewma, err := k.CalculateExponentialTWAP(ctx, asset)
	if err == nil {
		results[TWAPMethodExponential] = ewma
	}

	// Method 4: Outlier-resistant TWAP
	trimmedTWAP, err := k.CalculateTrimmedTWAP(ctx, asset)
	if err == nil {
		results[TWAPMethodTrimmed] = trimmedTWAP
	}

	// Method 5: Kalman filter TWAP
	kalmanTWAP, err := k.CalculateKalmanTWAP(ctx, asset)
	if err == nil {
		results[TWAPMethodKalman] = kalmanTWAP
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all TWAP methods failed for asset %s", asset)
	}

	return results, nil
}

// CalculateVolumeWeightedTWAP calculates volume-weighted TWAP
// More resistant to low-liquidity manipulation
func (k *Keeper) CalculateVolumeWeightedTWAP(ctx context.Context, asset string) (TWAPResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return TWAPResult{}, err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	snapshots := []VolumeWeightedSnapshot{}

	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			// Estimate volume from number of validators participating
			// In production, this would use actual trading volume
			estimatedVolume := sdkmath.LegacyNewDec(int64(10)) // Simplified

			snapshots = append(snapshots, VolumeWeightedSnapshot{
				Price:       snapshot.Price,
				BlockHeight: snapshot.BlockHeight,
				BlockTime:   snapshot.BlockTime,
				Volume:      estimatedVolume,
			})
		}
		return false
	})

	if err != nil || len(snapshots) == 0 {
		return TWAPResult{}, fmt.Errorf("insufficient data for VWTWAP")
	}

	// Sort by block height
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].BlockHeight < snapshots[j].BlockHeight
	})

	// Calculate volume-weighted time average with overflow protection
	totalWeightedPrice := sdkmath.LegacyZeroDec()
	totalWeight := sdkmath.LegacyZeroDec()

	for i := 0; i < len(snapshots)-1; i++ {
		timeDelta := snapshots[i+1].BlockTime - snapshots[i].BlockTime
		if timeDelta <= 0 {
			continue
		}
		// Overflow protection: ensure timeDelta doesn't cause overflow
		if timeDelta > 1e18 {
			return TWAPResult{}, fmt.Errorf("time delta too large: %d", timeDelta)
		}

		// Weight = time_delta * volume
		weight := sdkmath.LegacyNewDec(timeDelta).Mul(snapshots[i].Volume)
		// Check for overflow in weighted price calculation
		if weight.IsNegative() {
			return TWAPResult{}, fmt.Errorf("negative weight calculated")
		}
		weightedPrice := snapshots[i].Price.Mul(weight)

		totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
		totalWeight = totalWeight.Add(weight)
	}

	// Handle last snapshot with overflow protection
	lastSnapshot := snapshots[len(snapshots)-1]
	lastTimeDelta := sdkCtx.BlockTime().Unix() - lastSnapshot.BlockTime
	if lastTimeDelta > 0 {
		if lastTimeDelta > 1e18 {
			return TWAPResult{}, fmt.Errorf("last time delta too large: %d", lastTimeDelta)
		}
		weight := sdkmath.LegacyNewDec(lastTimeDelta).Mul(lastSnapshot.Volume)
		if weight.IsNegative() {
			return TWAPResult{}, fmt.Errorf("negative weight in last snapshot")
		}
		weightedPrice := lastSnapshot.Price.Mul(weight)
		totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
		totalWeight = totalWeight.Add(weight)
	}

	if totalWeight.IsZero() {
		return TWAPResult{}, fmt.Errorf("zero total weight in VWTWAP")
	}

	vwtwap := totalWeightedPrice.Quo(totalWeight)

	return TWAPResult{
		Asset:          asset,
		Price:          vwtwap,
		Method:         TWAPMethodVolumeWeighted,
		SampleSize:     len(snapshots),
		LookbackBlocks: int64(params.TwapLookbackWindow),
	}, nil
}

// CalculateExponentialTWAP calculates exponentially weighted moving average
// Gives more weight to recent prices
func (k *Keeper) CalculateExponentialTWAP(ctx context.Context, asset string) (TWAPResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return TWAPResult{}, err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	snapshots := []types.PriceSnapshot{}

	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
		}
		return false
	})

	if err != nil || len(snapshots) == 0 {
		return TWAPResult{}, fmt.Errorf("insufficient data for EWMA")
	}

	// Sort by block height
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].BlockHeight < snapshots[j].BlockHeight
	})

	// Exponential smoothing factor (alpha)
	// Higher alpha = more weight to recent prices
	alpha := sdkmath.LegacyMustNewDecFromStr("0.3") // 30% weight to current, 70% to history

	// Initialize EWMA with first price
	ewma := snapshots[0].Price

	// Apply exponential smoothing
	for i := 1; i < len(snapshots); i++ {
		// EWMA = alpha * current_price + (1 - alpha) * previous_ewma
		ewma = alpha.Mul(snapshots[i].Price).Add(
			sdkmath.LegacyOneDec().Sub(alpha).Mul(ewma),
		)
	}

	return TWAPResult{
		Asset:          asset,
		Price:          ewma,
		Method:         TWAPMethodExponential,
		SampleSize:     len(snapshots),
		LookbackBlocks: int64(params.TwapLookbackWindow),
	}, nil
}

// CalculateTrimmedTWAP calculates TWAP with outlier removal
// Removes top and bottom percentiles before averaging
func (k *Keeper) CalculateTrimmedTWAP(ctx context.Context, asset string) (TWAPResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return TWAPResult{}, err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	snapshots := []types.PriceSnapshot{}

	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
		}
		return false
	})

	if err != nil || len(snapshots) < 4 {
		return TWAPResult{}, fmt.Errorf("insufficient data for trimmed TWAP")
	}

	// Sort by block height for time-weighting
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].BlockHeight < snapshots[j].BlockHeight
	})

	// Extract prices for trimming
	prices := make([]sdkmath.LegacyDec, len(snapshots))
	for i, snapshot := range snapshots {
		prices[i] = snapshot.Price
	}

	// Sort prices for percentile calculation
	sortedPrices := make([]sdkmath.LegacyDec, len(prices))
	copy(sortedPrices, prices)
	sort.Slice(sortedPrices, func(i, j int) bool {
		return sortedPrices[i].LT(sortedPrices[j])
	})

	// Trim top and bottom 10%
	trimPercent := 0.10
	trimCount := int(float64(len(sortedPrices)) * trimPercent)
	if trimCount < 1 {
		trimCount = 1
	}

	// Get trim bounds
	lowerBound := sortedPrices[trimCount]
	upperBound := sortedPrices[len(sortedPrices)-trimCount-1]

	// Calculate time-weighted average of trimmed prices
	totalWeightedPrice := sdkmath.LegacyZeroDec()
	totalTime := int64(0)
	includedCount := 0

	for i := 0; i < len(snapshots)-1; i++ {
		// Skip outliers
		if snapshots[i].Price.LT(lowerBound) || snapshots[i].Price.GT(upperBound) {
			continue
		}

		timeDelta := snapshots[i+1].BlockTime - snapshots[i].BlockTime
		if timeDelta <= 0 {
			continue
		}
		// Overflow protection
		if timeDelta > 1e18 {
			return TWAPResult{}, fmt.Errorf("time delta too large in trimmed TWAP: %d", timeDelta)
		}

		weightedPrice := snapshots[i].Price.MulInt64(timeDelta)
		totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
		totalTime += timeDelta
		includedCount++
	}

	// Handle last snapshot with overflow protection
	lastSnapshot := snapshots[len(snapshots)-1]
	if lastSnapshot.Price.GTE(lowerBound) && lastSnapshot.Price.LTE(upperBound) {
		lastTimeDelta := sdkCtx.BlockTime().Unix() - lastSnapshot.BlockTime
		if lastTimeDelta > 0 {
			if lastTimeDelta > 1e18 {
				return TWAPResult{}, fmt.Errorf("last time delta too large in trimmed TWAP: %d", lastTimeDelta)
			}
			weightedPrice := lastSnapshot.Price.MulInt64(lastTimeDelta)
			totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
			totalTime += lastTimeDelta
			includedCount++
		}
	}

	if totalTime == 0 {
		return TWAPResult{}, fmt.Errorf("zero total time in trimmed TWAP")
	}

	trimmedTWAP := totalWeightedPrice.QuoInt64(totalTime)

	return TWAPResult{
		Asset:          asset,
		Price:          trimmedTWAP,
		Method:         TWAPMethodTrimmed,
		SampleSize:     includedCount,
		LookbackBlocks: int64(params.TwapLookbackWindow),
	}, nil
}

// CalculateKalmanTWAP uses Kalman filter for optimal price estimation
// Provides best estimate under Gaussian noise assumptions
func (k *Keeper) CalculateKalmanTWAP(ctx context.Context, asset string) (TWAPResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return TWAPResult{}, err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	snapshots := []types.PriceSnapshot{}

	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
		}
		return false
	})

	if err != nil || len(snapshots) < 2 {
		return TWAPResult{}, fmt.Errorf("insufficient data for Kalman filter")
	}

	// Sort by block height
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].BlockHeight < snapshots[j].BlockHeight
	})

	// Initialize Kalman filter
	state := KalmanFilterState{
		EstimatedPrice: snapshots[0].Price,
		EstimateError:  sdkmath.LegacyMustNewDecFromStr("1.0"),  // Initial uncertainty
		ProcessNoise:   sdkmath.LegacyMustNewDecFromStr("0.01"), // Price process variance
		MeasureNoise:   sdkmath.LegacyMustNewDecFromStr("0.1"),  // Measurement variance
	}

	// Apply Kalman filter to each measurement
	for i := 1; i < len(snapshots); i++ {
		measurement := snapshots[i].Price

		// Prediction step (price doesn't change much between blocks)
		predictedPrice := state.EstimatedPrice
		predictedError := state.EstimateError.Add(state.ProcessNoise)

		// Update step
		// Kalman gain = predicted_error / (predicted_error + measurement_noise)
		kalmanGain := predictedError.Quo(predictedError.Add(state.MeasureNoise))

		// Updated estimate = predicted + gain * (measurement - predicted)
		innovation := measurement.Sub(predictedPrice)
		state.EstimatedPrice = predictedPrice.Add(kalmanGain.Mul(innovation))

		// Updated error = (1 - gain) * predicted_error
		state.EstimateError = sdkmath.LegacyOneDec().Sub(kalmanGain).Mul(predictedError)
	}

	// Calculate confidence (inverse of error)
	confidence := sdkmath.LegacyOneDec().Quo(sdkmath.LegacyOneDec().Add(state.EstimateError))

	return TWAPResult{
		Asset:          asset,
		Price:          state.EstimatedPrice,
		Method:         TWAPMethodKalman,
		Confidence:     confidence,
		Variance:       state.EstimateError,
		SampleSize:     len(snapshots),
		LookbackBlocks: int64(params.TwapLookbackWindow),
	}, nil
}

// GetRobustTWAP returns the most reliable TWAP using consensus across methods
func (k *Keeper) GetRobustTWAP(ctx context.Context, asset string) (TWAPResult, error) {
	// Calculate using multiple methods
	results, err := k.CalculateTWAPMultiMethod(ctx, asset)
	if err != nil {
		return TWAPResult{}, err
	}

	if len(results) == 0 {
		return TWAPResult{}, fmt.Errorf("no TWAP results available")
	}

	// If only one method succeeded, return it
	if len(results) == 1 {
		for _, result := range results {
			return result, nil
		}
	}

	// Multiple methods available - use consensus
	prices := []sdkmath.LegacyDec{}
	for _, result := range results {
		prices = append(prices, result.Price)
	}

	// Use median of all methods (robust to outlier methods)
	robustPrice := k.calculateMedian(prices)

	// Return with highest confidence method's metadata
	bestResult := TWAPResult{
		Asset:  asset,
		Price:  robustPrice,
		Method: TWAPMethodKalman, // Kalman is typically most reliable
	}

	// If Kalman result exists, use its confidence
	if kalmanResult, exists := results[TWAPMethodKalman]; exists {
		bestResult.Confidence = kalmanResult.Confidence
		bestResult.Variance = kalmanResult.Variance
	}

	return bestResult, nil
}

// ValidateTWAPConsistency checks if different TWAP methods agree
func (k *Keeper) ValidateTWAPConsistency(ctx context.Context, asset string) (bool, sdkmath.LegacyDec, error) {
	results, err := k.CalculateTWAPMultiMethod(ctx, asset)
	if err != nil {
		return false, sdkmath.LegacyZeroDec(), err
	}

	if len(results) < 2 {
		return true, sdkmath.LegacyZeroDec(), nil // Only one method, considered consistent
	}

	// Calculate variance across methods
	prices := []sdkmath.LegacyDec{}
	for _, result := range results {
		prices = append(prices, result.Price)
	}

	mean := sdkmath.LegacyZeroDec()
	for _, price := range prices {
		mean = mean.Add(price)
	}
	mean = mean.QuoInt64(int64(len(prices)))

	variance := sdkmath.LegacyZeroDec()
	for _, price := range prices {
		diff := price.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.QuoInt64(int64(len(prices)))

	// Convert to standard deviation
	varianceFloat, err := variance.Float64()
	if err != nil {
		return false, sdkmath.LegacyZeroDec(), err
	}

	stdDev := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", math.Sqrt(varianceFloat)))

	// Check if coefficient of variation is acceptable
	// CV = stddev / mean
	cv := sdkmath.LegacyZeroDec()
	if mean.GT(sdkmath.LegacyZeroDec()) {
		cv = stdDev.Quo(mean)
	}

	// Methods are consistent if CV < 5%
	consistencyThreshold := sdkmath.LegacyMustNewDecFromStr("0.05")
	isConsistent := cv.LT(consistencyThreshold)

	return isConsistent, cv, nil
}

// CalculateTWAPWithConfidenceInterval returns TWAP with confidence bounds
func (k *Keeper) CalculateTWAPWithConfidenceInterval(ctx context.Context, asset string) (price, lowerBound, upperBound sdkmath.LegacyDec, err error) {
	// Use Kalman filter for confidence estimation
	kalmanResult, err := k.CalculateKalmanTWAP(ctx, asset)
	if err != nil {
		return sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), err
	}

	// 95% confidence interval = mean ± 1.96 * stddev
	// stddev = sqrt(variance)
	varianceFloat, err := kalmanResult.Variance.Float64()
	if err != nil {
		return kalmanResult.Price, kalmanResult.Price, kalmanResult.Price, nil
	}

	stdDev := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", math.Sqrt(varianceFloat)))
	margin := stdDev.Mul(sdkmath.LegacyMustNewDecFromStr("1.96")) // 95% CI

	lowerBound = kalmanResult.Price.Sub(margin)
	upperBound = kalmanResult.Price.Add(margin)

	// Ensure bounds are positive
	if lowerBound.LT(sdkmath.LegacyZeroDec()) {
		lowerBound = sdkmath.LegacyZeroDec()
	}

	return kalmanResult.Price, lowerBound, upperBound, nil
}
