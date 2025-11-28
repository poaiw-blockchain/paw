# Fuzzing Test Suite for PAW Blockchain

This directory contains fuzzing tests for the PAW blockchain to discover edge cases, crashes, and security vulnerabilities through automated input generation.

## Overview

Fuzzing (or fuzz testing) is an automated testing technique that provides invalid, unexpected, or random data as inputs to a program. The goal is to find bugs, crashes, memory leaks, and security vulnerabilities that might not be caught by traditional testing methods.

## Fuzzing Tools

### 1. Go-Fuzz

Go-fuzz is a coverage-guided fuzzing tool for Go programs.

**Installation:**

```bash
go install github.com/dvyukov/go-fuzz/go-fuzz@latest
go install github.com/dvyukov/go-fuzz/go-fuzz-build@latest
```

### 2. Go Native Fuzzing (Go 1.18+)

Go 1.18+ includes native fuzzing support.

**No additional installation required** - built into Go toolchain.

## Fuzzing Targets

### DEX Module

**Message Types:**

- `MsgCreatePool` - Pool creation with random token pairs and amounts
- `MsgSwap` - Swap operations with random amounts and slippage
- `MsgAddLiquidity` - Liquidity provision with random ratios
- `MsgRemoveLiquidity` - Liquidity removal with random amounts

**Test Areas:**

- Integer overflow/underflow in calculations
- Division by zero in AMM formulas
- Invalid token denominations
- Extreme amount values
- Pool state corruption

### Oracle Module

**Message Types:**

- `MsgSubmitPrice` - Price submissions with random values
- `MsgRegisterOracle` - Oracle registration with random addresses

**Test Areas:**

- Price manipulation with extreme values
- Decimal precision issues
- Asset name validation
- Aggregation logic with outliers
- Timestamp manipulation

### Compute Module

**Message Types:**

- `MsgRegisterProvider` - Provider registration with random endpoints
- `MsgSubmitJob` - Job submission with random parameters

**Test Areas:**

- Endpoint URL parsing and validation
- Resource allocation with extreme values
- Job state transitions
- Payment calculations

### Protobuf Messages

**Targets:**

- All protobuf message unmarshaling
- Field validation with random data
- Nested message structures
- Repeated fields with large counts

## Fuzzing Test Files

### 1. DEX Fuzzing (`dex_fuzz_test.go`)

```go
package security_test

import (
    "testing"
    "github.com/paw-chain/paw/x/dex/types"
)

func FuzzCreatePoolMsg(f *testing.F) {
    // Seed corpus
    f.Add("paw1creator", "upaw", "uusdt", int64(1000000), int64(2000000))

    f.Fuzz(func(t *testing.T, creator string, tokenA string, tokenB string, amountA int64, amountB int64) {
        msg := &types.MsgCreatePool{
            Creator: creator,
            TokenA:  tokenA,
            TokenB:  tokenB,
            AmountA: sdk.NewInt(amountA),
            AmountB: sdk.NewInt(amountB),
        }

        // Should not panic
        _ = msg.ValidateBasic()
    })
}

func FuzzSwapMsg(f *testing.F) {
    f.Add("paw1trader", uint64(1), "upaw", int64(100000), int64(90000))

    f.Fuzz(func(t *testing.T, trader string, poolId uint64, tokenIn string, amountIn int64, minOut int64) {
        msg := &types.MsgSwap{
            Trader:       trader,
            PoolId:       poolId,
            TokenIn:      tokenIn,
            AmountIn:     sdk.NewInt(amountIn),
            MinAmountOut: sdk.NewInt(minOut),
        }

        // Should not panic
        _ = msg.ValidateBasic()
    })
}
```

### 2. Oracle Fuzzing (`oracle_fuzz_test.go`)

```go
func FuzzSubmitPriceMsg(f *testing.F) {
    f.Add("paw1oracle", "BTC/USD", "50000.00")

    f.Fuzz(func(t *testing.T, oracle string, asset string, priceStr string) {
        price, err := sdk.NewDecFromStr(priceStr)
        if err != nil {
            return // Invalid decimal, skip
        }

        msg := &types.MsgSubmitPrice{
            Oracle: oracle,
            Asset:  asset,
            Price:  price,
        }

        _ = msg.ValidateBasic()
    })
}
```

### 3. Protobuf Fuzzing (`proto_fuzz_test.go`)

```go
func FuzzProtobufUnmarshal(f *testing.F) {
    // Seed with valid protobuf
    msg := &types.MsgCreatePool{
        Creator: "paw1test",
        TokenA:  "upaw",
        TokenB:  "uusdt",
    }
    seed, _ := msg.Marshal()
    f.Add(seed)

    f.Fuzz(func(t *testing.T, data []byte) {
        msg := &types.MsgCreatePool{}
        _ = msg.Unmarshal(data) // Should not panic
    })
}
```

## Running Fuzzing Tests

### Native Go Fuzzing

