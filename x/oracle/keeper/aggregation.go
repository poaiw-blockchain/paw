package keeper

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

type OutlierSeverity int

const (
	SeverityNone OutlierSeverity = iota
	SeverityLow
	SeverityModerate
	SeverityHigh
	SeverityExtreme

	// maxSnapshotsForVolatility caps the number of snapshots processed during volatility calculation
	// to prevent DoS via state bloat. This matches the limit used for TWAP calculations.
	maxSnapshotsForVolatility = 1000
)

type OutlierDetectionResult struct {
	ValidatorAddr string
	Price         sdkmath.LegacyDec
	Severity      OutlierSeverity
	Deviation     sdkmath.LegacyDec
	Reason        string
}

type FilteredPriceData struct {
	ValidPrices      []types.ValidatorPrice
	FilteredOutliers []OutlierDetectionResult
	Median           sdkmath.LegacyDec
	MAD              sdkmath.LegacyDec
	IQR              sdkmath.LegacyDec
}

// AggregateAssetPrice aggregates validator price submissions with institutional-grade outlier detection
func (k Keeper) AggregateAssetPrice(ctx context.Context, asset string) error {
	start := time.Now()
	defer func() {
		if k.metrics != nil {
			k.metrics.AggregationLatency.Observe(time.Since(start).Seconds())
		}
	}()

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Gas metering: Base price aggregation cost
	sdkCtx.GasMeter().ConsumeGas(30000, "oracle_aggregate_base")

	validatorPrices, err := k.GetValidatorPricesByAsset(ctx, asset)
	if err != nil {
		return err
	}

	if len(validatorPrices) == 0 {
		return fmt.Errorf("no price submissions for asset: %s", asset)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	sdkCtx.GasMeter().ConsumeGas(5000, "oracle_aggregate_get_prices")

	totalVotingPower, validPrices, err := k.calculateVotingPower(ctx, validatorPrices)
	if err != nil {
		return err
	}
	sdkCtx.GasMeter().ConsumeGas(uint64(len(validPrices)*1000), "oracle_aggregate_voting_power")

	if len(validPrices) == 0 {
		return fmt.Errorf("no valid price submissions for asset: %s", asset)
	}

	submittedVotingPower := int64(0)
	for _, vp := range validPrices {
		submittedVotingPower += vp.VotingPower
	}

	votePercentage := sdkmath.LegacyNewDec(submittedVotingPower).Quo(sdkmath.LegacyNewDec(totalVotingPower))
	if votePercentage.LT(params.VoteThreshold) {
		return fmt.Errorf("insufficient voting power: %s < %s", votePercentage.String(), params.VoteThreshold.String())
	}

	// Multi-stage statistical outlier detection
	sdkCtx.GasMeter().ConsumeGas(uint64(len(validPrices)*2000), "oracle_aggregate_outlier_detection")
	filteredData, err := k.detectAndFilterOutliers(ctx, asset, validPrices)
	if err != nil {
		return err
	}

	if len(filteredData.ValidPrices) == 0 {
		return fmt.Errorf("all prices filtered as outliers for asset: %s", asset)
	}

	// Slash validators with detected outliers
	for _, outlier := range filteredData.FilteredOutliers {
		if err := k.handleOutlierSlashing(ctx, asset, outlier); err != nil {
			sdkCtx.Logger().Error("failed to slash outlier validator",
				"validator", outlier.ValidatorAddr,
				"asset", asset,
				"severity", outlier.Severity,
				"error", err.Error(),
			)
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeOracleOutlier,
				sdk.NewAttribute(types.AttributeKeyValidator, outlier.ValidatorAddr),
				sdk.NewAttribute(types.AttributeKeyAsset, asset),
				sdk.NewAttribute(types.AttributeKeyPrice, outlier.Price.String()),
				sdk.NewAttribute(types.AttributeKeySeverity, fmt.Sprintf("%d", outlier.Severity)),
				sdk.NewAttribute(types.AttributeKeyDeviation, outlier.Deviation.String()),
				sdk.NewAttribute(types.AttributeKeyReason, outlier.Reason),
				sdk.NewAttribute(types.AttributeKeyMedian, filteredData.Median.String()),
				sdk.NewAttribute(types.AttributeKeyMAD, filteredData.MAD.String()),
			),
		)
	}

	// Calculate weighted median from filtered prices
	sdkCtx.GasMeter().ConsumeGas(uint64(len(filteredData.ValidPrices)*500), "oracle_aggregate_median")
	aggregatedPrice, err := k.calculateWeightedMedian(filteredData.ValidPrices)
	if err != nil {
		return err
	}

	price := types.Price{
		Asset:         asset,
		Price:         aggregatedPrice,
		BlockHeight:   sdkCtx.BlockHeight(),
		BlockTime:     sdkCtx.BlockTime().Unix(),
		NumValidators: uint32(len(filteredData.ValidPrices)),
	}

	sdkCtx.GasMeter().ConsumeGas(8000, "oracle_aggregate_set_price")
	if err := k.SetPrice(ctx, price); err != nil {
		return err
	}

	snapshot := types.PriceSnapshot{
		Asset:       asset,
		Price:       aggregatedPrice,
		BlockHeight: sdkCtx.BlockHeight(),
		BlockTime:   sdkCtx.BlockTime().Unix(),
	}
	if err := k.SetPriceSnapshot(ctx, snapshot); err != nil {
		return err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	if err := k.DeleteOldSnapshots(ctx, asset, minHeight); err != nil {
		return err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOraclePriceAggregated,
			sdk.NewAttribute(types.AttributeKeyAsset, asset),
			sdk.NewAttribute(types.AttributeKeyPrice, aggregatedPrice.String()),
			sdk.NewAttribute(types.AttributeKeyNumValidators, fmt.Sprintf("%d", len(filteredData.ValidPrices))),
			sdk.NewAttribute(types.AttributeKeyNumOutliers, fmt.Sprintf("%d", len(filteredData.FilteredOutliers))),
			sdk.NewAttribute(types.AttributeKeyMedian, filteredData.Median.String()),
			sdk.NewAttribute(types.AttributeKeyMAD, filteredData.MAD.String()),
		),
	)

	// Record metrics
	if k.metrics != nil {
		if k.metrics.AggregationCount != nil {
			k.metrics.AggregationCount.With(map[string]string{
				"asset":  asset,
				"status": "success",
			}).Inc()
		}

		if k.metrics.ValidatorParticipation != nil {
			k.metrics.ValidatorParticipation.With(map[string]string{
				"asset": asset,
			}).Set(float64(len(filteredData.ValidPrices)))
		}

		if k.metrics.OutliersDetected != nil {
			severityCounts := make(map[string]float64)
			for _, outlier := range filteredData.FilteredOutliers {
				severityKey := fmt.Sprintf("%d", outlier.Severity)
				severityCounts[severityKey]++
			}
			for severity, count := range severityCounts {
				k.metrics.OutliersDetected.With(map[string]string{
					"asset":    asset,
					"severity": severity,
				}).Add(count)
			}
		}
	}

	return nil
}

