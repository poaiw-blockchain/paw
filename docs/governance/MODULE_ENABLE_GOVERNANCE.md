# Module Enable Governance Plan

This document outlines the governance process for enabling post-MVP modules (DEX, Compute, Oracle) after the initial MVP launch.

## Overview

For MVP stability, the following modules are disabled at genesis:
- **x/dex** - Decentralized exchange functionality
- **x/compute** - Distributed compute pooling
- **x/oracle** - Price oracle feeds

These modules can be enabled via governance proposal after the network has demonstrated stability.

## Module Enable Parameters

Each module has an `enabled` boolean parameter in its params:

| Module | Parameter Path | Default (MVP) |
|--------|----------------|---------------|
| DEX | `dex.params.enabled` | `false` |
| Compute | `compute.params.enabled` | `false` |
| Oracle | `oracle.params.enabled` | `false` |

## Governance Proposal Templates

### Enable DEX Module

```json
{
  "title": "Enable DEX Module",
  "description": "Proposal to enable the DEX (Decentralized Exchange) module. This will allow users to create liquidity pools, swap tokens, and participate in cross-chain trading via IBC.",
  "changes": [
    {
      "subspace": "dex",
      "key": "Enabled",
      "value": "true"
    }
  ],
  "deposit": "10000000upaw"
}
```

**CLI Command:**
```bash
pawd tx gov submit-proposal param-change enable-dex.json \
  --from <key> \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-adjustment 1.4
```

### Enable Compute Module

```json
{
  "title": "Enable Compute Module",
  "description": "Proposal to enable the Compute module. This will allow providers to register compute resources and users to submit compute requests for distributed processing.",
  "changes": [
    {
      "subspace": "compute",
      "key": "Enabled",
      "value": "true"
    }
  ],
  "deposit": "10000000upaw"
}
```

### Enable Oracle Module

```json
{
  "title": "Enable Oracle Module",
  "description": "Proposal to enable the Oracle module. This will activate price feed submissions from validators and enable TWAP calculations for on-chain price data.",
  "changes": [
    {
      "subspace": "oracle",
      "key": "Enabled",
      "value": "true"
    }
  ],
  "deposit": "10000000upaw"
}
```

## Recommended Enable Order

1. **Oracle Module** (Week 2-4 post-launch)
   - Prerequisite for DEX price feeds
   - Lower risk, read-heavy operations
   - Validators should configure price feed sources first

2. **DEX Module** (Week 4-8 post-launch)
   - Requires Oracle for price validation
   - Higher economic risk, needs audit confirmation
   - Start with limited pool types

3. **Compute Module** (Week 8-12 post-launch)
   - Independent of other modules
   - Requires provider onboarding
   - Most complex operational requirements

## Pre-Enable Checklist

Before submitting an enable proposal:

- [ ] Module code has been audited
- [ ] Integration tests pass on testnet
- [ ] Documentation is complete
- [ ] Validators are prepared (for Oracle)
- [ ] Initial liquidity providers ready (for DEX)
- [ ] Compute providers registered (for Compute)
- [ ] Emergency procedures documented
- [ ] Rollback plan in place

## Voting Parameters

| Parameter | Value |
|-----------|-------|
| Voting Period | 2 days (testnet) / 7 days (mainnet) |
| Quorum | 33.4% |
| Threshold | 50% |
| Veto Threshold | 33.4% |

## Emergency Disable

If a critical issue is discovered after enabling:

1. **Emergency Admin Pause** (Oracle only):
   ```bash
   pawd tx oracle emergency-pause --from <emergency_admin>
   ```

2. **Governance Disable Proposal**:
   Submit reverse proposal setting `enabled: false`

3. **Chain Halt** (last resort):
   Coordinate validator halt if governance too slow

## Monitoring Post-Enable

After enabling a module, monitor:

- Transaction success/failure rates
- Gas consumption patterns
- State growth
- Error logs from validators
- User feedback channels

## Related Documents

- [MVP Release Report](../../MVP_RELEASE_REPORT.md)
- [Security Considerations](../SECURITY.md)
- [Module Documentation](../modules/)
