#!/usr/bin/env bash
#
# test-config-exhaustive.sh - Phase 2.2: Exhaustive Configuration Testing
#
# Tests every parameter in config.toml and app.toml to verify:
# 1. Node starts/stops correctly with each configuration
# 2. Behavior changes as expected or fails gracefully
# 3. Edge cases and invalid values are handled properly
#
# Usage:
#   ./scripts/test-config-exhaustive.sh [options]
#
# Options:
#   --quick              Run only critical parameters (faster)
#   --category <name>    Test only specific category (p2p, rpc, consensus, mempool, api, grpc, telemetry, storage)
#   --continue-on-error  Don't stop on first failure
#   --skip-cleanup       Keep test directories for manual inspection
#   --report <file>      Custom report output path (default: config-test-report-TIMESTAMP.md)
#
# Exit codes:
#   0 - All tests passed
#   1 - Some tests failed (see report)
#   2 - Script error (missing dependencies, etc.)

set -euo pipefail

# ============================================================================
# Configuration & Constants
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
TEST_BASE_DIR="/tmp/paw-config-test-${TIMESTAMP}"
REPORT_FILE="${PROJECT_ROOT}/config-test-report-${TIMESTAMP}.md"
PAWD_BINARY="${PROJECT_ROOT}/pawd"

# Test control flags
QUICK_MODE=0
CONTINUE_ON_ERROR=0
SKIP_CLEANUP=0
CATEGORY_FILTER=""

# Test statistics
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0
BLOCKED_TESTS=0

# Node startup timeout (seconds)
NODE_STARTUP_TIMEOUT=30
NODE_SHUTDOWN_TIMEOUT=10

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Utility Functions
# ============================================================================

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✓${NC} $*"
}

log_error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ✗${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠${NC} $*"
}

check_dependencies() {
    local missing_deps=()

    for cmd in docker curl jq toml; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        exit 2
    fi
}

build_pawd() {
    if [[ -x "$PAWD_BINARY" ]]; then
        log "Using existing pawd binary: ${PAWD_BINARY}"
        return 0
    fi

    log "Building pawd binary..."
    cd "$PROJECT_ROOT"
    if ! go build -o "$PAWD_BINARY" ./cmd/pawd 2>&1 | tee /tmp/pawd-build.log; then
        log_error "Failed to build pawd binary"
        cat /tmp/pawd-build.log
        exit 2
    fi
    log_success "pawd binary built successfully"
}

init_report() {
    cat > "$REPORT_FILE" <<EOF
# PAW Configuration Testing Report

**Generated:** $(date '+%Y-%m-%d %H:%M:%S')
**Test Mode:** $([ $QUICK_MODE -eq 1 ] && echo "Quick" || echo "Exhaustive")
**Category Filter:** ${CATEGORY_FILTER:-All}

## Summary

- **Total Tests:** TOTAL_PLACEHOLDER
- **Passed:** PASSED_PLACEHOLDER
- **Failed:** FAILED_PLACEHOLDER
- **Skipped:** SKIPPED_PLACEHOLDER
- **Blocked:** BLOCKED_PLACEHOLDER

## Test Results

EOF
}

update_report_summary() {
    sed -i "s/TOTAL_PLACEHOLDER/${TOTAL_TESTS}/g" "$REPORT_FILE"
    sed -i "s/PASSED_PLACEHOLDER/${PASSED_TESTS}/g" "$REPORT_FILE"
    sed -i "s/FAILED_PLACEHOLDER/${FAILED_TESTS}/g" "$REPORT_FILE"
    sed -i "s/SKIPPED_PLACEHOLDER/${SKIPPED_TESTS}/g" "$REPORT_FILE"
    sed -i "s/BLOCKED_PLACEHOLDER/${BLOCKED_TESTS}/g" "$REPORT_FILE"
}

append_to_report() {
    echo "$*" >> "$REPORT_FILE"
}

# ============================================================================
# Node Management Functions
# ============================================================================

create_test_home() {
    local test_name="$1"
    local test_home="${TEST_BASE_DIR}/${test_name}"

    mkdir -p "$test_home"

    # Initialize node
    if ! "$PAWD_BINARY" init "config-test-${test_name}" \
        --chain-id "config-test" \
        --home "$test_home" >/dev/null 2>&1; then
        log_error "Failed to initialize test home: ${test_name}"
        return 1
    fi

    echo "$test_home"
}

start_node() {
    local test_home="$1"
    local log_file="${test_home}/node.log"
    local pid_file="${test_home}/node.pid"

    # Start node in background
    "$PAWD_BINARY" start \
        --home "$test_home" \
        --log_level error \
        > "$log_file" 2>&1 &

    local pid=$!
    echo "$pid" > "$pid_file"

    # Wait for node to be ready or fail
    local elapsed=0
    while [[ $elapsed -lt $NODE_STARTUP_TIMEOUT ]]; do
        # Check if process is still running
        if ! kill -0 "$pid" 2>/dev/null; then
            log_error "Node process died during startup"
            return 1
        fi

        # Check if node is responding
        if curl -sf http://localhost:26657/status >/dev/null 2>&1; then
            return 0
        fi

        sleep 1
        elapsed=$((elapsed + 1))
    done

    log_error "Node startup timeout after ${NODE_STARTUP_TIMEOUT}s"
    return 1
}

