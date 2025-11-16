# DEX Circuit Breaker System

## Overview

The circuit breaker system protects the PAW DEX from price volatility, market manipulation, and cascading liquidations by automatically pausing trading when excessive price movements are detected.

## Features

### 1. Multi-Timeframe Volatility Detection

The circuit breaker monitors price changes across multiple time windows:

- **1 minute**: 10% threshold (default)
- **5 minutes**: 20% threshold (default)
- **15 minutes**: 25% threshold (default)
- **1 hour**: 30% threshold (default)

When price changes exceed any threshold, the circuit breaker automatically trips.

### 2. Automatic Trading Pause

When triggered, the circuit breaker:

- Immediately prevents all swap operations on the affected pool
- Records the trigger reason, time, and price
- Emits critical monitoring events
- Logs detailed information for analysis

### 3. Cooldown Period

After tripping, trading remains paused for a configurable cooldown period:

- Default: 10 minutes (600 seconds)
- Configurable via governance (1 second to 24 hours)
- Allows market conditions to stabilize

### 4. Gradual Resume Mechanism

When trading resumes (either after cooldown or via governance override):

- Optional volume limits on initial trades
- Default: 50% of pool reserves maximum swap size
- Prevents immediate large swaps that could re-trigger volatility
- Automatically lifts after 1 hour of stable trading

### 5. Governance Override

Governance can:

- Update circuit breaker configuration (thresholds, cooldown period)
- Override circuit breaker and resume trading early
- Enable/disable gradual resume feature
- Adjust volume limits for gradual resume

## Architecture

### Core Files

1. **circuit_breaker.go** - Main circuit breaker implementation
   - Volatility detection logic
   - Trip/resume functionality
   - Configuration management
   - Volume limit enforcement

2. **circuit_breaker_gov.go** - Governance integration
   - Proposal handlers for config updates
   - Proposal handlers for manual resume
   - Validation logic

3. **circuit_breaker_test.go** - Comprehensive test suite
   - Config validation tests
   - Trip/resume tests
   - Volatility detection tests
   - Volume limit tests

### Data Structures

#### CircuitBreakerConfig

```go
type CircuitBreakerConfig struct {
    Threshold1Min       math.LegacyDec  // 1-minute price change threshold
    Threshold5Min       math.LegacyDec  // 5-minute price change threshold
    Threshold15Min      math.LegacyDec  // 15-minute price change threshold
    Threshold1Hour      math.LegacyDec  // 1-hour price change threshold
    CooldownPeriod      int64           // Cooldown in seconds
    EnableGradualResume bool            // Enable volume limits on resume
    ResumeVolumeFactor  math.LegacyDec  // Max swap size as % of reserves
}
```

#### CircuitBreakerState

```go
type CircuitBreakerState struct {
    PoolId          uint64          // Pool identifier
    IsTripped       bool            // Whether circuit breaker is active
    TripReason      string          // Why it was tripped
    TrippedAt       int64           // Block height when tripped
    TrippedAtTime   int64           // Unix timestamp when tripped
    PriceAtTrip     math.LegacyDec  // Price when triggered
    CanResumeAt     int64           // When trading can resume
    GradualResume   bool            // Whether in gradual resume mode
    ResumeStartedAt int64           // When gradual resume started
}
```

## Integration Points

### 1. Swap Function Integration

The circuit breaker is integrated into the Swap function in `keeper.go`:

```go
func (k Keeper) Swap(...) (math.Int, error) {
    // Check if module is paused
    if err := k.RequireNotPaused(ctx); err != nil {
        return math.ZeroInt(), err
    }

    // Check circuit breaker before swap
    if err := k.CheckCircuitBreaker(ctx, poolId); err != nil {
        return math.ZeroInt(), err
    }

    // Check swap volume limits (for gradual resume)
    if err := k.CheckSwapVolumeLimit(ctx, poolId, amountIn); err != nil {
        return math.ZeroInt(), err
    }

    // ... rest of swap logic
}
```

### 2. TWAP Integration

Circuit breaker uses existing TWAP (Time-Weighted Average Price) infrastructure:

- Leverages price observations from `twap.go`
- Calculates price changes over time windows
- Detects abnormal volatility patterns

### 3. Pause Mechanism Integration

Circuit breaker complements the existing pause mechanism:

- Module-level pause: Pauses ALL pools
- Circuit breaker: Pauses SPECIFIC pools based on volatility

## Security Features

### 1. Cannot Be Disabled

- Circuit breaker cannot be completely disabled
- Only governance can update configuration
- Minimum thresholds enforced in validation
- Maximum cooldown period enforced (24 hours)

### 2. Event Emission

Critical events are emitted for monitoring:

```go
// Circuit breaker tripped
EventTypeCircuitBreakerTripped = "circuit_breaker_tripped"
// Attributes: pool_id, reason, tripped_at_height, price_at_trip, severity

// Circuit breaker resumed
EventTypeCircuitBreakerResumed = "circuit_breaker_resumed"
// Attributes: pool_id, resumed_at_height, governance_override
```

