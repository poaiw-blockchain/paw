# Frequently Asked Questions

Comprehensive answers to common questions about PAW Blockchain.

## General Questions

### What is PAW Blockchain?

PAW is a high-performance blockchain platform designed specifically for decentralized finance (DeFi) applications. Built on the Cosmos SDK with Tendermint consensus, PAW combines speed, security, and powerful built-in features including a native DEX, staking, and on-chain governance.

**Key Features:**
- 6-second block time
- Thousands of transactions per second
- Sub-cent transaction fees
- Built-in decentralized exchange
- Proof-of-Stake consensus
- IBC interoperability

### How is PAW different from other blockchains?

PAW distinguishes itself through:

1. **Native DEX**: Built into the blockchain protocol, not a separate smart contract
2. **Optimized for DeFi**: Purpose-built modules for trading, liquidity, and yield
3. **High Performance**: Faster and cheaper than many competitors
4. **Interoperability**: IBC-enabled for cross-chain transfers
5. **Community Governed**: On-chain governance with token holder voting

**Comparison:**

| Feature | PAW | Ethereum | BSC |
|---------|-----|----------|-----|
| Block Time | 6s | 12s | 3s |
| TX Fee | $0.001 | $5-50 | $0.10 |
| Finality | 6s | 13min | 45s |
| Native DEX | ✅ | ❌ | ❌ |

### Who created PAW?

PAW was created by a team of blockchain developers and DeFi experts. The project is now fully decentralized with governance controlled by token holders. Development is open-source with contributions from the global community.

### Is PAW open source?

Yes! All PAW code is open source under the Apache 2.0 license:
- **Core**: https://github.com/paw-chain/paw
- **SDKs**: https://github.com/paw-chain/sdks
- **Wallets**: https://github.com/paw-chain/wallets

### What is the PAW token used for?

The PAW token has multiple uses:

1. **Transaction Fees**: Pay for all on-chain operations
2. **Staking**: Secure the network and earn rewards
3. **Governance**: Vote on proposals and protocol changes
4. **DEX Trading**: Base trading pair for all assets
5. **Liquidity Provision**: Earn fees by providing liquidity

### What is the total supply of PAW?

