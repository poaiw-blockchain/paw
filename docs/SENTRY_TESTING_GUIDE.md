# PAW Sentry Architecture - Testing Guide

Complete guide for testing the sentry node architecture using automated test suites.

## Overview

The PAW sentry architecture includes comprehensive automated testing to validate:
- **Basic functionality** - Connectivity, peer relationships, consensus
- **Load distribution** - RPC performance, latency, load balancing
- **Network resilience** - Chaos scenarios, failure recovery, fault tolerance

## Test Suites

### 1. Basic Sentry Scenarios (`test-sentry-scenarios.sh`)

**What it tests:**
- Basic RPC connectivity through sentries
- Sentry peer connections (5 peers: 4 validators + 1 sentry)
- Validator consensus continuity
- Load distribution across sentries
- Validator isolation (validators accessible through sentries)
- PEX (Peer Exchange) functionality
- Sentry failure and recovery

**Run it:**
```bash
./scripts/devnet/test-sentry-scenarios.sh
```

**Expected results:**
- All 20 tests pass
- Sentries have 5 peer connections each
- Network continues producing blocks
- Sentries recover quickly after restart

**Duration:** ~2 minutes

---

### 2. Load Distribution Testing (`test-load-distribution.sh`)

**What it tests:**
- RPC latency comparison (validator vs sentries)
- Load balancing across sentries
- Data consistency across endpoints
- Concurrent request handling
- Optional: Sustained load testing

**Run it:**
```bash
# Basic load tests
./scripts/devnet/test-load-distribution.sh

# With sustained load test (30s duration)
./scripts/devnet/test-load-distribution.sh --sustained
```

**Expected results:**
- Sentry latency overhead <50ms (typically <5ms)
- 100% load balancing success rate
- Consistent blockchain state across all endpoints
- Similar performance under concurrent load

**Duration:** ~30 seconds (basic), ~2 minutes (with --sustained)

---

### 3. Network Chaos Testing (`test-network-chaos.sh`)

**What it tests:**
- Network partitions (split-brain scenarios)
- Sequential sentry failures
- Single validator failure
- Cascading failures (progressive node failures)
- Rapid restart cycles

**Run it:**
```bash
./scripts/devnet/test-network-chaos.sh
```

**⚠️ Warning:** This test temporarily disrupts network connectivity. The network automatically recovers after each test.

**Expected results:**
- Consensus continues with 3/4 validators
- Validators produce blocks even with both sentries down
- Network recovers automatically from all failure scenarios
- Sentries catch up quickly after restarts

**Duration:** ~4 minutes

**Note:** All network disruptions are automatically cleaned up, even if you interrupt the test (Ctrl+C).

---

### 4. Complete Test Suite (`test-sentry-all.sh`)

**What it does:**
Runs all three test suites in sequence with comprehensive reporting.

**Run it:**
```bash
# Run all tests (including chaos)
./scripts/devnet/test-sentry-all.sh

# Skip chaos tests
./scripts/devnet/test-sentry-all.sh --skip-chaos
```

**Expected results:**
- Network pre-check passes
- All test suites pass
- Comprehensive report generated in `/tmp/`

**Duration:** ~7 minutes (full), ~3 minutes (without chaos)

**Output:**
- Console output with color-coded results
- Detailed report saved to `/tmp/paw-sentry-test-report-TIMESTAMP.txt`

---

## Quick Start

### Prerequisites

1. **Network must be running:**
   ```bash
   docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
   ```

2. **Wait for network to stabilize:**
   ```bash
   sleep 60
   ```

3. **Verify basic status:**
   ```bash
   curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
   curl -s http://localhost:30658/status | jq '.result.sync_info.latest_block_height'
   ```

### Running Tests

**Option 1: Run all tests** (recommended for full validation)
```bash
./scripts/devnet/test-sentry-all.sh
```

**Option 2: Run individual test suites**
```bash
# Basic scenarios
./scripts/devnet/test-sentry-scenarios.sh

# Load testing
./scripts/devnet/test-load-distribution.sh

# Chaos testing
./scripts/devnet/test-network-chaos.sh
```

