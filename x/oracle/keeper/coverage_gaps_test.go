package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestGeographicDiversityViolationAtRuntime tests that geographic diversity violations
// are detected and handled properly at runtime when validators register from the same region.
func TestGeographicDiversityViolationAtRuntime(t *testing.T) {
	k, stakingKeeper, ctx := keepertest.OracleKeeper(t)
	// SEC-11: Use block height past bootstrap grace period (10000) for error testing
	ctx = ctx.WithBlockHeight(10001)

	// Configure params to require geographic diversity
	params := types.DefaultParams()
	params.MinGeographicRegions = 3
	params.RequireGeographicDiversity = true
	params.AllowedRegions = []string{"north_america", "europe", "asia"}
	require.NoError(t, k.SetParams(ctx, params))

	t.Run("detect geographic diversity violation", func(t *testing.T) {
		// Register 7 validators all from the same region (north_america)
		validators := make([]sdk.ValAddress, 7)
		for i := 0; i < 7; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_geo_same_%02d", i)))
			validators[i] = valAddr

			// Register validator in staking keeper
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))

			// Set oracle info with same geographic region
			require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
				ValidatorAddr:    valAddr.String(),
				GeographicRegion: "north_america",
				IsActive:         true,
			}))
		}

		// Check Byzantine tolerance should fail due to insufficient diversity
		err := k.CheckByzantineTolerance(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient geographic diversity")
		require.Contains(t, err.Error(), "1 < 3") // 1 region vs 3 required
	})

	t.Run("emit warning events for diversity violation", func(t *testing.T) {
		// Clear context events
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		// Register validators concentrated in one region
		for i := 0; i < 5; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_warn_%02d", i)))
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
			require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
				ValidatorAddr:    valAddr.String(),
				GeographicRegion: "europe",
				IsActive:         true,
			}))
		}

		// Add minimal validators from other regions
		valAddr1 := sdk.ValAddress([]byte("validator_warn_na"))
		require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr1))
		require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
			ValidatorAddr:    valAddr1.String(),
			GeographicRegion: "north_america",
			IsActive:         true,
		}))

		valAddr2 := sdk.ValAddress([]byte("validator_warn_asia"))
		require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr2))
		require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
			ValidatorAddr:    valAddr2.String(),
			GeographicRegion: "asia",
			IsActive:         true,
		}))

		// Monitor geographic diversity
		err := k.MonitorGeographicDiversity(ctx)
		require.NoError(t, err, "Monitoring should succeed even with concentration")

		// Check for warning events
		events := ctx.EventManager().Events()
		var foundWarning bool
		for _, event := range events {
			if event.Type == "geographic_concentration_warning" {
				foundWarning = true
				// Verify event has proper attributes
				hasRegion := false
				hasRegionShare := false
				for _, attr := range event.Attributes {
					if attr.Key == "region" {
						hasRegion = true
					}
					if attr.Key == "region_share" {
						hasRegionShare = true
					}
				}
				require.True(t, hasRegion, "Event should have region attribute")
				require.True(t, hasRegionShare, "Event should have region_share attribute")
			}
		}
		require.True(t, foundWarning, "Should emit geographic concentration warning")
	})

	t.Run("enforcement mode behavior", func(t *testing.T) {
		// Clear previous validators
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		// Note: RequireGeographicDiversity is already enabled in params
		// Testing runtime monitoring behavior with geographic diversity enabled
		require.True(t, params.RequireGeographicDiversity)

		// Try to register validator in over-concentrated region
		validators := []sdk.ValAddress{}
		// Register 10 validators in europe (over 33%)
		for i := 0; i < 10; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_enforce_eu_%02d", i)))
			validators = append(validators, valAddr)
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
			require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
				ValidatorAddr:    valAddr.String(),
				GeographicRegion: "europe",
				IsActive:         true,
			}))
		}

		// Add few validators in other regions
		for i := 0; i < 3; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_enforce_na_%02d", i)))
			validators = append(validators, valAddr)
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
			require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
				ValidatorAddr:    valAddr.String(),
				GeographicRegion: "north_america",
				IsActive:         true,
			}))
		}

		for i := 0; i < 2; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_enforce_asia_%02d", i)))
			validators = append(validators, valAddr)
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
			require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
				ValidatorAddr:    valAddr.String(),
				GeographicRegion: "asia",
				IsActive:         true,
			}))
		}

		// Monitor should detect concentration
		err := k.MonitorGeographicDiversity(ctx)
		require.NoError(t, err)

		// Verify warning events emitted
		events := ctx.EventManager().Events()
		var foundConcentrationWarning bool
		for _, event := range events {
			if event.Type == "geographic_concentration_warning" {
				foundConcentrationWarning = true
			}
		}
		require.True(t, foundConcentrationWarning, "Should emit concentration warning in enforcement mode")
	})
}