- **Total Supply**: 1,000,000,000 PAW
- **Initial Circulating**: 300,000,000 PAW
- **Inflation**: 7-20% annually (decreasing over time)
- **Distribution**: See [Tokenomics](#tokenomics)

## Wallet Questions

### How do I create a wallet?

PAW offers several wallet options:

1. **Browser Extension** (easiest):
   - Install from Chrome Web Store or Firefox Add-ons
   - Click "Create Wallet"
   - Save recovery phrase
   - Done!

2. **Desktop Application** (full-featured):
   - Download for Windows, macOS, or Linux
   - Install and launch
   - Create new wallet
   - Backup recovery phrase

3. **Mobile App** (on-the-go):
   - Download from App Store or Google Play
   - Create wallet with biometric security
   - Backup recovery phrase

**See**: [Wallet Setup Tutorial](#wallet-setup)

### What if I lose my recovery phrase?

Unfortunately, your recovery phrase is the ONLY way to restore your wallet. If lost:

❌ **Cannot be recovered** by PAW team
❌ **Cannot be reset** or regenerated
❌ **Funds are permanently inaccessible**

**Prevention:**
- Write on paper, never digital
- Store in fireproof/waterproof safe
- Consider metal backup plates
- Keep multiple copies in secure locations
- NEVER share with anyone

### Can I use the same wallet on multiple devices?

Yes! Import your wallet using your recovery phrase:

```
1. Install PAW wallet on new device
2. Click "Import Wallet"
3. Enter 24-word recovery phrase
4. Set new password (can be different)
5. Wallet restored with same address
```

⚠️ **Security Note:** More devices = more attack surfaces. Only import on devices you trust.

### Which wallet should I use?

**Recommendations:**

| Use Case | Recommended Wallet |
|----------|-------------------|
| Casual user | Browser Extension |
| Frequent trader | Desktop App |
| Mobile access | Mobile App |
| Large holdings | Desktop + Hardware Wallet |
| Developer | CLI + Desktop |

You can use multiple wallets simultaneously!

### How do I backup my wallet?

**Recovery Phrase Method** (Recommended):
```
1. Open wallet settings
2. Click "Reveal Recovery Phrase"
3. Enter password
4. Write down all 24 words IN ORDER
5. Verify by re-entering
6. Store safely offline
```

**Encrypted Export** (Desktop only):
```
1. File > Export Wallet
2. Choose location
3. Set export password
4. Save encrypted JSON file
5. Store in multiple locations
```

### Can I change my wallet address?

No. Your address is derived from your private key and cannot be changed. If you want a new address:

```
1. Create new wallet (new recovery phrase)
2. Transfer funds to new address
3. Use new wallet going forward
```

## Transaction Questions

### How long do transactions take?

**Typical confirmation times:**
- **Block inclusion**: 6 seconds (one block)
- **Final confirmation**: 6 seconds (Byzantine finality)
- **Exchange deposits**: 6-12 seconds (1-2 blocks)

**Factors affecting speed:**
- Network congestion (rare)
- Fee amount (higher = faster)
- Validator connectivity

### What are the transaction fees?

PAW uses a gas-based fee model. Typical fees:

| Transaction Type | Gas | Fee (at 0.025 upaw/gas) |
|-----------------|-----|-------------------------|
| Send Tokens | 100,000 | 0.0025 PAW (~$0.002) |
| DEX Swap | 200,000 | 0.005 PAW (~$0.005) |
| Add Liquidity | 250,000 | 0.00625 PAW (~$0.006) |
| Stake Tokens | 150,000 | 0.00375 PAW (~$0.004) |
| Vote | 100,000 | 0.0025 PAW (~$0.002) |

**Fee Levels:**
- Low: 0.01 upaw/gas (slower)
- Normal: 0.025 upaw/gas (recommended)
- High: 0.05 upaw/gas (faster)

### Why did my transaction fail?

**Common reasons:**

1. **Insufficient Balance**
   - Need balance + fee
   - Solution: Add more PAW

2. **Invalid Address**
   - Wrong format or typo
   - Solution: Verify address format

3. **Network Issues**
   - Node unavailable
   - Solution: Check network status

4. **Out of Gas**
   - Gas limit too low
   - Solution: Increase gas limit

5. **Slippage**
   - Price moved too much (DEX)
   - Solution: Increase slippage tolerance

**Check transaction on explorer**: https://explorer.pawchain.io

### Can I cancel a pending transaction?

Once broadcast, transactions cannot be cancelled. However:

- If truly stuck (rare), wait for timeout
- Don't re-send same transaction (creates duplicate)
- Contact support if urgent

**Prevention:**
- Double-check details before confirming
- Start with small test transaction
- Verify recipient address

### How do I speed up a slow transaction?

If using desktop wallet or CLI:

```
1. Check transaction status
2. If pending, can broadcast with higher fee
3. Use "Speed Up" feature if available
```

Most transactions confirm in 6 seconds, so "slow" is unusual.

### What is a memo and when should I use it?

A memo is optional text attached to transactions:

**When to use:**
- Exchange deposits (often required)
- Business transactions (invoice number)
- Personal notes (gift, payment for X)

**When NOT to use:**
- Normal wallet-to-wallet sends
- Privacy-sensitive transactions
- When not required

⚠️ Memos are publicly visible on the blockchain!

## DEX Questions

### How does the PAW DEX work?

The DEX uses an Automated Market Maker (AMM) model:

```
Liquidity Pools (e.g., PAW/USDC)
    ↓
Users trade against pool
    ↓
Price adjusts via x*y=k formula
    ↓
Liquidity providers earn 0.3% fee
```

**Benefits:**
- No order books
- Always liquid
- Permissionless
- Non-custodial

### What is slippage?

Slippage is the difference between expected and actual price:

**Example:**
```
You want: 100 USDC
Expected: Pay 100 PAW (1:1 rate)
Actual: Pay 100.5 PAW (rate moved)
Slippage: 0.5%
```

**Causes:**
- Large trade relative to pool size
- Price volatility
- Other trades executing simultaneously

**Setting slippage tolerance:**
- 0.1%: Very tight, may fail
- 0.5%: Recommended for stable pairs
- 1-5%: Volatile pairs or large trades

### How do I get the best price on swaps?

**Tips:**

1. **Compare Pools**
   - Multiple pools may exist for same pair
   - Check each for best rate

2. **Check Route**
   - Direct route: PAW → USDC
   - Multi-hop: PAW → ETH → USDC
   - Fewer hops = better price

3. **Timing**
   - Avoid peak trading hours
   - Check 24h volume trends

4. **Trade Size**
   - Smaller trades = less price impact
   - Consider splitting large orders

5. **Use Limit Orders** (coming soon)
   - Set exact price
   - Execute when available

### What is impermanent loss?

Impermanent loss occurs when token prices diverge:

**Example:**
```
Deposit: 100 PAW + 100 USDC (at 1:1)
Value: $200

PAW doubles to $2:
Pool rebalances to 70.7 PAW + 141.4 USDC
Value: $282.8

If just held: 100 PAW + 100 USDC
Value: $300

Impermanent Loss: $17.2 (5.7%)
```

**Mitigation:**
- Choose stable pairs (USDC/USDT)
- Fees compensate over time
- Understand risk before providing liquidity

**Not impermanent if:**
- Price returns to original ratio
- You withdraw at same price

### How do I provide liquidity?

**Step-by-step:**

```
1. Choose pool (e.g., PAW/USDC)
2. Deposit equal value of both tokens
3. Receive LP tokens representing your share
4. Earn 0.3% of all trading fees
5. Remove liquidity anytime by burning LP tokens
```

**Requirements:**
- Hold both tokens in pair
- Approve token spending (one-time)
- Pay gas for transactions

**See**: [Liquidity Tutorial](#liquidity-tutorial)

### What are LP tokens?

LP (Liquidity Provider) tokens are proof of your pool share:

```
You deposit: 100 PAW + 98.5 USDC
Pool total: 10,000 PAW + 9,850 USDC
Your share: 1%
You receive: 99.2 LP tokens

LP tokens = claim on:
- 1% of pool assets
- 1% of trading fees
```

**Important:**
- Don't lose LP tokens!
- Needed to withdraw liquidity
- Can be staked for extra rewards (coming soon)

## Staking Questions

### How does staking work?

Stake PAW to validators who secure the network:

```
You → Delegate PAW to Validator
Validator → Validates transactions and produces blocks
Network → Rewards validators
Validator → Shares rewards (minus commission)
You → Receive staking rewards
```

### What is the staking APY?

APY varies based on:
- Total amount staked (lower = higher APY)
- Inflation rate
- Validator commission

**Current estimates:**
- Base APY: 15-20%
- After 5% commission: 14.25-19%
- After 10% commission: 13.5-18%

**Check current rates**: https://dashboard.pawchain.io/staking

### What is the unbonding period?

When you undelegate (unstake), there's a **21-day unbonding period**:

⏳ **During unbonding:**
- Tokens locked (cannot use)
- No rewards earned
- Cannot cancel unbonding
- Cannot redelegate

✅ **After 21 days:**
- Tokens become liquid
- Can send/trade/stake elsewhere

**Purpose:** Security mechanism to prevent certain attacks

### How do I choose a validator?

**Key factors:**

1. **Commission** (5-10% typical)
   - Lower = more rewards for you
   - Too low = validator may not be sustainable

2. **Uptime** (aim for >99%)
   - Higher = more consistent rewards
   - Downtime = missed blocks = less rewards

3. **Voting Power** (<5% recommended)
   - Diversify for network health
   - Avoid top 10 validators

4. **Self-Delegation**
   - Validators with "skin in the game"
   - Shows commitment

5. **Community Involvement**
   - Active in governance
   - Good communication
   - Transparent operations

**Tool**: [Validator comparison](https://dashboard.pawchain.io/validators)

### Can I stake to multiple validators?

Yes! This is recommended:

**Benefits:**
- Risk diversification
- Support decentralization
- Optimize commission rates
- Maximize uptime

**Example strategy:**
```
1000 PAW split across:
- 400 PAW to Validator A (5% comm, 99.9% uptime)
- 300 PAW to Validator B (7% comm, 99.7% uptime)
- 300 PAW to Validator C (6% comm, 99.8% uptime)
```

### What happens if my validator gets slashed?

**Slashing** is a penalty for validator misbehavior:

**Reasons:**
- Double-signing blocks (severe)
- Extended downtime (minor)

**Penalties:**
- 5% slash for downtime
- 20% slash for double-signing (rare)

**Your stake:**
- You lose same percentage
- Very rare in practice
- Choose reputable validators

**Protection:**
- Diversify across validators
- Monitor validator performance
- Check community reputation

### How often should I claim rewards?

**Considerations:**

**Claim frequently:**
- ✅ Compound more often (higher returns)
- ❌ Pay more gas fees
- Best if: Large stake, low gas prices

**Claim infrequently:**
- ✅ Lower total gas fees
- ❌ Less compounding benefit
- Best if: Small stake

**Recommendation:**
- Small stake (<1000 PAW): Monthly
- Medium (1000-10000): Weekly
- Large (>10000): Daily or auto-compound when available

## Governance Questions

### How do I vote on proposals?

```
1. Go to governance dashboard
2. View active proposals
3. Click proposal to read details
4. Research and discuss
5. Click "Vote"
6. Choose: Yes, No, Abstain, or No with Veto
7. Confirm transaction
```

**Voting power** = your staked PAW

### What types of proposals are there?

1. **Text Proposals**
   - General community decisions
   - Non-binding recommendations

2. **Parameter Changes**
   - Modify blockchain parameters
   - E.g., fee amounts, block size

3. **Software Upgrades**
   - Coordinate network upgrades
   - Critical for major changes

4. **Community Pool Spending**
   - Allocate treasury funds
   - Fund development, marketing, etc.

### What does each vote option mean?

- **Yes**: Support the proposal
- **No**: Oppose but allow execution if passes
- **No with Veto**: Strong opposition, proposal fails if >33.4% veto
- **Abstain**: Participate in quorum but stay neutral

### Can I change my vote?

Yes, until the voting period ends:

```
1. Go to proposal
2. Click "Change Vote"
3. Select new option
4. Confirm
5. Previous vote overwritten
```

### What is the community pool?

A treasury of PAW tokens used for ecosystem development:

**Current balance**: Check on governance page

**Funding**: 2% of block rewards

**Usage**: Community spend proposals
- Development grants
- Marketing initiatives
- Security audits
- Community events

### How do I create a proposal?

**Requirements:**
- Minimum deposit: 100 PAW
- Clear description
- Community discussion

**Process:**
```
1. Draft proposal (use template)
2. Discuss on forum
3. Refine based on feedback
4. Submit on-chain
5. Deposit tokens
6. 2-week voting period
7. Implementation if passed
```

**See**: [Proposal Guide](#)

## Technical Questions

### What consensus mechanism does PAW use?

**Tendermint BFT** (Byzantine Fault Tolerant):

- Proof-of-Stake
- Instant finality
- Up to 1/3 malicious validators tolerated
- Used by Cosmos, Binance Chain, and others

### Is PAW compatible with Ethereum?

Not directly, but:

**Similar:**
- Same cryptography (secp256k1)
- Similar account system

**Different:**
- Different address format (bech32 vs hex)
- Different transaction structure
- Uses Cosmos SDK, not EVM

**Bridges:** Coming soon for cross-chain transfers

### What is IBC?

**Inter-Blockchain Communication Protocol**:

- Transfer assets between Cosmos chains
- Trustless bridging
- 100+ compatible chains
- PAW IBC: Coming Q2 2025

### Can I run my own validator?

Yes! Requirements:

**Hardware:**
- 4 CPU cores
- 16GB RAM
- 500GB SSD
- 100Mbps connection

**Stake:**
- Minimum self-delegation varies
- Recommend 10,000+ PAW

**Technical:**
- Linux server administration
- 24/7 monitoring
- Security best practices

**See**: [Validator Setup Guide](#)

## Security Questions

### Is PAW secure?

Yes, PAW employs multiple security measures:

- ✅ Audited codebase
- ✅ Tendermint BFT consensus
- ✅ Validator set of 100+
- ✅ Slashing for misbehavior
- ✅ Open-source code
- ✅ Bug bounty program

**Audits**: https://docs.pawchain.io/security

### What should I do if I'm scammed?

**Prevention** is key:
- Verify all URLs
- Never share recovery phrase
- Beware of impersonators
- Use official wallets only

**If scammed:**
1. **Don't panic**
2. **Move remaining funds** to new wallet immediately
3. **Report** to security@pawchain.io
4. **Alert community** on Discord/Forum
5. **File report** with local authorities if applicable

⚠️ Blockchain transactions are irreversible!

### How can I report a vulnerability?

PAW has a bug bounty program:

1. **Do NOT** disclose publicly
2. **Email**: security@pawchain.io
3. **Include**: Detailed report
4. **Wait**: For team response
5. **Receive**: Bounty reward

**Rewards**:
- Critical: $10,000 - $50,000
- High: $5,000 - $10,000
- Medium: $1,000 - $5,000
- Low: $500 - $1,000

**Details**: https://docs.pawchain.io/bug-bounty

## Troubleshooting

### My balance isn't showing

**Solutions:**
1. Refresh wallet
2. Check network connection
3. Verify correct network (mainnet vs testnet)
4. Try different RPC endpoint
5. Update wallet software
6. Re-import wallet if needed

### Transaction is stuck

**Steps:**
1. Wait 5 minutes (may just be delayed)
2. Check explorer: https://explorer.pawchain.io
3. Verify transaction was broadcast
4. Check RPC status: https://status.pawchain.io
5. Contact support if >1 hour

### Cannot connect to network

**Solutions:**
1. Check internet connection
2. Verify RPC endpoint
3. Try alternative RPC:
   - https://rpc.pawchain.io
   - https://rpc2.pawchain.io
4. Check firewall/VPN
5. Update wallet

## Getting Help

### Where can I get support?

**Documentation**: [docs.pawchain.io](https://docs.pawchain.io)
**Discord**: [discord.gg/pawchain](https://discord.gg/pawchain)
**Telegram**: [t.me/pawchain](https://t.me/pawchain)
**Forum**: [forum.pawchain.io](https://forum.pawchain.io)
**Email**: support@pawchain.io

**Response times:**
- Discord/Telegram: Minutes to hours
- Forum: Hours to 1 day
- Email: 1-3 business days

### Is there a knowledge base?

Yes! This documentation site:
- [Getting Started](#getting-started)
- [User Guide](#user-guide)
- [Developer Guide](#developer-guide)
- [Tutorials](#tutorials)
- [This FAQ](#faq)

### How can I contribute?

**Ways to help:**

1. **Code**: Submit PRs on 
2. **Documentation**: Improve docs
3. **Community**: Help others on Discord
4. **Translation**: Translate docs
5. **Testing**: Test new features
6. **Marketing**: Spread the word

**Start here**: https://github.com/paw-chain/paw

---

*Can't find your question? Ask on [Discord](https://discord.gg/pawchain) or [Forum](https://forum.pawchain.io)*
