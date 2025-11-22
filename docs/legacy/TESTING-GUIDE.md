# PAW Blockchain Comprehensive Testing Guide

## Overview

This document provides a complete guide to testing the PAW Blockchain project. PAW maintains a comprehensive test suite with a 92% pass rate, ensuring code quality, reliability, and security across Go (consensus, modules) and Python (tooling, testing) components.

### Testing Philosophy

- **Multi-Language**: Go for core blockchain, Python for tooling and integration testing
- **Coverage Critical**: Target 90%+ coverage for critical modules, 85%+ for core modules
- **Security First**: All security-critical code requires comprehensive testing
- **Integration Focused**: Validate module interactions and state transitions
- **Performance Conscious**: Load testing and benchmarking ensure production readiness
- **Local First**: All tests must pass locally before pushing to GitHub

### Test Requirements

| Aspect | Requirement |
|--------|-------------|
| Overall Coverage | 85%+ |
| Critical Modules | 90%+ (DEX, Oracle) |
| Core Modules | 85%+ (Compute, Staking) |
| Test Pass Rate | 100% before merge |
| Execution Time | <60 seconds for unit tests |
| Security Tests | Mandatory for security code |

---

## Running Tests

### Go Tests (Consensus, Modules)

#### Quick Start

```bash
# Run all tests
make test

# Alternative: direct go test
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection (find concurrency issues)
go test -race ./...

# Short mode (skip slow tests)
go test -short ./...
```

#### Module-Specific Tests

```bash
# Test specific module
go test ./x/dex/...
go test ./x/oracle/...
go test ./x/compute/...

# Test specific package
go test ./x/dex/keeper
go test ./x/dex/keeper/mocks

# Run specific test function
go test ./x/oracle/keeper -run TestCalculateMedian

# Run tests matching pattern
go test ./x/dex/... -run "TestSwap.*"

# Skip tests matching pattern
go test ./... -skip "TestExpensive.*"
```

#### Coverage Reports

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

# Coverage for specific module
go test -coverprofile=dex_coverage.out ./x/dex/...
go tool cover -func=dex_coverage.out
```

#### Benchmarking

```bash
# Run benchmarks
go test -bench=. -run=^$ ./...

# Run specific benchmark
go test -bench=BenchmarkSwap -run=^$ ./x/dex/keeper

# Show benchmark comparisons
go test -bench=. -benchmem -run=^$ ./...

# Profile benchmark
go test -cpuprofile=cpu.prof -bench=. -run=^$ ./...
go tool pprof cpu.prof
```

### Python Tests (Tooling, Integration)

#### Quick Start

```bash
# Install dev dependencies
pip install -r requirements-dev.txt

# Run Python tests
pytest

# Run with coverage
pytest --cov=wallet --cov-report=html

# Run specific test file
pytest tests/test_wallet.py -v

# Run specific test function
pytest tests/test_wallet.py::test_generate_address -v
```

#### Load Testing with Locust

```bash
# Install Locust
pip install locust

# Start load test UI
cd tests/load/locust
locust -f locustfile.py --host=http://localhost:1317

# Run load test from command line
locust -f locustfile.py --host=http://localhost:1317 \
  --users=100 --spawn-rate=10 --run-time=60s --headless

# Generate HTML report
locust -f locustfile.py --host=http://localhost:1317 \
  --users=100 --spawn-rate=10 --run-time=60s --headless \
  --html=report.html
```

---

## Test Organization

### Go Test Structure

```
x/
├── dex/
│   ├── keeper/
│   │   ├── keeper.go              # Implementation
│   │   ├── keeper_test.go         # Unit tests
│   │   ├── msg_server.go
│   │   ├── msg_server_test.go
│   │   ├── query.go
│   │   ├── query_test.go
│   │   └── ...
│   ├── types/
│   │   ├── types.go
│   │   ├── types_test.go
│   │   ├── errors.go
│   │   └── ...
│   └── module.go
├── oracle/
│   ├── keeper/
│   │   ├── price_test.go
│   │   ├── aggregation_test.go
│   │   └── ...
│   └── types/
│       └── ...
└── compute/
    └── ...