// detectAndFilterOutliers performs multi-stage statistical outlier detection
func (k Keeper) detectAndFilterOutliers(ctx context.Context, asset string, prices []types.ValidatorPrice) (*FilteredPriceData, error) {
	// Extract prices for statistical analysis
	priceValues := make([]sdkmath.LegacyDec, len(prices))
	for i, vp := range prices {
		priceValues[i] = vp.Price
	}

	const minSampleForAdvancedDetection = 5
	if len(prices) < minSampleForAdvancedDetection {
		// Preserve all submissions when the validator set is small; statistical filters are unreliable here.
		// However, we still need to check voting power threshold for security
		params, err := k.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		// Calculate total voting power from these prices
		totalVotingPower := sdkmath.LegacyZeroDec()
		for _, vp := range prices {
			totalVotingPower = totalVotingPower.Add(sdkmath.LegacyNewDec(vp.VotingPower))
		}

		// Get total bonded voting power
		bondedValidators, err := k.GetBondedValidators(ctx)
		if err != nil {
			return nil, err
		}

		totalBondedPower := int64(0)
		powerReduction := k.stakingKeeper.PowerReduction(ctx)
		for _, val := range bondedValidators {
			totalBondedPower += val.GetConsensusPower(powerReduction)
		}

		// Fallback: if no bonded validators found (test environments), use submitted power
		if totalBondedPower == 0 {
			totalBondedPower = totalVotingPower.TruncateInt64()
			if totalBondedPower == 0 {
				totalBondedPower = 1 // Avoid division by zero
			}
		}

		votingPowerPercentage := totalVotingPower.Quo(sdkmath.LegacyNewDec(totalBondedPower))

		// Check if validators meet minimum voting power threshold
		if votingPowerPercentage.LT(params.MinVotingPowerForConsensus) {
			return nil, types.ErrInsufficientOracleConsensus
		}

		median := k.calculateMedian(priceValues)
		return &FilteredPriceData{
			ValidPrices:      prices,
			FilteredOutliers: []OutlierDetectionResult{},
			Median:           median,
			MAD:              sdkmath.LegacyZeroDec(),
			IQR:              sdkmath.LegacyZeroDec(),
		}, nil
	}

	// OPTIMIZATION: Sort ONCE and reuse for all statistical calculations
	// This eliminates 3 redundant sorts (median, MAD, IQR each sorted independently before)
	sortedPrices := make([]sdkmath.LegacyDec, len(priceValues))
	copy(sortedPrices, priceValues)
	sort.Slice(sortedPrices, func(i, j int) bool {
		return sortedPrices[i].LT(sortedPrices[j])
	})

	// Stage 1: Calculate baseline statistics using pre-sorted data
	median := k.calculateMedianFromSorted(sortedPrices)
	mad := k.calculateMADFromSorted(sortedPrices, median)
	q1, q3, iqr := k.calculateIQRFromSorted(sortedPrices)

	// Get asset-specific volatility
	volatility := k.calculateVolatility(ctx, asset, 100)

	// Adjust thresholds based on volatility
	madThreshold := k.getMADThreshold(asset, volatility)

	outliers := []OutlierDetectionResult{}
	validPrices := []types.ValidatorPrice{}

	for _, vp := range prices {
		// Stage 2: Modified Z-Score using MAD (Median Absolute Deviation)
		severity, deviation := k.classifyOutlierSeverity(vp.Price, median, mad, madThreshold)

		if severity == SeverityNone {
			// Stage 3: IQR test for moderate outliers
			if !k.isIQROutlier(vp.Price, q1, q3, iqr, volatility) {
				// Stage 4: Grubbs' test for remaining suspicious values
				if len(priceValues) >= 7 && !k.grubbsTest(priceValues, vp.Price, 0.05) {
					validPrices = append(validPrices, vp)
					continue
				}
			}
			// Detected as outlier by IQR or Grubbs
			severity = SeverityModerate
			deviation = k.calculateDeviationFromMedian(vp.Price, median)
		}

		outliers = append(outliers, OutlierDetectionResult{
			ValidatorAddr: vp.ValidatorAddr,
			Price:         vp.Price,
			Severity:      severity,
			Deviation:     deviation,
			Reason:        k.getOutlierReason(severity),
		})
	}

	// Security: Check voting power threshold instead of validator count
	// This prevents manipulation by multiple low-stake validators
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	totalVotingPower := sdkmath.LegacyZeroDec()
	for _, vp := range validPrices {
		totalVotingPower = totalVotingPower.Add(sdkmath.LegacyNewDec(vp.VotingPower))
	}

	// Get total bonded voting power
	bondedValidators, err := k.GetBondedValidators(ctx)
	if err != nil {
		return nil, err
	}

	totalBondedPower := int64(0)
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	for _, val := range bondedValidators {
		totalBondedPower += val.GetConsensusPower(powerReduction)
	}

	// Fallback: if no bonded validators found (test environments), use submitted power
	if totalBondedPower == 0 {
		totalBondedPower = totalVotingPower.TruncateInt64()
		if totalBondedPower == 0 {
			totalBondedPower = 1 // Avoid division by zero
		}
	}

	votingPowerPercentage := totalVotingPower.Quo(sdkmath.LegacyNewDec(totalBondedPower))

	// Check if remaining validators meet minimum voting power threshold
	if votingPowerPercentage.LT(params.MinVotingPowerForConsensus) {
		return nil, types.ErrInsufficientOracleConsensus
	}

	return &FilteredPriceData{
		ValidPrices:      validPrices,
		FilteredOutliers: outliers,
		Median:           median,
		MAD:              mad,
		IQR:              iqr,
	}, nil
}

