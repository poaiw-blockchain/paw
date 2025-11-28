# Fuzzing Tests

This directory contains fuzzing tests for the PAW blockchain that use Go's native fuzzing support (Go 1.18+).

## Overview

Fuzzing is an automated testing technique that provides random/malformed inputs to find edge cases, panics, and security vulnerabilities. Our fuzzing tests cover:

1. **SafeMath Operations** (`safemath_fuzz.go`) - Tests all math operations for overflow/underflow
2. **DEX Calculations** (`dex_fuzz.go`) - Fuzz swap and liquidity calculations
3. **Protobuf Deserialization** (`proto_fuzz.go`) - Fuzz protobuf message parsing

## Running Fuzz Tests

### Run all fuzz tests (short duration for CI)
```bash
cd /home/decri/blockchain-projects/paw/tests/fuzz
go test -fuzz=. -fuzztime=30s
```

### Run specific fuzz test
```bash
# Fuzz SafeMath addition
go test -fuzz=FuzzSafeMathAdd -fuzztime=1m

# Fuzz DEX swap calculations
go test -fuzz=FuzzSwapCalculation -fuzztime=5m

# Fuzz protobuf deserialization
go test -fuzz=FuzzDexPoolProto -fuzztime=2m
```

### Long-running fuzz campaign (for thorough testing)
```bash
# Run for 1 hour
go test -fuzz=FuzzSwapCalculation -fuzztime=1h

# Run indefinitely until crash found
go test -fuzz=FuzzSafeMathAdd -fuzztime=0
```

## Understanding Results

### Success Output
```
fuzz: elapsed: 0s, gathering baseline coverage: 0/192 completed
fuzz: elapsed: 3s, execs: 325017 (108336/sec), new interesting: 11 (total: 202)
fuzz: elapsed: 6s, execs: 680218 (118402/sec), new interesting: 12 (total: 203)
PASS
```

### Failure Output
When a failure is found, Go creates a test file in `testdata/fuzz/FuzzTestName/`:
```
--- FAIL: FuzzSwapCalculation (3.14s)
    --- FAIL: FuzzSwapCalculation (0.00s)
        dex_fuzz.go:89: VIOLATION: k decreased - k_before=1000000, k_after=999999

    Failing input written to testdata/fuzz/FuzzSwapCalculation/a1b2c3d4e5f6...
```

To reproduce the failure:
```bash
go test -run=FuzzSwapCalculation
```

## Invariants Tested

### SafeMath Invariants
- No panics on any integer operations
- Results match `big.Int` calculations (ground truth)
- Double negation equals original value
- Absolute value is always non-negative

### DEX Invariants
- **Constant Product**: `k_after ≥ k_before` (due to fees)
- **Output Bounds**: `amountOut < reserveOut` (can't drain pool)
- **Price Impact**: Larger swaps have worse prices
- **LP Shares**: Total shares = √(reserveA × reserveB)
- **Proportionality**: Liquidity ratios maintained

### Protobuf Invariants
- No panics on malformed input
- Deterministic serialization (roundtrip equality)
- All numeric fields non-negative (where applicable)
- Valid enum values
- Logical constraints (e.g., escrowed ≤ max_payment)

## Corpus Management

Fuzzing builds a corpus of interesting inputs over time. These are stored in:
```
testdata/fuzz/FuzzTestName/
```

### Seed Corpus
Each fuzz function includes seed values that represent:
- Normal cases
- Edge cases (max/min values)
- Known interesting patterns
- Historical bugs

### Sharing Corpus
To share corpus between developers:
```bash
# Export corpus
tar czf fuzz-corpus.tar.gz testdata/

# Import corpus
tar xzf fuzz-corpus.tar.gz
```

## Integration with CI

In CI pipelines, run fuzzing with short duration:
```bash
# In hub/workflows/test.yml or similar
- name: Fuzz tests
  run: |
    cd tests/fuzz
    go test -fuzz=. -fuzztime=30s
```

For dedicated fuzzing infrastructure (OSS-Fuzz, etc.):
```bash
# Run continuously
go test -fuzz=. -fuzztime=0
```

## Tips for Writing Fuzz Tests

1. **Skip Invalid Inputs Early**: Use `return` to skip invalid inputs
   ```go
   if denominator == 0 {
       return // Skip division by zero
   }
   ```

2. **Test Invariants, Not Exact Values**: Focus on properties that should always hold
   ```go
   if result < 0 {
       t.Errorf("VIOLATION: negative result")
   }
   ```

3. **Use Ground Truth**: Compare against known-correct implementations
   ```go
   expected := big.NewInt(a).Add(big.NewInt(b))
   if !result.Equal(expected) {
       t.Errorf("incorrect result")
   }
   ```

4. **Seed Well**: Include edge cases in seed corpus
   ```go
   seeds := []int64{0, 1, -1, MaxInt64, MinInt64}
   ```

## Performance Notes

- Fuzzing is CPU-intensive; use multiple cores: `go test -fuzz=. -parallel=8`
- Corpus grows over time; clean periodically: `rm -rf testdata/fuzz/*/`
- New interesting inputs indicate good coverage expansion
- Execution speed varies by test complexity (10k-1M execs/sec typical)

## References

- [Go Fuzzing Tutorial](https://go.dev/doc/fuzz/)
- [Fuzzing Best Practices](https://github.com/google/oss-fuzz/blob/master/docs/ideal_integration.md)
- PAW Security Documentation: `/x/*/SECURITY_*.md`
