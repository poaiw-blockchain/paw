# Phase 3: Multi-Node Network & Consensus Testing

This directory contains comprehensive testing scripts for Phase 3 of the PAW blockchain local testing plan. These scripts test multi-node network functionality, consensus behavior, network resilience, and security.

## Overview

Phase 3 testing covers:
- **Phase 3.1**: 4-Node Devnet Baseline
- **Phase 3.2**: Consensus Liveness & Halt
- **Phase 3.3**: Network Variable Latency/Bandwidth
- **Phase 3.4**: Malicious Peer Ejection

## Prerequisites

### Required Tools
```bash
# Docker and Docker Compose
docker --version
docker compose version

# jq for JSON parsing
jq --version

# curl for API testing
curl --version

# tc (traffic control) for network simulation (Phase 3.3 only)
tc -Version

# grpcurl for gRPC testing (optional)
grpcurl --version
```

### Permissions
- Phase 3.3 requires **sudo** privileges for `tc` (traffic control) commands
- All other phases can run without sudo

### Network Ports
The 4-node devnet uses the following ports:
- Node 1: RPC 26657, gRPC 39090, API 1317
- Node 2: RPC 26667, gRPC 39091, API 1327
- Node 3: RPC 26677, gRPC 39092, API 1337
- Node 4: RPC 26687, gRPC 39093, API 1347

Ensure these ports are available before running tests.

## Quick Start

### Run All Phase 3 Tests
```bash
# From the project root
cd /home/hudson/blockchain-projects/paw

# Run all Phase 3 tests
./scripts/test-multinode.sh all

# Or simply
./scripts/test-multinode.sh
```

### Run Individual Phases
```bash
# Phase 3.1: 4-Node Devnet Baseline
./scripts/test-multinode.sh 3.1

# Phase 3.2: Consensus Liveness & Halt
./scripts/test-multinode.sh 3.2

# Phase 3.3: Network Conditions (requires sudo)
sudo ./scripts/test-multinode.sh 3.3

# Phase 3.4: Malicious Peer Ejection
./scripts/test-multinode.sh 3.4
```

### Cleanup
```bash
# Clean up all test resources
./scripts/test-multinode.sh cleanup
```

## Detailed Test Descriptions

### Phase 3.1: 4-Node Devnet Baseline

**Script**: `phase3.1-devnet-baseline.sh`

**Tests**:
1. Network Connectivity - Verifies all nodes connect to each other
2. Consensus Progression - Confirms blocks are being produced
3. API Endpoints - Tests REST API on all nodes
4. gRPC Endpoints - Tests gRPC on all nodes
5. Validator Set - Verifies validator configuration
6. Smoke Tests - Basic transaction functionality

**Usage**:
```bash
./scripts/phase3.1-devnet-baseline.sh test
./scripts/phase3.1-devnet-baseline.sh cleanup
```

**Expected Duration**: 5-10 minutes

**Success Criteria**:
- All 4 nodes start successfully
- Nodes discover and connect to peers
- Consensus produces blocks continuously
- All API/gRPC endpoints respond correctly

### Phase 3.2: Consensus Liveness & Halt

**Script**: `phase3.2-consensus-liveness.sh`

**Tests**:
1. 4-Node Consensus - Baseline with all validators
2. 3-Node Consensus - Stop 1 validator (75% voting power, should continue)
3. 2-Node Halt - Stop 2 validators (50% voting power, should halt)
4. Consensus Recovery - Restart validator to restore >2/3 majority
5. Full Network Recovery - Restore all 4 validators

**Usage**:
```bash
./scripts/phase3.2-consensus-liveness.sh test
./scripts/phase3.2-consensus-liveness.sh cleanup
```

**Expected Duration**: 10-15 minutes

**Success Criteria**:
- 4-node network produces blocks normally
- 3-node network continues (75% > 66.67% required)
- 2-node network halts (50% < 66.67% required)
- Network recovers when validator is restored
- All validators sync after recovery

### Phase 3.3: Network Variable Latency/Bandwidth

**Script**: `phase3.3-network-conditions.sh`

