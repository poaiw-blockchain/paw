# DEX Keeper Testing Guide

## Quick Start

### Run All DEX Keeper Tests
```bash
go test ./x/dex/keeper -v
```

### Run Tests with Coverage
```bash
go test ./x/dex/keeper -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out -o coverage.html
```

### View Coverage Report
```bash
go tool cover -func=coverage.out | grep total
```

---

## Running Specific Test Files

### Pool Tests Only (32 tests)
```bash
go test ./x/dex/keeper -run "Pool" -v
```

### Swap Tests Only (30 tests)
```bash
go test ./x/dex/keeper -run "Swap" -v
```

### Liquidity Tests Only (31 tests)
```bash
go test ./x/dex/keeper -run "Liquidity" -v
```

---

## Running Specific Test Categories

### Pool Creation Tests
```bash
go test ./x/dex/keeper -run "TestCreatePool" -v
```

### Pool Retrieval Tests
```bash
go test ./x/dex/keeper -run "TestGetPool|TestGetAllPools" -v
```

### Swap Success Cases
```bash
go test ./x/dex/keeper -run "TestSwap_Success|TestSwap_Reverse|TestSwap_Multiple" -v
```

### Swap Error Cases
```bash
go test ./x/dex/keeper -run "TestSwap.*Invalid|TestSwap.*Slippage|TestSwap.*Paused" -v
```

### Liquidity Addition Tests
```bash
go test ./x/dex/keeper -run "TestAddLiquidity" -v
```

### Liquidity Removal Tests
```bash
go test ./x/dex/keeper -run "TestRemoveLiquidity" -v
```

---

## Running Individual Tests

### Single Test
```bash
go test ./x/dex/keeper -run TestCreatePool_Success -v
```

### Multiple Related Tests
```bash
go test ./x/dex/keeper -run "TestCreatePool_Success|TestCreatePool_SameToken|TestCreatePool_Zero" -v
```

---

## Test Execution Options

### Run with Count (Detect Flaky Tests)
```bash
go test ./x/dex/keeper -run TestSwap -count=10
```

### Run with Timeout
```bash
go test ./x/dex/keeper -timeout 30s -v
```

### Run with Race Detector
```bash
go test ./x/dex/keeper -race -v
```

### Run with Short Flag (Skip Long Tests)
```bash
go test ./x/dex/keeper -short -v
```

---

## Coverage Analysis

