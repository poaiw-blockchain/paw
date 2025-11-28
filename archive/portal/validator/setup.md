# Validator Setup Guide

Complete guide to setting up a PAW validator node.

## Requirements

### Hardware

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 4 cores | 8 cores |
| RAM | 16GB | 32GB |
| Storage | 500GB SSD | 1TB NVMe SSD |
| Network | 100Mbps | 1Gbps |

### Software

- Ubuntu 22.04 LTS (recommended)
- Go 1.23.1+
- 
- Make

## Installation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install build-essential  curl wget jq -y

# Install Go
wget https://go.dev/dl/go1.23.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.1.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Clone PAW repository
 clone <REPO_URL>
cd paw
make install

# Verify installation
pawd version
```

## Initialize Node

```bash
# Set variables
MONIKER="my-validator"
CHAIN_ID="paw-mainnet-1"

# Initialize node
pawd init $MONIKER --chain-id $CHAIN_ID

# Download genesis
wget -O ~/.paw/config/genesis.json \
  <RAW_REPO_CONTENT_URL>/master/networks/mainnet/genesis.json

# Verify genesis
pawd validate-genesis

# Set seeds and peers
SEEDS="node1.paw.network:26656,node2.paw.network:26656"
PEERS="peer1.paw.network:26656,peer2.paw.network:26656"

sed -i "s/^seeds =.*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml
sed -i "s/^persistent_peers =.*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
```

## Configure Node

```bash
# Set minimum gas prices
sed -i 's/minimum-gas-prices =.*/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml

# Enable Prometheus metrics
sed -i 's/prometheus = false/prometheus = true/' ~/.paw/config/config.toml

# Set pruning (optional)
sed -i 's/pruning = "default"/pruning = "custom"/' ~/.paw/config/app.toml
sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100"/' ~/.paw/config/app.toml
sed -i 's/pruning-interval = "0"/pruning-interval = "10"/' ~/.paw/config/app.toml
```

## Create Systemd Service

```bash
# Create service file
sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Blockchain Node
After=network-online.target

[Service]
User=$USER
ExecStart=$(which pawd) start --home $HOME/.paw
Restart=on-failure
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable pawd
sudo systemctl start pawd

# Check status
sudo systemctl status pawd

# View logs
sudo journalctl -u pawd -f
```

## Sync Node

```bash
# Check sync status
pawd status | jq '.SyncInfo'

# State sync (faster)
SNAP_RPC="https://rpc.paw.network:443"
LATEST_HEIGHT=$(curl -s $SNAP_RPC/block | jq -r .result.block.header.height)
BLOCK_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "$SNAP_RPC/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

sed -i "s/^enable =.*/enable = true/" ~/.paw/config/config.toml
sed -i "s/^rpc_servers =.*/rpc_servers = \"$SNAP_RPC,$SNAP_RPC\"/" ~/.paw/config/config.toml
sed -i "s/^trust_height =.*/trust_height = $BLOCK_HEIGHT/" ~/.paw/config/config.toml
sed -i "s/^trust_hash =.*/trust_hash = \"$TRUST_HASH\"/" ~/.paw/config/config.toml

sudo systemctl restart pawd
```

## Create Validator

```bash
# Create validator key
pawd keys add validator

# Fund validator address (get tokens from faucet or exchange)
# Minimum: 1000 PAW + fees

# Create validator
pawd tx staking create-validator \
  --amount=1000000000upaw \
  --pubkey=$(pawd tendermint show-validator) \
  --moniker="$MONIKER" \
  --chain-id=$CHAIN_ID \
  --commission-rate="0.05" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1000000" \
  --gas="auto" \
  --gas-adjustment=1.3 \
  --fees="500upaw" \
  --from=validator

# Verify validator
pawd query staking validator $(pawd keys show validator --bech val -a)
```

## Backup Keys

```bash
# Backup validator key
cat ~/.paw/config/priv_validator_key.json

# Store securely offline!

# Backup wallet mnemonic
pawd keys export validator

# Also store securely offline!
```

::: danger CRITICAL
Never lose your `priv_validator_key.json` or wallet mnemonic. Store multiple encrypted backups in secure locations.
:::

## Security Hardening

```bash
# Configure firewall
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp  # SSH
sudo ufw allow 26656/tcp  # P2P
sudo ufw allow 26657/tcp  # RPC (if public)
sudo ufw enable

# Disable RPC if not needed
sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = ""/' ~/.paw/config/config.toml

# Use Sentry Nodes (recommended)
# See: /validator/security
```

## Monitoring

```bash
# Install Prometheus & Grafana
# See: /validator/monitoring

# Check validator status
pawd status

# Check signing info
pawd query slashing signing-info $(pawd tendermint show-validator)

# Monitor via dashboard
https://dashboard.paw.network
```

## Next Steps

- **[Operations](/validator/operations)** - Daily validator operations
- **[Security](/validator/security)** - Harden your validator
- **[Monitoring](/validator/monitoring)** - Set up monitoring
- **[Troubleshooting](/validator/troubleshooting)** - Fix common issues

---

**Next:** [Validator Operations](/validator/operations) →
