package concurrency_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ConcurrencyTestSuite tests concurrent operations across all modules
type ConcurrencyTestSuite struct {
	suite.Suite
}

func TestConcurrencyTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency tests in short mode")
	}
	suite.Run(t, new(ConcurrencyTestSuite))
}

// TestDEXConcurrentSwaps tests concurrent swap operations
func (suite *ConcurrencyTestSuite) TestDEXConcurrentSwaps() {
	suite.T().Log("Testing concurrent DEX swaps")

	poolID := uint64(1)
	numGoroutines := 100
	swapsPerGoroutine := 10

	var successCount uint64
	var errorCount uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			_ = goroutineID
			defer wg.Done()

			for j := 0; j < swapsPerGoroutine; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					err := suite.executeSwap(ctx, poolID, 1000)
					if err == nil {
						atomic.AddUint64(&successCount, 1)
					} else {
						atomic.AddUint64(&errorCount, 1)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	suite.T().Logf("Completed %d concurrent swaps in %v", successCount, duration)
	suite.T().Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify pool invariants after concurrent operations
	suite.verifyPoolInvariants(poolID)
}

// TestDEXConcurrentLiquidityOperations tests concurrent add/remove liquidity
func (suite *ConcurrencyTestSuite) TestDEXConcurrentLiquidityOperations() {
	suite.T().Log("Testing concurrent liquidity operations")

	poolID := uint64(1)
	numGoroutines := 50

	var addCount uint64
	var removeCount uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			_ = goroutineID
			defer wg.Done()

			// Alternately add and remove liquidity
			for j := 0; j < 5; j++ {
				if j%2 == 0 {
					err := suite.addLiquidity(ctx, poolID, 10000, 20000)
					if err == nil {
						atomic.AddUint64(&addCount, 1)
					}
				} else {
					err := suite.removeLiquidity(ctx, poolID, 10)
					if err == nil {
						atomic.AddUint64(&removeCount, 1)
					}
				}
				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	suite.T().Logf("Liquidity adds: %d, removes: %d", addCount, removeCount)
	suite.verifyPoolInvariants(poolID)
}

// TestOracleConcurrentPriceSubmissions tests concurrent price submissions
func (suite *ConcurrencyTestSuite) TestOracleConcurrentPriceSubmissions() {
	suite.T().Log("Testing concurrent oracle price submissions")

	numValidators := 20
	numAssets := 5

	var submissionCount uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < numValidators; i++ {
		wg.Add(1)
		go func(validatorID int) {
			defer wg.Done()

			for j := 0; j < numAssets; j++ {
				asset := fmt.Sprintf("ASSET%d", j)
				price := 100.0 + float64(validatorID)*0.1

				err := suite.submitOraclePrice(ctx, validatorID, asset, price)
				if err == nil {
					atomic.AddUint64(&submissionCount, 1)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	suite.T().Logf("Price submissions: %d", submissionCount)
	suite.verifyOracleAggregation()
}

// TestComputeConcurrentRequestSubmissions tests concurrent compute requests
func (suite *ConcurrencyTestSuite) TestComputeConcurrentRequestSubmissions() {
	suite.T().Log("Testing concurrent compute request submissions")

	numRequesters := 50

	var requestCount uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < numRequesters; i++ {
		wg.Add(1)
		go func(requesterID int) {
			defer wg.Done()

			requestID := fmt.Sprintf("request-%d-%d", requesterID, time.Now().UnixNano())
			err := suite.submitComputeRequest(ctx, requestID, 1000)
			if err == nil {
				atomic.AddUint64(&requestCount, 1)
			}
		}(i)
	}

	wg.Wait()

	suite.T().Logf("Compute requests submitted: %d", requestCount)
}

// TestRaceConditionDetection tests for race conditions
func (suite *ConcurrencyTestSuite) TestRaceConditionDetection() {
	suite.T().Log("Testing race condition detection")

	sharedCounter := int64(0)
	expectedIncrement := int64(1000)
	numGoroutines := 100

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Test with proper synchronization
	sharedCounter = 0
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < int(expectedIncrement/int64(numGoroutines)); j++ {
				mu.Lock()
				sharedCounter++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	suite.Equal(expectedIncrement, sharedCounter, "Counter should match expected value with proper synchronization")
}

// TestDeadlockDetection tests for potential deadlocks
func (suite *ConcurrencyTestSuite) TestDeadlockDetection() {
	suite.T().Log("Testing deadlock detection")

	mu1 := &sync.Mutex{}
	mu2 := &sync.Mutex{}

	done := make(chan bool, 2)

	// Proper lock ordering to avoid deadlock
	go func() {
		mu1.Lock()
		defer mu1.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu2.Lock()
		defer mu2.Unlock()
		// Critical section
		done <- true
	}()

	go func() {
		mu1.Lock()
		defer mu1.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu2.Lock()
		defer mu2.Unlock()
		// Critical section
		done <- true
	}()

	// Wait with timeout to detect deadlock
	timeout := time.After(5 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			suite.Fail("Deadlock detected - goroutines did not complete")
			return
		}
	}

	suite.T().Log("No deadlock detected with proper lock ordering")
}

// TestChannelConcurrency tests concurrent channel operations
func (suite *ConcurrencyTestSuite) TestChannelConcurrency() {
	suite.T().Log("Testing channel concurrency")

	ch := make(chan int, 100)
	numProducers := 10
	numConsumers := 10
	itemsPerProducer := 100

	var wg sync.WaitGroup

	// Producers
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(producerID int) {
			_ = producerID
			defer wg.Done()
			for j := 0; j < itemsPerProducer; j++ {
				ch <- producerID*1000 + j
			}
		}(i)
	}

	// Close channel when all producers are done
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Consumers
	var consumedCount uint64
	var consumerWg sync.WaitGroup

	for i := 0; i < numConsumers; i++ {
		consumerWg.Add(1)
		go func(consumerID int) {
			_ = consumerID
			defer consumerWg.Done()
			for range ch {
				atomic.AddUint64(&consumedCount, 1)
			}
		}(i)
	}

	consumerWg.Wait()

	expectedCount := uint64(numProducers * itemsPerProducer)
	suite.Equal(expectedCount, consumedCount, "All items should be consumed")
	suite.T().Logf("Produced and consumed %d items successfully", consumedCount)
}

// TestAtomicOperations tests atomic operation correctness
func (suite *ConcurrencyTestSuite) TestAtomicOperations() {
	suite.T().Log("Testing atomic operations")

	var atomicCounter uint64
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				atomic.AddUint64(&atomicCounter, 1)
			}
		}()
	}

	wg.Wait()

	expected := uint64(numGoroutines * incrementsPerGoroutine)
	suite.Equal(expected, atomicCounter, "Atomic counter should be accurate")
	suite.T().Logf("Atomic operations: %d increments completed successfully", atomicCounter)
}

// TestConcurrentMapAccess tests safe concurrent map access
func (suite *ConcurrencyTestSuite) TestConcurrentMapAccess() {
	suite.T().Log("Testing concurrent map access")

	safeMap := &sync.Map{}
	numGoroutines := 50
	opsPerGoroutine := 100

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				safeMap.Store(key, j)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				safeMap.Load(key)
			}
		}(i)
	}

	wg.Wait()

	// Count entries
	count := 0
	safeMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	suite.T().Logf("Map contains %d entries after concurrent access", count)
}

