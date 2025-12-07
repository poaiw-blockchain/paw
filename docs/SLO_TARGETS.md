# PAW Blockchain - Service Level Objectives (SLOs) and Performance Targets

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** Operations Teams, SRE, Platform Engineers, Management

---

## Table of Contents

1. [Overview](#overview)
2. [SLO Framework](#slo-framework)
3. [Validator Node SLOs](#validator-node-slos)
4. [API Service SLOs](#api-service-slos)
5. [Network Performance SLOs](#network-performance-slos)
6. [Storage and Database SLOs](#storage-and-database-slos)
7. [Monitoring and Alerting SLOs](#monitoring-and-alerting-slos)
8. [Error Budgets](#error-budgets)
9. [SLO Measurement and Reporting](#slo-measurement-and-reporting)
10. [Incident Response Targets](#incident-response-targets)

---

## Overview

### What are SLOs?

**Service Level Objectives (SLOs)** are target values or ranges for service levels, measured by Service Level Indicators (SLIs). SLOs define the expected performance and reliability of the PAW blockchain network.

### SLO Hierarchy

```
┌──────────────────────────────────────────────────────────────┐
│ SLA (Service Level Agreement)                               │
│ External commitments to users/validators                    │
│ Example: "99.9% uptime guaranteed or slashing penalty"      │
└──────────────────────────────────────────────────────────────┘
                         ↑
                    Supported by
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ SLO (Service Level Objective)                               │
│ Internal targets for service quality                        │
│ Example: "99.95% validator uptime target"                   │
└──────────────────────────────────────────────────────────────┘
                         ↑
                    Measured by
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ SLI (Service Level Indicator)                               │
│ Quantitative metrics of service behavior                    │
│ Example: "Missed blocks / Total blocks"                     │
└──────────────────────────────────────────────────────────────┘
```

### SLO Tiers

| Tier | Target | Use Case | Cost Impact |
|------|--------|----------|-------------|
| **Critical (99.99%)** | 52 min downtime/year | Mainnet validators | High (redundancy required) |
| **High (99.9%)** | 8.76 hours downtime/year | Public APIs, RPC endpoints | Medium (HA setup) |
| **Standard (99%)** | 3.65 days downtime/year | Testnet, monitoring | Low (single instance acceptable) |
| **Best Effort (95%)** | 18.25 days downtime/year | Development, experimental | Minimal |

---

## SLO Framework

### Key Principles

1. **User-Centric:** SLOs should reflect what users (validators, dApp developers, end users) care about
2. **Measurable:** All SLOs must have concrete, quantifiable metrics (SLIs)
3. **Achievable:** Targets should be realistic given infrastructure and budget
4. **Meaningful:** SLOs should align with business objectives and blockchain functionality
5. **Actionable:** When SLOs are violated, clear remediation steps exist

### SLI Categories

| Category | Examples | Why It Matters |
|----------|----------|----------------|
| **Availability** | Uptime %, Successful transactions | Users can access the service |
| **Latency** | Block time, API response time | Service is fast enough to be useful |
| **Throughput** | Transactions per second (TPS) | Service can handle load |
| **Error Rate** | Failed txs %, Invalid blocks | Service works correctly |
| **Consistency** | State sync time, Fork resolution | Blockchain maintains consensus |

---

## Validator Node SLOs

### Uptime and Availability

**SLO-VAL-001: Validator Uptime**
```
Target:         99.9% uptime per 30-day window
Measurement:    (Blocks signed / Total blocks) × 100
Threshold:      Sign ≥99.9% of blocks in rolling 30-day window
Consequence:    <99% uptime triggers slashing (chain parameter)
Monitoring:     Prometheus query: (1 - missed_blocks/total_blocks) × 100

Alert:          Warning at 99.5%, Critical at 99.0%
```

**Calculation:**
```
30-day window = 30 days × 28,800 blocks/day = 864,000 blocks
99.9% uptime = can miss up to 864 blocks (30 minutes of downtime)
99% uptime = can miss up to 8,640 blocks (5 hours of downtime)
```

**SLO-VAL-002: Consensus Participation**
```
Target:         ≥99.9% of pre-commit votes cast
Measurement:    Pre-commits signed / Total consensus rounds
Alert:          Warning if <99.9% in 1-hour window
Recovery:       Auto-restart if node is behind by >10 blocks
```

**SLO-VAL-003: Block Proposer Success Rate**
```
Target:         100% of assigned blocks proposed within timeout
Measurement:    (Blocks proposed / Blocks assigned) × 100
Alert:          Critical if any assigned block is missed
Cause:          Usually network latency or block processing delay
```

### Performance

**SLO-VAL-004: Block Processing Time**
```
Target:         Process blocks in <1 second (avg)
                p99 < 2 seconds
Measurement:    Time from block reception to commit
Alert:          Warning if p95 > 1.5s, Critical if p99 > 2.5s
Impact:         Slow processing → missed pre-votes → downtime
```

**SLO-VAL-005: Sync Speed (Catching Up)**
```
Target:         Sync at ≥500 blocks/second when behind
Measurement:    Block height increase per second during sync
Alert:          Warning if sync speed <200 blocks/sec
Recovery:       Use state sync if >10,000 blocks behind
```

### Resource Utilization

**SLO-VAL-006: CPU Utilization**
```
Target:         Average CPU <70%, p99 <90%
Measurement:    node_cpu_usage_percent
Alert:          Warning at 75%, Critical at 85%
Action:         Scale up instance if sustained >75% for 1 hour
```

**SLO-VAL-007: Memory Utilization**
```
Target:         Average memory <80%, p99 <90%
Measurement:    node_memory_usage_percent
Alert:          Critical at 90% (risk of OOM kill)
Action:         Restart node, investigate memory leak
```

**SLO-VAL-008: Disk I/O Wait**
```
Target:         I/O wait time <10% of CPU time
Measurement:    iostat %iowait
Alert:          Warning at 15%, Critical at 25%
Action:         Upgrade to faster storage (NVMe)
```

### Network Performance

**SLO-VAL-009: P2P Connectivity**
```
Target:         ≥10 peers connected at all times
                ≥3 peers in top 20 validators
Measurement:    tendermint_p2p_peers gauge
Alert:          Warning at <10 peers, Critical at <5 peers
Recovery:       Check firewall, restart P2P service
```

**SLO-VAL-010: Peer Latency**
```
Target:         p50 latency <50ms to peers
                p99 latency <200ms to peers
Measurement:    Ping RTT to persistent peers
Alert:          Warning if p99 >300ms (degraded consensus)
Action:         Review peer selection, consider geographic distribution
```

---

## API Service SLOs

### Availability

**SLO-API-001: API Endpoint Availability**
```
Target:         99.9% availability per endpoint per month
Measurement:    (Successful requests / Total requests) × 100
                HTTP 2xx, 3xx = success
                HTTP 5xx = failure (counted against SLO)
                HTTP 4xx = client error (not counted)
Threshold:      ≥99.9% success rate
Alert:          Warning at 99.5%, Critical at 99.0%
Error Budget:   0.1% = 43 minutes downtime/month
```

**SLO-API-002: Load Balancer Health**
```
Target:         ≥2 healthy backend nodes at all times
Measurement:    Load balancer health check success count
Alert:          Warning if 1 backend down, Critical if all backends down
Recovery:       Auto-scaling, rolling restart
```

### Latency

**SLO-API-003: REST API Response Time**
```
Target:         p50 <100ms, p95 <500ms, p99 <1000ms
Measurement:    HTTP request duration histogram
Endpoints:
  - GET /cosmos/bank/v1beta1/balances/{address}
  - GET /paw/dex/v1/pools
  - POST /cosmos/tx/v1beta1/txs

Alert:          Warning if p95 >750ms, Critical if p99 >2s
Action:         Check database query performance, add caching
```

**SLO-API-004: gRPC API Response Time**
```
Target:         p50 <50ms, p95 <200ms, p99 <500ms
Measurement:    gRPC request duration
Alert:          Warning if p95 >300ms
Note:           gRPC is faster than REST due to binary protocol
```

**SLO-API-005: WebSocket Connection Latency**
```
Target:         Event delivery within 500ms of occurrence
Measurement:    Time from block commit to WebSocket event push
Alert:          Warning if >1s delay
Use Case:       Real-time price feeds, transaction notifications
```

### Throughput

**SLO-API-006: API Request Handling Capacity**
```
Target:         Handle 1,000 requests/second per API node
                Handle 10,000 concurrent connections
Measurement:    HTTP requests per second, active connections
Alert:          Warning at 800 req/s (80% capacity)
Scaling:        Add API nodes when sustained >70% capacity
```

**SLO-API-007: Rate Limiting Accuracy**
```
Target:         Block >100 requests/minute from single IP
                Allow ≤99 requests/minute through
Measurement:    Rate limiter accuracy (false positives/negatives)
Alert:          Manual review if >1% false positive rate
```

### Error Rates

**SLO-API-008: API Error Rate**
```
Target:         <0.1% requests return 5xx errors
Measurement:    (5xx responses / Total requests) × 100
Alert:          Warning at 0.05%, Critical at 0.2%
Error Budget:   1 in 1,000 requests can fail
Exclusions:     Intentional rate limiting (429) not counted
```

**SLO-API-009: Transaction Broadcast Success Rate**
```
Target:         ≥99% of valid transactions successfully broadcast
Measurement:    (Accepted txs / Submitted txs) × 100
                (Excludes invalid transactions)
Alert:          Critical if <98% (mempool or network issue)
```

---

## Network Performance SLOs

### Consensus Performance

**SLO-NET-001: Block Time Consistency**
```
Target:         Average block time = 3 seconds ±0.5s
                p99 block time <6 seconds
Measurement:    Time between consecutive block commits
Alert:          Warning if avg block time >3.5s for 10 minutes
Impact:         Slower blocks = lower TPS, worse UX
```

**SLO-NET-002: Transaction Finality**
```
Target:         Transactions finalized within 2 blocks (6 seconds)
Measurement:    Block height when tx becomes irreversible
Alert:          Warning if any chain reorganizations occur
Note:           CometBFT provides instant finality (no reorgs in normal operation)
```

**SLO-NET-003: Consensus Round Duration**
```
Target:         Complete consensus round in <2 seconds
                Rounds: Propose → Prevote → Precommit → Commit
Measurement:    tendermint_consensus_round_duration_seconds
Alert:          Warning if round takes >3s
Cause:          Network latency, slow validators, Byzantine behavior
```

### Transaction Throughput

**SLO-NET-004: Sustained Transaction Throughput**
```
Target:         Process ≥300 transactions/second sustained
                Peak capacity ≥500 TPS for 1 minute
Measurement:    tendermint_consensus_num_txs / block_time
Alert:          Warning if mempool fills (>10,000 pending txs)
Bottleneck:     State machine execution (single-threaded)
```

**SLO-NET-005: Mempool Size**
```
Target:         Mempool <5,000 transactions under normal load
                Mempool <10,000 transactions under peak load
Measurement:    tendermint_mempool_size
Alert:          Warning at 8,000, Critical at 15,000
Action:         Increase min gas price, investigate spam
```

### Network Propagation

**SLO-NET-006: Block Propagation Time**
```
Target:         90% of validators receive block within 1 second
                99% of validators receive block within 2 seconds
Measurement:    Time from block proposal to validator receipt
Alert:          Warning if >2s to reach 90% of validators
Monitoring:     Analyze P2P gossip metrics
```

**SLO-NET-007: Transaction Propagation Time**
```
Target:         Transactions reach mempool of 50% of validators <500ms
                Transactions reach 90% of validators <2s
Measurement:    P2P gossip latency
Impact:         Slow propagation → uneven mempool → unfair tx inclusion
```

---

## Storage and Database SLOs

### Performance

**SLO-DB-001: State Database Read Latency**
```
Target:         p50 <1ms, p95 <10ms, p99 <50ms
Measurement:    LevelDB/RocksDB get() operation duration
Alert:          Warning if p95 >20ms (causes block processing delays)
Action:         Increase block cache size, upgrade to NVMe storage
```

**SLO-DB-002: State Database Write Latency**
```
Target:         p50 <5ms, p95 <50ms, p99 <100ms
Measurement:    LevelDB/RocksDB put() operation duration
Alert:          Critical if p99 >200ms (blocks consensus)
Action:         Increase write buffer, enable compression
```

**SLO-DB-003: Disk IOPS**
```
Target:         Sustained 5,000+ read IOPS
                Sustained 2,000+ write IOPS
Measurement:    iostat -x (r/s, w/s)
Alert:          Warning if IOPS <3,000 read or <1,000 write
Requirement:    NVMe SSD required for mainnet validators
```

### Storage Capacity

**SLO-DB-004: Disk Space Availability**
```
Target:         ≥20% free disk space at all times
                ≥50 GB free (absolute minimum)
Measurement:    df -h | grep /var/lib/paw
Alert:          Warning at 30% free, Critical at 15% free
Action:         Expand volume, enable pruning, archive old data
```

**SLO-DB-005: Pruning Effectiveness**
```
Target:         Pruned nodes: Storage growth <10 GB/month
                Archive nodes: Storage growth <200 GB/month
Measurement:    du -sh /var/lib/paw/data (weekly)
Alert:          Warning if growth >2× expected rate
Investigation:  Check pruning settings, database compaction
```

### Data Integrity

**SLO-DB-006: Backup Success Rate**
```
Target:         100% of scheduled backups complete successfully
                Backups taken every 6 hours
                7-day retention minimum
Measurement:    Backup job success/failure count
Alert:          Critical if any backup fails
Recovery Test:  Restore from backup monthly (verify data integrity)
```

**SLO-DB-007: Snapshot Validity**
```
Target:         100% of state sync snapshots are valid
Measurement:    State sync snapshot hash verification
Alert:          Critical if snapshot hash mismatch
Impact:         Invalid snapshots prevent new nodes from syncing
```

---

## Monitoring and Alerting SLOs

### Metrics Collection

**SLO-MON-001: Prometheus Scrape Success Rate**
```
Target:         ≥99% of scrapes succeed
                Scrape interval: 15 seconds
Measurement:    up{job="paw-validators"} == 1
Alert:          Warning if <95% scrapes succeed in 5 minutes
Impact:         Missing metrics → gaps in dashboards, missed alerts
```

**SLO-MON-002: Metrics Retention**
```
Target:         Retain 30 days of high-resolution metrics (15s intervals)
                Retain 90 days of aggregated metrics (5m intervals)
Measurement:    Prometheus TSDB size, oldest timestamp
Alert:          Warning if retention <25 days
Storage:        ~100 GB for 30-day retention (500 metrics × 10 nodes)
```

### Alerting Reliability

**SLO-MON-003: Alert Delivery Latency**
```
Target:         Alerts delivered within 1 minute of threshold breach
Measurement:    Time from metric violation to Slack/PagerDuty notification
Alert:          Meta-alert if alert delivery >2 minutes
Components:     Prometheus evaluation (15s) → AlertManager (30s) → Notification (15s)
```

**SLO-MON-004: Alert Accuracy (False Positives)**
```
Target:         <5% false positive rate on critical alerts
                <10% false positive rate on warning alerts
Measurement:    Manual incident review, alert classification
Action:         Tune alert thresholds, add context to alerts
```

**SLO-MON-005: On-Call Response Time**
```
Target:         Acknowledge critical alerts within 15 minutes
                Begin investigation within 30 minutes
Measurement:    PagerDuty acknowledgment timestamps
Alert:          Escalate to secondary on-call if no ack in 15 minutes
```

### Dashboard Performance

**SLO-MON-006: Grafana Dashboard Load Time**
```
Target:         Dashboards load in <5 seconds
Measurement:    Time to render dashboard in Grafana
Alert:          Warning if load time >10s (manual investigation)
Optimization:   Use recording rules, reduce query complexity
```

---

## Error Budgets

### Concept

**Error Budget** = (1 - SLO) × Time Window

Example: 99.9% uptime SLO → 0.1% error budget → 43 minutes downtime/month

Error budgets allow for:
- Planned maintenance
- Deployments and upgrades
- Unexpected failures
- Risk-taking (new features, experiments)

### Validator Error Budget (Monthly)

**SLO: 99.9% Uptime**
```
Time Window:        720 hours/month (30 days)
Error Budget:       0.1% × 720 hours = 0.72 hours = 43.2 minutes
In Blocks:          43.2 min × 60 sec/min ÷ 3 sec/block = 864 blocks

Budget Allocation:
  - Planned upgrades:     20 minutes (2 upgrades × 10 min)
  - Deployment changes:   10 minutes (monitoring, config)
  - Network issues:       5 minutes (transient connectivity)
  - Unexpected failures:  8.2 minutes (buffer)
                          ─────────
  Total:                  43.2 minutes
```

**Error Budget Tracking:**
```prometheus
# Prometheus query
100 - (
  (sum(increase(tendermint_consensus_height[30d])) - sum(increase(missed_blocks[30d])))
  / sum(increase(tendermint_consensus_height[30d]))
) * 100

# Result: 0.05% (21.6 min used of 43.2 min budget)
# Status: 50% of error budget consumed, 50% remaining
```

**Error Budget Policy:**
```
Budget Remaining      Action
─────────────────────────────────────────────────────────────
> 75%                 Normal operations, deploy freely
50-75%                Review deployments, prefer off-peak
25-50%                Freeze non-critical deployments
< 25%                 All-hands focus on reliability, no new features
0% (exhausted)        Incident response mode, post-mortem required
```

### API Error Budget (Monthly)

**SLO: 99.9% Availability**
```
Requests/month:     100M requests (example high-traffic API)
Error Budget:       0.1% × 100M = 100,000 failed requests
Per Day:            3,333 failed requests

Budget Allocation:
  - Deployments (5xx):        30,000 requests (3 deploys × 10k)
  - Database issues:          20,000 requests
  - Rate limiting errors:     Not counted (4xx)
  - Unexpected failures:      50,000 requests (buffer)
                              ─────────
  Total:                      100,000 requests
```

---

## SLO Measurement and Reporting

### SLI Collection (Prometheus Queries)

**Validator Uptime SLI:**
```prometheus
# 30-day uptime percentage
100 * (
  1 - (
    sum(increase(tendermint_consensus_missing_validators[30d]))
    / sum(increase(tendermint_consensus_height[30d]))
  )
)
```

**API Latency SLI (p99):**
```prometheus
# p99 API response time (last 24 hours)
histogram_quantile(0.99,
  rate(http_request_duration_seconds_bucket{job="paw-api"}[24h])
)
```

**API Error Rate SLI:**
```prometheus
# 5xx error rate (last 1 hour)
sum(rate(http_requests_total{status=~"5.."}[1h]))
/ sum(rate(http_requests_total[1h]))
* 100
```

### SLO Dashboard

**Grafana Dashboard Panels:**
```
┌────────────────────────────────────────────────────────────┐
│ PAW SLO Overview Dashboard                                │
├────────────────────────────────────────────────────────────┤
│ Row 1: Validator SLOs                                      │
│   - Uptime (30d): 99.95% [Target: 99.9%] ✓                │
│   - Missed Blocks: 432 / 864,000 [Budget: 864] ✓          │
│   - Error Budget Remaining: 50%                            │
├────────────────────────────────────────────────────────────┤
│ Row 2: API SLOs                                            │
│   - Availability (24h): 99.98% [Target: 99.9%] ✓          │
│   - p99 Latency: 450ms [Target: <1000ms] ✓                │
│   - Error Rate: 0.02% [Target: <0.1%] ✓                   │
├────────────────────────────────────────────────────────────┤
│ Row 3: Network SLOs                                        │
│   - Block Time (avg): 3.1s [Target: 3s ±0.5s] ✓           │
│   - TPS (current): 287 [Target: >300] ⚠                   │
│   - Mempool Size: 4,521 [Target: <5,000] ✓                │
├────────────────────────────────────────────────────────────┤
│ Row 4: Error Budgets                                       │
│   - Validator Budget: 50% remaining (21.6 min used)       │
│   - API Budget: 75% remaining (25k errors used)           │
│   - Status: GREEN (healthy)                                │
└────────────────────────────────────────────────────────────┘
```

### Weekly SLO Report

**Automated Weekly Report (Email/Slack):**
```
PAW Blockchain SLO Report - Week 49, 2025

┌─────────────────────────────────────────────────────────┐
│ Summary: All SLOs met this week ✓                      │
│ Error Budget Status: Healthy (60% remaining)           │
│ Incidents: 1 minor (resolved in 12 minutes)            │
└─────────────────────────────────────────────────────────┘

Validator Performance:
  ✓ Uptime: 99.97% (target: 99.9%)
  ✓ Missed Blocks: 259 / 201,600 (budget: 202)
  ⚠ 1 incident: Network connectivity issue (12 min downtime)

API Performance:
  ✓ Availability: 99.95% (target: 99.9%)
  ✓ p99 Latency: 580ms (target: <1000ms)
  ✓ Error Rate: 0.04% (target: <0.1%)

Network Performance:
  ✓ Avg Block Time: 3.05s (target: 3s ±0.5s)
  ✓ TPS Peak: 412 (target: >300)
  ✓ Mempool Max: 6,234 (target: <10,000)

Action Items:
  - Investigate network connectivity on validator-2 (completed)
  - Optimize API query caching (in progress)
  - Review mempool size increase trend (scheduled)

Next Week's Maintenance:
  - Planned upgrade to v1.2.0 (estimated 10 min downtime)
  - Budget impact: 10 min / 43 min monthly budget = 23%
```

---

## Incident Response Targets

### Severity Levels

| Severity | Impact | Response Time | Resolution Time | Examples |
|----------|--------|---------------|-----------------|----------|
| **SEV-1 (Critical)** | Mainnet down, validator slashed | 15 min | 1 hour | Consensus failure, all validators offline |
| **SEV-2 (High)** | Degraded service, SLO breach | 30 min | 4 hours | API down, high error rate, missed blocks |
| **SEV-3 (Medium)** | Partial impact, SLO at risk | 1 hour | 1 day | Single API node down, elevated latency |
| **SEV-4 (Low)** | Minimal impact, monitoring | 4 hours | 1 week | Dashboard broken, non-critical alert |

### Incident Response SLOs

**SLO-INC-001: Critical Incident Response**
```
Target:         Acknowledge within 15 minutes
                Incident commander assigned within 30 minutes
                Status update every 30 minutes
                Resolution within 1 hour (or escalate)
Measurement:    PagerDuty timestamps, incident log
Alert:          Escalate to VP Engineering if no resolution in 2 hours
```

**SLO-INC-002: Post-Mortem Completion**
```
Target:         Post-mortem published within 5 business days
                Root cause analysis, action items, timeline
Measurement:    Incident close date, post-mortem publish date
Review:         Post-mortem review meeting within 7 days
```

**SLO-INC-003: MTTR (Mean Time to Repair)**
```
Target:         SEV-1: <1 hour
                SEV-2: <4 hours
                SEV-3: <24 hours
Measurement:    Time from incident detection to resolution
Tracking:       Quarterly review, identify improvement areas
```

---

## Summary

### SLO Quick Reference

| Service | SLO | Target | Error Budget (Monthly) |
|---------|-----|--------|------------------------|
| **Validator Uptime** | SLO-VAL-001 | 99.9% | 43 minutes |
| **API Availability** | SLO-API-001 | 99.9% | 43 minutes |
| **API p99 Latency** | SLO-API-003 | <1000ms | N/A (latency) |
| **Block Time** | SLO-NET-001 | 3s ±0.5s | N/A (consistency) |
| **TPS** | SLO-NET-004 | >300 sustained | N/A (throughput) |
| **Disk Space** | SLO-DB-004 | ≥20% free | N/A (capacity) |

### Monitoring Checklist

- [ ] Prometheus scraping all targets every 15 seconds
- [ ] Grafana SLO dashboard configured and accessible
- [ ] Alerts configured for all critical SLOs (SEV-1, SEV-2)
- [ ] PagerDuty integration tested and on-call rotation defined
- [ ] Weekly SLO reports automated and sent to stakeholders
- [ ] Error budget tracking implemented and reviewed monthly
- [ ] Incident response runbooks documented and up-to-date
- [ ] Post-mortem template prepared for incident analysis

---

**Related Documentation:**
- [NETWORK_PORTS.md](NETWORK_PORTS.md) - Port configuration for monitoring
- [RESOURCE_REQUIREMENTS.md](RESOURCE_REQUIREMENTS.md) - Infrastructure sizing to meet SLOs
- [PERFORMANCE_TUNING.md](PERFORMANCE_TUNING.md) - Optimization to achieve SLO targets
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common SLO violation scenarios and fixes
- [VALIDATOR_OPERATOR_GUIDE.md](VALIDATOR_OPERATOR_GUIDE.md) - Validator-specific SLO guidance
