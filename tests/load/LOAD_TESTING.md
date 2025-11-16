# PAW Blockchain Load Testing Guide

This guide explains how to run comprehensive load tests on the PAW blockchain to evaluate performance, identify bottlenecks, and ensure the system can handle production workloads.

## Table of Contents

- [Overview](#overview)
- [Tools](#tools)
- [Getting Started](#getting-started)
- [Test Scenarios](#test-scenarios)
- [Running Tests](#running-tests)
- [Interpreting Results](#interpreting-results)
- [Performance Targets](#performance-targets)
- [Troubleshooting](#troubleshooting)

## Overview

The PAW load testing infrastructure includes multiple complementary tools:

- **k6** - Modern load testing tool for HTTP/WebSocket testing
- **Locust** - Python-based distributed load testing
- **tm-load-test** - Tendermint-specific load testing
- **gotester** - Custom Go-based load tester for blockchain operations

Each tool has specific strengths and use cases.

## Tools

### k6

**Best for:** HTTP API testing, WebSocket testing, complex scenarios

**Installation:**

```bash
# Windows (via Chocolatey)
choco install k6

# macOS
brew install k6

# Linux
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Locust

**Best for:** Distributed testing, Python scripting, custom scenarios

**Installation:**

```bash
pip install locust
```

### tm-load-test

**Best for:** Tendermint consensus layer testing, raw transaction throughput

**Installation:**

```bash
go install github.com/informalsystems/tm-load-test@latest
```

### Custom Go Tester

**Best for:** Blockchain-specific operations, detailed metrics

**Build:**

```bash
cd tests/load/gotester
go build -o gotester main.go
```

## Getting Started

### 1. Start Your Test Network

```bash
# Start local testnet
make localnet-start

# Or use development environment
make dev
```

### 2. Verify Connectivity

```bash
# Check API endpoint
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info

# Check RPC endpoint
curl http://localhost:26657/status
```

### 3. Configure Tests

Edit `tests/load/config.yaml` to adjust test parameters for your environment.

## Test Scenarios

### Light Load (Functional Testing)

- **Duration:** 5 minutes
- **Users:** 10
- **Target TPS:** 10
- **Purpose:** Verify basic functionality

```bash
make load-test SCENARIO=light
```

### Normal Load (Production Simulation)

- **Duration:** 10 minutes
- **Users:** 100
- **Target TPS:** 100
- **Purpose:** Simulate typical production load

```bash
make load-test SCENARIO=normal
```

### Peak Load

- **Duration:** 15 minutes
- **Users:** 500
- **Target TPS:** 500
- **Purpose:** Test peak usage capacity

```bash
make load-test SCENARIO=peak
```

### Stress Test

- **Duration:** 30 minutes
- **Users:** 1000
- **Target TPS:** 1000
- **Purpose:** Find system limits

```bash
make load-test SCENARIO=stress
```

### Endurance Test

- **Duration:** 2 hours
- **Users:** 200
- **Target TPS:** 200
- **Purpose:** Test long-term stability

```bash
make load-test SCENARIO=endurance
```

## Running Tests

### k6 Tests

#### Blockchain Load Test

```bash
k6 run tests/load/k6/blockchain-load-test.js
```

#### DEX-Specific Test

```bash
k6 run tests/load/k6/dex-swap-test.js
```

#### WebSocket Test

```bash
k6 run tests/load/k6/websocket-test.js
```

#### With Custom Options

```bash
k6 run --vus 100 --duration 10m tests/load/k6/blockchain-load-test.js
```

#### With Environment Variables

```bash
BASE_URL=http://localhost:1317 \
RPC_URL=http://localhost:26657 \
k6 run tests/load/k6/blockchain-load-test.js
```

### Locust Tests

#### Web UI Mode (Recommended)

```bash
locust -f tests/load/locust/locustfile.py
```

Then open http://localhost:8089 in your browser.

#### Headless Mode

```bash
locust -f tests/load/locust/locustfile.py \
  --headless \
  --users 100 \
  --spawn-rate 10 \
  --run-time 10m \
  --host http://localhost:1317
```

#### Distributed Testing

```bash
# Master node
locust -f tests/load/locust/locustfile.py --master

# Worker nodes (run on multiple machines)
locust -f tests/load/locust/locustfile.py --worker --master-host=<master-ip>
```

### Tendermint Load Test

```bash
tm-load-test \
  -c 10 \
  -T 60 \
  -r 100 \
  -s 250 \
  --broadcast-tx-method async \
  --endpoints ws://localhost:26657/websocket
```

**Options:**

- `-c` - Concurrent connections
- `-T` - Test duration (seconds)
- `-r` - Transaction rate (per second)
- `-s` - Transaction size (bytes)

### Custom Go Tester

```bash
cd tests/load/gotester

# Basic run
./gotester \
  --rpc http://localhost:26657 \
  --api http://localhost:1317 \
  --duration 5m \
  --concurrency 10 \
  --rate 100 \
  --type transactions

# DEX-focused test
./gotester \
  --duration 10m \
  --concurrency 20 \
  --rate 200 \
  --type dex \
  --output dex-test-report.json

# Mixed workload
./gotester \
  --duration 15m \
  --concurrency 50 \
  --rate 500 \
  --type mixed
```

### Go Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./tests/benchmarks/

# Specific module
go test -bench=. -benchmem ./tests/benchmarks/ -run BenchmarkDEX

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./tests/benchmarks/
go tool pprof -http=:8080 cpu.prof

# With memory profiling
go test -bench=. -memprofile=mem.prof ./tests/benchmarks/
go tool pprof -http=:8080 mem.prof
```

## Interpreting Results

### Key Metrics

#### Throughput

- **TPS (Transactions Per Second):** Number of successful transactions per second
- **Target:** 100+ TPS for normal load, 1000+ TPS peak capacity

#### Latency

- **p50 (Median):** 50% of requests complete within this time
- **p95:** 95% of requests complete within this time
- **p99:** 99% of requests complete within this time
- **Target:** p95 < 500ms for queries, p95 < 2s for transactions

#### Error Rate

- **HTTP Errors:** Failed HTTP requests
- **Transaction Errors:** Failed transaction submissions
- **Target:** < 1% error rate

#### Resource Usage

- **CPU:** Processor utilization
- **Memory:** RAM usage and growth
- **Disk I/O:** Read/write operations
- **Network:** Bandwidth usage

### k6 Output

```
     ✓ status is 200
     ✓ response time OK

     checks.........................: 100.00% ✓ 45000      ✗ 0
     data_received..................: 15 MB   50 kB/s
     data_sent......................: 8.5 MB  28 kB/s
     http_req_duration..............: avg=123.45ms min=45.23ms med=98.76ms max=987.65ms p(95)=234.56ms p(99)=456.78ms
     http_req_failed................: 0.12%   ✓ 54        ✗ 44946
     iterations.....................: 45000   150/s
     transaction_latency............: avg=1.2s min=0.5s med=1.1s max=4.5s p(95)=2.3s p(99)=3.8s
```

**Good signs:**

- ✓ checks near 100%
- p95 latency meeting targets
- Low error rate (< 1%)
- Stable performance over time

**Warning signs:**

- Increasing error rates
- Growing latencies
- Memory leaks (increasing RAM usage)
- Timeout errors

### Locust Output

```
Type     Name                              # reqs      # fails  |     Avg     Min     Max  Median  |   req/s failures/s
--------|----------------------------------|------------|---------|-------|-------|-------|--------|--------|-----------
GET      /cosmos/bank/v1beta1/balances    45000       12       |     156      45     987     120   |   150.0      0.04
POST     /cosmos/tx/v1beta1/txs           15000       8        |    1234     567    4567    1100   |    50.0      0.03
```

## Performance Targets

### Minimum Requirements

- **TPS:** 50+ sustained
- **Latency (p95):** < 1s for queries, < 3s for transactions
- **Error Rate:** < 2%
- **Uptime:** 99%+

### Production Targets

- **TPS:** 100+ sustained, 500+ peak
- **Latency (p95):** < 500ms for queries, < 2s for transactions
- **Error Rate:** < 1%
- **Uptime:** 99.9%+

### Optimal Performance

- **TPS:** 1000+ sustained
- **Latency (p95):** < 200ms for queries, < 1s for transactions
- **Error Rate:** < 0.1%
- **Uptime:** 99.99%+

## Troubleshooting

### High Error Rates

**Symptoms:** > 5% failed requests

**Possible causes:**

- Insufficient resources (CPU/RAM)
- Network congestion
- Database bottlenecks
- Rate limiting

**Solutions:**

```bash
# Check node logs
journalctl -u pawd -f

# Monitor resources
htop

# Check database
# Look for slow queries, connection limits

# Adjust test parameters
# Reduce concurrent users or transaction rate
```

### High Latency

**Symptoms:** p95 > 1s for queries, p95 > 5s for transactions

**Possible causes:**

- Slow database queries
- Network latency
- Insufficient processing power
- Memory swapping

**Solutions:**

```bash
# Profile the application
go tool pprof http://localhost:6060/debug/pprof/profile

# Check for slow queries
# Enable query logging

# Optimize database indexes
# Add indexes to frequently queried fields

# Scale resources
# Increase CPU/RAM allocation
```

### Memory Leaks

**Symptoms:** Continuously growing memory usage

**Solutions:**

```bash
# Profile memory
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# Look for goroutine leaks
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

### Test Failures

**Connection refused:**

```bash
# Ensure node is running
curl http://localhost:26657/status

# Check firewall rules
# Verify ports 1317 and 26657 are open
```

**WebSocket errors:**

```bash
# Test WebSocket connection
wscat -c ws://localhost:26657/websocket

# Check nginx/proxy configuration if using one
```

## Best Practices

1. **Start Small:** Begin with light load tests and gradually increase
2. **Monitor Resources:** Watch CPU, RAM, disk, network during tests
3. **Run Multiple Times:** Get consistent results across multiple runs
4. **Test Incrementally:** Test one change at a time
5. **Document Results:** Keep records of test configurations and results
6. **Test Realistic Scenarios:** Match production usage patterns
7. **Automate:** Integrate load tests into CI/CD pipeline
8. **Set Baselines:** Establish performance baselines for comparison
9. **Test Degradation:** Test how system degrades under extreme load
10. **Plan for Scaling:** Identify scaling points before they're needed

## Continuous Testing

### CI/CD Integration

Add to your CI pipeline:

```yaml
# .github/workflows/load-test.yml
name: Load Tests
on:
  schedule:
    - cron: '0 2 * * *' # Daily at 2 AM

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Start testnet
        run: make localnet-start
      - name: Run k6 tests
        run: k6 run tests/load/k6/blockchain-load-test.js
      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: load-test-results
          path: tests/load/reports/
```

### Scheduled Testing

```bash
# Add to crontab
0 2 * * * cd /path/to/paw && ./scripts/run-load-test.sh
```

## Additional Resources

- [k6 Documentation](https://k6.io/docs/)
- [Locust Documentation](https://docs.locust.io/)
- [Cosmos SDK Performance](https://docs.cosmos.network/)
- [Tendermint Performance Tuning](https://docs.tendermint.com/)

## Support

For load testing issues or questions:

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Discord: #load-testing channel
- Documentation: https://docs.paw.network/load-testing
