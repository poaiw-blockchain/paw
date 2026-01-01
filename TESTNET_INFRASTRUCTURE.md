# PAW Blockchain Testnet Infrastructure

> **Last Updated**: 2026-01-01
> **Maintainer**: AI Agents via Claude Code

## Quick Reference

| Item | Value |
|------|-------|
| **Server IP** | 54.39.103.49 |
| **SSH Access** | `ssh paw-testnet` (from WSL2) |
| **VPN IP** | 10.10.0.2 |
| **Chain ID** | paw-testnet-1 |
| **Denom** | upaw |
| **Binary** | `~/.paw/cosmovisor/genesis/bin/pawd` |
| **Home Dir** | `~/.paw` |
| **RPC Port** | 26657 (localhost) |
| **P2P Port** | 26656 |
| **gRPC Port** | 9090 |
| **API Port** | 1317 |

---

## 1. Server Access

### SSH Configuration (from WSL2)
```bash
# Direct access
ssh paw-testnet

# Or explicitly
ssh ubuntu@54.39.103.49
```

SSH is configured in `~/.ssh/config` on WSL2 with key-based authentication.

### WireGuard VPN
- **Interface**: wg0
- **Private IP**: 10.10.0.2/24
- **Config**: `/etc/wireguard/wg0.conf`
- **Peers**: AURA (10.10.0.1), XAI (10.10.0.3)

```bash
# Check VPN status
sudo wg show

# Ping other nodes
ping 10.10.0.1  # AURA
ping 10.10.0.3  # XAI
```

---

## 2. Directory Structure

```
~/.paw/
├── config/
│   ├── app.toml          # Application configuration
│   ├── client.toml       # Client configuration
│   ├── config.toml       # CometBFT configuration
│   ├── genesis.json      # Chain genesis
│   ├── node_key.json     # Node identity key
│   └── priv_validator_key.json  # Validator signing key
├── cosmovisor/
│   ├── current -> genesis
│   └── genesis/
│       └── bin/
│           └── pawd      # Main binary
├── data/
│   ├── application.db/   # Application state (IAVL)
│   ├── blockstore.db/    # Block storage
│   ├── state.db/         # CometBFT state
│   ├── tx_index.db/      # Transaction index
│   ├── evidence.db/      # Evidence storage
│   ├── snapshots/        # State sync snapshots
│   └── priv_validator_state.json
└── logs/
    └── node.log          # Node output log

~/paw/                    # Source code repository
```

---

## 3. Binary & CLI

### Location
```bash
# Primary binary (via cosmovisor)
~/.paw/cosmovisor/genesis/bin/pawd

# Alias configured in ~/.bashrc
alias pawd="~/.paw/cosmovisor/genesis/bin/pawd"
```

### Common Commands
```bash
# Node status
pawd status --home ~/.paw

# Query custom modules
pawd query compute params --home ~/.paw
pawd query compute providers --home ~/.paw
pawd query dex params --home ~/.paw
pawd query dex pools --home ~/.paw
pawd query oracle params --home ~/.paw
pawd query oracle prices --home ~/.paw

# Query standard modules
pawd query ibc channel channels --home ~/.paw
pawd query ibc client states --home ~/.paw

# Keys management
pawd keys list --home ~/.paw
pawd keys add <name> --home ~/.paw

# Transactions
pawd tx compute register-provider <args> --home ~/.paw --chain-id paw-testnet-1
pawd tx dex create-pool <args> --home ~/.paw --chain-id paw-testnet-1
```

---

## 4. Custom Modules

PAW has three custom modules beyond standard Cosmos SDK:

### Compute Module
- **Purpose**: Decentralized AI compute marketplace
- **Store Key**: `compute`
- **Queries**: `params`, `providers`, `requests`, `appeals`, `disputes`
- **Transactions**: `register-provider`, `submit-request`, `submit-result`

### DEX Module
- **Purpose**: Decentralized exchange with AMM
- **Store Key**: `dex`
- **Queries**: `params`, `pools`, `limit-orders`, `stats`
- **Transactions**: `create-pool`, `add-liquidity`, `swap`

### Oracle Module
- **Purpose**: Price feeds and data oracles
- **Store Key**: `oracle`
- **Queries**: `params`, `prices`, `validators`
- **Transactions**: `submit-price`, `register-validator`

---

## 5. Configuration Files

### app.toml (Key Settings)
```toml
# Location: ~/.paw/config/app.toml

minimum-gas-prices = "0.001upaw"
pruning = "default"
iavl-disable-fastnode = true

[api]
enable = true
address = "tcp://localhost:1317"

[grpc]
enable = true
address = "localhost:9090"

[state-sync]
snapshot-interval = 2000
snapshot-keep-recent = 5
```

### config.toml (Key Settings)
```toml
# Location: ~/.paw/config/config.toml

moniker = "paw-testnet"

[rpc]
laddr = "tcp://127.0.0.1:26657"

[p2p]
laddr = "tcp://0.0.0.0:26656"
```

