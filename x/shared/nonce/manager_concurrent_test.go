package nonce_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestConcurrentNextOutboundNonce tests NextOutboundNonce with concurrent goroutines
// using different channels/senders (realistic usage pattern).
// NOTE: In production, SDK contexts are not shared across goroutines. Each request
// has its own context. This test simulates multiple concurrent requests.
func TestConcurrentNextOutboundNonce(t *testing.T) {
	manager, ctx := setupManager(t)

	const goroutines = 100
	const noncesPerGoroutine = 10

	var wg sync.WaitGroup
	results := make(chan struct {
		channel string
		sender  string
		nonce   uint64
	}, goroutines*noncesPerGoroutine)

	// Each goroutine uses a different channel/sender pair (realistic)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			channel := fmt.Sprintf("channel-%d", idx)
			sender := fmt.Sprintf("sender-%d", idx)
			for j := 0; j < noncesPerGoroutine; j++ {
				n := manager.NextOutboundNonce(ctx, channel, sender)
				results <- struct {
					channel string
					sender  string
					nonce   uint64
				}{channel, sender, n}
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect results per channel/sender
	nonces := make(map[string][]uint64)
	for r := range results {
		key := fmt.Sprintf("%s:%s", r.channel, r.sender)
		nonces[key] = append(nonces[key], r.nonce)
	}

	// Verify each channel/sender has correct nonces
	for key, vals := range nonces {
		require.Len(t, vals, noncesPerGoroutine, "key %s", key)

		// Nonces should be sequential for each channel/sender
		expected := make(map[uint64]bool)
		for i := 1; i <= noncesPerGoroutine; i++ {
			expected[uint64(i)] = true
		}

		for _, n := range vals {
			require.True(t, expected[n], "unexpected nonce %d for %s", n, key)
		}
	}
}

// TestConcurrentValidateIncomingPacketNonce tests ValidateIncomingPacketNonce
// with concurrent goroutines using different channels (realistic usage).
func TestConcurrentValidateIncomingPacketNonce(t *testing.T) {
	manager, ctx := setupManager(t)

	const goroutines = 50

	var wg sync.WaitGroup
	results := make(chan error, goroutines)

	// Each goroutine validates packets on a different channel
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			channel := fmt.Sprintf("channel-%d", idx)
			sender := fmt.Sprintf("sender-%d", idx)
			err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 1, ctx.BlockTime().Unix())
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	// All validations should succeed since each uses a different channel
	for err := range results {
		require.NoError(t, err)
	}
}

// TestConcurrentReplayAttackDetection tests that concurrent replay attack attempts
// are all correctly detected and rejected.
func TestConcurrentReplayAttackDetection(t *testing.T) {
	manager, ctx := setupManager(t)

	const channel = "channel-0"
	const sender = "sender1"

	// First, set a valid nonce
	err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 5, ctx.BlockTime().Unix())
	require.NoError(t, err)

	const goroutines = 100
	var wg sync.WaitGroup
	results := make(chan error, goroutines)

	// All goroutines try to replay the same nonce
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 5, ctx.BlockTime().Unix())
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	// All attempts should fail with replay attack error
	for err := range results {
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack")
	}
}

// TestConcurrentMultipleSenders tests concurrent nonce validation from multiple senders.
func TestConcurrentMultipleSenders(t *testing.T) {
	manager, ctx := setupManager(t)

	const channel = "channel-0"
	const senders = 20
	const noncesPerSender = 50

	var wg sync.WaitGroup
	results := make(chan error, senders*noncesPerSender)

	// Each sender validates their own sequence of nonces
	for s := 0; s < senders; s++ {
		sender := fmt.Sprintf("sender%d", s)
		wg.Add(1)
		go func(senderID string) {
			defer wg.Done()
			for n := 1; n <= noncesPerSender; n++ {
				err := manager.ValidateIncomingPacketNonce(ctx, channel, senderID, uint64(n), ctx.BlockTime().Unix())
				results <- err
			}
		}(sender)
	}

	wg.Wait()
	close(results)

	// All validations should succeed since senders are independent
	for err := range results {
		require.NoError(t, err)
	}
}

