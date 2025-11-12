# PAW Blockchain Testing Infrastructure

This document describes the comprehensive testing framework for the PAW blockchain.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Writing Tests](#writing-tests)
- [CI/CD Integration](#cicd-integration)

## Overview

PAW uses a multi-layered testing approach:

1. **Unit Tests**: Test individual functions and keeper methods
2. **Integration Tests**: Test module interactions and app initialization
3. **End-to-End Tests**: Test complete user workflows across modules
4. **Simulation Tests**: Test chain behavior under various conditions

## Test Structure

```
paw/
├── testutil/
│   └── keeper/              # Test helpers for creating keepers
│       ├── dex.go          # DEX keeper test utilities
│       ├── compute.go      # Compute keeper test utilities
│       └── oracle.go       # Oracle keeper test utilities
├── x/
│   ├── dex/
│   │   ├── keeper/
│   │   │   └── keeper_test.go  # DEX keeper unit tests
│   │   └── types/
│   │       └── msg_test.go     # DEX message validation tests
│   ├── compute/
│   │   └── keeper/
│   │       └── keeper_test.go  # Compute keeper unit tests
│   └── oracle/
│       └── keeper/
│           └── keeper_test.go  # Oracle keeper unit tests
├── app/
│   └── app_test.go         # App integration tests
└── tests/
    └── e2e/
        └── e2e_test.go     # End-to-end workflow tests
```

## Running Tests

### All Tests

Run all tests with race detection and coverage:

```bash
make test
```

### Unit Tests Only

Run only unit tests (faster for development):

```bash
make test-unit
```

### Integration Tests

Run integration tests for app and cross-module testing:

```bash
make test-integration
```

### Keeper Tests

Run tests for all keeper modules:

```bash
make test-keeper
```

### Types Tests

Run message validation and type tests:

```bash
make test-types
```

### Coverage Report

Generate HTML coverage report:

```bash
make test-coverage
```

This creates `coverage.html` which you can open in a browser.

### Specific Module Tests

Test individual modules:

```bash
# DEX module only
go test -v ./x/dex/...

# Compute module only
go test -v ./x/compute/...

# Oracle module only
go test -v ./x/oracle/...
```

## Test Coverage

Current test coverage includes:

### DEX Module

- **Pool Creation**: Valid pools, duplicate tokens, zero amounts
- **Swap Operations**: AMM formula validation, slippage protection, fee calculations
- **Liquidity Management**: Add/remove liquidity, ratio validation, token issuance
- **Message Validation**: Address validation, parameter bounds, token validation

### Compute Module

- **Provider Registration**: Endpoint validation, stake requirements
- **Compute Requests**: API URL validation, fee handling, request tracking
- **Result Submission**: Request status updates, provider verification
- **Cross-module**: Provider rewards, fee distribution

### Oracle Module

- **Oracle Registration**: Validator authentication, duplicate prevention
- **Price Feeds**: Price validation, multi-oracle support, timestamp tracking
- **Median Calculation**: Aggregating prices from multiple oracles
- **Deviation Detection**: Price outlier detection, consensus mechanisms

### Integration Tests

- **App Initialization**: Module registration, account creation, genesis validation
- **Module Accounts**: Fee collector, distribution, module-specific accounts
- **Cross-module Flows**: DEX + Oracle price alignment, Compute + Fee distribution

### E2E Tests

- **DEX Workflow**: Create pool → Add liquidity → Swap → Remove liquidity
- **Compute Workflow**: Register provider → Request compute → Submit result
- **Oracle Workflow**: Register oracle → Submit prices → Verify aggregation
- **Cross-module**: Price feeds influencing DEX operations

## Writing Tests

### Test Helpers

Use the provided test utilities in `testutil/keeper/`:

```go
import keepertest "github.com/paw/testutil/keeper"

func TestMyFeature(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)

    // Create a test pool
    poolId := keepertest.CreateTestPool(t, k, ctx,
        "upaw", "uusdt",
        sdk.NewInt(1000000),
        sdk.NewInt(2000000))
}
```

### Table-Driven Tests

Follow the table-driven test pattern:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        msg     *types.MsgSomething
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid case",
            msg: &types.MsgSomething{...},
            wantErr: false,
        },
        {
            name: "invalid case",
            msg: &types.MsgSomething{...},
            wantErr: true,
            errMsg: "expected error",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### Test Suites

Use testify suites for complex test setups:

```go
type MyTestSuite struct {
    suite.Suite
    keeper keeper.Keeper
    ctx    sdk.Context
}

func (suite *MyTestSuite) SetupTest() {
    suite.keeper, suite.ctx = keepertest.DexKeeper(suite.T())
}

func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

### Assertions

Use testify/require for assertions:

```go
require.NoError(t, err)
require.NotNil(t, resp)
require.Equal(t, expected, actual)
require.True(t, condition)
require.Contains(t, str, substring)
```

## Test Best Practices

1. **Isolation**: Each test should be independent and not rely on other tests
2. **Cleanup**: Use test helpers that clean up after themselves
3. **Coverage**: Aim for 80%+ code coverage
4. **Edge Cases**: Test boundary conditions, zero values, overflow scenarios
5. **Error Paths**: Test both success and failure cases
6. **Documentation**: Use descriptive test names and comments

## CI/CD Integration

Tests run automatically via GitHub Actions on:

- Push to `master`, `main`, or `develop` branches
- Pull requests to these branches
- Multiple Go versions (1.21, 1.22)

### CI Workflow

1. **Unit Tests**: Fast feedback on code changes
2. **Integration Tests**: Module interaction validation
3. **Linting**: Code quality and style checks
4. **Build**: Verify binaries compile successfully
5. **Security**: Gosec vulnerability scanning
6. **Coverage**: Upload to Codecov

### Workflow Files

- `.github/workflows/test.yml`: Main test workflow
- Configuration includes race detection, timeout settings, and caching

## Debugging Tests

### Verbose Output

```bash
go test -v ./x/dex/keeper/...
```

### Run Specific Test

```bash
go test -v -run TestCreatePool ./x/dex/keeper/...
```

### Race Detection

```bash
go test -race ./...
```

### Profile Tests

```bash
go test -cpuprofile cpu.prof -memprofile mem.prof ./...
go tool pprof cpu.prof
```

## Simulation Tests

For comprehensive chain behavior testing:

```bash
make sim-test
```

This runs randomized transactions to test chain stability and invariants.

## Adding New Tests

When adding new features:

1. Write keeper unit tests in `x/{module}/keeper/keeper_test.go`
2. Add message validation tests in `x/{module}/types/msg_test.go`
3. Update integration tests in `app/app_test.go` if cross-module
4. Add E2E test scenarios in `tests/e2e/e2e_test.go` for workflows
5. Update this documentation with new test coverage

## Continuous Improvement

- Review test coverage reports regularly
- Add tests for bug fixes to prevent regression
- Refactor tests when code changes
- Keep test utilities DRY (Don't Repeat Yourself)
- Update test infrastructure as the chain evolves

## Resources

- [Cosmos SDK Testing Guide](https://docs.cosmos.network/main/building-modules/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
