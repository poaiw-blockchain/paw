#!/usr/bin/env bash
# PAW CLI Command Verification Script - Phase 2.3
# Tests EVERY CLI command with both valid and invalid parameters
# Generates a comprehensive test report

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Report file
REPORT_FILE="cli-test-report-$(date +%Y%m%d-%H%M%S).txt"
TEMP_DIR="/tmp/paw-cli-test-$$"
TEST_HOME="$TEMP_DIR/home"
BINARY="${BINARY:-./pawd}"

# Test configuration
CHAIN_ID="paw-cli-test"
TEST_KEY_NAME="cli-test-key"
TEST_MNEMONIC="abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"

# Store test results
declare -A TEST_RESULTS
declare -A TEST_DETAILS

# ============================================================================
# Helper Functions
# ============================================================================

log() {
    echo -e "${CYAN}[INFO]${NC} $*" | tee -a "$REPORT_FILE"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $*" | tee -a "$REPORT_FILE"
}

error() {
    echo -e "${RED}[FAIL]${NC} $*" | tee -a "$REPORT_FILE"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" | tee -a "$REPORT_FILE"
}

skip() {
    echo -e "${BLUE}[SKIP]${NC} $*" | tee -a "$REPORT_FILE"
}

section() {
    echo "" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}========================================${NC}" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}$*${NC}" | tee -a "$REPORT_FILE"
    echo -e "${CYAN}========================================${NC}" | tee -a "$REPORT_FILE"
}

# Run a test command and record results
# Args: test_name, expected_result (pass/fail), command...
run_test() {
    local test_name="$1"
    local expected_result="$2"
    shift 2
    local cmd=("$@")

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local output
    local exit_code

    # Run command and capture output
    set +e  # Temporarily disable exit on error
    output=$("${cmd[@]}" 2>&1)
    exit_code=$?
    set -e  # Re-enable exit on error

    # Determine if test passed based on expectation
    local test_passed=false
    if [[ "$expected_result" == "pass" && $exit_code -eq 0 ]]; then
        test_passed=true
    elif [[ "$expected_result" == "fail" && $exit_code -ne 0 ]]; then
        test_passed=true
    fi

    # Record result
    if [[ "$test_passed" == true ]]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        success "$test_name"
        TEST_RESULTS["$test_name"]="PASS"
        TEST_DETAILS["$test_name"]="Expected: $expected_result, Got: exit=$exit_code"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        error "$test_name"
        TEST_RESULTS["$test_name"]="FAIL"
        TEST_DETAILS["$test_name"]="Expected: $expected_result, Got: exit=$exit_code, Output: ${output:0:200}"
        echo "  Output: ${output:0:500}" >> "$REPORT_FILE"
    fi
}

# Test if help text is shown correctly
test_help() {
    local test_name="$1"
    shift
    local cmd=("$@")

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    local output
    local exit_code
    set +e  # Temporarily disable exit on error
    output=$("${cmd[@]}" --help 2>&1)
    exit_code=$?
    set -e  # Re-enable exit on error

    if [[ $exit_code -eq 0 ]] && echo "$output" | grep -qE "(Usage:|Available Commands:|Flags:)"; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        success "$test_name - help text"
        TEST_RESULTS["$test_name"]="PASS"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        error "$test_name - help text missing or malformed"
        TEST_RESULTS["$test_name"]="FAIL"
    fi
}

# ============================================================================
# Setup and Teardown
# ============================================================================

setup_test_environment() {
    section "Setting Up Test Environment"

    # Create temp directories
    mkdir -p "$TEMP_DIR"
    mkdir -p "$TEST_HOME"

    log "Temporary home: $TEST_HOME"
    log "Binary: $BINARY"

    # Check if binary exists
    if [[ ! -x "$BINARY" ]]; then
        if [[ -f "./pawd" ]]; then
            BINARY="./pawd"
        elif command -v pawd >/dev/null 2>&1; then
            BINARY=$(command -v pawd)
        else
            error "pawd binary not found. Please build it first or set BINARY env var."
            exit 1
        fi
    fi

    log "Using binary: $BINARY"

    # Initialize a test chain (keyring only, no node needed for CLI tests)
    export HOME="$TEST_HOME"

    # Add test key
    echo "$TEST_MNEMONIC" | $BINARY keys add "$TEST_KEY_NAME" \
        --recover \
        --keyring-backend test \
        --home "$TEST_HOME" \
        >/dev/null 2>&1 || true

    TEST_ADDRESS=$($BINARY keys show "$TEST_KEY_NAME" -a --keyring-backend test --home "$TEST_HOME" 2>/dev/null || echo "paw1test")

    log "Test key: $TEST_KEY_NAME"
    log "Test address: $TEST_ADDRESS"

    success "Test environment ready"
}

