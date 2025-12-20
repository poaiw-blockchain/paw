# IBC Channels Guide

Enable cross-chain features using Inter-Blockchain Communication (IBC) on PAW.

## Overview

IBC (Inter-Blockchain Communication) enables secure cross-chain communication between independent blockchains. PAW uses IBC to:

- Transfer tokens between PAW and other Cosmos chains
- Execute cross-chain DEX swaps (PAW ↔ Osmosis, Injective)
- Subscribe to price feeds from remote oracles (Band Protocol)
- Submit compute jobs to remote networks (Akash, Celestia)
- Use interchain accounts to control accounts on other chains

## Supported Chains

PAW has IBC connections with:

| Chain | Chain ID | Use Cases |
|-------|----------|-----------|
| Cosmos Hub | cosmoshub-4 | Token transfers, Oracle feeds, ICA |
| Osmosis | osmosis-1 | DEX aggregation, Token transfers |
| Celestia | celestia | Compute jobs, Data availability |
| Injective | injective-1 | DEX aggregation, Derivatives |

## Token Transfers

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
  --packet-timeout-timestamp $(($(date +%s + 600) * 1000000000)) \
  --yes
```

This sets a 10-minute timeout.

**Via TypeScript SDK:**

```typescript
import { PawClient } from '@poaiw-blockchain/paw-sdk';

const result = await client.ibc.transfer({
  sourcePort: 'transfer',
  sourceChannel: 'channel-0',
  token: { denom: 'upaw', amount: '1000000' },
  sender: 'paw1abc123...',
  receiver: 'osmo1xyz456...',
  timeoutTimestamp: Date.now() + 600000, // 10 minutes
  signer: wallet,
});
```

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

**Via REST API:**

```bash
curl "http://localhost:1317/ibc/apps/transfer/v1/denom_traces/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
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
  --chain-id paw-testnet-1 \
  --yes
```

**Route format:** `chainID:poolID:tokenIn:tokenOut,chainID:poolID:tokenIn:tokenOut`

### Simulate Cross-Chain Swap

```bash
pawd query dex simulate-cross-chain-swap \
  --route "paw-testnet-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo" \
  --amount-in 1000000upaw
```

**Via Python SDK:**

```python
from paw_sdk import PawClient

client = PawClient(rpc_endpoint='http://localhost:26657')

# Simulate cross-chain swap
simulation = client.dex.simulate_cross_chain_swap(
    route='paw-testnet-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo',
    amount_in='1000000',
)

# Execute if profitable
if simulation['expected_return'] > int(simulation['amount_in']) * 1.01:
    result = client.dex.cross_chain_swap(
        route='paw-testnet-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo',
        amount_in='1000000',
        min_out='950000',
        max_slippage=0.05,
        signer=wallet,
    )
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
  --chain-id paw-testnet-1 \
  --yes
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
  --chain-id paw-testnet-1 \
  --yes
```

## Interchain Accounts (ICA)

### Register Interchain Account

Create an account you control on Cosmos Hub:

```bash
pawd tx interchain-accounts controller register \
  connection-1 \
  --from alice \
  --chain-id paw-testnet-1 \
  --yes
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
  --chain-id paw-testnet-1 \
  --yes
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

