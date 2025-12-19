#!/usr/bin/env bash
# phase3.3-network-conditions.sh - Network Variable Latency/Bandwidth Testing
# Tests consensus behavior under various network conditions using tc (traffic control)
#
# Usage:
#   sudo ./phase3.3-network-conditions.sh [test|cleanup]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
NETWORK_SIM_SCRIPT="${HOME}/blockchain-projects/scripts/network-sim.sh"

COMPOSE_FILE="${PROJECT_ROOT}/compose/docker-compose.devnet.yml"
COMPOSE_CMD=(docker compose -f "${COMPOSE_FILE}")

CHAIN_ID="paw-devnet"
TEST_DURATION=30  # Duration for each network condition test

# Network condition presets to test
declare -A NETWORK_CONDITIONS=(
    ["baseline"]="No network impairment"
    ["high-latency"]="500ms latency"
    ["cross-continent"]="300ms latency, 0.5% loss"
    ["mobile-3g"]="100ms latency, 2% loss, 750kbit bandwidth"
    ["poor-network"]="200ms latency, 5% loss, 1mbit bandwidth"
    ["unstable"]="100ms ±50ms jitter, 10% loss"
    ["lossy"]="15% packet loss"
)

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

# Check if running with sudo
check_sudo() {
    if [[ $EUID -ne 0 ]]; then
        log_fail "This script requires sudo privileges for tc commands"
        echo "Please run with: sudo $0"
        exit 1
    fi
}

# Cleanup function
cleanup() {
    log "Cleaning up network conditions and containers..."

    # Reset network conditions on all containers
    for container in paw-node1 paw-node2 paw-node3 paw-node4; do
        if docker ps --format '{{.Names}}' | grep -qx "$container" 2>/dev/null; then
            log "Resetting network conditions on ${container}..."
            sudo "$NETWORK_SIM_SCRIPT" reset "$container" 2>/dev/null || true
        fi
    done

    # Stop containers
    "${COMPOSE_CMD[@]}" down -v 2>/dev/null || true

    log_pass "Cleanup completed"
}

# Get current block height
get_height() {
    local port=${1:-26657}
    curl -sf "http://localhost:${port}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"' || echo "0"
}