```bash
# Fuzz a specific test
go test -fuzz=FuzzCreatePoolMsg -fuzztime=30s ./tests/security/fuzzing/

# Fuzz with more iterations
go test -fuzz=FuzzCreatePoolMsg -fuzztime=5m ./tests/security/fuzzing/

# Fuzz all tests
go test -fuzz=. -fuzztime=1m ./tests/security/fuzzing/
```

### Go-Fuzz (Legacy)

```bash
# Build fuzz package
go-fuzz-build -o dex-fuzz.zip github.com/paw-chain/paw/tests/security/fuzzing

# Run fuzzer
go-fuzz -bin=dex-fuzz.zip -workdir=fuzz-workdir/dex

# View results
ls fuzz-workdir/dex/crashers/
ls fuzz-workdir/dex/suppressions/
```

## Corpus Management

### Seed Corpus

Create a `testdata/fuzz/` directory with seed inputs:

```
testdata/fuzz/FuzzCreatePoolMsg/
├── valid_pool.txt
├── max_amounts.txt
├── min_amounts.txt
└── special_chars.txt
```

### Minimizing Crashes

When a crash is found:

```bash
# Minimize the crashing input
go test -fuzz=FuzzCreatePoolMsg -run=FuzzCreatePoolMsg/crash-hash

# The minimized input will be saved to testdata/fuzz/
```

## Continuous Fuzzing

### OSS-Fuzz Integration

For continuous fuzzing, integrate with OSS-Fuzz:

1. Create `oss-fuzz/` directory in repository
2. Add `build.sh` script for building fuzzers
3. Add `project.yaml` configuration
4. Submit to OSS-Fuzz project

### CI/CD Integration

Add to  Actions:

```yaml
name: Fuzzing
on: [push, pull_request]

jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run Fuzz Tests
        run: |
          go test -fuzz=. -fuzztime=30s ./tests/security/fuzzing/
```

## Analyzing Results

### Coverage Reports

Generate coverage reports from fuzzing:

```bash
go test -fuzz=FuzzCreatePoolMsg -fuzztime=1m -coverprofile=fuzz.cov ./tests/security/fuzzing/
go tool cover -html=fuzz.cov -o fuzz-coverage.html
```

### Crash Analysis

When a crash is found:

1. Check `testdata/fuzz/FuzzName/` for the input
2. Reproduce with: `go test -run=FuzzName/crash-hash`
3. Debug with delve: `dlv test -- -test.run=FuzzName/crash-hash`
4. Fix the bug
5. Add test case to prevent regression

## Best Practices

1. **Start Small**: Fuzz individual functions before complex workflows
2. **Seed Corpus**: Provide valid inputs as starting points
3. **Monitor Resources**: Fuzzing is CPU and memory intensive
4. **Minimize Crashes**: Always minimize crashing inputs for easier debugging
5. **Continuous Fuzzing**: Run fuzzing continuously in CI/CD
6. **Coverage Tracking**: Monitor code coverage improvements
7. **Timeout Handling**: Set appropriate timeouts for complex operations
8. **Memory Leaks**: Use `-race` detector alongside fuzzing

## Common Vulnerabilities Found

Fuzzing typically discovers:

- **Integer Overflows**: In amount calculations
- **Division by Zero**: In AMM formulas
- **Nil Pointer Dereferences**: In validation logic
- **Infinite Loops**: In parsing logic
- **Stack Overflows**: In recursive functions
- **Memory Leaks**: In long-running operations
- **Panics**: In edge case handling
- **Buffer Overflows**: In fixed-size buffers

## Performance Tuning

### Parallel Fuzzing

```bash
# Run 4 parallel fuzzing instances
for i in {1..4}; do
  go test -fuzz=FuzzCreatePoolMsg -fuzztime=5m -parallel=1 &
done
wait
```

### Memory Limits

```bash
# Limit memory usage
go test -fuzz=FuzzCreatePoolMsg -fuzztime=5m -memprofile=mem.prof

# Analyze memory usage
go tool pprof mem.prof
```

## Security Considerations

1. **Sanitize Inputs**: Never trust fuzzer-generated inputs in production
2. **Isolation**: Run fuzzing in isolated environments
3. **Resource Limits**: Set CPU, memory, and time limits
4. **Disclosure**: Report found vulnerabilities responsibly
5. **Regression Tests**: Add crash cases to regression test suite

## Additional Resources

- [Go Fuzzing Documentation](https://go.dev/security/fuzz/)
- [Go-Fuzz ](https://github.com/dvyukov/go-fuzz)
- [OSS-Fuzz](https://github.com/google/oss-fuzz)
- [Fuzzing Best Practices](https://github.com/google/fuzzing)

## Support

For questions or issues with fuzzing:

- Open an issue in the PAW repository
- Contact the security team
- Review existing fuzz test examples

## Future Enhancements

- [ ] Structured fuzzing with grammar-based inputs
- [ ] Differential fuzzing against other DEX implementations
- [ ] State-aware fuzzing for complex state machines
- [ ] Custom mutators for blockchain-specific types
- [ ] Integration with symbolic execution tools
