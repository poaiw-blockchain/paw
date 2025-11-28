package v2

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

var (
	// Key prefixes - must match the keeper
	PriceKeyPrefix             = []byte{0x01}
	ValidatorOracleKeyPrefix   = []byte{0x02}
	ValidatorPriceKeyPrefix    = []byte{0x03}
	ParamsKey                  = []byte{0x04}
	PriceSnapshotKeyPrefix     = []byte{0x05}
	MissCounterKeyPrefix       = []byte{0x10}
	AssetListKey               = []byte{0x11}
)

// Migrate implements store migrations from v1 to v2 for the Oracle module.
// This migration performs the following operations:
// 1. Validates existing price feed state
// 2. Validates validator oracle registrations
// 3. Rebuilds price snapshot indexes
// 4. Initializes miss counters for validators
// 5. Updates params with new security fields
// 6. Cleans up stale data
func Migrate(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Starting Oracle module v1 to v2 migration")

	store := ctx.KVStore(storeKey)

	// Step 1: Validate and clean price data
	if err := validatePriceData(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate price data: %w", err)
	}

	// Step 2: Validate validator oracle registrations
	if err := validateValidatorOracles(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to validate validator oracles: %w", err)
	}

	// Step 3: Rebuild price snapshot indexes
	if err := rebuildSnapshotIndexes(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to rebuild snapshot indexes: %w", err)
	}

	// Step 4: Initialize miss counters
	if err := initializeMissCounters(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to initialize miss counters: %w", err)
	}

	// Step 5: Update params with new security fields
	if err := migrateParams(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to migrate params: %w", err)
	}

	// Step 6: Clean up stale snapshot data
	if err := cleanStaleSnapshots(ctx, store, cdc); err != nil {
		return fmt.Errorf("failed to clean stale snapshots: %w", err)
	}

	ctx.Logger().Info("Oracle module v1 to v2 migration completed successfully")
	return nil
}

// validatePriceData validates and fixes price data inconsistencies
func validatePriceData(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating price data")

	iterator := storetypes.KVStorePrefixIterator(store, PriceKeyPrefix)
	defer iterator.Close()

	fixed := 0
	deleted := 0

	for ; iterator.Valid(); iterator.Next() {
		var price types.Price
		if err := cdc.Unmarshal(iterator.Value(), &price); err != nil {
			ctx.Logger().Error("failed to unmarshal price, deleting", "error", err)
			store.Delete(iterator.Key())
			deleted++
			continue
		}

		needsUpdate := false

		// Validate price is positive
		if price.Price.IsNil() || price.Price.LTE(math.LegacyZeroDec()) {
			ctx.Logger().Warn("fixing invalid price", "asset", price.Asset, "old_price", price.Price)
			// Delete invalid prices rather than fix
			store.Delete(iterator.Key())
			deleted++
			continue
		}

		// Validate block height is non-negative
		if price.BlockHeight < 0 {
			ctx.Logger().Warn("fixing negative block height", "asset", price.Asset, "old_height", price.BlockHeight)
			price.BlockHeight = ctx.BlockHeight()
			needsUpdate = true
		}

		// Validate timestamp is reasonable
		if price.BlockTime < 0 {
			ctx.Logger().Warn("fixing negative block time", "asset", price.Asset, "old_time", price.BlockTime)
			price.BlockTime = ctx.BlockTime().Unix()
			needsUpdate = true
		}

		// Validate num validators is positive
		if price.NumValidators == 0 {
			ctx.Logger().Warn("fixing zero validators", "asset", price.Asset)
			price.NumValidators = 1
			needsUpdate = true
		}

		if needsUpdate {
			bz, err := cdc.Marshal(&price)
			if err != nil {
				return fmt.Errorf("failed to marshal price: %w", err)
			}
			store.Set(iterator.Key(), bz)
			fixed++
		}
	}

	ctx.Logger().Info("Price data validated", "fixed", fixed, "deleted", deleted)
	return nil
}

// validateValidatorOracles validates validator oracle registrations
func validateValidatorOracles(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Validating validator oracles")

	iterator := storetypes.KVStorePrefixIterator(store, ValidatorOracleKeyPrefix)
	defer iterator.Close()

	fixed := 0
	for ; iterator.Valid(); iterator.Next() {
		var validatorOracle types.ValidatorOracle
		if err := cdc.Unmarshal(iterator.Value(), &validatorOracle); err != nil {
			ctx.Logger().Error("failed to unmarshal validator oracle", "error", err)
			continue
		}

		needsUpdate := false

		// Ensure miss counter is non-negative
		if validatorOracle.MissCounter < 0 {
			ctx.Logger().Warn("fixing negative miss counter",
				"validator", validatorOracle.ValidatorAddr,
				"old", validatorOracle.MissCounter)
			validatorOracle.MissCounter = 0
			needsUpdate = true
		}

		// Validate validator address format
		if _, err := sdk.ValAddressFromBech32(validatorOracle.ValidatorAddr); err != nil {
			ctx.Logger().Warn("invalid validator address, skipping",
				"validator", validatorOracle.ValidatorAddr,
				"error", err)
			continue
		}

		if needsUpdate {
			bz, err := cdc.Marshal(&validatorOracle)
			if err != nil {
				return fmt.Errorf("failed to marshal validator oracle: %w", err)
			}
			store.Set(iterator.Key(), bz)
			fixed++
		}
	}

	ctx.Logger().Info("Validator oracles validated", "fixed", fixed)
	return nil
}

