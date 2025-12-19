# Recovery Test Suite

This directory contains comprehensive tests for blockchain state recovery, crash recovery, and Write-Ahead Log (WAL) replay functionality. These tests are critical for ensuring production reliability and data integrity.

## Overview

The recovery test suite validates that the PAW blockchain can:

1. **Survive crashes** at any point during operation
2. **Recover state** correctly after unexpected shutdown
3. **Replay WAL entries** to maintain consistency
4. **Create and restore snapshots** for fast recovery
5. **Preserve transaction ordering** through crashes
6. **Handle corrupted state** gracefully

## Test Files

### `helpers.go`
Common utilities and test infrastructure:
- `TestNode`: Full-featured test blockchain node
- `RecoveryTestConfig`: Configuration for recovery scenarios
- `SetupTestNode()`: Initialize test node with recovery features
- Helper functions for crash simulation, WAL, and snapshots

### `snapshot_test.go`
State snapshot creation and restoration tests:

**Basic Tests:**
- Snapshot creation at various heights
- Snapshot restoration and verification
- Multiple snapshot retention
- Snapshot metadata accuracy

**Advanced Tests:**
- Snapshots during active transactions
- Snapshot compression and chunking
- Incremental backup strategies
- Concurrent snapshot access
- Snapshot pruning policies
- Large state snapshots

**Performance Tests:**
- Benchmark snapshot creation
- Benchmark snapshot retrieval

### `crash_recovery_test.go`
Node crash and recovery scenario tests:

**Basic Crash Scenarios:**
- Simple crash and restart
- Crash during block processing
- Crash during state commit
- Crash during consensus rounds

**Complex Scenarios:**
- Multiple sequential crashes
- Crash with active transactions
- Quick successive crash/restart cycles
- Crash during state synchronization

**Data Integrity Tests:**
- Verify no data loss after crash
- State consistency across restarts
- Memory state consistency
- Recovery from potential corruption

**Performance Tests:**
- Benchmark crash recovery time
- Benchmark recovery with significant state

### `wal_replay_test.go`
Write-Ahead Log replay functionality tests:

**Basic WAL Tests:**
- Basic WAL replay after restart
- Transaction ordering preservation
- Partial block handling
- Consistency validation

**Advanced WAL Tests:**
- Large WAL file replay
- WAL with gaps
- Corrupted WAL entry handling
- WAL replay idempotency
- Different block sizes

**Integration Tests:**
- WAL replay with snapshots
- WAL replay after state sync
- Memory efficiency during replay

**Performance Tests:**
- Benchmark WAL replay speed
- Benchmark large WAL replay

## Running Tests

### Run All Recovery Tests
```bash
cd /home/hudson/blockchain-projects/paw
go test ./tests/recovery/... -v
```

### Run Specific Test Suite
```bash
# Snapshot tests only
go test ./tests/recovery -v -run TestSnapshotTestSuite

# Crash recovery tests only
go test ./tests/recovery -v -run TestCrashRecoveryTestSuite

# WAL replay tests only
go test ./tests/recovery -v -run TestWALReplayTestSuite
```

### Run Specific Test
```bash
# Test basic crash recovery
go test ./tests/recovery -v -run TestBasicCrashRecovery

# Test snapshot creation
go test ./tests/recovery -v -run TestSnapshotCreation

# Test WAL replay
go test ./tests/recovery -v -run TestBasicWALReplay
```

### Run with Race Detection
```bash
go test ./tests/recovery/... -v -race
```

### Run Benchmarks
```bash
# All benchmarks
go test ./tests/recovery -bench=. -benchmem

# Specific benchmarks
go test ./tests/recovery -bench=BenchmarkSnapshotCreation
go test ./tests/recovery -bench=BenchmarkCrashRecovery
go test ./tests/recovery -bench=BenchmarkWALReplay
```