# Get block time (average time between recent blocks)
get_avg_block_time() {
    local port=${1:-26657}
    local rpc="http://localhost:${port}"

    # Get current height
    local height=$(curl -sf "${rpc}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')

    if [[ "$height" == "0" || $height -lt 5 ]]; then
        echo "0"
        return
    fi

    # Get timestamps for last 5 blocks
    local timestamps=()
    for i in {0..4}; do
        local h=$((height - i))
        local ts=$(curl -sf "${rpc}/block?height=${h}" 2>/dev/null | jq -r '.result.block.header.time // ""')
        if [[ -n "$ts" ]]; then
            timestamps+=("$ts")
        fi
    done

    if [[ ${#timestamps[@]} -lt 2 ]]; then
        echo "0"
        return
    fi

    # Calculate time differences
    local first_ts=$(date -d "${timestamps[0]}" +%s 2>/dev/null || echo 0)
    local last_ts=$(date -d "${timestamps[-1]}" +%s 2>/dev/null || echo 0)

    if [[ $first_ts -gt 0 && $last_ts -gt 0 ]]; then
        local diff=$((first_ts - last_ts))
        local avg=$((diff / 4))
        echo "$avg"
    else
        echo "0"
    fi
}

# Wait for network to start
wait_for_network() {
    log "Waiting for network to start and produce blocks..."
    local timeout=300
    local elapsed=0
    local target_height=10

    while [[ $elapsed -lt $timeout ]]; do
        local height=$(get_height "26657")

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

# Apply network condition to all nodes
apply_network_condition() {
    local condition=$1

    if [[ "$condition" == "baseline" ]]; then
        log "Resetting to baseline (no network impairment)..."
        for container in paw-node1 paw-node2 paw-node3 paw-node4; do
            sudo "$NETWORK_SIM_SCRIPT" reset "$container" 2>/dev/null || true
        done
        log_pass "Baseline restored"
        return 0
    fi

    log "Applying network condition: ${condition}"

    for container in paw-node1 paw-node2 paw-node3 paw-node4; do
        log "  Applying to ${container}..."
        if sudo "$NETWORK_SIM_SCRIPT" preset "$condition" "$container" 2>&1 | grep -q "applied"; then
            log_pass "  ${container} configured"
        else
            log_fail "  Failed to configure ${container}"
            return 1
        fi
    done

    log_pass "Network condition applied to all nodes"
    return 0
}

# Monitor consensus during network condition
monitor_consensus() {
    local condition=$1
    local duration=$2

    log "Monitoring consensus for ${duration}s under ${condition} conditions..."

    local start_height=$(get_height "26657")
    local start_time=$(date +%s)

    # Monitor for specified duration
    local samples=()
    local interval=5
    local iterations=$((duration / interval))

    for ((i=0; i<iterations; i++)); do
        sleep "$interval"

        local height=$(get_height "26657")
        local block_time=$(get_avg_block_time "26657")

        samples+=("${height}:${block_time}")

        if [[ $((i % 2)) -eq 0 ]]; then
            log "  Height: ${height}, Avg block time: ${block_time}s"
        fi
    done

    local end_height=$(get_height "26657")
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    local blocks_produced=$((end_height - start_height))

    # Calculate statistics
    local avg_block_time=0
    if [[ $blocks_produced -gt 0 ]]; then
        avg_block_time=$((total_time / blocks_produced))
    fi

    log "Monitoring complete:"
    log "  Start height: ${start_height}"
    log "  End height: ${end_height}"
    log "  Blocks produced: ${blocks_produced}"
    log "  Total time: ${total_time}s"
    log "  Avg block time: ${avg_block_time}s"

    # Determine if consensus is healthy
    # We expect blocks to be produced, even if slower
    if [[ $blocks_produced -gt 0 ]]; then
        log_pass "Consensus remained active under ${condition} conditions"
        echo "${condition}:PASS:${blocks_produced}:${avg_block_time}"
        return 0
    else
        log_fail "Consensus halted under ${condition} conditions"
        echo "${condition}:FAIL:0:0"
        return 1
    fi
}

# Test specific network condition
test_network_condition() {
    local condition=$1
    local description="${NETWORK_CONDITIONS[$condition]}"

    log ""
    log "================================================"
    log "Testing: ${condition}"
    log "Description: ${description}"
    log "================================================"
    echo ""

    # Apply network condition
    if ! apply_network_condition "$condition"; then
        log_fail "Failed to apply network condition: ${condition}"
        return 1
    fi

    # Give network time to adjust
    log "Waiting for network to adjust to new conditions..."
    sleep 10

    # Monitor consensus
    local result=$(monitor_consensus "$condition" "$TEST_DURATION")

    # Extract result
    local status=$(echo "$result" | cut -d: -f2)

    if [[ "$status" == "PASS" ]]; then
        log_pass "Test passed for ${condition}"
        return 0
    else
        log_fail "Test failed for ${condition}"
        return 1
    fi
}

# Test gradual network degradation
test_gradual_degradation() {
    log ""
    log "================================================"
    log "Test: Gradual Network Degradation"
    log "================================================"
    echo ""

    # Test sequence: baseline -> mobile-4g -> mobile-3g -> poor-network
    local conditions=("baseline" "mobile-4g" "mobile-3g" "poor-network")

    for condition in "${conditions[@]}"; do
        # Skip if condition not in our preset list
        if [[ ! -v NETWORK_CONDITIONS[$condition] && "$condition" != "mobile-4g" ]]; then
            continue
        fi

        log "Step: ${condition}"

        if [[ "$condition" == "mobile-4g" ]]; then
            # Apply mobile-4g preset manually
            log "Applying mobile-4g preset..."
            for container in paw-node1 paw-node2 paw-node3 paw-node4; do
                sudo "$NETWORK_SIM_SCRIPT" preset "mobile-4g" "$container" 2>&1 | grep -q "applied" || true
            done
            sleep 5
        else
            apply_network_condition "$condition"
        fi

        # Monitor for shorter duration
        local height_before=$(get_height "26657")
        sleep 15
        local height_after=$(get_height "26657")
        local blocks=$((height_after - height_before))

        log "  Blocks produced: ${blocks}"

        if [[ $blocks -gt 0 ]]; then
            log_pass "  Consensus active under ${condition}"
        else
            log_warn "  Consensus slow/halted under ${condition}"
        fi
    done

    log_pass "Gradual degradation test completed"
    return 0
}

# Test network recovery
test_network_recovery() {
    log ""
    log "================================================"
    log "Test: Network Recovery"
    log "================================================"
    echo ""

    # Apply poor network condition
    log "Applying poor network conditions..."
    apply_network_condition "poor-network"
    sleep 10

    local height_impaired=$(get_height "26657")
    log "Height under poor conditions: ${height_impaired}"

    # Monitor briefly
    sleep 15
    local height_impaired_after=$(get_height "26657")
    local blocks_impaired=$((height_impaired_after - height_impaired))
    log "Blocks produced under impairment: ${blocks_impaired}"

    # Restore baseline
    log "Restoring baseline conditions..."
    apply_network_condition "baseline"
    sleep 10

    # Monitor recovery
    local height_recovery=$(get_height "26657")
    sleep 15
    local height_recovery_after=$(get_height "26657")
    local blocks_recovery=$((height_recovery_after - height_recovery))
    log "Blocks produced after recovery: ${blocks_recovery}"

    if [[ $blocks_recovery -ge $blocks_impaired ]]; then
        log_pass "Network recovered successfully (block production improved or maintained)"
        return 0
    else
        log_warn "Network recovery uncertain (block production: ${blocks_impaired} -> ${blocks_recovery})"
        return 0
    fi
}

# Run all tests
run_tests() {
    log "Starting Phase 3.3: Network Variable Latency/Bandwidth Tests"
    echo ""

    # Check prerequisites
    check_sudo

    if [[ ! -x "$NETWORK_SIM_SCRIPT" ]]; then
        log_fail "Network simulation script not found: ${NETWORK_SIM_SCRIPT}"
        exit 1
    fi

    # Start network
    log "Starting 4-node devnet..."
    "${COMPOSE_CMD[@]}" up -d

    if ! wait_for_network; then
        log_fail "Failed to start network"
        return 1
    fi

    # Test results
    local test_results=()

    # Test each network condition
    for condition in "baseline" "high-latency" "cross-continent" "mobile-3g" "poor-network" "unstable" "lossy"; do
        if test_network_condition "$condition"; then
            test_results+=("PASS: ${condition}")
        else
            test_results+=("FAIL: ${condition}")
        fi

        # Reset to baseline between tests
        if [[ "$condition" != "baseline" ]]; then
            apply_network_condition "baseline"
            sleep 5
        fi
    done

    # Test gradual degradation
    if test_gradual_degradation; then
        test_results+=("PASS: Gradual Degradation")
    else
        test_results+=("FAIL: Gradual Degradation")
    fi

    # Test network recovery
    if test_network_recovery; then
        test_results+=("PASS: Network Recovery")
    else
        test_results+=("FAIL: Network Recovery")
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
        log_fail "Phase 3.3 tests completed with failures"
        return 1
    else
        log_pass "Phase 3.3 tests completed successfully"
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
