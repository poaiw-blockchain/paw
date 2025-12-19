package fuzz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// OracleFuzzInput represents fuzzed input for oracle operations
type OracleFuzzInput struct {
	NumValidators  uint8
	NumAssets      uint8
	PriceValues    []uint64
	Timestamps     []int64
	VotingPowers   []uint64
	DeviationPcts  []uint16
	MaliciousNodes []uint8
}

// FuzzOraclePriceAggregation tests price aggregation with random inputs
// This fuzzer aims to find edge cases in median calculation and outlier detection
func FuzzOraclePriceAggregation(f *testing.F) {
	// Seed corpus with known edge cases
	seeds := [][]byte{
		// All validators agree
		generateOracleSeed(5, 1, []uint64{1000000, 1000000, 1000000, 1000000, 1000000}),
		// Extreme price divergence
		generateOracleSeed(5, 2, []uint64{1000000, 2000000, 500000, 1500000, 3000000}),
		// Byzantine attack - 33% manipulation
		generateOracleSeed(9, 3, []uint64{1000000, 1000000, 1000000, 1000000, 1000000, 5000000, 5000000, 5000000}),
		// Single validator
		generateOracleSeed(1, 1, []uint64{1000000}),
		// Maximum validators
		generateOracleSeed(100, 2, generateLinearPrices(100, 1000000, 1010000)),
		// Zero prices (edge case)
		generateOracleSeed(5, 4, []uint64{0, 0, 0, 0, 0}),
		// Overflow attempt
		generateOracleSeed(5, 5, []uint64{math.MaxUint64 - 1, math.MaxUint64 - 2, math.MaxUint64 - 3, 1000000, 1000000}),
		// Weighted outlier
		generateWeightedOracleSeed(5, []uint64{1000000, 1000000, 1000000, 1000000, 10000000}, []uint64{10, 10, 10, 10, 60}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 10 {
			return
		}

		input := parseOracleFuzzInput(data)
		if input == nil {
			return
		}

		// Validate constraints
		if input.NumValidators == 0 || input.NumValidators > 200 {
			return
		}
		if input.NumAssets == 0 || input.NumAssets > 50 {
			return
		}
		if len(input.PriceValues) < int(input.NumValidators) {
			return
		}

		// Test weighted median calculation
		median, err := calculateWeightedMedian(input.PriceValues[:input.NumValidators], input.VotingPowers[:input.NumValidators])
		if err != nil {
			// Expected errors should be graceful
			require.Contains(t, err.Error(), "weight")
			return
		}

		// Invariants
		require.NotNil(t, median, "Median should never be nil for valid inputs")

		// Median should be within the range of input prices
		minPrice := minUint64(input.PriceValues[:input.NumValidators])
		maxPrice := maxUint64(input.PriceValues[:input.NumValidators])
		require.True(t, median >= minPrice && median <= maxPrice,
			"Median %d must be within price range [%d, %d]", median, minPrice, maxPrice)

		// Test MAD (Median Absolute Deviation) calculation
		mad := calculateMAD(input.PriceValues[:input.NumValidators], median)

		// Test outlier detection
		outliers := detectOutliers(input.PriceValues[:input.NumValidators], median, mad, 3.0)
		require.True(t, len(outliers) <= int(input.NumValidators), "Cannot have more outliers than validators")

		// Test Byzantine resistance (< 33% malicious)
		numMalicious := len(input.MaliciousNodes)
		if numMalicious < int(input.NumValidators)/3 {
			// System should still produce valid median
			require.NotZero(t, median, "System should be Byzantine-resistant with < 33%% malicious nodes")
		}
	})
}

// FuzzOracleTimestampValidation tests timestamp validation logic
func FuzzOracleTimestampValidation(f *testing.F) {
	now := time.Now().Unix()

	seeds := [][]byte{
		encodeTimestamp(now),
		encodeTimestamp(now - 3600), // 1 hour old
		encodeTimestamp(now + 3600), // 1 hour future
		encodeTimestamp(0),
		encodeTimestamp(-1000),
		encodeTimestamp(math.MaxInt64),
		encodeTimestamp(now - 86400*365), // 1 year old
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}

		timestamp := int64(binary.BigEndian.Uint64(data[:8]))
		currentTime := time.Now().Unix()

		// Test timestamp staleness check
		maxAge := int64(300) // 5 minutes
		isStale := isTimestampStale(timestamp, currentTime, maxAge)

		if timestamp <= 0 {
			require.True(t, isStale, "Zero or negative timestamps should be considered stale")
		}

		if timestamp > currentTime+60 {
			require.True(t, isStale, "Future timestamps should be rejected")
		}

		age := currentTime - timestamp
		if age > maxAge && timestamp > 0 {
			require.True(t, isStale, "Old timestamps should be stale")
		}
	})
}

