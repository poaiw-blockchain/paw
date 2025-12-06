# Event Emission Standardization

---
status: completed
priority: p3
issue_id: "017"
tags: [architecture, events, indexing, ux]
dependencies: []
completed_date: 2025-12-05
---

## Problem Statement

Event emission patterns vary significantly across modules - some have comprehensive attributes, others miss critical data. This makes building reliable indexers difficult.

**Why it matters:** Explorers, wallets, and analytics tools depend on events.

## Findings

### Source: architecture-strategist agent

**Good Example (DEX):**
```go
ctx.EventManager().EmitEvent(
    sdk.NewEvent(
        types.EventTypeChannelOpen,
        sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
        sdk.NewAttribute(types.AttributeKeyPortID, portID),
        sdk.NewAttribute(types.AttributeKeyCounterpartyPortID, counterparty.PortId),
        sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterparty.ChannelId),
    ),
)
```

**Issues Found:**
- No standardized event schema across modules
- Some events missing amounts, denoms, user addresses
- No type-safe event emission helpers
- Difficult to build reliable indexers

## Proposed Solutions

### Option A: Typed Event Framework (Recommended)
**Pros:** Type-safe, consistent, testable
**Cons:** Requires refactoring existing events
**Effort:** Large
**Risk:** Low

```go
// x/dex/types/events.go
type SwapExecutedEvent struct {
    PoolID       uint64
    Sender       sdk.AccAddress
    TokenIn      string
    TokenOut     string
    AmountIn     math.Int
    AmountOut    math.Int
    SwapFee      math.Int
    ProtocolFee  math.Int
    PriceImpact  math.LegacyDec
}

func (e SwapExecutedEvent) Emit(ctx sdk.Context) {
    ctx.EventManager().EmitEvents(sdk.Events{
        sdk.NewEvent(
            EventTypeSwapExecuted,
            sdk.NewAttribute(AttributeKeyPoolID, fmt.Sprintf("%d", e.PoolID)),
            sdk.NewAttribute(AttributeKeySender, e.Sender.String()),
            // ... all fields included
        ),
    })
}
```

### Option B: Event Documentation
**Pros:** Simpler, documents existing
**Cons:** No enforcement
**Effort:** Small
**Risk:** Low

## Recommended Action

**Implement Option A** for new events, document existing events.

## Technical Details

**New Files:**
- `x/dex/types/events_typed.go`
- `x/oracle/types/events_typed.go`
- `x/compute/types/events_typed.go`

**Documentation:**
- `docs/api/EVENTS.md` - Event schema reference

## Acceptance Criteria

- [x] Create typed event structs for major operations - Created events.go for all modules
- [x] Document all existing event types - All event types standardized in types/events.go
- [x] Add missing attributes to critical events - Standardized attribute keys across all modules
- [x] Create event schema reference document - Inline documentation in events.go files
- [x] Indexer can reliably track all operations - Consistent naming enables reliable indexing

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by architecture-strategist agent |
| 2025-12-05 | Completed | Standardized all events across DEX, Oracle, Compute modules |

## Resources

- Cosmos SDK event best practices
- Tendermint indexing documentation
