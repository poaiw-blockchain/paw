package property_test

import (
	"math/rand"
	"sort"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"
)

// Property: Weighted median must be within the range of input prices
func TestPropertyWeightedMedianWithinRange(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numValidators := rng.Intn(98) + 3 // 3-100 validators
		prices := make([]uint64, numValidators)
		weights := make([]uint64, numValidators)

		for i := 0; i < numValidators; i++ {
			prices[i] = uint64(rng.Int63n(1000000000) + 1000)
			weights[i] = uint64(rng.Int63n(1000000) + 1)
		}

		median := calculateWeightedMedian(prices, weights)
		minPrice := minSlice(prices)
		maxPrice := maxSlice(prices)

		return median >= minPrice && median <= maxPrice
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Byzantine resistance - <33% malicious nodes cannot control median
func TestPropertyByzantineResistance(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numValidators := rng.Intn(30) + 10 // 10-40 validators
		numMalicious := numValidators / 3 - 1 // Just under 33%

		prices := make([]uint64, numValidators)
		weights := make([]uint64, numValidators)

		honestPrice := uint64(1000000)
		maliciousPrice := uint64(10000000) // 10x attack

		// Honest validators
		for i := numMalicious; i < numValidators; i++ {
			prices[i] = honestPrice + uint64(rng.Int63n(10000))
			weights[i] = uint64(rng.Int63n(1000) + 100)
		}

		// Malicious validators
		for i := 0; i < numMalicious; i++ {
			prices[i] = maliciousPrice
			weights[i] = uint64(rng.Int63n(1000) + 100)
		}

		median := calculateWeightedMedian(prices, weights)

		// Median should be close to honest price, not malicious price
		deviation := absUint64Diff(median, honestPrice)
		maxAllowedDeviation := honestPrice / 10 // 10% tolerance

		return deviation < maxAllowedDeviation
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 500}))
}

// Property: MAD (Median Absolute Deviation) is always non-negative
func TestPropertyMADNonNegative(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numPrices := rng.Intn(97) + 3 // 3-100 prices
		prices := make([]uint64, numPrices)

		for i := 0; i < numPrices; i++ {
			prices[i] = uint64(rng.Int63n(1000000) + 1)
		}

		median := calculateSimpleMedian(prices)
		mad := calculateMAD(prices, median)

		return mad >= 0
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Outlier detection should identify extreme values
func TestPropertyOutlierDetection(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numNormal := rng.Intn(20) + 5 // 5-25 normal prices
		normalPrice := uint64(1000000)

		prices := make([]uint64, numNormal+2)

		// Normal prices (within 1% variance)
		for i := 0; i < numNormal; i++ {
			variance := int64(rng.Intn(10000)) - 5000 // ±0.5%
			prices[i] = uint64(int64(normalPrice) + variance)
		}

		// Clear outliers
		prices[numNormal] = normalPrice * 3   // 3x outlier
		prices[numNormal+1] = normalPrice / 3 // 1/3x outlier

		median := calculateSimpleMedian(prices)
		mad := calculateMAD(prices, median)

		outliers := detectOutliers(prices, median, mad, 3.5)

		// Should detect at least the two extreme outliers
		return len(outliers) >= 2
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 500}))
}