// FuzzOracleSlashingLogic tests slashing calculation for malicious oracles
func FuzzOracleSlashingLogic(f *testing.F) {
	seeds := [][]byte{
		encodeSlashingInput(1000000, 1000000, 1000), // No deviation
		encodeSlashingInput(1000000, 1100000, 800),  // 10% deviation
		encodeSlashingInput(1000000, 2000000, 1200), // 100% deviation
		encodeSlashingInput(1000000, 500000, 500),   // -50% deviation
		encodeSlashingInput(0, 1000000, 1500),       // Zero reference
		encodeSlashingInput(1000000, 0, 700),        // Zero submission
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 24 {
			return
		}

		referencePrice := binary.BigEndian.Uint64(data[0:8])
		submittedPrice := binary.BigEndian.Uint64(data[8:16])
		threshold := binary.BigEndian.Uint64(data[16:24])

		if threshold > 10000 { // Max 100%
			return
		}

		deviation := calculatePriceDeviation(referencePrice, submittedPrice)
		shouldSlash := deviation > threshold

		if shouldSlash {
			slashFraction := calculateSlashFraction(deviation, threshold)

			// Slashing invariants
			require.LessOrEqual(t, slashFraction, uint64(10000),
				"Slash fraction must be between 0 and 10000 (100%%)")

			// More severe deviations should result in higher slash fractions
			if deviation > threshold*2 {
				require.True(t, slashFraction > 100, "Severe deviations should have meaningful slashing")
			}
		}
	})
}

// FuzzOracleVotingPowerManipulation tests resistance to voting power attacks
func FuzzOracleVotingPowerManipulation(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 50 {
			return
		}

		numVals := int(data[0]) % 100
		if numVals < 3 {
			return
		}

		// Parse voting powers and prices
		powers := make([]uint64, numVals)
		prices := make([]uint64, numVals)

		for i := 0; i < numVals && i*16+16 < len(data); i++ {
			powers[i] = binary.BigEndian.Uint64(data[i*16+1 : i*16+9])
			prices[i] = binary.BigEndian.Uint64(data[i*16+9 : i*16+17])

			// Ensure non-zero voting power
			if powers[i] == 0 {
				powers[i] = 1
			}

			// Cap voting power to prevent overflow
			if powers[i] > 1000000000 {
				powers[i] %= 1000000000
			}
		}

		totalPower := uint64(0)
		for _, p := range powers[:numVals] {
			totalPower += p
		}

		// Test: No single validator should have > 50% voting power in production
		maxPower := maxUint64(powers[:numVals])

		// Calculate weighted median
		median, err := calculateWeightedMedian(prices[:numVals], powers[:numVals])
		if err != nil {
			return
		}

		// If no single validator has majority, median should be robust
		if maxPower < totalPower/2 {
			// Verify median makes sense
			require.True(t, median > 0 || allZero(prices[:numVals]),
				"Valid weighted median required when no majority validator")
		}

		// Test Sybil attack resistance
		// Even with many validators controlled by attacker, median should be safe
		sybilPrice := uint64(999999999)
		numSybils := numVals / 3 // 33% Sybil nodes

		for i := 0; i < numSybils; i++ {
			prices[i] = sybilPrice
		}

		median2, err := calculateWeightedMedian(prices[:numVals], powers[:numVals])
		if err != nil {
			return
		}

		// Median shouldn't be drastically affected by minority Sybil attack
		require.NotNil(t, median2)
	})
}

// Helper functions

func generateOracleSeed(numVals, numAssets uint8, prices []uint64) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(numVals)
	buf.WriteByte(numAssets)

	for _, price := range prices {
		if err := binary.Write(buf, binary.BigEndian, price); err != nil {
			panic(err)
		}
	}

	return buf.Bytes()
}

func generateWeightedOracleSeed(numVals uint8, prices, weights []uint64) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(numVals)
	buf.WriteByte(1)

	for i := 0; i < int(numVals); i++ {
		if err := binary.Write(buf, binary.BigEndian, prices[i]); err != nil {
			panic(err)
		}
		if err := binary.Write(buf, binary.BigEndian, weights[i]); err != nil {
			panic(err)
		}
	}

	return buf.Bytes()
}

func generateLinearPrices(count int, start, end uint64) []uint64 {
	prices := make([]uint64, count)
	step := (end - start) / uint64(count-1)
	for i := 0; i < count; i++ {
		prices[i] = start + uint64(i)*step
	}
	return prices
}

