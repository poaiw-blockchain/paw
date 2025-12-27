package keeper_test

import (
	"sync"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TEST-5: Concurrent provider operation tests

func TestConcurrentProviderRegistration(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	t.Run("handles concurrent provider registrations", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Attempt to register 10 providers concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				provider := types.TestAddrWithSeed(idx)
				err := k.RegisterProvider(ctx, provider,
					"provider-"+string(rune('A'+idx)),
					"https://provider"+string(rune('0'+idx))+".com",
					math.NewInt(100_000))
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			require.NoError(t, err)
		}

		// Verify all providers registered
		providers, err := k.GetAllProviders(ctx)
		require.NoError(t, err)
		require.Len(t, providers, 10)
	})
}

func TestConcurrentJobSubmission(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register a provider first
	provider := types.TestAddr()
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", math.NewInt(1_000_000))
	require.NoError(t, err)

	t.Run("handles concurrent job submissions", func(t *testing.T) {
		var wg sync.WaitGroup
		requestIDs := make(chan uint64, 20)
		errors := make(chan error, 20)

		// Submit 20 jobs concurrently
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				requester := types.TestAddrWithSeed(100 + idx)

				request := &types.ComputeRequest{
					Requester:      requester.String(),
					ContainerImage: "docker.io/library/alpine:latest",
					Command:        []string{"echo", "hello"},
					MaxPayment:     math.NewInt(1000),
				}

				requestID, err := k.SubmitRequest(ctx, request)
				if err != nil {
					errors <- err
					return
				}
				requestIDs <- requestID
			}(i)
		}

		wg.Wait()
		close(requestIDs)
		close(errors)

		// Check for errors
		for err := range errors {
			require.NoError(t, err)
		}

		// Verify all requests got unique IDs
		ids := make(map[uint64]bool)
		for id := range requestIDs {
			require.False(t, ids[id], "duplicate request ID: %d", id)
			ids[id] = true
		}
		require.Len(t, ids, 20)
	})
}

func TestConcurrentResultSubmission(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Setup: register provider and create request
	provider := types.TestAddr()
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", math.NewInt(1_000_000))
	require.NoError(t, err)

	requester := types.TestAddrWithSeed(200)
	request := &types.ComputeRequest{
		Requester:      requester.String(),
		ContainerImage: "docker.io/library/alpine:latest",
		Command:        []string{"echo", "hello"},
		MaxPayment:     math.NewInt(1000),
	}
	requestID, err := k.SubmitRequest(ctx, request)
	require.NoError(t, err)

	t.Run("prevents duplicate result submission", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Try to submit same result from multiple goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				result := &types.ComputeResult{
					RequestId:  requestID,
					Provider:   provider.String(),
					OutputHash: "abc123",
					OutputUrl:  "https://storage.example.com/result",
					ExitCode:   0,
				}

				err := k.SubmitResult(ctx, result)
				mu.Lock()
				if err == nil {
					successCount++
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Only one should succeed
		require.Equal(t, 1, successCount, "only one result submission should succeed")
	})
}

func TestConcurrentProviderStakeUpdate(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", math.NewInt(1_000_000))
	require.NoError(t, err)

	t.Run("handles concurrent stake updates safely", func(t *testing.T) {
		var wg sync.WaitGroup

		// Concurrent stake additions and removals
		for i := 0; i < 10; i++ {
			wg.Add(2)

			// Add stake
			go func() {
				defer wg.Done()
				_ = k.AddProviderStake(ctx, provider, math.NewInt(1000))
			}()

			// Remove stake
			go func() {
				defer wg.Done()
				_ = k.RemoveProviderStake(ctx, provider, math.NewInt(500))
			}()
		}

		wg.Wait()

		// Verify stake is consistent (not negative, reasonable value)
		providerInfo, err := k.GetProvider(ctx, provider)
		require.NoError(t, err)
		require.True(t, providerInfo.Stake.IsPositive())
	})
}

func TestConcurrentDisputes(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Setup
	provider := types.TestAddr()
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", math.NewInt(1_000_000))
	require.NoError(t, err)

	t.Run("handles concurrent dispute creation", func(t *testing.T) {
		var wg sync.WaitGroup
		disputeIDs := make(chan uint64, 5)

		// Create 5 disputes concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				disputer := types.TestAddrWithSeed(300 + idx)
				disputeID, err := k.CreateDispute(ctx, uint64(idx+1), disputer, "test dispute")
				if err == nil {
					disputeIDs <- disputeID
				}
			}(i)
		}

		wg.Wait()
		close(disputeIDs)

		// Verify unique dispute IDs
		ids := make(map[uint64]bool)
		for id := range disputeIDs {
			require.False(t, ids[id], "duplicate dispute ID")
			ids[id] = true
		}
	})
}

func TestConcurrentReputationUpdates(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", math.NewInt(1_000_000))
	require.NoError(t, err)

	t.Run("handles concurrent reputation updates", func(t *testing.T) {
		var wg sync.WaitGroup

		// Concurrent positive and negative reputation changes
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					_ = k.RecordSuccessfulJob(ctx, provider)
				} else {
					_ = k.RecordFailedJob(ctx, provider)
				}
			}(i)
		}

		wg.Wait()

		// Verify reputation is valid
		rep, err := k.GetProviderReputation(ctx, provider)
		require.NoError(t, err)
		require.True(t, rep.GTE(math.LegacyZeroDec()))
	})
}

func TestConcurrentProviderSelection(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register multiple providers
	for i := 0; i < 5; i++ {
		provider := types.TestAddrWithSeed(400 + i)
		err := k.RegisterProvider(ctx, provider,
			"provider-"+string(rune('A'+i)),
			"https://provider.com",
			math.NewInt(int64(100_000*(i+1))))
		require.NoError(t, err)
	}

	t.Run("concurrent provider selection is thread-safe", func(t *testing.T) {
		var wg sync.WaitGroup
		selections := make(chan string, 50)

		// Concurrent provider selections
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				specs := &types.ComputeSpec{
					CpuCores: 1,
					MemoryMb: 512,
				}

				provider, err := k.SelectProvider(ctx, specs)
				if err == nil && provider != nil {
					selections <- provider.String()
				}
			}()
		}

		wg.Wait()
		close(selections)

		// Verify selections were made
		count := 0
		for range selections {
			count++
		}
		require.Greater(t, count, 0, "should have successful selections")
	})
}
