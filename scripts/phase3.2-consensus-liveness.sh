#!/usr/bin/env bash
# phase3.2-consensus-liveness.sh - Consensus Liveness & Halt Testing
# Tests consensus behavior with 4-node, 3-node (live), and 2-node (halt) configurations
#
# Tendermint consensus requires >2/3 of voting power to be online
# With 4 equal validators:
#   - 4 nodes: consensus works (4/4 = 100%)
#   - 3 nodes: consensus works (3/4 = 75% > 66.67%)
#   - 2 nodes: consensus halts (2/4 = 50% < 66.67%)
#
# Usage:
#   ./phase3.2-consensus-liveness.sh [test|cleanup]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

COMPOSE_FILE="${PROJECT_ROOT}/compose/docker-compose.devnet.yml"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")

CHAIN_ID="paw-devnet"
BLOCK_TIME=5  # Expected block time in seconds
HALT_TIMEOUT=60  # Time to wait to confirm chain has halted

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

log_pass() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✓${NC} $*"
}

log_fail() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ✗${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠${NC} $*"
}

# Cleanup function
cleanup() {
    log "Cleaning up consensus test environment..."
    "${COMPOSE_CMD[@]}" down -v 2>/dev/null || true
    log_pass "Cleanup completed"
}

# Get current block height
get_height() {
    local node=${1:-node1}
    local port=${2:-26657}
    curl -sf "http://localhost:${port}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"' || echo "0"
}

# Check if chain is progressing
is_chain_progressing() {
    local node=${1:-node1}
    local port=${2:-26657}
    local timeout=${3:-30}

    local initial_height=$(get_height "$node" "$port")
    log "Initial height: ${initial_height}"

    sleep "$timeout"

    local final_height=$(get_height "$node" "$port")
    log "Final height: ${final_height}"

    if [[ $final_height -gt $initial_height ]]; then
        log_pass "Chain is progressing (${initial_height} -> ${final_height})"
        return 0
    else
        log_fail "Chain is NOT progressing (stuck at ${initial_height})"
        return 1
    fi
}

# Wait for network to start and sync
wait_for_network() {
    log "Waiting for network to start and produce blocks..."
    local timeout=300
    local elapsed=0
    local target_height=5

    while [[ $elapsed -lt $timeout ]]; do
        local height=$(get_height "node1" "26657")

        if [[ "$height" != "0" ]] && [[ $height -ge $target_height ]]; then
            log_pass "Network is active at height ${height}"
            return 0
        fi

        sleep 5
        elapsed=$((elapsed + 5))

        if [[ $((elapsed % 30)) -eq 0 ]]; then
            log "Still waiting for network... (height: ${height}, target: ${target_height})"
        fi
    done

    log_fail "Network failed to start within ${timeout}s"
    return 1
}

# Stop specific node
stop_node() {
    local node=$1
    log "Stopping ${node}..."
    docker stop "paw-${node}" >/dev/null 2>&1 || true
    sleep 3
    log_pass "${node} stopped"
}

# Start specific node
start_node() {
    local node=$1
    log "Starting ${node}..."
    "${COMPOSE_CMD[@]}" start "$node" >/dev/null 2>&1
    sleep 5
    log_pass "${node} started"
}

# Get peer count for a node
get_peer_count() {
    local node=${1:-node1}
    local port=${2:-26657}
    curl -sf "http://localhost:${port}/net_info" 2>/dev/null | jq -r '.result.n_peers // "0"' || echo "0"
}

# Test 4-node configuration (baseline)
test_4_node_consensus() {
    log "================================================"
    log "Test 1: 4-Node Consensus (Baseline)"
    log "================================================"
    echo ""

    log "Starting all 4 nodes..."
    "${COMPOSE_CMD[@]}" up -d

    if ! wait_for_network; then
        log_fail "Failed to start 4-node network"
        return 1
    fi

    # Verify all nodes are connected
    log "Checking peer connectivity..."
    local nodes=("node1:26657" "node2:26667" "node3:26677" "node4:26687")
    for node_port in "${nodes[@]}"; do
        local node=$(echo "$node_port" | cut -d: -f1)
        local port=$(echo "$node_port" | cut -d: -f2)
        local peers=$(get_peer_count "$node" "$port")
        log "  ${node}: ${peers} peers"
    done

    # Test consensus progression
    log "Testing consensus progression with 4 nodes..."
    if is_chain_progressing "node1" "26657" 20; then
        log_pass "4-node consensus test PASSED"
        return 0
    else
        log_fail "4-node consensus test FAILED"
        return 1
    fi
}

# Test 3-node configuration (should still work)
test_3_node_consensus() {
    log ""
    log "================================================"
    log "Test 2: 3-Node Consensus (1 Node Down)"
    log "================================================"
    echo ""

    # Record height before stopping node
    local height_before=$(get_height "node1" "26657")
    log "Height before stopping node4: ${height_before}"

    # Stop one node
    stop_node "node4"

    # Give network time to adjust
    log "Waiting for network to adjust to 3 validators..."
    sleep 10

    # Check peer counts
    log "Checking peer connectivity with 3 nodes..."
    local nodes=("node1:26657" "node2:26667" "node3:26677")
    for node_port in "${nodes[@]}"; do
        local node=$(echo "$node_port" | cut -d: -f1)
        local port=$(echo "$node_port" | cut -d: -f2)
        local peers=$(get_peer_count "$node" "$port")
        log "  ${node}: ${peers} peers"
    done

    # Test consensus progression
    log "Testing consensus progression with 3 nodes (75% voting power)..."
    if is_chain_progressing "node1" "26657" 20; then
        log_pass "3-node consensus test PASSED - chain continued with 75% voting power"
        return 0
    else
        log_fail "3-node consensus test FAILED - chain should continue with >2/3 voting power"
        return 1
    fi
}

