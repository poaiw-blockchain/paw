# PAW Testnet Documentation Index

Complete index of all multi-validator testnet documentation.

## Quick Start

**Just want to run it?** → [TESTNET_QUICK_REFERENCE.md](TESTNET_QUICK_REFERENCE.md)

**Need full instructions?** → [MULTI_VALIDATOR_TESTNET.md](MULTI_VALIDATOR_TESTNET.md)

## Documentation Overview

### 1. Quick Reference (Start Here)
**File:** [TESTNET_QUICK_REFERENCE.md](TESTNET_QUICK_REFERENCE.md)

One-page cheat sheet with:
- Quick start commands
- Status checks
- Common operations
- Critical rules
- Quick troubleshooting

**Use when:** You know what you're doing and just need command reminders.

### 2. Complete Guide
**File:** [MULTI_VALIDATOR_TESTNET.md](MULTI_VALIDATOR_TESTNET.md)

Comprehensive guide covering:
- Detailed setup instructions
- Network configurations (2, 3, 4 validators)
- Monitoring and verification
- Graceful and emergency shutdown
- Troubleshooting with solutions
- What NOT to do (critical)
- Advanced testing scenarios

**Use when:** First time setup, troubleshooting issues, or learning how it works.

### 3. Script Documentation
**File:** [../scripts/devnet/README.md](../scripts/devnet/README.md)

Technical details about:
- `setup-validators.sh` - How genesis is generated
- `init_node.sh` - How nodes are initialized
- State directory structure
- SDK v0.50.x signing info bug explanation
- Consensus timeout fix explanation

**Use when:** Debugging scripts, modifying genesis process, or understanding internals.

### 4. Fix Summary
**File:** [../CONSENSUS_FIX_SUMMARY.md](../CONSENSUS_FIX_SUMMARY.md)

Complete summary of:
- Original problems identified
- Root causes and evidence
- Solutions implemented
- Testing results (2, 3, 4 validators)
- Technical background
- Success metrics

**Use when:** Understanding what was fixed, why it was broken, or reviewing the solution.

### 5. Sentry Architecture
**File:** [SENTRY_ARCHITECTURE.md](SENTRY_ARCHITECTURE.md)

Production-like testnet with sentry nodes:
- What are sentry nodes and why use them
- Network topology (validators + sentries)
- Complete setup guide with sentries
- Real-world testing scenarios
- Security considerations
- Troubleshooting sentry-specific issues

**Use when:** Setting up production-like testing, learning about validator protection, or simulating real-world network conditions.

### 6. Sentry Testing Guide
**File:** [SENTRY_TESTING_GUIDE.md](SENTRY_TESTING_GUIDE.md)

Automated testing for sentry architecture:
- Three comprehensive test suites (basic, load, chaos)
- Performance benchmarks and metrics
- Chaos engineering scenarios
- Test results interpretation
- CI/CD integration
- Troubleshooting failed tests

**Use when:** Validating sentry setup, performance testing, chaos engineering, or CI/CD integration.

### 7. Operational Dashboards
**File:** [DASHBOARDS_GUIDE.md](DASHBOARDS_GUIDE.md)

Production-ready web dashboards for:
- **Staking Dashboard** (Port 11100) - Validator discovery, delegation management, staking calculator, rewards tracking
- **Validator Dashboard** (Port 11110) - Real-time validator monitoring, uptime tracking, performance metrics
- **Governance Portal** (Port 11120) - Proposal voting, creation, analytics

Features:
- Docker containerized deployment
- Environment-based configuration
- Real-time WebSocket updates
- Comprehensive test coverage (85%+)
- Health monitoring and logging

**Use when:** Managing staking operations, monitoring validators, or participating in governance.

### 8. Main README
**File:** [../README.md](../README.md)

Project overview with:
- Quick links to testnet docs
- Single-node setup
- Build instructions
- Repository structure

**Use when:** Starting with PAW for the first time.

## Common Scenarios

### "I want to run a 4-validator testnet right now"

```bash
# Follow TESTNET_QUICK_REFERENCE.md
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

### "The testnet isn't working"

1. Check [MULTI_VALIDATOR_TESTNET.md - Common Issues](MULTI_VALIDATOR_TESTNET.md#common-issues)
2. Verify you followed the critical rules
3. Check logs: `docker logs paw-node1 2>&1 | grep -i error`

### "I want to understand the technical details"

1. Read [../CONSENSUS_FIX_SUMMARY.md](../CONSENSUS_FIX_SUMMARY.md) for the overview
2. Read [../scripts/devnet/README.md](../scripts/devnet/README.md) for script internals

### "I want to modify the genesis process"

1. Understand current process: [../scripts/devnet/README.md](../scripts/devnet/README.md)
2. Modify `scripts/devnet/setup-validators.sh`
3. Test with: `./scripts/devnet/setup-validators.sh 2`
4. Verify: `jq '.app_state.slashing.signing_infos | length' scripts/devnet/.state/genesis.json`

### "I want production-like testing with sentry nodes"

```bash
# Follow SENTRY_ARCHITECTURE.md
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
curl -s http://localhost:30658/status | jq '.result.sync_info'  # Sentry1
curl -s http://localhost:30668/status | jq '.result.sync_info'  # Sentry2
```

### "I want to validate the sentry architecture with automated tests"

```bash
# Follow SENTRY_TESTING_GUIDE.md
# Network must be running first
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60