// calculateMedian calculates the median of prices
func (k Keeper) calculateMedian(prices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	sorted := make([]sdkmath.LegacyDec, len(prices))
	copy(sorted, prices)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LT(sorted[j])
	})

	n := len(sorted)
	if n%2 == 0 {
		return sorted[n/2-1].Add(sorted[n/2]).Quo(sdkmath.LegacyNewDec(2))
	}
	return sorted[n/2]
}

// calculateMAD calculates the Median Absolute Deviation
func (k Keeper) calculateMAD(prices []sdkmath.LegacyDec, median sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	deviations := make([]sdkmath.LegacyDec, len(prices))
	for i, price := range prices {
		deviation := price.Sub(median).Abs()
		deviations[i] = deviation
	}

	madMedian := k.calculateMedian(deviations)

	// MAD is typically scaled by 1.4826 for normal distribution consistency
	scaleFactor := sdkmath.LegacyMustNewDecFromStr("1.4826")
	return madMedian.Mul(scaleFactor)
}

// calculateIQR calculates the Interquartile Range using linear interpolation (R-7 method).
// This is the standard method used by R, Excel, and most statistical software.
func (k Keeper) calculateIQR(prices []sdkmath.LegacyDec) (q1, q3, iqr sdkmath.LegacyDec) {
	if len(prices) < 4 {
		return sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()
	}

	sorted := make([]sdkmath.LegacyDec, len(prices))
	copy(sorted, prices)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LT(sorted[j])
	})

	n := len(sorted)
	nDec := sdkmath.LegacyNewDec(int64(n))

	// R-7 method (standard): position = 1 + (n-1) * p
	// For Q1 (p=0.25): position = 1 + (n-1) * 0.25
	// For Q3 (p=0.75): position = 1 + (n-1) * 0.75

	// Calculate Q1 (25th percentile) using linear interpolation
	q1Pos := sdkmath.LegacyOneDec().Add(nDec.Sub(sdkmath.LegacyOneDec()).Mul(sdkmath.LegacyNewDecWithPrec(25, 2)))
	q1IdxLow := int(q1Pos.TruncateInt64()) - 1 // Convert to 0-based index
	if q1IdxLow < 0 {
		q1IdxLow = 0
	}
	q1IdxHigh := q1IdxLow + 1
	if q1IdxHigh >= n {
		q1IdxHigh = n - 1
	}
	q1Frac := q1Pos.Sub(sdkmath.LegacyNewDec(int64(q1IdxLow + 1)))
	q1 = sorted[q1IdxLow].Add(sorted[q1IdxHigh].Sub(sorted[q1IdxLow]).Mul(q1Frac))

	// Calculate Q3 (75th percentile) using linear interpolation
	q3Pos := sdkmath.LegacyOneDec().Add(nDec.Sub(sdkmath.LegacyOneDec()).Mul(sdkmath.LegacyNewDecWithPrec(75, 2)))
	q3IdxLow := int(q3Pos.TruncateInt64()) - 1 // Convert to 0-based index
	if q3IdxLow < 0 {
		q3IdxLow = 0
	}
	q3IdxHigh := q3IdxLow + 1
	if q3IdxHigh >= n {
		q3IdxHigh = n - 1
	}
	q3Frac := q3Pos.Sub(sdkmath.LegacyNewDec(int64(q3IdxLow + 1)))
	q3 = sorted[q3IdxLow].Add(sorted[q3IdxHigh].Sub(sorted[q3IdxLow]).Mul(q3Frac))

	iqr = q3.Sub(q1)
	return q1, q3, iqr
}

