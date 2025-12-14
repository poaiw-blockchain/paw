# Jaeger Distributed Tracing for PAW Blockchain

This directory contains the configuration for deploying Jaeger for distributed tracing of the PAW blockchain application using OpenTelemetry.

## Overview

Jaeger is used to collect, store, and visualize distributed traces from the PAW blockchain. It provides insights into:

- Transaction execution flows
- Module interactions
- Block processing performance
- IBC packet routing
- Performance bottlenecks
- Error propagation

## Architecture

```
PAW Blockchain (OpenTelemetry SDK)
    |
    | OTLP/HTTP
    v
Jaeger Collector (:4318)
    |
    v
Jaeger Storage (In-Memory/BadgerDB)
    |
    v
Jaeger Query Service (:16686)
    |
    v
Jaeger UI (Web Interface)
```

## Quick Start

### 1. Start Jaeger

```bash
cd /home/hudson/blockchain-projects/paw/monitoring/jaeger
docker-compose up -d
```

### 2. Verify Jaeger is Running

```bash
docker ps | grep paw-jaeger
curl http://localhost:16686/
```

### 3. Configure PAW Node

Add to your `app.toml`:

```toml
[telemetry]
# Enable OpenTelemetry tracing
enabled = true

# Jaeger OTLP HTTP endpoint
jaeger-endpoint = "http://localhost:4318"

# Sample rate (0.0 to 1.0)
# 1.0 = trace every transaction (development)
# 0.1 = trace 10% of transactions (production)
sample-rate = 1.0

# Environment label for traces
environment = "testnet"

# Enable Prometheus metrics alongside tracing
prometheus-enabled = true
metrics-port = "26660"
```

### 4. Start PAW Node

```bash
pawd start --home ~/.paw
```

### 5. Access Jaeger UI

Open http://localhost:16686 in your browser.

## Ports

| Port  | Service                    | Protocol |
|-------|----------------------------|----------|
| 16686 | Jaeger UI                  | HTTP     |
| 4318  | OTLP HTTP Receiver         | HTTP     |
| 4317  | OTLP gRPC Receiver         | gRPC     |
| 14268 | Jaeger Thrift Receiver     | HTTP     |
| 14250 | Jaeger gRPC                | gRPC     |
| 9411  | Zipkin Receiver (optional) | HTTP     |
| 14269 | Health Check               | HTTP     |

## Sampling Strategies

The `sampling_strategies.json` file defines sampling rates for different operations:

- **transaction.execute**: 100% (trace all transactions)
- **module.execute**: 50% (trace half of module executions)
- **block.process**: 100% (trace all block processing)
- **ibc.packet**: 80% (trace most IBC packets)
- **default**: 10% (trace 10% of other operations)

Adjust these based on your needs:
- Development: High sampling rates (0.8-1.0)
- Staging: Medium sampling rates (0.3-0.5)
- Production: Low sampling rates (0.05-0.2)

## Storage Options

### In-Memory (Default - Development)

```yaml
environment:
  - SPAN_STORAGE_TYPE=memory
  - MEMORY_MAX_TRACES=10000
```

Pros:
- Fast, no setup required
- Good for development

Cons:
- Data lost on restart
- Limited capacity

### BadgerDB (Production - Local Persistence)

```yaml
environment:
  - SPAN_STORAGE_TYPE=badger
  - BADGER_DIRECTORY_VALUE=/badger/data
  - BADGER_DIRECTORY_KEY=/badger/key
  - BADGER_EPHEMERAL=false
  - BADGER_SPAN_STORE_TTL=168h  # 7 days

volumes:
  - ./badger-data:/badger/data
  - ./badger-key:/badger/key
```

Pros:
- Persistent storage
- No external dependencies
- Good for single-node deployments

Cons:
- Limited scalability
- No distributed query support

### Elasticsearch (Production - Distributed)

```yaml
environment:
  - SPAN_STORAGE_TYPE=elasticsearch
  - ES_SERVER_URLS=http://elasticsearch:9200
  - ES_INDEX_PREFIX=jaeger
```

Pros:
- Scalable
- Production-ready
- Advanced query capabilities

Cons:
- Requires Elasticsearch cluster
- More complex setup

## Using the Jaeger UI

### Finding Traces

