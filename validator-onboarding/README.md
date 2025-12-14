# PAW Blockchain - Validator Onboarding Bundle

**Version:** 1.0
**Release Date:** 2025-12-14
**Network:** paw-testnet-1

---

## Welcome!

Thank you for your interest in becoming a PAW blockchain validator! This bundle contains everything you need to set up and operate a production-ready validator.

## What's Included

### Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| **VALIDATOR_ONBOARDING_GUIDE.md** | Complete step-by-step setup guide | New validators |
| **VALIDATOR_QUICK_START.md** | One-page reference for experienced operators | Cosmos validators |
| **VALIDATOR_HARDWARE_REQUIREMENTS.md** | Detailed specs, cloud recommendations, costs | Infrastructure team |
| **VALIDATOR_ECONOMICS.md** | Staking, rewards, commission mechanics | All validators |
| **VALIDATOR_KEY_MANAGEMENT.md** | Comprehensive key security guide | All validators |
| **VALIDATOR_OPERATOR_GUIDE.md** | Day-to-day operations manual | Validator operators |
| **VALIDATOR_MONITORING.md** | Monitoring setup with Prometheus/Grafana | DevOps engineers |
| **VALIDATOR_SECURITY.md** | Security best practices and hardening | Security engineers |
| **SENTRY_ARCHITECTURE.md** | Sentry node setup for DDoS protection | Infrastructure team |

All documentation is in `/docs/` directory.

### Network Artifacts

Located in `/networks/paw-testnet-1/`:

- `genesis.json` - Official testnet genesis file
- `genesis.sha256` - Checksum for verification
- `peers.txt` - Persistent peers list
- `validator-config.toml.template` - CometBFT config template
- `validator-app.toml.template` - Application config template

### Scripts

Located in `/scripts/`:

- `register-validator.sh` - Interactive validator registration
- `faucet.sh` - Request testnet tokens

### Configuration Templates

Pre-configured settings for validators:

- `validator-config.toml.template` - CometBFT settings (P2P, consensus)
- `validator-app.toml.template` - Application settings (pruning, API)

---

## Quick Start (5 Steps)

### 1. Review Hardware Requirements

```bash
# Read hardware requirements
cat docs/VALIDATOR_HARDWARE_REQUIREMENTS.md

# Minimum for testnet:
# - 4 cores, 16 GB RAM, 200 GB NVMe SSD
# - 100 Mbps network
# - Ubuntu 22.04 LTS
```

### 2. Follow Onboarding Guide

```bash
# Complete setup guide
cat docs/VALIDATOR_ONBOARDING_GUIDE.md

# Or for experienced operators:
cat docs/VALIDATOR_QUICK_START.md
```

### 3. Use Network Artifacts

```bash
# Download genesis
curl -L https://raw.githubusercontent.com/decristofaroj/paw/main/networks/paw-testnet-1/genesis.json \
  > ~/.paw/config/genesis.json

# Verify checksum
sha256sum ~/.paw/config/genesis.json
# Compare with networks/paw-testnet-1/genesis.sha256

# Get persistent peers
cat networks/paw-testnet-1/peers.txt
```

### 4. Apply Configuration Templates

```bash
# Copy CometBFT config template
cp networks/paw-testnet-1/validator-config.toml.template ~/.paw/config/config.toml

# Edit: Update moniker and persistent_peers
vi ~/.paw/config/config.toml

# Copy application config template
cp networks/paw-testnet-1/validator-app.toml.template ~/.paw/config/app.toml
```

### 5. Register Validator

```bash
# Get testnet tokens from faucet
# Then run interactive registration script:
./scripts/register-validator.sh
```

---

## Documentation Roadmap

### For New Validators (Read in Order)

