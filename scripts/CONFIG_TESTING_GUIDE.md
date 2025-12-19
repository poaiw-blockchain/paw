# Configuration Testing Guide - Phase 2.2

This guide explains how to use the exhaustive configuration testing script for the PAW blockchain.

## Overview

The `test-config-exhaustive.sh` script systematically tests every parameter in `config.toml` and `app.toml` to verify that:

1. The node starts and stops correctly with each configuration
2. Behavior changes as expected or fails gracefully
3. Edge cases and invalid values are handled properly
4. P2P settings, timeouts, and cache sizes work correctly

## Quick Start

```bash
# Run all tests (comprehensive, takes time)
./scripts/test-config-exhaustive.sh

# Run only critical parameters (faster)
./scripts/test-config-exhaustive.sh --quick

# Test only P2P parameters
./scripts/test-config-exhaustive.sh --category p2p

# Continue testing even if some tests fail
./scripts/test-config-exhaustive.sh --continue-on-error

# Keep test directories for manual inspection
./scripts/test-config-exhaustive.sh --skip-cleanup
```

## Command-Line Options

| Option | Description |
|--------|-------------|
| `--quick` | Run only critical parameters (faster, ~15 tests vs ~150+) |
| `--category <name>` | Test only specific category (see categories below) |
| `--continue-on-error` | Don't stop on first failure |
| `--skip-cleanup` | Keep test directories for manual inspection |
| `--report <file>` | Custom report output path |
| `-h`, `--help` | Show usage information |

## Test Categories

The script tests parameters organized into the following categories:

### config.toml Categories

1. **base** - Core node settings (moniker, db_backend, log_level, log_format)
2. **rpc** - RPC server configuration (listen address, CORS, timeouts, limits)
3. **p2p** - Peer-to-peer networking (peers, connections, rates, timeouts)
4. **mempool** - Transaction mempool settings (size, cache, broadcast)
5. **consensus** - Consensus engine parameters (timeouts, block creation)
6. **statesync** - State synchronization settings (snapshots, discovery)
7. **storage** - Storage options (ABCI response retention)
8. **tx_index** - Transaction indexer configuration
9. **instrumentation** - Metrics and monitoring (Prometheus)

### app.toml Categories

10. **app-base** - Application base settings (gas prices, pruning, IAVL cache)
11. **telemetry** - Application telemetry configuration
12. **api** - REST API server settings
13. **grpc** - gRPC server configuration
14. **state-sync** - State sync snapshots
15. **app-mempool** - Application-side mempool

## Output and Reports

### Console Output

The script provides real-time colored output:
- 🔵 Blue: Informational messages
- ✅ Green: Successful tests
- ❌ Red: Failed tests
- ⚠️ Yellow: Warnings and skipped tests

### Markdown Report

After completion, a detailed markdown report is generated:

```
config-test-report-YYYYMMDD-HHMMSS.md
```

The report includes:
- Summary statistics
- Detailed results for each test
- Error messages and node logs for failed tests
- Configuration values tested

Example report structure:

```markdown
# PAW Configuration Testing Report

## Summary
- Total Tests: 150
- Passed: 145
- Failed: 3
- Skipped: 2
- Blocked: 0

## Test Results

### ✅ PASSED base/log-level-info
- Description: Test log level: info
- Config: `config.toml :: log_level = "info"`

### ❌ FAILED p2p/send-rate-invalid
- Description: Test invalid send rate
- Config: `config.toml :: p2p.send_rate = -1`
- Error: Node failed to start

<details><summary>Node Logs</summary>
[Error logs here]
</details>
```

## Test Methodology

Each test follows this workflow:

1. **Initialize** - Create isolated test home directory
2. **Modify** - Set configuration parameter to test value
3. **Verify** - Confirm parameter was set correctly
4. **Start** - Attempt to start the node
5. **Validate** - Run custom validation (if applicable)
6. **Stop** - Gracefully stop the node
7. **Evaluate** - Compare actual vs expected result
8. **Report** - Record results and cleanup

