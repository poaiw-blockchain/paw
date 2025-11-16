# PAW Blockchain Monitoring & Observability

Comprehensive monitoring, observability, and block explorer infrastructure for the PAW blockchain.

## Quick Start

### Start Full Observability Stack

```bash
# Start everything (monitoring + explorers)
make full-stack-start

# Or start individually
make monitoring-start  # Prometheus, Grafana, Loki, Jaeger, Alertmanager
make explorer-start    # Big Dipper, Mintscan
```

### Access Dashboards

```bash
# Open all dashboards in browser
make metrics
```

**URLs:**

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **Alertmanager**: http://localhost:9093
- **Big Dipper Explorer**: http://localhost:3001
- **Mintscan API**: http://localhost:8080

## Architecture

### Monitoring Stack

```
┌─────────────────────────────────────────────────────────┐
│                    PAW Blockchain                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ Node     │  │ DEX      │  │ API      │             │
│  │ :26660   │  │ :26662   │  │ :1317    │             │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
└───────┼─────────────┼─────────────┼────────────────────┘
        │             │             │
        ├─────────────┴─────────────┴──────► Prometheus
        │                                        │
        │                                        ▼
        ├──────────────────────────────────► Grafana
        │                                        │
        ├──────────────────────────────────► Alertmanager
        │                                        │
        ├──────────────────────────────────► Jaeger (traces)
        │
        └──────────────────────────────────► Loki (logs)
                                                 ▲
                                                 │
                                            Promtail
```

### Components

| Component         | Purpose                       | Port  |
| ----------------- | ----------------------------- | ----- |
| **Prometheus**    | Metrics collection & storage  | 9090  |
| **Grafana**       | Visualization & dashboards    | 3000  |
| **Loki**          | Log aggregation               | 3100  |
| **Promtail**      | Log shipping                  | 9080  |
| **Jaeger**        | Distributed tracing           | 16686 |
| **Alertmanager**  | Alert routing & notifications | 9093  |
| **Node Exporter** | System metrics                | 9100  |
| **cAdvisor**      | Container metrics             | 8081  |
| **Big Dipper**    | Block explorer UI             | 3001  |
| **Mintscan**      | Explorer API                  | 8080  |

## Metrics Categories

### 1. Blockchain Metrics

**Tendermint/CometBFT**

- `tendermint_consensus_height` - Current block height
- `tendermint_consensus_rounds` - Consensus rounds
- `tendermint_p2p_peers` - Connected peers
- `tendermint_mempool_size` - Pending transactions
- `tendermint_consensus_validators_power` - Validator power
- `tendermint_consensus_validator_missed_blocks` - Missed blocks

**Transactions**

- `cosmos_tx_total` - Total transactions
- `cosmos_tx_failed_total` - Failed transactions
- `cosmos_tx_processing_time` - Processing latency (histogram)
- `cosmos_tx_gas_used` - Gas consumption

### 2. DEX-Specific Metrics

**Swaps**

- `paw_dex_swaps_total` - Swap count (by pool, tokens)
- `paw_dex_swap_failures_total` - Failed swaps
- `paw_dex_swap_volume_24h` - 24h volume (USD)
- `paw_dex_swap_latency_ms` - Swap latency

**Pools**

- `paw_dex_pool_reserves` - Pool reserves
- `paw_dex_pool_liquidity_usd` - Total liquidity
- `paw_dex_pools_total` - Number of pools
- `paw_dex_pool_apy` - Pool APY

**Liquidity**

- `paw_dex_lp_tokens_minted_total` - LP tokens minted
- `paw_dex_lp_tokens_burned_total` - LP tokens burned
- `paw_dex_liquidity_added_total` - Liquidity added
- `paw_dex_liquidity_removed_total` - Liquidity removed

**Fees & Prices**

- `paw_dex_fees_collected_total` - Fees collected
- `paw_dex_token_price_usd` - Token prices
- `paw_dex_price_impact_percent` - Price impact
- `paw_dex_slippage_actual_percent` - Actual slippage

### 3. System Metrics

- `node_cpu_seconds_total` - CPU usage
- `node_memory_MemAvailable_bytes` - Available memory
- `node_filesystem_avail_bytes` - Disk space
- `node_network_receive_bytes_total` - Network ingress
- `node_network_transmit_bytes_total` - Network egress

## Alerts

### Critical Alerts (PagerDuty + Slack)

1. **BlockProductionStopped** - No blocks for 5+ minutes
2. **HighAPIErrorRate** - API errors > 5%
3. **LowDiskSpace** - Disk < 15%

### Warning Alerts (Slack)

1. **LowPeerCount** - < 3 peers for 10 minutes
2. **HighBlockTime** - Slow block production
3. **HighTransactionLatency** - p95 > 2 seconds
4. **HighCPUUsage** - CPU > 80%
5. **HighMemoryUsage** - Memory > 85%
6. **LowPoolLiquidity** - Pool reserves below threshold
7. **HighSwapFailureRate** - Swap failures > 5%

### Alert Configuration

Edit `C:\users\decri\gitclones\paw\infra\monitoring\alertmanager.yml`:

```yaml
global:
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

receivers:
  - name: 'critical-alerts'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
    slack_configs:
      - channel: '#paw-critical'
```

## Grafana Dashboards

### PAW Chain Overview Dashboard

