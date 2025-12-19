# PAW Grafana Dashboards

This directory contains production-ready Grafana dashboard definitions for monitoring PAW blockchain modules.

## Available Dashboards

### 1. Oracle Module Dashboard (`oracle-module.json`)

**Purpose**: Comprehensive monitoring of the Oracle module including price feeds, validator participation, TWAP aggregation, and cross-chain oracle operations.

**Panels**: 36 panels organized into 8 sections
**Lines of Code**: 1,281

#### Sections:
1. **Overview - Key Metrics** (6 panels)
   - Total assets tracked
   - Active validators
   - Price submissions (24h)
   - Average price deviation
   - Slashing events (24h)
   - Outliers detected (1h)

2. **Price Feed Health** (4 panels)
   - Price submission rate by asset
   - Aggregated prices
   - Price age by asset
   - Price deviation by asset

3. **Validator Performance** (4 panels)
   - Validator participation metrics (table)
   - Validator participation rate
   - Slashing events timeline

4. **TWAP & Aggregation** (4 panels)
   - TWAP vs spot price comparison
   - TWAP window size
   - Price aggregation rate
   - Aggregation latency distribution

5. **Security & Anomaly Detection** (4 panels)
   - Outlier detection by severity
   - Price rejections by reason
   - Manipulation detection
   - Circuit breaker triggers
   - Anomalous patterns detected

6. **Cross-Chain IBC Oracle Feeds** (4 panels)
   - IBC price feeds sent
   - IBC price feeds received
   - IBC timeout events
   - IBC operation latency

7. **System Health** (3 panels)
   - Stale data cleanups
   - TWAP updates
   - System operational metrics

#### Key Metrics:
- `paw_oracle_price_submissions_total`
- `paw_oracle_aggregated_price`
- `paw_oracle_price_deviation_percent`
- `paw_oracle_validator_reputation_score`
- `paw_oracle_slashing_events_total`
- `paw_oracle_twap_price`
- `paw_oracle_manipulation_detected_total`
- `paw_oracle_ibc_prices_sent_total`
- `paw_oracle_ibc_prices_received_total`

#### Variables:
- `$asset`: Filter by asset (multi-select)
- `$validator`: Filter by validator (multi-select)
- `$chain`: Filter by IBC chain (multi-select)

---

### 2. Compute Module Dashboard (`compute-module.json`)

**Purpose**: Comprehensive monitoring of the Compute module including job execution, ZK proof verification, escrow management, provider performance, and cross-chain compute operations.

**Panels**: 43 panels organized into 9 sections
**Lines of Code**: 1,502

#### Sections:
1. **Overview - Key Metrics** (6 panels)
   - Active providers
   - Job queue size
   - Jobs completed (24h)
   - Success rate (24h)
   - Total escrow locked
   - ZK proof success rate (1h)

2. **Job Queue & Execution** (4 panels)
   - Job submission rate by type
   - Job completion rate by provider
   - Job queue size over time
   - Job execution time distribution
   - Job failure rate by reason

3. **Provider Health & Reputation** (4 panels)
   - Provider performance dashboard (table)
   - Provider reputation over time
   - Provider slashing events
   - Provider registrations by capability

4. **ZK Proof Verification** (4 panels)
   - ZK proof verification rate
   - Proof verification latency
   - Invalid proof submissions
   - Circuit initializations/compilations

5. **Escrow Management** (2 panels)
   - Escrow activity (lock/release/refund)
   - Escrow balance by denomination

6. **Cross-Chain IBC Compute** (4 panels)
   - IBC jobs distributed to remote chains
   - IBC results received from remote chains
   - Remote providers discovered
   - Cross-chain job latency
   - IBC timeout events

7. **Security & Circuit Breakers** (4 panels)
   - Security incidents by type
   - Rate limit violations
   - Circuit breaker triggers
   - Panic recoveries (24h)

8. **System Maintenance & Cleanup** (4 panels)
   - Cleanup operations rate
   - Total nonces cleaned
   - State recoveries
   - Stale job cleanups

