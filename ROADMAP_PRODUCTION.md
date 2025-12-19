# PAW Production Roadmap - Pending Tasks

**Status**: 98% Complete - Only operational follow-ups and K8s testing remain.

---

## Operational Follow-ups

- [ ] Wire Alertmanager receivers (Slack/email/PagerDuty) for `service=ibc` alerts
- [ ] Bring up monitoring stack and confirm Prometheus loads IBC rule group
- [ ] Publish operator note linking dashboard + alert + runbook
- [ ] Add resolved notifications in Alertmanager routing for boundary alerts
- [ ] Complete per-module findings, severity tagging, and regression test commitments

---

## Kubernetes Infrastructure Testing (330 tests)

> Execute locally before cloud deployment. Scripts in `k8s/tests/`.

```bash
cd /home/hudson/blockchain-projects/paw/k8s
./scripts/setup-kind-cluster.sh && ./scripts/deploy-local.sh
./tests/smoke-tests.sh && ./tests/integration-tests.sh
./tests/chaos-tests.sh && ./tests/security-tests.sh
```

### Phase 1: Infrastructure (45 tests)

**Pod Lifecycle**: Running state SLA, init container ordering, genesis verification, graceful shutdown, preStop hooks, restart behavior, PVC attachment

**StatefulSet**: Ordered startup/termination, PDB compliance, DNS resolution, node ID persistence

**PersistentVolume**: Binding, RWO enforcement, storage requests, retention, online expansion

**Health Probes**: Readiness/liveness/startup probes, endpoint removal, container restarts

**Resources**: CPU/memory limits, throttling, OOMKill, ResourceQuota

**PDB**: minAvailable enforcement, drain behavior, rolling updates

**Rollback**: Rolling updates, PDB blocking, version restore, consensus resumption

### Phase 2: Security (55 tests)

**Network Policies**: Default deny, namespace isolation, P2P/RPC paths, DNS egress, metrics scraping

**Linkerd mTLS**: Control plane, sidecar injection, encryption verification, cert rotation, SPIFFE

**Communication**: Intra/inter-namespace, headless service, load balancing, gRPC paths

**TLS**: cert-manager, validity/renewal, chain completeness, TLS 1.2+, cipher strength

**RBAC**: ServiceAccount assignments, secret isolation, read-only access, audit logging

**Pod Security**: runAsNonRoot, no privilege escalation, dropped capabilities, seccomp, no host access

**Secrets**: ConfigMap separation, Vault sync, refresh intervals, file mounting, permissions

**Container Scanning**: CVE scans, digest pinning, pull policies, registry trust, layer inspection

### Phase 3: Chaos Engineering (65 tests)

**Pod Failure**: Single/multi crash, rapid restarts, OOMKill, eviction, consensus during failures

**Node Failure**: Isolation, resource exhaustion, disk full, network down, multi-node, drain

**Network Partition**: Split brain, partial isolation, minority partition, toggle, asymmetric, recovery

**Latency**: 50ms-1000ms injection, jitter, timeout behavior, recovery

**Resource Stress**: CPU throttle/saturation, memory pressure, disk I/O, bandwidth limits

**Byzantine**: Invalid signatures, double-signing, state transitions, withholding, equivocation

**Split-Brain**: No quorum, minority isolation, phase-specific splits, extended duration, healing

**Consensus Recovery**: Timeout recovery, locked rounds, round advancement, rejoin, catch-up

**State Sync**: Snapshot recovery, abort/retry, partition during sync, integrity verification

### Phase 4: Performance (70 tests)

**TPS**: Baseline/peak at 100-1000 users, per-type, degradation curve, mixed workload, stability

**Block Latency**: Time consistency, confirmation time, mempool, finality, propagation, round trip

**P2P**: Connection establishment, RTT, bandwidth, peer discovery, gossip, mempool propagation

**API Load**: RPS, latency percentiles, error rates, pagination, concurrent connections, timeouts

**gRPC**: Streaming, unary latency, HTTP/2 multiplexing, serialization, concurrent streams

**Database**: Read/write latency, AppHash, commit, compaction impact, pruning, growth rate

**Memory**: Baseline, growth rate, leak detection, GC pauses, mempool consumption, release

**CPU**: Baseline/peak, per-tx ratio, consensus/validation cost, hotspots, contention

**HPA**: Scale-up trigger/speed/stability, load rebalancing, min/max replicas, custom metrics

**VPA**: Recommendations, bounds, accuracy, modes, right-sizing

**Storage I/O**: Sequential/random IOPS, latency, throughput, queue depth, fsync

### Phase 5: Monitoring (80 tests)

**Prometheus**: Pod health, scrape targets, Tendermint/Cosmos/DEX metrics, retention, labels

**Grafana**: Pod health, datasources, dashboard panels, refresh, variable interpolation

**Alerts**: Rule loading, BlockProductionStopped, LowPeerCount, HighBlockTime, CPU/disk alerts, routing

**Loki**: Pod health, queries, filtering, pattern matching, rate calculations, JSON parsing, retention

**Tracing**: Jaeger health, OTLP endpoints, transaction/block traces, sampling, context propagation

**Custom Metrics**: consensus_height, p2p_peers, tx counters, DEX/oracle metrics, accuracy

**SLA/SLO**: Block time, uptime, availability, latency, participation, connectivity, error rate

**Backup/Restore**: Key backup, snapshots, PVC backup, integrity, encryption, restore verification

**Disaster Recovery**: Single/multi failure, data loss, partition, downtime, storage outage, RTO/RPO

**Key Rotation**: Workflow, backup, priv_validator/node_key rotation, downtime-free, Vault sync

**Drift Detection**: Baseline docs, ConfigMap/Secret/image/resource monitoring, checksums, GitOps

### Phase 6: Integration (15 tests)

**Cross-Component**: Validator↔Node, API↔Validator, Prometheus↔targets, Grafana↔datasources, Vault, Ingress, Linkerd

**Transaction Flows**: Bank send, DEX swap, oracle submission, IBC packets, multi-hop, error handling

**Upgrades**: Rolling update, binary upgrade, state migration, rollback, consensus continuity

---

### Test Tracking

| Phase | Category | Tests |
|-------|----------|-------|
| 1 | Infrastructure | 45 |
| 2 | Security | 55 |
| 3 | Chaos/Resilience | 65 |
| 4 | Performance | 70 |
| 5 | Monitoring | 80 |
| 6 | Integration | 15 |
| **Total** | | **330** |

### Success Criteria

- **100%** Phase 1-2 (Infrastructure + Security)
- **95%** Phase 3 (Chaos/Resilience)
- **90%** Phase 4 (Performance)
- **100%** Phase 5-6 (Monitoring + Integration)