Located: `C:\users\decri\gitclones\paw\infra\monitoring\grafana-dashboards\paw-chain.json`

**Panels:**

- Block Height & Production Rate
- Transaction Throughput
- P2P Peer Count & Network I/O
- Mempool Size
- Validator Voting Power
- DEX Swap Volume (24h)
- Pool Reserves & Liquidity
- CPU & Memory Usage
- Transaction Latency (p95)
- Consensus State

### Import Dashboard

1. Open Grafana: http://localhost:3000
2. Navigate to Dashboards → Import
3. Upload `paw-chain.json`
4. Select Prometheus datasource
5. Click Import

## Log Queries (Loki)

### Common Queries

```logql
# All PAW node logs
{job="paw-node"}

# Error logs only
{job="paw-errors"}

# DEX swap logs
{job="paw-dex"} |= "swap"

# Failed transactions
{job="paw-node"} |= "failed" |= "tx"

# Logs from specific pool
{job="paw-dex", pool_id="pool-1"}

# Logs in time range with regex
{job="paw-node"} |~ "block.*committed" | logfmt
```

### Log Levels

Access logs by level:

```logql
{job="paw-node", level="ERROR"}
{job="paw-node", level="WARN"}
{job="paw-node", level="INFO"}
```

## Distributed Tracing (Jaeger)

### Instrumented Operations

Traces are automatically collected for:

- Transaction execution
- Module execution (DEX, compute, oracle)
- DEX operations (swaps, liquidity changes)
- Database queries
- API requests

### Viewing Traces

1. Open Jaeger UI: http://localhost:16686
2. Select service: `paw-blockchain`
3. Search by:
   - **Operation**: `transaction.execute`, `module.execute`
   - **Tags**: `block.height=12345`, `pool_id=pool-1`
   - **Duration**: `> 1000ms` (slow transactions)

### Example Trace Tags

```
block.height: 12345
tx.msg.count: 3
module.name: dex
pool_id: pool-1
token_in: PAW
token_out: USDC
```

## Health Checks

### Endpoints

```bash
# Overall health
curl http://localhost:1317/health

# Liveness probe
curl http://localhost:1317/health/live

# Readiness probe
curl http://localhost:1317/health/ready

# Prometheus metrics
curl http://localhost:1317/metrics
```

### Health Response

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

## Makefile Commands

### Monitoring

```bash
make monitoring-start      # Start monitoring stack
make monitoring-stop       # Stop monitoring stack
make monitoring-restart    # Restart monitoring stack
make monitoring-logs       # View logs
make monitoring-status     # Show status
```

### Explorers

```bash
make explorer-start        # Start block explorers
make explorer-stop         # Stop explorers
make explorer-logs         # View logs
```

### Convenience

```bash
make metrics               # Open all dashboards in browser
make full-stack-start      # Start everything
make full-stack-stop       # Stop everything
```

## Troubleshooting

### No Data in Prometheus

1. Check targets: http://localhost:9090/targets
2. Verify PAW node is exposing metrics on `:26660`
3. Check Prometheus logs: `docker logs paw-prometheus`
4. Verify network: `docker network inspect paw-network`

### Grafana Dashboard Empty

1. Verify Prometheus datasource is configured
2. Check time range (last 15 minutes)
3. Test query in Prometheus first
4. Check Grafana logs: `docker logs paw-grafana`

### Alerts Not Firing

1. Check alert rules: http://localhost:9090/alerts
2. Verify Alertmanager config
3. Test webhook URLs
4. Check logs: `docker logs paw-alertmanager`

### Logs Not in Loki

1. Verify Promtail is running: `docker logs paw-promtail`
2. Check log file paths exist: `/var/log/paw/*.log`
3. Verify Loki connectivity
4. Check Promtail config syntax

### No Traces in Jaeger

1. Verify OTLP endpoint (`:4317`)
2. Check sampling rate (default: 100%)
3. Ensure app has telemetry enabled
4. Check logs: `docker logs paw-jaeger-aio`

## Production Recommendations

### Security

1. Change default passwords (Grafana)
2. Enable authentication on all services
3. Use TLS for metric endpoints
4. Restrict network access to monitoring ports
5. Sanitize alert messages
6. Configure webhook secrets

### Scalability

1. Separate monitoring from blockchain nodes
2. Use persistent volumes for data
3. Enable remote storage for long-term retention
4. Set up redundant Alertmanagers
5. Implement log rotation
6. Use service discovery for dynamic nodes

### High Availability

1. **Prometheus**: Federation or Thanos
2. **Loki**: Distributed mode
3. **Jaeger**: Elasticsearch backend
4. **Grafana**: Database backend (not SQLite)

### Data Retention

- **Prometheus**: 30 days (configurable)
- **Loki**: 30 days (configurable)
- **Jaeger**: 7 days (configurable)

Adjust in respective config files based on storage capacity.

## Additional Resources

- **Full Documentation**: `C:\users\decri\gitclones\paw\infra\monitoring\README.md`
- **Prometheus Docs**: https://prometheus.io/docs/
- **Grafana Docs**: https://grafana.com/docs/
- **Jaeger Docs**: https://www.jaegertracing.io/docs/
- **Loki Docs**: https://grafana.com/docs/loki/

## Support

For issues or questions:

- GitHub: https://github.com/paw-chain/paw
- Documentation: https://docs.paw.network
- Discord: https://discord.gg/paw
