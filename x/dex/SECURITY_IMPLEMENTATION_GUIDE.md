# DEX Security Implementation Guide

**For Developers and Auditors**

This guide explains how to use the secure DEX implementation and understand its security features.

---

## Quick Start

### Using Secure Operations

All message handlers now automatically use the secure implementations:

```go
// Creating a pool (automatically uses CreatePoolSecure)
msg := &types.MsgCreatePool{
    Creator: "paw1...",
    TokenA:  "atom",
    TokenB:  "usdc",
    AmountA: math.NewInt(1000000),
    AmountB: math.NewInt(1000000),
}
response, err := msgServer.CreatePool(ctx, msg)

// Adding liquidity (automatically uses AddLiquiditySecure)
msg := &types.MsgAddLiquidity{
    Provider: "paw1...",
    PoolId:   1,
    AmountA:  math.NewInt(100000),
    AmountB:  math.NewInt(100000),
}
response, err := msgServer.AddLiquidity(ctx, msg)

// Executing swap (automatically uses ExecuteSwapSecure)
msg := &types.MsgSwap{
    Trader:       "paw1...",
    PoolId:       1,
    TokenIn:      "atom",
    TokenOut:     "usdc",
    AmountIn:     math.NewInt(1000),
    MinAmountOut: math.NewInt(950), // 5% slippage tolerance
}
response, err := msgServer.Swap(ctx, msg)
```

---

## Security Features Explained

### 1. Reentrancy Protection

**What it prevents**: Attackers calling back into contract during execution

**How it works**:
```go
// All operations use WithReentrancyGuard
err := k.WithReentrancyGuard(ctx, poolID, "swap", func() error {
    return k.executeSwapInternal(...)
})
```

**Guard mechanism**:
- Stores lock state in context
- Prevents nested calls with same key
- Automatically releases on completion
- Separate locks per pool per operation

**Developer notes**:
- Don't call secure functions within secure functions
- Use internal implementations if needed
- Guard is per-context, safe for parallel execution

---

### 2. SafeMath Operations

**All arithmetic must use SafeMath**:

```go
// Addition with overflow check
result, err := keeper.SafeAdd(a, b)
if err != nil {
    return err // Overflow detected
}

// Subtraction with underflow check
result, err := keeper.SafeSub(a, b)
if err != nil {
    return err // Underflow detected
}

// Multiplication with overflow check
result, err := keeper.SafeMul(a, b)
if err != nil {
    return err // Overflow detected
}

// Division with zero check
result, err := keeper.SafeQuo(a, b)
if err != nil {
    return err // Division by zero
}
```

**Why it matters**:
- Prevents integer overflow attacks
- Catches underflow before corruption
- Ensures mathematical integrity
- Returns clear error messages

**When to use**:
- ANY pool reserve update
- ANY share calculation
- ANY fee calculation
- ANY ratio computation

---

### 3. Flash Loan Protection

**What it prevents**: Same-block liquidity manipulation

**How it works**:
```go
// Automatically checked in RemoveLiquiditySecure
err := k.CheckFlashLoanProtection(ctx, poolID, provider)
if err != nil {
    return err // Must wait MinLPLockBlocks
}
```

**Mechanism**:
- Records block height on AddLiquidity
- Records block height on RemoveLiquidity
- Requires MinLPLockBlocks (currently 1) between operations
- Prevents same-block add/remove

**Developer notes**:
- Applies to liquidity operations only
- Does NOT affect swaps (intended)
- Tracked per user per pool
- Cannot be bypassed

---

### 4. Circuit Breaker System

**Automatic Triggering**:

The circuit breaker automatically triggers when:
- Price deviation > 20% in single operation
- Anomalous conditions detected

```go
// Checked before every operation
err := k.CheckCircuitBreaker(ctx, pool, "swap")
if err != nil {
    return err // Pool paused
}
```

**Manual Control (Governance)**:

```go
// Emergency pause
err := k.EmergencyPausePool(ctx, poolID, "security incident", 24*time.Hour)

// Unpause
err := k.UnpausePool(ctx, poolID)
```

**State Management**:
```go
type CircuitBreakerState struct {
    Enabled       bool           // Is circuit breaker active?
    PausedUntil   time.Time      // When does pause expire?
    LastPrice     math.LegacyDec // Last known price
    TriggeredBy   string         // "automatic" or "governance"
    TriggerReason string         // Description
}
```

**Events emitted**:
```go
"circuit_breaker_triggered" - When automatically triggered
"pool_emergency_paused"     - When governance pauses
"pool_unpaused"             - When unpaused
```