---

## 6. Service Management

### Manual Start (Primary Method)
```bash
# Start node
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &

# Stop node
pkill -f pawd

# View logs
tail -f ~/.paw/logs/node.log
```

### Related Services
```bash
# Block explorer
sudo systemctl status paw-explorer

# Faucet
sudo systemctl status paw-faucet
```

---

## 7. Chain Operations

### Reset Chain (Full Reset)
```bash
# Stop node first
pkill -f pawd

# Reset all data
pawd tendermint unsafe-reset-all --home ~/.paw

# Start fresh
nohup pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

### Export State
```bash
pawd export --home ~/.paw > genesis_export.json
```

---

## 8. Monitoring & Debugging

### Check Node Status
```bash
# Basic status
pawd status --home ~/.paw | jq '.sync_info'

# Check block height
pawd status --home ~/.paw | jq '.sync_info.latest_block_height'

# Check if catching up
pawd status --home ~/.paw | jq '.sync_info.catching_up'
```

### Check Logs for Errors
```bash
# Filter out expected port errors
tail -f ~/.paw/logs/node.log | grep -v prometheus | grep -v "health check"

# Search for real errors
grep -i "error\|panic" ~/.paw/logs/node.log | grep -v prometheus | tail -20
```

### Known Benign Errors
These errors can be ignored:
- `prometheus server error: listen tcp :36660: bind: address already in use`
- `health check server error: listen tcp :36661: bind: address already in use`

---

## 9. Genesis & Validator Info

### Chain Parameters
- **Chain ID**: paw-testnet-1
- **Denom**: upaw (1 PAW = 1,000,000 upaw)
- **Block Time**: ~4 seconds
- **Bech32 Prefix**: paw

### Validator
```bash
# Get validator operator address
pawd keys show <key-name> --bech val --home ~/.paw

# Check validator in set
pawd query staking validators --home ~/.paw
```

---

## 10. Source Code Repository

### Location on Server
```bash
~/paw/
```

### GitHub
```
git@github.com:poaiw-blockchain/paw.git
```

### Build from Source
```bash
cd ~/paw
export PATH=$PATH:/usr/local/go/bin
git pull origin main
make build
cp build/pawd ~/.paw/cosmovisor/genesis/bin/pawd

# Restart node
pkill -f pawd
nohup pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

### Important: Go Path
Go is installed at `/usr/local/go/bin` but may not be in PATH by default:
```bash
export PATH=$PATH:/usr/local/go/bin
```

---

## 11. Network Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                     OVHcloud KS-5 Servers                       │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   AURA Testnet  │   PAW Testnet   │        XAI Testnet          │
│  158.69.119.76  │  54.39.103.49   │       54.39.129.11          │
│   10.10.0.1     │   10.10.0.2     │        10.10.0.3            │
│   (wg0)         │   (wg0)         │        (wg0)                │
└────────┬────────┴────────┬────────┴────────────┬────────────────┘
         │                 │                     │
         └─────────────────┼─────────────────────┘
                           │
                    WireGuard Mesh
                    (Port 51820)
```

---

## 12. Ports Summary

| Port | Service | Binding |
|------|---------|---------|
| 26656 | P2P | 0.0.0.0 |
| 26657 | RPC | localhost |
| 9090 | gRPC | localhost |
| 1317 | REST API | localhost |
| 36660 | Prometheus | localhost |
| 36661 | Health Check | localhost |
| 51820 | WireGuard | 0.0.0.0 |

---

## 13. Troubleshooting

### "Version Does Not Exist" Error
This was fixed by setting `ConsensusVersion = 1` in custom modules.
If it recurs after code changes:
```bash
# Check module versions
grep -n "ConsensusVersion" ~/paw/x/*/module.go

# Should return 1, not 2, for new genesis
```

### Node Won't Start
```bash
# Check for port conflicts
sudo lsof -i :26656
sudo lsof -i :26657

# Check disk space
df -h

# Review logs
tail -100 ~/.paw/logs/node.log | grep -v prometheus
```

### Build Errors
```bash
# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin

# Clean build
cd ~/paw
make clean
make build
```

---

## 14. Recent Fixes Applied

### ConsensusVersion Fix (2026-01-01)
**Commit**: `c32977d`
**Issue**: Custom modules had `ConsensusVersion = 2` but fresh genesis expects 1
**Fix**: Changed to `ConsensusVersion = 1` in:
- `x/compute/module.go`
- `x/dex/module.go`
- `x/oracle/module.go`

### API/gRPC Enable (2026-01-01)
**Commit**: `d730d75`
**Issue**: API and gRPC disabled by default
**Fix**: Set `enable = true` in `initAppConfig()` in `cmd/pawd/cmd/root.go`

---

## 15. Related Documentation

- [Cosmos SDK Docs](https://docs.cosmos.network)
- [CometBFT Docs](https://docs.cometbft.com)
- [PAW GitHub Repository](https://github.com/poaiw-blockchain/paw)
