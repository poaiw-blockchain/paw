# PAW Validator Quick Start

One-page reference for experienced Cosmos validator operators.

---

## Prerequisites

- Ubuntu 22.04 LTS, 8 cores, 32 GB RAM, 500 GB NVMe SSD
- Go 1.23+, Git, Make, jq
- Static IP, ports 26656 (P2P) open

---

## Installation (5 minutes)

```bash
# Install Go 1.23.5
wget https://go.dev/dl/go1.23.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Build pawd
git clone https://github.com/decristofaroj/paw.git && cd paw
git checkout $(git describe --tags `git rev-list --tags --max-count=1`)
make build && sudo install -m 0755 build/pawd /usr/local/bin/pawd

# Initialize
pawd init <moniker> --chain-id paw-testnet-1

# Download genesis and peers
curl -L https://raw.githubusercontent.com/decristofaroj/paw/main/networks/paw-testnet-1/genesis.json > ~/.paw/config/genesis.json
curl -L https://raw.githubusercontent.com/decristofaroj/paw/main/networks/paw-testnet-1/genesis.sha256 > /tmp/genesis.sha256
cd ~/.paw/config && sha256sum -c /tmp/genesis.sha256

# Configure peers
PEERS=$(curl -s https://raw.githubusercontent.com/decristofaroj/paw/main/networks/paw-testnet-1/peers.txt | tr '\n' ',' | sed 's/,$//')
sed -i "s/^persistent_peers *=.*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml

# Set minimum gas price
sed -i 's/^minimum-gas-prices *=.*/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml

# Enable Prometheus
sed -i 's/^prometheus *=.*/prometheus = true/' ~/.paw/config/config.toml

# Pruning (testnet)
sed -i 's/^pruning *=.*/pruning = "custom"/' ~/.paw/config/app.toml
sed -i 's/^pruning-keep-recent *=.*/pruning-keep-recent = "100000"/' ~/.paw/config/app.toml
sed -i 's/^pruning-interval *=.*/pruning-interval = "10"/' ~/.paw/config/app.toml
```

---

## Key Generation

```bash
# Create operator key (software for testnet, Ledger for mainnet)
pawd keys add validator-operator --keyring-backend os

# Backup mnemonic NOW (paper, fireproof safe)

# Get addresses
OPERATOR=$(pawd keys show validator-operator -a --keyring-backend os)
VALOPER=$(pawd keys show validator-operator --bech val -a --keyring-backend os)
CONSPUB=$(pawd tendermint show-validator)

echo "Operator: $OPERATOR"
echo "Validator: $VALOPER"
echo "ConsPubkey: $CONSPUB"
```

---

## Systemd Service

```bash
sudo useradd -m -s /bin/bash validator
sudo chown -R validator:validator ~/.paw

sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Validator
After=network-online.target

[Service]
User=validator
ExecStart=/usr/local/bin/pawd start --home /home/validator/.paw --minimum-gas-prices 0.001upaw
Restart=on-failure
RestartSec=10
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable pawd
sudo systemctl start pawd
```

---

## Get Testnet Tokens

```
Faucet: https://faucet.paw-testnet.io
Amount: 10,000,000 upaw (10 PAW)
```

Or Discord: `!faucet <address>` in #testnet-faucet

---

## Create Validator

Wait for sync: `pawd status | jq '.SyncInfo.catching_up'` → `false`

```bash
pawd tx staking create-validator \
  --from validator-operator \
  --amount 1000000upaw \
  --pubkey "$(pawd tendermint show-validator)" \
  --moniker "<moniker>" \
  --identity "<keybase-16-char-id>" \
  --website "https://yoursite.com" \
  --security-contact "security@yourdomain.com" \
  --details "Validator description" \
  --chain-id paw-testnet-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1000000 \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.001upaw \
  --keyring-backend os \
  --broadcast-mode block
```

---

## Verify

```bash
# Validator info
pawd query staking validator $(pawd keys show validator-operator --bech val -a --keyring-backend os)

# Signing status
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check if bonded
pawd query staking validators --output json | jq -r '.validators[] | select(.description.moniker=="<moniker>")'
```

---

## Monitoring

```bash
# Sync status
pawd status | jq '.SyncInfo'

# Missed blocks
pawd query slashing signing-info $(pawd tendermint show-validator) | jq '.missed_blocks_counter'

# Jailed check
pawd query staking validator $(pawd keys show validator-operator --bech val -a --keyring-backend os) | jq '.jailed'

# Logs
sudo journalctl -u pawd -f
```

---

## Essential Commands

```bash
# Unjail
pawd tx slashing unjail --from validator-operator --chain-id paw-testnet-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Withdraw rewards + commission
pawd tx distribution withdraw-rewards $(pawd keys show validator-operator --bech val -a --keyring-backend os) --commission --from validator-operator --chain-id paw-testnet-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Edit validator
pawd tx staking edit-validator --website "https://new.com" --details "Updated" --from validator-operator --chain-id paw-testnet-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Change commission
pawd tx staking edit-validator --commission-rate 0.12 --from validator-operator --chain-id paw-testnet-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Vote on proposal
pawd tx gov vote <id> yes --from validator-operator --chain-id paw-testnet-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y
```

---

## Network Info

| Item | Value |
|------|-------|
| **Chain ID** | paw-testnet-1 |
| **Denom** | upaw |
| **Block Time** | ~3s |
| **RPC** | https://rpc.paw-testnet.io |
| **Explorer** | https://explorer.paw-testnet.io |
| **Genesis** | https://raw.githubusercontent.com/decristofaroj/paw/main/networks/paw-testnet-1/genesis.json |

---

## Next Steps

1. Setup monitoring: [VALIDATOR_MONITORING.md](./VALIDATOR_MONITORING.md)
2. Harden security: [VALIDATOR_SECURITY.md](./VALIDATOR_SECURITY.md)
3. Configure sentry: [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md)
4. Join Discord: https://discord.gg/paw-blockchain

---

**Full Guide:** [VALIDATOR_ONBOARDING_GUIDE.md](./VALIDATOR_ONBOARDING_GUIDE.md)
