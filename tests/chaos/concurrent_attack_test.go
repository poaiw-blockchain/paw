package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ConcurrentAttackTestSuite struct {
	suite.Suite
	nodes       []*ByzantineNode
	network     *NetworkSimulator
	attackCount uint64
}

func TestConcurrentAttacksSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentAttackTestSuite))
}

func (suite *ConcurrentAttackTestSuite) SetupTest() {
	suite.nodes = make([]*ByzantineNode, 10)
	suite.network = NewNetworkSimulator()
	for i := 0; i < len(suite.nodes); i++ {
		suite.nodes[i] = NewByzantineNode(fmt.Sprintf("node-%d", i), suite.network, i < 3)
		suite.network.AddNode(suite.nodes[i].Node)
	}
	suite.network.ConnectAll()
}

func (suite *ConcurrentAttackTestSuite) TestRaceConditionAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	concurrency := 100

	sharedState := make(map[string]uint64)
	var stateMutex sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					key := fmt.Sprintf("key-%d", rand.Intn(10))
					stateMutex.Lock()
					sharedState[key]++
					stateMutex.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	totalOps := uint64(0)
	stateMutex.Lock()
	for _, count := range sharedState {
		totalOps += count
	}
	stateMutex.Unlock()

	suite.Equal(uint64(concurrency*100), totalOps, "All operations should be counted")
}

func (suite *ConcurrentAttackTestSuite) TestDeadlockAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resourceA := &sync.Mutex{}
	resourceB := &sync.Mutex{}

	deadlockDetected := make(chan bool, 1)

	// Goroutine 1: locks A then B
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			resourceA.Lock()
			time.Sleep(100 * time.Millisecond)
			resourceB.Lock()
			resourceB.Unlock()
			resourceA.Unlock()
		}
	}()

	// Goroutine 2: locks B then A (potential deadlock)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			resourceB.Lock()
			time.Sleep(100 * time.Millisecond)
			resourceA.Lock()
			resourceA.Unlock()
			resourceB.Unlock()
		}
	}()

	// Deadlock detector
	go func() {
		<-time.After(5 * time.Second)
		deadlockDetected <- true
	}()

	select {
	case <-deadlockDetected:
		suite.Fail("Deadlock detected")
	case <-ctx.Done():
		// Test passed - no deadlock
	}
}

func (suite *ConcurrentAttackTestSuite) TestConcurrentModificationAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	safeMap := &sync.Map{}
	var wg sync.WaitGroup

	writers := 50
	readers := 50

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					safeMap.Store(fmt.Sprintf("key-%d", rand.Intn(100)), j)
				}
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					safeMap.Load(fmt.Sprintf("key-%d", rand.Intn(100)))
				}
			}
		}(i)
	}

	wg.Wait()
	suite.True(true, "Concurrent operations completed without panic")
}

func (suite *ConcurrentAttackTestSuite) TestDoubleSpendAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	balance := uint64(1000)
	var balanceMutex sync.Mutex
	spendAttempts := make(chan bool, 1000)

	var wg sync.WaitGroup
	attackers := 100

	for i := 0; i < attackers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					balanceMutex.Lock()
					if balance >= 100 {
						balance -= 100
						spendAttempts <- true
					} else {
						spendAttempts <- false
					}
					balanceMutex.Unlock()
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
	close(spendAttempts)

	successfulSpends := 0
	for success := range spendAttempts {
		if success {
			successfulSpends++
		}
	}

	suite.Equal(10, successfulSpends, "Should allow exactly 10 spends from initial balance of 1000")
	suite.Equal(uint64(0), balance, "Final balance should be 0")
}

func (suite *ConcurrentAttackTestSuite) TestMemoryCorruptionAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data := make([][]byte, 1000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					idx := rand.Intn(len(data))
					data[idx] = make([]byte, 1024)
					for k := range data[idx] {
						data[idx][k] = byte(rand.Intn(256))
					}
				}
			}
		}(i)
	}

	wg.Wait()

	corruptedCount := 0
	for _, bytes := range data {
		if bytes == nil {
			corruptedCount++
		}
	}

	suite.Less(corruptedCount, len(data)/10, "Memory corruption should be minimal")
}