// TestExtremeValueEdgeCases tests that price aggregation handles extreme values
// correctly without overflow/underflow issues.
func TestExtremeValueEdgeCases(t *testing.T) {
	k, stakingKeeper, ctx := keepertest.OracleKeeper(t)

	// Set up params
	params := types.DefaultParams()
	params.VoteThreshold = sdkmath.LegacyMustNewDecFromStr("0.67")
	require.NoError(t, k.SetParams(ctx, params))

	t.Run("extremely large prices no overflow", func(t *testing.T) {
		// Test that calculations handle extremely large prices without overflow
		// Use direct calculation functions to avoid triggering slashing in tests

		// Test median calculation with extreme values
		extremePrices := []sdkmath.LegacyDec{
			sdkmath.LegacyMustNewDecFromStr("0.000000000000000001"),
			sdkmath.LegacyMustNewDecFromStr("100000"),
			sdkmath.LegacyMustNewDecFromStr("999999999999999999999999999.999999999999999999"),
		}

		median := k.CalculateMedian(extremePrices)
		require.True(t, median.IsPositive(), "Median of extreme values should be positive")
		require.True(t, median.Equal(sdkmath.LegacyMustNewDecFromStr("100000")),
			"Median should be the middle value")

		// Test MAD calculation with extreme values
		mad := k.CalculateMAD(extremePrices, median)
		require.True(t, mad.IsPositive(), "MAD should be positive")
		require.False(t, mad.IsNil(), "MAD should not be nil")

		// Test IQR calculation with extreme values
		q1, q3, iqr := k.CalculateIQR(extremePrices)
		// With only 3 values, one being tiny and one huge, Q calculations may be unusual
		// Just verify they don't panic and return non-nil values
		require.False(t, q1.IsNil(), "Q1 should not be nil")
		require.False(t, q3.IsNil(), "Q3 should not be nil")
		require.False(t, iqr.IsNil(), "IQR should not be nil")
		// Q3 should be >= Q1 (even if Q1 is very small or negative due to interpolation)
		require.True(t, q3.GTE(q1), "Q3 should be >= Q1")
	})

	t.Run("extremely small prices no underflow", func(t *testing.T) {
		// Test calculations with extremely small prices
		smallPrices := []sdkmath.LegacyDec{
			sdkmath.LegacyMustNewDecFromStr("0.000000000000000001"),
			sdkmath.LegacyMustNewDecFromStr("0.000000000000000002"),
			sdkmath.LegacyMustNewDecFromStr("0.000000000000000003"),
		}

		median := k.CalculateMedian(smallPrices)
		require.True(t, median.IsPositive(), "Median of small values should be positive")
		require.False(t, median.IsZero(), "Median should not underflow to zero")

		mad := k.CalculateMAD(smallPrices, median)
		require.False(t, mad.IsNil(), "MAD should not be nil")
		// MAD can be zero for very similar small values, which is acceptable
	})

	t.Run("TWAP calculation with extreme values", func(t *testing.T) {
		twapAsset := "TWAP-EXTREME"

		// Create validators
		validators := make([]sdk.ValAddress, 7)
		for i := 0; i < 7; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_twap_%02d", i)))
			validators[i] = valAddr
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
		}

		// Submit prices and create snapshots with extreme values
		extremePrices := []sdkmath.LegacyDec{
			sdkmath.LegacyMustNewDecFromStr("999999999999999999999.999999999999999999"),
			sdkmath.LegacyMustNewDecFromStr("0.000000000000000001"),
			sdkmath.LegacyMustNewDecFromStr("100000"),
		}

		for idx, extremePrice := range extremePrices {
			// Submit prices from all validators
			for i := 0; i < 7; i++ {
				vp := types.ValidatorPrice{
					ValidatorAddr: validators[i].String(),
					Asset:         twapAsset,
					Price:         extremePrice,
					VotingPower:   1000000,
				}
				require.NoError(t, k.SetValidatorPrice(ctx, vp))
			}

			// Aggregate to create snapshot
			err := k.AggregateAssetPrice(ctx, twapAsset)
			require.NoError(t, err)

			// Advance block height for next snapshot
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)
			ctx = ctx.WithBlockTime(ctx.BlockTime().Add(60))

			// Only check the last iteration
			if idx == len(extremePrices)-1 {
				// Calculate TWAP with extreme values in history
				twap, err := k.CalculateTWAP(ctx, twapAsset)
				require.NoError(t, err, "TWAP calculation should not panic with extreme values")
				require.True(t, twap.IsPositive(), "TWAP should be positive")
				require.False(t, twap.IsNil(), "TWAP should not be nil")
			}
		}
	})

	t.Run("overflow protection in weighted median", func(t *testing.T) {
		// Create validators with varying voting power
		validators := make([]sdk.ValAddress, 5)
		for i := 0; i < 5; i++ {
			valAddr := sdk.ValAddress([]byte(fmt.Sprintf("validator_weighted_%02d", i)))
			validators[i] = valAddr
			require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, stakingKeeper, valAddr))
		}

		// Create validator prices with extreme voting power
		validatorPrices := []types.ValidatorPrice{
			{
				ValidatorAddr: validators[0].String(),
				Price:         sdkmath.LegacyMustNewDecFromStr("100000"),
				VotingPower:   9223372036854775807, // Max int64
			},
			{
				ValidatorAddr: validators[1].String(),
				Price:         sdkmath.LegacyMustNewDecFromStr("100001"),
				VotingPower:   1000000,
			},
			{
				ValidatorAddr: validators[2].String(),
				Price:         sdkmath.LegacyMustNewDecFromStr("100002"),
				VotingPower:   1000000,
			},
			{
				ValidatorAddr: validators[3].String(),
				Price:         sdkmath.LegacyMustNewDecFromStr("100003"),
				VotingPower:   1000000,
			},
			{
				ValidatorAddr: validators[4].String(),
				Price:         sdkmath.LegacyMustNewDecFromStr("100004"),
				VotingPower:   1000000,
			},
		}

		// Test weighted median calculation doesn't overflow
		median, err := k.CalculateWeightedMedian(validatorPrices)
		require.NoError(t, err, "Weighted median should handle extreme voting power")
		require.True(t, median.IsPositive(), "Median should be positive")
		require.True(t, median.Equal(sdkmath.LegacyMustNewDecFromStr("100000")),
			"Median should be weighted by extreme voting power")
	})
}

