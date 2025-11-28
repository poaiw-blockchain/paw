# Gas Metering Tests - Quick Reference

## Files Overview

| File | Size | Purpose |
|------|------|---------|
| `README.md` | 9.4 KB | Complete testing documentation |
| `compute_gas_test.go` | 13 KB | Compute module gas tests |
| `dex_gas_test.go` | 13 KB | DEX module gas tests |
| `oracle_gas_test.go` | 15 KB | Oracle module gas tests |
| `dos_prevention_test.go` | 14 KB | DoS attack prevention tests |
| `benchmarks_test.go` | 11 KB | Gas benchmarking suite |
| `test_helpers.go` | 13 KB | Test helper functions |
| `GAS_TESTING_IMPLEMENTATION_REPORT.md` | 14 KB | Complete implementation report |
| `QUICK_REFERENCE.md` | This file | Quick reference guide |
| `../config/gas_limits.yaml` | 9.6 KB | Gas limits configuration |

**Total:** ~120 KB, 2,673 lines of code

## Quick Gas Limits Reference

### Compute Module

```
RegisterProvider:    50k - 150k gas   (baseline: 100k)
SubmitRequest:       80k - 200k gas   (baseline: 120k) + 200 gas/byte
SubmitResult:        100k - 300k gas  (baseline: 150k)
ZKVerification:      1M - 5M gas      (baseline: 2.5M) ⚠️ HIGH
LockEscrow:          30k - 80k gas    (baseline: 50k)
ReleaseEscrow:       40k - 100k gas   (baseline: 60k)
RefundEscrow:        40k - 100k gas   (baseline: 60k)
UpdateProvider:      30k - 80k gas    (baseline: 50k)
DeactivateProvider:  25k - 60k gas    (baseline: 40k)
```

### DEX Module

```
CreatePool:          80k - 200k gas   (baseline: 120k)
Swap:                60k - 150k gas   (baseline: 80k)
AddLiquidity:        50k - 120k gas   (baseline: 70k)
RemoveLiquidity:     50k - 120k gas   (baseline: 70k)
ConstantProduct:     5k - 10k gas     (baseline: 7k)
CircuitBreaker:      10k - 30k gas    (baseline: 20k)
PriceImpact:         8k - 20k gas     (baseline: 12k)
SwapOutput:          6k - 15k gas     (baseline: 10k)
```

### Oracle Module

```
RegisterOracle:      40k - 100k gas   (baseline: 60k)
SubmitPrice:         30k - 100k gas   (baseline: 50k)
AggregateVotes:      100k - 3M gas    (baseline: 250k for 7 validators)
OutlierDetection:    200k - 1M gas    (baseline: 400k) ⚠️ HIGH
TWAPCalculation:     50k - 200k gas   (baseline: 100k)
Volatility:          100k - 500k gas  (baseline: 250k) ⚠️ HIGH
SlashOracle:         80k - 200k gas   (baseline: 120k)
MedianCalculation:   30k - 300k gas   (baseline: 80k)
PriceAgeCheck:       3k - 10k gas     (baseline: 5k)
```

## Running Tests

### Quick Test Commands

```bash
# All gas tests
go test -v ./tests/gas/...

# Specific module
go test -v ./tests/gas -run TestCompute
go test -v ./tests/gas -run TestDEX
go test -v ./tests/gas -run TestOracle

# DoS prevention
go test -v ./tests/gas -run TestDoS

# Benchmarks
go test -bench=. -benchmem ./tests/gas

# With coverage
go test -v -coverprofile=coverage.out ./tests/gas/...
```

## Gas Scaling Formulas

### Oracle Aggregation
```
Gas = 100,000 + (ValidatorCount × 25,000)

7 validators   = 275,000 gas
21 validators  = 625,000 gas
50 validators  = 1,350,000 gas
100 validators = 2,600,000 gas
```

### Compute Request
```
Gas = 80,000 + (InputBytes × 200)

1 KB   = 284,800 gas
10 KB  = 2,128,000 gas
100 KB = 20,560,000 gas (exceeds tx limit!)
```

