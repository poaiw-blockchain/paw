# ADR-002: Multi-Layer Security Architecture

## Status
Accepted

## Context
Blockchain DEX security requires defense-in-depth. Single-point security checks are insufficient against sophisticated attacks (MEV, flash loans, sandwich attacks).

## Decision
Implement five security layers that all operations must pass:

### Layer 1: AnteHandler Validation
- Per-operation gas limits
- Transaction size limits
- Memo bounds
- Signature verification

### Layer 2: Keeper Guards
- Reentrancy prevention (`withReentrancyGuard`)
- Rate limiting per address
- Flash loan protection (block delay enforcement)
- Commit-reveal for large swaps

### Layer 3: Economic Protections
- Price impact validation (max 10% deviation)
- Slippage enforcement
- Minimum output requirements
- Constant product invariant (k = x * y)

### Layer 4: Circuit Breakers
- Per-pool pause capability
- Module-wide emergency halt
- Multi-sig governance for activation
- Automatic trigger on anomaly detection

### Layer 5: Post-Operation Verification
- Invariant checks after state mutation
- Event emission for monitoring
- TWAP update for oracle integration

## Implementation

```
Transaction Flow:
┌─────────────────┐
│   AnteHandler   │ ← Layer 1: Gas, size, signature
├─────────────────┤
│  Keeper Guard   │ ← Layer 2: Reentrancy, rate limit
├─────────────────┤
│ Economic Check  │ ← Layer 3: Price impact, slippage
├─────────────────┤
│ Circuit Breaker │ ← Layer 4: Pause check
├─────────────────┤
│ Core Operation  │ ← Actual swap/liquidity change
├─────────────────┤
│ Post-Verify     │ ← Layer 5: Invariants, events
└─────────────────┘
```

## Rationale
- Each layer defends against specific attack class
- Layers are independent and composable
- Failure at any layer rejects the transaction
- Audit scope is well-defined per layer

## Consequences

### Positive
- Defense in depth against multiple attack vectors
- Clear responsibility for each layer
- Easy to add new checks without disrupting existing ones

### Negative
- Increased gas cost per operation
- More complex codebase
- Potential for over-blocking legitimate transactions

### Mitigations
- Gas costs documented and benchmarked
- Each layer has dedicated tests
- Thresholds are governance-configurable

## Files
- `app/ante/` - Layer 1
- `x/dex/keeper/security.go` - Layers 2-4
- `x/dex/keeper/invariants.go` - Layer 5
