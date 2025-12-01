package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/oracle/types"
)

// OracleStateBackupData represents a complete backup of oracle module state
type OracleStateBackupData struct {
	Version            string
	Timestamp          time.Time
	BlockHeight        int64
	Params             types.Params
	AssetPrices        map[string]types.Price
	ValidatorPrevotes  map[string]map[string]types.AggregateExchangeRatePrevote // validator -> asset -> prevote
	ValidatorVotes     map[string]map[string]types.AggregateExchangeRateVote    // validator -> asset -> vote
	ValidatorDelegates map[string]string                                        // validator -> feeder
	MissCounters       map[string]uint64                                        // validator -> miss_count
	SlashingInfo       map[string]types.SlashingInfo                            // validator -> info
	TWAPData           map[string][]types.TWAPDataPoint                         // asset -> data_points
	Checksum           string
}

// ExportState exports the complete oracle module state for backup
func (k Keeper) ExportState(ctx context.Context) (*OracleStateBackupData, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	backup := &OracleStateBackupData{
		Version:            "1.0.0",
		Timestamp:          sdkCtx.BlockTime(),
		BlockHeight:        sdkCtx.BlockHeight(),
		AssetPrices:        make(map[string]types.Price),
		ValidatorPrevotes:  make(map[string]map[string]types.AggregateExchangeRatePrevote),
		ValidatorVotes:     make(map[string]map[string]types.AggregateExchangeRateVote),
		ValidatorDelegates: make(map[string]string),
		MissCounters:       make(map[string]uint64),
		SlashingInfo:       make(map[string]types.SlashingInfo),
		TWAPData:           make(map[string][]types.TWAPDataPoint),
	}

	// Export params
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}
	backup.Params = params

	// Export asset prices
	priceIterator := storetypes.KVStorePrefixIterator(store, types.PriceKeyPrefix)
	defer priceIterator.Close()

	for ; priceIterator.Valid(); priceIterator.Next() {
		asset := string(priceIterator.Key()[len(types.PriceKeyPrefix):])
		var price types.Price
		if err := k.cdc.Unmarshal(priceIterator.Value(), &price); err != nil {
			return nil, fmt.Errorf("failed to unmarshal price for %s: %w", asset, err)
		}
		backup.AssetPrices[asset] = price
	}

	// Export prevotes
	prevoteIterator := storetypes.KVStorePrefixIterator(store, types.PrevoteKeyPrefix)
	defer prevoteIterator.Close()

	for ; prevoteIterator.Valid(); prevoteIterator.Next() {
		// Parse composite key: prefix + validator + asset
		key := prevoteIterator.Key()[len(types.PrevoteKeyPrefix):]
		validator, asset := parseCompositeKey(key)

		var prevote types.AggregateExchangeRatePrevote
		if err := json.Unmarshal(prevoteIterator.Value(), &prevote); err != nil {
			return nil, fmt.Errorf("failed to unmarshal prevote: %w", err)
		}

		if backup.ValidatorPrevotes[validator] == nil {
			backup.ValidatorPrevotes[validator] = make(map[string]types.AggregateExchangeRatePrevote)
		}
		backup.ValidatorPrevotes[validator][asset] = prevote
	}

	// Export votes
	voteIterator := storetypes.KVStorePrefixIterator(store, types.VoteKeyPrefix)
	defer voteIterator.Close()

	for ; voteIterator.Valid(); voteIterator.Next() {
		key := voteIterator.Key()[len(types.VoteKeyPrefix):]
		validator, asset := parseCompositeKey(key)

		var vote types.AggregateExchangeRateVote
		if err := json.Unmarshal(voteIterator.Value(), &vote); err != nil {
			return nil, fmt.Errorf("failed to unmarshal vote: %w", err)
		}

		if backup.ValidatorVotes[validator] == nil {
			backup.ValidatorVotes[validator] = make(map[string]types.AggregateExchangeRateVote)
		}
		backup.ValidatorVotes[validator][asset] = vote
	}

	// Export feeder delegates
	delegateIterator := storetypes.KVStorePrefixIterator(store, types.DelegateKeyPrefix)
	defer delegateIterator.Close()

	for ; delegateIterator.Valid(); delegateIterator.Next() {
		validator := string(delegateIterator.Key()[len(types.DelegateKeyPrefix):])
		feeder := string(delegateIterator.Value())
		backup.ValidatorDelegates[validator] = feeder
	}

	// Export miss counters
	missIterator := storetypes.KVStorePrefixIterator(store, types.MissCounterKeyPrefix)
	defer missIterator.Close()

	for ; missIterator.Valid(); missIterator.Next() {
		validator := string(missIterator.Key()[len(types.MissCounterKeyPrefix):])
		missCount := sdk.BigEndianToUint64(missIterator.Value())
		backup.MissCounters[validator] = missCount
	}

	// Export TWAP data
	twapIterator := storetypes.KVStorePrefixIterator(store, types.TWAPKeyPrefix)
	defer twapIterator.Close()

	for ; twapIterator.Valid(); twapIterator.Next() {
		asset := string(twapIterator.Key()[len(types.TWAPKeyPrefix):])
		var dataPoints []types.TWAPDataPoint
		if err := json.Unmarshal(twapIterator.Value(), &dataPoints); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TWAP data for %s: %w", asset, err)
		}
		backup.TWAPData[asset] = dataPoints
	}

	// Calculate checksum
	backup.Checksum = calculateOracleChecksum(backup)

	return backup, nil
}