cleanup_test_environment() {
    section "Cleaning Up Test Environment"

    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
        log "Removed temporary directory: $TEMP_DIR"
    fi

    success "Cleanup complete"
}

# ============================================================================
# Core Command Tests
# ============================================================================

test_version_and_help() {
    section "Testing Core Commands: version, help, config"

    # Version
    test_help "pawd version" "$BINARY" version
    run_test "pawd version" "pass" "$BINARY" version

    # Help
    run_test "pawd --help" "pass" "$BINARY" --help
    run_test "pawd help" "pass" "$BINARY" help

    # Config
    test_help "pawd config" "$BINARY" config
    run_test "pawd config" "pass" "$BINARY" config --home "$TEST_HOME"
}

test_keys_commands() {
    section "Testing Keys Commands"

    # Help texts
    test_help "keys" "$BINARY" keys
    test_help "keys list" "$BINARY" keys list
    test_help "keys show" "$BINARY" keys show
    test_help "keys add" "$BINARY" keys add
    test_help "keys delete" "$BINARY" keys delete
    test_help "keys export" "$BINARY" keys export
    test_help "keys import" "$BINARY" keys import

    # Valid operations
    run_test "keys list" "pass" "$BINARY" keys list --keyring-backend test --home "$TEST_HOME"
    run_test "keys show existing" "pass" "$BINARY" keys show "$TEST_KEY_NAME" --keyring-backend test --home "$TEST_HOME"

    # Invalid operations
    run_test "keys show nonexistent" "fail" "$BINARY" keys show "nonexistent-key-12345" --keyring-backend test --home "$TEST_HOME"
    run_test "keys add without name" "fail" "$BINARY" keys add --keyring-backend test --home "$TEST_HOME"
    run_test "keys delete nonexistent" "fail" "$BINARY" keys delete "nonexistent-key-12345" --keyring-backend test --home "$TEST_HOME" --yes
}

test_init_gentx_commands() {
    section "Testing Init and Gentx Commands"

    # Help texts
    test_help "init" "$BINARY" init
    test_help "gentx" "$BINARY" gentx
    test_help "collect-gentxs" "$BINARY" collect-gentxs
    test_help "validate-genesis" "$BINARY" validate-genesis

    # Init tests (use separate temp dir to avoid conflicts)
    local init_home="$TEMP_DIR/init-test"
    mkdir -p "$init_home"

    run_test "init with moniker" "pass" "$BINARY" init "test-moniker" --chain-id "$CHAIN_ID" --home "$init_home"
    run_test "init without moniker" "fail" "$BINARY" init --chain-id "$CHAIN_ID" --home "$TEMP_DIR/init-fail"
    run_test "init without chain-id" "fail" "$BINARY" init "test" --home "$TEMP_DIR/init-fail2"
}

# ============================================================================
# Query Command Tests
# ============================================================================

test_query_commands() {
    section "Testing Query Commands"

    # Note: These will fail without a running node, but we test CLI parsing
    test_help "query" "$BINARY" query
    test_help "query bank" "$BINARY" query bank
    test_help "query staking" "$BINARY" query staking
    test_help "query gov" "$BINARY" query gov
    test_help "query distribution" "$BINARY" query distribution
    test_help "query slashing" "$BINARY" query slashing
}

# ============================================================================
# DEX Module Tests
# ============================================================================

