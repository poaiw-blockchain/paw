# Tokenomics

Complete economic model of PAW Blockchain.

## Token Distribution

**Total Supply**: 50,000,000 PAW

| Category | Allocation | % | Vesting |
|----------|-----------|---|---------|
| Public Sale | 7,000,000 | 14% | Immediate |
| Mining & Node Rewards | 10,500,000 | 21% | On-chain |
| API Donor Rewards | 8,400,000 | 16.8% | 4-year cliff |
| Team & Advisors | 3,500,000 | 7% | 2-year cliff |
| Foundation Treasury | 3,500,000 | 7% | Governance |
| Ecosystem Fund | 2,100,000 | 4.2% | Governance |
| Reserve | 15,000,000 | 30% | Governance |

## Emission Schedule

### Year 1
- **Daily Emission**: 2,870 PAW
- **Annual Emission**: ~1,047,550 PAW
- **Early Adopter Bonus**: 1.5x multiplier (first 180 days)

### Year 2
- **Daily Emission**: 1,435 PAW (50% reduction)
- **Annual Emission**: ~523,775 PAW

### Year 3+
- **Annual Halving**: Continues until minimum reached
- **Price Oracle Gating**: Prevents excessive inflation

## Reward Distribution

### Block Rewards

| Recipient | Allocation |
|-----------|-----------|
| Validators | 30% |
| Node Operators | 30% |
| Compute Agents | 50% |
| Ecosystem Fund | 5% |

### Transaction Fees

- **Base Fee**: 0.001 PAW per transaction
- **DEX Fee**: 0.3% per swap
- **Distribution**:
  - 70% to validators
  - 20% burned (deflationary)
  - 10% to community pool

## Staking Economics

### Staking APR

Current: **12-18% APR**

Factors affecting APR:
- Total staked amount
- Network inflation
- Validator commission
- Block rewards

### Validator Economics

**Revenue Sources:**
1. Block rewards (30% of emissions)
2. Transaction fees (70% of fees)
3. MEV opportunities
4. Commission from delegators

**Costs:**
- Infrastructure (server, bandwidth)
- Monitoring tools
- Security measures
- Personnel

**Break-even Analysis:**

```
Monthly Revenue: ~$2,000 (5% commission on $400K stake)
Monthly Costs: ~$500
Monthly Profit: ~$1,500
Annual ROI: ~360%
```

## Deflationary Mechanics

### Fee Burns

20% of all transaction fees are burned:

```
Daily Transactions: 10,000
Average Fee: 0.001 PAW
Daily Fees: 10 PAW
Daily Burn: 2 PAW
Annual Burn: ~730 PAW
```

### Emission Halving

Annual supply inflation decreases by 50%:

- Year 1: 2.1% inflation
- Year 2: 1.05% inflation
- Year 3: 0.525% inflation
- Year 10: <0.01% inflation

## Liquidity Mining

### DEX Incentives

Additional rewards for liquidity providers:

- **Pool Allocation**: 1,000,000 PAW over 2 years
- **Distribution**: Proportional to liquidity provided
- **Bonus Pairs**: USDC/PAW, ATOM/PAW (2x rewards)

### Calculation

```
Your Rewards = Total Pool Rewards × (Your Liquidity / Total Liquidity)

Example:
Total Pool Rewards: 1,000 PAW/day
Your Liquidity: $10,000
Total Liquidity: $1,000,000
Your Daily Rewards: 1,000 × (10,000/1,000,000) = 10 PAW
```

## Governance Treasury

**Community Pool**: 3,500,000 PAW + ongoing 5% of emissions

**Usage:**
- Development grants
- Marketing initiatives
- Security audits
- Ecosystem growth
- Emergency funds

**Spending**: Requires governance approval

## Token Utility

### Primary Uses

1. **Transaction Fees**: Pay for blockchain operations
2. **Staking**: Secure network, earn rewards
3. **Governance**: Vote on proposals
4. **DEX Trading**: Swap between assets
5. **Liquidity Provision**: Earn fees from trading
6. **Compute Credits**: Pay for secure computation

### Value Accrual

Token value increases through:
- Network usage (more fees)
- Staking demand (reduced circulating supply)
- DEX liquidity (trading volume)
- Deflationary burns (supply reduction)
- Ecosystem growth (utility expansion)

## Economic Security

### Attack Costs

**51% Attack Cost:**
```
51% of Staked PAW × Market Price
= 12,750,000 PAW × $1.50
= $19,125,000
```

**Validator Cartel:**
- Would need 67% of validators to collude
- Governance can jail malicious validators
- Slashing penalties make attacks costly

## Price Projections

::: warning Disclaimer
These are projections, not guarantees. Actual prices depend on market conditions.
:::

**Conservative Model:**
- Year 1: $1.00 - $2.00
- Year 2: $2.00 - $4.00
- Year 3: $4.00 - $8.00

**Optimistic Model:**
- Year 1: $2.00 - $5.00
- Year 2: $5.00 - $15.00
- Year 3: $15.00 - $50.00

**Assumptions:**
- Growing user base
- Increasing transaction volume
- Successful DEX adoption
- Active ecosystem development

## Comparison to Other Chains

| Metric | PAW | Cosmos | Osmosis |
|--------|-----|--------|---------|
| Max Supply | 50M | Infinite | 1B |
| Inflation | Decreasing | ~7-20% | ~7-20% |
| Staking APR | 12-18% | 15-20% | 10-25% |
| DEX Fees | 0.3% | N/A | 0.2-1% |
| Block Time | 4s | 6-7s | 6-7s |

---

**Previous:** [Architecture](/reference/architecture) | **Next:** [Network Specs](/reference/network-specs) →
