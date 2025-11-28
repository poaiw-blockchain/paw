# Frequently Asked Questions (FAQ)

Common questions about PAW Blockchain.

## General Questions

### What is PAW Blockchain?

PAW is a lean Layer-1 blockchain built on Cosmos SDK and Tendermint, featuring a native DEX, secure compute aggregation through TEE, and multi-device wallets. It's designed for rapid deployment and early adoption with a compact validator set and 4-second block times.

### What makes PAW different from other blockchains?

- **Built-in DEX**: Native decentralized exchange without smart contracts
- **Secure Compute**: TEE-protected API key aggregation
- **Manageable Scale**: Compact validator set (4-100) for faster decisions
- **Deflationary**: Annual halving with fee burns
- **Early Adopter Rewards**: 1.5x multiplier for first 180 days

### Is PAW production-ready?

Yes! PAW has undergone extensive testing with 92% test coverage, multiple security audits, and is currently running on testnet with mainnet launch scheduled for 2025-01-15.

## Getting Started

### How do I get PAW tokens?

**Testnet:**
- Use the [faucet](https://faucet.paw.network)
- Join Discord and use the faucet bot

**Mainnet:**
- Purchase on exchanges
- Earn through staking
- Earn through liquidity provision
- Earn as a validator

### Which wallet should I use?

- **Desktop**: Best for daily use and trading ([Download](/guide/wallets#desktop-wallet))
- **Mobile**: Best for on-the-go transactions ([iOS](https://apps.apple.com/app/paw-wallet) | [Android](https://play.google.com/store/apps/details?id=io.pawblockchain.wallet))
- **Web**: Quick access from any browser ([wallet.paw.network](https://wallet.paw.network))
- **CLI**: For developers and advanced users
- **Hardware**: Best for large amounts (Ledger/Trezor)

### How do I backup my wallet?

1. Write down your 24-word recovery phrase on paper
2. Store in multiple secure locations (fireproof safe, bank vault)
3. Never store digitally or take screenshots
4. Test recovery before storing large amounts

::: danger
Your recovery phrase is the ONLY way to recover your wallet. There is no password reset or customer support that can help if you lose it.
:::

## Staking

### What is the current APR for staking?

12-18% APR, depending on:
- Total amount staked network-wide
- Validator commission rate
- Network inflation rate
- Your compounding frequency

### How long does it take to unstake?

21 days (unbonding period). During this time:
- No rewards are earned
- Tokens cannot be transferred
- Tokens cannot be restaked

### Can I lose my staked tokens?

Staked tokens can be slashed if your validator:
- Double-signs blocks (5% slash)
- Has excessive downtime (0.01% slash)

Choose validators carefully based on their track record.

### How often should I claim staking rewards?

- **Daily**: Maximum rewards but higher fees
- **Weekly**: Good balance of rewards and fees
- **Monthly**: Minimal fees but slightly lower returns
- **Auto-compound**: Best long-term strategy

## DEX & Trading

### What fees does the DEX charge?

- **Trading Fee**: 0.3% per swap (goes to liquidity providers)
- **Network Fee**: ~0.001 PAW per transaction

### What is impermanent loss?

Impermanent loss occurs when token prices in a liquidity pool diverge from when you deposited them. Example:

```
Deposit: 1000 PAW + 1000 USDC (PAW = $1)
PAW doubles to $2
Holding: $3000 value
In Pool: $2828 value
Impermanent Loss: $172 (5.7%)
```

However, trading fees can offset impermanent loss over time.

### How do I minimize impermanent loss?

- Provide liquidity to stablecoin pairs (USDC/USDT)
- Choose correlated assets (ATOM/OSMO)
- Stay in pools long-term to earn more fees
- Monitor price ratios closely

### Can I trade 24/7?

Yes! The DEX operates 24/7/365 with no downtime for maintenance.

## Governance

### Who can vote on proposals?

Anyone who has staked PAW tokens can vote. Voting power is proportional to your staked amount.

### What happens if I don't vote?

Your vote is inherited from your validator. If your validator doesn't vote, you don't vote.

### Can I change my vote?

Yes, you can change your vote anytime during the voting period (14 days).

### What is NoWithVeto?

NoWithVeto is a strong rejection used for spam or malicious proposals. If >33.4% vote NoWithVeto, the proposal is rejected and deposits are burned.

## Validators

### How much do I need to become a validator?

**Minimum Requirements:**
- 1,000 PAW minimum self-delegation
- Reliable server hardware
- Technical knowledge
- 24/7 monitoring capability

**Realistic Requirements:**
- 10,000+ PAW self-delegation to be competitive
- High-quality infrastructure
- Community reputation
- Active participation in governance

### How do validators earn money?

**Revenue Sources:**
1. Block rewards (30% of emissions)
2. Transaction fees (70% of network fees)
3. Commission from delegators (typically 5-10%)
4. MEV opportunities (when available)

### What are the risks of running a validator?

- **Slashing**: Lose stake for misbehavior
- **Downtime**: Lose delegators and rewards
- **Infrastructure Costs**: Servers, bandwidth, monitoring
- **Opportunity Cost**: Time and capital investment
- **Reputation Risk**: Poor performance affects delegation

## Security

### Is PAW secure?

Yes. PAW has been audited by multiple firms:
- Trail of Bits (October 2024)
- CertiK (November 2024)
- Halborn (December 2024)

The code is open-source and continuously monitored.

### How do I keep my tokens safe?

1. Use hardware wallets for large amounts
2. Enable biometric authentication on mobile
3. Never share your recovery phrase
4. Verify all transaction details
5. Use official wallets only
6. Keep software updated

### What if I suspect a security issue?

Report to security@pawblockchain.io immediately. We have a [bug bounty program](<REPO_URL>/blob/master/docs/BUG_BOUNTY.md) for responsibly disclosed vulnerabilities.

## Technical

### What is the block time?

4 seconds, with immediate finality (1 block confirmation).

### How many transactions per second?

Current capacity: 1,000+ tx/s
With future optimizations: 10,000+ tx/s

### What consensus mechanism does PAW use?

Tendermint BFT-DPoS (Byzantine Fault Tolerant Delegated Proof of Stake)

### Is PAW compatible with IBC?

Yes! PAW supports IBC (Inter-Blockchain Communication) for cross-chain transfers and interoperability with other Cosmos chains.

### Can I run a full node without being a validator?

Yes! Running a full node helps decentralize the network. See the [Getting Started](/guide/getting-started) guide.

## Troubleshooting

### My transaction is stuck/pending

Transactions on PAW finalize in 4 seconds. If stuck:
1. Check if your node is synced
2. Verify sufficient balance for fees
3. Check transaction hash on [explorer](https://explorer.paw.network)
4. Try resubmitting with higher gas

### I can't see my balance

1. Ensure wallet is connected to correct network
2. Wait for node to fully sync
3. Verify you're checking the correct address
4. Check block explorer to confirm balance

### My validator is jailed

Check the reason:
```bash
pawd query staking validator YOUR_VALIDATOR
```

If due to downtime, unjail:
```bash
pawd tx slashing unjail --from validator
```

If due to double-signing, you cannot unjail (create new validator).

## Community & Support

### How do I get help?

- **Discord**: [discord.gg/pawblockchain](https://discord.gg/pawblockchain)
- **Documentation**: [docs.paw.network](/)
- **Email**: support@pawblockchain.io

### How can I contribute?

- Report bugs
- Submit pull requests
- Write documentation
- Create educational content
- Participate in governance
- Run infrastructure
- Build applications

See [CONTRIBUTING.md](<REPO_URL>/blob/master/CONTRIBUTING.md)

### Where can I discuss proposals?

- Discord #governance channel
- Commonwealth forum
- Community calls (bi-weekly)

## Economics

### What is the total supply?

50,000,000 PAW (fixed maximum supply)

### Is PAW inflationary or deflationary?

Initially inflationary (block rewards), becoming deflationary through:
- Annual emission halving
- 20% of fees burned
- Long-term deflationary target

### When does emission halving occur?

Every 12 months:
- Year 1: 2,870 PAW/day
- Year 2: 1,435 PAW/day (50% reduction)
- Year 3: 717 PAW/day (50% reduction)
- Continues until minimum reached

### Can the tokenomics change?

Only through governance. Any changes require:
- Community proposal
- 7-day deposit period
- 14-day voting period
- >50% yes votes with >40% participation

---

**Didn't find your answer?** Ask in [Discord](https://discord.gg/pawblockchain) or check the [Glossary](/glossary)
