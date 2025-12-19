# Configuration Testing Script - Technical Design

## Architecture Overview

The `test-config-exhaustive.sh` script is a production-ready bash-based testing framework for systematically validating configuration parameters in Cosmos SDK applications.

## Design Principles

1. **Isolation** - Each test runs in its own home directory
2. **Idempotency** - Tests can be run repeatedly without side effects
3. **Fail-Safe** - Automatic cleanup on exit or interrupt
4. **Comprehensive** - Test valid, invalid, and edge case values
5. **Observable** - Detailed logging and reporting
6. **Modular** - Easy to add new tests and validations

## Component Architecture

```
test-config-exhaustive.sh
│
├── Configuration & Constants
│   ├── Paths (test dirs, binary locations)
│   ├── Flags (quick mode, continue-on-error, etc.)
│   └── Statistics (counters for pass/fail/skip)
│
├── Utility Functions
│   ├── Logging (colored output)
│   ├── Dependency checks
│   └── Binary building
│
├── Node Management
│   ├── create_test_home()     - Initialize isolated test environment
│   ├── start_node()            - Launch node with timeout
│   ├── stop_node()             - Graceful/forced shutdown
│   └── get_node_logs()         - Extract logs for debugging
│
├── Configuration Management
│   ├── modify_config_toml()    - Change config.toml parameters
│   ├── modify_app_toml()       - Change app.toml parameters
│   └── get_config_value()      - Read current values
│
├── Test Execution
│   ├── run_test()              - Core test execution logic
│   ├── run_all_tests()         - Full test suite
│   └── run_quick_tests()       - Critical parameters only
│
├── Validation Functions
│   ├── validate_rpc_laddr()    - RPC endpoint verification
│   ├── validate_p2p_laddr()    - P2P port verification
│   └── validate_log_level()    - Log output verification
│
└── Reporting
    ├── init_report()           - Create markdown report
    ├── append_to_report()      - Add test results
    └── update_report_summary() - Update statistics
```

## Test Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Pre-Flight Checks                                        │
│    - Verify dependencies (yq, curl, jq, etc.)              │
│    - Build/locate pawd binary                               │
│    - Create test base directory                             │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. For Each Test:                                           │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ a) Create Isolated Test Home                         │  │
│  │    - Run: pawd init <name> --chain-id test          │  │
│  │    - Location: /tmp/paw-config-test-TS/category-test │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ b) Modify Configuration                               │  │
│  │    - Use yq to modify TOML in-place                  │  │
│  │    - config.toml OR app.toml                         │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ c) Verify Modification                                │  │
│  │    - Read back value with yq                         │  │
│  │    - Confirm it matches expected                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ d) Start Node                                         │  │
│  │    - Background process                               │  │
│  │    - Wait for RPC endpoint (max 30s)                 │  │
│  │    - Capture logs to file                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ e) Custom Validation (Optional)                       │  │
│  │    - validate_rpc_laddr() - Check port + HTTP        │  │
│  │    - validate_p2p_laddr() - Check P2P port           │  │
│  │    - validate_log_level() - Verify log output        │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ f) Stop Node                                          │  │
│  │    - SIGTERM (graceful)                               │  │
│  │    - SIGKILL after timeout (force)                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ g) Evaluate Result                                    │  │
│  │    - Compare actual vs expected                       │  │
│  │    - "pass" = should work                            │  │
│  │    - "fail" = should error                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓                                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ h) Report & Cleanup                                   │  │
│  │    - Append to markdown report                        │  │
│  │    - Remove test dir (unless --skip-cleanup)         │  │
│  │    - Update statistics                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Post-Processing                                          │
│    - Update report summary                                  │
│    - Print statistics                                       │
│    - Exit with appropriate code                             │
└─────────────────────────────────────────────────────────────┘
```

## Key Functions

### run_test()

The core test execution function. Signature:

```bash
run_test() {
    local category="$1"            # Test category (p2p, rpc, etc.)
    local test_name="$2"           # Short test name
    local description="$3"         # Human-readable description
    local config_file="$4"         # "config" or "app"
    local key_path="$5"            # TOML path (e.g., "p2p.send_rate")
    local test_value="$6"          # Value to set
    local expected_result="$7"     # "pass", "fail", or "skip"
    local validation_func="${8:-}" # Optional validation function
}
```

#### Example Usage

```bash
run_test "p2p" "send-rate-low" \
    "Test low send rate (throttled)" \
    "config" "p2p.send_rate" "102400" "pass"
