# Phase 3 Testing - Quick Reference Card

## One-Line Commands

```bash
# Run all Phase 3 tests
./scripts/test-multinode.sh

# Run specific phase
./scripts/test-multinode.sh 3.1    # Baseline
./scripts/test-multinode.sh 3.2    # Consensus
sudo ./scripts/test-multinode.sh 3.3    # Network conditions (needs sudo)
./scripts/test-multinode.sh 3.4    # Malicious peers

# Cleanup
./scripts/test-multinode.sh cleanup
```

## What Each Phase Tests

### 3.1 - Devnet Baseline (5-10 min)
✓ 4 nodes start and connect
✓ Blocks are produced
✓ APIs respond
✓ Validators are active

### 3.2 - Consensus Liveness (10-15 min)
✓ 4 nodes: consensus works (100%)
✓ 3 nodes: consensus works (75% > 67%)
✓ 2 nodes: consensus HALTS (50% < 67%)
✓ Recovery when node returns

### 3.3 - Network Conditions (20-30 min, needs sudo)
✓ High latency (500ms)
✓ Packet loss (up to 15%)
✓ Bandwidth limits (750kbit-1mbit)
✓ Jitter and unstable networks
✓ Recovery from poor conditions

### 3.4 - Malicious Peers (15-20 min)
✓ Invalid transactions rejected
✓ Spam attacks handled
✓ Oversized messages rejected
✓ Peer reputation tracking
✓ Network resilience maintained

## Expected Output

### Success
```
[PASS] ✓ Phase 3.1: 4-Node Devnet Baseline
[PASS] ✓ Phase 3.2: Consensus Liveness & Halt
[PASS] ✓ Phase 3.3: Network Variable Latency/Bandwidth
[PASS] ✓ Phase 3.4: Malicious Peer Ejection

Total Tests: 4
Passed: 4
Failed: 0
```

### Reports Location
```
test-reports/phase3/multinode_test_YYYYMMDD_HHMMSS.txt
```

## Common Issues

**Ports already in use**
```bash
docker compose -f compose/docker-compose.devnet.yml down -v
```

**Phase 3.3 permission denied**
```bash
sudo ./scripts/test-multinode.sh 3.3
```

**Network won't reset**
```bash
sudo ./scripts/test-multinode.sh cleanup
```

## Resource Requirements

- **CPU**: 4 cores recommended
- **RAM**: 4GB minimum (8GB recommended)
- **Disk**: 10GB free space
- **Ports**: 26657, 26667, 26677, 26687, 1317, 1327, 1337, 1347, 39090-39093
- **Time**: 50-75 minutes for full suite

## Prerequisites Check

```bash
# Verify tools installed
docker --version        # Required
docker compose version  # Required
jq --version           # Required
curl --version         # Required
tc -Version            # Required for Phase 3.3
grpcurl --version      # Optional

# Check ports available
sudo netstat -tlnp | grep -E '26657|26667|26677|26687'  # Should be empty
```

## Environment Variables

```bash
# Skip rebuild
SKIP_BUILD=1 ./scripts/test-multinode.sh

# Keep containers running (debugging)
KEEP_RUNNING=1 ./scripts/test-multinode.sh

# Verbose output
VERBOSE=1 ./scripts/test-multinode.sh
```

## Manual Testing

```bash
# Start network
docker compose -f compose/docker-compose.devnet.yml up -d

# Check node status
curl http://localhost:26657/status | jq .result.sync_info

# Check peers
curl http://localhost:26657/net_info | jq .result.n_peers

# Stop containers
docker compose -f compose/docker-compose.devnet.yml down -v
```

## Keyboard Shortcuts During Tests

- `Ctrl+C` - Stop current test (cleanup will run automatically)
- Watch logs: `docker compose -f compose/docker-compose.devnet.yml logs -f`

## Timeline

| Time | Phase | Status |
|------|-------|--------|
| 0-10 min | 3.1 Baseline | 6 tests |
| 10-25 min | 3.2 Consensus | 5 tests |
| 25-55 min | 3.3 Network | 9 tests |
| 55-75 min | 3.4 Malicious | 7 tests |

**Total: ~75 minutes for complete suite**

## Success Checklist

After running all tests, verify:

- [ ] All 4 phases show PASS
- [ ] Report generated in test-reports/phase3/
- [ ] No containers still running (`docker ps`)
- [ ] Network conditions reset (if Phase 3.3 ran)
- [ ] No error messages in final summary

## Next Steps

After Phase 3 passes:
1. Mark Phase 3 complete in LOCAL_TESTING_PLAN.md
2. Proceed to Phase 4: Security & Attack Simulation
3. Archive test reports for documentation
4. Consider running Phase 3 periodically to catch regressions
