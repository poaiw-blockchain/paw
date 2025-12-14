# PAW Blockchain - Validator Hardware Requirements

**Version:** 1.0
**Last Updated:** 2025-12-14
**Target:** External Validators

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Minimum Requirements (Testnet)](#minimum-requirements-testnet)
3. [Recommended Requirements (Mainnet)](#recommended-requirements-mainnet)
4. [Component Deep Dive](#component-deep-dive)
5. [Cloud Provider Recommendations](#cloud-provider-recommendations)
6. [Bare Metal Recommendations](#bare-metal-recommendations)
7. [Cost Estimates](#cost-estimates)
8. [Storage Sizing](#storage-sizing)
9. [Network Requirements](#network-requirements)
10. [Geographic Considerations](#geographic-considerations)

---

## Executive Summary

### Quick Reference

| Tier | Use Case | CPU | RAM | Storage | Network | Est. Cost/Month |
|------|----------|-----|-----|---------|---------|----------------|
| **Minimum (Testnet)** | Testing, learning | 4 cores @ 2.5 GHz | 16 GB | 200 GB NVMe SSD | 100 Mbps | $150 |
| **Recommended (Mainnet)** | Production validator | 8 cores @ 3.0 GHz | 32 GB ECC | 500 GB NVMe SSD | 1 Gbps | $350 |
| **Enterprise (Mainnet)** | Top 20 validators | 16 cores @ 3.5 GHz | 64 GB ECC | 1 TB NVMe SSD | 10 Gbps | $600 |

### Why These Specs Matter

| Component | Impact of Undersizing | Impact of Oversizing |
|-----------|----------------------|---------------------|
| **CPU** | Missed blocks, slow sync, slashing | Wasted money, minimal performance gain |
| **RAM** | OOM kills, crashes, downtime | Better caching, faster queries |
| **Storage** | Chain halt, corruption, downtime | Future-proofing, snapshots |
| **Network** | Late votes, missed blocks, isolation | Better resilience, faster sync |

---

## Minimum Requirements (Testnet)

### Hardware Specifications

```yaml
CPU:
  Cores: 4 physical cores (8 threads acceptable)
  Clock Speed: 2.5+ GHz base, 3.0+ GHz turbo
  Architecture: x86_64 (Intel/AMD)
  Required Features:
    - AVX2 (cryptographic acceleration)
    - AES-NI (encryption acceleration)
  Single-Thread Performance: Critical (Cosmos SDK is partially single-threaded)

RAM:
  Capacity: 16 GB DDR4-2666 or higher
  Type: Standard (ECC not required for testnet)
  Usage Breakdown:
    - 8 GB: Node operation (IAVL tree, mempool, consensus)
    - 4 GB: OS and system services
    - 4 GB: Buffer for peak loads

Storage:
  Capacity: 200 GB
  Type: NVMe SSD (M.2 or U.2)
  Performance:
    - Sequential Read: 200+ MB/s
    - Sequential Write: 200+ MB/s
    - Random IOPS Read: 3,000+
    - Random IOPS Write: 1,500+
    - Latency: < 5ms average
  Endurance: 200+ TBW (Terabytes Written over lifetime)

Network:
  Bandwidth: 100 Mbps symmetric (upload = download)
  Latency: < 100ms RTT to 80% of validator set
  Uptime: 99%+ (downtime acceptable for testnet)
  Traffic Estimates:
    - Ingress: ~20 GB/day
    - Egress: ~35 GB/day
```

### Operating System

```yaml
Recommended:
  - Ubuntu 22.04 LTS (preferred)
  - Debian 11/12

Acceptable:
  - CentOS Stream 8/9
  - RHEL 8/9
  - Rocky Linux 8/9

Not Recommended:
  - Windows (poor Go performance, complex setup)
  - macOS (developer use only, not production)
  - Alpine Linux (musl libc compatibility issues)

Kernel: 5.15+ (Ubuntu 22.04 ships with 5.15)
```

### Why These Are Minimums

**4 Cores:**
- CometBFT consensus runs concurrent goroutines
- P2P networking requires dedicated threads
- State machine execution is CPU-bound
- Fewer than 4 cores = frequent block misses

**16 GB RAM:**
- Cosmos SDK IAVL tree caching requires significant memory
- Mempool holds pending transactions in RAM
- Insufficient RAM causes disk swapping = slow block processing = slashing

**NVMe SSD:**
- LevelDB/RocksDB require fast random I/O
- SATA SSDs (500 IOPS) are too slow for production
- HDD (100 IOPS) will cause guaranteed slashing

---

## Recommended Requirements (Mainnet)

### Hardware Specifications

```yaml
CPU:
  Cores: 8 physical cores (16 threads)
  Clock Speed: 3.0+ GHz base, 3.5+ GHz turbo
  Recommended Processors:
    Intel:
      - Xeon Gold 6000 series (e.g., 6248R)
      - Xeon Platinum 8000 series
      - Core i9-12900K (if bare metal)
    AMD:
      - EPYC 7003 series (e.g., 7443P)
      - Ryzen 9 5950X (if bare metal)
  Required Features:
    - AVX2, AVX-512 (if available)
    - AES-NI
    - Virtualization support (Intel VT-x / AMD-V)

RAM:
  Capacity: 32 GB DDR4-3200 or DDR5
  Type: ECC (Error-Correcting Code) STRONGLY RECOMMENDED
  Configuration: Dual-channel or better
  Usage Breakdown:
    - 16 GB: Node baseline (state cache, consensus)
    - 8 GB: Database cache (LevelDB/RocksDB)
    - 8 GB: Peak load buffer, OS, monitoring

Storage:
  Capacity: 500 GB (minimum), 1 TB (recommended)
  Type: Enterprise NVMe SSD
  Recommended Models:
    - Samsung PM9A3 (enterprise, 1.3 PBW endurance)
    - Intel P5600 (datacenter, 2 PBW endurance)
    - Micron 7450 (enterprise, 4.8 PBW endurance)
  Performance:
    - Sequential Read: 3,000+ MB/s
    - Sequential Write: 2,000+ MB/s
    - Random IOPS Read: 50,000+
    - Random IOPS Write: 30,000+
    - Latency: < 1ms average, < 5ms p99
  Redundancy: RAID 1 (mirroring) recommended for data integrity
  Backup: 500 GB additional volume for snapshots (off-site)

Network:
  Bandwidth: 1 Gbps symmetric (dedicated, not shared)
  Latency:
    - < 50ms RTT to 50% of validators (regional peers)
    - < 100ms RTT to 90% of validators (global network)
  Uptime: 99.9%+ (SLA-backed)
  DDoS Protection: 10+ Gbps mitigation capacity
  Traffic Estimates:
    - Ingress: ~50 GB/day (500 GB/month)
    - Egress: ~100 GB/day (1 TB/month)
```

### Why These Are Recommended

**8 Cores:**
- Adequate parallelism for consensus + networking + state execution
- Room for monitoring agents (Prometheus exporters)
- Future-proofing as chain grows

**32 GB ECC RAM:**
- ECC prevents memory corruption (critical for financial infrastructure)
- Larger state cache = faster queries = better performance
- Headroom for network growth

**500 GB+ Enterprise NVMe:**
- Mainnet blockchain will grow 100-200 GB/year
- Enterprise drives have 10x endurance of consumer drives
- RAID 1 protects against drive failure without downtime

**1 Gbps Network:**
- Consensus requires sub-second block propagation
- Slow network = late prevotes/precommits = missed blocks = slashing
- DDoS protection shields validator from attacks

---

## Component Deep Dive

### CPU Selection

**Key Metrics:**
1. **Single-Thread Performance:** CometBFT consensus is partially single-threaded
2. **Core Count:** More cores handle P2P, state machine, monitoring
3. **Instruction Sets:** AVX2 accelerates cryptographic operations (ed25519 signatures)

**Benchmark Comparison (PassMark Single-Thread Score):**

| Processor | Cores | Base/Boost | Score | Notes |
|-----------|-------|------------|-------|-------|
| Intel Xeon Gold 6248R | 24C/48T | 3.0/4.0 GHz | 2,850 | Excellent for validators |
| AMD EPYC 7443P | 24C/48T | 2.85/4.0 GHz | 2,900 | Best price/performance |
| Intel Core i9-12900K | 16C/24T | 3.2/5.2 GHz | 4,200 | Bare metal option |
| AMD Ryzen 9 5950X | 16C/32T | 3.4/4.9 GHz | 3,500 | Bare metal option |

**Avoid:**
- ARM processors (AWS Graviton) - Go performance is lower
- Low-power CPUs (Intel Atom, Celeron) - insufficient for consensus
- Older generations (pre-2018) - missing instruction sets

### Memory Selection

**Why ECC Matters:**
- Cosmic rays can flip RAM bits (1 bit flip every few days in 32GB)
- Non-ECC: Corrupted state = chain halt, requires full resync
- ECC: Automatic error correction, no data loss

**ECC vs Non-ECC Cost:**
- ECC adds ~15-20% to memory cost
- Worth it for mainnet validators (protecting delegator funds)
- Optional for testnet

**Speed Considerations:**
- DDR4-2666: Minimum acceptable
- DDR4-3200: Recommended (5-10% performance gain)
- DDR5: Marginal benefit, expensive

### Storage Selection

**NVMe vs SATA vs HDD:**

| Metric | NVMe SSD | SATA SSD | HDD |
|--------|----------|----------|-----|
| Random IOPS Read | 50,000+ | 500 | 100 |
| Random IOPS Write | 30,000+ | 500 | 100 |
| Latency | <1ms | ~5ms | ~10ms |
| Block Processing | Excellent | Acceptable (testnet only) | WILL CAUSE SLASHING |

**Endurance (TBW - Terabytes Written):**
- Consumer SSD (Samsung 980): 600 TBW
- Enterprise SSD (PM9A3): 1,300 TBW (2.2x longer lifespan)
- Validators write ~50 GB/day = 18 TB/year
- Consumer drive lifespan: ~33 years (plenty for validators)
- Enterprise: Redundancy, better warranty

**Filesystem Recommendations:**
- **XFS:** Best for blockchain workloads (better than ext4)
- **Mount options:** `noatime,nodiratime` (disable access time updates)
- **TRIM:** Enable for SSD longevity

---

## Cloud Provider Recommendations

### AWS (Amazon Web Services)

**Recommended Instance: c6i.2xlarge**

```yaml
Specs:
  CPU: 8 vCPU (Intel Xeon Ice Lake 3.5 GHz)
  RAM: 16 GB
  Network: Up to 12.5 Gbps
  Storage: EBS gp3 500 GB (3,000 IOPS, 125 MB/s)

Pricing (us-east-1):
  On-Demand: $0.34/hour ($245/month)
  1-Year Reserved: $0.205/hour ($148/month)
  3-Year Reserved: $0.142/hour ($102/month)
  EBS gp3 500GB: $40/month
  Total (Reserved 1yr): $188/month

Pros:
  - Global availability
  - Excellent network performance
  - Mature ecosystem (CloudWatch, IAM)
  - DDoS protection (Shield Standard included)

Cons:
  - More expensive than competitors
  - EBS network latency (vs bare metal NVMe)

Setup:
  - Use Nitro instances (c6i, c6a) for best performance
  - Enable EBS-optimized
  - Use gp3 (not gp2) for cost savings
  - Provision IOPS: 3,000+ for validator
```

**Alternative for Mainnet: c6i.4xlarge**
- 16 vCPU, 32 GB RAM
- $0.68/hour ($490/month on-demand)
- Better performance, future-proof

### GCP (Google Cloud Platform)

**Recommended Instance: c2-standard-8**

```yaml
Specs:
  CPU: 8 vCPU (Cascade Lake 3.8 GHz all-core turbo)
  RAM: 32 GB
  Network: 16 Gbps
  Storage: Balanced persistent disk 500 GB

Pricing (us-central1):
  On-Demand: $0.3564/hour ($256/month)
  1-Year Committed: $0.2497/hour ($179/month)
  3-Year Committed: $0.1783/hour ($128/month)
  Disk 500GB: $50/month
  Total (1yr committed): $229/month

Pros:
  - Excellent CPU performance (highest clock speeds)
  - Generous free tier networking
  - Per-second billing (cost savings)
  - Live migration (no downtime for maintenance)

Cons:
  - Persistent disk adds network latency
  - Complex IAM (vs AWS)

Setup:
  - Use c2 series (compute-optimized)
  - Balanced persistent disk (good IOPS)
  - Enable live migration
  - Use Cloud Armor for DDoS protection
```

### Azure (Microsoft)

**Recommended Instance: F8s_v2**

```yaml
Specs:
  CPU: 8 vCPU (Intel Xeon Platinum 8272CL 3.4 GHz)
  RAM: 16 GB
  Network: 12,500 Mbps expected
  Storage: Premium SSD 512 GB (P20)

Pricing (East US):
  Pay-as-you-go: $0.398/hour ($286/month)
  1-Year Reserved: $0.265/hour ($190/month)
  3-Year Reserved: $0.174/hour ($125/month)
  Premium SSD P20: $75/month
  Total (1yr reserved): $265/month

Pros:
  - Enterprise-grade compliance
  - Good integration with Microsoft ecosystem
  - Premium SSD is actual local NVMe

Cons:
  - Higher cost than AWS/GCP
  - Complex portal UI

Setup:
  - Use Fsv2 series (compute-optimized)
  - Premium SSD (P20 or higher)
  - Enable Azure DDoS Protection Standard
```

### Hetzner (Best Value)

**Recommended Instance: CCX23**

```yaml
Specs:
  CPU: 8 vCPU (AMD EPYC 7003 series)
  RAM: 32 GB
  Network: 20 Gbps
  Storage: 240 GB NVMe (local, included)

Pricing (Germany):
  Monthly: €57.60 (~$62/month)

  Add 480 GB Volume: €15/month
  Total: €72.60 (~$78/month)

Pros:
  - BEST price/performance ratio
  - True local NVMe (not network storage)
  - Excellent network (DDoS protection included)
  - Generous bandwidth (20 TB/month included)

Cons:
  - EU-only datacenters (latency to US/Asia)
  - Less mature ecosystem than AWS/GCP
  - No live migration

Setup:
  - Use CCX line (AMD EPYC)
  - Order volume for additional storage
  - Enable Cloud Networks for private networking
```

### Digital Ocean

**Recommended Instance: c-8**

```yaml
Specs:
  CPU: 8 vCPU (Intel Xeon)
  RAM: 16 GB
  Network: 6 TB transfer
  Storage: 100 GB SSD (can add volumes)

Pricing:
  Monthly: $160
  Volume 500 GB: $50
  Total: $210/month

Pros:
  - Simple, predictable pricing
  - Easy-to-use interface
  - Good documentation
  - Snapshots included

Cons:
  - Lower performance than AWS/GCP
  - Limited global presence
  - 100 GB base storage (need volumes)

Setup:
  - Use CPU-optimized droplets
  - Add block storage volume (500 GB)
  - Enable monitoring
```

---

## Bare Metal Recommendations

### When to Use Bare Metal

**Advantages:**
- Maximum performance (no hypervisor overhead)
- Full sovereignty (you own the hardware)
- Local NVMe (lowest latency storage)
- No cloud vendor lock-in
- Lower long-term costs (own for 3+ years)

**Disadvantages:**
- Higher upfront cost
- Physical security required
- No auto-scaling
- Manual hardware replacement
- Requires colocation or home hosting

### Recommended Bare Metal Specs

**Budget Build (~$2,000):**

```yaml
CPU: AMD Ryzen 9 5900X (12C/24T, 3.7 GHz) - $400
Motherboard: ASUS ROG Strix B550-F - $180
RAM: 32 GB DDR4-3200 ECC (2x16GB) - $200
Storage: Samsung 980 Pro 1TB NVMe - $150
PSU: Corsair RM750x 750W 80+ Gold - $120
Case: Fractal Design Define R6 - $150
Cooling: Noctua NH-D15 - $100
UPS: CyberPower 1500VA - $200
Network: Intel X550-T2 Dual 10GbE - $200
Misc: Cables, fans - $100
Total: ~$1,800
```

**Enterprise Build (~$5,000):**

```yaml
CPU: AMD EPYC 7443P (24C/48T, 2.85 GHz) - $1,600
Motherboard: Supermicro H12SSL-i - $400
RAM: 64 GB DDR4-3200 ECC RDIMM (4x16GB) - $500
Storage: 2x Micron 7450 1TB NVMe (RAID 1) - $800
RAID Card: Broadcom 9560-16i - $600
PSU: Seasonic Prime TX-1000 - $300
Case: Supermicro 4U chassis - $400
Cooling: Noctua industrial fans - $200
UPS: APC Smart-UPS 3000VA - $800
Network: Mellanox ConnectX-5 25GbE - $400
Total: ~$6,000
```

### Colocation Options

| Provider | Location | Cost | Included |
|----------|----------|------|----------|
| **Equinix** | Global | $200-400/month | 1U, power, 1 Gbps |
| **Digital Realty** | US/EU | $150-300/month | 1U, power, bandwidth |
| **Flexential** | US | $100-250/month | 1U, power, 100 Mbps |

---

## Cost Estimates

### Monthly Operating Costs (Mainnet Validator)

| Component | Testnet | Mainnet | Enterprise |
|-----------|---------|---------|------------|
| **Infrastructure** | $150 | $350 | $600 |
| Compute | $100 | $250 | $450 |
| Storage | $40 | $75 | $100 |
| Network/DDoS | $10 | $25 | $50 |
| **Monitoring** | $20 | $50 | $100 |
| Prometheus/Grafana | Free (self-host) | $30 (Grafana Cloud) | $80 (Enterprise) |
| Alerting | Free | $20 (PagerDuty) | $20 |
| **Backups** | $10 | $30 | $50 |
| Snapshot storage | $10 (S3) | $30 (S3) | $50 (S3 + Glacier) |
| **Security** | $0 | $100 | $200 |
| HSM/tmkms | Free (software) | $100 (YubiHSM amortized) | $200 (CloudHSM) |
| **TOTAL/MONTH** | **$180** | **$530** | **$950** |

### Annual Costs

| Tier | Monthly | Annual | 3-Year Total |
|------|---------|--------|--------------|
| **Testnet** | $180 | $2,160 | $6,480 |
| **Mainnet** | $530 | $6,360 | $19,080 |
| **Enterprise** | $950 | $11,400 | $34,200 |

**ROI Considerations:**
- Mainnet validator earning 15% APR on 100,000 PAW stake
- At $1/PAW: $15,000/year revenue - $6,360 cost = $8,640 profit
- Break-even: ~42% APR needed at current costs

---

## Storage Sizing

### Blockchain Growth Projections

```yaml
Current State (Year 0):
  Testnet: 10 GB
  Mainnet (launch): 1 GB

Projected Growth (with 200 KB avg block, 28,800 blocks/day):

Year 1:
  Full History: 2 TB (no pruning)
  Pruned (100k blocks): 150 GB

Year 2:
  Full History: 5 TB
  Pruned: 250 GB

Year 3:
  Full History: 10 TB
  Pruned: 400 GB

Recommendation:
  - Testnet: 200 GB (headroom: 50 GB growth)
  - Mainnet: 500 GB (headroom: 350 GB growth)
  - Archive: 2 TB (expanding as needed)
```

### Pruning Strategies

**Default Pruning:**
- Keeps last 100,000 blocks (~3.5 days)
- Suitable for validators
- Storage: 100-150 GB

**Custom Pruning (Aggressive):**
```toml
pruning = "custom"
pruning-keep-recent = "10000"  # Last ~8 hours
pruning-interval = "100"
```
- Storage: 50 GB
- Risk: Less history for queries

**No Pruning (Archive):**
```toml
pruning = "nothing"
```
- Storage: 2+ TB
- Use case: Block explorers, analytics

---

## Network Requirements

### Bandwidth Calculations

**Validator Traffic (Mainnet, 200 KB blocks, 28,800 blocks/day):**

```
Ingress (Download):
  Block data: 200 KB × 28,800 = 5.5 GB/day
  Transaction relay: ~10 GB/day
  P2P overhead (handshakes, pings): ~5 GB/day
  Total: ~20 GB/day (600 GB/month)

Egress (Upload):
  Block propagation to peers: ~10 GB/day
  Vote/prevote/precommit: ~5 GB/day
  Serving state sync to new validators: ~20 GB/day
  Total: ~35 GB/day (1 TB/month)

Peak Bandwidth:
  During network upgrades: 5-10x normal
  State sync serving: 500 Mbps bursts
```

**Recommended Bandwidth:**
- **Testnet:** 100 Mbps symmetric (adequate for low activity)
- **Mainnet:** 1 Gbps symmetric (handles peaks, future-proof)
- **Enterprise:** 10 Gbps (top validators, handles DDoS)

### Latency Requirements

**Consensus Performance:**
- Validator to validator: <100ms RTT (ideal <50ms)
- Above 200ms: Consensus degrades (late votes)
- Above 500ms: Frequent missed blocks

**Geographic Latency Matrix:**

| From → To | US East | US West | EU West | Asia Pacific |
|-----------|---------|---------|---------|--------------|
| **US East** | 20ms | 70ms | 80ms | 180ms |
| **US West** | 70ms | 20ms | 150ms | 120ms |
| **EU West** | 80ms | 150ms | 20ms | 200ms |
| **Asia Pacific** | 180ms | 120ms | 200ms | 30ms |

**Recommendation:**
- Deploy in region with most validators
- Typically: US East, EU West, or Singapore
- Monitor latency to peers: `ping validator1.example.com`

### DDoS Protection

**Why DDoS Protection Matters:**
- Validators are high-value targets
- Downtime = slashing = financial loss
- Public RPC attracts automated attacks

**DDoS Mitigation Tiers:**

| Tier | Capacity | Provider | Cost |
|------|----------|----------|------|
| **Basic** | 1 Gbps | Cloud provider (AWS Shield, GCP Cloud Armor) | Free-$20/month |
| **Standard** | 10 Gbps | Cloudflare, Imperva | $200-500/month |
| **Enterprise** | 100+ Gbps | Akamai, Arbor Networks | $1,000+/month |

**For Validators:**
- Testnet: Basic (cloud provider included)
- Mainnet: Standard (10 Gbps)
- Top 10 validators: Enterprise

---

## Geographic Considerations

### Optimal Regions for Validators

**Criteria:**
1. Latency to existing validators
2. Political stability
3. Regulatory environment
4. Infrastructure quality
5. Cost

**Recommended Regions:**

| Region | Pros | Cons | Best For |
|--------|------|------|----------|
| **US East (Virginia)** | Low latency to US/EU, cheap | Regulatory scrutiny | Most validators |
| **EU West (Ireland/Germany)** | GDPR compliance, good latency | Higher costs | European operators |
| **Singapore** | Low latency to Asia, stable | Expensive | Asian market access |
| **Toronto** | Low latency to US, stable | Limited providers | Privacy-focused |

### Regulatory Considerations

**Countries to Avoid (for validators):**
- China (Great Firewall = high latency, censorship)
- Russia (geopolitical risk)
- North Korea, Iran (sanctions)

**GDPR Compliance (if applicable):**
- Blockchain data is pseudonymous (not personal data)
- Operator info may be personal data (if using real identity)
- Hosting in EU: GDPR applies

---

## Next Steps

After reviewing hardware requirements:

1. **Select your tier:** Testnet minimum vs Mainnet recommended
2. **Choose infrastructure:** Cloud vs bare metal
3. **Provision resources:** Order/launch instance
4. **Follow setup guide:** [VALIDATOR_ONBOARDING_GUIDE.md](./VALIDATOR_ONBOARDING_GUIDE.md)
5. **Setup monitoring:** [VALIDATOR_MONITORING.md](./VALIDATOR_MONITORING.md)
6. **Harden security:** [VALIDATOR_SECURITY.md](./VALIDATOR_SECURITY.md)

---

**Questions?** Join our Discord: https://discord.gg/paw-blockchain

**Last Updated:** 2025-12-14
**Maintained by:** PAW Blockchain Infrastructure Team