stop_node() {
    local test_home="$1"
    local pid_file="${test_home}/node.pid"

    if [[ ! -f "$pid_file" ]]; then
        return 0
    fi

    local pid
    pid=$(<"$pid_file")

    if ! kill -0 "$pid" 2>/dev/null; then
        # Already dead
        rm -f "$pid_file"
        return 0
    fi

    # Send SIGTERM
    kill "$pid" 2>/dev/null || true

    # Wait for graceful shutdown
    local elapsed=0
    while [[ $elapsed -lt $NODE_SHUTDOWN_TIMEOUT ]]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            rm -f "$pid_file"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    # Force kill
    log_warning "Forcing node shutdown"
    kill -9 "$pid" 2>/dev/null || true
    rm -f "$pid_file"
    return 0
}

get_node_logs() {
    local test_home="$1"
    local log_file="${test_home}/node.log"

    if [[ -f "$log_file" ]]; then
        tail -n 50 "$log_file"
    else
        echo "No log file found"
    fi
}

# ============================================================================
# Configuration Modification Functions
# ============================================================================

modify_config_toml() {
    local test_home="$1"
    local key_path="$2"
    local value="$3"
    local config_file="${test_home}/config/config.toml"

    # Use toml-cli to modify TOML in place
    toml set --toml-path "$config_file" "$key_path" "$value"
}

modify_app_toml() {
    local test_home="$1"
    local key_path="$2"
    local value="$3"
    local config_file="${test_home}/config/app.toml"

    toml set --toml-path "$config_file" "$key_path" "$value"
}

get_config_value() {
    local test_home="$1"
    local file="$2"  # "config" or "app"
    local key_path="$3"
    local config_file="${test_home}/config/${file}.toml"

    toml get --toml-path "$config_file" "$key_path"
}

# ============================================================================
# Test Execution Functions
# ============================================================================

run_test() {
    local category="$1"
    local test_name="$2"
    local description="$3"
    local config_file="$4"  # "config" or "app"
    local key_path="$5"
    local test_value="$6"
    local expected_result="$7"  # "pass", "fail", "skip"
    local validation_func="${8:-}"  # Optional validation function

    # Check category filter
    if [[ -n "$CATEGORY_FILTER" && "$CATEGORY_FILTER" != "$category" ]]; then
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        return 0
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local full_test_name="${category}/${test_name}"
    log "Running test: ${full_test_name}"
    log "  Description: ${description}"
    log "  Setting ${config_file}.toml :: ${key_path} = ${test_value}"

    # Create test home
    local test_home
    if ! test_home=$(create_test_home "${category}-${test_name}"); then
        log_error "Failed to create test home"
        append_to_report "### ❌ ${full_test_name}"
        append_to_report ""
        append_to_report "- **Description:** ${description}"
        append_to_report "- **Status:** FAILED (initialization)"
        append_to_report "- **Error:** Could not create test home directory"
        append_to_report ""
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # Modify configuration
    if [[ "$config_file" == "config" ]]; then
        modify_config_toml "$test_home" "$key_path" "$test_value"
    else
        modify_app_toml "$test_home" "$key_path" "$test_value"
    fi

    # Verify modification
    local actual_value
    actual_value=$(get_config_value "$test_home" "$config_file" "$key_path")
    if [[ "$actual_value" != "$test_value" ]]; then
        log_error "Configuration modification failed. Expected: ${test_value}, Got: ${actual_value}"
        append_to_report "### ❌ ${full_test_name}"
        append_to_report ""
        append_to_report "- **Description:** ${description}"
        append_to_report "- **Status:** FAILED (config modification)"
        append_to_report "- **Error:** Value not set correctly"
        append_to_report ""
        FAILED_TESTS=$((FAILED_TESTS + 1))
        [[ $SKIP_CLEANUP -eq 0 ]] && rm -rf "$test_home"
        return 1
    fi

    # Start node
    local start_result=0
    if ! start_node "$test_home"; then
        start_result=1
    fi

    # Run custom validation if provided
    local validation_result=0
    local validation_error=""
    if [[ -n "$validation_func" && $start_result -eq 0 ]]; then
        if ! validation_error=$($validation_func "$test_home" 2>&1); then
            validation_result=1
        fi
    fi

    # Stop node
    stop_node "$test_home"

    # Evaluate result
    local test_passed=0
    local status_text=""
    local error_text=""

    case "$expected_result" in
        pass)
            if [[ $start_result -eq 0 && $validation_result -eq 0 ]]; then
                test_passed=1
                status_text="✅ PASSED"
            else
                status_text="❌ FAILED"
                if [[ $start_result -ne 0 ]]; then
                    error_text="Node failed to start"
                else
                    error_text="Validation failed: ${validation_error}"
                fi
            fi
            ;;
        fail)
            if [[ $start_result -ne 0 ]]; then
                test_passed=1
                status_text="✅ PASSED (expected failure)"
            else
                status_text="❌ FAILED (should have failed)"
                error_text="Node started successfully when it should have failed"
            fi
            ;;
        skip)
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            status_text="⊘ SKIPPED"
            ;;
    esac

    # Report results
    if [[ $test_passed -eq 1 ]]; then
        log_success "${full_test_name}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        append_to_report "### ${status_text} ${full_test_name}"
        append_to_report ""
        append_to_report "- **Description:** ${description}"
        append_to_report "- **Config:** \`${config_file}.toml :: ${key_path} = ${test_value}\`"
        append_to_report ""
    else
        log_error "${full_test_name}: ${error_text}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        append_to_report "### ${status_text} ${full_test_name}"
        append_to_report ""
        append_to_report "- **Description:** ${description}"
        append_to_report "- **Config:** \`${config_file}.toml :: ${key_path} = ${test_value}\`"
        append_to_report "- **Error:** ${error_text}"
        append_to_report ""
        append_to_report "<details><summary>Node Logs</summary>"
        append_to_report ""
        append_to_report '```'
        append_to_report "$(get_node_logs "$test_home")"
        append_to_report '```'
        append_to_report ""
        append_to_report "</details>"
        append_to_report ""

        if [[ $CONTINUE_ON_ERROR -eq 0 ]]; then
            log_error "Stopping on first failure (use --continue-on-error to continue)"
            [[ $SKIP_CLEANUP -eq 0 ]] && rm -rf "$test_home"
            return 1
        fi
    fi

    # Cleanup unless requested otherwise
    if [[ $SKIP_CLEANUP -eq 0 ]]; then
        rm -rf "$test_home"
    fi

    return 0
}

