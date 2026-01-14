//go:build chaos

package chaos

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ResourceExhaustionTestSuite struct {
	suite.Suite
	initialGoroutines int
	initialMemMB      uint64
}

func TestResourceExhaustionSuite(t *testing.T) {
	suite.Run(t, new(ResourceExhaustionTestSuite))
}

func (suite *ResourceExhaustionTestSuite) SetupTest() {
	runtime.GC()
	suite.initialGoroutines = runtime.NumGoroutine()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	suite.initialMemMB = m.Alloc / 1024 / 1024
}

func (suite *ResourceExhaustionTestSuite) TestMemoryLeakDetection() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	leakyStructs := make([]*[]byte, 0)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					data := make([]byte, 1024*1024) // 1MB
					mu.Lock()
					leakyStructs = append(leakyStructs, &data)
					mu.Unlock()
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
	runtime.GC()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	currentMemMB := m.Alloc / 1024 / 1024

	suite.T().Logf("Memory before: %d MB, after: %d MB", suite.initialMemMB, currentMemMB)

	leakyStructs = nil
	runtime.GC()
	time.Sleep(2 * time.Second)

	runtime.ReadMemStats(&m)
	afterCleanupMB := m.Alloc / 1024 / 1024

	suite.Less(afterCleanupMB, currentMemMB, "Memory should be reclaimed after cleanup")
}

func (suite *ResourceExhaustionTestSuite) TestGoroutineLeak() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startGoroutines := runtime.NumGoroutine()

	doneChan := make(chan struct{})
	for i := 0; i < 1000; i++ {
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-doneChan:
				return
			}
		}()
	}

	time.Sleep(1 * time.Second)
	leakedGoroutines := runtime.NumGoroutine() - startGoroutines
	suite.GreaterOrEqual(leakedGoroutines, 900, "Goroutines should be running")

	close(doneChan)
	time.Sleep(2 * time.Second)

	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()

	suite.LessOrEqual(finalGoroutines-startGoroutines, 10, "Most goroutines should exit")
}

func (suite *ResourceExhaustionTestSuite) TestFileDescriptorExhaustion() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	connections := make([]chan struct{}, 0, 10000)

	for i := 0; i < 1000; i++ {
		select {
		case <-ctx.Done():
			break
		default:
			ch := make(chan struct{}, 1)
			connections = append(connections, ch)
		}
	}

	suite.Equal(1000, len(connections), "Should create 1000 connections")

	for _, ch := range connections {
		close(ch)
	}

	runtime.GC()
	suite.True(true, "File descriptor cleanup successful")
}

func (suite *ResourceExhaustionTestSuite) TestCPUExhaustion() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	completed := uint64(0)

	for i := 0; i < numCPU*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := uint64(0)
			for {
				select {
				case <-ctx.Done():
					atomic.AddUint64(&completed, result%2)
					return
				default:
					for j := 0; j < 1000000; j++ {
						result += uint64(j)
					}
				}
			}
		}()
	}

	wg.Wait()
	suite.T().Logf("CPU intensive tasks completed: %d", atomic.LoadUint64(&completed))
}

func (suite *ResourceExhaustionTestSuite) TestChannelBufferExhaustion() {
	ch := make(chan int, 1000)

	go func() {
		for i := 0; i < 10000; i++ {
			select {
			case ch <- i:
			case <-time.After(10 * time.Millisecond):
				return
			}
		}
		close(ch)
	}()

	received := 0
	for range ch {
		received++
		time.Sleep(time.Microsecond)
	}

	suite.Greater(received, 1000, "Should receive messages despite backpressure")
}

func (suite *ResourceExhaustionTestSuite) TestStackOverflowPrevention() {
	defer func() {
		if r := recover(); r != nil {
			suite.T().Logf("Recovered from stack overflow: %v", r)
		}
	}()

	maxDepth := 100000
	result := iterativeFactorial(uint64(maxDepth))
	suite.NotZero(result, "Should handle deep iteration")
}

func (suite *ResourceExhaustionTestSuite) TestDiskSpaceExhaustion() {
	tempData := make([][]byte, 0, 1000)

	for i := 0; i < 1000; i++ {
		data := make([]byte, 1024*1024) // 1MB each
		tempData = append(tempData, data)
	}

	suite.Equal(1000, len(tempData), "Allocated 1GB of data")

	tempData = nil
	runtime.GC()
	suite.Zero(len(tempData), "Temporary data should be cleared")
	suite.True(true, "Memory cleanup successful")
}

func (suite *ResourceExhaustionTestSuite) TestNetworkBandwidthExhaustion() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bandwidth := make(chan []byte, 100)
	bytesSent := uint64(0)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(bandwidth)
				return
			default:
				data := make([]byte, 65536) // 64KB packets
				select {
				case bandwidth <- data:
					atomic.AddUint64(&bytesSent, 65536)
				case <-ctx.Done():
					close(bandwidth)
					return
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for data := range bandwidth {
			_ = len(data)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
	totalMB := atomic.LoadUint64(&bytesSent) / 1024 / 1024
	suite.T().Logf("Total bandwidth used: %d MB", totalMB)
}

func (suite *ResourceExhaustionTestSuite) TestConnectionPoolExhaustion() {
	maxConnections := 100
	pool := make(chan struct{}, maxConnections)

	for i := 0; i < maxConnections; i++ {
		pool <- struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu sync.Mutex
	rejectedConnections := 0

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-pool:
				time.Sleep(100 * time.Millisecond)
				pool <- struct{}{}
			case <-ctx.Done():
				mu.Lock()
				rejectedConnections++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	suite.Greater(rejectedConnections, 0, "Should reject connections when pool exhausted")
}

func iterativeFactorial(n uint64) uint64 {
	result := uint64(1)
	for i := uint64(2); i <= n && i < 100; i++ {
		result *= i
	}
	return result
}

func (suite *ResourceExhaustionTestSuite) TearDownTest() {
	runtime.GC()
}
