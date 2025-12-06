# Privacy Module Unused - Decision Required

---
status: pending
priority: p3
issue_id: "016"
tags: [architecture, cleanup, decision]
dependencies: []
---

## Problem Statement

The `x/privacy/` module directory exists but has ZERO commits touching it. This is dead code in the production repository.

**Why it matters:** Maintenance burden, confusion about scope.

## Findings

### Source: git-history-analyzer agent

**Directory Structure:**
```
x/privacy/
├── keeper/
└── types/
```

**Git History:** Zero commits modifying this module

**Implications:**
- Module scaffolded but never implemented
- Unclear if it's planned or abandoned
- Dead code increases codebase complexity
- May confuse contributors

## Proposed Solutions

### Option A: Remove Module (Recommended if not planned)
**Pros:** Clean codebase
**Cons:** None if truly unused
**Effort:** Small
**Risk:** Low

### Option B: Add to Roadmap (If planned)
**Pros:** Clear intent
**Cons:** None
**Effort:** Small
**Risk:** Low

### Option C: Archive for Future
**Pros:** Preserves work
**Cons:** Still clutters codebase
**Effort:** Small
**Risk:** Low

## Recommended Action

**Decision required:** Is privacy module planned?
- If YES: Add to ROADMAP_PRODUCTION.md with timeline
- If NO: Delete `x/privacy/` directory

## Technical Details

**Affected Files:**
- `x/privacy/` (entire directory)
- `app/app.go` (if module is registered)

## Acceptance Criteria

- [ ] Decision documented in ROADMAP or commit message
- [ ] If removing: delete `x/privacy/` and all references
- [ ] If keeping: add implementation tasks to roadmap
- [ ] No orphaned imports or references

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by git-history-analyzer agent |

## Resources

- Project scope documentation
- Privacy feature requirements (if any)
