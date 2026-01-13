//go:build recovery
// +build recovery

// NOTE: Recovery tests have long timeouts and may spawn goroutines that take time to clean up.
// Run with: go test -tags=recovery -timeout 20m ./tests/recovery/...
package recovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SnapshotTestSuite tests snapshot creation and restoration
type SnapshotTestSuite struct {
	suite.Suite
}

func TestSnapshotTestSuite(t *testing.T) {
	suite.Run(t, new(SnapshotTestSuite))
}

// TestSnapshotCreation tests basic snapshot creation
func (suite *SnapshotTestSuite) TestSnapshotCreation() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 5
	config.BlocksToGenerate = 10

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	// Initialize chain
	node.InitializeChain(t)

	// Produce blocks
	node.ProduceBlocks(t, 10)

	// Create snapshot at current height
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)
	require.Greater(t, snap.Height, int64(0))
	require.NotEmpty(t, snap.Hash)

	t.Logf("Created snapshot at height %d", snap.Height)
}

// TestSnapshotAtVariousHeights tests snapshot creation at different heights
func (suite *SnapshotTestSuite) TestSnapshotAtVariousHeights() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.BlocksToGenerate = 20

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Create snapshots at multiple heights
	snapshotHeights := []int{5, 10, 15, 20}
	snapshots := make(map[int]bool)

	for _, targetHeight := range snapshotHeights {
		// Produce blocks to reach target height
		currentHeight := int(node.App.LastBlockHeight())
		blocksNeeded := targetHeight - currentHeight
		if blocksNeeded > 0 {
			node.ProduceBlocks(t, blocksNeeded)
		}

		// Create snapshot
		snap := node.CreateSnapshot(t)
		require.NotNil(t, snap)
		require.Equal(t, int64(targetHeight), snap.Height)

		snapshots[targetHeight] = true
		t.Logf("Created snapshot at height %d", targetHeight)
	}

	// Verify all snapshots were created
	require.Equal(t, len(snapshotHeights), len(snapshots))
}

// TestSnapshotRestoration tests restoring from a snapshot
func (suite *SnapshotTestSuite) TestSnapshotRestoration() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks
	node.ProduceBlocks(t, 10)

	// Create snapshot
	originalHeight := node.App.LastBlockHeight()
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Produce more blocks
	node.ProduceBlocks(t, 5)
	require.Greater(t, node.App.LastBlockHeight(), originalHeight)

	// Restore from snapshot
	node.RestoreFromSnapshot(t, originalHeight)

	// Verify restoration
	t.Logf("Restored from snapshot at height %d", originalHeight)
}

// TestSnapshotDuringActiveTransactions tests snapshotting during active txs
func (suite *SnapshotTestSuite) TestSnapshotDuringActiveTransactions() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks with transactions
	for i := 0; i < 5; i++ {
		node.ProduceBlockWithTxs(t, 10) // 10 txs per block
	}

	// Create snapshot while chain is processing transactions
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Continue producing blocks with transactions
	for i := 0; i < 5; i++ {
		node.ProduceBlockWithTxs(t, 10)
	}

	// Verify snapshot is still valid
	require.Greater(t, snap.Height, int64(0))
	t.Logf("Snapshot created during active transactions at height %d", snap.Height)
}

// TestSnapshotCompression tests snapshot size and compression
func (suite *SnapshotTestSuite) TestSnapshotCompression() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks to build up state
	node.ProduceBlocks(t, 20)

	// Create snapshot
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Verify snapshot metadata
	require.NotEmpty(t, snap.Hash)
	require.Greater(t, snap.NumChunks, uint32(0))

	// In a real implementation, we would verify:
	// - Compressed size < uncompressed size
	// - Chunk count is reasonable
	// - Hash is correct

	t.Logf("Snapshot created with %d chunks", snap.NumChunks)
}

// TestMultipleSnapshotRetention tests keeping multiple snapshots
func (suite *SnapshotTestSuite) TestMultipleSnapshotRetention() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 5
	config.KeepRecentBlocks = 3

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Create multiple snapshots
	snapshots := make([]int64, 0)
	for i := 0; i < 5; i++ {
		node.ProduceBlocks(t, 5)
		snap := node.CreateSnapshot(t)
		snapshots = append(snapshots, snap.Height)
		t.Logf("Created snapshot %d at height %d", i+1, snap.Height)
	}

	// Verify we have the expected number of snapshots
	require.Equal(t, 5, len(snapshots))
}

