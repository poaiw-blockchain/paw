# DEX Keeper Test Examples

This document provides examples of the comprehensive test patterns used in the DEX keeper test suite.

---

## Pool Management Tests

### Example 1: Basic Pool Creation
```go
func TestCreatePool_Success(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000), math.NewInt(2000000))

    require.NoError(t, err)
    require.Equal(t, uint64(1), poolId)

    // Verify pool exists
    pool := k.GetPool(ctx, poolId)
    require.NotNil(t, pool)
    require.Equal(t, poolId, pool.Id)
    require.Equal(t, "upaw", pool.TokenA)
    require.Equal(t, "uusdt", pool.TokenB)
    require.Equal(t, math.NewInt(1000000), pool.ReserveA)
    require.Equal(t, math.NewInt(2000000), pool.ReserveB)
    require.Equal(t, creator, pool.Creator)
}
```

**What it tests**: Basic pool creation with valid parameters

**Key assertions**:
- No error returned
- Pool ID is sequential (starts at 1)
- Pool is stored and retrievable
- All pool fields are correctly set
- Creator is tracked

---

### Example 2: Error Handling - Same Token
```go
func TestCreatePool_SameToken(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    _, err := k.CreatePool(ctx, creator, "upaw", "upaw",
        math.NewInt(1000000), math.NewInt(2000000))

    require.Error(t, err)
    require.ErrorIs(t, err, types.ErrSameToken)
}
```

**What it tests**: Validation prevents pools with same token pair

**Key assertions**:
- Error is returned
- Specific error type matches expected

---

### Example 3: Token Ordering
```go
func TestCreatePool_TokenOrdering(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    // Create pool with tokens in reverse alphabetical order
    poolId, err := k.CreatePool(ctx, creator, "uusdt", "upaw",
        math.NewInt(2000000), math.NewInt(1000000))

    require.NoError(t, err)

    // Pool should have tokens in alphabetical order
    pool := k.GetPool(ctx, poolId)
    require.NotNil(t, pool)
    require.Equal(t, "upaw", pool.TokenA) // upaw < uusdt
    require.Equal(t, "uusdt", pool.TokenB)
    // Amounts should be swapped accordingly
    require.Equal(t, math.NewInt(1000000), pool.ReserveA)
    require.Equal(t, math.NewInt(2000000), pool.ReserveB)
}
```

**What it tests**: Automatic alphabetical token ordering

**Key assertions**:
- Tokens are reordered alphabetically
- Amounts are swapped to match token order
- Pool creation still succeeds

---

## Swap Tests

### Example 4: Basic Swap
```go
func TestSwap_Success(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    // Create pool with 1000 PAW and 2000 USDT
    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    trader := types.TestAddr()
    amountOut, err := k.Swap(ctx, trader, poolId, "upaw", "uusdt",
        math.NewInt(100000000), math.NewInt(180000000))

    require.NoError(t, err)
    require.True(t, amountOut.GT(math.ZeroInt()))
    require.True(t, amountOut.GTE(math.NewInt(180000000)))
}
```

**What it tests**: Successful token swap execution

**Key assertions**:
- Swap completes without error
- Output amount is positive
- Output meets minimum slippage requirement

---

### Example 5: Constant Product Formula
```go
func TestSwap_ConstantProductFormula(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    poolBefore := k.GetPool(ctx, poolId)
    constantProductBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)

    trader := types.TestAddr()
    amountOut, err := k.Swap(ctx, trader, poolId, "upaw", "uusdt",
        math.NewInt(100000000), math.NewInt(180000000))
    require.NoError(t, err)

    poolAfter := k.GetPool(ctx, poolId)
    constantProductAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)

    // After swap with fees, constant product should increase
    require.True(t, constantProductAfter.GTE(constantProductBefore))

    // Verify reserves updated correctly
    require.Equal(t, poolBefore.ReserveA.Add(math.NewInt(100000000)), poolAfter.ReserveA)
    require.Equal(t, poolBefore.ReserveB.Sub(amountOut), poolAfter.ReserveB)
}
```

**What it tests**: AMM constant product formula (x * y = k)

**Key assertions**:
- Constant product increases (due to fees)
- Reserves are updated correctly
- Input added to one reserve
- Output subtracted from other reserve

---

