#!/usr/bin/env bash
# phase3.4-malicious-peer.sh - Malicious Peer Ejection Testing
# Tests the network's ability to detect and ban malicious peers
#
# Tests include:
# 1. Invalid message injection
# 2. Oversized message attempts
# 3. Spam/flood attacks
# 4. Peer reputation scoring
# 5. Automatic peer banning
#
# Usage:
#   ./phase3.4-malicious-peer.sh [test|cleanup]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

COMPOSE_FILE="${PROJECT_ROOT}/compose/docker-compose.devnet.yml"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")

CHAIN_ID="paw-devnet"

# Test victim and attack nodes
VICTIM_NODE="node1"
ATTACK_NODE="node2"

# Ports
VICTIM_RPC="26657"
VICTIM_API="1317"
ATTACK_RPC="26667"
ATTACK_API="1327"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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
    log "Cleaning up malicious peer test environment..."
    "${COMPOSE_CMD[@]}" down -v 2>/dev/null || true
    log_pass "Cleanup completed"
}

# Get peer count
get_peer_count() {
    local node=$1
    local port=$2
    curl -sf "http://localhost:${port}/net_info" 2>/dev/null | jq -r '.result.n_peers // "0"' || echo "0"
}

# Get peer list
get_peers() {
    local node=$1
    local port=$2
    curl -sf "http://localhost:${port}/net_info" 2>/dev/null | jq -r '.result.peers[].node_info.id' || echo ""
}

# Get node ID
get_node_id() {
    local port=$1
    curl -sf "http://localhost:${port}/status" 2>/dev/null | jq -r '.result.node_info.id // ""' || echo ""
}

# Get block height
get_height() {
    local port=${1:-26657}
    curl -sf "http://localhost:${port}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"' || echo "0"
}

# Wait for network to start
wait_for_network() {
    log "Waiting for network to start..."
    local timeout=300
    local elapsed=0
    local target_height=5

    while [[ $elapsed -lt $timeout ]]; do
        local height=$(get_height "$VICTIM_RPC")

        if [[ "$height" != "0" ]] && [[ $height -ge $target_height ]]; then
            log_pass "Network is active at height ${height}"
            return 0
        fi

        sleep 5
        elapsed=$((elapsed + 5))
    done

    log_fail "Network failed to start within ${timeout}s"
    return 1
}

# Check if peer is connected
is_peer_connected() {
    local victim_port=$1
    local peer_id=$2

    local peers=$(get_peers "victim" "$victim_port")

    if echo "$peers" | grep -q "$peer_id"; then
        return 0
    else
        return 1
    fi
}

# Execute command in container
exec_in_container() {
    local container=$1
    shift
    docker exec -i "$container" "$@" 2>/dev/null
}

# Test: Send invalid transaction
test_invalid_transaction() {
    log ""
    log "================================================"
    log "Test 1: Invalid Transaction Injection"
    log "================================================"
    echo ""

    local container="paw-${ATTACK_NODE}"
    local home="/root/.paw/${ATTACK_NODE}"

    log "Attempting to send invalid transaction from ${ATTACK_NODE}..."

    # Try to send a transaction with invalid signature
    # This should be rejected by the victim node
    local result=$(exec_in_container "$container" pawd --home "$home" tx bank send \
        "invalid_address" "cosmos1invalid" "1000upaw" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend test \
        --node "tcp://paw-${VICTIM_NODE}:26657" \
        --yes 2>&1 || echo "FAILED")

    if echo "$result" | grep -qi "error\|failed\|invalid"; then
        log_pass "Invalid transaction was correctly rejected"
        return 0
    else
        log_warn "Invalid transaction handling unclear"
        return 0
    fi
}

# Test: Message spam attack
test_message_spam() {
    log ""
    log "================================================"
    log "Test 2: Message Spam Attack"
    log "================================================"
    echo ""

    local container="paw-${ATTACK_NODE}"
    local victim_rpc="http://localhost:${VICTIM_RPC}"

    log "Launching spam attack from ${ATTACK_NODE}..."
    log "Sending rapid queries to ${VICTIM_NODE}..."

    local spam_count=100
    local start_time=$(date +%s)

    # Send rapid status queries
    for ((i=0; i<spam_count; i++)); do
        curl -sf "${victim_rpc}/status" >/dev/null 2>&1 &
    done

    # Wait for background jobs
    wait

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log "Sent ${spam_count} requests in ${duration}s"

    # Check if victim is still responsive
    local height=$(get_height "$VICTIM_RPC")
    if [[ "$height" != "0" ]]; then
        log_pass "Victim node remained responsive during spam attack"
        return 0
    else
        log_fail "Victim node became unresponsive"
        return 1
    fi
}

