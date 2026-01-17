package keeper_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
)

// Ensures CleanupStaleReentrancyLocks removes expired locks and leaves recent ones.
func TestCleanupStaleReentrancyLocks(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := ctx

	store := sdkCtx.KVStore(k.GetStoreKey())
	currentHeight := sdkCtx.BlockHeight()

	encode := func(h int64) []byte {
		b := make([]byte, 9)
		binary.BigEndian.PutUint64(b[:8], uint64(h))
		b[8] = 0x01
		return b
	}

	// Seed one stale lock (older than LockExpirationBlocks) and one fresh lock.
	staleKey := keeper.ReentrancyLockKey("stale_op")
	freshKey := keeper.ReentrancyLockKey("fresh_op")

	staleHeight := currentHeight - keeper.LockExpirationBlocks - 1
	freshHeight := currentHeight

	store.Set(staleKey, encode(staleHeight))
	store.Set(freshKey, encode(freshHeight))

	k.CleanupStaleReentrancyLocks(ctx)

	require.Nil(t, store.Get(staleKey))
	require.NotNil(t, store.Get(freshKey))
}
