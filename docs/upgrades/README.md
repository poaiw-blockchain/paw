# PAW Blockchain Upgrade Guide

## Overview

This directory contains documentation for PAW blockchain upgrades, including upgrade procedures, testing guides, and rollback instructions.

## Upgrade Philosophy

The PAW blockchain follows a structured upgrade process that prioritizes:

1. **Safety First**: Comprehensive testing before mainnet deployment
2. **Transparency**: Clear communication with validators and community
3. **Reversibility**: Well-documented rollback procedures
4. **Minimal Downtime**: Optimized upgrade procedures for quick execution

## Upgrade Process

### 1. Development Phase

```
┌─────────────────────────────────────────┐
│  1. Identify upgrade requirements      │
│  2. Implement changes and migrations   │
│  3. Write comprehensive tests          │
│  4. Code review + third-party audit*   │
└─────────────────────────────────────────┘
*See `docs/guides/security/THIRD_PARTY_AUDIT_PLAN.md`.
```

### 2. Testing Phase

```
┌─────────────────────────────────────────┐
│  1. Test on local development network  │
│  2. Deploy to public testnet           │
│  3. Run integration tests              │
│  4. Perform load testing               │
│  5. Community testing period           │
└─────────────────────────────────────────┘
```

### 3. Governance Phase

```
┌─────────────────────────────────────────┐
│  1. Submit upgrade proposal            │
│  2. Community discussion (7 days)      │
│  3. Voting period (7 days)             │
│  4. If approved, prepare for upgrade   │
└─────────────────────────────────────────┘
```

### 4. Execution Phase

```
┌─────────────────────────────────────────┐
│  1. Pre-upgrade validator preparation  │
│  2. Chain halts at upgrade height      │
│  3. Validators upgrade binaries        │
│  4. Chain restarts with new version    │
│  5. Post-upgrade verification          │
└─────────────────────────────────────────┘
```

## Upgrade Types

### Consensus-Breaking Upgrades

Changes that affect consensus and require all validators to upgrade:
- Module consensus version changes
- Store structure changes
- Transaction format changes
- Consensus parameter changes

**Process:** Governance proposal → Coordinated upgrade

### Non-Consensus Upgrades

Changes that don't affect consensus:
- Bug fixes
- Performance improvements
- API improvements
- Documentation updates

**Process:** Standard release → Gradual rollout

## Directory Structure

```
docs/upgrades/
├── README.md                 # This file
├── ROLLBACK.md              # Rollback procedures
├── v1.1.0.md                # v1.1.0 upgrade guide
├── v1.2.0.md                # v1.2.0 upgrade guide (TBD)
├── v1.3.0.md                # v1.3.0 upgrade guide (TBD)
├── templates/
│   ├── upgrade-proposal.md  # Proposal template
│   └── upgrade-guide.md     # Upgrade guide template
└── archive/
    └── (completed upgrades)
```

## Upgrade Checklist

### For Validators

#### Pre-Upgrade (1 week before)
- [ ] Read upgrade documentation
- [ ] Test upgrade on local node
- [ ] Backup node state and keys
- [ ] Setup Cosmovisor (recommended)
- [ ] Join validator coordination channel
- [ ] Verify system requirements

#### During Upgrade (Upgrade day)
- [ ] Monitor for upgrade height
- [ ] Stop node at upgrade height
- [ ] Install new binary
- [ ] Restart node
- [ ] Monitor logs for errors
- [ ] Verify block production

#### Post-Upgrade (Within 24 hours)
- [ ] Verify validator is signing blocks
- [ ] Test basic operations
- [ ] Report any issues
- [ ] Update monitoring systems

### For Node Operators

#### Pre-Upgrade
- [ ] Review upgrade guide
- [ ] Backup node data
- [ ] Update monitoring
- [ ] Schedule maintenance window
- [ ] Communicate downtime to users

#### During Upgrade
- [ ] Follow upgrade procedure
- [ ] Monitor node status
- [ ] Verify sync status

#### Post-Upgrade
- [ ] Test RPC endpoints
- [ ] Verify data consistency
- [ ] Update documentation

### For Developers/Integrators

#### Pre-Upgrade
- [ ] Review API changes
- [ ] Update client libraries
- [ ] Test against upgraded testnet
- [ ] Update documentation
- [ ] Prepare user communications

#### During Upgrade
- [ ] Pause automated systems
- [ ] Monitor upgrade progress

#### Post-Upgrade
- [ ] Resume automated systems
- [ ] Verify integration functionality
- [ ] Monitor for errors

## Upgrade Tools

### Cosmovisor

Recommended tool for automatic upgrades:

```bash
# Install Cosmovisor
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

# Setup directory structure
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
mkdir -p $DAEMON_HOME/cosmovisor/upgrades

# Copy current binary
cp $(which pawd) $DAEMON_HOME/cosmovisor/genesis/bin

# Setup upgrade
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin
cp pawd-v1.1.0 $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd

# Start with Cosmovisor
cosmovisor run start
```

### State Export

For backup and rollback:

```bash
# Export current state
pawd export > state-$(date +%Y%m%d-%H%M%S).json

# Export at specific height
pawd export --height 1000000 > state-height-1000000.json

# Verify export
jq '.app_state' state.json > /dev/null && echo "Valid" || echo "Invalid"
```

