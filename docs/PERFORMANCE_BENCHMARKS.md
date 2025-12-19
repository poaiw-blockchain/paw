# PAW Performance Benchmarks

**Status:** Initial Baseline Complete – update this file after every benchmark run  
**Audience:** Core engineering, performance, and DevOps teams  
**Related Docs:** `docs/PERFORMANCE_TUNING.md`, `monitoring/README.md`, `LOG_AGGREGATION_GUIDE.md`

---

## 1. Purpose & Goals

This document defines the repeatable benchmarks that must be executed before every production or public testnet push. All measurements should be reproducible on a single validator node plus supporting RPC/API instances that match the resource profile in `docs/RESOURCE_REQUIREMENTS.md`.

### Key Production Targets

| Metric | Target | Notes |
| --- | --- | --- |
| Block time | 3s average (≤2.5s in private nets) | Defined in `config/config.toml` consensus params |
| Sustained throughput | 350 TPS mixed workload | 40% DEX, 40% bank, 20% oracle/compute |
| Mempool depth | <60% of `max_txs_bytes` | Prevents eviction under load |
| API latency | REST: <100ms p50 / <800ms p95, gRPC similar | Measure via vegeta |
| Oracle pipeline | <4s from feeder submission to vote commit | Includes aggregation |
| Compute jobs | <15s end-to-end for “standard” job | Post-proof verification |
| Resource headroom | CPU <70%, RAM <80%, disk IO wait <10% | Sustained for an hour |

Any deviation ≥10% from the baseline requires an investigation and a note in `COVERAGE_SUMMARY.md`.

---

## 2. Test Environment

1. **Hardware:** 8 vCPU, 32 GB RAM, NVMe (3k+ MB/s), Ubuntu 22.04.
2. **Software:** Go 1.22+, Docker 26+, Prometheus/Grafana/Loki stack from `monitoring/README.md`.
3. **Chain State:** Use `networks/paw-testnet-1/` genesis with seeded validators. For development replay, run `./pawd start` with snapshot height ≥300k.
4. **Logging:** Enable structured JSON logging so Loki queries can correlate latency spikes (`LOG_FORMAT=json`).
5. **Monitoring:** Import dashboards from `monitoring/grafana/dashboards/*` and pin the Health + Log Aggregation dashboards before executing high-load tests.

---

## 3. Benchmark Matrix

| Scenario | Description | Tooling | Acceptance Criteria |
| --- | --- | --- | --- |
| **Throughput / TPS** | Measures block production speed under mixed transactions | `scripts/testing/track_benchmarks.sh`, load generator (`tests/benchmarks/`) | ≥350 TPS sustained for 10 minutes, <5% failed tx |
| **Block Propagation** | Verifies consensus timings + peer gossip | `tests/benchmarks/block_bench_test.go`, Prometheus `tendermint_consensus_height` deltas | p95 block commit <1.2s, Max peer latency <300ms |
| **DEX Workload** | Stresses pools + matching engine | `tests/benchmarks/dex_*`, `simapp` orderflow script | Swap execution latency <600ms average, max gas per block <85% |
| **Oracle Voting** | Price feeder to vote commit | `tests/benchmarks/oracle_*`, `scripts/oracle/load_test.sh` | End-to-end <4s, missed vote <1% |
| **Compute Jobs** | Measures queue + verification pipeline | `tests/benchmarks/compute_*`, synthetic WASM workload | 95% jobs finish <15s, escrow refunds <2s |
| **API / Query** | REST + gRPC latency with concurrent clients | Vegeta (`scripts/testing/vegeta_profiles/*.toml`) | REST/gRPC P95 <800ms, error rate <0.1% |
| **Node Sync & Catch-up** | Snapshot restore and sync speed | `scripts/testing/sync_bench.sh` | 500+ blocks/s initial catch-up, steady <5% CPU steal |

---

## 4. Execution Workflow

1. **Build + Reset**
   ```bash
   make build
   ./pawd tendermint unsafe-reset-all
   ```
   Bring up multi-node testnet via `networks/paw-testnet-1/localnet.sh` if a distributed scenario is required.

2. **Start Monitoring Stack**
   ```bash
   docker compose -f monitoring/docker-compose.yml up -d
   docker compose -f compose/docker-compose.logging.yml up -d
   ```

