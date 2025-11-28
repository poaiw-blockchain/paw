# PAW Mainnet Genesis Launch Checklist

This checklist ensures a smooth and coordinated mainnet launch.

## Pre-Genesis Phase (4-8 weeks before launch)

### Week -8 to -6: Planning & Coordination

- [ ] Finalize genesis time (December 15, 2025, 00:00:00 UTC)
- [ ] Recruit 25 genesis validators
- [ ] Set up validator coordination channel (Discord/Telegram)
- [ ] Publish validator requirements and application process
- [ ] Conduct security audit of genesis parameters
- [ ] Review and finalize tokenomics distribution

### Week -6 to -4: Validator Preparation

- [ ] Genesis validators generate keys on secure hardware
- [ ] Validators submit public keys and validator info
- [ ] Create multi-sig wallets for Foundation, Ecosystem, Reserve
- [ ] Replace placeholder addresses in genesis.json with real addresses
- [ ] Validators set up infrastructure (nodes, monitoring, sentries)
- [ ] Conduct testnet dry run with all genesis validators

### Week -4 to -2: Genesis File Preparation

- [ ] Finalize all module parameters
- [ ] Configure initial DEX pools
- [ ] Set up initial oracle price feeds
- [ ] Update vesting schedules for Team and API Donors
- [ ] All validators verify and approve final genesis parameters
- [ ] Create official genesis.json candidate

### Week -2 to Launch: Final Preparations

- [ ] Validators collect and submit gentx files
- [ ] Coordinator collects all gentxs
- [ ] Run `pawd collect-gentxs`
- [ ] Generate final genesis.json with all validators
- [ ] All validators verify genesis hash
- [ ] Publish genesis hash on official channels
- [ ] Validators sync genesis.json to nodes
- [ ] Final testnet simulation
- [ ] Prepare launch announcement
- [ ] Set up block explorer and monitoring tools

## Genesis Day Checklist

### T-24 Hours

- [ ] All validators confirm readiness
- [ ] Verify all nodes have correct genesis.json
- [ ] Verify all nodes have correct chain-id: `paw-mainnet-1`
- [ ] Check system time synchronization (NTP)
- [ ] Verify firewall and network configuration
- [ ] Confirm monitoring and alerting is active
- [ ] Final hash verification: All validators confirm same hash

### T-6 Hours

- [ ] Validators enter "no-deploy" freeze
- [ ] Final infrastructure checks
- [ ] Coordinate in validator channel
- [ ] Prepare for potential issues
- [ ] Have backup nodes ready

### T-1 Hour

- [ ] All validators online in coordination channel
- [ ] Final go/no-go poll
- [ ] Nodes ready to start (but not started)
- [ ] Monitoring dashboards active
- [ ] Social media and communication channels ready

### T-30 Minutes

- [ ] Validators begin countdown coordination
- [ ] Verify genesis_time in all configs
- [ ] Final system checks
- [ ] Prepare start commands

### T-15 Minutes

- [ ] Begin final countdown
- [ ] All validators report ready status
- [ ] Monitoring team on standby

### T-5 Minutes

- [ ] Validator nodes in standby mode
- [ ] Communication channels active
- [ ] Ready to execute start command

### T-0 (Genesis Time: 2025-12-15T00:00:00Z)

- [ ] **START NODES**: `pawd start`
- [ ] Validators monitor block production
- [ ] Verify blocks are being produced
- [ ] Confirm validator signatures appearing
- [ ] Check consensus participation

### T+5 Minutes

- [ ] Verify >67% of validators are signing
- [ ] Check block time (~4 seconds)
- [ ] Monitor for any errors or issues
- [ ] Confirm all genesis validators are active

### T+30 Minutes

- [ ] Verify continuous block production
- [ ] Check transaction processing
- [ ] Test basic operations (send, query)
- [ ] Monitor network stability
- [ ] Begin public announcement

### T+1 Hour

- [ ] Network stability confirmed
- [ ] All core functionality tested
- [ ] Public RPC endpoints active
- [ ] Block explorer synced and operational
- [ ] Official mainnet launch announcement

## Post-Genesis Phase

### Day 1

- [ ] Monitor validator performance
- [ ] Track uptime and signing rates
- [ ] Verify staking operations work
- [ ] Test governance proposals
- [ ] Verify DEX functionality
- [ ] Check oracle price feeds updating
- [ ] Monitor for any unusual activity
- [ ] Community support active

### Week 1

- [ ] Daily validator coordination calls
- [ ] Monitor inflation and emission
- [ ] Track staking participation
- [ ] First DEX liquidity pools operational
- [ ] Oracle feeds stable and accurate
- [ ] No slashing events (ideal)
- [ ] Community onboarding progressing
- [ ] Documentation feedback collected

### Week 2-4

- [ ] Validator set stable
- [ ] Staking APR normalized
- [ ] DEX volume tracked
- [ ] First governance proposal (if needed)
- [ ] Mobile wallet integration tested
- [ ] Performance metrics documented
- [ ] Security monitoring ongoing
- [ ] Plan first protocol upgrade (if needed)

### Month 2-3

- [ ] Onboard additional validators (up to 125 max)
- [ ] Community governance active
- [ ] DEX ecosystem growing
- [ ] Oracle price feeds expanded
- [ ] Compute module activated
- [ ] First major partnerships announced
- [ ] Consider parameter adjustments via governance

## Validator Node Checklist

### Hardware Requirements

