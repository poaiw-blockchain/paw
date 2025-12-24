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

---

## Public Testnet Readiness Review (December 2025)

> Multi-agent comprehensive review for public testnet launch and community engagement.

### 🔴 CRITICAL (P1) - Must Fix Before Public Testnet

#### Security

- [ ] **P1-SEC-1**: Implement IBC nonce cleanup/pruning (state growth unbounded)
  - Location: `x/shared/nonce/manager.go`, `x/oracle/ibc_module.go:219-228`
  - Add time-based nonce expiration (7-day TTL) and pruning in EndBlock
  - Risk: Disk space exhaustion, DoS vector

- [ ] **P1-SEC-2**: Validate ZK circuit parameters with hash verification
  - Location: `x/compute/keeper/keeper.go:246-288`
  - Pin circuit parameters in module params (governance-controlled)
  - Store expected circuit hash, verify on initialization
  - Risk: Acceptance of invalid compute results

- [ ] **P1-SEC-3**: Add integer overflow formal verification for AMM calculations
  - Location: `x/dex/keeper/swap.go`, `x/dex/keeper/pool.go`
  - Add explicit overflow checks in multi-step calculations
  - Implement fuzz testing for extreme value combinations

#### Data Integrity

- [ ] **P1-DATA-1**: Fix liquidity share validation mismatch (genesis vs invariant)
  - Location: `x/dex/keeper/genesis.go:105-117`
  - Genesis uses strict equality, invariant allows 10% variance
  - Will fail chain export/import with fee-accumulated pools
  - Align tolerance or enforce fee withdrawal before export

- [ ] **P1-DATA-2**: Make escrow timeout index creation atomic
  - Location: `x/compute/keeper/escrow.go:89-92`, `genesis.go:159-165`
  - Currently treats index creation failure as "non-critical"
  - Escrowed funds can be locked permanently without timeout
  - Implement rollback or add recovery mechanism

- [ ] **P1-DATA-3**: Implement two-phase commit for catastrophic failure scenarios
  - Location: `x/compute/keeper/escrow.go:83-84, 212-213, 287-288`
  - Bank transfers can succeed while state updates fail
  - Creates inconsistent state between modules
  - Add invariant for module balance vs escrow state reconciliation

#### Performance

- [ ] **P1-PERF-1**: Fix DEX GetOrdersByOwnerPaginated double iteration
  - Location: `x/dex/keeper/limit_orders.go:803-807`
  - Counts total before pagination = O(2n) iteration
  - 10,000 orders + page 1 request = 10,100 operations
  - Remove total count or estimate from page progression

- [ ] **P1-PERF-2**: Cap oracle volatility snapshot iteration (unbounded)
  - Location: `x/oracle/keeper/aggregation.go:464-469`
  - TWAP limited to 1000 snapshots (line 774), volatility is not
  - Bypasses line 774 safeguards, DoS vector via state bloat
  - Apply same 1000-snapshot limit

---

### 🟡 HIGH PRIORITY (P2) - Fix Before Mainnet

#### Security

- [ ] **P2-SEC-1**: Document MEV risks and implement commit-reveal scheme
  - Location: `x/dex/keeper/swap_secure.go`, `types/messages.go:184-186`
  - Deadline/MinAmountOut help but no encrypted mempool
  - For testnet: Document risks, recommend tight slippage
  - For mainnet: Implement commit-reveal or threshold encryption

- [ ] **P2-SEC-2**: Enable geographic diversity enforcement at runtime
  - Location: `x/oracle/keeper/genesis.go:20-50`
  - Only validated at genesis, not when validators join/change regions
  - Add check to `MsgRegisterOracle` handler
  - Monitor in BeginBlocker, emit warnings when diversity drops

- [ ] **P2-SEC-3**: Add emergency pause mechanism to Oracle module
  - Location: `x/oracle/keeper/` (missing)
  - DEX has circuit breakers, Oracle lacks equivalent
  - Add governance-controlled emergency halt for price feeds