**Tests**:
1. Baseline - No network impairment
2. High Latency - 500ms delay
3. Cross-Continent - 300ms latency, 0.5% packet loss
4. Mobile 3G - 100ms latency, 2% loss, 750kbit bandwidth
5. Poor Network - 200ms latency, 5% loss, 1mbit bandwidth
6. Unstable - 100ms ±50ms jitter, 10% packet loss
7. Lossy - 15% packet loss
8. Gradual Degradation - Progressive network quality reduction
9. Network Recovery - Recovery from poor conditions

**Usage**:
```bash
# Requires sudo for tc commands
sudo ./scripts/phase3.3-network-conditions.sh test
sudo ./scripts/phase3.3-network-conditions.sh cleanup
```

**Expected Duration**: 20-30 minutes

**Success Criteria**:
- Consensus remains active under all network conditions
- Block production may slow but doesn't halt
- Network recovers when conditions improve
- No permanent damage from network stress

**Technical Details**:
- Uses `tc` (traffic control) via `~/blockchain-projects/scripts/network-sim.sh`
- Applies network conditions to Docker container veth interfaces
- Monitors block production during each condition
- Automatically resets network conditions after each test

### Phase 3.4: Malicious Peer Ejection

**Script**: `phase3.4-malicious-peer.sh`

**Tests**:
1. Invalid Transaction - Rejects malformed transactions
2. Message Spam - Handles rapid message floods
3. Oversized Message - Rejects messages exceeding size limits
4. Peer Reputation - Tracks peer behavior scores
5. Peer Banning - Automatically bans misbehaving peers
6. Network Resilience - Consensus continues despite attacks
7. Peer Recovery - Misbehaving peer can reconnect after cool-down

**Usage**:
```bash
./scripts/phase3.4-malicious-peer.sh test
./scripts/phase3.4-malicious-peer.sh cleanup
```

**Expected Duration**: 15-20 minutes

**Success Criteria**:
- Invalid transactions are rejected
- Node remains responsive during spam attacks
- Oversized messages are rejected
- Reputation system tracks peer behavior (if enabled)
- Network consensus is unaffected by malicious activity

**Notes**:
- Some tests may show warnings if reputation module is not exposed via API
- Automatic banning behavior depends on configuration thresholds
- Tests verify defensive behavior, not necessarily peer disconnection

## Test Reports

All test runs generate detailed reports in:
```
/home/hudson/blockchain-projects/paw/test-reports/phase3/
```

Report naming convention:
```
multinode_test_YYYYMMDD_HHMMSS.txt
```

Each report includes:
- Timestamp and environment details
- Individual test results (PASS/FAIL)
- Duration for each phase
- Summary statistics
- Detailed logs from all tests

## Environment Variables

Customize test behavior with environment variables:

```bash
# Skip rebuilding pawd binary
SKIP_BUILD=1 ./scripts/test-multinode.sh

# Keep network running after tests (for debugging)
KEEP_RUNNING=1 ./scripts/test-multinode.sh

# Enable verbose output
VERBOSE=1 ./scripts/test-multinode.sh

# Keep containers running after smoke tests
PAW_SMOKE_KEEP_STACK=1 ./scripts/test-multinode.sh
```

## Troubleshooting

### Network Fails to Start

**Symptoms**: Nodes don't become healthy within timeout period

**Solutions**:
```bash
# Check if ports are already in use
sudo netstat -tlnp | grep -E '26657|26667|26677|26687'

# Clean up any existing containers
docker compose -f compose/docker-compose.devnet.yml down -v

# Rebuild and retry
docker compose -f compose/docker-compose.devnet.yml up -d --build
```

### Phase 3.3 Permission Denied

**Symptoms**: `tc` commands fail with permission errors

**Solutions**:
```bash
# Run with sudo
sudo ./scripts/test-multinode.sh 3.3

# Or run the individual script with sudo
sudo ./scripts/phase3.3-network-conditions.sh
```

### Network Conditions Not Resetting

**Symptoms**: Network remains slow after Phase 3.3 tests

**Solutions**:
```bash
# Manual cleanup
sudo ~/blockchain-projects/scripts/network-sim.sh reset paw-node1
sudo ~/blockchain-projects/scripts/network-sim.sh reset paw-node2
sudo ~/blockchain-projects/scripts/network-sim.sh reset paw-node3
sudo ~/blockchain-projects/scripts/network-sim.sh reset paw-node4

# Or run cleanup
sudo ./scripts/test-multinode.sh cleanup
```

