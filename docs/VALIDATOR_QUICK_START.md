# PAW Validator Quick Start

One-page reference for experienced Cosmos validator operators.

---

## Prerequisites

- Ubuntu 22.04 LTS, 8 cores, 32 GB RAM, 500 GB NVMe SSD
- Go 1.24+, Git, Make, jq
- Static IP, ports 26656 (P2P) open

---

## Installation (5 minutes)

```bash
# Install Go 1.24.11
wget https://go.dev/dl/go1.24.11.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.11.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Build pawd
git clone https://github.com/paw-chain/paw.git && cd paw
git checkout main
make build && sudo install -m 0755 build/pawd /usr/local/bin/pawd

# Initialize
pawd init <moniker> --chain-id paw-mvp-1

# Download genesis and peers
curl -L https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/genesis.json > ~/.paw/config/genesis.json
curl -L https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/genesis.sha256 > /tmp/genesis.sha256
cd ~/.paw/config && sha256sum -c /tmp/genesis.sha256

# Configure peers
PEERS=$(curl -s https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/peers.txt | tr '\n' ',' | sed 's/,$//')
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

Request small amounts from the faucet listed in `docs/TESTNET_QUICK_REFERENCE.md`, then verify:
```bash
pawd query bank balances <address> --node tcp://localhost:26657
```

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
  --chain-id paw-mvp-1 \
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
pawd tx slashing unjail --from validator-operator --chain-id paw-mvp-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Withdraw rewards + commission
pawd tx distribution withdraw-rewards $(pawd keys show validator-operator --bech val -a --keyring-backend os) --commission --from validator-operator --chain-id paw-mvp-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Edit validator
pawd tx staking edit-validator --website "https://new.com" --details "Updated" --from validator-operator --chain-id paw-mvp-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Change commission
pawd tx staking edit-validator --commission-rate 0.12 --from validator-operator --chain-id paw-mvp-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y

# Vote on proposal
pawd tx gov vote <id> yes --from validator-operator --chain-id paw-mvp-1 --gas auto --gas-prices 0.001upaw --keyring-backend os -y
```

---

## Network Info

| Item | Value |
|------|-------|
| **Chain ID** | paw-mvp-1 |
| **Denom** | upaw |
| **Genesis** | https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/genesis.json |
| **Genesis checksum** | https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/genesis.sha256 |
| **Peers** | https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/peers.txt |

---

## Next Steps

1. Operations: [VALIDATOR_OPERATOR_GUIDE.md](./VALIDATOR_OPERATOR_GUIDE.md)
2. Key management and security: [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md)
3. Sentry topology: [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md) and [SENTRY_TESTING_GUIDE.md](./SENTRY_TESTING_GUIDE.md)
4. Observability: [DASHBOARDS_GUIDE.md](./DASHBOARDS_GUIDE.md)

---

**Full Guide:** [VALIDATOR_ONBOARDING_GUIDE.md](./VALIDATOR_ONBOARDING_GUIDE.md)
