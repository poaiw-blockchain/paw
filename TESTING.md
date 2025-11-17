# PAW Blockchain Testing Guide

## Overview

PAW maintains a comprehensive test suite with **92% test pass rate**, ensuring high code quality and reliability. This document covers testing strategies, how to run tests, and how to contribute to the test suite.

## Test Statistics

| Metric | Value |
|--------|-------|
| **Test Pass Rate** | 92% |
| **Total Test Suites** | 25+ |
| **Total Test Cases** | 300+ |
| **Code Coverage** | ~85% |
| **Test Execution Time** | ~45 seconds |

## Test Categories

### 1. Unit Tests

Unit tests verify individual functions and methods in isolation.

**Coverage Areas**:
- Oracle keeper functions (price aggregation, median calculation)
- DEX keeper functions (pool operations, swaps, liquidity)
- Compute keeper functions (task management, provider registration)
- Type validation and encoding/decoding
- Helper utilities and converters

**Example Test**:
```go
func TestCalculateMedian(t *testing.T) {
    prices := []math.Int{
        math.NewInt(100),
        math.NewInt(102),
        math.NewInt(99),
    }

    median := CalculateMedian(prices)
    require.Equal(t, math.NewInt(100), median)
}
```

### 2. Integration Tests

Integration tests verify module interactions and state transitions.

**Coverage Areas**:
- Module initialization and genesis
- Message handling and routing
- Inter-module dependencies
- State persistence and queries
- Event emission

**Example Test**:
```go
func TestDEXSwapIntegration(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Create pool
    createPoolMsg := &dextypes.MsgCreatePool{...}
    _, err := app.DEXKeeper.CreatePool(ctx, createPoolMsg)
    require.NoError(t, err)

    // Execute swap
    swapMsg := &dextypes.MsgSwap{...}
    _, err = app.DEXKeeper.Swap(ctx, swapMsg)
    require.NoError(t, err)

    // Verify state changes
    pool, found := app.DEXKeeper.GetPool(ctx, poolID)
    require.True(t, found)
    require.Equal(t, expectedReserves, pool.Reserves)
}
```

### 3. Security Tests

Security tests verify protection mechanisms and attack resistance.

**Coverage Areas**:
- Input validation
- Authorization checks
- Circuit breaker triggers
- Slashing mechanisms
- Rate limiting
- MEV protection

**Example Test**:
```go
func TestCircuitBreakerTriggersOnLargeSwap(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Attempt swap that exceeds threshold
    largeSwap := &dextypes.MsgSwap{
        TokenIn: sdk.NewCoin("uapaw", math.NewInt(1000000000)),
        // ... large amount that should trigger breaker
    }

    _, err := app.DEXKeeper.Swap(ctx, largeSwap)
    require.Error(t, err)
    require.Contains(t, err.Error(), "circuit breaker triggered")
}
```

### 4. End-to-End (E2E) Tests

E2E tests verify complete user workflows across the entire system.

**Coverage Areas**:
- Multi-step transactions
- Wallet operations
- API endpoints
- CLI commands
- Complete user journeys

## Running Tests

### Prerequisites

```bash
# Install Go 1.23.1+
go version

# Install dependencies
go mod download

# Install testing tools
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install gotest.tools/gotestsum@latest
```

### Run All Tests

```bash
# Standard test execution
make test

# Alternative: direct go test
go test ./...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...
```

### Run Specific Module Tests

```bash
# DEX module tests
go test ./x/dex/...

# Oracle module tests
go test ./x/oracle/...

# Compute module tests
go test ./x/compute/...

# Keeper tests only
go test ./x/dex/keeper/...
```

### Run Individual Test Functions

```bash
# Run specific test by name
go test ./x/oracle/keeper -run TestCalculateMedian

# Run tests matching pattern
go test ./x/dex/... -run "TestSwap.*"

# Run with short mode (skip slow tests)
go test -short ./...
```

### Generate Coverage Reports

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open in browser (Windows)
start coverage.html

# Open in browser (Linux/Mac)
open coverage.html
```

### Coverage by Module

```bash
# DEX module coverage
go test -coverprofile=dex_coverage.out ./x/dex/...
go tool cover -func=dex_coverage.out

# Oracle module coverage
go test -coverprofile=oracle_coverage.out ./x/oracle/...
go tool cover -func=oracle_coverage.out

