package nonce

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// mockErrorProvider implements ErrorProvider for testing
type mockErrorProvider struct{}

func (m *mockErrorProvider) InvalidNonceError(msg string) error {
	return &nonceError{msg: "invalid nonce: " + msg}
}

func (m *mockErrorProvider) InvalidPacketError(msg string) error {
	return &nonceError{msg: "invalid packet: " + msg}
}

type nonceError struct {
	msg string
}

func (e *nonceError) Error() string {
	return e.msg
}

// setupNonceManagerTest creates a test environment for nonce manager
func setupNonceManagerTest(t *testing.T) (*Manager, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey("test_nonce")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	header := cmtproto.Header{
		Time: time.Now().UTC(),
	}
	ctx := sdk.NewContext(stateStore, header, false, log.NewNopLogger())

	manager := NewManager(storeKey, &mockErrorProvider{}, "test_module")
	return manager, ctx
}

// TestSEC1_5_NonceVersionRotation tests that nonces rotate epochs correctly
// when approaching overflow threshold (SEC-1.5 fix)
func TestSEC1_5_NonceVersionRotation(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	channel := "channel-0"
	sender := "sender1"

	// Test normal nonce generation
	nonce1 := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(1), nonce1)

	nonce2 := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(2), nonce2)

	// Verify epoch is still 0
	epoch := manager.GetCurrentEpoch(ctx, channel, sender)
	require.Equal(t, uint64(0), epoch)
}

// TestSEC1_5_EpochRotationAtThreshold tests that epoch rotates when
// nonce reaches the threshold
func TestSEC1_5_EpochRotationAtThreshold(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	channel := "channel-0"
	sender := "sender1"

	// Manually set the nonce to just below threshold
	manager.setSendNonce(ctx, channel, sender, NonceRotationThreshold-1)

	// Get next nonce - should be threshold
	nonce := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, NonceRotationThreshold, nonce)

	// Verify epoch is still 0
	epoch := manager.GetCurrentEpoch(ctx, channel, sender)
	require.Equal(t, uint64(0), epoch)

	// Next call should trigger rotation
	nonce = manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(1), nonce, "nonce should reset to 1 after epoch rotation")

	// Verify epoch incremented
	epoch = manager.GetCurrentEpoch(ctx, channel, sender)
	require.Equal(t, uint64(1), epoch, "epoch should increment after rotation")
}

// TestSEC1_5_VersionedNonce tests that versioned nonces are generated correctly
func TestSEC1_5_VersionedNonce(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	channel := "channel-0"
	sender := "sender1"

	// Get versioned nonce
	vn := manager.NextVersionedNonce(ctx, channel, sender)
	require.Equal(t, uint64(0), vn.Epoch)
	require.Equal(t, uint64(1), vn.Nonce)

	// Get another
	vn = manager.NextVersionedNonce(ctx, channel, sender)
	require.Equal(t, uint64(0), vn.Epoch)
	require.Equal(t, uint64(2), vn.Nonce)
}

// TestSEC1_5_EpochPersistence tests that epoch values persist correctly
func TestSEC1_5_EpochPersistence(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	channel := "channel-0"
	sender := "sender1"

	// Set epoch manually
	manager.setEpoch(ctx, channel, sender, 5)

	// Read back
	epoch := manager.GetCurrentEpoch(ctx, channel, sender)
	require.Equal(t, uint64(5), epoch)
}

// TestSEC1_5_MultipleChannelEpochs tests that epochs are independent per channel
func TestSEC1_5_MultipleChannelEpochs(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	sender := "sender1"

	// Set different nonces for different channels
	manager.setSendNonce(ctx, "channel-0", sender, NonceRotationThreshold)
	manager.setSendNonce(ctx, "channel-1", sender, 10)

	// Trigger rotation on channel-0
	nonce0 := manager.NextOutboundNonce(ctx, "channel-0", sender)
	require.Equal(t, uint64(1), nonce0, "channel-0 should reset after rotation")

	// channel-1 should continue normally
	nonce1 := manager.NextOutboundNonce(ctx, "channel-1", sender)
	require.Equal(t, uint64(11), nonce1, "channel-1 should continue normally")

	// Verify epochs are different
	epoch0 := manager.GetCurrentEpoch(ctx, "channel-0", sender)
	epoch1 := manager.GetCurrentEpoch(ctx, "channel-1", sender)
	require.Equal(t, uint64(1), epoch0)
	require.Equal(t, uint64(0), epoch1)
}

// TestSEC1_5_OverflowPreventionConstants tests the overflow threshold values
func TestSEC1_5_OverflowPreventionConstants(t *testing.T) {
	// Verify threshold is about 90% of max uint64
	maxUint64 := ^uint64(0)
	threshold90Percent := maxUint64 / 10 * 9 // ~90%

	// Allow some variance (within 1%)
	require.InDelta(t, float64(threshold90Percent), float64(NonceRotationThreshold), float64(maxUint64/100))

	// Verify max nonce is actually max uint64
	require.Equal(t, maxUint64, MaxNonceValue)
}

// TestSEC1_5_IncomingNonceValidationWithEpoch tests that incoming nonce
// validation still works correctly with the epoch system
func TestSEC1_5_IncomingNonceValidationWithEpoch(t *testing.T) {
	manager, ctx := setupNonceManagerTest(t)

	channel := "channel-0"
	sender := "sender1"

	// Validate increasing nonces
	err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	err = manager.ValidateIncomingPacketNonce(ctx, channel, sender, 2, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Replay should fail
	err = manager.ValidateIncomingPacketNonce(ctx, channel, sender, 1, ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack detected")
}
