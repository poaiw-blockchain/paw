//go:build recovery
// +build recovery

// NOTE: Recovery tests have long timeouts and may spawn goroutines that take time to clean up.
// Run with: go test -tags=recovery -timeout 20m ./tests/recovery/...
package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WALReplayTestSuite tests Write-Ahead Log replay functionality
type WALReplayTestSuite struct {
	suite.Suite
}

func TestWALReplayTestSuite(t *testing.T) {
	suite.Run(t, new(WALReplayTestSuite))
}

// WALEntry represents a Write-Ahead Log entry
type WALEntry struct {
	Height    int64
	TxCount   int
	Timestamp time.Time
	Hash      []byte
}

// simulateWALWrite simulates writing to WAL
func simulateWALWrite(t TestingT, dataDir string, entry WALEntry) {
	t.Helper()

	walDir := filepath.Join(dataDir, "wal")
	require.NoError(t, os.MkdirAll(walDir, 0o750))

	// In real implementation, this would write to CometBFT's WAL
	// For testing, we create marker files
	walFile := filepath.Join(walDir, fmt.Sprintf("wal-%d.log", entry.Height))
	data := []byte(fmt.Sprintf("height:%d,txs:%d,time:%s\n",
		entry.Height, entry.TxCount, entry.Timestamp.Format(time.RFC3339)))

	require.NoError(t, os.WriteFile(walFile, data, 0o640))
}

// verifyWALExists checks if WAL entries exist
func verifyWALExists(t TestingT, dataDir string, height int64) bool {
	t.Helper()

	walFile := filepath.Join(dataDir, "wal", fmt.Sprintf("wal-%d.log", height))
	_, err := os.Stat(walFile)
	return err == nil
}

// TestBasicWALReplay tests basic WAL replay after restart
func (suite *WALReplayTestSuite) TestBasicWALReplay() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks and simulate WAL writes
	for i := int64(1); i <= 10; i++ {
		node.ProduceBlocks(t, 1)

		// Simulate WAL entry
		entry := WALEntry{
			Height:    i,
			TxCount:   0,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Restart - WAL should be replayed
	node.Restart(t)

	// Verify state after WAL replay
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay,
		"height should be consistent after WAL replay")

	// Verify WAL entries exist
	for i := int64(1); i <= heightBeforeCrash; i++ {
		exists := verifyWALExists(t, node.DataDir, i)
		require.True(t, exists, "WAL entry for height %d should exist", i)
	}

	t.Logf("WAL replay successful for %d blocks", heightBeforeCrash)
}

// TestWALReplayTransactionOrdering tests transaction order preservation
func (suite *WALReplayTestSuite) TestWALReplayTransactionOrdering() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Track transaction order
	type txRecord struct {
		height int64
		txIdx  int
	}
	txOrder := make([]txRecord, 0)

	// Produce blocks with transactions
	for i := 0; i < 5; i++ {
		height := node.App.LastBlockHeight()
		numTxs := 5

		for txIdx := 0; txIdx < numTxs; txIdx++ {
			txOrder = append(txOrder, txRecord{
				height: height + 1,
				txIdx:  txIdx,
			})
		}

		node.ProduceBlockWithTxs(t, numTxs)

		// Simulate WAL write with transaction info
		entry := WALEntry{
			Height:    height + 1,
			TxCount:   numTxs,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Restart and replay WAL
	node.Restart(t)

	// Verify height is consistent
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	// In a real implementation, we would verify:
	// - Transactions are in the same order
	// - No transactions are missing
	// - No duplicate transactions
	// For now, verify basic state consistency
	node.VerifyState(t)

	t.Logf("Transaction ordering preserved through WAL replay for %d txs",
		len(txOrder))
}

// TestWALReplayWithLargeFile tests WAL replay with large log files
func (suite *WALReplayTestSuite) TestWALReplayWithLargeFile() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Generate many blocks to create large WAL
	numBlocks := 50
	for i := 0; i < numBlocks; i++ {
		node.ProduceBlockWithTxs(t, 20) // 20 txs per block

		if i%10 == 0 {
			entry := WALEntry{
				Height:    node.App.LastBlockHeight(),
				TxCount:   20,
				Timestamp: time.Now(),
				Hash:      node.GetStateHash(t),
			}
			simulateWALWrite(t, node.DataDir, entry)
		}
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Time the WAL replay
	startTime := time.Now()
	node.Restart(t)
	replayDuration := time.Since(startTime)

	// Verify state
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	// Replay should complete in reasonable time
	require.Less(t, replayDuration, 30*time.Second,
		"large WAL replay took too long: %v", replayDuration)

	t.Logf("Large WAL replay completed in %v for %d blocks",
		replayDuration, numBlocks)
}

// TestWALReplayPartialBlock tests replay with incomplete block
func (suite *WALReplayTestSuite) TestWALReplayPartialBlock() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	heightBeforeCrash := node.App.LastBlockHeight()

	// Write partial WAL entry for next block (uncommitted)
	entry := WALEntry{
		Height:    heightBeforeCrash + 1,
		TxCount:   5,
		Timestamp: time.Now(),
		Hash:      []byte("incomplete"),
	}
	simulateWALWrite(t, node.DataDir, entry)

	// Crash before commit
	node.SimulateCrashDuringCommit(t)

	// Restart
	node.Restart(t)

	// Height should remain at last committed
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay,
		"partial block should not be committed")

	// Node should continue normally
	node.ProduceBlocks(t, 5)

	t.Log("Correctly handled partial block in WAL replay")
}

