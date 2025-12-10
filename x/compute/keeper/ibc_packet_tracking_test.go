package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func TestGetPacketNonceKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-0", 1)
		require.Equal(t, key1, key2)
	})

	t.Run("different channels produce different keys", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-1", 1)
		require.NotEqual(t, key1, key2)
	})

	t.Run("different sequences produce different keys", func(t *testing.T) {
		key1 := GetPacketNonceKey("channel-0", 1)
		key2 := GetPacketNonceKey("channel-0", 2)
		require.NotEqual(t, key1, key2)
	})
}

func TestHasPacketBeenProcessed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns false for unprocessed packet", func(t *testing.T) {
		processed := k.HasPacketBeenProcessed(sdkCtx, "channel-0", 1)
		require.False(t, processed)
	})
}

func TestMarkPacketAsProcessed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("marks packet as processed", func(t *testing.T) {
		err := k.MarkPacketAsProcessed(sdkCtx, "channel-0", 1)
		require.NoError(t, err)
		processed := k.HasPacketBeenProcessed(sdkCtx, "channel-0", 1)
		require.True(t, processed)
	})
}

func TestValidatePacketOrdering(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("first packet validates", func(t *testing.T) {
		packet := channeltypes.Packet{
			Sequence:           1,
			SourcePort:         "compute",
			SourceChannel:      "channel-0",
			DestinationPort:    "compute",
			DestinationChannel: "channel-0",
		}
		err := k.ValidatePacketOrdering(sdkCtx, packet)
		require.NoError(t, err)
	})
}

func TestGetLastProcessedSequence(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns 0 for new channel", func(t *testing.T) {
		seq := k.GetLastProcessedSequence(sdkCtx, "channel-new")
		require.Equal(t, uint64(0), seq)
	})
}

func TestSetLastProcessedSequence(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("sets and gets sequence", func(t *testing.T) {
		k.SetLastProcessedSequence(sdkCtx, "channel-0", 5)
		seq := k.GetLastProcessedSequence(sdkCtx, "channel-0")
		require.Equal(t, uint64(5), seq)
	})
}

func TestCleanupOldPacketNonces(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("cleanup with no nonces", func(t *testing.T) {
		err := k.CleanupOldPacketNonces(sdkCtx, 100)
		require.NoError(t, err)
	})
}

func TestGetJobStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("returns error for non-existent job", func(t *testing.T) {
		status, err := k.GetJobStatus(sdkCtx, "non-existent-job")
		require.Error(t, err)
		require.Nil(t, status)
	})
}
