# PAW Blockchain - Validator Economics Guide

**Version:** 1.1
**Last Updated:** 2026-01-12

---

## Table of Contents

1. [Overview](#overview)
2. [Staking Requirements](#staking-requirements)
3. [Reward Structure](#reward-structure)
4. [Commission Rates](#commission-rates)
5. [Reward Calculation Notes](#reward-calculation-notes)
6. [Slashing Penalties](#slashing-penalties)
7. [Delegation Mechanics](#delegation-mechanics)
8. [Claiming and Compounding Rewards](#claiming-and-compounding-rewards)
9. [Delegator Selection Criteria](#delegator-selection-criteria)
10. [Next Steps](#next-steps)

---

## Overview

PAW blockchain uses Proof-of-Stake (PoS) consensus where validators stake PAW tokens to secure the network and earn rewards. This guide explains the economics for validators and delegators.

### Key Concepts

```
Staking: Locking PAW tokens to become validator or delegate to validator
Validator: Node operator who stakes tokens and participates in consensus
Delegator: Token holder who delegates stake to validator
Commission: Percentage of delegator rewards kept by validator
Reward: Tokens earned for validating blocks
Slashing: Penalty for misbehavior (downtime or double-signing)
```

---

## Staking Requirements

### Minimum Self-Delegation

```yaml
Testnet: 1,000,000 upaw (1 PAW)
Mainnet: 10,000,000 upaw (10 PAW) recommended
```

**What is minimum self-delegation?**
- Amount validator must stake with their own tokens
- Cannot unbond below this amount without exiting validator set
- Set during validator creation, cannot be changed

### Competitive Stake Amounts

To join the active validator set (top 100 validators by voting power):

```
Early Testnet: 1-10 PAW (low competition)
Mature Testnet: 100-1,000 PAW
Early Mainnet: 10,000-100,000 PAW (estimated)
Mature Mainnet: 1,000,000+ PAW (competitive market)
```

### Checking Required Stake

```bash
# View all validators sorted by voting power
pawd query staking validators --output json | \
  jq -r '.validators | sort_by(.tokens | tonumber) | reverse | .[] | "\(.description.moniker): \(.tokens)"'

# Check rank 100 validator (minimum to be in active set)
pawd query staking validators --output json | \
  jq -r '.validators | sort_by(.tokens | tonumber) | reverse | .[99] | "Rank 100: \(.tokens) tokens"'
```

---

## Reward Structure

### Reward Sources

**1. Block Rewards (Inflation)**
- Annual inflation: 7-20% (adjustable by governance)
- New PAW tokens minted each block
- Distributed proportionally to all validators based on voting power

**2. Transaction Fees**
- Fees paid by users for transactions
- Collected by block proposer
- Distributed to all validators

**3. Module Fees**
- DEX trading fees (portion goes to validators)
- Oracle data submission fees
- Compute job fees

### Reward Distribution

```
Total Block Reward (100%)
    │
    ├─► Community Pool (2%)
    │
    └─► Active Validator Set (98%)
            │
            ├─► Validator 1 (proportional to voting power)
            │      ├─► Commission (5-20%)
            │      └─► Delegators (80-95%)
            │
            ├─► Validator 2 (proportional to voting power)
            └─► ...
```

**Example:**
```
Block reward: 1000 PAW
Community pool: 20 PAW (2%)
Validator set: 980 PAW

Your validator:
  Voting power: 1% of total
  Your share: 9.8 PAW
  Commission: 10%
    - Your commission: 0.98 PAW
    - Delegators: 8.82 PAW
  Your self-delegation: 50%
    - Your self-delegation reward: 4.41 PAW

Total you earn: 0.98 + 4.41 = 5.39 PAW per block
```

---

## Commission Rates

### Setting Commission

Configured when creating validator:

```bash
--commission-rate 0.10           # 10% commission
--commission-max-rate 0.20       # Can never exceed 20%
--commission-max-change-rate 0.01 # Max 1% change per day
```

**Commission Parameters:**
- `commission-rate`: Current rate (0.00 = 0%, 1.00 = 100%)
- `commission-max-rate`: Upper limit (set at creation, cannot change)
- `commission-max-change-rate`: Maximum daily adjustment

### Typical Commission Rates

| Rate | Strategy | Typical Validators |
|------|----------|-------------------|
| **0%** | Attract delegators (unsustainable) | New validators, promotional |
| **5%** | Very competitive | High-reputation validators |
| **10%** | Market standard | Most validators |
| **15-20%** | Premium service | Enterprise validators, enhanced services |

### Changing Commission

```bash
# Increase from 10% to 12%
pawd tx staking edit-validator \
  --commission-rate 0.12 \
  --from validator-operator \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

# Limitations:
# - Cannot exceed commission-max-rate
# - Cannot change more than commission-max-change-rate per day
# - Changes are immediate (delegators can react)
```

### Commission Strategy

**Best Practices:**
1. **Start Low (5-8%):** Build trust, attract initial delegators
2. **Communicate Changes:** Announce 1 week before rate increases
3. **Justify Increases:** Explain infrastructure improvements
4. **Monitor Competition:** Check other validators' rates
5. **Gradual Adjustments:** 1-2% changes, not sudden jumps

---

## Reward Calculation Notes

- Rewards depend on on-chain parameters (inflation, bonded ratio, community tax) and validator commission.
- To estimate emissions, fetch current parameters instead of using static examples:
  ```bash
  pawd q mint params
  pawd q distribution params
  pawd q staking params | jq '.params.bond_denom'
  ```
- Validator reward rate ≈ `(inflation / bonded_ratio) × (1 - community_tax)`.
- Delegator share = `validator_reward_rate × (1 - commission)`.
- Recalculate periodically; values change through governance and network conditions.

---

## Slashing Penalties

### Downtime Slashing

**Trigger:**
- Missed blocks > 5% in 50,000 block window (~2,500 blocks)

**Penalties:**
- 0.01% of bonded stake slashed
- Validator jailed for 10 minutes
- Must manually unjail

**Example:**
```
Stake: 100,000 PAW
Slashed: 100,000 × 0.0001 = 10 PAW
Remaining: 99,990 PAW
```

**Prevention:**
- Maintain 99.9%+ uptime
- Setup monitoring and alerts
- Use sentry architecture
- Have standby infrastructure

### Double-Sign Slashing

**Trigger:**
- Signing two different blocks at same height
- Usually caused by running duplicate validators

**Penalties:**
- 5% of bonded stake slashed
- Validator permanently tombstoned (cannot unjail)
- Must create new validator with new consensus key

**Example:**
```
Stake: 100,000 PAW
Slashed: 100,000 × 0.05 = 5,000 PAW
Remaining: 95,000 PAW

Validator PERMANENTLY disabled
Must start new validator (reputation damage)
```

**Prevention:**
- NEVER run duplicate validators
- Use HSM/tmkms with state protection
- Careful failover procedures
- Single source of truth for consensus key

---

## Delegation Mechanics

### Delegating to Validators

**Delegators stake to validators to:**
- Earn rewards without running infrastructure
- Participate in network security
- Support preferred validators

**Delegation Process:**
```bash
# Delegate 1000 PAW to validator
pawd tx staking delegate <validator-address> 1000000000upaw \
  --from delegator-account \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes
```

### Redelegation

**Move stake between validators without unbonding period:**
```bash
pawd tx staking redelegate \
  <source-validator> \
  <destination-validator> \
  1000000000upaw \
  --from delegator-account \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

# Limitations:
# - Cannot redelegate again for 21 days
# - Still subject to slashing from source validator
```

### Unbonding

**Withdraw stake (with 21-day waiting period):**
```bash
pawd tx staking unbond <validator-address> 1000000000upaw \
  --from delegator-account \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

# Funds locked for unbonding period (21 days)
# Still subject to slashing during unbonding
# After 21 days, funds automatically returned
```

---

## Claiming and Compounding Rewards

### Checking Rewards

```bash
# Check pending rewards
pawd query distribution rewards <delegator-address>

# Check validator commission
pawd query distribution commission <validator-address>
```

### Withdrawing Rewards

```bash
# Withdraw all delegator rewards
pawd tx distribution withdraw-all-rewards \
  --from delegator-account \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

# Withdraw validator commission
pawd tx distribution withdraw-rewards <validator-address> \
  --commission \
  --from validator-operator \
  --chain-id paw-mvp-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes
```

### Auto-Compounding Script

```bash
#!/bin/bash
# auto-compound.sh - Withdraw and re-stake rewards daily

VALIDATOR_OPERATOR="validator-operator"
VALIDATOR_ADDR=$(pawd keys show $VALIDATOR_OPERATOR --bech val -a)
OPERATOR_ADDR=$(pawd keys show $VALIDATOR_OPERATOR -a)
CHAIN_ID="paw-mvp-1"

# Withdraw commission
pawd tx distribution withdraw-rewards $VALIDATOR_ADDR \
  --commission \
  --from $VALIDATOR_OPERATOR \
  --chain-id $CHAIN_ID \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

sleep 10

# Get balance (reserve 10000 upaw for fees)
BALANCE=$(pawd query bank balances $OPERATOR_ADDR --output json | \
  jq -r '.balances[] | select(.denom=="upaw") | .amount')
DELEGATE_AMOUNT=$((BALANCE - 10000))

# Re-delegate rewards
if [ $DELEGATE_AMOUNT -gt 100000 ]; then
  pawd tx staking delegate $VALIDATOR_ADDR ${DELEGATE_AMOUNT}upaw \
    --from $VALIDATOR_OPERATOR \
    --chain-id $CHAIN_ID \
    --gas auto \
    --gas-prices 0.001upaw \
    --yes
fi

# Schedule daily: crontab -e
# 0 0 * * * /home/validator/auto-compound.sh >> /var/log/auto-compound.log 2>&1
```

---

## Delegator Selection Criteria

Delegators choose validators based on:

1. **Commission** (lower = more rewards)
2. **Uptime** (high = less slashing risk)
3. **Reputation** (trusted operators)
4. **Services** (governance participation, community engagement)
5. **Decentralization** (avoid concentration in top validators)

**Attracting Delegators:**
- Maintain high uptime (99.9%+)
- Competitive commission (5-10%)
- Active governance participation
- Strong security posture
- Community engagement (Twitter, Discord, validator website)
- Transparency (publish validator metrics)

---

## Next Steps

- **Operations:** [VALIDATOR_ONBOARDING_GUIDE.md](./VALIDATOR_ONBOARDING_GUIDE.md) and [VALIDATOR_OPERATOR_GUIDE.md](./VALIDATOR_OPERATOR_GUIDE.md)
- **Key management and hardening:** [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md) and [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md)
- **Observability:** [DASHBOARDS_GUIDE.md](./DASHBOARDS_GUIDE.md) and [OBSERVABILITY.md](./OBSERVABILITY.md)

---

**Last Updated:** 2026-01-12
**Maintained by:** PAW Protocol Team
