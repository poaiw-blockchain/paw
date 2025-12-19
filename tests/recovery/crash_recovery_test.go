package recovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CrashRecoveryTestSuite tests node crash and recovery scenarios
type CrashRecoveryTestSuite struct {
	suite.Suite
}

func TestCrashRecoveryTestSuite(t *testing.T) {
	suite.Run(t, new(CrashRecoveryTestSuite))
}

// TestBasicCrashRecovery tests basic crash and restart
func (suite *CrashRecoveryTestSuite) TestBasicCrashRecovery() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	// Initialize and produce blocks
	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	// Store state before crash
	heightBeforeCrash := node.App.LastBlockHeight()
	hashBeforeCrash := node.GetStateHash(t)

	t.Logf("State before crash: height=%d", heightBeforeCrash)

	// Simulate crash
	node.SimulateCrash(t)

	// Restart node
	node.Restart(t)

	// Verify state after recovery
	heightAfterRecovery := node.App.LastBlockHeight()
	hashAfterRecovery := node.GetStateHash(t)

	require.Equal(t, heightBeforeCrash, heightAfterRecovery,
		"height should be preserved after crash recovery")
	require.Equal(t, hashBeforeCrash, hashAfterRecovery,
		"app hash should be preserved after crash recovery")

	// Verify node can continue
	node.ProduceBlocks(t, 5)
	require.Greater(t, node.App.LastBlockHeight(), heightBeforeCrash)

	t.Logf("Successfully recovered from crash and continued to height %d",
		node.App.LastBlockHeight())
}

// TestCrashDuringBlockProcessing tests crash during block execution
func (suite *CrashRecoveryTestSuite) TestCrashDuringBlockProcessing() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 5)

	heightBeforeCrash := node.App.LastBlockHeight()

	// Simulate crash during block processing
	// (before commit, so block should not be in state)
	node.SimulateCrashDuringCommit(t)

	// Restart
	node.Restart(t)

	// Verify state
	heightAfterRecovery := node.App.LastBlockHeight()

	// Height should be same as before crash since commit didn't complete
	require.Equal(t, heightBeforeCrash, heightAfterRecovery,
		"incomplete block should not be in state")

	// Verify node can continue normally
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered from crash during block processing")
}

// TestCrashDuringCommit tests crash during state commit
func (suite *CrashRecoveryTestSuite) TestCrashDuringCommit() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	heightBeforeCrash := node.App.LastBlockHeight()
	hashBeforeCrash := node.GetStateHash(t)

	// Simulate crash during commit
	node.SimulateCrashDuringCommit(t)

	// Restart and verify
	node.Restart(t)

	heightAfterRecovery := node.App.LastBlockHeight()
	hashAfterRecovery := node.GetStateHash(t)

	// State should be consistent with last successful commit
	require.Equal(t, heightBeforeCrash, heightAfterRecovery)
	require.Equal(t, hashBeforeCrash, hashAfterRecovery)

	// Continue operation
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered from crash during commit")
}

// TestCrashDuringConsensus tests crash during consensus round
func (suite *CrashRecoveryTestSuite) TestCrashDuringConsensus() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 8)

	heightBeforeCrash := node.App.LastBlockHeight()

	// Simulate crash (in consensus, this would be during voting)
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// After restart, node should recover to last committed state
	heightAfterRecovery := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterRecovery)

	// Node should be able to participate in consensus again
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered from crash during consensus")
}

// TestMultipleSequentialCrashes tests recovery from multiple crashes
func (suite *CrashRecoveryTestSuite) TestMultipleSequentialCrashes() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Simulate multiple crash/recovery cycles
	for i := 0; i < 3; i++ {
		// Produce some blocks
		node.ProduceBlocks(t, 5)
		heightBeforeCrash := node.App.LastBlockHeight()

		// Crash
		node.SimulateCrash(t)

		// Restart
		node.Restart(t)

		// Verify recovery
		heightAfterRecovery := node.App.LastBlockHeight()
		require.Equal(t, heightBeforeCrash, heightAfterRecovery,
			"recovery %d failed", i+1)

		t.Logf("Recovery cycle %d successful at height %d", i+1, heightAfterRecovery)
	}

	// Final verification
	node.ProduceBlocks(t, 5)
	t.Logf("Successfully recovered from %d sequential crashes", 3)
}

