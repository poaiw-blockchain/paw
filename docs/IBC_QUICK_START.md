# PAW IBC Quick Start Guide

Get started with Inter-Blockchain Communication (IBC) on PAW to transfer tokens and interact with other Cosmos chains.

## What is IBC?

IBC (Inter-Blockchain Communication) is a protocol that enables secure cross-chain communication between independent blockchains. PAW uses IBC to:

- Transfer tokens between PAW and other Cosmos chains
- Execute cross-chain DEX swaps (PAW ↔ Osmosis, Injective)
- Subscribe to price feeds from remote oracles (Band Protocol)
- Submit compute jobs to remote networks (Akash, Celestia)
- Use interchain accounts to control accounts on other chains

## Prerequisites

- PAW daemon installed (`pawd`)
- Wallet with PAW tokens
- IBC relayer running (Hermes) or access to public relayer
- Target chain connection established

## Supported Chains

PAW has IBC connections with:

| Chain | Chain ID | Use Cases |
|-------|----------|-----------|
| Cosmos Hub | cosmoshub-4 | Token transfers, Oracle feeds, ICA |
| Osmosis | osmosis-1 | DEX aggregation, Token transfers |
| Celestia | celestia | Compute jobs, Data availability |
| Injective | injective-1 | DEX aggregation, Derivatives |

## Basic Token Transfers

### Check IBC Channels

```bash
# List all IBC channels
pawd query ibc channel channels

# Check specific channel
pawd query ibc channel end transfer channel-0
```

### Send Tokens to Another Chain

Transfer PAW tokens to Osmosis:

```bash
pawd tx ibc-transfer transfer \
  transfer \
  channel-0 \
  osmo1recipientaddress... \
  1000000upaw \
  --from alice \
  --chain-id paw-testnet-1 \
  --packet-timeout-height 0-0 \
  --packet-timeout-timestamp 0
```

**With timeout:**

```bash
pawd tx ibc-transfer transfer \
  transfer \
  channel-0 \
  osmo1recipientaddress... \
  1000000upaw \
  --from alice \
  --packet-timeout-timestamp $(($(date +%s + 600) * 1000000000))
```

This sets a 10-minute timeout.

### Receive Tokens from Another Chain

Tokens automatically appear in your balance when received via IBC:

```bash
# Check for IBC tokens (have "ibc/" prefix)
pawd query bank balances $(pawd keys show alice -a)

# Example output:
# - amount: "1000000"
#   denom: ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2
```

### Query IBC Token Denom Trace

Find the original token information:

```bash
# Get hash from balance (everything after "ibc/")
pawd query ibc-transfer denom-trace 27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2

# Returns: path: transfer/channel-0, base_denom: uosmo
```

## Cross-Chain DEX Trading

### Query Remote Pool Liquidity

Check pools on Osmosis:

```bash
pawd query dex cross-chain-pools upaw uosmo osmosis-1
```

### Execute Cross-Chain Swap

Swap PAW → ATOM → OSMO across multiple chains:

```bash
pawd tx dex cross-chain-swap \
  --route "paw-testnet-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo" \
  --amount-in 1000000upaw \
  --min-out 950000uosmo \
  --max-slippage 0.05 \
  --from alice \
  --chain-id paw-testnet-1
```

**Route format:** `chainID:poolID:tokenIn:tokenOut,chainID:poolID:tokenIn:tokenOut`

### Simulate Cross-Chain Swap

```bash
pawd query dex simulate-cross-chain-swap \
  --route "paw-testnet-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo" \
  --amount-in 1000000upaw
```

## Oracle Price Feeds

### Subscribe to Remote Oracle

Subscribe to Band Protocol price feeds:

```bash
pawd tx oracle subscribe-prices \
  --symbols "BTC/USD,ETH/USD,ATOM/USD" \
  --sources "band-laozi-testnet4" \
  --interval 60 \
  --from alice \
  --chain-id paw-testnet-1
```

Updates every 60 seconds.

### Query Cross-Chain Prices

```bash
# Query aggregated price from multiple oracles
pawd query oracle aggregated-price BTC/USD

# Query specific oracle source
pawd query oracle price BTC/USD --source band-laozi-testnet4
```

### Register Oracle Source

```bash
pawd tx oracle register-source \
  band-laozi-testnet4 \
  band \
  connection-1 \
  channel-3 \
  --from validator \
  --chain-id paw-testnet-1
```

## Compute Jobs

