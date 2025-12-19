# PAW Blockchain Monitoring & Logging

Complete observability stack for PAW blockchain with metrics, logs, traces, and dashboards.

## Stack Components

### Metrics & Dashboards
- **Prometheus** (port 9091): Time-series metrics database
- **Grafana** (port 11030): Visualization and dashboards
- **Node Exporter** (port 9100): System metrics
- **cAdvisor** (port 11082): Container metrics
- **Alertmanager** (port 9093): Alert management

### Logging
- **Loki** (port 11025): Log aggregation database
- **Promtail**: Log collection agent (scrapes Docker containers)

### Tracing
- **Jaeger** (port 16686): Distributed tracing UI
- **OpenTelemetry**: Trace collection via OTLP (ports 4318 HTTP, 11317 gRPC)

## Quick Start

### Start Full Stack
```bash
cd /home/hudson/blockchain-projects/paw/compose

# Start monitoring (Prometheus + Grafana)
docker-compose -f docker-compose.monitoring.yml up -d

# Start logging (Loki + Promtail)
./scripts/monitoring/deploy_logging_stack.sh        # up/down/status helper

# Start tracing (Jaeger)
docker-compose -f docker-compose.tracing.yml up -d
```

### Access URLs
- **Grafana**: http://localhost:11030 (admin/grafana_secure_password)
- **Prometheus**: http://localhost:9091
- **Loki**: http://localhost:11025
- **Jaeger**: http://localhost:16686
- **Alertmanager**: http://localhost:9093 (routes include `service=ibc` boundary alerts)

## Loki Log Aggregation

### Architecture
Loki collects logs from all PAW Docker containers via Promtail:
- Automatic Docker container discovery
- Label-based log organization
- 30-day retention period
- Compressed storage with filesystem backend

### Log Collection
Promtail automatically scrapes:
1. **PAW Nodes** - Blockchain consensus and state logs
2. **Monitoring Services** - Prometheus, Grafana, Alertmanager
3. **Dashboard Services** - Web UIs and portals
4. **System Logs** - /var/log/syslog

### Query Examples

#### View All Logs
```logql
{job="paw-docker"}
```

#### Blockchain Node Logs
```logql
{container_name=~"paw-node.*"}
```

#### Error Logs Only
```logql
{job="paw-docker"} |= "ERROR"
```

#### Specific Service
```logql
{service_name="prometheus"}
```

#### Transaction Logs
```logql
{job="paw-blockchain"} |~ "tx_hash=[A-Fa-f0-9]{64}"
```

#### Consensus State
```logql
{container_name=~"paw-node.*"} |~ "height=\\d+ round=\\d+"
```

#### Rate of Errors (5min)
```logql
rate({job="paw-docker"} |= "ERROR" [5m])
```

#### Log Pattern Matching
```logql
{job="paw-docker"} |~ "(?i)(panic|fatal|critical)"
```

### Advanced Queries

#### Count Errors by Container
```logql
sum by(container_name) (count_over_time({job="paw-docker"} |= "ERROR" [1h]))
```

#### Filter by Log Level
```logql
{job="paw-docker"} | json | level="ERROR"
```

#### Logs from Last Hour with Context
```logql
{container_name="paw-node1"} |= "validator" [1h]
```

## Grafana Integration

### Datasources
Pre-configured datasources:
1. **Prometheus** (default) - Metrics
2. **Loki** - Logs
3. **Jaeger** - Traces

### Explore Logs
1. Open Grafana: http://localhost:11030
2. Navigate to **Explore**
3. Select **Loki** datasource
4. Enter LogQL query
5. Set time range and run query

### Create Log Dashboard
1. Go to **Dashboards** → **New Dashboard**
2. Add **Logs** panel
3. Select **Loki** datasource
4. Configure query and labels
5. Save dashboard

## Configuration Files

```
compose/
├── docker-compose.logging.yml       # Loki stack
├── docker-compose.monitoring.yml    # Prometheus + Grafana
└── docker-compose.tracing.yml       # Jaeger

compose/docker/
├── logging/
│   ├── loki-config.yml             # Loki server config
│   └── promtail-config.yml         # Log collection config
└── monitoring/grafana/
    └── datasources/
        ├── prometheus.yml           # Prometheus datasource
        └── loki.yml                # Loki datasource
```

## Troubleshooting

### Check Service Status
```bash
docker ps | grep paw-loki
docker ps | grep paw-promtail
docker logs paw-loki
docker logs paw-promtail
```

### Verify Loki is Receiving Logs
```bash
curl http://localhost:11025/ready
curl http://localhost:11025/metrics
```

### Test Log Query
```bash
curl -G -s "http://localhost:11025/loki/api/v1/query" \
  --data-urlencode 'query={job="paw-docker"}' \
  --data-urlencode 'limit=10' | jq
```

### Restart Services
```bash
cd /home/hudson/blockchain-projects/paw/compose
docker-compose -f docker-compose.logging.yml restart
```

## Performance

### Storage
- **Retention**: 30 days
- **Compression**: Snappy
- **Max ingestion**: 10 MB/s per distributor
- **Max line size**: 256 KB

### Limits
- Max 5000 global streams per tenant
- Max 5000 entries per query
- Max 30 days query range
- Max 2M chunks per query

## Maintenance

### View Storage Usage
```bash
docker exec paw-loki du -sh /loki/*
```

### Clear Old Logs (if needed)
```bash
docker-compose -f docker-compose.logging.yml down
docker volume rm compose_loki-data
docker-compose -f docker-compose.logging.yml up -d
```

### Update Configuration
```bash
# Edit config
vim compose/docker/logging/loki-config.yml
vim compose/docker/logging/promtail-config.yml

# Restart to apply
docker-compose -f docker-compose.logging.yml restart
```
