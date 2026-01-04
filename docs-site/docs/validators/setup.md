# Validator Setup

Run a validator node on the PAW blockchain network.

## Overview

Validators are responsible for:
- Proposing and validating blocks
- Participating in consensus
- Securing the network through staked tokens
- Earning staking rewards and transaction fees

## Requirements

### Hardware Requirements

**Minimum:**
- CPU: 4 cores
- RAM: 8 GB
- Storage: 500 GB SSD
- Network: 100 Mbps

**Recommended:**
- CPU: 8+ cores (Intel Xeon, AMD EPYC)
- RAM: 32 GB
- Storage: 1 TB NVMe SSD
- Network: 1 Gbps dedicated

### Software Requirements

- Ubuntu 22.04 LTS (recommended) or similar Linux distribution
- Go 1.23+
- Git
- Make
- Firewall (ufw or iptables)

## Installation

### 1. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y build-essential git curl jq wget

# Install Go
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 2. Build PAW Binary

```bash
# Clone repository
git clone https://github.com/poaiw-blockchain/paw.git
cd paw

# Build binary
make build

# Install to system
sudo cp build/pawd /usr/local/bin/

# Verify installation
pawd version
```

### 3. Initialize Node

```bash
# Initialize node
pawd init <your-moniker> --chain-id paw-1

# Download genesis file
curl -o ~/.paw/config/genesis.json https://raw.githubusercontent.com/poaiw-blockchain/networks/main/mainnet/genesis.json

# Verify genesis
pawd validate-genesis
```

### 4. Configure Node

Edit `~/.paw/config/config.toml`:

```toml
# Set minimum gas prices
minimum-gas-prices = "0.001upaw"

# Set peers
persistent_peers = "peer1@node1.paw.com:26656,peer2@node2.paw.com:26656"

# Enable prometheus metrics
prometheus = true
prometheus_listen_addr = ":26660"

# Set pruning (keep last 100,000 blocks)
pruning = "custom"
pruning-keep-recent = "100000"
pruning-interval = "10"
```

Edit `~/.paw/config/app.toml`:

```toml
# Minimum gas prices
minimum-gas-prices = "0.001upaw"

# Enable API
[api]
enable = true
swagger = true
address = "tcp://0.0.0.0:1317"

# Enable gRPC
[grpc]
enable = true
address = "0.0.0.0:9090"
```

### 5. Set Up State Sync (Recommended)

State sync allows your node to sync quickly:

```bash
SNAP_RPC="https://rpc.paw.com:443"

LATEST_HEIGHT=$(curl -s $SNAP_RPC/block | jq -r .result.block.header.height)
TRUST_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "$SNAP_RPC/block?height=$TRUST_HEIGHT" | jq -r .result.block_id.hash)

sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true| ; \
s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$SNAP_RPC,$SNAP_RPC\"| ; \
s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$TRUST_HEIGHT| ; \
s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml
```

### 6. Create Systemd Service

```bash
sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Blockchain Node
After=network-online.target

[Service]
User=$USER
ExecStart=/usr/local/bin/pawd start --home $HOME/.paw
Restart=on-failure
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable pawd
sudo systemctl start pawd
```

### 7. Verify Node is Syncing

```bash
# Check status
pawd status

# View logs
journalctl -u pawd -f

# Check sync status
pawd status | jq .SyncInfo
```

Wait for your node to fully sync before creating a validator.

## Create Validator

### 1. Create Validator Key

```bash
# Create new key
pawd keys add validator

# Or recover from mnemonic
pawd keys add validator --recover
```

**Save your mnemonic in a secure location!**

### 2. Fund Validator Account

You need PAW tokens to create a validator. Acquire tokens through:
- Token sale or airdrop
- Transfer from another account
- Purchase from exchange

Verify balance:
```bash
pawd query bank balances $(pawd keys show validator -a)
```

### 3. Create Validator Transaction

```bash
pawd tx staking create-validator \
  --amount=1000000upaw \
  --pubkey=$(pawd tendermint show-validator) \
  --moniker="<your-moniker>" \
  --chain-id=paw-1 \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --gas="auto" \
  --gas-adjustment="1.5" \
  --gas-prices="0.001upaw" \
  --from=validator
```

Parameters:
- `amount`: Initial self-delegation (1 PAW = 1,000,000 upaw)
- `commission-rate`: Your commission rate (10% = 0.10)
- `commission-max-rate`: Maximum commission rate
- `commission-max-change-rate`: Max daily commission change
- `min-self-delegation`: Minimum self-delegation

### 4. Verify Validator is Active

```bash
# Query your validator
pawd query staking validator $(pawd keys show validator --bech val -a)

# Check if in active set
pawd query tendermint-validator-set | grep $(pawd tendermint show-validator)
```

## Security Best Practices

### Firewall Configuration

