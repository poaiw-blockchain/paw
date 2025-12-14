# PAW Health Checks Guide

This guide documents the comprehensive health check system implemented for all PAW services.

## Table of Contents

1. [Overview](#overview)
2. [Health Check Endpoints](#health-check-endpoints)
3. [Service Endpoints](#service-endpoints)
4. [Response Formats](#response-formats)
5. [Kubernetes Integration](#kubernetes-integration)
6. [Monitoring and Alerting](#monitoring-and-alerting)
7. [Troubleshooting](#troubleshooting)

## Overview

PAW implements a comprehensive health check system following Kubernetes best practices:

- **Liveness probes**: Check if the service is alive (restart if fails)
- **Readiness probes**: Check if the service can handle traffic (remove from load balancer if fails)
- **Startup probes**: Give services time to initialize before running other checks

All health checks are exposed via HTTP endpoints and integrate with Prometheus for monitoring.

## Health Check Endpoints

### 1. Basic Liveness Check

**Endpoint**: `GET /health`

**Purpose**: Check if the process is alive

**Use Case**: Kubernetes liveness probe, load balancer health check

**Success Criteria**: Process is running

**Response**:
```json
{
  "status": "ok",
  "timestamp": "2025-12-14T12:34:56Z"
}
```

**HTTP Status Codes**:
- `200 OK`: Service is alive

**Characteristics**:
- Always returns 200 if process is alive
- Very fast (<10ms)
- No external dependencies checked
- Safe to call frequently (every 5 seconds)

### 2. Readiness Check

**Endpoint**: `GET /health/ready`

**Purpose**: Check if the service can handle traffic

**Use Case**: Kubernetes readiness probe, load balancer routing decisions

**Success Criteria**:
- Database connection is healthy
- RPC connectivity is working
- Node is not syncing (caught up)
- Consensus is participating (for validators)

**Response** (Healthy):
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

**Response** (Not Ready):
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

**HTTP Status Codes**:
- `200 OK`: Service is ready
- `503 Service Unavailable`: Service is not ready

**Characteristics**:
- Checks critical dependencies
- Returns 503 if any check fails
- May take up to 5 seconds
- Should be called every 10-30 seconds

### 3. Detailed Health Check

**Endpoint**: `GET /health/detailed`

**Purpose**: Comprehensive health and metrics information

**Use Case**: Debugging, monitoring dashboards, operations

**Success Criteria**: Aggregates all health checks plus additional metrics

**Response**:
```json
{
  "status": "healthy",
  "uptime_seconds": 3600,
  "version": "v1.0.0",
  "checks": {
    "rpc": {"status": "ok"},
    "sync": {"status": "ok"},
    "consensus": {"status": "ok"}
  },
  "modules": {
    "dex": {
      "status": "ok",
      "metrics": {
        "pools": 10,
        "volume_24h": 1000000
      }
    },
    "oracle": {
      "status": "ok",
      "metrics": {
        "active_validators": 7,
        "price_pairs": 15
      }
    },
    "compute": {
      "status": "ok",
      "metrics": {
        "active_providers": 3,
        "pending_requests": 5
      }
    }
  },
  "system": {
    "memory_mb": 512,
    "goroutines": 150,
    "peers": 10,
    "block_height": 12345
  }
}
```

**HTTP Status Codes**:
- `200 OK`: Always returns 200 (status is in response body)

**Characteristics**:
- Includes all readiness checks
- Adds module-specific metrics
- Adds system metrics (memory, goroutines, peers)
- Cached for 5 seconds to avoid overload
- May take up to 5 seconds
- Should be called sparingly (every 60 seconds or on-demand)

### 4. Startup Probe

**Endpoint**: `GET /health/startup`

**Purpose**: Check if the service has finished initializing

**Use Case**: Kubernetes startup probe (delays other probes until service is ready)

**Success Criteria**:
- Service has been running for at least 30 seconds
- All readiness checks pass

**Response**: Same as readiness check

**HTTP Status Codes**:
- `200 OK`: Service is fully started
- `503 Service Unavailable`: Service is still starting

**Characteristics**:
- Returns 503 for first 30 seconds
- Then delegates to readiness check
- Prevents premature liveness probe failures
- Should be called every 10 seconds

## Service Endpoints

### Main Daemon (pawd)

- **Port**: 36661
- **Liveness**: http://localhost:36661/health
- **Readiness**: http://localhost:36661/health/ready
- **Detailed**: http://localhost:36661/health/detailed
- **Startup**: http://localhost:36661/health/startup

### Explorer API

- **Port**: 11080
- **Liveness**: http://localhost:11080/health
- **Readiness**: http://localhost:11080/health/ready
- **Detailed**: http://localhost:11080/health/detailed

### Prometheus

- **Port**: 9091
- **Health**: http://localhost:9091/-/healthy
- **Ready**: http://localhost:9091/-/ready

### Grafana

- **Port**: 11030
- **Health**: http://localhost:11030/api/health

### Alertmanager

- **Port**: 9093
- **Health**: http://localhost:9093/-/healthy
- **Ready**: http://localhost:9093/-/ready

## Response Formats

### Status Values

**Service Status**:
- `ok`: Service is healthy
- `ready`: Service is ready to handle traffic
- `not_ready`: Service is alive but not ready
- `starting`: Service is initializing
- `degraded`: Service is partially functional
- `unhealthy`: Service is not functional
- `healthy`: Overall health is good

**Check Status**:
- `ok`: Check passed
- `syncing`: Node is syncing (expected during catch-up)
- `degraded`: Check partially passed
- `unhealthy`: Check failed

### Error Messages

When a check fails, the response includes a `message` field with details:

```json
{
  "status": "unhealthy",
  "message": "rpc unreachable: connection refused"
}
```

## Kubernetes Integration

### Example Deployment with Probes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pawd
spec:
  template:
    spec:
      containers:
      - name: pawd
        image: paw-chain/pawd:latest
        ports:
        - containerPort: 26657
          name: rpc
        - containerPort: 36661
          name: health

        # Startup probe - gives 5 minutes for initialization
        startupProbe:
          httpGet:
            path: /health/startup
            port: health
          periodSeconds: 10
          failureThreshold: 30  # 30 * 10s = 5 minutes

        # Liveness probe - restart if fails
        livenessProbe:
          httpGet:
            path: /health
            port: health
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 3  # Restart after 90 seconds of failures

        # Readiness probe - remove from service if fails
        readinessProbe:
          httpGet:
            path: /health/ready
            port: health
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3  # Remove after 30 seconds of failures
```

### Probe Configuration Guidelines

**Liveness Probe**:
- Use `/health` endpoint
- Set `periodSeconds` to 30-60 (don't check too frequently)
- Set `failureThreshold` to 3-5 (allow temporary failures)
- Only restarts on prolonged failures

**Readiness Probe**:
- Use `/health/ready` endpoint
- Set `periodSeconds` to 10-30
- Set `failureThreshold` to 2-3
- Removes from load balancer quickly when unhealthy

**Startup Probe**:
- Use `/health/startup` endpoint
- Set `periodSeconds` to 10
- Set `failureThreshold` high enough for full initialization
- Prevents premature liveness probe failures

## Monitoring and Alerting

### Prometheus Metrics

All health checks export metrics:

```
# Total health check requests
paw_health_check_total{endpoint="health",status="200"} 1000

# Health check duration
paw_health_check_duration_seconds{endpoint="ready",quantile="0.99"} 0.05

# Service health status (1=healthy, 0=unhealthy)
paw_service_healthy{service="rpc"} 1
paw_service_healthy{service="sync"} 1
paw_service_healthy{service="consensus"} 1
```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
- name: paw_health
  rules:
  - alert: PAWServiceUnhealthy
    expr: paw_service_healthy == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "PAW service {{ $labels.service }} is unhealthy"
      description: "Service {{ $labels.service }} has been unhealthy for 1 minute"

  - alert: PAWHealthCheckFailing
    expr: rate(paw_health_check_total{status!="200"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "PAW health checks failing"
      description: "Health check {{ $labels.endpoint }} is failing"

  - alert: PAWHealthCheckSlow
    expr: histogram_quantile(0.99, paw_health_check_duration_seconds) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "PAW health checks slow"
      description: "99th percentile health check duration is over 1 second"
```

### Grafana Dashboard

Create a dashboard panel showing health status:

```json
{
  "title": "Service Health",
  "targets": [
    {
      "expr": "paw_service_healthy",
      "legendFormat": "{{ service }}"
    }
  ],
  "type": "stat",
  "options": {
    "colorMode": "background",
    "graphMode": "none",
    "textMode": "value_and_name"
  },
  "fieldConfig": {
    "defaults": {
      "mappings": [
        {
          "type": "value",
          "options": {
            "0": { "text": "Unhealthy", "color": "red" },
            "1": { "text": "Healthy", "color": "green" }
          }
        }
      ]
    }
  }
}
```

## Troubleshooting

### Health Check Script

Use the aggregator script to check all services:

```bash
./scripts/health-check-all.sh
```

Output example:
```
======================================================================
PAW Health Check Report - 2025-12-14 12:34:56
======================================================================

Service Health Status:
----------------------------------------------------------------------
✓ pawd: healthy (http://localhost:36661/health)
✓ pawd-detailed: healthy (http://localhost:36661/health/detailed)
✓ explorer: healthy (http://localhost:11080/health)
⚠ explorer-ready: not ready (http://localhost:11080/health/ready)
✓ prometheus: healthy (http://localhost:9091/-/healthy)
✓ grafana: healthy (http://localhost:11030/api/health)
✓ alertmanager: healthy (http://localhost:9093/-/healthy)

======================================================================

Overall Status: HEALTHY
```

### Common Issues

#### Service Returns 503 on Readiness Check

**Symptoms**: `/health/ready` returns 503

**Causes**:
1. Database connection failed
2. Node is syncing
3. RPC is unreachable

**Resolution**:
1. Check detailed health: `curl http://localhost:36661/health/detailed | jq`
2. Look at the `checks` section to see which check failed
3. Check logs for the specific component

#### Health Checks Timing Out

**Symptoms**: Health check requests timeout

**Causes**:
1. Service is overloaded
2. Database query is slow
3. Network issues

**Resolution**:
1. Check system metrics (CPU, memory)
2. Check database performance
3. Increase timeout values in probe configuration

#### Liveness Probe Restarting Service Repeatedly

**Symptoms**: Service keeps restarting

**Causes**:
1. Startup time exceeds liveness probe threshold
2. Liveness probe is too aggressive

**Resolution**:
1. Add or tune startup probe
2. Increase liveness probe `periodSeconds` and `failureThreshold`
3. Check if service is actually unhealthy (use detailed endpoint)

### Manual Health Checks

```bash
# Basic liveness
curl http://localhost:36661/health

# Readiness
curl http://localhost:36661/health/ready

# Detailed health with pretty JSON
curl http://localhost:36661/health/detailed | jq

# Check specific service status
curl http://localhost:36661/health/detailed | jq '.checks.rpc'

# Check all modules
curl http://localhost:36661/health/detailed | jq '.modules'

# Check system metrics
curl http://localhost:36661/health/detailed | jq '.system'
```

### Health Check in CI/CD

```bash
# Wait for service to be healthy
timeout 300 bash -c '
  until curl -f http://localhost:36661/health/ready >/dev/null 2>&1; do
    echo "Waiting for service to be ready..."
    sleep 5
  done
'

# Check if service is healthy before deploying
if ! curl -f http://localhost:36661/health/ready >/dev/null 2>&1; then
  echo "Service is not healthy, aborting deployment"
  exit 1
fi
```

## Best Practices

1. **Use appropriate probes for each use case**:
   - Liveness: Simple, fast checks
   - Readiness: Check all dependencies
   - Startup: Allow time for initialization

2. **Don't make health checks expensive**:
   - Cache results when appropriate
   - Use timeouts to prevent hanging
   - Avoid complex calculations

3. **Set reasonable thresholds**:
   - Allow for temporary failures
   - Don't restart too aggressively
   - Remove from load balancer quickly when truly unhealthy

4. **Monitor health check metrics**:
   - Alert on sustained failures
   - Track health check duration
   - Correlate with service performance

5. **Test health checks in development**:
   - Verify they detect actual failures
   - Ensure they return quickly
   - Check they don't cause false positives

## Version History

- **v1.0.0** (2025-12-14): Initial implementation
  - Basic liveness check
  - Readiness check with dependency checks
  - Detailed health check with metrics
  - Startup probe for initialization
  - Prometheus metrics integration
  - Kubernetes-ready configuration
