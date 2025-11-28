# Validator Troubleshooting

Solutions to common validator issues.

## Node Won't Start

```bash
# Check logs
sudo journalctl -u pawd -f

# Common fixes:
# 1. Wrong genesis file
wget -O ~/.paw/config/genesis.json https://...

# 2. Corrupted state
pawd unsafe-reset-all

# 3. Port conflict
netstat -tulpn | grep 26657
```

## Node Not Syncing

```bash
# Check peers
curl localhost:26657/net_info | jq '.result.n_peers'

# Add peers
PEERS="peer1:26656,peer2:26656"
sed -i "s/^persistent_peers =.*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
sudo systemctl restart pawd

# Try state sync
# See setup guide
```

## Validator Jailed

```bash
# Check reason
pawd query staking validator $(pawd keys show validator --bech val -a)

# Downtime jail:
pawd tx slashing unjail --from validator

# Double-sign (severe):
# Cannot unjail, must create new validator
```

## High Memory Usage

```bash
# Enable pruning
sed -i 's/pruning = "default"/pruning = "custom"/' ~/.paw/config/app.toml
sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "100"/' ~/.paw/config/app.toml

# Restart
sudo systemctl restart pawd
```

## Missed Blocks

```bash
# Check signing status
pawd query slashing signing-info $(pawd tendermint show-validator)

# Possible causes:
# - Node offline
# - Clock drift (install NTP)
# - Network issues
# - Hardware problems
```

## Disk Full

```bash
# Check usage
df -h

# Clean old logs
sudo journalctl --vacuum-time=7d

# Compact database
pawd compact-db

# Enable pruning (see above)
```

## Getting Help

- Discord: [discord.gg/pawblockchain](https://discord.gg/pawblockchain)
- Validator Chat: `#validator-support`

---

**Previous:** [Monitoring](/validator/monitoring) | **Next:** [Architecture](/reference/architecture) →