# Run all test suites
./scripts/devnet/test-sentry-all.sh

# Or run individual tests
./scripts/devnet/test-sentry-scenarios.sh     # Basic scenarios
./scripts/devnet/test-load-distribution.sh    # Load testing
./scripts/devnet/test-network-chaos.sh        # Chaos engineering
```

### "I want to use the operational dashboards"

```bash
# Deploy all dashboards
./scripts/deploy-dashboards.sh

# Access dashboards
# Staking Dashboard:    http://localhost:11100
# Validator Dashboard:  http://localhost:11110
# Governance Portal:    http://localhost:11120

# Verify health
./scripts/verify-dashboards.sh

# Stop dashboards
./scripts/stop-dashboards.sh
```

## Critical Rules (Memorize These)

From [TESTNET_QUICK_REFERENCE.md](TESTNET_QUICK_REFERENCE.md):

### ✅ DO:
- **ALWAYS** clean before generating new genesis
- **ALWAYS** wait 30 seconds after startup
- **MATCH** genesis validator count with docker-compose file
- **USE** `-v` flag when switching configurations

### ❌ DON'T:
- **NEVER** skip cleaning step
- **NEVER** mix validator counts
- **NEVER** check status immediately after startup
- **NEVER** manually edit genesis after collect-gentxs

## Troubleshooting Flow

```
Problem?
   │
   ├─→ Read Quick Reference troubleshooting section
   │   └─→ Problem solved? → Done!
   │
   ├─→ Read Complete Guide common issues
   │   └─→ Problem solved? → Done!
   │
   ├─→ Check script documentation for technical details
   │   └─→ Problem solved? → Done!
   │
   └─→ Review CONSENSUS_FIX_SUMMARY for root causes
       └─→ Still stuck? → Check logs, create issue
```

## File Locations

```
paw/
├── README.md                                    # Project overview with testnet links
├── CONSENSUS_FIX_SUMMARY.md                     # Complete fix summary
├── docs/
│   ├── TESTNET_DOCUMENTATION_INDEX.md          # This file
│   ├── TESTNET_QUICK_REFERENCE.md              # One-page cheat sheet
│   ├── MULTI_VALIDATOR_TESTNET.md              # Complete guide
│   ├── SENTRY_ARCHITECTURE.md                  # Sentry node guide
│   ├── SENTRY_TESTING_GUIDE.md                 # Automated testing guide
│   └── ...
├── scripts/
│   └── devnet/
│       ├── README.md                            # Script documentation
│       ├── setup-validators.sh                  # Genesis generator
│       ├── init_node.sh                         # Validator initializer
│       ├── init_sentry.sh                       # Sentry initializer
│       ├── test-sentry-scenarios.sh             # Basic sentry tests
│       ├── test-load-distribution.sh            # Load/performance tests
│       ├── test-network-chaos.sh                # Chaos engineering tests
│       ├── test-sentry-all.sh                   # Run all tests
│       └── .state/                              # Generated files (git-ignored)
│           ├── genesis.json
│           ├── node*.priv_validator_key.json
│           ├── node*_validator.mnemonic
│           ├── node*.id                         # Validator node IDs
│           └── sentry*.id                       # Sentry node IDs
└── compose/
    ├── docker-compose.2nodes.yml               # 2-validator config
    ├── docker-compose.3nodes.yml               # 3-validator config
    ├── docker-compose.4nodes.yml               # 4-validator config
    └── docker-compose.4nodes-with-sentries.yml # 4 validators + 2 sentries
```

## Version Information

- **PAW Version:** Latest (Cosmos SDK v0.50.14)
- **Tested Configurations:** 2, 3, and 4 validators (with/without sentries)
- **Sentry Support:** 2 sentry nodes with 4-validator network
- **Status:** Production-ready
- **Last Updated:** 2025-12-14

## Support

If you encounter issues not covered in the documentation:

1. **Check logs:** `docker logs paw-node1 2>&1 | tail -100`
2. **Verify setup:** Follow the troubleshooting checklist in MULTI_VALIDATOR_TESTNET.md
3. **Review fixes:** Read CONSENSUS_FIX_SUMMARY.md to understand what was fixed
4. **Create issue:** With logs and steps to reproduce

## Quick Links

- [Quick Reference](TESTNET_QUICK_REFERENCE.md) - Fastest way to get started
- [Complete Guide](MULTI_VALIDATOR_TESTNET.md) - Full instructions
- [Sentry Architecture](SENTRY_ARCHITECTURE.md) - Production-like testing with sentries
- [Sentry Testing Guide](SENTRY_TESTING_GUIDE.md) - Automated sentry tests
- [Dashboards Guide](DASHBOARDS_GUIDE.md) - Operational dashboards (staking, validator, governance)
- [Script Docs](../scripts/devnet/README.md) - Technical details
- [Fix Summary](../CONSENSUS_FIX_SUMMARY.md) - What was fixed and why
- [Main README](../README.md) - Project overview

---

**Remember:** Clean → Generate → Start → Wait → Verify
