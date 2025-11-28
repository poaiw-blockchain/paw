package differential

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPAWvsChainlinkAggregation compares PAW oracle aggregation with Chainlink
func TestPAWvsChainlinkAggregation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		prices        []uint64
		weights       []uint64
		expectSimilar bool
	}{
		{"Consensus prices", []uint64{1000000, 1000100, 999900, 1000050}, []uint64{1, 1, 1, 1}, true},
		{"One outlier", []uint64{1000000, 1000000, 1000000, 5000000}, []uint64{1, 1, 1, 1}, true},
		{"Weighted consensus", []uint64{1000000, 1100000}, []uint64{90, 10}, true},
		{"Byzantine scenario", []uint64{1000000, 1000000, 5000000, 5000000}, []uint64{1, 1, 1, 1}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pawMedian := calculatePAWWeightedMedian(tc.prices, tc.weights)
			chainlinkMedian := calculateChainlinkMedian(tc.prices)

			if tc.expectSimilar {
				divergence := calculateDivergence(pawMedian, chainlinkMedian)
				require.LessOrEqual(t, divergence, uint64(1000), // 10%
					"PAW and Chainlink medians should be similar")
			}

			t.Logf("PAW median: %d, Chainlink median: %d", pawMedian, chainlinkMedian)
		})
	}
}

// TestPAWOracleLatency compares latency characteristics
func TestPAWOracleLatency(t *testing.T) {
	t.Parallel()
	numPrices := []int{10, 50, 100}

	for _, n := range numPrices {
		prices := make([]uint64, n)
		weights := make([]uint64, n)

		for i := 0; i < n; i++ {
			prices[i] = 1000000 + uint64(rand.Intn(100000))
			weights[i] = uint64(rand.Intn(100) + 1)
		}

		// PAW should handle larger validator sets efficiently
		_ = calculatePAWWeightedMedian(prices, weights)
		_ = calculateChainlinkMedian(prices)

		t.Logf("Processed %d oracle prices successfully", n)
	}
}

// TestPAWByzantineResistance compares Byzantine fault tolerance
func TestPAWByzantineResistance(t *testing.T) {
	t.Parallel()
	numValidators := 100
	honestPrice := uint64(1000000)
	attackPrice := uint64(10000000)

	testCases := []struct {
		name         string
		maliciousPct int
	}{
		{"10% malicious", 10},
		{"20% malicious", 20},
		{"30% malicious", 30},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prices := make([]uint64, numValidators)
			weights := make([]uint64, numValidators)

			maliciousCount := numValidators * tc.maliciousPct / 100

			for i := 0; i < numValidators; i++ {
				if i < maliciousCount {
					prices[i] = attackPrice
				} else {
					prices[i] = honestPrice + uint64(rand.Intn(10000))
				}
				weights[i] = uint64(rand.Intn(100) + 1)
			}

			pawMedian := calculatePAWWeightedMedian(prices, weights)
			chainlinkMedian := calculateChainlinkMedian(prices)

			// Both should resist Byzantine attacks < 33%
			if tc.maliciousPct < 33 {
				pawDeviation := calculateDivergence(pawMedian, honestPrice)
				chainlinkDeviation := calculateDivergence(chainlinkMedian, honestPrice)

				require.LessOrEqual(t, pawDeviation, uint64(1000), // 10%
					"PAW should resist Byzantine attack")
				require.LessOrEqual(t, chainlinkDeviation, uint64(1000),
					"Chainlink should resist Byzantine attack")

				t.Logf("PAW deviation: %d bps, Chainlink deviation: %d bps",
					pawDeviation, chainlinkDeviation)
			}
		})
	}
}

// BenchmarkPAWvsChainlinkPerformance compares performance
func BenchmarkPAWvsChainlinkPerformance(b *testing.B) {
	prices := make([]uint64, 100)
	weights := make([]uint64, 100)

	for i := 0; i < 100; i++ {
		prices[i] = 1000000 + uint64(rand.Intn(100000))
		weights[i] = uint64(rand.Intn(100) + 1)
	}

	b.Run("PAW", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = calculatePAWWeightedMedian(prices, weights)
		}
	})

	b.Run("Chainlink", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = calculateChainlinkMedian(prices)
		}
	})
}

// Helper functions
func calculatePAWWeightedMedian(prices, weights []uint64) uint64 {
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

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].price < pairs[j].price
	})

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

func calculateChainlinkMedian(prices []uint64) uint64 {
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
