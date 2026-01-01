# SEC-3.10: Typed SDK Error Migration Guide

**Status**: Partially Complete (3/30 keeper files migrated)
**Date**: January 1, 2026

## Summary

Replacing generic `fmt.Errorf` calls with typed Cosmos SDK errors improves error handling, provides better context for clients, and enables structured error recovery.

## Completed Files

### x/compute/keeper/
- ✅ **msg_server.go** - All message handlers now use typed errors
- ✅ **provider.go** - Provider operations use typed errors
- ✅ **params.go** - Parameter operations use typed errors

## New Error Types Added

Added to `x/compute/types/errors.go`:

```go
// Serialization errors
ErrMarshalFailed   = sdkerrors.Register(ModuleName, 90, "failed to marshal data")
ErrUnmarshalFailed = sdkerrors.Register(ModuleName, 91, "failed to unmarshal data")

// Validation errors
ErrInvalidAddress      = sdkerrors.Register(ModuleName, 95, "invalid address")
ErrInvalidParameters   = sdkerrors.Register(ModuleName, 96, "invalid parameters")
ErrValidationFailed    = sdkerrors.Register(ModuleName, 97, "validation failed")
ErrInsufficientBalance = sdkerrors.Register(ModuleName, 98, "insufficient balance")

// State errors
ErrStateCorruption = sdkerrors.Register(ModuleName, 100, "state corruption detected")
ErrCacheMiss       = sdkerrors.Register(ModuleName, 101, "cache miss")
ErrCacheDisabled   = sdkerrors.Register(ModuleName, 102, "cache is disabled")
ErrCacheStale      = sdkerrors.Register(ModuleName, 103, "cache is stale")

// Appeal errors
ErrAppealNotFound          = sdkerrors.Register(ModuleName, 110, "appeal not found")
ErrAppealNotAcceptingVotes = sdkerrors.Register(ModuleName, 111, "appeal not accepting votes")
ErrInsufficientAppealDeposit = sdkerrors.Register(ModuleName, 112, "insufficient appeal deposit")

// Dispute errors
ErrDisputeNotFound            = sdkerrors.Register(ModuleName, 115, "dispute not found")
ErrDisputeNotAcceptingVotes   = sdkerrors.Register(ModuleName, 116, "dispute not accepting votes")
ErrInsufficientDisputeDeposit = sdkerrors.Register(ModuleName, 117, "insufficient dispute deposit")
ErrChallengePeriodActive      = sdkerrors.Register(ModuleName, 118, "challenge period is still active")
ErrConsensusThresholdNotMet   = sdkerrors.Register(ModuleName, 119, "consensus threshold not met")

// Resource limit errors
ErrConcurrentRequestLimit = sdkerrors.Register(ModuleName, 120, "concurrent request limit reached")
ErrCircuitTooComplex      = sdkerrors.Register(ModuleName, 121, "circuit too complex")
ErrCircuitNotInitialized  = sdkerrors.Register(ModuleName, 122, "circuit not initialized")

// Provider state errors
ErrProviderAlreadyRegistered = sdkerrors.Register(ModuleName, 125, "provider already registered")
ErrProviderAlreadyInactive   = sdkerrors.Register(ModuleName, 126, "provider already inactive")

// Randomness errors
ErrCommitmentNotFound      = sdkerrors.Register(ModuleName, 130, "randomness commitment not found")
ErrCommitmentAlreadyRevealed = sdkerrors.Register(ModuleName, 131, "commitment already revealed")
ErrCommitmentExpired       = sdkerrors.Register(ModuleName, 132, "commitment has expired")
ErrRevealVerificationFailed = sdkerrors.Register(ModuleName, 133, "reveal verification failed")
ErrMaxParticipantsReached  = sdkerrors.Register(ModuleName, 134, "maximum participants reached")

// Generic operation errors
ErrOperationFailed = sdkerrors.Register(ModuleName, 140, "operation failed")
ErrStorageFailed   = sdkerrors.Register(ModuleName, 141, "storage operation failed")
```

## Migration Patterns

### Pattern 1: Address Validation Errors
**Before:**
```go
addr, err := sdk.AccAddressFromBech32(msg.Provider)
if err != nil {
    return nil, fmt.Errorf("RegisterProvider: invalid provider address: %w", err)
}
```

**After:**
```go
addr, err := sdk.AccAddressFromBech32(msg.Provider)
if err != nil {
    return nil, types.ErrInvalidAddress.Wrapf("invalid provider address: %v", err)
}
```

### Pattern 2: Message Validation Errors
**Before:**
```go
if err := msg.ValidateBasic(); err != nil {
    return nil, fmt.Errorf("RegisterProvider: validate: %w", err)
}
```

**After:**
```go
if err := msg.ValidateBasic(); err != nil {
    return nil, types.ErrValidationFailed.Wrap(err.Error())
}
```