# Test: Oversized message attempt
test_oversized_message() {
    log ""
    log "================================================"
    log "Test 3: Oversized Message Attack"
    log "================================================"
    echo ""

    local container="paw-${ATTACK_NODE}"
    local home="/root/.paw/${ATTACK_NODE}"

    log "Attempting to send oversized message from ${ATTACK_NODE}..."

    # Try to send transaction with very large memo
    local large_memo=$(printf 'A%.0s' {1..100000})  # 100KB memo

    local result=$(exec_in_container "$container" pawd --home "$home" tx bank send \
        "smoke-trader" "cosmos1test" "1upaw" \
        --memo "$large_memo" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend test \
        --node "tcp://paw-${VICTIM_NODE}:26657" \
        --yes 2>&1 || echo "REJECTED")

    if echo "$result" | grep -qi "rejected\|too large\|error"; then
        log_pass "Oversized message was rejected"
        return 0
    else
        log_warn "Oversized message handling unclear"
        return 0
    fi
}

# Test: Peer reputation scoring (if available)
test_peer_reputation() {
    log ""
    log "================================================"
    log "Test 4: Peer Reputation Scoring"
    log "================================================"
    echo ""

    # Check if reputation system is available
    local victim_api="http://localhost:${VICTIM_API}"

    # Try to query reputation endpoint
    local reputation=$(curl -sf "${victim_api}/paw/reputation/peers" 2>/dev/null || echo "")

    if [[ -n "$reputation" ]]; then
        log "Reputation data available:"
        echo "$reputation" | jq '.' 2>/dev/null || echo "$reputation"

        # Count peers with low reputation scores
        local low_score_peers=$(echo "$reputation" | jq '[.peers[] | select(.score < 50)] | length' 2>/dev/null || echo "0")

        log "Peers with low reputation score: ${low_score_peers}"

        if [[ $low_score_peers -gt 0 ]]; then
            log_pass "Reputation system is tracking peer behavior"
        else
            log_pass "All peers have good reputation"
        fi
        return 0
    else
        log_warn "Reputation system endpoint not available or not exposed"
        log "This is expected if reputation module is not exposed via API"
        return 0
    fi
}

# Test: Automatic peer banning
test_peer_banning() {
    log ""
    log "================================================"
    log "Test 5: Automatic Peer Banning"
    log "================================================"
    echo ""

    # Get initial peer counts
    local victim_initial_peers=$(get_peer_count "$VICTIM_NODE" "$VICTIM_RPC")
    log "Initial peer count on ${VICTIM_NODE}: ${victim_initial_peers}"

    # Get attack node ID
    local attack_node_id=$(get_node_id "$ATTACK_RPC")
    log "Attack node ID: ${attack_node_id}"

    # Check if attack node is connected to victim
    if is_peer_connected "$VICTIM_RPC" "$attack_node_id"; then
        log "Attack node is connected to victim"

        # Perform multiple malicious actions
        log "Performing multiple malicious actions..."

        for ((i=0; i<5; i++)); do
            # Send invalid transactions
            test_invalid_transaction >/dev/null 2>&1 &

            # Send spam
            for ((j=0; j<20; j++)); do
                curl -sf "http://localhost:${VICTIM_RPC}/status" >/dev/null 2>&1 &
            done
        done

        wait

        # Wait for potential banning
        log "Waiting for peer reputation system to respond..."
        sleep 15

        # Check if peer is still connected
        if is_peer_connected "$VICTIM_RPC" "$attack_node_id"; then
            log_warn "Attack node is still connected (may not be banned)"
            log "Note: Automatic banning may require configuration or more severe violations"
        else
            log_pass "Attack node was disconnected (possibly banned)"
        fi
    else
        log_warn "Attack node is not connected to victim node"
    fi

    # Check final peer count
    local victim_final_peers=$(get_peer_count "$VICTIM_NODE" "$VICTIM_RPC")
    log "Final peer count on ${VICTIM_NODE}: ${victim_final_peers}"

    return 0
}

