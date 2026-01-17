package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// If height <= retention threshold, cleanup should be a no-op.
func TestCleanupOldRateLimitData_NoOpEarlyBlocks(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Block height very small, cutoff <= 0 -> no cleanup
	require.NoError(t, k.CleanupOldRateLimitData(ctx))
}
