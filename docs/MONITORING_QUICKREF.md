# PAW Monitoring - Quick Reference Card

## Start/Stop Commands

```bash
# Start everything
make full-stack-start

# Start individually
make monitoring-start    # Prometheus, Grafana, Loki, Jaeger
make explorer-start      # Big Dipper, Mintscan

# Stop
make full-stack-stop
make monitoring-stop
make explorer-stop

# View logs
make monitoring-logs
make explorer-logs

# Open dashboards
make metrics
```

## Access URLs

| Service      | URL                    |
| ------------ | ---------------------- |
| Grafana      | http://localhost:3000  |
| Prometheus   | http://localhost:9090  |
| Jaeger       | http://localhost:16686 |
| Alertmanager | http://localhost:9093  |
| Big Dipper   | http://localhost:3001  |
| Mintscan     | http://localhost:8080  |

**Grafana Credentials**: admin/admin

## Health Checks

```bash
curl http://localhost:1317/health         # Overall health
curl http://localhost:1317/health/live    # Liveness
curl http://localhost:1317/health/ready   # Readiness
curl http://localhost:1317/metrics        # Prometheus metrics
```

## Key Metrics

### Blockchain

- `tendermint_consensus_height` - Block height
- `tendermint_p2p_peers` - Peer count
- `cosmos_tx_total` - Total transactions
- `cosmos_tx_processing_time` - TX latency

### DEX

- `paw_dex_swaps_total` - Swap count
- `paw_dex_swap_volume_24h` - 24h volume
- `paw_dex_pool_reserves` - Pool reserves
- `paw_dex_pool_liquidity_usd` - Total liquidity

### System

- `node_cpu_seconds_total` - CPU usage
- `node_memory_MemAvailable_bytes` - Memory
- `node_filesystem_avail_bytes` - Disk space

## Log Queries (Loki)

```logql
{job="paw-node"}                    # All node logs
{job="paw-errors"}                  # Errors only
{job="paw-dex"} |= "swap"          # DEX swaps
{job="paw-node"} |= "failed"       # Failed operations
```

## Critical Alerts

1. **BlockProductionStopped** - No blocks for 5+ minutes
2. **HighAPIErrorRate** - API errors > 5%
3. **LowDiskSpace** - Disk < 15%

## Files

```
docker-compose.monitoring.yml       # Main stack
explorer/docker-compose.yml         # Explorers
infra/monitoring/prometheus.yml     # Metrics config
infra/monitoring/alertmanager.yml   # Alerts
app/telemetry.go                    # Tracing
x/dex/keeper/metrics.go             # DEX metrics
api/health/health.go                # Health checks
```

## Documentation

- **Quick Start**: `MONITORING.md`
- **Full Docs**: `infra/monitoring/README.md`
- **Summary**: `docs/OBSERVABILITY_SUMMARY.md`

## Support

- GitHub: https://github.com/paw-chain/paw
- Docs: https://docs.paw.network
- Discord: https://discord.gg/paw
