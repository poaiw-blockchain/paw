# ADR-004: IBC Integration Design

## Status
Accepted

## Context
PAW requires cross-chain communication for:
- Cross-chain swaps via DEX module
- Oracle price feeds from other chains
- Compute job distribution across chains

Standard IBC patterns needed adaptation for our security requirements.

## Decision

### Channel Architecture
- Dedicated channels per module (dex, oracle, compute)
- Unified `ChannelOperation` type in `x/shared/ibc/types.go`
- Packet nonce tracking to prevent replay attacks

### Packet Types
```
DEX:     SwapRequest, SwapResponse, LiquidityTransfer
Oracle:  PriceUpdate, ValidatorAttestation
Compute: JobAssignment, ResultSubmission
```

### Security Measures
1. **Nonce tracking**: Per-channel sequence validation
2. **Timeout handling**: Automatic refunds on timeout
3. **Channel close cleanup**: Graceful state cleanup via `CleanupChannelClose()`

### Error Handling
- Failed packets trigger automatic rollback
- Acknowledgment errors logged with full context
- Circuit breaker integration for repeated failures

## Consequences

**Positive:**
- Clean separation of cross-chain concerns
- Unified error handling across modules
- Replay attack prevention built-in

**Negative:**
- Channel management complexity
- Additional gas costs for nonce tracking

## References
- [IBC Specification](https://github.com/cosmos/ibc)
- ADR-002: Security Layers
