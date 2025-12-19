# Configuration Testing - Quick Reference

## Quick Start

```bash
# Full exhaustive test (150+ tests, ~30 min)
make test-config

# Quick mode (critical parameters only, ~5 min)
make test-config-quick

# Test specific category
make test-config-category CATEGORY=p2p
make test-config-category CATEGORY=rpc
make test-config-category CATEGORY=consensus
```

## All Available Categories

| Category | Tests | Description |
|----------|-------|-------------|
| `base` | 9 | Core settings (moniker, db, logging) |
| `rpc` | 14 | RPC server configuration |
| `p2p` | 20 | P2P networking settings |
| `mempool` | 12 | Transaction mempool |
| `consensus` | 11 | Consensus engine timeouts |
| `statesync` | 4 | State sync configuration |
| `storage` | 2 | Storage and ABCI responses |
| `tx_index` | 2 | Transaction indexing |
| `instrumentation` | 3 | Prometheus metrics |
| `app-base` | 11 | Application base settings |
| `telemetry` | 3 | App telemetry |
| `api` | 5 | REST API server |
| `grpc` | 3 | gRPC server |
| `state-sync` | 3 | State sync snapshots |
| `app-mempool` | 3 | App-side mempool |

## Common Use Cases

### Test P2P Settings
```bash
./scripts/test-config-exhaustive.sh --category p2p
```

### Test Timeouts (RPC + Consensus + P2P)
```bash
./scripts/test-config-exhaustive.sh --category rpc
./scripts/test-config-exhaustive.sh --category consensus
./scripts/test-config-exhaustive.sh --category p2p
```

### Test Cache Sizes
```bash
./scripts/test-config-exhaustive.sh --category mempool
./scripts/test-config-exhaustive.sh --category app-base  # IAVL cache
```

### Continue on Failures
```bash
./scripts/test-config-exhaustive.sh --continue-on-error
```

### Debug Failed Tests
```bash
./scripts/test-config-exhaustive.sh --skip-cleanup --category p2p
# Check /tmp/paw-config-test-*/p2p-*/
```

## Output Files

After running tests, you'll find:

- **Report**: `config-test-report-YYYYMMDD-HHMMSS.md` (markdown)
- **Test dirs**: `/tmp/paw-config-test-TIMESTAMP/` (if --skip-cleanup)
- **Node logs**: `<test-home>/node.log` for each test

## Parameters Tested

### High-Impact Parameters

These parameters have the most significant effect on node behavior:

**config.toml:**
- `p2p.max_num_inbound_peers` - Connection limits
- `p2p.max_num_outbound_peers` - Outbound connections
- `p2p.send_rate` / `recv_rate` - Network throughput
- `mempool.size` - Transaction capacity
- `mempool.cache_size` - Transaction cache
- `consensus.timeout_propose` - Block proposal timeout
- `consensus.timeout_commit` - Block commit timeout
- `rpc.max_open_connections` - RPC capacity

**app.toml:**
- `minimum-gas-prices` - Transaction acceptance
- `pruning` - State retention strategy
- `iavl-cache-size` - Memory vs performance tradeoff
- `state-sync.snapshot-interval` - Snapshot frequency

### Edge Cases Tested

- Zero values (unlimited where applicable)
- Very small values (1, 10, 100)
- Very large values (10000+)
- Invalid values (negative, wrong type)
- Empty strings
- Invalid formats

## Success Criteria

A test passes when:
- ✅ **Expected "pass"**: Node starts successfully and operates correctly
- ✅ **Expected "fail"**: Node fails to start with invalid configuration
- ✅ **Custom validation**: Specific behavior verified (e.g., port listening)

A test fails when:
- ❌ **Expected "pass"**: Node fails to start or validation fails
- ❌ **Expected "fail"**: Node starts when it should have failed

## Exit Codes

- `0` - All tests passed
- `1` - Some tests failed (check report)
- `2` - Script error (dependencies, build failure)

## Performance Tips

1. **Quick mode first** - Verify basic functionality quickly
2. **Category by category** - Isolate problem areas
3. **Parallel runs** - Use `--category` in separate terminals (different test dirs)
4. **Skip cleanup during debug** - `--skip-cleanup` to inspect failures

## Troubleshooting

### Port Already in Use
```bash
# Kill any running pawd instances
pkill pawd

# Check for processes using default ports
lsof -i :26656,26657
```

### Out of Disk Space
```bash
# Clean old test directories
rm -rf /tmp/paw-config-test-*

# Check space
df -h /tmp
```

### Tests Timing Out
```bash
# Increase timeout in script (edit NODE_STARTUP_TIMEOUT)
# Or check system load
top
```

### All Tests Failing
```bash
# Rebuild binary
make build

# Check binary works
./pawd version

# Run single test
./scripts/test-config-exhaustive.sh --category base
```

## Integration with Test Plan

This addresses **Phase 2.2** of LOCAL_TESTING_PLAN.md:
```
[ ] 2.2: Exhaustive Configuration Testing
    Description: Script the modification of every parameter in config.toml
                 and app.toml to verify the node's behavior changes as
                 expected or fails gracefully.
```

Mark as complete after successful full run: `make test-config`

## Examples

### Full Production Run
```bash
# Build fresh binary
make build

# Run all tests with report
make test-config

# Review report
cat config-test-report-*.md
```

### Debug Specific Failures
```bash
# Run category that's failing
make test-config-category CATEGORY=consensus

# Keep test directories
./scripts/test-config-exhaustive.sh --category consensus --skip-cleanup

# Inspect logs
ls -la /tmp/paw-config-test-*/consensus-*/
tail -100 /tmp/paw-config-test-*/consensus-*/node.log
```

### CI/CD Integration
```bash
# Quick validation in CI
./scripts/test-config-exhaustive.sh --quick --continue-on-error

# Exit code indicates pass/fail
if [ $? -eq 0 ]; then
    echo "Configuration tests passed"
else
    echo "Configuration tests failed - see report"
fi
```

## See Also

- Full documentation: `scripts/CONFIG_TESTING_GUIDE.md`
- Testing plan: `LOCAL_TESTING_PLAN.md`
- Script help: `./scripts/test-config-exhaustive.sh --help`
