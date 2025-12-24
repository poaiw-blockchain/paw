# ADR-001: Secure Swap Implementation Pattern

## Status
Accepted

## Context
The DEX module requires swap functionality that is both efficient and secure. Initial implementation had a single `ExecuteSwap` function that was modified over time to add security checks, making it complex and harder to audit.

## Decision
Implement a parallel secure variant pattern:

1. **Original function** (`ExecuteSwap`) - Core AMM logic only
2. **Secure variant** (`ExecuteSwapSecure`) - Wraps original with security layers

```go
// ExecuteSwap - Core AMM calculation (pure, auditable)
func (k Keeper) ExecuteSwap(ctx context.Context, params SwapParams) (*SwapResult, error)

// ExecuteSwapSecure - Production entry point with all protections
func (k Keeper) ExecuteSwapSecure(ctx context.Context, params SwapParams) (*SwapResult, error) {
    // 1. Reentrancy guard
    // 2. Rate limiting
    // 3. Flash loan protection
    // 4. Price impact validation
    // 5. Circuit breaker check
    // 6. Call ExecuteSwap
    // 7. Invariant verification
    return result, nil
}
```

## Rationale
- **Separation of concerns**: AMM math stays pure, security layers are explicit
- **Auditability**: Security reviewers can inspect each layer independently
- **Testability**: Core logic testable without security overhead
- **Flexibility**: Easy to add/remove security checks via configuration

## Consequences

### Positive
- Clear audit trail for security features
- Easier to reason about core swap math
- Security layers can be governance-controlled

### Negative
- Two functions to maintain per operation
- Potential for calling non-secure variant accidentally

### Mitigations
- All message handlers use `*Secure` variants
- Non-secure variants are internal (lowercase in keeper)
- Tests verify message handlers use secure paths

## Files Changed
- `x/dex/keeper/swap.go` - Core swap logic
- `x/dex/keeper/secure_variants.go` - Security wrappers
- `x/dex/keeper/msg_server.go` - Uses secure variants only
