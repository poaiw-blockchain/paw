# Custom Grafana Dashboards Guide

This guide provides comprehensive documentation for the custom Grafana dashboards created for the PAW blockchain's Oracle and Compute modules.

## Table of Contents

- [Dashboard Overview](#dashboard-overview)
- [Oracle Module Dashboard](#oracle-module-dashboard)
- [Compute Module Dashboard](#compute-module-dashboard)
- [Advanced Modules Overview Dashboard](#advanced-modules-overview-dashboard)
- [Alert Thresholds](#alert-thresholds)
- [Customization Guide](#customization-guide)
- [Troubleshooting](#troubleshooting)
- [Architecture Reference](#architecture-reference)

## Dashboard Overview

Three custom dashboards have been created for monitoring advanced blockchain modules:

1. **PAW Oracle Module Metrics** - Comprehensive monitoring for the Oracle price feed system
2. **PAW Compute Module Metrics** - Detailed visibility into ZK compute jobs and providers
3. **PAW Advanced Modules Overview** - Combined health status and cross-module comparison

All dashboards follow the design patterns established in the Blockchain Control Center, including:
- Consistent color schemes and thresholds
- Standard metric naming conventions
- Prometheus query optimization
- Template variables for filtering
- Auto-refresh at 30-second intervals

## Oracle Module Dashboard

**Location**: `PAW/Oracle` folder in Grafana
**Refresh Rate**: 30 seconds
**Default Time Range**: Last 6 hours

### Purpose

Monitor the Oracle module's price feed system, including validator participation, price aggregation, TWAP calculations, and cross-chain price distribution via IBC.

### Key Metrics Explained

#### Top Row - Summary Stats

1. **Total Assets Tracked**: Number of price feeds currently monitored
   - **Good**: Matches expected assets
   - **Concern**: Sudden drop indicates feed failures

2. **Active Validators**: Validators currently submitting oracle prices
   - **Good**: >67% of total validators
   - **Concern**: <50% indicates participation issues

3. **Total Price Submissions (24h)**: Volume of price submissions
   - **Good**: Steady rate matching expected frequency
   - **Concern**: Sharp drops or spikes

4. **Avg Price Deviation**: Average validator deviation from median price
   - **Green**: <2%
   - **Yellow**: 2-5%
   - **Red**: >5% (may indicate data quality issues)

5. **Slash Events (24h)**: Validator slashing for oracle misbehavior
   - **Green**: 0
   - **Yellow**: 1-10
   - **Red**: >10 (investigate immediately)

6. **Outliers Detected (1h)**: Price submissions flagged as statistical outliers
   - **Green**: <5
   - **Yellow**: 5-20
   - **Red**: >20 (possible market volatility or data issues)

#### Price Submission Metrics

- **Price Submission Rate**: Submissions per second by asset
  - Monitors feed health and validator activity
  - Should show steady patterns matching oracle config

- **Aggregated Prices**: Current consensus price for each asset
  - Real-time view of oracle output
  - Compare against known market prices for validation

- **Price Age**: Time since last update per asset
  - **Green**: <60 seconds
  - **Yellow**: 60-300 seconds
  - **Red**: >300 seconds (stale data)

#### Validator Performance

- **Validator Participation Table**: Multi-metric validator view
  - **Submissions/hour**: Activity rate
  - **Reputation**: Score 0-100 (80+ is good)
  - **Missed Votes (24h)**: Participation failures

- **Price Deviation by Asset**: Validator price variance
  - Shows which validators submit outlier prices
  - Used for reputation scoring

- **Slash Events Timeline**: Historical slashing events
  - By validator and reason
  - Helps identify problematic validators

#### TWAP (Time-Weighted Average Price)

- **TWAP Values**: Time-weighted price vs spot price
  - Smooths out manipulation attempts
  - Should track spot price without sharp deviations

- **TWAP Window Size**: Calculation window per asset
  - Typically 5-30 minutes
  - Longer windows = more stable, less responsive

- **Manipulation Detection**: Detected manipulation attempts
  - **Green**: 0
  - **Yellow**: 1-5 per hour
  - **Red**: >5 per hour (investigate)

#### IBC Price Feeds

- **IBC Prices Sent/Received**: Cross-chain price distribution
  - Monitors IBC channel health
  - Should show balanced send/receive for relay chains

- **IBC Timeouts**: Failed IBC transfers
  - **Green**: 0
  - **Yellow**: 1-10 per hour
  - **Red**: >10 per hour (channel issues)

- **IBC Latency**: Cross-chain operation timing
  - p50, p95, p99 latencies
  - Typical: <5s (p95)

### Template Variables

- **Asset**: Filter by specific price pair (e.g., BTC/USD, ETH/USD)
- **Validator**: Filter by specific validator address

### Alert Panels

The following panels have threshold-based alerts:

1. **Avg Price Deviation** - Warns at >2%, critical at >5%
2. **Slash Events** - Warns at 1, critical at 10
3. **Outliers Detected** - Warns at 5, critical at 20
4. **Price Age** - Warns at 60s, critical at 300s
5. **Manipulation Detection** - Warns at 1, critical at 5
6. **IBC Timeouts** - Warns at 1, critical at 10

## Compute Module Dashboard

**Location**: `PAW/Compute` folder in Grafana
**Refresh Rate**: 30 seconds
**Default Time Range**: Last 6 hours

### Purpose

Monitor the Compute module's ZK computation marketplace, including job execution, provider performance, escrow management, and cross-chain compute distribution.

### Key Metrics Explained

#### Top Row - Summary Stats

1. **Active Providers**: Currently registered and active compute providers
   - **Good**: Stable count matching expectations
   - **Concern**: Sudden drops indicate provider failures

2. **Job Queue Size**: Jobs waiting for execution
   - **Green**: <50
   - **Yellow**: 50-200
   - **Red**: >200 (capacity issues)

3. **Jobs Completed (24h)**: Total completed jobs
   - Indicates system throughput
   - Compare against submission rate

4. **Success Rate (24h)**: Percentage of successful job completions
   - **Green**: >98%
   - **Yellow**: 90-98%
   - **Red**: <90% (investigate failures)

5. **Total Escrow Locked**: Funds held in escrow
   - Should correlate with active jobs
   - Monitor for unusual spikes/drops

6. **ZK Proof Verify Rate (1h)**: Proof verification success rate
   - **Green**: >99%
   - **Yellow**: 95-99%
   - **Red**: <95% (proof quality issues)

#### Job Execution Metrics

- **Job Submission Rate**: Jobs submitted per second by type
  - Monitors demand patterns
  - Helps with capacity planning

- **Job Completion Rate**: Completions per provider and status
  - Shows provider activity and success rates
  - Identify underperforming providers

- **Job Execution Time**: Latency distribution (p50, p95, p99)
  - Typical: <60s for most jobs
  - Use for SLA monitoring

- **Job Failure Rate**: Failures by provider and reason
  - **Green**: <0.1 per second
  - **Yellow**: 0.1-1 per second
  - **Red**: >1 per second

#### Provider Performance

- **Provider Performance Table**: Multi-metric provider view
  - **Reputation**: Score 0-100 (80+ is good)
  - **Jobs/hour**: Activity level
  - **Completed (24h)**: Total successful jobs
  - **Stake**: Provider stake amount

- **Provider Reputation Distribution**: Current reputation scores
  - Visualize provider quality distribution
  - Low scores trigger provider rotation

- **Provider Slashing Events**: Slashing for misbehavior
  - **Green**: 0
  - **Yellow**: 1-5 per hour
  - **Red**: >5 per hour

#### ZK Proof Metrics

- **ZK Proof Verification**: Verification rate by proof type and status
  - Monitor proof types (compute, escrow, result)
  - Success vs failure trends

- **Proof Verification Time**: Latency distribution
  - Typical: <1s for most proofs
  - Slower times may indicate complexity issues

- **Invalid Proofs**: Failed verification attempts
  - **Green**: 0-1 per hour
  - **Yellow**: 1-10 per hour
  - **Red**: >10 per hour (investigate provider)

#### Escrow Management

- **Escrow Activity**: Lock, release, and refund rates
  - Should balance over time
  - High refund rate indicates job failures

- **Escrow Balance**: Current locked funds by denomination
  - Monitor for unexpected changes
  - Should match active job count

#### IBC Compute

- **IBC Jobs Distributed**: Jobs sent to remote chains
  - Cross-chain compute distribution
  - Should show active channels

- **IBC Results Received**: Results from remote providers
  - Should correlate with distributed jobs
  - Monitor for timeouts

- **Remote Providers Discovered**: Providers on other chains
  - Indicates IBC channel health
  - Enables cross-chain compute

- **Cross-Chain Latency**: Job execution time via IBC
  - Typical: <60s (p95)
  - Includes network and execution time

- **IBC Timeouts**: Failed cross-chain operations
  - **Green**: 0-1 per hour
  - **Yellow**: 1-10 per hour
  - **Red**: >10 per hour

#### Security Metrics

- **Security Incidents**: Detected security events by type and severity
  - **Green**: 0
  - **Yellow**: 1-5 per hour
  - **Red**: >5 per hour (investigate immediately)

- **Rate Limit Violations**: Users exceeding rate limits
  - Indicates potential abuse
  - Monitor for DDoS patterns

- **Circuit Breaker Triggers**: Automatic safety activations
  - **Green**: 0
  - **Yellow**: 1-5 per hour
  - **Red**: >5 per hour

- **Cleanup Operations**: Maintenance activities
  - Timeout cleanups
  - Stale job cleanups
  - Nonce cleanups
  - State recoveries

- **Panic Recoveries (24h)**: Recovered panic events
  - **Green**: 0
  - **Yellow**: 1-10
  - **Red**: >10 (code quality issues)

### Template Variables

- **Provider**: Filter by specific provider address
- **Job Type**: Filter by job category

### Alert Panels

The following panels have threshold-based alerts:

1. **Job Queue Size** - Warns at 50, critical at 200
2. **Success Rate** - Warns at <98%, critical at <90%
3. **ZK Proof Verify Rate** - Warns at <99%, critical at <95%
4. **Job Failure Rate** - Warns at 0.1/s, critical at 1/s
5. **Invalid Proofs** - Warns at 1, critical at 10
6. **Provider Slashing** - Warns at 1, critical at 5
7. **IBC Timeouts** - Warns at 1, critical at 10
8. **Security Incidents** - Warns at 1, critical at 5
9. **Panic Recoveries** - Warns at 1, critical at 10

## Advanced Modules Overview Dashboard

**Location**: `PAW/Advanced` folder in Grafana
**Refresh Rate**: 30 seconds
**Default Time Range**: Last 6 hours

### Purpose

Provides a high-level overview of both Oracle and Compute modules with side-by-side comparisons and cross-module health monitoring.

### Dashboard Sections

#### Oracle Module Overview (Row 1)

Quick stats for Oracle module:
- Assets Tracked
- Submissions (1h)
- Avg Deviation
- Health Status (success rate)
- Slash Events (24h)
- IBC Activity

#### Compute Module Overview (Row 2)

Quick stats for Compute module:
- Active Providers
- Jobs (1h)
- Queue Size
- Health Status (success rate)
- Escrow Locked
- IBC Activity

#### Side-by-Side Comparison (Row 3)

Comparative visualizations:

1. **Module Activity Rate**: Submission rates for both modules
   - Oracle: Price submissions/sec
   - Compute: Job submissions/sec

2. **Module Success Rates**: Health comparison
   - Both should maintain >98%
   - Divergence indicates module-specific issues

3. **IBC Activity - Cross-Module**: Cross-chain operations
   - Oracle: Price feeds sent/received
   - Compute: Jobs distributed/results received
   - Should show balanced activity

4. **Security Events - Cross-Module**: Combined security view
   - Circuit breakers
   - Slashing events
   - Security incidents
   - Should remain near zero

#### Resource Utilization (Row 4)

System resource monitoring:

1. **Processing Latency Comparison**: Performance benchmarks
   - Oracle: Aggregation latency
   - Compute: Job execution latency
   - Compute: Proof verification latency

2. **Cleanup Operations**: Maintenance activity
   - Oracle: Stale data cleanups
   - Compute: Multiple cleanup types
   - Helps monitor system health

3. **Module Health Summary Table**: Comprehensive comparison
   - Assets/Providers count
   - Operations per hour
   - Success rates
   - Quick health assessment

### Use Cases

1. **Daily Monitoring**: Check overview dashboard for system health
2. **Incident Response**: Identify which module has issues
3. **Performance Comparison**: Understand relative module load
4. **Capacity Planning**: Track growth trends across modules

## Alert Thresholds

### Oracle Module Alerts

| Metric | Warning | Critical | Meaning |
|--------|---------|----------|---------|
| Price Deviation | >2% | >5% | Validators diverging from consensus |
| Price Age | >60s | >300s | Stale price data |
| Slash Events | 1/day | 10/day | Validator misbehavior |
| Outliers Detected | 5/hour | 20/hour | Data quality issues |
| Manipulation Detected | 1/hour | 5/hour | Possible price manipulation |
| Circuit Breakers | 1/hour | 5/hour | Automatic safety triggers |
| IBC Timeouts | 1/hour | 10/hour | Cross-chain failures |
| Validator Participation | <80% | <50% | Low participation rate |

### Compute Module Alerts

| Metric | Warning | Critical | Meaning |
|--------|---------|----------|---------|
| Queue Size | 50 | 200 | Jobs backing up |
| Success Rate | <98% | <90% | High failure rate |
| Job Failure Rate | 0.1/s | 1/s | Jobs failing |
| Invalid Proofs | 1/hour | 10/hour | Proof quality issues |
| Provider Slashing | 1/hour | 5/hour | Provider misbehavior |
| Security Incidents | 1/hour | 5/hour | Security events |
| IBC Timeouts | 1/hour | 10/hour | Cross-chain failures |
| Panic Recoveries | 1/day | 10/day | Code stability issues |
| ZK Verify Rate | <99% | <95% | Proof failures |

### Response Procedures

#### Warning Level (Yellow)
1. Document the event
2. Monitor for trend (increasing/decreasing)
3. Review logs for context
4. Plan intervention if trend continues

#### Critical Level (Red)
1. Alert on-call engineer
2. Investigate root cause immediately
3. Check related metrics for cascade effects
4. Prepare rollback/mitigation plan
5. Document incident for post-mortem

## Customization Guide

### Adding New Panels

1. **Identify the Metric**: Locate metric name in keeper code
   ```go
   // Example from x/oracle/keeper/metrics.go
   paw_oracle_price_submissions_total
   ```

2. **Choose Panel Type**: Based on data type
   - Counter: Use `graph` with `rate()` or `increase()`
   - Gauge: Use `graph` or `stat`
   - Histogram: Use `graph` with `histogram_quantile()`

3. **Write Prometheus Query**:
   ```promql
   # Counter - rate over time
   rate(paw_oracle_price_submissions_total[5m])

   # Gauge - current value
   paw_oracle_assets_tracked_total

   # Histogram - percentiles
   histogram_quantile(0.95, rate(paw_oracle_aggregation_latency_seconds_bucket[5m]))
   ```

4. **Add to Dashboard JSON**: Follow existing panel structure
   ```json
   {
     "id": 99,
     "type": "graph",
     "title": "My New Metric",
     "gridPos": {
       "x": 0,
       "y": 0,
       "w": 12,
       "h": 8
     },
     "targets": [
       {
         "expr": "your_prometheus_query",
         "legendFormat": "{{label}} - Description",
         "refId": "A"
       }
     ]
   }
   ```

### Adjusting Time Ranges

Default time range is 6 hours. To change:

1. **Dashboard-Wide**: Edit `time` object in dashboard JSON
   ```json
   "time": {
     "from": "now-24h",  // Last 24 hours
     "to": "now"
   }
   ```

2. **Per-Panel**: Override in individual panel settings

### Modifying Thresholds

Edit `thresholds` in `fieldConfig`:

```json
"thresholds": {
  "mode": "absolute",
  "steps": [
    {
      "value": null,
      "color": "green"
    },
    {
      "value": 50,      // Warning threshold
      "color": "yellow"
    },
    {
      "value": 100,     // Critical threshold
      "color": "red"
    }
  ]
}
```

### Adding Template Variables

Add to `templating.list` in dashboard JSON:

```json
{
  "name": "my_variable",
  "type": "query",
  "label": "My Label",
  "query": "label_values(metric_name, label_name)",
  "datasource": "Prometheus",
  "multi": true,
  "includeAll": true,
  "allValue": ".*"
}
```

Use in queries with `$my_variable`:
```promql
metric_name{label=~"$my_variable"}
```

### Refresh Intervals

Dashboard auto-refreshes every 30 seconds. To change:

```json
"refresh": "10s"  // Options: "5s", "10s", "30s", "1m", "5m", etc.
```

## Troubleshooting

### Missing Metrics

**Problem**: Panel shows "No data"

**Possible Causes**:
1. Metrics not being exported
2. Prometheus not scraping the target
3. Incorrect metric name in query
4. Time range doesn't contain data

**Solutions**:
1. Check if module is running: `curl http://localhost:26660/metrics | grep paw_oracle`
2. Verify Prometheus targets: `http://prometheus:9090/targets`
3. Test query in Prometheus UI: `http://prometheus:9090/graph`
4. Check metric name matches keeper code exactly
5. Verify time range includes recent data

### Incorrect Values

**Problem**: Metrics show but values seem wrong

**Possible Causes**:
1. Wrong aggregation function
2. Missing `by` clause in grouping
3. Counter vs gauge confusion
4. Unit mismatch

**Solutions**:
1. Verify metric type (counter, gauge, histogram)
2. Add appropriate labels: `sum(...) by (label_name)`
3. Use `rate()` for counters, direct query for gauges
4. Check unit in `fieldConfig.defaults.unit`

### Query Timeouts

**Problem**: Panel shows timeout error

**Possible Causes**:
1. Too large time range
2. High cardinality labels
3. Complex aggregations
4. Prometheus under load

**Solutions**:
1. Reduce time range or add `[5m]` range selector
2. Limit label cardinality: `topk(10, ...)`
3. Simplify query or add recording rules
4. Increase Prometheus resources

### Dashboard Not Updating

**Problem**: Dashboard doesn't auto-refresh

**Possible Causes**:
1. Refresh interval too long
2. Browser tab backgrounded
3. Grafana connection issues

**Solutions**:
1. Check `refresh` setting in dashboard JSON
2. Keep browser tab active
3. Verify Grafana is running: `docker ps | grep grafana`
4. Check Grafana logs: `docker logs grafana`

### Template Variables Not Working

**Problem**: Variables don't populate or filter

**Possible Causes**:
1. Incorrect query syntax
2. No matching labels
3. Wrong datasource

**Solutions**:
1. Test variable query in Prometheus UI
2. Verify labels exist: `label_values(metric_name, label_name)`
3. Ensure datasource is set to "Prometheus"
4. Check variable is used in panel: `{label=~"$variable"}`

### Panels Overlapping

**Problem**: Panels display on top of each other

**Possible Causes**:
1. Incorrect `gridPos` values
2. Panel width exceeds 24 units
3. JSON formatting errors

**Solutions**:
1. Verify `gridPos.x + gridPos.w <= 24`
2. Ensure `gridPos.y` values don't overlap
3. Validate JSON syntax
4. Use Grafana UI to rearrange, then export JSON

## Architecture Reference

### Design Patterns from Blockchain Control Center

These dashboards follow established patterns from the existing control center:

1. **Metric Naming**: `paw_<module>_<metric>_<type>`
   - Example: `paw_oracle_price_submissions_total`
   - Consistent with DEX: `paw_dex_trades_total`

2. **Panel Layout**: 24-unit grid system
   - Top row: Key stats (4-6 units each)
   - Body: Graphs (12 units each, side-by-side)
   - Bottom: Tables and detailed views

3. **Color Scheme**: Traffic light system
   - Green: Healthy/Normal
   - Yellow: Warning/Caution
   - Red: Critical/Alert

4. **Time Ranges**: Consistent across dashboards
   - Default: 6 hours for operational dashboards
   - DEX: 24 hours for trading analytics
   - Use same pattern for module type

5. **Refresh Rates**: Based on data volatility
   - Fast (10s): Blockchain consensus metrics
   - Medium (30s): Module operations (Oracle, Compute)
   - Slow (1m): Aggregated statistics

6. **Query Patterns**: Prometheus best practices
   ```promql
   # Rate for counters
   rate(metric_total[5m])

   # Increase for absolute counts
   increase(metric_total[1h])

   # Histogram percentiles
   histogram_quantile(0.95, rate(metric_bucket[5m]))

   # Aggregation with grouping
   sum(rate(metric[5m])) by (label)
   ```

7. **Template Variables**: Filter without rewrite
   - `{label=~"$variable"}` for multi-select
   - `includeAll: true` with `allValue: ".*"`
   - Match label names to actual metric labels

### Metric Collection Architecture

```
┌─────────────────┐
│  Keeper Code    │ (metrics.go files)
│  - NewMetrics() │
│  - Record...()  │
└────────┬────────┘
         │
         │ Prometheus Client
         │
┌────────▼────────┐
│  /metrics       │ (HTTP endpoint :26660)
│  Endpoint       │
└────────┬────────┘
         │
         │ Scrape every 15s
         │
┌────────▼────────┐
│  Prometheus     │ (Time-series DB)
│  - Storage      │
│  - Queries      │
└────────┬────────┘
         │
         │ PromQL Queries
         │
┌────────▼────────┐
│  Grafana        │ (Visualization)
│  - Dashboards   │
│  - Alerts       │
└─────────────────┘
```

### Dashboard Provisioning Flow

```
1. Dashboard JSON created in:
   /home/hudson/blockchain-projects/paw/infra/grafana/dashboards/

2. Provisioning config updated:
   /home/hudson/blockchain-projects/paw/infra/grafana/provisioning/dashboards/dashboard.yml

3. Docker volume mount:
   - Host: ./infra/grafana/dashboards/
   - Container: /var/lib/grafana/dashboards/

4. Grafana startup:
   - Reads provisioning config
   - Loads dashboard JSON files
   - Creates folders and dashboards

5. Auto-updates:
   - Checks every 10 seconds (updateIntervalSeconds)
   - Reloads changed files
   - Preserves UI customizations (allowUiUpdates: true)
```

### Related Components

- **Prometheus Config**: `/home/hudson/blockchain-projects/paw/infra/prometheus/prometheus.yml`
- **Metrics Keeper Code**:
  - `/home/hudson/blockchain-projects/paw/x/oracle/keeper/metrics.go`
  - `/home/hudson/blockchain-projects/paw/x/compute/keeper/metrics.go`
- **Grafana Datasource**: `/home/hudson/blockchain-projects/paw/infra/grafana/provisioning/datasources/prometheus.yml`
- **Existing Dashboards**:
  - Blockchain: `/home/hudson/blockchain-projects/paw/infra/grafana/dashboards/blockchain-metrics.json`
  - DEX: `/home/hudson/blockchain-projects/paw/infra/grafana/dashboards/dex-metrics.json`
  - Node: `/home/hudson/blockchain-projects/paw/infra/grafana/dashboards/node-metrics.json`

## Best Practices

1. **Start with Overview Dashboard**: Use Advanced Modules dashboard for daily monitoring
2. **Drill Down on Issues**: Switch to module-specific dashboard when anomalies detected
3. **Use Template Variables**: Filter by specific assets/providers when investigating
4. **Monitor Trends**: Look for gradual changes, not just absolute values
5. **Compare Across Time**: Use time range selector to compare current vs historical
6. **Correlate Metrics**: Check related panels when investigating issues
7. **Document Anomalies**: Use Grafana annotations to mark incidents
8. **Review Regularly**: Schedule daily/weekly dashboard reviews
9. **Update Thresholds**: Adjust alert levels based on operational experience
10. **Share Insights**: Export and share dashboard snapshots with team

## Additional Resources

- [Prometheus Query Documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/)
- [PAW Oracle Module Documentation](../x/oracle/README.md)
- [PAW Compute Module Documentation](../x/compute/README.md)
- [Blockchain Control Center Reference](../infra/grafana/dashboards/) (existing dashboard examples)

---

**Last Updated**: 2025-12-14
**Version**: 1.0
**Maintainer**: PAW DevOps Team