// calculateVolatility calculates rolling volatility for an asset
func (k Keeper) calculateVolatility(ctx context.Context, asset string, window int) sdkmath.LegacyDec {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get recent snapshots
	minHeight := sdkCtx.BlockHeight() - int64(window)
	snapshots := []types.PriceSnapshot{}

	err := k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
			// P1-PERF-2: Cap snapshot count to prevent DoS via unbounded iteration
			if len(snapshots) >= maxSnapshotsForVolatility {
				return true // Stop iteration when limit reached
			}
		}
		return false
	})

	if err != nil || len(snapshots) < 2 {
		// Default volatility for unknown assets
		return sdkmath.LegacyMustNewDecFromStr("0.05") // 5% default
	}

	// Calculate returns
	returns := make([]sdkmath.LegacyDec, len(snapshots)-1)
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i-1].Price.IsPositive() {
			ret := snapshots[i].Price.Sub(snapshots[i-1].Price).Quo(snapshots[i-1].Price)
			returns[i-1] = ret
		}
	}

	if len(returns) == 0 {
		return sdkmath.LegacyMustNewDecFromStr("0.05")
	}

	// Calculate standard deviation of returns
	mean := sdkmath.LegacyZeroDec()
	for _, ret := range returns {
		mean = mean.Add(ret)
	}
	mean = mean.Quo(sdkmath.LegacyNewDec(int64(len(returns))))

	variance := sdkmath.LegacyZeroDec()
	for _, ret := range returns {
		diff := ret.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Quo(sdkmath.LegacyNewDec(int64(len(returns))))

	// Convert to float for sqrt calculation
	varianceFloat, err := variance.Float64()
	if err != nil || varianceFloat < 0 {
		return sdkmath.LegacyMustNewDecFromStr("0.05")
	}

	stdDev := math.Sqrt(varianceFloat)

	// Clamp volatility between 0.01 and 1.0
	if stdDev < 0.01 {
		stdDev = 0.01
	}
	if stdDev > 1.0 {
		stdDev = 1.0
	}

	return sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.6f", stdDev))
}

