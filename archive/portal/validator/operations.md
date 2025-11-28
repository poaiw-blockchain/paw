# Validator Operations

Daily operations and maintenance for PAW validators.

## Monitoring Your Validator

```bash
# Check status
pawd status | jq

# View validator info
pawd query staking validator $(pawd keys show validator --bech val -a)

# Check signing status
pawd query slashing signing-info $(pawd tendermint show-validator)

# View delegations
pawd query staking delegations-to $(pawd keys show validator --bech val -a)

# Check rewards
pawd query distribution commission $(pawd keys show validator --bech val -a)
```

## Claiming Commission

```bash
# Withdraw commission
pawd tx distribution withdraw-rewards \
  $(pawd keys show validator --bech val -a) \
  --commission \
  --from validator \
  --fees 500upaw \
  --chain-id paw-mainnet-1

# Auto-compound (stake commission)
# Add to cron for automation
```

## Updating Commission

```bash
# Change commission rate
pawd tx staking edit-validator \
  --commission-rate="0.08" \
  --from validator \
  --fees 500upaw
```

## Unjailing

If validator gets jailed:

```bash
# Check if jailed
pawd query staking validator $(pawd keys show validator --bech val -a) | jq '.jailed'

# Unjail
pawd tx slashing unjail \
  --from validator \
  --fees 500upaw

# Resume operations
sudo systemctl restart pawd
```

## Upgrades

```bash
# Backup before upgrade
sudo systemctl stop pawd
cp -r ~/.paw ~/.paw.backup

# Download new version
cd ~/paw
 fetch --all
 checkout v2.0.0
make install

# Restart
sudo systemctl start pawd

# Verify
pawd version
```

## Backup & Recovery

```bash
# Backup critical files
tar -czf paw-backup-$(date +%Y%m%d).tar.gz \
  ~/.paw/config/priv_validator_key.json \
  ~/.paw/config/node_key.json \
  ~/.paw/data/priv_validator_state.json

# Store encrypted backups offsite
```

---

**Previous:** [Setup](/validator/setup) | **Next:** [Security](/validator/security) →