// ImportState imports a backup to restore oracle module state
func (k Keeper) ImportState(ctx context.Context, backup *OracleStateBackupData) error {
	// Verify checksum
	expectedChecksum := calculateOracleChecksum(backup)
	if backup.Checksum != expectedChecksum {
		return fmt.Errorf("backup checksum mismatch: expected %s, got %s", expectedChecksum, backup.Checksum)
	}

	store := k.getStore(ctx)

	// Clear existing state
	k.clearOracleState(ctx)

	// Import params
	if err := k.SetParams(ctx, backup.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Import prices
	for asset, price := range backup.AssetPrices {
		if err := k.SetPrice(ctx, price); err != nil {
			return fmt.Errorf("failed to set price for %s: %w", asset, err)
		}
	}

	// Import prevotes
	for validator, prevotes := range backup.ValidatorPrevotes {
		for asset, prevote := range prevotes {
			key := makeCompositeKey(types.PrevoteKeyPrefix, validator, asset)
			bz, err := json.Marshal(prevote)
			if err != nil {
				return fmt.Errorf("failed to marshal prevote: %w", err)
			}
			store.Set(key, bz)
		}
	}

	// Import votes
	for validator, votes := range backup.ValidatorVotes {
		for asset, vote := range votes {
			key := makeCompositeKey(types.VoteKeyPrefix, validator, asset)
			bz, err := json.Marshal(vote)
			if err != nil {
				return fmt.Errorf("failed to marshal vote: %w", err)
			}
			store.Set(key, bz)
		}
	}

	// Import delegates
	for validator, feeder := range backup.ValidatorDelegates {
		key := append(types.DelegateKeyPrefix, []byte(validator)...)
		store.Set(key, []byte(feeder))
	}

	// Import miss counters
	for validator, missCount := range backup.MissCounters {
		key := append(types.MissCounterKeyPrefix, []byte(validator)...)
		store.Set(key, sdk.Uint64ToBigEndian(missCount))
	}

	// Import TWAP data
	for asset, dataPoints := range backup.TWAPData {
		key := append(types.TWAPKeyPrefix, []byte(asset)...)
		bz, err := json.Marshal(dataPoints)
		if err != nil {
			return fmt.Errorf("failed to marshal TWAP data: %w", err)
		}
		store.Set(key, bz)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_state_imported",
			sdk.NewAttribute("assets_count", fmt.Sprintf("%d", len(backup.AssetPrices))),
			sdk.NewAttribute("validators_count", fmt.Sprintf("%d", len(backup.ValidatorDelegates))),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", backup.BlockHeight)),
		),
	)

	return nil
}

// ValidateState performs comprehensive state validation
func (k Keeper) ValidateState(ctx context.Context) error {
	store := k.getStore(ctx)

	// Validate asset prices
	priceIterator := storetypes.KVStorePrefixIterator(store, types.PriceKeyPrefix)
	defer priceIterator.Close()

	for ; priceIterator.Valid(); priceIterator.Next() {
		asset := string(priceIterator.Key()[len(types.PriceKeyPrefix):])
		var price types.Price
		if err := k.cdc.Unmarshal(priceIterator.Value(), &price); err != nil {
			return fmt.Errorf("corrupt price data for %s: %w", asset, err)
		}

		// Validate price fields
		if price.Price.IsNil() || price.Price.IsNegative() {
			return fmt.Errorf("invalid price for %s: %s", asset, price.Price)
		}

		if price.BlockTime <= 0 {
			return fmt.Errorf("invalid timestamp for %s: %d", asset, price.BlockTime)
		}
	}

	// Validate prevotes match votes
	prevoteIterator := storetypes.KVStorePrefixIterator(store, types.PrevoteKeyPrefix)
	defer prevoteIterator.Close()

	for ; prevoteIterator.Valid(); prevoteIterator.Next() {
		key := prevoteIterator.Key()[len(types.PrevoteKeyPrefix):]
		validator, asset := parseCompositeKey(key)

		var prevote types.AggregateExchangeRatePrevote
		if err := json.Unmarshal(prevoteIterator.Value(), &prevote); err != nil {
			return fmt.Errorf("corrupt prevote data: %w", err)
		}

		// Validate prevote fields
		if prevote.Hash == "" {
			return fmt.Errorf("prevote for %s/%s has empty hash", validator, asset)
		}
	}

	// Validate TWAP data points
	twapIterator := storetypes.KVStorePrefixIterator(store, types.TWAPKeyPrefix)
	defer twapIterator.Close()

	for ; twapIterator.Valid(); twapIterator.Next() {
		asset := string(twapIterator.Key()[len(types.TWAPKeyPrefix):])
		var dataPoints []types.TWAPDataPoint
		if err := json.Unmarshal(twapIterator.Value(), &dataPoints); err != nil {
			return fmt.Errorf("corrupt TWAP data for %s: %w", asset, err)
		}

		// Validate data points are chronological
		for i := 1; i < len(dataPoints); i++ {
			if dataPoints[i].Timestamp <= dataPoints[i-1].Timestamp {
				return fmt.Errorf("TWAP data points for %s not chronological", asset)
			}
		}
	}

	return nil
}

