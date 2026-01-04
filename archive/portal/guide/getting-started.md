# Getting Started with PAW Blockchain

Welcome to PAW Blockchain! This guide will help you get started with PAW, from setting up your development environment to making your first transaction.

## What is PAW?

PAW is a lean Layer-1 blockchain featuring:

- **Fast Consensus**: 4-second block times with instant finality
- **Built-in DEX**: Native decentralized exchange for seamless trading
- **Secure Compute**: TEE-protected API aggregation for verified tasks
- **Multi-Device Support**: Wallets for desktop, mobile, and web
- **Deflationary Economics**: Annual emission halving with fee burns

## Prerequisites

Before you begin, ensure you have:

- A computer running Windows, macOS, or Linux
- Basic understanding of blockchain concepts
- Internet connection
- 20GB of free disk space (for running a full node)

## Installation Options

### Option 1: Using Pre-built Binaries (Recommended)

Download the latest release for your operating system:

```bash
# For Linux
wget <REPO_URL>/releases/latest/download/pawd-linux-amd64
chmod +x pawd-linux-amd64
sudo mv pawd-linux-amd64 /usr/local/bin/pawd

# For macOS
wget <REPO_URL>/releases/latest/download/pawd-darwin-amd64
chmod +x pawd-darwin-amd64
sudo mv pawd-darwin-amd64 /usr/local/bin/pawd

# For Windows
# Download pawd-windows-amd64.exe from  releases
# Add to your PATH
```

Verify the installation:

```bash
pawd version
```

### Option 2: Building from Source

::: tip Requirements
- Go 1.23.1 or higher
- 
- Make
:::

```bash
# Clone the repository
 clone <REPO_URL>
cd paw

# Install dependencies
go mod download

# Build the binary
make build

# The binary will be in ./build/pawd
./build/pawd version
```

### Option 3: Using Docker

```bash
# Pull the official image
docker pull <IMAGE_REGISTRY>/<image>:latest

# Run a node
docker run -d \
  --name paw-node \
  -p 26657:26657 \
  -p 1317:1317 \
  -v ~/.paw:/root/.paw \
  <IMAGE_REGISTRY>/<image>:latest
```

## Quick Start: 5-Minute Setup

### Step 1: Initialize Your Node

```bash
# Initialize the node with a moniker (your node's name)
pawd init my-node --chain-id paw-testnet-1

# This creates a ~/.paw directory with configuration files
```

### Step 2: Configure Your Node

Download the genesis file:

```bash
# For testnet
wget -O ~/.paw/config/genesis.json \
  <RAW_REPO_CONTENT_URL>/master/networks/testnet/genesis.json

# For mainnet
wget -O ~/.paw/config/genesis.json \
  <RAW_REPO_CONTENT_URL>/master/networks/mainnet/genesis.json
```

Add seed nodes to your config:

```bash
# Edit ~/.paw/config/config.toml
# Find the seeds line and add:
seeds = "node1.paw.network:26656,node2.paw.network:26656"
```

### Step 3: Start Your Node

```bash
# Start the node
pawd start

# Or run as a background service
pawd start --log_level info > ~/paw.log 2>&1 &
```

### Step 4: Check Node Status

```bash
# Check sync status
pawd status

# View latest block
curl http://localhost:26657/status | jq '.result.sync_info'
```

## Creating Your First Wallet

### Generate a New Wallet

```bash
# Create a new wallet
pawd keys add my-wallet

# You'll see output like:
# - name: my-wallet
#   type: local
#   address: paw1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
#   pubkey: pawpub1xxxxxxxxxxxxxxxxxxxxxxxxx
#   mnemonic: ""
#
# **Important** write this mnemonic phrase in a safe place:
# word1 word2 word3 ... word24
```

::: danger IMPORTANT
Save your 24-word mnemonic phrase in a secure location. This is the ONLY way to recover your wallet if you lose access. Never share it with anyone!
:::

### Import an Existing Wallet

```bash
# Import using mnemonic
pawd keys add my-wallet --recover

# Then paste your 24-word mnemonic when prompted
```

### View Your Wallet Address

```bash
# List all wallets
pawd keys list

# Show specific wallet
pawd keys show my-wallet

# Show address only
pawd keys show my-wallet --address
```

## Getting Testnet Tokens

To start using PAW on testnet, you'll need some test tokens:

### Option 1: Web Faucet

Visit the [PAW Testnet Faucet](https://faucet.paw.network) and enter your address.

### Option 2: CLI Faucet

```bash
# Request tokens from the faucet
curl -X POST https://faucet-api.paw.network/request \
  -H "Content-Type: application/json" \
  -d '{"address":"paw1xxxxxxxxxxxxx"}'
```

### Option 3: Discord Bot

Join our [Discord server](https://discord.gg/DBHTc2QV) and use:

```
!faucet paw1xxxxxxxxxxxxx
```

## Making Your First Transaction

### Check Your Balance

```bash
# Query your balance
pawd query bank balances paw1xxxxxxxxxxxxx

# Or using the REST API
curl http://localhost:1317/cosmos/bank/v1beta1/balances/paw1xxxxxxxxxxxxx
```

### Send Tokens

```bash
# Send tokens to another address
pawd tx bank send \
  my-wallet \
  paw1yyyyyyyyyyyyyyyyyyyyyy \
  1000000upaw \
  --chain-id paw-testnet-1 \
  --fees 500upaw \
  --gas auto \
  --gas-adjustment 1.3

# Confirm the transaction when prompted
```

::: tip Token Denominations
PAW uses micro-denominations:
- 1 PAW = 1,000,000 upaw
- Always specify amounts in upaw for transactions
:::

### Check Transaction Status

```bash
# Query by transaction hash
pawd query tx <TX_HASH>

# Or using REST API
curl http://localhost:1317/cosmos/tx/v1beta1/txs/<TX_HASH>
```

## Next Steps

Congratulations! You've successfully:

✅ Installed PAW
✅ Set up a node
✅ Created a wallet
✅ Received testnet tokens
✅ Made your first transaction

### What to Do Next

- **[Create a Wallet](/guide/wallets)** - Set up desktop, mobile, or web wallet
- **[Use the DEX](/guide/dex)** - Trade tokens on the decentralized exchange
- **[Start Staking](/guide/staking)** - Earn rewards by staking PAW
- **[Participate in Governance](/guide/governance)** - Vote on proposals

### Helpful Resources

- [Developer Quick Start](/developer/quick-start) - Build applications on PAW
- [Validator Setup](/validator/setup) - Run a validator node
- [API Reference](/developer/api) - Complete API documentation
- [FAQ](/faq) - Frequently asked questions

## Troubleshooting

### Node Won't Start

If your node fails to start:

1. Check you downloaded the correct genesis file
2. Verify seeds are correctly configured
3. Ensure ports 26656 and 26657 are available
4. Check logs: `tail -f ~/paw.log`

### Node Not Syncing

If your node is stuck:

```bash
# Reset the node (WARNING: deletes all local data)
pawd unsafe-reset-all

# Re-download genesis and restart
```

### Can't See Balance

If your balance doesn't appear:

1. Wait for node to fully sync
2. Verify you're querying the correct address
3. Check the transaction was successful on explorer

## Getting Help

Need assistance? We're here to help:

- 💬 [Discord Support](https://discord.gg/DBHTc2QV)
- 📧 Email: support@pawblockchain.io
- 📖 [Full Documentation](/)

## Video Tutorial

<div class="video-container">
  <iframe
    src="https://www.youtube.com/embed/VIDEO_ID"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen>
  </iframe>
</div>

---

**Next:** [Creating a Wallet](/guide/wallets) →