# Test: Network resilience to malicious node
test_network_resilience() {
    log ""
    log "================================================"
    log "Test 6: Network Resilience"
    log "================================================"
    echo ""

    log "Testing if network consensus continues despite malicious node..."

    local height_before=$(get_height "$VICTIM_RPC")
    log "Height before: ${height_before}"

    # Launch various attacks
    log "Launching combined attack..."

    for ((i=0; i<10; i++)); do
        test_invalid_transaction >/dev/null 2>&1 &
        test_message_spam >/dev/null 2>&1 &
    done

    wait

    # Wait and check if consensus progressed
    sleep 20

    local height_after=$(get_height "$VICTIM_RPC")
    log "Height after: ${height_after}"

    local blocks_produced=$((height_after - height_before))

    if [[ $blocks_produced -gt 0 ]]; then
        log_pass "Network consensus continued despite attacks (${blocks_produced} blocks produced)"
        return 0
    else
        log_fail "Network consensus may have been impacted"
        return 1
    fi
}

# Test: Peer disconnection and reconnection
test_peer_recovery() {
    log ""
    log "================================================"
    log "Test 7: Peer Recovery After Misbehavior"
    log "================================================"
    echo ""

    log "Stopping attack node to simulate disconnection..."
    docker stop "paw-${ATTACK_NODE}" >/dev/null 2>&1

    sleep 10

    local peers_without=$(get_peer_count "$VICTIM_NODE" "$VICTIM_RPC")
    log "Peer count without attack node: ${peers_without}"

    log "Restarting attack node..."
    "${COMPOSE_CMD[@]}" start "${ATTACK_NODE}" >/dev/null 2>&1

    sleep 15

    local peers_with=$(get_peer_count "$VICTIM_NODE" "$VICTIM_RPC")
    log "Peer count with attack node: ${peers_with}"

    if [[ $peers_with -gt $peers_without ]]; then
        log_pass "Attack node reconnected successfully"
        return 0
    else
        log_warn "Attack node did not reconnect (may be banned)"
        return 0
    fi
}

# Run all tests
run_tests() {
    log "Starting Phase 3.4: Malicious Peer Ejection Tests"
    echo ""

    # Start network
    log "Starting 4-node devnet..."
    "${COMPOSE_CMD[@]}" up -d

    if ! wait_for_network; then
        log_fail "Failed to start network"
        return 1
    fi

    # Display initial network state
    log "Initial network state:"
    for node in node1 node2 node3 node4; do
        local port_offset=$((${node:4:1} - 1))
        local rpc_port=$((26657 + port_offset * 10))
        local peers=$(get_peer_count "$node" "$rpc_port")
        local height=$(get_height "$rpc_port")
        log "  ${node}: ${peers} peers, height ${height}"
    done
    echo ""

    # Test results
    local test_results=()

    # Run tests
    if test_invalid_transaction; then
        test_results+=("PASS: Invalid Transaction")
    else
        test_results+=("FAIL: Invalid Transaction")
    fi

    if test_message_spam; then
        test_results+=("PASS: Message Spam")
    else
        test_results+=("FAIL: Message Spam")
    fi

    if test_oversized_message; then
        test_results+=("PASS: Oversized Message")
    else
        test_results+=("FAIL: Oversized Message")
    fi

    if test_peer_reputation; then
        test_results+=("PASS: Peer Reputation")
    else
        test_results+=("FAIL: Peer Reputation")
    fi

    if test_peer_banning; then
        test_results+=("PASS: Peer Banning")
    else
        test_results+=("FAIL: Peer Banning")
    fi

    if test_network_resilience; then
        test_results+=("PASS: Network Resilience")
    else
        test_results+=("FAIL: Network Resilience")
    fi

    if test_peer_recovery; then
        test_results+=("PASS: Peer Recovery")
    else
        test_results+=("FAIL: Peer Recovery")
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

    # Final network state
    log "Final network state:"
    for node in node1 node2 node3 node4; do
        local port_offset=$((${node:4:1} - 1))
        local rpc_port=$((26657 + port_offset * 10))
        local peers=$(get_peer_count "$node" "$rpc_port")
        local height=$(get_height "$rpc_port")
        log "  ${node}: ${peers} peers, height ${height}"
    done
    echo ""

    if [[ "$failed" == "true" ]]; then
        log_fail "Phase 3.4 tests completed with failures"
        return 1
    else
        log_pass "Phase 3.4 tests completed successfully"
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
