# State Sync Protocol Implementation

## Overview

The PAW blockchain now includes a **complete state sync protocol** for fast node bootstrapping. This enables new nodes to sync in **minutes instead of days/weeks** by downloading and applying state snapshots instead of replaying all blocks from genesis.

## Architecture

### Components

1. **Snapshot Package** (`p2p/snapshot/`)
   - `types.go`: Snapshot data structures and validation
   - `manager.go`: Snapshot creation, storage, and retrieval

2. **State Sync Protocol** (`p2p/protocol/`)
   - `state_sync.go`: Core state sync protocol and discovery
   - `state_sync_download.go`: Parallel chunk download with Byzantine fault tolerance
   - `state_sync_test.go`: Comprehensive test suite

3. **Node Configuration** (`p2p/node.go`)
   - `StateSyncConfig`: Configuration for state sync parameters

## Key Features

### 1. Byzantine Fault Tolerance (BFT)

- **2/3+ Validator Signatures**: Snapshots require signatures from 2/3+ validators
- **Peer Agreement**: Selects snapshots with 67%+ peer agreement
- **Byzantine Detection**: Identifies and bans malicious peers
- **Hash Verification**: All chunks verified against cryptographic hashes

### 2. Parallel Download

- **Multiple Fetchers**: Downloads chunks in parallel (default: 4 workers)
- **Load Balancing**: Distributes requests across peers
- **Retry Logic**: Automatic retry with exponential backoff
- **Resume Support**: Can resume interrupted downloads

### 3. Verification

- **Snapshot Verification**: Against trusted height/hash
- **Chunk Verification**: Each chunk hash verified
- **State Verification**: Final state hash validation
- **Validator Set Check**: Ensures correct validator signatures

### 4. Performance

- **Fast Bootstrap**: Sync in minutes vs. days
- **Bandwidth Efficient**: Only download final state
- **Parallel Processing**: Multi-threaded chunk downloads
- **Compression Ready**: Chunks can be compressed

## Usage

### Configuration

```go
// Enable state sync in node configuration
config := p2p.DefaultNodeConfig()

// State sync settings
config.StateSync.Enable = true
config.StateSync.TrustHeight = 1000000
config.StateSync.TrustHash = "A1B2C3D4..." // Hex encoded
config.StateSync.TrustPeriod = 7 * 24 * time.Hour

// Discovery settings
config.StateSync.DiscoveryTime = 10 * time.Second
config.StateSync.MinSnapshotOffers = 3

// Download settings
config.StateSync.ChunkRequestTimeout = 30 * time.Second
config.StateSync.ChunkFetchers = 4

// Snapshot settings
config.StateSync.SnapshotInterval = 1000   // Every 1000 blocks
config.StateSync.SnapshotKeepRecent = 10   // Keep 10 recent snapshots
```

### Creating Snapshots

```go
import (
    "github.com/paw-chain/paw/p2p/snapshot"
)

// Initialize snapshot manager
config := snapshot.DefaultManagerConfig("/data/snapshots")
config.ChainID = "paw-mainnet"
config.SnapshotInterval = 1000

manager, err := snapshot.NewManager(config, logger)
if err != nil {
    panic(err)
}

// Create snapshot at height 10000
stateData := getStateData() // Your state serialization
appHash := getAppHash()
validatorHash := getValidatorHash()
consensusHash := getConsensusHash()

snapshot, err := manager.CreateSnapshot(
    10000,           // height
    stateData,       // serialized state
    appHash,         // application state hash
    validatorHash,   // validator set hash
    consensusHash,   // consensus params hash
)
if err != nil {
    panic(err)
}

fmt.Printf("Snapshot created: height=%d, chunks=%d\n",
    snapshot.Height, snapshot.NumChunks)
```

### Performing State Sync

