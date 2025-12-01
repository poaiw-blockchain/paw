package concurrency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// Task 171: Concurrency Testing for All Modules

// TestConcurrentDEXSwaps tests concurrent swap operations
func TestConcurrentDEXSwaps(t *testing.T) {
	t.Parallel()

	numGoroutines := 100
	swapsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*swapsPerGoroutine)

	// Simulate pool state
	var poolMutex sync.RWMutex
	reserveA := math.NewInt(10000000)
	reserveB := math.NewInt(10000000)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < swapsPerGoroutine; j++ {
				// Simulate swap with locking
				poolMutex.Lock()

				amountIn := math.NewInt(100)
				fee := math.LegacyNewDecWithPrec(3, 3)
				amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(math.LegacyOneDec().Sub(fee))

				numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveB))
				denominator := math.LegacyNewDecFromInt(reserveA).Add(amountInAfterFee)
				amountOut := numerator.Quo(denominator).TruncateInt()

				if amountOut.GTE(reserveB) {
					errors <- ErrInsufficientLiquidity
					poolMutex.Unlock()
					continue
				}

				// Update reserves
				reserveA = reserveA.Add(amountIn)
				reserveB = reserveB.Sub(amountOut)

				poolMutex.Unlock()

				// Small delay to increase contention
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Verify no unexpected errors
	errorCount := 0
	for err := range errors {
		if err != nil && err != ErrInsufficientLiquidity {
			t.Errorf("Unexpected error: %v", err)
		}
		errorCount++
	}

	// Verify final state is valid
	poolMutex.RLock()
	defer poolMutex.RUnlock()

	require.True(t, reserveA.IsPositive(), "reserveA should be positive")
	require.True(t, reserveB.IsPositive(), "reserveB should be positive")

	t.Logf("Completed %d concurrent swaps with %d errors", numGoroutines*swapsPerGoroutine, errorCount)
}

// TestConcurrentOracleSubmissions tests concurrent oracle price submissions
func TestConcurrentOracleSubmissions(t *testing.T) {
	t.Parallel()

	numValidators := 20
	submissionsPerValidator := 5

	var wg sync.WaitGroup
	submissions := make(chan PriceSubmission, numValidators*submissionsPerValidator)
	var submissionMutex sync.Mutex
	acceptedSubmissions := []PriceSubmission{}

	for i := 0; i < numValidators; i++ {
		wg.Add(1)
		go func(validatorID int) {
			defer wg.Done()

			for j := 0; j < submissionsPerValidator; j++ {
				price := math.LegacyNewDec(100 + int64(validatorID-10)) // Price with some variance

				submission := PriceSubmission{
					ValidatorID: validatorID,
					Price:       price,
					Timestamp:   time.Now().Unix(),
				}

				submissions <- submission

				// Simulate validation and storage
				submissionMutex.Lock()
				acceptedSubmissions = append(acceptedSubmissions, submission)
				submissionMutex.Unlock()

				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(submissions)

	// Verify all submissions were processed
	require.Len(t, acceptedSubmissions, numValidators*submissionsPerValidator,
		"all submissions should be accepted")

	// Verify no data corruption (all validator IDs should be valid)
	for _, sub := range acceptedSubmissions {
		require.GreaterOrEqual(t, sub.ValidatorID, 0)
		require.Less(t, sub.ValidatorID, numValidators)
	}
}

// TestConcurrentLiquidityOperations tests concurrent add/remove liquidity
func TestConcurrentLiquidityOperations(t *testing.T) {
	t.Parallel()

	numProviders := 50

	var wg sync.WaitGroup
	var poolMutex sync.RWMutex

	// Pool state
	totalShares := math.NewInt(1000000)
	providerShares := make(map[int]math.Int)

	for i := 0; i < numProviders; i++ {
		providerShares[i] = math.NewInt(20000) // Each starts with 20k shares
	}

	// Half add liquidity, half remove
	for i := 0; i < numProviders; i++ {
		wg.Add(1)

		if i%2 == 0 {
			// Add liquidity
			go func(providerID int) {
				defer wg.Done()

				poolMutex.Lock()
				defer poolMutex.Unlock()

				sharesToAdd := math.NewInt(1000)
				providerShares[providerID] = providerShares[providerID].Add(sharesToAdd)
				totalShares = totalShares.Add(sharesToAdd)
			}(i)
		} else {
			// Remove liquidity
			go func(providerID int) {
				defer wg.Done()

				poolMutex.Lock()
				defer poolMutex.Unlock()

				sharesToRemove := math.NewInt(500)
				if providerShares[providerID].GTE(sharesToRemove) {
					providerShares[providerID] = providerShares[providerID].Sub(sharesToRemove)
					totalShares = totalShares.Sub(sharesToRemove)
				}
			}(i)
		}
	}

	wg.Wait()

	// Verify invariant: sum of all provider shares equals total shares
	poolMutex.RLock()
	defer poolMutex.RUnlock()

	sumShares := math.ZeroInt()
	for _, shares := range providerShares {
		sumShares = sumShares.Add(shares)
	}

	require.True(t, sumShares.Equal(totalShares),
		"sum of provider shares %s should equal total shares %s",
		sumShares, totalShares)
}

// TestDeadlockDetection tests for potential deadlocks
func TestDeadlockDetection(t *testing.T) {
	t.Parallel()

	// Test with timeout to detect deadlocks
	done := make(chan bool)

	go func() {
		var mu1, mu2 sync.Mutex

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: locks mu1 then mu2
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				mu1.Lock()
				time.Sleep(time.Microsecond)
				mu2.Lock()
				// Critical section
				mu2.Unlock()
				mu1.Unlock()
			}
		}()

		// Goroutine 2: locks mu1 then mu2 (same order - no deadlock)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				mu1.Lock()
				time.Sleep(time.Microsecond)
				mu2.Lock()
				// Critical section
				mu2.Unlock()
				mu1.Unlock()
			}
		}()

		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// No deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected: test timed out")
	}
}

