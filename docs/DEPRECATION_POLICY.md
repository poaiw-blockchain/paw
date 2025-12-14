# PAW Blockchain Deprecation Policy

This document defines the deprecation policy for the PAW blockchain, including APIs, features, modules, and breaking changes.

## Table of Contents

1. [Versioning Strategy](#versioning-strategy)
2. [Deprecation Timeline](#deprecation-timeline)
3. [Backward Compatibility](#backward-compatibility)
4. [Migration Support](#migration-support)
5. [Communication Plan](#communication-plan)
6. [Deprecation Process](#deprecation-process)
7. [Examples](#examples)

---

## Versioning Strategy

### Semantic Versioning

PAW follows [Semantic Versioning 2.0.0](https://semver.org/) with blockchain-specific interpretations:

**Format**: `MAJOR.MINOR.PATCH`

- **MAJOR** (v1.x.x → v2.0.0): Consensus-breaking changes, major API redesigns, state migrations
- **MINOR** (v1.0.x → v1.1.0): New features, non-breaking API additions, performance improvements
- **PATCH** (v1.0.0 → v1.0.1): Bug fixes, security patches, non-breaking optimizations

### Pre-1.0 Development

- **v0.x.x**: Pre-mainnet releases, subject to rapid iteration
- **Breaking changes allowed** in MINOR versions during v0.x.x phase
- **Deprecation timelines reduced** to 30 days minimum

### Version Support

| Version Type | Support Duration | Updates |
|--------------|------------------|---------|
| **Latest Major** | Active development | Features, bug fixes, security |
| **Previous Major** | 12 months | Bug fixes, critical security |
| **Older Versions** | End of life | Security only (90 days) |

**Example**: When v2.0.0 releases, v1.x.x receives 12 months of maintenance support.

---

## Deprecation Timeline

### Standard Deprecation Process

**Total Timeline**: Minimum 90 days (3 months) from announcement to removal

| Phase | Duration | Activities |
|-------|----------|------------|
| **Announcement** | Day 0 | Public notice, documentation updates |
| **Warning Period** | 60 days | Deprecation warnings in logs/CLI |
| **Migration Window** | 30 days | Feature disabled by default, opt-in available |
| **Removal** | Day 91+ | Feature removed in next MAJOR release |

### Accelerated Deprecation

For **security vulnerabilities** or **critical bugs**, timeline may be reduced to:

- **Announcement**: Immediate
- **Warning Period**: 14 days
- **Removal**: Next PATCH or MINOR release

**Requirements**:
- Security advisory published
- Core team approval
- Emergency governance proposal (if consensus-breaking)

### Component-Specific Timelines

| Component | Minimum Timeline | Notes |
|-----------|------------------|-------|
| **RPC/gRPC APIs** | 180 days (6 months) | High external usage |
| **CLI Commands** | 90 days (3 months) | Standard timeline |
| **Module APIs** | 90 days (3 months) | Chain developers affected |
| **Config Parameters** | 60 days (2 months) | Lower impact |
| **Internal APIs** | 30 days (1 month) | Development team only |

---

## Backward Compatibility

### Compatibility Guarantees

**Within MAJOR version** (e.g., v1.0.0 → v1.9.0):
- ✅ RPC/gRPC API signatures unchanged
- ✅ CLI command syntax stable
- ✅ State format compatible (no migrations)
- ✅ Config file format compatible
- ⚠️ New fields may be added (additive changes only)

**Across MAJOR versions** (e.g., v1.x.x → v2.0.0):
- ❌ Breaking changes allowed
- ✅ Migration guides provided
- ✅ Deprecated features removed
- ✅ State migration tools included

### Breaking Change Definition

A change is **breaking** if it:
- Removes or renames public APIs
- Changes RPC/gRPC response formats
- Modifies consensus behavior
- Requires state migration
- Changes CLI command syntax
- Alters config file structure

### Non-Breaking Change Examples

- Adding new optional parameters
- Adding new fields to responses
- New RPC/gRPC endpoints
- Performance optimizations (no behavior change)
- Additional log messages

---

## Migration Support

### Migration Path Requirements

Every deprecation must include:

1. **Alternative Solution**: Replacement API/feature documented
2. **Migration Guide**: Step-by-step instructions
3. **Automation Tools**: Scripts/commands where possible
4. **Validation Tools**: Verify successful migration

### Developer Support

**During Deprecation Period**:
- Migration examples in documentation
- Automated migration scripts (where applicable)
- Community forum support
- Office hours for complex migrations

**Post-Removal**:
- Version-specific migration guides maintained
- Historical documentation archived
- Community support for stragglers

### State Migration

For consensus-breaking changes requiring state export/import:

```bash
# Export state before upgrade
pawd export --height <upgrade-height> > genesis_export.json

# Run migration tool
pawd migrate v2 genesis_export.json --chain-id=paw-2 > genesis_v2.json

# Validate migrated state
pawd validate-genesis genesis_v2.json

# Start with new binary
pawd start --home ~/.paw
```

**Migration Tool Requirements**:
- Idempotent (safe to run multiple times)
- Comprehensive error messages
- Rollback capability
- Progress indicators for large states

---

## Communication Plan

### Deprecation Announcement Channels

**Priority Order**:

1. **GitHub Release Notes**: Detailed technical changes
2. **Developer Discord**: Real-time discussion
3. **Blog Post**: User-facing explanation
4. **Twitter/X**: Public announcement
5. **Email Newsletter**: Validator operators

### Announcement Template

```markdown
## Deprecation Notice: [Feature Name]

**Status**: Deprecated
**Announced**: YYYY-MM-DD
**Removal Planned**: v[X.Y.Z] (YYYY-MM-DD)
**Reason**: [Brief explanation]

### Impact
- [Who is affected]
- [What breaks]

### Migration Path
[Link to migration guide]

### Alternative
Use `[new-feature]` instead. See: [documentation link]

### Questions?
Discord: #deprecations
GitHub Discussions: [link]
```

### In-Code Warnings

**CLI Warning** (last 30 days before removal):
```
⚠️  WARNING: Command 'pawd legacy-cmd' is deprecated and will be removed in v2.0.0
    Use 'pawd new-cmd' instead. See: https://docs.paw.network/migration/legacy-cmd
```

**Log Warning** (throughout deprecation period):
```
WARN [module] Deprecated API call: GetLegacyBalance (removed in v2.0.0). Use GetBalance instead.
```

**RPC Response** (added to deprecated endpoints):
```json
{
  "result": {...},
  "deprecated": {
    "notice": "This endpoint is deprecated and will be removed in v2.0.0",
    "replacement": "/cosmos/bank/v1beta1/balances/{address}",
    "docs": "https://docs.paw.network/api/migration"
  }
}
```

---

## Deprecation Process

### Step-by-Step Workflow

#### 1. Proposal (Core Team)
- [ ] Document reason for deprecation
- [ ] Identify affected users/developers
- [ ] Design replacement API/feature
- [ ] Draft migration guide
- [ ] Estimate timeline

#### 2. Announcement (Day 0)
- [ ] Update documentation with deprecation notice
- [ ] Add in-code warnings
- [ ] Publish blog post
- [ ] Notify on all channels
- [ ] Create tracking GitHub issue

#### 3. Warning Period (Days 1-60)
- [ ] Monitor usage metrics
- [ ] Answer migration questions
- [ ] Update migration guide based on feedback
- [ ] Send reminder at Day 30

#### 4. Migration Window (Days 61-90)
- [ ] Disable by default (opt-in flag required)
- [ ] Final reminder at Day 75
- [ ] Publish upgrade timeline

#### 5. Removal (Next MAJOR Release)
- [ ] Remove deprecated code
- [ ] Update changelog
- [ ] Update documentation
- [ ] Archive migration guide

### Governance Integration

**For consensus-breaking changes**:
- Submit governance proposal
- 14-day voting period
- Requires 33.4% quorum, >50% Yes votes
- Include upgrade handler code

**Example Proposal**:
```bash
pawd tx gov submit-proposal software-upgrade v2.0.0 \
  --title "Upgrade to v2.0.0: Remove Deprecated APIs" \
  --description "$(cat upgrade_v2_description.md)" \
  --upgrade-height 1000000 \
  --deposit 10000000upaw \
  --from validator
```

---

## Examples

### Example 1: Deprecating RPC Endpoint

**Scenario**: Remove `/legacy/balance` endpoint in favor of standard Cosmos endpoint

**Timeline**:
```
2025-01-15: Announcement (v1.5.0 release)
2025-03-15: Warning logs enabled
2025-04-15: Endpoint returns 410 Gone (opt-in flag to enable)
2025-07-15: Removal in v2.0.0
```

**Migration Guide** (`docs/migration/legacy-balance.md`):
```markdown
# Migrating from /legacy/balance

## Old Endpoint (Deprecated)
GET /legacy/balance/{address}

## New Endpoint
GET /cosmos/bank/v1beta1/balances/{address}

## Code Example
```javascript
// Before
const balance = await client.get(`/legacy/balance/${address}`);

// After
const response = await client.get(`/cosmos/bank/v1beta1/balances/${address}`);
const balance = response.balances.find(b => b.denom === 'upaw');
```
```

### Example 2: Deprecating CLI Flag

**Scenario**: Remove `--legacy-format` from `pawd query` command

**Code Changes**:
```go
// cmd/pawd/cmd/query.go
cmd.Flags().Bool("legacy-format", false, "[DEPRECATED] Use --output=json instead. Removed in v2.0.0")
cmd.Flags().MarkDeprecated("legacy-format", "use --output=json instead (removal: v2.0.0)")
```

**Runtime Warning**:
```
$ pawd query bank balances paw1... --legacy-format
⚠️  Flag --legacy-format is deprecated: use --output=json instead (removal: v2.0.0)
```

### Example 3: Module API Deprecation

**Scenario**: Deprecate `keeper.GetLegacyParams()` in favor of `keeper.GetParams()`

**Code**:
```go
// x/dex/keeper/params.go

// GetLegacyParams returns module parameters (DEPRECATED).
//
// Deprecated: Use GetParams() instead. This method will be removed in v2.0.0.
func (k Keeper) GetLegacyParams(ctx sdk.Context) types.LegacyParams {
    k.Logger(ctx).Warn("GetLegacyParams is deprecated, use GetParams instead (removal: v2.0.0)")
    return k.migrateParams(k.GetParams(ctx))
}
```

**Documentation**:
```markdown
## ~~GetLegacyParams~~ (Deprecated)

**Deprecated**: Use `GetParams()` instead. Removed in v2.0.0.

Migration:
```go
// Before
params := keeper.GetLegacyParams(ctx)

// After
params := keeper.GetParams(ctx)
```
```

---

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Cosmos SDK Deprecation Guidelines](https://github.com/cosmos/cosmos-sdk/blob/main/UPGRADING.md)
- [PAW Upgrade Procedures](./UPGRADE_PROCEDURES.md)
- [PAW Governance Guide](./guides/GOVERNANCE_PROPOSALS.md)
