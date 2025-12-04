package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestRateLimitTokenExhaustion ensures that once the burst bucket is empty, the next request is rejected.
func TestRateLimitTokenExhaustion(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("rate_limit_request"))

	now := ctx.BlockTime()
	bucket := types.RateLimitBucket{
		Account:          requester.String(),
		Tokens:           1,
		MaxTokens:        1,
		RefillRate:       1,
		LastRefill:       now,
		RequestsThisHour: 0,
		RequestsToday:    0,
		HourResetAt:      now.Add(time.Hour),
		DayResetAt:       now.Add(24 * time.Hour),
	}
	require.NoError(t, k.SetRateLimitBucket(ctx, bucket))

	require.NoError(t, k.CheckRateLimit(ctx, requester))

	err := k.CheckRateLimit(ctx, requester)
	require.ErrorIs(t, err, types.ErrRateLimitExceeded)
}
