# PAW Blockchain Metrics Configuration

## Overview

This document describes the Prometheus metrics configuration for the PAW blockchain network, including all available metrics, how to enable/disable them, and integration with Grafana.

## Metrics Architecture

PAW uses a multi-layer metrics architecture:

1. **CometBFT/Tendermint Metrics** - Consensus layer metrics (port 26660)
2. **Cosmos SDK Telemetry** - Application layer metrics (in-memory sink)
3. **Custom Module Metrics** - DEX, Oracle, and Compute module metrics (OpenTelemetry)
4. **Infrastructure Metrics** - Node Exporter, cAdvisor

## Current Status

### ✅ Working Metrics (31/51 targets UP)

#### CometBFT Consensus Metrics (Port 26660)
All 4 validator nodes are exposing CometBFT metrics successfully:

- **Endpoint**: `http://paw-node[1-4]:26660/metrics`
- **Status**: ✅ UP (16/16 targets)
- **Scrape Interval**: 10-30s depending on target

**Key Metrics Available**:
- `cometbft_consensus_height` - Current blockchain height
- `cometbft_consensus_validators` - Number of validators
- `cometbft_consensus_validator_power` - Validator voting power
- `cometbft_consensus_missing_validators` - Missing validators count
- `cometbft_consensus_byzantine_validators` - Byzantine validators detected
- `cometbft_consensus_block_interval_seconds` - Block time interval
- `cometbft_consensus_rounds` - Consensus rounds per block
- `cometbft_consensus_num_txs` - Transactions per block
- `cometbft_mempool_size` - Mempool size
- `cometbft_mempool_tx_size_bytes` - Transaction sizes in mempool
- `cometbft_p2p_peers` - Connected peers
- `cometbft_p2p_peer_receive_bytes_total` - Bytes received from peers
- `cometbft_p2p_peer_send_bytes_total` - Bytes sent to peers
- `cometbft_state_block_processing_time` - Block processing time
- `cometbft_abci_connection_method_timing_seconds` - ABCI method timings

#### Infrastructure Metrics
- **Node Exporter**: ✅ UP - System metrics (CPU, memory, disk, network)
- **cAdvisor**: ✅ UP - Container metrics
- **Prometheus**: ✅ UP - Self-monitoring

### ⚠️ Partially Working Metrics

#### Cosmos SDK Telemetry (In-Memory Sink Only)
The Cosmos SDK telemetry is **enabled** and collecting metrics, but it uses an in-memory sink rather than serving them via HTTP endpoint.

- **Configuration**: Enabled in `app.toml`
- **Retention Time**: 600 seconds
- **Status**: ⚠️ Collecting but not exposed via HTTP
- **Ports 9090-9093**: Exposed but not serving Prometheus metrics

**To access these metrics**, you would need to:
1. Query via Cosmos SDK gRPC interface
2. Implement a custom Prometheus exporter
3. Use the OpenTelemetry integration (if configured)

### ❌ Not Available via HTTP

#### API Endpoint (port 1317)
- **Status**: ❌ Returns JSON instead of Prometheus format
- **Reason**: Cosmos REST API doesn't serve Prometheus metrics by default
- **Workaround**: Use CometBFT metrics for blockchain stats

#### Module-Specific Metrics
Custom modules (DEX, Oracle, Compute) have metrics implementations but require:
1. HTTP server to expose them
2. OpenTelemetry collector integration
3. Or integration with Cosmos SDK telemetry sink

## Prometheus Target Configuration

### All 15 Blockchain Targets

1. **paw-tendermint** (4 endpoints) - ✅ UP
   - CometBFT consensus metrics
   - Scrape interval: 10s

2. **paw-app** (4 endpoints) - ⚠️ DOWN
   - Cosmos SDK application metrics
   - Requires HTTP metrics server

3. **paw-api** (1 endpoint) - ❌ DOWN
   - REST API metrics
   - Not available in Prometheus format

4. **paw-dex** (4 endpoints) - ⚠️ DOWN
   - DEX module metrics
   - Filtered: `paw_dex_.*`

5. **paw-validators** (4 endpoints) - ✅ UP (via Tendermint)
   - Validator-specific metrics
   - Filtered: `cometbft_consensus_validators|paw_validator_.*`

6. **paw-oracle** (4 endpoints) - ⚠️ DOWN
   - Oracle module metrics
   - Filtered: `paw_oracle_.*`

7. **paw-compute** (4 endpoints) - ⚠️ DOWN
   - Compute module metrics
   - Filtered: `paw_compute_.*`

8. **paw-ibc** (4 endpoints) - ✅ UP (via Tendermint)
   - IBC metrics
   - Filtered: `ibc_.*|cometbft_p2p_.*`

9. **paw-staking** (4 endpoints) - ✅ UP (via Tendermint)
   - Staking module metrics
   - Filtered: `cometbft_consensus_.*|cometbft_state_.*`

