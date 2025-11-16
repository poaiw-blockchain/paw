# Circuit Breaker Quick Reference

## What It Does

Automatically pauses trading on a pool when price changes too quickly, preventing:

- Market manipulation
- Cascading liquidations
- Flash loan attacks
- Extreme volatility exploitation

## Default Thresholds

| Time Window | Threshold | Description            |
| ----------- | --------- | ---------------------- |
| 1 minute    | 10%       | Fastest protection     |
| 5 minutes   | 20%       | Short-term protection  |
| 15 minutes  | 25%       | Medium-term protection |
| 1 hour      | 30%       | Long-term protection   |

**Cooldown**: 10 minutes (automatically resumes after)

## Quick Status Check

```bash
# Check if pool 1 is trading normally
pawd query dex circuit-breaker-status 1

# See all pools with circuit breakers tripped
pawd query dex active-circuit-breakers
```

## Response to Circuit Breaker Trip

### 1. Verify the Trip

```bash
pawd query dex circuit-breaker-state 1
```

### 2. Check Pool Details

```bash
pawd query dex pool 1
pawd query dex twap 1 --window=3600
```

### 3. Investigate Cause

- Check transaction history
- Review recent swaps
- Analyze price movements
- Look for manipulation patterns

### 4. Decide Action

**Option A: Wait for Automatic Resume** (10 minutes)

- No action needed
- Trading resumes automatically

**Option B: Governance Override** (immediate)

```bash
pawd tx gov submit-proposal circuit-breaker-resume \
  --pool-id=1 \
  --title="Resume Pool 1" \
  --description="Verified safe after investigation" \
  --deposit="10000000upaw" \
  --from=mykey
```

## Common Commands

### Query Commands

```bash
# Get configuration
pawd query dex circuit-breaker-config

# Get pool status
pawd query dex circuit-breaker-status <pool-id>

# Get pool state (detailed)
pawd query dex circuit-breaker-state <pool-id>

# Get all states
pawd query dex all-circuit-breaker-states

# Get only active (tripped) ones
pawd query dex active-circuit-breakers
```

### Governance Commands

```bash
# Update configuration
pawd tx gov submit-proposal circuit-breaker-config \
  --title="Update Thresholds" \
  --threshold-1min="0.15" \
  --threshold-5min="0.25" \
  --threshold-15min="0.35" \
  --threshold-1hour="0.50" \
  --cooldown-period=900 \
  --deposit="10000000upaw" \
  --from=mykey

# Resume trading (override)
pawd tx gov submit-proposal circuit-breaker-resume \
  --pool-id=1 \
  --title="Resume Trading" \
  --description="Safe to resume" \
  --deposit="10000000upaw" \
  --from=mykey
```

## Status Indicators

### Normal

- ✅ Trading: Unrestricted
- Max swap size: 100% of reserves
- No delays

### Gradual Resume

- ⚠️ Trading: Limited
- Max swap size: 50% of reserves (default)
- Duration: 1 hour after resume
- Then returns to normal

### Cooldown

- 🔒 Trading: PAUSED
- Duration: 10 minutes (default)
- Countdown displayed
- Auto-resumes after

### Active (Tripped)

- 🚨 Trading: PAUSED
- Governance override needed
- Investigate before resuming

## Event Monitoring

### Subscribe to Events

```bash
# All circuit breaker events
pawd query txs --events 'circuit_breaker_tripped.pool_id EXISTS'

# Specific pool
pawd query txs --events 'circuit_breaker_tripped.pool_id=1'

# Resume events
pawd query txs --events 'circuit_breaker_resumed.pool_id=1'
```

### Event Types

- `circuit_breaker_tripped` - Circuit breaker activated
- `circuit_breaker_resumed` - Trading resumed
- `circuit_breaker_config_updated` - Config changed

## Alert Script (5-Minute Setup)

```bash
#!/bin/bash
# save as: monitor-circuit-breakers.sh

WEBHOOK_URL="https://your-webhook-url"  # Slack/Discord/etc

while true; do
  ACTIVE=$(pawd query dex active-circuit-breakers --output json)
  COUNT=$(echo $ACTIVE | jq '.states | length')

  if [ "$COUNT" -gt 0 ]; then
    MESSAGE="🚨 $COUNT circuit breaker(s) active!"
    echo $ACTIVE | jq -r '.states[] | "Pool \(.pool_id): \(.trip_reason)"'

    # Send webhook
    curl -X POST $WEBHOOK_URL \
      -H 'Content-Type: application/json' \
      -d "{\"text\":\"$MESSAGE\"}"
  fi

  sleep 60  # Check every minute
done
```

Run with: `./monitor-circuit-breakers.sh &`

## Troubleshooting

### "Circuit breaker tripped" error when swapping

**Cause**: Price volatility exceeded threshold
**Solution**: Wait 10 minutes or propose governance override

### "Swap amount exceeds gradual resume limit"

**Cause**: Pool in gradual resume mode
**Solution**: Reduce swap size to <50% of reserves or wait 1 hour

### Circuit breaker not triggering despite volatility

**Cause**: Thresholds may be too high
**Solution**: Propose lower thresholds via governance

### False positives (too many triggers)

**Cause**: Thresholds may be too low
**Solution**: Propose higher thresholds via governance

## Security Notes

- ✅ Cannot be completely disabled
- ✅ Only governance can modify config
- ✅ All actions logged and emitted as events
- ✅ Pool-specific (doesn't affect other pools)
- ✅ Automatic resume after cooldown
- ✅ Race condition protection

## Performance Impact

- Gas overhead per swap: ~18,000 gas
- Storage per pool: ~300 bytes
- Query latency: <10ms

## Files Location

```
x/dex/keeper/
  ├── circuit_breaker.go          # Main implementation
  ├── circuit_breaker_gov.go      # Governance proposals
  ├── circuit_breaker_test.go     # Tests
  └── query_circuit_breaker.go    # Query handlers

x/dex/types/
  └── events.go                   # Event types

x/dex/
  ├── CIRCUIT_BREAKER.md          # Full documentation
  └── CIRCUIT_BREAKER_CLI.md      # CLI examples
```

## Key Functions

### For Developers

```go
// Check before swap
err := k.CheckCircuitBreaker(ctx, poolId)

// Check volume limit
err := k.CheckSwapVolumeLimit(ctx, poolId, amountIn)

// Get status
state := k.GetCircuitBreakerState(ctx, poolId)
isTripped := k.IsCircuitBreakerTripped(ctx, poolId)

// Manual operations
k.TripCircuitBreaker(ctx, poolId, reason, price)
err := k.ResumeTrading(ctx, poolId, isGovernanceOverride)
```

## Best Practices

1. **Monitor continuously** - Set up alerts for trips
2. **Investigate all trips** - Understand the cause
3. **Don't rush to override** - Wait for cooldown unless urgent
4. **Document decisions** - Record why you override
5. **Tune thresholds** - Adjust based on market conditions
6. **Test on testnet** - Before updating config on mainnet

## Support

- Full docs: `x/dex/CIRCUIT_BREAKER.md`
- CLI examples: `x/dex/CIRCUIT_BREAKER_CLI.md`
- Implementation: `CIRCUIT_BREAKER_IMPLEMENTATION.md`
- Code: `x/dex/keeper/circuit_breaker.go`
