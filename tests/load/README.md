# PAW Blockchain Load Testing

This directory contains comprehensive load testing infrastructure for the PAW blockchain.

## Quick Start

```bash
# Install load testing tools
make dev-setup

# Run all load tests
make load-test-all

# Or run individual tests
make load-test           # k6 blockchain test
make load-test-dex       # k6 DEX test
make load-test-locust-ui # Locust with web UI
```

## Directory Structure

```
tests/load/
├── k6/                      # k6 load testing scripts
│   ├── blockchain-load-test.js   # General blockchain API testing
│   ├── dex-swap-test.js          # DEX-specific load testing
│   └── websocket-test.js         # WebSocket connection testing
├── locust/                  # Locust load testing
│   └── locustfile.py             # Python-based load scenarios
├── tm-load-test/           # Tendermint load testing
│   └── config.toml              # tm-load-test configuration
├── gotester/               # Custom Go load tester
│   └── main.go                  # Blockchain-specific tester
├── reports/                # Test results and reports
├── config.yaml            # Global load test configuration
├── LOAD_TESTING.md       # Comprehensive documentation
└── README.md             # This file
```

## Testing Tools

### k6 (HTTP/WebSocket Testing)

- Fast, modern load testing tool
- JavaScript-based test scripts
- Great for API and WebSocket testing
- Built-in metrics and thresholds

**Example:**

```bash
k6 run tests/load/k6/blockchain-load-test.js
```

### Locust (Python Distributed Testing)

- Python-based load testing
- Web UI for real-time monitoring
- Distributed testing support
- Flexible scenario definitions

**Example:**

```bash
# Web UI mode
locust -f tests/load/locust/locustfile.py

# Headless mode
make load-test-locust
```

### tm-load-test (Tendermint Testing)

- Tendermint consensus layer testing
- Raw transaction throughput measurement
- Direct RPC/WebSocket testing

**Example:**

```bash
tm-load-test -c 10 -T 60 -r 100 --endpoints ws://localhost:26657/websocket
```

### Custom Go Tester

- Blockchain-specific operations
- Detailed performance metrics
- Custom transaction patterns

**Example:**

```bash
cd tests/load/gotester
./gotester --duration 5m --concurrency 10 --rate 100 --type mixed
```

## Test Scenarios

Load tests are organized into scenarios defined in `config.yaml`:

1. **Light Load** (10 users, 5 min)
   - Functional testing
   - Basic performance validation

2. **Normal Load** (100 users, 10 min)
   - Expected production load
   - Baseline performance

3. **Peak Load** (500 users, 15 min)
   - Peak usage hours
   - Scalability testing

4. **Stress Test** (1000 users, 30 min)
   - System limits
   - Breaking point identification

5. **Endurance Test** (200 users, 2 hours)
   - Long-term stability
   - Memory leak detection

## Performance Targets

| Metric              | Minimum | Target  | Optimal |
| ------------------- | ------- | ------- | ------- |
| TPS                 | 50+     | 100+    | 1000+   |
| Query Latency (p95) | < 1s    | < 500ms | < 200ms |
| TX Latency (p95)    | < 3s    | < 2s    | < 1s    |
| Error Rate          | < 2%    | < 1%    | < 0.1%  |

## Quick Commands

```bash
# All tests
make load-test-all

# Individual k6 tests
make load-test
make load-test-dex
make load-test-websocket

# Locust tests
make load-test-locust        # Headless
make load-test-locust-ui     # Web UI

# Go benchmarks
make benchmark
make benchmark-dex
make perf-profile
make perf-profile-interactive

# Custom scenarios
SCENARIO=stress make load-test-all
BASE_URL=http://api.testnet.paw.network make load-test
```

## Interpreting Results

### k6 Metrics

- **http_req_duration**: Request latency (p95, p99)
- **http_req_failed**: Error rate
- **iterations**: Total requests
- **vus**: Virtual users

### Locust Metrics

- **RPS**: Requests per second
- **Response time**: Latency percentiles
- **Failures**: Error count and types

### Go Benchmarks

- **ns/op**: Nanoseconds per operation
- **B/op**: Bytes allocated per operation
- **allocs/op**: Number of allocations

## Reports

Test results are saved in `tests/load/reports/`:

- HTML reports with visualizations
- JSON data for further analysis
- Performance profiles (CPU, memory)

View the latest report:

```bash
# Linux/Mac
xdg-open tests/load/reports/load-test-latest.html

# Windows
start tests/load/reports/load-test-latest.html
```

## Troubleshooting

### Connection Refused

```bash
# Ensure blockchain is running
make localnet-start

# Check endpoints
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info
curl http://localhost:26657/status
```

### High Error Rates

- Check node logs: `journalctl -u pawd -f`
- Reduce concurrent users
- Check database performance
- Monitor system resources

### Memory Issues

- Run memory profiling: `make perf-profile`
- Check for goroutine leaks
- Monitor with `htop` or similar

## Advanced Usage

### Custom Test Duration

```bash
k6 run --duration 30m --vus 200 tests/load/k6/blockchain-load-test.js
```

### Distributed Locust Testing

```bash
# Master
locust -f tests/load/locust/locustfile.py --master

# Workers (different machines)
locust -f tests/load/locust/locustfile.py --worker --master-host=<master-ip>
```

### Profile Analysis

```bash
# Run with profiling
make perf-profile

# Analyze CPU profile
go tool pprof -http=:8080 tests/load/reports/cpu-latest.prof

# Analyze memory profile
go tool pprof -http=:8080 tests/load/reports/mem-latest.prof
```

## CI/CD Integration

Load tests can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Load Tests
  run: |
    make localnet-start
    make load-test-all

- name: Upload Reports
  uses: actions/upload-artifact@v2
  with:
    name: load-test-reports
    path: tests/load/reports/
```

## Documentation

For detailed information, see:

- [LOAD_TESTING.md](LOAD_TESTING.md) - Complete guide
- [config.yaml](config.yaml) - Configuration reference
- [k6 Documentation](https://k6.io/docs/)
- [Locust Documentation](https://docs.locust.io/)

## Support

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Discord: #load-testing
- Documentation: https://docs.paw.network/load-testing