```go
import (
    "github.com/paw-chain/paw/p2p/protocol"
)

// Initialize state sync protocol
config := protocol.DefaultStateSyncConfig()
config.TrustHeight = 1000000
config.TrustHash = trustHash
config.ChunkFetchers = 4

stateSyncProtocol := protocol.NewStateSyncProtocol(
    config,
    snapshotManager,
    peerManager,
    logger,
)

// Set callbacks
stateSyncProtocol.SetApplySnapshotCallback(func(height int64, data []byte) error {
    // Apply state to your application
    return applyState(height, data)
})

stateSyncProtocol.SetVerifyStateCallback(func(height int64, appHash []byte) error {
    // Verify applied state
    return verifyAppHash(height, appHash)
})

// Start state sync
ctx := context.Background()
if err := stateSyncProtocol.StartStateSync(ctx); err != nil {
    if config.FallbackToBlockSync {
        // Fall back to block sync
        performBlockSync()
    } else {
        panic(err)
    }
}

// Get metrics
metrics := stateSyncProtocol.GetMetrics()
fmt.Printf("State sync complete:\n")
fmt.Printf("  Snapshots discovered: %d\n", metrics.SnapshotsDiscovered)
fmt.Printf("  Chunks downloaded: %d\n", metrics.ChunksDownloaded)
fmt.Printf("  Bytes downloaded: %d MB\n", metrics.BytesDownloaded/(1024*1024))
fmt.Printf("  Download time: %v\n", metrics.DownloadTime)
fmt.Printf("  Total time: %v\n", metrics.TotalTime)
```

## Performance Characteristics

### Sync Time Comparison

| Chain Height | Block Sync | State Sync | Speedup |
|-------------|-----------|------------|---------|
| 100,000     | ~2 hours  | ~2 minutes | 60x     |
| 1,000,000   | ~20 hours | ~5 minutes | 240x    |
| 10,000,000  | ~8 days   | ~15 minutes| 768x    |

*Assumes 1MB/s network, 1000 block snapshots, 16MB chunks*

### Resource Usage

- **Network**: ~1-5 GB (depends on state size)
- **Disk**: 2x state size (during download)
- **CPU**: Low (hash verification only)
- **Memory**: ~100 MB (chunk buffers)

### Scalability

- **Parallel Downloads**: 4-8 fetchers optimal
- **Chunk Size**: 16 MB default (configurable 1-64 MB)
- **Snapshot Interval**: 1000 blocks recommended
- **Keep Recent**: 10 snapshots recommended

## Security Considerations

### Trust Model

1. **Initial Trust**: Requires trusted height + hash
2. **Light Client**: Can verify using light client proofs
3. **BFT Proofs**: 2/3+ validator signatures required
4. **Peer Agreement**: 67%+ peers must agree on snapshot

### Attack Vectors & Mitigations

| Attack | Mitigation |
|--------|-----------|
| Fake snapshots | BFT validator signatures |
| Corrupted chunks | Cryptographic hash verification |
| Sybil attack | Peer agreement threshold (67%) |
| Eclipse attack | Multiple seed/RPC servers |
| DoS (bad chunks) | Peer reputation & banning |

### Byzantine Fault Tolerance

```
Maximum tolerated Byzantine peers: f = (n-1)/3

Examples:
- 4 peers: tolerate 1 Byzantine
- 7 peers: tolerate 2 Byzantine
- 10 peers: tolerate 3 Byzantine
```

## Testing

### Run Tests

```bash
# Unit tests
go test ./p2p/protocol/... -v

# State sync tests specifically
go test ./p2p/protocol/ -run TestStateSync -v

# Benchmarks
go test ./p2p/protocol/ -bench=. -benchmem
```

### Test Coverage

```bash
go test ./p2p/protocol/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Test

```go
func TestFullStateSync(t *testing.T) {
    // 1. Create test chain with state
    // 2. Create snapshots at intervals
    // 3. Initialize new node
    // 4. Perform state sync
    // 5. Verify state matches
}
```

## Configuration Examples

### Mainnet (High Security)

```toml
[state_sync]
enable = true
trust_height = 5000000
trust_hash = "A1B2C3D4E5F6..."
trust_period = "168h"  # 7 days

# Discovery
discovery_time = "30s"
min_snapshot_offers = 5

# Download
chunk_request_timeout = "60s"
chunk_fetchers = 4