```

This:
1. Creates test home: `/tmp/paw-config-test-TS/p2p-send-rate-low/`
2. Modifies: `config.toml :: p2p.send_rate = 102400`
3. Starts node
4. Expects: Success (node starts and runs)
5. Reports: Pass/Fail based on outcome

### Validation Functions

Custom validation functions extend basic "did it start?" checks.

#### validate_rpc_laddr()

```bash
validate_rpc_laddr() {
    local test_home="$1"
    local port=$(get_config_value "$test_home" "config" "rpc.laddr" | grep -oP ':\d+$' | tr -d ':')

    # Check port is listening
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "RPC port ${port} is not listening"
        return 1
    fi

    # Check HTTP endpoint responds
    if ! curl -sf "http://localhost:${port}/status" >/dev/null 2>&1; then
        echo "Cannot query RPC status on port ${port}"
        return 1
    fi

    return 0
}
```

This ensures:
- Port is actually listening (netcat check)
- HTTP server is responding (curl check)
- RPC endpoint is functional (status query)

## Configuration Modification

Uses `yq` (v4+) for TOML manipulation:

```bash
modify_config_toml() {
    local test_home="$1"
    local key_path="$2"      # e.g., "p2p.send_rate"
    local value="$3"         # e.g., "102400"
    local config_file="${test_home}/config/config.toml"

    "$YQ" eval -i ".${key_path} = ${value}" -o toml "$config_file"
}
```

### TOML Path Examples

- Simple: `"log_level"` → `log_level = "debug"`
- Nested: `"p2p.send_rate"` → `[p2p] send_rate = 5120000`
- Array: `"rpc.cors_allowed_origins"` → `cors_allowed_origins = ["*"]`

## Node Management

### Startup with Timeout

```bash
start_node() {
    local test_home="$1"

    # Start in background
    "$PAWD_BINARY" start --home "$test_home" > node.log 2>&1 &
    local pid=$!

    # Wait for RPC or timeout
    for ((elapsed=0; elapsed<NODE_STARTUP_TIMEOUT; elapsed++)); do
        if curl -sf http://localhost:26657/status >/dev/null 2>&1; then
            return 0  # Success
        fi
        sleep 1
    done

    return 1  # Timeout
}
```

Key features:
- Background execution with `&`
- PID tracking for cleanup
- HTTP polling for readiness
- Configurable timeout (default 30s)

### Graceful Shutdown

```bash
stop_node() {
    local pid_file="${test_home}/node.pid"
    local pid=$(<"$pid_file")

    # Try graceful shutdown
    kill "$pid"

    # Wait for exit
    for ((elapsed=0; elapsed<NODE_SHUTDOWN_TIMEOUT; elapsed++)); do
        if ! kill -0 "$pid" 2>/dev/null; then
            return 0  # Exited
        fi
        sleep 1
    done

    # Force kill
    kill -9 "$pid"
}
```

## Test Categories

Tests are organized into categories matching config file sections:

### config.toml Categories

1. **base** - Core node settings
2. **rpc** - RPC server
3. **p2p** - Peer-to-peer networking
4. **mempool** - Transaction pool
5. **consensus** - Consensus engine
6. **statesync** - State synchronization
7. **storage** - Storage options
8. **tx_index** - Transaction indexer
9. **instrumentation** - Metrics

### app.toml Categories

10. **app-base** - Application base
11. **telemetry** - App telemetry
12. **api** - REST API
13. **grpc** - gRPC server
14. **state-sync** - State sync snapshots
15. **app-mempool** - App mempool

## Reporting

### Report Format

Markdown format for readability and version control:

```markdown
# PAW Configuration Testing Report

## Summary
- Total Tests: 150
- Passed: 145
- Failed: 3

## Test Results

### ✅ PASSED p2p/send-rate-low
- Description: Test low send rate
- Config: `config.toml :: p2p.send_rate = 102400`

### ❌ FAILED p2p/send-rate-invalid
- Description: Test invalid send rate
- Config: `config.toml :: p2p.send_rate = -1`
- Error: Node failed to start

<details><summary>Node Logs</summary>
[Error output here]
</details>
```

### Report Updates

Reports are built incrementally:
1. `init_report()` - Create header and placeholder summary
2. `append_to_report()` - Add test results as they complete
3. `update_report_summary()` - Replace placeholders with final counts

## Error Handling

### Trap-Based Cleanup

```bash
cleanup_all() {
    if [[ -d "$TEST_BASE_DIR" ]]; then
        rm -rf "$TEST_BASE_DIR"
    fi
}