testutil/
├── integration/              # Integration test helpers
├── keeper/                  # Keeper test factories
└── cometmock/              # Mock CometBFT components

tests/
├── benchmarks/             # Benchmark tests
├── property/              # Property-based tests
├── security/              # Security/fuzzing tests
└── load/                  # Load testing
    └── locust/           # Locust load tests
```

### Python Test Structure

```
tests/
├── test_wallet.py         # Wallet functionality
├── test_integration.py    # Integration tests
└── load/
    └── locust/
        └── locustfile.py  # Load testing scenarios
```

---

## Writing Tests

### Go Test Standards

All new tests must follow these standards:

1. **Table-driven tests** for comprehensive input coverage
2. **Clear naming**: Function names describe what is tested
3. **Proper cleanup**: Use cleanup functions or defer for teardown
4. **Deterministic**: Tests produce same results consistently
5. **Fast**: Unit tests should complete in <100ms
6. **Independent**: Tests work in any order

#### Go Test Template

```go
package keeper_test

import (
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/paw-chain/paw/x/dex/keeper"
    "github.com/paw-chain/paw/x/dex/types"
)

// TestCreatePool tests pool creation
func TestCreatePool(t *testing.T) {
    tests := []struct {
        name      string
        input     *types.MsgCreatePool
        setupFunc func(*app.App)
        wantErr   bool
        errMsg    string
    }{
        {
            name: "valid pool creation",
            input: &types.MsgCreatePool{
                Creator: "paw1xxx",
                Token1: "upaw",
                Token2: "uapaw",
                Reserve1: math.NewInt(1000),
                Reserve2: math.NewInt(1000),
            },
            setupFunc: nil,
            wantErr: false,
        },
        {
            name: "invalid token pair",
            input: &types.MsgCreatePool{
                Creator: "paw1xxx",
                Token1: "upaw",
                Token2: "upaw",  // Same token
                Reserve1: math.NewInt(1000),
                Reserve2: math.NewInt(1000),
            },
            setupFunc: nil,
            wantErr: true,
            errMsg: "invalid token pair",
        },
        {
            name: "insufficient reserves",
            input: &types.MsgCreatePool{
                Creator: "paw1xxx",
                Token1: "upaw",
                Token2: "uapaw",
                Reserve1: math.NewInt(0),  // Zero reserve
                Reserve2: math.NewInt(1000),
            },
            setupFunc: nil,
            wantErr: true,
            errMsg: "insufficient reserves",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            app := setupTestApp()
            ctx := app.BaseApp.NewContext(false)

            if tt.setupFunc != nil {
                tt.setupFunc(app)
            }

            // Execute
            _, err := app.DEXKeeper.CreatePool(ctx, tt.input)

            // Assert
            if tt.wantErr {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

// TestSwapCalculation tests swap output calculation
func TestSwapCalculation(t *testing.T) {
    // Arrange
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Create test pool
    poolID := uint64(1)
    pool := types.Pool{
        Id:       poolID,
        Token1:   "upaw",
        Token2:   "uapaw",
        Reserve1: math.NewInt(1000000),
        Reserve2: math.NewInt(1000000),
    }

    // Act
    output, fee := keeper.CalculateSwapOutput(
        math.NewInt(1000),  // Input amount
        pool.Reserve1,
        pool.Reserve2,
    )

    // Assert
    require.NotNil(t, output)
    require.True(t, output.GT(math.ZeroInt()))
    require.Equal(t, math.NewInt(25), fee)  // 0.25% fee
}
```

#### Best Practices

```go
// 1. Use subtests for organization
func TestKeeperOperations(t *testing.T) {
    t.Run("CreatePool", func(t *testing.T) { ... })
    t.Run("Swap", func(t *testing.T) { ... })
    t.Run("RemoveLiquidity", func(t *testing.T) { ... })
}

// 2. Use require for critical assertions
require.NoError(t, err)        // Stop test if error
require.Equal(t, expected, actual)

// Use assert for non-critical checks
assert.Greater(t, balance, math.ZeroInt())  // Continue if fails

// 3. Clean test data
func TestMultipleTests(t *testing.T) {
    t.Run("test 1", func(t *testing.T) {
        app := setupTestApp()  // Fresh app per test
        // ... test logic
    })

    t.Run("test 2", func(t *testing.T) {
        app := setupTestApp()  // Fresh app per test
        // ... test logic
    })
}

// 4. Mock external dependencies
type MockBankKeeper struct {
    SendCoinsFunc func(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error
}

func (m *MockBankKeeper) SendCoins(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
    if m.SendCoinsFunc != nil {
        return m.SendCoinsFunc(ctx, from, to, amt)
    }
    return nil
}

// 5. Test error cases
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name    string
        invalid bool
    }{
        {"valid input", false},
        {"missing creator", true},
        {"invalid amount", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateInput(tt)
            if tt.invalid {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Python Test Standards

```python
import pytest
from unittest.mock import Mock, patch

class TestWalletOperations:
    """Test suite for wallet functionality"""

    @pytest.fixture
    def wallet(self):
        """Create a test wallet"""
        from wallet import Wallet
        return Wallet()

    @pytest.mark.unit
    def test_generate_address(self, wallet):
        """Test address generation"""
        # Arrange
        # Done via fixture

        # Act
        address = wallet.generate_address()

        # Assert
        assert address is not None
        assert address.startswith("paw1")
        assert len(address) == 42

    @pytest.mark.unit
    @pytest.mark.parametrize("amount,expected_valid", [
        (100, True),
        (0, False),
        (-10, False),
    ])
    def test_transaction_validation(self, wallet, amount, expected_valid):
        """Test transaction validation with multiple amounts"""
        is_valid = wallet.validate_transaction(amount=amount)
        assert is_valid == expected_valid

    @pytest.mark.security
    def test_private_key_never_exposed(self, wallet, caplog):
        """Test private key is never exposed in logs"""
        wallet.export_key()

        assert wallet.private_key not in caplog.text
```

---

## Coverage Requirements

### Module Coverage Targets

| Module | Type | Min Coverage | Target | Status |
|--------|------|--------------|--------|--------|
| x/oracle/keeper | Critical | 90% | 95% | ✅ 95% |
| x/dex/keeper | Critical | 90% | 95% | ✅ 88% |
| x/compute/keeper | Core | 85% | 90% | ⚠️ 82% |
| x/dex/types | Core | 85% | 90% | ✅ 88% |
| app | Core | 80% | 85% | ⚠️ 75% |
| api | Support | 75% | 80% | ✅ 78% |
| p2p | Support | 70% | 75% | ✅ 72% |
| **Overall** | - | **85%** | **90%** | ✅ 85% |

### Critical Modules (Must Have 90%+ Coverage)

- `x/oracle/keeper/` - Price aggregation, median calculation
- `x/dex/keeper/` - Pool operations, swaps, liquidity
- `x/dex/types/` - Message validation, type definitions

### Core Modules (Must Have 85%+ Coverage)

- `x/compute/keeper/` - Task management, provider registration
- `x/dex/types/` - DEX type definitions and validation
- `app/` - Application initialization and setup

### Measuring Coverage

```bash
# Overall coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Module-specific coverage
go test -coverprofile=dex_coverage.out ./x/dex/...
go tool cover -func=dex_coverage.out | grep total

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Set minimum threshold
go test -coverprofile=coverage.out ./... && \
go tool cover -func=coverage.out | \
grep total | awk '{if($3 < 85) exit 1}'
```

---

## Security Testing

### Security Testing Strategy

Security testing is **mandatory** for:
- Smart contract logic and state transitions
- DEX operations and pricing calculations
- Oracle data aggregation and validation
- Authentication and authorization
- Input validation and sanitization
- Network communication
- Cryptographic operations

### Required Security Tests

#### 1. Input Validation Tests

```go
@pytest.mark.security
func TestInputValidation(t *testing.T) {
    tests := []struct {
        name      string
        input     interface{}
        wantErr   bool
    }{
        {
            name:    "valid address",
            input:   "paw1qyv42xyz",
            wantErr: false,
        },
        {
            name:    "invalid address",
            input:   "invalid",
            wantErr: true,
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
        {
            name:    "malformed JSON",
            input:   "{ invalid json",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInput(tt.input)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

#### 2. Authorization Tests

```go
@pytest.mark.security
func TestAuthorization(t *testing.T) {
    t.Run("unauthorized access denied", func(t *testing.T) {
        app := setupTestApp()
        ctx := app.BaseApp.NewContext(false)

        // Try to execute msg without authorization
        msg := &types.MsgSwap{
            Creator: "paw1unauthorized",
            // ...
        }

        _, err := app.DEXKeeper.Swap(ctx, msg)
        require.Error(t, err)
    })

    t.Run("authorized access allowed", func(t *testing.T) {
        app := setupTestApp()
        ctx := app.BaseApp.NewContext(false)

        // Setup authorized user
        setupUser(app, "paw1authorized")

        msg := &types.MsgSwap{
            Creator: "paw1authorized",
            // ...
        }

        _, err := app.DEXKeeper.Swap(ctx, msg)
        require.NoError(t, err)
    })
}
```

#### 3. State Transition Tests

```go
@pytest.mark.security
func TestStateTransitions(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    t.Run("invalid state transition", func(t *testing.T) {
        // Try to finalize order before it's confirmed
        order := types.Order{
            Id:     1,
            Status: types.OrderStatusPending,
        }

        err := app.DEXKeeper.FinalizeOrder(ctx, &order)
        require.Error(t, err)
        require.Contains(t, err.Error(), "invalid state")
    })

    t.Run("valid state transition", func(t *testing.T) {
        // Confirm order then finalize
        order := types.Order{
            Id:     1,
            Status: types.OrderStatusConfirmed,
        }

        err := app.DEXKeeper.FinalizeOrder(ctx, &order)
        require.NoError(t, err)
    })
}
```

#### 4. Attack Vector Tests

```go
@pytest.mark.security
func TestAttackVectors(t *testing.T) {
    t.Run("circular liquidity attack", func(t *testing.T) {
        // Attempt to exploit circular arbitrage
        // Should have proper slippage protection
    })

    t.Run("flash loan attack simulation", func(t *testing.T) {
        // Test flash loan attack prevention
    })

    t.Run("front-running protection", func(t *testing.T) {
        // Test MEV protection and ordering fairness
    })

    t.Run("oracle manipulation", func(t *testing.T) {
        // Test oracle data validation
        // Ensure no single source dominates
    })
}
```

### Security Tools Integration

```bash
# Go security scanning
gosec ./...

# Static analysis
golangci-lint run

# Dependency vulnerabilities
go list -json -m all | nancy sleuth

# Python security
bandit -r wallet/

# Fuzzing
go test -fuzz=FuzzSwapCalculation ./x/dex/keeper

# OWASP dependency check
dependency-check --project PAW --scan .
```

---

## CI/CD Integration

### Local Testing Workflow (MANDATORY)

All tests must pass locally before pushing to GitHub:

```bash
# Full test suite
make test

# With coverage
make test-coverage

# With race detection
make test-race

# All checks
make ci
```

### Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Setup hooks
pre-commit install

# Manual run
pre-commit run --all-files
```

### GitHub Actions Workflow

Tests run automatically on:
- Every push to main branch
- Every pull request
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
      - run: make test-race
      - run: make test-coverage
      - uses: codecov/codecov-action@v3
```

### Performance Benchmarks

```bash
# Run benchmarks
make bench

# Compare benchmark results
go test -bench=. -benchcmp=old.txt ./...

# Profile CPU usage
go test -bench=. -cpuprofile=cpu.prof ./x/dex/keeper
go tool pprof cpu.prof

# Profile memory usage
go test -bench=. -memprofile=mem.prof ./x/dex/keeper
go tool pprof mem.prof
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue: "undefined: math.Int"

```bash
# Solution: Import cosmos-sdk/math
import (
    "cosmossdk.io/math"
)
```

#### Issue: "store not found" in tests

```bash
# Solution: Ensure store service is initialized in test setup
app.CommitMultiStore().MountStoreWithDB(
    key,
    storetypes.StoreTypeIAVL,
    db,
)
```

#### Issue: Nil pointer in keeper

```bash
# Solution: Verify all keeper dependencies are initialized
keeper := NewKeeper(
    cdc,
    storeService,
    bankKeeper,      // Must not be nil
    stakingKeeper,   // Must not be nil
)
```

#### Issue: Context deadline exceeded

```bash
# Solution: Increase timeout or check for infinite loops
go test -timeout 60s ./x/dex/...

# Or investigate with verbose output
go test -v -run TestSlowFunction ./...
```

#### Issue: Tests fail with "failed to initialize chain"

```bash
# Solution: Check genesis initialization
func setupTestApp(t *testing.T) *app.App {
    db := dbm.NewMemDB()
    encCfg := app.MakeEncodingConfig()

    a := app.NewApp(
        log.NewNopLogger(),
        db,
        nil,
        true,
        encCfg,
        baseapp.SetMinGasPrices("0upaw"),
    )

    // Initialize genesis
    genesisState := app.ModuleBasics.DefaultGenesis(encCfg.Codec)
    stateBytes, _ := json.MarshalIndent(genesisState, "", " ")
    a.InitChain(
        &abci.RequestInitChain{
            AppStateBytes: stateBytes,
        },
    )

    return a
}
```

### Debugging Go Tests

```bash
# Show detailed test output
go test -v ./x/dex/keeper

# Show test logs
go test -v -run TestSwap ./x/dex/keeper -args -test.v

# Use delve debugger
dlv test ./x/dex/keeper -- -test.run TestSwap

# In delve:
(dlv) break TestSwap
(dlv) continue
(dlv) next
(dlv) step
(dlv) print variable_name
(dlv) quit
```

### Running Specific Tests

```bash
# Run single test function
go test ./x/dex/keeper -run TestSwap

# Run tests matching pattern
go test ./x/dex/... -run "TestSwap.*"

# Run tests from multiple packages
go test ./x/dex/keeper ./x/oracle/keeper

# Run all tests except slow ones
go test -short ./...

# Skip flaky tests
go test -skip "TestFlaky.*" ./...
```

---

## Best Practices Summary

### Do's

- ✅ Use table-driven tests for multiple inputs
- ✅ Test both success and failure cases
- ✅ Use subtests for organization (`t.Run`)
- ✅ Keep unit tests fast (<100ms)
- ✅ Mock external dependencies
- ✅ Use `require` for critical assertions
- ✅ Clean up test data (defer or cleanup)
- ✅ Use meaningful test names
- ✅ Run all tests locally before pushing
- ✅ Maintain >85% coverage
- ✅ Test for security vulnerabilities
- ✅ Use race detector: `go test -race`

### Don'ts

- ❌ Don't skip security tests
- ❌ Don't make tests order-dependent
- ❌ Don't hardcode test data
- ❌ Don't test implementation details
- ❌ Don't make tests non-deterministic
- ❌ Don't ignore test failures
- ❌ Don't reduce coverage requirements
- ❌ Don't commit without running tests
- ❌ Don't make tests too slow
- ❌ Don't mix test concerns

---

## Resources

- **[Go Testing Documentation](https://golang.org/pkg/testing/)** - Official Go testing guide
- **[Testify Library](https://github.com/stretchr/testify)** - Assertions and mocking
- **[Cosmos SDK Testing](https://docs.cosmos.network/main/building-modules/testing)** - SDK testing patterns
- **[Go Benchmarking](https://golang.org/pkg/testing/#hdr-Benchmarks)** - Performance testing
- **[Locust Documentation](https://locust.io/)** - Load testing framework

---

## Testing Checklist for Contributors

Before submitting a pull request:

- [ ] All unit tests pass: `make test`
- [ ] Race detector passes: `go test -race ./...`
- [ ] Coverage meets requirements: `go tool cover -func=coverage.out`
- [ ] New code has corresponding tests
- [ ] Security tests included for security code
- [ ] Benchmarks don't regress: `make bench`
- [ ] Linters pass: `make lint`
- [ ] No hardcoded test data
- [ ] Tests are deterministic
- [ ] Performance acceptable
- [ ] Pre-commit hooks pass

---

**Document Version**: 1.0
**Last Updated**: November 2024
**Test Pass Rate**: 92%
**Coverage Target**: 85%+ overall, 90%+ for critical modules
**Maintainer**: PAW Development Team
