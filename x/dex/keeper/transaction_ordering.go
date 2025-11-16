package keeper

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// TransactionOrderingManager handles fair ordering of transactions
type TransactionOrderingManager struct {
	keeper *Keeper
}

// NewTransactionOrderingManager creates a new transaction ordering manager
func NewTransactionOrderingManager(keeper *Keeper) *TransactionOrderingManager {
	return &TransactionOrderingManager{
		keeper: keeper,
	}
}

// ValidateTransactionTimestamp validates that a transaction timestamp is reasonable
// This prevents attackers from using fake timestamps to manipulate ordering
func (tom *TransactionOrderingManager) ValidateTransactionTimestamp(
	ctx sdk.Context,
	txTimestamp time.Time,
) error {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		return nil
	}

	blockTime := ctx.BlockTime()

	// Check if timestamp is not too far in the future
	maxFutureTime := blockTime.Add(time.Duration(config.MaxReorderingWindow) * time.Second)
	if txTimestamp.After(maxFutureTime) {
		return types.ErrInvalidTransactionTimestamp.Wrapf(
			"timestamp too far in future: %s > %s",
			txTimestamp.Format(time.RFC3339),
			maxFutureTime.Format(time.RFC3339),
		)
	}

	// Check if timestamp is not too far in the past
	// We allow some buffer for network delays (2x the reordering window)
	minPastTime := blockTime.Add(-2 * time.Duration(config.MaxReorderingWindow) * time.Second)
	if txTimestamp.Before(minPastTime) {
		return types.ErrInvalidTransactionTimestamp.Wrapf(
			"timestamp too far in past: %s < %s",
			txTimestamp.Format(time.RFC3339),
			minPastTime.Format(time.RFC3339),
		)
	}

	return nil
}

// OrderTransactionsByTimestamp sorts transactions by their timestamps
// This implements fair ordering based on when users submitted transactions
func (tom *TransactionOrderingManager) OrderTransactionsByTimestamp(
	ctx sdk.Context,
	txRecords []types.TransactionRecord,
) []types.TransactionRecord {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		return txRecords
	}

	// Create a copy to avoid modifying the original slice
	orderedTxs := make([]types.TransactionRecord, len(txRecords))
	copy(orderedTxs, txRecords)

	// Sort by timestamp (ascending order - earliest first)
	sort.SliceStable(orderedTxs, func(i, j int) bool {
		return orderedTxs[i].Timestamp < orderedTxs[j].Timestamp
	})

	return orderedTxs
}

// CheckTransactionOrdering verifies that transactions are properly ordered
// Returns true if ordering is fair, false if potential manipulation detected
func (tom *TransactionOrderingManager) CheckTransactionOrdering(
	ctx sdk.Context,
	currentTx types.TransactionRecord,
	poolID uint64,
) (bool, string) {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		return true, ""
	}

	// Get recent transactions for this pool
	recentTxs := tom.keeper.GetRecentPoolTransactions(ctx, poolID, 10)
	if len(recentTxs) == 0 {
		return true, ""
	}

	// Check if current transaction timestamp is reasonable compared to recent ones
	blockTime := ctx.BlockTime().Unix()
	timeDiff := blockTime - currentTx.Timestamp

	// If timestamp is within acceptable range, ordering is fair
	if timeDiff >= 0 && timeDiff <= config.MaxReorderingWindow {
		return true, ""
	}

	// Check if there are transactions with earlier timestamps that came later
	for _, recentTx := range recentTxs {
		// If a recent transaction has a later timestamp but was processed earlier
		// This might indicate timestamp manipulation
		if recentTx.Timestamp > currentTx.Timestamp && recentTx.BlockHeight >= currentTx.BlockHeight {
			return false, fmt.Sprintf(
				"potential timestamp manipulation: current tx timestamp %d is earlier than recent tx %d",
				currentTx.Timestamp,
				recentTx.Timestamp,
			)
		}
	}

	return true, ""
}

// GetTransactionPriority calculates transaction priority for ordering
// Priority is based on timestamp (earlier = higher priority)
// This can be extended to include other factors like gas fees
func (tom *TransactionOrderingManager) GetTransactionPriority(
	ctx sdk.Context,
	tx types.TransactionRecord,
) int64 {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		// If timestamp ordering is disabled, use default priority
		return 0
	}

	// Earlier timestamp = higher priority (lower priority number)
	// We use negative timestamp so that earlier times have higher priority
	return -tx.Timestamp
}

// EnforceTimestampOrdering enforces timestamp-based ordering for a transaction
func (tom *TransactionOrderingManager) EnforceTimestampOrdering(
	ctx sdk.Context,
	trader string,
	poolID uint64,
	txTimestamp int64,
) error {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		return nil
	}

	// Get the last processed transaction timestamp for this pool
	lastTxTimestamp := tom.keeper.GetLastTransactionTimestamp(ctx, poolID)

	// Check if current transaction timestamp is within acceptable ordering window
	blockTime := ctx.BlockTime().Unix()

	// Transactions should be processed roughly in timestamp order
	// Allow some flexibility for network delays and block batching
	if lastTxTimestamp > 0 {
		// If current tx timestamp is significantly earlier than last processed tx
		// and the time difference exceeds the reordering window, reject it
		timeDiff := lastTxTimestamp - txTimestamp
		if timeDiff > config.MaxReorderingWindow {
			return types.ErrTransactionOrderingViolation.Wrapf(
				"transaction timestamp %d too far before last processed %d (diff: %d, max: %d)",
				txTimestamp,
				lastTxTimestamp,
				timeDiff,
				config.MaxReorderingWindow,
			)
		}
	}

	// Validate against block time
	blockTimeDiff := blockTime - txTimestamp
	if blockTimeDiff < 0 {
		// Transaction from the future
		if -blockTimeDiff > config.MaxReorderingWindow {
			return types.ErrTransactionOrderingViolation.Wrapf(
				"transaction timestamp %d is in the future (block time: %d)",
				txTimestamp,
				blockTime,
			)
		}
	}

	// Update last transaction timestamp
	tom.keeper.SetLastTransactionTimestamp(ctx, poolID, txTimestamp)

	// Emit event for monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTimestampOrdering,
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("tx_timestamp", fmt.Sprintf("%d", txTimestamp)),
			sdk.NewAttribute("block_time", fmt.Sprintf("%d", blockTime)),
			sdk.NewAttribute("last_tx_timestamp", fmt.Sprintf("%d", lastTxTimestamp)),
		),
	)

	return nil
}

