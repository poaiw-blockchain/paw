# Quick Start

Get up and running with PAW in minutes.

## Single Node Setup

The fastest way to start a local PAW node for development.

### 1. Initialize Node

```bash
# Create a new node
pawd init my-node --chain-id paw-test-1 --home ./localnode

# Create a key
pawd keys add validator --keyring-backend test --home ./localnode
```

### 2. Add Genesis Account

```bash
# Get the address
ADDR=$(pawd keys show validator -a --keyring-backend test --home ./localnode)

# Add account with tokens
pawd add-genesis-account $ADDR 1000000000upaw --home ./localnode --keyring-backend test
```

### 3. Create Genesis Transaction

```bash
pawd gentx validator 700000000upaw \
  --chain-id paw-test-1 \
  --home ./localnode \
  --keyring-backend test

pawd collect-gentxs --home ./localnode
```

### 4. Start the Node

```bash
pawd start \
  --home ./localnode \
  --minimum-gas-prices 0.001upaw \
  --grpc.address 127.0.0.1:19090 \
  --api.address tcp://127.0.0.1:1318 \
  --rpc.laddr tcp://127.0.0.1:26658
```

Your node is now running! Check status:
```bash
pawd status
```

## Join Testnet

Connect to the live PAW testnet.

### 1. Initialize for Testnet

```bash
pawd init my-node --chain-id paw-testnet-1
```

### 2. Download Genesis

```bash
curl -o ~/.paw/config/genesis.json https://raw.githubusercontent.com/poaiw-blockchain/networks/main/testnet/genesis.json
```

### 3. Configure Peers

Add persistent peers to `~/.paw/config/config.toml`:

```toml
persistent_peers = "peer1@seed1.paw-testnet.com:26656,peer2@seed2.paw-testnet.com:26656"
```

Or set via command:
```bash
PEERS="peer1@seed1.paw-testnet.com:26656,peer2@seed2.paw-testnet.com:26656"
sed -i.bak -e "s/^persistent_peers *=.*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
```

### 4. Configure State Sync (Optional but Recommended)

Sync faster by using state sync:

```bash
# Get trust height and hash from a recent block
SNAP_RPC="https://rpc.paw-testnet.com:443"
LATEST_HEIGHT=$(curl -s $SNAP_RPC/block | jq -r .result.block.header.height)
TRUST_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "$SNAP_RPC/block?height=$TRUST_HEIGHT" | jq -r .result.block_id.hash)

# Configure state sync
sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true| ; \
s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$SNAP_RPC,$SNAP_RPC\"| ; \
s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$TRUST_HEIGHT| ; \
s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml
```

### 5. Start Node

```bash
pawd start --minimum-gas-prices 0.001upaw
```

Your node will sync with the testnet.

## Docker Quick Start

### Single Node

```bash
docker run -d \
  --name paw-node \
  -p 26657:26657 \
  -p 1317:1317 \
  -p 9090:9090 \
  ghcr.io/poaiw-blockchain/paw:latest
```

### Docker Compose (Multi-Validator Testnet)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  validator1:
    image: ghcr.io/poaiw-blockchain/paw:latest
    ports:
      - "26657:26657"
      - "1317:1317"
      - "9090:9090"
    volumes:
      - validator1-data:/root/.paw
    command: start --minimum-gas-prices 0.001upaw

  validator2:
    image: ghcr.io/poaiw-blockchain/paw:latest
    ports:
      - "26658:26657"
      - "1318:1317"
      - "9091:9090"
    volumes:
      - validator2-data:/root/.paw
    command: start --minimum-gas-prices 0.001upaw

volumes:
  validator1-data:
  validator2-data:
```

Start:
```bash
docker-compose up -d
```

## First Transactions

### Check Balance

```bash
pawd query bank balances $(pawd keys show validator -a --keyring-backend test)
```

### Send Tokens

```bash
# Create recipient key
pawd keys add alice --keyring-backend test

# Send tokens
pawd tx bank send validator $(pawd keys show alice -a --keyring-backend test) 1000000upaw \
  --chain-id paw-test-1 \
  --keyring-backend test \
  --yes
```

### Query Transaction

```bash
# Get transaction by hash
pawd query tx <TXHASH>
```

## DEX Quick Start

### Create a Pool

```bash
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 \
  --from validator \
  --chain-id paw-test-1 \
  --keyring-backend test \
  --yes
```

### Execute a Swap

```bash
pawd tx dex swap 1 upaw 1000000000 uusdt 1900000000 \
  --from validator \
  --chain-id paw-test-1 \
  --keyring-backend test \
  --yes
```

### Add Liquidity

```bash
pawd tx dex add-liquidity 1 100000000 200000000 \
  --from validator \
  --chain-id paw-test-1 \
  --keyring-backend test \
  --yes
```

For more DEX features, see the [DEX Integration Guide](../developers/dex-integration.md).

## Useful Commands

### Node Status
```bash
pawd status
```

### Query Account
```bash
pawd query bank balances <address>
```

### List Keys
```bash
pawd keys list --keyring-backend test
```

### View Logs
```bash
# If running with systemd
journalctl -u pawd -f

# If running in Docker
docker logs -f paw-node
```

### Stop Node
```bash
# If running in foreground: Ctrl+C

# If running with systemd
sudo systemctl stop pawd

# If running in Docker
docker stop paw-node
```

## Troubleshooting

### "connection refused"

Node is not running. Start it with `pawd start`.

### "account not found"

Account needs tokens. Request from faucet or create genesis account.

### "insufficient gas"

Add `--gas auto --gas-adjustment 1.5` to your transaction.

### Port Already in Use

Change ports in `~/.paw/config/config.toml` and `~/.paw/config/app.toml`.

## Next Steps

- [DEX Trading](../developers/dex-integration.md)
- [IBC Channels](../developers/ibc-channels.md)
- [Run a Validator](../validators/setup.md)
