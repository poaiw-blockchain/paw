# Two-Phase Commit Fix for Escrow Module (P1-DATA-3)

## Problem
Bank transfers could succeed while state updates failed, creating catastrophic inconsistent state between modules in escrow operations.

## Solution
Implemented two-phase commit pattern using `CacheContext` to ensure atomicity.

### Changes Made

#### 1. `x/compute/keeper/escrow.go`
- **LockEscrow**: Wrapped bank transfer, state storage, and timeout index creation in CacheContext
- **ReleaseEscrow**: Wrapped bank transfer and state update in CacheContext
- **RefundEscrow**: Wrapped bank transfer and state update in CacheContext

**Pattern Applied**:
```go
cacheCtx, writeFn := sdkCtx.CacheContext()

// Phase 1: Bank transfer
if err := k.bankKeeper.SendCoins(..., cacheCtx, ...); err != nil {
    // Cache discarded automatically - no cleanup needed
    return err
}

// Phase 2: State update
if err := k.SetEscrowState(cacheCtx, ...); err != nil {
    // Cache discarded - bank transfer rolled back
    return err
}

// COMMIT: Both succeeded - write atomically
writeFn()
```

#### 2. `x/compute/keeper/invariants.go`
Added `EscrowStateConsistencyInvariant` to detect:
- Escrows missing timeout indexes
- Inconsistent state fields (RELEASED without ReleasedAt, etc.)
- Orphaned timeout indexes
- Unresolved catastrophic failures

#### 3. `x/compute/keeper/escrow_two_phase_commit_test.go`
Comprehensive tests for:
- All-or-nothing atomicity
- No funds lost on duplicate request IDs
- Idempotent refunds/releases
- No leaked funds across operations
- Atomic expired escrow processing

## Security Benefits
1. **No Catastrophic Failures**: Bank transfer and state update are truly atomic
2. **No Fund Loss**: Failed operations leave system in consistent state
3. **Early Detection**: Invariant catches any inconsistencies immediately
4. **Recovery**: Existing catastrophic failure infrastructure remains as safety net

## Testing
Run: `go test ./x/compute/keeper -run "TestLockEscrow|TestReleaseEscrow|TestRefundEscrow"`