### Generate Coverage by Function
```bash
go test ./x/dex/keeper -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Generate HTML Coverage Report
```bash
go test ./x/dex/keeper -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html in browser
```

### Coverage for Specific File
```bash
go test ./x/dex/keeper -coverprofile=coverage.out
go tool cover -func=coverage.out | grep keeper.go
```

### Coverage Summary
```bash
go test ./x/dex/keeper -cover
```

---

## Debugging Tests

### Run Single Test with Verbose Output
```bash
go test ./x/dex/keeper -run TestSwap_Success -v
```

### Run with Print Statements
```bash
go test ./x/dex/keeper -run TestSwap -v 2>&1 | tee test_output.log
```

### List All Tests Without Running
```bash
go test ./x/dex/keeper -list ".*"
```

---

## Performance Testing

### Benchmark Tests (if added)
```bash
go test ./x/dex/keeper -bench=. -benchmem
```

### CPU Profiling
```bash
go test ./x/dex/keeper -cpuprofile=cpu.prof -run TestSwap
go tool pprof cpu.prof
```

### Memory Profiling
```bash
go test ./x/dex/keeper -memprofile=mem.prof -run TestSwap
go tool pprof mem.prof
```

---

## CI/CD Integration

### Standard CI Command
```bash
go test ./x/dex/keeper -v -coverprofile=coverage.out -covermode=atomic
```

### Generate Coverage Report for CI
```bash
go test ./x/dex/keeper -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | grep total | awk '{print $3}'
```

### Fail Build if Coverage Below Threshold
```bash
#!/bin/bash
COVERAGE=$(go test ./x/dex/keeper -coverprofile=coverage.out -covermode=atomic | \
  go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
THRESHOLD=50

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
  echo "Coverage $COVERAGE% is below threshold $THRESHOLD%"
  exit 1
fi
```

---

## Test File Organization

### Pool Tests (pool_test.go)
- **Pool Creation**: 13 tests
- **Pool Retrieval**: 6 tests
- **Pool State Management**: 7 tests
- **Edge Cases**: 6 tests

### Swap Tests (swap_test.go)
- **Basic Swaps**: 10 tests
- **AMM Formula**: 5 tests
- **Advanced Features**: 8 tests
- **Slippage & Edge Cases**: 7 tests

### Liquidity Tests (liquidity_test.go)
- **Add Liquidity**: 11 tests
- **Remove Liquidity**: 10 tests
- **Liquidity Tracking**: 4 tests
- **Integration**: 6 tests

---

## Common Test Patterns

### Table-Driven Test Example
```go
func TestCalculateSwapAmount_Formula(t *testing.T) {
    k, _ := keepertest.DexKeeper(t)

    tests := []struct {
        name        string
        reserveIn   math.Int
        reserveOut  math.Int
        amountIn    math.Int
        expectedMin math.Int
        expectedMax math.Int
    }{
        {
            name:        "basic swap",
            reserveIn:   math.NewInt(1000000),
            reserveOut:  math.NewInt(2000000),
            amountIn:    math.NewInt(100000),
            expectedMin: math.NewInt(180000),
            expectedMax: math.NewInt(182000),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            amountOut := k.CalculateSwapAmount(tt.reserveIn, tt.reserveOut, tt.amountIn)
            require.True(t, amountOut.GTE(tt.expectedMin))
            require.True(t, amountOut.LTE(tt.expectedMax))
        })
    }
}
```

### Setup-Execute-Assert Pattern
```go
func TestCreatePool_Success(t *testing.T) {
    // Setup
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    // Execute
    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000), math.NewInt(2000000))

    // Assert
    require.NoError(t, err)
    require.Equal(t, uint64(1), poolId)

    pool := k.GetPool(ctx, poolId)
    require.NotNil(t, pool)
    require.Equal(t, "upaw", pool.TokenA)
}
```

### State Verification Pattern
```go
func TestSwap_ReservesUpdate(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    poolId := createTestPool(k, ctx)

    // Capture before state
    poolBefore := k.GetPool(ctx, poolId)

    // Execute operation
    amountOut, err := k.Swap(ctx, trader, poolId, "upaw", "uusdt",
        amountIn, minOut)
    require.NoError(t, err)

    // Verify after state
    poolAfter := k.GetPool(ctx, poolId)
    require.Equal(t, poolBefore.ReserveA.Add(amountIn), poolAfter.ReserveA)
}
```

---

## Troubleshooting

### Test Fails with "pool not found"
Make sure the test creates a pool before using it:
```go
poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
    math.NewInt(1000000), math.NewInt(2000000))
require.NoError(t, err)
```

### Test Fails with "invalid bech32"
Use `types.TestAddr()` instead of hardcoded strings:
```go
creator := types.TestAddr()  // ✅ Correct
creator := "paw1test"        // ❌ Wrong
```

### Test Fails with "module is paused"
Ensure module is not paused before operation:
```go
err := k.UnpauseModule(ctx)
require.NoError(t, err)
```

### Flaky Test Results
- Run with `-count=10` to detect non-determinism
- Check for time-based dependencies
- Ensure test isolation

---

## Best Practices

1. **Use Descriptive Test Names**
   - Good: `TestCreatePool_WhenPaused`
   - Bad: `TestCreatePool1`

2. **One Assertion Per Test**
   - Focus each test on a single behavior
   - Use subtests for related scenarios

3. **Test Both Success and Failure**
   - Happy path AND error cases
   - Edge cases and boundaries

4. **Keep Tests Fast**
   - All tests should run in < 1 second
   - Use mocks instead of real dependencies

5. **Make Tests Deterministic**
   - No random values
   - No time dependencies
   - Fixed test data

6. **Verify State Changes**
   - Check before and after state
   - Verify side effects (events, storage)

7. **Use testify/require**
   - Clear error messages
   - Test stops on first failure

---

## Coverage Goals

| Component | Current | Target |
|-----------|---------|--------|
| Overall Keeper | 53.4% | 70%+ |
| CreatePool | 92.3% | 95%+ |
| Swap | 88.5% | 95%+ |
| AddLiquidity | 89.3% | 95%+ |
| RemoveLiquidity | 91.7% | 95%+ |
| Helper Functions | 80-100% | 95%+ |

---

## Next Steps

1. Fix failing MEV protection tests
2. Add more integration tests
3. Improve edge case coverage
4. Add benchmark tests
5. Document test scenarios
