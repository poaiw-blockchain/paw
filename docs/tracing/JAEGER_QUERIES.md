# Jaeger Query Examples

Quick reference for common Jaeger UI queries when analyzing PAW blockchain traces.

## Table of Contents

- [Basic Queries](#basic-queries)
- [Performance Analysis](#performance-analysis)
- [Module-Specific Queries](#module-specific-queries)
- [Error Investigation](#error-investigation)
- [IBC Tracing](#ibc-tracing)
- [Advanced Filtering](#advanced-filtering)

---

## Basic Queries

### View All Recent Traces

**Settings:**
- Service: `paw-blockchain`
- Lookback: `Last 1 Hour`
- Results: `100`

**Use Case:** General overview of system activity

---

### Find Transactions

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Lookback: `Last 1 Hour`

**Use Case:** See all transaction executions

---

### Find Block Processing

**Settings:**
- Service: `paw-blockchain`
- Operation: `block.begin` OR `block.end`
- Lookback: `Last 1 Hour`

**Use Case:** Analyze block processing overhead

---

## Performance Analysis

### Find Slow Transactions (>1s)

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Min Duration: `1s`
- Lookback: `Last 6 Hours`

**Analysis Steps:**
1. Sort results by duration (descending)
2. Click slowest trace
3. Identify widest child span (bottleneck)
4. Check span tags for transaction details

---

### Find Slow Block Processing

**Settings:**
- Service: `paw-blockchain`
- Operation: `block.begin`
- Min Duration: `500ms`
- Lookback: `Last 24 Hours`

**Analysis:**
- Compare with normal block time (~6s)
- Identify modules causing delays

---

### Average Transaction Time

**Query Multiple Traces:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Lookback: `Last 1 Hour`
- Results: `200`

**Calculate Average:**
1. Export results to CSV
2. Average duration column
3. Compare with baseline (e.g., 100ms)

**Alternative:** Use Prometheus metrics
```promql
avg(rate(tx_processing_time_sum[5m]) / rate(tx_processing_time_count[5m]))
```

---

## Module-Specific Queries

### DEX Swap Operations

**Settings:**
- Service: `paw-blockchain`
- Tags: `module.name=dex`
- Operation: `module.dex.swap` (if instrumented)
- Lookback: `Last 1 Hour`

**Common Tags to Filter:**
- `pool.id` - Specific pool
- `trader` - Specific address
- `amount_in` - Trade size

**Analysis:**
- Time in pool lookup
- Price calculation overhead
- Token transfer latency

---

### Oracle Price Submissions

**Settings:**
- Service: `paw-blockchain`
- Tags: `module.name=oracle`
- Operation: `module.oracle.submit`
- Lookback: `Last 1 Hour`

**Tags:**
- `validator` - Submitter address
- `symbol` - Price feed (e.g., BTC/USD)
- `price` - Submitted value

**Metrics:**
- Submission validation time
- Aggregation compute time
- Vote power calculation

---

### Compute Request Processing

**Settings:**
- Service: `paw-blockchain`
- Tags: `module.name=compute`
- Operation: `module.compute.process`
- Lookback: `Last 1 Hour`

**Tags:**
- `request.id` - Request identifier
- `provider` - Provider address
- `verified` - Result verification status

**Bottlenecks:**
- ZK proof verification
- Escrow management
- Result submission

---

## Error Investigation

### Find Failed Transactions

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Tags: `error=true` OR `tx.status=failed`
- Lookback: `Last 6 Hours`

**Steps:**
1. Click failed trace
2. Look for red spans (errors)
3. Check span logs for error messages
4. Examine span tags for context (gas, sender, etc.)

---

### Find Specific Error Type

**Settings:**
- Service: `paw-blockchain`
- Tags: `error.type=InsufficientFunds`
- Lookback: `Last 24 Hours`

**Common Error Types:**
- `InsufficientFunds`
- `InvalidSignature`
- `GasExceeded`
- `PoolNotFound` (DEX)
- `PriceStale` (Oracle)

---

### Find Module Errors

**Settings:**
- Service: `paw-blockchain`
- Tags: `module.name=dex` AND `error=true`
- Lookback: `Last 6 Hours`

**Analysis:**
- Error rate per module
- Most common error types
- Error patterns (time-based, load-based)

---

## IBC Tracing

### Trace IBC Packet Send

**Settings:**
- Service: `paw-blockchain`
- Operation: `ibc.packet.send`
- Lookback: `Last 1 Hour`

**Tags:**
- `ibc.channel` - Channel ID
- `ibc.port` - Port name (e.g., transfer, dex)
- `packet.sequence` - Sequence number

**Trace Flow:**
```
ibc.packet.send
├── channel.verify
├── packet.commit
└── event.emit
```

---

### Trace IBC Packet Receive

**Settings:**
- Service: `paw-blockchain`
- Operation: `ibc.packet.recv`
- Lookback: `Last 1 Hour`

**Tags:**
- `ibc.channel`
- `packet.sequence`
- `proof.height` - Proof block height

**Verify:**
- Proof verification time (should be <100ms)
- Module handler execution
- Acknowledgment write

---

### Find IBC Timeouts

**Settings:**
- Service: `paw-blockchain`
- Tags: `ibc.timeout=true`
- Lookback: `Last 24 Hours`

**Analysis:**
- Timeout patterns
- Channel reliability
- Relayer issues

---

## Advanced Filtering

### Transactions by Specific User

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Tags: `tx.sender=paw1abc123...`
- Lookback: `Custom Range`

**Use Case:** User behavior analysis, debugging user issues

---

### High Gas Transactions

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Tags: `tx.gas_used>1000000`
- Lookback: `Last 6 Hours`

**Analysis:**
- Identify gas-heavy operations
- Optimize or set better gas limits

---

### Transactions at Specific Block

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Tags: `block.height=12345`
- Lookback: `Custom (around block time)`

**Use Case:** Investigate specific block, reproduce issues

---

### Filter by Message Type

**Settings:**
- Service: `paw-blockchain`
- Operation: `transaction.execute`
- Tags: `msg.type=/paw.dex.MsgSwap`
- Lookback: `Last 1 Hour`

**Common Message Types:**
- `/paw.dex.MsgSwap`
- `/paw.dex.MsgAddLiquidity`
- `/paw.oracle.MsgSubmitPrice`
- `/paw.compute.MsgRequestCompute`
- `/cosmos.bank.v1beta1.MsgSend`

---

### Combine Multiple Filters

**Example: Failed swaps on specific pool**

**Settings:**
- Service: `paw-blockchain`
- Tags:
  - `module.name=dex`
  - `pool.id=pool-1`
  - `error=true`
- Lookback: `Last 24 Hours`

**Example: Slow oracle submissions by validator**

**Settings:**
- Service: `paw-blockchain`
- Operation: `module.oracle.submit`
- Tags: `validator=pawvaloper1xyz...`
- Min Duration: `500ms`
- Lookback: `Last 6 Hours`

---

## Exporting Results

### Export Trace to JSON

1. Open trace detail view
2. Click "JSON" button (top right)
3. Copy or download JSON
4. Use for offline analysis or bug reports

**Example JSON Structure:**
```json
{
  "traceID": "a1b2c3d4e5f6...",
  "spans": [
    {
      "spanID": "1234567890",
      "operationName": "transaction.execute",
      "startTime": 1234567890000000,
      "duration": 250000,
      "tags": [
        { "key": "block.height", "value": 1000 },
        { "key": "tx.hash", "value": "ABCD..." }
      ]
    }
  ]
}
```

---

### Compare Two Traces

**Use Case:** Before/after optimization comparison

1. Export both traces to JSON
2. Compare durations of matching spans
3. Calculate % improvement

**Example:**
- Before: `module.dex.swap` = 200ms
- After: `module.dex.swap` = 150ms
- Improvement: 25%

---

## Tips and Best Practices

### Use Specific Operations

Instead of viewing all traces:
- Start broad: All traces
- Narrow down: Specific operation
- Add filters: Tags, duration

### Leverage Tags

Always add meaningful tags when instrumenting code:
```go
span.SetAttributes(
    attribute.String("module.name", "dex"),
    attribute.String("pool.id", poolID),
    attribute.Int64("amount", amount.Int64()),
)
```

### Time Range Selection

- **Real-time debugging**: Last 5 minutes
- **Performance analysis**: Last 1 hour
- **Trend analysis**: Last 24 hours
- **Incident investigation**: Custom (around incident time)

### Result Limits

- Start with 20 results (fast)
- Increase to 100 for broader view
- Use 200+ for statistical analysis
- Export to CSV for large datasets

### Combine with Metrics

**Workflow:**
1. Prometheus alert fires (high tx latency)
2. Note timestamp
3. Query Jaeger for same time range
4. Find slow traces, identify root cause

---

## Summary

Jaeger queries are most effective when:
1. **Specific**: Use tags and operations to narrow results
2. **Targeted**: Focus on time range of interest
3. **Iterative**: Start broad, refine filters
4. **Combined**: Use with logs and metrics for full picture

**Pro Tip:** Save frequently used queries as browser bookmarks (Jaeger preserves query params in URL).

---

## Additional Examples

See `docs/DISTRIBUTED_TRACING_GUIDE.md` for:
- Reading trace diagrams
- Performance optimization workflows
- Troubleshooting guide
- Custom instrumentation examples