# ============================================================================
# Validation Functions (Custom checks for specific parameters)
# ============================================================================

validate_rpc_laddr() {
    local test_home="$1"
    local port
    port=$(get_config_value "$test_home" "config" "rpc.laddr" | grep -oP ':\d+$' | tr -d ':')

    # Check if port is listening
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "RPC port ${port} is not listening"
        return 1
    fi

    # Check if we can query status
    if ! curl -sf "http://localhost:${port}/status" >/dev/null 2>&1; then
        echo "Cannot query RPC status on port ${port}"
        return 1
    fi

    return 0
}

validate_p2p_laddr() {
    local test_home="$1"
    local port
    port=$(get_config_value "$test_home" "config" "p2p.laddr" | grep -oP ':\d+$' | tr -d ':')

    # Check if port is listening
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "P2P port ${port} is not listening"
        return 1
    fi

    return 0
}

validate_log_level() {
    local test_home="$1"
    local log_file="${test_home}/node.log"
    local expected_level
    expected_level=$(get_config_value "$test_home" "config" "log_level")

    # Check if log file contains appropriate level messages
    if [[ ! -f "$log_file" ]]; then
        echo "Log file not found"
        return 1
    fi

    # For debug level, expect more verbose output
    if [[ "$expected_level" == "debug" ]]; then
        if ! grep -qi "debug" "$log_file"; then
            echo "Expected DEBUG level logs but found none"
            return 1
        fi
    fi

    return 0
}

# ============================================================================
# Test Definitions
# ============================================================================