// getMADThreshold returns the MAD threshold multiplier based on asset and volatility
func (k Keeper) getMADThreshold(asset string, volatility sdkmath.LegacyDec) sdkmath.LegacyDec {
	baseThreshold := sdkmath.LegacyMustNewDecFromStr("3.5") // Modified Z-score threshold

	// Adjust threshold based on volatility
	// Higher volatility = more tolerant (higher threshold)
	// Lower volatility = less tolerant (lower threshold)

	volatilityFloat, err := volatility.Float64()
	if err != nil {
		return baseThreshold
	}

	// Volatility adjustment factor: 1.0 + (volatility * 10)
	// e.g., 5% volatility = 1.5x multiplier, 10% volatility = 2.0x multiplier
	adjustmentFactor := 1.0 + (volatilityFloat * 10.0)

	// Clamp adjustment between 1.0 and 3.0
	if adjustmentFactor < 1.0 {
		adjustmentFactor = 1.0
	}
	if adjustmentFactor > 3.0 {
		adjustmentFactor = 3.0
	}

	adjustment := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.2f", adjustmentFactor))
	return baseThreshold.Mul(adjustment)
}

// classifyOutlierSeverity classifies the severity of an outlier using Modified Z-Score
func (k Keeper) classifyOutlierSeverity(price, median, mad, threshold sdkmath.LegacyDec) (OutlierSeverity, sdkmath.LegacyDec) {
	if mad.IsZero() {
		// If MAD is zero, all prices are identical
		if !price.Equal(median) {
			return SeverityExtreme, price.Sub(median).Abs()
		}
		return SeverityNone, sdkmath.LegacyZeroDec()
	}

	// Modified Z-score = 0.6745 * (price - median) / MAD
	deviation := price.Sub(median).Abs()
	modifiedZScore := deviation.Mul(sdkmath.LegacyMustNewDecFromStr("0.6745")).Quo(mad)

	// Severity thresholds
	extremeThreshold := threshold.Mul(sdkmath.LegacyMustNewDecFromStr("1.4"))  // ~5 sigma
	highThreshold := threshold                                                 // ~3.5 sigma
	moderateThreshold := threshold.Mul(sdkmath.LegacyMustNewDecFromStr("0.7")) // ~2.5 sigma
	lowThreshold := threshold.Mul(sdkmath.LegacyMustNewDecFromStr("0.5"))      // ~1.75 sigma

	if modifiedZScore.GTE(extremeThreshold) {
		return SeverityExtreme, deviation
	} else if modifiedZScore.GTE(highThreshold) {
		return SeverityHigh, deviation
	} else if modifiedZScore.GTE(moderateThreshold) {
		return SeverityModerate, deviation
	} else if modifiedZScore.GTE(lowThreshold) {
		return SeverityLow, deviation
	}

	return SeverityNone, sdkmath.LegacyZeroDec()
}

// isIQROutlier checks if a price is an outlier using IQR method
func (k Keeper) isIQROutlier(price, q1, q3, iqr, volatility sdkmath.LegacyDec) bool {
	if iqr.IsZero() {
		return false
	}

	// Adjust IQR multiplier based on volatility
	// Standard is 1.5, but we adjust based on volatility
	iqrMultiplier := sdkmath.LegacyMustNewDecFromStr("1.5")

	volatilityFloat, err := volatility.Float64()
	if err == nil {
		// Higher volatility = more tolerant
		adjustedMultiplier := 1.5 + (volatilityFloat * 5.0)
		if adjustedMultiplier > 3.0 {
			adjustedMultiplier = 3.0
		}
		iqrMultiplier = sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.2f", adjustedMultiplier))
	}

	lowerBound := q1.Sub(iqr.Mul(iqrMultiplier))
	upperBound := q3.Add(iqr.Mul(iqrMultiplier))

	return price.LT(lowerBound) || price.GT(upperBound)
}

