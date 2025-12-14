# PAW Testing Control Panel - User Guide

This guide will help you get started with the PAW Testing Control Panel, whether you're a blockchain expert or completely new to blockchain technology.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Understanding the Dashboard](#understanding-the-dashboard)
3. [Common Tasks](#common-tasks)
4. [Testing Scenarios](#testing-scenarios)
5. [Tips & Tricks](#tips--tricks)
6. [Troubleshooting](#troubleshooting)

---

## Getting Started

### What is this dashboard?

The PAW Testing Control Panel is a web application that lets you interact with the PAW blockchain without writing any code. You can:
- Send and receive tokens
- Create wallets
- Monitor blockchain activity
- Test all features of the network
- See real-time statistics

### Opening the Dashboard

**Simplest Method:**
1. Navigate to the `testing-dashboard` folder
2. Double-click `index.html`
3. Your browser will open the dashboard

**Using a Web Server (Recommended):**
```bash
# If you have Python installed:
cd testing-dashboard
python -m http.server 8080

# Then open your browser to: http://localhost:8080
```

### Choosing Your Network

When you first open the dashboard, you'll see a network selector at the top:

- **Local Testnet**: Use this if you're running a PAW node on your computer
- **Public Testnet**: Use this to connect to the shared test network
- **Mainnet (Read-Only)**: Use this to monitor the real network (you can't send transactions)

**For beginners**: Start with **Public Testnet**

---

## Understanding the Dashboard

### Dashboard Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [PAW Logo]  Testing Control Panel      [Network] [Status]  │
├───────────┬─────────────────────────────────┬───────────────┤
│           │                                 │               │
│  Quick    │    Main Display Area            │  Live Logs    │
│  Actions  │    - Network Overview           │  & Events     │
│           │    - Blocks/Transactions        │               │
│  Testing  │    - Validators/Proposals       │  System       │
│  Tools    │    - Liquidity Pools            │  Metrics      │
│           │                                 │               │
│  Test     │                                 │               │
│  Scenarios│                                 │               │
│           │                                 │               │
└───────────┴─────────────────────────────────┴───────────────┘
```

### Left Sidebar: Quick Actions

These are the most common tasks you'll perform:

1. **Send Transaction** - Transfer tokens to another wallet
2. **Create Wallet** - Generate a new blockchain wallet
3. **Delegate Tokens** - Stake your tokens with a validator
4. **Submit Proposal** - Create a governance proposal
5. **Swap Tokens** - Trade tokens on the DEX
6. **Query Balance** - Check any wallet's balance

### Center: Main Display

This shows different information depending on which tab you select:

- **Recent Blocks**: Latest blocks added to the blockchain
- **Recent Transactions**: Recent transaction activity
- **Validators**: Active validators you can stake with
- **Proposals**: Governance proposals you can vote on
- **Liquidity Pools**: DEX pools for trading

### Right Sidebar: Monitoring

- **Live Logs**: See what's happening in real-time
- **Recent Events**: Important blockchain events
- **System Metrics**: Node performance (CPU, Memory, Disk)

---

## Common Tasks

### Task 1: Create Your First Wallet

**Why?** You need a wallet to interact with the blockchain.

**Steps:**
1. Click **"Create Wallet"** in the left sidebar
2. A popup will appear with three important pieces:
   - **Address**: Your wallet's public address (like an account number)
   - **Mnemonic Phrase**: 12-24 words to recover your wallet
   - **Private Key**: Secret key to access your wallet

3. **IMPORTANT**: Write down your mnemonic phrase on paper!
4. Click **"Copy Address"** to copy your wallet address
5. Click **"Close"**

**What you learned**: Your wallet address starts with "paw1..."

---

### Task 2: Get Test Tokens

**Why?** You need tokens to test sending transactions.

**Steps:**
1. Make sure you have a wallet address (from Task 1)
2. Click **"Request Tokens"** in Testing Tools section
3. Paste your wallet address
4. Click **"Request Tokens"**
5. Wait 10-30 seconds
6. Check the logs (right side) for "Tokens received"

**What you learned**: The faucet gives you free test tokens (not real money!)

---

### Task 3: Check Your Balance

**Why?** Verify that you received tokens from the faucet.

**Steps:**
1. Click **"Query Balance"** in Quick Actions
2. Enter your wallet address
3. Click **"Query"**
4. You'll see your balance displayed

**What you learned**: Token amounts are shown in "upaw" (micro-PAW). 1 PAW = 1,000,000 upaw

---

### Task 4: Send a Transaction

**Why?** This is the most common blockchain operation.

**Steps:**
1. Click **"Send Transaction"** in Quick Actions
2. Fill in the form:
   - **From**: Your wallet address
   - **To**: Recipient's address
   - **Amount**: How much to send (in upaw)
   - **Memo**: Optional message

3. **OR** Click **"Use Test Data"** to auto-fill
4. Click **"Send Transaction"**
5. Watch the logs for confirmation

**What you learned**: Transactions need a "from" address, "to" address, and amount.

---

### Task 5: View Validators

**Why?** To see who's securing the network.

**Steps:**
1. Click the **"Validators"** tab in the main area
2. You'll see a list of validators with:
   - Name (Moniker)
   - Status (Active/Inactive)
   - Voting Power
   - Commission rate

**What you learned**: Validators process transactions and create blocks.

---

### Task 6: Run a Test Scenario

**Why?** To test complete workflows automatically.

**Steps:**
1. Scroll down in the left sidebar to "Test Scenarios"
2. Click **"Transaction Flow"**
3. Watch the logs as it:
   - Creates a wallet
   - Requests tokens
   - Checks balance
   - Simulates a transaction

**What you learned**: Test scenarios automate multiple steps for you!

---

## Testing Scenarios

### What are Test Scenarios?

Test scenarios are pre-programmed sequences that automatically test blockchain features. Think of them as "autopilot" for testing.

### Available Scenarios

#### 1. Transaction Flow
**What it does:**
- Creates a new test wallet
- Requests tokens from the faucet
- Checks the wallet balance
- Attempts to send a transaction

**When to use:** Testing basic wallet and transaction functionality

**How to use:** Click "Transaction Flow" button and watch the logs

---

#### 2. Staking Flow
**What it does:**
- Fetches the list of active validators
- Gets current staking information
- Simulates delegating tokens to a validator

**When to use:** Testing staking/delegation features

**How to use:** Click "Staking Flow" button

---

#### 3. Governance Flow
**What it does:**
- Lists all active governance proposals
- Simulates submitting a new proposal
- Simulates voting on a proposal

**When to use:** Testing governance and voting features

**How to use:** Click "Governance Flow" button

---

#### 4. DEX Trading Flow
**What it does:**
- Fetches liquidity pools
- Simulates a token swap
- Simulates adding liquidity to a pool

**When to use:** Testing DEX (exchange) features

**How to use:** Click "DEX Trading Flow" button

---

## Tips & Tricks

### Tip 1: Use Test Data

Many forms have a **"Use Test Data"** button that automatically fills in valid test values. This saves time and ensures correct format.

### Tip 2: Watch the Logs

The live logs (right sidebar) show everything that happens. If something doesn't work, check the logs for error messages.

### Tip 3: Export Logs

Click the download icon above the logs to export them as a JSON file. Useful for sharing errors with support.

### Tip 4: Switch Themes

Click the moon/sun icon in the top-right to toggle between light and dark themes. Your preference is saved automatically.

### Tip 5: Network Status

Always check the status indicator (top-right):
- 🟢 Green = Connected and ready
- 🟡 Yellow = Connecting
- 🔴 Red = Disconnected (check your network)

### Tip 6: Use Help Button

Click the **?** button (bottom-right) anytime for quick help and documentation links.

### Tip 7: Read-Only Mainnet

When on Mainnet, the dashboard is READ-ONLY. You can view everything but cannot send transactions. This protects against accidental real-money transactions.

---

## Troubleshooting

### Problem: Dashboard Won't Load

**Symptoms:**
- White/blank screen
- Error messages in browser

**Solutions:**
1. Try a different browser (Chrome, Firefox, Safari)
2. Clear your browser cache
3. Make sure JavaScript is enabled
4. Check browser console for errors (F12 key)

---

### Problem: Status Shows "Disconnected"

**Symptoms:**
- Red status indicator
- No data loading
- "Loading..." messages don't go away

**Solutions:**

**For Local Testnet:**
1. Make sure your PAW node is running
2. Check if node is listening on port 26657 and 1317
3. Try: `curl http://localhost:26657/status`

**For Public Testnet:**
1. Check your internet connection
2. Try switching to a different network
3. Refresh the page

**For All Networks:**
1. Check `config.js` for correct URLs
2. Look for errors in browser console
3. Check if firewall is blocking connections

---

### Problem: Transaction Won't Send

**Symptoms:**
- "Transaction failed" in logs
- Error messages

**Common Causes & Solutions:**

**Cause 1: On Mainnet**
- Solution: Switch to Testnet (can't send tx on mainnet)

**Cause 2: Insufficient Balance**
- Solution: Request tokens from faucet first
- Check balance before sending

**Cause 3: Invalid Address**
- Solution: Make sure addresses start with "paw1"
- Use "Use Test Data" to get valid addresses

**Cause 4: Wrong Amount Format**
- Solution: Amount should be in upaw (whole numbers)
- Example: 1000000 (not 1.0 or 1,000,000)

---

### Problem: Faucet Not Working

**Symptoms:**
- "Faucet request failed" in logs
- No tokens received

**Solutions:**
1. Make sure you're on Local or Public Testnet (not Mainnet)
2. Wait 60 seconds between requests (rate limited)
3. Check if faucet service is running (for local)
4. Verify address format is correct

---

### Problem: Balance Shows Zero

**Symptoms:**
- Query returns empty or zero balance

**Solutions:**
1. Wait 30 seconds after faucet request
2. Make sure you used the correct address
3. Check if transaction was successful in logs
4. Try refreshing the page

---

### Problem: Data Not Updating

**Symptoms:**
- Old data showing
- Numbers don't change

**Solutions:**
1. Click on a different tab, then back
2. Refresh your browser
3. Check if auto-refresh is enabled (config.js)
4. Check network connection

---

## Understanding Blockchain Terms

### Wallet
A digital account that can hold tokens. Like a bank account but you control it completely.

### Address
Your wallet's public identifier. Like an account number that others can send tokens to.

### Mnemonic / Seed Phrase
12-24 words that can restore your wallet. Like a master password - NEVER share it!

### Private Key
Secret key that proves you own a wallet. Like your PIN code - NEVER share it!

### Transaction (TX)
A transfer of tokens from one wallet to another.

### Block
A group of transactions added to the blockchain together.

### Validator
A node that processes transactions and secures the network.

### Staking / Delegation
Locking your tokens with a validator to help secure the network and earn rewards.

### Proposal
A suggestion for changing the blockchain rules that token holders vote on.

### DEX (Decentralized Exchange)
A platform for trading tokens without a central authority.

### Liquidity Pool
A pool of tokens that enables trading on the DEX.

### Gas / Fee
A small amount paid to process transactions (like a transaction fee).

### Testnet
A test version of the blockchain using fake tokens (for testing only).

### Mainnet
The real blockchain with real tokens that have real value.

---

## Getting Help

### Built-in Help
- Click the **?** button anytime
- Hover over buttons for tooltips
- Check the logs for error messages

### Documentation
- [PAW Documentation](https://docs.paw.network)
- [API Reference](https://docs.paw.network/api)
- [Video Tutorials](https://docs.paw.network/videos)

### Community Support
- Discord: Join #testing-help channel
- Forum: Post in Testing category
- : Open an issue

### Contact Support
- Email: support@paw.network
- Include:
  - What you were trying to do
  - Error messages from logs
  - Screenshots if helpful
  - Browser and operating system

---

## Next Steps

Once you're comfortable with the basics:

1. **Explore All Tabs**: Click through Blocks, Transactions, Validators, Proposals, and Liquidity
2. **Try All Quick Actions**: Test each button in the Quick Actions section
3. **Run All Test Scenarios**: Execute each automated test flow
4. **Customize Settings**: Edit `config.js` to change update intervals or endpoints
5. **Read the Code**: All code is documented - learn how it works!

---

## Safety Reminders

⚠️ **NEVER use this dashboard with real private keys or mnemonics**

⚠️ **NEVER send real tokens - this is for testing only**

⚠️ **ALWAYS save your mnemonic phrase securely (write it down!)**

⚠️ **ALWAYS verify you're on Testnet before experimenting**

⚠️ **NEVER share your private keys or mnemonics with anyone**

---

**Happy Testing!**

The PAW Testing Control Panel is designed to make blockchain testing easy and accessible for everyone. Don't be afraid to experiment - that's what testnets are for!

If you have questions or feedback, please reach out to the community or development team.

*Made with ❤️ for the PAW Community*