run_all_tests() {
    log "Starting exhaustive configuration tests..."

    # ========================================================================
    # config.toml Tests
    # ========================================================================

    # ------------------------------------------------------------------------
    # Base Configuration Tests
    # ------------------------------------------------------------------------

    run_test "base" "moniker-valid" \
        "Test valid moniker change" \
        "config" "moniker" '"test-node-custom"' "pass"

    run_test "base" "moniker-empty" \
        "Test empty moniker (should work with default)" \
        "config" "moniker" '""' "pass"

    run_test "base" "db-backend-goleveldb" \
        "Test goleveldb backend (default)" \
        "config" "db_backend" '"goleveldb"' "pass"

    run_test "base" "db-backend-invalid" \
        "Test invalid database backend" \
        "config" "db_backend" '"invaliddb"' "fail"

    run_test "base" "log-level-info" \
        "Test log level: info" \
        "config" "log_level" '"info"' "pass" validate_log_level

    run_test "base" "log-level-debug" \
        "Test log level: debug" \
        "config" "log_level" '"debug"' "pass" validate_log_level

    run_test "base" "log-level-error" \
        "Test log level: error" \
        "config" "log_level" '"error"' "pass"

    run_test "base" "log-level-invalid" \
        "Test invalid log level" \
        "config" "log_level" '"invalid"' "fail"

    run_test "base" "log-format-plain" \
        "Test log format: plain" \
        "config" "log_format" '"plain"' "pass"

    run_test "base" "log-format-json" \
        "Test log format: json" \
        "config" "log_format" '"json"' "pass"

    run_test "base" "filter-peers-true" \
        "Test filter_peers enabled" \
        "config" "filter_peers" "true" "pass"

    run_test "base" "filter-peers-false" \
        "Test filter_peers disabled" \
        "config" "filter_peers" "false" "pass"

    # ------------------------------------------------------------------------
    # RPC Configuration Tests
    # ------------------------------------------------------------------------

    run_test "rpc" "laddr-default" \
        "Test default RPC listen address" \
        "config" "rpc.laddr" '"tcp://127.0.0.1:26657"' "pass" validate_rpc_laddr

    run_test "rpc" "laddr-custom-port" \
        "Test custom RPC port" \
        "config" "rpc.laddr" '"tcp://127.0.0.1:36657"' "pass"

    run_test "rpc" "laddr-all-interfaces" \
        "Test RPC listen on all interfaces" \
        "config" "rpc.laddr" '"tcp://0.0.0.0:26657"' "pass"

    run_test "rpc" "laddr-invalid-format" \
        "Test invalid RPC address format" \
        "config" "rpc.laddr" '"invalid-address"' "fail"

    run_test "rpc" "unsafe-true" \
        "Test unsafe RPC enabled" \
        "config" "rpc.unsafe" "true" "pass"

    run_test "rpc" "unsafe-false" \
        "Test unsafe RPC disabled (default)" \
        "config" "rpc.unsafe" "false" "pass"

    run_test "rpc" "max-open-connections-default" \
        "Test default max open connections (900)" \
        "config" "rpc.max_open_connections" "900" "pass"

    run_test "rpc" "max-open-connections-low" \
        "Test low max open connections" \
        "config" "rpc.max_open_connections" "10" "pass"

    run_test "rpc" "max-open-connections-high" \
        "Test high max open connections" \
        "config" "rpc.max_open_connections" "5000" "pass"

    run_test "rpc" "max-open-connections-zero" \
        "Test unlimited connections (0)" \
        "config" "rpc.max_open_connections" "0" "pass"

    run_test "rpc" "max-subscription-clients-default" \
        "Test default max subscription clients" \
        "config" "rpc.max_subscription_clients" "100" "pass"

    run_test "rpc" "max-subscription-clients-high" \
        "Test high max subscription clients" \
        "config" "rpc.max_subscription_clients" "1000" "pass"

    run_test "rpc" "timeout-broadcast-tx-commit-default" \
        "Test default broadcast timeout (10s)" \
        "config" "rpc.timeout_broadcast_tx_commit" '"10s"' "pass"

    run_test "rpc" "timeout-broadcast-tx-commit-long" \
        "Test long broadcast timeout (30s)" \
        "config" "rpc.timeout_broadcast_tx_commit" '"30s"' "pass"

    run_test "rpc" "max-body-bytes-default" \
        "Test default max body bytes (1MB)" \
        "config" "rpc.max_body_bytes" "1000000" "pass"

    run_test "rpc" "max-body-bytes-large" \
        "Test large max body bytes (10MB)" \
        "config" "rpc.max_body_bytes" "10000000" "pass"

    run_test "rpc" "cors-allow-all" \
        "Test CORS allow all origins" \
        "config" "rpc.cors_allowed_origins" '["*"]' "pass"

    # ------------------------------------------------------------------------
    # P2P Configuration Tests
    # ------------------------------------------------------------------------

    run_test "p2p" "laddr-default" \
        "Test default P2P listen address" \
        "config" "p2p.laddr" '"tcp://0.0.0.0:26656"' "pass" validate_p2p_laddr

    run_test "p2p" "laddr-custom-port" \
        "Test custom P2P port" \
        "config" "p2p.laddr" '"tcp://0.0.0.0:36656"' "pass"

    run_test "p2p" "max-num-inbound-peers-default" \
        "Test default max inbound peers (40)" \
        "config" "p2p.max_num_inbound_peers" "40" "pass"

    run_test "p2p" "max-num-inbound-peers-low" \
        "Test low max inbound peers" \
        "config" "p2p.max_num_inbound_peers" "5" "pass"

    run_test "p2p" "max-num-inbound-peers-high" \
        "Test high max inbound peers" \
        "config" "p2p.max_num_inbound_peers" "200" "pass"

    run_test "p2p" "max-num-inbound-peers-zero" \
        "Test zero inbound peers" \
        "config" "p2p.max_num_inbound_peers" "0" "pass"

    run_test "p2p" "max-num-outbound-peers-default" \
        "Test default max outbound peers (10)" \
        "config" "p2p.max_num_outbound_peers" "10" "pass"

    run_test "p2p" "max-num-outbound-peers-high" \
        "Test high max outbound peers" \
        "config" "p2p.max_num_outbound_peers" "50" "pass"

    run_test "p2p" "send-rate-default" \
        "Test default send rate (5120000 bytes/s)" \
        "config" "p2p.send_rate" "5120000" "pass"

    run_test "p2p" "send-rate-low" \
        "Test low send rate (throttled)" \
        "config" "p2p.send_rate" "102400" "pass"

    run_test "p2p" "send-rate-high" \
        "Test high send rate" \
        "config" "p2p.send_rate" "52428800" "pass"

    run_test "p2p" "recv-rate-default" \
        "Test default receive rate" \
        "config" "p2p.recv_rate" "5120000" "pass"

    run_test "p2p" "recv-rate-low" \
        "Test low receive rate" \
        "config" "p2p.recv_rate" "102400" "pass"

    run_test "p2p" "pex-enabled" \
        "Test peer exchange enabled (default)" \
        "config" "p2p.pex" "true" "pass"

    run_test "p2p" "pex-disabled" \
        "Test peer exchange disabled" \
        "config" "p2p.pex" "false" "pass"

    run_test "p2p" "seed-mode-enabled" \
        "Test seed mode enabled" \
        "config" "p2p.seed_mode" "true" "pass"

    run_test "p2p" "seed-mode-disabled" \
        "Test seed mode disabled (default)" \
        "config" "p2p.seed_mode" "false" "pass"

    run_test "p2p" "allow-duplicate-ip-true" \
        "Test allow duplicate IP enabled" \
        "config" "p2p.allow_duplicate_ip" "true" "pass"

    run_test "p2p" "allow-duplicate-ip-false" \
        "Test allow duplicate IP disabled (default)" \
        "config" "p2p.allow_duplicate_ip" "false" "pass"

    run_test "p2p" "handshake-timeout-default" \
        "Test default handshake timeout (20s)" \
        "config" "p2p.handshake_timeout" '"20s"' "pass"

    run_test "p2p" "handshake-timeout-short" \
        "Test short handshake timeout (5s)" \
        "config" "p2p.handshake_timeout" '"5s"' "pass"

    run_test "p2p" "handshake-timeout-long" \
        "Test long handshake timeout (60s)" \
        "config" "p2p.handshake_timeout" '"60s"' "pass"

    run_test "p2p" "dial-timeout-default" \
        "Test default dial timeout (3s)" \
        "config" "p2p.dial_timeout" '"3s"' "pass"

    run_test "p2p" "dial-timeout-short" \
        "Test short dial timeout (1s)" \
        "config" "p2p.dial_timeout" '"1s"' "pass"

    run_test "p2p" "flush-throttle-timeout-default" \
        "Test default flush throttle timeout (100ms)" \
        "config" "p2p.flush_throttle_timeout" '"100ms"' "pass"

    run_test "p2p" "flush-throttle-timeout-low" \
        "Test low flush throttle timeout (10ms)" \
        "config" "p2p.flush_throttle_timeout" '"10ms"' "pass"

    run_test "p2p" "max-packet-msg-payload-size-default" \
        "Test default max packet message payload size" \
        "config" "p2p.max_packet_msg_payload_size" "1024" "pass"

    run_test "p2p" "max-packet-msg-payload-size-large" \
        "Test large max packet message payload size" \
        "config" "p2p.max_packet_msg_payload_size" "65536" "pass"

    # ------------------------------------------------------------------------
    # Mempool Configuration Tests
    # ------------------------------------------------------------------------

    run_test "mempool" "type-flood" \
        "Test flood mempool type (default)" \
        "config" "mempool.type" '"flood"' "pass"

    run_test "mempool" "type-nop" \
        "Test nop mempool type" \
        "config" "mempool.type" '"nop"' "pass"

    run_test "mempool" "recheck-enabled" \
        "Test mempool recheck enabled (default)" \
        "config" "mempool.recheck" "true" "pass"

    run_test "mempool" "recheck-disabled" \
        "Test mempool recheck disabled" \
        "config" "mempool.recheck" "false" "pass"

    run_test "mempool" "broadcast-enabled" \
        "Test mempool broadcast enabled (default)" \
        "config" "mempool.broadcast" "true" "pass"

    run_test "mempool" "broadcast-disabled" \
        "Test mempool broadcast disabled" \
        "config" "mempool.broadcast" "false" "pass"

    run_test "mempool" "size-default" \
        "Test default mempool size (5000)" \
        "config" "mempool.size" "5000" "pass"

    run_test "mempool" "size-small" \
        "Test small mempool size" \
        "config" "mempool.size" "100" "pass"

    run_test "mempool" "size-large" \
        "Test large mempool size" \
        "config" "mempool.size" "50000" "pass"

    run_test "mempool" "max-txs-bytes-default" \
        "Test default max txs bytes (1GB)" \
        "config" "mempool.max_txs_bytes" "1073741824" "pass"

    run_test "mempool" "max-txs-bytes-small" \
        "Test small max txs bytes (10MB)" \
        "config" "mempool.max_txs_bytes" "10485760" "pass"

    run_test "mempool" "cache-size-default" \
        "Test default cache size (10000)" \
        "config" "mempool.cache_size" "10000" "pass"

    run_test "mempool" "cache-size-small" \
        "Test small cache size" \
        "config" "mempool.cache_size" "1000" "pass"

    run_test "mempool" "cache-size-large" \
        "Test large cache size" \
        "config" "mempool.cache_size" "100000" "pass"

    run_test "mempool" "max-tx-bytes-default" \
        "Test default max tx bytes (1MB)" \
        "config" "mempool.max_tx_bytes" "1048576" "pass"

    run_test "mempool" "max-tx-bytes-small" \
        "Test small max tx bytes (10KB)" \
        "config" "mempool.max_tx_bytes" "10240" "pass"

    # ------------------------------------------------------------------------
    # Consensus Configuration Tests
    # ------------------------------------------------------------------------

    run_test "consensus" "timeout-propose-default" \
        "Test default timeout propose" \
        "config" "consensus.timeout_propose" '"3s"' "pass"

    run_test "consensus" "timeout-propose-short" \
        "Test short timeout propose (1s)" \
        "config" "consensus.timeout_propose" '"1s"' "pass"

    run_test "consensus" "timeout-propose-long" \
        "Test long timeout propose (10s)" \
        "config" "consensus.timeout_propose" '"10s"' "pass"

    run_test "consensus" "timeout-commit-default" \
        "Test default timeout commit (5s)" \
        "config" "consensus.timeout_commit" '"5s"' "pass"

    run_test "consensus" "timeout-commit-zero" \
        "Test zero timeout commit" \
        "config" "consensus.timeout_commit" '"0s"' "pass"

    run_test "consensus" "skip-timeout-commit-true" \
        "Test skip timeout commit enabled" \
        "config" "consensus.skip_timeout_commit" "true" "pass"

    run_test "consensus" "skip-timeout-commit-false" \
        "Test skip timeout commit disabled (default)" \
        "config" "consensus.skip_timeout_commit" "false" "pass"

    run_test "consensus" "create-empty-blocks-true" \
        "Test create empty blocks enabled (default)" \
        "config" "consensus.create_empty_blocks" "true" "pass"

    run_test "consensus" "create-empty-blocks-false" \
        "Test create empty blocks disabled" \
        "config" "consensus.create_empty_blocks" "false" "pass"

    run_test "consensus" "create-empty-blocks-interval-default" \
        "Test default create empty blocks interval (0s)" \
        "config" "consensus.create_empty_blocks_interval" '"0s"' "pass"

    run_test "consensus" "create-empty-blocks-interval-custom" \
        "Test custom create empty blocks interval (30s)" \
        "config" "consensus.create_empty_blocks_interval" '"30s"' "pass"

    run_test "consensus" "peer-gossip-sleep-duration-default" \
        "Test default peer gossip sleep duration" \
        "config" "consensus.peer_gossip_sleep_duration" '"100ms"' "pass"

    run_test "consensus" "peer-gossip-sleep-duration-short" \
        "Test short peer gossip sleep duration" \
        "config" "consensus.peer_gossip_sleep_duration" '"10ms"' "pass"

    # ------------------------------------------------------------------------
    # State Sync Configuration Tests
    # ------------------------------------------------------------------------

    run_test "statesync" "enable-false" \
        "Test state sync disabled (default)" \
        "config" "statesync.enable" "false" "pass"

    run_test "statesync" "discovery-time-default" \
        "Test default discovery time (15s)" \
        "config" "statesync.discovery_time" '"15s"' "pass"

    run_test "statesync" "discovery-time-short" \
        "Test short discovery time (5s)" \
        "config" "statesync.discovery_time" '"5s"' "pass"

    run_test "statesync" "chunk-request-timeout-default" \
        "Test default chunk request timeout (10s)" \
        "config" "statesync.chunk_request_timeout" '"10s"' "pass"

    run_test "statesync" "chunk-fetchers-default" \
        "Test default chunk fetchers (4)" \
        "config" "statesync.chunk_fetchers" '"4"' "pass"

    run_test "statesync" "chunk-fetchers-many" \
        "Test many chunk fetchers (16)" \
        "config" "statesync.chunk_fetchers" '"16"' "pass"

    # ------------------------------------------------------------------------
    # Storage Configuration Tests
    # ------------------------------------------------------------------------

    run_test "storage" "discard-abci-responses-false" \
        "Test keep ABCI responses (default)" \
        "config" "storage.discard_abci_responses" "false" "pass"

    run_test "storage" "discard-abci-responses-true" \
        "Test discard ABCI responses" \
        "config" "storage.discard_abci_responses" "true" "pass"

    # ------------------------------------------------------------------------
    # Transaction Indexer Configuration Tests
    # ------------------------------------------------------------------------

    run_test "tx_index" "indexer-kv" \
        "Test kv indexer (default)" \
        "config" "tx_index.indexer" '"kv"' "pass"

    run_test "tx_index" "indexer-null" \
        "Test null indexer (disabled)" \
        "config" "tx_index.indexer" '"null"' "pass"

    # ------------------------------------------------------------------------
    # Instrumentation Configuration Tests
    # ------------------------------------------------------------------------

    run_test "instrumentation" "prometheus-enabled" \
        "Test Prometheus metrics enabled" \
        "config" "instrumentation.prometheus" "true" "pass"

    run_test "instrumentation" "prometheus-disabled" \
        "Test Prometheus metrics disabled (default)" \
        "config" "instrumentation.prometheus" "false" "pass"

    run_test "instrumentation" "prometheus-listen-addr-default" \
        "Test default Prometheus listen address" \
        "config" "instrumentation.prometheus_listen_addr" '":26660"' "pass"

    run_test "instrumentation" "prometheus-listen-addr-custom" \
        "Test custom Prometheus listen address" \
        "config" "instrumentation.prometheus_listen_addr" '":9090"' "pass"

    # ========================================================================
    # app.toml Tests
    # ========================================================================

    # ------------------------------------------------------------------------
    # Base App Configuration Tests
    # ------------------------------------------------------------------------

    run_test "app-base" "minimum-gas-prices-default" \
        "Test default minimum gas prices" \
        "app" "minimum-gas-prices" '"0.001upaw"' "pass"

    run_test "app-base" "minimum-gas-prices-zero" \
        "Test zero minimum gas prices" \
        "app" "minimum-gas-prices" '"0upaw"' "pass"

    run_test "app-base" "minimum-gas-prices-high" \
        "Test high minimum gas prices" \
        "app" "minimum-gas-prices" '"1.0upaw"' "pass"

    run_test "app-base" "pruning-default" \
        "Test default pruning strategy" \
        "app" "pruning" '"default"' "pass"

    run_test "app-base" "pruning-nothing" \
        "Test pruning nothing (archive node)" \
        "app" "pruning" '"nothing"' "pass"

    run_test "app-base" "pruning-everything" \
        "Test pruning everything" \
        "app" "pruning" '"everything"' "pass"

    run_test "app-base" "pruning-custom" \
        "Test custom pruning strategy" \
        "app" "pruning" '"custom"' "pass"

    run_test "app-base" "halt-height-zero" \
        "Test halt height zero (disabled)" \
        "app" "halt-height" "0" "pass"

    run_test "app-base" "halt-height-custom" \
        "Test custom halt height" \
        "app" "halt-height" "1000" "pass"

    run_test "app-base" "inter-block-cache-enabled" \
        "Test inter-block cache enabled (default)" \
        "app" "inter-block-cache" "true" "pass"

    run_test "app-base" "inter-block-cache-disabled" \
        "Test inter-block cache disabled" \
        "app" "inter-block-cache" "false" "pass"

    run_test "app-base" "iavl-cache-size-default" \
        "Test default IAVL cache size" \
        "app" "iavl-cache-size" "781250" "pass"

    run_test "app-base" "iavl-cache-size-small" \
        "Test small IAVL cache size" \
        "app" "iavl-cache-size" "10000" "pass"

    run_test "app-base" "iavl-cache-size-large" \
        "Test large IAVL cache size" \
        "app" "iavl-cache-size" "5000000" "pass"

    run_test "app-base" "iavl-disable-fastnode-false" \
        "Test IAVL fast node enabled (default)" \
        "app" "iavl-disable-fastnode" "false" "pass"

    run_test "app-base" "iavl-disable-fastnode-true" \
        "Test IAVL fast node disabled" \
        "app" "iavl-disable-fastnode" "true" "pass"

    # ------------------------------------------------------------------------
    # Telemetry Configuration Tests
    # ------------------------------------------------------------------------

    run_test "telemetry" "enabled-true" \
        "Test telemetry enabled" \
        "app" "telemetry.enabled" "true" "pass"

    run_test "telemetry" "enabled-false" \
        "Test telemetry disabled (default)" \
        "app" "telemetry.enabled" "false" "pass"

    run_test "telemetry" "enable-hostname-true" \
        "Test enable hostname in telemetry" \
        "app" "telemetry.enable-hostname" "true" "pass"

    run_test "telemetry" "enable-hostname-label-true" \
        "Test enable hostname label in telemetry" \
        "app" "telemetry.enable-hostname-label" "true" "pass"

    # ------------------------------------------------------------------------
    # API Configuration Tests
    # ------------------------------------------------------------------------

    run_test "api" "enable-false" \
        "Test API disabled (default)" \
        "app" "api.enable" "false" "pass"

    run_test "api" "enable-true" \
        "Test API enabled" \
        "app" "api.enable" "true" "pass"

    run_test "api" "swagger-enabled" \
        "Test Swagger enabled" \
        "app" "api.swagger" "true" "pass"

    run_test "api" "max-open-connections-default" \
        "Test default API max open connections" \
        "app" "api.max-open-connections" "1000" "pass"

    run_test "api" "max-open-connections-low" \
        "Test low API max open connections" \
        "app" "api.max-open-connections" "100" "pass"

    run_test "api" "rpc-read-timeout-default" \
        "Test default RPC read timeout (10s)" \
        "app" "api.rpc-read-timeout" "10" "pass"

    run_test "api" "rpc-read-timeout-long" \
        "Test long RPC read timeout (60s)" \
        "app" "api.rpc-read-timeout" "60" "pass"

    # ------------------------------------------------------------------------
    # gRPC Configuration Tests
    # ------------------------------------------------------------------------

    run_test "grpc" "enable-true" \
        "Test gRPC enabled (default)" \
        "app" "grpc.enable" "true" "pass"

    run_test "grpc" "enable-false" \
        "Test gRPC disabled" \
        "app" "grpc.enable" "false" "pass"

    run_test "grpc" "max-recv-msg-size-default" \
        "Test default gRPC max receive message size" \
        "app" "grpc.max-recv-msg-size" '"10485760"' "pass"

    run_test "grpc" "max-recv-msg-size-large" \
        "Test large gRPC max receive message size" \
        "app" "grpc.max-recv-msg-size" '"52428800"' "pass"

    # ------------------------------------------------------------------------
    # State Sync Configuration Tests
    # ------------------------------------------------------------------------

    run_test "state-sync" "snapshot-interval-zero" \
        "Test state sync snapshots disabled (default)" \
        "app" "state-sync.snapshot-interval" "0" "pass"

    run_test "state-sync" "snapshot-interval-enabled" \
        "Test state sync snapshots enabled (every 1000 blocks)" \
        "app" "state-sync.snapshot-interval" "1000" "pass"

    run_test "state-sync" "snapshot-keep-recent-default" \
        "Test default snapshot keep recent (2)" \
        "app" "state-sync.snapshot-keep-recent" "2" "pass"

    run_test "state-sync" "snapshot-keep-recent-many" \
        "Test keep many recent snapshots (10)" \
        "app" "state-sync.snapshot-keep-recent" "10" "pass"

    # ------------------------------------------------------------------------
    # Mempool Configuration Tests (App)
    # ------------------------------------------------------------------------

    run_test "app-mempool" "max-txs-disabled" \
        "Test app-side mempool disabled (default)" \
        "app" "mempool.max-txs" "-1" "pass"

    run_test "app-mempool" "max-txs-unlimited" \
        "Test app-side mempool unlimited" \
        "app" "mempool.max-txs" "0" "pass"

    run_test "app-mempool" "max-txs-limited" \
        "Test app-side mempool limited (5000)" \
        "app" "mempool.max-txs" "5000" "pass"
}