---

## Test Results Interpretation

### Success Indicators

✅ **Basic Scenarios:**
- All tests pass: 20/20
- Sentries have 5 peers each
- Catching_up: false for all nodes
- Blocks continuously produced

✅ **Load Distribution:**
- Latency overhead <50ms (typically 0-5ms)
- 100% load balancing success rate
- Consistent blockchain state
- No failed requests

✅ **Chaos Testing:**
- Network recovers from all failure scenarios
- Consensus continues with ≥3/4 validators
- Sentries reconnect and sync after restart
- No permanent network damage

### Common Issues

❌ **"Network not running"**
```bash
# Start network
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
```

❌ **"Sentries have wrong peer count"**
```bash
# Wait for peers to connect
sleep 30

# If still wrong, restart sentries
docker restart paw-sentry1 paw-sentry2
sleep 30
```

❌ **"Blockchain not producing blocks"**
```bash
# Check logs
docker logs paw-node1 2>&1 | tail -50

# Restart network
docker compose -f compose/docker-compose.4nodes-with-sentries.yml restart
```

❌ **"High latency overhead"**
- This is unusual - typically <5ms overhead
- Check system load: `top`, `htop`
- Check Docker resource limits
- May indicate network congestion

---

## Performance Benchmarks

Based on local testnet results (6-core CPU, 16GB RAM, SSD):

### Latency Metrics
```
Direct Validator:  9-11ms  (baseline)
Sentry1:          9-11ms  (0-2ms overhead)
Sentry2:          9-11ms  (0-2ms overhead)
```

### Concurrent Load (10 requests)
```
                  Avg     Min     Max
Direct Validator  10ms    9ms     11ms
Sentry1           10ms    9ms     11ms
Sentry2           10ms    10ms    11ms
```

### Sustained Load (30s, 5 req/s)
```
Total requests:    150
Successful:        150 (100%)
Failed:            0
Average latency:   10-12ms
```

### Recovery Times
```
Sentry restart:           6s
Validator catchup:        15s
Network partition repair: 15-20s
```

---

## Chaos Testing Scenarios Explained

### 1. Network Partition (Split Brain)
**Scenario:** Disconnects 2/4 validators from network

**Expected behavior:**
- With 2/4 validators: Consensus may halt (not BFT majority)
- With 3/4 validators: Consensus continues
- After reconnection: Network stabilizes within 15s

**Real-world equivalent:** Data center network split

---

### 2. Sequential Sentry Failures
**Scenario:** Stops sentry1, then sentry2

**Expected behavior:**
- After sentry1 failure: Network still accessible via sentry2
- After both sentries down: Validators still produce blocks
- After restart: Both sentries catch up within 20s

**Real-world equivalent:** DDoS attack on public-facing sentries

---

### 3. Single Validator Failure
**Scenario:** Stops 1 of 4 validators

**Expected behavior:**
- Consensus continues (3/4 validators sufficient for BFT)
- Sentries lose 1 peer connection (4 peers remaining)
- After restart: Validator catches up within 15s

**Real-world equivalent:** Validator node crash or maintenance

---

### 4. Cascading Failures
**Scenario:** Progressive failures (sentry1 → node4 → sentry2)

**Expected behavior:**
- Network adapts to each failure
- Consensus continues with 3 validators
- After recovery: Full network restored

**Real-world equivalent:** Multiple simultaneous infrastructure failures

---

### 5. Rapid Restarts
**Scenario:** 3 cycles of stop/start both sentries

**Expected behavior:**
- Sentries recover after each cycle
- Peer connections re-established
- No degradation over multiple cycles

**Real-world equivalent:** Container orchestration updates (Kubernetes rolling updates)

---

## CI/CD Integration

### Running in CI Pipeline

```yaml
# Example GitHub Actions workflow
- name: Test Sentry Architecture
  run: |
    docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
    sleep 60
    ./scripts/devnet/test-sentry-all.sh
  timeout-minutes: 10
```

### Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed
- Check `/tmp/paw-sentry-test-report-*.txt` for details

