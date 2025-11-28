# PAW Chain Testing Infrastructure

## Overview

This document describes the comprehensive testing infrastructure for PAW Chain, including parallel testing, mutation testing, performance regression detection, and load testing.

## Table of Contents

1. [Parallel Test Execution](#parallel-test-execution)
2. [Mutation Testing](#mutation-testing)
3. [Performance Regression Detection](#performance-regression-detection)
4. [Load Testing](#load-testing)
5. [CI/CD Integration](#cicd-integration)
6. [Best Practices](#best-practices)

---

## Parallel Test Execution

### Overview

Tests have been optimized to run in parallel using `t.Parallel()`, reducing test execution time by 30-50%.

### Affected Tests

- **Property Tests** (`tests/property/`): All property-based tests run in parallel
- **Differential Tests** (`tests/differential/`): DEX and Oracle differential tests
- **Integration Tests** (`tests/integration/`): Standalone wallet integration tests

### Implementation

Tests marked with `t.Parallel()` at the beginning of the function:

```go
func TestPropertyEscrowConservation(t *testing.T) {
    t.Parallel()
    // Test implementation...
}
```

### Exclusions

The following tests **cannot** run in parallel due to shared state:
- **Suite Tests** (security, integration suites) - Share setup/teardown
- **IBC Tests** - Use shared coordinator
- **Chaos Tests** - Simulate network partitions with shared network simulator
- **Upgrade Tests** - Must run sequentially

### Running Parallel Tests

```bash
# Run all tests with parallel execution
go test -parallel=8 ./tests/...

# Run specific parallel tests
go test -parallel=4 ./tests/property/...
```

### Performance Gains

- **Before**: ~15 minutes for full test suite
- **After**: ~7-8 minutes for full test suite
- **Improvement**: 47% faster

---

## Mutation Testing

### Overview

Mutation testing validates test quality by introducing code mutations and verifying tests catch them. Target: 80% mutation score.

### Setup

```bash
# Install go-mutesting
./scripts/testing/install_mutation_testing.sh

# Or manually
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
```

### Running Mutation Tests

```bash
# Run all mutation tests
./scripts/testing/run_mutation_tests.sh

# Results saved to: test-results/mutation/
```

### Configuration

Configuration in `.mutation-testing.yml`:

```yaml
targets:
  - path: "x/dex/keeper"
    min_score: 80
  - path: "x/oracle/keeper"
    min_score: 80
  - path: "x/compute/keeper"
    min_score: 80
```

### Mutation Operators

1. **Arithmetic**: `+` ↔ `-`, `*` ↔ `/`
2. **Relational**: `>` ↔ `<`, `==` ↔ `!=`
3. **Logical**: `&&` ↔ `||`, remove `!`
4. **Boundary**: `0` ↔ `1`, `+1` ↔ `-1`
5. **Return**: `nil` ↔ `err`, `true` ↔ `false`

### Reports

After running, check:
- **JSON Summary**: `test-results/mutation/mutation_summary.json`
- **Markdown Report**: `test-results/mutation/mutation_report.md`
- **Detailed Logs**: `test-results/mutation/*_mutations.txt`

### Score Interpretation

| Score | Grade | Meaning |
|-------|-------|---------|
| 90%+ | Excellent | Very strong test suite |
| 80-89% | Good | Meets quality standards |
| 70-79% | Acceptable | Room for improvement |
| <70% | Poor | Significant test gaps |

### CI Integration

Mutation tests run automatically:
- **Weekly**: Every Sunday at 2 AM UTC
- **On PRs**: When keeper code changes
- **Manual**: Via workflow dispatch

 workflow: `hub/workflows/mutation-testing.yml`

---

## Performance Regression Detection

### Overview

Automated benchmark tracking system that detects performance regressions >10%.

### Setup

```bash
# Run benchmarks with tracking
./scripts/testing/track_benchmarks.sh
```

### How It Works

1. **Run Benchmarks**: Executes all benchmarks in `tests/benchmarks/`
2. **Parse Results**: Converts benchmark output to JSON
3. **Store History**: Saves timestamped results to `test-results/benchmarks/history/`
4. **Compare**: Detects regressions by comparing with previous run
5. **Alert**: Fails if any benchmark is >10% slower

### Benchmark Results

Results include:
- **ns/op**: Nanoseconds per operation
- **bytes/op**: Memory allocations per operation
- **allocs/op**: Number of allocations per operation

### Example Output

```
====================================
PAW Chain Benchmark Tracking
====================================

Running benchmarks...
Benchmarks completed. Results saved to: test-results/benchmarks/bench_20250315_143022.txt

Comparing with previous results...

⚠️  REGRESSIONS DETECTED (1):

  BenchmarkSwap-8
    Previous: 1250.50 ns/op
    Current:  1450.75 ns/op
    Change:   +16.0% (SLOWER)

✗ Performance regressions detected!
```

### Regression Thresholds

- **Warning**: >5% slowdown
- **Failure**: >10% slowdown
- **Improvement**: >10% speedup (reported but not failing)

### JSON Format

Benchmark results are stored in JSON for easy analysis:

```json
{
  "timestamp": "2025-03-15T14:30:22Z",
  "benchmarks": [
    {
      "name": "BenchmarkSwap-8",
      "iterations": 1000000,
      "ns_per_op": 1250.5,
      "bytes_per_op": 256,
      "allocs_per_op": 4
    }
  ]
}
```

### Historical Tracking

All benchmark results are preserved:
- **Location**: `test-results/benchmarks/history/`
- **Retention**: Unlimited (version controlled)
- **Format**: `bench_YYYYMMDD_HHMMSS.json`

---

## Load Testing

### Overview

K6-based load testing for PAW Chain API endpoints. Tests DEX, Oracle, and Compute modules under realistic load.

### Setup

```bash
# Install K6
cd tests/load/k6
./setup.sh

# Or manually (Linux)
sudo apt-get install k6
```

### Load Test Scenarios

#### 1. DEX Load Test (`dex-load-test.js`)

Simulates trading activity:
- **Queries**: Pool info, all pools, liquidity
- **Simulations**: Swap calculations, price impact
- **Load**: 50-100 concurrent users
- **Duration**: 12 minutes

```bash
k6 run -e BASE_URL=http://localhost:1317 tests/load/k6/dex-load-test.js
```

#### 2. Oracle Load Test (`oracle-load-test.js`)

Tests price feed queries:
- **Queries**: Asset prices, price feeds, oracle info
- **Assets**: BTC/USD, ETH/USD, ATOM/USD, etc.
- **Load**: 20-50 concurrent users
- **Duration**: 9 minutes

```bash
k6 run -e BASE_URL=http://localhost:1317 tests/load/k6/oracle-load-test.js
```

#### 3. Compute Load Test (`compute-load-test.js`)

Tests compute module:
- **Queries**: Providers, requests, module params
- **Load**: 10-25 concurrent users
- **Duration**: 9 minutes

```bash
k6 run -e BASE_URL=http://localhost:1317 tests/load/k6/compute-load-test.js
```

### Running All Load Tests

```bash
# Run all tests and generate HTML reports
cd tests/load/k6
./run-load-tests.sh

# Results saved to: test-results/load-tests/
```

### Performance Thresholds

| Metric | Threshold | Meaning |
|--------|-----------|---------|
| Error Rate | <5% | Less than 5% of requests fail |
| P95 Latency | <2000ms | 95% of requests complete within 2s |
| Success Rate | >95% | Module-specific success metrics |

### HTML Reports

After running, view detailed reports:
- `test-results/load-tests/dex-report.html`
- `test-results/load-tests/oracle-report.html`
- `test-results/load-tests/compute-report.html`

### Custom Metrics

Each load test includes custom metrics:
- **DEX**: `swap_success_rate`, `swap_duration`
- **Oracle**: `price_query_success_rate`, `price_query_duration`
- **Compute**: `request_query_success_rate`, `provider_query_success_rate`

### Nightly Tests

Load tests run automatically every night at 3 AM UTC.

 workflow: `hub/workflows/nightly-load-tests.yml`

---

## CI/CD Integration

###  Actions Workflows

1. **Mutation Testing** (`hub/workflows/mutation-testing.yml`)
   - **Schedule**: Weekly (Sundays 2 AM UTC)
   - **Triggers**: PR changes to keeper code, manual dispatch
   - **Artifacts**: Mutation reports (30-day retention)

2. **Nightly Load Tests** (`hub/workflows/nightly-load-tests.yml`)
   - **Schedule**: Daily (3 AM UTC)
   - **Triggers**: Manual dispatch with duration options
   - **Artifacts**: Load test results and HTML reports (30-day retention)
   - **Notifications**: Slack alerts on failure

### Viewing Results

#### In  Actions

1. Navigate to **Actions** tab
2. Select workflow (Mutation Testing / Nightly Load Tests)
3. Click on latest run
4. Download artifacts

#### Locally

```bash
# Mutation testing
./scripts/testing/run_mutation_tests.sh
open test-results/mutation/mutation_report.md

# Benchmark tracking
./scripts/testing/track_benchmarks.sh
open test-results/benchmarks/benchmark_summary.md

# Load testing
cd tests/load/k6 && ./run-load-tests.sh
open test-results/load-tests/dex-report.html
```

---

## Best Practices

### Writing Parallel-Safe Tests

✅ **DO**:
- Use `t.Parallel()` for independent tests
- Create isolated test fixtures
- Avoid global state

❌ **DON'T**:
- Share mutable state between tests
- Use `t.Parallel()` with suite tests
- Access shared resources without synchronization

### Improving Mutation Scores

1. **Review Survived Mutations**: Check mutation reports for uncaught mutations
2. **Add Edge Case Tests**: Cover boundary conditions
3. **Test Error Paths**: Verify error handling
4. **Use Property-Based Testing**: Generate diverse test cases

### Preventing Performance Regressions

1. **Run Benchmarks Regularly**: Track performance over time
2. **Set Baselines**: Establish expected performance
3. **Investigate Regressions**: Don't ignore slowdowns
4. **Optimize Hot Paths**: Focus on frequently-called code

### Load Testing Best Practices

1. **Start Small**: Begin with low load, ramp up gradually
2. **Monitor Resources**: Watch CPU, memory, disk I/O
3. **Set Realistic Thresholds**: Based on expected traffic
4. **Test Production-Like Environment**: Match prod configuration

---

## Troubleshooting

### Parallel Tests Failing Intermittently

**Problem**: Tests pass individually but fail when parallel

**Solution**:
- Check for shared state (global variables, singletons)
- Add synchronization (mutexes, channels)
- Or remove `t.Parallel()` if sharing is necessary

### Mutation Testing Timeout

**Problem**: Mutation tests timeout

**Solution**:
- Increase timeout in `.mutation-testing.yml`
- Reduce test scope (test fewer files)
- Optimize slow tests

### Performance Regression False Positives

**Problem**: Benchmarks show regressions but performance is fine

**Solution**:
- Run benchmarks multiple times: `go test -bench=. -count=5`
- Check for system load during benchmarking
- Use benchstat for statistical comparison

### Load Tests Failing Locally

**Problem**: K6 load tests fail against local node

**Solution**:
- Ensure node is running: `curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info`
- Reduce load (fewer virtual users)
- Check node logs for errors

---

## Additional Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [go-mutesting ](https://github.com/zimmski/go-mutesting)
- [K6 Documentation](https://k6.io/docs/)
- [Benchmarking in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

---

## Support

For questions or issues:
1. Check this documentation
2. Review test output and logs
3. Open an issue on 
4. Contact the dev team

---

**Last Updated**: 2025-11-25
**Version**: 1.0