// RecoverFromCorruption attempts to recover from state corruption
func (k Keeper) RecoverFromCorruption(ctx context.Context, backupPath string) error {
	// Load backup
	backup, err := LoadOracleBackupFromFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}

	// Validate current state
	if err := k.ValidateState(ctx); err == nil {
		return fmt.Errorf("state is valid, recovery not needed")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Error("Oracle state corruption detected, initiating recovery", "backup", backupPath)

	// Import backup state
	if err := k.ImportState(ctx, backup); err != nil {
		return fmt.Errorf("failed to import backup: %w", err)
	}

	// Validate recovered state
	if err := k.ValidateState(ctx); err != nil {
		return fmt.Errorf("recovered state is still invalid: %w", err)
	}

	sdkCtx.Logger().Info("Oracle state recovery completed successfully")

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_state_recovered",
			sdk.NewAttribute("backup_height", fmt.Sprintf("%d", backup.BlockHeight)),
			sdk.NewAttribute("backup_timestamp", backup.Timestamp.String()),
		),
	)

	return nil
}

// CreateCheckpoint creates a state checkpoint
func (k Keeper) CreateCheckpoint(ctx context.Context, name string) error {
	backup, err := k.ExportState(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	filename := fmt.Sprintf("oracle_checkpoint_%s_%d.json", name, sdkCtx.BlockHeight())

	return backup.SaveToFile(filename)
}

// SaveToFile saves backup data to a file
func (backup *OracleStateBackupData) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// LoadOracleBackupFromFile loads backup data from a file
func LoadOracleBackupFromFile(filepath string) (*OracleStateBackupData, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup OracleStateBackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Verify checksum
	expectedChecksum := calculateOracleChecksum(&backup)
	if backup.Checksum != expectedChecksum {
		return nil, fmt.Errorf("backup file corrupted: checksum mismatch")
	}

	return &backup, nil
}

// calculateOracleChecksum calculates a checksum for backup data
func calculateOracleChecksum(backup *OracleStateBackupData) string {
	// Create a copy without checksum for hashing
	temp := *backup
	temp.Checksum = ""

	data, _ := json.Marshal(temp)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// clearOracleState clears all oracle module state (use with caution!)
func (k Keeper) clearOracleState(ctx context.Context) {
	store := k.getStore(ctx)

	// Clear prices
	clearOracleStorePrefix(store, types.PriceKeyPrefix)

	// Clear prevotes
	clearOracleStorePrefix(store, types.PrevoteKeyPrefix)

	// Clear votes
	clearOracleStorePrefix(store, types.VoteKeyPrefix)

	// Clear delegates
	clearOracleStorePrefix(store, types.DelegateKeyPrefix)

	// Clear miss counters
	clearOracleStorePrefix(store, types.MissCounterKeyPrefix)

	// Clear TWAP data
	clearOracleStorePrefix(store, types.TWAPKeyPrefix)
}

// clearOracleStorePrefix removes all keys with a given prefix
func clearOracleStorePrefix(store storetypes.KVStore, prefix []byte) {
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	keys := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}

	for _, key := range keys {
		store.Delete(key)
	}
}

// Helper functions for composite keys
func parseCompositeKey(key []byte) (string, string) {
	// Simple implementation - actual should handle proper key parsing
	parts := string(key)
	// Assuming format: validator + "|" + asset
	for i := 0; i < len(parts); i++ {
		if parts[i] == '|' {
			return parts[:i], parts[i+1:]
		}
	}
	return "", ""
}

func makeCompositeKey(prefix []byte, validator, asset string) []byte {
	key := append(prefix, []byte(validator)...)
	key = append(key, '|')
	key = append(key, []byte(asset)...)
	return key
}

// Placeholder methods (should be implemented based on actual keeper)

/*
func (k Keeper) SetPrice(ctx context.Context, asset string, price types.Price) error {
	// Implementation needed
	return nil
}
*/
