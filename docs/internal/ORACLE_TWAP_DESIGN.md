# Oracle TWAP Fallback Design (SEC-3.8)

Multiple TWAP methods for flash-loan resistance. See `x/oracle/keeper/twap_advanced.go`.

## TWAP Methods (Priority Order)

| Priority | Method | Description | Min Samples |
|----------|--------|-------------|-------------|
| 1 | Standard TWAP | Basic time-weighted average | 1 |
| 2 | Volume-Weighted | Weights by estimated volume | 1 |
| 3 | Exponential (EWMA) | 30% recent, 70% historical | 1 |
| 4 | Trimmed | Removes top/bottom 10% | 4 |
| 5 | Kalman Filter | Optimal noise estimation | 2 |

## Fallback Logic

`GetRobustTWAP()` collects all successful results:
- 1 succeeds: use it
- Multiple succeed: median of all, confidence from Kalman

## Configuration (governance-configurable)

- Lookback: `params.TwapLookbackWindow` blocks
- EWMA alpha: 0.3; Trim: 10% each end
- Kalman: process=0.01, measure=0.1

## Consistency

`ValidateTWAPConsistency()`: CV < 5% = consistent; high divergence = manipulation risk

## Errors

- All fail: error; Insufficient snapshots: skip method; Overflow: error

## Security

- Flash-loan resistant via multiple methods
- Outlier rejection via trimmed TWAP
- Confidence scoring via Kalman filter
