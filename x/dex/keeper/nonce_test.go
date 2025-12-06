package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

func TestValidateIncomingPacketNonce(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	timestamp := now.Unix()

	channel := "channel-0"
	sender := "sender1"

	t.Run("success - first valid nonce", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 1, timestamp)
		require.NoError(t, err)
	})

	t.Run("success - monotonically increasing nonce", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 2, timestamp)
		require.NoError(t, err)

		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, 3, timestamp)
		require.NoError(t, err)
	})

	t.Run("fail - replay attack with same nonce", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 3, timestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("fail - replay attack with lower nonce", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 2, timestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("fail - zero nonce", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, "new_sender", 0, timestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nonce must be greater than zero")
	})

	t.Run("fail - empty channel", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, "", sender, 1, timestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "source channel missing")
	})
}

func TestValidateIncomingPacketNonce_TimestampValidation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	currentTime := now.Unix()

	channel := "channel-1"
	sender := "timestamp_sender"

	t.Run("success - recent timestamp", func(t *testing.T) {
		recentTimestamp := currentTime - 100 // 100 seconds ago
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 1, recentTimestamp)
		require.NoError(t, err)
	})

	t.Run("success - timestamp within 24 hours", func(t *testing.T) {
		timestamp23HoursAgo := currentTime - 82800 // 23 hours ago
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 2, timestamp23HoursAgo)
		require.NoError(t, err)
	})

	t.Run("fail - timestamp too old (over 24 hours)", func(t *testing.T) {
		oldTimestamp := currentTime - 86401 // 24 hours + 1 second ago
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 3, oldTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too old")
	})

	t.Run("fail - timestamp far too old", func(t *testing.T) {
		veryOldTimestamp := currentTime - 604800 // 1 week ago
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 4, veryOldTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too old")
	})

	t.Run("success - timestamp slightly in future (clock drift)", func(t *testing.T) {
		futureTimestamp := currentTime + 100 // 100 seconds in future (within 5 min tolerance)
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 5, futureTimestamp)
		require.NoError(t, err)
	})

	t.Run("fail - timestamp too far in future", func(t *testing.T) {
		farFutureTimestamp := currentTime + 400 // 400 seconds in future (> 5 min tolerance)
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 6, farFutureTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too far in future")
	})

	t.Run("fail - zero timestamp", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 7, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "timestamp must be positive")
	})

	t.Run("fail - negative timestamp", func(t *testing.T) {
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 8, -100)
		require.Error(t, err)
		require.Contains(t, err.Error(), "timestamp must be positive")
	})
}

func TestValidateIncomingPacketNonce_MultipleChannels(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	timestamp := now.Unix()

	sender := "multi_channel_sender"

	t.Run("different channels have independent nonce sequences", func(t *testing.T) {
		// Channel 0
		err := k.ValidateIncomingPacketNonce(ctx, "channel-0", sender, 1, timestamp)
		require.NoError(t, err)

		err = k.ValidateIncomingPacketNonce(ctx, "channel-0", sender, 2, timestamp)
		require.NoError(t, err)

		// Channel 1 starts fresh
		err = k.ValidateIncomingPacketNonce(ctx, "channel-1", sender, 1, timestamp)
		require.NoError(t, err)

		// Channel 0 continues
		err = k.ValidateIncomingPacketNonce(ctx, "channel-0", sender, 3, timestamp)
		require.NoError(t, err)

		// Channel 1 continues
		err = k.ValidateIncomingPacketNonce(ctx, "channel-1", sender, 2, timestamp)
		require.NoError(t, err)
	})
}

func TestValidateIncomingPacketNonce_MultipleSenders(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	timestamp := now.Unix()

	channel := "channel-2"

	t.Run("different senders have independent nonce sequences", func(t *testing.T) {
		// Sender A
		err := k.ValidateIncomingPacketNonce(ctx, channel, "senderA", 1, timestamp)
		require.NoError(t, err)

		err = k.ValidateIncomingPacketNonce(ctx, channel, "senderA", 2, timestamp)
		require.NoError(t, err)

		// Sender B starts fresh
		err = k.ValidateIncomingPacketNonce(ctx, channel, "senderB", 1, timestamp)
		require.NoError(t, err)

		// Sender A continues
		err = k.ValidateIncomingPacketNonce(ctx, channel, "senderA", 3, timestamp)
		require.NoError(t, err)

		// Sender B continues
		err = k.ValidateIncomingPacketNonce(ctx, channel, "senderB", 2, timestamp)
		require.NoError(t, err)
	})
}

func TestValidateIncomingPacketNonce_ReplayAttackScenarios(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	currentTime := now.Unix()

	channel := "channel-replay"
	sender := "attacker"

	t.Run("cannot replay old packet with same nonce and timestamp", func(t *testing.T) {
		timestamp := currentTime - 100
		nonce := uint64(10)

		// First packet succeeds
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, nonce, timestamp)
		require.NoError(t, err)

		// Replay attempt fails (same nonce)
		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, nonce, timestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("cannot replay old packet with newer timestamp but old nonce", func(t *testing.T) {
		oldNonce := uint64(11)
		newTimestamp := currentTime

		// First packet
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, oldNonce, currentTime-200)
		require.NoError(t, err)

		// Advance nonce
		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, 12, currentTime-100)
		require.NoError(t, err)

		// Try to replay with fresh timestamp but old nonce - should fail
		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, oldNonce, newTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("cannot use old packet even if timestamp is modified", func(t *testing.T) {
		nonce := uint64(15)

		// Original packet
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, nonce, currentTime-300)
		require.NoError(t, err)

		// Try to replay with modified timestamp (attacker trying to bypass timestamp check)
		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, nonce, currentTime)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("old packet with expired timestamp is rejected", func(t *testing.T) {
		expiredTimestamp := time.Now().Unix() - 86401 // 24+ hours ago
		newNonce := uint64(20)

		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, newNonce, expiredTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too old")
	})
}