// grubbsTest performs Grubbs' test for outlier detection
func (k Keeper) grubbsTest(prices []sdkmath.LegacyDec, testPrice sdkmath.LegacyDec, alpha float64) bool {
	if len(prices) < 7 {
		// Grubbs' test requires reasonable sample size
		return false
	}

	// Calculate mean
	sum := sdkmath.LegacyZeroDec()
	for _, p := range prices {
		sum = sum.Add(p)
	}
	mean := sum.Quo(sdkmath.LegacyNewDec(int64(len(prices))))

	// Calculate standard deviation
	variance := sdkmath.LegacyZeroDec()
	for _, p := range prices {
		diff := p.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Quo(sdkmath.LegacyNewDec(int64(len(prices))))

	varianceFloat, err := variance.Float64()
	if err != nil || varianceFloat <= 0 {
		return false
	}

	stdDev := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", math.Sqrt(varianceFloat)))

	if stdDev.IsZero() {
		return !testPrice.Equal(mean)
	}

	// Calculate Grubbs' statistic
	deviation := testPrice.Sub(mean).Abs()
	grubbsStat := deviation.Quo(stdDev)

	// Critical value approximation for alpha = 0.05
	// For n >= 7, critical value ≈ (n-1)/sqrt(n) * sqrt(t^2/(n-2+t^2))
	// Simplified: for n=7, critical ≈ 2.02; n=10, critical ≈ 2.29; n=20, critical ≈ 2.71
	n := float64(len(prices))
	criticalValue := (n - 1.0) / math.Sqrt(n) * math.Sqrt(4.0/(n-2.0+4.0))

	grubbsStatFloat, err := grubbsStat.Float64()
	if err != nil {
		return false
	}

	return grubbsStatFloat > criticalValue
}

// calculateDeviationFromMedian calculates absolute deviation from median
func (k Keeper) calculateDeviationFromMedian(price, median sdkmath.LegacyDec) sdkmath.LegacyDec {
	return price.Sub(median).Abs()
}

// getOutlierReason returns a human-readable reason for the outlier classification
func (k Keeper) getOutlierReason(severity OutlierSeverity) string {
	switch severity {
	case SeverityExtreme:
		return "extreme_outlier_mad_test"
	case SeverityHigh:
		return "high_outlier_mad_test"
	case SeverityModerate:
		return "moderate_outlier_iqr_test"
	case SeverityLow:
		return "low_outlier_preliminary"
	default:
		return "valid"
	}
}

// calculateVotingPower calculates total voting power and filters valid prices
func (k Keeper) calculateVotingPower(ctx context.Context, validatorPrices []types.ValidatorPrice) (int64, []types.ValidatorPrice, error) {
	totalVotingPower := int64(0)
	validPrices := []types.ValidatorPrice{}

	bondedValidators, err := k.GetBondedValidators(ctx)
	if err != nil {
		return 0, nil, err
	}

	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	for _, val := range bondedValidators {
		totalVotingPower += val.GetConsensusPower(powerReduction)
	}

	// Fallback: if no bonded validators are found (test environments), derive total power from submissions.
	if totalVotingPower == 0 {
		for _, vp := range validatorPrices {
			totalVotingPower += vp.VotingPower
		}
	}

	for _, vp := range validatorPrices {
		valAddr, err := sdk.ValAddressFromBech32(vp.ValidatorAddr)
		if err != nil {
			continue
		}

		isActive, err := k.IsActiveValidator(ctx, valAddr)
		if err != nil || !isActive {
			continue
		}

		if vp.Price.IsNil() || vp.Price.LTE(sdkmath.LegacyZeroDec()) {
			continue
		}

		validPrices = append(validPrices, vp)
	}

	if totalVotingPower == 0 {
		// Avoid division by zero; treat as single unit power to continue aggregation in degenerate setups.
		totalVotingPower = 1
	}

	return totalVotingPower, validPrices, nil
}

// calculateWeightedMedian calculates the weighted median of validator prices
func (k Keeper) calculateWeightedMedian(validatorPrices []types.ValidatorPrice) (sdkmath.LegacyDec, error) {
	if len(validatorPrices) == 0 {
		return sdkmath.LegacyDec{}, fmt.Errorf("no prices to aggregate")
	}

	sort.Slice(validatorPrices, func(i, j int) bool {
		return validatorPrices[i].Price.LT(validatorPrices[j].Price)
	})

	totalPower := int64(0)
	for _, vp := range validatorPrices {
		totalPower += vp.VotingPower
	}

	halfPower := (totalPower + 1) / 2
	cumulativePower := int64(0)

	for _, vp := range validatorPrices {
		cumulativePower += vp.VotingPower
		if cumulativePower >= halfPower {
			return vp.Price, nil
		}
	}

	return validatorPrices[len(validatorPrices)-1].Price, nil
}

// CalculateTWAP calculates the Time-Weighted Average Price for an asset
func (k Keeper) CalculateTWAP(ctx context.Context, asset string) (sdkmath.LegacyDec, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Gas metering: TWAP calculation base cost
	sdkCtx.GasMeter().ConsumeGas(20000, "oracle_twap_base")

	params, err := k.GetParams(ctx)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}

	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)
	snapshots := []types.PriceSnapshot{}

	// HIGH-7 Fix: Cap snapshot count to prevent OOM attacks
	const maxSnapshots = 1000

	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
			// Stop iteration if we reach the maximum snapshot count
			if len(snapshots) >= maxSnapshots {
				return true
			}
		}
		return false
	})
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}

	if len(snapshots) == 0 {
		return sdkmath.LegacyDec{}, fmt.Errorf("no snapshots available for TWAP calculation")
	}

	// Gas per snapshot processed
	sdkCtx.GasMeter().ConsumeGas(uint64(len(snapshots)*1000), "oracle_twap_snapshots")

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].BlockHeight < snapshots[j].BlockHeight
	})

	totalWeightedPrice := sdkmath.LegacyZeroDec()
	totalTime := int64(0)

	for i := 0; i < len(snapshots)-1; i++ {
		timeDelta := snapshots[i+1].BlockTime - snapshots[i].BlockTime
		if timeDelta <= 0 {
			continue
		}
		// Overflow protection: ensure timeDelta is reasonable
		if timeDelta > 1e18 {
			return sdkmath.LegacyDec{}, fmt.Errorf("time delta too large: %d", timeDelta)
		}
		weightedPrice := snapshots[i].Price.MulInt64(timeDelta)
		totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
		totalTime += timeDelta
	}

	lastSnapshot := snapshots[len(snapshots)-1]
	lastTimeDelta := sdkCtx.BlockTime().Unix() - lastSnapshot.BlockTime
	if lastTimeDelta > 0 {
		// Overflow protection
		if lastTimeDelta > 1e18 {
			return sdkmath.LegacyDec{}, fmt.Errorf("last time delta too large: %d", lastTimeDelta)
		}
		weightedPrice := lastSnapshot.Price.MulInt64(lastTimeDelta)
		totalWeightedPrice = totalWeightedPrice.Add(weightedPrice)
		totalTime += lastTimeDelta
	}

	if totalTime == 0 {
		sumPrices := sdkmath.LegacyZeroDec()
		for _, snapshot := range snapshots {
			sumPrices = sumPrices.Add(snapshot.Price)
		}
		return sumPrices.QuoInt64(int64(len(snapshots))), nil
	}

	return totalWeightedPrice.QuoInt64(totalTime), nil
}

