package nonce_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/shared/nonce"
)

// MockErrorProvider implements ErrorProvider for testing.
type MockErrorProvider struct{}

func (m *MockErrorProvider) InvalidNonceError(msg string) error {
	return &testError{msg: "invalid nonce: " + msg}
}

func (m *MockErrorProvider) InvalidPacketError(msg string) error {
	return &testError{msg: "invalid packet: " + msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func setupManager(t *testing.T) (*nonce.Manager, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx = ctx.WithBlockTime(time.Now())

	errorProvider := &MockErrorProvider{}
	manager := nonce.NewManager(storeKey, errorProvider, "testmodule")

	return manager, ctx
}

func TestValidateIncomingPacketNonce_Success(t *testing.T) {
	manager, ctx := setupManager(t)

	// First nonce should succeed
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Second nonce (higher) should succeed
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 2, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Different sender can have same nonce
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender2", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

func TestValidateIncomingPacketNonce_ReplayAttack(t *testing.T) {
	manager, ctx := setupManager(t)

	// First nonce should succeed
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 5, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Same nonce should fail (replay attack)
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 5, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack")

	// Lower nonce should fail
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 3, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack")
}

func TestValidateIncomingPacketNonce_ZeroNonce(t *testing.T) {
	manager, ctx := setupManager(t)

	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 0, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "greater than zero")
}

func TestValidateIncomingPacketNonce_EmptyChannel(t *testing.T) {
	manager, ctx := setupManager(t)

	err := manager.ValidateIncomingPacketNonce(ctx, "", "sender1", 1, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "source channel")
}

func TestValidateIncomingPacketNonce_InvalidTimestamp(t *testing.T) {
	manager, ctx := setupManager(t)

	// Zero timestamp
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")

	// Negative timestamp
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, -1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "positive")
}

func TestValidateIncomingPacketNonce_TimestampTooOld(t *testing.T) {
	manager, ctx := setupManager(t)

	// Timestamp more than 24 hours old
	oldTimestamp := ctx.BlockTime().Unix() - nonce.MaxTimestampAge - 1
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, oldTimestamp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too old")
}

func TestValidateIncomingPacketNonce_TimestampTooFuture(t *testing.T) {
	manager, ctx := setupManager(t)

	// Timestamp more than 5 minutes in the future
	futureTimestamp := ctx.BlockTime().Unix() + nonce.MaxFutureDrift + 1
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, futureTimestamp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "future")
}

func TestNextOutboundNonce(t *testing.T) {
	manager, ctx := setupManager(t)

	// First nonce should be 1
	nonce1 := manager.NextOutboundNonce(ctx, "channel-0", "sender1")
	require.Equal(t, uint64(1), nonce1)

	// Second should be 2
	nonce2 := manager.NextOutboundNonce(ctx, "channel-0", "sender1")
	require.Equal(t, uint64(2), nonce2)

	// Different channel starts at 1
	nonce3 := manager.NextOutboundNonce(ctx, "channel-1", "sender1")
	require.Equal(t, uint64(1), nonce3)

	// Different sender starts at 1
	nonce4 := manager.NextOutboundNonce(ctx, "channel-0", "sender2")
	require.Equal(t, uint64(1), nonce4)
}

func TestNextOutboundNonce_EmptyChannel(t *testing.T) {
	manager, ctx := setupManager(t)

	// Empty channel should be normalized to "unknown"
	nonce1 := manager.NextOutboundNonce(ctx, "", "sender1")
	require.Equal(t, uint64(1), nonce1)

	// Same normalized channel should increment
	nonce2 := manager.NextOutboundNonce(ctx, "", "sender1")
	require.Equal(t, uint64(2), nonce2)
}

func TestNonceSeparation(t *testing.T) {
	manager, ctx := setupManager(t)

	// Outbound nonce
	outbound := manager.NextOutboundNonce(ctx, "channel-0", "sender1")
	require.Equal(t, uint64(1), outbound)

	// Incoming nonce should be independent - can use same value
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Both should continue independently
	outbound2 := manager.NextOutboundNonce(ctx, "channel-0", "sender1")
	require.Equal(t, uint64(2), outbound2)

	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 2, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

func TestEmptySenderNormalization(t *testing.T) {
	manager, ctx := setupManager(t)

	// Empty sender should be normalized to module name
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Using module name explicitly should conflict (same normalized key)
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "testmodule", 1, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack")

	// But nonce 2 should work
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "testmodule", 2, ctx.BlockTime().Unix())
	require.NoError(t, err)
}
