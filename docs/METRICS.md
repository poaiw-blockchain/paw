# PAW Blockchain Metrics Documentation

**Version:** 1.0.0
**Last Updated:** 2025-12-07
**Audience:** Operators, SREs, DevOps Engineers

---

## Table of Contents

1. [Overview](#overview)
2. [Metrics Endpoints](#metrics-endpoints)
3. [Cosmos SDK Metrics](#cosmos-sdk-metrics)
4. [CometBFT/Tendermint Metrics](#cometbfttendermint-metrics)
5. [Custom PAW Metrics](#custom-paw-metrics)
6. [Prometheus Configuration](#prometheus-configuration)
7. [Grafana Dashboards](#grafana-dashboards)
8. [Alert Rules](#alert-rules)
9. [Troubleshooting](#troubleshooting)

---

## Overview

PAW blockchain exposes metrics in Prometheus format for monitoring node health, consensus performance, transaction throughput, and module-specific functionality. This document describes all available metrics, their meanings, and how to configure monitoring infrastructure.

### Metric Types

PAW uses four metric types:

- **Counter**: Monotonically increasing value (e.g., total transactions)
- **Gauge**: Current value that can go up or down (e.g., block height)
- **Histogram**: Distribution of values (e.g., transaction processing time)
- **Summary**: Similar to histogram with quantiles (e.g., request latency percentiles)

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ PAW Node                                                        │
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│  │ CometBFT         │  │ Cosmos SDK       │  │ Custom       │ │
│  │ Consensus        │  │ Framework        │  │ Modules      │ │
│  │                  │  │                  │  │              │ │
│  │ Port 26660       │  │ Port 1317        │  │ Port 26661   │ │
│  │ /metrics         │  │ /metrics         │  │ /metrics     │ │
│  └──────────────────┘  └──────────────────┘  └──────────────┘ │
│           │                     │                     │         │
└───────────┼─────────────────────┼─────────────────────┼─────────┘
            │                     │                     │
            └─────────────────────┴─────────────────────┘
                                  │
                          ┌───────▼────────┐
                          │  Prometheus    │
                          │  (Port 9090)   │
                          └───────┬────────┘
                                  │
                          ┌───────▼────────┐
                          │  Grafana       │
                          │  (Port 3000)   │
                          └────────────────┘
```

---

## Metrics Endpoints

### Endpoint Health Verification

PAW includes an automated metrics verification script that checks all Prometheus endpoints after node startup:

```bash
# Basic usage
./scripts/verify-metrics.sh

# Check specific node
./scripts/verify-metrics.sh --node-host validator-1.example.com

# Wait 30 seconds after startup before checking
./scripts/verify-metrics.sh --wait 30

# JSON output for automation
./scripts/verify-metrics.sh --json

# Verbose output with debugging
./scripts/verify-metrics.sh --verbose

# Custom ports
./scripts/verify-metrics.sh \
  --cometbft-port 26660 \
  --api-port 1317 \
  --app-port 26661 \
  --timeout 10
```

**Exit Codes:**
- `0` - All metrics endpoints accessible
- `1` - One or more endpoints failed
- `2` - Invalid arguments

**Startup Health Check:**
The PAW application automatically performs a telemetry health check during startup. Check logs for:
```
INFO OpenTelemetry tracing initialized jaeger_endpoint=http://localhost:4318
INFO Telemetry health check passed prometheus_enabled=true
```

If health check fails, look for:
```
ERROR Telemetry health check failed error="meter provider not initialized"
```

### Primary Endpoint: CometBFT Metrics

**URL:** `http://<node-ip>:26660/metrics`
**Protocol:** HTTP
**Format:** Prometheus text format

This endpoint exposes CometBFT consensus engine metrics and is the primary source for blockchain health monitoring.

**Enable in `config.toml`:**

```toml
#######################################################
###       Instrumentation Configuration Options     ###
#######################################################
[instrumentation]

# When true, Prometheus metrics are served under /metrics on
# PrometheusListenAddr.
# Check out the documentation for the list of available metrics.
prometheus = true

# Address to listen for Prometheus collector(s) connections
prometheus_listen_addr = ":26660"

# Maximum number of simultaneous connections.
# If you want to accept a larger number than the default, make sure
# you increase your OS limits.
# 0 - unlimited.
max_open_connections = 3

# Instrumentation namespace
namespace = "paw"
```

### Secondary Endpoint: Cosmos SDK API Metrics

**URL:** `http://<node-ip>:1317/metrics`
**Protocol:** HTTP
**Format:** Prometheus text format

Exposes REST API request metrics (if telemetry is enabled in `app.toml`).

**Enable in `app.toml`:**

```toml
###############################################################################
###                           Telemetry Configuration                       ###
###############################################################################

[telemetry]

# Prefixed with keys to separate services.
service-name = "paw"

# Enabled enables the application telemetry functionality. When enabled,
# an in-memory sink is also enabled by default. Operators may also enabled
# other sinks such as Prometheus.
enabled = true

# Enable prefixing gauge values with hostname.
enable-hostname = true

# Enable adding hostname to labels.
enable-hostname-label = true

# Enable adding service to labels.
enable-service-label = true

# PrometheusRetentionTime, when positive, enables a Prometheus metrics sink.
# It defines the metrics retention duration in seconds.
prometheus-retention-time = 600

# GlobalLabels defines a global set of name/value label tuples applied to all
# metrics emitted using the wrapper functions defined in telemetry package.
#
# Example:
# [["chain_id", "paw-testnet-1"], ["role", "validator"]]
global-labels = [
  ["chain_id", "paw-testnet-1"],
  ["network", "testnet"]
]
```

### Custom Application Metrics

**URL:** `http://<node-ip>:26661/metrics`
**Protocol:** HTTP
**Format:** Prometheus text format

Exposes custom PAW application metrics via OpenTelemetry exporter.

**Configuration:** Set in application code (see `app/telemetry.go`).

---

## Cosmos SDK Metrics

The Cosmos SDK exposes standard metrics for all blockchain applications. These metrics use the `cosmos_` prefix.

### Transaction Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `cosmos_tx_total` | Counter | Total number of transactions processed | `tx.type`, `tx.status` |
| `cosmos_tx_processing_time` | Histogram | Transaction processing time in milliseconds | `tx.type`, `tx.status` |
| `cosmos_tx_gas_used` | Histogram | Gas consumed by transaction | `tx.type`, `tx.status` |
| `cosmos_tx_size_bytes` | Histogram | Transaction size in bytes | `tx.type` |
| `cosmos_tx_failed_total` | Counter | Total failed transactions | `tx.type`, `error_type` |

**Labels:**
- `tx.type`: Message type (e.g., `cosmos.bank.v1beta1.MsgSend`, `paw.dex.v1.MsgSwap`)
- `tx.status`: `success` or `failed`
- `error_type`: Error category (e.g., `insufficient_funds`, `invalid_signature`)

**Example Query:**
```promql
# Transaction throughput (TPS)
rate(cosmos_tx_total[1m])

# Failed transaction rate
rate(cosmos_tx_failed_total[5m]) / rate(cosmos_tx_total[5m])

# Average transaction processing time
rate(cosmos_tx_processing_time_sum[5m]) / rate(cosmos_tx_processing_time_count[5m])
```

### Block Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `cosmos_block_height` | Gauge | Current block height | - |
| `cosmos_block_processing_time` | Histogram | Block processing time in milliseconds | - |
| `cosmos_block_size_bytes` | Gauge | Current block size in bytes | - |
| `cosmos_block_num_txs` | Gauge | Number of transactions in current block | - |
| `cosmos_block_interval_seconds` | Gauge | Time since last block in seconds | - |

**Example Query:**
```promql
# Blocks per minute
rate(cosmos_block_height[1m]) * 60

# Average block size
avg_over_time(cosmos_block_size_bytes[5m])

# Block time variance
stddev_over_time(cosmos_block_interval_seconds[10m])
```

### Module Execution Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `cosmos_module_execution_time` | Histogram | Module execution time in milliseconds | `module.name` |
| `cosmos_module_gas_used` | Counter | Gas consumed by module | `module.name` |
| `cosmos_module_errors_total` | Counter | Total module errors | `module.name`, `error_type` |

**Modules:**
- `bank`: Token transfers
- `staking`: Validator staking operations
- `distribution`: Reward distribution
- `gov`: Governance proposals
- `slashing`: Validator slashing
- `dex`: DEX operations (PAW custom)
- `oracle`: Price oracle (PAW custom)
- `compute`: Compute requests (PAW custom)

**Example Query:**
```promql
# Slowest modules
topk(5,
  rate(cosmos_module_execution_time_sum[5m]) /
  rate(cosmos_module_execution_time_count[5m])
)

# Gas usage by module
sum by (module_name) (rate(cosmos_module_gas_used[5m]))
```

### State Sync Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `cosmos_statesync_chunks_total` | Counter | Total state sync chunks processed | `status` |
| `cosmos_statesync_duration_seconds` | Gauge | State sync duration | - |
| `cosmos_statesync_snapshot_height` | Gauge | Snapshot height for state sync | - |

---

## CometBFT/Tendermint Metrics

CometBFT (formerly Tendermint) provides consensus and networking metrics with the `tendermint_` prefix (or custom namespace from config).

### Consensus Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tendermint_consensus_height` | Gauge | Consensus height | - |
| `tendermint_consensus_validators` | Gauge | Number of validators | - |
| `tendermint_consensus_validators_power` | Gauge | Total voting power | - |
| `tendermint_consensus_missing_validators` | Gauge | Number of missing validators | - |
| `tendermint_consensus_byzantine_validators` | Gauge | Number of byzantine validators | - |
| `tendermint_consensus_block_interval_seconds` | Histogram | Time between blocks | - |
| `tendermint_consensus_rounds` | Gauge | Current consensus round | - |
| `tendermint_consensus_num_txs` | Gauge | Transactions in current block | - |
| `tendermint_consensus_total_txs` | Counter | Total transactions processed | - |
| `tendermint_consensus_block_size_bytes` | Gauge | Current block size | - |
| `tendermint_consensus_step_duration` | Histogram | Duration of consensus steps | `step` |

**Steps:**
- `Propose`: Block proposal
- `Prevote`: Validator prevotes
- `Precommit`: Validator precommits
- `Commit`: Block commit

**Example Query:**
```promql
# Missed blocks (validator not participating)
increase(tendermint_consensus_missing_validators[10m])

# Consensus rounds (should be 0 for healthy network)
avg(tendermint_consensus_rounds) > 0

# Block time
histogram_quantile(0.99,
  rate(tendermint_consensus_block_interval_seconds_bucket[5m])
)
```

### P2P Network Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tendermint_p2p_peers` | Gauge | Number of connected peers | - |
| `tendermint_p2p_peer_receive_bytes_total` | Counter | Bytes received from peers | `peer_id`, `chID` |
| `tendermint_p2p_peer_send_bytes_total` | Counter | Bytes sent to peers | `peer_id`, `chID` |
| `tendermint_p2p_peer_pending_send_bytes` | Gauge | Pending bytes to send | `peer_id` |
| `tendermint_p2p_num_txs` | Gauge | Transactions in P2P message queue | `peer_id` |
| `tendermint_p2p_message_send_bytes_total` | Counter | Bytes sent by message type | `message_type` |
| `tendermint_p2p_message_receive_bytes_total` | Counter | Bytes received by message type | `message_type` |

**Channel IDs (chID):**
- `48`: Block sync
- `32`: Consensus
- `33`: Consensus data
- `34`: Consensus vote
- `35`: Consensus vote set bits
- `56`: Evidence
- `64`: Mempool
- `96`: State sync snapshot
- `97`: State sync chunk

**Example Query:**
```promql
# Network bandwidth by peer
topk(10, sum by (peer_id) (
  rate(tendermint_p2p_peer_send_bytes_total[5m]) +
  rate(tendermint_p2p_peer_receive_bytes_total[5m])
))

# Peer connection stability
changes(tendermint_p2p_peers[1h])
```

### Mempool Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tendermint_mempool_size` | Gauge | Number of transactions in mempool | - |
| `tendermint_mempool_tx_size_bytes` | Histogram | Transaction sizes in mempool | - |
| `tendermint_mempool_failed_txs` | Counter | Rejected transactions | - |
| `tendermint_mempool_recheck_times` | Counter | Mempool rechecks | - |

**Example Query:**
```promql
# Mempool congestion
avg_over_time(tendermint_mempool_size[10m]) > 1000

# Transaction rejection rate
rate(tendermint_mempool_failed_txs[5m])
```

### State Storage Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tendermint_state_block_processing_time` | Histogram | Block processing time | - |
| `tendermint_state_consensus_param_updates` | Counter | Consensus parameter updates | - |
| `tendermint_state_validator_set_updates` | Counter | Validator set updates | - |

---

## Custom PAW Metrics

PAW exposes module-specific metrics for DEX, Oracle, and Compute functionality.

### DEX Module Metrics

**Prefix:** `paw_dex_`

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `paw_dex_swap_count_total` | Counter | Total swap operations | `pool_id`, `status` |
| `paw_dex_swap_volume_total` | Counter | Total swap volume (in base currency) | `pool_id`, `token_in`, `token_out` |
| `paw_dex_swap_duration_seconds` | Histogram | Swap operation duration | `pool_id` |
| `paw_dex_swap_slippage_percent` | Histogram | Swap slippage percentage | `pool_id` |
| `paw_dex_pool_liquidity` | Gauge | Pool total value locked (TVL) | `pool_id`, `token` |
| `paw_dex_pool_reserves` | Gauge | Pool reserve amounts | `pool_id`, `token` |
| `paw_dex_pool_count` | Gauge | Total number of pools | - |
| `paw_dex_pool_fee_collected_total` | Counter | Total fees collected | `pool_id`, `token` |
| `paw_dex_limit_order_count` | Gauge | Open limit orders | `pool_id`, `order_type` |
| `paw_dex_limit_order_placed_total` | Counter | Total limit orders placed | `pool_id`, `order_type` |
| `paw_dex_limit_order_filled_total` | Counter | Total limit orders filled | `pool_id`, `order_type` |
| `paw_dex_limit_order_cancelled_total` | Counter | Total limit orders cancelled | `pool_id` |
| `paw_dex_limit_order_expired_total` | Counter | Total limit orders expired | `pool_id` |
| `paw_dex_liquidity_add_total` | Counter | Liquidity add operations | `pool_id` |
| `paw_dex_liquidity_remove_total` | Counter | Liquidity remove operations | `pool_id` |
| `paw_dex_price_impact_percent` | Histogram | Price impact of swaps | `pool_id` |

**Labels:**
- `pool_id`: Unique pool identifier
- `token_in`, `token_out`: Token denominations
- `status`: `success`, `failed`, `slippage_exceeded`
- `order_type`: `buy`, `sell`

**Example Query:**
```promql
# Swap throughput
sum(rate(paw_dex_swap_count_total{status="success"}[5m])) by (pool_id)

# Pool TVL
sum(paw_dex_pool_liquidity) by (pool_id)

# Average slippage
histogram_quantile(0.95,
  rate(paw_dex_swap_slippage_percent_bucket[10m])
)

# Limit order fill rate
sum(rate(paw_dex_limit_order_filled_total[1h])) /
sum(rate(paw_dex_limit_order_placed_total[1h]))
```

### Oracle Module Metrics

**Prefix:** `paw_oracle_`

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `paw_oracle_price_updates_total` | Counter | Total price updates | `pair`, `validator` |
| `paw_oracle_price_current` | Gauge | Current oracle price | `pair` |
| `paw_oracle_price_deviation` | Gauge | Price deviation from median | `pair`, `validator` |
| `paw_oracle_validator_participation` | Gauge | Validators submitting prices | `pair` |
| `paw_oracle_outlier_rejected_total` | Counter | Rejected outlier prices | `pair`, `validator` |
| `paw_oracle_staleness_seconds` | Gauge | Time since last price update | `pair` |
| `paw_oracle_aggregation_duration_ms` | Histogram | Price aggregation duration | `pair`, `method` |
| `paw_oracle_twap_price` | Gauge | Time-weighted average price | `pair`, `window` |
| `paw_oracle_volatility_percent` | Gauge | Price volatility (rolling stddev) | `pair`, `window` |
| `paw_oracle_confidence_percent` | Gauge | Price confidence level | `pair` |
| `paw_oracle_byzantine_detected_total` | Counter | Byzantine behavior detections | `validator` |
| `paw_oracle_circuit_breaker_triggered_total` | Counter | Circuit breaker activations | `pair`, `reason` |

**Labels:**
- `pair`: Trading pair (e.g., `BTC/USD`, `ETH/USD`)
- `validator`: Validator operator address
- `method`: Aggregation method (e.g., `median`, `vwap`, `twap`, `kalman`)
- `window`: Time window (e.g., `1h`, `24h`)
- `reason`: Circuit breaker reason (e.g., `high_volatility`, `low_participation`)

**Example Query:**
```promql
# Price update frequency
rate(paw_oracle_price_updates_total[5m])

# Validator participation rate
avg(paw_oracle_validator_participation) by (pair)

# Data staleness alert
max(paw_oracle_staleness_seconds) by (pair) > 300

# Outlier rate by validator
topk(5, sum by (validator) (
  rate(paw_oracle_outlier_rejected_total[1h])
))
```

### Compute Module Metrics

**Prefix:** `paw_compute_`

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `paw_compute_request_count_total` | Counter | Total compute requests | `provider`, `status` |
| `paw_compute_request_duration_seconds` | Histogram | Request processing time | `provider` |
| `paw_compute_verification_duration_ms` | Histogram | Proof verification time | `proof_type` |
| `paw_compute_verification_success_total` | Counter | Successful verifications | `proof_type` |
| `paw_compute_verification_failed_total` | Counter | Failed verifications | `proof_type`, `reason` |
| `paw_compute_provider_count` | Gauge | Active compute providers | `region` |
| `paw_compute_provider_capacity_percent` | Gauge | Provider capacity utilization | `provider` |
| `paw_compute_escrow_locked` | Gauge | Total escrowed tokens | `provider` |
| `paw_compute_rate_limit_exceeded_total` | Counter | Rate limit violations | `provider`, `requester` |
| `paw_compute_circuit_breaker_active` | Gauge | Circuit breaker state (0/1) | `provider` |
| `paw_compute_zk_proof_size_bytes` | Histogram | ZK proof size | `proof_type` |

**Labels:**
- `provider`: Compute provider address
- `status`: `pending`, `completed`, `failed`, `timeout`
- `proof_type`: `merkle`, `ed25519`, `zk_snark`
- `region`: Geographic region (e.g., `us-east`, `eu-west`)
- `reason`: Failure reason (e.g., `invalid_proof`, `timeout`, `nonce_reuse`)

**Example Query:**
```promql
# Compute request throughput
sum(rate(paw_compute_request_count_total{status="completed"}[5m]))

# Verification failure rate
sum(rate(paw_compute_verification_failed_total[10m])) /
sum(rate(paw_compute_verification_success_total[10m]) +
    rate(paw_compute_verification_failed_total[10m]))

# Provider capacity
avg(paw_compute_provider_capacity_percent) by (provider)
```

---

## Prometheus Configuration

### Basic Scrape Configuration

Create or update `/etc/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'paw-testnet'
    chain_id: 'paw-testnet-1'
    blockchain: 'paw'

# Scrape configurations
scrape_configs:
  # CometBFT consensus metrics (primary endpoint)
  - job_name: 'paw-consensus'
    static_configs:
      - targets: ['paw-node-1:26660', 'paw-node-2:26660', 'paw-node-3:26660']
        labels:
          component: 'consensus'
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 5s

  # Cosmos SDK API metrics
  - job_name: 'paw-api'
    static_configs:
      - targets: ['paw-node-1:1317', 'paw-node-2:1317', 'paw-node-3:1317']
        labels:
          component: 'api'
    metrics_path: '/metrics'
    scrape_interval: 15s

  # Custom application metrics
  - job_name: 'paw-app'
    static_configs:
      - targets: ['paw-node-1:26661', 'paw-node-2:26661', 'paw-node-3:26661']
        labels:
          component: 'application'
    metrics_path: '/metrics'
    scrape_interval: 10s

  # System metrics (node exporter)
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['paw-node-1:9100', 'paw-node-2:9100', 'paw-node-3:9100']
    scrape_interval: 15s
```

### Kubernetes Service Discovery

For Kubernetes deployments, use pod discovery:

```yaml
scrape_configs:
  - job_name: 'paw-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - paw-blockchain
    relabel_configs:
      # Scrape pods with prometheus.io/scrape=true annotation
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true

      # Use custom port if specified
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: (.+)
        replacement: ${1}:26660

      # Use custom metrics path if specified
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)

      # Add pod metadata as labels
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
      - source_labels: [__meta_kubernetes_pod_node_name]
        target_label: node
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
```

### Docker Compose Configuration

For development environments:

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:v2.48.0
    container_name: prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
      - '--web.enable-lifecycle'
    ports:
      - '9090:9090'
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - paw-network
    restart: unless-stopped

volumes:
  prometheus-data:

networks:
  paw-network:
    external: true
```

---

## Grafana Dashboards

Pre-built Grafana dashboards are available in `/infra/monitoring/dashboards/`:

### 1. Node Metrics Dashboard

**File:** `node-metrics.json`
**Description:** System-level metrics (CPU, memory, disk, network)

**Panels:**
- CPU usage by core
- Memory utilization
- Disk I/O and space
- Network traffic
- Go runtime metrics (goroutines, GC)
- Process metrics

**Import:** Dashboard ID TBD or import from file

### 2. Blockchain Metrics Dashboard

**File:** `blockchain-metrics.json`
**Description:** Consensus and blockchain health metrics

**Panels:**
- Block height and block time
- Transaction throughput (TPS)
- Validator participation
- Consensus rounds
- Mempool size
- P2P peer connections
- State storage size

**Import:** Dashboard ID TBD or import from file

### 3. DEX Metrics Dashboard

**File:** `dex-metrics.json`
**Description:** DEX-specific metrics

**Panels:**
- Swap volume and count
- Pool TVL by token
- Slippage distribution
- Limit order metrics
- Fee collection
- Liquidity changes
- Price impact distribution

**Import:** Dashboard ID TBD or import from file

### Dashboard Provisioning

Configure Grafana to auto-load dashboards:

**`/etc/grafana/provisioning/dashboards/dashboard.yml`:**

```yaml
apiVersion: 1

providers:
  - name: 'PAW Dashboards'
    orgId: 1
    folder: 'PAW Blockchain'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards/paw
      foldersFromFilesStructure: true
```

**Datasource Configuration:**

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
    jsonData:
      timeInterval: '15s'
      httpMethod: POST
```

---

## Alert Rules

### Consensus Alerts

**File:** `prometheus-rules/consensus-alerts.yml`

```yaml
groups:
  - name: consensus
    interval: 30s
    rules:
      - alert: NodeBehindChain
        expr: |
          increase(tendermint_consensus_height[5m]) == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Node {{ $labels.instance }} is not producing blocks"
          description: "Block height has not increased in 5 minutes"

      - alert: HighConsensusRounds
        expr: |
          avg(tendermint_consensus_rounds) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Consensus requiring multiple rounds"
          description: "Average rounds: {{ $value }}, indicates network issues"

      - alert: ValidatorMissing
        expr: |
          tendermint_consensus_missing_validators > 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Validators not participating in consensus"
          description: "{{ $value }} validators missing"

      - alert: ByzantineValidator
        expr: |
          increase(tendermint_consensus_byzantine_validators[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Byzantine behavior detected"
          description: "Byzantine validator detected on {{ $labels.instance }}"
```

### Performance Alerts

```yaml
  - name: performance
    interval: 1m
    rules:
      - alert: HighBlockTime
        expr: |
          histogram_quantile(0.95,
            rate(tendermint_consensus_block_interval_seconds_bucket[5m])
          ) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Block time is high"
          description: "95th percentile block time: {{ $value }}s"

      - alert: MempoolCongestion
        expr: |
          tendermint_mempool_size > 5000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Mempool congestion detected"
          description: "Mempool size: {{ $value }} transactions"

      - alert: LowPeerCount
        expr: |
          tendermint_p2p_peers < 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low peer count"
          description: "Only {{ $value }} peers connected"
```

### DEX Module Alerts

```yaml
  - name: dex
    interval: 1m
    rules:
      - alert: HighSwapSlippage
        expr: |
          histogram_quantile(0.95,
            rate(paw_dex_swap_slippage_percent_bucket[10m])
          ) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High swap slippage on pool {{ $labels.pool_id }}"
          description: "95th percentile slippage: {{ $value }}%"

      - alert: LowPoolLiquidity
        expr: |
          paw_dex_pool_liquidity < 10000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low liquidity in pool {{ $labels.pool_id }}"
          description: "TVL: {{ $value }}"

      - alert: HighOrderExpiration
        expr: |
          rate(paw_dex_limit_order_expired_total[1h]) > 10
        for: 15m
        labels:
          severity: info
        annotations:
          summary: "High limit order expiration rate"
          description: "{{ $value }} orders/sec expiring"
```

### Oracle Module Alerts

```yaml
  - name: oracle
    interval: 30s
    rules:
      - alert: StalePriceData
        expr: |
          paw_oracle_staleness_seconds > 300
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Stale price data for {{ $labels.pair }}"
          description: "Last update {{ $value }}s ago"

      - alert: LowValidatorParticipation
        expr: |
          paw_oracle_validator_participation < 7
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low validator participation for {{ $labels.pair }}"
          description: "Only {{ $value }} validators submitting prices"

      - alert: OracleCircuitBreaker
        expr: |
          increase(paw_oracle_circuit_breaker_triggered_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Oracle circuit breaker triggered for {{ $labels.pair }}"
          description: "Reason: {{ $labels.reason }}"

      - alert: HighOutlierRate
        expr: |
          sum by (validator) (rate(paw_oracle_outlier_rejected_total[1h])) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High outlier rate from validator {{ $labels.validator }}"
          description: "Outlier rate: {{ $value }}/sec"
```

---

## Troubleshooting

### Using the Verification Script

The quickest way to diagnose metrics issues is to use the built-in verification script:

```bash
# Run with verbose output to see detailed checks
./scripts/verify-metrics.sh --verbose

# Check a specific node
./scripts/verify-metrics.sh --node-host 192.168.1.100 --verbose

# Get machine-readable output
./scripts/verify-metrics.sh --json | jq .
```

**Common Script Output:**

Success:
```
======================================
  PAW Metrics Verification
======================================
Host: localhost
Timeout: 5s

[INFO] Checking CometBFT consensus metrics...
[✓] CometBFT - OK (147 metrics)
[INFO] Checking Cosmos SDK API metrics...
[✓] Cosmos SDK API - OK (89 metrics)
[INFO] Checking custom application metrics...
[✓] Application - OK (34 metrics)

======================================
  Summary
======================================
Total checks: 3
Passed: 3
Failed: 0

✓ All metrics endpoints are accessible
```

Failure:
```
[✗] CometBFT - Connection failed (curl exit code: 7)
[⚠] Application - Port 26661 not listening or not reachable

✗ Some metrics endpoints failed

Troubleshooting:
  1. Check if the node is running: systemctl status pawd
  2. Verify metrics are enabled in config
  3. Check firewall rules
  4. Review logs: journalctl -u pawd -n 100
```

### Metrics Not Available

**Symptom:** Prometheus cannot scrape metrics endpoint

**Automated Check:**
```bash
./scripts/verify-metrics.sh
# Exit code 0 = all OK, 1 = failures detected
```

**Manual Checks:**

1. Verify metrics are enabled in `config.toml`:
   ```bash
   grep -A5 "\[instrumentation\]" ~/.paw/config/config.toml
   # Should show: prometheus = true
   ```

2. Check port is listening:
   ```bash
   netstat -tuln | grep 26660
   # Should show: tcp6 0 0 :::26660 :::* LISTEN
   ```

3. Test endpoint manually:
   ```bash
   curl http://localhost:26660/metrics
   # Should return Prometheus-format metrics
   ```

4. Check firewall rules:
   ```bash
   sudo iptables -L -n | grep 26660
   # Should allow inbound connections from Prometheus
   ```

5. Check application logs for health check:
   ```bash
   journalctl -u pawd -n 100 | grep -i "health check"
   # Should show: Telemetry health check passed
   ```

### Missing Custom Metrics

**Symptom:** Cosmos SDK metrics available, but custom PAW metrics missing

**Checks:**

1. Verify telemetry is enabled in `app.toml`:
   ```bash
   grep -A10 "\[telemetry\]" ~/.paw/config/app.toml
   # Should show: enabled = true
   ```

2. Check application logs for telemetry errors:
   ```bash
   journalctl -u pawd -n 100 | grep -i telemetry
   ```

3. Verify OpenTelemetry exporter is configured (see `app/telemetry.go`)

### High Cardinality Issues

**Symptom:** Prometheus consuming excessive memory/storage

**Solution:**

1. Limit label cardinality - avoid high-cardinality labels like transaction hashes or addresses
2. Use recording rules to pre-aggregate high-resolution metrics
3. Configure Prometheus retention time:
   ```yaml
   storage.tsdb.retention.time: 30d
   storage.tsdb.retention.size: 50GB
   ```

### Metric Label Mismatches

**Symptom:** Queries return no results despite metrics being scraped

**Solution:**

1. Check label consistency:
   ```promql
   # List all label names for a metric
   label_names(tendermint_consensus_height)

   # List all label values
   label_values(job)
   ```

2. Verify relabeling configuration in Prometheus scrape config
3. Check for typos in label names (case-sensitive)

### Grafana Dashboard Not Loading

**Symptom:** Dashboard shows "No data" despite Prometheus having metrics

**Checks:**

1. Verify datasource connection in Grafana:
   - Settings → Data Sources → Prometheus → Test

2. Check time range - blockchain metrics may not exist for historical queries before chain start

3. Inspect browser console for errors (F12)

4. Test queries directly in Prometheus:
   ```
   http://prometheus:9090/graph
   ```

### Alert Not Firing

**Symptom:** Expected alert not triggering despite condition being met

**Checks:**

1. Verify rules are loaded:
   ```
   http://prometheus:9090/rules
   ```

2. Check rule evaluation:
   ```
   http://prometheus:9090/alerts
   ```

3. Verify Alertmanager configuration:
   ```bash
   amtool config show --alertmanager.url=http://localhost:9093
   ```

4. Check alert routing rules and silences

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [CometBFT Metrics](https://docs.cometbft.com/v0.38/core/metrics)
- [Cosmos SDK Telemetry](https://docs.cosmos.network/main/build/building-modules/telemetry)
- [OpenTelemetry](https://opentelemetry.io/docs/)

---

**For additional support, see:**
- [TROUBLESHOOTING.md](/docs/TROUBLESHOOTING.md)
- [VALIDATOR_OPERATOR_GUIDE.md](/docs/VALIDATOR_OPERATOR_GUIDE.md)
- [DEPLOYMENT_QUICKSTART.md](/docs/DEPLOYMENT_QUICKSTART.md)