3. **Warm-Up Run**
   - Replay 100 blocks of historical transactions (`scripts/testing/replay_blocks.sh`).
   - Confirm health endpoints respond with `scripts/health-check-all.sh`.

4. **Benchmark Harness**
   ```bash
   ./scripts/testing/track_benchmarks.sh
   ```
   This runs every Go benchmark under `tests/benchmarks/...`, saves raw output to `test-results/benchmarks/bench_<timestamp>.txt`, and publishes JSON summaries under `test-results/benchmarks/history/`. The script auto-compares against the previous run and reports regressions if `ns/op` slowed by >10%.

5. **Scenario-Specific Load**
   - **Throughput:** Run `tests/benchmarks/loadgen/tx_spammer.go` with `--tps 400`.
   - **DEX:** `go test ./tests/benchmarks -run Dex -bench DexSwapBenchmark`.
   - **API:** `vegeta attack -targets scripts/testing/vegeta_profiles/rest_high_load.txt -duration=5m | vegeta report`.
   - **Oracle:** `scripts/oracle/load_test.sh --feeds 12 --validators 5`.
   - **Compute:** `scripts/compute/run_job_batch.sh --jobs 50 --providers 4`.

6. **Capture Metrics**
   - Export Grafana dashboards (Health, Log Aggregation, Advanced Modules) as JSON and store under `test-results/benchmarks/grafana/<timestamp>/`.
   - Record Prometheus snapshots via `curl localhost:9090/api/v1/query_range`.
   - Save Loki queries for errors/timeouts to `test-results/benchmarks/loki/<timestamp>.log`.

7. **Report & Gate**
   - Update `test-results/benchmarks/benchmark_summary.md` (template below).
   - If regressions detected, open an item in `REMAINING_TESTS.md` with component + owner.
   - Block release until regressions have root cause or waiver.

---

## 5. Benchmark Summary Template

Copy this block into `test-results/benchmarks/benchmark_summary.md` for every run:

```
## Benchmark Run – <DATE> UTC
- Commit: <git sha>
- Environment: <hardware + OS>
- Chain Height During Test: <height>

| Scenario | Result | Target | Pass/Fail | Notes |
| --- | --- | --- | --- | --- |
| Throughput | 372 TPS sustained (5% failed) | ≥350 TPS / <5% fail | ✅ | Slight latency spike ~min 8 |
| Block Propagation | 0.98s p95 | <1.2s | ✅ | --- |
| DEX Swap Latency | 540ms avg | <600ms | ✅ | Pools: atom/osmo, paw/usdc |
| Oracle E2E | 3.4s avg | <4s | ✅ | Missed vote 0.3% |
| Compute Jobs | 14.2s p95 | <15s | ✅ | Job type: WASM standard |
| REST/gRPC | 620ms p95 | <800ms | ✅ | vegeta 500 rps |
| Sync Speed | 520 blocks/s | ≥500 blocks/s | ✅ | Snapshot height 300k |

### Observations
- <Add callouts on CPU/RAM headroom, mempool depth, etc.>
### Actions
- <List follow-ups> 
```

---

## 6. Troubleshooting & Tips

- **Benchmark noise:** Disable other heavy workloads, pin validators to dedicated cores, and flush disk caches before runs (`sync; echo 3 | sudo tee /proc/sys/vm/drop_caches`).
- **Prometheus gaps:** Ensure scrape interval ≤5s for short tests; gaps invalidate p95 calculations.
- **Loki ingestion:** When pushing 5k+ log lines/sec, bump `ingester.max-transfer-retries` in `compose/docker/logging/loki-config.yml`.
- **Vegeta TLS errors:** Some services expose self-signed certs; append `-insecure` to the vegeta attack command when pointing at HTTPS endpoints.
- **Compute job variance:** Each provider should have `COMPUTE_MAX_CONCURRENCY` ≥5; otherwise queue latency dominates and fails the SLA.

---

## 7. Next Steps

1. Automate nightly benchmark runs in CI with artifact upload to S3/GCS.
2. Extend `scripts/testing/track_benchmarks.sh` to parse vegeta and loadgen outputs for unified regression reporting.
3. Hook Prometheus queries into the tracker (block time, mempool usage, CPU) for single-pane dashboards.

Keeping this document updated is now part of the production-readiness gate tracked in `ROADMAP_PRODUCTION.md`. Update the roadmap checkbox once the first benchmark report is filed.