---

## Troubleshooting

### Tests Fail Immediately

**Cause:** Network not ready

**Solution:**
```bash
# Ensure network is running and stable
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60  # Wait for consensus to stabilize

# Verify before testing
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

### Chaos Tests Don't Clean Up

**Cause:** Test interrupted before cleanup

**Solution:**
```bash
# Manual cleanup
docker start paw-node1 paw-node2 paw-node3 paw-node4 paw-sentry1 paw-sentry2
for container in paw-node1 paw-node2 paw-node3 paw-node4 paw-sentry1 paw-sentry2; do
  docker network connect pawnet "$container" 2>/dev/null || true
done

# Wait for network to stabilize
sleep 30
```

### Inconsistent Test Results

**Cause:** System resource constraints

**Solution:**
- Close other applications
- Check Docker resource limits
- Increase Docker memory/CPU allocation
- Consider running on dedicated test machine

---

## Advanced Testing Scenarios

### Custom Load Testing

Modify `test-load-distribution.sh` to test specific scenarios:

```bash
# Edit sustained load parameters
# Line 89-91:
local duration=60  # Increase to 60 seconds
local requests_per_second=10  # Increase to 10 req/s
```

### Stress Testing

Combine with external load tools:

```bash
# Run load tests in background
./scripts/devnet/test-load-distribution.sh --sustained &

# Add external load via Apache Bench
ab -n 1000 -c 10 http://localhost:30658/status

# Or use wrk
wrk -t4 -c100 -d30s http://localhost:30658/status
```

### Custom Chaos Scenarios

Create custom failure scenarios:

```bash
# Stop specific nodes manually
docker stop paw-node2 paw-node3

# Monitor recovery
watch -n 2 'curl -s http://localhost:26657/status | jq ".result.sync_info.latest_block_height"'

# Restart after testing
docker start paw-node2 paw-node3
```

---

## Best Practices

### Before Testing

1. ✅ Clean state: `docker compose down -v && rm -f scripts/devnet/.state/*.json`
2. ✅ Fresh genesis: `./scripts/devnet/setup-validators.sh 4`
3. ✅ Start network: `docker compose up -d`
4. ✅ Wait for stability: `sleep 60`

### During Testing

1. ✅ Run tests sequentially (avoid parallel test execution)
2. ✅ Monitor logs: `docker logs paw-node1 -f` in separate terminal
3. ✅ Check system resources: `docker stats`
4. ✅ Save test reports for later analysis

### After Testing

1. ✅ Review test reports in `/tmp/`
2. ✅ Verify network state: All nodes running, blocks produced
3. ✅ Clean up if switching configurations
4. ✅ Document any anomalies or unexpected behavior

---

## Test Coverage Summary

| Test Suite | Coverage | Duration |
|------------|----------|----------|
| Basic Scenarios | Functionality, connectivity, peers | ~2 min |
| Load Distribution | Performance, latency, load balancing | ~30 sec |
| Network Chaos | Resilience, recovery, fault tolerance | ~4 min |
| **Total** | **Complete sentry validation** | **~7 min** |

---

## Next Steps

After validating the sentry architecture locally:

1. **Production Deployment:** Use sentry patterns for mainnet/testnet
2. **Monitoring Setup:** Add Prometheus/Grafana for sentry metrics
3. **Geographic Distribution:** Deploy sentries in multiple regions
4. **DDoS Protection:** Configure rate limiting on sentry APIs
5. **Advanced Testing:** Integrate with CI/CD pipeline

---

## Support

For issues with testing:

1. Check test reports in `/tmp/paw-sentry-test-report-*.txt`
2. Review logs: `docker logs <container> 2>&1 | tail -100`
3. Consult main docs: [SENTRY_ARCHITECTURE.md](SENTRY_ARCHITECTURE.md)
4. Verify configuration: [TESTNET_QUICK_REFERENCE.md](TESTNET_QUICK_REFERENCE.md)

---

**Remember:** Automated testing validates the sentry architecture is production-ready for real-world network conditions.
