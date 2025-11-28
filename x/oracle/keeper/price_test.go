package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Helper functions for oracle tests

func createTestValidator(t *testing.T) sdk.ValAddress {
	return sdk.ValAddress([]byte("test_validator_addr_"))
}

func createTestValidatorWithIndex(t *testing.T, index int) sdk.ValAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("test_validator_"))
	addr[19] = byte(index)
	return sdk.ValAddress(addr)
}

func createTestFeeder(t *testing.T) sdk.AccAddress {
	return sdk.AccAddress([]byte("test_feeder_address_"))
}

// TestSetPrice tests setting aggregated price
func TestSetPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	price := types.Price{
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   100,
		NumValidators: 10,
	}

	err := k.SetPrice(ctx, price)
	require.NoError(t, err)

	// Verify price was stored
	retrieved, err := k.GetPrice(ctx, "BTC")
	require.NoError(t, err)
	require.Equal(t, price.Asset, retrieved.Asset)
	require.Equal(t, price.Price, retrieved.Price)
	require.Equal(t, price.BlockHeight, retrieved.BlockHeight)
	require.Equal(t, price.NumValidators, retrieved.NumValidators)
}

// TestGetPrice_NotFound tests retrieval of non-existent price
func TestGetPrice_NotFound(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	_, err := k.GetPrice(ctx, "NONEXISTENT")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestDeletePrice tests price deletion
func TestDeletePrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	price := types.Price{
		Asset:         "ETH",
		Price:         math.LegacyNewDec(3000),
		BlockHeight:   100,
		NumValidators: 10,
	}

	err := k.SetPrice(ctx, price)
	require.NoError(t, err)

	// Delete price
	k.DeletePrice(ctx, "ETH")

	// Verify price is gone
	_, err = k.GetPrice(ctx, "ETH")
	require.Error(t, err)
}

// TestIteratePrices tests price iteration
func TestIteratePrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set multiple prices
	assets := []string{"BTC", "ETH", "ATOM", "SOL"}
	for i, asset := range assets {
		price := types.Price{
			Asset:         asset,
			Price:         math.LegacyNewDec(int64((i + 1) * 1000)),
			BlockHeight:   100,
			NumValidators: 10,
		}
		err := k.SetPrice(ctx, price)
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err := k.IteratePrices(ctx, func(price types.Price) bool {
		count++
		require.NotEmpty(t, price.Asset)
		require.True(t, price.Price.IsPositive())
		return false // continue
	})
	require.NoError(t, err)
	require.Equal(t, len(assets), count)
}

// TestGetAllPrices tests getting all prices
func TestGetAllPrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set multiple prices
	assets := []string{"BTC", "ETH", "ATOM"}
	for i, asset := range assets {
		price := types.Price{
			Asset:         asset,
			Price:         math.LegacyNewDec(int64((i + 1) * 1000)),
			BlockHeight:   100,
			NumValidators: 10,
		}
		err := k.SetPrice(ctx, price)
		require.NoError(t, err)
	}

	// Get all prices
	prices, err := k.GetAllPrices(ctx)
	require.NoError(t, err)
	require.Equal(t, len(assets), len(prices))

	// Verify all prices are valid
	for _, price := range prices {
		require.NotEmpty(t, price.Asset)
		require.True(t, price.Price.IsPositive())
	}
}

// TestSetValidatorPrice tests validator price submission
func TestSetValidatorPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)

	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: validator.String(),
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   100,
	}

	err := k.SetValidatorPrice(ctx, validatorPrice)
	require.NoError(t, err)

	// Verify validator price was stored
	retrieved, err := k.GetValidatorPrice(ctx, validator, "BTC")
	require.NoError(t, err)
	require.Equal(t, validatorPrice.ValidatorAddr, retrieved.ValidatorAddr)
	require.Equal(t, validatorPrice.Asset, retrieved.Asset)
	require.Equal(t, validatorPrice.Price, retrieved.Price)
	require.Equal(t, validatorPrice.BlockHeight, retrieved.BlockHeight)
}

// TestSetValidatorPrice_InvalidAddress tests rejection of invalid validator address
func TestSetValidatorPrice_InvalidAddress(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: "invalid_address",
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   100,
	}

	err := k.SetValidatorPrice(ctx, validatorPrice)
	require.Error(t, err)
}

// TestGetValidatorPrice_NotFound tests retrieval of non-existent validator price
func TestGetValidatorPrice_NotFound(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)

	_, err := k.GetValidatorPrice(ctx, validator, "BTC")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestDeleteValidatorPrice tests validator price deletion
func TestDeleteValidatorPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)

	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: validator.String(),
		Asset:         "ETH",
		Price:         math.LegacyNewDec(3000),
		BlockHeight:   100,
	}

	err := k.SetValidatorPrice(ctx, validatorPrice)
	require.NoError(t, err)

	// Delete validator price
	k.DeleteValidatorPrice(ctx, validator, "ETH")

	// Verify price is gone
	_, err = k.GetValidatorPrice(ctx, validator, "ETH")
	require.Error(t, err)
}

// TestIterateValidatorPrices tests validator price iteration
func TestIterateValidatorPrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Submit prices from multiple validators
	numValidators := 5
	for i := 0; i < numValidators; i++ {
		validator := createTestValidatorWithIndex(t, i)
		validatorPrice := types.ValidatorPrice{
			ValidatorAddr: validator.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(int64(50000 + i*100)),
			BlockHeight:   100,
		}
		err := k.SetValidatorPrice(ctx, validatorPrice)
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err := k.IterateValidatorPrices(ctx, "BTC", func(vp types.ValidatorPrice) bool {
		count++
		require.Equal(t, "BTC", vp.Asset)
		require.True(t, vp.Price.IsPositive())
		return false // continue
	})
	require.NoError(t, err)
	require.Equal(t, numValidators, count)
}

// TestGetValidatorPricesByAsset tests getting all validator prices for an asset
func TestGetValidatorPricesByAsset(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Submit prices from multiple validators
	numValidators := 3
	for i := 0; i < numValidators; i++ {
		validator := createTestValidatorWithIndex(t, i)
		validatorPrice := types.ValidatorPrice{
			ValidatorAddr: validator.String(),
			Asset:         "ETH",
			Price:         math.LegacyNewDec(int64(3000 + i*10)),
			BlockHeight:   100,
		}
		err := k.SetValidatorPrice(ctx, validatorPrice)
		require.NoError(t, err)
	}

	// Get all validator prices for ETH
	prices, err := k.GetValidatorPricesByAsset(ctx, "ETH")
	require.NoError(t, err)
	require.Equal(t, numValidators, len(prices))

	// Verify all prices are for ETH
	for _, vp := range prices {
		require.Equal(t, "ETH", vp.Asset)
		require.True(t, vp.Price.IsPositive())
	}
}

// TestSetPriceSnapshot tests setting price snapshot
func TestSetPriceSnapshot(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	snapshot := types.PriceSnapshot{
		Asset:       "BTC",
		Price:       math.LegacyNewDec(50000),
		BlockHeight: 100,
		BlockTime:   1000000,
	}

	err := k.SetPriceSnapshot(ctx, snapshot)
	require.NoError(t, err)

	// Verify snapshot was stored
	retrieved, err := k.GetPriceSnapshot(ctx, "BTC", 100)
	require.NoError(t, err)
	require.Equal(t, snapshot.Asset, retrieved.Asset)
	require.Equal(t, snapshot.Price, retrieved.Price)
	require.Equal(t, snapshot.BlockHeight, retrieved.BlockHeight)
	require.Equal(t, snapshot.BlockTime, retrieved.BlockTime)
}

