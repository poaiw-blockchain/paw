package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// createTestValidatorAddr creates a test validator address
func createTestValidatorAddr() sdk.ValAddress {
	return sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
}

// TestAggregation_VotingPowerThreshold tests that price aggregation requires
// minimum voting power threshold, preventing manipulation by low-stake validators
func TestAggregation_VotingPowerThreshold(t *testing.T) {
	t.Run("attack scenario: 3 validators with 1% stake each should fail", func(t *testing.T) {
		k, ctx := keepertest.OracleKeeper(t)

		// Set params with 10% minimum voting power
		params := types.DefaultParams()
		params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		// Create 3 low-stake validators (1M tokens each = 1 consensus power)
		val1 := createTestValidatorAddr()
		val2 := createTestValidatorAddr()
		val3 := createTestValidatorAddr()

		// Create 97 additional validators to make a total bonded power of 100
		// (so 3 validators = 3% of total power)
		for i := 0; i < 97; i++ {
			require.NoError(t, keepertest.EnsureBondedValidator(ctx, createTestValidatorAddr()))
		}

		// Ensure our 3 test validators are bonded
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val2))
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val3))

		// Create validator prices - each validator has 1 consensus power (total 3/100 = 3%)
		validatorPrices := []types.ValidatorPrice{
			{
				ValidatorAddr: val1.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50000),
				BlockHeight:   100,
				VotingPower:   1, // 1 consensus power
			},
			{
				ValidatorAddr: val2.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50100),
				BlockHeight:   100,
				VotingPower:   1, // 1 consensus power
			},
			{
				ValidatorAddr: val3.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(49900),
				BlockHeight:   100,
				VotingPower:   1, // 1 consensus power
			},
		}

		// Store validator prices
		for _, vp := range validatorPrices {
			err := k.SetValidatorPrice(ctx, vp)
			require.NoError(t, err, "Failed to set validator price")
		}

		// Attempt aggregation - should FAIL because 3% < 10% minimum voting power
		err = k.AggregateAssetPrice(ctx, "BTC")
		require.Error(t, err, "Attack should be prevented by voting power threshold")
		require.ErrorIs(t, err, types.ErrInsufficientOracleConsensus, "Should return insufficient consensus error")

		// Verify price was NOT set
		_, err = k.GetPrice(ctx, "BTC")
		require.Error(t, err, "Price should not be available after failed aggregation")
	})

	t.Run("legitimate scenario: validators with 15% stake should succeed", func(t *testing.T) {
		k, ctx := keepertest.OracleKeeper(t)

		// Set params with 10% minimum voting power
		params := types.DefaultParams()
		params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		// Create 2 validators
		val1 := createTestValidatorAddr()
		val2 := createTestValidatorAddr()

		// Ensure validators are bonded
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val2))

		// Create validator prices with 8% and 7% voting power (total 15%)
		validatorPrices := []types.ValidatorPrice{
			{
				ValidatorAddr: val1.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50000),
				BlockHeight:   100,
				VotingPower:   8, // 8% of total power
			},
			{
				ValidatorAddr: val2.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50100),
				BlockHeight:   100,
				VotingPower:   7, // 7% of total power
			},
		}

		// Store validator prices
		for _, vp := range validatorPrices {
			err := k.SetValidatorPrice(ctx, vp)
			require.NoError(t, err)
		}

		// Attempt aggregation - should SUCCEED because 15% >= 10%
		err = k.AggregateAssetPrice(ctx, "BTC")
		require.NoError(t, err, "Legitimate scenario with sufficient voting power should succeed")

		// Verify price was set
		price, err := k.GetPrice(ctx, "BTC")
		require.NoError(t, err, "Price should be available after successful aggregation")
		require.True(t, price.Price.GT(math.LegacyZeroDec()), "Price should be positive")
	})

	t.Run("edge case: exactly 10% voting power should succeed", func(t *testing.T) {
		k, ctx := keepertest.OracleKeeper(t)

		// Set params with 10% minimum voting power
		params := types.DefaultParams()
		params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		val1 := createTestValidatorAddr()
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))

		validatorPrices := []types.ValidatorPrice{
			{
				ValidatorAddr: val1.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50000),
				BlockHeight:   100,
				VotingPower:   10, // Exactly 10% of total power
			},
		}

		for _, vp := range validatorPrices {
			err := k.SetValidatorPrice(ctx, vp)
			require.NoError(t, err)
		}

		// Should succeed with exactly 10%
		err = k.AggregateAssetPrice(ctx, "BTC")
		require.NoError(t, err, "Edge case: exactly minimum threshold should succeed")

		price, err := k.GetPrice(ctx, "BTC")
		require.NoError(t, err)
		require.True(t, price.Price.GT(math.LegacyZeroDec()))
	})

	t.Run("edge case: just below 10% voting power should fail", func(t *testing.T) {
		k, ctx := keepertest.OracleKeeper(t)

		// Set params with 10% minimum voting power
		params := types.DefaultParams()
		params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		val1 := createTestValidatorAddr()
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))

		validatorPrices := []types.ValidatorPrice{
			{
				ValidatorAddr: val1.String(),
				Asset:         "BTC",
				Price:         math.LegacyNewDec(50000),
				BlockHeight:   100,
				VotingPower:   9, // Just below 10%
			},
		}

		for _, vp := range validatorPrices {
			err := k.SetValidatorPrice(ctx, vp)
			require.NoError(t, err)
		}

		// Should fail with 9% < 10%
		err = k.AggregateAssetPrice(ctx, "BTC")
		require.Error(t, err, "Edge case: just below threshold should fail")
		require.ErrorIs(t, err, types.ErrInsufficientOracleConsensus)

		_, err = k.GetPrice(ctx, "BTC")
		require.Error(t, err)
	})
}