## Expected Results

Tests define one of three expected results:

- **pass** - Node should start successfully and operate correctly
- **fail** - Node should fail to start or reject invalid configuration
- **skip** - Test is not applicable or blocked

## Custom Validations

Some tests include custom validation functions beyond just "node starts":

- **validate_rpc_laddr** - Verifies RPC port is listening and responds
- **validate_p2p_laddr** - Verifies P2P port is listening
- **validate_log_level** - Checks log output matches configured level

## What Gets Tested

### config.toml Parameters

#### Base Configuration
- ✅ moniker (valid, empty)
- ✅ db_backend (goleveldb, invalid)
- ✅ log_level (info, debug, error, invalid)
- ✅ log_format (plain, json)
- ✅ filter_peers (true/false)

#### RPC Server
- ✅ laddr (default, custom port, all interfaces, invalid)
- ✅ unsafe (enabled/disabled)
- ✅ max_open_connections (default, low, high, unlimited)
- ✅ max_subscription_clients (default, high)
- ✅ timeout_broadcast_tx_commit (default, long)
- ✅ max_body_bytes (default, large)
- ✅ CORS settings

#### P2P Networking
- ✅ laddr (default, custom port)
- ✅ max_num_inbound_peers (default, low, high, zero)
- ✅ max_num_outbound_peers (default, high)
- ✅ send_rate (default, low, high)
- ✅ recv_rate (default, low)
- ✅ pex (enabled/disabled)
- ✅ seed_mode (enabled/disabled)
- ✅ allow_duplicate_ip (enabled/disabled)
- ✅ handshake_timeout (default, short, long)
- ✅ dial_timeout (default, short)
- ✅ flush_throttle_timeout (default, low)
- ✅ max_packet_msg_payload_size (default, large)

#### Mempool
- ✅ type (flood, nop)
- ✅ recheck (enabled/disabled)
- ✅ broadcast (enabled/disabled)
- ✅ size (default, small, large)
- ✅ max_txs_bytes (default, small)
- ✅ cache_size (default, small, large)
- ✅ max_tx_bytes (default, small)

#### Consensus
- ✅ timeout_propose (default, short, long)
- ✅ timeout_commit (default, zero)
- ✅ skip_timeout_commit (enabled/disabled)
- ✅ create_empty_blocks (enabled/disabled)
- ✅ create_empty_blocks_interval (default, custom)
- ✅ peer_gossip_sleep_duration (default, short)

#### State Sync
- ✅ enable (enabled/disabled)
- ✅ discovery_time (default, short)
- ✅ chunk_request_timeout (default)
- ✅ chunk_fetchers (default, many)

#### Storage
- ✅ discard_abci_responses (enabled/disabled)

#### Transaction Indexer
- ✅ indexer (kv, null)

#### Instrumentation
- ✅ prometheus (enabled/disabled)
- ✅ prometheus_listen_addr (default, custom)

### app.toml Parameters

#### Base Application
- ✅ minimum-gas-prices (default, zero, high)
- ✅ pruning (default, nothing, everything, custom)
- ✅ halt-height (zero, custom)
- ✅ inter-block-cache (enabled/disabled)
- ✅ iavl-cache-size (default, small, large)
- ✅ iavl-disable-fastnode (enabled/disabled)

#### Telemetry
- ✅ enabled (enabled/disabled)
- ✅ enable-hostname (enabled/disabled)
- ✅ enable-hostname-label (enabled/disabled)

#### API
- ✅ enable (enabled/disabled)
- ✅ swagger (enabled/disabled)
- ✅ max-open-connections (default, low)
- ✅ rpc-read-timeout (default, long)

#### gRPC
- ✅ enable (enabled/disabled)
- ✅ max-recv-msg-size (default, large)

#### State Sync
- ✅ snapshot-interval (disabled, enabled)
- ✅ snapshot-keep-recent (default, many)

#### Mempool
- ✅ max-txs (disabled, unlimited, limited)

## Performance Considerations