### Upgrade Info

Check upgrade schedule:

```bash
# Query scheduled upgrade
pawd query upgrade plan

# Query module versions
pawd query upgrade module_versions

# Query applied upgrades
pawd query upgrade applied <upgrade-name>
```

## Common Issues and Solutions

### Issue: Chain halts at upgrade height but validators can't start

**Solution:**
```bash
# Verify binary version
pawd version

# Check logs for errors
journalctl -u pawd -n 100

# Verify upgrade info
ls -l $DAEMON_HOME/cosmovisor/upgrades/
```

### Issue: State export fails

**Solution:**
```bash
# Export at last known good height
pawd export --height <HEIGHT-1> > state.json

# If database is corrupted, use database backup
cp -r ~/.paw-backup/data ~/.paw/data
```

### Issue: Validator not signing after upgrade

**Solution:**
```bash
# Check validator status
pawd query staking validator $(pawd keys show validator --bech val -a)

# Check signing info
pawd query slashing signing-info $(pawd tendermint show-validator)

# Restart node
sudo systemctl restart pawd
```

## Testing Upgrades

### Local Testing

```bash
# 1. Setup local test chain
pawd init test --chain-id test-1
pawd keys add validator

# 2. Create genesis
pawd genesis add-genesis-account validator 1000000000upaw
pawd genesis gentx validator 1000000upaw --chain-id test-1
pawd genesis collect-gentxs

# 3. Start chain
pawd start

# 4. Submit upgrade proposal
pawd tx gov submit-proposal software-upgrade v1.1.0 \
  --title "Test Upgrade" \
  --description "Testing v1.1.0" \
  --upgrade-height 100 \
  --from validator \
  --yes

# 5. Vote on proposal
pawd tx gov vote 1 yes --from validator --yes

# 6. Wait for upgrade height and verify
```

### Testnet Testing

```bash
# Connect to testnet
pawd config chain-id paw-mvp-1
pawd config node https://rpc.testnet.paw.com:443

# Submit test proposal
pawd tx gov submit-proposal software-upgrade v1.1.0 \
  --title "Testnet Upgrade" \
  --description "Testing v1.1.0 on testnet" \
  --upgrade-height <HEIGHT> \
  --from validator \
  --deposit 10000000upaw \
  --yes
```

## Upgrade Governance

### Proposal Submission

```bash
# Prepare upgrade info JSON
cat > upgrade-info.json <<EOF
{
  "binaries": {
    "linux/amd64": "https://github.com/paw-chain/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-amd64.tar.gz",
    "linux/arm64": "https://github.com/paw-chain/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-arm64.tar.gz",
    "darwin/amd64": "https://github.com/paw-chain/paw/releases/download/v1.1.0/pawd-v1.1.0-darwin-amd64.tar.gz"
  }
}
EOF

# Submit proposal
pawd tx gov submit-proposal software-upgrade v1.1.0 \
  --title "PAW v1.1.0 Upgrade" \
  --description "$(cat upgrade-description.md)" \
  --upgrade-height <HEIGHT> \
  --upgrade-info "$(cat upgrade-info.json)" \
  --deposit 10000000upaw \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --yes
```

### Voting

```bash
# Vote yes
pawd tx gov vote <proposal-id> yes \
  --from validator \
  --chain-id paw-1 \
  --yes

# Check vote status
pawd query gov votes <proposal-id>

# Check tally
pawd query gov tally <proposal-id>
```

## Communication Channels

During upgrades, coordinate through:

1. **Validator Discord**: Real-time coordination
2. **Governance Forum**: Upgrade discussions
3. **Twitter**: Public announcements
4. **Telegram**: Community support

## Resources

### Official Documentation
- [Cosmos SDK Upgrades](https://docs.cosmos.network/main/core/upgrade)
- [Cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor)
- [PAW Documentation](https://docs.paw-chain.org)

### Tools
- [Cosmovisor](https://github.com/cosmos/cosmos-sdk/tree/main/tools/cosmovisor)
- [Chain Registry](https://github.com/cosmos/chain-registry)

### Support
- **Discord**: https://discord.gg/DBHTc2QV
- **Telegram**: https://t.me/paw_chain
- **Email**: support@paw-chain.org

## Upgrade History

| Version | Date | Type | Description |
|---------|------|------|-------------|
| v1.1.0 | TBD | Consensus | State validation and security improvements |
| v1.2.0 | TBD | Consensus | TBD |
| v1.3.0 | TBD | Consensus | TBD |

## FAQ

**Q: How long does an upgrade take?**
A: Typically 10-30 minutes for the chain to halt and restart.

**Q: Will I lose my tokens during an upgrade?**
A: No, upgrades preserve all state including balances.

**Q: What if I miss the upgrade?**
A: You must upgrade your binary before you can sync with the chain.

**Q: Can I opt out of an upgrade?**
A: No, consensus upgrades require all validators to upgrade.

**Q: What happens to pending transactions?**
A: Pending transactions are cleared and must be resubmitted.

---

**Last Updated:** 2025-11-25
**Maintainer:** PAW Core Development Team