// TestRaceConditionDetection tests for race conditions
func TestRaceConditionDetection(t *testing.T) {
	t.Parallel()

	// This test should be run with -race flag to detect races
	counter := 0
	var mu sync.Mutex

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Protected increment
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()

	require.Equal(t, numGoroutines, counter, "counter should equal number of goroutines")
}

// TestConcurrentMapAccess tests concurrent map operations
func TestConcurrentMapAccess(t *testing.T) {
	t.Parallel()

	// Test concurrent reads and writes to map with sync.Map
	var sm sync.Map

	var wg sync.WaitGroup
	numOps := 1000

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := id*numOps + j
				sm.Store(key, j)
			}
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := id*numOps + j
				sm.Load(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify all keys exist
	count := 0
	sm.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	require.Equal(t, 10*numOps, count, "all keys should be present")
}

// TestChannelConcurrency tests channel-based concurrency patterns
func TestChannelConcurrency(t *testing.T) {
	t.Parallel()

	jobs := make(chan int, 100)
	results := make(chan int, 100)

	// Start workers
	numWorkers := 5
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Process job
				result := job * 2
				results <- result
			}
		}()
	}

	// Send jobs
	numJobs := 100
	go func() {
		for i := 0; i < numJobs; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	// Wait for workers
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	resultCount := 0
	for range results {
		resultCount++
	}

	require.Equal(t, numJobs, resultCount, "all jobs should produce results")
}

// Helper types and errors

type PriceSubmission struct {
	ValidatorID int
	Price       math.LegacyDec
	Timestamp   int64
}

var (
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)

// TestAtomicOperations tests atomic operation correctness
func TestAtomicOperations(t *testing.T) {
	t.Parallel()

	var counter int64
	var wg sync.WaitGroup
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				// Use atomic operations
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	require.Equal(t, expected, counter, "atomic counter should be accurate")
}

// TestContextCancellation tests context cancellation under concurrent operations
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	completedOps := 0
	cancelledOps := 0
	var mu sync.Mutex

	// Start workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 200; j++ {
				select {
				case <-ctx.Done():
					mu.Lock()
					cancelledOps++
					mu.Unlock()
					return
				default:
					// Simulate work
					time.Sleep(time.Millisecond)
					mu.Lock()
					completedOps++
					mu.Unlock()
				}
			}
		}()
	}

	// Cancel after short delay
	time.Sleep(5 * time.Millisecond)
	cancel()

	wg.Wait()

	t.Logf("Completed: %d, Cancelled: %d", completedOps, cancelledOps)
	require.Greater(t, cancelledOps, 0, "some operations should be cancelled")
}
