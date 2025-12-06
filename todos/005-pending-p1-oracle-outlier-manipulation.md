# Oracle Price Manipulation via Outlier Filter Bypass

---
status: pending
priority: p1
issue_id: "005"
tags: [security, oracle, price-manipulation, critical]
dependencies: []
---

## Problem Statement

The oracle outlier detection in `x/oracle/keeper/aggregation.go:270-295` preserves at least 3 validators even if all are outliers. An attacker controlling 3 validators can manipulate prices by submitting coordinated extreme values that pass the minimum validator threshold.

**Why it matters:** DEX trades and oracle-dependent protocols would execute at manipulated prices.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/oracle/keeper/aggregation.go:270-295`

```go
// Ensure we keep at least some validators if too many filtered
minValidators := 3
if len(validPrices) < minValidators && len(prices) >= minValidators {
    // Keep the closest prices to median
    validPrices = k.keepClosestToMedian(prices, median, minValidators)
    // ... recalculates outliers but still keeps 3
}
```

**Attack Scenario:**
1. Attacker controls 3 validators (stake acquisition or collusion)
2. All 3 submit coordinated price: $150 for asset worth $100
3. Honest validators submit $100
4. Statistical filters detect ALL prices as outliers (high deviation from mixed data)
5. Code keeps 3 "closest to median" - but if 3 attackers are close to each other, they might all be kept
6. Resulting price is manipulated

**Conditions for Attack:**
- Attacker controls >= 3 validators
- Attacker prices are consistent with each other (appear as a cluster)
- Honest prices are diverse enough to appear as outliers when mixed

## Proposed Solutions

### Option A: Voting Power Threshold (Recommended)
**Pros:** Aligns with PoS security model
**Cons:** Requires staking keeper integration
**Effort:** Medium
**Risk:** Low

```go
// Replace validator count check with voting power check
minVotingPower := params.MinVotingPowerForConsensus // e.g., 10%
totalVotingPower := k.calculateTotalVotingPower(ctx, validPrices)

if totalVotingPower < minVotingPower {
    // Not enough voting power behind remaining prices
    return types.ErrInsufficientOracleConsensus
}
```

### Option B: Geographic Diversity Enforcement
**Pros:** Uses existing params (MinGeographicRegions)
**Cons:** Requires reliable region data
**Effort:** Medium
**Risk:** Medium

### Option C: TWAP as Primary Price
**Pros:** Resistant to single-block manipulation
**Cons:** Lagging indicator, not suitable for all use cases
**Effort:** Small
**Risk:** Low

## Recommended Action

**Implement Option A** with:
1. Replace validator count with voting power threshold
2. Require minimum 10% voting power behind accepted prices
3. Add geographic diversity as secondary check (Option B)
4. Use TWAP for high-value operations (Option C)

## Technical Details

**Affected Files:**
- `x/oracle/keeper/aggregation.go`
- `x/oracle/types/params.go` (add MinVotingPowerForConsensus)

**Database Changes:** None

## Acceptance Criteria

- [ ] Voting power threshold replaces validator count
- [ ] Minimum voting power configurable via governance
- [ ] Geographic diversity enforced (already in params, needs implementation)
- [ ] Test: 3 validators with 1% stake each can't manipulate price
- [ ] Test: validators with >10% combined stake set price
- [ ] Test: manipulation attempt triggers circuit breaker

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- Related: ROADMAP HIGH-2 (Oracle Outlier Detection) - marked completed but this is a different issue
- Related: Geographic Diversity Enforcement (also flagged as missing)
- Chainlink: Uses stake-weighted aggregation