### 3. Comprehensive Logging

All circuit breaker operations are logged:

- Trip events (ERROR level)
- Resume events (INFO level)
- Config updates (INFO level)
- Detection failures (ERROR level)

### 4. Race Condition Prevention

The circuit breaker is designed to prevent race conditions:

- Atomic state updates
- Check-before-swap pattern
- Consistent error handling
- Stateless validation functions

## Usage Examples

### 1. Query Circuit Breaker State

```go
state := keeper.GetCircuitBreakerState(ctx, poolId)
if state.IsTripped {
    fmt.Printf("Circuit breaker active: %s\n", state.TripReason)
    fmt.Printf("Can resume at: %d\n", state.CanResumeAt)
}
```

### 2. Update Configuration via Governance

```go
proposal := CircuitBreakerProposal{
    Title:       "Update Circuit Breaker Thresholds",
    Description: "Increase thresholds for more volatile market conditions",
    Config: CircuitBreakerConfig{
        Threshold1Min:       math.LegacyNewDecWithPrec(15, 2), // 15%
        Threshold5Min:       math.LegacyNewDecWithPrec(25, 2), // 25%
        Threshold15Min:      math.LegacyNewDecWithPrec(35, 2), // 35%
        Threshold1Hour:      math.LegacyNewDecWithPrec(50, 2), // 50%
        CooldownPeriod:      900,                              // 15 minutes
        EnableGradualResume: true,
        ResumeVolumeFactor:  math.LegacyNewDecWithPrec(3, 1), // 30%
    },
}
```

### 3. Manual Resume via Governance

```go
proposal := CircuitBreakerResumeProposal{
    Title:       "Resume Trading on Pool 1",
    Description: "Market conditions have stabilized, safe to resume",
    PoolId:      1,
}
```

### 4. Monitoring Circuit Breaker States

```go
allStates := keeper.GetAllCircuitBreakerStates(ctx)
for _, state := range allStates {
    if state.IsTripped {
        // Alert monitoring system
        alert("Circuit breaker active", state.PoolId, state.TripReason)
    }
}
```

## Best Practices

### 1. Monitoring

Set up monitoring for circuit breaker events:

- Alert on `circuit_breaker_tripped` events
- Track frequency of trips per pool
- Monitor price volatility trends
- Set up dashboards for circuit breaker states

### 2. Response Procedures

When circuit breaker trips:

1. Investigate the cause (market manipulation? legitimate volatility?)
2. Analyze on-chain data around the trigger time
3. Check if other pools are affected
4. Determine if governance intervention is needed
5. Consider adjusting thresholds if trips are too frequent

### 3. Configuration Tuning

Adjust thresholds based on:

- Market conditions (bull/bear markets)
- Pool maturity (new pools may be more volatile)
- Asset characteristics (stablecoins vs volatile tokens)
- Historical volatility data

### 4. Testing

Before deploying config changes:

- Test on testnet with realistic scenarios
- Simulate various market conditions
- Verify gradual resume mechanics
- Ensure monitoring systems are connected

## Governance Considerations

### Updating Thresholds

Consider these factors when proposing threshold updates:

- Historical volatility of pools
- Market conditions and trends
- User feedback on false positives
- Security vs usability tradeoff

### Emergency Resume

Use governance override carefully:

- Only when confident market has stabilized
- After investigating the trigger cause
- With community consensus
- Document the decision rationale

## Performance Considerations

### Gas Costs

Circuit breaker adds minimal gas overhead:

- Single state check per swap (~5k gas)
- Price observation lookup (~10k gas)
- Total overhead: ~15k gas per swap

### Storage

Circuit breaker storage is efficient:

- One config per module (~500 bytes)
- One state per pool (~300 bytes)
- Reuses existing TWAP price observations

## Error Codes

- `ErrCircuitBreakerTripped` (18): Circuit breaker is active for this pool
- `ErrInvalidParams` (15): Invalid circuit breaker configuration
- `ErrInvalidAmount` (7): Swap amount exceeds gradual resume limit

## Future Enhancements

Potential improvements for future versions:

1. **Dynamic Thresholds**: Adjust thresholds based on realized volatility
2. **Cross-Pool Detection**: Trip circuit breaker if multiple pools show volatility
3. **Predictive Triggers**: Use ML to predict and prevent manipulation
4. **Partial Limits**: Reduce allowed swap size instead of full pause
5. **Time-Based Recovery**: Gradually increase volume limits over time
6. **Pool-Specific Config**: Different thresholds for different pool types

## References

- Time-Weighted Average Price (TWAP): `twap.go`
- Pause Mechanism: `pause.go`
- Flash Loan Detection: `flashloan.go`
- DEX Keeper: `keeper.go`