### Containers Won't Stop

**Symptoms**: Docker containers remain running after cleanup

**Solutions**:
```bash
# Force remove containers
docker rm -f paw-node1 paw-node2 paw-node3 paw-node4

# Remove volumes
docker volume prune -f
```

### Tests Fail Intermittently

**Symptoms**: Random test failures on repeated runs

**Possible Causes**:
- System resource constraints (CPU, memory, disk)
- Network timing issues
- Docker performance on host system

**Solutions**:
```bash
# Increase test timeouts by editing scripts
# Check system resources
docker stats

# Ensure adequate disk space
df -h

# Check Docker resource limits
docker info | grep -i memory
```

## Integration with LOCAL_TESTING_PLAN.md

These scripts fulfill the Phase 3 requirements in `LOCAL_TESTING_PLAN.md`:

- [x] 3.1: 4-Node Devnet Baseline
- [x] 3.2: Consensus Liveness & Halt (4-node, 3-node, 2-node)
- [x] 3.3: Network Variable Latency/Bandwidth
- [x] 3.4: Malicious Peer Ejection

## Architecture

### Script Hierarchy
```
test-multinode.sh              (Main orchestrator)
├── phase3.1-devnet-baseline.sh
├── phase3.2-consensus-liveness.sh
├── phase3.3-network-conditions.sh
└── phase3.4-malicious-peer.sh
```

### Shared Dependencies
- `compose/docker-compose.devnet.yml` - 4-node network definition
- `scripts/devnet/lib.sh` - Shared helper functions
- `scripts/devnet/init_node.sh` - Node initialization
- `~/blockchain-projects/scripts/network-sim.sh` - Network simulation (Phase 3.3)

### Data Flow
1. Main orchestrator initializes test environment
2. Individual phase scripts execute specific tests
3. Results collected and aggregated
4. Report generated with summary
5. Cleanup performed (unless KEEP_RUNNING=1)

## Development

### Adding New Tests

To add a new test to an existing phase:

1. Edit the appropriate `phase3.X-*.sh` script
2. Add test function following naming convention: `test_<name>()`
3. Call test function in `run_tests()`
4. Add result to `test_results` array
5. Update documentation in this README

### Creating a New Phase

To add Phase 3.5:

1. Create `scripts/phase3.5-new-test.sh`
2. Follow structure of existing phase scripts
3. Implement `run_tests()` and `cleanup()` functions
4. Add to `test-multinode.sh` main switch statement
5. Update this README with phase description

## Performance Benchmarks

Typical execution times on recommended hardware:

| Phase | Duration | Resource Usage |
|-------|----------|----------------|
| 3.1 | 5-10 min | CPU: Low, Memory: 2GB, Disk I/O: Low |
| 3.2 | 10-15 min | CPU: Low, Memory: 2GB, Disk I/O: Low |
| 3.3 | 20-30 min | CPU: Medium, Memory: 2GB, Disk I/O: Low |
| 3.4 | 15-20 min | CPU: Medium, Memory: 2GB, Disk I/O: Medium |
| **Total** | **50-75 min** | **Peak: 4 nodes × ~500MB each** |

## CI/CD Integration

These scripts are designed for local testing but can be adapted for CI/CD:

```yaml
# Example GitHub Actions workflow
name: Phase 3 Tests
on: [push, pull_request]
jobs:
  multinode-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Phase 3.1
        run: ./scripts/test-multinode.sh 3.1
      - name: Run Phase 3.2
        run: ./scripts/test-multinode.sh 3.2
      - name: Upload Reports
        uses: actions/upload-artifact@v3
        with:
          name: phase3-reports
          path: test-reports/phase3/
```

## References

- [Tendermint Consensus](https://docs.tendermint.com/master/spec/consensus/)
- [Cosmos SDK Testing Guide](https://docs.cosmos.network/main/building-modules/testing)
- [Docker Compose Networking](https://docs.docker.com/compose/networking/)
- [Linux Traffic Control (tc)](https://man7.org/linux/man-pages/man8/tc.8.html)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review test reports in `test-reports/phase3/`
3. Examine Docker logs: `docker compose -f compose/docker-compose.devnet.yml logs`
4. Check individual node logs in containers
