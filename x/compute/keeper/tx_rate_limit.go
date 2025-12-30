package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

var (
	// RequestRateLimitKeyPrefix is the prefix for rate limit tracking of compute requests
	// Key: prefix + requester address + day (calculated from block height)
	// Value: number of requests in that day
	RequestRateLimitKeyPrefix = []byte{0x01, 0x29}

	// RequestRateLimitByHeightPrefix is the prefix for indexing rate limits by block height for cleanup
	// Key: prefix + height + requester address + day
	// Value: dummy value (0x01)
	// This enables efficient cleanup of old rate limit data
	RequestRateLimitByHeightPrefix = []byte{0x01, 0x2A}

	// LastRequestBlockKeyPrefix is the prefix for tracking the last request block per address (for cooldown)
	// Key: prefix + requester address
	// Value: block height of last request
	LastRequestBlockKeyPrefix = []byte{0x01, 0x2B}
)

// RequestRateLimitKey returns the store key for rate limit tracking
// window represents a day calculated from block height
func RequestRateLimitKey(user sdk.AccAddress, window int64) []byte {
	windowBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(windowBytes, uint64(window))
	key := append(RequestRateLimitKeyPrefix, user.Bytes()...)
	key = append(key, windowBytes...)
	return key
}

// RequestRateLimitByHeightKey returns the index key for rate limits by height for cleanup
func RequestRateLimitByHeightKey(height int64, user sdk.AccAddress, window int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(height))
	windowBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(windowBytes, uint64(window))
	key := append(RequestRateLimitByHeightPrefix, heightBytes...)
	key = append(key, user.Bytes()...)
	key = append(key, windowBytes...)
	return key
}

// RequestRateLimitByHeightPrefixForHeight returns the prefix for all rate limits at a specific height
func RequestRateLimitByHeightPrefixForHeight(height int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(height))
	return append(RequestRateLimitByHeightPrefix, heightBytes...)
}

// LastRequestBlockKey returns the store key for tracking last request block
func LastRequestBlockKey(user sdk.AccAddress) []byte {
	return append(LastRequestBlockKeyPrefix, user.Bytes()...)
}

// CheckRequestRateLimit checks if the requester has exceeded rate limits
// Returns error if rate limit is exceeded
func (k Keeper) CheckRequestRateLimit(ctx context.Context, requester sdk.AccAddress) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("CheckRequestRateLimit: get params: %w", err)
	}

	// Check cooldown between requests
	if err := k.checkRequestCooldown(ctx, requester, params.RequestCooldownBlocks); err != nil {
		return fmt.Errorf("CheckRequestRateLimit: cooldown check: %w", err)
	}

	// Check daily request limit
	if err := k.checkDailyRequestLimit(ctx, requester, params.MaxRequestsPerAddressPerDay); err != nil {
		return fmt.Errorf("CheckRequestRateLimit: daily limit check: %w", err)
	}

	return nil
}

