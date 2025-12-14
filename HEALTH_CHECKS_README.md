# PAW Health Checks - Quick Reference

## Quick Start

### Check All Services

```bash
./scripts/health-check-all.sh
```

### Check Individual Services

```bash
# Main daemon
curl http://localhost:36661/health
curl http://localhost:36661/health/ready
curl http://localhost:36661/health/detailed | jq

# Explorer
curl http://localhost:11080/health
curl http://localhost:11080/health/ready
curl http://localhost:11080/health/detailed | jq
```

## Health Check Endpoints

| Service | Port | Liveness | Readiness | Detailed |
|---------|------|----------|-----------|----------|
| pawd | 36661 | `/health` | `/health/ready` | `/health/detailed` |
| Explorer | 11080 | `/health` | `/health/ready` | `/health/detailed` |
| Prometheus | 9091 | `/-/healthy` | `/-/ready` | - |
| Grafana | 11030 | `/api/health` | - | - |
| Alertmanager | 9093 | `/-/healthy` | `/-/ready` | - |

## Endpoint Usage

### `/health` - Liveness Probe
- **Purpose**: Check if process is alive
- **Returns**: Always 200 if running
- **Use for**: Kubernetes liveness probe, load balancer checks
- **Frequency**: Every 30 seconds

### `/health/ready` - Readiness Probe
- **Purpose**: Check if can handle traffic
- **Returns**: 200 if ready, 503 if not
- **Checks**: Database, RPC, sync status, consensus
- **Use for**: Kubernetes readiness probe, routing decisions
- **Frequency**: Every 10 seconds

### `/health/detailed` - Detailed Health
- **Purpose**: Comprehensive health and metrics
- **Returns**: Always 200 (check status field)
- **Includes**: All checks + module metrics + system metrics
- **Use for**: Debugging, dashboards, operations
- **Frequency**: Every 60 seconds or on-demand
- **Cached**: 5 seconds

### `/health/startup` - Startup Probe
- **Purpose**: Check if initialization is complete
- **Returns**: 503 for first 30 seconds, then same as ready
- **Use for**: Kubernetes startup probe
- **Frequency**: Every 10 seconds

## Response Examples

### Healthy Response (`/health/ready`)
```json
{
  "status": "ready",
  "checks": {
    "rpc": {"status": "ok"},
    "sync": {"status": "ok"},
    "consensus": {"status": "ok"}
  }
}
```

### Not Ready Response (`/health/ready`)
```json
{
  "status": "not_ready",
  "checks": {
    "rpc": {"status": "ok"},
    "sync": {
      "status": "syncing",
      "message": "catching up at height 12345"
    },
    "consensus": {"status": "ok"}
  }
}
```

### Detailed Response (`/health/detailed`)
```json
{
  "status": "healthy",
  "uptime_seconds": 3600,
  "version": "v1.0.0",
  "checks": { ... },
  "modules": {
    "dex": {"status": "ok", "metrics": {"pools": 10}},
    "oracle": {"status": "ok", "metrics": {"price_pairs": 15}},
    "compute": {"status": "ok", "metrics": {"active_providers": 3}}
  },
  "system": {
    "memory_mb": 512,
    "goroutines": 150,
    "peers": 10,
    "block_height": 12345
  }
}
```

## Kubernetes Configuration

### Basic Probes
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: health
  periodSeconds: 30
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health/ready
    port: health
  periodSeconds: 10
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /health/startup
    port: health
  periodSeconds: 10
  failureThreshold: 60
```

See [deployments/kubernetes/](deployments/kubernetes/) for complete examples.

## Monitoring

### Prometheus Metrics
```
paw_health_check_total{endpoint, status}
paw_health_check_duration_seconds{endpoint}
paw_service_healthy{service}
```

### Query Examples
```promql
# Service health status
paw_service_healthy

# Health check failure rate
rate(paw_health_check_total{status!="200"}[5m])

# P99 health check latency
histogram_quantile(0.99, rate(paw_health_check_duration_seconds_bucket[5m]))
```

## Troubleshooting

### Service Returning 503

1. Check detailed health:
   ```bash
   curl http://localhost:36661/health/detailed | jq
   ```

2. Look at the `checks` section to see which failed

3. Common issues:
   - **RPC failing**: Check if node is running
   - **Sync syncing**: Node is catching up (expected)
   - **Consensus degraded**: Non-validator (safe to ignore)

### Health Checks Slow

1. Check duration metrics:
   ```bash
   curl http://localhost:36660/metrics | grep health_check_duration
   ```

2. Common causes:
   - Database slow
   - RPC overloaded
   - Network latency

3. Solutions:
   - Optimize database queries
   - Increase timeouts
   - Check system resources

### Liveness Probe Restarting Pod

1. Add/tune startup probe to allow initialization time
2. Increase liveness `periodSeconds` and `failureThreshold`
3. Check if service is actually unhealthy

## CI/CD Integration

### Wait for Service to be Ready
```bash
timeout 300 bash -c '
  until curl -f http://localhost:36661/health/ready >/dev/null 2>&1; do
    echo "Waiting for service..."
    sleep 5
  done
'
```

### Health Gate Before Deployment
```bash
if ! curl -f http://localhost:36661/health/ready >/dev/null 2>&1; then
  echo "Service unhealthy, aborting deployment"
  exit 1
fi
```

## Documentation

For complete documentation, see [HEALTH_CHECKS_GUIDE.md](HEALTH_CHECKS_GUIDE.md).

Topics covered:
- Detailed endpoint specifications
- Response format reference
- Kubernetes integration guide
- Monitoring and alerting setup
- Comprehensive troubleshooting guide
- Best practices

## Files

- `cmd/pawd/health.go` - Health check server implementation
- `cmd/pawd/node_checker.go` - Node health checker
- `cmd/pawd/health_test.go` - Health check tests
- `explorer/indexer/internal/api/server.go` - Explorer health endpoints
- `scripts/health-check-all.sh` - Health check aggregator
- `deployments/kubernetes/*.yaml` - Kubernetes manifests with probes
- `deployments/prometheus/health-alerts.yml` - Prometheus alerting rules
- `deployments/grafana/health-dashboard.json` - Grafana dashboard

## Testing

```bash
# Run health check tests
go test ./cmd/pawd -v -run TestHealthCheck

# Test health check script
./scripts/health-check-all.sh

# Manual testing
curl -v http://localhost:36661/health
curl -v http://localhost:36661/health/ready
curl -v http://localhost:36661/health/detailed | jq
curl -v http://localhost:36661/health/startup
```
