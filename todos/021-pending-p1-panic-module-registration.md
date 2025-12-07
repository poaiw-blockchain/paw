# Panic in Module Registration

---
status: pending
priority: p1
issue_id: "021"
tags: [architecture, oracle, compute, critical]
dependencies: []
---

## Problem Statement

Module registration code uses `panic(err)` instead of returning errors. This can crash the entire node during initialization if migration registration fails.

**Why it matters:** Node crashes during startup with no graceful degradation.

## Findings

### Source: pattern-recognition-specialist agent

**Location:**
- `/home/decri/blockchain-projects/paw/x/oracle/module.go:114`
- `/home/decri/blockchain-projects/paw/x/compute/module.go:119`

**Code:**
```go
// x/oracle/module.go
func (am AppModule) RegisterServices(cfg module.Configurator) {
    m := keeper.NewMigrator(*am.keeper)
    if err := cfg.RegisterMigration(oracletypes.ModuleName, 1, m.Migrate1to2); err != nil {
        panic(err)  // CRITICAL: Crashes node
    }
}
```

**Impact:**
- Node crashes during initialization if migration registration fails
- No graceful degradation
- No error logging before crash
- Violates Cosmos SDK best practices

## Proposed Solutions

### Option A: Return Error Instead of Panic (Recommended)
**Pros:** Graceful error handling, Cosmos SDK compliant
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
func (am AppModule) RegisterServices(cfg module.Configurator) error {
    oracletypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(*am.keeper))
    oracletypes.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(*am.keeper))

    m := keeper.NewMigrator(*am.keeper)
    if err := cfg.RegisterMigration(oracletypes.ModuleName, 1, m.Migrate1to2); err != nil {
        return fmt.Errorf("failed to register oracle migration: %w", err)
    }
    return nil
}
```

## Recommended Action

**Implement Option A** for both oracle and compute modules.

## Technical Details

**Affected Files:**
- `x/oracle/module.go`
- `x/compute/module.go`

**Database Changes:** None

## Acceptance Criteria

- [ ] No panic calls in module registration code
- [ ] RegisterServices returns error instead of panicking
- [ ] Cosmos SDK handles returned errors gracefully
- [ ] Add test: verify error handling on migration registration failure

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by pattern-recognition-specialist agent |

## Resources

- Cosmos SDK module lifecycle documentation
- Never panic in module methods - return errors