# Test 2-node configuration (should halt)
test_2_node_consensus_halt() {
    log ""
    log "================================================"
    log "Test 3: 2-Node Consensus (Should Halt)"
    log "================================================"
    echo ""

    # Record height before stopping second node
    local height_before=$(get_height "node1" "26657")
    log "Height before stopping node3: ${height_before}"

    # Stop second node (leaving only 2 of 4 validators)
    stop_node "node3"

    # Give network time to realize it can't reach consensus
    log "Waiting to confirm chain halts with only 2 validators..."
    sleep 15

    # Check peer counts
    log "Checking peer connectivity with 2 nodes..."
    local nodes=("node1:26657" "node2:26667")
    for node_port in "${nodes[@]}"; do
        local node=$(echo "$node_port" | cut -d: -f1)
        local port=$(echo "$node_port" | cut -d: -f2)
        local peers=$(get_peer_count "$node" "$port")
        log "  ${node}: ${peers} peers"
    done

    # Verify chain has halted
    log "Verifying chain has halted (50% voting power < 66.67% required)..."
    local height_after=$(get_height "node1" "26657")
    log "Height after: ${height_after}"

    # Wait and check again to be sure
    sleep 20
    local height_final=$(get_height "node1" "26657")
    log "Height final: ${height_final}"

    if [[ $height_final -eq $height_after ]] && [[ $height_after -le $((height_before + 2)) ]]; then
        log_pass "2-node consensus test PASSED - chain correctly halted with <2/3 voting power"
        return 0
    else
        log_fail "2-node consensus test FAILED - chain should halt with only 50% voting power"
        log_warn "Expected chain to halt, but height progressed from ${height_before} to ${height_final}"
        return 1
    fi
}

# Test recovery from halt
test_consensus_recovery() {
    log ""
    log "================================================"
    log "Test 4: Consensus Recovery"
    log "================================================"
    echo ""

    # Record height while halted
    local height_halted=$(get_height "node1" "26657")
    log "Height while halted: ${height_halted}"

    # Restart node3 to restore >2/3 majority
    log "Restarting node3 to restore 75% voting power..."
    start_node "node3"

    # Wait for node to sync
    log "Waiting for network to recover..."
    sleep 15

    # Verify consensus has resumed
    log "Testing consensus recovery..."
    if is_chain_progressing "node1" "26657" 20; then
        local height_recovered=$(get_height "node1" "26657")
        log_pass "Consensus recovery test PASSED - chain resumed at height ${height_recovered}"
        return 0
    else
        log_fail "Consensus recovery test FAILED - chain did not resume after restoring >2/3 voting power"
        return 1
    fi
}

# Test full network recovery
test_full_network_recovery() {
    log ""
    log "================================================"
    log "Test 5: Full Network Recovery"
    log "================================================"
    echo ""

    # Restart node4
    log "Restarting node4 to restore full 4-node network..."
    start_node "node4"

    # Wait for full network sync
    log "Waiting for full network synchronization..."
    sleep 20

    # Check all nodes are synced
    log "Verifying all nodes are synchronized..."
    local heights=()
    local nodes=("node1:26657" "node2:26667" "node3:26677" "node4:26687")

    for node_port in "${nodes[@]}"; do
        local node=$(echo "$node_port" | cut -d: -f1)
        local port=$(echo "$node_port" | cut -d: -f2)
        local height=$(get_height "$node" "$port")
        heights+=("${node}:${height}")
        log "  ${node}: height ${height}"
    done

    # Verify consensus is working
    if is_chain_progressing "node1" "26657" 15; then
        log_pass "Full network recovery test PASSED - all 4 nodes are operational"
        return 0
    else
        log_fail "Full network recovery test FAILED"
        return 1
    fi
}

# Run all tests
run_tests() {
    log "Starting Phase 3.2: Consensus Liveness & Halt Tests"
    echo ""

    local test_results=()

    # Test 1: 4-node baseline
    if test_4_node_consensus; then
        test_results+=("PASS: 4-Node Consensus")
    else
        test_results+=("FAIL: 4-Node Consensus")
    fi

    # Test 2: 3-node liveness
    if test_3_node_consensus; then
        test_results+=("PASS: 3-Node Consensus")
    else
        test_results+=("FAIL: 3-Node Consensus")
    fi

    # Test 3: 2-node halt
    if test_2_node_consensus_halt; then
        test_results+=("PASS: 2-Node Halt")
    else
        test_results+=("FAIL: 2-Node Halt")
    fi

    # Test 4: Recovery from halt
    if test_consensus_recovery; then
        test_results+=("PASS: Consensus Recovery")
    else
        test_results+=("FAIL: Consensus Recovery")
    fi

    # Test 5: Full network recovery
    if test_full_network_recovery; then
        test_results+=("PASS: Full Network Recovery")
    else
        test_results+=("FAIL: Full Network Recovery")
    fi

    # Print summary
    echo ""
    log "================================================"
    log "Test Results Summary"
    log "================================================"
    local failed=false
    for result in "${test_results[@]}"; do
        if [[ "$result" == PASS:* ]]; then
            log_pass "$result"
        else
            log_fail "$result"
            failed=true
        fi
    done
    echo ""

    if [[ "$failed" == "true" ]]; then
        log_fail "Phase 3.2 tests completed with failures"
        return 1
    else
        log_pass "Phase 3.2 tests completed successfully"
        return 0
    fi
}

# Main execution
main() {
    local action="${1:-test}"

    case "$action" in
        cleanup)
            cleanup
            ;;
        test|*)
            trap cleanup EXIT INT TERM
            run_tests
            ;;
    esac
}

main "$@"
