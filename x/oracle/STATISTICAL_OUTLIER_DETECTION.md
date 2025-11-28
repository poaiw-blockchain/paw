# Statistical Outlier Detection - Institutional Grade Implementation

## Overview

The PAW oracle module implements sophisticated statistical outlier detection matching the standards of institutional oracle systems like Chainlink and Band Protocol. This system uses multi-stage statistical analysis to identify and filter malicious or erroneous price submissions while maintaining robustness against false positives.

## Multi-Stage Detection Pipeline

### Stage 1: Modified Z-Score (MAD-based)
**Method**: Median Absolute Deviation (MAD)
**Purpose**: Detect extreme outliers using robust statistics

The Modified Z-Score method uses MAD instead of standard deviation because:
- MAD is resistant to outliers (doesn't use mean)
- More accurate for non-normal distributions
- Computationally efficient for blockchain

**Formula**:
```
MAD = median(|Xi - median(X)|) × 1.4826
Modified Z-Score = 0.6745 × |price - median| / MAD
```

**Severity Classification**:
- **Extreme** (>5 sigma): Modified Z-Score ≥ threshold × 1.4
- **High** (>3.5 sigma): Modified Z-Score ≥ threshold
- **Moderate** (>2.5 sigma): Modified Z-Score ≥ threshold × 0.7
- **Low** (>1.75 sigma): Modified Z-Score ≥ threshold × 0.5

### Stage 2: Interquartile Range (IQR)
**Method**: Box plot outlier detection
**Purpose**: Detect moderate outliers using quartile ranges

**Formula**:
```
Q1 = 25th percentile
Q3 = 75th percentile
IQR = Q3 - Q1
Lower Bound = Q1 - (1.5 × IQR)
Upper Bound = Q3 + (1.5 × IQR)
```

**Volatility Adjustment**:
```
Adjusted Multiplier = 1.5 + (volatility × 5.0)
Capped at 3.0 for extreme volatility
```

### Stage 3: Grubbs' Test
**Method**: Statistical hypothesis test
**Purpose**: Single outlier detection for remaining suspicious values

**Requirements**:
- Minimum 7 data points
- Assumes approximately normal distribution
- Uses alpha = 0.05 significance level

**Formula**:
```
G = |price - mean| / std_dev
Critical Value ≈ (n-1)/√n × √(t²/(n-2+t²))

Outlier if: G > Critical Value
```

### Stage 4: Volatility-Adjusted Acceptance
**Method**: Dynamic threshold adjustment
**Purpose**: Adapt to asset-specific characteristics

**Volatility Calculation**:
```
Returns = (price[i] - price[i-1]) / price[i-1]
Volatility = std_dev(returns over 100 blocks)
```

**Threshold Adjustment**:
```
Base Threshold = 3.5
Adjustment Factor = 1.0 + (volatility × 10)
Final Threshold = Base Threshold × Adjustment Factor
Capped between 3.5 and 10.5
```

## Volatility Analysis

### Rolling Volatility Calculation
The system calculates rolling 100-block volatility for each asset:

```go
1. Collect price snapshots (last 100 blocks)
2. Calculate returns: ret[i] = (price[i] - price[i-1]) / price[i-1]
3. Calculate mean of returns
4. Calculate variance: Σ(ret[i] - mean)² / n
5. Standard deviation: √variance
6. Clamp between 0.01 and 1.0
```

### Asset Classification Examples
- **Stablecoins** (volatility ~0.01): Low tolerance, strict outlier detection
- **Major Crypto** (volatility ~0.05-0.10): Moderate tolerance
- **Volatile Crypto** (volatility >0.20): High tolerance, relaxed thresholds

## Severity-Based Slashing

### Slash Fractions

| Severity | First Offense | Repeated Offender | Jail |
|----------|---------------|-------------------|------|
| **Extreme** | 0.05% | 0.10% (2x) | Yes |
| **High** | 0.02% | 0.04% (2x) | Only if repeated |
| **Moderate** | No slash (grace) | 0.01% | No |
| **Low** | No slash (grace) | 0.005% (only 6+ outliers) | No |

### Grace Period
- **Duration**: 1000 blocks (~2 hours at 6s blocks)
- **Applies to**: First offense with severity < High
- **Purpose**: Avoid punishing validators for temporary data source issues

### Repeated Offender Logic
- **Threshold**: 3 outliers within 1000 blocks
- **Penalty**: 2x slash fraction
- **Jail**: Automatic for moderate+ severity
- **Reset**: History older than 1000 blocks is dropped

## Reputation Tracking

### Reputation Score Calculation
```
Penalty Points = Σ(severity_weight[outlier])

Weights:
- Extreme: 1.0
- High: 0.5
- Moderate: 0.25
- Low: 0.1

Reputation = 1 / (1 + penalty_points)
```

**Score Interpretation**:
- 1.0: Perfect (no outliers)
- 0.67: One moderate outlier
- 0.50: One extreme outlier
- 0.33: Multiple severe outliers

### Use Cases
- Validator selection for sensitive operations
- Dynamic reward multipliers
- Governance voting weight adjustment

## Edge Case Handling

### Low Validator Count
**Problem**: Statistical tests need sufficient sample size
**Solution**: Keep minimum 3 closest prices to median

```go
if len(validPrices) < 3 && len(allPrices) >= 3 {
    validPrices = keepClosestToMedian(allPrices, median, 3)
}
```

### Identical Prices (MAD = 0)
**Problem**: Division by zero in Modified Z-Score
**Solution**: Any different price is an extreme outlier

```go
if mad.IsZero() {
    if !price.Equal(median) {
        return SeverityExtreme
    }
}
```

### High Volatility Events
**Problem**: Legitimate price movements flagged as outliers
**Solution**: Dynamic threshold adjustment based on recent volatility

```go
threshold = baseThreshold × (1.0 + volatility × 10)
```

### Insufficient Historical Data
**Problem**: Can't calculate volatility for new assets
**Solution**: Default to moderate volatility (5%)

```go
if len(snapshots) < 2 {
    return LegacyDec("0.05") // 5% default
}
```

## Gas Optimization

### Efficient Algorithms
- **Median**: O(n log n) using quickselect
- **MAD**: Single pass after median calculation
- **IQR**: Reuses sorted array from median
- **Grubbs**: Only for n ≥ 7 to justify computation

### Storage Strategy
- Outlier history: Keyed by validator+asset+height
- Automatic cleanup of old history (>1000 blocks)
- Minimal state storage per outlier

### Computation Limits
- Volatility window: 100 blocks max
- Statistical tests: Only when n ≥ 3
- Grubbs test: Only when n ≥ 7

## Event Emissions

### Outlier Detection Event
```
Event: oracle_outlier_detected
Attributes:
- validator: validator address
- asset: asset identifier
- price: submitted price
- severity: 0-4 (none to extreme)
- deviation: absolute deviation from median
- reason: detection method used
- median: current median price
- mad: current MAD value
```

### Slash Event
```
Event: oracle_slash_outlier
Attributes:
- validator: validator address
- asset: asset identifier
- severity: outlier severity
- slash_fraction: actual slash amount
- jailed: true/false
- price: submitted price
- deviation: deviation from median
- reason: why slashed
- block_height: when slashed
```

### Aggregation Event
```
Event: price_aggregated
Attributes:
- asset: asset identifier
- price: final aggregated price
- num_validators: validators included
- num_outliers: outliers filtered
- median: median of valid prices
- mad: MAD of valid prices
```

## Comparison to Industry Standards

### Chainlink
- **Similarity**: Uses median-based aggregation
- **PAW Advantage**: More sophisticated outlier detection with MAD
- **Chainlink Advantage**: More data sources

### Band Protocol
- **Similarity**: Multi-stage validation
- **PAW Advantage**: Volatility-adjusted thresholds
- **Band Advantage**: Cross-chain validation

### Tellor
- **Similarity**: Dispute-based governance
- **PAW Advantage**: Automatic statistical detection
- **Tellor Advantage**: User-initiated disputes

## Mathematical Soundness

### Why MAD over Standard Deviation?
1. **Robustness**: Not influenced by outliers
2. **Breakdown Point**: 50% vs 0% for std dev
3. **Efficiency**: Similar computation, better results

### Why Modified Z-Score?
1. **Scale Consistency**: 0.6745 factor normalizes to std dev
2. **Distribution-Free**: Works for non-normal data
3. **Threshold Clarity**: 3.5 ≈ 99.9% confidence interval

### Why IQR?
1. **Complement to MAD**: Catches moderate outliers
2. **Visual Intuition**: Matches box plot analysis
3. **Adjustability**: Easy to tune with volatility

### Why Grubbs' Test?
1. **Statistical Rigor**: Formal hypothesis testing
2. **Single Outlier**: Optimized for one bad actor
3. **False Positive Control**: Alpha parameter tunable

## Configuration Parameters

### Default Values
```
Base MAD Threshold: 3.5
IQR Multiplier: 1.5
Grubbs Alpha: 0.05
Volatility Window: 100 blocks
Reputation Window: 1000 blocks
Grace Period: 1000 blocks
Repeated Offender: 3 outliers
```

### Tuning Guidelines
- **Conservative** (stablecoins): Lower thresholds, shorter grace
- **Standard** (major crypto): Default parameters
- **Permissive** (volatile assets): Higher thresholds, longer grace

## Future Enhancements

### Asset-Specific Configuration
Planned feature to allow per-asset parameter tuning:
```go
type AssetConfig struct {
    Asset              string
    VolatilityFactor   LegacyDec  // Multiplier for thresholds
    OutlierThreshold   LegacyDec  // Base MAD multiplier
    RequiredValidators uint32     // Minimum submissions
    GracePeriod        uint64     // Blocks before slashing
}
```

### Machine Learning Integration
- Historical pattern recognition
- Validator behavior profiling
- Adaptive threshold learning

### Cross-Asset Correlation
- Detect coordinated manipulation
- Correlation-based outlier detection
- Multi-asset reputation scoring

## Testing Recommendations

### Unit Tests
- Test each statistical function independently
- Verify edge cases (n=0, n=1, n=2, n=3+)
- Test volatility calculations
- Test reputation scoring

### Integration Tests
- Simulate various price distributions
- Test multi-validator scenarios
- Test slashing conditions
- Test grace period behavior

### Stress Tests
- High validator count (100+)
- Extreme volatility scenarios
- Coordinated attacks (Byzantine)
- Network partition scenarios

## References

1. **Modified Z-Score**: Iglewicz, B. and Hoaglin, D. (1993). "Volume 16: How to Detect and Handle Outliers", The ASQC Basic References in Quality Control: Statistical Techniques.

2. **Grubbs' Test**: Grubbs, F. E. (1950). "Sample Criteria for Testing Outlying Observations". Annals of Mathematical Statistics.

3. **MAD**: Rousseeuw, P. J. and Croux, C. (1993). "Alternatives to the Median Absolute Deviation". Journal of the American Statistical Association.

4. **Blockchain Oracles**: Caldarelli, G. (2020). "Understanding the blockchain oracle problem: A call for action". Information.