1. **VALIDATOR_HARDWARE_REQUIREMENTS.md** - Choose your infrastructure
2. **VALIDATOR_ONBOARDING_GUIDE.md** - Complete setup process
3. **VALIDATOR_KEY_MANAGEMENT.md** - Secure your keys
4. **VALIDATOR_ECONOMICS.md** - Understand rewards and commission
5. **VALIDATOR_MONITORING.md** - Setup monitoring
6. **VALIDATOR_SECURITY.md** - Harden security
7. **SENTRY_ARCHITECTURE.md** - (Optional) Advanced DDoS protection

### For Experienced Validators

1. **VALIDATOR_QUICK_START.md** - One-page setup
2. **VALIDATOR_ECONOMICS.md** - PAW-specific economics
3. **VALIDATOR_MONITORING.md** - Prometheus/Grafana setup
4. **VALIDATOR_OPERATOR_GUIDE.md** - Reference manual

---

## Support and Community

### Official Channels

| Platform | Link | Purpose |
|----------|------|---------|
| **Discord** | https://discord.gg/paw-blockchain | Real-time support, announcements |
| **Telegram** | https://t.me/pawvalidators | Validator-specific discussions |
| **Forum** | https://forum.paw.network | Technical discussions |
| **GitHub** | https://github.com/decristofaroj/paw | Code, issues, releases |
| **Twitter** | https://twitter.com/PAWBlockchain | Network status, updates |

### Getting Help

**Before asking:**
1. Check documentation in this bundle
2. Search Discord/Telegram history
3. Review GitHub issues

**When asking, provide:**
- Node version: `pawd version`
- Sync status: `pawd status | jq '.SyncInfo'`
- Error logs: `sudo journalctl -u pawd -n 50`

### Validator Communication

**Subscribe to announcements:**
- GitHub: Watch repository for releases
- Discord: #validator-announcements channel
- Email: validators@paw.network (for critical updates)

**Coordination channels:**
- Discord: #validator-tech (upgrades, issues)
- Telegram: @pawvalidators (real-time coordination)

---

## Network Information

### PAW Testnet-1

```yaml
Chain ID: paw-testnet-1
Genesis Time: 2025-12-13T00:00:00Z
Block Time: ~3 seconds
Consensus: CometBFT (Tendermint)
Denomination: upaw (1 PAW = 1,000,000 upaw)
```

### Public Endpoints

| Service | URL |
|---------|-----|
| RPC | https://rpc.paw-testnet.io |
| REST | https://api.paw-testnet.io |
| gRPC | https://grpc.paw-testnet.io:443 |
| Explorer | https://explorer.paw-testnet.io |
| Faucet | https://faucet.paw-testnet.io |

### Key Dates

| Event | Date | Status |
|-------|------|--------|
| Testnet Launch | 2025-12-13 | ✅ Live |
| Validator Onboarding | 2025-12-14 | ✅ Open |
| Testnet Phase 1 | 2025-12 to 2026-01 | 🔄 Active |
| Mainnet Launch | TBD | 📅 Planned 2026 |

---

## Validator Checklist

### Pre-Registration

- [ ] Hardware provisioned (meets minimum requirements)
- [ ] Ubuntu 22.04 LTS installed
- [ ] Firewall configured (ports 26656 open)
- [ ] Static IP or dynamic DNS configured
- [ ] pawd binary built and installed
- [ ] Node initialized and synced

### Registration

- [ ] Operator key created and backed up
- [ ] Mnemonic phrase secured (paper, safe)
- [ ] Testnet tokens received from faucet
- [ ] Validator created using `register-validator.sh`
- [ ] Verified validator appears in explorer

### Post-Registration

- [ ] Monitoring setup (Prometheus + Grafana)
- [ ] Alerts configured (missed blocks, downtime)
- [ ] Security hardened (firewall, SSH keys, fail2ban)
- [ ] Backup procedures documented
- [ ] Emergency contact list created
- [ ] Joined Discord and Telegram

### Mainnet Preparation

- [ ] HSM/tmkms planned for consensus key
- [ ] Ledger hardware wallet for operator key
- [ ] Sentry architecture designed
- [ ] DDoS protection evaluated
- [ ] Multi-sig considered (for teams)
- [ ] Insurance reviewed (if applicable)

