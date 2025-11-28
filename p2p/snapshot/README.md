# Snapshot Package

## Overview

The snapshot package provides complete snapshot creation, management, and restoration capabilities for the PAW blockchain. Snapshots enable fast node bootstrapping through state sync.

## Features

- **Snapshot Creation**: Create snapshots of blockchain state at any height
- **Chunking**: Automatic splitting of state into manageable chunks
- **Verification**: Cryptographic hash verification for all data
- **Storage**: Efficient disk-based storage and retrieval
- **Pruning**: Automatic cleanup of old snapshots
- **BFT Support**: Validator signature verification
- **Resumable**: Support for interrupted operations

## Components

### Types (`types.go`)

#### Snapshot

Represents a complete state snapshot at a specific block height.

```go
type Snapshot struct {
    Height          int64     // Block height
    Hash            []byte    // State hash
    Timestamp       int64     // Creation timestamp
    Format          uint32    // Snapshot format version
    ChainID         string    // Chain identifier
    NumChunks       uint32    // Number of chunks
    ChunkHashes     [][]byte  // Hash of each chunk
    AppHash         []byte    // Application state hash
    ValidatorHash   []byte    // Validator set hash
    ConsensusHash   []byte    // Consensus params hash
    Signature       []byte    // Snapshot signature
    ValidatorSigs   [][]byte  // Validator signatures
    SignedBy        []string  // Validator IDs
    VotingPower     int64     // Total voting power of signers
    TotalPower      int64     // Total validator voting power
}
```

#### SnapshotChunk

Represents a single chunk of snapshot data.

```go
type SnapshotChunk struct {
    Height int64  // Snapshot height
    Index  uint32 // Chunk index
    Data   []byte // Chunk data
    Hash   []byte // Chunk hash
}
```

### Manager (`manager.go`)

Handles snapshot lifecycle management.

```go
type Manager struct {
    // Configuration
    config      *ManagerConfig
    logger      log.Logger
    snapshotDir string

    // State
    snapshots      map[int64]*Snapshot
    latestSnapshot *Snapshot
}
```

## Usage

### Initialize Manager

```go
import "github.com/paw-chain/paw/p2p/snapshot"

config := &snapshot.ManagerConfig{
    SnapshotDir:        "/data/snapshots",
    SnapshotInterval:   1000,
    SnapshotKeepRecent: 10,
    ChunkSize:          16 * 1024 * 1024, // 16 MB
    PruneOldSnapshots:  true,
    MinSnapshotsToKeep: 2,
    ChainID:           "paw-mainnet",
}

manager, err := snapshot.NewManager(config, logger)
if err != nil {
    panic(err)
}
```

### Create Snapshot

```go
// Serialize your application state
stateData := serializeApplicationState()

// Get state hashes
appHash := getAppHash()
validatorHash := getValidatorSetHash()
consensusHash := getConsensusParamsHash()

// Create snapshot
snapshot, err := manager.CreateSnapshot(
    height,        // Current block height
    stateData,     // Serialized state
    appHash,       // App state hash
    validatorHash, // Validator set hash
    consensusHash, // Consensus params hash
)
if err != nil {
    panic(err)
}

fmt.Printf("Created snapshot at height %d with %d chunks\n",
    snapshot.Height, snapshot.NumChunks)
```

### Load Snapshot

```go
// Load snapshot metadata
snapshot, err := manager.LoadSnapshot(height)
if err != nil {
    panic(err)
}

// Load specific chunk
chunk, err := manager.LoadChunk(height, chunkIndex)
if err != nil {
    panic(err)
}

fmt.Printf("Loaded chunk %d: %d bytes\n",
    chunk.Index, len(chunk.Data))
```

### Restore from Snapshot

```go
// Load snapshot
snapshot, err := manager.LoadSnapshot(height)
if err != nil {
    panic(err)
}

// Restore state
stateData, err := manager.RestoreFromSnapshot(snapshot)
if err != nil {
    panic(err)
}

// Apply to application
applyState(stateData)
```

### Query Snapshots

```go
// Get latest snapshot
latest := manager.GetLatestSnapshot()
if latest != nil {
    fmt.Printf("Latest snapshot: height=%d\n", latest.Height)
}

// Get all snapshots
snapshots := manager.GetSnapshots()
for _, snap := range snapshots {
    fmt.Printf("Snapshot: height=%d, chunks=%d, size=%dMB\n",
        snap.Height, snap.NumChunks,
        calculateSize(snap)/(1024*1024))
}

// Get snapshots in range
snapshots = manager.GetSnapshotsInRange(1000, 10000)

// Check if snapshot exists
if manager.HasSnapshot(5000) {
    fmt.Println("Snapshot exists at height 5000")
}
```