#### Data Integrity

- [ ] **P2-DATA-1**: Implement store key namespace separation
  - Location: `x/compute/keeper/keys.go`, `x/dex/keeper/keys.go`, `x/oracle/keeper/keys.go`
  - All modules use overlapping prefixes (0x01-0x25)
  - IBCPacketNonceKeyPrefix = 0x0D in all three modules
  - Prefix keys with module-specific byte

- [ ] **P2-DATA-2**: Preserve circuit breaker state across chain upgrades
  - Location: `x/dex/keeper/genesis.go:148-169`
  - Currently resets pause state to zero on export
  - Paused pools become immediately unpaused after upgrade
  - Add `UpgradePreserveCircuitBreakerState` parameter

#### Performance

- [ ] **P2-PERF-1**: Implement GeoIP result caching with TTL
  - Location: `x/oracle/keeper/keeper.go:54-58`
  - No caching for GeoIP lookups (file I/O every validation)
  - 100 validators = 100 lookups per aggregation
  - Add in-memory LRU cache with TTL

- [ ] **P2-PERF-2**: Cache provider reputation for compute requests
  - Location: `x/compute/keeper/request.go:64`
  - Linear O(n) iteration through providers per request
  - 100 providers: 3000 gas becomes 50,000 gas
  - Cache top 10 providers, refresh every N blocks

---

### 🔵 MEDIUM PRIORITY (P3) - Post-Testnet Polish

#### Code Simplification (~6,500 LOC reduction potential)

- [ ] **P3-SIMP-1**: Remove duplicate swap/pool/liquidity implementations
  - Delete: `x/dex/keeper/swap.go`, `pool.go`, `liquidity.go`
  - Keep: `*_secure.go` variants (rename to remove suffix)
  - Impact: ~800 LOC, eliminates dual-maintenance

- [ ] **P3-SIMP-2**: Simplify circuit manager abstraction
  - Location: `x/compute/keeper/circuit_manager.go` (648 lines)
  - Over-engineered for 3 fixed circuits
  - Use simple package-level variables instead
  - Impact: ~200 LOC

- [ ] **P3-SIMP-3**: Remove IBC channel authorization wrapper duplication
  - Location: `x/*/keeper/keeper.go` (255 LOC total)
  - All three modules have identical thin wrappers
  - Embed ibcutil.ChannelStore interface directly
  - Impact: ~200 LOC

- [ ] **P3-SIMP-4**: Remove compute keeper channel cache (premature optimization)
  - Location: `x/compute/keeper/keeper.go:46-49, 140-146, 177-215`
  - DEX/Oracle work fine without cache
  - 50 LOC of double-checked locking complexity
  - Direct GetParams() call is sufficient

- [ ] **P3-SIMP-5**: Consolidate fragmented test files
  - Location: `x/compute/keeper/` (72 test files, 14,675 lines)
  - Many `_extended`, `_cover`, `_helpers` suffixes
  - Merge related tests (all IBC tests → ibc_test.go)
  - Impact: ~3,000 LOC, improved discoverability

- [ ] **P3-SIMP-6**: Delete incomplete oracle advanced features
  - Location: `x/oracle/keeper/oracle_advanced.go`
  - Contains stubbed signature verification ("TODO: implement")
  - YAGNI violation
  - Impact: ~300 LOC

- [ ] **P3-SIMP-7**: Remove dual ZK verification systems
  - `x/compute/keeper/zk_verification.go` (600+ lines)
  - `x/compute/keeper/circuit_manager.go` (648 lines)
  - Keep circuit_manager (more complete), delete zk_verification
  - Impact: ~400 LOC

#### Performance Optimizations

- [ ] **P3-PERF-1**: Use CacheContext for atomic swap operations
  - Location: `x/dex/keeper/swap.go:174-183`
  - Manual reversion pattern uses ~4000 gas
  - CacheContext provides automatic rollback
  - Reduces error handling complexity