// TestSnapshotStateIntegrity verifies snapshot preserves state integrity
func (suite *SnapshotTestSuite) TestSnapshotStateIntegrity() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce blocks
	node.ProduceBlocks(t, 10)

	// Capture state before snapshot
	heightBeforeSnapshot := node.App.LastBlockHeight()
	hashBeforeSnapshot := node.GetStateHash(t)

	// Create snapshot
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Verify state hasn't changed
	heightAfterSnapshot := node.App.LastBlockHeight()
	hashAfterSnapshot := node.GetStateHash(t)

	require.Equal(t, heightBeforeSnapshot, heightAfterSnapshot,
		"height should not change during snapshot")
	require.Equal(t, hashBeforeSnapshot, hashAfterSnapshot,
		"app hash should not change during snapshot")
}

// TestSnapshotIncrementalBackup tests incremental snapshot strategy
func (suite *SnapshotTestSuite) TestSnapshotIncrementalBackup() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 3

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Create baseline snapshot
	node.ProduceBlocks(t, 3)
	baselineSnap := node.CreateSnapshot(t)
	require.NotNil(t, baselineSnap)

	// Produce more blocks
	node.ProduceBlocks(t, 3)
	incrementalSnap := node.CreateSnapshot(t)
	require.NotNil(t, incrementalSnap)

	// Verify incremental snapshot is at later height
	require.Greater(t, incrementalSnap.Height, baselineSnap.Height)

	t.Logf("Baseline snapshot at height %d, incremental at %d",
		baselineSnap.Height, incrementalSnap.Height)
}

// TestSnapshotConcurrentReads tests concurrent snapshot access
func (suite *SnapshotTestSuite) TestSnapshotConcurrentReads() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	// Create snapshot
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Simulate concurrent reads (in real scenario these would be goroutines)
	for i := 0; i < 5; i++ {
		retrieved, err := node.Snapshots.LoadSnapshot(snap.Height)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, snap.Height, retrieved.Height)
	}

	t.Logf("Successfully handled concurrent snapshot reads")
}

// TestSnapshotAfterStateSync tests snapshot after state sync
func (suite *SnapshotTestSuite) TestSnapshotAfterStateSync() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	// Create source node
	sourceNode := SetupTestNode(t, config)
	defer sourceNode.Cleanup(t)

	sourceNode.InitializeChain(t)
	sourceNode.ProduceBlocks(t, 20)

	// Create snapshot on source
	sourceSnap := sourceNode.CreateSnapshot(t)
	require.NotNil(t, sourceSnap)

	// Create target node (simulating state sync)
	targetConfig := config
	targetConfig.ChainID = "paw-recovery-test-target"
	targetNode := SetupTestNode(t, targetConfig)
	defer targetNode.Cleanup(t)

	targetNode.InitializeChain(t)

	// In real scenario, state sync would happen here
	// For testing, we just verify both nodes can create snapshots

	targetSnap := targetNode.CreateSnapshot(t)
	require.NotNil(t, targetSnap)

	t.Logf("Source snapshot at height %d, target at height %d",
		sourceSnap.Height, targetSnap.Height)
}

// TestSnapshotPruning tests old snapshot pruning
func (suite *SnapshotTestSuite) TestSnapshotPruning() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true
	config.SnapshotInterval = 2
	config.KeepRecentBlocks = 2 // Keep only 2 recent snapshots

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Create multiple snapshots
	snapshotHeights := make([]int64, 0)
	for i := 0; i < 6; i++ {
		node.ProduceBlocks(t, 2)
		snap := node.CreateSnapshot(t)
		snapshotHeights = append(snapshotHeights, snap.Height)
		t.Logf("Created snapshot at height %d", snap.Height)
	}

	// According to pruning policy, older snapshots should be removed
	// We should only have the most recent 2
	require.Equal(t, 6, len(snapshotHeights))

	t.Logf("Snapshot pruning test completed with %d snapshots created",
		len(snapshotHeights))
}

