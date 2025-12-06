# Oracle Module

## Overview

The Oracle module provides a secure, decentralized price feed system for the PAW blockchain. It aggregates price data from validators, computes time-weighted average prices (TWAP), and provides manipulation-resistant price feeds for use by other modules (DEX, lending protocols, derivatives, etc.).

## Concepts

### Decentralized Price Oracle

Unlike centralized oracles that rely on single data providers, the Oracle module uses a validator-based consensus mechanism:

- **Validators Submit Prices**: Active validators submit price data for various assets
- **Weighted Voting**: Prices are weighted by validator stake
- **Consensus Aggregation**: Module aggregates submissions to determine canonical prices
- **Geographic Diversity**: Requires minimum regional distribution for security
- **Slashing Protection**: Validators who submit incorrect data face penalties

### Price Consensus

Price determination follows a multi-stage process:

1. **Submission Phase**: Validators submit prices during vote period
2. **Aggregation Phase**: Module computes median/weighted average
3. **Validation Phase**: Outliers are detected and excluded
4. **Finalization Phase**: Canonical price is stored and emitted

### Time-Weighted Average Price (TWAP)

TWAP provides manipulation-resistant pricing:

```
TWAP = (Σ(price_i * duration_i)) / total_duration
```

Benefits:
- Resists flash loan attacks
- Smooths out temporary price spikes
- Provides reliable pricing for DeFi protocols
- Configurable lookback window (default: 1000 blocks)

### Slashing Mechanism

Validators who submit inaccurate prices are subject to slashing:

- **Miss Rate Tracking**: Tracks submissions vs requirements
- **Slash Window**: Rolling window for tracking (default: 10,000 blocks)
- **Minimum Valid**: Minimum valid submissions per window (default: 100)
- **Slash Fraction**: Percentage of stake slashed (default: 1%)
- **Jail Duration**: Validators may be temporarily jailed

### Geographic Distribution

To prevent regional manipulation, the oracle requires:
- **Allowed Regions**: Whitelist of geographic regions
- **Minimum Regions**: Minimum regional diversity (default: 1)
- **Voting Power**: Minimum voting power for consensus (default: 10%)

This ensures prices reflect global market data, not localized manipulation.

### IBC Price Feeds

Cross-chain price synchronization:
- **Price Packets**: IBC packets carrying price data
- **Authorized Channels**: Whitelist of trusted chains
- **Bidirectional Sync**: Both send and receive price updates
- **Timeout Handling**: Graceful handling of failed transmissions

## State

The module stores the following data:

### Prices
- **Key**: `prices/{asset}`
- **Value**: Price struct
- **Description**: Current canonical price for each asset

```go
type Price struct {
    Asset          string          // e.g., "BTC/USD"
    Price          math.LegacyDec  // Current price
    LastUpdateTime int64           // Block time of last update
    BlockHeight    int64           // Block height of last update
    Source         string          // Price source identifier
}
```

### Validator Prices
- **Key**: `validator_prices/{asset}/{validator}`
- **Value**: ValidatorPrice struct
- **Description**: Individual validator price submissions

```go
type ValidatorPrice struct {
    Asset          string
    Validator      string          // Validator address
    Price          math.LegacyDec
    SubmittedAt    int64
    Power          math.Int        // Validator voting power
}
```

### Validator Oracles
- **Key**: `validator_oracles/{validator}`
- **Value**: ValidatorOracle struct
- **Description**: Validator oracle participation and performance tracking

```go
type ValidatorOracle struct {
    Validator        string
    MissCounter      uint64  // Missed submissions in window
    SubmitCounter    uint64  // Total submissions in window
    Region           string  // Geographic region
    LastSubmitHeight int64
}
```

### Price Snapshots (TWAP)
- **Key**: `price_snapshots/{asset}/{blockHeight}`
- **Value**: PriceSnapshot struct
- **Description**: Historical price data for TWAP calculation

```go
type PriceSnapshot struct {
    Asset          string
    Price          math.LegacyDec
    Timestamp      int64
    BlockHeight    int64
    AccumulatorValue math.LegacyDec  // Cumulative price*time
}
```

### Parameters
- **Key**: `params`
- **Value**: Params struct

```go
type Params struct {
    VotePeriod                  uint64          // 30 blocks
    VoteThreshold               math.LegacyDec  // 0.67 (67%)
    SlashFraction               math.LegacyDec  // 0.01 (1%)
    SlashWindow                 uint64          // 10000 blocks
    MinValidPerWindow           uint64          // 100 submissions
    TwapLookbackWindow          uint64          // 1000 blocks
    AuthorizedChannels          []AuthorizedChannel
    AllowedRegions              []string
    MinGeographicRegions        uint64
    MinVotingPowerForConsensus  math.LegacyDec  // 0.10 (10%)
}
```

## Messages

### MsgSubmitPrice

Validators submit price data for an asset.

**CLI Command:**
```bash
pawd tx oracle submit-price \
  --asset "BTC/USD" \
  --price 65000.50 \
  --region "na" \
  --from validator \
  --chain-id paw-1
```

