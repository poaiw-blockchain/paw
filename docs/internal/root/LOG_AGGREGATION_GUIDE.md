# PAW Blockchain - Log Aggregation Guide

## Overview

This guide covers the centralized logging infrastructure for PAW blockchain using **Loki** and **Promtail**. Loki provides scalable, cost-effective log aggregation, while Promtail collects logs from all PAW services.

## Architecture

```
┌─────────────────────┐
│  Docker Containers  │
│  (PAW Services)     │
└──────────┬──────────┘
           │ JSON logs
           ▼
    ┌──────────────┐
    │   Promtail   │ ← Collects logs from containers
    └──────┬───────┘
           │ Push logs
           ▼
     ┌─────────────┐
     │    Loki     │ ← Stores and indexes logs
     └──────┬──────┘
            │ Query logs
            ▼
      ┌──────────┐
      │ Grafana  │ ← Visualize and search logs
      └──────────┘
```

### Components

- **Loki (port 11101)**: Log aggregation database with 30-day retention
- **Promtail**: Log collection agent that scrapes Docker container logs
- **Grafana**: Visualization and log exploration interface

## Deployment

### Start Logging Stack

```bash
cd /home/hudson/blockchain-projects/paw

# Start Loki and Promtail
docker compose -f compose/docker-compose.logging.yml up -d

# Verify services are running
docker ps --filter "name=paw-loki" --filter "name=paw-promtail"
```

### Check Health

```bash
# Loki health
curl http://localhost:11101/ready

# Promtail metrics
curl http://localhost:9080/metrics

# Check Loki is receiving logs
curl http://localhost:11101/loki/api/v1/labels
```

### Stop Logging Stack

```bash
docker compose -f compose/docker-compose.logging.yml down
```

## Log Collection

### How Logs Are Collected

Promtail uses Docker service discovery to automatically find and scrape logs from:

1. **PAW Nodes** (`paw-node1`, `paw-node2`, etc.) - Blockchain consensus logs
2. **Monitoring Services** (`paw-prometheus`, `paw-grafana`, etc.)
3. **Dashboard Services** (`paw-staking-dashboard`, etc.)
4. **System Logs** (`/var/log/syslog`)

### Labels Applied to Logs

Each log entry is tagged with labels for filtering:

- `container_name` - Docker container name (e.g., `paw-node1`)
- `container_id` - Short container ID
- `service_name` - Docker Compose service name
- `project` - Project name (`paw`)
- `host` - Host machine (`bcpc`)
- `job` - Job category (`paw-docker`, `paw-blockchain`, etc.)
- `level` - Log level (ERROR, WARN, INFO, DEBUG) if available
- `node` - Node identifier for blockchain nodes

## LogQL Query Language

LogQL is Loki's query language, similar to PromQL but for logs.

### Basic Syntax

```logql
{label="value"} |= "search text"
```

### Query Components

1. **Log Stream Selector**: `{container_name="paw-node1"}`
2. **Line Filter**: `|= "error"` (contains), `!= "debug"` (not contains)
3. **Regex Filter**: `|~ "(?i)error"` (case-insensitive regex)
4. **JSON Parser**: `| json | line_format "{{.level}}"`
5. **Aggregation**: `count_over_time({...}[5m])`

## Common Log Queries

### Find All Errors

```logql
{container_name=~"paw-.*"} |~ "(?i)(error|fail|exception|fatal|panic)"
```

### Errors from Specific Service

```logql
{container_name="paw-node1"} |~ "(?i)error"
```

### Consensus Logs

```logql
{container_name=~"paw-node.*"} |~ "(?i)consensus"
```

### DEX Transactions

```logql
{container_name=~"paw-node.*"} |~ "(?i)dex"
```

### Oracle Price Feeds

```logql
{container_name=~"paw-node.*"} |~ "(?i)oracle"
```

### Logs for Specific Transaction Hash

```logql
{container_name=~"paw-.*"} |~ "tx_hash=[A-Fa-f0-9]{64}"
```

### Log Count by Service (Last Hour)

```logql
sum by (container_name) (count_over_time({container_name=~"paw-.*"}[1h]))
```

### Error Rate Over Time

```logql
sum by (level) (count_over_time({container_name=~"paw-.*"} | json | level=~"ERROR|WARN" [5m]))
```

## Using the CLI Query Tool

The `scripts/query-logs.sh` script provides easy CLI access to logs.

### Basic Usage

