# PAW Testnet Prelaunch Runbook (Parallel-Safe, Extensive)

This runbook is designed to catch rookie errors before a public testnet launch. It is split into local (pre-testnet) tests and testnet validation. Parallel-safe steps can be run by separate agents in parallel across PAW roles.

## Goals

- Catch coding errors before public exposure
- Validate protocol behavior, APIs, and infrastructure
- Stress critical services without cross-agent interference

## Requirements

- bash, curl, jq
- `pawd` CLI for tx tests
- Optional: `k6` for load and WebSocket tests

## Setup

1) Copy the env template and edit values:

```bash
cp scripts/testnet/runbook/env.example scripts/testnet/runbook/.env
```

2) Edit `scripts/testnet/runbook/.env`:

- Set `RUN_ID` to a unique value per agent (e.g., `paw-agent1-YYYYMMDDHHMMSS`).
- Use unique `KEY_NAME`/`KEY_NAME_DST` per agent to avoid key collisions.
- If running from within VPN, you can swap RPC/REST to private endpoints.
- If archive node is available, set `ARCHIVE_RPC_URL`.

3) Load the env file:

```bash
set -a
. scripts/testnet/runbook/.env
set +a
```

## Phase 0: Local Pre-Testnet Gates (must pass before public testnet)

### 0.1 Advanced test suite (recommended)

PAW includes a full fuzz/property/chaos/benchmark/differential suite. Run these before public testnet:

```bash
cd tests
./run_all.sh
```

### 0.2 Chain tests

```bash
go test ./... -timeout 30m
```

## Phase 1: Parallel-Safe Testnet Suite

These steps are safe to run in parallel across multiple PAW agents. They are read-only or minimal-impact.

### 1.1 Smoke checks

```bash
bash scripts/testnet/runbook/smoke.sh
```

### 1.2 Height progression + RPC/REST consistency

```bash
bash scripts/testnet/runbook/height_watch.sh
```

### 1.3 Error contract checks (invalid input should fail safely)

```bash
bash scripts/testnet/runbook/error_contract.sh
```

### 1.4 Archive node validation (optional)

```bash
bash scripts/testnet/runbook/archive_check.sh
```

### 1.5 Read-only load (safe defaults)

```bash
k6 run scripts/testnet/runbook/k6_rpc_read.js
k6 run scripts/testnet/runbook/k6_rest_read.js
k6 run scripts/testnet/runbook/k6_graphql_read.js
k6 run scripts/testnet/runbook/k6_ws_subscribe.js
```

Optional tuning (still safe):

```bash
VUS=10 DURATION=3m k6 run scripts/testnet/runbook/k6_rpc_read.js
```

Baseline+peak wrapper (recommended):

```bash
bash scripts/testnet/runbook/run_k6_baseline_peak.sh
```

Summary output:

- `./out/<RUN_ID>/k6/summary.md`

Optional thresholds (override defaults):

```bash
P95_MS=500 P99_MS=2000 ERROR_RATE=0.001 k6 run scripts/testnet/runbook/k6_rpc_read.js
```

Calibration mode (prints suggested thresholds from observed p95/p99):

```bash
CALIBRATE=1 k6 run scripts/testnet/runbook/k6_rpc_read.js
```

Peak profile (higher load + relaxed thresholds):

```bash
PROFILE=peak k6 run scripts/testnet/runbook/k6_rpc_read.js
```

### 1.6 Minimal tx lifecycle (optional)

This uses the faucet and sends a minimal transaction. Use unique keys per agent.

```bash
bash scripts/testnet/runbook/tx.sh
```

## Phase 2: Coordinated-Only Testnet Suite (schedule these)

These tests can affect other agents and must be scheduled to avoid interference.

### 2.1 Module-specific lifecycle

- DEX: create pool, add/remove liquidity, swap
- Oracle: submit price updates, verify aggregation
- Compute: escrow workflow (create, verify, release)

### 2.2 Governance lifecycle

- Submit proposal, deposit, vote, verify tally

### 2.3 Failover drills

- Stop primary validator and validate liveness with secondary
- Stop SERVICES server and verify chain still produces blocks

### 2.4 High load / chaos

Use PAW's load tooling during a dedicated window:

```bash
# k6 load tests
k6 run tests/load/k6/blockchain-load-test.js
k6 run tests/load/k6/dex-swap-test.js
k6 run tests/load/k6/websocket-test.js

# tm-load-test (if configured)
# see tests/load/LOAD_TESTING.md
```

## Pass/Fail Criteria

- RPC/REST/GraphQL respond successfully; chain ID matches `CHAIN_ID`
- Peer count >= 1 via `/net_info`
- Heights increase or remain consistent, with low RPC/REST lag
- Invalid input returns client errors, not 200/500
- Optional tx test confirms inclusion via REST
- k6 read-only load stays within agreed p95 latency and error rate thresholds

## Output Artifacts

- Outputs are written to `${OUT_DIR}/${RUN_ID}`
- Attach results to the overall prelaunch report
