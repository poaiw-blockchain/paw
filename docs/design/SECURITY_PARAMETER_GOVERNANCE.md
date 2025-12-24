# Security Parameter Governance Design

This document describes the path to governance control for security parameters in the DEX and Oracle modules.

## Current State

Security parameters are hard-coded constants to prevent governance attacks where malicious proposals weaken security.

## DEX Module Parameters

| Parameter | Value | Purpose |
|-----------|-------|---------|
| MaxPriceDeviation | 0.25 (25%) | Triggers circuit breaker on large price swings |
| MaxSwapSizePercent | 0.1 (10%) | Limits single swap to prevent MEV/sandwich attacks |
| MinLPLockBlocks | 1 | Prevents flash loan add/remove exploits |
| PriceUpdateTolerance | 0.001 (0.1%) | Invariant check tolerance for k=x*y |

## Oracle Module Parameters

| Parameter | Value | Purpose |
|-----------|-------|---------|
| MinValidatorsForSecurity | 7 | Minimum validators for BFT (tolerates 2 Byzantine) |
| MinGeographicRegions | 3 | Prevents regional network attacks |
| MinBlocksBetweenSubmissions | 1 | Breaks flash loan atomicity |
| MaxDataStalenessBlocks | 100 | ~10 min freshness requirement |
| MaxSubmissionsPerWindow | 10 | Rate limiting per 100 blocks |

## Path to Governance (If Required)

### Design Changes Needed
1. Move constants to module Params (`.proto` definitions)
2. Add strict validation bounds for each parameter
3. Implement time-locks on parameter changes (7-day DEX, 30-day Oracle)
4. Require supermajority (>66%) for security-critical changes
5. Add emergency governance override to restore safe defaults

### Oracle-Specific Considerations
Oracle parameters are MORE sensitive than DEX parameters because:
- Compromised oracle can drain ALL DEX pools simultaneously
- Recovery from oracle attacks is harder than DEX exploits
- Byzantine tolerance math must be exact

**Recommendation:** MinValidatorsForSecurity and MinGeographicRegions should NEVER be governable.

### Implementation Steps
1. Add parameters to `.proto` with validation bounds
2. Implement parameter change proposals via x/gov
3. Add time-lock mechanism (pending params, delayed application)
4. Add monitoring for malicious governance proposals
5. Gradual rollout: disabled initially, enabled after audit

### Risks of Governance
- Malicious proposals weakening security (e.g., MaxSwapSizePercent → 1.0)
- Governance attacks via stake accumulation
- Front-running of parameter changes by MEV searchers

### Audit Requirements
- Full security audit before enabling governance
- Formal verification of parameter bounds
- Economic modeling of attack costs
- Adversarial testnet testing

## Recommendation

Keep hard-coded for mainnet launch. Consider governance for non-critical parameters after 6-12 months of stable operation.
