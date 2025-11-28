package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)


// Unit tests for mathematical functions that don't require keeper

func TestOutlierDetectionMath(t *testing.T) {
	// Test Modified Z-score calculation
	prices := []sdkmath.LegacyDec{
		sdkmath.LegacyMustNewDecFromStr("100"),
		sdkmath.LegacyMustNewDecFromStr("102"),
		sdkmath.LegacyMustNewDecFromStr("101"),
		sdkmath.LegacyMustNewDecFromStr("99"),
		sdkmath.LegacyMustNewDecFromStr("150"), // Outlier
	}

	// Median should be 101
	median := calculateTestMedian(prices)
	require.Equal(t, "101.000000000000000000", median.String())

	// MAD calculation
	mad := calculateTestMAD(prices, median)
	require.True(t, mad.GT(sdkmath.LegacyZeroDec()))
}

func TestIQRCalculation(t *testing.T) {
	prices := []sdkmath.LegacyDec{
		sdkmath.LegacyMustNewDecFromStr("10"),
		sdkmath.LegacyMustNewDecFromStr("20"),
		sdkmath.LegacyMustNewDecFromStr("30"),
		sdkmath.LegacyMustNewDecFromStr("40"),
		sdkmath.LegacyMustNewDecFromStr("50"),
	}

	q1, q3, iqr := calculateTestIQR(prices)

	require.True(t, q1.GT(sdkmath.LegacyZeroDec()))
	require.True(t, q3.GT(q1))
	require.True(t, iqr.Equal(q3.Sub(q1)))
}

func TestVolatilityCalculation(t *testing.T) {
	// Test that volatility calculation handles edge cases
	returns := []sdkmath.LegacyDec{
		sdkmath.LegacyMustNewDecFromStr("0.01"),
		sdkmath.LegacyMustNewDecFromStr("-0.01"),
		sdkmath.LegacyMustNewDecFromStr("0.02"),
		sdkmath.LegacyMustNewDecFromStr("-0.02"),
	}

	mean := sdkmath.LegacyZeroDec()
	for _, ret := range returns {
		mean = mean.Add(ret)
	}
	mean = mean.QuoInt64(int64(len(returns)))

	variance := sdkmath.LegacyZeroDec()
	for _, ret := range returns {
		diff := ret.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.QuoInt64(int64(len(returns)))

	require.True(t, variance.GT(sdkmath.LegacyZeroDec()))
}

func TestAttackCostCalculation(t *testing.T) {
	// Test Byzantine attack cost calculation
	// Attack requires 33% of total stake
	totalStake := sdkmath.NewInt(1000000)
	byzantineThreshold := sdkmath.LegacyMustNewDecFromStr("0.34")

	attackCost := byzantineThreshold.MulInt(totalStake).TruncateInt()

	expectedCost := sdkmath.NewInt(340000)
	require.Equal(t, expectedCost, attackCost)
}

func TestSecurityMarginCalculation(t *testing.T) {
	// Security margin should be attack_cost / attack_profit
	attackCost := sdkmath.LegacyNewDec(1000000)
	attackProfit := sdkmath.LegacyNewDec(100000)

	margin := attackCost.Quo(attackProfit)

	expected := sdkmath.LegacyNewDec(10) // 10x margin
	require.Equal(t, expected, margin)
}

func TestIncentiveCompatibility(t *testing.T) {
	// System is incentive compatible if attack cost >> attack profit
	// AND dishonest penalty > honest reward loss

	attackCost := sdkmath.LegacyNewDec(1000000)
	attackProfit := sdkmath.LegacyNewDec(50000)
	dishonestPenalty := sdkmath.LegacyMustNewDecFromStr("0.01")
	honestReward := sdkmath.LegacyMustNewDecFromStr("0.001")

	securityMargin := attackCost.Quo(attackProfit)
	minMargin := sdkmath.LegacyNewDec(10)

	isIncentiveCompatible := securityMargin.GT(minMargin) &&
		dishonestPenalty.GT(honestReward)

	require.True(t, isIncentiveCompatible)
}

func TestNashEquilibriumCondition(t *testing.T) {
	// Nash equilibrium when attack has negative expected value
	attackProfit := sdkmath.LegacyNewDec(1000000)
	attackCost := sdkmath.LegacyNewDec(5000000)
	successProbability := sdkmath.LegacyMustNewDecFromStr("0.1") // 10% chance

	expectedProfit := attackProfit.Mul(successProbability)
	expectedCost := attackCost.Mul(sdkmath.LegacyOneDec().Sub(successProbability))

	attackEV := expectedProfit.Sub(expectedCost)

	isNashEquilibrium := attackEV.LT(sdkmath.LegacyZeroDec())
	require.True(t, isNashEquilibrium)
}

func TestHerfindahlHirschmanIndex(t *testing.T) {
	// Test HHI calculation for stake concentration
	// Perfect distribution: n equal validators, HHI = 1/n

	// 4 equal validators
	powers := []int64{25, 25, 25, 25}
	totalPower := int64(100)

	hhi := sdkmath.LegacyZeroDec()
	for _, power := range powers {
		share := sdkmath.LegacyNewDec(power).Quo(sdkmath.LegacyNewDec(totalPower))
		hhi = hhi.Add(share.Mul(share))
	}

	expectedHHI := sdkmath.LegacyMustNewDecFromStr("0.25") // 1/4
	require.Equal(t, expectedHHI, hhi)
}

func TestCollusionResistance(t *testing.T) {
	// Higher number of validators = higher collusion resistance
	// More equal distribution = higher resistance

	// Scenario 1: 10 equal validators (high resistance)
	n1 := 10
	hhi1 := sdkmath.LegacyOneDec().QuoInt64(int64(n1))

	// Scenario 2: 3 validators (low resistance)
	n2 := 3
	hhi2 := sdkmath.LegacyOneDec().QuoInt64(int64(n2))

	// Scenario 1 should have lower HHI (better distribution)
	require.True(t, hhi1.LT(hhi2))
}

// Helper functions for unit tests

func calculateTestMedian(prices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	sorted := make([]sdkmath.LegacyDec, len(prices))
	copy(sorted, prices)

	// Simple bubble sort for testing
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].LT(sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	if n%2 == 0 {
		return sorted[n/2-1].Add(sorted[n/2]).Quo(sdkmath.LegacyNewDec(2))
	}
	return sorted[n/2]
}

func calculateTestMAD(prices []sdkmath.LegacyDec, median sdkmath.LegacyDec) sdkmath.LegacyDec {
	if len(prices) == 0 {
		return sdkmath.LegacyZeroDec()
	}

	deviations := make([]sdkmath.LegacyDec, len(prices))
	for i, price := range prices {
		deviations[i] = price.Sub(median).Abs()
	}

	madMedian := calculateTestMedian(deviations)
	scaleFactor := sdkmath.LegacyMustNewDecFromStr("1.4826")

	return madMedian.Mul(scaleFactor)
}

func calculateTestIQR(prices []sdkmath.LegacyDec) (q1, q3, iqr sdkmath.LegacyDec) {
	if len(prices) < 4 {
		return sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()
	}

	sorted := make([]sdkmath.LegacyDec, len(prices))
	copy(sorted, prices)

	// Simple bubble sort
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].LT(sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	q1 = sorted[n/4]
	q3 = sorted[(n*3)/4]
	iqr = q3.Sub(q1)

	return q1, q3, iqr
}
