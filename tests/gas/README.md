# Gas Metering Tests

## Overview

This directory contains comprehensive gas metering tests to prevent DoS attacks and ensure fair resource usage across all PAW blockchain modules. Gas metering is critical for:

- **DoS Prevention**: Preventing denial-of-service attacks via gas-expensive operations
- **Fair Resource Allocation**: Ensuring equitable gas pricing for all operations
- **Block Time Stability**: Preventing unbounded operations that could delay blocks
- **Economic Security**: Preventing gas price manipulation attacks
- **Performance Monitoring**: Detecting gas usage regressions

## Gas Limits by Module

### x/compute Module

| Operation | Min Gas | Max Gas | Notes |
|-----------|---------|---------|-------|
| RegisterProvider | 50,000 | 150,000 | Includes state storage |
| SubmitRequest | 80,000 | 200,000 | Scales with input size |
| SubmitResult | 100,000 | 300,000 | Scales with proof size |
| ZKVerification | 1,000,000 | 5,000,000 | Most expensive operation |
| LockEscrow | 30,000 | 80,000 | Bank transfer + state |
| ReleaseEscrow | 40,000 | 100,000 | Bank transfer + cleanup |
| RefundEscrow | 40,000 | 100,000 | Bank transfer + cleanup |
| UpdateProvider | 30,000 | 80,000 | State update only |
| DeactivateProvider | 25,000 | 60,000 | State cleanup |

### x/dex Module

| Operation | Min Gas | Max Gas | Notes |
|-----------|---------|---------|-------|
| CreatePool | 80,000 | 200,000 | Initial liquidity setup |
| Swap | 60,000 | 150,000 | Constant product calculation |
| AddLiquidity | 50,000 | 120,000 | LP token minting |
| RemoveLiquidity | 50,000 | 120,000 | LP token burning |
| ConstantProductCalc | 5,000 | 10,000 | Pure math operation |
| CircuitBreakerCheck | 10,000 | 30,000 | Price validation |
| UpdatePoolParams | 20,000 | 50,000 | Governance only |

### x/oracle Module

| Operation | Min Gas | Max Gas | Notes |
|-----------|---------|---------|-------|
| RegisterOracle | 40,000 | 100,000 | Validator registration |
| SubmitPrice | 30,000 | 100,000 | Single price submission |
| AggregateVotes | Variable | 3,000,000 | Scales with validator count |
| OutlierDetection | 200,000 | 1,000,000 | Statistical computation |
| TWAPCalculation | 50,000 | 200,000 | Time-weighted average |
| VolatilityCalc | 100,000 | 500,000 | Rolling standard deviation |
| SlashOracle | 80,000 | 200,000 | Slashing mechanism |

## Gas Scaling Formulas

### Oracle Vote Aggregation
```
Gas = BaseGas + (ValidatorCount * PerValidatorGas)
BaseGas = 100,000
PerValidatorGas = 20,000-30,000
```

For 100 validators: ~2,100,000 - 3,100,000 gas

### DEX Swap
```
Gas = BaseGas + (InputSize * PerByteGas)
BaseGas = 60,000
PerByteGas = 100
```

### Compute Request
```
Gas = BaseGas + (InputSize * PerByteGas) + (ProofSize * ProofByteGas)
BaseGas = 80,000
PerByteGas = 200
ProofByteGas = 500
```

## Testing Methodology

### 1. Unit Gas Tests

Test individual operations for gas consumption:

```go
func TestOperationGas(t *testing.T) {
    ctx = ctx.WithGasMeter(sdk.NewGasMeter(maxGas))

    // Execute operation
    err := keeper.Operation(ctx, params...)
    require.NoError(t, err)

    gasUsed := ctx.GasMeter().GasConsumed()

    // Assert within bounds
    require.Less(t, gasUsed, maxGas)
    require.Greater(t, gasUsed, minGas)
}
```

### 2. Scaling Tests

Test how gas scales with input parameters:

```go
func TestGasScaling(t *testing.T) {
    for _, size := range []int{100, 1000, 10000} {
        ctx = ctx.WithGasMeter(sdk.NewGasMeter(10000000))

        input := make([]byte, size)
        err := keeper.Process(ctx, input)

        gasPerByte := gasUsed / uint64(size)
        require.Less(t, gasPerByte, expectedMax)
    }
}
```

### 3. DoS Prevention Tests

Test resistance to denial-of-service attacks:

```go
func TestDosResistance(t *testing.T) {
    // Test 1: Unbounded loops should hit gas limit
    ctx = ctx.WithGasMeter(sdk.NewGasMeter(10000000))
    err := keeper.LargeOperation(ctx, 1000000)
    require.Error(t, err)
    require.Contains(t, err.Error(), "out of gas")

    // Test 2: Large inputs should be rejected or metered
    largeInput := make([]byte, 100*1024*1024) // 100MB
    err = keeper.Process(ctx, largeInput)
    require.Error(t, err)
}
```

### 4. Regression Detection

Track gas usage over time to detect regressions:

```go
func TestGasRegression(t *testing.T) {
    gasUsed := ctx.GasMeter().GasConsumed()

    // Compare against baseline
    baseline := uint64(100000)
    tolerance := uint64(10000) // 10% tolerance

    require.InDelta(t, baseline, gasUsed, float64(tolerance),
        "Gas usage changed significantly - possible regression")
}
```

### 5. Benchmarking

