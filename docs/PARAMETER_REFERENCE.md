# PAW Blockchain Parameter Reference

Complete reference for all configurable parameters across PAW modules.

## Quick Navigation

- [DEX Module Parameters](#dex-module-parameters)
- [Oracle Module Parameters](#oracle-module-parameters)
- [Compute Module Parameters](#compute-module-parameters)
- [How to Query Parameters](#how-to-query-parameters)
- [How to Change Parameters](#how-to-change-parameters)

---

## DEX Module Parameters

**Module**: `dex`
**Default Location**: `x/dex/keeper/params.go`

### Fee Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `swap_fee` | `sdk.Dec` | `0.003` | `0.001` | `0.01` | Total fee charged per swap (0.3%) |
| `lp_fee` | `sdk.Dec` | `0.0025` | `0.0` | `swap_fee` | Fee distributed to liquidity providers (0.25%, 83% of swap fee) |
| `protocol_fee` | `sdk.Dec` | `0.0005` | `0.0` | `swap_fee` | Fee sent to protocol treasury (0.05%, 17% of swap fee) |

**Constraints**:
- `lp_fee + protocol_fee = swap_fee` (always enforced)
- Fees are basis points (0.001 = 0.1%)
- Changes require governance proposal

**Example Values**:
```go
SwapFee:     math.LegacyNewDecWithPrec(3, 3)  // 0.003 (0.3%)
LpFee:       math.LegacyNewDecWithPrec(25, 4) // 0.0025 (0.25%)
ProtocolFee: math.LegacyNewDecWithPrec(5, 4)  // 0.0005 (0.05%)
```

**Query**:
```bash
pawd query dex params | jq '.params.swap_fee'
```

---

### Liquidity Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `min_liquidity` | `sdk.Int` | `1000` | `100` | `1000000` | Minimum initial liquidity required for pool creation |

**Purpose**: Prevent spam pools with negligible liquidity

**Impact**:
- Lower value: Easier to create pools, more spam risk
- Higher value: Harder to bootstrap new markets

**Example**:
```go
MinLiquidity: math.NewInt(1000)  // 1000 base units
```

**Query**:
```bash
pawd query dex params | jq '.params.min_liquidity'
```

---

### Slippage & Protection Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `max_slippage_percent` | `sdk.Dec` | `0.05` | `0.01` | `0.50` | Maximum allowed slippage per transaction (5%) |
| `max_pool_drain_percent` | `sdk.Dec` | `0.30` | `0.10` | `0.50` | Maximum percentage of pool that can be drained in single swap (30%) |

**Security**:
- `max_slippage_percent`: Protects users from excessive price impact
- `max_pool_drain_percent`: Prevents pool manipulation and flash loan attacks

**Examples**:
```go
MaxSlippagePercent:  math.LegacyNewDecWithPrec(5, 2)  // 0.05 (5%)
MaxPoolDrainPercent: math.LegacyNewDecWithPrec(30, 2) // 0.30 (30%)
```

**Use Case**:
```bash
# Swap fails if price impact > 5%
pawd tx dex swap 1 upaw ubtc 1000000 --max-slippage 0.05

# Swap fails if draining > 30% of pool
# Pool reserves: 1,000,000 upaw
# Max swap amount: 300,000 upaw
```

---

### Flash Loan Protection

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `flash_loan_protection_blocks` | `int64` | `10` | `1` | `100` | Blocks to track for multi-transaction attack detection |

**Mechanism**:
- Tracks user activity across blocks
- Flags patterns like: add liquidity → swap → remove liquidity in same block
- Prevents sandwich attacks and flash loan exploits

**Tuning**:
- Lower: Less protection, more false negatives
- Higher: More protection, potential false positives (legitimate arbitrage)

**Example**:
```go
FlashLoanProtectionBlocks: 10  // 10 blocks ~= 1 minute
```

**Query**:
```bash
pawd query dex params | jq '.params.flash_loan_protection_blocks'
```

---

### Gas Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `pool_creation_gas` | `uint64` | `1000` | `100` | `10000` | Additional gas for creating a pool |
| `swap_validation_gas` | `uint64` | `1500` | `100` | `10000` | Additional gas for swap validation (oracle checks, etc.) |
| `liquidity_gas` | `uint64` | `1200` | `100` | `10000` | Additional gas for liquidity operations |

**Purpose**: Account for computational cost of complex operations

**Gas Calculation**:
```
Total Gas = Base SDK Gas + Module Gas
Swap Gas = 5000 (base) + 1500 (validation) = 6500
```

**Optimization**:
- Lower values: Cheaper transactions, risk of DoS
- Higher values: More expensive, better DoS protection

**Example**:
```go
PoolCreationGas:   1000
SwapValidationGas: 1500
LiquidityGas:      1200
```

---

### IBC Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `authorized_channels` | `[]AuthorizedChannel` | `[]` | List of authorized IBC channels for cross-chain DEX operations |

**Structure**:
```go
type AuthorizedChannel struct {
    PortId    string  // e.g., "dex"
    ChannelId string  // e.g., "channel-0"
}
```

**Security**: Only packets from authorized channels are processed

**Management**:
- Add channels via governance proposal
- Requires IBC channel handshake completion
- Verify counterparty chain before authorization

**Example**:
```json
{
  "authorized_channels": [
    {"port_id": "dex", "channel_id": "channel-0"},
    {"port_id": "dex", "channel_id": "channel-42"}
  ]
}
```

**Query**:
```bash
pawd query dex params | jq '.params.authorized_channels'
```

---

### Complete DEX Params Structure

```go
type Params struct {
    SwapFee                   math.LegacyDec       `json:"swap_fee"`
    LpFee                     math.LegacyDec       `json:"lp_fee"`
    ProtocolFee               math.LegacyDec       `json:"protocol_fee"`
    MinLiquidity              math.Int             `json:"min_liquidity"`
    MaxSlippagePercent        math.LegacyDec       `json:"max_slippage_percent"`
    MaxPoolDrainPercent       math.LegacyDec       `json:"max_pool_drain_percent"`
    FlashLoanProtectionBlocks int64                `json:"flash_loan_protection_blocks"`
    AuthorizedChannels        []AuthorizedChannel  `json:"authorized_channels"`
    PoolCreationGas           uint64               `json:"pool_creation_gas"`
    SwapValidationGas         uint64               `json:"swap_validation_gas"`
    LiquidityGas              uint64               `json:"liquidity_gas"`
}
```

---

## Oracle Module Parameters

**Module**: `oracle`
**Default Location**: `x/oracle/types/params.go`

### Voting Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `vote_period` | `uint64` | `30` | `1` | `3600` | Blocks between price aggregations |
| `vote_threshold` | `sdk.Dec` | `0.67` | `0.50` | `1.00` | Minimum voting power required for consensus (67%) |

**Vote Period**:
- Frequency of price updates
- Lower: More frequent updates, higher validator load
- Higher: Less frequent, stale data risk

**Calculation**:
```
Block time: ~5 seconds
Vote period: 30 blocks
Update frequency: 30 * 5 = 150 seconds = 2.5 minutes
```

**Vote Threshold**:
- Byzantine fault tolerance: 0.67 (2/3)
- Lower: Easier consensus, less security
- Higher: Harder consensus, more security

**Example**:
```go
VotePeriod:    30  // blocks
VoteThreshold: math.LegacyMustNewDecFromStr("0.67")  // 67%
```

---

### Slashing Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `slash_fraction` | `sdk.Dec` | `0.01` | `0.00` | `1.00` | Percentage of stake slashed for oracle misbehavior (1%) |
| `slash_window` | `int64` | `10000` | `100` | `100000` | Blocks to track for slashing (21 hours at 5s/block) |
| `min_valid_per_window` | `int64` | `100` | `1` | `slash_window` | Minimum valid votes required per window |

**Slashing Triggers**:
1. Missing votes: < `min_valid_per_window` in `slash_window`
2. Invalid prices: Statistical outliers (>3σ from median)
3. Manipulation: Coordinated false prices

**Example**:
```go
SlashFraction:     math.LegacyMustNewDecFromStr("0.01")  // 1%
SlashWindow:       10000  // blocks
MinValidPerWindow: 100    // votes
```

**Calculation**:
```
Slash window: 10,000 blocks * 5s = 50,000s ≈ 14 hours
Vote period: 30 blocks
Expected votes per window: 10,000 / 30 ≈ 333 votes
Minimum required: 100 votes (30% participation)
```

---

### TWAP Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `twap_lookback_window` | `int64` | `1000` | `10` | `10000` | Blocks to include in TWAP calculation |

**Purpose**: Flash loan resistance via time-weighted averaging

**Window Size Tradeoffs**:
- Smaller window (100 blocks): More responsive, less manipulation resistance
- Larger window (10000 blocks): Less responsive, better manipulation resistance

**Example**:
```go
TwapLookbackWindow: 1000  // blocks ≈ 1.4 hours
```

**Calculation**:
```
Window: 1000 blocks * 5s = 5000s ≈ 83 minutes
Manipulation cost: Attacker must sustain false prices for 83 minutes
```

---

### Geographic Diversity Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `allowed_regions` | `[]string` | `["global", "na", "eu", "apac", "latam", "africa"]` | Geographic regions for validator distribution |
| `min_geographic_regions` | `uint32` | `1` | Minimum number of distinct regions required |
| `max_validators_per_ip` | `uint32` | `3` | Maximum validators allowed per IP address |
| `max_validators_per_asn` | `uint32` | `5` | Maximum validators per Autonomous System Number |

**Region Codes**:
- `global`: Worldwide (no restriction)
- `na`: North America
- `eu`: Europe
- `apac`: Asia-Pacific
- `latam`: Latin America
- `africa`: Africa

**Security**:
- Prevents geographic centralization
- Detects Sybil attacks (many validators, same location)
- Ensures censorship resistance

**Examples**:
```go
AllowedRegions:      []string{"global", "na", "eu", "apac", "latam", "africa"}
MinGeographicRegions: 1  // Production should use 3+
MaxValidatorsPerIp:   3
MaxValidatorsPerAsn:  5
```

**Enforcement**:
```bash
# Validator registration checks:
1. Query IP geolocation (GeoIP database)
2. Count validators in same region
3. Reject if region count > max_validators_per_ip
4. Reject if ASN count > max_validators_per_asn
```

---

### Consensus Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `min_voting_power_for_consensus` | `sdk.Dec` | `0.10` | `0.01` | `0.50` | Minimum voting power after outlier removal (10%) |

**Purpose**: Prevent manipulation by low-stake validators

**Mechanism**:
1. Collect all price submissions
2. Remove statistical outliers (>3σ)
3. Calculate remaining voting power
4. Reject if < `min_voting_power_for_consensus`

**Example**:
```go
MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10")  // 10%
```

**Scenario**:
```
Total voting power: 100M PAW
Submissions:
- 10 validators, 1M PAW each: Price = $100 (outliers)
- 90 validators, 1M PAW each: Price = $50 (consensus)

After outlier removal:
- Remaining: 90 validators, 90M PAW (90% voting power)
- 90% > 10% minimum ✓
- Use median of 90 validators ($50)
```

---

### IBC Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `authorized_channels` | `[]AuthorizedChannel` | `[]` | Authorized IBC channels for cross-chain oracle data |

**Use Cases**:
- Aggregate prices from multiple chains
- Cross-chain oracle consensus
- Price feed distribution

**Example**:
```json
{
  "authorized_channels": [
    {"port_id": "oracle", "channel_id": "channel-0"},
    {"port_id": "oracle", "channel_id": "channel-1"}
  ]
}
```

---

### Complete Oracle Params Structure

```go
type Params struct {
    VotePeriod                 uint64               `json:"vote_period"`
    VoteThreshold              math.LegacyDec       `json:"vote_threshold"`
    SlashFraction              math.LegacyDec       `json:"slash_fraction"`
    SlashWindow                int64                `json:"slash_window"`
    MinValidPerWindow          int64                `json:"min_valid_per_window"`
    TwapLookbackWindow         int64                `json:"twap_lookback_window"`
    AuthorizedChannels         []AuthorizedChannel  `json:"authorized_channels"`
    AllowedRegions             []string             `json:"allowed_regions"`
    MinGeographicRegions       uint32               `json:"min_geographic_regions"`
    MinVotingPowerForConsensus math.LegacyDec       `json:"min_voting_power_for_consensus"`
    MaxValidatorsPerIp         uint32               `json:"max_validators_per_ip"`
    MaxValidatorsPerAsn        uint32               `json:"max_validators_per_asn"`
}
```

---

## Compute Module Parameters

**Module**: `compute`
**Default Location**: `x/compute/types/params.go`

### Provider Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `min_provider_stake` | `sdk.Int` | `1000000` | `100000` | `1000000000` | Minimum stake required to become compute provider (1 PAW) |
| `min_reputation_score` | `int32` | `50` | `0` | `100` | Minimum reputation score to accept requests |

**Stake Requirements**:
- Lower: More providers, lower quality risk
- Higher: Fewer providers, higher barrier to entry

**Reputation Scoring**:
- Scale: 0-100
- Starts at: 70 (new providers)
- Increases: Successful verifications (+1 per success)
- Decreases: Failed verifications (-10 per failure)
- Minimum: Providers below threshold cannot accept new requests

**Example**:
```go
MinProviderStake:   math.NewInt(1000000)  // 1 PAW = 1,000,000 upaw
MinReputationScore: 50
```

---

### Timeout Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `verification_timeout_seconds` | `uint64` | `300` | `60` | `3600` | Time allowed for proof verification (5 minutes) |
| `max_request_timeout_seconds` | `uint64` | `3600` | `300` | `86400` | Maximum timeout for compute request (1 hour) |
| `escrow_release_delay_seconds` | `uint64` | `3600` | `0` | `86400` | Delay before escrow release after verification (1 hour) |

**Verification Timeout**:
- Time for ZK proof verification on-chain
- Includes circuit loading, proof parsing, verification
- Too short: Complex proofs fail
- Too long: Providers can delay results

**Request Timeout**:
- Total time for provider to complete computation
- Requester specifies, max is this parameter
- After timeout: Escrow refunded, provider penalized

**Escrow Release Delay**:
- Dispute window after verification
- Allows time for challenges
- Zero = immediate release (risky)

**Example**:
```go
VerificationTimeoutSeconds: 300    // 5 minutes
MaxRequestTimeoutSeconds:   3600   // 1 hour
EscrowReleaseDelaySeconds:  3600   // 1 hour
```

---

### Slashing Parameters

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `reputation_slash_percentage` | `uint32` | `10` | `1` | `50` | Percentage of reputation slashed for failures (10%) |
| `stake_slash_percentage` | `uint32` | `1` | `1` | `100` | Percentage of stake slashed for fraud (1%) |

**Reputation Slashing**:
- Triggered by: Invalid proofs, timeouts, disputes lost
- Recovery: Successful completions restore reputation slowly
- Permanent ban: If reputation falls below min_reputation_score

**Stake Slashing**:
- Triggered by: Proven fraud, malicious behavior
- Slashed amount sent to: Community pool (50%), requester (50%)
- Provider must re-stake to continue

**Example**:
```go
ReputationSlashPercentage: 10  // 10% of reputation
StakeSlashPercentage:      1   // 1% of stake
```

**Calculation**:
```
Provider reputation: 80
Failed verification: -10% = -8 points
New reputation: 72

Provider stake: 10 PAW
Fraud detected: -1% = -0.1 PAW
Remaining stake: 9.9 PAW
```

---

### IBC Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `authorized_channels` | `[]AuthorizedChannel` | `[]` | Authorized IBC channels for cross-chain compute requests |
| `nonce_retention_blocks` | `int64` | `17280` | Blocks to retain nonce data for replay prevention (~24 hours) |

**Nonce Retention**:
- Purpose: Prevent replay attacks on IBC packets
- Calculation: `17280 blocks * 5s = 86,400s = 24 hours`
- Cleanup: Old nonces deleted to save storage

**Example**:
```go
NonceRetentionBlocks: 17280  // 24 hours at 5s/block
```

---

### Governance Parameters (Dispute Resolution)

**Location**: `x/compute/types/params.go` - `DefaultGovernanceParams()`

| Parameter | Type | Default | Min | Max | Description |
|-----------|------|---------|-----|-----|-------------|
| `dispute_deposit` | `sdk.Int` | `1000000` | `100000` | `100000000` | Deposit required to open dispute (1 PAW) |
| `evidence_period_seconds` | `int64` | `86400` | `3600` | `604800` | Time to submit evidence (24 hours) |
| `voting_period_seconds` | `int64` | `86400` | `3600` | `604800` | Time for validators to vote on dispute (24 hours) |
| `quorum_percentage` | `sdk.Dec` | `0.334` | `0.20` | `0.75` | Minimum participation for valid vote (33.4%) |
| `consensus_threshold` | `sdk.Dec` | `0.5` | `0.50` | `1.00` | Percentage of votes to resolve dispute (50%) |
| `slash_percentage` | `sdk.Dec` | `0.1` | `0.01` | `1.00` | Stake slashed if provider found fraudulent (10%) |
| `appeal_deposit_percentage` | `sdk.Dec` | `0.05` | `0.01` | `0.50` | Additional deposit for appeal (5% of original) |
| `max_evidence_size` | `uint64` | `10485760` | `1048576` | `104857600` | Maximum evidence file size (10 MB) |

**Dispute Workflow**:

```
1. Requester disputes result
   - Submit dispute with deposit (1 PAW)
   - Provide evidence (<10 MB)

2. Evidence period (24 hours)
   - Provider submits counter-evidence
   - Both parties can add evidence

3. Voting period (24 hours)
   - Validators vote on validity
   - Need 33.4% participation (quorum)
   - Need 50% agreement (threshold)

4. Resolution
   - If provider fraudulent: 10% stake slashed
   - If requester wrong: dispute deposit burned
   - Appeals allowed with 5% additional deposit
```

**Example**:
```go
type GovernanceParams struct {
    DisputeDeposit:          math.NewInt(1_000_000),
    EvidencePeriodSeconds:   86400,
    VotingPeriodSeconds:     86400,
    QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
    ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
    SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
    AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
    MaxEvidenceSize:         10 * 1024 * 1024,
}
```

---

### Complete Compute Params Structure

```go
type Params struct {
    MinProviderStake           math.Int             `json:"min_provider_stake"`
    VerificationTimeoutSeconds uint64               `json:"verification_timeout_seconds"`
    MaxRequestTimeoutSeconds   uint64               `json:"max_request_timeout_seconds"`
    ReputationSlashPercentage  uint32               `json:"reputation_slash_percentage"`
    StakeSlashPercentage       uint32               `json:"stake_slash_percentage"`
    MinReputationScore         int32                `json:"min_reputation_score"`
    EscrowReleaseDelaySeconds  uint64               `json:"escrow_release_delay_seconds"`
    AuthorizedChannels         []AuthorizedChannel  `json:"authorized_channels"`
    NonceRetentionBlocks       int64                `json:"nonce_retention_blocks"`
}

type GovernanceParams struct {
    DisputeDeposit          math.Int       `json:"dispute_deposit"`
    EvidencePeriodSeconds   int64          `json:"evidence_period_seconds"`
    VotingPeriodSeconds     int64          `json:"voting_period_seconds"`
    QuorumPercentage        math.LegacyDec `json:"quorum_percentage"`
    ConsensusThreshold      math.LegacyDec `json:"consensus_threshold"`
    SlashPercentage         math.LegacyDec `json:"slash_percentage"`
    AppealDepositPercentage math.LegacyDec `json:"appeal_deposit_percentage"`
    MaxEvidenceSize         uint64         `json:"max_evidence_size"`
}
```

---

## How to Query Parameters

### CLI Queries

**DEX Parameters**:
```bash
# All params
pawd query dex params

# Specific param
pawd query dex params | jq '.params.swap_fee'

# JSON output
pawd query dex params --output json
```

**Oracle Parameters**:
```bash
# All params
pawd query oracle params

# Vote period
pawd query oracle params | jq '.params.vote_period'
```

**Compute Parameters**:
```bash
# Standard params
pawd query compute params

# Governance params (dispute resolution)
pawd query compute governance-params

# Min provider stake
pawd query compute params | jq '.params.min_provider_stake'
```

---

### REST API Queries

**DEX**:
```bash
curl http://localhost:1317/paw/dex/v1/params
```

**Oracle**:
```bash
curl http://localhost:1317/paw/oracle/v1/params
```

**Compute**:
```bash
curl http://localhost:1317/paw/compute/v1/params
curl http://localhost:1317/paw/compute/v1/governance-params
```

---

### gRPC Queries

**DEX**:
```bash
grpcurl -plaintext localhost:9090 paw.dex.v1.Query/Params
```

**Oracle**:
```bash
grpcurl -plaintext localhost:9090 paw.oracle.v1.Query/Params
```

**Compute**:
```bash
grpcurl -plaintext localhost:9090 paw.compute.v1.Query/Params
```

---

## How to Change Parameters

### Via Governance Proposal

**Step 1: Create proposal JSON**

```json
{
  "title": "Update DEX Swap Fee",
  "description": "Increase swap fee from 0.3% to 0.5%",
  "changes": [
    {
      "subspace": "dex",
      "key": "SwapFee",
      "value": "\"0.005\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

**Step 2: Submit proposal**

```bash
pawd tx gov submit-proposal param-change proposal.json \
  --from proposer \
  --chain-id paw-1 \
  --gas auto
```

**Step 3: Vote**

```bash
pawd tx gov vote 42 yes --from validator
```

**Step 4: Wait for execution**

Parameters automatically update when proposal passes.

---

### Multiple Parameter Changes

**Example**: Update multiple DEX params simultaneously

```json
{
  "title": "DEX Fee and Limit Updates",
  "description": "Comprehensive update to DEX economics",
  "changes": [
    {
      "subspace": "dex",
      "key": "SwapFee",
      "value": "\"0.005\""
    },
    {
      "subspace": "dex",
      "key": "LpFee",
      "value": "\"0.004\""
    },
    {
      "subspace": "dex",
      "key": "ProtocolFee",
      "value": "\"0.001\""
    },
    {
      "subspace": "dex",
      "key": "MaxSlippagePercent",
      "value": "\"0.10\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

---

### Cross-Module Parameter Updates

**Example**: Update IBC channels for all modules

```json
{
  "title": "Authorize Cosmos Hub Channel for All Modules",
  "description": "Enable DEX, Oracle, and Compute on channel-0",
  "changes": [
    {
      "subspace": "dex",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"dex\",\"channel_id\":\"channel-0\"}]"
    },
    {
      "subspace": "oracle",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"oracle\",\"channel_id\":\"channel-0\"}]"
    },
    {
      "subspace": "compute",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"compute\",\"channel_id\":\"channel-0\"}]"
    }
  ],
  "deposit": "1000000000upaw"
}
```

---

## Parameter Update Best Practices

### 1. Testing

**Always test on testnet first**:

```bash
# Submit to testnet
pawd tx gov submit-proposal param-change proposal.json \
  --from proposer \
  --chain-id paw-testnet-1

# Monitor effects
pawd query dex pools --chain-id paw-testnet-1

# Verify metrics
curl http://testnet-node:26657/metrics
```

---

### 2. Impact Analysis

**Before proposing, analyze**:

- Economic impact (fees, rewards, costs)
- Technical impact (gas usage, performance)
- Security impact (attack vectors, vulnerabilities)
- User impact (UX changes, breaking changes)

**Example Impact Statement**:

```
Parameter: swap_fee
Current: 0.003 (0.3%)
Proposed: 0.005 (0.5%)

Economic Impact:
- Daily volume: $1,000,000
- Current fees: $3,000/day
- Proposed fees: $5,000/day
- Increase: +$2,000/day (+66%)

Technical Impact:
- No code changes required
- No gas impact
- No migration needed

Security Impact:
- Higher fees may reduce wash trading
- May discourage small swaps (spam)

User Impact:
- 66% increase in swap costs
- May drive users to competitors
- Benefits: Higher LP yields
```

---

### 3. Gradual Rollout

**For sensitive parameters, use phased approach**:

```
Phase 1 (Week 1): Increase swap_fee to 0.004 (33% increase)
Phase 2 (Week 3): Monitor volume and user feedback
Phase 3 (Week 5): Increase to 0.005 if no issues (final 25% increase)
```

---

### 4. Rollback Plan

**Always have a rollback ready**:

```json
{
  "title": "ROLLBACK: Revert Swap Fee to 0.3%",
  "description": "Revert recent fee increase due to 50% volume drop",
  "changes": [
    {
      "subspace": "dex",
      "key": "SwapFee",
      "value": "\"0.003\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

**Trigger conditions**:
- Volume drop > 30%
- User complaints > threshold
- Unexpected technical issues

---

## Common Parameter Tuning Scenarios

### Scenario 1: Increase Oracle Security

**Goal**: Reduce risk of price manipulation

**Changes**:
```json
{
  "changes": [
    {
      "subspace": "oracle",
      "key": "VoteThreshold",
      "value": "\"0.75\""
    },
    {
      "subspace": "oracle",
      "key": "MinVotingPowerForConsensus",
      "value": "\"0.20\""
    },
    {
      "subspace": "oracle",
      "key": "TwapLookbackWindow",
      "value": "\"2000\""
    }
  ]
}
```

**Effects**:
- Harder to manipulate (75% consensus vs 67%)
- Better outlier filtering (20% min vs 10%)
- Longer TWAP (2.8 hours vs 1.4 hours)

---

### Scenario 2: Bootstrap New Compute Market

**Goal**: Attract more compute providers

**Changes**:
```json
{
  "changes": [
    {
      "subspace": "compute",
      "key": "MinProviderStake",
      "value": "\"500000\""
    },
    {
      "subspace": "compute",
      "key": "MinReputationScore",
      "value": "\"40\""
    }
  ]
}
```

**Effects**:
- Lower barrier to entry (0.5 PAW vs 1 PAW)
- Accept lower reputation (40 vs 50)
- More providers, but higher quality monitoring needed

---

### Scenario 3: Optimize DEX for High Volume

**Goal**: Handle 10x volume increase

**Changes**:
```json
{
  "changes": [
    {
      "subspace": "dex",
      "key": "MaxSlippagePercent",
      "value": "\"0.10\""
    },
    {
      "subspace": "dex",
      "key": "MaxPoolDrainPercent",
      "value": "\"0.40\""
    },
    {
      "subspace": "dex",
      "key": "FlashLoanProtectionBlocks",
      "value": "\"5\""
    }
  ]
}
```

**Effects**:
- Higher slippage tolerance (10% vs 5%) for large swaps
- Larger single swaps allowed (40% vs 30%)
- Faster flash loan detection (5 blocks vs 10)

---

## Parameter Validation

### DEX Validation Rules

```go
// Enforced by ValidateBasic()
func (p Params) Validate() error {
    // Fees must be positive and < 100%
    if p.SwapFee.IsNegative() || p.SwapFee.GT(math.LegacyOneDec()) {
        return fmt.Errorf("swap_fee must be 0-1")
    }

    // Fees must sum correctly
    if !p.LpFee.Add(p.ProtocolFee).Equal(p.SwapFee) {
        return fmt.Errorf("lp_fee + protocol_fee must equal swap_fee")
    }

    // Min liquidity must be positive
    if !p.MinLiquidity.IsPositive() {
        return fmt.Errorf("min_liquidity must be positive")
    }

    // Percentages in valid range
    if p.MaxSlippagePercent.GT(math.LegacyOneDec()) {
        return fmt.Errorf("max_slippage_percent must be ≤ 1.0")
    }

    return nil
}
```

---

### Oracle Validation Rules

```go
func (p Params) Validate() error {
    // Vote period must be positive
    if p.VotePeriod == 0 {
        return fmt.Errorf("vote_period must be positive")
    }

    // Threshold in Byzantine range
    if p.VoteThreshold.LT(math.LegacyMustNewDecFromStr("0.5")) {
        return fmt.Errorf("vote_threshold must be ≥ 0.5")
    }

    // Slash fraction valid
    if p.SlashFraction.IsNegative() || p.SlashFraction.GT(math.LegacyOneDec()) {
        return fmt.Errorf("slash_fraction must be 0-1")
    }

    // Geographic diversity
    if p.MinGeographicRegions < 1 {
        return fmt.Errorf("min_geographic_regions must be ≥ 1")
    }

    return nil
}
```

---

### Compute Validation Rules

```go
func (p Params) Validate() error {
    // Stake must be positive
    if !p.MinProviderStake.IsPositive() {
        return fmt.Errorf("min_provider_stake must be positive")
    }

    // Timeouts must be reasonable
    if p.VerificationTimeoutSeconds < 60 {
        return fmt.Errorf("verification_timeout must be ≥ 60s")
    }

    // Reputation score in range
    if p.MinReputationScore < 0 || p.MinReputationScore > 100 {
        return fmt.Errorf("min_reputation_score must be 0-100")
    }

    return nil
}
```

---

## Monitoring Parameter Effects

### Metrics to Track

**After DEX parameter change**:
```prometheus
# Volume changes
paw_dex_total_volume_24h
paw_dex_swap_count_total

# Price impact
paw_dex_average_price_impact

# Fee collection
paw_dex_fees_collected_total
```

**After Oracle parameter change**:
```prometheus
# Participation
paw_oracle_active_validators
paw_oracle_vote_participation_rate

# Price quality
paw_oracle_price_deviation
paw_oracle_outlier_count

# Slashing
paw_oracle_slash_events_total
```

**After Compute parameter change**:
```prometheus
# Provider activity
paw_compute_active_providers
paw_compute_provider_registrations

# Request volume
paw_compute_requests_total
paw_compute_average_completion_time

# Disputes
paw_compute_disputes_total
paw_compute_dispute_success_rate
```

---

## Related Documentation

- [Governance Proposals Guide](guides/GOVERNANCE_PROPOSALS.md)
- [Error Codes Reference](api/guides/ERROR_CODES_REFERENCE.md)
- [Cross-Module Integration](implementation/CROSS_MODULE_INTEGRATION.md)

---

## Parameter Quick Reference Card

| Module | Key Parameters | Typical Values |
|--------|----------------|----------------|
| **DEX** | `swap_fee` | 0.003 (0.3%) |
| | `max_slippage_percent` | 0.05 (5%) |
| | `flash_loan_protection_blocks` | 10 blocks |
| **Oracle** | `vote_period` | 30 blocks |
| | `vote_threshold` | 0.67 (67%) |
| | `twap_lookback_window` | 1000 blocks |
| **Compute** | `min_provider_stake` | 1 PAW |
| | `verification_timeout_seconds` | 300s (5 min) |
| | `dispute_deposit` | 1 PAW |

---

**Last Updated**: Generated from codebase scan
**Version**: PAW v1.0.0
**Maintainers**: PAW Core Team
