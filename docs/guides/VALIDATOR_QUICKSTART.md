# PAW Validator Quickstart Guide

Operational checklist for bringing up a single PAW validator from a clean Ubuntu 22.04 host. Follow the steps in order; every command assumes the default Cosmos SDK home at `~/.paw` unless noted.

## 1. Hardware and Network Requirements
| Tier | CPU | RAM | Storage | Network |
|------|-----|-----|---------|---------|
| **Minimum (testnet/dev)** | 4 vCPU (x86_64) | 8 GB | 250 GB SSD | 100 Mbps symmetrical |
| **Recommended (mainnet-ready)** | 8 vCPU / 16 threads | 32 GB | 1 TB NVMe SSD (single partition) | ≥1 Gbps dedicated uplink |

Additional requirements:
- Ubuntu 22.04 LTS or equivalent systemd-based Linux
- Static public IP and open inbound ports 26656 (P2P), 26657 (RPC for sentries), 1317 (REST, optional)
- Shell access with sudo privileges

## 2. Install Base Dependencies
```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y build-essential git curl jq unzip systemd

# Install Go 1.23+ (example uses 1.23.5)
curl -L https://go.dev/dl/go1.23.5.linux-amd64.tar.gz -o /tmp/go.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz
cat <<'EOP' >> ~/.bashrc
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
EOP
source ~/.bashrc
```
Verify:
```bash
go version
```

## 3. Build the `pawd` Binary
```bash
cd ~/blockchain-projects/paw   # adjust if cloned elsewhere
git fetch origin && git checkout main
make build                     # outputs ./build/pawd
sudo install -m 0755 build/pawd /usr/local/bin/pawd
pawd version
```

## 4. Initialize the Validator Home
```bash
export PAW_HOME=$HOME/.paw   # optional; default is already ~/.paw
pawd init <moniker> --chain-id paw-testnet-1 --home $PAW_HOME
```
The command generates `config/app.toml`, `config/config.toml`, and validator keys under `$PAW_HOME`.

## 5. Create a Key and Fund the Genesis Account
```bash
pawd keys add <validator-key> --keyring-backend os --home $PAW_HOME
# Record the Bech32 address returned above.

pawd add-genesis-account <validator-address> 1000000000upaw \
  --home $PAW_HOME --keyring-backend os
```
For shared testnets obtain tokens from the faucet instead of editing genesis.

## 6. Generate and Submit a Gentx
```bash
pawd gentx <validator-key> 700000000upaw \
  --chain-id paw-testnet-1 --home $PAW_HOME --keyring-backend os

# If you are orchestrating a new network:
pawd collect-gentxs --home $PAW_HOME
```
Compress and send the generated `gentx/*.json` to the coordinating team for multi-validator launches.

## 7. Configure Networking and Start the Node
1. **Set persistent peers** in `$PAW_HOME/config/config.toml` (comma-separated `node-id@ip:26656`).
2. **Set minimum gas price** in `$PAW_HOME/config/app.toml` → `minimum-gas-prices = "0.001upaw"`.
3. **Optional**: enable Prometheus by setting `prometheus = true` in `config/config.toml`.

Start the node in the foreground to confirm operation:
```bash
pawd start --home $PAW_HOME \
  --grpc.address 0.0.0.0:19090 \
  --api.address tcp://0.0.0.0:1317 \
  --rpc.laddr tcp://0.0.0.0:26657 \
  --minimum-gas-prices 0.001upaw
```
On first boot the node will create the data directory under `$PAW_HOME/data`; wait for it to catch up before exposing RPC endpoints.

## 8. Systemd Service Example
Create `/etc/systemd/system/pawd.service`:
```ini
[Unit]
Description=PAW Validator
After=network-online.target
Wants=network-online.target

[Service]
User=validator
Group=validator
Environment="PAW_HOME=/home/validator/.paw"
ExecStart=/usr/local/bin/pawd start --home ${PAW_HOME} --minimum-gas-prices 0.001upaw
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```
Steps:
```bash
sudo useradd -m -s /bin/bash validator  # skip if user exists
sudo chown -R validator:validator $PAW_HOME
sudo systemctl daemon-reload
sudo systemctl enable --now pawd
```
Check status:
```bash
systemctl status pawd
journalctl -fu pawd
```

## 9. Monitoring & Alerting
- **Node health**: `curl localhost:26657/status` (RPC) and `pawd status --node http://localhost:26657`.
- **Prometheus metrics**: enable in `config/config.toml` and scrape `http://localhost:26660/metrics`.
- **Block production**: `pawd query staking validators --node http://localhost:26657 | jq '.validators[] | select(.description.moniker=="<moniker>")'`.
- **Log shipping**: follow `journalctl -u pawd -S -5m` for recent events; forward to Loki/Elastic as needed.
- **Alert ideas**: missed blocks (consensus address), disk usage, process restart count, peer count (`pawd status | jq '.SyncInfo.CatchingUp'`).

## 10. Post-Install Checklist
- [ ] Node appears in `pawd query staking validators`
- [ ] No `CatchingUp` in RPC status
- [ ] Ports 26656/26657 reachable from sentries/relayers
- [ ] Prometheus metrics collected and alerts configured
- [ ] Off-site backups of `priv_validator_key.json` and `node_key.json`

Once the validator is stable, rotate keys and enroll in the monitoring/alerting stack required by the PAW testnet coordinators.