### Delete Snapshot

```go
err := manager.DeleteSnapshot(height)
if err != nil {
    panic(err)
}
```

### Statistics

```go
stats := manager.GetSnapshotStats()

fmt.Printf("Total snapshots: %d\n", stats["total_snapshots"])
fmt.Printf("Total size: %d MB\n", stats["total_size_mb"])
fmt.Printf("Latest height: %d\n", stats["latest_height"])
fmt.Printf("Heights: %v\n", stats["heights"])
```

## Configuration

### ManagerConfig

```go
type ManagerConfig struct {
    // Directory for storing snapshots
    SnapshotDir string

    // Create snapshot every N blocks
    SnapshotInterval uint64

    // Keep N most recent snapshots
    SnapshotKeepRecent uint32

    // Size of each chunk in bytes
    ChunkSize uint32

    // Automatically prune old snapshots
    PruneOldSnapshots bool

    // Minimum snapshots to always keep
    MinSnapshotsToKeep uint32

    // Chain identifier
    ChainID string
}
```

### Default Configuration

```go
config := snapshot.DefaultManagerConfig("/data/snapshots")

// Defaults:
// - SnapshotInterval: 1000 blocks
// - SnapshotKeepRecent: 10 snapshots
// - ChunkSize: 16 MB
// - PruneOldSnapshots: true
// - MinSnapshotsToKeep: 2
```

## Snapshot Format

### Directory Structure

```
snapshots/
├── chunks/
│   ├── 1000-0.chunk
│   ├── 1000-1.chunk
│   ├── 1000-2.chunk
│   ├── 2000-0.chunk
│   └── 2000-1.chunk
├── snapshot-1000.json
└── snapshot-2000.json
```

### Metadata File Format

```json
{
  "height": 1000,
  "hash": "A1B2C3D4...",
  "timestamp": 1234567890,
  "format": 1,
  "chain_id": "paw-mainnet",
  "num_chunks": 5,
  "chunk_hashes": [
    "hash1...",
    "hash2...",
    "hash3...",
    "hash4...",
    "hash5..."
  ],
  "app_hash": "ABC123...",
  "validator_hash": "DEF456...",
  "consensus_hash": "GHI789...",
  "signature": "...",
  "validator_sigs": [...],
  "signed_by": ["val1", "val2", "val3"],
  "voting_power": 700,
  "total_power": 1000
}
```

## Validation

### Snapshot Validation

```go
err := snapshot.Validate()
// Checks:
// - Height > 0
// - Hash present
// - NumChunks > 0
// - ChunkHashes count matches NumChunks
// - ChainID not empty
// - AppHash present
// - BFT proof (if TotalPower > 0): VotingPower >= 2/3 TotalPower
```

### Chunk Validation

```go
err := chunk.Validate()
// Checks:
// - Height > 0
// - Data not empty
// - Hash present
// - Hash matches computed hash of data
```

### BFT Proof

```go
isTrusted := snapshot.IsTrusted()
// Returns true if:
// - TotalPower > 0
// - VotingPower / TotalPower >= 0.67 (2/3+)
```

## Helper Functions

### Hash Data

```go
hash := snapshot.HashData(data)
// Returns SHA256 hash of data
```

### Split into Chunks

```go
chunks := snapshot.SplitIntoChunks(data, chunkSize)
// Splits data into chunks of specified size
```

### Combine Chunks

```go
data := snapshot.CombineChunks(chunks)
// Combines chunk array into single data blob
```

### Serialize/Deserialize

```go
// Serialize
jsonData, err := snapshot.Serialize()

// Deserialize
snapshot, err := snapshot.DeserializeSnapshot(jsonData)
```

## Constants

```go
const (
    // Default chunk size: 16 MB
    DefaultChunkSize = 16 * 1024 * 1024

    // Minimum chunk size: 1 MB
    MinChunkSize = 1 * 1024 * 1024

    // Maximum chunk size: 64 MB
    MaxChunkSize = 64 * 1024 * 1024

    // Current snapshot format version
    SnapshotFormatV1 = 1

    // Minimum validator signatures (2/3+)
    MinValidatorSignaturesFraction = 0.67
)
```