```bash
# Show all errors in last hour
./scripts/query-logs.sh errors

# Show errors in last 24 hours
./scripts/query-logs.sh errors 24

# Show warnings
./scripts/query-logs.sh warnings 6

# Find logs for specific transaction
./scripts/query-logs.sh tx A1B2C3D4E5F6...

# Get logs from specific service
./scripts/query-logs.sh service paw-node1 2

# Consensus logs
./scripts/query-logs.sh consensus 1

# DEX transaction logs
./scripts/query-logs.sh dex 1

# Oracle logs
./scripts/query-logs.sh oracle 1

# Custom LogQL query
./scripts/query-logs.sh query '{container_name="paw-node1"} |~ "height"'

# List all labels
./scripts/query-logs.sh labels

# List all container names
./scripts/query-logs.sh values container_name

# Export logs to file
./scripts/query-logs.sh export paw-node1 24 node1-logs.txt
```

## Grafana Integration

### Accessing the Dashboard

1. Open Grafana: http://localhost:11030
2. Login with credentials (default: admin/grafana_secure_password)
3. Navigate to **Dashboards** → **PAW - Log Aggregation**

### Dashboard Panels

The log aggregation dashboard includes:

1. **Log Volume by Service** - Time series of log rates per container
2. **Error Rate Over Time** - Errors/warnings trend
3. **Log Level Distribution** - Pie chart of ERROR/WARN/INFO/DEBUG
4. **Recent Errors** - Live feed of error logs
5. **Consensus Logs** - Filtered view of consensus activity
6. **DEX Transaction Logs** - DEX-related logs
7. **Statistics** - Total errors, warnings, log volume
8. **All Service Logs** - Unfiltered log stream

### Exploring Logs in Grafana

1. Click **Explore** in left sidebar
2. Select **Loki** as datasource
3. Enter LogQL query (e.g., `{container_name="paw-node1"}`)
4. Click **Run query**
5. Use filters and labels to refine search

### Creating Custom Queries

Example: Find all failed transaction logs from last 5 minutes:

```logql
{container_name=~"paw-node.*"}
  | json
  | line_format "{{.msg}}"
  |~ "(?i)(tx.*fail|transaction.*error)"
```

## Log Retention and Storage

### Retention Policy

- **Default retention**: 30 days (720 hours)
- **Compaction interval**: 10 minutes
- **Deletion delay**: 2 hours after retention expiration

### Storage Configuration

- **Storage backend**: Filesystem (local Docker volume)
- **Index format**: boltdb-shipper
- **Chunk size**: 1.5 MB (compressed with snappy)
- **Chunk retention**: Flushed after 5 minutes idle or 1 hour max age

### Managing Storage

```bash
# Check Loki data volume size
docker volume inspect compose_loki-data | jq '.[0].Mountpoint' | xargs sudo du -sh

# Clean up old logs (adjust retention in loki-config.yml)
# Restart Loki to apply changes
docker compose -f compose/docker-compose.logging.yml restart loki
```

## Troubleshooting

### Loki Not Starting

```bash
# Check Loki logs
docker logs paw-loki

# Common issues:
# - Port 11101 already in use
# - Config file permissions (should be 644)
# - Volume mount errors
```

### Promtail Not Collecting Logs

```bash
# Check Promtail logs
docker logs paw-promtail

# Verify Docker socket access
docker exec paw-promtail ls -la /var/run/docker.sock

# Check Promtail targets
curl http://localhost:9080/targets
```

### No Logs in Grafana

1. Verify Loki datasource is configured:
   - Grafana → Configuration → Data Sources → Loki
   - URL should be: `http://paw-loki:3100`

2. Check Loki has data:
   ```bash
   curl http://localhost:11101/loki/api/v1/labels
   ```

3. Verify log query syntax in Grafana Explore

### High Memory Usage

Loki memory usage can be adjusted in `loki-config.yml`:

```yaml
ingester:
  wal:
    replay_memory_ceiling: 1GB  # Reduce if needed

limits_config:
  max_entries_limit_per_query: 5000  # Reduce for less memory
```

Restart after changes:
```bash
docker compose -f compose/docker-compose.logging.yml restart loki
```

## Performance Tuning

### Optimize Log Volume

Reduce noise in logs by filtering healthcheck messages (already configured in Promtail):

```yaml
- match:
    selector: '{container_name=~"paw-.*"} |~ "healthcheck"'
    action: drop
```

### Query Performance

- Use specific label filters: `{container_name="paw-node1"}` instead of `{container_name=~"paw-.*"}`
- Limit time range for large queries
- Use `count_over_time` for aggregations instead of pulling all logs

### Indexing Strategy

For high-volume environments, consider:
- Increasing `chunk_target_size` for fewer chunks
- Adjusting `max_chunk_age` for less frequent flushes
- Tuning `split_queries_by_interval` for query parallelization

## API Reference

### Loki HTTP API