// TestGetPriceSnapshot_NotFound tests retrieval of non-existent snapshot
func TestGetPriceSnapshot_NotFound(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	_, err := k.GetPriceSnapshot(ctx, "BTC", 999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestIteratePriceSnapshots tests snapshot iteration
func TestIteratePriceSnapshots(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Create multiple snapshots
	numSnapshots := 10
	for i := 0; i < numSnapshots; i++ {
		snapshot := types.PriceSnapshot{
			Asset:       "BTC",
			Price:       math.LegacyNewDec(int64(50000 + i*100)),
			BlockHeight: int64(100 + i),
			BlockTime:   int64(1000000 + i*10),
		}
		err := k.SetPriceSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err := k.IteratePriceSnapshots(ctx, "BTC", func(snapshot types.PriceSnapshot) bool {
		count++
		require.Equal(t, "BTC", snapshot.Asset)
		require.True(t, snapshot.Price.IsPositive())
		return false // continue
	})
	require.NoError(t, err)
	require.Equal(t, numSnapshots, count)
}

// TestDeleteOldSnapshots tests snapshot cleanup
func TestDeleteOldSnapshots(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Create snapshots at different heights
	for i := 0; i < 20; i++ {
		snapshot := types.PriceSnapshot{
			Asset:       "BTC",
			Price:       math.LegacyNewDec(int64(50000 + i*100)),
			BlockHeight: int64(100 + i),
			BlockTime:   int64(1000000 + i*10),
		}
		err := k.SetPriceSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Delete snapshots older than block 110
	err := k.DeleteOldSnapshots(ctx, "BTC", 110)
	require.NoError(t, err)

	// Verify old snapshots are gone
	for i := 0; i < 10; i++ {
		_, err := k.GetPriceSnapshot(ctx, "BTC", int64(100+i))
		require.Error(t, err, "snapshot at height %d should be deleted", 100+i)
	}

	// Verify recent snapshots remain
	for i := 10; i < 20; i++ {
		_, err := k.GetPriceSnapshot(ctx, "BTC", int64(100+i))
		require.NoError(t, err, "snapshot at height %d should exist", 100+i)
	}
}

// TestSetFeederDelegation tests feeder delegation
func TestSetFeederDelegation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)
	feeder := createTestFeeder(t)

	err := k.SetFeederDelegation(ctx, validator, feeder)
	require.NoError(t, err)

	// Verify delegation was stored
	retrieved, err := k.GetFeederDelegation(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, feeder, retrieved)
}

// TestGetFeederDelegation_NotFound tests retrieval when no delegation exists
func TestGetFeederDelegation_NotFound(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)

	// No delegation exists, should return nil without error
	retrieved, err := k.GetFeederDelegation(ctx, validator)
	require.NoError(t, err)
	require.Nil(t, retrieved)
}

// TestDeleteFeederDelegation tests delegation deletion
func TestDeleteFeederDelegation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)
	feeder := createTestFeeder(t)

	err := k.SetFeederDelegation(ctx, validator, feeder)
	require.NoError(t, err)

	// Delete delegation
	k.DeleteFeederDelegation(ctx, validator)

	// Verify delegation is gone
	retrieved, err := k.GetFeederDelegation(ctx, validator)
	require.NoError(t, err)
	require.Nil(t, retrieved)
}

// TestMultipleValidatorPrices tests handling prices from multiple validators
func TestMultipleValidatorPrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Submit prices from 10 validators
	numValidators := 10
	for i := 0; i < numValidators; i++ {
		validator := createTestValidatorWithIndex(t, i)
		validatorPrice := types.ValidatorPrice{
			ValidatorAddr: validator.String(),
			Asset:         "BTC",
			Price:         math.LegacyNewDec(int64(49000 + i*200)), // Prices range from 49000 to 50800
			BlockHeight:   100,
		}
		err := k.SetValidatorPrice(ctx, validatorPrice)
		require.NoError(t, err)
	}

	// Verify all prices stored
	prices, err := k.GetValidatorPricesByAsset(ctx, "BTC")
	require.NoError(t, err)
	require.Equal(t, numValidators, len(prices))
}

// TestValidatorPriceUpdate tests updating validator price
func TestValidatorPriceUpdate(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	validator := createTestValidator(t)

	// Submit initial price
	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: validator.String(),
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   100,
	}
	err := k.SetValidatorPrice(ctx, validatorPrice)
	require.NoError(t, err)

	// Update price
	updatedPrice := types.ValidatorPrice{
		ValidatorAddr: validator.String(),
		Asset:         "BTC",
		Price:         math.LegacyNewDec(51000),
		BlockHeight:   101,
	}
	err = k.SetValidatorPrice(ctx, updatedPrice)
	require.NoError(t, err)

	// Verify price was updated
	retrieved, err := k.GetValidatorPrice(ctx, validator, "BTC")
	require.NoError(t, err)
	require.Equal(t, updatedPrice.Price, retrieved.Price)
	require.Equal(t, updatedPrice.BlockHeight, retrieved.BlockHeight)
}

// TestPriceSnapshot_TimeSeriesData tests creating time series of snapshots
func TestPriceSnapshot_TimeSeriesData(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Create snapshots over time
	basePrice := int64(50000)
	for i := 0; i < 100; i++ {
		// Simulate price volatility
		priceChange := int64((i % 10) - 5) * 100
		snapshot := types.PriceSnapshot{
			Asset:       "BTC",
			Price:       math.LegacyNewDec(basePrice + priceChange),
			BlockHeight: int64(1000 + i),
			BlockTime:   int64(10000000 + i*6), // 6 seconds per block
		}
		err := k.SetPriceSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Verify all snapshots stored
	count := 0
	err := k.IteratePriceSnapshots(ctx, "BTC", func(snapshot types.PriceSnapshot) bool {
		count++
		require.Equal(t, "BTC", snapshot.Asset)
		return false
	})
	require.NoError(t, err)
	require.Equal(t, 100, count)
}
