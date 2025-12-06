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

// TestAggregation_NoVotingPowerManipulation tests the core security fix:
// 3 colluding low-stake validators cannot set price even if they pass outlier detection
func TestAggregation_NoVotingPowerManipulation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params with 10% minimum voting power (default)
	params := types.DefaultParams()
	// Lower vote threshold so test can focus on voting power check
	params.VoteThreshold = math.LegacyMustNewDecFromStr("0.01") // Only need 1% to attempt aggregation
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 3 low-stake validators (1M tokens each = 1 consensus power)
	val1 := createTestValidatorAddr()
	val2 := createTestValidatorAddr()
	val3 := createTestValidatorAddr()

	// Create 97 additional validators to make total bonded power of 100
	// (so 3 validators = 3% of total power < 10% threshold)
	for i := 0; i < 97; i++ {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, createTestValidatorAddr()))
	}

	// Ensure our 3 test validators are bonded
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val2))
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val3))

	// Attack scenario: 3 colluding validators with 1% stake each (3% total)
	// trying to set a manipulated price that bypasses outlier detection
	// (they submit similar prices to avoid being detected as outliers)
	attackValidators := []types.ValidatorPrice{
		{
			ValidatorAddr: val1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(1000000), // Manipulated price (20x real)
			BlockHeight:   100,
			VotingPower:   1, // 1 consensus power
		},
		{
			ValidatorAddr: val2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(1000100), // Close to first (avoid outlier detection)
			BlockHeight:   100,
			VotingPower:   1, // 1 consensus power
		},
		{
			ValidatorAddr: val3.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(999900), // Close to first (avoid outlier detection)
			BlockHeight:   100,
			VotingPower:   1, // 1 consensus power
		},
	}

	// Store attack validator prices
	for _, vp := range attackValidators {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Attempt aggregation - should FAIL because 3% < 10% minimum voting power
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.Error(t, err, "Attack should be prevented by voting power threshold")
	require.ErrorIs(t, err, types.ErrInsufficientOracleConsensus, "Should return insufficient consensus error")

	// Verify price was NOT set
	_, err = k.GetPrice(ctx, "BTC")
	require.Error(t, err, "Price should not be available after failed aggregation")
}

// TestAggregation_VotingPowerThreshold_Legitimate tests that legitimate
// validators with sufficient voting power can still set prices
func TestAggregation_VotingPowerThreshold_Legitimate(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.VoteThreshold = math.LegacyMustNewDecFromStr("0.01") // Only need 1% to attempt
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10") // 10% minimum
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 2 validators (will have 10% of total power)
	val1 := createTestValidatorAddr()
	val2 := createTestValidatorAddr()

	// Create 18 additional validators for total of 20
	// 2/20 = 10% exactly at threshold
	for i := 0; i < 18; i++ {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, createTestValidatorAddr()))
	}

	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val2))

	// Legitimate validators with 10% voting power (should succeed)
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
	}

	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should succeed because 2/20 = 10% >= 10% threshold
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.NoError(t, err, "Legitimate validators with exactly 10% should succeed")

	// Verify price was set
	price, err := k.GetPrice(ctx, "BTC")
	require.NoError(t, err)
	require.True(t, price.Price.GT(math.LegacyZeroDec()))
}

// TestAggregation_VotingPowerThreshold_JustBelow tests edge case just below threshold
func TestAggregation_VotingPowerThreshold_JustBelow(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.VoteThreshold = math.LegacyMustNewDecFromStr("0.01")
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 1 validator
	val1 := createTestValidatorAddr()

	// Create 10 additional validators for total of 11
	// 1/11 = 9.09% which is just below 10% threshold
	for i := 0; i < 10; i++ {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, createTestValidatorAddr()))
	}

	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))

	validatorPrices := []types.ValidatorPrice{
		{
			ValidatorAddr: val1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50000),
			BlockHeight:   100,
			VotingPower:   1, // 1/11 = 9.09%
		},
	}

	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should fail because 9.09% < 10%
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.Error(t, err, "Just below threshold should fail")
	require.ErrorIs(t, err, types.ErrInsufficientOracleConsensus)

	_, err = k.GetPrice(ctx, "BTC")
	require.Error(t, err)
}