- **Full test suite**: ~150+ tests, ~15-30 minutes (depends on node startup time)
- **Quick mode**: ~15 tests, ~3-5 minutes
- **Per-test overhead**: ~5-10 seconds (init + start + validate + stop)

Each test runs in isolation with its own home directory, so tests don't interfere with each other.

## Troubleshooting

### Test Failures

If tests fail unexpectedly:

1. **Check the report** - Review error messages and node logs
2. **Run with --skip-cleanup** - Inspect test directories manually
3. **Test single category** - Isolate the problem area
4. **Check system resources** - Ensure enough disk, memory, ports available

### Common Issues

**Port conflicts:**
```bash
# Check if ports are already in use
netstat -tuln | grep -E '26656|26657'

# Stop conflicting processes
pkill pawd
```

**Insufficient resources:**
```bash
# Check available disk space
df -h /tmp

# Check available memory
free -h
```

**Permission issues:**
```bash
# Ensure script is executable
chmod +x scripts/test-config-exhaustive.sh

# Ensure /tmp is writable
ls -ld /tmp
```

### Debug Mode

For additional debugging, you can:

1. Keep test directories: `--skip-cleanup`
2. Run single test by modifying script temporarily
3. Check individual test logs in `/tmp/paw-config-test-*/*/node.log`

## Integration with Testing Plan

This script addresses **Phase 2.2** of the LOCAL_TESTING_PLAN.md:

```markdown
*   **[ ] 2.2: Exhaustive Configuration Testing:**
    *   **Description:** Script the modification of every parameter in
        `config.toml` and `app.toml` to verify the node's behavior changes
        as expected or fails gracefully.
    *   **Action:** Pay special attention to P2P settings, timeouts, and
        cache sizes.
```

After running this script successfully, you can mark Phase 2.2 as complete.

## Advanced Usage

### Adding New Tests

To add tests for new parameters:

1. Add a `run_test` call in the appropriate section of `run_all_tests()`
2. Specify category, test name, description, config file, key path, value, and expected result
3. Optionally provide a custom validation function

Example:

```bash
run_test "p2p" "new-parameter-test" \
    "Test description here" \
    "config" "p2p.new_parameter" "test_value" "pass" validate_new_parameter
```

### Custom Validation Functions

Create validation functions that take `test_home` as an argument:

```bash
validate_new_parameter() {
    local test_home="$1"

    # Your validation logic here
    if [[ condition ]]; then
        echo "Validation failed: reason"
        return 1
    fi

    return 0
}
```

## Dependencies

Required commands:
- `docker` - For container management (if needed)
- `curl` - For HTTP requests to RPC
- `jq` - For JSON parsing
- `yq` - For TOML manipulation (v4+ required)
- `nc` (netcat) - For port checking
- `pawd` - PAW binary (auto-built by script)

All dependencies are checked before tests begin.

## Exit Codes

- `0` - All tests passed
- `1` - Some tests failed (see report)
- `2` - Script error (missing dependencies, build failure, etc.)

## Best Practices

1. **Run quick mode first** - Verify critical parameters work
2. **Fix issues before full run** - Don't waste time on known problems
3. **Review report carefully** - Some "failures" may be expected behavior
4. **Keep reports** - Useful for comparing across versions
5. **Run in CI** - Automate configuration testing on each build

## Future Enhancements

Potential improvements to consider:

- [ ] Parallel test execution (careful with port conflicts)
- [ ] Docker-based isolation (more reliable port management)
- [ ] Network chaos testing integration (simulate poor connections)
- [ ] Performance benchmarking (measure impact of config changes)
- [ ] Historical report comparison (regression detection)
- [ ] JSON report format (machine-readable)
- [ ] Integration with Grafana (visualize test trends)

## Support

For issues or questions:
1. Check this guide and the script's `--help` output
2. Review the generated report for detailed error information
3. Inspect test directories with `--skip-cleanup`
4. Refer to LOCAL_TESTING_PLAN.md for context

## License

Part of the PAW blockchain project.
