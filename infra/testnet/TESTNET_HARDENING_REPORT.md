# PAW Testnet Hardening Report

**Date:** 2026-01-12
**Target:** paw-testnet-1 (4-validator deployment)

## Executive Summary

Comprehensive hardening analysis and fixes for the PAW blockchain testnet. The testnet is currently **HEALTHY** with all 4 validators operational and in sync.

## Current Status

| Validator | Server | Height | Peers | Status |
|-----------|--------|--------|-------|--------|
| val1 | paw-testnet (54.39.103.49) | 45378 | 3 | Synced |
| val2 | paw-testnet (54.39.103.49) | 45378 | 3 | Synced |
| val3 | services-testnet (139.99.149.160) | 45378 | 3 | Synced |
| val4 | services-testnet (139.99.149.160) | 45378 | 3 | Synced |

**Block Production:** ~5 seconds per block (healthy)
**Restarts (7 days):** 0 across all validators
**Resource Usage:** Memory ~280MB per validator (excellent)

## Findings

### No Critical Issues Found

The testnet is currently stable with:
- Zero crashes in the past service lifetime
- Consistent block production
- All validators connected (3 peers each)
- Low resource usage (62GB RAM available, only 3.5GB used)
- No OOM kills in kernel logs

### Historical Issue (Resolved)

**Jan 9, 2026 - CONSENSUS FAILURE**
- WAL file missing during initial testnet startup
- Error: `failed to write msg to consensus WAL due to no such file or directory`
- **Status:** Resolved - WAL files healthy, chain running stably since

### Minor Issues Identified

1. **High P2P Handshake Rejections** (INFO level)
   - External scanners/bots attempting to connect
   - Not affecting validator operation
   - Recommendation: Consider sentry node architecture

2. **Cross-Server Latency** (~212ms)
   - VPN tunnel between paw-testnet and services-testnet
   - Consensus timeouts adjusted to accommodate

3. **Prometheus Disabled**
   - Metrics not being collected
   - Fixed in hardened configuration

4. **WAL File Accumulation**
   - 38+ WAL files on val1 (~400MB total)
   - Consider pruning old WAL files periodically

## Hardening Implemented

### 1. Systemd Service (`pawd-val@.service.hardened`)
- Aggressive restart policy (RestartSec=5)
- Security hardening (NoNewPrivileges, ProtectSystem, etc.)
- Resource limits (LimitNOFILE=1048576, MemoryMax=16G)
- CPU priority (Nice=-10)
- Graceful shutdown (TimeoutStopSec=60)

### 2. CometBFT Configuration (`config-hardened.toml`)
- **Consensus:** Increased timeout_prevote/precommit to 1.5s for latency
- **Mempool:** Reduced size to 2000 txs, 512MB max
- **P2P:** Rate limits doubled, handshake timeout extended
- **RPC:** Connection limits reduced to prevent DoS
- **Monitoring:** Prometheus enabled

### 3. App Configuration (`app-hardened.toml`)
- **Pruning:** Custom strategy (keep 100k blocks, prune every 100)
- **Telemetry:** Enabled with chain labels
- **Snapshots:** Every 1000 blocks

### 4. Monitoring Tools
- `health-check-simple.sh` - Quick status check
- `validator-health-monitor.sh` - Continuous monitoring
- `deploy-hardened-config.sh` - Config deployment with backup

## Deployment Instructions

### Apply Hardened Configuration

```bash
# Dry run first
./deploy-hardened-config.sh --dry-run

# Deploy to single validator
./deploy-hardened-config.sh --validator val1

# Deploy to all validators (with 10s delay between)
./deploy-hardened-config.sh
```

### Verify After Deployment

```bash
./health-check-simple.sh
```

### Rollback

Backups are saved to `/tmp/paw-config-backup-TIMESTAMP/`

```bash
# Restore original config
ssh paw-testnet "cp ~/.paw-val1/config/config.toml.bak ~/.paw-val1/config/config.toml"
ssh paw-testnet "sudo systemctl restart pawd-val@1"
```

## Monitoring Checklist

### Daily Checks
- [ ] Run `health-check-simple.sh`
- [ ] Verify height diff < 2 blocks
- [ ] Check peer counts (should be 3 each)

### Weekly Checks
- [ ] Review journalctl logs for errors
- [ ] Check disk usage (`df -h`)
- [ ] Verify pruning is working (`du -sh ~/.paw-val*/data`)

### After Upgrades
- [ ] Monitor height progression for 10 minutes
- [ ] Check for consensus errors in logs
- [ ] Verify all validators back in sync

## Configuration Files Created

| File | Purpose |
|------|---------|
| `paw/infra/systemd/pawd-val@.service.hardened` | Hardened systemd template |
| `paw/infra/testnet/config-hardened.toml` | CometBFT optimizations |
| `paw/infra/testnet/app-hardened.toml` | App layer optimizations |
| `paw/infra/testnet/health-check-simple.sh` | Quick health check |
| `paw/infra/testnet/validator-health-monitor.sh` | Continuous monitoring |
| `paw/infra/testnet/deploy-hardened-config.sh` | Deployment script |

## Mainnet Readiness

### Completed
- [x] Resource limits configured
- [x] Security hardening in systemd
- [x] Consensus timeouts tuned for network
- [x] Pruning strategy defined
- [x] Monitoring scripts created

### Recommended Before Mainnet
- [ ] Enable sentry node architecture
- [ ] Set up external alerting (PagerDuty/Slack)
- [ ] Configure state sync snapshots
- [ ] Set up off-site backups
- [ ] Document disaster recovery procedures
- [ ] Load test with transaction volume
