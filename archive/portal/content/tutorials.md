# Tutorials

Step-by-step guides for common PAW Blockchain tasks.

## Beginner Tutorials

### Tutorial 1: Creating Your First Wallet

**Duration:** 5 minutes
**Difficulty:** Beginner
**Prerequisites:** None

#### What You'll Learn
- How to create a new wallet
- Understanding recovery phrases
- Basic wallet security

#### Steps

**1. Download Wallet**

Choose your platform:
- Browser Extension: [Chrome](#) | [Firefox](#)
- Desktop: [Windows](#) | [macOS](#) | [Linux](#)
- Mobile: [iOS](#) | [Android](#)

**2. Install and Launch**

```
Browser: Click extension icon
Desktop: Open application
Mobile: Tap app icon
```

**3. Create New Wallet**

Click "Create New Wallet" and follow prompts:

```
1. Choose strong password (12+ characters)
2. Save 24-word recovery phrase
3. Verify recovery phrase
4. Wallet created!
```

**4. Backup Recovery Phrase**

⚠️ **Critical Step:**
```
Write down all 24 words in order
Store in safe place (fireproof, waterproof)
Never store digitally
Never share with anyone
```

**5. Verify Backup**

Re-enter words to confirm you saved them correctly.

**6. Your Address**

```
paw1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0
```

Copy or share QR code to receive tokens.

#### Video Tutorial
[![Watch Tutorial](/assets/video-thumbnail.svg)](#)

---

### Tutorial 2: Making Your First Transaction

**Duration:** 10 minutes
**Difficulty:** Beginner
**Prerequisites:** Created wallet, have test tokens

#### What You'll Learn
- How to send PAW tokens
- Understanding transaction fees
- Verifying transactions

#### Steps

**1. Get Test Tokens**

Visit the faucet: https://faucet.pawchain.io
```
1. Enter your address
2. Complete captcha
3. Click "Request Tokens"
4. Receive 100 test PAW
```

**2. Open Send Interface**

In your wallet:
```
Click "Send" or "Transfer" button
```

**3. Enter Details**

```
Recipient: paw1def...uvw (or scan QR code)
Amount: 10 PAW
Memo: "My first transaction" (optional)
```

**4. Choose Fee**

```
🐢 Low: 0.0001 PAW (slower)
🚶 Normal: 0.001 PAW (recommended)
🚀 High: 0.01 PAW (faster)
```

**5. Review Transaction**

```
From: paw1abc...xyz
To: paw1def...uvw
Amount: 10 PAW
Fee: 0.001 PAW
Total: 10.001 PAW
```

**6. Confirm and Send**

Click "Confirm" and enter password/biometric.

**7. Track Transaction**

```
Transaction Hash: ABC123...
Status: Pending → Confirmed ✓
Time: ~6 seconds
```

View on explorer: https://explorer.pawchain.io/tx/ABC123...

#### Troubleshooting

**Transaction Failed:**
- Check balance includes fee
- Verify recipient address
- Try higher fee

**Transaction Pending:**
- Normal for up to 30 seconds
- Network may be congested
- Don't send again (duplicate)

---

### Tutorial 3: Using the DEX to Swap Tokens

**Duration:** 15 minutes
**Difficulty:** Beginner
**Prerequisites:** Wallet with PAW tokens

#### What You'll Learn
- How to swap tokens on DEX
- Understanding slippage
- Getting best prices

#### Steps

**1. Navigate to DEX**

```
Option A: Built-in wallet DEX tab
Option B: https://dex.pawchain.io
```

**2. Connect Wallet**

Click "Connect Wallet" and authorize.

**3. Select Token Pair**

```
From: PAW
To: USDC
```

**4. Enter Amount**

```
Input: 100 PAW
Expected Output: ~98.5 USDC
Exchange Rate: 1 PAW = 0.985 USDC
```

**5. Check Slippage**

Click settings ⚙️:
```
Slippage Tolerance: 0.5% (recommended)

Too low = transaction may fail
Too high = may get worse price
```

**6. Review Swap Details**

```
Minimum Received: 97.5 USDC (with 0.5% slippage)
Liquidity Provider Fee: 0.3%
Route: PAW → USDC (direct)
```

**7. Execute Swap**

```
1. Click "Swap"
2. Review one more time
3. Confirm transaction
4. Wait for confirmation
```

**8. Verify New Balance**

Check wallet:
```
PAW Balance: 900 (was 1000)
USDC Balance: 98.5 (was 0)
```

#### Advanced Tips

**Getting Best Price:**
- Compare multiple pools
- Check route optimization
- Consider timing (avoid peak hours)
- Use limit orders (coming soon)

**Understanding Price Impact:**
```
Small trades (<1% of pool): Minimal impact
Large trades (>5% of pool): Significant impact
```

---

## Intermediate Tutorials

### Tutorial 4: Providing Liquidity to Earn Fees

**Duration:** 20 minutes
**Difficulty:** Intermediate
**Prerequisites:** Understanding of DEX, tokens in wallet

#### What You'll Learn
- How liquidity pools work
- Adding liquidity
- Earning trading fees
- Impermanent loss basics

#### Understanding Liquidity Pools

```
Pool = PAW + USDC
Users trade → Pay 0.3% fee
Fees distributed to liquidity providers
You earn proportional to your share
```

#### Steps

**1. Choose Pool**

Browse available pools:
```
PAW/USDC
- TVL: $10M
- Volume (24h): $500K
- APY: 45%
- Your Share: 0%

PAW/ETH
- TVL: $5M
- Volume (24h): $300K
- APY: 38%
- Your Share: 0%
```

**2. Calculate Required Amounts**

For balanced deposit:
```
Pool Ratio: 1 PAW = 0.985 USDC
If depositing 100 PAW:
Need: 98.5 USDC
```

**3. Add Liquidity**

```
1. Click "Add Liquidity"
2. Enter amount for one token
3. Other token auto-calculated
4. Review pool share percentage
5. Approve token spending (first time)
6. Confirm transaction
```

**4. Receive LP Tokens**

```
You receive: 99.2 PAW-USDC LP tokens
Represents your pool share: 0.5%
```

**5. Track Earnings**

```
Initial: 100 PAW + 98.5 USDC
Fees Earned: 0.15 PAW + 0.148 USDC/day
APY: ~45% (variable)
```

**6. Removing Liquidity**

When ready to exit:
```
1. Go to "Your Liquidity"
2. Select pool
3. Choose amount (1-100%)
4. Preview tokens to receive
5. Confirm removal
6. Receive tokens + fees
```

#### Understanding Impermanent Loss

```
If prices diverge:
- You may have less value than just holding
- Trading fees compensate over time
- More stable pairs = less risk
```

**Example:**
```
Deposited: 100 PAW + 98.5 USDC
PAW price doubles
Pool rebalances: 70.7 PAW + 139.3 USDC
Value: $278.6

If held: 100 PAW + 98.5 USDC
Value: $295.0

Impermanent Loss: $16.4 (5.6%)
Fees Earned (30 days): $25.0
Net Profit: $8.6
```

---

### Tutorial 5: Staking PAW Tokens

**Duration:** 25 minutes
**Difficulty:** Intermediate
**Prerequisites:** PAW tokens in wallet

#### What You'll Learn
- How staking works
- Choosing validators
- Managing delegations
- Claiming rewards

#### How Staking Works

```
Your PAW → Delegate to Validator
Validator → Secures Network
Network → Produces Blocks
Block Rewards → Distributed to Stakers
You Earn → ~15-20% APY
```

#### Steps

**1. Research Validators**

Key metrics to compare:

```
Validator Performance:
┌─────────────────┬──────────┬────────┬─────────┐
│ Validator       │ Voting   │ Comm.  │ Uptime  │
├─────────────────┼──────────┼────────┼─────────┤
│ Validator A     │ 2.5%     │ 5%     │ 99.9%   │
│ Validator B     │ 8.0%     │ 10%    │ 99.5%   │
│ Validator C     │ 1.2%     │ 7%     │ 99.8%   │
└─────────────────┴──────────┴────────┴─────────┘

Lower voting power = better decentralization
Lower commission = higher rewards
Higher uptime = more reliable
```

**2. Delegate Tokens**

```
1. Navigate to "Staking"
2. Browse/search validators
3. Click validator
4. Click "Delegate"
5. Enter amount
6. Review details:
   - Unbonding period: 21 days
   - Estimated APY: ~18%
7. Confirm transaction
```

**3. Track Your Stake**

```
Staked: 1000 PAW
Validator: Validator A
Commission: 5%
Daily Rewards: ~0.49 PAW
Monthly: ~14.8 PAW
Yearly: ~180 PAW (18% APY)
```

**4. Claim Rewards**

Rewards accumulate and don't auto-compound:

```
1. Go to "Rewards" tab
2. View accumulated rewards
3. Click "Claim Rewards"
4. Confirm transaction
5. Rewards added to balance
```

**Best Practice:** Claim and restake regularly to compound.

**5. Restaking (Compounding)**

```
1. Claim rewards
2. Immediately delegate again
3. Increases future rewards
4. Max compound effect
```

**Compound Calculator:**
```
Initial Stake: 1000 PAW
APY: 18%
Strategy: Monthly compound

1 year: 1196 PAW (19.6% effective)
5 years: 2331 PAW
10 years: 5436 PAW
```

**6. Undelegating**

To get your tokens back:

```
1. Click "Undelegate"
2. Enter amount
3. Confirm
4. Wait 21 days (unbonding period)
5. Tokens become liquid
```

⚠️ **During unbonding:**
- No rewards earned
- Cannot use tokens
- Cannot cancel undelegation

**7. Redelegating**

Move stake between validators instantly:

```
1. Click "Redelegate"
2. Choose new validator
3. Enter amount
4. Confirm
5. Instant move (once per 21 days)
```

#### Advanced Strategies

**Diversification:**
```
Split stake across 3-5 validators:
- Reduces single validator risk
- Supports decentralization
- Maintains good rewards
```

**Commission Shopping:**
```
5% commission on 18% APY = 17.1% effective
10% commission on 18% APY = 16.2% effective
Difference: 0.9% per year

On 10,000 PAW = 90 PAW/year difference
```

**Validator Selection Checklist:**
- ✅ Good uptime (>99%)
- ✅ Reasonable commission (<10%)
- ✅ Active governance participation
- ✅ Transparent communication
- ✅ Strong security practices
- ✅ Not in top 10 (decentralization)

---

## Advanced Tutorials

### Tutorial 6: Building Your First dApp

**Duration:** 2 hours
**Difficulty:** Advanced
**Prerequisites:** JavaScript knowledge, Node.js installed

#### What You'll Build

A simple token balance checker dApp.

#### Project Setup

```bash
# Create project
mkdir paw-dapp
cd paw-dapp
npm init -y

# Install dependencies
npm install @paw-chain/sdk vite

# Create files
touch index.html app.js
```

#### Code Implementation

**index.html:**
```html
<!DOCTYPE html>
<html>
<head>
    <title>PAW Balance Checker</title>
</head>
<body>
    <h1>PAW Balance Checker</h1>
    <input id="address" placeholder="Enter PAW address">
    <button onclick="checkBalance()">Check Balance</button>
    <div id="result"></div>
    <script type="module" src="/app.js"></script>
</body>
</html>
```

**app.js:**
```javascript
import { PAWClient } from '@paw-chain/sdk';

const client = new PAWClient({
  rpcEndpoint: 'https://rpc.pawchain.io',
  chainId: 'paw-1'
});

window.checkBalance = async () => {
  const address = document.getElementById('address').value;
  const result = document.getElementById('result');

  try {
    const balance = await client.bank.getBalance(address, 'upaw');
    const paw = (parseInt(balance.amount) / 1000000).toFixed(6);
    result.innerHTML = `<h2>${paw} PAW</h2>`;
  } catch (error) {
    result.innerHTML = `<p style="color: red;">${error.message}</p>`;
  }
};
```

#### Run Development Server

```bash
npx vite
# Open http://localhost:5173
```

#### Test Your dApp

```
Enter address: paw1abc...xyz
Click "Check Balance"
See: 1000.000000 PAW
```

#### Next Steps

- Add wallet connection
- Display multiple token balances
- Show transaction history
- Add send functionality
- Deploy to production

---

## Video Tutorials

### Getting Started Series
- **Introduction to PAW** (5:00) - Overview of PAW Blockchain
- **Creating Your Wallet** (8:00) - Step-by-step wallet setup
- **Your First Transaction** (6:00) - Sending and receiving tokens
- **Using the Faucet** (3:00) - Getting test tokens

### Trading on DEX
- **DEX Basics** (10:00) - Understanding the DEX
- **Token Swapping** (12:00) - How to swap tokens
- **Providing Liquidity** (15:00) - Earning from liquidity pools
- **Advanced Trading** (20:00) - Strategies and tips

### Staking Guide
- **Staking Explained** (8:00) - How staking works
- **Choosing Validators** (12:00) - Validator selection guide
- **Managing Stakes** (10:00) - Delegating and claiming rewards
- **Advanced Staking** (18:00) - Optimization strategies

### Developer Tutorials
- **Development Setup** (15:00) - Setting up environment
- **Building with SDK** (30:00) - Using PAW SDKs
- **Smart Contracts** (45:00) - Writing and deploying contracts
- **Full dApp Tutorial** (60:00) - Building complete application

## Community Tutorials

Submit your own tutorials! Visit [](https://github.com/paw-chain/paw) to contribute.

## Support

Need help with tutorials?
- **Discord**: [discord.gg/pawchain](https://discord.gg/pawchain)
- **Forum**: [forum.pawchain.io](https://forum.pawchain.io)
- **Documentation**: [Full docs](#)
