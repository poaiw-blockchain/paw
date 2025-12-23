package keeper_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestMigrator_Migrate1to2(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	// Seed price feed data to exercise migration paths
	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0x1}, 20))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

	// Register validator oracle
	oracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr.String(),
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
		GeographicRegion: "us-west",
	}
	require.NoError(t, k.SetValidatorOracle(ctx, oracle))

	// Submit a price
	price := types.Price{
		Asset:         "BTC/USD",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	require.NoError(t, k.SetPrice(ctx, price))

	// Add validator price
	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: valAddr.String(),
		Asset:         "BTC/USD",
		Price:         math.LegacyMustNewDecFromStr("50000.00"),
		BlockHeight:   ctx.BlockHeight(),
		VotingPower:   100,
	}
	require.NoError(t, k.SetValidatorPrice(ctx, validatorPrice))

	// Add price snapshot
	snapshot := types.PriceSnapshot{
		Asset:       "BTC/USD",
		Price:       math.LegacyMustNewDecFromStr("49500.00"),
		BlockHeight: ctx.BlockHeight() - 1,
		BlockTime:   ctx.BlockTime().Unix() - 100,
	}
	require.NoError(t, k.SetPriceSnapshot(ctx, snapshot))

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Verify data still exists and is valid after migration
	retrievedPrice, err := k.GetPrice(ctx, "BTC/USD")
	require.NoError(t, err)
	require.Equal(t, price.Asset, retrievedPrice.Asset)
	require.True(t, retrievedPrice.Price.GT(math.LegacyZeroDec()))

	retrievedOracle, err := k.GetValidatorOracle(ctx, valAddr.String())
	require.NoError(t, err)
	require.Equal(t, oracle.ValidatorAddr, retrievedOracle.ValidatorAddr)
}

func TestMigrator_Migrate1to2_NoData(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Test migration with minimal setup (only params from test setup)
	// Migration should run successfully without any price/oracle data
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Migration should complete without error even with no data
	// (Params are initialized by test setup and migration preserves them)
}

func TestMigrator_Migrate1to2_WithExistingData(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0x2}, 20))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

	// Create valid price
	price := types.Price{
		Asset:         "ETH/USD",
		Price:         math.LegacyMustNewDecFromStr("3000.00"),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	require.NoError(t, k.SetPrice(ctx, price))

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Price should remain valid after migration
	retrievedPrice, err := k.GetPrice(ctx, "ETH/USD")
	require.NoError(t, err)
	require.Equal(t, price.Asset, retrievedPrice.Asset)
	require.True(t, retrievedPrice.Price.Equal(math.LegacyMustNewDecFromStr("3000.00")))
}

func TestMigrator_Migrate1to2_ValidatorOraclePreserved(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0x3}, 20))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

	// Create validator oracle with miss counter
	oracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr.String(),
		MissCounter:      42,
		TotalSubmissions: 200,
		IsActive:         true,
		GeographicRegion: "eu-central",
	}
	require.NoError(t, k.SetValidatorOracle(ctx, oracle))

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Verify validator oracle data is preserved
	retrievedOracle, err := k.GetValidatorOracle(ctx, valAddr.String())
	require.NoError(t, err)
	require.Equal(t, oracle.ValidatorAddr, retrievedOracle.ValidatorAddr)
	require.Equal(t, oracle.MissCounter, retrievedOracle.MissCounter)
	require.Equal(t, oracle.TotalSubmissions, retrievedOracle.TotalSubmissions)
	require.Equal(t, oracle.IsActive, retrievedOracle.IsActive)
}

func TestMigrator_Migrate1to2_CleanStaleSnapshots(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	// Create old snapshot that should be cleaned up
	oldHeight := ctx.BlockHeight() - int64(params.TwapLookbackWindow*2) - 1000
	staleSnapshot := types.PriceSnapshot{
		Asset:       "STALE/USD",
		Price:       math.LegacyMustNewDecFromStr("1.00"),
		BlockHeight: oldHeight,
		BlockTime:   ctx.BlockTime().Unix() - 100000,
	}
	require.NoError(t, k.SetPriceSnapshot(ctx, staleSnapshot))

	// Create recent snapshot that should be kept
	recentSnapshot := types.PriceSnapshot{
		Asset:       "RECENT/USD",
		Price:       math.LegacyMustNewDecFromStr("2.00"),
		BlockHeight: ctx.BlockHeight() - 10,
		BlockTime:   ctx.BlockTime().Unix() - 100,
	}
	require.NoError(t, k.SetPriceSnapshot(ctx, recentSnapshot))

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Check if stale snapshot was removed
	_, err = k.GetPriceSnapshot(ctx, "STALE/USD", oldHeight)
	require.Error(t, err, "stale snapshot should have been deleted")

	// Check if recent snapshot is still there
	retrievedRecent, err := k.GetPriceSnapshot(ctx, "RECENT/USD", ctx.BlockHeight()-10)
	require.NoError(t, err, "recent snapshot should have been kept")
	require.Equal(t, "RECENT/USD", retrievedRecent.Asset)
}

