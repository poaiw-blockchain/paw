# PAW Blockchain Simulation Tests

This directory contains comprehensive simulation tests for the PAW blockchain. Simulation tests exercise the entire application with randomized operations to catch state machine bugs, non-determinism, and consensus issues.

## Overview

Simulation testing runs the blockchain application with:
- **Random operations** across all modules
- **Multiple blocks** (typically 500-1000)
- **Invariant checking** at every block
- **Determinism verification** (same seed = same result)
- **State export/import** testing

## Test Files

### Main Simulation Tests (`sim_test.go` - 554 lines)

#### TestFullAppSimulation
Runs full application simulation for 500 blocks with 200 random operations per block.

**Configuration**:
- Blocks: 500
- Operations per block: 200
- Total operations: ~100,000
- Seed: 42 (deterministic)
- Invariants: Enabled
- Runtime: ~60-90 minutes

**What it tests**:
- All modules work together correctly
- Random operations don't break state
- Invariants hold after every block
- Application can handle high transaction load

#### TestAppStateDeterminism
Verifies that simulation is deterministic - same seed produces same final state.

**Configuration**:
- Seeds tested: 3
- Runs per seed: 5
- Blocks per run: 100
- Operations per block: 200
- Runtime: ~30-45 minutes

**Critical**: Non-determinism indicates consensus bugs that could cause chain halts.

#### TestAppImportExport
Tests state export and import functionality.

**What it tests**:
1. Run simulation and export state
2. Create new app and import exported state
3. Verify all module stores match exactly
4. No data loss during export/import

**Runtime**: ~45-60 minutes

#### TestAppSimulationAfterImport
Verifies simulation continues to work after importing exported state.

**What it tests**:
1. Run initial simulation
2. Export state
3. Import into new app
4. Continue simulation
5. Verify invariants still hold

**Runtime**: ~60-90 minutes

#### TestMultiSeedFullSimulation
Runs full simulation with multiple random seeds in parallel.

**Configuration**:
- Seeds: [1, 2, 3, 5, 8, 13, 21]
- Blocks per seed: 200
- Operations per block: 100
- Runs: Parallel
- Runtime: ~90-120 minutes

**Purpose**: Catch seed-specific bugs and edge cases.

### Simulation Operations (`operations.go` - 369 lines)

Defines weighted random operations for all modules:

#### Bank Operations (Weight: 100)
- **MsgSend**: Transfer tokens between accounts

#### Staking Operations (Various weights)
- **MsgDelegate**: Delegate tokens to validators
- **MsgUndelegate**: Unbond tokens from validators
- **MsgRedelegate**: Move delegation between validators
- **MsgCreateValidator**: Create new validators

#### DEX Operations
- **MsgCreatePool** (Weight: 30): Create new liquidity pools
- **MsgSwap** (Weight: 50): Perform token swaps
- **MsgAddLiquidity** (Weight: 40): Add liquidity to pools
- **MsgRemoveLiquidity** (Weight: 20): Remove liquidity from pools

#### Oracle Operations
- **MsgSubmitPrice** (Weight: 60): Submit price feeds
- **MsgDelegateFeeder** (Weight: 10): Delegate price feed authority

#### Compute Operations
- **MsgSubmitRequest** (Weight: 20): Submit compute requests
- **MsgSubmitResult** (Weight: 15): Submit computation results
- **MsgRegisterProvider** (Weight: 10): Register as compute provider

**Higher weights = more frequent operations**

### Random Parameters (`params.go` - 309 lines)

Generates random genesis state and parameters:

#### DEX Parameters
- **SwapFee**: 0.1% - 1.0%
- **LPFee**: 0.01% - 0.5%
- **ProtocolFee**: 0.01% - 0.1%
- **MinLiquidity**: 100 - 10,000
- **MaxSlippage**: 1% - 10%