run_quick_tests() {
    log "Running quick (critical parameters only) configuration tests..."

    # Only test critical parameters that are most likely to cause issues

    # Critical base parameters
    run_test "base" "log-level-info" \
        "Test log level: info" \
        "config" "log_level" '"info"' "pass"

    # Critical RPC parameters
    run_test "rpc" "laddr-default" \
        "Test default RPC listen address" \
        "config" "rpc.laddr" '"tcp://127.0.0.1:26657"' "pass" validate_rpc_laddr

    run_test "rpc" "max-open-connections-default" \
        "Test default max open connections" \
        "config" "rpc.max_open_connections" "900" "pass"

    # Critical P2P parameters
    run_test "p2p" "laddr-default" \
        "Test default P2P listen address" \
        "config" "p2p.laddr" '"tcp://0.0.0.0:26656"' "pass" validate_p2p_laddr

    run_test "p2p" "max-num-inbound-peers-default" \
        "Test default max inbound peers" \
        "config" "p2p.max_num_inbound_peers" "40" "pass"

    run_test "p2p" "handshake-timeout-default" \
        "Test default handshake timeout" \
        "config" "p2p.handshake_timeout" '"20s"' "pass"

    # Critical mempool parameters
    run_test "mempool" "size-default" \
        "Test default mempool size" \
        "config" "mempool.size" "5000" "pass"

    run_test "mempool" "cache-size-default" \
        "Test default cache size" \
        "config" "mempool.cache_size" "10000" "pass"

    # Critical consensus parameters
    run_test "consensus" "timeout-propose-default" \
        "Test default timeout propose" \
        "config" "consensus.timeout_propose" '"3s"' "pass"

    run_test "consensus" "timeout-commit-default" \
        "Test default timeout commit" \
        "config" "consensus.timeout_commit" '"5s"' "pass"

    # Critical app parameters
    run_test "app-base" "minimum-gas-prices-default" \
        "Test default minimum gas prices" \
        "app" "minimum-gas-prices" '"0.001upaw"' "pass"

    run_test "app-base" "pruning-default" \
        "Test default pruning strategy" \
        "app" "pruning" '"default"' "pass"

    run_test "app-base" "iavl-cache-size-default" \
        "Test default IAVL cache size" \
        "app" "iavl-cache-size" "781250" "pass"
}

