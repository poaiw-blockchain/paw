# Staking Guide

Earn rewards by staking your PAW tokens and securing the network.

## What is Staking?

Staking allows you to:
- **Earn Rewards**: Receive PAW tokens for participating in network security
- **Vote in Governance**: Influence network decisions
- **Support Validators**: Help secure the blockchain
- **Compound Returns**: Automatically increase your stake

## Current Staking Stats

| Metric | Value |
|--------|-------|
| Current APR | 12-18% |
| Total Staked | 25M PAW (50% of supply) |
| Unbonding Period | 21 days |
| Minimum Stake | 1 PAW |
| Validator Count | 100 active |

## How to Stake

### Using Desktop Wallet

1. Open PAW Desktop Wallet
2. Navigate to "Staking" tab
3. Click "Stake Now"
4. Select a validator
5. Enter amount to stake
6. Confirm transaction

### Using CLI

```bash
# Delegate to a validator
pawd tx staking delegate \
  pawvaloper1xxxxx... \
  1000000000upaw \
  --from my-wallet \
  --fees 500upaw

# Check delegation
pawd query staking delegation \
  paw1xxxxx... \
  pawvaloper1xxxxx...
```

## Choosing a Validator

### Important Factors

1. **Commission Rate** (0-100%)
   - Lower = more rewards for you
   - Typical range: 5-10%

2. **Uptime/Performance** (>99%)
   - High uptime = more rewards
   - Check validator status

3. **Voting Power** (<10% recommended)
   - Avoid centralization
   - Distribute stakes

4. **Community Involvement**
   - Active in governance
   - Responsive to delegators

### Validator Comparison

```bash
# List all validators
pawd query staking validators

# Get validator details
pawd query staking validator pawvaloper1xxxxx...

# Check validator commission
pawd query staking validator pawvaloper1xxxxx... | jq '.commission'
```

## Calculating Rewards

### Reward Formula

```
Annual Rewards = Staked Amount × APR × (1 - Validator Commission)

Example:
- Stake: 10,000 PAW
- APR: 15%
- Commission: 5%
- Annual Rewards: 10,000 × 0.15 × 0.95 = 1,425 PAW
- Monthly: ~119 PAW
- Daily: ~3.9 PAW
```

### Compounding

```bash
# Claim and restake rewards (compound)
pawd tx distribution withdraw-rewards pawvaloper1xxxxx... \
  --from my-wallet \
  --commission \
  --fees 500upaw

# Then restake
pawd tx staking delegate pawvaloper1xxxxx... <amount> --from my-wallet
```

## Managing Stakes

### Claiming Rewards

```bash
# Withdraw rewards from one validator
pawd tx distribution withdraw-rewards \
  pawvaloper1xxxxx... \
  --from my-wallet

# Withdraw from all validators
pawd tx distribution withdraw-all-rewards \
  --from my-wallet

# Check pending rewards
pawd query distribution rewards paw1xxxxx...
```

### Redelegating

Switch validators without unbonding period:

```bash
# Redelegate to different validator
pawd tx staking redelegate \
  pawvaloper1xxxxx... \  # From validator
  pawvaloper1yyyyy... \  # To validator
  1000000000upaw \
  --from my-wallet
```

::: warning
You can only redelegate from a validator once per unbonding period (21 days).
:::

### Unbonding (Unstaking)

```bash
# Undelegate tokens
pawd tx staking unbond \
  pawvaloper1xxxxx... \
  1000000000upaw \
  --from my-wallet

# Check unbonding status
pawd query staking unbonding-delegation \
  paw1xxxxx... \
  pawvaloper1xxxxx...
```

::: danger Unbonding Period
Tokens take 21 days to unbond. During this time:
- No rewards earned
- Cannot transfer tokens
- Cannot redelegate
:::

## Staking Strategies

### Conservative Strategy

- Stake with top 10 validators
- Diversify across 3-5 validators
- Claim and compound monthly
- Target: 12-15% APR

### Aggressive Strategy

- Stake with smaller validators
- Accept slightly higher risk
- Compound weekly
- Target: 15-18% APR

### Balanced Strategy

- Mix of large and small validators
- Rebalance quarterly
- Compound bi-weekly
- Target: 13-16% APR

## Risks and Considerations

### Slashing

Validators can be penalized for:

| Offense | Penalty | Impact |
|---------|---------|--------|
| Double Signing | 5% slash | High |
| Downtime | 0.01% slash | Low |
| Invalid Proposals | No slash | None |

::: tip
Choose validators with good track records to minimize slashing risk.
:::

### Opportunity Cost

Staked tokens:
- Cannot be traded immediately
- Require 21 days to unbond
- Miss short-term trading opportunities

## Advanced Features

### Auto-Compounding

Set up automatic reward compounding:

```bash
# Using a cron job (Linux/Mac)
crontab -e

# Add this line (compound daily at midnight)
0 0 * * * /path/to/compound-script.sh
```

### Multi-Validator Staking

```bash
# Stake across multiple validators
validators=("pawvaloper1xxx..." "pawvaloper1yyy..." "pawvaloper1zzz...")
amount_each=333333333upaw

for val in "${validators[@]}"; do
  pawd tx staking delegate $val $amount_each --from my-wallet -y
done
```

## Monitoring Performance

### Track Your Stakes

```bash
# View all delegations
pawd query staking delegations paw1xxxxx...

# Calculate total staked
pawd query staking delegations paw1xxxxx... | jq '.delegation_responses[] | .balance.amount' | awk '{s+=$1} END {print s/1000000 " PAW"}'

# View rewards
pawd query distribution rewards paw1xxxxx...
```

### Analytics Dashboard

Visit [staking.paw.network](https://staking.paw.network) for:
- Portfolio overview
- Reward history
- Validator comparison
- APR calculator
- Performance tracking

## Tax Implications

::: warning Tax Reporting
Staking rewards may be taxable in your jurisdiction. Consult a tax professional.
:::

**Records to keep:**
- Delegation transactions
- Reward claims
- Commission payments
- Unbonding events

## Troubleshooting

### Rewards Not Appearing

```bash
# Ensure you're checking correct delegator address
pawd query distribution rewards paw1xxxxx...

# Rewards accumulate each block
# May take 5-10 minutes to show first rewards
```

### Cannot Unbond

Check if:
- Sufficient tokens delegated
- Correct validator address
- Not already unbonding same tokens

### Low Returns

Possible causes:
- Validator downtime
- High commission rate
- Network-wide APR decrease
- Validator jailed

---

**Previous:** [DEX Guide](/guide/dex) | **Next:** [Governance](/guide/governance) →