## Performance

### Snapshot Creation

| State Size | Chunks (16MB) | Creation Time | Disk Space |
|-----------|---------------|---------------|-----------|
| 100 MB    | 7             | ~1s           | 100 MB    |
| 1 GB      | 64            | ~5s           | 1 GB      |
| 10 GB     | 640           | ~45s          | 10 GB     |
| 100 GB    | 6400          | ~7min         | 100 GB    |

### Chunk Operations

| Operation | 1 MB Chunk | 16 MB Chunk | 64 MB Chunk |
|-----------|-----------|-------------|-------------|
| Read      | ~1ms      | ~10ms       | ~40ms       |
| Write     | ~2ms      | ~20ms       | ~80ms       |
| Hash      | ~1ms      | ~10ms       | ~40ms       |
| Verify    | ~2ms      | ~20ms       | ~80ms       |

## Best Practices

### 1. Chunk Size Selection

- **Small state (<1 GB)**: 4-8 MB chunks
- **Medium state (1-10 GB)**: 16 MB chunks (default)
- **Large state (>10 GB)**: 32-64 MB chunks

### 2. Snapshot Interval

- **High activity**: Every 500-1000 blocks
- **Medium activity**: Every 1000-5000 blocks
- **Low activity**: Every 5000-10000 blocks

### 3. Retention Policy

- **Production**: Keep 5-10 recent snapshots
- **Archive nodes**: Keep 20-50 snapshots
- **Development**: Keep 2-3 snapshots

### 4. Storage

- **Use SSD**: For faster chunk I/O
- **Separate disk**: Dedicated disk for snapshots
- **Regular cleanup**: Enable pruning
- **Monitor space**: Alert on low disk space

## Troubleshooting

### Snapshot Creation Fails

**Error**: "failed to create snapshot directory"
- Check disk space
- Verify directory permissions
- Check parent directory exists

**Error**: "failed to serialize snapshot"
- Check state data is valid
- Verify hashes are computed
- Check memory availability

### Chunk Load Fails

**Error**: "failed to read chunk"
- Verify chunk file exists
- Check file permissions
- Verify disk is not corrupted

**Error**: "chunk hash mismatch"
- Chunk file may be corrupted
- Re-create snapshot
- Check disk integrity

### Restoration Fails

**Error**: "chunk hash mismatch"
- Some chunks are corrupted
- Re-download from peers
- Verify snapshot metadata

**Error**: "restored state hash mismatch"
- Snapshot metadata corrupted
- Re-download snapshot
- Verify source peer

## Examples

### Complete Snapshot Workflow

```go
// 1. Initialize manager
manager, _ := snapshot.NewManager(config, logger)

// 2. Create snapshot every 1000 blocks
if height % 1000 == 0 {
    stateData := getState()
    snapshot, _ := manager.CreateSnapshot(
        height, stateData, appHash, valHash, consHash)

    // Sign snapshot (validator nodes)
    signature := signSnapshot(snapshot)
    snapshot.Signature = signature
}

// 3. Serve chunks to peers
func serveChunk(height int64, chunkIndex uint32) []byte {
    chunk, _ := manager.LoadChunk(height, chunkIndex)
    return chunk.Data
}

// 4. Restore on new node
func bootstrap(targetHeight int64) {
    snapshot, _ := downloadSnapshot(targetHeight)
    stateData, _ := manager.RestoreFromSnapshot(snapshot)
    applyState(stateData)
}
```

## Testing

```bash
# Run snapshot tests
go test ./p2p/snapshot/... -v

# Test with race detector
go test ./p2p/snapshot/... -race

# Benchmark
go test ./p2p/snapshot/... -bench=. -benchmem
```

## Thread Safety

All public methods of `Manager` are thread-safe through the use of mutexes:

- `CreateSnapshot()`: Thread-safe
- `LoadSnapshot()`: Thread-safe (concurrent reads)
- `LoadChunk()`: Thread-safe (concurrent reads)
- `DeleteSnapshot()`: Thread-safe
- `GetSnapshots()`: Thread-safe (returns copy)

## Migration

### Upgrading Snapshot Format

When upgrading snapshot format version:

1. Create migration script
2. Convert old snapshots to new format
3. Update `Format` field
4. Re-compute hashes
5. Verify integrity

## License

MIT License - See LICENSE file for details
