# PAW Blockchain Testing Guide

This directory contains comprehensive testing infrastructure for the PAW blockchain, including advanced Cosmos SDK-specific testing tools and methodologies.

## Table of Contents

1. [Overview](#overview)
2. [Testing Levels](#testing-levels)
3. [CometMock Testing](#cometmock-testing)
4. [Invariant Testing](#invariant-testing)
5. [Property-Based Testing](#property-based-testing)
6. [Simulation Testing](#simulation-testing)
7. [Integration Testing](#integration-testing)
8. [Running Tests](#running-tests)
9. [Best Practices](#best-practices)

## Overview

PAW uses multiple layers of testing to ensure correctness, security, and performance:

- **Unit Tests**: Test individual functions and modules
- **Integration Tests**: Test module interactions
- **E2E Tests**: Test complete user workflows
- **CometMock Tests**: Fast consensus-free testing
- **Invariant Tests**: Verify state machine invariants
- **Property Tests**: Mathematical property verification
- **Simulation Tests**: Randomized operation testing

## Testing Levels

### Unit Tests

Located throughout the codebase in `*_test.go` files.

```bash
# Run all unit tests
make test-unit

# Run specific module tests
go test ./x/dex/...
go test ./x/compute/...
go test ./x/oracle/...
```

### Integration Tests

Located in `tests/e2e/` and module integration test files.

```bash
# Run integration tests
make test-integration
```

### Keeper Tests

Test keeper logic in isolation.

```bash
# Run all keeper tests
make test-keeper
```

## CometMock Testing

CometMock is a drop-in replacement for CometBFT that enables **much faster** end-to-end testing by bypassing consensus.

### Why Use CometMock?

- **Speed**: 100-1000x faster than real consensus
- **Deterministic**: Reproducible test results
- **Simplified**: No network overhead
- **Identical**: Same interface as CometBFT

### When to Use CometMock vs Real CometBFT

**Use CometMock for:**
- Rapid iteration during development
- CI/CD pipelines
- Unit and integration tests
- Testing application logic
- Debugging state transitions

**Use Real CometBFT for:**
- Consensus testing
- Network behavior testing
- Performance benchmarks
- Validator integration tests
- Production-like scenarios

### CometMock Examples

```go
// Create a CometMock instance
config := cometmock.DefaultCometMockConfig()
app := cometmock.SetupCometMock(t, config)

// Produce blocks rapidly
app.NextBlocks(1000) // Creates 1000 blocks in milliseconds

// Test transactions
tx := CreateTestTx()
app.BeginBlock([][]byte{tx})
app.EndBlock()

// Query state
ctx := app.Context()
balance := app.BankKeeper.GetBalance(ctx, addr, "upaw")
```

### Running CometMock Tests

```bash
# Run E2E tests with CometMock
make test-cometmock

# Or set environment variable
USE_COMETMOCK=true go test ./tests/e2e/...
```

### CometMock Configuration

```go
// Fast mode (minimal validators, instant blocks)
config := cometmock.FastMockConfig()

// Realistic mode (many validators, realistic timing)
config := cometmock.RealisticMockConfig()

// Custom configuration
config := cometmock.DefaultMockConfig().
    WithValidators(100).
    WithBlockTime(5 * time.Second).
    WithFastMode(false)
```

## Invariant Testing

Invariants are properties that should **always** be true in the state machine.

### Bank Module Invariants

Located in `tests/invariants/bank_invariants_test.go`:

- **Total Supply**: Sum of all account balances equals total supply
- **Non-Negative Balances**: No account has negative balance
- **Denom Metadata**: All denoms with supply have metadata

### Staking Module Invariants

Located in `tests/invariants/staking_invariants_test.go`:

- **Module Account Balance**: Bonded pool balance matches bonded tokens
- **Validator Bonded Tokens**: Sum of bonded validators equals total bonded
- **Delegation Shares**: Delegation shares sum equals validator shares
- **Positive Delegations**: All delegations have positive shares

### DEX Module Invariants

Located in `tests/invariants/dex_invariants_test.go`:

- **Constant Product (x*y=k)**: Pool reserves maintain k invariant
- **LP Shares**: Sum of LP shares equals pool total
- **No Negative Reserves**: All reserves are positive
- **Pool Balances**: Reserves match actual token balances
- **Minimum Liquidity**: All pools maintain minimum liquidity

### Running Invariant Tests

```bash
# Run all invariant tests
make test-invariants

# Run specific invariants
go test ./tests/invariants/ -run TestBankInvariants
go test ./tests/invariants/ -run TestStakingInvariants
go test ./tests/invariants/ -run TestDEXInvariants
```

### Creating Custom Invariants

```go
func (s *MyTestSuite) InvariantMyProperty() (string, bool) {
    var msg string
    var broken bool

    // Check invariant
    if conditionViolated {
        broken = true
        msg = sdk.FormatInvariant(
            "module",
            "invariant-name",
            "description of violation",
        )
    }

    return msg, broken
}
```

## Property-Based Testing

Property-based tests verify mathematical properties using randomized inputs.

### DEX Properties

Located in `tests/property/dex_properties_test.go`:

1. **Pool Creation Commutative**: Creating pool (A,B) same as (B,A)
2. **Swap Never Increases Reserves**: Swaps don't increase reserves beyond fees
3. **Add/Remove Liquidity Roundtrip**: Adding then removing returns proportional amounts
4. **Price Impact Increases**: Larger swaps have worse price per token
5. **Swap Aggregation**: Multiple small swaps similar to one large swap
6. **Reserves Stay Positive**: Reserves never go negative
7. **K Never Decreases**: Constant product never decreases

### Running Property Tests

```bash
# Run all property tests
make test-properties

# Run with more iterations for thorough testing
go test ./tests/property/... -quick.count=10000
```

### Property Test Example

```go
func TestPropertySwapInvariant(t *testing.T) {
    property := func(reserveA, reserveB, swapAmount uint64) bool {
        // Skip invalid inputs
        if reserveA == 0 || reserveB == 0 {
            return true
        }

        // Test property
        k1 := reserveA * reserveB
        // ... perform swap
        k2 := newReserveA * newReserveB

        // K should never decrease
        return k2 >= k1
    }

    err := quick.Check(property, &quick.Config{
        MaxCount: 1000,
    })
    require.NoError(t, err)
}
```

## Simulation Testing

Simulation runs thousands of random operations to find edge cases.

### Full App Simulation

Located in `tests/simulation/sim_test.go`:

```bash
# Run full simulation (1000+ random operations)
make test-simulation

# Run with specific seed for reproducibility
go test ./tests/simulation/... -SimulationSeed=42

# Run with more operations
go test ./tests/simulation/... -SimulationNumBlocks=1000

# Run with all invariants enabled
go test ./tests/simulation/... -SimulationAllInvariants=true
```

### Simulation Parameters

Configured in `simapp/params.go`:

- Number of accounts
- Number of validators
- Operation probabilities
- Initial balances
- Pool configurations

### Simulation Features

1. **Randomized Operations**: Swaps, liquidity operations, transfers, etc.
2. **State Export/Import**: Verify state can be exported and imported
3. **Determinism**: Same seed produces same result
4. **Invariant Checking**: Optionally check invariants every block

### Running Simulations

```bash
# Quick simulation (100 blocks)
go test ./tests/simulation/... -SimulationNumBlocks=100

# Full simulation (default 500 blocks)
make test-simulation

# Determinism test
go test ./tests/simulation/... -run TestAppStateDeterminism

# After import test
go test ./tests/simulation/... -run TestAppSimulationAfterImport

# With all invariants
go test ./tests/simulation/... -run TestSimulationWithInvariants
```

## Integration Testing

Integration test helpers located in `testutil/integration/`:

### Multi-Node Networks

```go
// Create test network
config := integration.DefaultNetworkConfig()
network := integration.New(t, config)
defer network.Cleanup()

// Initialize chain
network.InitChain(t)

// Wait for height
network.WaitForHeight(10)
```

### Test Accounts

```go
// Create account manager
manager := integration.NewTestAccountManager()

// Create individual accounts
alice := manager.CreateAccount("alice")
bob := manager.CreateAccount("bob")

// Create pre-funded accounts
accounts, balances := manager.CreateFundedAccounts(
    10,
    "user",
    sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000)),
)

// Use default account set
accountSet := integration.CreateDefaultAccountSet()
alice := accountSet.Alice
validator := accountSet.Validator1
```

### Test Contracts

```go
// Create contract manager
contracts := integration.NewContractManager()

// Store contract
info, err := contracts.StoreContract("cw20", "./testdata/cw20.wasm")

// Instantiate contract
initMsg := integration.NewCW20InitMsg("Token", "TKN", 6, balances)
contract, err := contracts.InstantiateContract("my-token", admin, initMsg)

// Execute contract
executeMsg := integration.NewAMMSwapMsg("upaw", "uusdc", "1000000")
helper := integration.NewContractExecuteHelper(contract.Address, sender, funds)
msgBytes, err := helper.ExecuteMsg(executeMsg)
```

## Running Tests

### Quick Test Commands

```bash
# Run all tests
make test

# Run all advanced tests
make test-all-advanced

# Run specific test suites
make test-invariants      # Invariant tests
make test-properties      # Property-based tests
make test-simulation      # Simulation tests
make test-cometmock       # CometMock E2E tests

# Run with coverage
make test-coverage

# Run benchmarks
make benchmark
```

### Test Flags

```bash
# Verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./...

# Race detection
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...

# Timeout
go test -timeout 30m ./tests/simulation/...
```

## Best Practices

### 1. Test Pyramid

- Many unit tests (fast, focused)
- Moderate integration tests (medium speed)
- Few E2E tests (slow, comprehensive)
- Regular simulation runs (find edge cases)

### 2. Use CometMock for Speed

```go
// Instead of spinning up full nodes
if os.Getenv("USE_COMETMOCK") == "true" {
    app := cometmock.SetupCometMock(t, config)
    // Fast testing
} else {
    // Full consensus testing
}
```

### 3. Check Invariants Regularly

```go
// After operations, verify invariants
msg, broken := s.InvariantTotalSupply()
s.Require().False(broken, msg)
```

### 4. Use Property Tests for Math

Properties > Examples for mathematical functions:

```go
// Instead of testing specific values
// Test mathematical properties
func TestCommutative(t *testing.T) {
    property := func(a, b int) bool {
        return add(a, b) == add(b, a)
    }
    quick.Check(property, nil)
}
```

### 5. Simulation for Edge Cases

Run simulations regularly to find unexpected edge cases:

```bash
# Run nightly
make test-simulation

# Run with different seeds
for i in {1..10}; do
    go test ./tests/simulation/... -SimulationSeed=$RANDOM
done
```

### 6. Test Account Management

```go
// Use test account helpers
manager := integration.NewTestAccountManager()
accounts, balances := manager.CreateFundedAccounts(10, "test", funding)

// Clean separation of concerns
alice := accountSet.Alice
validator := accountSet.Validator1
```

### 7. Deterministic Tests

```go
// Use fixed seeds for reproducibility
r := rand.New(rand.NewSource(42))

// Or use test name as seed
seed := int64(fnv.New64a().Sum64([]byte(t.Name())))
r := rand.New(rand.NewSource(seed))
```

## Continuous Integration

Tests run automatically on:

- Every commit (unit + integration)
- Pull requests (all tests including simulation)
- Nightly (extended simulation with multiple seeds)

See `.github/workflows/` for CI configuration.

## Troubleshooting

### Simulation Failures

```bash
# Run with specific seed that failed
go test ./tests/simulation/... -SimulationSeed=<failed-seed>

# Enable verbose logging
go test -v ./tests/simulation/... -SimulationVerbose=true
```

### CometMock Issues

```bash
# Verify environment variable is set
echo $USE_COMETMOCK

# Check for import errors
go mod tidy
```

### Invariant Failures

```bash
# Run invariants in isolation
go test ./tests/invariants/ -run <specific-invariant>

# Enable detailed output
go test -v ./tests/invariants/...
```

## Additional Resources

- [Cosmos SDK Testing Guide](https://docs.cosmos.network/main/building-modules/testing)
- [CometMock Documentation](https://github.com/informalsystems/CometMock)
- [Property-Based Testing](https://golang.org/pkg/testing/quick/)
- [Simulation Testing](https://github.com/cosmos/cosmos-sdk/tree/main/x/simulation)

## Contributing

When adding new features:

1. Add unit tests for new functions
2. Add invariants for new state
3. Add property tests for math
4. Update simulation operations
5. Add integration test examples

See `CONTRIBUTING.md` for more details.
