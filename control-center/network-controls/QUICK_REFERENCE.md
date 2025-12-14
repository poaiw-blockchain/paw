# Network Controls Quick Reference

## Circuit Breaker Status Values

- `closed` - Normal operation (default)
- `open` - Operations blocked
- `half-open` - Testing recovery

## Common Operations

### Pause DEX
```bash
curl -X POST http://localhost:11050/api/v1/controls/dex/pause \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","reason":"maintenance","auto_resume_mins":60}'
```

### Resume DEX
```bash
curl -X POST http://localhost:11050/api/v1/controls/dex/resume \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","reason":"complete"}'
```

### Check Status
```bash
curl http://localhost:11050/api/v1/controls/status | jq
```

### Emergency Halt All
```bash
curl -X POST http://localhost:11050/api/v1/controls/emergency/halt \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","reason":"incident","modules":[],"signature":"0x..."}'
```

### Override Oracle Price
```bash
curl -X POST http://localhost:11050/api/v1/controls/oracle/override-price \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","pair":"BTC/USD","price":"50000000000","duration":3600,"reason":"emergency"}'
```

### Cancel Compute Job
```bash
curl -X POST http://localhost:11050/api/v1/controls/compute/job/req-123/cancel \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","job_id":"req-123","reason":"malicious"}'
```

### Pause Pool
```bash
curl -X POST http://localhost:11050/api/v1/controls/dex/pool/1/pause \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","reason":"anomaly"}'
```

### Pause Provider
```bash
curl -X POST http://localhost:11050/api/v1/controls/compute/provider/pawaddr123/pause \
  -H "Content-Type: application/json" \
  -d '{"actor":"admin","reason":"investigation"}'
```

## Prometheus Metrics

```
circuit_breaker_status{module,submodule}
circuit_breaker_transitions_total{module,submodule,from,to}
circuit_breaker_auto_resumes_total
```

## SDK Integration Checks

### In Message Handler
```go
if err := k.CheckCircuitBreaker(ctx); err != nil {
    return nil, err
}
```

### Check Pool-Specific
```go
if err := k.CheckPoolCircuitBreaker(ctx, poolID); err != nil {
    return nil, err
}
```

### Check Provider-Specific
```go
if err := k.CheckProviderCircuitBreaker(ctx, providerAddr); err != nil {
    return nil, err
}
```

## Health Check

```bash
curl http://localhost:11050/api/v1/controls/health
```

## Port Allocation

- PAW Control Center: 11050
- Aura Control Center: 10050
- XAI Control Center: 12050
