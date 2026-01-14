# Testing OpenTelemetry Tracing with Jaeger

This guide walks through testing the OpenTelemetry tracing integration with Jaeger.

## Prerequisites

- Docker installed and running
- PAW blockchain compiled (`go build -o pawd ./cmd/pawd`)
- Node initialized (`pawd init test-node --chain-id paw-mvp-1`)

## Step 1: Start Jaeger

```bash
cd /home/hudson/blockchain-projects/paw/monitoring/jaeger
./start-jaeger.sh
```

Expected output:
```
Starting Jaeger distributed tracing for PAW blockchain...
Starting Jaeger container...
Waiting for Jaeger to be ready...
Jaeger is ready!

Jaeger UI:        http://localhost:16686
OTLP HTTP:        http://localhost:4318
...
```

Verify Jaeger is running:
```bash
curl -s http://localhost:16686/ | grep -q "Jaeger UI" && echo "Jaeger UI is accessible"
curl -s http://localhost:14269/ | grep -q "Server available" && echo "Jaeger is healthy"
```

## Step 2: Configure PAW Node

Edit your `~/.paw/config/app.toml` and add:

```toml
[telemetry]
enabled = true
jaeger-endpoint = "http://localhost:4318"
sample-rate = 1.0
environment = "testnet"
prometheus-enabled = true
metrics-port = "26660"
```

## Step 3: Start PAW Node

```bash
pawd start --home ~/.paw
```

Look for these log messages:
```
OpenTelemetry tracing initialized jaeger_endpoint=http://localhost:4318 sample_rate=1.0 environment=testnet chain_id=paw-mvp-1
Telemetry health check passed prometheus_enabled=true
```

If you see errors:
```
Failed to initialize OpenTelemetry error=<error>
```

Check:
1. Jaeger is running: `docker ps | grep paw-jaeger`
2. OTLP endpoint is accessible: `curl http://localhost:4318/v1/traces`
3. No firewall blocking port 4318

## Step 4: Generate Test Transactions

Create a test account:
```bash
pawd keys add test-user --keyring-backend test
```

Fund the account (if using a faucet or genesis account):
```bash
# Example: transfer from genesis account
pawd tx bank send genesis test-user 1000000upaw \
  --from genesis \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

Generate various transactions:

### Bank Transfer
```bash
pawd tx bank send test-user $(pawd keys show test-user -a --keyring-backend test) 1000upaw \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

### DEX Operations
```bash
# Create liquidity pool
pawd tx dex create-pool upaw uatom 1000000 1000000 \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes

# Swap tokens
pawd tx dex swap 1 upaw 100000 uatom 0 \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

### Oracle Operations
```bash
# Submit price feed
pawd tx oracle feed-price btc-usd 45000000000 \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

### Governance
```bash
# Submit proposal
pawd tx gov submit-proposal \
  --title "Test Proposal" \
  --description "Testing tracing" \
  --type Text \
  --deposit 10000000upaw \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

## Step 5: View Traces in Jaeger UI

1. Open http://localhost:16686 in your browser

2. Select service: `paw-blockchain`

3. Click "Find Traces"

You should see traces for:
- Block processing (`block.process`)
- Transaction execution (`transaction.execute`)
- Module operations (`module.dex.swap`, `module.oracle.update`, etc.)

## Step 6: Analyze a Transaction Trace

1. Click on a transaction trace in the results

2. Examine the trace details:
   - **Root Span**: `transaction.execute`
   - **Child Spans**: Individual module executions
   - **Tags**:
     - `block.height`
     - `tx.msg.count`
     - `module.name`
     - `tx.status` (success/failed)

3. Check timing:
   - Total transaction time
   - Time spent in each module
   - Identify bottlenecks (longest spans)

## Step 7: Filter and Search

### By Operation
```
Service: paw-blockchain
Operation: module.dex.swap
```

### By Block Height
Add tag filter:
```
block.height=12345
```

### By Status
```
tx.status=failed
```

### By Module
```
module.name=dex
```

## Step 8: Verify Metrics Integration

If Prometheus is enabled, check metrics endpoint:

```bash
curl http://localhost:26660/metrics | grep -E "^cosmos_"
```

Expected metrics:
```
cosmos_tx_total{tx.type="bank.MsgSend",tx.status="success"} 5
cosmos_tx_processing_time_bucket{tx.type="dex.MsgSwap",...} 150
cosmos_block_height 12345
```

## Step 9: Test Sampling

### High Sample Rate (Development)
```toml
[telemetry]
sample-rate = 1.0  # Trace everything
```

Generate 10 transactions, verify all 10 appear in Jaeger.

### Low Sample Rate (Production)
```toml
[telemetry]
sample-rate = 0.1  # Trace 10%
```

Generate 100 transactions, verify ~10 appear in Jaeger.

## Step 10: Test Error Tracing

Trigger an error (e.g., insufficient balance):

```bash
pawd tx bank send test-user $(pawd keys show test-user -a --keyring-backend test) 999999999999upaw \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

In Jaeger:
1. Find the failed transaction trace
2. Look for:
   - `tx.status=failed` tag
   - Error events in span logs
   - Red markers indicating errors

## Step 11: Test Multi-Span Operations