func TestMigrator_Migrate1to2_SuccessfulExecution(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Migration should complete successfully
	// The migration is idempotent and safe to run multiple times
	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)
}

func TestMigrator_Migrate1to2_MultipleValidators(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	// Create multiple validators
	validators := []sdk.ValAddress{
		sdk.ValAddress(bytes.Repeat([]byte{0x10}, 20)),
		sdk.ValAddress(bytes.Repeat([]byte{0x11}, 20)),
		sdk.ValAddress(bytes.Repeat([]byte{0x12}, 20)),
	}

	for _, valAddr := range validators {
		require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

		oracle := types.ValidatorOracle{
			ValidatorAddr:    valAddr.String(),
			MissCounter:      1,
			TotalSubmissions: 10,
			IsActive:         true,
			GeographicRegion: "global",
		}
		require.NoError(t, k.SetValidatorOracle(ctx, oracle))
	}

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// All validators should still exist
	for _, valAddr := range validators {
		retrievedOracle, err := k.GetValidatorOracle(ctx, valAddr.String())
		require.NoError(t, err)
		require.Equal(t, valAddr.String(), retrievedOracle.ValidatorAddr)
	}
}

func TestMigrator_Migrate1to2_Idempotence(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0x5}, 20))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

	// Seed data
	price := types.Price{
		Asset:         "ADA/USD",
		Price:         math.LegacyMustNewDecFromStr("0.50"),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	require.NoError(t, k.SetPrice(ctx, price))

	oracle := types.ValidatorOracle{
		ValidatorAddr:    valAddr.String(),
		MissCounter:      3,
		TotalSubmissions: 50,
		IsActive:         true,
		GeographicRegion: "asia-pacific",
	}
	require.NoError(t, k.SetValidatorOracle(ctx, oracle))

	// Execute migration twice
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Verify data is still valid and not corrupted
	retrievedPrice, err := k.GetPrice(ctx, "ADA/USD")
	require.NoError(t, err)
	require.Equal(t, price.Asset, retrievedPrice.Asset)
	require.True(t, retrievedPrice.Price.Equal(math.LegacyMustNewDecFromStr("0.50")))

	retrievedOracle, err := k.GetValidatorOracle(ctx, valAddr.String())
	require.NoError(t, err)
	require.Equal(t, oracle.ValidatorAddr, retrievedOracle.ValidatorAddr)
	require.Equal(t, oracle.MissCounter, retrievedOracle.MissCounter)
}

func TestMigrator_Migrate1to2_MultipleAssets(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	valAddr := sdk.ValAddress(bytes.Repeat([]byte{0x6}, 20))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, valAddr))

	// Seed multiple assets
	assets := []string{"BTC/USD", "ETH/USD", "SOL/USD", "AVAX/USD"}
	for i, asset := range assets {
		price := types.Price{
			Asset:         asset,
			Price:         math.LegacyMustNewDecFromStr("100.00"),
			BlockHeight:   ctx.BlockHeight(),
			BlockTime:     ctx.BlockTime().Unix(),
			NumValidators: 1,
		}
		require.NoError(t, k.SetPrice(ctx, price))

		snapshot := types.PriceSnapshot{
			Asset:       asset,
			Price:       math.LegacyMustNewDecFromStr("99.00"),
			BlockHeight: ctx.BlockHeight() - int64(i+1),
			BlockTime:   ctx.BlockTime().Unix() - int64((i+1)*100),
		}
		require.NoError(t, k.SetPriceSnapshot(ctx, snapshot))
	}

	// Execute migration
	migrator := keeper.NewMigrator(k)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// Verify all assets survived migration
	for _, asset := range assets {
		retrievedPrice, err := k.GetPrice(ctx, asset)
		require.NoError(t, err, "asset %s should exist after migration", asset)
		require.Equal(t, asset, retrievedPrice.Asset)
		require.True(t, retrievedPrice.Price.GT(math.LegacyZeroDec()))
	}

	// Verify snapshots were indexed - iterate and count
	snapshotCount := 0
	for _, asset := range assets {
		err := k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
			snapshotCount++
			return false // continue iteration
		})
		require.NoError(t, err)
	}
	require.GreaterOrEqual(t, snapshotCount, len(assets))
}
