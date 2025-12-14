# PAW Blockchain - Monitoring Infrastructure Deployment

## Deployment Status

**Date:** 2025-12-14
**Status:** ✅ DEPLOYED
**Version:** Prometheus v2.48.0, Grafana 10.2.0, Alertmanager v0.26.0

## Services Running

All monitoring services have been successfully deployed and are running:

| Service | Container Name | Status | Port | Access URL |
|---------|---------------|--------|------|------------|
| **Prometheus** | paw-prometheus | ✅ Healthy | 9091 | http://localhost:9091 |
| **Grafana** | paw-grafana | ✅ Healthy | 11030 | http://localhost:11030 |
| **Grafana DB** | paw-grafana-db | ✅ Healthy | (internal) | PostgreSQL backend |
| **Alertmanager** | paw-alertmanager | ✅ Healthy | 9093 | http://localhost:9093 |
| **Node Exporter** | paw-node-exporter | ✅ Running | 9100 | http://localhost:9100 |
| **cAdvisor** | paw-cadvisor | ✅ Running | 11082 | http://localhost:11082 |

## Access Information

### Grafana Dashboard
- **URL:** http://localhost:11030
- **Default Credentials:**
  - Username: `admin`
  - Password: `grafana_secure_password`

### Prometheus
- **URL:** http://localhost:9091
- **Targets:** http://localhost:9091/targets
- **Rules:** http://localhost:9091/rules

### Alertmanager
- **URL:** http://localhost:9093
- **Configuration:** Webhook-based (local development)

## Provisioned Dashboards

3 Grafana dashboards have been automatically provisioned:

1. **PAW Blockchain Metrics** - Overall blockchain health and performance
2. **PAW DEX Metrics** - DEX-specific metrics (liquidity pools, swaps, volume)
3. **PAW Node Metrics** - Individual node performance and system resources

All dashboards are located in the `PAW` folder in Grafana.

## Alert Rules

16 alert rules have been loaded across 5 groups:

| Group | Rules | Description |
|-------|-------|-------------|
| **blockchain_health** | 4 | Blockchain consensus and block production |
| **api_health** | 2 | API endpoint availability |
| **dex_health** | 3 | DEX operation health |
| **transaction_health** | 3 | Transaction processing metrics |
| **system_resources** | 4 | System resource utilization (CPU, memory, disk) |

### Alert Notification

Currently configured with webhook notifications for local development. To enable production alerting:

1. Edit `/home/hudson/blockchain-projects/paw/compose/docker/monitoring/alertmanager/config.yml`
2. Add Slack, PagerDuty, or email configuration
3. Restart Alertmanager: `docker restart paw-alertmanager`

## Scrape Targets

### Currently Working (3/18)

- ✅ **prometheus** - Prometheus self-monitoring
- ✅ **cadvisor** - Container metrics
- ✅ **node-exporter** - System metrics

### Pending Node Metrics Configuration (15/18)

The following targets are configured but require PAW node metrics to be enabled:

- ⏳ **paw-tendermint** (6 targets) - Tendermint/CometBFT consensus metrics
  - node1 (26657), node2 (26667), node3 (26677), node4 (26687)
  - sentry1 (30658), sentry2 (30668)

- ⏳ **paw-app** (6 targets) - Application-specific metrics
  - Ports 39090-39095 for all 6 nodes

- ⏳ **paw-api** - Cosmos SDK REST API metrics (port 1317)
- ⏳ **paw-dex** - DEX-specific metrics (port 1317/dex/metrics)
- ⏳ **paw-validators** - Validator metrics (port 1317/validator/metrics)

## Enabling Node Metrics

To enable metrics on PAW nodes, update the node configuration:

### Option 1: Enable Tendermint Metrics

Edit `~/.paw/config/config.toml`:

```toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
max_open_connections = 3
namespace = "tendermint"
```

### Option 2: Enable App Metrics

Edit `~/.paw/config/app.toml`:

```toml
[telemetry]
enabled = true
prometheus-retention-time = 60

[api]
enable = true
prometheus = true
```

### Restart Nodes

After configuration changes:

```bash
docker compose -f compose/docker-compose.yml restart
```

## Verification

Run the verification script to check all services:

```bash
/home/hudson/blockchain-projects/paw/scripts/verify-monitoring.sh
```

Expected output:
- All containers running
- All services responding
- Prometheus targets being scraped
- Grafana dashboards accessible
- Alert rules loaded

## Data Retention

- **Prometheus:** 30 days
- **Grafana:** Persistent (PostgreSQL backend)
- **Alertmanager:** In-memory (resets on restart)

## Storage Volumes