---

### 5. MEV Protection

**Swap Size Limits**:

Maximum swap size is 10% of reserve:
```go
const MaxSwapSizePercent = "0.1"

// Automatically validated
err := k.ValidateSwapSize(amountIn, reserveIn)
```

**Price Impact Limits**:

Maximum price impact is 50%:
```go
err := k.ValidatePriceImpact(amountIn, reserveIn, reserveOut, amountOut)
```

**Slippage Protection**:

Users MUST specify minimum output:
```go
msg := &types.MsgSwap{
    AmountIn:     math.NewInt(1000),
    MinAmountOut: math.NewInt(950), // Required!
}
```

**Why this works**:
- Large MEV swaps are rejected
- Price impact is limited
- Users protected by slippage
- Sandwich attacks less profitable

---

### 6. Invariant Validation

**Constant Product Check**:

After every operation:
```go
oldK := pool.ReserveA.Mul(pool.ReserveB)

// ... perform operation ...

err := k.ValidatePoolInvariant(ctx, pool, oldK)
```

**What it checks**:
- k = x * y must not decrease
- k can increase (fees)
- Prevents pool corruption

**Pool State Validation**:

```go
err := k.ValidatePoolState(pool)
```

**Checks**:
- Reserves non-negative
- Shares non-negative
- If reserves, must have shares
- If shares, must have reserves

---

## Security Patterns

### Checks-Effects-Interactions

**ALWAYS follow this order**:

```go
func SecureOperation(...) error {
    // 1. CHECKS - Validate all inputs
    if amountIn.IsZero() {
        return ErrInvalidInput
    }

    // 2. CHECKS - Validate state
    pool, err := k.GetPool(ctx, poolID)
    if err != nil {
        return err
    }

    // 3. CHECKS - Security validations
    if err := k.CheckCircuitBreaker(ctx, pool, "operation"); err != nil {
        return err
    }

    // 4. EFFECTS - Record old state
    oldK := pool.ReserveA.Mul(pool.ReserveB)

    // 5. INTERACTIONS - External calls (receives)
    err = k.bankKeeper.SendCoins(ctx, user, module, coins)
    if err != nil {
        return err
    }

    // 6. EFFECTS - Update state
    pool.ReserveA, err = SafeAdd(pool.ReserveA, amountIn)
    if err != nil {
        return err
    }

    // 7. EFFECTS - Validate invariants
    if err := k.ValidatePoolInvariant(ctx, pool, oldK); err != nil {
        return err
    }

    // 8. EFFECTS - Save state
    if err := k.SetPool(ctx, pool); err != nil {
        return err
    }

    // 9. INTERACTIONS - External calls (sends)
    err = k.bankKeeper.SendCoins(ctx, module, user, coins)
    if err != nil {
        return err
    }

    // 10. EFFECTS - Emit events
    ctx.EventManager().EmitEvent(...)

    return nil
}
```

---

## Error Handling

### Security Error Types

```go
// Reentrancy
types.ErrReentrancy              // Nested call detected

// Math errors
types.ErrOverflow                // Arithmetic overflow
types.ErrUnderflow               // Arithmetic underflow
types.ErrDivisionByZero          // Divide by zero

// State errors
types.ErrInvalidPoolState        // Pool corrupted
types.ErrInvariantViolation      // k=x*y violated

// Attack prevention
types.ErrFlashLoanDetected       // Same-block manipulation
types.ErrReentrancy              // Reentrancy attempt
types.ErrCircuitBreakerTriggered // Pool paused

// MEV protection
types.ErrSwapTooLarge            // Swap > 10% reserve
types.ErrPriceImpactTooHigh      // Price impact > 50%
types.ErrSlippageTooHigh         // Output < minimum

// General
types.ErrInvalidInput            // Bad parameters
types.ErrInsufficientLiquidity   // Not enough liquidity
types.ErrPoolNotFound            // Pool doesn't exist
types.ErrInsufficientShares      // Not enough LP shares
```

### Error Handling Pattern

```go
if err != nil {
    // Log with context
    logger.Error("operation failed",
        "pool_id", poolID,
        "user", user,
        "error", err,
    )

    // Return with wrapped error
    return types.ErrOperationFailed.Wrapf(
        "failed for pool %d: %w",
        poolID, err,
    )
}
```

---

## Testing Security Features

### Unit Tests