// TestAggregation_VotingPowerThreshold_HighStake tests that high-stake validator succeeds
func TestAggregation_VotingPowerThreshold_HighStake(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.VoteThreshold = math.LegacyMustNewDecFromStr("0.01")
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 5 validators with one having high stake
	val1 := createTestValidatorAddr()

	// Create 4 additional validators for total of 5
	// 1/5 = 20% which is well above 10% threshold
	for i := 0; i < 4; i++ {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, createTestValidatorAddr()))
	}

	require.NoError(t, keepertest.EnsureBondedValidator(ctx, val1))

	validatorPrices := []types.ValidatorPrice{
		{
			ValidatorAddr: val1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50000),
			BlockHeight:   100,
			VotingPower:   1, // 1/5 = 20%
		},
	}

	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should succeed because 20% >= 10%
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.NoError(t, err, "High-stake validator should succeed")

	price, err := k.GetPrice(ctx, "BTC")
	require.NoError(t, err)
	require.True(t, price.Price.GT(math.LegacyZeroDec()))
}

// TestAggregation_VotingPowerAfterOutlierFiltering tests that voting power
// threshold is checked AFTER outlier filtering
func TestAggregation_VotingPowerAfterOutlierFiltering(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set params
	params := types.DefaultParams()
	params.VoteThreshold = math.LegacyMustNewDecFromStr("0.01")
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create 2 legitimate + 5 malicious validators = 7 total
	// This ensures we have enough validators (>= 5) for outlier detection to run
	// 2/7 = 28.6% legitimate validators (will pass after outlier filtering)
	// 5/7 = 71.4% malicious validators (will be filtered out)
	legit1 := createTestValidatorAddr()
	legit2 := createTestValidatorAddr()
	mal1 := createTestValidatorAddr()
	mal2 := createTestValidatorAddr()
	mal3 := createTestValidatorAddr()
	mal4 := createTestValidatorAddr()
	mal5 := createTestValidatorAddr()

	// Ensure all are bonded
	for _, val := range []sdk.ValAddress{legit1, legit2, mal1, mal2, mal3, mal4, mal5} {
		require.NoError(t, keepertest.EnsureBondedValidator(ctx, val))
	}

	// Scenario: 2 legitimate validators (28.6% of total power)
	// 5 malicious validators with outlier prices (71.4% of total power)
	// After outlier filtering, only legitimate validators remain (28.6% > 10%)
	validatorPrices := []types.ValidatorPrice{
		// Legitimate validators with similar prices
		{
			ValidatorAddr: legit1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50000),
			BlockHeight:   100,
			VotingPower:   1,
		},
		{
			ValidatorAddr: legit2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(50100),
			BlockHeight:   100,
			VotingPower:   1,
		},
		// Malicious validators with extreme outlier prices (will be filtered)
		{
			ValidatorAddr: mal1.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(100000), // 2x outlier
			BlockHeight:   100,
			VotingPower:   1,
		},
		{
			ValidatorAddr: mal2.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(25000), // 0.5x outlier
			BlockHeight:   100,
			VotingPower:   1,
		},
		{
			ValidatorAddr: mal3.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(150000), // 3x outlier
			BlockHeight:   100,
			VotingPower:   1,
		},
		{
			ValidatorAddr: mal4.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(200000), // 4x outlier
			BlockHeight:   100,
			VotingPower:   1,
		},
		{
			ValidatorAddr: mal5.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(10000), // 0.2x outlier
			BlockHeight:   100,
			VotingPower:   1,
		},
	}

	// Store validator prices
	for _, vp := range validatorPrices {
		err := k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Should succeed because after filtering outliers, remaining validators have 28.6% power
	err = k.AggregateAssetPrice(ctx, "BTC")
	require.NoError(t, err, "Should succeed after outlier filtering leaves sufficient voting power")

	// Verify price was set and is based on legitimate validators
	price, err := k.GetPrice(ctx, "BTC")
	require.NoError(t, err)
	require.True(t, price.Price.GT(math.LegacyNewDec(49000)), "Price should be close to legitimate submissions")
	require.True(t, price.Price.LT(math.LegacyNewDec(51000)), "Price should be close to legitimate submissions")
}