Complex operations create nested spans:

```bash
# Add liquidity (multiple module interactions)
pawd tx dex add-liquidity 1 1000000upaw 1000000uatom 0 0 \
  --from test-user \
  --chain-id paw-mvp-1 \
  --keyring-backend test \
  --yes
```

In Jaeger, the trace should show:
```
transaction.execute
├── module.dex.add_liquidity
│   ├── module.bank.send (transfer token A)
│   ├── module.bank.send (transfer token B)
│   └── module.bank.mint (mint LP tokens)
```

## Step 12: Continuous Monitoring

Let the node run for 30 minutes and observe:

1. **Trace Volume**:
   ```bash
   # Check Jaeger metrics
   curl -s http://localhost:14269/metrics | grep jaeger_collector_spans_received_total
   ```

2. **Memory Usage**:
   ```bash
   docker stats paw-jaeger --no-stream
   ```

3. **Storage Growth** (if using BadgerDB):
   ```bash
   du -sh ./badger-data
   ```

## Expected Results

### Successful Deployment
- ✅ Jaeger UI accessible at :16686
- ✅ OTLP endpoint responding at :4318
- ✅ PAW logs show "OpenTelemetry tracing initialized"
- ✅ Traces appear in Jaeger UI within seconds
- ✅ All transaction types are traced
- ✅ Span hierarchy is correct
- ✅ Tags contain relevant metadata
- ✅ Failed transactions show error information

### Performance Impact
- 📊 With sample-rate=1.0: <5% overhead
- 📊 With sample-rate=0.1: <1% overhead
- 📊 Jaeger memory usage: 100-500MB (in-memory storage)
- 📊 Trace export latency: <100ms

## Troubleshooting

### No Traces Appearing

**Problem**: Transactions execute but no traces in Jaeger

**Solutions**:
1. Verify telemetry is enabled:
   ```bash
   grep "telemetry" ~/.paw/config/app.toml
   ```

2. Check PAW logs for telemetry initialization:
   ```bash
   pawd start --home ~/.paw 2>&1 | grep -i telemetry
   ```

3. Test OTLP endpoint:
   ```bash
   curl -v http://localhost:4318/v1/traces
   # Should return 405 Method Not Allowed (it only accepts POST)
   ```

4. Check Jaeger logs:
   ```bash
   docker logs paw-jaeger | grep -i error
   ```

### Traces Missing Spans

**Problem**: Transaction traces exist but missing module spans

**Solutions**:
1. Check sampling rate affects all spans equally
2. Verify modules are instrumented (check source code)
3. Look for errors in module execution

### High Memory Usage

**Problem**: Jaeger container using excessive memory

**Solutions**:
1. Reduce `MEMORY_MAX_TRACES` in docker-compose.yml
2. Lower sample rate
3. Switch to BadgerDB storage
4. Enable trace TTL

### Connection Refused

**Problem**: `connection refused` errors in PAW logs

**Solutions**:
1. Verify Jaeger is running:
   ```bash
   docker ps | grep paw-jaeger
   ```

2. Check port mapping:
   ```bash
   docker port paw-jaeger
   ```

3. Test from inside Docker network:
   ```bash
   docker exec paw-jaeger curl http://localhost:4318/v1/traces
   ```

## Cleanup

Stop and remove everything:

```bash
# Stop PAW node
pkill pawd

# Stop Jaeger
cd /home/hudson/blockchain-projects/paw/monitoring/jaeger
docker-compose down

# Remove all data
docker-compose down -v
```

## Advanced Testing

### Load Testing with Traces

Generate high transaction volume:

```bash
for i in {1..1000}; do
  pawd tx bank send test-user $(pawd keys show test-user -a --keyring-backend test) 1upaw \
    --from test-user \
    --chain-id paw-mvp-1 \
    --keyring-backend test \
    --yes &

  if [ $((i % 100)) -eq 0 ]; then
    wait
    echo "Completed $i transactions"
  fi
done
```

Monitor Jaeger:
- Trace ingestion rate
- Query performance
- UI responsiveness

### Multi-Node Tracing

Start multiple nodes with the same Jaeger endpoint:

```bash
# Node 1
pawd start --home ~/.paw1 --p2p.laddr tcp://0.0.0.0:26656

# Node 2
pawd start --home ~/.paw2 --p2p.laddr tcp://0.0.0.0:26756
```

Both nodes send traces to the same Jaeger instance.

Filter by node using custom tags (requires code modification to add node ID tag).

## Success Criteria

- [x] Jaeger starts successfully
- [x] PAW node initializes telemetry without errors
- [x] All transaction types generate traces
- [x] Traces appear in Jaeger UI within 5 seconds
- [x] Span hierarchy is correct (parent-child relationships)
- [x] Tags contain accurate metadata
- [x] Failed transactions show error information
- [x] Sampling works correctly
- [x] Performance overhead is acceptable (<5%)
- [x] Metrics integration works (if enabled)

## Next Steps

After successful testing:

1. Configure for production (lower sample rate, persistent storage)
2. Set up alerts on trace errors
3. Create Grafana dashboards linking metrics to traces
4. Document common trace patterns for debugging
5. Train team on using Jaeger for troubleshooting
