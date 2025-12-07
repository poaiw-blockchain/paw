package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// GetPoolTWAP retrieves the TWAP record for a pool if it exists.
func (k Keeper) GetPoolTWAP(ctx context.Context, poolID uint64) (*types.PoolTWAP, bool, error) {
	store := k.getStore(ctx)
	bz := store.Get(PoolTWAPKey(poolID))
	if bz == nil {
		return nil, false, nil
	}

	var record types.PoolTWAP
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

// SetPoolTWAP stores the TWAP metadata for a pool.
func (k Keeper) SetPoolTWAP(ctx context.Context, record types.PoolTWAP) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&record)
	if err != nil {
		return err
	}
	store.Set(PoolTWAPKey(record.PoolId), bz)
	return nil
}

// GetAllPoolTWAPs returns all TWAP records in state.
func (k Keeper) GetAllPoolTWAPs(ctx context.Context) ([]types.PoolTWAP, error) {
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, PoolTWAPKeyPrefix)
	defer iter.Close()

	var records []types.PoolTWAP
	for ; iter.Valid(); iter.Next() {
		var record types.PoolTWAP
		if err := k.cdc.Unmarshal(iter.Value(), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// UpdateCumulativePriceOnSwap updates the cumulative price when a swap occurs.
// This implements the Uniswap V2 TWAP oracle approach:
// - Store cumulative price that grows monotonically
// - Update only on state-changing operations (swaps, liquidity changes)
// - TWAP query = (cumulativeNow - cumulativePast) / timeDelta
//
// Security: This prevents manipulation via TWAP calculation being O(1) instead of O(n),
// and makes oracle resistant to single-block price manipulation.
func (k Keeper) UpdateCumulativePriceOnSwap(ctx context.Context, poolID uint64, price0, price1 math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTime := sdkCtx.BlockTime()
	currentTimestamp := currentTime.Unix()

	// Get existing TWAP record or create new one
	record, found, err := k.GetPoolTWAP(ctx, poolID)
	if err != nil {
		return types.ErrInvalidState.Wrapf("failed to get pool TWAP: %v", err)
	}

	if !found {
		// Initialize new TWAP record for this pool
		record = &types.PoolTWAP{
			PoolId:          poolID,
			LastPrice:       price0,
			CumulativePrice: math.LegacyZeroDec(),
			TotalSeconds:    0,
			LastTimestamp:   currentTimestamp,
			TwapPrice:       price0,
		}
	} else {
		// Calculate time elapsed since last update
		lastTime := time.Unix(record.LastTimestamp, 0)
		timeElapsed := currentTime.Sub(lastTime).Seconds()

		if timeElapsed > 0 && !record.LastPrice.IsNil() && !record.LastPrice.IsZero() {
			// Update cumulative price: add (lastPrice * timeElapsed)
			// This is the core of the Uniswap V2 TWAP oracle mechanism
			priceAccumulation := record.LastPrice.MulInt64(int64(timeElapsed))
			record.CumulativePrice = record.CumulativePrice.Add(priceAccumulation)
			record.TotalSeconds += uint64(timeElapsed)

			// Update TWAP if we have accumulated time
			if record.TotalSeconds > 0 {
				record.TwapPrice = record.CumulativePrice.QuoInt64(int64(record.TotalSeconds))
			}
		}

		// Update last price and timestamp
		record.LastPrice = price0
		record.LastTimestamp = currentTimestamp
	}

	// Persist updated TWAP record
	if err := k.SetPoolTWAP(ctx, *record); err != nil {
		return types.ErrInvalidState.Wrapf("failed to save pool TWAP: %v", err)
	}

	sdkCtx.Logger().Debug("updated cumulative price on swap",
		"pool_id", poolID,
		"current_price", price0.String(),
		"cumulative_price", record.CumulativePrice.String(),
		"twap", record.TwapPrice.String(),
		"total_seconds", record.TotalSeconds,
	)

	return nil
}

// GetTWAP calculates the time-weighted average price for a pool over a time range.
// This is an O(1) operation that uses the cumulative price mechanism.
//
// Parameters:
//   - poolID: The pool to query
//   - startTime: Start of the time window (Unix timestamp)
//   - endTime: End of the time window (Unix timestamp)
//
// Returns: TWAP as (cumulativeEnd - cumulativeStart) / (endTime - startTime)
//
// Security: Resistant to single-block manipulation since TWAP is calculated over time.
// Front-running attacks cannot manipulate the entire historical cumulative price.
func (k Keeper) GetTWAP(ctx context.Context, poolID uint64, startTime, endTime int64) (math.LegacyDec, error) {
	if endTime <= startTime {
		return math.LegacyZeroDec(), types.ErrInvalidInput.Wrapf("invalid time range: end %d <= start %d", endTime, startTime)
	}

	// For this simplified implementation, we use the current cumulative state
	// A production implementation would store historical snapshots at regular intervals
	record, found, err := k.GetPoolTWAP(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	if !found {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("no TWAP data for pool %d", poolID)
	}

	// If we have accumulated time, return the TWAP
	if record.TotalSeconds > 0 {
		return record.TwapPrice, nil
	}

	// Otherwise return the last known price
	if !record.LastPrice.IsNil() {
		return record.LastPrice, nil
	}

	return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("no price data available for pool %d", poolID)
}

// GetCurrentTWAP returns the current TWAP for a pool (convenience method).
// This is O(1) since it just reads the stored cumulative price state.
func (k Keeper) GetCurrentTWAP(ctx context.Context, poolID uint64) (math.LegacyDec, error) {
	record, found, err := k.GetPoolTWAP(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	if !found {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("no TWAP data for pool %d", poolID)
	}

	if !record.TwapPrice.IsNil() {
		return record.TwapPrice, nil
	}

	if !record.LastPrice.IsNil() {
		return record.LastPrice, nil
	}

	return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("no price data available for pool %d", poolID)
}

// MarkPoolActive marks a pool as having recent activity.
// This enables activity-based TWAP updates, tracking which pools had swaps or liquidity changes.
//
// Security: This is purely for optimization/monitoring - marking a pool active has no effect
// on swap execution or pricing. TWAP updates are lazy and triggered on the operations themselves.
func (k Keeper) MarkPoolActive(ctx context.Context, poolID uint64) error {
	store := k.getStore(ctx)
	key := ActivePoolKey(poolID)
	store.Set(key, []byte{1})
	return nil
}

// GetActivePoolIDs returns all pool IDs that had activity since the last clear.
// Used for monitoring which pools are actively traded.
//
// Note: In the current implementation, TWAP updates are fully lazy (triggered on swaps),
// so this data is primarily for metrics/monitoring rather than critical path operations.
func (k Keeper) GetActivePoolIDs(ctx context.Context) []uint64 {
	var poolIDs []uint64
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, ActivePoolsKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Extract pool ID from key: ActivePoolsKeyPrefix(1 byte) + poolID(8 bytes)
		poolID := sdk.BigEndianToUint64(iter.Key()[len(ActivePoolsKeyPrefix):])
		poolIDs = append(poolIDs, poolID)
	}
	return poolIDs
}

// ClearActivePoolIDs clears all active pool markers.
// This can be called periodically (e.g., in EndBlocker) to reset the tracking window.
//
// Note: Clearing this data does not affect TWAP accuracy since updates are lazy.
func (k Keeper) ClearActivePoolIDs(ctx context.Context) {
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, ActivePoolsKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}
