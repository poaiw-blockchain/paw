# PAW Blockchain Invariant Tests

This directory contains comprehensive invariant tests for the PAW blockchain. Invariant tests are critical for ensuring state machine consistency and catching bugs before they reach production.

## Overview

Invariants are properties that must always hold true in the blockchain state. These tests verify that core assumptions about the system remain valid after any operation.

## Test Files

### Bank Module Invariants (`bank_invariants_test.go` - 343 lines)

Tests critical bank module invariants:

1. **Total Supply Conservation**: Sum of all account balances equals total supply
2. **Non-Negative Balances**: All account balances are >= 0
3. **Module Account Consistency**: Module accounts have correct balances and types
4. **Coin Denomination Validation**: All coins have valid denominations
5. **Supply Tracking**: Mint/burn operations correctly update total supply
6. **Send Coins Conservation**: Transfers don't create or destroy tokens
7. **Locked Coins**: Locked coin accounting is correct

### DEX Module Invariants (`dex_invariants_test.go` - 477 lines)

Tests critical DEX module invariants:

1. **Constant Product Formula**: K = reserveA × reserveB never decreases (only increases with fees)
2. **Pool Reserve Balance Matching**: Pool reserves equal module account balance
3. **LP Share Conservation**: Sum of all LP shares equals pool's total shares
4. **Non-Negative Reserves**: All pool reserves are >= 0
5. **Reserve Ratio Bounds**: Pool ratios stay within reasonable bounds
6. **Swap Preservation**: Swaps maintain all invariants
7. **Minimum Liquidity**: Pools maintain minimum liquidity requirement

**Critical**: The constant product invariant (K) is the most important DEX invariant. If K decreases during a swap, it indicates value extraction bugs.

### Staking Module Invariants (`staking_invariants_test.go` - 428 lines)

Tests critical staking module invariants:

1. **Bonded Tokens Consistency**: Bonded validator tokens equal bonded pool balance
2. **Delegation Shares Sum**: Sum of delegator shares equals validator's total shares
3. **Unbonding Queue Integrity**: Unbonding entries have valid completion times and balances
4. **Validator Power Consistency**: Validator power equals tokens / power reduction
5. **Non-Negative Tokens**: All validators have non-negative tokens and shares
6. **Not-Bonded Pool Consistency**: Not-bonded pool tracks unbonding tokens correctly
7. **Redelegation Integrity**: Redelegation entries are valid
8. **Jailed Validators**: Jailed validators are not bonded
9. **Commission Rates**: Commission rates are within [0, 1]
10. **Minimum Self-Delegation**: Validators meet minimum self-delegation

### Compute Module Invariants (`compute_invariants_test.go` - 421 lines)

Tests critical compute module invariants:

1. **Escrow Balance Conservation**: Module balance >= sum of all active escrows
2. **Escrow Status Consistency**: Escrowed amounts match request status
3. **Request Nonce Uniqueness**: All request IDs are unique
4. **Positive Max Payments**: All requests have positive max payment
5. **Provider Quota Limits**: Providers don't exceed concurrent request quotas
6. **Valid Compute Specs**: All requests have valid CPU/memory/disk/timeout specs
7. **Timestamp Consistency**: Request timestamps are logically ordered
8. **Provider Stake Requirements**: Active providers meet minimum stake
9. **Result Hash Presence**: Completed requests have result hashes
10. **Reputation Bounds**: Provider reputation scores are in [0, 100]

### Oracle Module Invariants (`oracle_invariants_test.go` - 480 lines)

Tests critical oracle module invariants:

1. **Price Bounds**: All prices are within reasonable bounds [ε, 1B]
2. **Weighted Median Accuracy**: Aggregate prices match weighted median of submissions
3. **Miss Counter Consistency**: Miss counters don't exceed slash window
4. **Slash Counter Accuracy**: Validators exceeding threshold are slashed/jailed
5. **Voting Power Consistency**: Price submission voting power matches staking power
6. **Price Block Heights**: Block heights are monotonic and not in the future
7. **Validator Count Accuracy**: Reported validator counts don't exceed total validators
8. **Active Validator Records**: All bonded validators have oracle records
9. **Parameter Validity**: All oracle parameters are within valid ranges
10. **Timestamp Reasonableness**: Price timestamps are not in distant future