// checkRequestCooldown verifies the cooldown period between requests
func (k Keeper) checkRequestCooldown(ctx context.Context, requester sdk.AccAddress, cooldownBlocks uint64) error {
	if cooldownBlocks == 0 {
		return nil // Cooldown disabled
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	lastBlockKey := LastRequestBlockKey(requester)
	bz := store.Get(lastBlockKey)

	if bz != nil {
		lastBlock := int64(binary.BigEndian.Uint64(bz))
		blocksSinceLastRequest := sdkCtx.BlockHeight() - lastBlock

		if blocksSinceLastRequest < int64(cooldownBlocks) {
			blocksRemaining := int64(cooldownBlocks) - blocksSinceLastRequest
			return types.ErrRateLimitExceeded.Wrapf(
				"must wait %d blocks between compute requests (last: %d, current: %d, remaining: %d)",
				cooldownBlocks, lastBlock, sdkCtx.BlockHeight(), blocksRemaining,
			)
		}
	}

	return nil
}

// checkDailyRequestLimit verifies the daily request limit
func (k Keeper) checkDailyRequestLimit(ctx context.Context, requester sdk.AccAddress, maxRequestsPerDay uint64) error {
	if maxRequestsPerDay == 0 {
		return nil // Daily limit disabled
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Calculate current day window (blocks per day / block time)
	// Assuming ~5 second blocks: 86400 seconds / 5 = 17,280 blocks per day
	blocksPerDay := int64(17280)
	currentDay := sdkCtx.BlockHeight() / blocksPerDay

	requestCount := k.getRequestCountForDay(ctx, requester, currentDay)

	if requestCount >= maxRequestsPerDay {
		return types.ErrRateLimitExceeded.Wrapf(
			"exceeded maximum %d compute requests per day (current: %d)",
			maxRequestsPerDay, requestCount,
		)
	}

	return nil
}

// getRequestCountForDay returns the number of requests made by address in the given day
func (k Keeper) getRequestCountForDay(ctx context.Context, requester sdk.AccAddress, day int64) uint64 {
	store := k.getStore(ctx)
	key := RequestRateLimitKey(requester, day)

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

// RecordComputeRequest records a compute request submission for rate limiting
// This should be called after successful request validation but before execution
func (k Keeper) RecordComputeRequest(ctx context.Context, requester sdk.AccAddress) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Update last request block
	lastBlockKey := LastRequestBlockKey(requester)
	store.Set(lastBlockKey, sdk.Uint64ToBigEndian(uint64(sdkCtx.BlockHeight())))

	// Update daily request count
	blocksPerDay := int64(17280) // ~24 hours at 5 second blocks
	currentDay := sdkCtx.BlockHeight() / blocksPerDay

	requestCountKey := RequestRateLimitKey(requester, currentDay)
	currentCount := k.getRequestCountForDay(ctx, requester, currentDay)
	newCount := currentCount + 1
	store.Set(requestCountKey, sdk.Uint64ToBigEndian(newCount))

	// Create height index for cleanup
	heightIndexKey := RequestRateLimitByHeightKey(sdkCtx.BlockHeight(), requester, currentDay)
	store.Set(heightIndexKey, []byte{0x01})

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"request_rate_limit_recorded",
			sdk.NewAttribute("requester", requester.String()),
			sdk.NewAttribute("day", fmt.Sprintf("%d", currentDay)),
			sdk.NewAttribute("count", fmt.Sprintf("%d", newCount)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)
}

// CleanupOldRequestRateLimitData removes old rate limit tracking data to prevent state bloat
// This should be called periodically in EndBlocker
// Retention period: 48 hours worth of blocks (2 * 17,280 = 34,560 blocks)
func (k Keeper) CleanupOldRequestRateLimitData(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Rate limit data older than 48 hours can be safely removed
	// This gives a 2-day buffer beyond the 24-hour rate limit window
	const RateLimitRetentionBlocks = 34560 // ~48 hours at 5s blocks

	cutoffHeight := sdkCtx.BlockHeight() - RateLimitRetentionBlocks
	if cutoffHeight <= 0 {
		return nil
	}

	cleanedCount := 0

	// Clean up rate limit data for the past 10 blocks before cutoff
	// This batches cleanup to avoid processing all old data at once
	for height := cutoffHeight - 10; height < cutoffHeight; height++ {
		if height <= 0 {
			continue
		}

		// Get all rate limits at this height
		heightPrefix := RequestRateLimitByHeightPrefixForHeight(height)
		iterator := store.Iterator(heightPrefix, storetypes.PrefixEndBytes(heightPrefix))
		defer iterator.Close()

		rateLimitsToDelete := [][]byte{}
		for ; iterator.Valid(); iterator.Next() {
			rateLimitsToDelete = append(rateLimitsToDelete, iterator.Key())
			cleanedCount++
		}

		// Delete the rate limit index entries
		for _, key := range rateLimitsToDelete {
			store.Delete(key)

			// Also delete the actual rate limit entry
			// Key format: RequestRateLimitByHeightPrefix(2) + height(8) + user(20) + day(8) = 38 bytes minimum
			if len(key) >= 38 {
				userStart := 10 // After prefix(2) + height(8)
				userEnd := userStart + 20
				dayStart := userEnd

				if len(key) >= dayStart+8 {
					user := sdk.AccAddress(key[userStart:userEnd])
					dayBytes := key[dayStart : dayStart+8]
					day := int64(binary.BigEndian.Uint64(dayBytes))

					// Delete the actual rate limit entry
					rateLimitKey := RequestRateLimitKey(user, day)
					store.Delete(rateLimitKey)
				}
			}
		}
	}

	// Emit cleanup event
	if cleanedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"request_rate_limit_data_cleaned",
				sdk.NewAttribute("cutoff_height", fmt.Sprintf("%d", cutoffHeight)),
				sdk.NewAttribute("current_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
				sdk.NewAttribute("rate_limits_cleaned", fmt.Sprintf("%d", cleanedCount)),
			),
		)
	}

	return nil
}
