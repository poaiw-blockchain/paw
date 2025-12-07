# PAW Blockchain - Performance Tuning Guide

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** DevOps Engineers, SRE, Validator Operators, Performance Engineers

---

## Table of Contents

1. [Overview](#overview)
2. [CometBFT Configuration Tuning](#cometbft-configuration-tuning)
3. [Cosmos SDK Application Tuning](#cosmos-sdk-application-tuning)
4. [Database Optimization](#database-optimization)
5. [P2P Network Optimization](#p2p-network-optimization)
6. [Operating System Tuning](#operating-system-tuning)
7. [Storage and I/O Optimization](#storage-and-io-optimization)
8. [Memory Management](#memory-management)
9. [API and Query Performance](#api-and-query-performance)
10. [Monitoring Performance Metrics](#monitoring-performance-metrics)
11. [Benchmarking and Testing](#benchmarking-and-testing)
12. [Common Performance Issues](#common-performance-issues)

---

## Overview

### Performance Goals

| Metric | Target | Notes |
|--------|--------|-------|
| **Block Time** | 3 seconds average | CometBFT consensus parameter |
| **TPS (Throughput)** | 300-500 TPS sustained | Depends on transaction complexity |
| **Block Processing** | <1 second p99 | From block receipt to commit |
| **API Latency** | <100ms p50, <1s p99 | REST/gRPC queries |
| **Sync Speed** | 500+ blocks/second | When catching up |
| **Memory Usage** | <80% of available RAM | Avoid swapping |
| **CPU Usage** | <70% average | Headroom for spikes |
| **Disk I/O Wait** | <10% | Minimize storage bottlenecks |

### Performance Tuning Philosophy

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Measure First (Baseline)                                │
│    - Identify actual bottlenecks with profiling            │
│    - Don't optimize blindly                                │
├─────────────────────────────────────────────────────────────┤
│ 2. Tune Configuration (Low-Risk)                           │
│    - Adjust config parameters                              │
│    - No code changes required                              │
├─────────────────────────────────────────────────────────────┤
│ 3. Optimize System Resources (Medium-Risk)                 │
│    - OS-level tuning                                       │
│    - Requires testing and validation                       │
├─────────────────────────────────────────────────────────────┤
│ 4. Scale Horizontally (High-Impact)                        │
│    - Add more nodes (API servers, sentries)                │
│    - Most effective for query load                         │
├─────────────────────────────────────────────────────────────┤
│ 5. Measure Again (Validate)                                │
│    - Verify improvements                                   │
│    - Watch for regressions                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## CometBFT Configuration Tuning

### Consensus Parameters

**File:** `config/config.toml`

#### Block Size and Time

```toml
[consensus]
# How long we wait for a proposal block before prevoting nil
timeout_propose = "3s"
# How much timeout_propose increases with each round
timeout_propose_delta = "500ms"

# How long we wait after receiving +2/3 prevotes for "anything"
timeout_prevote = "1s"
timeout_prevote_delta = "500ms"

# How long we wait after receiving +2/3 precommits for "anything"
timeout_precommit = "1s"
timeout_precommit_delta = "500ms"

# How long we wait after committing a block, before starting on the new
# height (this gives us a chance to receive some more precommits)
timeout_commit = "3s"

# Make progress as soon as we have all the precommits (as if TimeoutCommit = 0)
skip_timeout_commit = false  # Set to true for faster blocks in private testnets
```

**Tuning Recommendations:**

**For Low-Latency Networks (<50ms validator latency):**
```toml
timeout_propose = "2s"         # Faster block proposals
timeout_prevote = "800ms"      # Quicker consensus rounds
timeout_precommit = "800ms"
timeout_commit = "2s"          # Reduce to 2s for 2-second block time

# Result: ~2s block time with well-connected validators
```

**For High-Latency Networks (>100ms validator latency):**
```toml
timeout_propose = "4s"         # Give slow validators more time
timeout_prevote = "1500ms"     # Prevent timeout-induced nil prevotes
timeout_precommit = "1500ms"
timeout_commit = "3s"

# Result: More reliable consensus, fewer missed blocks
```

**For Private Testnet (Development):**
```toml
skip_timeout_commit = true     # Don't wait for stragglers
timeout_propose = "500ms"
timeout_prevote = "200ms"
timeout_precommit = "200ms"
timeout_commit = "500ms"

# Result: Sub-second block time for rapid testing
```

#### Mempool Configuration

```toml
[mempool]
# Maximum number of transactions in the mempool
size = 5000

# Limit the total size of all txs in the mempool (MB)
max_txs_bytes = 1073741824  # 1 GB

# Size of the cache (used to filter transactions we saw earlier)
cache_size = 10000

# Do not remove invalid transactions from the cache
keep-invalid-txs-in-cache = false

# Maximum size of a single transaction (bytes)
max_tx_bytes = 1048576  # 1 MB

# Maximum gas per block (consensus parameter, not config)
# Set in genesis.json or via governance
# max_gas = "-1"  # Unlimited
```

**Tuning Recommendations:**

**High-Throughput Configuration:**
```toml
size = 10000               # Double mempool size for high TPS
max_txs_bytes = 2147483648  # 2 GB mempool
cache_size = 20000         # Larger cache to filter duplicates
max_tx_bytes = 2097152     # 2 MB max transaction size

# Benefit: Handle traffic spikes, prevent mempool overflow
# Cost: Higher memory usage (~2-3 GB)
```

**Low-Memory Configuration:**
```toml
size = 2000                # Smaller mempool
max_txs_bytes = 524288000   # 500 MB
cache_size = 5000
max_tx_bytes = 524288      # 512 KB max tx

# Benefit: Lower memory footprint
# Cost: May reject transactions during spikes
```

### RPC Configuration

```toml
[rpc]
# TCP or UNIX socket address for the RPC server to listen on
laddr = "tcp://127.0.0.1:26657"

# Maximum number of simultaneous connections
max_open_connections = 900

# Maximum number of unique clientIPs per IPC connection
max_subscription_clients = 100

# Maximum number of unique queries per IPC connection
max_subscriptions_per_client = 5

# Timeout for broadcasting tx commit
timeout_broadcast_tx_commit = "10s"

# Maximum size of request body, in bytes
max_body_bytes = 1000000  # 1 MB

# Maximum size of request header, in bytes
max_header_bytes = 1048576  # 1 MB
```

**Tuning Recommendations:**

**Public API Node:**
```toml
laddr = "tcp://0.0.0.0:26657"   # Listen on all interfaces (behind firewall)
max_open_connections = 2000     # Handle many concurrent clients
max_subscription_clients = 500  # More WebSocket subscribers
max_subscriptions_per_client = 10
max_body_bytes = 5000000        # 5 MB (larger queries)

# Nginx reverse proxy recommended for rate limiting
```

**Validator Node (Private RPC):**
```toml
laddr = "tcp://127.0.0.1:26657"  # Localhost only
max_open_connections = 100       # Minimal connections (internal use)
max_subscription_clients = 10
max_subscriptions_per_client = 2

# Validators should NOT expose RPC publicly
```

### Instrumentation (Metrics)

```toml
[instrumentation]
# When true, Prometheus metrics are served under /metrics on
# PrometheusListenAddr
prometheus = true

# Address to listen for Prometheus collector(s) connections
prometheus_listen_addr = ":26660"

# Maximum number of simultaneous connections
max_open_connections = 3

# Instrumentation namespace
namespace = "tendermint"
```

**Always enable for production monitoring.**

---

## Cosmos SDK Application Tuning

### App Configuration

**File:** `config/app.toml`

#### State Sync Configuration

```toml
[state-sync]
# State sync snapshots allow other nodes to rapidly join the network
# without replaying historical blocks
snapshot-interval = 1000  # Create snapshot every 1000 blocks (~50 min)
snapshot-keep-recent = 2  # Keep 2 most recent snapshots

# Pruning options (CRITICAL for storage performance)
[pruning]
# default, nothing, everything, custom
pruning = "custom"
pruning-keep-recent = "100"    # Keep last 100 blocks
pruning-keep-every = "1000"    # Keep every 1000th block (for snapshots)
pruning-interval = "10"        # Prune every 10 blocks
```

**Pruning Strategy Comparison:**

| Strategy | Storage (Year 1) | Query Capability | Use Case |
|----------|------------------|------------------|----------|
| **nothing** | 2 TB+ | Full history | Archive nodes, explorers |
| **default** | 500 GB | Last ~6 weeks | General full nodes |
| **custom (100/1000)** | 200 GB | Last ~5 minutes | Validators, sentry nodes |
| **everything** | 100 GB | Current state only | **DO NOT USE** (breaks state sync) |

**Recommended Pruning:**

**Validator / Sentry:**
```toml
pruning = "custom"
pruning-keep-recent = "100"    # Minimal recent history
pruning-keep-every = "1000"    # State sync snapshots
pruning-interval = "10"
```

**API Node:**
```toml
pruning = "custom"
pruning-keep-recent = "10000"  # ~8 hours of history for queries
pruning-keep-every = "1000"
pruning-interval = "100"
```

**Archive Node:**
```toml
pruning = "nothing"  # Keep everything
```

#### API Configuration

```toml
[api]
# Enable defines if the API server should be enabled
enable = false  # true for API nodes, false for validators

# Swagger defines if swagger documentation should automatically be registered
swagger = false

# Address defines the API server to listen on
address = "tcp://0.0.0.0:1317"

# MaxOpenConnections defines the number of maximum open connections
max-open-connections = 1000

# RPCReadTimeout defines the Tendermint RPC read timeout (in seconds)
rpc-read-timeout = 10

# RPCWriteTimeout defines the Tendermint RPC write timeout (in seconds)
rpc-write-timeout = 10

# RPCMaxBodyBytes defines the Tendermint maximum response body (in bytes)
rpc-max-body-bytes = 1000000  # 1 MB

# EnableUnsafeCORS defines if CORS should be enabled (unsafe - use only for dev)
enabled-unsafe-cors = false
```

**Tuning Recommendations:**

**High-Load API Node:**
```toml
enable = true
max-open-connections = 5000    # Handle many concurrent requests
rpc-read-timeout = 30          # Allow longer queries
rpc-write-timeout = 30
rpc-max-body-bytes = 10000000  # 10 MB for large responses

# Use nginx reverse proxy with caching for production
```

#### gRPC Configuration

```toml
[grpc]
# Enable defines if the gRPC server should be enabled
enable = false  # true for API nodes

# Address defines the gRPC server address to bind to
address = "0.0.0.0:9090"

# MaxRecvMsgSize defines the max message size in bytes the server can receive
max-recv-msg-size = "10485760"  # 10 MB

# MaxSendMsgSize defines the max message size in bytes the server can send
max-send-msg-size = "2147483647"  # 2 GB
```

**Tuning for High Throughput:**
```toml
enable = true
max-recv-msg-size = "52428800"     # 50 MB (large tx batches)
max-send-msg-size = "2147483647"   # 2 GB (large query results)
```

#### State Storage Configuration

```toml
[store]
# The type of database for application and snapshots databases
db_backend = "goleveldb"  # Options: goleveldb, rocksdb

# Pruning options (duplicates [pruning] section for compatibility)
[store.pruning]
# See [pruning] section above
```

**Database Backend Comparison:**

| Backend | Read IOPS | Write IOPS | Compression | Stability | Recommendation |
|---------|-----------|------------|-------------|-----------|----------------|
| **goleveldb** | Good | Good | Fair | Stable | Default, reliable |
| **rocksdb** | Excellent | Excellent | Good | Stable | Best for high-load, requires separate build |

**Using RocksDB (Recommended for Validators):**

```bash
# Build with RocksDB support
sudo apt install -y librocksdb-dev
go build -tags rocksdb -o pawd ./cmd/...

# Update app.toml
db_backend = "rocksdb"
```

**RocksDB Configuration (create config/rocksdb.ini):**
```ini
[default]
# Increase block cache for better read performance
block_cache_size=8589934592  # 8 GB

# Increase write buffer for better write performance
write_buffer_size=67108864   # 64 MB
max_write_buffer_number=4

# Compression
compression=snappy
bottommost_compression=zstd

# Parallelism
max_background_jobs=8
max_background_compactions=4
max_background_flushes=2

# Tuning
level0_file_num_compaction_trigger=4
level0_slowdown_writes_trigger=20
level0_stop_writes_trigger=36
```

---

## Database Optimization

### LevelDB Tuning

**File:** `config/app.toml` or environment variables

```bash
# Increase LevelDB cache size (default: 8 MB → recommended: 512 MB - 2 GB)
export LEVELDB_CACHE_SIZE=536870912  # 512 MB

# Increase write buffer (default: 4 MB → recommended: 64 MB)
export LEVELDB_WRITE_BUFFER_SIZE=67108864  # 64 MB

# Increase max open files (default: 1000 → recommended: 10000)
export LEVELDB_MAX_OPEN_FILES=10000
```

**Alternatively, modify Cosmos SDK code (requires rebuild):**
```go
// app/app.go
import "github.com/syndtr/goleveldb/leveldb/opt"

// In NewApp() function:
baseappOptions = append(baseappOptions, func(bapp *baseapp.BaseApp) {
    bapp.SetStoreLoader(func(ms sdk.CommitMultiStore) error {
        return ms.LoadLatestVersionAndUpgrade(&storeUpgrades, &opt.Options{
            BlockCacheCapacity:     512 * 1024 * 1024,  // 512 MB
            WriteBuffer:            64 * 1024 * 1024,   // 64 MB
            CompactionTableSize:    32 * 1024 * 1024,   // 32 MB
            CompactionTotalSize:    512 * 1024 * 1024,  // 512 MB
        })
    })
})
```

### RocksDB Tuning (Advanced)

**Optimized for Validators (High Write Throughput):**
```ini
[default]
# Memory budgets
block_cache_size=4294967296        # 4 GB block cache
write_buffer_size=134217728        # 128 MB write buffer
max_write_buffer_number=6          # 6 write buffers = 768 MB total
db_write_buffer_size=1073741824    # 1 GB total write buffer across all column families

# Compaction
max_background_jobs=16             # More parallelism (for 8+ core systems)
max_background_compactions=8
max_background_flushes=4
max_subcompactions=4

# Level 0 tuning (reduce write stalls)
level0_file_num_compaction_trigger=2    # Start compaction sooner
level0_slowdown_writes_trigger=10       # Slowdown threshold
level0_stop_writes_trigger=15           # Hard stop threshold

# Bloom filters (faster point lookups)
bloom_filter_bits_per_key=10

# Compression (balance CPU vs storage)
compression=lz4                    # Faster than snappy, good compression
bottommost_compression=zstd        # Best compression for older data
```

**Optimized for Archive/API Nodes (High Read Throughput):**
```ini
[default]
# Large block cache for read-heavy workload
block_cache_size=17179869184       # 16 GB block cache (64 GB RAM system)
write_buffer_size=67108864         # 64 MB (writes less important)
max_write_buffer_number=3

# Read optimization
max_open_files=100000              # More file handles for large database
max_file_opening_threads=16        # Parallel file opening

# Bloom filters (critical for query performance)
bloom_filter_bits_per_key=15       # Better false positive rate for queries

# Compression (save storage on archive nodes)
compression=zstd
bottommost_compression=zstd

# Compaction (background, don't impact reads)
max_background_jobs=4              # Lower priority for compaction
max_background_compactions=2
```

### Database Compaction

**Manual Compaction (When Performance Degrades):**
```bash
# Stop the node
sudo systemctl stop pawd

# Compact the database (LevelDB)
# Note: No built-in tool, compaction happens automatically during operation

# For RocksDB, use ldb tool (if available)
ldb compact --db=/home/validator/.paw/data/application.db

# Restart the node
sudo systemctl start pawd
```

**Automatic Compaction Tuning:**
```toml
# config/app.toml
[store.compaction]
# Trigger compaction when database size grows by this factor
compaction_trigger = 4

# Maximum number of concurrent compactions
max_compactions = 2
```

---

## P2P Network Optimization

### Peer Configuration

**File:** `config/config.toml`

```toml
[p2p]
# Address to listen for incoming connections
laddr = "tcp://0.0.0.0:26656"

# External address to advertise to peers (auto-detected if empty)
external_address = ""  # Set to "your.public.ip:26656" if behind NAT

# Comma separated list of seed nodes
seeds = "seed1-node-id@seed1.paw.network:26656,seed2-node-id@seed2.paw.network:26656"

# Comma separated list of persistent peers
persistent_peers = ""  # Validators should list their sentries here

# Comma separated list of node IDs to keep private (not gossip to peers)
private_peer_ids = ""  # Validators: add your node ID here (sentry architecture)

# Maximum number of inbound peers
max_num_inbound_peers = 40

# Maximum number of outbound peers (excluding persistent)
max_num_outbound_peers = 10

# Seed mode, in which node constantly crawls the network and looks for
# peers. If another node asks it for addresses, it responds and disconnects
seed_mode = false

# Toggle to disable guard against peers connecting from the same IP
allow_duplicate_ip = false

# Peer connection configuration
handshake_timeout = "20s"
dial_timeout = "3s"

# Rate at which packets can be sent, in bytes/second
send_rate = 5120000  # 5 MB/s

# Rate at which packets can be received, in bytes/second
recv_rate = 5120000  # 5 MB/s

# Maximum size of a message packet payload, in bytes
max_packet_msg_payload_size = 1024  # 1 KB

# Flushing interval for batching messages
flush_throttle_timeout = "100ms"

# Peer exchange reactor
pex = true  # Enable for full nodes, disable for validators (use persistent_peers only)

# Address book file
addr_book_file = "config/addrbook.json"

# Set true for strict address routability rules
addr_book_strict = true
```

**Tuning Recommendations:**

**Validator Node (Sentry Architecture):**
```toml
# Connect only to trusted sentry nodes
persistent_peers = "sentry1-id@sentry1-ip:26656,sentry2-id@sentry2-ip:26656"
private_peer_ids = "validator-node-id"  # Don't advertise validator to network
pex = false                              # Don't participate in peer exchange
max_num_inbound_peers = 2                # Only allow sentries
max_num_outbound_peers = 0               # No outbound connections
allow_duplicate_ip = true                # If sentries are on same IP block

# Higher bandwidth for reliable communication with sentries
send_rate = 10240000  # 10 MB/s
recv_rate = 10240000  # 10 MB/s
```

**Sentry Node (Public-Facing):**
```toml
# Connect to validator privately
persistent_peers = "validator-id@validator-private-ip:26656"
unconditional_peer_ids = "validator-id"  # Always maintain connection to validator
private_peer_ids = "validator-id"        # Don't gossip validator address

# Allow many public peers
max_num_inbound_peers = 100              # Accept many incoming connections
max_num_outbound_peers = 50              # Actively connect to peers
pex = true                                # Participate in peer discovery

# Moderate bandwidth
send_rate = 5120000   # 5 MB/s
recv_rate = 5120000   # 5 MB/s

# DDoS protection
handshake_timeout = "10s"  # Drop slow handshakes faster
allow_duplicate_ip = false # Prevent single IP from opening many connections
```

**Full Node (Maximum Connectivity):**
```toml
seeds = "seed1@ip1:26656,seed2@ip2:26656,seed3@ip3:26656"
max_num_inbound_peers = 50
max_num_outbound_peers = 50
pex = true
send_rate = 10240000   # 10 MB/s
recv_rate = 10240000   # 10 MB/s
```

### Connection Pool Tuning

```toml
[p2p]
# Advanced connection management (usually don't need to change)

# Minimum number of outbound peers (overrides max if set higher)
# min_num_outbound_peers = 10

# Time to wait before flushing messages out on the connection
flush_throttle_timeout = "100ms"  # Lower = more frequent sends (higher CPU)

# Maximum pause when sending msg rate is limited
max_msg_batch_size_bytes = 0  # 0 = unlimited batching
```

**For Low-Latency Communication (Validators):**
```toml
flush_throttle_timeout = "10ms"  # Flush more frequently (trade CPU for latency)
```

**For Bandwidth Optimization (Sentries):**
```toml
flush_throttle_timeout = "500ms"  # Batch more messages (save bandwidth)
max_msg_batch_size_bytes = 4194304  # 4 MB batches
```

---

## Operating System Tuning

### Linux Kernel Parameters

**File:** `/etc/sysctl.conf` or `/etc/sysctl.d/99-paw.conf`

```bash
# Network performance
net.core.rmem_max = 134217728          # 128 MB receive buffer (P2P)
net.core.wmem_max = 134217728          # 128 MB send buffer
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728
net.core.netdev_max_backlog = 5000     # Queue size for incoming packets

# Connection limits
net.core.somaxconn = 4096              # Max pending connections
net.ipv4.tcp_max_syn_backlog = 8192    # SYN backlog
net.ipv4.ip_local_port_range = 1024 65535  # Available ports

# TCP optimization
net.ipv4.tcp_congestion_control = bbr  # BBR congestion control (requires kernel 4.9+)
net.ipv4.tcp_slow_start_after_idle = 0 # Disable slow start after idle
net.ipv4.tcp_tw_reuse = 1              # Reuse TIME_WAIT sockets

# File descriptors
fs.file-max = 2097152                  # System-wide max file descriptors

# Virtual memory (for database performance)
vm.swappiness = 1                      # Avoid swapping (0 = disable, 1 = minimal)
vm.dirty_ratio = 10                    # Start flushing dirty pages at 10% RAM
vm.dirty_background_ratio = 5          # Background flush at 5% RAM
vm.vfs_cache_pressure = 50             # Reduce inode/dentry cache pressure

# Apply changes
sudo sysctl -p
```

### Systemd Service Limits

**File:** `/etc/systemd/system/pawd.service`

```ini
[Unit]
Description=PAW Blockchain Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=validator
Group=validator
WorkingDirectory=/home/validator
ExecStart=/usr/local/bin/pawd start --home /home/validator/.paw

# Performance tuning
LimitNOFILE=65536           # Max file descriptors (default: 1024)
LimitNPROC=65536            # Max processes/threads
LimitMEMLOCK=infinity       # Allow unlimited memory locking

# Resource limits
MemoryMax=32G               # Hard memory limit (adjust to your instance)
CPUQuota=800%               # Use up to 8 cores (800% = 8 × 100%)

# Restart policy
Restart=on-failure
RestartSec=10s
KillSignal=SIGTERM
TimeoutStopSec=60s

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pawd

[Install]
WantedBy=multi-user.target
```

**Apply Changes:**
```bash
sudo systemctl daemon-reload
sudo systemctl restart pawd
```

### CPU Governor (Performance Mode)

```bash
# Check current governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Set to performance mode (max frequency, no scaling)
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Make permanent (Ubuntu/Debian)
sudo apt install -y cpufrequtils
echo 'GOVERNOR="performance"' | sudo tee /etc/default/cpufrequtils
sudo systemctl restart cpufrequtils

# Verify
cpufreq-info
```

**Impact:** 5-10% performance improvement for validators (eliminates frequency scaling delay)

---

## Storage and I/O Optimization

### Filesystem Tuning

**XFS (Recommended for Blockchain Workloads):**

```bash
# Format partition with XFS
sudo mkfs.xfs -f -L paw-data /dev/nvme0n1

# Mount with performance options
sudo mkdir -p /var/lib/paw
sudo mount -o noatime,nodiratime,logbsize=256k,allocsize=4M /dev/nvme0n1 /var/lib/paw

# Add to /etc/fstab for persistence
echo 'UUID=xxx /var/lib/paw xfs noatime,nodiratime,logbsize=256k,allocsize=4M 0 2' | sudo tee -a /etc/fstab
```

**Mount Options Explained:**
- `noatime`: Don't update access time (reduces writes)
- `nodiratime`: Don't update directory access time
- `logbsize=256k`: Larger log buffer (better write performance)
- `allocsize=4M`: Larger preallocation size (reduces fragmentation)

**EXT4 (Alternative):**

```bash
# Format with optimizations
sudo mkfs.ext4 -F -L paw-data -E lazy_itable_init=0,lazy_journal_init=0 /dev/nvme0n1

# Mount with performance options
sudo mount -o noatime,nodiratime,data=ordered,barrier=0,commit=60 /dev/nvme0n1 /var/lib/paw

# /etc/fstab
echo 'UUID=xxx /var/lib/paw ext4 noatime,nodiratime,data=ordered,barrier=0,commit=60 0 2' | sudo tee -a /etc/fstab
```

### I/O Scheduler

**For NVMe SSDs (No Scheduler Needed):**
```bash
# Check scheduler (should be "none" for NVMe)
cat /sys/block/nvme0n1/queue/scheduler

# If not "none", set it
echo none | sudo tee /sys/block/nvme0n1/queue/scheduler
```

**For SATA SSDs (Use deadline or mq-deadline):**
```bash
echo mq-deadline | sudo tee /sys/block/sda/queue/scheduler

# Make permanent
echo 'ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/scheduler}="mq-deadline"' | sudo tee /etc/udev/rules.d/60-ioscheduler.rules
```

### Direct I/O and Caching

**Database O_DIRECT (Advanced, Requires Code Changes):**

For RocksDB, enable direct I/O to bypass page cache:
```ini
# config/rocksdb.ini
use_direct_reads=true
use_direct_io_for_flush_and_compaction=true
```

**Benefits:**
- Reduces memory pressure (database manages its own cache)
- More predictable performance

**Drawbacks:**
- Requires tuning database cache sizes carefully
- May reduce performance if cache is too small

---

## Memory Management

### Swap Configuration

**Validators Should Minimize Swapping:**

```bash
# Check current swap usage
free -h
swapon --show

# Reduce swappiness to 1 (almost no swapping)
sudo sysctl vm.swappiness=1

# Make permanent
echo 'vm.swappiness=1' | sudo tee -a /etc/sysctl.conf

# Optionally, disable swap entirely (only if you have enough RAM)
# WARNING: Node will crash if OOM, use with caution
# sudo swapoff -a
# sudo sed -i '/swap/d' /etc/fstab
```

**Swap Sizing Recommendations:**
- **16-32 GB RAM:** 4-8 GB swap (emergency only)
- **64+ GB RAM:** 2-4 GB swap or none
- **Validators:** Minimize swap, add RAM instead

### Huge Pages (Advanced)

**Enable Transparent Huge Pages for Database Performance:**

```bash
# Check current status
cat /sys/kernel/mm/transparent_hugepage/enabled

# Enable always (best for databases)
echo always | sudo tee /sys/kernel/mm/transparent_hugepage/enabled

# Make permanent
echo 'echo always > /sys/kernel/mm/transparent_hugepage/enabled' | sudo tee -a /etc/rc.local
chmod +x /etc/rc.local
```

**Impact:** 10-20% improvement in database read/write performance

---

## API and Query Performance

### REST API Optimization

**Use Nginx Reverse Proxy with Caching:**

```nginx
# /etc/nginx/sites-available/paw-api
upstream paw_backend {
    least_conn;
    server 127.0.0.1:1317 max_fails=3 fail_timeout=30s;
}

# Cache configuration
proxy_cache_path /var/cache/nginx/paw levels=1:2 keys_zone=paw_cache:10m max_size=1g inactive=60m use_temp_path=off;

server {
    listen 443 ssl http2;
    server_name api.paw.network;

    ssl_certificate /etc/letsencrypt/live/api.paw.network/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.paw.network/privkey.pem;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/m;
    limit_req zone=api_limit burst=20 nodelay;

    location / {
        # Caching for GET requests
        proxy_cache paw_cache;
        proxy_cache_methods GET;
        proxy_cache_valid 200 15s;        # Cache successful responses for 15s
        proxy_cache_valid 404 1m;         # Cache 404s for 1 minute
        proxy_cache_use_stale error timeout updating;
        proxy_cache_background_update on;
        add_header X-Cache-Status $upstream_cache_status;

        # Compression
        gzip on;
        gzip_types application/json application/xml text/plain;
        gzip_comp_level 6;

        proxy_pass http://paw_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # Timeouts
        proxy_connect_timeout 5s;
        proxy_read_timeout 30s;
        proxy_send_timeout 30s;
    }

    # Disable caching for transaction broadcasts
    location /cosmos/tx/v1beta1/txs {
        proxy_cache off;
        proxy_pass http://paw_backend;
    }
}
```

### gRPC Load Balancing

**Nginx gRPC Proxy:**

```nginx
upstream paw_grpc {
    server grpc1:9090 max_fails=3 fail_timeout=30s;
    server grpc2:9090 max_fails=3 fail_timeout=30s;
    server grpc3:9090 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl http2;
    server_name grpc.paw.network;

    ssl_certificate /etc/letsencrypt/live/grpc.paw.network/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/grpc.paw.network/privkey.pem;

    location / {
        grpc_pass grpc://paw_grpc;
        grpc_set_header X-Real-IP $remote_addr;
        grpc_connect_timeout 5s;
        grpc_read_timeout 30s;
        grpc_send_timeout 30s;
    }
}
```

### Query Result Caching (Redis)

**Install Redis:**
```bash
sudo apt install -y redis-server
sudo systemctl enable redis-server
```

**Configure Redis for Query Caching:**
```bash
# /etc/redis/redis.conf
maxmemory 4gb
maxmemory-policy allkeys-lru  # Evict least recently used keys
```

**Application-Level Caching (Pseudocode):**
```go
// Cache frequently accessed queries
func GetAccountBalance(address string) (balance Coin, err error) {
    // Check cache first
    cacheKey := fmt.Sprintf("balance:%s", address)
    if cached, found := redisClient.Get(cacheKey); found {
        return cached, nil
    }

    // Query blockchain
    balance, err = queryBlockchain(address)
    if err != nil {
        return nil, err
    }

    // Cache result for 15 seconds
    redisClient.Set(cacheKey, balance, 15*time.Second)
    return balance, nil
}
```

---

## Monitoring Performance Metrics

### Key Metrics to Track

**Prometheus Queries:**

```prometheus
# Block processing time (p99)
histogram_quantile(0.99,
  rate(tendermint_consensus_block_processing_time_bucket[5m])
)

# Transaction throughput (TPS)
rate(tendermint_consensus_num_txs_total[1m])

# Mempool size
tendermint_mempool_size

# CPU usage
100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Memory usage
(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100

# Disk I/O wait
rate(node_disk_io_time_seconds_total[5m]) * 100

# Network throughput
rate(node_network_receive_bytes_total[5m])
rate(node_network_transmit_bytes_total[5m])

# API latency (p95)
histogram_quantile(0.95,
  rate(http_request_duration_seconds_bucket[5m])
)
```

### Performance Dashboard

**Create Grafana Dashboard:**
```json
{
  "dashboard": {
    "title": "PAW Performance Metrics",
    "panels": [
      {
        "title": "Block Processing Time (p99)",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(tendermint_consensus_block_processing_time_bucket[5m]))",
            "legendFormat": "p99"
          }
        ],
        "alert": {
          "conditions": [
            {
              "evaluator": { "params": [2], "type": "gt" },
              "query": { "params": ["A", "5m", "now"] }
            }
          ],
          "message": "Block processing time exceeded 2 seconds"
        }
      }
    ]
  }
}
```

---

## Benchmarking and Testing

### Baseline Performance Test

```bash
# 1. Record current block height
START_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

# 2. Wait 5 minutes
sleep 300

# 3. Record new block height
END_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

# 4. Calculate blocks processed
BLOCKS=$((END_HEIGHT - START_HEIGHT))
echo "Blocks in 5 minutes: $BLOCKS"
echo "Blocks per second: $((BLOCKS / 300))"
echo "Average block time: $((300 / BLOCKS)) seconds"
```

### Load Testing API

```bash
# Install Apache Bench
sudo apt install -y apache2-utils

# Test simple query (account balance)
ab -n 10000 -c 100 http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos1...

# Expected results:
# Requests per second: 500-2000 (depends on instance type)
# Mean latency: 50-200ms
```

**Using wrk (More Advanced):**
```bash
# Install wrk
sudo apt install -y wrk

# Load test
wrk -t12 -c400 -d30s --latency http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos1...

# Results interpretation:
# Latency: p50 <100ms, p99 <500ms = Good
# Throughput: >1000 req/sec = Good
```

---

## Common Performance Issues

### Issue 1: High Block Processing Time

**Symptoms:**
- Block processing >2 seconds
- Missed blocks
- High CPU usage

**Diagnosis:**
```bash
# Check block processing time
curl -s http://localhost:26657/status | jq '.result.sync_info'

# Check CPU usage
top -p $(pgrep pawd)
```

**Solutions:**
1. **Upgrade CPU:** Move to faster instance type
2. **Optimize Database:** Switch to RocksDB, increase cache
3. **Reduce Module Overhead:** Review custom module performance
4. **Pruning:** Enable aggressive pruning to reduce state size

---

### Issue 2: High Memory Usage / OOM Kills

**Symptoms:**
- Memory usage >90%
- Node crashes with OOM
- Slow performance before crash

**Diagnosis:**
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head

# Check OOM killer logs
sudo dmesg | grep -i "out of memory"
```

**Solutions:**
1. **Increase RAM:** Upgrade instance to 32 GB or 64 GB
2. **Reduce Mempool Size:** Lower `size` in config.toml
3. **Enable Pruning:** Reduce state retention
4. **Restart Periodically:** Weekly restarts to clear memory leaks (temporary fix)

---

### Issue 3: Slow Sync Speed

**Symptoms:**
- Syncing <100 blocks/second
- Weeks to catch up to chain tip

**Diagnosis:**
```bash
# Check sync speed
curl -s http://localhost:26657/status | jq '.result.sync_info.catching_up'

# Monitor block height growth
watch -n 1 'curl -s http://localhost:26657/status | jq ".result.sync_info.latest_block_height"'
```

**Solutions:**
1. **Use State Sync:** Sync from snapshot instead of replaying history
   ```toml
   # config/config.toml
   [statesync]
   enable = true
   rpc_servers = "https://rpc1.paw.network:26657,https://rpc2.paw.network:26657"
   trust_height = 1000000
   trust_hash = "ABC123..."
   ```
2. **Faster Storage:** Upgrade to NVMe SSD
3. **More Peers:** Increase `max_num_outbound_peers`
4. **Download Snapshot:** Use community-provided snapshots

---

### Issue 4: High API Latency

**Symptoms:**
- API queries taking >1 second
- Slow dashboard loading
- User complaints

**Diagnosis:**
```bash
# Test query latency
time curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos1...
```

**Solutions:**
1. **Add Caching:** Use Redis or nginx proxy cache
2. **Add Indexing:** Create database indices for frequent queries
3. **Horizontal Scaling:** Add more API nodes behind load balancer
4. **Optimize Queries:** Review slow queries, reduce data fetching
5. **Use gRPC:** gRPC is 2-3× faster than REST for same queries

---

**Related Documentation:**
- [RESOURCE_REQUIREMENTS.md](RESOURCE_REQUIREMENTS.md) - Infrastructure sizing
- [NETWORK_PORTS.md](NETWORK_PORTS.md) - Network configuration
- [SLO_TARGETS.md](SLO_TARGETS.md) - Performance targets and SLOs
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [VALIDATOR_OPERATOR_GUIDE.md](VALIDATOR_OPERATOR_GUIDE.md) - Validator-specific tuning