## Running Tests

### Run All Invariant Tests
```bash
cd tests/invariants
go test -v ./...
```

### Run Specific Module Tests
```bash
# Bank invariants
go test -v -run TestBankInvariantTestSuite

# DEX invariants
go test -v -run TestDEXInvariantTestSuite

# Staking invariants
go test -v -run TestStakingInvariantTestSuite

# Compute invariants
go test -v -run TestComputeInvariantTestSuite

# Oracle invariants
go test -v -run TestOracleInvariantTestSuite
```

### Run Specific Invariant Test
```bash
# Run just the constant product invariant
go test -v -run TestDEXInvariantTestSuite/TestConstantProductInvariant
```

### Run with Coverage
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Execution Time

- **Bank invariants**: ~2-5 minutes
- **DEX invariants**: ~3-7 minutes
- **Staking invariants**: ~5-10 minutes
- **Compute invariants**: ~3-6 minutes
- **Oracle invariants**: ~4-8 minutes
- **Total (all invariants)**: ~20-30 minutes

## CI/CD Integration

These tests run automatically on:
- Every push to master/main
- Every pull request
- Nightly at 2 AM UTC
- Manual workflow dispatch

See `hub/workflows/simulation-tests.yml` for details.

## Critical Invariants for Production

The following invariants are **CRITICAL** and their violation indicates serious bugs:

### DEX
- **Constant Product (K)**: Must never decrease during swaps
- **Pool Reserves = Module Balance**: Token accounting must be exact
- **LP Share Conservation**: Shares can't be created or destroyed improperly

### Bank
- **Total Supply = Sum of Balances**: Fundamental token conservation
- **Non-Negative Balances**: Negative balances should be impossible

### Staking
- **Bonded Tokens = Pool Balance**: Staking pool accounting must be exact
- **Delegation Shares Conservation**: Shares must sum to validator total

### Compute
- **Escrow Conservation**: Escrowed funds must be fully accounted for
- **Escrow Status Consistency**: Finalized requests must have zero escrow

### Oracle
- **Price Bounds**: Prices outside bounds indicate data corruption
- **Weighted Median Accuracy**: Aggregation algorithm correctness

## Debugging Failed Invariants

When an invariant test fails:

1. **Review the assertion message** - it shows expected vs actual values
2. **Check recent code changes** - invariant breaks often trace to recent commits
3. **Run the specific test in isolation** - helps reproduce the issue
4. **Add logging** - insert fmt.Printf statements to trace state
5. **Check module state** - use keeper queries to inspect current state
6. **Review related operations** - check what operations modified state

## Adding New Invariants

When adding new functionality:

1. Identify state invariants that must hold
2. Add test function to appropriate suite
3. Use descriptive names: `TestModuleNameInvariant`
4. Add comprehensive assertions with clear error messages
5. Test edge cases and boundary conditions
6. Update this README with the new invariant

## Test Structure

All invariant tests follow this pattern:

```go
func (suite *TestSuite) TestInvariantName() {
    // 1. Setup state (create accounts, pools, etc.)

    // 2. Perform operations

    // 3. Check invariant holds
    suite.Require().True(
        invariantCondition,
        "Invariant violated: expected=%s, got=%s",
        expected, actual,
    )
}
```

## Best Practices

1. **Test independence**: Each test should be independent
2. **Clear assertions**: Use descriptive error messages
3. **Edge cases**: Test boundary conditions
4. **State cleanup**: SetupTest runs before each test
5. **Deterministic**: Tests should be reproducible
6. **Fast execution**: Keep tests efficient

## Related Documentation

- [Simulation Tests](../simulation/README.md)
- [Testing Guide](../../docs/guides/testing/RUN-TESTS-LOCALLY.md)
- [Module Implementation](../../docs/implementation/)