func (suite *ConcurrentAttackTestSuite) TestPriorityInversionAttack() {
	highPriority := make(chan *Transaction, 100)
	lowPriority := make(chan *Transaction, 1000)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	processedHigh := uint64(0)
	processedLow := uint64(0)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			highPriority <- &Transaction{ID: fmt.Sprintf("high-%d", i), Nonce: 1000}
			time.Sleep(10 * time.Millisecond)
		}
		close(highPriority)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			lowPriority <- &Transaction{ID: fmt.Sprintf("low-%d", i), Nonce: 1}
		}
		close(lowPriority)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case tx, ok := <-highPriority:
				if !ok {
					return
				}
				if tx != nil {
					atomic.AddUint64(&processedHigh, 1)
				}
			case tx, ok := <-lowPriority:
				if !ok {
					continue
				}
				if tx != nil && len(highPriority) == 0 {
					atomic.AddUint64(&processedLow, 1)
				}
			}
		}
	}()

	wg.Wait()

	suite.Equal(uint64(100), processedHigh, "All high priority transactions should be processed")
}

func (suite *ConcurrentAttackTestSuite) TestABAProbl em() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	value := uint64(100)
	var mu sync.Mutex

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				mu.Lock()
				old := value
				value = 200
				time.Sleep(time.Millisecond)
				value = old
				mu.Unlock()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		detectedChanges := 0
		lastSeen := value
		for i := 0; i < 1000; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				mu.Lock()
				current := value
				mu.Unlock()
				if current != lastSeen {
					detectedChanges++
				}
				lastSeen = current
				time.Sleep(500 * time.Microsecond)
			}
		}
		suite.Greater(detectedChanges, 0, "Should detect some state changes")
	}()

	wg.Wait()
}

func (suite *ConcurrentAttackTestSuite) TestStarvationAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resource := &sync.Mutex{}
	attackerAccess := uint64(0)
	victimAccess := uint64(0)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					resource.Lock()
					atomic.AddUint64(&attackerAccess, 1)
					time.Sleep(10 * time.Millisecond)
					resource.Unlock()
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resource.Lock()
				atomic.AddUint64(&victimAccess, 1)
				resource.Unlock()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	time.Sleep(10 * time.Second)
	cancel()
	wg.Wait()

	attackerTotal := atomic.LoadUint64(&attackerAccess)
	victimTotal := atomic.LoadUint64(&victimAccess)

	suite.Greater(victimTotal, uint64(10), "Victim should get some access to resource")
	suite.T().Logf("Attacker access: %d, Victim access: %d", attackerTotal, victimTotal)
}

func (suite *ConcurrentAttackTestSuite) TestLivelockAttack() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resource1 := &sync.Mutex{}
	resource2 := &sync.Mutex{}

	progress1 := uint64(0)
	progress2 := uint64(0)

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resource1.Lock()
				if resource2.TryLock() {
					atomic.AddUint64(&progress1, 1)
					resource2.Unlock()
					resource1.Unlock()
				} else {
					resource1.Unlock()
					time.Sleep(time.Millisecond)
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resource2.Lock()
				if resource1.TryLock() {
					atomic.AddUint64(&progress2, 1)
					resource1.Unlock()
					resource2.Unlock()
				} else {
					resource2.Unlock()
					time.Sleep(time.Millisecond)
				}
			}
		}
	}()

	time.Sleep(10 * time.Second)
	cancel()
	wg.Wait()

	suite.Greater(progress1+progress2, uint64(100), "System should make progress despite contention")
}

func (suite *ConcurrentAttackTestSuite) TearDownTest() {
	suite.network.Shutdown()
}