- [ ] 8+ CPU cores (16 recommended)
- [ ] 32GB RAM minimum (64GB recommended)
- [ ] 1TB NVMe SSD (high IOPS required)
- [ ] 1 Gbps network connection
- [ ] Uninterruptible Power Supply (UPS)
- [ ] Redundant internet connections

### Software Setup

- [ ] Operating System: Ubuntu 22.04 LTS or later
- [ ] Go 1.23.1 or higher installed
- [ ] `pawd` binary version 1.0.0+ installed
- [ ] Firewall configured (only allow necessary ports)
- [ ] SSH hardened (key-based auth only)
- [ ] Monitoring tools installed (Prometheus, Grafana)
- [ ] Backup and recovery procedures tested

### Security Checklist

- [ ] Validator key stored on HSM or secure enclave
- [ ] Sentry node architecture implemented
- [ ] DDoS protection enabled
- [ ] Rate limiting configured
- [ ] Security patches up to date
- [ ] Intrusion detection system active
- [ ] Backup validators on standby
- [ ] Incident response plan documented

### Monitoring Setup

- [ ] Node health monitoring
- [ ] Block signing monitoring
- [ ] Missed blocks alerting
- [ ] Disk space monitoring
- [ ] Network connectivity monitoring
- [ ] Peer count tracking
- [ ] Memory and CPU usage alerts
- [ ] Slashing condition alerts

## Genesis File Verification Commands

### Verify Genesis Hash
```bash
sha256sum config/genesis-mainnet.json
# Expected: [To be published by coordinator]
```

### Validate Genesis Structure
```bash
pawd validate-genesis ~/.paw/config/genesis.json
```

### Check Chain ID
```bash
jq -r '.chain_id' ~/.paw/config/genesis.json
# Expected: paw-mainnet-1
```

### Verify Total Supply
```bash
jq -r '.app_state.bank.supply[0].amount' ~/.paw/config/genesis.json
# Expected: 50000000000000 (50M PAW)
```

### Check Your Validator in Genesis
```bash
jq '.app_state.genutil.gen_txs[] | select(.body.messages[0].delegator_address=="YOUR_ADDRESS")' ~/.paw/config/genesis.json
```

## Emergency Procedures

### If Network Doesn't Start

1. **Check Consensus**:
   - Verify >67% of validators are online
   - Check validator coordination channel
   - Verify all have same genesis hash

2. **Common Issues**:
   - Wrong genesis.json (verify hash)
   - Incorrect chain-id
   - Time synchronization issues (check NTP)
   - Network connectivity problems

3. **Restart Procedure**:
   ```bash
   # Stop node
   pkill pawd

   # Clear state (if instructed by coordinator)
   pawd unsafe-reset-all

   # Restart with correct genesis
   pawd start
   ```

### If Validator is Missing Blocks

1. **Check Node Status**:
   ```bash
   curl http://localhost:26657/status
   ```

2. **Check Logs**:
   ```bash
   journalctl -u pawd -f
   ```

3. **Verify Signing**:
   ```bash
   pawd query slashing signing-info $(pawd tendermint show-validator)
   ```

4. **Unjail (if jailed)**:
   ```bash
   pawd tx slashing unjail --from validator --chain-id paw-mainnet-1
   ```

### Emergency Contacts

- **Validator Coordination**: [Discord/Telegram Channel]
- **Technical Support**: [ Issues]
- **Security Incidents**: [security@paw.network]
- **24/7 Coordinator**: [Emergency Contact]

## Success Criteria

### Network Health
- [ ] >95% validator uptime
- [ ] 4-second average block time
- [ ] <1% orphaned blocks
- [ ] >67% continuous consensus

### Functionality
- [ ] Transactions processing successfully
- [ ] Staking operations functional
- [ ] Governance proposals can be created
- [ ] DEX swaps executing correctly
- [ ] Oracle prices updating regularly
- [ ] No critical bugs discovered

### Decentralization
- [ ] 25+ active validators
- [ ] No single validator >10% voting power
- [ ] Geographic distribution of validators
- [ ] Multiple client implementations (future)

### Community
- [ ] >1000 wallet addresses
- [ ] >$100k TVL in DEX pools (week 1)
- [ ] Active community channels
- [ ] Documentation complete and accessible

## Post-Launch Monitoring

### Daily Checks (First Month)
- Validator uptime and signing rate
- Block production consistency
- Network transaction volume
- DEX liquidity and volume
- Oracle price feed accuracy
- Governance proposal activity
- Slashing events
- Validator set changes

### Weekly Reports
- Network performance metrics
- Validator participation stats
- Economic metrics (staking APR, inflation)
- DEX analytics
- Security incidents
- Community growth metrics

---

**Contact Information**

- **Genesis Coordinator**: [Name and contact]
- **Technical Lead**: [Name and contact]
- **Security Team**: [Contact]
- **Community Manager**: [Contact]

**Resources**

- Genesis File: `config/genesis-mainnet.json`
- Documentation: `config/genesis-README.md`
- Verification Script: `scripts/verify-genesis.sh`
- Node Config: `config/node-config.toml.template`
- Whitepaper: `PAW Extensive whitepaper.md`

**Official Channels**

- Website: [https://paw.network]
- Discord: [Link]
- Twitter: [@PAWBlockchain]
- Telegram: [Link]

---

**Last Updated**: November 19, 2025
**Version**: 1.0
**Status**: Pre-Genesis