Measure gas consumption across iterations:

```go
func BenchmarkOperationGas(b *testing.B) {
    var totalGas uint64

    for i := 0; i < b.N; i++ {
        ctx = ctx.WithGasMeter(sdk.NewGasMeter(10000000))
        keeper.Operation(ctx, params...)
        totalGas += ctx.GasMeter().GasConsumed()
    }

    avgGas := totalGas / uint64(b.N)
    b.ReportMetric(float64(avgGas), "gas/op")
}
```

## DoS Attack Vectors Tested

### 1. Unbounded Loops
- **Attack**: Submit operation requiring iteration over unbounded state
- **Defense**: Gas metering on iterations, pagination limits
- **Test**: Verify gas limit is hit before timeout

### 2. Large Input Data
- **Attack**: Submit extremely large transaction data
- **Defense**: Input size limits, gas per byte charges
- **Test**: Verify rejection or appropriate gas charges

### 3. Expensive Computations
- **Attack**: Trigger computationally expensive operations
- **Defense**: Higher gas costs for complex operations
- **Test**: ZK verification, statistical analysis bounded

### 4. State Explosion
- **Attack**: Create excessive state entries
- **Defense**: Gas charges for state writes
- **Test**: Verify state operations properly metered

### 5. Nested Operations
- **Attack**: Deep nesting of operations
- **Defense**: Gas accumulation across nested calls
- **Test**: Verify gas compounds appropriately

### 6. Malformed Data
- **Attack**: Submit malformed data requiring expensive validation
- **Defense**: Early validation, fail-fast patterns
- **Test**: Verify validation is efficient

## Running Tests

### All Gas Tests
```bash
cd /home/decri/blockchain-projects/paw
go test -v ./tests/gas/...
```

### Specific Module
```bash
go test -v ./tests/gas -run TestCompute
go test -v ./tests/gas -run TestDEX
go test -v ./tests/gas -run TestOracle
```

### DoS Prevention Tests
```bash
go test -v ./tests/gas -run TestDoS
```

### Benchmarks
```bash
go test -bench=. -benchmem ./tests/gas
```

### With Coverage
```bash
go test -v -coverprofile=coverage.out ./tests/gas/...
go tool cover -html=coverage.out
```

## Gas Cost Analysis

### Cost Categories

1. **Storage Operations**
   - Write: 200-1000 gas per byte
   - Read: 10-50 gas per byte
   - Delete: 100-500 gas

2. **Computation**
   - Simple math: 5-20 gas
   - Cryptographic: 1000-100000 gas
   - Iteration: 100 gas per item

3. **External Calls**
   - Bank transfers: 50000+ gas
   - Staking operations: 30000+ gas
   - IBC packets: 100000+ gas

### Optimization Strategies

1. **Early Validation**: Fail fast before expensive operations
2. **Caching**: Cache expensive computations
3. **Pagination**: Limit iteration bounds
4. **Batching**: Batch operations to amortize costs
5. **Gas Metering**: Explicit gas charges for expensive ops

## Monitoring and Alerts

### Gas Usage Monitoring

Track gas usage in production:

```go
// Emit gas usage metrics
ctx.EventManager().EmitEvent(
    sdk.NewEvent(
        "gas_usage",
        sdk.NewAttribute("operation", "swap"),
        sdk.NewAttribute("gas_used", fmt.Sprintf("%d", gasUsed)),
    ),
)
```

### Alert Thresholds

Set alerts for:
- Operations exceeding expected gas limits
- Sudden increases in average gas usage
- Unusual gas consumption patterns
- Out of gas errors spike

## Best Practices

### 1. Always Set Gas Limits
```go
// Good
ctx = ctx.WithGasMeter(sdk.NewGasMeter(maxGas))

// Bad - unlimited gas
ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
```

### 2. Charge Gas for Expensive Operations
```go
// Charge upfront for expensive operation
ctx.GasMeter().ConsumeGas(100000, "expensive computation")

// Perform operation
result := expensiveComputation()
```

### 3. Validate Before Processing
```go
// Validate inputs first (cheap)
if !isValid(input) {
    return ErrInvalid
}

// Then perform expensive operation
return processInput(ctx, input)
```

### 4. Use Pagination
```go
// Limit iterations
maxIterations := 100
for i := 0; i < len(items) && i < maxIterations; i++ {
    process(items[i])
}
```

### 5. Monitor Gas Usage
```go
gasUsed := ctx.GasMeter().GasConsumed()
if gasUsed > warningThreshold {
    log.Warn("High gas usage detected", "gas", gasUsed)
}
```

## Gas Limit Configuration

Gas limits are defined in `config/gas_limits.yaml` and enforced in tests.

## Contributing

When adding new operations:

1. Add gas limit to configuration
2. Write gas metering tests
3. Add DoS prevention tests
4. Update this README
5. Run benchmarks to establish baseline

## References

- [Cosmos SDK Gas Metering](https://docs.cosmos.network/main/basics/gas-fees)
- [Gas Cost Analysis](https://github.com/cosmos/cosmos-sdk/blob/main/docs/basics/gas-fees.md)
- [DoS Prevention Best Practices](https://github.com/cosmos/cosmos-sdk/blob/main/docs/building-modules/security.md)

## Maintenance

This test suite should be run:
- On every commit (CI/CD)
- Before releases
- After performance optimizations
- When adding new operations

Last updated: 2025-11-25