Docker volumes created for persistent storage:

- `compose_prometheus-data` - Prometheus time-series data
- `compose_grafana-data` - Grafana dashboards and settings
- `compose_grafana-postgres-data` - Grafana PostgreSQL database
- `compose_alertmanager-data` - Alertmanager state

## Network Configuration

- **Network:** `compose_monitoring` (bridge)
- **Subnet:** 172.31.0.0/16
- **Host Access:** Targets accessible via `172.17.0.1` (Docker bridge)

## Managing the Monitoring Stack

### Start All Services

```bash
docker compose -f compose/docker-compose.monitoring.yml up -d
```

### Stop All Services

```bash
docker compose -f compose/docker-compose.monitoring.yml down
```

### View Logs

```bash
# All services
docker compose -f compose/docker-compose.monitoring.yml logs -f

# Specific service
docker compose -f compose/docker-compose.monitoring.yml logs -f prometheus
docker compose -f compose/docker-compose.monitoring.yml logs -f grafana
```

### Restart a Service

```bash
docker restart paw-prometheus
docker restart paw-grafana
docker restart paw-alertmanager
```

## Troubleshooting

### Prometheus Targets Down

**Issue:** PAW node targets showing as "down"
**Cause:** Node metrics not enabled
**Solution:** Enable metrics in node configuration (see "Enabling Node Metrics" above)

### Grafana Login Issues

**Issue:** Cannot log in to Grafana
**Credentials:**
- Username: `admin`
- Password: `grafana_secure_password`

### Dashboards Not Showing Data

**Issue:** Dashboards show "No data"
**Causes:**
1. Prometheus targets are down
2. Metrics not being collected yet (wait 1-2 minutes)
3. Time range too narrow (adjust Grafana time picker)

### Alert Rules Not Loading

**Issue:** Alert rules not appearing in Prometheus
**Solution:**
1. Check Prometheus logs: `docker logs paw-prometheus`
2. Validate rules file: `/home/hudson/blockchain-projects/paw/compose/docker/monitoring/prometheus/rules/alert_rules.yml`
3. Restart Prometheus: `docker restart paw-prometheus`

### Port Conflicts

All monitoring services use the PAW port range (11000+):
- Grafana: 11030
- cAdvisor: 11082

Other services use standard ports:
- Prometheus: 9091
- Alertmanager: 9093
- Node Exporter: 9100

## Next Steps

1. ✅ **Deployed** - All monitoring infrastructure is running
2. ⏳ **Configure Node Metrics** - Enable metrics on PAW nodes
3. ⏳ **Validate Dashboards** - Ensure all dashboards show live data
4. ⏳ **Test Alerting** - Trigger test alerts to verify notification flow
5. ⏳ **Production Alerting** - Configure Slack/PagerDuty for production

## Configuration Files

All monitoring configuration is stored in:

```
compose/docker/monitoring/
├── prometheus.yml                    # Prometheus scrape config
├── prometheus/rules/
│   └── alert_rules.yml              # Alert rule definitions
├── alertmanager/
│   └── config.yml                   # Alert routing config
└── grafana/
    ├── datasources/
    │   └── prometheus.yml           # Prometheus datasource
    └── dashboards/
        ├── dashboard.yml            # Dashboard provisioning
        ├── blockchain-metrics.json  # Blockchain dashboard
        ├── dex-metrics.json        # DEX dashboard
        └── node-metrics.json       # Node dashboard
```

## Maintenance

### Update Prometheus Config

1. Edit `/home/hudson/blockchain-projects/paw/compose/docker/monitoring/prometheus.yml`
2. Reload config: `curl -X POST http://localhost:9091/-/reload`
   Or restart: `docker restart paw-prometheus`

### Update Alert Rules

1. Edit `/home/hudson/blockchain-projects/paw/compose/docker/monitoring/prometheus/rules/alert_rules.yml`
2. Reload config: `curl -X POST http://localhost:9091/-/reload`

### Update Grafana Dashboards

Dashboards are provisioned from JSON files. To update:

1. Edit JSON file in `compose/docker/monitoring/grafana/dashboards/`
2. Dashboards auto-reload every 10 seconds (configured in `dashboard.yml`)

## Support

For issues or questions:
1. Run verification script: `/home/hudson/blockchain-projects/paw/scripts/verify-monitoring.sh`
2. Check service logs: `docker compose -f compose/docker-compose.monitoring.yml logs <service>`
3. Consult Prometheus targets page: http://localhost:9091/targets

---

**Deployment completed:** 2025-12-14
**Deployed by:** Claude Code Automation
**Infrastructure:** Docker Compose v2 on Linux