### DEX Swap
```
Gas = 60,000 + (InputBytes × 100)
```

## DoS Prevention Limits

```yaml
Max Input Size:       1 MB
Max Proof Size:       256 KB
Max Batch Ops:        100
Max Iterations:       1,000
Max Nested Calls:     10
Max Price Impact:     10%
Max Pool Swap Size:   30% of pool
```

## High Gas Operations ⚠️

Watch these operations - they consume significant gas:

1. **ZK Verification:** 1M - 5M gas
2. **Oracle Aggregation (100 validators):** ~2.6M gas
3. **Outlier Detection:** 200k - 1M gas
4. **Volatility Calculation:** 100k - 500k gas

## Block Limits

```
Max Gas per Block: 50,000,000 gas
Max Gas per TX:    10,000,000 gas
Buffer:            20% (1.2x multiplier)
```

### Operations per Block (estimated)

- ZK Verifications: ~10
- Oracle Aggregations (100 validators): ~19
- DEX Swaps: ~625
- Compute Requests: ~250

## Gas Prices

```
Minimum:      0.025 upaw
Low Priority: 0.05 upaw
Average:      0.1 upaw
High:         0.5 upaw
Urgent:       1.0 upaw
```

### Transaction Cost Examples (at 0.1 upaw)

```
Simple Swap:          ~8,000 upaw ($0.008)
Compute Request:      ~88,000 upaw ($0.088)
ZK Verification:      ~250,000 upaw ($0.25)
Oracle Aggregation:   ~300,000 upaw ($0.30)
```

## Test Statistics

- **Total Test Cases:** 59
- **Total Benchmarks:** 15
- **Operations Tested:** 26
- **DoS Scenarios:** 17
- **Modules Covered:** 3

## Common Test Patterns

### Gas Limit Test
```go
ctx = ctx.WithGasMeter(sdk.NewGasMeter(maxGas))
err := keeper.Operation(ctx, params...)
gasUsed := ctx.GasMeter().GasConsumed()
require.Less(t, gasUsed, maxGas)
```

### Scaling Test
```go
for _, size := range sizes {
    gasUsed := testWithSize(size)
    gasPerUnit := gasUsed / size
    require.Less(t, gasPerUnit, limit)
}
```

### Regression Test
```go
baseline := 100000
tolerance := 15000
require.InDelta(t, baseline, gasUsed, tolerance)
```

## Key Files to Review

1. **Start Here:** `README.md` - Complete testing methodology
2. **Configuration:** `../config/gas_limits.yaml` - All gas limits
3. **Report:** `GAS_TESTING_IMPLEMENTATION_REPORT.md` - Full details
4. **Tests:** `*_gas_test.go` - Module-specific tests
5. **DoS:** `dos_prevention_test.go` - Attack prevention
6. **Benchmarks:** `benchmarks_test.go` - Performance testing

## Maintenance

### Daily
- Run full test suite in CI/CD
- Monitor gas usage in production

### Weekly
- Review gas benchmark trends
- Check for anomalies

### Monthly
- Update baselines if optimizations made
- Review and adjust limits if needed

### Per Release
- Comprehensive gas regression analysis
- Update documentation

## Alert Thresholds

Monitor for:
- Gas usage exceeding 80% of limits
- Operation gas changed >20% from baseline
- Out of gas errors spike
- Unusual consumption patterns

## Getting Help

1. Read `README.md` for detailed methodology
2. Check `GAS_TESTING_IMPLEMENTATION_REPORT.md` for comprehensive analysis
3. Review `config/gas_limits.yaml` for current limits
4. Run tests with `-v` flag for detailed output
5. Run benchmarks to establish new baselines

## Version

- **Version:** 1.0
- **Last Updated:** 2025-11-25
- **Status:** Production Ready

---

For complete documentation, see:
- `/tests/gas/README.md`
- `/tests/gas/GAS_TESTING_IMPLEMENTATION_REPORT.md`
- `/config/gas_limits.yaml`
