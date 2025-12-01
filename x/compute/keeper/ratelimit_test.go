package keeper_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
)

// TestRateLimiterBasic tests basic rate limiting functionality
func TestRateLimiterBasic(t *testing.T) {
	t.Parallel()

	// Create limiter with 10 requests/second, burst of 5
	limiter := keeper.NewRateLimiter(10, 5)

	clientID := "test-client"

	// First burst should be allowed (up to 5 requests)
	for i := 0; i < 5; i++ {
		allowed := limiter.Allow(clientID)
		require.True(t, allowed, "request %d should be allowed", i)
	}

	// 6th request should be denied (burst exhausted)
	allowed := limiter.Allow(clientID)
	require.False(t, allowed, "6th request should be denied")
}

// TestRateLimiterRefill tests that tokens refill over time
func TestRateLimiterRefill(t *testing.T) {
	t.Parallel()

	// Create limiter with 10 requests/second, burst of 2
	limiter := keeper.NewRateLimiter(10, 2)

	clientID := "test-client"

	// Exhaust tokens
	for i := 0; i < 2; i++ {
		limiter.Allow(clientID)
	}

	// Should be denied now
	require.False(t, limiter.Allow(clientID))

	// Wait for tokens to refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should have at least 1 token now
	require.True(t, limiter.Allow(clientID))
}

// TestRateLimiterDifferentClients tests that different clients have separate buckets
func TestRateLimiterDifferentClients(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(10, 3)

	// Exhaust client1's tokens
	for i := 0; i < 3; i++ {
		limiter.Allow("client1")
	}

	// client1 should be denied
	require.False(t, limiter.Allow("client1"))

	// client2 should still have tokens
	require.True(t, limiter.Allow("client2"))
	require.True(t, limiter.Allow("client2"))
	require.True(t, limiter.Allow("client2"))

	// Now client2 is also exhausted
	require.False(t, limiter.Allow("client2"))
}

// TestRateLimiterConcurrent tests rate limiter under concurrent access
func TestRateLimiterConcurrent(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(100, 50)

	clientID := "concurrent-client"

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Start 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow(clientID) {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should have allowed approximately the burst size
	// Allow some variance for timing
	require.GreaterOrEqual(t, allowedCount, 40, "should allow at least 40 requests")
	require.LessOrEqual(t, allowedCount, 60, "should not allow more than 60 requests")
}

// TestRateLimiterBurstCap tests that tokens don't exceed burst capacity
func TestRateLimiterBurstCap(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(100, 5)

	clientID := "burst-test"

	// Make one request to create bucket
	limiter.Allow(clientID)

	// Wait for refill (200ms at 100/sec should give us ~20 tokens)
	time.Sleep(200 * time.Millisecond)

	// But burst cap should limit it to 5
	allowedCount := 0
	for i := 0; i < 20; i++ {
		if limiter.Allow(clientID) {
			allowedCount++
		}
	}

	// Should be capped near burst size (5) plus any tokens accumulated during the loop
	require.LessOrEqual(t, allowedCount, 7, "should be limited by burst cap")
}

// TestRateLimiterZeroRate tests behavior with zero rate
func TestRateLimiterZeroRate(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(0, 3)

	clientID := "zero-rate"

	// Initial burst should work
	require.True(t, limiter.Allow(clientID))
	require.True(t, limiter.Allow(clientID))
	require.True(t, limiter.Allow(clientID))

	// After burst exhausted, no more allowed since rate is 0
	require.False(t, limiter.Allow(clientID))

	// Wait should not help since rate is 0
	time.Sleep(100 * time.Millisecond)
	require.False(t, limiter.Allow(clientID))
}

// TestRateLimiterManyClients tests with many unique clients
func TestRateLimiterManyClients(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(10, 5)

	// Create many clients
	for i := 0; i < 100; i++ {
		clientID := string(rune('a'+i%26)) + string(rune('a'+i/26))

		// Each client should get their burst allowance
		for j := 0; j < 5; j++ {
			require.True(t, limiter.Allow(clientID), "client %s request %d should be allowed", clientID, j)
		}

		// Then be denied
		require.False(t, limiter.Allow(clientID), "client %s burst should be exhausted", clientID)
	}
}

// TestNewRateLimiter tests constructor
func TestNewRateLimiter(t *testing.T) {
	t.Parallel()

	limiter := keeper.NewRateLimiter(50, 10)
	require.NotNil(t, limiter)

	// Should work with new limiter
	require.True(t, limiter.Allow("new-client"))
}

// BenchmarkRateLimiterAllow benchmarks the Allow method
func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := keeper.NewRateLimiter(1000000, 1000000) // High limits to avoid blocking

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow("benchmark-client")
		}
	})
}

// BenchmarkRateLimiterMultipleClients benchmarks with multiple clients
func BenchmarkRateLimiterMultipleClients(b *testing.B) {
	limiter := keeper.NewRateLimiter(1000000, 1000000)

	clients := []string{"client1", "client2", "client3", "client4", "client5"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			limiter.Allow(clients[i%len(clients)])
			i++
		}
	})
}