test_dex_query_commands() {
    section "Testing DEX Query Commands"

    # Help texts
    test_help "query dex" "$BINARY" query dex
    test_help "query dex params" "$BINARY" query dex params
    test_help "query dex pool" "$BINARY" query dex pool
    test_help "query dex pools" "$BINARY" query dex pools
    test_help "query dex pool-by-tokens" "$BINARY" query dex pool-by-tokens
    test_help "query dex liquidity" "$BINARY" query dex liquidity
    test_help "query dex simulate-swap" "$BINARY" query dex simulate-swap
    test_help "query dex limit-order" "$BINARY" query dex limit-order
    test_help "query dex limit-orders" "$BINARY" query dex limit-orders
    test_help "query dex orders-by-owner" "$BINARY" query dex orders-by-owner
    test_help "query dex orders-by-pool" "$BINARY" query dex orders-by-pool
    test_help "query dex order-book" "$BINARY" query dex order-book

    # Invalid parameters (CLI parsing checks)
    run_test "query dex pool without ID" "fail" "$BINARY" query dex pool --home "$TEST_HOME"
    run_test "query dex pool with invalid ID" "fail" "$BINARY" query dex pool "invalid" --home "$TEST_HOME"
    run_test "query dex pool-by-tokens missing token" "fail" "$BINARY" query dex pool-by-tokens upaw --home "$TEST_HOME"
    run_test "query dex simulate-swap missing args" "fail" "$BINARY" query dex simulate-swap 1 upaw --home "$TEST_HOME"
    run_test "query dex liquidity missing args" "fail" "$BINARY" query dex liquidity 1 --home "$TEST_HOME"
}

