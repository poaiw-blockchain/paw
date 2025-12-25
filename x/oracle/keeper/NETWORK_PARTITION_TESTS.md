# Oracle Network Partition Simulation Tests

## Overview
Comprehensive test suite verifying oracle module behavior during network partitions.

## Test Coverage

### 1. Majority Partition Consensus
- **Test**: `TestNetworkPartition_MajorityCanReachConsensus`
- **Scenario**: 7/10 validators (70%) in majority partition
- **Verifies**: Majority can aggregate prices and reach consensus

### 2. Minority Partition Failure
- **Test**: `TestNetworkPartition_MinorityCannotReachConsensus`
- **Scenario**: 3/10 validators (30%) in minority partition
- **Verifies**: Minority fails aggregation with insufficient voting power error

### 3. Graceful Failure Handling
- **Test**: `TestNetworkPartition_GracefulFailure`
- **Scenarios**:
  - 60% voting power (below 67% threshold)
  - Zero submissions (complete isolation)
  - Single validator (extreme partition)
- **Verifies**: No panics, proper error messages, descriptive failures

### 4. State Consistency After Healing
- **Test**: `TestNetworkPartition_StateConsistency`
- **Scenario**: 11/15 validators partitioned, then network heals
- **Verifies**: State remains consistent, prices update correctly after healing

### 5. Byzantine Tolerance During Partition
- **Test**: `TestNetworkPartition_ByzantineToleranceDuringPartition`
- **Scenario**: Geographic diversity requirements with regional partition
- **Verifies**: Byzantine checks still enforced on bonded validator set

### 6. Exact Threshold Edge Cases
- **Test**: `TestNetworkPartition_ExactThreshold`
- **Scenarios**:
  - Exactly 67/100 validators (passes)
  - Exactly 66/100 validators (fails)
- **Verifies**: Precise threshold enforcement at boundary

### 7. State Corruption Prevention
- **Test**: `TestNetworkPartition_NoStateCorruption`
- **Scenario**: Failed aggregation doesn't corrupt existing price state
- **Verifies**: Old prices remain accessible, block heights unchanged

### 8. Outlier Filtering During Partition
- **Test**: `TestNetworkPartition_OutlierFilteringDuringPartition`
- **Scenario**: 5 honest + 2 Byzantine validators in 70% partition
- **Verifies**: Outlier filtering reduces voting power, triggering consensus failure

### 9. Three-Way Network Split
- **Test**: `TestNetworkPartition_ThreeWaySplit`
- **Scenario**: 3 partitions with 33% each
- **Verifies**: No partition can reach 67% consensus threshold

### 10. Error Propagation
- **Test**: `TestNetworkPartition_PropagatesErrors`
- **Verifies**: Descriptive error messages for all partition scenarios

## Key Security Properties Verified

1. **Vote Threshold Enforcement**: 67% voting power required for consensus
2. **No Panic Conditions**: All failures are graceful with proper errors
3. **State Consistency**: Failed aggregations don't corrupt price data
4. **Byzantine Tolerance**: Geographic and voting power checks maintained
5. **Outlier Filtering**: Security checks work correctly during partitions

## Test Execution
```bash
go test -v ./x/oracle/keeper -run TestNetworkPartition
```

All 10 tests pass successfully (0.133s total runtime).
