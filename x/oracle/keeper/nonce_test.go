package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

func TestValidateIncomingPacketNonce(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)

	channel := "channel-0"
	sender := "sender1"
	timestamp := now.Unix()

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
	k, ctx := keepertest.OracleKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)

	channel := "channel-1"
	sender := "timestamp_sender"
	currentTime := now.Unix()

	t.Run("success - recent timestamp", func(t *testing.T) {
		recentTimestamp := currentTime - 100
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 1, recentTimestamp)
		require.NoError(t, err)
	})

	t.Run("success - timestamp within 24 hours", func(t *testing.T) {
		timestamp23HoursAgo := currentTime - 82800
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 2, timestamp23HoursAgo)
		require.NoError(t, err)
	})

	t.Run("fail - timestamp too old (over 24 hours)", func(t *testing.T) {
		oldTimestamp := currentTime - 86401
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 3, oldTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too old")
	})

	t.Run("success - timestamp slightly in future (clock drift)", func(t *testing.T) {
		futureTimestamp := currentTime + 100
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, 5, futureTimestamp)
		require.NoError(t, err)
	})

	t.Run("fail - timestamp too far in future", func(t *testing.T) {
		farFutureTimestamp := currentTime + 400
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

func TestValidateIncomingPacketNonce_ReplayAttackScenarios(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Set block time to current time so timestamp validation works
	now := time.Now()
	ctx = ctx.WithBlockTime(now)

	channel := "channel-replay"
	sender := "attacker"
	currentTime := now.Unix()

	t.Run("cannot replay old packet with same nonce and timestamp", func(t *testing.T) {
		timestamp := currentTime - 100
		nonce := uint64(10)

		// First packet succeeds
		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, nonce, timestamp)
		require.NoError(t, err)

		// Replay attempt fails
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

		// Try to replay with fresh timestamp but old nonce
		err = k.ValidateIncomingPacketNonce(ctx, channel, sender, oldNonce, newTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack detected")
	})

	t.Run("old packet with expired timestamp is rejected", func(t *testing.T) {
		expiredTimestamp := time.Now().Unix() - 86401
		newNonce := uint64(20)

		err := k.ValidateIncomingPacketNonce(ctx, channel, sender, newNonce, expiredTimestamp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "packet timestamp too old")
	})
}