test_dex_tx_commands() {
    section "Testing DEX Transaction Commands"

    # Help texts
    test_help "tx dex" "$BINARY" tx dex
    test_help "tx dex create-pool" "$BINARY" tx dex create-pool
    test_help "tx dex add-liquidity" "$BINARY" tx dex add-liquidity
    test_help "tx dex remove-liquidity" "$BINARY" tx dex remove-liquidity
    test_help "tx dex swap" "$BINARY" tx dex swap

    # Invalid parameters (CLI parsing checks - use --generate-only to avoid node requirement)
    run_test "tx dex create-pool missing args" "fail" "$BINARY" tx dex create-pool upaw --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex create-pool same tokens" "fail" "$BINARY" tx dex create-pool upaw 1000 upaw 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex create-pool negative amount" "fail" "$BINARY" tx dex create-pool upaw -1000 uatom 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex create-pool zero amount" "fail" "$BINARY" tx dex create-pool upaw 0 uatom 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex create-pool invalid amount" "fail" "$BINARY" tx dex create-pool upaw "abc" uatom 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    run_test "tx dex add-liquidity missing pool ID" "fail" "$BINARY" tx dex add-liquidity --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex add-liquidity invalid pool ID" "fail" "$BINARY" tx dex add-liquidity "invalid" 1000 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex add-liquidity negative amount" "fail" "$BINARY" tx dex add-liquidity 1 -1000 1000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    run_test "tx dex remove-liquidity missing args" "fail" "$BINARY" tx dex remove-liquidity 1 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex remove-liquidity zero shares" "fail" "$BINARY" tx dex remove-liquidity 1 0 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex remove-liquidity negative shares" "fail" "$BINARY" tx dex remove-liquidity 1 -100 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    run_test "tx dex swap missing args" "fail" "$BINARY" tx dex swap 1 upaw 1000 uatom --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex swap same tokens" "fail" "$BINARY" tx dex swap 1 upaw 1000 upaw 900 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex swap negative amount" "fail" "$BINARY" tx dex swap 1 upaw -1000 uatom 0 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx dex swap negative min-out" "fail" "$BINARY" tx dex swap 1 upaw 1000 uatom -1 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

# ============================================================================
# Oracle Module Tests
# ============================================================================

test_oracle_query_commands() {
    section "Testing Oracle Query Commands"

    # Help texts
    test_help "query oracle" "$BINARY" query oracle
    test_help "query oracle params" "$BINARY" query oracle params
    test_help "query oracle price" "$BINARY" query oracle price
    test_help "query oracle prices" "$BINARY" query oracle prices
    test_help "query oracle validator" "$BINARY" query oracle validator
    test_help "query oracle validators" "$BINARY" query oracle validators
    test_help "query oracle validator-price" "$BINARY" query oracle validator-price

    # Invalid parameters
    run_test "query oracle price missing asset" "fail" "$BINARY" query oracle price --home "$TEST_HOME"
    run_test "query oracle validator missing address" "fail" "$BINARY" query oracle validator --home "$TEST_HOME"
    run_test "query oracle validator-price missing args" "fail" "$BINARY" query oracle validator-price pawvaloper1test --home "$TEST_HOME"
}

test_oracle_tx_commands() {
    section "Testing Oracle Transaction Commands"

    # Help texts
    test_help "tx oracle" "$BINARY" tx oracle
    test_help "tx oracle submit-price" "$BINARY" tx oracle submit-price
    test_help "tx oracle delegate-feeder" "$BINARY" tx oracle delegate-feeder

    # Invalid parameters
    run_test "tx oracle submit-price missing args" "fail" "$BINARY" tx oracle submit-price pawvaloper1test BTC --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx oracle submit-price invalid validator" "fail" "$BINARY" tx oracle submit-price "invalid-addr" BTC 50000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx oracle submit-price invalid price" "fail" "$BINARY" tx oracle submit-price pawvaloper1test BTC "invalid" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx oracle submit-price negative price" "fail" "$BINARY" tx oracle submit-price pawvaloper1test BTC -100 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx oracle submit-price zero price" "fail" "$BINARY" tx oracle submit-price pawvaloper1test BTC 0 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    run_test "tx oracle delegate-feeder missing args" "fail" "$BINARY" tx oracle delegate-feeder --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx oracle delegate-feeder invalid address" "fail" "$BINARY" tx oracle delegate-feeder "invalid-addr" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

# ============================================================================
# Compute Module Tests
# ============================================================================

test_compute_query_commands() {
    section "Testing Compute Query Commands"

    # Help texts
    test_help "query compute" "$BINARY" query compute
    test_help "query compute params" "$BINARY" query compute params
    test_help "query compute provider" "$BINARY" query compute provider
    test_help "query compute providers" "$BINARY" query compute providers
    test_help "query compute active-providers" "$BINARY" query compute active-providers
    test_help "query compute request" "$BINARY" query compute request
    test_help "query compute requests" "$BINARY" query compute requests
    test_help "query compute requests-by-requester" "$BINARY" query compute requests-by-requester
    test_help "query compute requests-by-provider" "$BINARY" query compute requests-by-provider
    test_help "query compute requests-by-status" "$BINARY" query compute requests-by-status
    test_help "query compute result" "$BINARY" query compute result
    test_help "query compute estimate-cost" "$BINARY" query compute estimate-cost
    test_help "query compute dispute" "$BINARY" query compute dispute
    test_help "query compute disputes" "$BINARY" query compute disputes
    test_help "query compute disputes-by-request" "$BINARY" query compute disputes-by-request
    test_help "query compute disputes-by-status" "$BINARY" query compute disputes-by-status
    test_help "query compute evidence" "$BINARY" query compute evidence
    test_help "query compute slash-record" "$BINARY" query compute slash-record
    test_help "query compute slash-records" "$BINARY" query compute slash-records
    test_help "query compute slash-records-by-provider" "$BINARY" query compute slash-records-by-provider
    test_help "query compute appeal" "$BINARY" query compute appeal
    test_help "query compute appeals" "$BINARY" query compute appeals
    test_help "query compute appeals-by-status" "$BINARY" query compute appeals-by-status
    test_help "query compute governance-params" "$BINARY" query compute governance-params

    # Invalid parameters
    run_test "query compute provider missing address" "fail" "$BINARY" query compute provider --home "$TEST_HOME"
    run_test "query compute request missing ID" "fail" "$BINARY" query compute request --home "$TEST_HOME"
    run_test "query compute request invalid ID" "fail" "$BINARY" query compute request "invalid" --home "$TEST_HOME"
    run_test "query compute requests-by-status missing status" "fail" "$BINARY" query compute requests-by-status --home "$TEST_HOME"
    run_test "query compute requests-by-status invalid status" "fail" "$BINARY" query compute requests-by-status "invalid-status" --home "$TEST_HOME"
    run_test "query compute disputes-by-status invalid status" "fail" "$BINARY" query compute disputes-by-status "invalid-status" --home "$TEST_HOME"
}

test_compute_tx_commands() {
    section "Testing Compute Transaction Commands"

    # Help texts
    test_help "tx compute" "$BINARY" tx compute
    test_help "tx compute register-provider" "$BINARY" tx compute register-provider
    test_help "tx compute update-provider" "$BINARY" tx compute update-provider
    test_help "tx compute deactivate-provider" "$BINARY" tx compute deactivate-provider
    test_help "tx compute submit-request" "$BINARY" tx compute submit-request
    test_help "tx compute cancel-request" "$BINARY" tx compute cancel-request
    test_help "tx compute submit-result" "$BINARY" tx compute submit-result
    test_help "tx compute create-dispute" "$BINARY" tx compute create-dispute
    test_help "tx compute vote-dispute" "$BINARY" tx compute vote-dispute
    test_help "tx compute submit-evidence" "$BINARY" tx compute submit-evidence
    test_help "tx compute appeal-slashing" "$BINARY" tx compute appeal-slashing
    test_help "tx compute vote-appeal" "$BINARY" tx compute vote-appeal
    test_help "tx compute resolve-dispute" "$BINARY" tx compute resolve-dispute
    test_help "tx compute resolve-appeal" "$BINARY" tx compute resolve-appeal
    test_help "tx compute update-governance-params" "$BINARY" tx compute update-governance-params

    # Register provider - missing required flags
    run_test "tx compute register-provider missing moniker" "fail" "$BINARY" tx compute register-provider --endpoint "http://test.com" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute register-provider missing endpoint" "fail" "$BINARY" tx compute register-provider --moniker "test" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Submit request - missing required flags
    run_test "tx compute submit-request missing container" "fail" "$BINARY" tx compute submit-request --max-payment 1000000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute submit-request missing max-payment" "fail" "$BINARY" tx compute submit-request --container-image "ubuntu:22.04" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Cancel request - missing/invalid ID
    run_test "tx compute cancel-request missing ID" "fail" "$BINARY" tx compute cancel-request --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute cancel-request invalid ID" "fail" "$BINARY" tx compute cancel-request "invalid" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Submit result - missing required flags
    run_test "tx compute submit-result missing ID" "fail" "$BINARY" tx compute submit-result --output-hash "abc123" --output-url "http://test.com" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute submit-result missing output-hash" "fail" "$BINARY" tx compute submit-result 1 --output-url "http://test.com" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute submit-result missing output-url" "fail" "$BINARY" tx compute submit-result 1 --output-hash "abc123" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Create dispute - missing required flags
    run_test "tx compute create-dispute missing ID" "fail" "$BINARY" tx compute create-dispute --reason "test" --deposit-amount 1000000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute create-dispute missing reason" "fail" "$BINARY" tx compute create-dispute 1 --deposit-amount 1000000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute create-dispute missing deposit" "fail" "$BINARY" tx compute create-dispute 1 --reason "test" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Vote on dispute - invalid vote option
    run_test "tx compute vote-dispute missing ID" "fail" "$BINARY" tx compute vote-dispute --vote "provider_fault" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute vote-dispute missing vote" "fail" "$BINARY" tx compute vote-dispute 1 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute vote-dispute invalid vote" "fail" "$BINARY" tx compute vote-dispute 1 --vote "invalid_option" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Submit evidence - missing required flags
    run_test "tx compute submit-evidence missing ID" "fail" "$BINARY" tx compute submit-evidence --evidence "/tmp/test.json" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute submit-evidence missing evidence" "fail" "$BINARY" tx compute submit-evidence 1 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Appeal slashing - missing required flags
    run_test "tx compute appeal-slashing missing ID" "fail" "$BINARY" tx compute appeal-slashing --justification "test" --deposit-amount 1000000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute appeal-slashing missing justification" "fail" "$BINARY" tx compute appeal-slashing 1 --deposit-amount 1000000 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute appeal-slashing missing deposit" "fail" "$BINARY" tx compute appeal-slashing 1 --justification "test" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"

    # Vote on appeal - invalid vote
    run_test "tx compute vote-appeal missing ID" "fail" "$BINARY" tx compute vote-appeal --vote "approve" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute vote-appeal missing vote" "fail" "$BINARY" tx compute vote-appeal 1 --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx compute vote-appeal invalid vote" "fail" "$BINARY" tx compute vote-appeal 1 --vote "invalid_option" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

# ============================================================================
# Standard Cosmos SDK Module Tests
# ============================================================================

test_bank_commands() {
    section "Testing Bank Module Commands"

    # Help texts
    test_help "query bank" "$BINARY" query bank
    test_help "query bank balances" "$BINARY" query bank balances
    test_help "query bank total" "$BINARY" query bank total
    test_help "tx bank" "$BINARY" tx bank
    test_help "tx bank send" "$BINARY" tx bank send

    # Invalid parameters
    run_test "query bank balances missing address" "fail" "$BINARY" query bank balances --home "$TEST_HOME"
    run_test "tx bank send missing args" "fail" "$BINARY" tx bank send --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
    run_test "tx bank send invalid amount" "fail" "$BINARY" tx bank send "$TEST_ADDRESS" "$TEST_ADDRESS" "invalid" --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

test_staking_commands() {
    section "Testing Staking Module Commands"

    # Help texts
    test_help "query staking" "$BINARY" query staking
    test_help "query staking validators" "$BINARY" query staking validators
    test_help "query staking validator" "$BINARY" query staking validator
    test_help "query staking delegation" "$BINARY" query staking delegation
    test_help "tx staking" "$BINARY" tx staking
    test_help "tx staking delegate" "$BINARY" tx staking delegate
    test_help "tx staking unbond" "$BINARY" tx staking unbond
    test_help "tx staking redelegate" "$BINARY" tx staking redelegate

    # Invalid parameters
    run_test "query staking validator missing address" "fail" "$BINARY" query staking validator --home "$TEST_HOME"
    run_test "tx staking delegate missing args" "fail" "$BINARY" tx staking delegate --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

test_gov_commands() {
    section "Testing Governance Module Commands"

    # Help texts
    test_help "query gov" "$BINARY" query gov
    test_help "query gov proposals" "$BINARY" query gov proposals
    test_help "query gov proposal" "$BINARY" query gov proposal
    test_help "tx gov" "$BINARY" tx gov
    test_help "tx gov submit-proposal" "$BINARY" tx gov submit-proposal
    test_help "tx gov vote" "$BINARY" tx gov vote
    test_help "tx gov deposit" "$BINARY" tx gov deposit

    # Invalid parameters
    run_test "query gov proposal missing ID" "fail" "$BINARY" query gov proposal --home "$TEST_HOME"
    run_test "tx gov vote missing args" "fail" "$BINARY" tx gov vote --from "$TEST_KEY_NAME" --generate-only --home "$TEST_HOME"
}

# ============================================================================
# Report Generation
# ============================================================================

generate_report() {
    section "Test Summary Report"

    echo "" | tee -a "$REPORT_FILE"
    echo "Total Tests:  $TOTAL_TESTS" | tee -a "$REPORT_FILE"
    echo "Passed:       $PASSED_TESTS ($(( PASSED_TESTS * 100 / TOTAL_TESTS ))%)" | tee -a "$REPORT_FILE"
    echo "Failed:       $FAILED_TESTS ($(( FAILED_TESTS * 100 / TOTAL_TESTS ))%)" | tee -a "$REPORT_FILE"
    echo "Skipped:      $SKIPPED_TESTS" | tee -a "$REPORT_FILE"
    echo "" | tee -a "$REPORT_FILE"

    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo -e "${RED}Failed Tests:${NC}" | tee -a "$REPORT_FILE"
        echo "" | tee -a "$REPORT_FILE"
        for test_name in "${!TEST_RESULTS[@]}"; do
            if [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
                echo "  - $test_name" | tee -a "$REPORT_FILE"
                echo "    ${TEST_DETAILS[$test_name]}" | tee -a "$REPORT_FILE"
            fi
        done
        echo "" | tee -a "$REPORT_FILE"
    fi

    echo "Full report saved to: $REPORT_FILE" | tee -a "$REPORT_FILE"

    if [[ $FAILED_TESTS -eq 0 ]]; then
        success "All tests passed!"
        return 0
    else
        error "Some tests failed. Review the report for details."
        return 1
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "PAW CLI Command Verification - Phase 2.3"
    echo "========================================"
    echo "Date: $(date)"
    echo "Report: $REPORT_FILE"
    echo ""

    # Trap to ensure cleanup
    trap cleanup_test_environment EXIT

    # Setup
    setup_test_environment

    # Run all test suites
    test_version_and_help
    test_keys_commands
    test_init_gentx_commands
    test_query_commands

    # Custom modules
    test_dex_query_commands
    test_dex_tx_commands
    test_oracle_query_commands
    test_oracle_tx_commands
    test_compute_query_commands
    test_compute_tx_commands

    # Standard modules
    test_bank_commands
    test_staking_commands
    test_gov_commands

    # Generate final report
    generate_report
}

# Run main function
main "$@"