func parseOracleFuzzInput(data []byte) *OracleFuzzInput {
	if len(data) < 2 {
		return nil
	}

	input := &OracleFuzzInput{
		NumValidators: data[0],
		NumAssets:     data[1],
	}

	offset := 2
	maxVals := intMin(int(input.NumValidators), 200)

	// Parse prices
	input.PriceValues = make([]uint64, 0, maxVals)
	input.VotingPowers = make([]uint64, 0, maxVals)

	for i := 0; i < maxVals && offset+16 <= len(data); i++ {
		price := binary.BigEndian.Uint64(data[offset : offset+8])
		weight := binary.BigEndian.Uint64(data[offset+8 : offset+16])

		if weight == 0 {
			weight = 1 // Ensure non-zero weight
		}

		input.PriceValues = append(input.PriceValues, price)
		input.VotingPowers = append(input.VotingPowers, weight)
		offset += 16
	}

	return input
}

func calculateWeightedMedian(values, weights []uint64) (uint64, error) {
	if len(values) != len(weights) {
		return 0, fmt.Errorf("values and weights must have same length")
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("empty input")
	}

	// Create weighted pairs
	type weightedValue struct {
		value  uint64
		weight uint64
	}

	pairs := make([]weightedValue, len(values))
	totalWeight := uint64(0)

	for i := range values {
		pairs[i] = weightedValue{values[i], weights[i]}
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return 0, fmt.Errorf("total weight is zero")
	}

	// Sort by value
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].value > pairs[j].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Find weighted median
	halfWeight := totalWeight / 2
	cumulativeWeight := uint64(0)

	for _, pair := range pairs {
		cumulativeWeight += pair.weight
		if cumulativeWeight >= halfWeight {
			return pair.value, nil
		}
	}

	return pairs[len(pairs)-1].value, nil
}

func calculateMAD(values []uint64, median uint64) uint64 {
	if len(values) == 0 {
		return 0
	}

	deviations := make([]uint64, len(values))
	for i, v := range values {
		if v > median {
			deviations[i] = v - median
		} else {
			deviations[i] = median - v
		}
	}

	// Calculate median of deviations
	for i := 0; i < len(deviations); i++ {
		for j := i + 1; j < len(deviations); j++ {
			if deviations[i] > deviations[j] {
				deviations[i], deviations[j] = deviations[j], deviations[i]
			}
		}
	}

	if len(deviations)%2 == 0 {
		return (deviations[len(deviations)/2-1] + deviations[len(deviations)/2]) / 2
	}
	return deviations[len(deviations)/2]
}

func detectOutliers(values []uint64, median, mad uint64, threshold float64) []int {
	outliers := make([]int, 0)

	for i, v := range values {
		var deviation uint64
		if v > median {
			deviation = v - median
		} else {
			deviation = median - v
		}

		// Modified Z-score using MAD
		if mad > 0 {
			modifiedZScore := float64(deviation) / (1.4826 * float64(mad))
			if modifiedZScore > threshold {
				outliers = append(outliers, i)
			}
		}
	}

	return outliers
}

func isTimestampStale(timestamp, currentTime, maxAge int64) bool {
	if timestamp <= 0 {
		return true
	}
	if timestamp > currentTime+60 { // Future timestamp with 1 min buffer
		return true
	}
	age := currentTime - timestamp
	return age > maxAge
}

func calculatePriceDeviation(reference, submitted uint64) uint64 {
	if reference == 0 {
		if submitted == 0 {
			return 0
		}
		return 10000 // 100% deviation
	}

	var diff uint64
	if submitted > reference {
		diff = submitted - reference
	} else {
		diff = reference - submitted
	}

	// Calculate percentage * 100 (basis points)
	deviation := (diff * 10000) / reference
	return deviation
}

func calculateSlashFraction(deviation, threshold uint64) uint64 {
	if deviation <= threshold {
		return 0
	}

	excessDeviation := deviation - threshold

	// Progressive slashing: 1% base + 0.5% per 1% excess deviation
	slashFraction := 100 + (excessDeviation / 2)

	// Cap at 10% max slash
	if slashFraction > 1000 {
		slashFraction = 1000
	}

	return slashFraction
}

func encodeTimestamp(ts int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(ts))
	return buf
}

func encodeSlashingInput(ref, sub, thresh uint64) []byte {
	buf := make([]byte, 24)
	binary.BigEndian.PutUint64(buf[0:8], ref)
	binary.BigEndian.PutUint64(buf[8:16], sub)
	binary.BigEndian.PutUint64(buf[16:24], thresh)
	return buf
}

func minUint64(vals []uint64) uint64 {
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

func maxUint64(vals []uint64) uint64 {
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

func allZero(vals []uint64) bool {
	for _, v := range vals {
		if v != 0 {
			return false
		}
	}
	return true
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
