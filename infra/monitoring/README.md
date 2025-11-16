# PAW Blockchain Monitoring and Observability

Comprehensive monitoring, observability, and block explorer infrastructure for PAW blockchain.

## Architecture Overview

### Components

1. **Metrics Collection** - Prometheus
2. **Visualization** - Grafana
3. **Log Aggregation** - Loki + Promtail
4. **Distributed Tracing** - Jaeger
5. **Alerting** - Alertmanager
6. **Block Explorers** - Big Dipper + Mintscan
7. **System Metrics** - Node Exporter + cAdvisor

## Quick Start

### Start All Monitoring Services

```bash
# Start monitoring stack
make monitoring-start

# Start block explorers
make explorer-start

# View services
make metrics
```

### Access Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger UI**: http://localhost:16686
- **Alertmanager**: http://localhost:9093
- **Big Dipper**: http://localhost:3001
- **Mintscan API**: http://localhost:8080

## Metrics

### Blockchain Metrics

#### Tendermint/CometBFT Metrics

- `tendermint_consensus_height` - Current block height
- `tendermint_consensus_rounds` - Consensus rounds
- `tendermint_p2p_peers` - Number of connected peers
- `tendermint_mempool_size` - Mempool transaction count
- `tendermint_consensus_validators_power` - Validator voting power
- `tendermint_consensus_validator_missed_blocks` - Missed blocks count

#### Transaction Metrics

- `cosmos_tx_total` - Total transactions processed
- `cosmos_tx_failed_total` - Failed transactions
- `cosmos_tx_processing_time` - Transaction processing latency (histogram)
- `cosmos_tx_gas_used` - Gas consumed by transactions

#### Module Metrics

- `cosmos_module_execution_time` - Module execution latency
- `cosmos_block_height` - Current block height

### DEX-Specific Metrics

#### Swap Metrics

- `paw_dex_swaps_total` - Total swap count (by pool_id, token_in, token_out)
- `paw_dex_swap_failures_total` - Failed swaps (by pool_id, reason)
- `paw_dex_swap_volume_24h` - 24-hour volume in USD (by token)
- `paw_dex_swap_latency_ms` - Swap execution latency (histogram)

#### Pool Metrics

- `paw_dex_pool_reserves` - Pool reserves (by pool_id, token)
- `paw_dex_pool_liquidity_usd` - Total pool liquidity
- `paw_dex_pools_total` - Number of active pools
- `paw_dex_pool_apy` - Pool APY percentage

#### Liquidity Provider Metrics

- `paw_dex_lp_tokens_minted_total` - LP tokens minted
- `paw_dex_lp_tokens_burned_total` - LP tokens burned
- `paw_dex_liquidity_added_total` - Liquidity deposits
- `paw_dex_liquidity_removed_total` - Liquidity withdrawals

#### Fee Metrics

- `paw_dex_fees_collected_total` - Fees collected from swaps
- `paw_dex_fees_distributed_total` - Fees distributed to LPs

#### Price & Slippage Metrics

- `paw_dex_token_price_usd` - Token prices
- `paw_dex_price_impact_percent` - Price impact (histogram)
- `paw_dex_slippage_actual_percent` - Actual slippage (histogram)
- `paw_dex_slippage_exceeded_total` - Slippage tolerance exceeded

### System Metrics (Node Exporter)

- `node_cpu_seconds_total` - CPU usage
- `node_memory_MemAvailable_bytes` - Available memory
- `node_filesystem_avail_bytes` - Disk space
- `node_network_receive_bytes_total` - Network ingress
- `node_network_transmit_bytes_total` - Network egress

## Alerts

### Critical Alerts

1. **BlockProductionStopped** - No blocks for 5+ minutes
2. **HighAPIErrorRate** - API error rate > 5%
3. **LowDiskSpace** - Disk space < 15%

### Warning Alerts

1. **LowPeerCount** - Fewer than 3 peers
2. **HighBlockTime** - Slow block production
3. **HighTransactionLatency** - p95 latency > 2s
4. **HighCPUUsage** - CPU > 80%
5. **HighMemoryUsage** - Memory > 85%
6. **LowPoolLiquidity** - Pool reserves < threshold
7. **HighSwapFailureRate** - Swap failures > 5%

### Alert Routing