```go
func TestSecureOperation(t *testing.T) {
    // Setup
    keeper, ctx := setupTest()

    // Test success case
    result, err := keeper.SecureOperation(ctx, validParams)
    require.NoError(t, err)
    require.NotNil(t, result)

    // Test reentrancy protection
    _, err = keeper.SecureOperation(ctx, validParams)
    require.Error(t, err)
    require.Contains(t, err.Error(), "reentrancy")

    // Test overflow protection
    _, err = keeper.SecureOperation(ctx, overflowParams)
    require.Error(t, err)
    require.Contains(t, err.Error(), "overflow")

    // Test invariant validation
    _, err = keeper.SecureOperation(ctx, invalidParams)
    require.Error(t, err)
    require.Contains(t, err.Error(), "invariant")
}
```

### Attack Simulation Tests

```go
func TestFlashLoanAttack(t *testing.T) {
    keeper, ctx := setupTest()

    // Add liquidity
    shares, err := keeper.AddLiquiditySecure(ctx, user, poolID, amount1, amount2)
    require.NoError(t, err)

    // Attempt immediate removal (should fail)
    _, _, err = keeper.RemoveLiquiditySecure(ctx, user, poolID, shares)
    require.Error(t, err)
    require.Contains(t, err.Error(), "flash loan")
}

func TestReentrancyAttack(t *testing.T) {
    guard := keeper.NewReentrancyGuard()

    // Lock
    err := guard.Lock("test")
    require.NoError(t, err)

    // Attempt nested lock (should fail)
    err = guard.Lock("test")
    require.Error(t, err)
}
```

---

## Security Configuration

### Tunable Parameters

```go
// In keeper/security.go
const (
    MaxPriceDeviation  = "0.2"   // 20% price deviation triggers circuit breaker
    MaxSwapSizePercent = "0.1"   // 10% of reserve maximum swap size
    MinLPLockBlocks    = int64(1) // 1 block minimum between liquidity ops
    MaxPools           = uint64(1000) // Maximum number of pools
    PriceUpdateTolerance = "0.001" // 0.1% price update tolerance
)
```

### Adjusting Security Parameters

**To change swap size limit**:
```go
// Modify MaxSwapSizePercent in security.go
// Requires code change + deployment
```

**To change circuit breaker sensitivity**:
```go
// Modify MaxPriceDeviation in security.go
// More sensitive = lower value (e.g., "0.1" for 10%)
// Less sensitive = higher value (e.g., "0.3" for 30%)
```

**To change flash loan protection**:
```go
// Modify MinLPLockBlocks in security.go
// Higher = more protection, less UX
// Lower = less protection, better UX
// Minimum recommended: 1 block
```

---

## Monitoring and Alerts

### Events to Monitor

```go
// Security events
"circuit_breaker_triggered" - Pool automatically paused
"pool_emergency_paused"     - Governance paused pool
"flash_loan_detected"       - Flash loan attempt blocked

// Operation events
"swap_executed"    - Track swap volumes
"liquidity_added"  - Track liquidity changes
"liquidity_removed" - Track liquidity exits

// State changes
"pool_created"     - New pool created
```

### Metrics to Track

1. **Circuit Breaker Triggers**
   - Frequency per pool
   - Trigger reasons
   - Duration of pauses

2. **Rejected Swaps**
   - Size limit violations
   - Price impact violations
   - Slippage violations

3. **Flash Loan Attempts**
   - Frequency
   - Affected pools
   - User patterns

4. **Pool Invariants**
   - k values over time
   - Deviation from expected
   - Correlation with events

### Alert Thresholds

```go
// Alert if:
- Circuit breaker triggers > 5 times/hour
- Same pool paused > 3 times/day
- Flash loan attempts > 10/day from same user
- Swap rejections > 50% of attempts
- Any invariant violation (CRITICAL)
```

---

## Governance Operations

### Emergency Pause

```go
// Proposal to pause pool
proposal := GovProposal{
    Title: "Emergency Pause Pool 42",
    Description: "Anomalous activity detected",
    Messages: []sdk.Msg{
        &MsgEmergencyPausePool{
            PoolId:   42,
            Reason:   "suspicious price manipulation",
            Duration: 24 * time.Hour,
        },
    },
}
```

### Unpause Pool

```go
proposal := GovProposal{
    Title: "Unpause Pool 42",
    Description: "Issue resolved",
    Messages: []sdk.Msg{
        &MsgUnpausePool{
            PoolId: 42,
        },
    },
}
```

---

## Best Practices for Integrators

### 1. Always Set Slippage