10. **paw-governance** (1 endpoint) - ⚠️ DOWN
    - Governance module metrics

11. **paw-bank** (4 endpoints) - ✅ UP (via Tendermint)
    - Bank module metrics

12. **paw-distribution** (1 endpoint) - ✅ UP (via Tendermint)
    - Distribution module metrics

13. **paw-slashing** (4 endpoints) - ✅ UP (via Tendermint)
    - Slashing module metrics

14. **paw-evidence** (4 endpoints) - ✅ UP (via Tendermint)
    - Evidence module metrics

15. **paw-upgrade** (1 endpoint) - ⚠️ DOWN
    - Upgrade module metrics

## Configuration Files

### Node Configuration

**CometBFT Metrics** (`config/config.toml`):
```toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
```

**Cosmos SDK Telemetry** (`config/app.toml`):
```toml
[telemetry]
service-name = "paw"
enabled = true
enable-hostname = true
enable-hostname-label = true
enable-service-label = true
prometheus-retention-time = 600
global-labels = []
```

### Prometheus Configuration

Location: `compose/docker/monitoring/prometheus.yml`

**Scrape Intervals**:
- Consensus metrics: 10s
- Module metrics: 10-15s
- Validators: 30s
- Governance/Upgrade: 60s

## Enabling/Disabling Metrics

### Enable Metrics on All Nodes

```bash
# Run the enable script
./scripts/enable-node-metrics.sh

# Restart nodes
docker compose -f compose/docker-compose.4nodes.yml restart
```

### Disable Metrics

Edit each node's configuration:

```bash
# In config/config.toml
prometheus = false

# In config/app.toml
enabled = false
prometheus-retention-time = 0
```

Then restart the nodes.

### Per-Module Configuration

Metrics collection happens automatically when modules are active. To reduce overhead:

1. Use metric relabeling in Prometheus to filter
2. Adjust scrape intervals for less critical metrics
3. Disable unused exporters

## Accessing Metrics

### Direct Access

**CometBFT Metrics**:
```bash
# From inside node container
curl http://localhost:26660/metrics

# From host (node1 example)
curl http://paw-node1:26660/metrics
```

**Via Prometheus**:
```bash
# Query specific metric
curl 'http://localhost:9091/api/v1/query?query=cometbft_consensus_height'

# Query with labels
curl 'http://localhost:9091/api/v1/query?query=cometbft_consensus_height{job="paw-tendermint"}'
```

### Prometheus UI

Access at: **http://localhost:9091**

**Useful Queries**:

```promql
# Current block height
cometbft_consensus_height

# Block time (avg over 5m)
rate(cometbft_consensus_block_interval_seconds_sum[5m]) /
rate(cometbft_consensus_block_interval_seconds_count[5m])

# Transactions per second
rate(cometbft_consensus_num_txs[1m])

# P2P bandwidth in
rate(cometbft_p2p_peer_receive_bytes_total[5m])

# P2P bandwidth out
rate(cometbft_p2p_peer_send_bytes_total[5m])

# Validator count
cometbft_consensus_validators

# Missing validators
cometbft_consensus_missing_validators

# Block processing time (95th percentile)
histogram_quantile(0.95,
  rate(cometbft_state_block_processing_time_bucket[5m]))

# ABCI commit time (95th percentile)
histogram_quantile(0.95,
  rate(cometbft_abci_connection_method_timing_seconds_bucket{method="commit"}[5m]))
```

### Grafana Integration

Access at: **http://localhost:11030**
- Username: `admin`
- Password: `grafana_secure_password`

Prometheus is pre-configured as a datasource.

**Creating Dashboards**:

1. Go to Dashboards → New Dashboard
2. Add Panel
3. Use PromQL queries from above
4. Visualize as:
   - Time series (for trends)
   - Gauge (for current values)
   - Stat (for single numbers)
   - Table (for multiple values)

**Recommended Panels**:
- Block Height (stat + time series)
- Block Time (gauge + time series)
- TPS (gauge + time series)
- Validators (stat)
- P2P Peers (stat)
- Mempool Size (gauge)
- Network Bandwidth (time series)

## Troubleshooting

### Metrics Not Appearing

1. **Check node configuration**:
   ```bash
   docker exec paw-node1 grep prometheus /root/.paw/node1/config/config.toml
   ```

2. **Verify endpoint is accessible**:
   ```bash
   docker exec paw-node1 wget -O- http://localhost:26660/metrics
   ```

3. **Check Prometheus targets**:
   ```bash
   curl http://localhost:9091/api/v1/targets | jq '.data.activeTargets[] | select(.health=="down")'
   ```

4. **Check Prometheus logs**:
   ```bash
   docker logs paw-prometheus
   ```

### Network Connectivity Issues