// CheckMissedVotes checks which validators missed submitting prices and updates counters
func (k Keeper) CheckMissedVotes(ctx context.Context, asset string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	bondedValidators, err := k.GetBondedValidators(ctx)
	if err != nil {
		return err
	}
	validatorPrices, err := k.GetValidatorPricesByAsset(ctx, asset)
	if err != nil {
		return err
	}

	submitted := make(map[string]bool)
	for _, vp := range validatorPrices {
		submitted[vp.ValidatorAddr] = true
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	for _, validator := range bondedValidators {
		valAddr := validator.GetOperator()

		if submitted[valAddr] {
			if err := k.ResetMissCounter(ctx, valAddr); err != nil {
				return err
			}
		} else {
			if err := k.IncrementMissCounter(ctx, valAddr); err != nil {
				return err
			}

			validatorOracle, err := k.GetValidatorOracle(ctx, valAddr)
			if err != nil {
				return err
			}

			if validatorOracle.MissCounter >= params.MinValidPerWindow {
				if err := k.SlashMissVote(ctx, valAddr); err != nil {
					sdkCtx.Logger().Error("failed to slash validator for missed vote",
						"validator", valAddr,
						"error", err.Error(),
					)
				}
			}
		}
	}

	return nil
}

// calculateMedianFromSorted calculates median from pre-sorted prices (optimized version)
func (k Keeper) calculateMedianFromSorted(sortedPrices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(sortedPrices) == 0 {
		return sdkmath.LegacyZeroDec()
	}
	n := len(sortedPrices)
	if n%2 == 0 {
		return sortedPrices[n/2-1].Add(sortedPrices[n/2]).Quo(sdkmath.LegacyNewDec(2))
	}
	return sortedPrices[n/2]
}

// calculateMADFromSorted calculates MAD from pre-sorted prices (optimized version)
func (k Keeper) calculateMADFromSorted(sortedPrices []sdkmath.LegacyDec, median sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(sortedPrices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	// Calculate deviations (unsorted)
	deviations := make([]sdkmath.LegacyDec, len(sortedPrices))
	for i, price := range sortedPrices {
		deviation := price.Sub(median).Abs()
		deviations[i] = deviation
	}

	// Must sort deviations to find their median
	sort.Slice(deviations, func(i, j int) bool {
		return deviations[i].LT(deviations[j])
	})
	madMedian := k.calculateMedianFromSorted(deviations)

	// MAD is typically scaled by 1.4826 for normal distribution consistency
	scaleFactor := sdkmath.LegacyMustNewDecFromStr("1.4826")
	return madMedian.Mul(scaleFactor)
}

// calculateIQRFromSorted calculates IQR from pre-sorted prices (optimized version)
func (k Keeper) calculateIQRFromSorted(sortedPrices []sdkmath.LegacyDec) (q1, q3, iqr sdkmath.LegacyDec) {
	if len(sortedPrices) < 4 {
		return sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()
	}

	n := len(sortedPrices)
	nDec := sdkmath.LegacyNewDec(int64(n))

	// R-7 method (standard): position = 1 + (n-1) * p
	// For Q1 (p=0.25): position = 1 + (n-1) * 0.25
	// For Q3 (p=0.75): position = 1 + (n-1) * 0.75

	// Calculate Q1 (25th percentile) using linear interpolation
	q1Pos := sdkmath.LegacyOneDec().Add(nDec.Sub(sdkmath.LegacyOneDec()).Mul(sdkmath.LegacyNewDecWithPrec(25, 2)))
	q1IdxLow := int(q1Pos.TruncateInt64()) - 1 // Convert to 0-based index
	if q1IdxLow < 0 {
		q1IdxLow = 0
	}
	q1IdxHigh := q1IdxLow + 1
	if q1IdxHigh >= n {
		q1IdxHigh = n - 1
	}
	q1Frac := q1Pos.Sub(sdkmath.LegacyNewDec(int64(q1IdxLow + 1)))
	q1 = sortedPrices[q1IdxLow].Add(sortedPrices[q1IdxHigh].Sub(sortedPrices[q1IdxLow]).Mul(q1Frac))

	// Calculate Q3 (75th percentile) using linear interpolation
	q3Pos := sdkmath.LegacyOneDec().Add(nDec.Sub(sdkmath.LegacyOneDec()).Mul(sdkmath.LegacyNewDecWithPrec(75, 2)))
	q3IdxLow := int(q3Pos.TruncateInt64()) - 1 // Convert to 0-based index
	if q3IdxLow < 0 {
		q3IdxLow = 0
	}
	q3IdxHigh := q3IdxLow + 1
	if q3IdxHigh >= n {
		q3IdxHigh = n - 1
	}
	q3Frac := q3Pos.Sub(sdkmath.LegacyNewDec(int64(q3IdxLow + 1)))
	q3 = sortedPrices[q3IdxLow].Add(sortedPrices[q3IdxHigh].Sub(sortedPrices[q3IdxLow]).Mul(q3Frac))

	iqr = q3.Sub(q1)
	return q1, q3, iqr
}