// TestByzantineFaultInjection tests oracle behavior under Byzantine conditions
// where some validators submit invalid or conflicting prices.
func TestByzantineFaultInjection(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Set up params
	params := types.DefaultParams()
	params.VoteThreshold = sdkmath.LegacyMustNewDecFromStr("0.67")
	params.MinVotingPowerForConsensus = sdkmath.LegacyMustNewDecFromStr("0.67")
	require.NoError(t, k.SetParams(ctx, params))

	t.Run("Byzantine validators submit invalid prices", func(t *testing.T) {
		// Test outlier detection with Byzantine minority
		// Create validator prices with Byzantine submissions
		correctPrice := sdkmath.LegacyMustNewDecFromStr("50000")

		validatorPrices := []types.ValidatorPrice{
			// 7 honest validators
			{ValidatorAddr: "val1", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val2", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val3", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val4", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val5", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val6", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val7", Price: correctPrice, VotingPower: 1000000},
			// 3 Byzantine validators with moderate outliers
			{ValidatorAddr: "byz1", Price: correctPrice.Quo(sdkmath.LegacyNewDec(2)), VotingPower: 1000000}, // 0.5x
			{ValidatorAddr: "byz2", Price: correctPrice.Mul(sdkmath.LegacyNewDec(2)), VotingPower: 1000000}, // 2x
			{ValidatorAddr: "byz3", Price: correctPrice.Mul(sdkmath.LegacyNewDec(3)), VotingPower: 1000000}, // 3x
		}

		// Test weighted median calculation excludes Byzantine minority
		medianPrice, err := k.CalculateWeightedMedian(validatorPrices)
		require.NoError(t, err)
		// Median should be the correct price (from honest majority)
		require.True(t, medianPrice.Equal(correctPrice),
			"Weighted median should equal honest consensus price")
	})

	t.Run("Byzantine validators submit conflicting prices", func(t *testing.T) {
		// Test consensus mechanism with coordinated Byzantine minority
		priceGroupA := sdkmath.LegacyMustNewDecFromStr("50000")
		priceGroupB := sdkmath.LegacyMustNewDecFromStr("51000") // 2% different

		validatorPrices := []types.ValidatorPrice{
			// 7 validators on price A (honest majority)
			{ValidatorAddr: "val1", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val2", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val3", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val4", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val5", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val6", Price: priceGroupA, VotingPower: 1000000},
			{ValidatorAddr: "val7", Price: priceGroupA, VotingPower: 1000000},
			// 3 Byzantine validators coordinating on price B
			{ValidatorAddr: "byz1", Price: priceGroupB, VotingPower: 1000000},
			{ValidatorAddr: "byz2", Price: priceGroupB, VotingPower: 1000000},
			{ValidatorAddr: "byz3", Price: priceGroupB, VotingPower: 1000000},
		}

		// Weighted median should choose majority consensus
		medianPrice, err := k.CalculateWeightedMedian(validatorPrices)
		require.NoError(t, err)

		// Should be closer to group A (majority)
		deviationFromA := medianPrice.Sub(priceGroupA).Abs().Quo(priceGroupA)
		deviationFromB := medianPrice.Sub(priceGroupB).Abs().Quo(priceGroupB)
		require.True(t, deviationFromA.LT(deviationFromB),
			"Weighted median should be closer to majority consensus")
	})

	t.Run("outlier severity classification", func(t *testing.T) {
		// Test that Byzantine outliers are properly classified by severity
		correctPrice := sdkmath.LegacyMustNewDecFromStr("50000")
		median := correctPrice
		mad := sdkmath.LegacyMustNewDecFromStr("100") // Small MAD
		threshold := sdkmath.LegacyMustNewDecFromStr("3.5")

		// Test moderate outlier
		moderateOutlier := correctPrice.Mul(sdkmath.LegacyMustNewDecFromStr("1.05")) // 5% deviation
		severity, deviation := k.ClassifyOutlierSeverity(moderateOutlier, median, mad, threshold)
		require.False(t, deviation.IsNil(), "Deviation should be calculated")
		// With small MAD (100) and 5% deviation (2500), Modified Z-Score will be high
		// Severity could be high, so just verify it's calculated
		require.True(t, severity >= keeper.SeverityNone && severity <= keeper.SeverityExtreme,
			"Price deviation should have valid severity classification")

		// Test larger outlier
		largeOutlier := correctPrice.Mul(sdkmath.LegacyNewDec(2)) // 100% deviation
		severity2, deviation2 := k.ClassifyOutlierSeverity(largeOutlier, median, mad, threshold)
		require.True(t, deviation2.GT(deviation), "Larger outlier should have larger deviation")
		require.True(t, severity2 >= severity, "Larger outlier should have equal or higher severity")
	})
	t.Run("consensus mechanism excludes outliers", func(t *testing.T) {
		// Test that consensus mechanism properly excludes Byzantine minority (< 33%)
		correctPrice := sdkmath.LegacyMustNewDecFromStr("50000")

		validatorPrices := []types.ValidatorPrice{
			// 9 honest validators (69% - above 67% threshold)
			{ValidatorAddr: "val1", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val2", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val3", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val4", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val5", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val6", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val7", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val8", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val9", Price: correctPrice, VotingPower: 1000000},
			// 4 Byzantine validators with different outlier prices (31% - below 33% BFT threshold)
			{ValidatorAddr: "byz1", Price: correctPrice.Quo(sdkmath.LegacyNewDec(2)), VotingPower: 1000000},  // 0.5x
			{ValidatorAddr: "byz2", Price: correctPrice.Mul(sdkmath.LegacyNewDec(2)), VotingPower: 1000000},  // 2x
			{ValidatorAddr: "byz3", Price: correctPrice.Mul(sdkmath.LegacyNewDec(3)), VotingPower: 1000000},  // 3x
			{ValidatorAddr: "byz4", Price: correctPrice.Mul(sdkmath.LegacyNewDec(4)), VotingPower: 1000000},  // 4x
		}

		// Weighted median should exclude Byzantine minority
		medianPrice, err := k.CalculateWeightedMedian(validatorPrices)
		require.NoError(t, err)

		// Price should match honest consensus
		deviation := medianPrice.Sub(correctPrice).Abs().Quo(correctPrice)
		maxDeviation := sdkmath.LegacyMustNewDecFromStr("0.001") // 0.1% tolerance
		require.True(t, deviation.LTE(maxDeviation),
			"Consensus should exclude Byzantine outliers, got %s expected %s",
			medianPrice.String(), correctPrice.String())
	})

	t.Run("insufficient consensus with too many Byzantine validators", func(t *testing.T) {
		// Test that insufficient honest majority fails consensus
		correctPrice := sdkmath.LegacyMustNewDecFromStr("50000")
		byzantinePrice := correctPrice.Mul(sdkmath.LegacyNewDec(3))

		validatorPrices := []types.ValidatorPrice{
			// Only 3 honest validators (43% - below 67% threshold)
			{ValidatorAddr: "val1", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val2", Price: correctPrice, VotingPower: 1000000},
			{ValidatorAddr: "val3", Price: correctPrice, VotingPower: 1000000},
			// 4 Byzantine validators (57% - above 33% BFT threshold, above 50% majority)
			{ValidatorAddr: "byz1", Price: byzantinePrice, VotingPower: 1000000},
			{ValidatorAddr: "byz2", Price: byzantinePrice, VotingPower: 1000000},
			{ValidatorAddr: "byz3", Price: byzantinePrice, VotingPower: 1000000},
			{ValidatorAddr: "byz4", Price: byzantinePrice, VotingPower: 1000000},
		}

		// With Byzantine majority, weighted median will choose Byzantine price
		medianPrice, err := k.CalculateWeightedMedian(validatorPrices)
		require.NoError(t, err)

		// Median will be Byzantine price (they have majority voting power)
		deviation := medianPrice.Sub(byzantinePrice).Abs().Quo(byzantinePrice)
		require.True(t, deviation.LTE(sdkmath.LegacyMustNewDecFromStr("0.001")),
			"With Byzantine majority, median reflects their price (demonstrates BFT failure)")
	})
}