Or download pre-built binary from [Hermes releases](https://github.com/informalsystems/hermes/releases).

### Configure Relayer

Create `~/.hermes/config.toml`:

```toml
[global]
log_level = 'info'

[[chains]]
id = 'paw-testnet-1'
rpc_addr = 'http://localhost:26657'
grpc_addr = 'http://localhost:9090'
event_source = { mode = 'push', url = 'ws://localhost:26657/websocket', batch_delay = '500ms' }
account_prefix = 'paw'
key_name = 'relayer'
store_prefix = 'ibc'
gas_price = { price = 0.001, denom = 'upaw' }
max_gas = 3000000

[[chains]]
id = 'osmosis-1'
rpc_addr = 'https://rpc.osmosis.zone:443'
grpc_addr = 'https://grpc.osmosis.zone:443'
event_source = { mode = 'push', url = 'wss://rpc.osmosis.zone:443/websocket', batch_delay = '500ms' }
account_prefix = 'osmo'
key_name = 'relayer'
store_prefix = 'ibc'
gas_price = { price = 0.0025, denom = 'uosmo' }
max_gas = 3000000
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
hermes start
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
hermes --log-level debug start
```

## Building IBC Applications

### Example: Cross-Chain Transfer App

```typescript
import { PawClient } from '@poaiw-blockchain/paw-sdk';

class CrossChainTransferApp {
  private client: PawClient;

  async transferToOsmosis(amount: string, recipient: string) {
    // Transfer tokens to Osmosis
    const result = await this.client.ibc.transfer({
      sourcePort: 'transfer',
      sourceChannel: 'channel-0',
      token: { denom: 'upaw', amount: amount },
      sender: this.wallet.address,
      receiver: recipient,
      timeoutTimestamp: Date.now() + 600000,
      signer: this.wallet,
    });

    // Wait for acknowledgement
    const ack = await this.waitForAcknowledgement(result.txHash);

    if (ack.success) {
      console.log('Transfer successful!');
    } else {
      console.error('Transfer failed:', ack.error);
    }
  }

  async waitForAcknowledgement(txHash: string): Promise<any> {
    // Poll for transaction status
    for (let i = 0; i < 60; i++) {
      const tx = await this.client.getTx(txHash);
      if (tx.code === 0 && tx.events.some(e => e.type === 'acknowledge_packet')) {
        return { success: true };
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    return { success: false, error: 'Timeout waiting for acknowledgement' };
  }
}
```

### Example: Cross-Chain Price Oracle

```python
from paw_sdk import PawClient
import time

class CrossChainOracle:
    def __init__(self, rpc_endpoint):
        self.client = PawClient(rpc_endpoint=rpc_endpoint)

    def subscribe_band_protocol(self, symbols):
        """Subscribe to Band Protocol price feeds"""
        self.client.oracle.subscribe_prices(
            symbols=symbols,
            sources=['band-laozi-testnet4'],
            interval=60,
            signer=self.wallet,
        )

    def get_aggregated_price(self, symbol):
        """Get aggregated price from multiple sources"""
        return self.client.oracle.get_aggregated_price(symbol)

    def monitor_prices(self, symbols):
        """Monitor prices in real-time"""
        while True:
            for symbol in symbols:
                price = self.get_aggregated_price(symbol)
                print(f'{symbol}: ${price["price"]} (sources: {price["source_count"]})')
            time.sleep(10)

# Usage
oracle = CrossChainOracle('http://localhost:26657')
oracle.subscribe_band_protocol(['BTC/USD', 'ETH/USD'])
oracle.monitor_prices(['BTC/USD', 'ETH/USD'])
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

- Verify channel IDs before transfers
- Use trusted relayers or run your own
- Never send to unknown addresses
- Start with testnet before mainnet
- Keep relayer keys secure (use OS keychain)
- Monitor relayer balance for fees

## Troubleshooting

### Packet Timeout

**Solution:**
```bash
# Increase timeout to 10 minutes
TIMEOUT=$(($(date +%s + 600) * 1000000000))
pawd tx ibc-transfer transfer transfer channel-0 \
  osmo1... 1000000upaw --from alice \
  --packet-timeout-timestamp $TIMEOUT \
  --yes
```

### Relayer Not Running

**Check:**
```bash
hermes health-check
hermes query packet pending --chain paw-testnet-1 --port transfer --channel channel-0
```

**Fix:**
```bash
hermes start
```

### Wrong IBC Denom

This is normal. Query denom trace to see original:
```bash
pawd query ibc-transfer denom-trace <hash>
```

### Channel Not Found

**Create channel:**
```bash
hermes create channel --a-chain paw-testnet-1 \
  --a-connection connection-0 --a-port transfer --b-port transfer
```

### Light Client Expired

**Update:**
```bash
hermes update client --host-chain paw-testnet-1 --client 07-tendermint-0
```

Or configure automatic refresh in `config.toml`:
```toml
[mode.clients]
enabled = true
refresh = true
```

## Next Steps

- [DEX Integration Guide](dex-integration.md)
- [Hermes Documentation](https://hermes.informal.systems/)
- [IBC Protocol Specification](https://ibcprotocol.org/)
- [Example IBC App](https://github.com/poaiw-blockchain/examples/tree/main/ibc-transfer)
