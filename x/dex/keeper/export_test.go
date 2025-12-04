package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
