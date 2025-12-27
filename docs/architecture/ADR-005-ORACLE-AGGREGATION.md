# ADR-005: Oracle Price Aggregation

## Status
Accepted

## Context
DEX and other modules require reliable price data. Traditional oracle designs are vulnerable to:
- Single point of failure
- Manipulation by colluding validators
- Stale data issues

## Decision

### Aggregation Algorithm
1. Collect prices from all active validators
2. Filter outliers using median absolute deviation (MAD)
3. Calculate volume-weighted median for final price
4. Cache results per block for O(1) subsequent lookups

### Validator Participation
- Validators submit prices via `MsgSubmitPrice`
- Prices weighted by validator stake
- Minimum 2/3 participation required for valid aggregation

### Outlier Detection
```go
// Prices > 3 * MAD from median are excluded
threshold := median + (3 * medianAbsoluteDeviation)
```

### Staleness Protection
- Prices older than `MaxPriceAge` (default: 100 blocks) rejected
- Last valid price cached with timestamp
- Emergency fallback to governance-set reference price

### Performance Optimization
- `GetCachedTotalVotingPower()` avoids recalculation per asset
- Indexed price storage by asset and height
- Batch price updates in single transaction

## Consequences

**Positive:**
- Byzantine fault tolerant (up to 1/3 malicious validators)
- Gas-efficient with caching
- Outlier resistant

**Negative:**
- Requires validator coordination
- Initial price bootstrap complexity

## References
- [Chainlink Price Feeds](https://docs.chain.link/data-feeds)
- ADR-002: Security Layers