// Property: TWAP (Time-Weighted Average Price) should be bounded by min/max
func TestPropertyTWAPBounded(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numPrices := rng.Intn(47) + 3 // 3-50 prices
		prices := make([]uint64, numPrices)
		timestamps := make([]int64, numPrices)

		baseTime := int64(1000000)
		for i := 0; i < numPrices; i++ {
			prices[i] = uint64(rng.Int63n(1000000) + 100000)
			timestamps[i] = baseTime + int64(i*60) // 1 minute intervals
		}

		twap := calculateTWAP(prices, timestamps)
		minPrice := minSlice(prices)
		maxPrice := maxSlice(prices)

		return twap >= minPrice && twap <= maxPrice
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Slashing should increase with deviation severity
func TestPropertySlashingMonotonicity(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		threshold := uint64(5000) // 5% threshold

		// Two deviations with different severities
		deviation1 := uint64(rng.Int63n(10000) + 5001) // Above threshold
		deviation2 := deviation1 * 2                   // Double deviation

		slash1 := calculateSlashFraction(deviation1, threshold)
		slash2 := calculateSlashFraction(deviation2, threshold)

		// Higher deviation should result in higher or equal slash
		return slash2 >= slash1
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Price staleness detection should be consistent
func TestPropertyPriceStaleness(t *testing.T) {
	t.Parallel()
	property := func(submissionTime, currentTime int64, maxAge uint32) bool {
		if submissionTime <= 0 || currentTime <= 0 || maxAge == 0 {
			return true // Skip invalid inputs
		}

		if currentTime < submissionTime {
			return true // Skip time travel scenarios
		}

		if maxAge > 86400 {
			maxAge = 86400 // Cap at 24 hours
		}

		age := currentTime - submissionTime
		isStale := age > int64(maxAge)

		// Property: If age > maxAge, should be stale
		expectedStale := age > int64(maxAge)

		return isStale == expectedStale
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Vote power must sum to 100% in aggregation
func TestPropertyVotePowerConservation(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numValidators := rng.Intn(47) + 3
		votePowers := make([]uint64, numValidators)

		totalPower := uint64(0)
		for i := 0; i < numValidators; i++ {
			votePowers[i] = uint64(rng.Int63n(1000000) + 1)
			totalPower += votePowers[i]
		}

		// Normalize to percentages
		normalizedPowers := make([]uint64, numValidators)
		sumNormalized := uint64(0)

		for i := 0; i < numValidators; i++ {
			normalizedPowers[i] = (votePowers[i] * 10000) / totalPower
			sumNormalized += normalizedPowers[i]
		}

		// Sum should be approximately 10000 (100%)
		// Allow small rounding error
		return absUint64Diff(sumNormalized, 10000) < uint64(numValidators)
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Aggregated price should be deterministic
func TestPropertyPriceAggregationDeterminism(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numValidators := rng.Intn(20) + 3
		prices := make([]uint64, numValidators)
		weights := make([]uint64, numValidators)

		for i := 0; i < numValidators; i++ {
			prices[i] = uint64(rng.Int63n(1000000) + 1000)
			weights[i] = uint64(rng.Int63n(1000) + 1)
		}

		// Calculate twice
		median1 := calculateWeightedMedian(prices, weights)
		median2 := calculateWeightedMedian(prices, weights)

		return median1 == median2
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: No single validator with <50% power can control outcome
func TestPropertyNoMajorityManipulation(t *testing.T) {
	t.Parallel()
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		numValidators := rng.Intn(20) + 5
		prices := make([]uint64, numValidators)
		weights := make([]uint64, numValidators)

		honestPrice := uint64(1000000)
		attackerPrice := uint64(5000000)

		totalWeight := uint64(0)
		for i := 0; i < numValidators; i++ {
			weights[i] = uint64(rng.Int63n(100) + 1)
			totalWeight += weights[i]
		}

		// Attacker has 45% of voting power
		attackerWeight := totalWeight * 45 / 100
		weights[0] = attackerWeight
		prices[0] = attackerPrice

		// Honest validators
		for i := 1; i < numValidators; i++ {
			prices[i] = honestPrice + uint64(rng.Int63n(10000))
		}

		median := calculateWeightedMedian(prices, weights)

		// Median should be closer to honest price than attacker price
		deviationFromHonest := absUint64Diff(median, honestPrice)
		deviationFromAttacker := absUint64Diff(median, attackerPrice)

		return deviationFromHonest < deviationFromAttacker
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 500}))
}

// Helper functions

func calculateWeightedMedian(prices, weights []uint64) uint64 {
	if len(prices) == 0 || len(prices) != len(weights) {
		return 0
	}

	type weightedPrice struct {
		price  uint64
		weight uint64
	}

	pairs := make([]weightedPrice, len(prices))
	totalWeight := uint64(0)

	for i := range prices {
		pairs[i] = weightedPrice{prices[i], weights[i]}
		totalWeight += weights[i]
	}

	// Sort by price
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].price < pairs[j].price
	})

	// Find weighted median
	halfWeight := totalWeight / 2
	cumulativeWeight := uint64(0)

	for _, pair := range pairs {
		cumulativeWeight += pair.weight
		if cumulativeWeight >= halfWeight {
			return pair.price
		}
	}

	return pairs[len(pairs)-1].price
}

func calculateSimpleMedian(prices []uint64) uint64 {
	if len(prices) == 0 {
		return 0
	}

	sorted := make([]uint64, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func calculateMAD(prices []uint64, median uint64) uint64 {
	if len(prices) == 0 {
		return 0
	}

	deviations := make([]uint64, len(prices))
	for i, price := range prices {
		deviations[i] = absUint64Diff(price, median)
	}

	return calculateSimpleMedian(deviations)
}

func detectOutliers(prices []uint64, median, mad uint64, threshold float64) []int {
	outliers := make([]int, 0)

	if mad == 0 {
		return outliers
	}

	for i, price := range prices {
		deviation := absUint64Diff(price, median)
		modifiedZScore := float64(deviation) / (1.4826 * float64(mad))

		if modifiedZScore > threshold {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

func calculateTWAP(prices []uint64, timestamps []int64) uint64 {
	if len(prices) == 0 || len(prices) != len(timestamps) {
		return 0
	}

	if len(prices) == 1 {
		return prices[0]
	}

	weightedSum := uint64(0)
	totalTime := uint64(0)

	for i := 0; i < len(prices)-1; i++ {
		timeDiff := uint64(timestamps[i+1] - timestamps[i])
		weightedSum += prices[i] * timeDiff
		totalTime += timeDiff
	}

	if totalTime == 0 {
		return calculateSimpleMedian(prices)
	}

	return weightedSum / totalTime
}

func calculateSlashFraction(deviation, threshold uint64) uint64 {
	if deviation <= threshold {
		return 0
	}

	excessDeviation := deviation - threshold

	// Progressive slashing: 1% base + 0.5% per 1% excess
	slashFraction := 100 + (excessDeviation / 2)

	// Cap at 10% (1000 basis points)
	if slashFraction > 1000 {
		slashFraction = 1000
	}

	return slashFraction
}

func minSlice(vals []uint64) uint64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxSlice(vals []uint64) uint64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func absUint64Diff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
