# User Guide

Complete guide for PAW Blockchain users covering wallets, transactions, DEX trading, staking, and governance.

## Wallet Management

### Browser Extension Wallet

#### Installation
1. Visit [Chrome Web Store](#) or [Firefox Add-ons](#)
2. Click "Add to Browser"
3. Accept permissions
4. Pin extension to toolbar

#### Creating a Wallet
```
1. Click PAW extension icon
2. Select "Create New Wallet"
3. Choose strong password
4. Save 24-word recovery phrase
5. Verify recovery phrase
6. Wallet ready to use
```

#### Importing Existing Wallet
```
1. Click "Import Wallet"
2. Enter 24-word recovery phrase
3. Set new password
4. Wallet restored
```

### Desktop Wallet

Full-featured application available for all platforms.

#### Features
- Multi-account management
- Transaction history with export
- Address book
- Hardware wallet support
- Network configuration
- Dark/Light theme

#### Backup and Recovery
```bash
# Export wallet (encrypted)
File > Export Wallet > Save JSON

# Import wallet
File > Import Wallet > Select JSON file
```

### Mobile Wallet

#### Features
- Biometric authentication (Face ID, Touch ID, Fingerprint)
- QR code scanner for addresses
- Push notifications
- Quick send with contacts
- Portfolio tracking

#### Security Settings
- Auto-lock timer
- Require authentication for transactions
- Hide balance on home screen
- Screenshot prevention

## Sending and Receiving Tokens

### Receiving Tokens

#### Get Your Address
1. Open wallet
2. Click "Receive"
3. Copy address or show QR code
4. Share with sender

#### Address Format
```
paw1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0
```

### Sending Tokens

#### Step-by-Step
1. Click "Send" or "Transfer"
2. Enter recipient address (or scan QR)
3. Enter amount
4. Choose fee level:
   - 🐢 Low: Slower confirmation
   - 🚶 Normal: Standard speed
   - 🚀 High: Fast confirmation
5. Review transaction details
6. Confirm and sign

#### Transaction Example
```javascript
From: paw1abc...xyz
To: paw1def...uvw
Amount: 100 PAW
Fee: 0.001 PAW (Normal)
Total: 100.001 PAW
```

## Using the DEX

PAW's built-in decentralized exchange allows peer-to-peer token trading.

### Swapping Tokens

#### Simple Swap
1. Go to DEX interface
2. Select "Swap" tab
3. Choose token pair (e.g., PAW/USDC)
4. Enter amount to swap
5. Review exchange rate and fees
6. Set slippage tolerance (0.5% - 5%)
7. Click "Swap"
8. Confirm transaction

#### Advanced Settings
```
Slippage Tolerance: 0.5% - 5%
- Higher slippage = more likely to succeed
- Lower slippage = better price guarantee

Transaction Deadline: 20 minutes
- Cancel if not executed within time limit
```

### Adding Liquidity

Provide liquidity to earn trading fees.

#### Step 1: Select Pool
```
Available Pools:
- PAW/USDC (APY: 45%)
- PAW/ETH (APY: 38%)
- USDC/USDT (APY: 12%)
```

#### Step 2: Add Liquidity
1. Click pool to enter
2. Enter amount for both tokens
3. Preview pool share percentage
4. Review estimated APY
5. Approve token spending (first time)
6. Add liquidity
7. Receive LP tokens

#### Earning Fees
- Earn proportional share of trading fees
- Fees automatically compound
- Claim anytime

### Removing Liquidity
1. Go to "Your Liquidity"
2. Select pool
3. Choose percentage to remove (1-100%)
4. Preview tokens to receive
5. Confirm removal
6. Receive tokens + earned fees

## Staking

Stake PAW tokens to earn rewards and secure the network.

### How Staking Works

```
Your Stake → Validator → Network Security
                ↓
            Block Rewards
                ↓
         Your Staking Rewards
```

### Choosing a Validator

#### Key Metrics
- **Voting Power**: Size of validator (diversify for network health)
- **Commission**: Fee charged (typically 5-10%)
- **Uptime**: Reliability (aim for 99%+)
- **Self-Stake**: Validator's own stake (skin in the game)

#### Example Comparison
```
Validator A:
- Voting Power: 2.5%
- Commission: 5%
- Uptime: 99.9%
- APY: ~18%

Validator B:
- Voting Power: 8.0%
- Commission: 10%
- Uptime: 99.5%
- APY: ~16%
```

### Delegating Tokens

1. Go to "Staking" tab
2. Browse validator list
3. Click validator to view details
4. Click "Delegate"
5. Enter amount to stake
6. Review terms:
   - Unbonding period: 21 days
   - Rewards: Claimed separately
   - Slashing: Rare, for misbehavior
7. Confirm delegation

### Managing Delegations

#### Claiming Rewards
```
1. Go to "Staking" > "Rewards"
2. View accumulated rewards
3. Click "Claim All" or select specific validator
4. Confirm transaction
5. Rewards added to balance
```

#### Restaking Rewards
Auto-compound by immediately restaking claimed rewards.

#### Undelegating
```
1. Select delegated validator
2. Click "Undelegate"
3. Enter amount
4. Confirm (21-day unbonding period starts)
5. Tokens liquid after 21 days
```

#### Redelegating
Move stake to different validator instantly (once per 21 days).

```
1. Select current validator
2. Click "Redelegate"
3. Choose new validator
4. Enter amount
5. Confirm
```

## Governance

Participate in PAW's on-chain governance.

### Proposal Types

1. **Text Proposal**: General community decisions
2. **Parameter Change**: Modify blockchain parameters
3. **Software Upgrade**: Coordinate network upgrades
4. **Community Spend**: Allocate community pool funds

### Voting Process

#### Step 1: View Proposals
```
Proposal #42: Increase Block Size
Status: Voting Period
Ends: 2025-11-26 14:00 UTC
Description: Proposal to increase max block size...
```

#### Step 2: Research
- Read proposal details
- Review discussion on forum
- Check validator votes
- Consider implications

#### Step 3: Vote
```
Vote Options:
- Yes: Support the proposal
- No: Oppose the proposal
- Abstain: Counted in quorum, but neutral
- No with Veto: Strong opposition (triggers rejection if >33.4%)
```

#### Step 4: Submit Vote
1. Click "Vote"
2. Select your choice
3. Optionally add comment
4. Confirm transaction

### Creating Proposals

Requirements:
- Minimum deposit: 100 PAW
- Clear description
- Implementation plan

Process:
1. Draft proposal text
2. Discuss in community forum
3. Submit on-chain
4. Deposit tokens
5. 2-week voting period
6. Implementation if passed

## Security Best Practices

### Wallet Security

✅ **DO:**
- Use strong, unique passwords
- Enable 2FA where available
- Store recovery phrase offline
- Use hardware wallet for large amounts
- Keep software updated
- Verify addresses before sending
- Test with small amounts first

❌ **DON'T:**
- Share private keys or recovery phrases
- Use public WiFi for transactions
- Click suspicious links
- Trust unsolicited messages
- Store recovery phrases digitally
- Reuse passwords

### Transaction Security

- Always verify recipient address
- Double-check amounts
- Review fees before confirming
- Be cautious of phishing sites
- Bookmark official URLs
- Enable transaction notifications

### Common Scams

⚠️ **Phishing**: Fake websites stealing credentials
⚠️ **Impersonation**: Fake support asking for keys
⚠️ **Ponzi Schemes**: Unrealistic returns promises
⚠️ **Fake Airdrops**: Malicious smart contracts

### If Compromised

1. **Don't Panic**: Act quickly but carefully
2. **Move Funds**: Transfer to new wallet immediately
3. **Report**: Notify PAW security team
4. **Investigate**: Identify how compromise occurred
5. **Secure**: Create new wallet with better security

## Troubleshooting

### Transaction Failed

**Causes:**
- Insufficient balance for fees
- Invalid address format
- Network congestion
- Outdated wallet software

**Solutions:**
- Check balance includes fee
- Verify address is correct PAW address
- Increase gas price
- Update wallet

### Can't See Balance

**Solutions:**
- Check network connection
- Verify correct network (mainnet/testnet)
- Refresh wallet
- Check RPC endpoint status
- Try alternative RPC endpoint

### Lost Recovery Phrase

**Unfortunately:**
- Recovery phrase is the ONLY way to restore wallet
- No recovery possible without it
- PAW team cannot help (we don't have access)
- Funds are permanently inaccessible

**Prevention:**
- Write on paper, store in safe
- Consider metal backup plates
- Store in multiple secure locations
- Never digital photos or cloud storage

## Support Resources

- **Documentation**: [docs.pawchain.io](#)
- **FAQ**: [#faq](#faq)
- **Discord**: [discord.gg/DBHTc2QV](https://discord.gg/DBHTc2QV)
- **Telegram**: [t.me/pawchain](https://t.me/pawchain)
- **Forum**: [forum.pawchain.io](https://forum.pawchain.io)
- **Email**: support@pawchain.io
