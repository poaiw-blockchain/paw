# PAW Governance

## Overview

PAW uses on-chain governance to manage protocol upgrades, parameter changes, and community proposals. All PAW token holders can participate in governance.

## Governance Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| Voting Period | 7 days | Time to vote on proposals |
| Min Deposit | 10,000 PAW | Minimum deposit to activate proposal |
| Deposit Period | 14 days | Time to reach min deposit |
| Quorum | 33.4% | Minimum participation required |
| Threshold | 50% | Votes needed to pass |
| Veto Threshold | 33.4% | Votes to veto proposal |

## Proposal Types

### 1. Parameter Changes
Modify module parameters without code changes:
- DEX: swap fees, pool creation fees, circuit breaker thresholds
- Compute: provider stake requirements, verification timeouts
- Oracle: slash fractions, vote periods, diversity requirements

### 2. Software Upgrades
Coordinate chain upgrades at specific block heights:
- Announce upgrade plan
- Validators prepare new binary
- Chain halts and upgrades at designated height

### 3. Community Spend
Allocate community pool funds:
- Development grants
- Marketing initiatives
- Ecosystem incentives

### 4. Text Proposals
Signal community intent without on-chain execution.

## How to Participate

### Submit a Proposal
```bash
pawd tx gov submit-proposal \
  --title "Proposal Title" \
  --description "Detailed description" \
  --type Text \
  --deposit 10000000upaw \
  --from <wallet> \
  --chain-id paw-testnet-1
```

### Deposit on a Proposal
```bash
pawd tx gov deposit <proposal-id> 1000000upaw \
  --from <wallet> \
  --chain-id paw-testnet-1
```

### Vote on a Proposal
```bash
pawd tx gov vote <proposal-id> yes \
  --from <wallet> \
  --chain-id paw-testnet-1
```

Vote options: `yes`, `no`, `abstain`, `no_with_veto`

### Query Proposals
```bash
# List all proposals
pawd query gov proposals

# Get proposal details
pawd query gov proposal <proposal-id>

# Check vote tally
pawd query gov tally <proposal-id>
```

## Best Practices

1. **Discuss First**: Use community channels to discuss proposals before submission
2. **Clear Description**: Provide detailed rationale and expected outcomes
3. **Adequate Deposit**: Ensure deposit is met to activate voting
4. **Engage Validators**: Validators have significant voting power
5. **Consider Timing**: Avoid holidays and busy periods

## Emergency Actions

For critical security issues, the following mechanisms exist:
- **Circuit Breakers**: Automatic pause on anomalous activity
- **Emergency Pause**: Governance-controlled module pause
- **Emergency Admin**: Designated admin for time-sensitive actions (mainnet only)

## Resources

- [Cosmos SDK Governance](https://docs.cosmos.network/main/modules/gov)
- [PAW Discord](https://discord.gg/DBHTc2QV) - #governance channel
- [Forum](https://forum.paw.network) - Long-form discussions