# Compute module coverage
go test -coverprofile=compute_coverage.out ./x/compute/...
go tool cover -func=compute_coverage.out
```

## Test Structure

### Standard Test File Organization

```
x/dex/
├── keeper/
│   ├── keeper.go              # Implementation
│   ├── keeper_test.go         # Unit tests
│   ├── circuit_breaker.go
│   ├── circuit_breaker_test.go
│   ├── msg_server.go
│   └── msg_server_test.go
└── types/
    ├── types.go
    └── types_test.go
```

### Test Naming Conventions

```go
// Function being tested: CreatePool
func TestCreatePool(t *testing.T) { ... }

// Table-driven tests
func TestCreatePool(t *testing.T) {
    tests := []struct {
        name    string
        input   *types.MsgCreatePool
        wantErr bool
    }{
        {
            name: "valid pool creation",
            input: &types.MsgCreatePool{...},
            wantErr: false,
        },
        {
            name: "invalid token pair",
            input: &types.MsgCreatePool{...},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### Test Setup Helpers

```go
// Common test setup
func setupTestApp() *app.App {
    db := dbm.NewMemDB()
    encCfg := app.MakeEncodingConfig()

    return app.NewApp(
        log.NewNopLogger(),
        db,
        nil,
        true,
        encCfg,
        baseapp.SetMinGasPrices("0upaw"),
    )
}

// Setup with genesis state
func setupWithGenesis(t *testing.T, genState types.GenesisState) (sdk.Context, keeper.Keeper) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    app.DEXKeeper.InitGenesis(ctx, genState)

    return ctx, app.DEXKeeper
}
```

## Test Best Practices

### 1. Table-Driven Tests

Use table-driven tests for comprehensive input coverage:

```go
func TestValidateBasic(t *testing.T) {
    tests := []struct {
        name    string
        msg     *types.MsgSwap
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid swap",
            msg: &types.MsgSwap{
                Trader: "paw1xxx",
                TokenIn: sdk.NewCoin("uapaw", math.NewInt(1000)),
                TokenOutMin: sdk.NewCoin("upaw", math.NewInt(900)),
            },
            wantErr: false,
        },
        {
            name: "empty trader",
            msg: &types.MsgSwap{
                Trader: "",
            },
            wantErr: true,
            errMsg: "empty trader address",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.msg.ValidateBasic()
            if tt.wantErr {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### 2. Use Assertions

Prefer `require` over `assert` for critical checks:

```go
// Good - test stops on failure
require.NoError(t, err)
require.Equal(t, expected, actual)

// Use assert for non-critical checks
assert.Greater(t, balance, math.ZeroInt())
```

### 3. Clean Test Data

Ensure tests don't interfere with each other:

```go
func TestMultipleTests(t *testing.T) {
    t.Run("test 1", func(t *testing.T) {
        app := setupTestApp() // Fresh app per test
        // ... test logic
    })

    t.Run("test 2", func(t *testing.T) {
        app := setupTestApp() // Fresh app per test
        // ... test logic
    })
}
```

### 4. Test Error Cases

Always test both success and failure paths:

```go
func TestSwap(t *testing.T) {
    // Test success case
    t.Run("successful swap", func(t *testing.T) {
        // ... setup and execute
        require.NoError(t, err)
    })

    // Test failure cases
    t.Run("insufficient balance", func(t *testing.T) {
        // ... test with insufficient funds
        require.Error(t, err)
    })

    t.Run("pool not found", func(t *testing.T) {
        // ... test with invalid pool
        require.Error(t, err)
    })
}
```

### 5. Mock External Dependencies

Use interfaces and mocks for external dependencies:

```go
// Good - use mock bank keeper
type MockBankKeeper struct {
    SendCoinsFunc func(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error
}

func (m *MockBankKeeper) SendCoins(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
    if m.SendCoinsFunc != nil {
        return m.SendCoinsFunc(ctx, from, to, amt)
    }
    return nil
}
```

## Continuous Integration

### GitHub Actions Workflow

Tests run automatically on:
- Every pull request
- Every push to main
- Nightly builds

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23.1'
      - run: make test
      - run: go test -race ./...
      - run: go test -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
```

## Test Coverage Goals

### Current Coverage

| Module | Coverage | Status |
|--------|----------|--------|
| x/oracle | 95% | ✅ Excellent |
| x/dex | 88% | ✅ Good |
| x/compute | 82% | ⚠️ Needs improvement |
| app | 75% | ⚠️ Needs improvement |
| Overall | 85% | ✅ Good |

### Target Coverage

- **Critical modules** (oracle, dex): 90%+
- **Core modules** (compute, staking): 85%+
- **Utility code**: 70%+
- **Overall target**: 90%+

## Writing New Tests

### 1. Create Test File

```bash
# For keeper functions
touch x/mymodule/keeper/keeper_test.go

# For types
touch x/mymodule/types/types_test.go
```

### 2. Write Test Function

```go
package keeper_test

import (
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/paw-chain/paw/x/mymodule/keeper"
)

func TestMyFunction(t *testing.T) {
    // Arrange
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Act
    result, err := app.MyModuleKeeper.MyFunction(ctx, input)

    // Assert
    require.NoError(t, err)
    require.Equal(t, expected, result)
}
```

### 3. Run Your Test

```bash
# Run the specific test
go test ./x/mymodule/keeper -run TestMyFunction -v

# Verify coverage
go test -cover ./x/mymodule/keeper
```

## Load Testing

### Locust Setup

```bash
# Install Locust
pip install locust

# Run load tests
cd tests/load
locust -f locustfile.py --host=http://localhost:1317
```

### Performance Targets

| Operation | Target TPS | Current |
|-----------|-----------|---------|
| Bank transfers | 1000+ | ✅ 1200 |
| DEX swaps | 500+ | ✅ 600 |
| Oracle submissions | 100+ | ✅ 150 |
| Compute requests | 50+ | ✅ 75 |

## Security Testing

### Fuzzing

```bash
# Install go-fuzz
go install github.com/dvyukov/go-fuzz/go-fuzz@latest
go install github.com/dvyukov/go-fuzz/go-fuzz-build@latest

# Run fuzzing tests
cd tests/security/fuzzing
go-fuzz-build
go-fuzz
```

### Static Analysis

```bash
# Run security scanner
gosec ./...

# Run static analysis
golangci-lint run

# Check for vulnerabilities
go list -json -m all | nancy sleuth
```

## Debugging Tests

### Verbose Output

```bash
# Show detailed test output
go test -v ./x/dex/keeper

# Show logs during tests
go test -v ./x/dex/keeper -args -test.v
```

### Debug Single Test

```bash
# Use delve debugger
dlv test ./x/dex/keeper -- -test.run TestSwap

# In delve:
(dlv) break TestSwap
(dlv) continue
```

### Common Issues

**Issue**: Tests fail with "store not found"
```bash
# Solution: Ensure store service is initialized
app.CommitMultiStore().MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
```

**Issue**: Nil pointer in keeper
```bash
# Solution: Verify all keeper dependencies are initialized
keeper := NewKeeper(cdc, storeService, bankKeeper, stakingKeeper)
```

**Issue**: Context deadline exceeded
```bash
# Solution: Increase timeout or use -timeout flag
go test -timeout 30s ./x/dex/...
```

## Contributing Tests

### Before Submitting PR

1. **Run full test suite**:
   ```bash
   make test
   ```

2. **Check coverage**:
   ```bash
   go test -cover ./...
   ```

3. **Run linters**:
   ```bash
   make lint
   ```

4. **Format code**:
   ```bash
   make fmt
   ```

### Test Requirements

- All new functions must have tests
- Target 85%+ coverage for new code
- Include both success and error cases
- Add comments explaining complex test logic
- Use descriptive test names
- Follow existing test patterns

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Cosmos SDK Testing Guide](https://docs.cosmos.network/main/building-modules/testing)
- [PAW Test Examples](./tests/README.md)

## Test Metrics Dashboard

View real-time test metrics:
- [GitHub Actions](https://github.com/decristofaroj/paw/actions)
- [Codecov Dashboard](https://codecov.io/gh/decristofaroj/paw)
- [SonarCloud Quality](https://sonarcloud.io/project/overview?id=decristofaroj_paw)

---

**Document Version**: 1.0
**Last Updated**: November 2025
**Test Pass Rate**: 92%
**Maintainer**: PAW Development Team