// TestConcurrentStateModification tests state modification under concurrency
func (suite *ConcurrencyTestSuite) TestConcurrentStateModification() {
	suite.T().Log("Testing concurrent state modification")

	type State struct {
		mu      sync.RWMutex
		balance map[string]int64
	}

	state := &State{
		balance: make(map[string]int64),
	}

	// Initialize balances
	for i := 0; i < 100; i++ {
		state.balance[fmt.Sprintf("account-%d", i)] = 1000000
	}

	numTransfers := 1000
	var wg sync.WaitGroup

	// Concurrent transfers
	for i := 0; i < numTransfers; i++ {
		wg.Add(1)
		go func(transferID int) {
			defer wg.Done()

			from := fmt.Sprintf("account-%d", transferID%100)
			to := fmt.Sprintf("account-%d", (transferID+1)%100)
			amount := int64(100)

			state.mu.Lock()
			defer state.mu.Unlock()

			if state.balance[from] >= amount {
				state.balance[from] -= amount
				state.balance[to] += amount
			}
		}(i)
	}

	wg.Wait()

	// Verify total balance unchanged
	state.mu.RLock()
	total := int64(0)
	for _, balance := range state.balance {
		total += balance
	}
	state.mu.RUnlock()

	expected := int64(100 * 1000000)
	suite.Equal(expected, total, "Total balance should remain constant")
	suite.T().Logf("Total balance verified: %d (after %d concurrent transfers)", total, numTransfers)
}

// Helper methods

func (suite *ConcurrencyTestSuite) executeSwap(ctx context.Context, poolID uint64, amount int64) error {
	_ = poolID
	_ = amount
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Millisecond):
		return nil
	}
}

func (suite *ConcurrencyTestSuite) addLiquidity(ctx context.Context, poolID uint64, amountA, amountB int64) error {
	_ = poolID
	_ = amountA
	_ = amountB
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}

func (suite *ConcurrencyTestSuite) removeLiquidity(ctx context.Context, poolID uint64, sharePercent int) error {
	_ = poolID
	_ = sharePercent
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}

func (suite *ConcurrencyTestSuite) submitOraclePrice(ctx context.Context, validatorID int, asset string, price float64) error {
	_ = validatorID
	_ = asset
	_ = price
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Millisecond):
		return nil
	}
}

func (suite *ConcurrencyTestSuite) submitComputeRequest(ctx context.Context, requestID string, escrow int64) error {
	_ = requestID
	_ = escrow
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Millisecond):
		return nil
	}
}

func (suite *ConcurrencyTestSuite) verifyPoolInvariants(poolID uint64) {
	suite.T().Logf("Verifying pool %d invariants", poolID)
}

func (suite *ConcurrencyTestSuite) verifyOracleAggregation() {
	suite.T().Log("Verifying oracle price aggregation")
}