Base URL: `http://localhost:11101`

#### Query Logs (Range)
```bash
curl -G "http://localhost:11101/loki/api/v1/query_range" \
  --data-urlencode "query={container_name=\"paw-node1\"}" \
  --data-urlencode "limit=1000"
```

#### Query Logs (Instant)
```bash
curl -G "http://localhost:11101/loki/api/v1/query" \
  --data-urlencode "query={container_name=\"paw-node1\"}"
```

#### Get Labels
```bash
curl http://localhost:11101/loki/api/v1/labels
```

#### Get Label Values
```bash
curl http://localhost:11101/loki/api/v1/label/container_name/values
```

#### Push Logs (Manual)
```bash
curl -X POST "http://localhost:11101/loki/api/v1/push" \
  -H "Content-Type: application/json" \
  -d '{
    "streams": [
      {
        "stream": {"container_name": "test"},
        "values": [["'$(date +%s%N)'", "Test log message"]]
      }
    ]
  }'
```

### Promtail API

Base URL: `http://localhost:9080`

#### Health Check
```bash
curl http://localhost:9080/ready
```

#### Metrics
```bash
curl http://localhost:9080/metrics
```

#### Current Targets
```bash
curl http://localhost:9080/targets
```

## Advanced Use Cases

### Log-Based Alerting

Create alerting rules in `loki-config.yml`:

```yaml
ruler:
  enable_api: true
  alertmanager_url: http://paw-alertmanager:9093
```

Example alert rule (create in `/loki/rules/`):

```yaml
groups:
  - name: paw_alerts
    interval: 1m
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate({container_name=~"paw-.*"} |~ "(?i)error" [5m])) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
```

### Structured Logging Best Practices

For optimal Loki integration, emit JSON logs:

```json
{
  "level": "error",
  "msg": "Transaction failed",
  "tx_hash": "A1B2C3D4...",
  "height": 12345,
  "module": "dex"
}
```

Promtail will automatically parse JSON and extract fields as labels.

### Correlation with Metrics

Link logs to Prometheus metrics in Grafana:
1. Add data links in metric panels
2. Use label matching (e.g., `container_name`)
3. Jump from metric spike to relevant logs

## Configuration Files

### Loki Config
Location: `compose/docker/logging/loki-config.yml`

Key sections:
- `ingester`: Chunk management and WAL
- `schema_config`: Index structure
- `storage_config`: Filesystem backend
- `limits_config`: Rate limiting and retention
- `compactor`: Log cleanup

### Promtail Config
Location: `compose/docker/logging/promtail-config.yml`

Key sections:
- `scrape_configs`: Log sources (Docker, files)
- `pipeline_stages`: Log parsing and labeling
- `relabel_configs`: Label transformations

### Docker Compose
Location: `compose/docker-compose.logging.yml`

Services:
- `loki`: Log aggregation server
- `promtail`: Log collection agent

## Security Considerations

1. **Authentication**: Loki runs without auth by default (single-tenant mode)
   - For production, enable multi-tenancy and auth
   - Use reverse proxy (Nginx) with basic auth

2. **Network Isolation**: Loki and Promtail run on isolated `paw-logging` network

3. **Log Sensitivity**: Be aware logs may contain sensitive data
   - Filter secrets in Promtail pipeline
   - Limit access to Grafana

4. **Resource Limits**: Set memory/CPU limits in docker-compose to prevent resource exhaustion

## Monitoring Loki Itself

Monitor Loki health with these metrics (scraped by Prometheus):

- `loki_ingester_chunks_created_total` - Chunk creation rate
- `loki_ingester_bytes_received_total` - Ingestion throughput
- `loki_request_duration_seconds` - Query performance
- `loki_distributor_lines_received_total` - Log lines ingested

Add Loki as Prometheus target in `prometheus.yml`:
```yaml
- job_name: 'loki'
  static_configs:
    - targets: ['paw-loki:3100']
```

## Support and Resources

- **Loki Documentation**: https://grafana.com/docs/loki/latest/
- **LogQL Reference**: https://grafana.com/docs/loki/latest/logql/
- **Promtail Pipelines**: https://grafana.com/docs/loki/latest/clients/promtail/pipelines/

## Summary

- **Loki** provides scalable, cost-effective log aggregation for all PAW services
- **Promtail** automatically discovers and collects logs from Docker containers
- **Grafana** offers powerful visualization and exploration
- **CLI tool** (`query-logs.sh`) enables quick log analysis
- **30-day retention** with automatic compaction and cleanup
- **LogQL** provides flexible querying with grep-like syntax

For questions or issues, check container logs:
```bash
docker logs paw-loki
docker logs paw-promtail
```
