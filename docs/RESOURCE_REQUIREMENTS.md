# PAW Blockchain - Resource Requirements and Sizing Guide

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** DevOps Engineers, Infrastructure Teams, Node Operators

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Node Type Comparison](#node-type-comparison)
3. [Validator Node Requirements](#validator-node-requirements)
4. [Full Node Requirements](#full-node-requirements)
5. [Archive Node Requirements](#archive-node-requirements)
6. [API Node Requirements](#api-node-requirements)
7. [Sentry Node Requirements](#sentry-node-requirements)
8. [Development Environment Requirements](#development-environment-requirements)
9. [Monitoring Stack Requirements](#monitoring-stack-requirements)
10. [Network Requirements](#network-requirements)
11. [Storage Considerations](#storage-considerations)
12. [Kubernetes Resource Quotas](#kubernetes-resource-quotas)
13. [Scaling Guidelines](#scaling-guidelines)
14. [Performance Benchmarks](#performance-benchmarks)

---

## Executive Summary

### Quick Reference Table

| Node Type | CPU | RAM | Storage (Year 1) | Network | Monthly Cost (AWS) |
|-----------|-----|-----|------------------|---------|-------------------|
| **Validator (Testnet)** | 4 cores | 16 GB | 200 GB NVMe | 100 Mbps | ~$150 |
| **Validator (Mainnet)** | 8 cores | 32 GB | 500 GB NVMe | 1 Gbps | ~$350 |
| **Full Node (Pruned)** | 4 cores | 16 GB | 200 GB SSD | 100 Mbps | ~$120 |
| **Archive Node** | 8 cores | 64 GB | 2 TB NVMe | 500 Mbps | ~$600 |
| **API Node** | 8 cores | 32 GB | 500 GB SSD | 1 Gbps | ~$350 |
| **Sentry Node** | 4 cores | 16 GB | 200 GB SSD | 500 Mbps | ~$150 |
| **Development** | 2 cores | 8 GB | 50 GB SSD | 10 Mbps | ~$40 |

### Growth Projections (Annual)

| Metric | Year 1 | Year 2 | Year 3 | Notes |
|--------|--------|--------|--------|-------|
| **Block Size (avg)** | 200 KB | 300 KB | 500 KB | Depends on tx throughput |
| **Blocks per Day** | 28,800 | 28,800 | 28,800 | 3-second block time |
| **Storage Growth (pruned)** | 150 GB | 250 GB | 400 GB | With default pruning (100 blocks) |
| **Storage Growth (archive)** | 2 TB | 5 TB | 10 TB | Full history retention |
| **RAM Usage (validator)** | 16 GB | 24 GB | 32 GB | With network growth |
| **P2P Connections** | 50 peers | 100 peers | 200 peers | Mainnet growth |

---

## Node Type Comparison

### Validator Node

**Purpose:** Participate in consensus, sign blocks, earn rewards.

**Characteristics:**
- **Uptime Requirement:** 99.9%+ (downtime = slashing)
- **Security Requirement:** Highest (controls staked funds)
- **API Exposure:** None (internal RPC only)
- **Sync Requirement:** Must be at chain tip (no lag tolerance)
- **Redundancy:** Active/standby failover recommended

**When to Use:**
- You are staking PAW tokens and validating blocks
- Participating in governance
- Earning staking rewards

---

### Full Node (Pruned)

**Purpose:** Verify transactions, relay blocks, maintain recent state.

**Characteristics:**
- **Uptime Requirement:** 95%+ (no slashing, but network benefits from uptime)
- **Security Requirement:** Medium (no staked funds)
- **API Exposure:** Optional (RPC/API can be enabled)
- **Sync Requirement:** Should stay synced, can catch up if behind
- **Redundancy:** Optional (failover reduces resync time)

**When to Use:**
- Running a private node for wallet/dApp access
- Contributing to network decentralization
- Testing applications against live network
- Not validating (no stake at risk)

---

### Archive Node

**Purpose:** Maintain full blockchain history for analytics, indexing, explorers.

**Characteristics:**
- **Uptime Requirement:** 95%+ (depends on use case)
- **Security Requirement:** Low to Medium
- **API Exposure:** Often public (for block explorers)
- **Sync Requirement:** Must have full history from genesis
- **Redundancy:** Recommended (long resync times)

**When to Use:**
- Running a block explorer
- Providing historical data APIs
- Blockchain analytics and research
- Audit and compliance requirements

---

### API Node

**Purpose:** Serve public API/RPC requests, no consensus participation.

**Characteristics:**
- **Uptime Requirement:** 99%+ (user-facing)
- **Security Requirement:** Medium (DDoS protection critical)
- **API Exposure:** Public (behind rate limiting/auth)
- **Sync Requirement:** Must be synced, slight lag acceptable
- **Redundancy:** Highly recommended (load balancing required)

**When to Use:**
- Providing RPC/API service to wallets/dApps
- Running as infrastructure for your application
- Public service to community

---

### Sentry Node

**Purpose:** Shield validators from direct internet exposure (sentry architecture).

**Characteristics:**
- **Uptime Requirement:** 99%+ (validator connectivity depends on it)
- **Security Requirement:** High (DDoS protection)
- **API Exposure:** None (P2P only)
- **Sync Requirement:** Must be synced with validator
- **Redundancy:** Required (minimum 2 sentries per validator)

**When to Use:**
- Operating a mainnet validator (recommended architecture)
- High-value validator operations (top 20 validators)
- Enhanced security posture

---

### Development Environment

**Purpose:** Local development, testing, not connected to live network.

**Characteristics:**
- **Uptime Requirement:** None (local use only)
- **Security Requirement:** Low (dev environment)
- **API Exposure:** Localhost only
- **Sync Requirement:** None (can reset anytime)
- **Redundancy:** None

**When to Use:**
- Smart contract development
- Testing transactions
- Module development
- CI/CD pipelines

---

## Validator Node Requirements

### Testnet Validator

**Minimum Specifications:**
```
CPU:     4 cores @ 2.5+ GHz (Intel Xeon, AMD EPYC, or equivalent)
         - Must support AVX2 instructions (for cryptographic operations)
         - Multi-threaded performance critical (CometBFT is concurrent)

RAM:     16 GB DDR4/DDR5
         - 8 GB for node operation
         - 4 GB for OS and system services
         - 4 GB buffer for peak loads

Storage: 200 GB NVMe SSD
         - IOPS: 3,000+ read, 1,500+ write (minimum)
         - Latency: <5ms average
         - Throughput: 200+ MB/s sequential read/write
         - Endurance: 200+ TBW (terabytes written)

Network: 100 Mbps symmetric (upload = download)
         - Latency: <100ms to 80% of validator set
         - Bandwidth: 50 GB/month ingress, 100 GB/month egress
         - Uptime: 99%+

OS:      Ubuntu 22.04 LTS (recommended)
         - Alternatives: Debian 11/12, CentOS 8, RHEL 8
```

**Recommended Specifications:**
```
CPU:     8 cores @ 3.0+ GHz
RAM:     32 GB
Storage: 500 GB NVMe SSD (enterprise-grade)
Network: 500 Mbps symmetric
```

**Rationale:**
- **CPU:** CometBFT consensus requires fast single-thread performance for vote processing. More cores help with P2P networking and state machine execution.
- **RAM:** Cosmos SDK applications cache significant state in memory. Insufficient RAM causes disk swapping, degrading performance and potentially causing missed blocks.
- **Storage:** Fast storage is critical for state database (LevelDB/RocksDB). Slow disks cause block processing delays, risking slashing.
- **Network:** Validators must receive and propagate blocks/votes within 1-2 seconds. Slow network = missed prevotes/precommits = downtime penalties.

---

### Mainnet Validator

**Minimum Specifications:**
```
CPU:     8 cores @ 3.0+ GHz (Intel Xeon Gold, AMD EPYC 7xx3, or better)
         - AVX2, AES-NI instruction support required
         - Turbo boost for single-thread performance
         - ECC memory support recommended

RAM:     32 GB DDR4-3200 or DDR5 (ECC recommended)
         - 16 GB for node baseline
         - 8 GB for mempool and consensus
         - 8 GB buffer for traffic spikes

Storage: 500 GB NVMe SSD (Enterprise/Datacenter grade)
         - IOPS: 10,000+ read, 5,000+ write
         - Latency: <1ms average, <5ms p99
         - Throughput: 500+ MB/s sequential
         - Endurance: 1+ PBW (petabytes written)
         - RAID 1 (mirroring) recommended for data integrity

Network: 1 Gbps symmetric (dedicated, not shared)
         - Latency: <50ms to 50% of validators, <100ms to 90%
         - Bandwidth: 500 GB/month ingress, 1 TB/month egress
         - Uptime: 99.9%+ (SLA-backed if using cloud provider)
         - DDoS protection (10+ Gbps mitigation capacity)

OS:      Ubuntu 22.04 LTS (hardened, minimal install)
```

**Recommended Specifications (Top 20 Validators):**
```
CPU:     16 cores @ 3.5+ GHz (Intel Xeon Platinum, AMD EPYC 7xx3)
RAM:     64 GB DDR4-3200 ECC
Storage: 1 TB NVMe SSD (enterprise) + 2 TB backup volume
Network: 10 Gbps symmetric with DDoS protection
Redundancy: Active/standby validator + 2+ sentry nodes
```

**Backup Requirements:**
```
Backup Frequency: Every 6 hours (state snapshots)
Backup Retention:  7 days of snapshots
Backup Storage:    500 GB (off-site, encrypted)
Backup Network:    Separate from primary (out-of-band)
RTO (Recovery Time Objective): <15 minutes
RPO (Recovery Point Objective): <6 hours
```

**Geographic Considerations:**
- **Latency:** Deploy in region with good connectivity to other validators (typically US-East, EU-West, or Asia-Pacific)
- **Redundancy:** Standby validator in different geographic region
- **Regulatory:** Ensure jurisdiction allows blockchain validation

---

### Resource Allocation Breakdown (Mainnet Validator)

```
┌─────────────────────────────────────────────────────────────┐
│ CPU Allocation (8 cores)                                    │
├─────────────────────────────────────────────────────────────┤
│ Core 0-1:  CometBFT Consensus (vote processing, block prop)│
│ Core 2-3:  P2P Networking (50+ peer connections)            │
│ Core 4-5:  Cosmos SDK State Machine (Tx execution)          │
│ Core 6:    Database I/O (LevelDB compaction)                │
│ Core 7:    OS / Monitoring / Spare capacity                 │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ RAM Allocation (32 GB)                                      │
├─────────────────────────────────────────────────────────────┤
│ 12 GB:  Application state cache (Cosmos SDK IAVL tree)      │
│  4 GB:  Mempool (pending transactions)                      │
│  4 GB:  P2P networking buffers                              │
│  4 GB:  CometBFT consensus state                            │
│  4 GB:  Database cache (LevelDB block cache)                │
│  2 GB:  OS, monitoring agents, logs                         │
│  2 GB:  Spare capacity (peak load buffer)                   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Storage Allocation (500 GB NVMe)                            │
├─────────────────────────────────────────────────────────────┤
│ 250 GB: Blockchain data (state database, blocks, indices)   │
│ 100 GB: Snapshots (local state snapshots for quick restore) │
│  50 GB: Logs (systemd journal, application logs)            │
│  50 GB: WAL (write-ahead log for crash recovery)            │
│  50 GB: Spare capacity (growth buffer)                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Network Bandwidth (1 Gbps)                                  │
├─────────────────────────────────────────────────────────────┤
│ Ingress (download):                                         │
│   - Block propagation: ~10 Mbps average, 50 Mbps peak       │
│   - P2P sync/discovery: ~5 Mbps                             │
│   - Transaction relay: ~10 Mbps average, 100 Mbps peak      │
│                                                             │
│ Egress (upload):                                            │
│   - Block propagation: ~20 Mbps average, 100 Mbps peak      │
│   - Transaction relay: ~10 Mbps                             │
│   - Peer syncing: ~50 Mbps (when serving state sync)        │
└─────────────────────────────────────────────────────────────┘
```

---

## Full Node Requirements

### Pruned Full Node

**Specifications:**
```
CPU:     4 cores @ 2.5+ GHz
RAM:     16 GB
Storage: 200 GB SSD (NVMe preferred, SATA acceptable)
         - Pruning strategy: Keep last 100 blocks + state snapshots every 1000 blocks
         - IOPS: 1,000+ read, 500+ write
Network: 100 Mbps symmetric
OS:      Ubuntu 22.04 LTS
```

**Pruning Configuration:**
```toml
# config/app.toml
[pruning]
pruning = "custom"
pruning-keep-recent = "100"      # Keep last 100 blocks
pruning-keep-every = "1000"      # Snapshot every 1000 blocks
pruning-interval = "10"          # Prune every 10 blocks
```

**Storage Growth:**
- **Month 1:** 50 GB
- **Month 6:** 150 GB
- **Year 1:** 200 GB (with pruning)
- **Year 2:** 300 GB

---

### Default Full Node (Keep Everything)

**Specifications:**
```
CPU:     4 cores @ 2.5+ GHz
RAM:     16 GB
Storage: 500 GB SSD (growing to 1 TB in year 1)
Network: 100 Mbps
```

**Pruning Configuration:**
```toml
# config/app.toml
[pruning]
pruning = "default"
# Keeps last 362880 blocks (~6 weeks with 3s block time)
```

---

## Archive Node Requirements

**Specifications:**
```
CPU:     8 cores @ 3.0+ GHz
         - Higher single-thread performance for historical queries
         - More cores for concurrent RPC requests

RAM:     64 GB
         - Large state cache for historical queries
         - Query result caching
         - Multiple concurrent RPC connections

Storage: 2 TB NVMe SSD (Year 1) → 5 TB (Year 2) → 10 TB (Year 3)
         - IOPS: 5,000+ read, 2,000+ write
         - Consider RAID 10 for performance + redundancy
         - Backup to cold storage (S3 Glacier, etc.)

Network: 500 Mbps symmetric (1 Gbps recommended if serving public API)
         - High egress bandwidth for serving historical data

OS:      Ubuntu 22.04 LTS
```

**Pruning Configuration:**
```toml
# config/app.toml
[pruning]
pruning = "nothing"  # Keep all historical state
```

**Database Tuning for Archive Nodes:**
```toml
# config/app.toml
[state-sync]
snapshot-interval = 1000
snapshot-keep-recent = 10

[store]
streamers = []

# Increase LevelDB cache
[db]
cache_size = 16384  # 16 GB cache
```

**Storage Breakdown (Archive Node - Year 1):**
```
Blockchain Data:     1.5 TB (blocks + transactions + state)
Indices:             200 GB (tx index, address index)
Snapshots:           100 GB (state snapshots for fast sync)
Logs:                50 GB
Spare:               150 GB
Total:               2 TB
```

---

## API Node Requirements

**Specifications:**
```
CPU:     8 cores @ 3.0+ GHz
         - High concurrent request handling (RPC, REST, gRPC)
         - Query processing overhead

RAM:     32 GB
         - 16 GB for node operation
         - 8 GB for query result caching (Redis/in-memory)
         - 8 GB for connection pooling

Storage: 500 GB SSD (pruned mode acceptable)
         - IOPS: 3,000+ read, 1,000+ write
         - Fast queries require fast storage

Network: 1 Gbps symmetric (public-facing)
         - Ingress: API requests (up to 10k req/sec with caching)
         - Egress: API responses (large query results)
         - DDoS protection mandatory (10+ Gbps mitigation)

OS:      Ubuntu 22.04 LTS
```

**Load Balancer Configuration (Kubernetes):**
```yaml
# Horizontal scaling: 3+ API nodes behind load balancer
apiVersion: v1
kind: Service
metadata:
  name: paw-api-lb
spec:
  type: LoadBalancer
  selector:
    app: paw-api
  ports:
    - name: rest
      port: 1317
      targetPort: 1317
    - name: grpc
      port: 9090
      targetPort: 9090
  sessionAffinity: ClientIP  # Sticky sessions for WebSocket
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
```

**API Node Caching Strategy:**
```
Redis Cache:
  - TTL: 15 seconds for rapidly changing data (balances, pools)
  - TTL: 5 minutes for semi-static data (validator info, params)
  - TTL: 1 hour for static data (historical blocks, tx)

CDN (Cloudflare/Fastly):
  - Cache static API responses (historical data)
  - DDoS protection
  - Global distribution

Application-level Cache:
  - In-memory LRU cache for hot queries
  - Size: 4 GB (configurable)
```

---

## Sentry Node Requirements

**Specifications:**
```
CPU:     4 cores @ 2.5+ GHz
RAM:     16 GB
Storage: 200 GB SSD (pruned)
Network: 500 Mbps symmetric
         - DDoS protection critical (absorbs attacks for validator)
OS:      Ubuntu 22.04 LTS
```

**Sentry Architecture:**
```
Internet
   ↓
[Firewall / DDoS Protection]
   ↓
[Sentry Node 1] ←→ [Validator] ←→ [Sentry Node 2]
   ↓                                    ↓
[Public P2P Network]          [Public P2P Network]
```

**Configuration:**
```toml
# Validator config/config.toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = "sentry1_node_id@sentry1_ip:26656,sentry2_node_id@sentry2_ip:26656"
private_peer_ids = ""
addr_book_strict = true
pex = false  # Disable peer exchange (only connect to sentries)

# Sentry Node config/config.toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = "validator_node_id@validator_private_ip:26656"
private_peer_ids = "validator_node_id"  # Don't advertise validator to network
pex = true  # Enable peer exchange (connect to public network)
unconditional_peer_ids = "validator_node_id"
```

**Minimum Sentry Count:**
- **Testnet:** 1 sentry (acceptable risk)
- **Mainnet:** 2+ sentries (recommended: 3 for redundancy)

---

## Development Environment Requirements

**Minimal Local Development:**
```
CPU:     2 cores (laptop/desktop adequate)
RAM:     8 GB
Storage: 50 GB SSD
Network: 10 Mbps (internet for downloading dependencies)
OS:      Ubuntu 22.04, macOS 12+, Windows 11 WSL2
```

**Docker Development Environment:**
```yaml
# docker-compose.yml
services:
  paw-dev:
    image: paw:dev
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

**CI/CD Pipeline Resources:**
```
GitHub Actions Runner:
  CPU:  2 cores
  RAM:  7 GB (GitHub-hosted runner)
  Storage: 14 GB SSD

GitLab CI Runner:
  CPU:  2 cores
  RAM:  4 GB
  Storage: 20 GB
```

---

## Monitoring Stack Requirements

### Prometheus + Grafana + Loki (Full Stack)

**Specifications:**
```
CPU:     4 cores @ 2.5+ GHz
         - Prometheus query engine is CPU-intensive
         - Grafana dashboard rendering

RAM:     16 GB
         - 8 GB for Prometheus TSDB
         - 4 GB for Grafana
         - 2 GB for Loki
         - 2 GB for AlertManager

Storage: 500 GB SSD
         - Prometheus metrics: 100 GB (30-day retention)
         - Loki logs: 200 GB (7-day retention)
         - Grafana dashboards: 10 GB
         - Spare: 190 GB

Network: 100 Mbps
         - Scrapes from all nodes every 15 seconds

OS:      Ubuntu 22.04 LTS
```

**Storage Growth (Prometheus):**
```
Metrics Retention Calculation:
  - Metrics scraped: 500 metrics per node
  - Scrape interval: 15 seconds
  - Nodes monitored: 10
  - Storage per sample: ~2 bytes (compressed)

  Daily storage = (500 metrics * 10 nodes * 86400 seconds / 15 sec interval * 2 bytes)
                = ~576 MB/day

  30-day retention = 576 MB * 30 = ~17 GB
  90-day retention = 576 MB * 90 = ~52 GB
```

**Production Monitoring Stack (High Availability):**
```
Prometheus:
  - 3 replicas (HA with Thanos)
  - 8 GB RAM each
  - 200 GB storage each

Grafana:
  - 2 replicas (load balanced)
  - 4 GB RAM each

Loki:
  - 3 ingesters + 2 queriers (distributed)
  - 8 GB RAM per ingester
  - 500 GB S3 backend storage
```

---

## Network Requirements

### Bandwidth Estimation

**Validator Node (Mainnet):**
```
Ingress (Download):
  - Block data: 200 KB/block * 28,800 blocks/day = 5.5 GB/day
  - Transaction relay: ~10 GB/day (depends on mempool activity)
  - P2P overhead: ~5 GB/day
  Total: ~20 GB/day ingress

Egress (Upload):
  - Block propagation: ~10 GB/day
  - Vote propagation: ~5 GB/day
  - Peer sync (serving blocks): ~20 GB/day
  Total: ~35 GB/day egress

Monthly Total: ~1.6 TB ingress + ~1 TB egress = 2.6 TB/month
```

**API Node (Public):**
```
Egress dominated (serving API requests):
  - 10,000 requests/day * 50 KB average response = 500 MB/day
  - High traffic days: up to 10 GB/day

Monthly Total: ~300 GB/month (low traffic) to 5 TB/month (high traffic)
```

### Latency Requirements

**Validator to Validator:**
- **Target:** <100ms RTT (round-trip time)
- **Maximum:** <200ms RTT (consensus may degrade above this)
- **Measurement:** Ping other validators regularly, monitor in Grafana

**API Response Times:**
- **Simple queries:** <100ms (account balance, single tx)
- **Complex queries:** <1 second (pool analytics, aggregations)
- **Historical queries:** <5 seconds (archive nodes)

**Network Quality Monitoring:**
```bash
# Monitor latency to peer validators
for peer in validator1.paw.network validator2.paw.network; do
  ping -c 10 $peer | tail -1 | awk '{print $4}' | cut -d'/' -f2
done

# Monitor packet loss
ping -c 100 validator1.paw.network | grep 'packet loss'
```

---

## Storage Considerations

### Storage Technology Comparison

| Technology | IOPS | Latency | Cost | Use Case |
|------------|------|---------|------|----------|
| **NVMe SSD** | 10,000+ | <1ms | High | Validators, API nodes (required) |
| **SATA SSD** | 1,000 | ~5ms | Medium | Full nodes (acceptable) |
| **SAS HDD** | 200 | ~10ms | Low | Archive cold storage (not for hot path) |
| **Network SSD (EBS gp3)** | 3,000-16,000 | <10ms | Medium | Cloud deployments (acceptable with tuning) |

### Database Tuning

**LevelDB (Default):**
```toml
# config/app.toml
[leveldb]
block-cache-size = 8388608     # 8 GB block cache
write-buffer-size = 67108864   # 64 MB write buffer
max-open-files = 1024
```

**RocksDB (Alternative, better for large datasets):**
```bash
# Build with RocksDB support
go build -tags rocksdb -o pawd ./cmd/...
```

```toml
# config/app.toml
[rocksdb]
max-open-files = 1024
max-file-opening-threads = 16
```

### Filesystem Optimization

```bash
# XFS recommended for blockchain workloads (better than ext4)
sudo mkfs.xfs -f -L paw-data /dev/nvme0n1
sudo mount -o noatime,nodiratime /dev/nvme0n1 /var/lib/paw

# Add to /etc/fstab for persistence
echo 'UUID=xxx /var/lib/paw xfs noatime,nodiratime 0 2' | sudo tee -a /etc/fstab

# Disable access time updates (performance boost)
sudo mount -o remount,noatime,nodiratime /var/lib/paw
```

---

## Kubernetes Resource Quotas

### Validator StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: paw-validator
  namespace: paw-blockchain
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: validator
          image: paw:v1.0.0
          resources:
            requests:
              cpu: '4'            # Guaranteed CPU
              memory: '16Gi'      # Guaranteed RAM
              ephemeral-storage: '10Gi'
            limits:
              cpu: '8'            # Max CPU (burstable)
              memory: '32Gi'      # Max RAM (hard limit)
              ephemeral-storage: '20Gi'
          volumeMounts:
            - name: data
              mountPath: /root/.paw
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ['ReadWriteOnce']
        storageClassName: fast-ssd  # NVMe storage class
        resources:
          requests:
            storage: 500Gi
```

### API Node Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: paw-api
  namespace: paw-blockchain
spec:
  replicas: 3  # Horizontal scaling
  template:
    spec:
      containers:
        - name: api
          image: paw:v1.0.0
          resources:
            requests:
              cpu: '4'
              memory: '16Gi'
            limits:
              cpu: '8'
              memory: '32Gi'
          volumeMounts:
            - name: data
              mountPath: /root/.paw
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: paw-api-data
```

### Namespace Resource Quota

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: paw-blockchain-quota
  namespace: paw-blockchain
spec:
  hard:
    requests.cpu: '50'          # Total CPU across all pods
    requests.memory: '200Gi'    # Total RAM
    requests.storage: '5Ti'     # Total storage
    persistentvolumeclaims: '20'
    pods: '50'
```

### Horizontal Pod Autoscaler (API Nodes)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: paw-api-hpa
  namespace: paw-blockchain
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: paw-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70  # Scale when CPU > 70%
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80  # Scale when RAM > 80%
```

---

## Scaling Guidelines

### Vertical Scaling (Increase Resources)

**When to Scale Up:**
- CPU usage consistently >80%
- Memory usage >90% (risk of OOM kills)
- Disk I/O wait time >20%
- Increased block processing time (missing blocks)

**Scaling Path (Validator):**
```
Tier 1 (Testnet):     4 cores, 16 GB RAM, 200 GB storage
Tier 2 (Early Mainnet): 8 cores, 32 GB RAM, 500 GB storage
Tier 3 (Growth):      16 cores, 64 GB RAM, 1 TB storage
Tier 4 (High Volume): 32 cores, 128 GB RAM, 2 TB storage
```

### Horizontal Scaling (Add Nodes)

**API Nodes (Recommended for public APIs):**
```
Low Traffic (<1k req/min):    2 API nodes
Medium Traffic (1-10k req/min): 5 API nodes
High Traffic (10-50k req/min):  10+ API nodes
```

**Load Distribution:**
- Use Kubernetes HPA (Horizontal Pod Autoscaler)
- Target 60-70% CPU utilization across pods
- DNS round-robin or L4 load balancer

### Geographic Distribution

**Multi-Region Deployment:**
```
Region 1 (US-East):     Validator 1, Sentry 1, API nodes
Region 2 (EU-West):     Validator 2, Sentry 2, API nodes
Region 3 (Asia-Pacific): Validator 3, Sentry 3, API nodes
```

**Benefits:**
- Lower latency for global users
- Higher resilience (region outages)
- Better consensus participation (distributed validators)

---

## Performance Benchmarks

### Transaction Throughput

**Hardware:** 8 cores, 32 GB RAM, NVMe SSD
```
Simple Transfers:        500 TPS (transactions per second)
DEX Swaps:               200 TPS
Complex Contracts:       50 TPS
Mixed Workload:          300 TPS average
```

**Bottleneck:** State machine execution (single-threaded in Cosmos SDK)

### Block Processing Time

**Target:** <1 second per block (with 3-second block time)
```
Consensus Time:          ~500ms (pre-commit, commit, finalize)
State Machine Execution: ~300ms (tx validation, state updates)
Database Commit:         ~100ms (write to disk)
P2P Propagation:         ~100ms (broadcast to peers)
```

### Query Performance (API Node)

**Simple Queries:**
```
Account Balance:         10-20ms
Single Transaction:      10-30ms
Block by Height:         20-50ms
```

**Complex Queries:**
```
DEX Pool Analytics:      100-500ms
Oracle Price History:    200-1000ms
Compute Job Listings:    100-300ms
```

**Optimization:**
- Use query result caching (Redis)
- Enable pruning for API nodes (don't need full history)
- Index frequently queried data

---

## Summary

### Quick Decision Matrix

**I want to:** → **Choose this node type:**

| Goal | Node Type | Min Specs |
|------|-----------|-----------|
| Earn staking rewards | Validator (Mainnet) | 8 cores, 32 GB, 500 GB NVMe |
| Test validator setup | Validator (Testnet) | 4 cores, 16 GB, 200 GB SSD |
| Support network (no stake) | Full Node (Pruned) | 4 cores, 16 GB, 200 GB SSD |
| Run block explorer | Archive Node | 8 cores, 64 GB, 2 TB NVMe |
| Serve public API/RPC | API Node | 8 cores, 32 GB, 500 GB SSD |
| Protect my validator | Sentry Node (2+) | 4 cores, 16 GB, 200 GB SSD |
| Local development | Dev Environment | 2 cores, 8 GB, 50 GB SSD |

### Cost Optimization Tips

1. **Use pruning for non-archive nodes** (saves 70% storage)
2. **Spot instances for non-critical nodes** (saves 60-80% on cloud)
3. **Reserved instances for validators** (saves 30-40% with commitment)
4. **Separate API from validator workloads** (scale independently)
5. **Geographic placement** (deploy in low-cost regions where latency permits)

---

**For additional guidance:**
- See [NETWORK_PORTS.md](NETWORK_PORTS.md) for port configuration
- See [COST_ESTIMATES.md](COST_ESTIMATES.md) for detailed pricing
- See [PERFORMANCE_TUNING.md](PERFORMANCE_TUNING.md) for optimization
- See [VALIDATOR_OPERATOR_GUIDE.md](VALIDATOR_OPERATOR_GUIDE.md) for validator-specific setup
