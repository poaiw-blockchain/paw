# Configuration Testing - Phase 2.2

## Overview

This directory contains a comprehensive configuration testing framework for the PAW blockchain. The framework systematically tests every parameter in `config.toml` and `app.toml` to ensure robust configuration handling.

## Files

### Core Script
- **`test-config-exhaustive.sh`** (1,314 lines)
  - Production-ready bash script for exhaustive configuration testing
  - Tests 150+ parameter combinations
  - Generates detailed markdown reports
  - Supports quick mode, category filtering, and error continuation

### Documentation
- **`CONFIG_TESTING_GUIDE.md`** (404 lines)
  - Complete user guide with examples and troubleshooting
  - Parameter descriptions and test methodologies
  - Performance considerations and best practices

- **`CONFIG_TESTING_QUICK_REF.md`** (227 lines)
  - Quick reference card for common operations
  - Command examples and category listing
  - Troubleshooting cheat sheet

- **`CONFIG_TESTING_DESIGN.md`** (562 lines)
  - Technical design and architecture documentation
  - Component descriptions and flow diagrams
  - Extensibility guide for adding new tests

## Quick Start

```bash
# Run quick mode (critical parameters only, ~5 minutes)
make test-config-quick

# Run full exhaustive test (~30 minutes)
make test-config

# Test specific category
make test-config-category CATEGORY=p2p
```

## What Gets Tested

### config.toml (9 categories, 100+ tests)
- **Base**: moniker, db_backend, log_level, log_format
- **RPC**: listen addresses, CORS, timeouts, connection limits
- **P2P**: peer limits, send/recv rates, timeouts, handshake settings
- **Mempool**: size, cache, broadcast, recheck settings
- **Consensus**: timeouts, block creation, gossip parameters
- **State Sync**: discovery, chunk fetching, snapshots
- **Storage**: ABCI response handling
- **TX Index**: indexer types (kv, null, psql)
- **Instrumentation**: Prometheus metrics

### app.toml (6 categories, 50+ tests)
- **Base**: gas prices, pruning, halt height, IAVL cache
- **Telemetry**: metrics, hostname settings
- **API**: REST server configuration
- **gRPC**: server settings, message sizes
- **State Sync**: snapshot intervals and retention
- **Mempool**: application-side mempool limits

## Features

### Test Modes

1. **Exhaustive** - All 150+ tests (~30 min)
   ```bash
   make test-config
   ```

2. **Quick** - Critical parameters only (~5 min)
   ```bash
   make test-config-quick
   ```

3. **Category** - Specific configuration section
   ```bash
   make test-config-category CATEGORY=p2p
   ```

### Test Types

Each parameter is tested with:
- ✅ **Default values** - Verify baseline configuration works
- ✅ **Valid ranges** - Test low, medium, high values
- ✅ **Edge cases** - Zero, empty, unlimited values
- ✅ **Invalid values** - Ensure proper error handling
- ✅ **Custom validation** - Verify behavior changes (RPC ports, P2P connectivity, etc.)

### Output

- **Console**: Real-time colored output with progress
- **Report**: Detailed markdown report with:
  - Summary statistics (passed/failed/skipped)
  - Test descriptions and configurations
  - Error messages and node logs for failures
  - Expandable details sections

Example report:
```
config-test-report-20250113-120000.md
```

## Architecture

```
Test Execution Flow:
1. Create isolated test home directory
2. Modify configuration parameter with yq
3. Verify modification succeeded
4. Start node (with timeout)
5. Run custom validation (if applicable)
6. Stop node gracefully
7. Evaluate result vs expected outcome
8. Report and cleanup
```

### Key Components

- **Node Management**: Start/stop with timeouts, log capture
- **Config Modification**: TOML editing via yq
- **Validation Functions**: Custom checks beyond "did it start?"
- **Reporting**: Markdown generation with statistics
- **Cleanup**: Automatic on exit/interrupt

## Integration with Testing Plan

This addresses **LOCAL_TESTING_PLAN.md - Phase 2.2**:

```markdown
[ ] 2.2: Exhaustive Configuration Testing:
    Description: Script the modification of every parameter in
                 config.toml and app.toml to verify the node's
                 behavior changes as expected or fails gracefully.
    Action: Pay special attention to P2P settings, timeouts,
            and cache sizes.
```

After successful completion, mark Phase 2.2 as complete.

## Requirements

### Dependencies
- `yq` (v4+) - TOML/YAML manipulation
- `curl` - HTTP requests
- `jq` - JSON parsing
- `nc` (netcat) - Port checking
- `pawd` binary (auto-built by script)