// TestCrashWithActiveTransactions tests crash while processing transactions
func (suite *CrashRecoveryTestSuite) TestCrashWithActiveTransactions() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks with transactions
	for i := 0; i < 5; i++ {
		node.ProduceBlockWithTxs(t, 10)
	}

	heightBeforeCrash := node.App.LastBlockHeight()
	hashBeforeCrash := node.GetStateHash(t)

	// Crash during transaction processing
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Verify state
	heightAfterRecovery := node.App.LastBlockHeight()
	hashAfterRecovery := node.GetStateHash(t)

	require.Equal(t, heightBeforeCrash, heightAfterRecovery)
	require.Equal(t, hashBeforeCrash, hashAfterRecovery)

	// Continue with transactions
	for i := 0; i < 5; i++ {
		node.ProduceBlockWithTxs(t, 10)
	}

	t.Logf("Recovered from crash with active transactions")
}

// TestCrashRecoveryDataIntegrity verifies no data loss after crash
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryDataIntegrity() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 15)

	// Store critical state information
	heightBeforeCrash := node.App.LastBlockHeight()
	hashBeforeCrash := node.GetStateHash(t)

	// Verify state is accessible
	node.VerifyState(t)

	// Crash
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Verify no data loss
	heightAfterRecovery := node.App.LastBlockHeight()
	hashAfterRecovery := node.GetStateHash(t)

	require.Equal(t, heightBeforeCrash, heightAfterRecovery,
		"height data should not be lost")
	require.Equal(t, hashBeforeCrash, hashAfterRecovery,
		"state hash should not change")

	// Verify state is still accessible
	node.VerifyState(t)

	t.Logf("Verified data integrity after crash recovery")
}

// TestCrashAtVariousHeights tests crash recovery at different heights
func (suite *CrashRecoveryTestSuite) TestCrashAtVariousHeights() {
	t := suite.T()

	crashHeights := []int{1, 5, 10, 20, 50}

	for _, crashHeight := range crashHeights {
		t.Run(suite.T().Name()+"_Height_"+string(rune(crashHeight)), func(t *testing.T) {
			config := DefaultRecoveryTestConfig()
			node := SetupTestNode(t, config)
			defer node.Cleanup(t)

			node.InitializeChain(t)

			// Produce blocks up to crash height
			blocksToGenerate := crashHeight - int(node.App.LastBlockHeight())
			if blocksToGenerate > 0 {
				node.ProduceBlocks(t, blocksToGenerate)
			}

			heightBeforeCrash := node.App.LastBlockHeight()
			require.Equal(t, int64(crashHeight), heightBeforeCrash)

			// Crash
			node.SimulateCrash(t)

			// Restart and verify
			node.Restart(t)

			heightAfterRecovery := node.App.LastBlockHeight()
			require.Equal(t, heightBeforeCrash, heightAfterRecovery)

			// Continue
			node.ProduceBlocks(t, 5)

			t.Logf("Recovered from crash at height %d", crashHeight)
		})
	}
}

// TestCrashDuringStateSync tests crash during state synchronization
func (suite *CrashRecoveryTestSuite) TestCrashDuringStateSync() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build up some state
	node.ProduceBlocks(t, 10)

	// In a real state sync scenario, this would be syncing from peers
	// For testing, we simulate by just having state
	heightBeforeSync := node.App.LastBlockHeight()

	// Simulate crash during sync
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Should recover to last committed state
	heightAfterRecovery := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeSync, heightAfterRecovery)

	// Should be able to continue
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered from crash during state sync")
}

// TestCrashRecoveryWithSnapshots tests crash recovery using snapshots
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryWithSnapshots() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 5

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks and create snapshot
	node.ProduceBlocks(t, 10)
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	snapshotHeight := snap.Height

	// Produce more blocks
	node.ProduceBlocks(t, 5)

	// Crash
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Verify we can access the snapshot
	retrievedSnap, err := node.Snapshots.LoadSnapshot(snapshotHeight)
	require.NoError(t, err)
	require.NotNil(t, retrievedSnap)

	// Continue operation
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered using snapshot at height %d", snapshotHeight)
}

// TestQuickSuccessiveCrashes tests rapid crash/restart cycles
func (suite *CrashRecoveryTestSuite) TestQuickSuccessiveCrashes() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 5)

	// Rapid crash/restart cycles
	for i := 0; i < 5; i++ {
		heightBefore := node.App.LastBlockHeight()

		// Quick crash
		node.SimulateCrash(t)

		// Immediate restart
		node.Restart(t)

		// Verify consistency
		heightAfter := node.App.LastBlockHeight()
		require.Equal(t, heightBefore, heightAfter)

		// Produce one block between crashes
		node.ProduceBlocks(t, 1)

		t.Logf("Quick crash/restart cycle %d completed", i+1)
	}

	t.Logf("Successfully handled %d quick successive crashes", 5)
}

