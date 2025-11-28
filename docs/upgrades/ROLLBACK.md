# PAW Blockchain Upgrade Rollback Procedures

## Overview

This document provides detailed procedures for rolling back a failed or problematic chain upgrade. Rollbacks should only be performed in critical situations where the upgrade has caused chain halt, consensus failures, or critical data integrity issues.

**IMPORTANT:** Rollbacks require coordination among validators and should only be initiated by chain governance or emergency consensus among validators.

## When to Rollback

Consider a rollback in these situations:

1. **Chain Halt**: The chain fails to produce blocks after upgrade
2. **Consensus Failure**: Validators cannot reach consensus on the upgraded chain
3. **Critical Bugs**: Upgrade introduces critical bugs affecting chain operation
4. **State Corruption**: Upgrade causes state corruption or data loss
5. **Security Vulnerability**: Upgrade introduces security vulnerabilities

**DO NOT rollback for:**
- Minor bugs that don't affect consensus
- Individual validator configuration issues
- Temporary network issues
- Issues that can be fixed with a quick patch

## Prerequisites

Before attempting a rollback:

1. **Consensus**: Achieve consensus among validators (67%+ voting power)
2. **Coordination**: Establish communication channel for all validators
3. **Backups**: Ensure all validators have pre-upgrade state backups
4. **Documentation**: Document the reason for rollback
5. **Timeline**: Establish a coordinated rollback time

## Rollback Procedure

### Phase 1: Preparation (Before Rollback)

#### 1. Establish Coordination

```bash
# Create a coordination channel (Discord, Telegram, etc.)
# Share the following information:
# - Current chain state
# - Upgrade height
# - Rollback target height
# - Coordination timestamp
```

#### 2. Verify Backup Availability

```bash
# Verify you have pre-upgrade state export
ls -lh pre-upgrade-state.json

# Verify backup integrity
jq '.app_state' pre-upgrade-state.json > /dev/null && echo "Backup valid" || echo "Backup corrupted"

# Verify backup size
du -h pre-upgrade-state.json
```

#### 3. Prepare Rollback Binary

```bash
# Locate previous binary version
which pawd-v1.0.0  # Or wherever you stored it

# Verify binary version
pawd-v1.0.0 version
# Should output: v1.0.0

# Test binary
pawd-v1.0.0 status --node tcp://localhost:26657
```

### Phase 2: Coordinated Rollback

#### Step 1: Stop All Validators (Coordinated)

```bash
# At the agreed-upon time, ALL validators must stop simultaneously
sudo systemctl stop pawd

# Verify process is stopped
ps aux | grep pawd
```

#### Step 2: Backup Current State (Even if Failed)

```bash
# Backup failed upgrade state for analysis
cp -r ~/.paw ~/.paw-failed-upgrade-backup
tar -czf failed-upgrade-$(date +%Y%m%d-%H%M%S).tar.gz ~/.paw-failed-upgrade-backup

# Export current state if possible
pawd-v1.0.0 export --home ~/.paw > failed-upgrade-state.json 2>&1 || echo "Export failed"
```

#### Step 3: Restore Previous Binary

```bash
# Replace binary with previous version
sudo cp pawd-v1.0.0 /usr/local/bin/pawd

# If using Cosmovisor
cp pawd-v1.0.0 $DAEMON_HOME/cosmovisor/current/bin/pawd

# Verify version
pawd version
# Should output: v1.0.0
```

#### Step 4: Reset to Pre-Upgrade State

**Option A: Using State Export (Recommended)**

```bash
# Reset application state
pawd tendermint unsafe-reset-all --home ~/.paw

# Initialize with pre-upgrade genesis
pawd init <moniker> --chain-id paw-1 --home ~/.paw --overwrite

# Import pre-upgrade state
cp pre-upgrade-state.json ~/.paw/config/genesis.json

# Verify genesis hash
jq -S -c '.app_state' ~/.paw/config/genesis.json | shasum -a 256
# All validators must have the same hash
```

**Option B: Using Database Backup**

```bash
# Stop node (already done)
sudo systemctl stop pawd

# Remove current data
rm -rf ~/.paw/data

# Restore from backup
cp -r ~/.paw-pre-upgrade-backup/data ~/.paw/data

# Verify data directory
ls -lh ~/.paw/data
```

#### Step 5: Coordinate Chain Restart

```bash
# At the agreed-upon time, ALL validators restart simultaneously
sudo systemctl start pawd

# Monitor startup
journalctl -u pawd -f

# Check sync status
pawd status | jq '.SyncInfo'
```

### Phase 3: Verification

#### 1. Verify Chain Health

```bash
# Check if chain is producing blocks
pawd status | jq '.SyncInfo.latest_block_height'
# Run multiple times, height should increase

# Check validator set
pawd query staking validators --output json | jq '.validators[] | {moniker, status}'

# Verify your validator is active
pawd query slashing signing-info $(pawd tendermint show-validator)
```

#### 2. Verify State Consistency