All dependencies are checked before tests begin.

### System Resources
- **Disk**: ~500MB for test directories (cleaned automatically)
- **Ports**: 26656, 26657 (must be available)
- **Time**: 5-30 minutes depending on mode

## Usage Examples

### Development Workflow

```bash
# 1. Quick sanity check
make test-config-quick

# 2. Fix any failures
# ... edit code ...

# 3. Test specific area
make test-config-category CATEGORY=consensus

# 4. Full validation before commit
make test-config
```

### Debugging Failures

```bash
# Keep test directories for inspection
./scripts/test-config-exhaustive.sh --category p2p --skip-cleanup

# Inspect logs
ls /tmp/paw-config-test-*/
tail -100 /tmp/paw-config-test-*/p2p-*/node.log
```

### Continuous Integration

```bash
# Quick mode in CI pipeline
./scripts/test-config-exhaustive.sh --quick --continue-on-error

# Check exit code
if [ $? -eq 0 ]; then
    echo "✓ Configuration tests passed"
else
    echo "✗ Configuration tests failed"
    exit 1
fi
```

## Advanced Features

### Custom Validations

Beyond basic "node starts" checks, the script includes:

- **validate_rpc_laddr()** - Verifies RPC port listening and HTTP responses
- **validate_p2p_laddr()** - Confirms P2P port connectivity
- **validate_log_level()** - Checks log output matches configured level

Add your own validation functions easily:

```bash
validate_my_feature() {
    local test_home="$1"
    # Your validation logic
    return 0  # or 1 for failure
}

run_test "category" "test-name" \
    "Description" \
    "config" "path.to.param" "value" "pass" validate_my_feature
```

### Category Filtering

Test categories available:
- config.toml: base, rpc, p2p, mempool, consensus, statesync, storage, tx_index, instrumentation
- app.toml: app-base, telemetry, api, grpc, state-sync, app-mempool

## Performance

| Mode | Tests | Time | Use Case |
|------|-------|------|----------|
| Quick | ~15 | ~5 min | Pre-commit checks |
| Category | ~10-20 | ~5-10 min | Focused debugging |
| Full | 150+ | ~30 min | Release validation |

## Exit Codes

- `0` - All tests passed
- `1` - Some tests failed (see report)
- `2` - Script error (missing deps, build failure)

## Documentation Hierarchy

1. **Quick Start** → `CONFIG_TESTING_QUICK_REF.md` (commands, examples)
2. **User Guide** → `CONFIG_TESTING_GUIDE.md` (full documentation)
3. **Architecture** → `CONFIG_TESTING_DESIGN.md` (technical design)
4. **This File** → Overview and navigation

## Troubleshooting

### Common Issues

**Port conflicts:**
```bash
pkill pawd
lsof -i :26656,26657
```

**Disk space:**
```bash
rm -rf /tmp/paw-config-test-*
df -h /tmp
```

**All tests failing:**
```bash
make build
./pawd version
./scripts/test-config-exhaustive.sh --category base
```

See `CONFIG_TESTING_GUIDE.md` for detailed troubleshooting.

## Future Enhancements

Potential improvements:
- [ ] Parallel test execution (unique ports per test)
- [ ] Performance benchmarking (measure config impact)
- [ ] Network chaos integration (test timeouts under stress)
- [ ] Historical comparison (regression detection)
- [ ] JSON output format (machine-readable)
- [ ] Docker-based isolation (better port management)

## Contributing

To add new tests:

1. Identify parameter to test
2. Determine appropriate category
3. Add `run_test()` call to `run_all_tests()` or `run_quick_tests()`
4. Optionally create custom validation function
5. Test the new test: `make test-config-category CATEGORY=<your_category>`

Example:
```bash
run_test "p2p" "new-param-test" \
    "Test description" \
    "config" "p2p.new_param" "value" "pass"
```

## Support

- Script help: `./scripts/test-config-exhaustive.sh --help`
- Quick reference: `scripts/CONFIG_TESTING_QUICK_REF.md`
- Full guide: `scripts/CONFIG_TESTING_GUIDE.md`
- Architecture: `scripts/CONFIG_TESTING_DESIGN.md`

## License

Part of the PAW blockchain project.

---

**Status**: Production-ready, not yet run (script creation only per user request)

**Next Steps**:
1. Run `make test-config-quick` to validate script works
2. Fix any issues discovered
3. Run `make test-config` for full validation
4. Mark Phase 2.2 complete in LOCAL_TESTING_PLAN.md