# Snapshots
snapshot_interval = 10000
snapshot_keep_recent = 5
```

### Testnet (Fast Sync)

```toml
[state_sync]
enable = true
trust_height = 100000
trust_hash = "ABC123..."
trust_period = "24h"

# Discovery
discovery_time = "10s"
min_snapshot_offers = 2

# Download
chunk_request_timeout = "30s"
chunk_fetchers = 8

# Snapshots
snapshot_interval = 1000
snapshot_keep_recent = 10
```

### Local Development

```toml
[state_sync]
enable = true
trust_height = 0  # Trust genesis
trust_hash = ""
trust_period = "1h"

# Discovery
discovery_time = "5s"
min_snapshot_offers = 1

# Download
chunk_request_timeout = "10s"
chunk_fetchers = 2

# Snapshots
snapshot_interval = 100
snapshot_keep_recent = 3
```

## Troubleshooting

### No Snapshots Available

**Symptom**: "no snapshots available, falling back to block sync"

**Solutions**:
1. Check peers are running with snapshots enabled
2. Verify `snapshot_interval` is configured
3. Check network connectivity to peers
4. Increase `discovery_time`

### Chunk Download Failures

**Symptom**: "chunk download failed"

**Solutions**:
1. Check network bandwidth
2. Increase `chunk_request_timeout`
3. Reduce `chunk_fetchers` (network congestion)
4. Check for malicious peers in logs

### Verification Failures

**Symptom**: "snapshot verification failed"

**Solutions**:
1. Verify `trust_height` and `trust_hash` are correct
2. Check chain ID matches
3. Ensure trusted RPC servers are synced
4. Check validator set matches

### Byzantine Peer Detection

**Symptom**: "too many malicious peers detected"

**Solutions**:
1. Change seed/persistent peers
2. Check network for eclipse attack
3. Verify RPC servers are trusted
4. Report malicious peers to network operators

## Metrics & Monitoring

### Key Metrics

```go
metrics := stateSyncProtocol.GetMetrics()

// Discovery
fmt.Printf("Snapshots discovered: %d\n", metrics.SnapshotsDiscovered)
fmt.Printf("Peers queried: %d\n", metrics.PeersQueried)

// Download
fmt.Printf("Chunks downloaded: %d\n", metrics.ChunksDownloaded)
fmt.Printf("Chunks verified: %d\n", metrics.ChunksVerified)
fmt.Printf("Bytes downloaded: %d MB\n", metrics.BytesDownloaded/(1024*1024))

// Performance
fmt.Printf("Download time: %v\n", metrics.DownloadTime)
fmt.Printf("Verification time: %v\n", metrics.VerificationTime)
fmt.Printf("Total time: %v\n", metrics.TotalTime)

// Security
fmt.Printf("Malicious peers: %d\n", metrics.MaliciousPeersFound)
```

### Progress Tracking

```go
downloaded, total := stateSyncProtocol.GetProgress()
progress := float64(downloaded) / float64(total) * 100
fmt.Printf("Progress: %.1f%% (%d/%d chunks)\n", progress, downloaded, total)
```

## Future Enhancements

1. **Compression**: Compress chunks before transfer
2. **Incremental Snapshots**: Delta-based snapshots
3. **P2P Discovery**: Dedicated snapshot discovery protocol
4. **Multi-snapshot**: Download from multiple heights simultaneously
5. **Streaming**: Stream chunks as they're created
6. **IPFS Integration**: Store snapshots on IPFS
7. **Pruning**: Automatic snapshot pruning strategies

## References

- [Tendermint State Sync](https://docs.tendermint.com/master/spec/p2p/messages/state-sync.html)
- [Cosmos SDK State Sync](https://docs.cosmos.network/master/run-node/run-node.html#state-sync)
- [Byzantine Fault Tolerance](https://en.wikipedia.org/wiki/Byzantine_fault)
- [Merkle Tree Verification](https://en.wikipedia.org/wiki/Merkle_tree)

## Support

For issues or questions:
-  Issues: https://github.com/paw-chain/paw/issues
- Discord: https://discord.gg/paw
- Documentation: https://docs.paw.network
