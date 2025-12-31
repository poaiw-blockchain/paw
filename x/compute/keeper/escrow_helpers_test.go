package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestRecordCatastrophicFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	addr := sdk.AccAddress([]byte("test_address"))
	amount := math.NewInt(1000)

	k.recordCatastrophicFailure(ctx, 1, addr, amount, "test failure reason")

	events := sdkCtx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "escrow_catastrophic_failure" {
			found = true
			for _, attr := range event.Attributes {
				if attr.Key == "severity" {
					require.Equal(t, "CRITICAL", attr.Value)
				}
				if attr.Key == "reason" {
					require.Equal(t, "test failure reason", attr.Value)
				}
				if attr.Key == "request_id" {
					require.Equal(t, "1", attr.Value)
				}
			}
		}
	}
	require.True(t, found, "catastrophic failure event should be emitted")
}

func TestRecordEscrowWarning(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.recordEscrowWarning(ctx, 42, "test warning message")

	events := sdkCtx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "escrow_warning" {
			found = true
			for _, attr := range event.Attributes {
				if attr.Key == "request_id" {
					require.Equal(t, "42", attr.Value)
				}
				if attr.Key == "message" {
					require.Equal(t, "test warning message", attr.Value)
				}
			}
		}
	}
	require.True(t, found, "escrow warning event should be emitted")
}

func TestRemoveEscrowTimeoutIndexSlow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requestID := uint64(123)
	expiresAt := time.Now().Add(time.Hour)

	err := k.setEscrowTimeoutIndex(ctx, requestID, expiresAt)
	require.NoError(t, err)

	k.removeEscrowTimeoutIndexSlow(ctx, requestID)

	var found bool
	err = k.IterateEscrowTimeouts(ctx, expiresAt.Add(time.Second), func(rid uint64, _ time.Time) (bool, error) {
		if rid == requestID {
			found = true
		}
		return false, nil
	})
	require.NoError(t, err)
	require.False(t, found, "timeout index should be removed")
}

func TestEscrowStateKey(t *testing.T) {
	t.Run("generates correct key format", func(t *testing.T) {
		key := EscrowStateKey(123)
		require.NotNil(t, key)
		require.Len(t, key, 9)
		require.Equal(t, byte(0x20), key[0])
	})

	t.Run("different IDs produce different keys", func(t *testing.T) {
		key1 := EscrowStateKey(1)
		key2 := EscrowStateKey(2)
		require.NotEqual(t, key1, key2)
	})

	t.Run("same ID produces same key", func(t *testing.T) {
		key1 := EscrowStateKey(100)
		key2 := EscrowStateKey(100)
		require.Equal(t, key1, key2)
	})

	t.Run("max uint64", func(t *testing.T) {
		key := EscrowStateKey(^uint64(0))
		require.NotNil(t, key)
		require.Len(t, key, 9)
	})
}

func TestEscrowTimeoutKey(t *testing.T) {
	t.Run("generates correct key format", func(t *testing.T) {
		now := time.Now()
		key := EscrowTimeoutKey(now, 123)
		require.NotNil(t, key)
		require.Len(t, key, 17)
		require.Equal(t, byte(0x21), key[0])
	})

	t.Run("different times produce different keys", func(t *testing.T) {
		t1 := time.Now()
		t2 := t1.Add(time.Hour)
		key1 := EscrowTimeoutKey(t1, 123)
		key2 := EscrowTimeoutKey(t2, 123)
		require.NotEqual(t, key1, key2)
	})

	t.Run("different request IDs produce different keys", func(t *testing.T) {
		t1 := time.Now()
		key1 := EscrowTimeoutKey(t1, 1)
		key2 := EscrowTimeoutKey(t1, 2)
		require.NotEqual(t, key1, key2)
	})

	t.Run("keys are ordered by time", func(t *testing.T) {
		t1 := time.Unix(1000, 0)
		t2 := time.Unix(2000, 0)
		key1 := EscrowTimeoutKey(t1, 1)
		key2 := EscrowTimeoutKey(t2, 1)
		require.True(t, string(key1) < string(key2))
	})
}