---

## Cost Estimates

### Testnet Validator

```
Infrastructure: $150/month (cloud) or $2,000 (bare metal)
Monitoring: $20/month (self-hosted) or $50/month (Grafana Cloud)
Security: $0 (software keys)
Total: ~$170/month or $2,040/year
```

### Mainnet Validator (Recommended)

```
Infrastructure: $350/month (cloud) or $5,000 (bare metal)
Monitoring: $50/month (professional)
Security: $100/month (HSM amortized)
Backup: $30/month (snapshots)
Total: ~$530/month or $6,360/year
```

**ROI:** See VALIDATOR_ECONOMICS.md for detailed profitability analysis.

---

## Frequently Asked Questions

### Q: How much PAW do I need to become a validator?

**A:** Minimum is 1,000,000 upaw (1 PAW) for testnet. Mainnet will require more to be competitive (estimated 10,000+ PAW for active set).

### Q: What is the difference between validator and delegator?

**A:** Validators run infrastructure and participate in consensus. Delegators stake tokens to validators without running infrastructure. Validators earn commission from delegator rewards.

### Q: Can I run a validator from home?

**A:** Possible but not recommended for mainnet. Testnet is fine for learning. Mainnet validators should use professional hosting (cloud or colocation) for reliability and DDoS protection.

### Q: What happens if my validator has downtime?

**A:** Short downtime (<5% missed blocks) has no penalty. Exceeding downtime threshold results in 0.01% slashing and jailing (must manually unjail).

### Q: How do I update my validator commission?

**A:** Use `pawd tx staking edit-validator --commission-rate <new-rate>`. Limited by `commission-max-rate` and `commission-max-change-rate` set at creation.

### Q: Can I move my validator to new infrastructure?

**A:** Yes, but carefully. See VALIDATOR_OPERATOR_GUIDE.md "Validator Migration" section. Never run duplicate validators (causes double-sign slashing).

---

## Security Warnings

**CRITICAL:**
1. **NEVER share your mnemonic phrase** - Anyone with it can steal your funds
2. **NEVER run duplicate validators** - Causes 5% slashing and permanent tombstoning
3. **NEVER skip backups** - Key loss = permanent fund loss
4. **NEVER expose validator RPC publicly** - Use sentry architecture
5. **ALWAYS use Ledger for mainnet operator keys** - Software keys acceptable for testnet only

**Best Practices:**
- Backup keys in 3+ locations (fireproof safe, bank vault, trusted partner)
- Test recovery procedures quarterly
- Use HSM/tmkms for mainnet consensus keys
- Enable 2FA on all accounts (GitHub, cloud provider, etc.)
- Join validator security channels for threat intelligence

---

## What's Next?

After setting up your validator:

1. **Announce yourself:** Discord #validator-introductions
2. **Create website/identity:** Attract delegators with professional presence
3. **Participate in governance:** Vote on all proposals
4. **Engage community:** Twitter, blog posts, educational content
5. **Optimize performance:** Fine-tune configuration, monitor metrics
6. **Plan for mainnet:** Upgrade infrastructure, implement HSM
7. **Build relationships:** Network with other validators

---

## Changelog

### Version 1.0 (2025-12-14)

- Initial release for paw-testnet-1
- Complete validator onboarding documentation
- Interactive registration script
- Configuration templates
- Network artifacts packaged

---

## Feedback and Contributions

**Found an error?** Open an issue: https://github.com/decristofaroj/paw/issues

**Want to improve docs?** Submit a PR: https://github.com/decristofaroj/paw/pulls

**Questions?** Email: validators@paw.network

---

## License

Documentation: CC BY 4.0
Code and scripts: Apache 2.0

---

**Welcome to the PAW validator community! 🐾**

Together we're building a secure, decentralized blockchain network.

---

**Last Updated:** 2025-12-14
**Maintained by:** PAW Blockchain Validator Operations Team