trap cleanup_all EXIT
```

Ensures cleanup on:
- Normal exit
- Script error
- User interrupt (Ctrl+C)

### Fail-Fast vs Continue-On-Error

Default behavior: Stop on first failure
```bash
if [[ $test_failed && $CONTINUE_ON_ERROR -eq 0 ]]; then
    exit 1
fi
```

Optional: Continue testing
```bash
./scripts/test-config-exhaustive.sh --continue-on-error
```

## Performance Optimizations

### Quick Mode

Only tests critical parameters (15 vs 150+ tests):

```bash
run_quick_tests() {
    # Only essential parameters
    run_test "rpc" "laddr-default" ...
    run_test "p2p" "laddr-default" ...
    run_test "consensus" "timeout-commit-default" ...
    # etc.
}
```

### Category Filtering

Test specific areas:

```bash
./scripts/test-config-exhaustive.sh --category p2p
```

Implementation:
```bash
if [[ -n "$CATEGORY_FILTER" && "$CATEGORY_FILTER" != "$category" ]]; then
    ((SKIPPED_TESTS++))
    return 0
fi
```

## Dependencies

### Required Tools

- **yq** (v4+) - TOML/YAML/JSON manipulation
- **curl** - HTTP requests for RPC checks
- **jq** - JSON parsing for API responses
- **nc** (netcat) - Port connectivity checks
- **docker** - Container management (optional)

### Dependency Verification

```bash
check_dependencies() {
    local missing_deps=()

    for cmd in docker curl jq; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ! -x "$YQ" ]]; then
        missing_deps+=("yq")
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing: ${missing_deps[*]}"
        exit 2
    fi
}
```

## Extensibility

### Adding New Tests

1. Choose appropriate category
2. Call `run_test()` in `run_all_tests()` or `run_quick_tests()`
3. Provide all required arguments
4. Optionally create custom validation function

Example:

```bash
# In run_all_tests()
run_test "p2p" "new-param-test" \
    "Test new P2P parameter" \
    "config" "p2p.new_parameter" '"value"' "pass" validate_new_param

# Custom validation
validate_new_param() {
    local test_home="$1"
    # Your validation logic
    return 0  # Success
}
```

### Adding New Validation Functions

Pattern:

```bash
validate_<feature>() {
    local test_home="$1"

    # 1. Extract relevant config value
    local value=$(get_config_value "$test_home" "config" "path.to.param")

    # 2. Perform validation
    if [[ <condition> ]]; then
        echo "Validation failed: <reason>"
        return 1
    fi

    # 3. Success
    return 0
}
```

## Test Data

### Value Selection Strategy

For each parameter, test:

1. **Default** - Verify baseline works
2. **Small/Low** - Test lower bounds
3. **Large/High** - Test upper bounds
4. **Zero/Empty** - Test special cases
5. **Invalid** - Verify error handling

Example for `mempool.size`:

```bash
run_test "mempool" "size-default" ... "5000" "pass"   # Default
run_test "mempool" "size-small" ... "100" "pass"      # Lower bound
run_test "mempool" "size-large" ... "50000" "pass"    # Upper bound
run_test "mempool" "size-zero" ... "0" "fail"         # Invalid
```

## Security Considerations

### Isolation

Each test runs in a unique directory:
```
/tmp/paw-config-test-20250101-120000/
├── base-log-level-info/
├── p2p-send-rate-low/
└── consensus-timeout-commit-default/
```

No shared state between tests.

### Cleanup

Automatic removal prevents:
- Disk space exhaustion
- Information disclosure (keys, configs)
- Interference with subsequent runs

### Port Management

Tests use default ports (26656, 26657), so:
- Only one test runs at a time
- Ports are freed between tests
- Conflicts are detected early

## Future Enhancements

Potential improvements:

1. **Parallel Execution**
   - Use unique port ranges per test
   - Docker-based isolation
   - Speed up full test suite

2. **Performance Benchmarking**
   - Measure impact of config changes
   - Compare throughput/latency
   - Identify optimal settings

3. **Chaos Testing Integration**
   - Combine with network simulation
   - Test configs under stress
   - Validate timeout behavior

4. **Machine-Readable Output**
   - JSON report format
   - TAP (Test Anything Protocol)
   - JUnit XML for CI integration

5. **Historical Tracking**
   - Store reports in Git
   - Compare across versions
   - Detect regressions

## References

- Cosmos SDK Configuration: https://docs.cosmos.network/main/run-node/run-node
- CometBFT Config: https://docs.cometbft.com/v0.38/core/configuration
- yq Documentation: https://mikefarah.gitbook.io/yq/
- PAW Testing Plan: `LOCAL_TESTING_PLAN.md`