# ============================================================================
# Cleanup Functions
# ============================================================================

cleanup_all() {
    log "Cleaning up test directories..."
    if [[ -d "$TEST_BASE_DIR" ]]; then
        rm -rf "$TEST_BASE_DIR"
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --quick)
                QUICK_MODE=1
                shift
                ;;
            --category)
                CATEGORY_FILTER="$2"
                shift 2
                ;;
            --continue-on-error)
                CONTINUE_ON_ERROR=1
                shift
                ;;
            --skip-cleanup)
                SKIP_CLEANUP=1
                shift
                ;;
            --report)
                REPORT_FILE="$2"
                shift 2
                ;;
            -h|--help)
                head -n 30 "$0" | grep "^#" | sed 's/^# \?//'
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 2
                ;;
        esac
    done
}

main() {
    parse_args "$@"

    log "PAW Configuration Testing - Phase 2.2"
    log "======================================"
    log ""
    log "Test directory: ${TEST_BASE_DIR}"
    log "Report file: ${REPORT_FILE}"
    log ""

    # Pre-flight checks
    check_dependencies
    build_pawd

    # Create test infrastructure
    mkdir -p "$TEST_BASE_DIR"
    init_report

    # Run tests
    if [[ $QUICK_MODE -eq 1 ]]; then
        run_quick_tests
    else
        run_all_tests
    fi

    # Finalize report
    update_report_summary

    # Cleanup
    if [[ $SKIP_CLEANUP -eq 0 ]]; then
        cleanup_all
    else
        log_warning "Test directories preserved at: ${TEST_BASE_DIR}"
    fi

    # Print summary
    echo ""
    log "======================================"
    log "Test Summary"
    log "======================================"
    log "Total tests:    ${TOTAL_TESTS}"
    log_success "Passed:         ${PASSED_TESTS}"
    log_error "Failed:         ${FAILED_TESTS}"
    log_warning "Skipped:        ${SKIPPED_TESTS}"
    log_warning "Blocked:        ${BLOCKED_TESTS}"
    log ""
    log "Detailed report: ${REPORT_FILE}"
    log ""

    # Exit with appropriate code
    if [[ $FAILED_TESTS -gt 0 ]]; then
        log_error "Some tests failed. Review the report for details."
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

# Trap to ensure cleanup on exit
trap 'cleanup_all' EXIT

# Run main function
main "$@"