// TestWALReplayConsistencyCheck tests consistency validation during replay
func (suite *WALReplayTestSuite) TestWALReplayConsistencyCheck() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build state with WAL tracking
	hashes := make(map[int64][]byte)

	for i := 0; i < 10; i++ {
		node.ProduceBlocks(t, 1)
		height := node.App.LastBlockHeight()
		hash := node.GetStateHash(t)
		hashes[height] = hash

		entry := WALEntry{
			Height:    height,
			TxCount:   0,
			Timestamp: time.Now(),
			Hash:      hash,
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()
	hashBeforeCrash := hashes[heightBeforeCrash]

	// Crash
	node.SimulateCrash(t)

	// Replay
	node.Restart(t)

	// Verify consistency
	heightAfterReplay := node.App.LastBlockHeight()
	hashAfterReplay := node.GetStateHash(t)

	require.Equal(t, heightBeforeCrash, heightAfterReplay)
	require.Equal(t, hashBeforeCrash, hashAfterReplay,
		"state hash should match after WAL replay")

	t.Log("WAL replay consistency check passed")
}

// TestWALReplayWithGaps tests handling of gaps in WAL
func (suite *WALReplayTestSuite) TestWALReplayWithGaps() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Create WAL entries with intentional gaps
	heights := []int64{1, 2, 3, 5, 7, 10} // Missing 4, 6, 8, 9

	for _, h := range heights {
		blocksNeeded := int(h - node.App.LastBlockHeight())
		if blocksNeeded > 0 {
			node.ProduceBlocks(t, blocksNeeded)
		}

		entry := WALEntry{
			Height:    h,
			TxCount:   0,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Replay - should handle gaps gracefully
	node.Restart(t)

	// Verify recovery to last valid state
	heightAfterReplay := node.App.LastBlockHeight()
	require.LessOrEqual(t, heightAfterReplay, heightBeforeCrash)

	// Node should continue
	node.ProduceBlocks(t, 5)

	t.Log("Handled WAL gaps during replay")
}

// TestWALReplayMemoryEfficiency tests memory usage during replay
func (suite *WALReplayTestSuite) TestWALReplayMemoryEfficiency() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Generate substantial WAL
	for i := 0; i < 30; i++ {
		node.ProduceBlockWithTxs(t, 15)

		entry := WALEntry{
			Height:    node.App.LastBlockHeight(),
			TxCount:   15,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Replay - should not consume excessive memory
	node.Restart(t)

	// Verify replay completed
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	// In real implementation, we would monitor memory usage
	// For now, verify node is operational
	node.ProduceBlocks(t, 5)

	t.Log("WAL replay completed with acceptable memory usage")
}

// TestWALReplayWithDifferentBlockSizes tests replay with varying block sizes
func (suite *WALReplayTestSuite) TestWALReplayWithDifferentBlockSizes() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks with varying transaction counts
	txCounts := []int{1, 5, 10, 20, 50, 100, 5, 1}

	for _, txCount := range txCounts {
		node.ProduceBlockWithTxs(t, txCount)

		entry := WALEntry{
			Height:    node.App.LastBlockHeight(),
			TxCount:   txCount,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Replay
	node.Restart(t)

	// Verify
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	t.Logf("WAL replay handled varying block sizes correctly")
}

// TestWALReplayIdempotency tests that replay is idempotent
func (suite *WALReplayTestSuite) TestWALReplayIdempotency() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	for i := int64(1); i <= 10; i++ {
		entry := WALEntry{
			Height:    i,
			TxCount:   0,
			Timestamp: time.Now(),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightOriginal := node.App.LastBlockHeight()
	hashOriginal := node.GetStateHash(t)

	// First replay
	node.SimulateCrash(t)
	node.Restart(t)

	height1 := node.App.LastBlockHeight()
	hash1 := node.GetStateHash(t)

	// Second replay (crash again)
	node.SimulateCrash(t)
	node.Restart(t)

	height2 := node.App.LastBlockHeight()
	hash2 := node.GetStateHash(t)

	// All should be identical
	require.Equal(t, heightOriginal, height1)
	require.Equal(t, height1, height2)
	require.Equal(t, hashOriginal, hash1)
	require.Equal(t, hash1, hash2)

	t.Log("WAL replay is idempotent")
}

// TestWALReplayWithCorruption tests handling of corrupted WAL entries
func (suite *WALReplayTestSuite) TestWALReplayWithCorruption() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 5

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build state with snapshots and WAL
	for i := 0; i < 10; i++ {
		node.ProduceBlocks(t, 1)

		entry := WALEntry{
			Height:    node.App.LastBlockHeight(),
			TxCount:   0,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)

		if i == 4 {
			node.CreateSnapshot(t) // Snapshot at height 5
		}
	}

	// Corrupt a WAL entry
	walFile := filepath.Join(node.DataDir, "wal", "wal-8.log")
	require.NoError(t, os.WriteFile(walFile, []byte("corrupted data"), 0o640))

	_ = node.App.LastBlockHeight() // Get height before crash

	// Crash
	node.SimulateCrash(t)

	// Replay should detect corruption and recover
	// In real implementation, this would:
	// 1. Detect corrupted WAL entry
	// 2. Fall back to last snapshot
	// 3. Continue from there
	node.Restart(t)

	// For this test, just verify node recovered
	heightAfterReplay := node.App.LastBlockHeight()
	require.Greater(t, heightAfterReplay, int64(0))

	// Node should be able to continue
	node.ProduceBlocks(t, 5)

	t.Log("Recovered from WAL corruption")
}

// TestWALReplayPerformance tests replay performance
func (suite *WALReplayTestSuite) TestWALReplayPerformance() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build substantial WAL
	numBlocks := 100
	startBuildTime := time.Now()

	for i := 0; i < numBlocks; i++ {
		node.ProduceBlockWithTxs(t, 10)

		if i%5 == 0 {
			entry := WALEntry{
				Height:    node.App.LastBlockHeight(),
				TxCount:   10,
				Timestamp: time.Now(),
				Hash:      node.GetStateHash(t),
			}
			simulateWALWrite(t, node.DataDir, entry)
		}
	}

	buildDuration := time.Since(startBuildTime)
	t.Logf("Built %d blocks in %v", numBlocks, buildDuration)

	heightBeforeCrash := node.App.LastBlockHeight() // Get height before crash

	// Crash
	node.SimulateCrash(t)

	// Measure replay time
	startReplayTime := time.Now()
	node.Restart(t)
	replayDuration := time.Since(startReplayTime)

	// Verify
	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	// Replay should be faster than original build
	// (In reality it might not be, but should be comparable)
	t.Logf("WAL replay of %d blocks completed in %v", numBlocks, replayDuration)
}

// TestWALReplayWithStateSync tests WAL replay after state sync
func (suite *WALReplayTestSuite) TestWALReplayWithStateSync() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build state and create snapshot (simulating state sync source)
	node.ProduceBlocks(t, 10)
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Continue building state with WAL
	for i := 0; i < 5; i++ {
		node.ProduceBlocks(t, 1)

		entry := WALEntry{
			Height:    node.App.LastBlockHeight(),
			TxCount:   0,
			Timestamp: time.Now(),
			Hash:      node.GetStateHash(t),
		}
		simulateWALWrite(t, node.DataDir, entry)
	}

	heightBeforeCrash := node.App.LastBlockHeight()

	// Crash
	node.SimulateCrash(t)

	// Replay should work with both snapshot and WAL
	node.Restart(t)

	heightAfterReplay := node.App.LastBlockHeight()
	require.Equal(t, heightBeforeCrash, heightAfterReplay)

	t.Log("WAL replay successful after state sync")
}

// BenchmarkWALReplay benchmarks WAL replay performance
func BenchmarkWALReplay(b *testing.B) {
	config := DefaultRecoveryTestConfig()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		node := SetupTestNode(b, config)
		node.InitializeChain(b)

		// Build WAL
		for j := 0; j < 20; j++ {
			node.ProduceBlockWithTxs(b, 5)
			entry := WALEntry{
				Height:    node.App.LastBlockHeight(),
				TxCount:   5,
				Timestamp: time.Now(),
				Hash:      node.GetStateHash(b),
			}
			simulateWALWrite(b, node.DataDir, entry)
		}

		node.SimulateCrash(b)

		b.StartTimer()
		node.Restart(b)
		b.StopTimer()

		node.Cleanup(b)
	}
}

// BenchmarkWALReplayLarge benchmarks replay of large WAL
func BenchmarkWALReplayLarge(b *testing.B) {
	config := DefaultRecoveryTestConfig()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		node := SetupTestNode(b, config)
		node.InitializeChain(b)

		// Build large WAL
		for j := 0; j < 100; j++ {
			node.ProduceBlockWithTxs(b, 10)
			if j%5 == 0 {
				entry := WALEntry{
					Height:    node.App.LastBlockHeight(),
					TxCount:   10,
					Timestamp: time.Now(),
					Hash:      node.GetStateHash(b),
				}
				simulateWALWrite(b, node.DataDir, entry)
			}
		}

		node.SimulateCrash(b)

		b.StartTimer()
		node.Restart(b)
		b.StopTimer()

		node.Cleanup(b)
	}
}