- [ ] **P3-PERF-2**: Add gas metering to paginated queries
  - Location: `x/dex/keeper/query_server.go:69-79`
  - Queries are free regardless of limit
  - Add proportional gas charge: `limit * 100 gas`

- [ ] **P3-PERF-3**: Pre-size memory allocations with capacity hints
  - Location: `x/oracle/keeper/aggregation.go:778` and others
  - `append()` without capacity hint causes reallocations
  - Use `make([]T, 0, expectedSize)`

#### Security Hardening

- [ ] **P3-SEC-1**: Add compute request rate limiting
  - Location: `x/compute/keeper/msg_server.go:107-140`
  - DEX has spam prevention, compute does not
  - Add: 100 requests/address/day, 10-block cooldown

- [ ] **P3-SEC-2**: Increase oracle slash fraction for mainnet
  - Location: `x/oracle/types/params.go` (default 1%)
  - Too lenient: 1% loss acceptable for >1% manipulation gain
  - Mainnet: Increase to 5-10%, add progressive slashing

- [ ] **P3-SEC-3**: Add finalization flag check to request status invariant
  - Location: `x/compute/keeper/invariants.go:186-251`
  - Completed requests should have finalization flag set
  - Prevents double-settlement edge cases

---

### Repository Organization for Public Launch

#### Documentation

- [ ] Create `docs/ARCHITECTURE.md` - High-level system design overview
- [ ] Create `docs/STATE.md` - Document store key allocation strategy
- [ ] Create `docs/SECURITY_MODEL.md` - Security assumptions and threat model
- [ ] Update README.md with testnet participation instructions
- [ ] Add `GOVERNANCE.md` - Community governance process
- [ ] Create module-level README files for x/compute, x/dex, x/oracle

#### GitHub Configuration

- [x] CODEOWNERS file with team assignments ✓
- [x] Issue templates (bug, feature) ✓
- [x] Pull request template ✓
- [x] CI workflows (build, test, coverage, security) ✓
- [ ] Add branch protection rules for `main`
- [ ] Configure dependabot for dependency updates
- [ ] Add DCO (Developer Certificate of Origin) requirement
- [ ] Create FUNDING.yml for sponsorship

#### Community Readiness

- [ ] Create Discord/Telegram community channels
- [ ] Prepare testnet faucet with rate limiting
- [ ] Create validator onboarding documentation
- [ ] Prepare genesis ceremony documentation
- [ ] Create block explorer public instance
- [ ] Prepare network status dashboard

---

### Test Coverage Gaps

> Add tests for identified edge cases

- [ ] Export→Import with fee-accumulated pools (DEX)
- [ ] Migration with orphaned escrow timeout indexes (Compute)
- [ ] Catastrophic failure state recovery (Compute)
- [ ] Pool ID collision after migration (DEX)
- [ ] Geographic diversity violation at runtime (Oracle)
- [ ] Circuit breaker state preservation across upgrade (DEX)
- [ ] Concurrent escrow lock attempts
- [ ] Extreme value edge cases (overflow scenarios)
- [ ] Byzantine fault injection scenarios
- [ ] Network partition simulation

---

### Review Summary

**Review Date:** December 24, 2025
**Agents Used:** Security Sentinel, Architecture Strategist, Performance Oracle, Code Simplicity Reviewer, Data Integrity Guardian

**Overall Assessment:**
- **Security:** B+ (Strong foundation, needs P1 fixes)
- **Performance:** B+ (Well-optimized with minor issues)
- **Data Integrity:** B (Good patterns, critical edge cases to fix)
- **Code Quality:** B+ (Some simplification opportunities)
- **Documentation:** B (Comprehensive but needs organization)
- **Test Coverage:** A- (815+ tests, need edge case coverage)

**Verdict:** READY FOR PUBLIC TESTNET after P1 items addressed (estimated 3-5 days).
MAINNET READY after P2 items + external audit.
