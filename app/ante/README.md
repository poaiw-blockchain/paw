# PAW AnteHandler Architecture

## Overview

The AnteHandler is a middleware chain that validates transactions before they enter the mempool or are executed. PAW extends the standard Cosmos SDK AnteHandler with custom decorators for module-specific validation.

## Decorator Chain Order

Decorators execute in order (first to last):

1. `SetUpContextDecorator` - Initializes context with gas meter
2. `TimeValidatorDecorator` - Rejects invalid block timestamps
3. `GasLimitDecorator` - Enforces per-message/tx gas caps
4. `ExtensionOptionsDecorator` - Validates extension options
5. `ValidateBasicDecorator` - Calls `ValidateBasic()` on all messages
6. `TxTimeoutHeightDecorator` - Checks tx hasn't expired
7. `ValidateMemoDecorator` - SDK memo validation
8. `MemoLimitDecorator` - Hard cap at 256 bytes
9. `ConsumeGasForTxSizeDecorator` - Charges gas for tx bytes
10. `DeductFeeDecorator` - Deducts fees from sender
11. `SetPubKeyDecorator` - Sets signer public keys
12. `ValidateSigCountDecorator` - Limits signature count
13. `SigGasConsumeDecorator` - Charges gas for signatures
14. `SigVerificationDecorator` - Verifies signatures
15. `IncrementSequenceDecorator` - Increments account sequences
16. `RedundantRelayDecorator` - IBC duplicate relay check
17. `ComputeDecorator` - Compute module validation (optional)
18. `DEXDecorator` - DEX module validation (optional)
19. `OracleDecorator` - Oracle module validation (optional)

## Custom Decorator Requirements

### Must Be Read-Only

Custom decorators **MUST NOT** write state. AnteHandlers execute during:
- `CheckTx` (mempool admission)
- `DeliverTx` (block execution)
- `SimulateTx` (gas estimation)

State writes in ante would cause:
- Mempool inconsistency (CheckTx writes lost)
- Double-writes during DeliverTx
- Non-determinism between nodes

### Implementation Pattern

```go
func (d *CustomDecorator) AnteHandle(
    ctx sdk.Context,
    tx sdk.Tx,
    simulate bool,
    next sdk.AnteHandler,
) (sdk.Context, error) {
    // 1. Read-only validation only
    for _, msg := range tx.GetMsgs() {
        if err := d.validateReadOnly(ctx, msg); err != nil {
            return ctx, err  // Reject tx
        }
    }

    // 2. Pass to next decorator
    return next(ctx, tx, simulate)
}
```

### Gas Limits

| Operation | Max Gas | Rationale |
|-----------|---------|-----------|
| Swap | 200,000 | Single pool AMM calculation |
| Pool Creation | 300,000 | State initialization |
| Liquidity Add/Remove | 150,000 | Reserve updates |
| Compute Request | 250,000 | Job scheduling |
| ZK Verification | 500,000 | Groth16 proof check |
| Price Feed | 100,000 | Oracle submission |

Transaction limits:
- Max 10 messages per tx
- Max 10,000,000 gas per tx
- Max 500,000 gas per message

## Decorator-Specific Behavior

### TimeValidatorDecorator
Rejects blocks with timestamps outside acceptable drift window.

### GasLimitDecorator
Enforces per-operation caps to prevent gas exhaustion attacks.

### MemoLimitDecorator
Hard caps memo to 256 bytes regardless of SDK params.

### ComputeDecorator
- Validates compute requests have required fields
- Checks provider eligibility (read-only keeper queries)
- Rejects malformed ZK proof submissions

### DEXDecorator
- Validates token pairs exist
- Checks minimum amounts
- Enforces slippage bounds

### OracleDecorator
- Validates price feed format
- Checks validator authorization

## Testing Decorators

Each decorator has a `_test.go` file with:
- Unit tests for validation logic
- Integration tests with mock keepers
- Edge case coverage

Run tests:
```bash
go test ./app/ante/... -v
```

## Adding New Decorators

1. Create `{module}_decorator.go` implementing `sdk.AnteDecorator`
2. Add corresponding `{module}_decorator_test.go`
3. Register in `ante.go` NewAnteHandler() after module keepers
4. Ensure all validation is read-only
5. Document gas assumptions
