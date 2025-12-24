# ADR-003: Commit-Reveal for Large Swaps

## Status
Accepted

## Context
Large DEX swaps are vulnerable to front-running and MEV extraction. Attackers can see pending transactions and insert their own trades before/after to profit.

## Decision
Implement a commit-reveal scheme for swaps above threshold:

### Phase 1: Commit
User submits hash of swap details:
```
commitment = SHA256(poolId || amountIn || minOut || sender || nonce || secret)
```
- Commitment stored on-chain with timestamp
- No swap details visible to observers

### Phase 2: Reveal (after N blocks)
User reveals swap parameters:
- Chain verifies commitment matches reveal
- Swap executes if commitment valid and not expired

### Parameters
- Threshold: 10,000 USD equivalent
- Reveal window: 2-10 blocks (configurable)
- Commitment fee: 0.01% of swap value

## Rationale
- Front-runners cannot know swap direction until reveal
- By reveal time, transactions are ordered
- Economic cost discourages spam commits

## Implementation

```go
// Commitment structure
type SwapCommitment struct {
    Sender       string
    CommitHash   []byte
    CommitHeight int64
    Expiry       int64
    Status       CommitmentStatus
}

// Reveal verification
func (k Keeper) VerifyAndExecuteCommittedSwap(
    ctx context.Context,
    reveal *SwapReveal,
) (*SwapResult, error)
```

## Flow

```
Block N:   User commits hash → stored in state
Block N+2: User reveals params → hash verified → swap executes
Block N+10: Commitment expires if not revealed
```

## Consequences

### Positive
- Eliminates front-running for large swaps
- MEV extraction significantly reduced
- User retains option to not reveal if market moves

### Negative
- Two transactions required (higher total gas)
- Minimum 2 block delay for large swaps
- Additional state storage for commitments

### Mitigations
- Small swaps bypass commit-reveal (instant execution)
- Expired commitments auto-cleanup in EndBlocker
- Commitment fee covers storage costs

## Files
- `x/dex/keeper/commit_reveal.go` - Core implementation
- `x/dex/types/commit_reveal.go` - Types and validation
- `x/dex/keeper/abci.go` - Expiry cleanup