#### Oracle Parameters
- **VotePeriod**: 5 - 50 blocks
- **VoteThreshold**: 50% - 67%
- **SlashFraction**: 0.01% - 1%
- **SlashWindow**: 100 - 1,000 blocks
- **MinValidPerWindow**: 50% - 90% of slash window
- **TwapLookback**: 10 - 100 blocks

#### Compute Parameters
- **MinProviderStake**: 1,000 - 100,000
- **RequestTimeout**: 100 - 1,000 blocks
- **ResultTimeout**: 50 - 500 blocks
- **MinGasPrice**: 0.001 - 0.1
- **MaxRequestsPerBlock**: 10 - 100

#### Account Funding
Each simulation account starts with:
- 100,000 STAKE (default bond denom)
- 100,000 ATOM
- 100,000 OSMO

### Type Definitions (`types.go` - 55 lines)

Defines keeper interfaces used by simulation operations.

## Running Simulation Tests

### Run Full Simulation
```bash
cd tests/simulation
go test -v -run TestFullAppSimulation -timeout 2h
```

### Run Determinism Test
```bash
go test -v -run TestAppStateDeterminism -timeout 2h
```

### Run Import/Export Test
```bash
go test -v -run TestAppImportExport -timeout 2h
```

### Run Multi-Seed Test
```bash
go test -v -run TestMultiSeedFullSimulation -timeout 4h
```

### Run All Simulation Tests
```bash
go test -v ./... -timeout 6h
```

### Custom Configuration
```bash
# Run with custom parameters
go test -v -run TestFullAppSimulation \
  -NumBlocks=1000 \
  -BlockSize=300 \
  -Commit=true \
  -Enabled=true \
  -Verbose=true \
  -timeout 4h
```

### Benchmark Simulation
```bash
go test -bench=BenchmarkFullAppSimulation -benchtime=3x
```

## Test Execution Time

| Test | Duration | CPU Usage | Memory |
|------|----------|-----------|--------|
| TestFullAppSimulation | 60-90 min | High | 2-4 GB |
| TestAppStateDeterminism | 30-45 min | Very High | 3-5 GB |
| TestAppImportExport | 45-60 min | High | 3-6 GB |
| TestAppSimulationAfterImport | 60-90 min | High | 4-6 GB |
| TestMultiSeedFullSimulation | 90-120 min | Very High | 4-8 GB |

**Recommended**: Use 8-core machine with 16GB RAM for optimal performance.

## CI/CD Integration

Simulation tests run via  Actions (`hub/workflows/simulation-tests.yml`):

### Scheduled Runs
- **Nightly**: Full simulation suite at 2 AM UTC
- **On-demand**: Manual workflow dispatch with custom parameters

### On Push/PR
- Invariant tests (fast feedback)
- Basic simulation tests (100 blocks)

### Test Matrix
- **Invariant tests**: Run in parallel for each module
- **Simulation tests**: Run on 8-core runners
- **Determinism**: Separate job with extended timeout
- **Multi-seed**: Only on schedule/manual trigger

## Interpreting Results

### Success Indicators
✅ All invariants pass at every block
✅ Same seed produces identical app hash
✅ Export/import preserves all state
✅ No panics or errors during simulation

### Failure Indicators
❌ **Invariant violation**: State machine bug - CRITICAL
❌ **Non-determinism**: Different app hash with same seed - CRITICAL
❌ **Panic during simulation**: Application crash
❌ **Export/import mismatch**: State serialization bug

## Debugging Failed Simulations

### Step 1: Identify the Failure
Check the error message:
- Invariant name tells you which module failed
- Block height tells you when it failed
- Operation log shows what operations were executed

### Step 2: Reproduce Locally
```bash
# Use the same seed that failed in CI
go test -v -run TestFullAppSimulation \
  -Seed=<failed_seed> \
  -Verbose=true
```

### Step 3: Add Detailed Logging
```go
// In the failing operation
fmt.Printf("DEBUG: Operation state before: %+v\n", state)
// Execute operation
fmt.Printf("DEBUG: Operation state after: %+v\n", state)
```