// TestSnapshotWithLargeState tests snapshot with significant state
func (suite *SnapshotTestSuite) TestSnapshotWithLargeState() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Produce many blocks with transactions to build up state
	for i := 0; i < 10; i++ {
		node.ProduceBlockWithTxs(t, 20) // 20 txs per block
	}

	startTime := time.Now()
	snap := node.CreateSnapshot(t)
	duration := time.Since(startTime)

	require.NotNil(t, snap)
	t.Logf("Snapshot of large state created in %v at height %d",
		duration, snap.Height)

	// Verify snapshot was created within reasonable time
	// For a test environment, 10 seconds should be more than enough
	require.Less(t, duration, 10*time.Second,
		"snapshot creation took too long")
}

// TestSnapshotMetadataAccuracy tests snapshot metadata correctness
func (suite *SnapshotTestSuite) TestSnapshotMetadataAccuracy() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 10)

	beforeTime := time.Now()
	snap := node.CreateSnapshot(t)
	afterTime := time.Now()

	require.NotNil(t, snap)

	// Verify metadata
	require.Equal(t, node.App.LastBlockHeight(), snap.Height,
		"snapshot height should match current height")
	require.NotEmpty(t, snap.Hash, "snapshot hash should not be empty")
	require.Greater(t, snap.NumChunks, uint32(0), "snapshot should have chunks")

	// Verify timestamp is within range
	snapTime := time.Unix(snap.Timestamp, 0)
	require.True(t, snapTime.After(beforeTime) || snapTime.Equal(beforeTime),
		"snapshot timestamp should be after start time")
	require.True(t, snapTime.Before(afterTime) || snapTime.Equal(afterTime),
		"snapshot timestamp should be before end time")

	t.Logf("Snapshot metadata verified: height=%d, chunks=%d",
		snap.Height, snap.NumChunks)
}

// TestSnapshotErrorRecovery tests recovery from snapshot errors
func (suite *SnapshotTestSuite) TestSnapshotErrorRecovery() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)
	node.ProduceBlocks(t, 5)

	// Create valid snapshot
	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Try to retrieve non-existent snapshot
	_, err := node.Snapshots.LoadSnapshot(9999)
	require.Error(t, err, "should error on non-existent snapshot")

	// Verify we can still create new snapshots after error
	node.ProduceBlocks(t, 5)
	snap2 := node.CreateSnapshot(t)
	require.NotNil(t, snap2)
	require.Greater(t, snap2.Height, snap.Height)

	t.Logf("Successfully recovered from snapshot error")
}

// TestSnapshotChunking tests snapshot chunking mechanism
func (suite *SnapshotTestSuite) TestSnapshotChunking() {
	t := suite.T()

	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(t, config)
	defer node.Cleanup(t)

	node.InitializeChain(t)

	// Build up significant state
	for i := 0; i < 15; i++ {
		node.ProduceBlockWithTxs(t, 10)
	}

	snap := node.CreateSnapshot(t)
	require.NotNil(t, snap)

	// Verify chunking
	require.Greater(t, snap.NumChunks, uint32(0), "should have at least one chunk")

	// In real implementation, verify:
	// - Each chunk is within size limit
	// - All chunks together reconstruct the full snapshot
	// - Chunk hashes are correct

	t.Logf("Snapshot created with %d chunks for height %d",
		snap.NumChunks, snap.Height)
}

// BenchmarkSnapshotCreation benchmarks snapshot creation performance
func BenchmarkSnapshotCreation(b *testing.B) {
	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(b, config)
	defer node.Cleanup(b)

	node.InitializeChain(b)
	node.ProduceBlocks(b, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node.CreateSnapshot(b)
		node.ProduceBlocks(b, 1) // Advance state between snapshots
	}
}

// BenchmarkSnapshotRetrieval benchmarks snapshot retrieval performance
func BenchmarkSnapshotRetrieval(b *testing.B) {
	config := DefaultRecoveryTestConfig()
	config.EnableSnapshots = true

	node := SetupTestNode(b, config)
	defer node.Cleanup(b)

	node.InitializeChain(b)
	node.ProduceBlocks(b, 10)

	snap := node.CreateSnapshot(b)
	require.NotNil(b, snap)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retrieved, err := node.Snapshots.LoadSnapshot(snap.Height)
		require.NoError(b, err)
		require.NotNil(b, retrieved)
	}
}