func TestIterateEscrowTimeouts(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	now := time.Now()
	requests := []uint64{1, 2, 3, 4, 5}

	for i, rid := range requests {
		expiresAt := now.Add(time.Duration(i) * time.Minute)
		err := k.setEscrowTimeoutIndex(ctx, rid, expiresAt)
		require.NoError(t, err)
	}

	t.Run("iterate finds all before time", func(t *testing.T) {
		var found []uint64
		err := k.IterateEscrowTimeouts(ctx, now.Add(10*time.Minute), func(rid uint64, _ time.Time) (bool, error) {
			found = append(found, rid)
			return false, nil
		})
		require.NoError(t, err)
		require.Len(t, found, 5)
	})

	t.Run("iterate respects stop signal", func(t *testing.T) {
		var count int
		err := k.IterateEscrowTimeouts(ctx, now.Add(10*time.Minute), func(rid uint64, _ time.Time) (bool, error) {
			count++
			return count >= 2, nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, count)
	})

	t.Run("iterate with early cutoff time", func(t *testing.T) {
		var found []uint64
		err := k.IterateEscrowTimeouts(ctx, now.Add(30*time.Second), func(rid uint64, _ time.Time) (bool, error) {
			found = append(found, rid)
			return false, nil
		})
		require.NoError(t, err)
		require.Len(t, found, 1)
	})
}

func TestSetAndGetEscrowState(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	now := time.Now()
	state := types.EscrowState{
		RequestId: 1,
		Requester: "cosmos1requester",
		Provider:  "cosmos1provider",
		Amount:    math.NewInt(1000),
		Status:    types.ESCROW_STATUS_LOCKED,
		LockedAt:  now,
		ExpiresAt: now.Add(time.Hour),
		Nonce:     42,
	}

	err := k.SetEscrowState(ctx, state)
	require.NoError(t, err)

	retrieved, err := k.GetEscrowState(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, state.RequestId, retrieved.RequestId)
	require.Equal(t, state.Requester, retrieved.Requester)
	require.Equal(t, state.Provider, retrieved.Provider)
	require.True(t, state.Amount.Equal(retrieved.Amount))
	require.Equal(t, state.Status, retrieved.Status)
	require.Equal(t, state.Nonce, retrieved.Nonce)
}

func TestGetEscrowState_NotFound(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	_, err := k.GetEscrowState(ctx, 999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestGetNextEscrowNonce(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	// SEC-2.5: Nonces now include entropy mixing, so we test uniqueness instead of sequential values
	nonce1, err := k.getNextEscrowNonce(ctx)
	require.NoError(t, err)
	require.NotZero(t, nonce1)

	nonce2, err := k.getNextEscrowNonce(ctx)
	require.NoError(t, err)
	require.NotZero(t, nonce2)
	require.NotEqual(t, nonce1, nonce2, "nonces should be unique")

	nonce3, err := k.getNextEscrowNonce(ctx)
	require.NoError(t, err)
	require.NotZero(t, nonce3)
	require.NotEqual(t, nonce1, nonce3, "nonces should be unique")
	require.NotEqual(t, nonce2, nonce3, "nonces should be unique")
}

func TestSetEscrowTimeoutIndex(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("set single timeout", func(t *testing.T) {
		err := k.setEscrowTimeoutIndex(ctx, 1, time.Now().Add(time.Hour))
		require.NoError(t, err)
	})

	t.Run("set multiple timeouts", func(t *testing.T) {
		for i := uint64(10); i < 15; i++ {
			err := k.setEscrowTimeoutIndex(ctx, i, time.Now().Add(time.Duration(i)*time.Minute))
			require.NoError(t, err)
		}
	})
}