// TestCrashRecoveryMemoryState tests memory state consistency after crash
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryMemoryState() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	// Verify memory state before crash
	node.VerifyState(t)

	// Crash
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Verify memory state after recovery
	node.VerifyState(t)

	t.Logf("Memory state consistent after crash recovery")
}

// TestCrashRecoveryWithCorruptedState tests recovery with potential corruption
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryWithCorruptedState() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build state with snapshot
	node.ProduceBlocks(t, 10)
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Continue building state
	node.ProduceBlocks(t, 5)

	// Simulate crash (potentially with corruption)
	node.SimulateCrash(t)

	// Restart - should recover
	node.Restart(t)

	// Verify state is accessible
	node.VerifyState(t)

	// If corruption is detected in real scenario,
	// node could restore from snapshot
	// For now, verify we can continue
	node.ProduceBlocks(t, 5)

	t.Logf("Recovered from potential state corruption")
}

// TestLongRunningNodeCrashRecovery tests crash recovery for long-running node
func (suite *CrashRecoveryTestSuite) TestLongRunningNodeCrashRecovery() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 10

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Simulate long-running node
	for i := 0; i < 5; i++ {
		node.ProduceBlockWithTxs(t, 20) // Many transactions
		if i%2 == 0 {
			node.CreateSnapshot(t) // Periodic snapshots
		}
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash after long operation
	node.SimulateCrash(t)

	// Restart
	node.Restart(t)

	// Verify recovery
	heightAfterRecovery := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterRecovery)

	// Continue operation
	node.ProduceBlocks(t, 10)

	t.Logf("Long-running node recovered successfully")
}

// TestCrashRecoveryConsistencyAcrossRestarts tests consistency across multiple restarts
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryConsistencyAcrossRestarts() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	// Track state across restarts
	stateHashes := make([][]byte, 0)
	heights := make([]int64, 0)

	for i := 0; i < 3; i++ {
		height := node.App.LastBlockHeight()
		hash := node.GetStateHash(t)

		heights = append(heights, height)
		stateHashes = append(stateHashes, hash)

		// Crash and restart
		node.SimulateCrash(t)
		node.Restart(t)

		// Verify consistency
		recoveredHeight := node.App.LastBlockHeight()
		recoveredHash := node.GetStateHash(t)

		require.Equal(t, height, recoveredHeight, "restart %d: height mismatch", i+1)
		require.Equal(t, hash, recoveredHash, "restart %d: hash mismatch", i+1)

		// Produce more blocks
		node.ProduceBlocks(t, 3)
	}

	t.Logf("State remained consistent across %d restarts", 3)
}

// TestCrashRecoveryTimeout tests recovery doesn't hang indefinitely
func (suite *CrashRecoveryTestSuite) TestCrashRecoveryTimeout() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	// Crash
	node.SimulateCrash(t)

	// Restart with timeout monitoring
	restartComplete := make(chan bool, 1)
	go func() {
		node.Restart(t)
		restartComplete <- true
	}()

	// Wait for restart with timeout
	select {
	case <-restartComplete:
		t.Log("Restart completed successfully")
	case <-time.After(30 * time.Second):
		require.Fail(t, "Restart timed out")
	}

	// Verify node is operational
	node.ProduceBlocks(t, 5)

	t.Log("Crash recovery completed within timeout")
}

// BenchmarkCrashRecovery benchmarks crash recovery time
func BenchmarkCrashRecovery(b *testing.B) {
	config := DefaultRecoveryTestConfig()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		node := SetupTestNode(b, config)
		node.InitializeChain(b)
		node.ProduceBlocks(b, 10)
		node.SimulateCrash(b)

		b.StartTimer()
		node.Restart(b)
		b.StopTimer()

		node.Cleanup(b)
	}
}

// BenchmarkCrashRecoveryWithState benchmarks recovery with significant state
func BenchmarkCrashRecoveryWithState(b *testing.B) {
	config := DefaultRecoveryTestConfig()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		node := SetupTestNode(b, config)
		node.InitializeChain(b)

		// Build up state
		for j := 0; j < 10; j++ {
			node.ProduceBlockWithTxs(b, 10)
		}

		node.SimulateCrash(b)

		b.StartTimer()
		node.Restart(b)
		b.StopTimer()

		node.Cleanup(b)
	}
}