### Discover Remote Compute Providers

Find providers on Akash:

```bash
pawd query compute discover-providers \
  --chains akashnet-2 \
  --capabilities gpu,tee \
  --max-price 10
```

### Submit Cross-Chain Compute Job

```bash
pawd tx compute submit-job \
  --job-type wasm \
  --job-data @job.wasm \
  --target-chain akashnet-2 \
  --provider provider-123 \
  --cpu 4 \
  --memory 8192 \
  --gpu \
  --tee \
  --payment 1000000upaw \
  --from requester \
  --chain-id paw-testnet-1
```

### Query Job Status

```bash
pawd query compute job-status job-123
```

## Interchain Accounts (ICA)

### Register Interchain Account

Create an account you control on Cosmos Hub:

```bash
pawd tx interchain-accounts controller register \
  connection-1 \
  --from alice \
  --chain-id paw-testnet-1
```

### Query ICA Address

```bash
pawd query interchain-accounts controller interchain-account \
  $(pawd keys show alice -a) \
  connection-1
```

### Execute Remote Transaction

Stake tokens on Cosmos Hub using your ICA:

```bash
pawd tx interchain-accounts controller send-tx \
  connection-1 \
  @msg.json \
  --from alice \
  --chain-id paw-testnet-1
```

**msg.json:**
```json
{
  "@type": "/cosmos.staking.v1beta1.MsgDelegate",
  "delegator_address": "cosmos1...",
  "validator_address": "cosmosvaloper1...",
  "amount": {
    "denom": "uatom",
    "amount": "1000000"
  }
}
```

## Setting Up IBC Channels

### Check Existing Connections

```bash
pawd query ibc connection connections
pawd query ibc connection end connection-0
```

### Create New Channel

Using Hermes relayer:

```bash
# Create channel for token transfers
hermes create channel \
  --a-chain paw-testnet-1 \
  --a-connection connection-0 \
  --a-port transfer \
  --b-port transfer

# Create channel for DEX
hermes create channel \
  --a-chain paw-testnet-1 \
  --a-connection connection-0 \
  --a-port dex \
  --b-port dex
```

### Verify Channel Creation

```bash
pawd query ibc channel channels
```

## Relayer Setup

### Install Hermes Relayer

```bash
cargo install ibc-relayer-cli --bin hermes
```

### Configure Relayer

Edit `/home/hudson/blockchain-projects/paw/ibc/relayer-config.yaml`:

```yaml
[[chains]]
id = 'paw-testnet-1'
rpc_addr = 'http://localhost:26657'
grpc_addr = 'http://localhost:9090'
account_prefix = 'paw'
key_name = 'relayer'
```

### Add Relayer Keys

```bash
# Create relayer key
pawd keys add relayer --keyring-backend test

# Export for Hermes
pawd keys export relayer --unarmored-hex --unsafe > paw-relayer.key

# Add to Hermes
hermes keys add \
  --chain paw-testnet-1 \
  --key-file paw-relayer.key
```

### Start Relayer

```bash
hermes --config /home/hudson/blockchain-projects/paw/ibc/relayer-config.yaml start
```

## Monitoring IBC

### Check Relayer Health

```bash
hermes health-check
```

### Query Pending Packets

```bash
hermes query packet pending \
  --chain paw-testnet-1 \
  --port transfer \
  --channel channel-0
```

### Clear Pending Packets

```bash
hermes clear packets \
  --chain paw-testnet-1 \
  --port transfer \
  --channel channel-0
```

### Monitor Relayer Logs

```bash
hermes --config relayer-config.yaml --log-level debug start
```

## Complete Workflows

### Workflow 1: Transfer Tokens to Osmosis and Swap

```bash
# Step 1: Send PAW to Osmosis
pawd tx ibc-transfer transfer transfer channel-0 \
  osmo1youraddress... 1000000upaw --from alice

# Step 2: Wait for transfer (check Osmosis balance)
# On Osmosis side:
osmosisd query bank balances osmo1youraddress...

# Step 3: Swap on Osmosis
osmosisd tx gamm swap-exact-amount-in 1 1000000ibc/... \
  uosmo 950000 --from alice
```

### Workflow 2: Cross-Chain DEX Swap (One Transaction)

```bash
# Execute multi-hop swap on PAW (relayer handles IBC)
pawd tx dex cross-chain-swap \
  --route "paw-testnet-1:1:upaw:uatom,osmosis-1:42:uatom:uosmo" \
  --amount-in 1000000upaw \
  --min-out 950000uosmo \
  --from alice
```