// GetRecentPoolTransactions retrieves recent transactions for a pool
func (k Keeper) GetRecentPoolTransactions(
	ctx sdk.Context,
	poolID uint64,
	limit int,
) []types.TransactionRecord {
	store := ctx.KVStore(k.storeKey)

	// Use a prefix for pool transactions
	prefix := types.GetPoolTransactionPrefix(poolID)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var records []types.TransactionRecord
	count := 0

	// Iterate in reverse to get most recent first
	for ; iterator.Valid() && count < limit; iterator.Next() {
		var record types.TransactionRecord
		if err := json.Unmarshal(iterator.Value(), &record); err != nil {
			continue
		}
		records = append(records, record)
		count++
	}

	return records
}

// RecordTransaction stores a transaction record for MEV detection
func (k Keeper) RecordTransaction(
	ctx sdk.Context,
	txHash string,
	trader string,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, amountOut math.Int,
	priceImpact math.LegacyDec,
) {
	record := types.NewTransactionRecord(
		txHash,
		trader,
		poolID,
		tokenIn,
		tokenOut,
		amountIn,
		amountOut,
		ctx.BlockTime().Unix(),
		ctx.BlockHeight(),
		0, // TX index would need to be passed from the context
		priceImpact,
	)

	store := ctx.KVStore(k.storeKey)
	key := types.GetTransactionRecordKey(poolID, ctx.BlockHeight(), txHash)
	bz, err := json.Marshal(&record)
	if err != nil {
		return
	}
	store.Set(key, bz)

	// Also store in recent transactions cache (limited size)
	k.addToRecentTransactionsCache(ctx, poolID, record)
}

// addToRecentTransactionsCache adds a transaction to the recent cache
func (k Keeper) addToRecentTransactionsCache(
	ctx sdk.Context,
	poolID uint64,
	record types.TransactionRecord,
) {
	// Store up to 100 most recent transactions per pool
	// This is a simple implementation - production would use a circular buffer
	store := ctx.KVStore(k.storeKey)
	key := types.GetRecentTransactionKey(poolID, ctx.BlockHeight(), record.TxHash)
	bz, err := json.Marshal(&record)
	if err != nil {
		return
	}
	store.Set(key, bz)

	// Cleanup old records (older than 1000 blocks)
	k.cleanupOldTransactionRecords(ctx, poolID, ctx.BlockHeight()-1000)
}

// cleanupOldTransactionRecords removes old transaction records
func (k Keeper) cleanupOldTransactionRecords(
	ctx sdk.Context,
	poolID uint64,
	beforeHeight int64,
) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.GetPoolTransactionPrefix(poolID)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var keysToDelete [][]byte
	for ; iterator.Valid(); iterator.Next() {
		var record types.TransactionRecord
		if err := json.Unmarshal(iterator.Value(), &record); err != nil {
			continue
		}

		if record.BlockHeight < beforeHeight {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
	}

	// Delete old records
	for _, key := range keysToDelete {
		store.Delete(key)
	}
}

// GetLastTransactionTimestamp gets the last transaction timestamp for a pool
func (k Keeper) GetLastTransactionTimestamp(ctx sdk.Context, poolID uint64) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.GetLastTransactionTimestampKey(poolID)
	bz := store.Get(key)
	if bz == nil {
		return 0
	}
	return int64(sdk.BigEndianToUint64(bz))
}

// SetLastTransactionTimestamp sets the last transaction timestamp for a pool
func (k Keeper) SetLastTransactionTimestamp(ctx sdk.Context, poolID uint64, timestamp int64) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetLastTransactionTimestampKey(poolID)
	store.Set(key, sdk.Uint64ToBigEndian(uint64(timestamp)))
}

// ValidateTransactionSequence validates that a sequence of transactions follows fair ordering
func (tom *TransactionOrderingManager) ValidateTransactionSequence(
	ctx sdk.Context,
	txs []types.TransactionRecord,
) error {
	config := tom.keeper.GetMEVProtectionConfig(ctx)
	if !config.EnableTimestampOrdering {
		return nil
	}

	// Ensure transactions are ordered by timestamp
	for i := 1; i < len(txs); i++ {
		timeDiff := txs[i].Timestamp - txs[i-1].Timestamp

		// If timestamps are out of order beyond the reordering window, reject
		if timeDiff < -config.MaxReorderingWindow {
			return types.ErrTransactionOrderingViolation.Wrapf(
				"transactions out of order: tx[%d] timestamp %d before tx[%d] timestamp %d by %d seconds",
				i,
				txs[i].Timestamp,
				i-1,
				txs[i-1].Timestamp,
				-timeDiff,
			)
		}
	}

	return nil
}