1. **Service**: Select `paw-blockchain`
2. **Operation**: Choose specific operations:
   - `transaction.execute` - Transaction traces
   - `block.process` - Block processing traces
   - `module.dex.swap` - DEX swap traces
   - `module.oracle.update` - Oracle update traces
   - `ibc.packet` - IBC packet traces

3. **Lookback**: Choose time range (Last hour, Last 6 hours, etc.)
4. **Tags**: Filter by:
   - `block.height=12345`
   - `tx.status=failed`
   - `module.name=dex`
   - `ibc.channel=channel-0`

### Analyzing a Trace

Each trace shows:
- **Timeline**: Visual representation of span durations
- **Spans**: Individual operations with timing
- **Tags**: Metadata (block height, transaction hash, etc.)
- **Logs**: Events during execution
- **Errors**: Error messages and stack traces

### Performance Analysis

Look for:
- **Long-running spans**: Potential bottlenecks
- **High span counts**: Complex operations
- **Error patterns**: Failing operations
- **Latency distribution**: Performance consistency

## Metrics Integration

Jaeger exposes Prometheus metrics at http://localhost:14269/metrics

Key metrics:
- `jaeger_collector_spans_received_total`
- `jaeger_collector_traces_received_total`
- `jaeger_collector_spans_saved_total`
- `jaeger_query_requests_total`

Add to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'jaeger'
    static_configs:
      - targets: ['localhost:14269']
```

## Troubleshooting

### No Traces Appearing

1. Check Jaeger is running:
   ```bash
   curl http://localhost:14269/
   ```

2. Verify PAW is sending traces:
   ```bash
   # Check PAW logs for "OpenTelemetry tracing initialized"
   journalctl -u pawd -f | grep -i telemetry
   ```

3. Check Jaeger logs:
   ```bash
   docker logs paw-jaeger
   ```

4. Test OTLP endpoint:
   ```bash
   curl -v http://localhost:4318/v1/traces
   ```

### High Memory Usage

- Reduce `MEMORY_MAX_TRACES` in docker-compose.yml
- Lower sample rates in `sampling_strategies.json`
- Switch to BadgerDB or Elasticsearch for storage

### Missing Spans

- Check sample rate configuration
- Verify operation names match sampling strategies
- Enable debug logging in PAW

### Performance Impact

If tracing impacts node performance:
- Reduce sample rate to 0.1 or lower
- Disable module-level tracing
- Use batch exporter settings (already configured)

## Advanced Configuration

### Custom Sampling

Create operation-specific sampling rules in `sampling_strategies.json`:

```json
{
  "operation": "module.dex.add_liquidity",
  "type": "probabilistic",
  "param": 0.3
}
```

### Trace Context Propagation

For multi-node setups, ensure trace context is propagated:
- PAW automatically propagates context in IBC packets
- Use distributed trace IDs for correlation

### Integration with Grafana

1. Add Jaeger as a data source in Grafana
2. Create dashboards combining metrics and traces
3. Use trace IDs to link metrics to specific traces

## Production Deployment

For production deployments:

1. Use Elasticsearch storage backend
2. Set low sample rates (0.05-0.2)
3. Configure retention policies (7-30 days)
4. Enable TLS for OTLP endpoints
5. Set up authentication
6. Configure alerts on trace errors
7. Monitor Jaeger resource usage

Example production docker-compose:

```yaml
services:
  jaeger:
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch:9200
      - COLLECTOR_OTLP_ENABLED=true
      - COLLECTOR_OTLP_HTTP_TLS_ENABLED=true
      - COLLECTOR_OTLP_HTTP_TLS_CERT=/certs/cert.pem
      - COLLECTOR_OTLP_HTTP_TLS_KEY=/certs/key.pem
```

## Health Checks

Jaeger provides health check endpoints:

```bash
# Overall health
curl http://localhost:14269/

# Specific components
curl http://localhost:14269/health

# Readiness (for Kubernetes)
curl http://localhost:14269/ready
```

## Cleanup

### Stop Jaeger

```bash
docker-compose down
```

### Remove All Data

```bash
docker-compose down -v
```

## References

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [OTLP Specification](https://opentelemetry.io/docs/reference/specification/protocol/otlp/)
- [Sampling Strategies](https://www.jaegertracing.io/docs/latest/sampling/)

## Support

For issues with Jaeger tracing:
1. Check this README
2. Review Jaeger logs: `docker logs paw-jaeger`
3. Verify PAW configuration in `app.toml`
4. Check OpenTelemetry SDK logs in PAW output