**Validation:**
- Sender must be an active validator
- Asset must be supported
- Price must be positive
- Region must be in allowed list
- Cannot submit duplicate in same vote period

**Access Control:**
- Only active validators can submit prices
- Must be called by validator operator key

### MsgAggregatePrices

Trigger manual price aggregation (typically done automatically in EndBlock).

**CLI Command:**
```bash
pawd tx oracle aggregate-prices \
  --from validator \
  --chain-id paw-1
```

**Access Control:**
- Permissionless (anyone can trigger)
- Useful for testing or recovery scenarios

## Queries

### Query Price

Get the current canonical price for an asset.

```bash
pawd query oracle price "BTC/USD"
```

**Response:**
```json
{
  "price": {
    "asset": "BTC/USD",
    "price": "65000.500000000000000000",
    "last_update_time": "1701234567",
    "block_height": "1234567",
    "source": "consensus"
  }
}
```

### Query Prices

List all current prices with pagination.

```bash
pawd query oracle prices --page 1 --limit 50
```

### Query TWAP

Get time-weighted average price over a specific window.

```bash
pawd query oracle twap "BTC/USD" --window 1000
```

**Response:**
```json
{
  "asset": "BTC/USD",
  "twap": "64987.250000000000000000",
  "start_height": "1233567",
  "end_height": "1234567",
  "data_points": 1000
}
```

### Query Validator Price

Get a specific validator's submitted price.

```bash
pawd query oracle validator-price "BTC/USD" pawvaloper1abc...xyz
```

### Query Validator Prices

List all validator submissions for an asset.

```bash
pawd query oracle validator-prices "BTC/USD"
```

### Query Validator Oracle Info

Get validator oracle participation statistics.

```bash
pawd query oracle validator-oracle pawvaloper1abc...xyz
```

**Response:**
```json
{
  "validator": "pawvaloper1abc...xyz",
  "miss_counter": "5",
  "submit_counter": "995",
  "region": "na",
  "last_submit_height": "1234567",
  "miss_rate": "0.005025125628140704"
}
```

### Query Params

Get module parameters.

```bash
pawd query oracle params
```

## Events

### EventPriceSubmitted
Emitted when a validator submits a price.

**Attributes:**
- `validator`: Validator address
- `asset`: Asset identifier (e.g., "BTC/USD")
- `price`: Submitted price
- `region`: Geographic region
- `timestamp`: Submission timestamp

### EventPriceAggregated
Emitted when prices are aggregated into canonical price.

**Attributes:**
- `asset`: Asset identifier
- `canonical_price`: Final consensus price
- `num_validators`: Number of validators who submitted
- `total_power`: Total voting power in submissions
- `median_price`: Median of submitted prices
- `weighted_avg_price`: Stake-weighted average

### EventValidatorSlashed
Emitted when a validator is slashed for oracle misbehavior.

**Attributes:**
- `validator`: Slashed validator address
- `slash_fraction`: Percentage of stake slashed
- `reason`: Reason for slashing (e.g., "missed_submissions")
- `miss_rate`: Validator's miss rate

### EventPriceSnapshotCreated
Emitted when a new TWAP snapshot is created.

**Attributes:**
- `asset`: Asset identifier
- `price`: Price at snapshot
- `block_height`: Snapshot block height
- `accumulator`: Updated accumulator value

## Parameters

Module parameters can be updated via governance.

### Governance Update Example

```bash
# Create parameter change proposal
pawd tx gov submit-proposal param-change proposal.json \
  --from validator \
  --chain-id paw-1

# proposal.json
{
  "title": "Reduce Oracle Vote Period",
  "description": "Reduce vote period from 30 to 20 blocks for faster price updates",
  "changes": [
    {
      "subspace": "oracle",
      "key": "VotePeriod",
      "value": "20"
    }
  ],
  "deposit": "10000000upaw"
}
```

## Security Considerations

### Manipulation Resistance

The oracle employs multiple defense mechanisms:

**1. Stake-Weighted Consensus**
- Larger validators have proportionally larger influence
- Attackers must control significant stake

**2. Geographic Distribution**
- Requires minimum regional diversity
- Prevents localized market manipulation

**3. Outlier Detection**
- Submissions far from median are excluded
- Prevents single malicious validator impact

**4. TWAP Integration**
- Time-weighting resists flash attacks
- Smooths temporary price spikes

### Slashing Risks

Validators face slashing for:
- **High Miss Rate**: Failing to submit prices regularly
- **Extreme Deviation**: Submitting prices far from consensus (future)
- **Downtime**: Extended periods of non-participation

**Mitigation:**
- Monitor validator oracle participation
- Automate price submission processes
- Set up alerting for missed submissions
- Use reliable data sources

### Data Source Quality

Price accuracy depends on validator data sources:
- Use multiple reputable exchanges (Binance, Coinbase, Kraken)
- Implement median filtering across sources
- Monitor for exchange outages or manipulation
- Consider volume-weighted prices

## Integration Examples

