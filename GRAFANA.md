# PAW Blockchain - Grafana Metrics Integration

**Target Audience**: AI coding agents and developers

## Metrics Architecture

**Status**: ⚠️ No custom metrics implemented - uses Cosmos SDK defaults only
**Location**: N/A (no custom metrics code)
**Namespace**: Standard Cosmos SDK namespaces (`tendermint_*`, `cosmos_*`)
**Dashboard**: PAW Network & DEX Live Metrics (Grafana Cloud)

### Metrics Endpoints

PAW exposes Prometheus metrics on **5 ports** (all standard Cosmos SDK):

1. **Port 36660** - Tendermint consensus metrics
   - Block production, consensus, validator voting

2. **Port 2317** - REST API metrics
   - API request counts, latencies

3. **Port 36661** - Application metrics
   - Transaction processing, module operations

4. **Port 36662** - DEX metrics (if DEX module enabled)
   - Liquidity pools, swap operations

5. **Port 36663** - Validator metrics
   - Validator status, voting power

## Available Metrics

### Standard Cosmos SDK Metrics Only

```
# Tendermint Consensus (port 36660)
tendermint_consensus_height
tendermint_consensus_validators
tendermint_consensus_missing_validators
tendermint_consensus_byzantine_validators
tendermint_consensus_block_interval_seconds
tendermint_consensus_rounds
tendermint_consensus_num_txs

# Tendermint Mempool
tendermint_mempool_size
tendermint_mempool_tx_size_bytes
tendermint_mempool_failed_txs

# Tendermint P2P
tendermint_p2p_peers
tendermint_p2p_peer_receive_bytes_total
tendermint_p2p_peer_send_bytes_total

# Cosmos SDK Application (port 36661)
cosmos_app_tx_count
cosmos_app_tx_size
cosmos_app_block_processing_time
```

**Note**: PAW does **NOT** have custom metrics like Aura. The codebase contains no custom Prometheus instrumentation.

## Exposing Metrics

### Automatic Exposure
Metrics are **automatically exposed** when the PAW node starts via Cosmos SDK's built-in Prometheus exporter.

```bash
cd /home/decri/blockchain-projects/paw
./pawd start --home ~/.paw
```

**Metrics immediately available at:**
- `http://localhost:36660/metrics` (Tendermint)
- `http://localhost:2317/metrics` (API)
- `http://localhost:36661/metrics` (App)
- `http://localhost:36662/metrics` (DEX, if enabled)
- `http://localhost:36663/metrics` (Validators)

### Verification

```bash
# Check Tendermint metrics
curl -s http://localhost:36660/metrics | grep tendermint

# Check Cosmos SDK metrics
curl -s http://localhost:36661/metrics | grep cosmos

# Verify Prometheus is scraping
curl -s http://localhost:9091/targets | grep paw
```

## Prometheus Configuration

**Location**: `/etc/prometheus/prometheus.yml`

```yaml
scrape_configs:
  # PAW Tendermint
  - job_name: 'paw-tendermint'
    static_configs:
      - targets: ['localhost:36660']
        labels:
          blockchain: paw
          component: consensus

  # PAW REST API
  - job_name: 'paw-api'
    static_configs:
      - targets: ['localhost:2317']
        labels:
          blockchain: paw
          component: api

  # PAW Cosmos SDK App
  - job_name: 'paw-app'
    static_configs:
      - targets: ['localhost:36661']
        labels:
          blockchain: paw
          component: application

  # PAW DEX Module
  - job_name: 'paw-dex'
    static_configs:
      - targets: ['localhost:36662']
        labels:
          blockchain: paw
          component: dex

  # PAW Validators
  - job_name: 'paw-validators'
    static_configs:
      - targets: ['localhost:36663']
        labels:
          blockchain: paw
          component: validators
```

**Remote Write**: Configured to send to Grafana Cloud (already set up)

## Grafana Dashboard

**Location**: Grafana Cloud - https://altrestackmon.grafana.net
**Dashboard Name**: "PAW Network & DEX Live Metrics"
**Public Access**: Enabled (share via external link)

### Accessing the Dashboard

1. **Grafana Cloud** (recommended for investors/stakeholders):
   ```
   https://altrestackmon.grafana.net/dashboards
   Click: "PAW Network & DEX Live Metrics"
   Share → Share externally → Copy external link
   ```

2. **Local Grafana** (development):
   ```
   http://localhost:3000
   Login: admin/admin
   ```

### Dashboard Panels

The dashboard shows:
- Block height and production rate
- Transaction count and throughput
- Validator count and status
- Mempool size
- Peer count and network traffic
- DEX liquidity and swap volume (if DEX module active)
- API request metrics

**Limitation**: Dashboard queries are configured for `paw_*` namespace, but PAW only exposes `tendermint_*` and `cosmos_*` metrics. Dashboard will show limited data until custom metrics are added.

