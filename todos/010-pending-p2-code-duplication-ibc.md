# Triplicate IBC Authorization Code

---
status: pending
priority: p2
issue_id: "010"
tags: [code-quality, ibc, duplication, maintainability]
dependencies: []
---

## Problem Statement

Identical IBC channel authorization code is copy-pasted across all three modules: DEX, Oracle, and Compute. This creates maintenance burden and inconsistency risk.

**Why it matters:** Bug fixes must be applied 3x, changes may be missed in one module.

## Findings

### Source: pattern-recognition-specialist agent

**Duplicate Locations:**
- `/home/decri/blockchain-projects/paw/x/oracle/keeper/keeper.go:115-167`
- `/home/decri/blockchain-projects/paw/x/dex/keeper/keeper.go:108-161`
- `/home/decri/blockchain-projects/paw/x/compute/keeper/keeper.go:135-187`

**Duplicated Functions (~150 lines each):**
- `AuthorizeChannel(ctx, portID, channelID)`
- `SetAuthorizedChannels(ctx, channels)`
- `IsAuthorizedChannel(ctx, portID, channelID)`

**Example of identical code:**
```go
// IDENTICAL in all three modules:
func (k Keeper) AuthorizeChannel(ctx sdk.Context, portID, channelID string) error {
    portID = strings.TrimSpace(portID)
    channelID = strings.TrimSpace(channelID)
    if portID == "" || channelID == "" {
        return errorsmod.Wrap(types.ErrInvalidXXX, "port_id and channel_id must be non-empty")
    }
    // ... exact same logic repeated
}
```

**Total Duplication:** ~450 lines (3 modules × ~150 lines)

## Proposed Solutions

### Option A: Shared IBC Utility Package (Recommended)
**Pros:** Single source of truth, testable
**Cons:** Adds import dependency
**Effort:** Medium
**Risk:** Low

```go
// app/ibcutil/channel_authorization.go
package ibcutil

type ChannelAuthorizer interface {
    GetAuthorizedChannels(ctx context.Context) ([]AuthorizedChannel, error)
    SetAuthorizedChannels(ctx context.Context, channels []AuthorizedChannel) error
}

func AuthorizeChannel(ctx context.Context, auth ChannelAuthorizer, portID, channelID string) error {
    portID = strings.TrimSpace(portID)
    channelID = strings.TrimSpace(channelID)
    if portID == "" || channelID == "" {
        return ErrInvalidChannel
    }

    channels, err := auth.GetAuthorizedChannels(ctx)
    if err != nil {
        return err
    }

    // Check if already authorized
    for _, ch := range channels {
        if ch.PortId == portID && ch.ChannelId == channelID {
            return nil // Already authorized
        }
    }

    channels = append(channels, AuthorizedChannel{PortId: portID, ChannelId: channelID})
    return auth.SetAuthorizedChannels(ctx, channels)
}
```

### Option B: Code Generation
**Pros:** Auto-generates from template
**Cons:** Build complexity
**Effort:** Large
**Risk:** Medium

## Recommended Action

**Implement Option A** - create `app/ibcutil` package, refactor all three keepers.

## Technical Details

**New Files:**
- `app/ibcutil/channel_authorization.go`
- `app/ibcutil/channel_authorization_test.go`

**Modified Files:**
- `x/dex/keeper/keeper.go`
- `x/oracle/keeper/keeper.go`
- `x/compute/keeper/keeper.go`

## Acceptance Criteria

- [ ] Create `app/ibcutil` package
- [ ] Implement shared `ChannelAuthorizer` interface
- [ ] Refactor DEX keeper to use shared code
- [ ] Refactor Oracle keeper to use shared code
- [ ] Refactor Compute keeper to use shared code
- [ ] Remove duplicated code (~450 lines)
- [ ] Add comprehensive tests for shared code
- [ ] Existing IBC tests pass

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by pattern-recognition-specialist agent |

## Resources

- Cosmos SDK IBC middleware patterns
- Go interface design patterns