### JavaScript/TypeScript
```typescript
import { SigningStargateClient } from "@cosmjs/stargate";

const client = await SigningStargateClient.connectWithSigner(
  "https://rpc.paw.network",
  signer
);

// Query current price
const price = await client.queryContractSmart(
  "oracle",
  { price: { asset: "BTC/USD" } }
);

// Query TWAP
const twap = await client.queryContractSmart(
  "oracle",
  { twap: { asset: "BTC/USD", window: 1000 } }
);
```

### Python
```python
from cosmospy import BroadcastMode, Transaction

# Query price via REST API
import requests
response = requests.get("https://api.paw.network/paw/oracle/v1/prices/BTC%2FUSD")
price_data = response.json()

# Submit price (validator only)
submit_msg = {
    "type": "paw/oracle/MsgSubmitPrice",
    "value": {
        "validator": "pawvaloper1abc...xyz",
        "asset": "BTC/USD",
        "price": "65000.50",
        "region": "na"
    }
}
```

### Go
```go
import (
    oracletypes "github.com/paw-chain/paw/x/oracle/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// Query price via keeper
price, err := k.GetPrice(ctx, "BTC/USD")
if err != nil {
    return err
}

// Query TWAP
twap, err := k.GetTWAP(ctx, "BTC/USD", 1000)
if err != nil {
    return err
}

// Use in your module logic
if amount.GT(twap.Price.MulInt64(collateral)) {
    return errors.New("insufficient collateral")
}
```

## Validator Oracle Setup

### Automated Price Submission

Validators should automate price submission:

```bash
#!/bin/bash
# oracle-feeder.sh

VALIDATOR="pawvaloper1abc...xyz"
REGION="na"
INTERVAL=30  # Submit every 30 seconds

while true; do
    # Fetch BTC price from exchanges
    BTC_PRICE=$(fetch_btc_price.py)

    # Submit to chain
    pawd tx oracle submit-price \
        --asset "BTC/USD" \
        --price "$BTC_PRICE" \
        --region "$REGION" \
        --from validator \
        --gas auto \
        --gas-adjustment 1.5 \
        --yes

    sleep $INTERVAL
done
```

### Price Data Sources

Example price aggregation script:

```python
# fetch_btc_price.py
import requests
import statistics

def get_binance_price():
    r = requests.get("https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT")
    return float(r.json()["price"])

def get_coinbase_price():
    r = requests.get("https://api.coinbase.com/v2/prices/BTC-USD/spot")
    return float(r.json()["data"]["amount"])

def get_kraken_price():
    r = requests.get("https://api.kraken.com/0/public/Ticker?pair=XBTUSD")
    return float(r.json()["result"]["XXBTZUSD"]["c"][0])

# Fetch from multiple sources
prices = [
    get_binance_price(),
    get_coinbase_price(),
    get_kraken_price()
]

# Use median to resist outliers
median_price = statistics.median(prices)
print(f"{median_price:.2f}")
```

## Monitoring

### Key Metrics

Validators should monitor:
- **Submission Rate**: Percentage of vote periods with submissions
- **Miss Counter**: Number of missed submissions in slash window
- **Price Deviation**: How far submissions deviate from consensus
- **Regional Participation**: Number of active regions

### Prometheus Metrics

The module exposes metrics:
- `oracle_price_count`: Number of tracked assets
- `oracle_validator_submissions`: Submissions per validator
- `oracle_miss_rate{validator}`: Miss rate per validator
- `oracle_price_age{asset}`: Time since last price update
- `oracle_validator_count{region}`: Validators per region

### Alerting

Set up alerts for:
```yaml
# Prometheus alert rules
groups:
  - name: oracle_alerts
    rules:
      - alert: HighMissRate
        expr: oracle_miss_rate > 0.10
        for: 1h
        annotations:
          summary: "Oracle miss rate above 10%"

      - alert: StalePrice
        expr: oracle_price_age > 300
        for: 5m
        annotations:
          summary: "Price hasn't updated in 5 minutes"
```

## Testing

### Unit Tests
```bash
# Run oracle module tests
go test ./x/oracle/...

# Test with coverage
go test -cover ./x/oracle/...

# Test specific functionality
go test ./x/oracle/keeper -run TestPriceAggregation -v
```

### Integration Tests
```bash
# Run integration tests
go test ./x/oracle/keeper -run TestIntegration -v
```

## Future Enhancements

### Planned Features
- Support for additional asset types (equities, commodities, forex)
- Advanced outlier detection algorithms
- Dynamic vote period based on market volatility
- Reputation scoring for validators
- Multi-sig oracle for critical assets

### Research Areas
- Cryptographic price commitments (commit-reveal)
- Zero-knowledge price proofs
- Cross-chain oracle aggregation
- MEV-resistant oracle design

## References

- [Chainlink Price Feeds](https://docs.chain.link/data-feeds)
- [Band Protocol](https://docs.bandchain.org/)
- [UMA Oracle](https://docs.umaproject.org/oracle/overview)
- [Cosmos SDK Slashing Module](https://docs.cosmos.network/main/modules/slashing)

---

**Module Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-06
