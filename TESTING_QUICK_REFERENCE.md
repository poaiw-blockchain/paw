# PAW Blockchain - Testing Quick Reference

## Quick Commands

```bash
# Run all tests
make test

# Run all advanced tests
make test-all-advanced

# Individual test suites
make test-invariants          # State machine invariants
make test-properties          # Mathematical properties
make test-simulation          # Randomized operations
make test-cometmock          # Fast E2E tests

# Coverage
make test-coverage

# Benchmarks
make benchmark
make benchmark-cometmock
make benchmark-invariants
```

## Test Categories

| Category        | Files                          | Purpose                            | Speed  |
| --------------- | ------------------------------ | ---------------------------------- | ------ |
| **CometMock**   | `testutil/cometmock/`          | Fast consensus-free testing        | ⚡⚡⚡ |
| **Invariants**  | `tests/invariants/`            | State correctness verification     | ⚡⚡   |
| **Properties**  | `tests/property/`              | Mathematical property verification | ⚡⚡⚡ |
| **Simulation**  | `tests/simulation/`, `simapp/` | Randomized operation testing       | ⚡     |
| **Integration** | `testutil/integration/`        | Network/account/contract helpers   | ⚡⚡   |

## CometMock Quick Start

```go
import "github.com/paw-chain/paw/testutil/cometmock"

// Fast mode (1 validator, instant blocks)
config := cometmock.FastMockConfig()
app := cometmock.SetupCometMock(t, config)

// Produce blocks rapidly
app.NextBlocks(100)  // 100 blocks in milliseconds

// Test state
ctx := app.Context()
balance := app.BankKeeper.GetBalance(ctx, addr, "upaw")
```

## Invariant Tests

### Bank Module

```go
InvariantTotalSupply()          // Supply = sum of accounts
InvariantNonNegativeBalances()  // No negative balances
InvariantDenomMetadata()        // All denoms have metadata
```

### Staking Module

```go
InvariantModuleAccountCoins()   // Pool balances match
InvariantValidatorsBonded()     // Validators sum correctly
InvariantDelegationShares()     // Delegation shares match
InvariantPositiveDelegation()   // All delegations positive
```

### DEX Module

```go
InvariantPoolReservesXYK()      // x*y=k maintained
InvariantPoolLPShares()         // LP shares sum correctly
InvariantNoNegativeReserves()   // All reserves positive
InvariantPoolBalances()         // Reserves match balances
InvariantMinimumLiquidity()     // Min liquidity locked
```

## Property Tests

```go
// Example: Commutative property
property := func(a, b uint64) bool {
    pool1 := createPool(a, b)
    pool2 := createPool(b, a)
    return pool1.K.Equal(pool2.K)
}

err := quick.Check(property, &quick.Config{MaxCount: 1000})
```

### DEX Properties

- Pool creation is commutative
- Swaps never increase reserves
- Add/remove liquidity roundtrip
- Price impact increases with size
- Reserves stay positive
- K never decreases

## Simulation Tests

```bash
# Full simulation (500 blocks, random ops)
make test-simulation

# Determinism test
make test-simulation-determinism

# State export/import
make test-simulation-import-export

# With all invariants
make test-simulation-with-invariants
```

## Integration Helpers

### Networks

```go
config := integration.DefaultNetworkConfig()
network := integration.New(t, config)
defer network.Cleanup()

network.InitChain(t)
network.WaitForHeight(10)
```

### Accounts

```go
// Account manager
manager := integration.NewTestAccountManager()
alice := manager.CreateAccount("alice")

// Pre-configured set
accounts := integration.CreateDefaultAccountSet()
alice := accounts.Alice
validator := accounts.Validator1

// Funded accounts
accounts, balances := manager.CreateFundedAccounts(
    10, "user", sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000)),
)
```

### Contracts

```go
// Contract manager
contracts := integration.NewContractManager()

// CW20 token
initMsg := integration.NewCW20InitMsg("Token", "TKN", 6, balances)
contract, err := contracts.InstantiateContract("my-token", admin, initMsg)

// AMM operations
swapMsg := integration.NewAMMSwapMsg("upaw", "uusdc", "1000000")
addLiqMsg := integration.NewAMMAddLiquidityMsg("1000000", "1000000")
```

## Environment Variables

```bash
# Enable CometMock for E2E tests
export USE_COMETMOCK=true

# Simulation configuration
export SimulationSeed=42                    # Reproducible tests
export SimulationNumBlocks=1000            # More blocks
export SimulationAllInvariants=true        # Check all invariants
export SimulationVerbose=true              # Detailed logging
```

## Common Patterns

### Test with Invariants

```go
func (s *Suite) TestOperationMaintainsInvariants() {
    // Perform operation
    err := s.app.DoSomething(s.ctx, params)
    s.Require().NoError(err)

    // Check invariants
    msg, broken := s.InvariantTotalSupply()
    s.Require().False(broken, msg)
}
```

### Property-Based Test

```go
func TestMathProperty(t *testing.T) {
    property := func(x, y uint64) bool {
        if x == 0 || y == 0 {
            return true  // Skip invalid
        }
        // Test mathematical property
        return result == expected
    }

    err := quick.Check(property, nil)
    require.NoError(t, err)
}
```

### CometMock vs Real Consensus

```go
if os.Getenv("USE_COMETMOCK") == "true" {
    app := cometmock.SetupCometMock(t, config)
    // Fast testing
} else {
    // Full consensus testing
}
```

## File Locations

```
testutil/
├── cometmock/           # CometMock integration
│   ├── setup.go         # Main setup and block production
│   ├── config.go        # Configuration options
│   └── errors.go        # Error definitions
└── integration/         # Integration helpers
    ├── network.go       # Multi-node networks
    ├── accounts.go      # Test account management
    └── contracts.go     # Contract deployment

tests/
├── e2e/
│   └── cometmock_test.go    # CometMock E2E tests
├── invariants/
│   ├── bank_invariants_test.go
│   ├── staking_invariants_test.go
│   └── dex_invariants_test.go
├── property/
│   └── dex_properties_test.go
└── simulation/
    └── sim_test.go          # Full app simulation

simapp/
├── params.go            # Simulation parameters
└── state.go             # Random state generation
```

## Performance Tips

1. **Use CometMock for development** - 100-1000x faster
2. **Run invariants selectively** - Not every test needs all invariants
3. **Use quick.Check with reasonable MaxCount** - 1000 is usually enough
4. **Parallel tests** - Use `t.Parallel()` when safe
5. **Skip long simulations in CI** - Use tags or env vars

## Troubleshooting

### Tests won't compile

```bash
go mod tidy
go mod download
```

### CometMock tests skipped

```bash
export USE_COMETMOCK=true
make test-cometmock
```

### Simulation fails with specific seed

```bash
go test ./tests/simulation/... -SimulationSeed=<seed> -v
```

### Invariant violation

```bash
# Run specific invariant with verbose output
go test ./tests/invariants/ -run TestBankInvariants -v
```

## Next Steps

1. Review `tests/README.md` for comprehensive guide
2. Check `TESTING_TOOLS_SUMMARY.md` for detailed documentation
3. Implement module-specific invariants
4. Add simulation operations for custom modules
5. Create more property tests for business logic

## Resources

- **Detailed Guide**: `tests/README.md`
- **Complete Summary**: `TESTING_TOOLS_SUMMARY.md`
- **Cosmos SDK Docs**: https://docs.cosmos.network/main/building-modules/testing
- **CometMock**: https://github.com/informalsystems/CometMock