- **Critical** → PagerDuty + Slack (#paw-critical)
- **Warning** → Slack (#paw-alerts)
- **Info** → Slack (#paw-info)

## Grafana Dashboards

### PAW Chain Overview

Located at: `infra/monitoring/grafana-dashboards/paw-chain.json`

**Panels:**

1. Block Height (time series)
2. Blocks Per Second (graph)
3. Transaction Throughput (graph)
4. P2P Peer Count (stat)
5. Mempool Size (stat)
6. Validator Voting Power (pie chart)
7. DEX Swap Volume 24h (graph)
8. DEX Swaps Total (graph)
9. Pool Reserves (graph)
10. CPU Usage (graph)
11. Memory Usage (graph)
12. Transaction Latency p95 (graph)
13. Consensus State (stat)
14. Network I/O (graph)

### Importing Dashboards

```bash
# Import via Grafana UI
1. Navigate to Grafana → Dashboards → Import
2. Upload `paw-chain.json`
3. Select Prometheus datasource
```

## Logging

### Log Sources

Promtail collects logs from:

- `/var/log/paw/node.log` - Node logs
- `/var/log/paw/tendermint.log` - Consensus logs
- `/var/log/paw/dex.log` - DEX module logs
- `/var/log/paw/api.log` - API logs
- `/var/log/paw/error.log` - Error logs
- `/var/log/syslog` - System logs

### Log Queries (Loki)

```logql
# All PAW logs
{job="paw-node"}

# Error logs only
{job="paw-errors"}

# DEX swap logs
{job="paw-dex"} |= "swap"

# Failed transactions
{job="paw-node"} |= "failed" |= "tx"

# Logs from specific pool
{job="paw-dex", pool_id="pool-1"}
```

## Distributed Tracing

### Jaeger Integration

Traces are automatically collected for:

- Transaction execution
- Module execution
- DEX operations
- Database queries

### Viewing Traces

1. Open Jaeger UI: http://localhost:16686
2. Select service: `paw-blockchain`
3. Search by:
   - Operation name (e.g., "transaction.execute")
   - Tags (e.g., "block.height=12345")
   - Duration threshold

### Trace Examples

```go
// In transaction handler
ctx, span := TraceTxExecution(ctx, tx, height)
defer span()

// In module
ctx, span := TraceModuleExecution(ctx, "dex")
defer span()
```

## Health Checks

### Endpoints

- `GET /health` - Overall health with all checks
- `GET /health/live` - Liveness probe (always returns 200)
- `GET /health/ready` - Readiness probe (checks dependencies)
- `GET /metrics` - Prometheus metrics

### Health Check Response

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection OK",
      "latency": "5ms"
    },
    "rpc": {
      "status": "healthy",
      "message": "RPC connection OK",
      "latency": "10ms"
    },
    "consensus": {
      "status": "healthy",
      "message": "Node is validating",
      "latency": "2ms"
    }
  }
}
```

## Block Explorers

### Big Dipper

**Features:**

- Block explorer UI
- Transaction search
- Validator information
- Network statistics
- Governance proposals

**Database:** PostgreSQL
**Access:** http://localhost:3001

### Mintscan Backend

**Features:**

- REST API for blockchain data
- Transaction indexing
- Account balances
- Staking information

**API Endpoint:** http://localhost:8080
**Database:** PostgreSQL

## Troubleshooting

### No Data in Prometheus

1. Check targets: http://localhost:9090/targets
2. Verify metrics endpoints are accessible
3. Check Prometheus logs: `docker logs paw-prometheus`
4. Ensure network connectivity: `docker network inspect paw-network`

### Grafana Dashboard Empty

1. Verify Prometheus datasource configured
2. Check query syntax
3. Verify time range selection
4. Check Grafana logs: `docker logs paw-grafana`

### Alerts Not Firing

1. Check alert rules: http://localhost:9090/alerts
2. Verify Alertmanager config
3. Test webhook URLs
4. Check Alertmanager logs: `docker logs paw-alertmanager`

### Logs Not Appearing in Loki

1. Verify Promtail is running: `docker logs paw-promtail`
2. Check log file paths exist
3. Verify Loki endpoint connectivity
4. Check Promtail config syntax

### Jaeger No Traces

1. Verify OTLP endpoint configuration
2. Check sampling rate (default: 100%)
3. Ensure app is instrumented
4. Check Jaeger logs: `docker logs paw-jaeger-aio`

## Performance Tuning

### Prometheus Retention

Default: 30 days. Adjust in `docker-compose.monitoring.yml`:

```yaml
command:
  - '--storage.tsdb.retention.time=90d'
```

### Loki Retention

Default: 30 days. Adjust in `loki-config.yaml`:

```yaml
table_manager:
  retention_period: 720h # 30 days
```

### Scrape Intervals

Adjust in `prometheus.yml` based on needs:

- **High-frequency** (5-10s): Consensus, transactions
- **Medium** (15-30s): API, system metrics
- **Low** (1-5m): Aggregated stats

## Security Considerations

1. **Change default passwords** in production
2. **Enable authentication** on Grafana/Prometheus
3. **Use TLS** for metric endpoints
4. **Restrict network access** to monitoring ports
5. **Sanitize alert messages** to avoid leaking sensitive data
6. **Configure webhook secrets** for Alertmanager

## Production Deployment

### Recommendations

1. **Separate monitoring stack** from blockchain nodes
2. **Use persistent volumes** for time-series data
3. **Enable remote storage** for long-term retention
4. **Set up redundant Alertmanagers** for HA
5. **Configure backup** for Grafana dashboards
6. **Use service discovery** for dynamic node discovery
7. **Implement log rotation** for disk space management

### Scaling

- **Prometheus**: Federation or Thanos for multi-cluster
- **Loki**: Distributed mode for high log volume
- **Jaeger**: Elasticsearch backend for production scale

## Support

For issues or questions:

- Documentation: https://docs.paw.network
- GitHub Issues: https://github.com/paw/paw
- Discord: https://discord.gg/paw
