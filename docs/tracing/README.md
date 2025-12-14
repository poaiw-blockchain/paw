# Distributed Tracing - Quick Reference

## TL;DR

**Deploy Jaeger:**
```bash
docker compose -f compose/docker-compose.tracing.yml up -d
```

**Enable in PAW:**
```toml
# ~/.paw/config/app.toml
[telemetry]
enabled = true
jaeger-endpoint = "http://localhost:4318"
sample-rate = 0.1
```

**Access UI:**
- **Jaeger UI**: http://localhost:16686
- **OTLP HTTP**: http://localhost:4318
- **OTLP gRPC**: http://localhost:11317

**View Traces:**
1. Open http://localhost:16686
2. Select service: `paw-blockchain`
3. Click "Find Traces"

---

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Trace** | Complete journey of a request (e.g., a transaction) |
| **Span** | Single unit of work within a trace (e.g., pool lookup) |
| **Tag** | Metadata on a span (e.g., `module.name=dex`) |
| **Sampling** | Percentage of traces to collect (e.g., 10%) |

---

## Common Operations

### Find Slow Transactions
```
Service: paw-blockchain
Operation: transaction.execute
Min Duration: 1s
```

### Trace DEX Swaps
```
Service: paw-blockchain
Tags: module.name=dex
```

### Trace Block Processing
```
Service: paw-blockchain
Operation: block.begin
Lookback: Last 1 Hour
```

### Find Errors
```
Service: paw-blockchain
Tags: error=true
```

---

## Port Reference

| Port | Service | Description |
|------|---------|-------------|
| 16686 | Jaeger UI | Web interface |
| 4318 | OTLP HTTP | OpenTelemetry endpoint (used by PAW) |
| 11317 | OTLP gRPC | OpenTelemetry gRPC (mapped from 4317) |
| 14268 | Jaeger Collector | Legacy Jaeger endpoint |

---

## Configuration Files

| File | Purpose |
|------|---------|
| `compose/docker-compose.tracing.yml` | Jaeger deployment |
| `compose/docker/tracing/sampling_strategies.json` | Sampling configuration |
| `config/app.toml.example` | App configuration example |
| `~/.paw/config/app.toml` | Actual app configuration |

---

## Sampling Rates

| Environment | Sample Rate | Description |
|-------------|-------------|-------------|
| **Development** | 1.0 (100%) | Trace everything |
| **Testnet** | 0.1 (10%) | Recommended starting point |
| **Production** | 0.01 (1%) | Minimize overhead |

---

## Instrumented Operations

| Operation | Description |
|-----------|-------------|
| `block.begin` | BeginBlock processing |
| `block.end` | EndBlock processing |
| `transaction.execute` | Transaction routing and execution |
| `module.execute` | Module-specific operations |

---

## Troubleshooting

**No traces appearing?**
1. Check Jaeger is running: `docker ps | grep jaeger`
2. Verify config: `grep telemetry ~/.paw/config/app.toml`
3. Restart node after config change
4. Increase sample rate to 1.0 for testing

**Jaeger container restarting?**
```bash
docker logs paw-jaeger
```

Common fixes:
- Port conflicts: Change port in docker-compose.tracing.yml
- Permission errors: `chmod 644 compose/docker/tracing/sampling_strategies.json`

---

## Full Documentation

- **Comprehensive Guide**: [DISTRIBUTED_TRACING_GUIDE.md](../DISTRIBUTED_TRACING_GUIDE.md)
- **Query Examples**: [JAEGER_QUERIES.md](JAEGER_QUERIES.md)
- **OpenTelemetry Docs**: https://opentelemetry.io/docs/
- **Jaeger Docs**: https://www.jaegertracing.io/docs/

---

## Quick Commands

**Start Jaeger:**
```bash
docker compose -f compose/docker-compose.tracing.yml up -d
```

**Stop Jaeger:**
```bash
docker compose -f compose/docker-compose.tracing.yml down
```

**View Logs:**
```bash
docker logs -f paw-jaeger
```

**Check Health:**
```bash
curl http://localhost:16686/api/services
```

**Test OTLP Endpoint:**
```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[]}'
```

---

## Next Steps

1. ✅ Deploy Jaeger
2. ✅ Enable tracing in app.toml
3. ✅ Start PAW node
4. ✅ Generate some transactions
5. ✅ Open Jaeger UI and explore traces
6. 📖 Read full guide for advanced usage

**Happy Tracing!**
