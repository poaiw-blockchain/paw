# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for the PAW blockchain.

## Index

| ID | Title | Status | Date |
|----|-------|--------|------|
| [ADR-001](ADR-001-SECURE-SWAP-VARIANTS.md) | Secure Swap Variants | Accepted | 2025-12 |
| [ADR-002](ADR-002-SECURITY-LAYERS.md) | Security Layers | Accepted | 2025-12 |
| [ADR-003](ADR-003-COMMIT-REVEAL-SWAPS.md) | Commit-Reveal Swaps (MEV Protection) | Accepted | 2025-12 |
| [ADR-004](ADR-004-IBC-INTEGRATION.md) | IBC Integration Design | Accepted | 2025-12 |
| [ADR-005](ADR-005-ORACLE-AGGREGATION.md) | Oracle Price Aggregation | Accepted | 2025-12 |
| [ADR-006](ADR-006-COMPUTE-VERIFICATION.md) | Compute Verification & ZK Proofs | Accepted | 2025-12 |

## ADR Template

New ADRs should follow this structure:

```markdown
# ADR-XXX: Title

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
Why is this decision needed?

## Decision
What was decided?

## Consequences
What are the positive and negative outcomes?

## References
Related ADRs, issues, or external resources.
```

## Categories

- **DEX**: Swap mechanics, liquidity, MEV protection
- **Security**: Authentication, authorization, attack prevention
- **IBC**: Cross-chain communication
- **Oracle**: Price feeds and aggregation
- **Compute**: Off-chain computation verification
