# PAW Testnet Incident Response & Upgrade Runbook

**Chain ID:** `paw-testnet-1` | **Primary:** `54.39.103.49` | **Secondary:** `139.99.149.160` | **VPN:** `10.10.0.2`

---

## 1. Node Down Recovery

```bash
# SSH to server
ssh paw-testnet

# Check if process is running
pgrep -f pawd || echo "Node is DOWN"

# Check logs for crash reason
tail -100 ~/.paw/logs/node.log | grep -i "error\|panic\|fatal"

# Check disk space and memory
df -h && free -m

# Restart node
pkill -f pawd
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &

# Verify it's running
sleep 5 && curl -s localhost:11657/status | jq '.result.sync_info'
```

## 2. Missed Blocks Recovery

```bash
# Check current signing status
curl -s localhost:11657/status | jq '.result.validator_info'

# Check if syncing (should be false)
curl -s localhost:11657/status | jq '.result.sync_info.catching_up'

# If behind, check peer count
curl -s localhost:11657/net_info | jq '.result.n_peers'

# Add persistent peers if needed
sed -i 's/persistent_peers = ""/persistent_peers = "PEER_ID@IP:PORT"/' ~/.paw/config/config.toml

# Restart and monitor
pkill -f pawd && sleep 2
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
tail -f ~/.paw/logs/node.log | grep -i "committed\|signed"
```

## 3. Chain Halt Procedures

```bash
# Confirm halt - no new blocks
watch -n 5 'curl -s localhost:11657/status | jq ".result.sync_info.latest_block_height"'

# Check consensus state
curl -s localhost:11657/dump_consensus_state | jq '.result.round_state.height_vote_set'

# Coordinate with other validators (check Discord/Telegram)
# If >2/3 validators agree, may need coordinated restart:

# 1. All validators stop at same time
pkill -f pawd

# 2. Export state (if needed)
~/.paw/cosmovisor/genesis/bin/pawd export --home ~/.paw > genesis_export.json

# 3. Coordinated restart at agreed time
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

## 4. Cosmovisor Upgrade Steps

```bash
# Pre-upgrade checklist
~/.paw/cosmovisor/genesis/bin/pawd version  # Current version
df -h ~/.paw                                  # Ensure disk space

# Build new binary locally (on bcpc)
cd ~/blockchain-projects/paw && make build

# Copy to server
scp build/pawd paw-testnet:/tmp/pawd_new

# On server: prepare upgrade directory
ssh paw-testnet
UPGRADE_NAME="v2.0.0"  # Match governance proposal name
mkdir -p ~/.paw/cosmovisor/upgrades/$UPGRADE_NAME/bin
mv /tmp/pawd_new ~/.paw/cosmovisor/upgrades/$UPGRADE_NAME/bin/pawd
chmod +x ~/.paw/cosmovisor/upgrades/$UPGRADE_NAME/bin/pawd

# Verify binary
~/.paw/cosmovisor/upgrades/$UPGRADE_NAME/bin/pawd version

# Cosmovisor handles switch automatically at upgrade height
# Monitor logs during upgrade
tail -f ~/.paw/logs/node.log | grep -i "upgrade\|applying"
```

## 5. Manual Binary Upgrade (Emergency)

```bash
# Stop node immediately
pkill -f pawd

# Backup current binary
cp ~/.paw/cosmovisor/genesis/bin/pawd ~/.paw/cosmovisor/genesis/bin/pawd.bak

# Replace binary
scp bcpc:~/blockchain-projects/paw/build/pawd paw-testnet:~/.paw/cosmovisor/genesis/bin/pawd
chmod +x ~/.paw/cosmovisor/genesis/bin/pawd

# Verify version
~/.paw/cosmovisor/genesis/bin/pawd version

# Start with unsafe-reset-all ONLY if instructed (DESTROYS DATA)
# ~/.paw/cosmovisor/genesis/bin/pawd tendermint unsafe-reset-all --home ~/.paw

# Normal start
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

## 6. Database Corruption Recovery

```bash
# Stop node
pkill -f pawd

# Backup corrupted data (for analysis)
tar -czvf ~/.paw/data_corrupted_$(date +%Y%m%d).tar.gz ~/.paw/data

# Option A: Restore from snapshot
cd ~/.paw
rm -rf data
wget https://artifacts.poaiw.org/snapshots/latest.tar.gz
tar -xzvf latest.tar.gz

# Option B: State sync (faster, less storage)
# Edit config.toml with trust height/hash from working node
sed -i 's/enable = false/enable = true/' ~/.paw/config/config.toml
# Set trust_height, trust_hash, rpc_servers

# Option C: Full resync (slowest, most reliable)
rm -rf ~/.paw/data
mkdir -p ~/.paw/data

# Restart
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

## 7. Key Compromise Response

```bash
# IMMEDIATE: Stop signing
pkill -f pawd

# Rotate validator key
mv ~/.paw/config/priv_validator_key.json ~/.paw/config/priv_validator_key.json.COMPROMISED
mv ~/.paw/data/priv_validator_state.json ~/.paw/data/priv_validator_state.json.COMPROMISED

# Generate new key (requires re-delegation)
~/.paw/cosmovisor/genesis/bin/pawd init temp --home /tmp/newkey
cp /tmp/newkey/config/priv_validator_key.json ~/.paw/config/
echo '{"height":"0","round":0,"step":0}' > ~/.paw/data/priv_validator_state.json

# Create new validator with new key (old one will be jailed)
# Notify delegators to redelegate

# If node key compromised (less critical)
rm ~/.paw/config/node_key.json
# Will regenerate on restart
```

## 8. Network Split/Fork Recovery

```bash
# Check your chain vs canonical
curl -s localhost:11657/status | jq '.result.sync_info.latest_block_hash'
# Compare with other validators

# If on wrong fork:
pkill -f pawd

# Find last common block
GOOD_HEIGHT=12345  # Get from other validators

# Rollback to common ancestor
~/.paw/cosmovisor/genesis/bin/pawd rollback --home ~/.paw
# Repeat until at GOOD_HEIGHT

# Or nuclear option: state sync from scratch
rm -rf ~/.paw/data
# Configure state sync in config.toml
nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &
```

## 9. Emergency Contacts

| Role | Contact | Method |
|------|---------|--------|
| Lead Dev | [NAME] | Discord: @handle |
| Infra | [NAME] | Telegram: @handle |
| Security | [NAME] | Signal: +1XXX |
| OVH Support | N/A | support.ovh.com |

**Escalation Path:** Discord #validators -> Telegram group -> Direct call

## 10. Monitoring Dashboard Links

| Service | URL |
|---------|-----|
| Node Status | `curl localhost:11657/status` |
| Prometheus | `http://10.10.0.2:11660/metrics` |
| Block Explorer | https://explorer.poaiw.org |
| Netdata (bcpc) | http://192.168.100.2:19999 |
| Grafana | [Configure if available] |

**Quick Health Check:**
```bash
# One-liner status
curl -s localhost:11657/status | jq '{catching_up:.result.sync_info.catching_up,height:.result.sync_info.latest_block_height,peers:.result.n_peers}'
```

---

**Last Updated:** 2026-01-03 | **Version:** 1.0