```go
// BAD - No slippage protection
msg := &types.MsgSwap{
    AmountIn:     amount,
    MinAmountOut: math.ZeroInt(), // DON'T DO THIS
}

// GOOD - Reasonable slippage tolerance
expectedOut := CalculateExpectedOutput(amount)
minOut := expectedOut.Mul(95).Quo(100) // 5% slippage
msg := &types.MsgSwap{
    AmountIn:     amount,
    MinAmountOut: minOut, // ALWAYS SET THIS
}
```

### 2. Handle Circuit Breaker

```go
amountOut, err := keeper.ExecuteSwapSecure(ctx, trader, poolID, ...)
if err != nil {
    if errors.Is(err, types.ErrCircuitBreakerTriggered) {
        // Pool is paused - inform user
        return fmt.Errorf("pool temporarily paused for security: %w", err)
    }
    return err
}
```

### 3. Check Pool State First

```go
// Validate pool before building transaction
pool, err := keeper.GetPoolSecure(ctx, poolID)
if err != nil {
    return err // Pool not found or invalid
}

// Check circuit breaker
state, _ := keeper.GetCircuitBreakerState(ctx, poolID)
if state.Enabled && time.Now().Before(state.PausedUntil) {
    return fmt.Errorf("pool paused until %s", state.PausedUntil)
}
```

### 4. Simulate Before Execute

```go
// Simulate to get expected output
simulatedOut, err := keeper.SimulateSwapSecure(ctx, poolID, tokenIn, tokenOut, amountIn)
if err != nil {
    return err // Swap would fail
}

// Set slippage based on simulation
minOut := simulatedOut.Mul(95).Quo(100)

// Execute actual swap
actualOut, err := keeper.ExecuteSwapSecure(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minOut)
```

---

## Upgrade Path

### From Old to New Implementation

```go
// OLD CODE
pool, err := keeper.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)

// NEW CODE (automatic via msg_server)
pool, err := keeper.CreatePoolSecure(ctx, creator, tokenA, tokenB, amountA, amountB)

// OLD CODE
shares, err := keeper.AddLiquidity(ctx, provider, poolID, amountA, amountB)

// NEW CODE (automatic via msg_server)
shares, err := keeper.AddLiquiditySecure(ctx, provider, poolID, amountA, amountB)

// OLD CODE
amountOut, err := keeper.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minOut)

// NEW CODE (automatic via msg_server)
amountOut, err := keeper.ExecuteSwapSecure(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minOut)
```

### Migration Checklist

- [ ] Update all keeper method calls to *Secure versions
- [ ] Add slippage protection to all swaps
- [ ] Handle circuit breaker errors
- [ ] Update error handling for new error types
- [ ] Add monitoring for security events
- [ ] Test flash loan protection
- [ ] Verify reentrancy protection
- [ ] Load test with security features enabled

---

## Security Checklist

### Before Production

- [ ] All operations use Secure versions
- [ ] All math uses SafeMath
- [ ] All swaps have slippage protection
- [ ] Circuit breaker monitoring configured
- [ ] Flash loan protection tested
- [ ] Reentrancy tests passing
- [ ] Invariant checks validated
- [ ] External audit completed
- [ ] Bug bounty program active
- [ ] Incident response plan ready
- [ ] Governance controls tested
- [ ] Emergency pause procedures documented

### During Operation

- [ ] Monitor circuit breaker triggers
- [ ] Track rejected transactions
- [ ] Verify pool invariants
- [ ] Check for flash loan attempts
- [ ] Review large swaps
- [ ] Validate price movements
- [ ] Track error rates
- [ ] Monitor gas usage

---

## Support and Resources

### Documentation

- **Security Audit Report**: `/x/dex/SECURITY_AUDIT_REPORT.md`
- **Implementation Guide**: `/x/dex/SECURITY_IMPLEMENTATION_GUIDE.md` (this file)
- **Test Suite**: `/x/dex/keeper/security_test.go`

### Code References

- **Security Infrastructure**: `/x/dex/keeper/security.go`
- **Secure Swaps**: `/x/dex/keeper/swap_secure.go`
- **Secure Liquidity**: `/x/dex/keeper/liquidity_secure.go`
- **Secure Pools**: `/x/dex/keeper/pool_secure.go`
- **Error Definitions**: `/x/dex/types/errors.go`

### Getting Help

For security concerns:
1. Review this implementation guide
2. Check the security audit report
3. Review the test suite for examples
4. Contact security team

---

**Last Updated**: 2025-11-24
**Version**: 1.0 (Production-Ready)
**Security Standard**: Enterprise-Grade
