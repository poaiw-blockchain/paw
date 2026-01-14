# Distributed Tracing Guide

## Table of Contents

- [Overview](#overview)
- [What is Distributed Tracing?](#what-is-distributed-tracing)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Jaeger UI Navigation](#jaeger-ui-navigation)
- [Reading Trace Diagrams](#reading-trace-diagrams)
- [Common Queries and Use Cases](#common-queries-and-use-cases)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)

---

## Overview

The PAW blockchain uses **OpenTelemetry** for distributed tracing with **Jaeger** as the tracing backend. This enables deep visibility into transaction execution, block processing, module operations, and IBC packet flows.

**Key Benefits:**
- **Performance Analysis**: Identify slow operations and bottlenecks
- **Debugging**: Trace transaction flow across modules
- **System Understanding**: Visualize how components interact
- **Production Monitoring**: Track performance trends over time

---

## What is Distributed Tracing?

Distributed tracing tracks the execution path of requests through a system. Each operation creates a **span** (a timed unit of work), and related spans form a **trace**.

### Key Concepts

**Trace**: Complete journey of a request from start to finish
- Example: A DEX swap transaction from submission to execution

**Span**: Single unit of work within a trace
- Example: Pool lookup, price calculation, token transfer

**Parent-Child Relationships**: Spans can nest to show call hierarchies
```
Transaction Execution (parent)
├── Module Router (child)
│   ├── DEX Handler (grandchild)
│   │   ├── Pool Lookup
│   │   ├── Price Calculation
│   │   └── Token Transfer
```

**Attributes**: Metadata attached to spans
- Transaction hash, block height, module name, gas used, etc.

**Events**: Time-stamped annotations within a span
- State changes, errors, important checkpoints

---

## Architecture

### Components

```
┌─────────────┐          ┌──────────────┐          ┌─────────────┐
│             │  OTLP    │              │          │             │
│  PAW Node   ├─────────>│   Jaeger     ├─────────>│  Jaeger UI  │
│             │  HTTP    │  Collector   │  Query   │  (Browser)  │
│             │  :4318   │              │          │   :16686    │
└─────────────┘          └──────────────┘          └─────────────┘
                               │
                               v
                         ┌──────────┐
                         │  Storage │
                         │ (Memory/ │
                         │  Badger) │
                         └──────────┘
```

**PAW Node**: Generates trace spans during operation
**OpenTelemetry SDK**: Batches and exports traces
**Jaeger Collector**: Receives traces via OTLP HTTP (port 4318)
**Jaeger Storage**: Stores traces (in-memory or persistent)
**Jaeger UI**: Web interface for querying and visualizing traces

### Instrumented Operations

The PAW blockchain automatically traces:

1. **Block Processing**
   - BeginBlock: `block.begin`
   - EndBlock: `block.end`

2. **Transaction Execution**
   - Transaction routing: `transaction.execute`
   - Message count and block height attributes

3. **Module Operations** (when implemented)
   - DEX: Swaps, liquidity operations, pool management
   - Oracle: Price submissions, aggregation, voting
   - Compute: Request processing, verification, disputes
   - IBC: Packet send/receive, channel operations

---

## Quick Start

### 1. Deploy Jaeger

```bash
cd /home/hudson/blockchain-projects/paw
docker compose -f compose/docker-compose.tracing.yml up -d
```

**Verify Jaeger is running:**
```bash
docker ps | grep jaeger
# Should show: paw-jaeger ... Up ... (healthy)

curl -s http://localhost:16686/api/services
# Should return JSON (empty initially)
```

### 2. Enable Tracing in PAW

Create or update `~/.paw/config/app.toml`:

```toml
[telemetry]
enabled = true
jaeger-endpoint = "http://localhost:4318"
prometheus-enabled = true
sample-rate = 0.1  # Sample 10% of traces
```

**Configuration Options:**
- `enabled`: Set to `true` to enable tracing
- `jaeger-endpoint`: OTLP HTTP endpoint (default: `http://localhost:4318`)
- `sample-rate`: Fraction of traces to sample (0.0 to 1.0)
  - `1.0` = 100% (development/testing only)
  - `0.1` = 10% (recommended for testnet)
  - `0.01` = 1% (recommended for production)

### 3. Start PAW Node

```bash
cd /home/hudson/blockchain-projects/paw
pawd start
```

Look for the log message:
```
Telemetry initialized jaeger_endpoint=http://localhost:4318 sample_rate=0.1
```

### 4. Access Jaeger UI

Open browser: **http://localhost:16686**

Once transactions are processed, you'll see `paw-blockchain` in the service dropdown.

---

## Configuration

### Sampling Strategies

Sampling controls what percentage of traces are collected. This is critical for production to avoid overwhelming the system.

**Default Strategy** (defined in `compose/docker/tracing/sampling_strategies.json`):

```json
{
  "service_strategies": [
    {
      "service": "paw-blockchain",
      "type": "probabilistic",
      "param": 0.1,
      "operation_strategies": [
        {
          "operation": "transaction.execute",
          "type": "probabilistic",
          "param": 0.5
        },
        {
          "operation": "block.begin",
          "type": "probabilistic",
          "param": 1.0
        }
      ]
    }
  ]
}
```

**Strategy Breakdown:**
- **Default**: 10% of all traces
- **Transactions**: 50% sampling (higher for detailed analysis)
- **Blocks**: 100% sampling (always trace block processing)

**Adjust Sampling:**
1. Edit `compose/docker/tracing/sampling_strategies.json`
2. Restart Jaeger: `docker compose -f compose/docker-compose.tracing.yml restart`

### Environment Variables

Override ports via environment variables:

```bash
export JAEGER_UI_PORT=16686
export JAEGER_OTLP_HTTP_PORT=4318
export JAEGER_OTLP_GRPC_PORT=11317
docker compose -f compose/docker-compose.tracing.yml up -d
```

### Storage Backend

**Memory Storage** (default for development):
- Fast, no disk I/O
- Limited to 10,000 traces
- Data lost on restart

**Badger Storage** (persistent):
Edit `compose/docker-compose.tracing.yml`:

```yaml
environment:
  - SPAN_STORAGE_TYPE=badger
  - BADGER_DIRECTORY_VALUE=/badger/data
  - BADGER_DIRECTORY_KEY=/badger/key
```

**Elasticsearch/Cassandra** (production):
See [Jaeger Documentation](https://www.jaegertracing.io/docs/latest/deployment/) for production storage backends.

---

## Jaeger UI Navigation

### Home Screen

**Service Dropdown**: Select `paw-blockchain`

**Operation Dropdown**: Select specific operation to filter
- `block.begin`
- `block.end`
- `transaction.execute`
- `module.dex.swap`
- `module.oracle.submit`

**Tags**: Add filters like:
- `block.height: 1000`
- `tx.status: failed`
- `module.name: dex`

**Lookback**: Time range (Last Hour, Last 6 Hours, Custom)

**Limit Results**: Number of traces to fetch (default 20)

### Search Results

**Trace List**: Shows all matching traces
- Service name, operation, duration, span count
- Click to view detailed trace

**Timeline**: Visual representation of when traces occurred

### Trace Detail View

**Trace Timeline**: Horizontal gantt chart
- Each row is a span
- Indentation shows parent-child relationships
- Width shows duration

**Span Details**: Click any span to see:
- Tags (attributes)
- Logs (events)
- Process info
- References (parent/child links)

---

## Reading Trace Diagrams

### Example: DEX Swap Transaction

```
transaction.execute                    [████████████████████████] 250ms
├── module.dex.swap                    [██████████████████] 200ms
│   ├── pool.lookup                    [██] 10ms
│   ├── price.calculate                [████] 25ms
│   ├── slippage.check                 [██] 8ms
│   ├── token.transfer                 [████████] 50ms
│   └── event.emit                     [█] 5ms
└── state.commit                       [████] 40ms
```

**Reading the Diagram:**
1. **Total Duration**: 250ms (full bar)
2. **Module Swap**: 200ms out of 250ms (80% of time)
3. **Bottleneck**: `token.transfer` took 50ms (25% of module time)
4. **Sequential vs Parallel**: Vertical alignment shows sequential execution

### Interpreting Colors

- **Blue**: Normal span
- **Red**: Span with errors or warnings
- **Yellow**: Long-running spans (outliers)

### Critical Path Analysis

The **critical path** is the longest sequence of dependent spans.

In the example above:
```
transaction.execute → module.dex.swap → token.transfer
```

Optimizing the critical path has the most impact on performance.

---

## Common Queries and Use Cases

### 1. Find Slow Transactions

**Goal**: Identify transactions taking longer than 1 second

**Query:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Min Duration: `1s`

**Analysis**: Look for spans contributing most to duration (widest bars)

### 2. Trace DEX Swap Operations

**Goal**: Understand DEX swap execution flow

**Query:**
- Service: `paw-blockchain`
- Tags: `module.name=dex`
- Operation: `module.dex.swap` (if instrumented)

**Insights:**
- Time spent in pool lookup vs execution
- Token transfer overhead
- Event emission costs

### 3. Trace Oracle Price Submissions

**Goal**: Analyze oracle price aggregation latency

**Query:**
- Service: `paw-blockchain`
- Tags: `module.name=oracle`
- Operation: `module.oracle.submit`

**Metrics to Check:**
- Validation time
- Aggregation compute time
- Vote power calculation

### 4. Trace IBC Packet Workflows

**Goal**: Follow IBC packet from send to acknowledgment

**Query:**
- Service: `paw-blockchain`
- Tags: `ibc.channel=channel-0`
- Operation: `ibc.packet.send` or `ibc.packet.recv`

**Trace Flow:**
```
ibc.packet.send
├── channel.verify
├── packet.commit
└── event.emit

[Later, on counterparty chain]

ibc.packet.recv
├── proof.verify
├── module.handle
└── ack.write
```

### 5. Block Processing Performance

**Goal**: Measure BeginBlock and EndBlock overhead

**Query:**
- Service: `paw-blockchain`
- Operation: `block.begin` or `block.end`
- Lookback: Last 1 hour

**Compare Across Blocks**:
- Average duration
- Outliers (slow blocks)
- Module-specific overhead

### 6. Error Analysis

**Goal**: Find failed transactions and understand why

**Query:**
- Service: `paw-blockchain`
- Tags: `error=true` or `tx.status=failed`

**Examine Span Logs**:
- Error messages
- Stack traces
- Retry attempts

---

## Performance Optimization

### 1. Identify Bottlenecks

**Workflow:**
1. Run representative load (e.g., 100 swaps)
2. Query Jaeger for `module.dex.swap`
3. Sort by duration (longest first)
4. Identify the widest child span (bottleneck)

**Common Bottlenecks:**
- Database queries (IAVL tree lookups)
- Cryptographic operations (signature verification)
- IBC proof verification
- Event emission

### 2. Optimize Critical Path

**Before Optimization:**
```
swap [200ms]
├── pool_lookup [50ms]     <-- Bottleneck
├── calculate [30ms]
└── transfer [100ms]
```

**Optimization:** Cache pool state in memory

**After Optimization:**
```
swap [150ms]
├── pool_lookup [5ms]      <-- Cached
├── calculate [30ms]
└── transfer [100ms]
```

**Result:** 25% faster swaps

### 3. Reduce Span Count

Too many spans add overhead. Keep spans meaningful:

**Good:**
```
transaction.execute
└── module.dex.swap
    ├── validate_input
    ├── execute_swap
    └── emit_events
```

**Bad (too granular):**
```
transaction.execute
└── module.dex.swap
    ├── check_param_1
    ├── check_param_2
    ├── check_param_3
    ├── load_pool
    ├── validate_pool
    ├── calculate_price_step1
    ├── calculate_price_step2
    ...
```

### 4. Use Sampling Effectively

**Development:** `sample-rate = 1.0` (100%)
**Testnet:** `sample-rate = 0.1` (10%)
**Production:** `sample-rate = 0.01` (1%)

**Operation-Specific Sampling:**
- Always sample: Errors, slow transactions (>1s)
- Rarely sample: Fast, high-frequency operations

---

## Troubleshooting

### Jaeger Container Not Starting

**Check logs:**
```bash
docker logs paw-jaeger
```

**Common Issues:**
1. **Port conflict**: Change `JAEGER_UI_PORT` in docker-compose
2. **Permission denied**: Check sampling_strategies.json permissions
3. **Network conflict**: Adjust subnet in docker-compose.tracing.yml

### No Traces in Jaeger UI

**Checklist:**
1. Jaeger running: `docker ps | grep jaeger`
2. Node config enabled: `telemetry.enabled = true` in app.toml
3. Node restarted after config change
4. Traffic generated (transactions sent)
5. Sample rate not too low (try `1.0` for testing)

**Verify OTLP endpoint:**
```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[]}'
```

Should return `200 OK`.

**Check node logs:**
```bash
grep -i telemetry ~/.paw/logs/pawd.log
```

Should show: `Telemetry initialized`

### Spans Missing Attributes

**Cause:** Attributes not set in code

**Fix:** Add attributes when creating spans:
```go
span.SetAttributes(
    attribute.String("module.name", "dex"),
    attribute.Int64("block.height", ctx.BlockHeight()),
)
```

### High Overhead from Tracing

**Symptoms:** Node performance degraded, high CPU

**Solutions:**
1. Lower sample rate: `sample-rate = 0.01`
2. Reduce span granularity (fewer child spans)
3. Disable tracing: `telemetry.enabled = false`

**Measure Overhead:**
```bash
# Baseline (tracing disabled)
time pawd tx bank send ... --chain-id paw-mvp-1

# With tracing (sample-rate = 1.0)
time pawd tx bank send ... --chain-id paw-mvp-1

# Compare durations
```

### Jaeger UI Slow

**Causes:**
1. Too many traces stored (memory backend)
2. Large time range queried
3. Network latency

**Solutions:**
1. Restart Jaeger to clear memory: `docker compose -f compose/docker-compose.tracing.yml restart`
2. Use smaller time ranges (last 1 hour vs last 7 days)
3. Add more specific filters (tags)
4. Upgrade to persistent storage (Badger/Cassandra)

---

## Advanced Topics

### Custom Instrumentation

Add tracing to custom modules:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (k Keeper) ProcessSwap(ctx sdk.Context, msg *types.MsgSwap) error {
    // Create tracer
    tracer := otel.Tracer("paw-blockchain")

    // Start span
    spanCtx, span := tracer.Start(ctx.Context(), "module.dex.swap")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("pool.id", msg.PoolId),
        attribute.String("trader", msg.Trader),
        attribute.Int64("amount_in", msg.AmountIn.Int64()),
    )

    // Wrap SDK context
    ctx = ctx.WithContext(spanCtx)

    // Business logic
    result, err := k.executeSwap(ctx, msg)
    if err != nil {
        span.RecordError(err)
        return err
    }

    span.SetAttributes(
        attribute.Int64("amount_out", result.AmountOut.Int64()),
    )

    return nil
}
```

### Trace Context Propagation

For IBC or cross-chain calls, propagate trace context:

```go
// Sending chain
spanCtx, span := tracer.Start(ctx.Context(), "ibc.packet.send")
defer span.End()

// Extract trace context
carrier := propagation.MapCarrier{}
otel.GetTextMapPropagator().Inject(spanCtx, carrier)

// Include in IBC packet data
packetData := types.PacketData{
    TraceContext: carrier,
    // ... other fields
}

// Receiving chain
carrier := propagation.MapCarrier(packetData.TraceContext)
spanCtx := otel.GetTextMapPropagator().Extract(ctx.Context(), carrier)

// Continue trace
_, span := tracer.Start(spanCtx, "ibc.packet.recv")
defer span.End()
```

### Integration with Prometheus

Correlate metrics with traces:

**Prometheus Query:**
```promql
rate(tx_processing_time_sum[5m]) / rate(tx_processing_time_count[5m])
```

**Jaeger Query:**
- Find slow transactions in same time window
- Compare metric spike with trace details

### Export to Grafana Tempo

For long-term storage and Grafana integration:

1. Deploy Grafana Tempo
2. Update `app.toml`:
```toml
jaeger-endpoint = "http://tempo:4318"
```

3. Add Tempo as data source in Grafana
4. Link traces from Grafana dashboards

### Continuous Profiling

Combine tracing with profiling:

1. **Identify slow span** in Jaeger
2. **Enable profiling** during same operation:
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
   ```
3. **Correlate**: Match slow span duration with CPU hotspots

---

## Additional Resources

- **OpenTelemetry Docs**: https://opentelemetry.io/docs/
- **Jaeger Docs**: https://www.jaegertracing.io/docs/
- **Cosmos SDK Telemetry**: https://docs.cosmos.network/main/core/telemetry
- **PAW Monitoring**: See `docs/MONITORING_GUIDE.md`

---

## Summary

Distributed tracing with Jaeger provides unprecedented visibility into PAW blockchain operations. By following this guide, you can:

- **Deploy Jaeger** in minutes
- **Trace transactions** end-to-end
- **Identify bottlenecks** and optimize performance
- **Debug issues** with detailed execution context
- **Monitor production** systems effectively

Start with 10% sampling on testnet, then adjust based on your needs. Always profile before and after optimizations to measure impact.

**Happy Tracing!**