```bash
# Export current state
pawd export > post-rollback-state.json

# Compare with pre-upgrade state
diff <(jq -S '.app_state' pre-upgrade-state.json) \
     <(jq -S '.app_state' post-rollback-state.json)

# Verify account balances
pawd query bank total

# Verify staking state
pawd query staking pool
```

#### 3. Test Basic Operations

```bash
# Test query operations
pawd query bank balances $(pawd keys show -a validator)

# Test transaction submission
pawd tx bank send validator <recipient> 1000upaw \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --yes

# Verify transaction success
pawd query tx <TX_HASH>
```

## Rollback Scenarios

### Scenario 1: Chain Halted During Upgrade

**Symptoms:**
- Chain stops producing blocks at upgrade height
- Validators cannot sync

**Rollback Steps:**
1. Stop all validators
2. Revert to pre-upgrade binary
3. Use state export from height before upgrade
4. Coordinate restart

**Expected Time:** 30-60 minutes

### Scenario 2: Consensus Failure After Upgrade

**Symptoms:**
- Chain produces blocks but validators can't agree
- Fork detection

**Rollback Steps:**
1. Identify fork point
2. Stop all validators
3. Revert to pre-upgrade binary
4. Reset to last common block
5. Coordinate restart

**Expected Time:** 1-2 hours

### Scenario 3: State Corruption

**Symptoms:**
- Invalid state errors
- Module state inconsistencies
- Invariant violations

**Rollback Steps:**
1. Stop all validators
2. Revert to pre-upgrade binary
3. Use database backup or state export
4. Verify state integrity
5. Coordinate restart

**Expected Time:** 1-3 hours

## Emergency Procedures

### If Majority of Validators Lost Backups

```bash
# Identify validators with valid backups
# Coordinate state sharing

# Option 1: Share state export
# Validator with backup:
gzip pre-upgrade-state.json
scp pre-upgrade-state.json.gz <other-validator>:/tmp/

# Option 2: Share database snapshot
tar -czf paw-db-backup.tar.gz ~/.paw/data
# Upload to shared storage
```

### If Rollback Fails

```bash
# Last resort: Hard reset to genesis
pawd tendermint unsafe-reset-all

# Rebuild from genesis
# This requires full chain replay - NOT RECOMMENDED
```

## Post-Rollback Actions

### 1. Root Cause Analysis

```bash
# Collect logs from failed upgrade
journalctl -u pawd --since "1 hour ago" > failed-upgrade-logs.txt

# Collect state exports
# Share with development team for analysis
```

### 2. Communication

- Announce successful rollback to community
- Explain reason for rollback
- Provide timeline for fix and retry

### 3. Planning Next Upgrade

- Fix identified issues
- Additional testing on testnet
- Coordinate new upgrade proposal

## Preventive Measures

To minimize rollback risk:

1. **Comprehensive Testing**
   - Test on local network
   - Test on public testnet
   - Perform load testing
   - Run upgrade simulations

2. **Staged Rollout**
   - Upgrade testnet first
   - Wait for community feedback
   - Monitor for issues

3. **Backup Strategy**
   - Automated state exports before upgrade
   - Database snapshots
   - Multiple backup locations
   - Regular backup verification

4. **Monitoring**
   - Real-time validator monitoring
   - Alert system for consensus issues
   - Performance metrics tracking

5. **Communication**
   - Clear upgrade documentation
   - Validator coordination channel
   - Emergency response plan

## Rollback Checklist

### Pre-Rollback
- [ ] Achieve validator consensus (67%+)
- [ ] Establish coordination channel
- [ ] Verify all validators have backups
- [ ] Document rollback reason
- [ ] Set coordinated rollback time
- [ ] Prepare communication for community

### During Rollback
- [ ] Stop all validators simultaneously
- [ ] Backup failed state for analysis
- [ ] Revert to previous binary
- [ ] Restore pre-upgrade state
- [ ] Verify state consistency
- [ ] Coordinate chain restart

### Post-Rollback
- [ ] Verify chain producing blocks
- [ ] Verify validator participation
- [ ] Test basic operations
- [ ] Export and verify state
- [ ] Communicate status to community
- [ ] Begin root cause analysis
- [ ] Plan next upgrade attempt

## Support Contacts

For rollback assistance:

- **Emergency Validator Channel**: [Discord/Telegram Link]
- **Core Development Team**: dev@paw-chain.org
- **Security Team**: security@paw-chain.org

## Additional Resources

- [Cosmos Hub Upgrade Rollback Guide](https://hub.cosmos.network/main/governance/proposals.html)
- [Tendermint Rollback Procedures](https://docs.tendermint.com/master/tendermint-core/state-sync.html)
- [Validator Operations Guide](../guides/deployment/VALIDATOR_SETUP.md)

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-11-25 | Initial rollback procedures |

---

**IMPORTANT:** This document should be reviewed and updated after each upgrade attempt, whether successful or not, to incorporate lessons learned.

**Last Updated:** 2025-11-25
**Authors:** PAW Core Development Team