### Run with Coverage
```bash
go test ./tests/recovery/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Scenarios

### Crash Scenarios Tested

1. **Mid-Block Crash**
   - Node crashes during block execution
   - Before state is committed
   - State should rollback to previous block

2. **Mid-Commit Crash**
   - Node crashes during state commit
   - Partially written state
   - Should recover to last complete commit

3. **Consensus Crash**
   - Node crashes during consensus voting
   - May have sent/received partial votes
   - Should rejoin consensus cleanly

4. **Multiple Sequential Crashes**
   - Repeated crash/recovery cycles
   - Tests recovery resilience
   - Verifies no state drift

5. **Crash with Active Transactions**
   - Crash while processing many transactions
   - Verify no transaction loss or duplication
   - Transaction ordering preserved

### Recovery Mechanisms Tested

1. **WAL Replay**
   - Write-Ahead Log ensures durability
   - Uncommitted operations can be replayed
   - Transaction ordering maintained

2. **Snapshot Restoration**
   - Fast recovery from snapshots
   - State at specific heights
   - Fallback when WAL is corrupted

3. **State Consistency Checks**
   - App hash verification
   - Height verification
   - Module state validation

## Configuration

### Recovery Test Configuration

```go
type RecoveryTestConfig struct {
    ChainID            string  // Test chain ID
    InitialHeight      int64   // Starting height
    BlocksToGenerate   int     // Blocks to produce
    SnapshotInterval   uint64  // Blocks between snapshots
    KeepRecentBlocks   uint32  // Pruning setting
    EnableSnapshots    bool    // Enable snapshot manager
    SimulateCrashAt    int64   // Height to crash (0 = no crash)
    CrashDuringCommit  bool    // Crash during commit
    CrashDuringConsensus bool  // Crash during consensus
}
```

### Default Configuration

```go
config := DefaultRecoveryTestConfig()
// ChainID: "paw-recovery-test"
// InitialHeight: 1
// BlocksToGenerate: 10
// SnapshotInterval: 5
// KeepRecentBlocks: 3
// EnableSnapshots: true
// SimulateCrashAt: 0
```

## Implementation Details

### Test Node Setup

Each test creates an isolated test node:
- Temporary data directory (cleaned up automatically)
- In-memory or LevelDB database
- Full snapshot manager
- WAL simulation
- Crash simulation capabilities

### Crash Simulation

Crashes are simulated by:
1. Closing database without proper shutdown
2. Not flushing pending writes
3. Leaving WAL in inconsistent state

This mimics real crash scenarios:
- Power loss
- Process kill
- OS crash
- Hardware failure

### State Verification

After recovery, tests verify:
- **Height consistency**: Last block height matches
- **Hash consistency**: App hash matches
- **State accessibility**: All modules queryable
- **Continuation**: Node can produce new blocks

## Expected Results

### All Tests Should Pass

These tests validate critical recovery functionality. All tests must pass before production deployment.

### Performance Expectations

- Snapshot creation: < 5 seconds for 1000 blocks
- Crash recovery: < 10 seconds
- WAL replay: < 1 second per 100 blocks
- State verification: < 1 second

### Known Limitations

1. **WAL Simulation**: Uses simplified WAL (real CometBFT WAL is more complex)
2. **Consensus**: Single-node tests (multi-node in separate suite)
3. **Network**: No network partition scenarios (in chaos tests)

## Integration with Other Test Suites

### Related Test Suites

- **tests/chaos/**: Network partition and Byzantine tests
- **tests/statemachine/**: State machine consistency tests
- **tests/e2e/**: End-to-end multi-node tests
- **tests/verification/**: Formal verification tests

### Coverage Goals

The recovery test suite aims for:
- **Line coverage**: > 90% of recovery code
- **Branch coverage**: > 85% of error paths
- **Scenario coverage**: All critical crash scenarios

## Troubleshooting

### Tests Failing

If tests fail, check:

1. **Database locks**: Ensure no other process is using test databases
2. **Disk space**: Ensure sufficient space for test data
3. **Permissions**: Test directories must be writable
4. **Timing**: Some tests have timeouts; slow systems may fail

### Common Issues

**"Database locked"**
- Another test is still running
- Force cleanup: `rm -rf /tmp/go-build*`

**"Timeout during recovery"**
- System may be slow
- Increase timeout values in test config

**"Snapshot not found"**
- Snapshot manager not initialized
- Check `EnableSnapshots: true` in config

**"WAL corruption"**
- Expected in corruption tests
- Should be handled gracefully

## Development Guidelines

### Adding New Tests

1. **Follow naming conventions**: `Test<Component><Scenario>`
2. **Use test suites**: Group related tests
3. **Clean up resources**: Defer `Cleanup()`
4. **Log progress**: Use `t.Logf()` for debugging
5. **Verify thoroughly**: Check height, hash, and state

### Test Structure

```go
func (suite *TestSuite) TestScenario() {
    t := suite.T()

    // 1. Setup
    config := DefaultRecoveryTestConfig()
    node := SetupTestNode(t, config)
    defer node.Cleanup(t)

    // 2. Initialize
    node.InitializeChain(t)

    // 3. Execute scenario
    node.ProduceBlocks(t, 10)

    // 4. Crash
    node.SimulateCrash(t)

    // 5. Recover
    node.Restart(t)

    // 6. Verify
    require.Equal(t, expectedHeight, actualHeight)
}
```

### Performance Testing

Add benchmarks for new recovery mechanisms:

```go
func BenchmarkNewRecovery(b *testing.B) {
    for i := 0; i < b.N; i++ {
        b.StopTimer()
        // Setup
        b.StartTimer()
        // Measure
        b.StopTimer()
        // Cleanup
    }
}
```

## Production Readiness

These tests are **critical for production deployment**. Before releasing:

1. ✅ All tests must pass
2. ✅ Benchmarks must meet performance targets
3. ✅ Coverage must exceed 90%
4. ✅ No race conditions detected
5. ✅ Tests pass on target deployment environment

## References

- [CometBFT WAL Documentation](https://docs.cometbft.com/v0.38/core/consensus-algorithm)
- [Cosmos SDK State Management](https://docs.cosmos.network/main/core/store)
- [ABCI Specification](https://docs.cometbft.com/v0.38/spec/abci/)

## Maintenance

**Test Owner**: Blockchain Core Team
**Last Updated**: 2025-12-14
**Review Frequency**: Every major release
**CI Integration**: Runs on every PR and nightly

## Contact

For questions or issues with recovery tests:
- Create GitHub issue with label `test-recovery`
- Discuss in #blockchain-testing channel
