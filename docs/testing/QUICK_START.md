# Testing Infrastructure Quick Start

Quick commands to run all testing features in PAW Chain.

## Prerequisites

```bash
# Install dependencies
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

# Install K6 (Linux)
sudo apt-get install k6

# Or use setup scripts
./tests/load/k6/setup.sh
```

## Quick Commands

### 1. Run All Tests (Parallel Enabled)

```bash
# Run all tests with 8 parallel workers
go test -parallel=8 -v ./tests/...

# Run specific test directories
go test -parallel=4 -v ./tests/property/...
go test -parallel=4 -v ./tests/differential/...
```

### 2. Mutation Testing

```bash
# Run mutation tests on all modules
./scripts/testing/run_mutation_tests.sh

# View results
cat test-results/mutation/mutation_report.md
```

### 3. Benchmark Tracking

```bash
# Run benchmarks with regression detection
./scripts/testing/track_benchmarks.sh

# View summary
cat test-results/benchmarks/benchmark_summary.md
```

### 4. Load Testing

```bash
# Run all load tests
cd tests/load/k6
./run-load-tests.sh

# Run individual tests
k6 run -e BASE_URL=http://localhost:1317 dex-load-test.js
k6 run -e BASE_URL=http://localhost:1317 oracle-load-test.js
k6 run -e BASE_URL=http://localhost:1317 compute-load-test.js

# View HTML reports
open ../../test-results/load-tests/dex-report.html
```

## One-Liner Test Suite

```bash
# Complete test suite
go test -parallel=8 ./tests/... && \
./scripts/testing/track_benchmarks.sh && \
./scripts/testing/run_mutation_tests.sh && \
cd tests/load/k6 && ./run-load-tests.sh
```

## CI Commands

```bash
# Trigger mutation testing (manual)
gh workflow run mutation-testing.yml

# Trigger nightly load tests (manual)
gh workflow run nightly-load-tests.yml

# View workflow status
gh run list --workflow=mutation-testing.yml
gh run list --workflow=nightly-load-tests.yml
```

## Results Locations

```
test-results/
├── benchmarks/
│   ├── bench_TIMESTAMP.json          # Latest benchmark results
│   ├── benchmark_summary.md          # Human-readable summary
│   └── history/                      # Historical benchmarks
│       └── bench_*.json
├── mutation/
│   ├── mutation_summary.json         # JSON summary
│   ├── mutation_report.md           # Markdown report
│   └── *_mutations.txt              # Detailed mutation logs
└── load-tests/
    ├── dex-results.json             # K6 raw results
    ├── dex-summary.json             # K6 summary
    ├── dex-report.html              # HTML report
    └── performance-report.md        # Combined report
```

## Troubleshooting

### Tests hanging?
```bash
# Add timeout
go test -timeout=10m -parallel=8 ./tests/...
```

### Benchmarks inconsistent?
```bash
# Run multiple times
go test -bench=. -count=10 ./tests/benchmarks/...
```

### Load tests failing?
```bash
# Check node is running
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info

# Reduce load
# Edit load test file and decrease virtual users
```

## Performance Tips

1. **Parallel Tests**: 47% faster test execution
2. **Mutation Testing**: Run weekly, not on every commit
3. **Benchmark Tracking**: Use for PR reviews, not local dev
4. **Load Testing**: Run nightly, investigate failures promptly

## Next Steps

- Read full documentation: `docs/testing/TESTING_INFRASTRUCTURE.md`
- Review test patterns: `docs/implementation/testing/GO_TESTING_GUIDE.md`
- Check CI configuration: `hub/workflows/`

---

**Quick Links**:
- [Full Testing Infrastructure Docs](./TESTING_INFRASTRUCTURE.md)
- [Go Testing Guide](../implementation/testing/GO_TESTING_GUIDE.md)
- [Advanced Testing Summary](../implementation/testing/ADVANCED_TESTING_IMPLEMENTATION_SUMMARY.md)