// TestConcurrentMultipleChannels tests concurrent nonce validation across multiple channels.
func TestConcurrentMultipleChannels(t *testing.T) {
	manager, ctx := setupManager(t)

	const channels = 20
	const noncesPerChannel = 50
	const sender = "sender1"

	var wg sync.WaitGroup
	results := make(chan error, channels*noncesPerChannel)

	// Each channel validates its own sequence of nonces
	for c := 0; c < channels; c++ {
		channel := fmt.Sprintf("channel-%d", c)
		wg.Add(1)
		go func(ch string) {
			defer wg.Done()
			for n := 1; n <= noncesPerChannel; n++ {
				err := manager.ValidateIncomingPacketNonce(ctx, ch, sender, uint64(n), ctx.BlockTime().Unix())
				results <- err
			}
		}(channel)
	}

	wg.Wait()
	close(results)

	// All validations should succeed since channels are independent
	for err := range results {
		require.NoError(t, err)
	}
}

// TestConcurrentOutboundAndInbound tests concurrent outbound nonce generation
// and inbound nonce validation on the same channel/sender pair.
func TestConcurrentOutboundAndInbound(t *testing.T) {
	manager, ctx := setupManager(t)

	const channel = "channel-0"
	const sender = "sender1"
	const goroutines = 100

	var wg sync.WaitGroup

	// Goroutines generating outbound nonces
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.NextOutboundNonce(ctx, channel, sender)
		}()
	}

	// Goroutines validating inbound nonces
	for i := 1; i <= goroutines; i++ {
		wg.Add(1)
		nonce := uint64(i)
		go func(n uint64) {
			defer wg.Done()
			_ = manager.ValidateIncomingPacketNonce(ctx, channel, sender, n, ctx.BlockTime().Unix())
		}(nonce)
	}

	wg.Wait()

	// Test should complete without deadlock or panic
}

// TestConcurrentDifferentOperations tests a mix of different concurrent operations.
func TestConcurrentDifferentOperations(t *testing.T) {
	manager, ctx := setupManager(t)

	const duration = 100 * time.Millisecond
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Goroutine 1: Generate outbound nonces for channel-0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = manager.NextOutboundNonce(ctx, "channel-0", "sender1")
			}
		}
	}()

	// Goroutine 2: Generate outbound nonces for channel-1
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = manager.NextOutboundNonce(ctx, "channel-1", "sender2")
			}
		}
	}()

	// Goroutine 3: Validate increasing inbound nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		nonce := uint64(1)
		for {
			select {
			case <-done:
				return
			default:
				_ = manager.ValidateIncomingPacketNonce(ctx, "channel-2", "sender3", nonce, ctx.BlockTime().Unix())
				nonce++
			}
		}
	}()

	// Let them run for a bit
	time.Sleep(duration)
	close(done)
	wg.Wait()

	// Test should complete without deadlock, panic, or race conditions
}

// TestRaceConditionDetection tests that there are no race conditions in the manager.
// This test is designed to be run with -race flag.
func TestRaceConditionDetection(t *testing.T) {
	manager, ctx := setupManager(t)

	const goroutines = 100
	var wg sync.WaitGroup

	// Mix of operations that could potentially race
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			channel := fmt.Sprintf("channel-%d", idx%5)
			sender := fmt.Sprintf("sender-%d", idx%3)

			// Each goroutine does multiple operations
			_ = manager.NextOutboundNonce(ctx, channel, sender)
			_ = manager.ValidateIncomingPacketNonce(ctx, channel, sender, uint64(idx+1), ctx.BlockTime().Unix())
			_ = manager.NextOutboundNonce(ctx, channel, sender)
		}(i)
	}

	wg.Wait()
}

// TestHighVolumeNonceGeneration tests nonce generation under high volume (sequential).
func TestHighVolumeNonceGeneration(t *testing.T) {
	manager, ctx := setupManager(t)

	const total = 10000
	const channel = "channel-0"
	const sender = "sender1"

	for i := 1; i <= total; i++ {
		n := manager.NextOutboundNonce(ctx, channel, sender)
		require.Equal(t, uint64(i), n, "expected nonce %d, got %d", i, n)
	}

	// Next nonce should be total+1
	n := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(total+1), n)
}

// TestStressIncomingNonceValidation tests incoming nonce validation under stress.
func TestStressIncomingNonceValidation(t *testing.T) {
	manager, ctx := setupManager(t)

	const total = 10000
	const channel = "channel-0"
	const sender = "sender1"

	for i := 1; i <= total; i++ {
		err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, uint64(i), ctx.BlockTime().Unix())
		require.NoError(t, err, "failed at nonce %d", i)
	}

	// Verify replay attack is still detected after stress test
	err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, uint64(total/2), ctx.BlockTime().Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack")
}