#### Key Metrics:
- `paw_compute_job_queue_size`
- `paw_compute_jobs_submitted_total`
- `paw_compute_jobs_completed_total`
- `paw_compute_provider_reputation_score`
- `paw_compute_proofs_verified_total`
- `paw_compute_escrow_balance`
- `paw_compute_ibc_jobs_distributed_total`
- `paw_compute_security_incidents_total`
- `paw_compute_circuit_breaker_triggers_total`

#### Variables:
- `$provider`: Filter by provider (multi-select)
- `$job_type`: Filter by job type (multi-select)
- `$chain`: Filter by IBC chain (multi-select)

---

### 3. IBC Boundary & Validation Dashboard (`ibc-boundary.json`)

**Purpose**: Surface packet validation failures across DEX/Oracle/Compute ports with per-reason and per-channel breakdowns for boundary hardening and relayer triage.

**Panels**:
- Rate of validation failures by reason (5m rate, stacked)
- Top channels by failures over the last hour (port/channel/reason breakdown)

**Key Metric**: `ibc_packet_validation_failed` counter with `port`, `channel`, `reason` labels.

**Variables**: `$port`, `$channel`, `$reason` (all support multi-select with regex `.*` when "All" is selected).

**Alerting**: Prometheus rule `monitoring/grafana/alerts/ibc-boundary-alerts.yml` fires on any 5m increase of validation failures per port/channel/reason; triage via `monitoring/runbooks/ibc-boundary-triage.md`.

---

## Installation

### Method 1: Grafana UI Import