```bash
# Enable firewall
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow P2P
sudo ufw allow 26656/tcp

# Allow RPC (only if public node)
sudo ufw allow 26657/tcp

# Allow API (only if public node)
sudo ufw allow 1317/tcp

# Check status
sudo ufw status
```

### Key Management

1. **Never store keys on the validator node in production**
2. **Use a hardware wallet** (Ledger) for signing
3. **Enable Key Management System (KMS)** for production validators
4. **Backup your mnemonic** in multiple secure locations
5. **Use strong passwords** for encrypted keys

### Sentry Node Architecture

Protect your validator behind sentry nodes:

```
Internet → Sentry Node 1 → Validator (private)
Internet → Sentry Node 2 →
```

Configure validator to only connect to sentries:
```toml
# In validator's config.toml
persistent_peers = "sentry1-id@10.0.0.1:26656,sentry2-id@10.0.0.2:26656"
pex = false
addr_book_strict = false
```

Configure sentries to expose public IP:
```toml
# In sentry's config.toml
private_peer_ids = "validator-node-id"
pex = true
```

### Monitoring and Alerts

Set up monitoring to track:
- Node uptime
- Block height sync status
- Validator signing rate
- Disk space
- CPU/RAM usage

See [Monitoring Guide](monitoring.md) for details.

## Validator Operations

### Check Validator Status

```bash
# Get validator info
pawd query staking validator $(pawd keys show validator --bech val -a)

# Check voting power
pawd status | jq .ValidatorInfo.VotingPower

# Check if jailed
pawd query staking validator $(pawd keys show validator --bech val -a) | grep jailed
```

### Edit Validator

```bash
pawd tx staking edit-validator \
  --moniker="New Moniker" \
  --website="https://validator.com" \
  --identity="<keybase-id>" \
  --details="Description of validator" \
  --commission-rate="0.05" \
  --from=validator \
  --chain-id=paw-1
```

### Unjail Validator

If your validator is jailed for downtime:

```bash
pawd tx slashing unjail \
  --from=validator \
  --chain-id=paw-1 \
  --gas=auto
```

### Delegate More Tokens

```bash
pawd tx staking delegate \
  $(pawd keys show validator --bech val -a) \
  1000000upaw \
  --from=validator \
  --chain-id=paw-1
```

## Backup and Recovery

### Backup Validator Key

```bash
# Backup priv_validator_key.json
cp ~/.paw/config/priv_validator_key.json ~/validator_key_backup.json

# Backup node key
cp ~/.paw/config/node_key.json ~/node_key_backup.json
```

Store backups securely offline.

### Restore Validator

```bash
# Stop node
sudo systemctl stop pawd

# Restore keys
cp ~/validator_key_backup.json ~/.paw/config/priv_validator_key.json
cp ~/node_key_backup.json ~/.paw/config/node_key.json

# Start node
sudo systemctl start pawd
```

## Upgrading

### Binary Upgrade

```bash
# Stop node
sudo systemctl stop pawd

# Backup current binary
sudo cp /usr/local/bin/pawd /usr/local/bin/pawd.backup

# Build new version
cd ~/paw
git fetch --all
git checkout v2.0.0
make build
sudo cp build/pawd /usr/local/bin/

# Start node
sudo systemctl start pawd
```

### Using Cosmovisor (Recommended)

Cosmovisor enables automatic binary upgrades:

```bash
# Install cosmovisor
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

# Set up directory structure
mkdir -p ~/.paw/cosmovisor/genesis/bin
mkdir -p ~/.paw/cosmovisor/upgrades

# Copy current binary
cp /usr/local/bin/pawd ~/.paw/cosmovisor/genesis/bin/

# Update systemd service
sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Blockchain Node with Cosmovisor
After=network-online.target

[Service]
User=$USER
Environment="DAEMON_HOME=$HOME/.paw"
Environment="DAEMON_NAME=pawd"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
ExecStart=$HOME/go/bin/cosmovisor run start
Restart=on-failure
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF

# Restart service
sudo systemctl daemon-reload
sudo systemctl restart pawd
```

## Troubleshooting

### Node Not Syncing

Check peers:
```bash
pawd query tendermint-validator-set
curl localhost:26657/net_info
```

Add more peers if needed.

### Validator Not Signing

Check if node is synced:
```bash
pawd status | jq .SyncInfo.catching_up
```

Check logs for errors:
```bash
journalctl -u pawd -f | grep -i error
```

### High Memory Usage

Enable pruning in `app.toml`:
```toml
pruning = "custom"
pruning-keep-recent = "100000"
pruning-interval = "10"
```

### Disk Space Full

Check disk usage:
```bash
df -h
du -sh ~/.paw/data
```

Consider increasing disk size or enabling more aggressive pruning.

## Next Steps

- [Monitoring Guide](monitoring.md) - Set up monitoring and alerts
- [Validator Best Practices](https://docs.cosmos.network/main/validators/validator-faq)
- [Join Discord](https://discord.gg/DBHTc2QV) - Connect with other validators
