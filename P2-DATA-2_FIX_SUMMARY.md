# P2-DATA-2: Circuit Breaker State Preservation Fix

## Problem
Circuit breaker pause state was reset during chain upgrades, causing paused pools to become immediately unpaused.

## Root Cause
Genesis export/import intentionally cleared runtime state (PausedUntil, TriggeredBy, TriggerReason) to prevent transient states carrying over.

## Solution
Added governance-controlled `UpgradePreserveCircuitBreakerState` parameter (default: true).

### Behavior
- **true (default)**: Full pause state preserved across upgrades
- **false**: Only persistent config exported, runtime cleared (intentional reset)

### Implementation
1. **Proto**: Added bool param at field 12 in Params message
2. **Export**: Conditionally exports PausedUntil/TriggeredBy/TriggerReason based on param
3. **Import**: Conditionally restores time.Unix(PausedUntil) based on param
4. **Tests**: 4 comprehensive tests covering enabled/disabled/expired/multiple pools

### Edge Cases
- Expired pauses preserve timestamp but don't block operations
- Zero time values handled correctly (not exported)
- Multiple pools with different states export/import correctly

## Files Modified
- `proto/paw/dex/v1/dex.proto` (new param)
- `x/dex/keeper/params.go` (default value)
- `x/dex/keeper/genesis.go` (conditional logic)
- `x/dex/keeper/genesis_test.go` (4 new tests)

## Verification
Types build successfully. Tests added but require fixing compilation errors in commit-reveal implementation to run.