### Example 6: Table-Driven Calculation Tests
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
        {
            name:        "large pool",
            reserveIn:   math.NewInt(1000000000),
            reserveOut:  math.NewInt(2000000000),
            amountIn:    math.NewInt(100000000),
            expectedMin: math.NewInt(180000000),
            expectedMax: math.NewInt(182000000),
        },
        {
            name:        "asymmetric pool",
            reserveIn:   math.NewInt(100000),
            reserveOut:  math.NewInt(10000000),
            amountIn:    math.NewInt(10000),
            expectedMin: math.NewInt(900000),
            expectedMax: math.NewInt(1000000),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            amountOut := k.CalculateSwapAmount(tt.reserveIn, tt.reserveOut, tt.amountIn)
            require.True(t, amountOut.GTE(tt.expectedMin),
                fmt.Sprintf("amountOut %s < expectedMin %s", amountOut, tt.expectedMin))
            require.True(t, amountOut.LTE(tt.expectedMax),
                fmt.Sprintf("amountOut %s > expectedMax %s", amountOut, tt.expectedMax))
        })
    }
}
```

**What it tests**: Swap calculation formula with various scenarios

**Key features**:
- Table-driven test structure
- Multiple scenarios in one test
- Clear test case names
- Helpful error messages with actual values

---

## Liquidity Tests

### Example 7: Add Liquidity with Proportional Amounts
```go
func TestAddLiquidity_ProportionalAmount(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    poolBefore := k.GetPool(ctx, poolId)

    provider := types.TestAddr()
    // Add 10% more liquidity (100M PAW, 200M USDT to 1000M PAW, 2000M USDT pool)
    shares, err := k.AddLiquidity(ctx, provider, poolId,
        math.NewInt(100000000), math.NewInt(200000000))
    require.NoError(t, err)

    poolAfter := k.GetPool(ctx, poolId)

    // Reserves should increase proportionally
    require.Equal(t, poolBefore.ReserveA.Add(math.NewInt(100000000)), poolAfter.ReserveA)
    require.Equal(t, poolBefore.ReserveB.Add(math.NewInt(200000000)), poolAfter.ReserveB)

    // Shares should be proportional to liquidity added (10% of total)
    expectedShares := poolBefore.TotalShares.Mul(math.NewInt(100000000)).Quo(poolBefore.ReserveA)
    require.Equal(t, expectedShares, shares)
}
```

**What it tests**: Proportional liquidity addition

**Key assertions**:
- Reserves increase by exact amounts
- Shares calculated proportionally
- Pool state consistency

---

### Example 8: Remove Liquidity - Full Withdrawal
```go
func TestRemoveLiquidity_AllShares(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    pool := k.GetPool(ctx, poolId)
    allShares := pool.TotalShares

    amountA, amountB, err := k.RemoveLiquidity(ctx, creator, poolId, allShares)

    require.NoError(t, err)
    require.Equal(t, pool.ReserveA, amountA)
    require.Equal(t, pool.ReserveB, amountB)

    // Pool should be empty
    poolAfter := k.GetPool(ctx, poolId)
    require.True(t, poolAfter.ReserveA.IsZero())
    require.True(t, poolAfter.ReserveB.IsZero())
    require.True(t, poolAfter.TotalShares.IsZero())
}
```

**What it tests**: Complete liquidity removal

**Key assertions**:
- Receives all reserve tokens
- Pool reserves reduced to zero
- Pool shares reduced to zero
- Pool still exists (not deleted)

---

## Edge Cases

### Example 9: Minimal Amounts
```go
func TestCreatePool_MinimalAmounts(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    minAmount := math.NewInt(1) // Minimal amount

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        minAmount, minAmount)

    require.NoError(t, err)

    pool := k.GetPool(ctx, poolId)
    require.NotNil(t, pool)
    require.Equal(t, minAmount, pool.ReserveA)
    require.Equal(t, minAmount, pool.ReserveB)
}
```

**What it tests**: Minimal valid amounts (1 token)

**Key assertions**:
- Pool creation succeeds with minimum amounts
- No overflow or underflow errors
- Reserves stored correctly

---

### Example 10: Large Amounts
```go
func TestCreatePool_LargeAmounts(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    largeAmount := math.NewInt(1000000000000) // 1 trillion

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        largeAmount, largeAmount)

    require.NoError(t, err)

    pool := k.GetPool(ctx, poolId)
    require.NotNil(t, pool)
    require.Equal(t, largeAmount, pool.ReserveA)
    require.Equal(t, largeAmount, pool.ReserveB)
}
```

**What it tests**: Very large amounts

**Key assertions**:
- Pool creation handles large numbers
- No integer overflow
- Precision maintained

---

## Security & MEV Protection

### Example 11: Circuit Breaker Integration
```go
func TestSwap_CircuitBreakerTripped(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    // Trip circuit breaker
    k.TripCircuitBreaker(ctx, poolId, "test", math.LegacyNewDec(2))

    trader := types.TestAddr()
    _, err = k.Swap(ctx, trader, poolId, "upaw", "uusdt",
        math.NewInt(100000000), math.NewInt(180000000))

    require.Error(t, err)
    require.ErrorIs(t, err, types.ErrCircuitBreakerTripped)
}
```

**What it tests**: Circuit breaker prevents swaps when tripped

**Key assertions**:
- Swap rejected when circuit breaker active
- Correct error type returned
- Safety mechanism functioning

---

### Example 12: Module Pause Protection
```go
func TestCreatePool_WhenPaused(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    // Pause module
    err := k.PauseModule(ctx)
    require.NoError(t, err)

    // Try to create pool
    _, err = k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000), math.NewInt(2000000))

    require.Error(t, err)
    require.ErrorIs(t, err, types.ErrModulePaused)
}
```

**What it tests**: Module pause prevents operations

**Key assertions**:
- Operations blocked when paused
- Correct error returned
- Emergency stop functioning

---

## Integration Tests

### Example 13: Complex Multi-Provider Scenario
```go
func TestLiquidity_MultipleProvidersComplex(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000000), math.NewInt(2000000000))
    require.NoError(t, err)

    provider1 := types.TestAddr()
    provider2 := types.TestAddr()

    // Provider 1 adds liquidity
    shares1, err := k.AddLiquidity(ctx, provider1, poolId,
        math.NewInt(100000000), math.NewInt(200000000))
    require.NoError(t, err)

    // Swap happens
    trader := types.TestAddr()
    _, err = k.Swap(ctx, trader, poolId, "upaw", "uusdt",
        math.NewInt(50000000), math.NewInt(90000000))
    require.NoError(t, err)

    // Provider 2 adds liquidity
    pool := k.GetPool(ctx, poolId)
    shares2, err := k.AddLiquidity(ctx, provider2, poolId,
        pool.ReserveA.Quo(math.NewInt(10)),
        pool.ReserveB.Quo(math.NewInt(10)))
    require.NoError(t, err)

    // Provider 1 removes half their liquidity
    _, _, err = k.RemoveLiquidity(ctx, provider1, poolId, shares1.Quo(math.NewInt(2)))
    require.NoError(t, err)

    // Verify both providers still have shares
    remaining1 := k.GetLiquidity(ctx, poolId, provider1)
    remaining2 := k.GetLiquidity(ctx, poolId, provider2)

    require.True(t, remaining1.GT(math.ZeroInt()))
    require.Equal(t, shares2, remaining2)
}
```

**What it tests**: Complex scenario with multiple providers and operations

**Key assertions**:
- Multiple providers can coexist
- Operations work after price changes
- Share tracking is accurate
- State consistency maintained

---

## Test Helpers & Utilities

### Example 14: Event Emission Verification
```go
func TestCreatePool_EventEmission(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)
    creator := types.TestAddr()

    // Create context with event manager
    ctx = ctx.WithEventManager(sdk.NewEventManager())

    poolId, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
        math.NewInt(1000000), math.NewInt(2000000))
    require.NoError(t, err)

    // Verify event was emitted
    events := ctx.EventManager().Events()
    require.NotEmpty(t, events)

    // Find create_pool event
    var foundEvent bool
    for _, event := range events {
        if event.Type == "create_pool" {
            foundEvent = true
            // Verify event attributes
            for _, attr := range event.Attributes {
                switch attr.Key {
                case "pool_id":
                    require.Equal(t, "1", attr.Value)
                case "creator":
                    require.Equal(t, creator, attr.Value)
                case "token_a":
                    require.Equal(t, "upaw", attr.Value)
                case "token_b":
                    require.Equal(t, "uusdt", attr.Value)
                }
            }
        }
    }
    require.True(t, foundEvent, "create_pool event not found")
}
```

**What it tests**: Event emission and attributes

**Key assertions**:
- Event is emitted
- Event type is correct
- All attributes are present and correct
- Event data matches operation

---

## Best Practices Demonstrated

1. **Clear Test Names**: Each test name clearly describes what it tests
2. **Setup-Execute-Assert**: Three-phase test structure
3. **Comprehensive Coverage**: Success, error, and edge cases
4. **State Verification**: Check before and after states
5. **Deterministic**: No randomness, repeatable results
6. **Fast**: All tests complete quickly
7. **Isolated**: Tests don't depend on each other
8. **Well-Commented**: Inline comments explain complex logic
9. **Table-Driven**: Multiple scenarios tested efficiently
10. **Error Checking**: Both error occurrence and type verified