## Adding Custom Metrics (Recommended)

PAW **does not have custom metrics**. To match Aura's monitoring capabilities:

### Option 1: Add Custom Metrics Module (Recommended)

Create a PAW monitoring module similar to Aura:

1. **Create module structure**:
   ```bash
   cd /home/decri/blockchain-projects/paw/x
   mkdir -p monitoring/metrics
   ```

2. **Copy Aura's metrics as template**:
   ```bash
   cp /home/decri/blockchain-projects/aura/chain/x/monitoring/metrics/prometheus.go \
      /home/decri/blockchain-projects/paw/x/monitoring/metrics/prometheus.go
   ```

3. **Modify namespace**:
   ```go
   // Change "aura" to "paw"
   Namespace: "paw",
   Subsystem: "monitoring",
   ```

4. **Register module** in `app/app.go`

5. **Expose on port 39090**:
   ```go
   // In app.go
   go func() {
       http.Handle("/metrics", promhttp.Handler())
       http.ListenAndServe(":39090", nil)
   }()
   ```

6. **Update Prometheus config** to scrape port 39090

### Option 2: Use Cosmos SDK Telemetry (Simpler)

Enable Cosmos SDK's built-in telemetry in `app.toml`:

```toml
[telemetry]
enabled = true
prometheus-retention-time = 60
```

This provides basic metrics but not as comprehensive as custom instrumentation.

### Option 3: Query Existing Metrics Differently

Update dashboard queries to use `tendermint_*` and `cosmos_*` namespaces instead of `paw_*`:

```promql
# Instead of: paw_block_height
# Use: tendermint_consensus_height{blockchain="paw"}

# Instead of: paw_transactions_total
# Use: cosmos_app_tx_count{blockchain="paw"}
```

## Current State

**What Works Now:**
- ✅ Standard Cosmos SDK metrics exposed automatically
- ✅ Prometheus scraping all 5 endpoints
- ✅ Metrics flowing to Grafana Cloud
- ✅ Dashboard created and public

**What Needs Work:**
- ⚠️ Dashboard queries expect `paw_*` namespace but only `tendermint_*/cosmos_*` available
- ⚠️ No custom PAW-specific metrics (DEX, liquidity pools, etc.)
- ⚠️ Limited observability compared to Aura

**Recommendation**: Add custom metrics module (Option 1) to match Aura's monitoring depth.

## Troubleshooting

### Metrics Not Showing

```bash
# 1. Verify node is running
ps aux | grep pawd

# 2. Check metrics endpoints respond
curl http://localhost:36660/metrics
curl http://localhost:36661/metrics

# 3. Verify Prometheus is scraping
curl http://localhost:9091/targets | grep paw

# 4. Check Prometheus logs
sudo journalctl -u prometheus -n 50

# 5. Check node logs
tail -f ~/.paw/data/paw.log
```

### Empty Dashboard

**Cause 1**: Node not running
**Solution**: Start the PAW node

```bash
cd /home/decri/blockchain-projects/paw
./pawd start --home ~/.paw
```

**Cause 2**: Dashboard queries use wrong namespace
**Solution**: Edit dashboard JSON to query `tendermint_*` metrics:

```json
{
  "targets": [{
    "expr": "tendermint_consensus_height{blockchain=\"paw\"}"
  }]
}
```

Metrics appear within 15 seconds of node start (Prometheus scrape interval).

## Summary: Metrics Wiring Status

| Feature | Aura | XAI | PAW |
|---------|------|-----|-----|
| Custom metrics code | ✅ Yes | ✅ Yes | ❌ No |
| Auto-exposed on startup | ✅ Yes | ✅ Yes | ✅ Yes |
| Dashboard wired | ✅ Yes | ✅ Yes | ⚠️ Partial |
| Prometheus scraping | ✅ Yes | ✅ Yes | ✅ Yes |
| Grafana Cloud push | ✅ Yes | ✅ Yes | ✅ Yes |
| **Action Required** | None | None | **Add custom metrics** |

**Bottom Line**: PAW will show **basic metrics** (blocks, txs, validators) but lacks the **rich custom metrics** that Aura and XAI provide. Custom metrics recommended for production monitoring.

## Reference Documents

- **Metrics List**: `/home/decri/blockchain-projects/METRICS_REFERENCE.md`
- **Setup Status**: `/home/decri/blockchain-projects/SETUP_STATUS.md`
- **Prometheus Config**: `/etc/prometheus/prometheus.yml`
- **Dashboard JSON**: `/home/decri/blockchain-projects/dashboards/paw-network-dashboard.json`
- **Aura Metrics Template**: `/home/decri/blockchain-projects/aura/chain/x/monitoring/metrics/prometheus.go`