### Workflow 3: Subscribe to Oracle and Use Prices

```bash
# Step 1: Register oracle source
pawd tx oracle register-source band-laozi-testnet4 band connection-1 channel-3 --from validator

# Step 2: Subscribe to prices
pawd tx oracle subscribe-prices --symbols "BTC/USD,ETH/USD" \
  --sources "band-laozi-testnet4" --interval 60 --from alice

# Step 3: Query prices (available after first update)
pawd query oracle aggregated-price BTC/USD

# Step 4: Use in your application
pawd tx compute submit-job --oracle-price BTC/USD ...
```

## Troubleshooting

### Packet Timeout

**Issue:** Transfer didn't complete within timeout period.

**Solution:**
```bash
# Increase timeout to 10 minutes
TIMEOUT=$(($(date +%s + 600) * 1000000000))
pawd tx ibc-transfer transfer transfer channel-0 \
  osmo1... 1000000upaw --from alice \
  --packet-timeout-timestamp $TIMEOUT
```

### Relayer Not Running

**Issue:** Transactions stuck, no acknowledgements.

**Check:**
```bash
hermes health-check
hermes query packet pending --chain paw-testnet-1 --port transfer --channel channel-0
```

**Fix:**
```bash
# Restart relayer
hermes --config relayer-config.yaml start
```

### Wrong IBC Denom

**Issue:** Token shows as "ibc/HASH" instead of readable name.

**This is normal.** Query denom trace to see original:
```bash
pawd query ibc-transfer denom-trace <hash>
```

### Channel Not Found

**Issue:** Channel doesn't exist between chains.

**Create channel:**
```bash
hermes create channel --a-chain paw-testnet-1 \
  --a-connection connection-0 --a-port transfer --b-port transfer
```

### Light Client Expired

**Issue:** IBC client needs updating.

**Update:**
```bash
hermes update client --host-chain paw-testnet-1 --client 07-tendermint-0
```

Or configure automatic refresh in `relayer-config.yaml`:
```yaml
[mode.clients]
enabled = true
refresh = true
```

## Best Practices

### For Token Transfers

1. **Set reasonable timeouts** (5-10 minutes for most chains)
2. **Use small test amounts** first
3. **Verify recipient address** matches destination chain's format
4. **Check relayer is active** before large transfers
5. **Monitor transaction** until acknowledgement received

### For Cross-Chain Swaps

1. **Simulate swaps** before execution
2. **Set slippage limits** (1-5% for cross-chain)
3. **Check liquidity** on both chains
4. **Use deadline parameters** to prevent stale execution
5. **Monitor both chains** during multi-hop swaps

### Security

- **Verify channel IDs** before transfers
- **Use trusted relayers** or run your own
- **Never send to unknown addresses**
- **Start with testnet** before mainnet
- **Keep relayer keys secure** (use OS keychain)
- **Monitor relayer balance** for fees

## Advanced Topics

### IBC Fee Payment

Pay relayers to prioritize your packets:

```bash
pawd tx ibc-fee pay-packet-fee \
  transfer \
  channel-0 \
  --recv-fee 1000upaw \
  --ack-fee 500upaw \
  --timeout-fee 500upaw \
  --from alice
```

### Multi-Hop Transfers

Route through intermediary chains:

```bash
# PAW → Cosmos Hub → Osmosis
pawd tx ibc-transfer transfer transfer channel-2 \
  cosmos1... 1000000upaw --from alice \
  --memo '{"forward":{"receiver":"osmo1...","port":"transfer","channel":"channel-141"}}'
```

### Query IBC Client State

```bash
pawd query ibc client state 07-tendermint-0
```

### Query Connection State

```bash
pawd query ibc connection end connection-0
```

## Next Steps

- **Cross-chain DEX:** See [DEX_QUICK_START.md](DEX_QUICK_START.md)
- **Full IBC implementation:** [implementation/ibc/IBC_IMPLEMENTATION.md](implementation/ibc/IBC_IMPLEMENTATION.md)
- **Relayer security:** `/ibc/RELAYER_SECURITY.md`
- **IBC protocol:** https://ibcprotocol.org/

## Getting Help

- Discord: https://discord.gg/paw-chain
- Documentation: https://docs.paw-chain.com/ibc
- IBC Protocol Docs: https://ibcprotocol.org/
- Hermes Docs: https://hermes.informal.systems/
