# MEV Protection Documentation

## Overview

The PAW blockchain implements comprehensive Maximal Extractable Value (MEV) protection mechanisms to ensure fair trading and prevent exploitation of users through front-running, sandwich attacks, and other MEV strategies.

## Table of Contents

1. [What is MEV?](#what-is-mev)
2. [Implemented Protection Mechanisms](#implemented-protection-mechanisms)
3. [Configuration](#configuration)
4. [Detection Algorithms](#detection-algorithms)
5. [Future Enhancements](#future-enhancements)
6. [Monitoring and Metrics](#monitoring-and-metrics)
7. [API Reference](#api-reference)

## What is MEV?

Maximal Extractable Value (MEV) refers to the maximum value that can be extracted from block production beyond the standard block rewards and gas fees by including, excluding, or reordering transactions within a block.

### Common MEV Attack Types

1. **Front-Running**: A malicious actor observes a pending transaction and submits their own transaction with a higher gas fee to be executed first.

2. **Sandwich Attacks**: An attacker places two transactions around a victim's transaction:
   - **Front-run transaction**: Large buy that increases the price
   - **Victim transaction**: User's trade executes at worse price
   - **Back-run transaction**: Attacker sells for profit

3. **Back-Running**: Executing a transaction immediately after a target transaction to profit from the price movement.

4. **Transaction Reordering**: Validators manipulating transaction order for profit.

## Implemented Protection Mechanisms

### 1. Timestamp-Based Transaction Ordering

**Status**: ✅ Implemented

Fair ordering based on transaction submission timestamps prevents validators from arbitrarily reordering transactions.

**How it works**:

- Each transaction includes a timestamp
- Transactions are validated to ensure timestamps are reasonable
- Ordering is enforced within a configurable time window
- Prevents validators from placing their transactions ahead of earlier submissions

**Configuration**:

```go
EnableTimestampOrdering: true
MaxReorderingWindow: 30 seconds
```

**Benefits**:

- Prevents arbitrary transaction reordering
- Ensures first-come-first-served fairness
- Transparent and auditable ordering

**Limitations**:

- Network latency can affect timestamp accuracy
- Requires honest timestamp reporting from users

### 2. Sandwich Attack Detection

**Status**: ✅ Implemented

Sophisticated pattern detection identifies and blocks sandwich attacks in real-time.

**Detection Algorithm**:

The system analyzes transaction patterns looking for:

1. **Large buy transaction** (front-run)
   - Same token pair as victim
   - Significantly larger than average (configurable ratio)
   - Within detection time window

2. **Victim transaction**
   - Normal-sized trade
   - Same direction as front-run
   - Follows front-run closely in time

3. **Large sell transaction** (back-run)
   - Opposite direction to front-run
   - Similar size to front-run
   - Completes the sandwich pattern

**Confidence Scoring**:

```
Confidence = (SizeRatio × 0.6) + (TimeProximity × 0.4)

Where:
- SizeRatio: Normalized ratio of attacker trade to victim trade
- TimeProximity: How close transactions are in time (0-1 scale)
```

**Threshold**: Confidence > 70% triggers blocking

**Configuration**:

```go
EnableSandwichDetection: true
SandwichDetectionWindow: 60 seconds
SandwichMinRatio: 2.0x
```

**Actions Taken**:

- Transaction rejected with clear error message
- Event emitted for monitoring
- Pattern recorded for analysis
- Metrics updated

### 3. Price Impact Limits

**Status**: ✅ Enhanced

Maximum price impact per transaction prevents excessive slippage and market manipulation.

**How it works**:

Price impact is calculated as:

```
PriceImpact = |1 - (actualOutput / expectedOutput)|

Where:
- expectedOutput = amountIn × (reserveOut / reserveIn)
- actualOutput = calculated AMM output
```

**Default Limit**: 5% maximum price impact

**Configuration**:

```go
EnablePriceImpactLimits: true
MaxPriceImpact: 0.05 (5%)
```

**Benefits**:

- Protects users from excessive slippage
- Prevents large manipulative trades
- Limits market impact of single transactions

**Example**:

```
Pool: 1000 ATOM / 2000 USDC
User swaps: 200 ATOM

Expected output (no slippage): 400 USDC
Actual AMM output: 364 USDC (due to curve)
Price Impact: (400 - 364) / 400 = 9%

Result: REJECTED (exceeds 5% limit)
```

### 4. Front-Running Detection

**Status**: ✅ Implemented (Monitoring Mode)

Detects potential front-running patterns and flags them for review.

**Detection Algorithm**:

- Identifies large trades in same direction
- Checks time proximity (< 10 seconds)
- Calculates confidence score
- Logs pattern but doesn't automatically block

**Configuration**:

```go
Threshold: 50% confidence
Action: Flag for review (not automatic blocking)
```

**Rationale**: Front-running can be legitimate in some cases, so we monitor and flag rather than automatically block.

### 5. Transaction Recording and Analysis

**Status**: ✅ Implemented

All DEX transactions are recorded for MEV pattern analysis.

**Recorded Data**:

- Transaction hash
- Trader address
- Pool ID
- Tokens and amounts
- Timestamp and block height
- Price impact
- Transaction index

**Storage**:

- Recent transactions cached (100 per pool)
- Historical data available for analysis
- Automatic cleanup of old records (> 1000 blocks)

**Uses**:

- Pattern detection
- Historical analysis
- Auditing
- Metrics and reporting

## Configuration

### MEV Protection Configuration Structure

```go
type MEVProtectionConfig struct {
    // Core Features (Implemented)
    EnableTimestampOrdering bool          // Enable timestamp-based ordering
    EnableSandwichDetection bool          // Enable sandwich attack detection
    EnablePriceImpactLimits bool          // Enable price impact limits

    // Parameters
    MaxPriceImpact          math.LegacyDec  // Max price impact (default: 5%)
    SandwichDetectionWindow int64           // Detection window (default: 60s)
    SandwichMinRatio        math.LegacyDec  // Min size ratio (default: 2.0x)
    MaxReorderingWindow     int64           // Reordering window (default: 30s)

    // Future Features (Documented, Not Implemented)
    EnableCommitReveal      bool            // Commit-reveal scheme
    CommitRevealTimeout     uint64          // Commit-reveal timeout
    EnableMempoolEncryption bool            // Mempool encryption
    EnableBatching          bool            // Transaction batching
    BatchingThreshold       uint64          // Batching threshold
}
```

### Default Configuration

```go
config := types.DefaultMEVProtectionConfig()
// Returns:
{
    EnableTimestampOrdering: true,
    EnableSandwichDetection: true,
    EnablePriceImpactLimits: true,
    MaxPriceImpact:          0.05,  // 5%
    SandwichDetectionWindow: 60,    // 60 seconds
    SandwichMinRatio:        2.0,   // 2x
    MaxReorderingWindow:     30,    // 30 seconds
    EnableCommitReveal:      false, // Future
    CommitRevealTimeout:     10,    // Future
    EnableMempoolEncryption: false, // Future
    EnableBatching:          false, // Future
    BatchingThreshold:       5,     // Future
}
```

### Updating Configuration

Configuration can be updated through governance proposals:

```go
// Example: Update max price impact
config := keeper.GetMEVProtectionConfig(ctx)
config.MaxPriceImpact = math.LegacyNewDecWithPrec(3, 2) // 3%
keeper.SetMEVProtectionConfig(ctx, config)
```

## Detection Algorithms

### Sandwich Attack Detection Algorithm

```
FUNCTION DetectSandwichAttack(currentTx):
    recentTxs = GetRecentTransactions(pool, detectionWindow)

    FOR each tx IN recentTxs:
        IF IsSameDirection(tx, currentTx):
            IF IsLargeTrade(tx, currentTx, minRatio):
                potentialFrontRun = tx

                // Look for victim between front-run and current
                FOR each victimTx IN recentTxs:
                    IF victimTx.timestamp > potentialFrontRun.timestamp
                       AND victimTx.timestamp < currentTx.timestamp
                       AND IsSameDirection(victimTx, potentialFrontRun)
                       AND IsSmallTrade(victimTx, potentialFrontRun):

                        // Current tx is potential back-run
                        IF IsOppositeDirection(currentTx, potentialFrontRun):
                            confidence = CalculateConfidence(
                                potentialFrontRun,
                                victimTx,
                                currentTx
                            )

                            IF confidence > 0.7:
                                RETURN SandwichDetected(
                                    frontRun: potentialFrontRun,
                                    victim: victimTx,
                                    backRun: currentTx,
                                    confidence: confidence
                                )

    RETURN NoSandwich
```

### Confidence Score Calculation

```
FUNCTION CalculateConfidence(frontRun, victim, backRun):
    // Size ratio component (60% weight)
    sizeRatio = frontRun.amount / victim.amount
    normalizedSizeRatio = min(sizeRatio / 10, 1.0)

    // Time proximity component (40% weight)
    totalWindow = backRun.timestamp - frontRun.timestamp
    victimDelay = victim.timestamp - frontRun.timestamp
    timeProximity = 1.0 - (victimDelay / totalWindow)

    // Combined confidence
    confidence = (normalizedSizeRatio × 0.6) + (timeProximity × 0.4)

    RETURN confidence
```

### Price Impact Calculation

```
FUNCTION CalculatePriceImpact(reserveIn, reserveOut, amountIn, amountOut):
    // Calculate expected output without AMM curve impact
    expectedOut = amountIn × (reserveOut / reserveIn)

    // Calculate actual impact
    priceImpact = |expectedOut - amountOut| / expectedOut

    RETURN priceImpact
```

## Future Enhancements

### 1. Commit-Reveal Scheme

**Status**: 📋 Planned (Requires Consensus Changes)

**Concept**:

- Users submit transaction commitments (hashes)
- Transactions hidden until reveal phase
- Prevents validators from seeing transaction details before inclusion

**Implementation Requirements**:

- Two-phase transaction protocol
- Consensus modifications
- Timeout handling
- Penalty for non-reveal

**Benefits**:

- Complete transaction privacy in mempool
- Eliminates mempool front-running
- Validators cannot see transaction details before commitment

**Challenges**:

- Requires significant protocol changes
- Increases transaction latency (two phases)
- Complex timeout and penalty logic

### 2. Mempool Encryption

**Status**: 📋 Planned (Requires Infrastructure Changes)

**Concept**:

- Encrypt transactions in mempool
- Threshold decryption after block proposal
- Prevents transaction detail leakage

**Implementation Requirements**:

- Threshold cryptography system
- Key management infrastructure
- Validator coordination
- Decryption verification

**Benefits**:

- Strong privacy guarantees
- Prevents all mempool-based MEV
- Maintains decentralization

**Challenges**:

- Complex cryptographic setup
- Performance overhead
- Key management complexity
- Requires validator coordination

### 3. Transaction Batching and Atomic Execution

**Status**: 📋 Planned

**Concept**:

- Batch related transactions together
- Atomic execution (all or nothing)
- Prevents partial execution MEV

**Use Cases**:

- Multi-step DeFi operations
- Complex trading strategies
- Cross-pool arbitrage

**Benefits**:

- Eliminates partial execution vulnerabilities
- Enables complex atomic operations
- Better user experience

**Implementation**:

```go
type TransactionBatch struct {
    BatchID      string
    Transactions []Transaction
    Status       BatchStatus
}
```

### 4. Fair Sequencing Service (FSS)

**Status**: 💡 Research Phase

**Concept**:

- Dedicated fair ordering service
- Cryptographic proof of ordering
- Decentralized sequencer network

**Benefits**:

- Provably fair ordering
- Independent of validators
- Auditable and transparent

### 5. MEV Redistribution

**Status**: 💡 Research Phase

**Concept**:

- Capture MEV at protocol level
- Redistribute to stakeholders
- Align incentives

**Mechanisms**:

- Protocol-level MEV capture
- Redistribution to liquidity providers
- Treasury allocation for development

## Monitoring and Metrics

### MEV Protection Metrics

The system tracks comprehensive metrics:

```go
type MEVProtectionMetrics struct {
    TotalTransactions           uint64  // Total processed
    TotalMEVDetected           uint64  // Total attacks detected
    TotalMEVBlocked            uint64  // Total attacks blocked
    SandwichAttacksDetected    uint64  // Sandwich attacks
    FrontRunningDetected       uint64  // Front-running
    BackRunningDetected        uint64  // Back-running
    PriceImpactViolations      uint64  // Price impact exceeded
    TimestampOrderingEnforced  uint64  // Ordering enforcements
    LastUpdated                int64   // Last update time
}
```

### Accessing Metrics

```bash
# Query MEV metrics
pawchaind query dex mev-metrics

# Example output:
{
  "total_transactions": 10000,
  "total_mev_detected": 45,
  "total_mev_blocked": 32,
  "sandwich_attacks_detected": 25,
  "front_running_detected": 15,
  "price_impact_violations": 5,
  "detection_rate": "0.45%",
  "block_rate": "71.11%"
}
```

### Events

All MEV-related events are emitted for monitoring:

#### Sandwich Attack Event

```json
{
  "type": "sandwich_attack_detected",
  "attributes": {
    "pool_id": "1",
    "attacker": "paw1...",
    "front_run_tx": "hash1...",
    "victim_tx": "hash2...",
    "confidence": "0.85",
    "blocked": "true"
  }
}
```

#### Front-Running Event

```json
{
  "type": "front_running_detected",
  "attributes": {
    "pool_id": "1",
    "trader": "paw1...",
    "suspected_front_run_tx": "hash...",
    "confidence": "0.65"
  }
}
```

#### Price Impact Exceeded Event

```json
{
  "type": "price_impact_exceeded",
  "attributes": {
    "pool_id": "1",
    "price_impact": "0.08",
    "max_allowed": "0.05"
  }
}
```

#### MEV Blocked Event

```json
{
  "type": "mev_attack_blocked",
  "attributes": {
    "pool_id": "1",
    "trader": "paw1...",
    "attack_type": "sandwich_attack",
    "confidence": "0.85",
    "reason": "sandwich attack detected: ..."
  }
}
```

### Monitoring Dashboard

Recommended metrics to track:

1. **Detection Rate**: MEV detected / Total transactions
2. **Block Rate**: MEV blocked / MEV detected
3. **False Positive Rate**: Requires manual review
4. **Attack Type Distribution**: Pie chart of attack types
5. **Time Series**: MEV attacks over time
6. **Pool-Specific Metrics**: Which pools see most MEV

## API Reference

### Keeper Methods

#### GetMEVProtectionConfig

```go
func (k Keeper) GetMEVProtectionConfig(ctx sdk.Context) types.MEVProtectionConfig
```

Retrieves current MEV protection configuration.

#### SetMEVProtectionConfig

```go
func (k Keeper) SetMEVProtectionConfig(ctx sdk.Context, config types.MEVProtectionConfig) error
```

Updates MEV protection configuration (governance only).

#### DetectMEVAttack

```go
func (mpm *MEVProtectionManager) DetectMEVAttack(
    ctx sdk.Context,
    trader string,
    poolID uint64,
    tokenIn, tokenOut string,
    amountIn, amountOut math.Int,
    priceImpact math.LegacyDec,
) types.MEVDetectionResult
```

Performs comprehensive MEV attack detection.

#### RecordTransaction

```go
func (k Keeper) RecordTransaction(
    ctx sdk.Context,
    txHash string,
    trader string,
    poolID uint64,
    tokenIn, tokenOut string,
    amountIn, amountOut math.Int,
    priceImpact math.LegacyDec,
)
```

Records transaction for MEV pattern analysis.

#### GetMEVMetrics

```go
func (k Keeper) GetMEVMetrics(ctx sdk.Context) types.MEVProtectionMetrics
```

Retrieves MEV protection metrics.

### Transaction Flow

```
User Transaction
    ↓
Timestamp Validation
    ↓
Order Enforcement
    ↓
Pool Validation
    ↓
Calculate Price Impact
    ↓
MEV Detection
    ↓
    ├─→ [BLOCKED] → Return Error + Emit Event
    ↓
Execute Swap
    ↓
Record Transaction
    ↓
Update Metrics
    ↓
Emit Events
    ↓
Return Success
```

## Best Practices

### For Users

1. **Set Appropriate Slippage**: Use reasonable slippage tolerances (1-5%)
2. **Monitor Transaction Status**: Check for MEV-related rejections
3. **Timing**: Be aware that timestamp-based ordering is enforced
4. **Large Trades**: Split large trades to reduce price impact

### For Developers

1. **Test MEV Scenarios**: Include MEV attack tests
2. **Monitor Events**: Subscribe to MEV-related events
3. **Handle Rejections**: Gracefully handle MEV blocking errors
4. **Update Configs**: Participate in governance for configuration updates

### For Validators

1. **Honest Timestamps**: Report accurate block times
2. **Fair Ordering**: Don't attempt to manipulate transaction order
3. **Monitor Metrics**: Track MEV detection rates
4. **Stay Updated**: Keep node software current with MEV protections

## Security Considerations

### Attack Vectors

1. **Timestamp Manipulation**: Users may try to fake timestamps
   - **Mitigation**: Timestamp validation with reasonable bounds

2. **Detection Evasion**: Attackers may try to evade detection
   - **Mitigation**: Multi-layered detection, continuous algorithm updates

3. **False Positives**: Legitimate trades might be flagged
   - **Mitigation**: Confidence thresholds, human review process

4. **Configuration Attacks**: Malicious governance proposals
   - **Mitigation**: Multi-sig governance, timelock, community review

### Audit Trail

All MEV-related decisions are:

- Logged in events
- Stored in state (patterns)
- Tracked in metrics
- Auditable on-chain

## Testing

### Test Scenarios

1. **Normal Trading**: Verify no false positives
2. **Sandwich Attacks**: Verify detection and blocking
3. **Front-Running**: Verify detection and flagging
4. **Price Impact**: Verify limit enforcement
5. **Timestamp Ordering**: Verify ordering enforcement

### Example Test Cases

```go
// Test sandwich attack detection
func TestSandwichAttackDetection(t *testing.T) {
    // Setup: Create pool
    // 1. Execute large buy (front-run)
    // 2. Execute normal trade (victim)
    // 3. Execute large sell (back-run)
    // Assert: Back-run is blocked
}

// Test price impact limit
func TestPriceImpactLimit(t *testing.T) {
    // Setup: Small pool
    // Execute: Large trade (>5% impact)
    // Assert: Transaction rejected
}
```

## Conclusion

PAW blockchain's MEV protection provides comprehensive safeguards against common MEV attack vectors while maintaining efficiency and usability. The implemented mechanisms (timestamp ordering, sandwich detection, price impact limits) provide immediate protection, while planned enhancements (commit-reveal, mempool encryption) will further strengthen defenses.

For questions or contributions, please contact the development team or submit issues/PRs to the repository.

---

**Last Updated**: 2025
**Version**: 1.0
**Status**: Active Protection