If targets show "server misbehaving" or DNS errors:

```bash
# Connect Prometheus to the pawnet network
docker network connect compose_pawnet paw-prometheus

# Restart Prometheus
docker compose -f compose/docker-compose.monitoring.yml restart prometheus
```

### High Cardinality Warnings

If you see "too many time series" warnings:

1. Reduce scrape frequency
2. Use metric relabeling to drop unnecessary labels
3. Increase Prometheus retention limits

## Verification

Run the verification script:

```bash
./scripts/verify-node-metrics.sh
```

This checks:
- Prometheus service status
- Node configuration
- Metrics endpoint accessibility
- Prometheus target status
- Key metrics availability
- Infrastructure metrics

## Future Improvements

### Short Term
1. ✅ Enable CometBFT metrics (DONE)
2. ✅ Configure Prometheus scraping (DONE)
3. ✅ Network connectivity between containers (DONE)
4. ⏳ Create Grafana dashboards
5. ⏳ Set up alerting rules

### Medium Term
1. Implement HTTP metrics server for Cosmos SDK telemetry
2. Export custom module metrics (DEX, Oracle, Compute)
3. Add OpenTelemetry collector for distributed tracing
4. Create metrics aggregation for multi-node stats

### Long Term
1. Implement custom exporters for application-specific metrics
2. Add business logic metrics (swap volumes, oracle prices, etc.)
3. Integrate with external monitoring (Datadog, New Relic, etc.)
4. Implement metrics-based auto-scaling

## References

- [CometBFT Metrics](https://docs.cometbft.com/v0.38/core/metrics)
- [Cosmos SDK Telemetry](https://docs.cosmos.network/main/learn/advanced/telemetry)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)

## Appendix: Complete Metrics List

### CometBFT Consensus Metrics

- `cometbft_consensus_height` - Current block height
- `cometbft_consensus_validators` - Total validators
- `cometbft_consensus_validator_power` - Voting power
- `cometbft_consensus_missing_validators` - Missing validators
- `cometbft_consensus_byzantine_validators` - Byzantine validators
- `cometbft_consensus_block_interval_seconds` - Block time histogram
- `cometbft_consensus_rounds` - Consensus rounds histogram
- `cometbft_consensus_num_txs` - Transactions per block
- `cometbft_consensus_block_size_bytes` - Block size histogram
- `cometbft_consensus_total_txs` - Total transaction count
- `cometbft_consensus_block_parts` - Block parts count
- `cometbft_consensus_fast_syncing` - Fast sync status

### CometBFT Mempool Metrics

- `cometbft_mempool_size` - Current mempool size
- `cometbft_mempool_tx_size_bytes` - Transaction size histogram
- `cometbft_mempool_failed_txs` - Failed transactions
- `cometbft_mempool_recheck_times` - Recheck count

### CometBFT P2P Metrics

- `cometbft_p2p_peers` - Connected peers
- `cometbft_p2p_peer_receive_bytes_total` - Bytes received
- `cometbft_p2p_peer_send_bytes_total` - Bytes sent
- `cometbft_p2p_peer_pending_send_bytes` - Pending send bytes
- `cometbft_p2p_num_txs` - P2P transactions
- `cometbft_p2p_message_send_bytes_total` - Message bytes sent
- `cometbft_p2p_message_receive_bytes_total` - Message bytes received

### CometBFT State Metrics

- `cometbft_state_block_processing_time` - Block processing time histogram
- `cometbft_state_consensus_param_updates` - Consensus param updates
- `cometbft_state_validator_set_updates` - Validator set updates

### CometBFT ABCI Metrics

- `cometbft_abci_connection_method_timing_seconds` - ABCI method timings
  - Methods: `check_tx`, `deliver_tx`, `commit`, `begin_block`, `end_block`

### Custom Module Metrics (when implemented)

#### DEX Module
- `paw_dex_swaps_total` - Total swaps
- `paw_dex_swap_volume` - Swap volume
- `paw_dex_liquidity_added` - Liquidity added
- `paw_dex_liquidity_removed` - Liquidity removed
- `paw_dex_pools_total` - Total pools
- `paw_dex_pool_tvl` - Pool TVL
- `paw_dex_fees_collected` - Fees collected

#### Oracle Module
- `paw_oracle_price_updates` - Price updates
- `paw_oracle_validator_misses` - Validator misses
- `paw_oracle_validator_votes` - Validator votes
- `paw_oracle_price_deviation` - Price deviation

#### Compute Module
- `paw_compute_requests_total` - Compute requests
- `paw_compute_results_submitted` - Results submitted
- `paw_compute_verification_time` - Verification time
- `paw_compute_provider_performance` - Provider performance

---

**Last Updated**: 2025-12-14
**PAW Version**: devnet
**Prometheus Version**: 2.48.0
**CometBFT Version**: 0.38.17
