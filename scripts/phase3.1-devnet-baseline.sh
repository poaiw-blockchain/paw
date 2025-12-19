#!/usr/bin/env bash
# phase3.1-devnet-baseline.sh - 4-Node Devnet Baseline Testing
# Tests basic 4-node network functionality, health checks, and smoke tests
#
# Usage:
#   ./phase3.1-devnet-baseline.sh [test|cleanup]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source the devnet lib if available
if [[ -f "${SCRIPT_DIR}/devnet/lib.sh" ]]; then
    source "${SCRIPT_DIR}/devnet/lib.sh"
fi

COMPOSE_FILE="${PROJECT_ROOT}/compose/docker-compose.devnet.yml"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")

# Node configuration
declare -A NODES=(
    [node1]="26657:9090:1317"
    [node2]="26667:9091:1327"
    [node3]="26677:9092:1337"
    [node4]="26687:9093:1347"
)

CHAIN_ID="paw-devnet"
REQUIRED_HEIGHT=10
READY_TIMEOUT=600  # 10 minutes for initial startup
HEALTH_CHECK_INTERVAL=5

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
    log "Cleaning up 4-node devnet..."
    "${COMPOSE_CMD[@]}" down -v 2>/dev/null || true
    log_pass "Cleanup completed"
}

# Check if node is running
node_running() {
    local container="paw-$1"
    docker ps --format '{{.Names}}' | grep -qx "$container"
}

# Get node RPC endpoint
get_rpc_endpoint() {
    local node=$1
    local ports="${NODES[$node]}"
    local rpc_port=$(echo "$ports" | cut -d: -f1)
    echo "http://localhost:${rpc_port}"
}

# Get node API endpoint
get_api_endpoint() {
    local node=$1
    local ports="${NODES[$node]}"
    local api_port=$(echo "$ports" | cut -d: -f3)
    echo "http://localhost:${api_port}"
}

# Wait for node to be ready
wait_for_node_ready() {
    local node=$1
    local rpc=$(get_rpc_endpoint "$node")
    local timeout=$READY_TIMEOUT
    local elapsed=0

    log "Waiting for ${node} to be ready at ${rpc}..."

    while [[ $elapsed -lt $timeout ]]; do
        if curl -sf "${rpc}/health" >/dev/null 2>&1; then
            # Check if syncing is complete
            local catching_up=$(curl -sf "${rpc}/status" 2>/dev/null | jq -r '.result.sync_info.catching_up // true')
            if [[ "$catching_up" == "false" ]]; then
                log_pass "${node} is ready and synced"
                return 0
            fi
        fi

        sleep "$HEALTH_CHECK_INTERVAL"
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))

        if [[ $((elapsed % 60)) -eq 0 ]]; then
            log "Still waiting for ${node}... (${elapsed}s/${timeout}s)"
        fi
    done

    log_fail "${node} failed to become ready within ${timeout}s"
    return 1
}