1. Open Grafana web interface (default: http://localhost:3000)
2. Navigate to Dashboards → Import
3. Click "Upload JSON file"
4. Select `oracle-module.json` or `compute-module.json`
5. Configure datasource (select Prometheus instance)
6. Click "Import"

### Method 2: Provisioning (Recommended for Production)

Add to your Grafana provisioning configuration:

```yaml
# /etc/grafana/provisioning/dashboards/paw.yaml
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
      path: /home/hudson/blockchain-projects/paw/monitoring/grafana/dashboards
```

### Method 3: Docker Compose

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards/paw:ro
      - ./monitoring/grafana/provisioning.yaml:/etc/grafana/provisioning/dashboards/provisioning.yaml:ro
    ports:
      - "3000:3000"
```

---

## Prerequisites

### Required Prometheus Metrics

Both dashboards require a Prometheus instance scraping metrics from the PAW blockchain node. Ensure the following are configured:

1. **Enable Prometheus in node config** (`~/.paw/config/app.toml`):
   ```toml
   [telemetry]
   enabled = true
   prometheus-retention-time = 600

   [api]
   prometheus = true
   prometheus-listen-addr = ":26660"
   ```

2. **Prometheus scrape config** (`prometheus.yml`):
   ```yaml
   scrape_configs:
     - job_name: 'paw-node'
       static_configs:
         - targets: ['localhost:26660']
       scrape_interval: 15s
   ```

3. **Verify metrics endpoint**:
   ```bash
   curl http://localhost:26660/metrics | grep paw_oracle
   curl http://localhost:26660/metrics | grep paw_compute
   ```

### Required Grafana Datasource

Configure Prometheus datasource in Grafana:
- Name: `Prometheus` (or update datasource references in JSON files)
- Type: Prometheus
- URL: `http://localhost:9090` (or your Prometheus URL)
- Access: Server (default)

---

## Dashboard Features

### Common Features (Both Dashboards)

1. **Auto-refresh**: 30-second refresh interval
2. **Time range**: Default 6-hour window, fully adjustable
3. **Variables**: Multi-select filters for assets, validators, providers, chains
4. **Thresholds**: Color-coded alerts (green/yellow/orange/red)
5. **Percentiles**: p50, p95, p99 latency tracking
6. **Units**: Automatic unit formatting (ops, seconds, percent, etc.)

### Oracle Dashboard Highlights

- Real-time price feed health monitoring
- Validator reputation and slashing tracking
- TWAP vs spot price comparison
- Price manipulation detection
- Cross-chain oracle feed monitoring
- Anomaly detection and outlier tracking

### Compute Dashboard Highlights

- Job queue depth and execution metrics
- Provider reputation and performance tracking
- ZK proof verification monitoring
- Escrow balance tracking
- Cross-chain compute job distribution
- Security incident tracking
- System cleanup and maintenance metrics

---

## Customization

### Adding Custom Panels

1. Edit the JSON file directly, or
2. Import dashboard, modify in Grafana UI, then export updated JSON

### Modifying Thresholds

Edit the `thresholds` sections in each panel's `fieldConfig`:

```json
"thresholds": {
  "mode": "absolute",
  "steps": [
    {"value": null, "color": "green"},
    {"value": 50, "color": "yellow"},
    {"value": 100, "color": "red"}
  ]
}
```

### Adjusting Refresh Rate

Change the `refresh` field at the dashboard root:

```json
"refresh": "30s"  // Options: 5s, 10s, 30s, 1m, 5m, etc.
```

---

## Alerting

While these dashboards provide visualization, configure Prometheus alerts for critical metrics:

```yaml
# Example: Oracle price staleness alert
groups:
  - name: oracle
    rules:
      - alert: OraclePriceStale
        expr: paw_oracle_price_age_seconds > 300
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Oracle price feed stale for {{ $labels.asset }}"
```

---

## Troubleshooting

### Dashboard shows "No Data"

1. Verify Prometheus is scraping the node:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Check if metrics are being exposed:
   ```bash
   curl http://localhost:26660/metrics | grep paw_
   ```

3. Verify Grafana datasource configuration

### Metrics not updating

1. Check Prometheus scrape interval
2. Verify node telemetry is enabled
3. Check Prometheus retention time

### Variables not populating

1. Ensure Prometheus datasource is named "Prometheus"
2. Verify metrics exist: `paw_oracle_price_submissions_total`, `paw_compute_provider_reputation_score`
3. Check Grafana query inspector for errors

---

## Metrics Reference

### Oracle Module Metrics

All metrics are prefixed with `paw_oracle_`:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `price_submissions_total` | Counter | asset, validator | Total price submissions |
| `aggregated_price` | Gauge | asset | Current aggregated price |
| `price_deviation_percent` | Gauge | asset, validator | Price deviation from median |
| `validator_reputation_score` | Gauge | validator | Validator reputation (0-100) |
| `slashing_events_total` | Counter | validator, reason | Slashing events |
| `twap_price` | Gauge | asset | Time-weighted average price |
| `ibc_prices_sent_total` | Counter | destination_chain, asset | IBC prices sent |
| `manipulation_detected_total` | Counter | asset, detection_method | Manipulation attempts |

### Compute Module Metrics

All metrics are prefixed with `paw_compute_`:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `job_queue_size` | Gauge | - | Current queue size |
| `jobs_submitted_total` | Counter | job_type | Jobs submitted |
| `jobs_completed_total` | Counter | provider, status | Jobs completed |
| `provider_reputation_score` | Gauge | provider | Provider reputation (0-100) |
| `proofs_verified_total` | Counter | proof_type, status | ZK proofs verified |
| `escrow_balance` | Gauge | denom | Current escrow balance |
| `ibc_jobs_distributed_total` | Counter | target_chain | Jobs sent via IBC |
| `security_incidents_total` | Counter | type, severity | Security incidents |

---

## Maintenance

### Regular Updates

1. Monitor dashboard performance
2. Adjust panel queries for efficiency
3. Update thresholds based on production metrics
4. Add new panels as metrics are added

### Version Control

Both dashboards are version-controlled in the PAW repository:
- Location: `monitoring/grafana/dashboards/`
- Commit changes after modifications
- Test in staging before production deployment

---

## Support

For issues or questions:
- Check Grafana documentation: https://grafana.com/docs/
- Review Prometheus query syntax: https://prometheus.io/docs/prometheus/latest/querying/basics/
- PAW blockchain documentation: `/docs/`

---

**Created**: 2025-12-14
**Version**: 1.0
**Schema Version**: Grafana 38
**Datasource**: Prometheus