// TestAggregation_VotingPowerAfterOutlierFiltering tests that voting power
// threshold is checked AFTER outlier filtering, not before
func TestAggregation_VotingPowerAfterOutlierFiltering(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params with 10% minimum voting power
	params := types.DefaultParams()
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 5 validators
	legit1 := createTestValidatorAddr()
	legit2 := createTestValidatorAddr()
	mal1 := createTestValidatorAddr()
	mal2 := createTestValidatorAddr()
	mal3 := createTestValidatorAddr()

	// Ensure all are bonded
	for _, val := range []sdk.ValAddress{legit1, legit2, mal1, mal2, mal3} {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val))
	}

	// Scenario: 2 high-stake validators with legitimate prices (30% total power)
	// 3 low-stake validators with outlier prices (3% total power)
	// After outlier filtering, only high-stake validators remain (30% > 10%)
	validatorPrices := []types.ValidatorPrice{
		// Legitimate validators with high stake
		{
			ValidatorAddr: legit1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50000),
			BlockHeight:   100,
			VotingPower:   15, // 15% of total power
		},
		{
			ValidatorAddr: legit2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50100),
			BlockHeight:   100,
			VotingPower:   15, // 15% of total power
		},
		// Malicious validators with low stake and outlier prices
		{
			ValidatorAddr: mal1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(100000), // 2x outlier
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
		{
			ValidatorAddr: mal2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(25000), // 0.5x outlier
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
		{
			ValidatorAddr: mal3.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(150000), // 3x outlier
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
	}

	// Store validator prices
	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should succeed because after filtering outliers, remaining validators have 30% power
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.NoError(t, err, "Should succeed after outlier filtering leaves sufficient voting power")

	// Verify price was set and is based on legitimate validators
	price, err := k.GetPrice(ctx, "BTC")
	require.NoError(t, err)
	require.True(t, price.Price.GT(math.LegacyNewDec(49000)), "Price should be close to legitimate submissions")
	require.True(t, price.Price.LT(math.LegacyNewDec(51000)), "Price should be close to legitimate submissions")
}

// TestAggregation_NoVotingPowerManipulation tests that the fix prevents
// the specific attack scenario: 3 colluding low-stake validators
func TestAggregation_NoVotingPowerManipulation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set default params (10% minimum voting power)
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 3 colluding validators
	attacker1 := createTestValidatorAddr()
	attacker2 := createTestValidatorAddr()
	attacker3 := createTestValidatorAddr()

	// Ensure all are bonded
	for _, val := range []sdk.ValAddress{attacker1, attacker2, attacker3} {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val))
	}

	// Attack scenario: 3 colluding validators with 1% stake each
	// trying to set manipulated price
	attackValidators := []types.ValidatorPrice{
		{
			ValidatorAddr: attacker1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(1000000), // Manipulated price (20x)
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
		{
			ValidatorAddr: attacker2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(1000100), // Manipulated price (close to first)
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
		{
			ValidatorAddr: attacker3.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(999900), // Manipulated price (close to first)
			BlockHeight:   100,
			VotingPower:   1, // 1% of total power
		},
	}

	// Store attack validator prices
	for _, vp := range attackValidators {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should FAIL because 3% < 10% minimum voting power
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.Error(t, err, "Attack should be prevented by voting power threshold")
	require.ErrorIs(t, err, types.ErrInsufficientOracleConsensus, "Should return insufficient consensus error")

	// Verify price was NOT set
	_, err = k.GetPrice(ctx, "BTC")
	require.Error(t, err, "Price should not be available after failed aggregation")
}

// TestAggregation_ZeroVotingPowerHandling tests edge case handling
func TestAggregation_ZeroVotingPowerHandling(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	val1 := createTestValidatorAddr()
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))

	// Test with validator that has zero voting power (edge case)
	validatorPrices := []types.ValidatorPrice{
		{
			ValidatorAddr: val1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50000),
			BlockHeight:   100,
			VotingPower:   0, // Zero voting power
		},
	}

	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should fail due to zero voting power
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.Error(t, err, "Zero voting power should fail threshold check")
}