### Pattern 3: Nested Keeper Calls
**Before:**
```go
if err := ms.Keeper.RegisterProvider(...); err != nil {
    return nil, fmt.Errorf("RegisterProvider: %w", err)
}
```

**After:**
```go
if err := ms.Keeper.RegisterProvider(...); err != nil {
    return nil, err  // Keeper already returns typed error
}
```

### Pattern 4: Storage/Marshaling Errors
**Before:**
```go
bz, err := k.cdc.Marshal(&provider)
if err != nil {
    return fmt.Errorf("SetProvider: marshal: %w", err)
}
```

**After:**
```go
bz, err := k.cdc.Marshal(&provider)
if err != nil {
    return types.ErrMarshalFailed.Wrapf("failed to marshal provider: %v", err)
}
```

### Pattern 5: Resource Validation Errors
**Before:**
```go
if spec.CpuCores == 0 {
    return spec, fmt.Errorf("cpu_cores must be greater than 0")
}
```

**After:**
```go
if spec.CpuCores == 0 {
    return spec, types.ErrInvalidResourceSpec.Wrap("cpu_cores must be greater than 0")
}
```

### Pattern 6: Not Found Errors
**Before:**
```go
if bz == nil {
    return nil, fmt.Errorf("provider not found")
}
```

**After:**
```go
if bz == nil {
    return nil, types.ErrProviderNotFound
}
```

## Remaining Work

### Keeper Files Pending Migration (24 files)

High Priority (user-facing):
- [ ] `query_server.go` - Query handlers
- [ ] `randomness.go` - Randomness operations
- [ ] `request.go` - Request operations
- [ ] `escrow.go` - Escrow operations
- [ ] `dispute.go` - Dispute handling
- [ ] `slash.go` - Slashing operations

Medium Priority:
- [ ] `abci.go` - ABCI lifecycle handlers
- [ ] `verification.go` - Verification logic
- [ ] `circuit_breaker.go` - Circuit breaker
- [ ] `circuit_verification.go` - Circuit verification
- [ ] `provider_management.go` - Provider management
- [ ] `tx_rate_limit.go` - Rate limiting
- [ ] `ratelimit.go` - Rate limit queries

Lower Priority (internal/infrastructure):
- [ ] `keeper.go` - Keeper construction
- [ ] `ibc_compute.go` - IBC operations
- [ ] `genesis.go` - Genesis handling
- [ ] `migrations.go` - State migrations
- [ ] `channel_close.go` - IBC channel closing
- [ ] `ceremony_sink.go` - MPC ceremony
- [ ] `circuit_manager.go` - Circuit management
- [ ] `circuit_params.go` - Circuit params
- [ ] `ibc_packet_tracking.go` - IBC packet tracking
- [ ] `invariants.go` - Invariant checks
- [ ] `keys.go` - Storage keys
- [ ] `metrics.go` - Metrics collection
- [ ] `migration.go` - Migration helpers
- [ ] `nonce.go` - Nonce management
- [ ] `provider_cache.go` - Provider caching
- [ ] `reputation.go` - Reputation scoring
- [ ] `request_validation.go` - Request validation
- [ ] `security.go` - Security checks
- [ ] `zk_enhancements.go` - ZK proof enhancements

## Testing Verification

Build verification:
```bash
go build ./x/compute/keeper/...
```

No errors after migration of msg_server.go, provider.go, and params.go.

## Guidelines for Completing Migration

1. **Preserve Error Messages**: Keep existing descriptive error messages
2. **Use Appropriate Types**: Match error type to the context (ErrInvalidAddress for address errors, ErrValidationFailed for validation errors, etc.)
3. **Wrap with Context**: Use `.Wrapf()` to add context while preserving error chain
4. **Don't Wrap Twice**: If calling a keeper function that already returns typed errors, don't wrap again
5. **Build After Each File**: Verify compilation after each file migration
6. **Test Critical Paths**: For user-facing handlers, verify error responses are still correct

## Error Code Allocation

Reserved ranges in errors.go:
- 2-9: Request validation
- 10-19: Provider errors
- 20-29: Request lifecycle
- 30-39: Escrow errors
- 40-49: Verification errors
- 50-59: ZK proof errors
- 60-69: Resource errors
- 70-79: Security errors
- 80-89: IBC errors + Circuit breaker
- 90-99: Serialization + Validation
- 100-109: State errors
- 110-114: Appeal errors
- 115-119: Dispute errors
- 120-124: Resource limit errors
- 125-129: Provider state errors
- 130-134: Randomness errors
- 140-149: Generic operation errors

## Notes

- fmt.Errorf is still allowed in test files
- Some fmt.Errorf uses in IBC test files are intentional (for creating test error acknowledgements)
- Focus on keeper directory first (user-facing operations)
- Setup directory can be done later (less critical)