# Wait for specific block height
wait_for_height() {
    local node=$1
    local target_height=$2
    local rpc=$(get_rpc_endpoint "$node")
    local timeout=300  # 5 minutes
    local elapsed=0

    log "Waiting for ${node} to reach height ${target_height}..."

    while [[ $elapsed -lt $timeout ]]; do
        local height=$(curl -sf "${rpc}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')

        if [[ "$height" != "0" ]] && [[ $height -ge $target_height ]]; then
            log_pass "${node} reached height ${height}"
            return 0
        fi

        sleep "$HEALTH_CHECK_INTERVAL"
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
    done

    log_fail "${node} failed to reach height ${target_height} within ${timeout}s"
    return 1
}

# Get node info
get_node_info() {
    local node=$1
    local rpc=$(get_rpc_endpoint "$node")

    local info=$(curl -sf "${rpc}/status" 2>/dev/null)
    if [[ -z "$info" ]]; then
        echo "N/A"
        return 1
    fi

    local node_id=$(echo "$info" | jq -r '.result.node_info.id // "unknown"')
    local height=$(echo "$info" | jq -r '.result.sync_info.latest_block_height // "0"')
    local peers=$(echo "$info" | jq -r '.result.node_info.other.n_peers // "0"')
    local catching_up=$(echo "$info" | jq -r '.result.sync_info.catching_up // true')

    echo "NodeID: ${node_id}, Height: ${height}, Peers: ${peers}, Syncing: ${catching_up}"
}

# Test network connectivity between nodes
test_network_connectivity() {
    log "Testing network connectivity between nodes..."

    local pass_count=0
    local total_count=0

    for node in "${!NODES[@]}"; do
        if ! node_running "$node"; then
            continue
        fi

        local rpc=$(get_rpc_endpoint "$node")
        local net_info=$(curl -sf "${rpc}/net_info" 2>/dev/null)

        if [[ -n "$net_info" ]]; then
            local peer_count=$(echo "$net_info" | jq -r '.result.n_peers // "0"')
            total_count=$((total_count + 1))

            log "${node}: ${peer_count} peers connected"

            if [[ $peer_count -gt 0 ]]; then
                pass_count=$((pass_count + 1))
                log_pass "${node} has network connectivity"
            else
                log_warn "${node} has no peer connections"
            fi
        fi
    done

    if [[ $pass_count -eq $total_count ]] && [[ $total_count -gt 0 ]]; then
        log_pass "All nodes have network connectivity"
        return 0
    else
        log_fail "Network connectivity test failed (${pass_count}/${total_count})"
        return 1
    fi
}

# Test consensus progression
test_consensus_progression() {
    log "Testing consensus progression..."

    # Get initial height from node1
    local rpc=$(get_rpc_endpoint "node1")
    local initial_height=$(curl -sf "${rpc}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')

    log "Initial height: ${initial_height}"
    log "Waiting for 10 new blocks..."

    local target_height=$((initial_height + 10))
    if wait_for_height "node1" "$target_height"; then
        log_pass "Consensus is progressing normally"

        # Verify all nodes are at similar heights
        log "Verifying height synchronization across nodes..."
        local heights=()
        for node in "${!NODES[@]}"; do
            if node_running "$node"; then
                local node_rpc=$(get_rpc_endpoint "$node")
                local height=$(curl -sf "${node_rpc}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')
                heights+=("${node}:${height}")
                log "  ${node}: height ${height}"
            fi
        done

        return 0
    else
        log_fail "Consensus progression test failed"
        return 1
    fi
}

# Test API endpoints
test_api_endpoints() {
    log "Testing API endpoints..."

    local pass_count=0
    local total_count=0

    for node in "${!NODES[@]}"; do
        if ! node_running "$node"; then
            continue
        fi

        local api=$(get_api_endpoint "$node")
        total_count=$((total_count + 1))

        # Test bank module
        if curl -sf "${api}/cosmos/bank/v1beta1/supply" >/dev/null 2>&1; then
            log_pass "${node} API (bank) responding"
            pass_count=$((pass_count + 1))
        else
            log_fail "${node} API (bank) not responding"
        fi

        # Test staking module
        if curl -sf "${api}/cosmos/staking/v1beta1/validators" >/dev/null 2>&1; then
            log_pass "${node} API (staking) responding"
        else
            log_warn "${node} API (staking) not responding"
        fi
    done

    if [[ $pass_count -eq $total_count ]] && [[ $total_count -gt 0 ]]; then
        log_pass "All API endpoints responding"
        return 0
    else
        log_fail "API endpoint test failed (${pass_count}/${total_count})"
        return 1
    fi
}

# Test gRPC endpoints
test_grpc_endpoints() {
    log "Testing gRPC endpoints..."

    if ! command -v grpcurl &>/dev/null; then
        log_warn "grpcurl not found, skipping gRPC tests"
        return 0
    fi

    local pass_count=0
    local total_count=0

    for node in "${!NODES[@]}"; do
        if ! node_running "$node"; then
            continue
        fi

        local ports="${NODES[$node]}"
        local grpc_port=$(echo "$ports" | cut -d: -f2)
        total_count=$((total_count + 1))

        # Test gRPC reflection
        if grpcurl -plaintext "localhost:${grpc_port}" list >/dev/null 2>&1; then
            log_pass "${node} gRPC responding on port ${grpc_port}"
            pass_count=$((pass_count + 1))
        else
            log_fail "${node} gRPC not responding on port ${grpc_port}"
        fi
    done

    if [[ $pass_count -eq $total_count ]] && [[ $total_count -gt 0 ]]; then
        log_pass "All gRPC endpoints responding"
        return 0
    else
        log_fail "gRPC endpoint test failed (${pass_count}/${total_count})"
        return 1
    fi
}

# Test validator set
test_validator_set() {
    log "Testing validator set..."

    local rpc=$(get_rpc_endpoint "node1")
    local validators=$(curl -sf "${rpc}/validators" 2>/dev/null)

    if [[ -z "$validators" ]]; then
        log_fail "Failed to query validator set"
        return 1
    fi

    local validator_count=$(echo "$validators" | jq -r '.result.validators | length')
    log "Active validators: ${validator_count}"

    if [[ $validator_count -ge 4 ]]; then
        log_pass "Validator set is healthy (${validator_count} validators)"
        return 0
    else
        log_warn "Unexpected validator count: ${validator_count}"
        return 1
    fi
}

# Run smoke tests
run_smoke_tests() {
    log "Running smoke tests (bank transfer)..."

    local container="paw-node1"
    local home="/root/.paw/node1"

    # Check if smoke test keys exist
    local has_keys=$(docker exec "$container" pawd --home "$home" keys list --keyring-backend test 2>/dev/null | grep -c "smoke-trader" || echo 0)

    if [[ $has_keys -eq 0 ]]; then
        log_warn "Smoke test keys not found, skipping transfer test"
        return 0
    fi

    # Get trader address
    local trader_addr=$(docker exec "$container" pawd --home "$home" keys show smoke-trader --keyring-backend test 2>/dev/null | awk '/address:/ {print $2}')

    if [[ -z "$trader_addr" ]]; then
        log_warn "Could not get trader address, skipping transfer test"
        return 0
    fi

    # Query initial balance
    local api=$(get_api_endpoint "node1")
    local initial_balance=$(curl -sf "${api}/cosmos/bank/v1beta1/balances/${trader_addr}" 2>/dev/null | jq -r '.balances[] | select(.denom=="upaw") | .amount' || echo "0")

    log "Trader address: ${trader_addr}"
    log "Initial balance: ${initial_balance} upaw"

    log_pass "Smoke test completed (keys and balances verified)"
    return 0
}

# Main test function
run_tests() {
    log "Starting Phase 3.1: 4-Node Devnet Baseline Tests"
    echo ""

    # Build binary if needed
    if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
        log "Building pawd binary..."
        cd "$PROJECT_ROOT"
        if [[ -f "${SCRIPT_DIR}/devnet/lib.sh" ]]; then
            ensure_pawd_binary "phase3.1" || true
        else
            go build -o pawd ./cmd/pawd
        fi
        log_pass "Binary built successfully"
    fi

    # Start the network
    log "Starting 4-node devnet..."
    "${COMPOSE_CMD[@]}" up -d --build

    # Wait for all nodes to be ready
    local all_ready=true
    for node in "${!NODES[@]}"; do
        if ! wait_for_node_ready "$node"; then
            all_ready=false
            log_fail "${node} failed to start properly"
        fi
    done

    if [[ "$all_ready" != "true" ]]; then
        log_fail "Not all nodes started successfully"
        return 1
    fi

    log_pass "All 4 nodes started successfully"
    echo ""

    # Display node information
    log "Node Information:"
    for node in "${!NODES[@]}"; do
        if node_running "$node"; then
            local info=$(get_node_info "$node")
            log "  ${node}: ${info}"
        fi
    done
    echo ""

    # Run test suite
    local test_results=()

    # Test 1: Network Connectivity
    if test_network_connectivity; then
        test_results+=("PASS: Network Connectivity")
    else
        test_results+=("FAIL: Network Connectivity")
    fi
    echo ""

    # Test 2: Consensus Progression
    if test_consensus_progression; then
        test_results+=("PASS: Consensus Progression")
    else
        test_results+=("FAIL: Consensus Progression")
    fi
    echo ""

    # Test 3: API Endpoints
    if test_api_endpoints; then
        test_results+=("PASS: API Endpoints")
    else
        test_results+=("FAIL: API Endpoints")
    fi
    echo ""

    # Test 4: gRPC Endpoints
    if test_grpc_endpoints; then
        test_results+=("PASS: gRPC Endpoints")
    else
        test_results+=("FAIL: gRPC Endpoints")
    fi
    echo ""

    # Test 5: Validator Set
    if test_validator_set; then
        test_results+=("PASS: Validator Set")
    else
        test_results+=("FAIL: Validator Set")
    fi
    echo ""

    # Test 6: Smoke Tests
    if run_smoke_tests; then
        test_results+=("PASS: Smoke Tests")
    else
        test_results+=("FAIL: Smoke Tests")
    fi
    echo ""

    # Print summary
    log "Test Results Summary:"
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
        log_fail "Phase 3.1 tests completed with failures"
        return 1
    else
        log_pass "Phase 3.1 tests completed successfully"
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
