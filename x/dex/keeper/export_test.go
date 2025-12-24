package keeper

import (
	"context"
	"encoding/binary"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// StorePendingRemoteSwapForTest exposes the internal pending swap storage helper for white-box tests.
func StorePendingRemoteSwapForTest(k *Keeper, ctx sdk.Context, channelID string, swapSeq, transferSeq uint64, sender string, amountIn math.Int, step SwapStep) {
	k.storePendingRemoteSwap(ctx, channelID, swapSeq, transferSeq, sender, amountIn, step)
}

// StorePendingQueryForTest exposes the pending query helper so keeper_test packages can seed state.
func StorePendingQueryForTest(k *Keeper, ctx sdk.Context, channelID string, sequence uint64, chainID, tokenA, tokenB string) {
	k.storePendingQuery(ctx, channelID, sequence, chainID, tokenA, tokenB)
}

// SetRateLimitEntryForTest seeds a rate limit bucket and its height index for cleanup tests.
func SetRateLimitEntryForTest(k *Keeper, ctx sdk.Context, height int64, user sdk.AccAddress, window int64) {
	store := k.getStore(ctx)
	store.Set(RateLimitKey(user, window), []byte{0x01})
	store.Set(RateLimitByHeightKey(height, user, window), []byte{0x01})
}

// RateLimitEntryExistsForTest reports if the primary rate limit entry remains in store.
func RateLimitEntryExistsForTest(k *Keeper, ctx sdk.Context, user sdk.AccAddress, window int64) bool {
	store := k.getStore(ctx)
	return store.Has(RateLimitKey(user, window))
}

// RateLimitIndexExistsForTest reports if the rate limit index entry for a specific height remains.
func RateLimitIndexExistsForTest(k *Keeper, ctx sdk.Context, height int64, user sdk.AccAddress, window int64) bool {
	store := k.getStore(ctx)
	return store.Has(RateLimitByHeightKey(height, user, window))
}

// GetPoolCreationCountForTesting exposes the pool creation counter for unit tests.
func (k Keeper) GetPoolCreationCountForTesting(ctx context.Context, creator sdk.AccAddress, currentHeight int64) int {
	return k.getPoolCreationCount(ctx, creator, currentHeight)
}

// SetPoolCreationRecordForTesting seeds a pool creation record (testing only).
func (k Keeper) SetPoolCreationRecordForTesting(ctx context.Context, creator sdk.AccAddress, blockHeight int64) {
	store := k.getStore(ctx)
	key := append([]byte("pool_creation_count/"), creator.Bytes()...)
	key = append(key, sdk.Uint64ToBigEndian(uint64(blockHeight))...)
	store.Set(key, sdk.Uint64ToBigEndian(uint64(blockHeight)))
}

// ListPoolCreationRecordsForTesting lists remaining records for assertions (testing only).
func (k Keeper) ListPoolCreationRecordsForTesting(ctx context.Context, creator sdk.AccAddress) []int64 {
	store := k.getStore(ctx)
	prefix := append([]byte("pool_creation_count/"), creator.Bytes()...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var heights []int64
	for ; iter.Valid(); iter.Next() {
		heights = append(heights, int64(binary.BigEndian.Uint64(iter.Value())))
	}
	return heights
}

// CheckPoolPriceDeviationForTesting exposes price deviation checks to tests without relaxing production visibility.
func (k Keeper) CheckPoolPriceDeviationForTesting(ctx context.Context, pool *types.Pool, operation string) error {
	return k.checkPoolPriceDeviation(ctx, pool, operation)
}