// rebuildSnapshotIndexes rebuilds price snapshot indexes
func rebuildSnapshotIndexes(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Rebuilding snapshot indexes")

	// Build asset list from all snapshots
	assetMap := make(map[string]bool)

	iterator := storetypes.KVStorePrefixIterator(store, PriceSnapshotKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.PriceSnapshot
		if err := cdc.Unmarshal(iterator.Value(), &snapshot); err != nil {
			ctx.Logger().Error("failed to unmarshal snapshot", "error", err)
			continue
		}

		// Validate snapshot
		if snapshot.Price.IsNil() || snapshot.Price.LTE(math.LegacyZeroDec()) {
			ctx.Logger().Warn("deleting invalid snapshot", "asset", snapshot.Asset)
			store.Delete(iterator.Key())
			continue
		}

		assetMap[snapshot.Asset] = true
		count++
	}

	// Store asset list
	assets := make([]string, 0, len(assetMap))
	for asset := range assetMap {
		assets = append(assets, asset)
	}

	/*
	if len(assets) > 0 {
		assetListBz, err := cdc.Marshal(&types.AssetList{Assets: assets})
		if err != nil {
			return fmt.Errorf("failed to marshal asset list: %w", err)
		}
		store.Set(AssetListKey, assetListBz)
	}
	*/

	ctx.Logger().Info("Snapshot indexes rebuilt", "snapshots", count, "assets", len(assets))
	return nil
}

// initializeMissCounters initializes miss counters for all validators
func initializeMissCounters(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Initializing miss counters")

	iterator := storetypes.KVStorePrefixIterator(store, ValidatorOracleKeyPrefix)
	defer iterator.Close()

	initialized := 0
	for ; iterator.Valid(); iterator.Next() {
		var validatorOracle types.ValidatorOracle
		if err := cdc.Unmarshal(iterator.Value(), &validatorOracle); err != nil {
			continue
		}

		// Initialize miss counter if not already present
		missCounterKey := getMissCounterKey(validatorOracle.ValidatorAddr)
		if !store.Has(missCounterKey) {
			counterBz := make([]byte, 8)
			binary.BigEndian.PutUint64(counterBz, uint64(validatorOracle.MissCounter))
			store.Set(missCounterKey, counterBz)
			initialized++
		}
	}

	ctx.Logger().Info("Miss counters initialized", "count", initialized)
	return nil
}

// migrateParams updates params with new security fields
func migrateParams(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Migrating params")

	bz := store.Get(ParamsKey)
	if bz == nil {
		// No params yet, use defaults
		params := types.DefaultParams()
		newBz, err := cdc.Marshal(&params)
		if err != nil {
			return fmt.Errorf("failed to marshal default params: %w", err)
		}
		store.Set(ParamsKey, newBz)
		ctx.Logger().Info("Initialized default params")
		return nil
	}

	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// Add new fields with default values if they're zero
	updated := false

	if params.VoteThreshold.IsNil() || params.VoteThreshold.IsZero() {
		params.VoteThreshold = math.LegacyMustNewDecFromStr("0.67") // 67%
		updated = true
	}

	if params.MinValidPerWindow == 0 {
		params.MinValidPerWindow = 100
		updated = true
	}

	if params.SlashFraction.IsNil() || params.SlashFraction.IsZero() {
		params.SlashFraction = math.LegacyMustNewDecFromStr("0.01") // 1%
		updated = true
	}

	if params.TwapLookbackWindow == 0 {
		params.TwapLookbackWindow = 1000 // 1000 blocks
		updated = true
	}



	if updated {
		newBz, err := cdc.Marshal(&params)
		if err != nil {
			return fmt.Errorf("failed to marshal updated params: %w", err)
		}
		store.Set(ParamsKey, newBz)
		ctx.Logger().Info("Updated params with new fields")
	}

	return nil
}

// cleanStaleSnapshots removes price snapshots older than retention period
func cleanStaleSnapshots(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("Cleaning stale snapshots")

	// Get params to determine retention period
	bz := store.Get(ParamsKey)
	if bz == nil {
		return nil
	}

	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err != nil {
		return err
	}

	// Calculate cutoff height (keep 2x lookback window for safety)
	retentionBlocks := int64(params.TwapLookbackWindow * 2)
	cutoffHeight := ctx.BlockHeight() - retentionBlocks

	iterator := storetypes.KVStorePrefixIterator(store, PriceSnapshotKeyPrefix)
	defer iterator.Close()

	var staleKeys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.PriceSnapshot
		if err := cdc.Unmarshal(iterator.Value(), &snapshot); err != nil {
			continue
		}

		if snapshot.BlockHeight < cutoffHeight {
			staleKeys = append(staleKeys, iterator.Key())
		}
	}

	// Delete stale snapshots
	for _, key := range staleKeys {
		store.Delete(key)
	}

	ctx.Logger().Info("Stale snapshots cleaned", "deleted", len(staleKeys), "cutoff_height", cutoffHeight)
	return nil
}

// Helper functions

func getMissCounterKey(validatorAddr string) []byte {
	return append(MissCounterKeyPrefix, []byte(validatorAddr)...)
}