### Step 4: Run with Shorter Simulation
```bash
# Reduce blocks to isolate failure faster
go test -v -run TestFullAppSimulation \
  -NumBlocks=50 \
  -BlockSize=10 \
  -Seed=<failed_seed>
```

### Step 5: Check Invariants Manually
```bash
# Run just the invariant tests
cd ../invariants
go test -v -run Test<FailedModule>InvariantTestSuite
```

## Common Issues

### Non-Determinism
**Symptom**: Same seed produces different app hash
**Causes**:
- Map iteration (non-deterministic in Go)
- Timestamp usage in state
- Float arithmetic (use sdk.Dec instead)
- Concurrent operations without proper ordering

**Fix**: Use deterministic data structures and operations

### Invariant Violations
**Symptom**: Assertion failure during simulation
**Causes**:
- Logic bug in module keeper
- Missing validation in message handler
- Incorrect state updates
- Race conditions

**Fix**: Review the operation that triggered the violation

### Memory Leaks
**Symptom**: Memory usage grows unbounded
**Causes**:
- Not cleaning up old state
- Storing unbounded data in memory
- Cache without eviction

**Fix**: Implement proper state pruning and caching

### Slow Performance
**Symptom**: Simulation takes too long
**Causes**:
- Inefficient queries
- Too many invariant checks
- Large state size

**Fix**: Optimize hot paths, reduce invariant frequency for long sims

## Best Practices

1. **Run simulations before major releases**: Catch bugs early
2. **Test with multiple seeds**: Different seeds expose different edge cases
3. **Monitor memory usage**: Simulations should have bounded memory
4. **Check determinism**: Critical for consensus
5. **Review failed operations**: Understand what triggered failures
6. **Keep simulations fast**: Optimize for CI/CD pipeline
7. **Add new operations**: When adding features, add simulation ops

## Extending Simulations

### Adding New Operations

1. **Define the operation function** in `operations.go`:
```go
func SimulateMsgYourMessage(ak AccountKeeper, k YourKeeper) simtypes.Operation {
    return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
                accs []simtypes.Account, chainID string) (
        simtypes.OperationMsg, []simtypes.FutureOperation, error) {

        // Generate random parameters
        // Execute operation
        // Return result
    }
}
```

2. **Add to operations list** in `SimulationOperations()`:
```go
simulation.NewWeightedOperation(
    50, // weight
    SimulateMsgYourMessage(app.AccountKeeper, app.YourKeeper),
),
```

3. **Test the operation**:
```bash
go test -v -run TestFullAppSimulation -NumBlocks=100
```

### Adding New Parameters

1. **Add random generator** in `params.go`:
```go
func randomYourParam(r *rand.Rand) YourType {
    return YourType(simtypes.RandIntBetween(r, min, max))
}
```

2. **Update genesis generation**:
```go
yourGenesis := &yourtypes.GenesisState{
    Params: randomYourParams(r),
}
genesisState[yourtypes.ModuleName] = cdc.MustMarshalJSON(yourGenesis)
```

## Performance Optimization

### Reduce Invariant Checking Frequency
For long simulations (1000+ blocks), check invariants less frequently:
```bash
go test -run TestFullAppSimulation \
  -Period=10  # Check every 10 blocks instead of every block
```

### Disable Commit for Speed
Testing logic without DB writes:
```bash
go test -run TestFullAppSimulation -Commit=false
```

### Parallel Execution
Run multi-seed tests in parallel:
```bash
go test -run TestMultiSeedFullSimulation -parallel=4
```

## Related Documentation

- [Invariant Tests](../invariants/README.md)
- [Testing Guide](../../docs/guides/testing/RUN-TESTS-LOCALLY.md)
- [Cosmos SDK Simulation Guide](https://docs.cosmos.network/main/building-modules/simulator)
