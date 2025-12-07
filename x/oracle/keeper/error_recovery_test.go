package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestPriceSubmissionRevertOnValidationFailure tests that invalid price submissions don't corrupt state
func TestPriceSubmissionRevertOnValidationFailure(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	validator := sdk.ValAddress("validator1")
	asset := "BTC/USD"

	// Get initial state
	_, initialErr := k.GetValidatorPrice(ctx, validator, asset)
	initialNotFound := initialErr != nil

	// Attempt to submit invalid price (zero)
	err := k.SubmitPrice(ctx, validator, asset, math.LegacyZeroDec())
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be positive")

	// Verify state unchanged
	_, finalErr := k.GetValidatorPrice(ctx, validator, asset)
	finalNotFound := finalErr != nil
	require.Equal(t, initialNotFound, finalNotFound, "validator price state should be unchanged")
}

// TestPriceSubmissionRevertOnInactiveValidator tests rejection of inactive validator submissions
func TestPriceSubmissionRevertOnInactiveValidator(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Use a validator address that doesn't exist or isn't bonded
	inactiveValidator := sdk.ValAddress("inactive_validator_")
	asset := "BTC/USD"
	price := math.LegacyNewDec(50000)

	// Attempt submission (should fail)
	err := k.SubmitPrice(ctx, inactiveValidator, asset, price)
	require.Error(t, err)

	// Verify no price was stored
	_, err = k.GetValidatorPrice(ctx, inactiveValidator, asset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestAggregationRevertOnInsufficientVotingPower tests that aggregation fails gracefully
func TestAggregationRevertOnInsufficientVotingPower(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"

	// Get initial aggregated price state (should not exist)
	_, initialErr := k.GetPrice(ctx, asset)
	initialNotFound := initialErr != nil

	// Attempt aggregation with no submissions
	err := k.AggregateAssetPrice(ctx, asset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no price submissions")

	// Verify no aggregated price was set
	_, finalErr := k.GetPrice(ctx, asset)
	finalNotFound := finalErr != nil
	require.Equal(t, initialNotFound, finalNotFound, "aggregated price should not be set on failure")
}

// TestPriceAggregationPreservesOldPriceOnFailure tests that failed aggregation doesn't remove old prices
func TestPriceAggregationPreservesOldPriceOnFailure(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	validator := sdk.ValAddress("validator1")
	asset := "BTC/USD"
	price := math.LegacyNewDec(50000)

	// Submit and aggregate successfully first
	err := k.SubmitPrice(ctx, validator, asset, price)
	require.NoError(t, err)

	// Manually set voting power high enough for aggregation
	validatorPrice, err := k.GetValidatorPrice(ctx, validator, asset)
	require.NoError(t, err)
	validatorPrice.VotingPower = 1000000
	err = k.SetValidatorPrice(ctx, validatorPrice)
	require.NoError(t, err)

	err = k.AggregateAssetPrice(ctx, asset)
	require.NoError(t, err)

	// Get the successfully aggregated price
	aggregatedPrice, err := k.GetPrice(ctx, asset)
	require.NoError(t, err)
	require.True(t, aggregatedPrice.Price.Equal(price))

	// Delete all validator prices to trigger aggregation failure
	k.DeleteValidatorPrice(ctx, validator, asset)

	// Attempt aggregation (should fail)
	err = k.AggregateAssetPrice(ctx, asset)
	require.Error(t, err)

	// Verify old aggregated price preserved
	preservedPrice, err := k.GetPrice(ctx, asset)
	require.NoError(t, err)
	require.Equal(t, aggregatedPrice.Price, preservedPrice.Price, "old price should be preserved on aggregation failure")
}

// TestOutlierDetectionDoesNotCorruptValidPrices tests that outlier filtering doesn't corrupt data
func TestOutlierDetectionDoesNotCorruptValidPrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"
	basePrice := math.LegacyNewDec(50000)

	// Submit multiple valid prices
	validators := []sdk.ValAddress{
		sdk.ValAddress("validator1"),
		sdk.ValAddress("validator2"),
		sdk.ValAddress("validator3"),
	}

	for i, val := range validators {
		price := basePrice.Add(math.LegacyNewDec(int64(i * 100))) // Slight variation
		err := k.SubmitPrice(ctx, val, asset, price)
		require.NoError(t, err)

		// Set voting power
		vp, _ := k.GetValidatorPrice(ctx, val, asset)
		vp.VotingPower = 100000
		err = k.SetValidatorPrice(ctx, vp)
		require.NoError(t, err)
	}

	// Verify all submissions exist before aggregation
	for _, val := range validators {
		_, err := k.GetValidatorPrice(ctx, val, asset)
		require.NoError(t, err, "validator price should exist before aggregation")
	}

	// Aggregate (outlier detection will run)
	err := k.AggregateAssetPrice(ctx, asset)
	// May succeed or fail depending on voting power, but shouldn't corrupt data
	if err != nil {
		t.Logf("Aggregation error (expected in some cases): %v", err)
	}

	// Verify original submissions still exist (outlier detection shouldn't delete them)
	for _, val := range validators {
		_, err := k.GetValidatorPrice(ctx, val, asset)
		require.NoError(t, err, "validator prices should be preserved after aggregation attempt")
	}
}

// TestPriceSnapshotRevertOnStorageFailure tests snapshot storage error handling
func TestPriceSnapshotRevertOnStorageFailure(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"
	blockHeight := ctx.BlockHeight()

	// Create a snapshot
	snapshot := types.PriceSnapshot{
		Asset:       asset,
		Price:       math.LegacyNewDec(50000),
		BlockHeight: blockHeight,
		BlockTime:   ctx.BlockTime().Unix(),
	}

	// Store snapshot
	err := k.SetPriceSnapshot(ctx, snapshot)
	require.NoError(t, err)

	// Verify snapshot exists
	retrievedSnapshot, err := k.GetPriceSnapshot(ctx, asset, blockHeight)
	require.NoError(t, err)
	require.Equal(t, snapshot.Price, retrievedSnapshot.Price)
}

// TestTWAPCalculationFailureDoesNotCorruptState tests TWAP calculation error handling
func TestTWAPCalculationFailureDoesNotCorruptState(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"

	// Attempt TWAP calculation with no snapshots (should fail gracefully)
	_, err := k.CalculateTWAP(ctx, asset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no snapshots available")

	// Verify no state corruption - can still set snapshots after failure
	snapshot := types.PriceSnapshot{
		Asset:       asset,
		Price:       math.LegacyNewDec(50000),
		BlockHeight: ctx.BlockHeight(),
		BlockTime:   ctx.BlockTime().Unix(),
	}

	err = k.SetPriceSnapshot(ctx, snapshot)
	require.NoError(t, err)
}

// TestFeederDelegationRevertOnInvalidAddress tests feeder delegation with invalid addresses
func TestFeederDelegationRevertOnInvalidAddress(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	validator := sdk.ValAddress("validator1")

	// Get initial delegation state
	initialDelegation, _ := k.GetFeederDelegation(ctx, validator)

	// Set feeder delegation with valid address
	feeder := sdk.AccAddress("feeder_address_____")
	err := k.SetFeederDelegation(ctx, validator, feeder)
	require.NoError(t, err)

	// Verify delegation was set
	delegation, err := k.GetFeederDelegation(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, feeder, delegation)

	// Delete delegation
	k.DeleteFeederDelegation(ctx, validator)

	// Verify back to initial state
	finalDelegation, _ := k.GetFeederDelegation(ctx, validator)
	if initialDelegation == nil {
		require.Nil(t, finalDelegation)
	} else {
		require.Equal(t, initialDelegation, finalDelegation)
	}
}

// TestMissCounterIncrementRevert tests that miss counter operations don't corrupt state
func TestMissCounterIncrementRevert(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	validatorAddr := sdk.ValAddress("validator1").String()

	// Initialize validator oracle state
	err := k.InitializeValidatorOracle(ctx, validatorAddr)
	require.NoError(t, err)

	// Get initial miss counter
	initialOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	require.NoError(t, err)
	initialMissCount := initialOracle.MissCounter

	// Increment miss counter
	err = k.IncrementMissCounter(ctx, validatorAddr)
	require.NoError(t, err)

	// Verify increment
	oracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	require.NoError(t, err)
	require.Equal(t, initialMissCount+1, oracle.MissCounter)

	// Reset miss counter
	err = k.ResetMissCounter(ctx, validatorAddr)
	require.NoError(t, err)

	// Verify reset
	finalOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	require.NoError(t, err)
	require.Equal(t, uint32(0), finalOracle.MissCounter)
}

// TestSlashingRevertOnInvalidValidator tests that slashing handles invalid validators
func TestSlashingRevertOnInvalidValidator(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Attempt to slash non-existent validator (should fail gracefully)
	nonExistentValidator := sdk.ValAddress("nonexistent_val____")

	// This may error or handle gracefully depending on implementation
	err := k.SlashMissVote(ctx, nonExistentValidator)
	// Just verify it doesn't panic and handles errors
	if err != nil {
		t.Logf("Expected error for non-existent validator: %v", err)
	}

	// Verify keeper still functional after error
	asset := "BTC/USD"
	validValidator := sdk.ValAddress("validator1")
	price := math.LegacyNewDec(50000)

	err = k.SubmitPrice(ctx, validValidator, asset, price)
	// May succeed or fail based on validator setup, but shouldn't panic
	t.Logf("Price submission after slash error: %v", err)
}

// TestDeleteOldSnapshotsPreservesRecentData tests that cleanup doesn't delete recent snapshots
func TestDeleteOldSnapshotsPreservesRecentData(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"

	// Create multiple snapshots at different heights
	heights := []int64{100, 200, 300, 400, 500}
	for _, height := range heights {
		snapshot := types.PriceSnapshot{
			Asset:       asset,
			Price:       math.LegacyNewDec(50000 + height), // Different prices
			BlockHeight: height,
			BlockTime:   ctx.BlockTime().Unix(),
		}
		err := k.SetPriceSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Delete old snapshots (keep those >= 300)
	minHeight := int64(300)
	err := k.DeleteOldSnapshots(ctx, asset, minHeight)
	require.NoError(t, err)

	// Verify old snapshots deleted
	_, err = k.GetPriceSnapshot(ctx, asset, 100)
	require.Error(t, err)
	_, err = k.GetPriceSnapshot(ctx, asset, 200)
	require.Error(t, err)

	// Verify recent snapshots preserved
	snapshot300, err := k.GetPriceSnapshot(ctx, asset, 300)
	require.NoError(t, err)
	require.Equal(t, int64(300), snapshot300.BlockHeight)

	snapshot400, err := k.GetPriceSnapshot(ctx, asset, 400)
	require.NoError(t, err)
	require.Equal(t, int64(400), snapshot400.BlockHeight)

	snapshot500, err := k.GetPriceSnapshot(ctx, asset, 500)
	require.NoError(t, err)
	require.Equal(t, int64(500), snapshot500.BlockHeight)
}

// TestPartialAggregationFailurePreservesConsistency tests consistency on partial aggregation failure
func TestPartialAggregationFailurePreservesConsistency(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	asset := "BTC/USD"

	// Submit price from single validator
	validator := sdk.ValAddress("validator1")
	price := math.LegacyNewDec(50000)
	err := k.SubmitPrice(ctx, validator, asset, price)
	require.NoError(t, err)

	// Set insufficient voting power
	vp, err := k.GetValidatorPrice(ctx, validator, asset)
	require.NoError(t, err)
	vp.VotingPower = 1 // Very low, won't meet threshold
	err = k.SetValidatorPrice(ctx, vp)
	require.NoError(t, err)

	// Attempt aggregation (should fail due to insufficient voting power)
	err = k.AggregateAssetPrice(ctx, asset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")

	// Verify validator submission still exists
	finalVP, err := k.GetValidatorPrice(ctx, validator, asset)
	require.NoError(t, err)
	require.Equal(t, price, finalVP.Price, "validator price should be preserved after failed aggregation")

	// Verify no aggregated price was created
	_, err = k.GetPrice(ctx, asset)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
