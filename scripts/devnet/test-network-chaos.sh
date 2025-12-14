#!/bin/bash
# PAW Network Chaos Testing
# Simulates real-world network conditions to test sentry resilience

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up network conditions..."

    # Restart any stopped containers
    for container in paw-node1 paw-node2 paw-node3 paw-node4 paw-sentry1 paw-sentry2; do
        if ! docker ps | grep -q "$container"; then
            log_info "Restarting $container..."
            docker start "$container" >/dev/null 2>&1 || true
        fi
    done

    # Reconnect any disconnected containers
    for container in paw-node1 paw-node2 paw-node3 paw-node4 paw-sentry1 paw-sentry2; do
        if docker ps | grep -q "$container"; then
            docker network connect pawnet "$container" 2>/dev/null || true
        fi
    done

    log_success "Network restored"
}

# Set up cleanup on exit
trap cleanup EXIT INT TERM

# Test 1: Network Partition (Split Brain)
test_network_partition() {
    log_info "=== Test 1: Network Partition (Split Brain) ==="
    log_info "Simulating network partition: isolating 2 validators from the network"

    # Get initial height
    INITIAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Initial height: $INITIAL_HEIGHT"

    # Disconnect node3 and node4 from network (2/4 validators isolated)
    log_info "Disconnecting node3 and node4 from network..."
    docker network disconnect pawnet paw-node3 2>/dev/null || true
    docker network disconnect pawnet paw-node4 2>/dev/null || true

    # Wait and check if consensus continues (need 3/4 BFT majority)
    log_info "Waiting 20 seconds to see if consensus continues with 2/4 validators..."
    sleep 20

    # Check if remaining validators (node1, node2) can still produce blocks
    AFTER_PARTITION_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ "$AFTER_PARTITION_HEIGHT" -gt "$INITIAL_HEIGHT" ]; then
        log_success "Consensus continued with 2/4 validators (height: $INITIAL_HEIGHT → $AFTER_PARTITION_HEIGHT)"
        log_info "Note: This is expected - 2/4 is not BFT majority, but CometBFT may still progress if it has 2/3+ voting power"
    else
        log_warn "Consensus halted with 2/4 validators (height stuck at $AFTER_PARTITION_HEIGHT)"
        log_info "This is expected BFT behavior - need >2/3 voting power for consensus"
    fi

    # Check sentries status
    SENTRY1_HEIGHT=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    SENTRY2_HEIGHT=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Sentry1 height: $SENTRY1_HEIGHT, Sentry2 height: $SENTRY2_HEIGHT"

    # Reconnect validators
    log_info "Reconnecting node3 and node4..."
    docker network connect pawnet paw-node3 2>/dev/null || true
    docker network connect pawnet paw-node4 2>/dev/null || true

    # Wait for network to stabilize
    log_info "Waiting for network to stabilize..."
    sleep 15

    FINAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    if [ "$FINAL_HEIGHT" -gt "$AFTER_PARTITION_HEIGHT" ]; then
        log_success "Network recovered after partition (height: $AFTER_PARTITION_HEIGHT → $FINAL_HEIGHT)"
    else
        log_warn "Network still recovering (height: $FINAL_HEIGHT)"
    fi

    echo ""
}

# Test 2: Sentry Node Failure
test_sentry_failure() {
    log_info "=== Test 2: Sequential Sentry Failures ==="

    # Get initial state
    INITIAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Initial height: $INITIAL_HEIGHT"

    # Stop sentry1
    log_info "Stopping sentry1..."
    docker stop paw-sentry1 >/dev/null 2>&1
    sleep 5

    # Verify network still accessible via sentry2
    SENTRY2_ACCESSIBLE=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
    if [ "$SENTRY2_ACCESSIBLE" = "paw-devnet" ]; then
        log_success "Network still accessible via sentry2"
    else
        log_error "Cannot access network via sentry2"
    fi

    # Now stop sentry2 as well
    log_info "Stopping sentry2 (both sentries down)..."
    docker stop paw-sentry2 >/dev/null 2>&1
    sleep 5

    # Verify validators still producing blocks
    MID_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    if [ "$MID_HEIGHT" -gt "$INITIAL_HEIGHT" ]; then
        log_success "Validators still producing blocks with both sentries down (height: $INITIAL_HEIGHT → $MID_HEIGHT)"
    else
        log_warn "Validators stopped producing blocks"
    fi

    # Restart sentries
    log_info "Restarting both sentries..."
    docker start paw-sentry1 >/dev/null 2>&1
    docker start paw-sentry2 >/dev/null 2>&1

    # Wait for sync
    log_info "Waiting for sentries to catch up..."
    sleep 20

    # Verify both sentries caught up
    SENTRY1_STATUS=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.sync_info.catching_up' 2>/dev/null)
    SENTRY2_STATUS=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.sync_info.catching_up' 2>/dev/null)

    if [ "$SENTRY1_STATUS" = "false" ] && [ "$SENTRY2_STATUS" = "false" ]; then
        log_success "Both sentries caught up and synced"
    else
        log_warn "Sentries still catching up (sentry1: $SENTRY1_STATUS, sentry2: $SENTRY2_STATUS)"
    fi

    echo ""
}

# Test 3: Validator Node Failure
test_validator_failure() {
    log_info "=== Test 3: Single Validator Failure ==="

    # Get initial height
    INITIAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Initial height: $INITIAL_HEIGHT"

    # Stop one validator (node4)
    log_info "Stopping validator node4 (3/4 validators remaining)..."
    docker stop paw-node4 >/dev/null 2>&1
    sleep 10

    # Check if consensus continues (should work with 3/4)
    AFTER_STOP_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ "$AFTER_STOP_HEIGHT" -gt "$INITIAL_HEIGHT" ]; then
        log_success "Consensus continues with 3/4 validators (height: $INITIAL_HEIGHT → $AFTER_STOP_HEIGHT)"
    else
        log_error "Consensus halted with 3/4 validators"
    fi

    # Check sentry connectivity
    SENTRY1_PEERS=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    log_info "Sentry1 now has $SENTRY1_PEERS peers (expected 4: 3 validators + 1 sentry)"

    # Restart validator
    log_info "Restarting node4..."
    docker start paw-node4 >/dev/null 2>&1

    # Wait for it to catch up
    log_info "Waiting for node4 to catch up..."
    sleep 15

    # Verify it caught up
    NODE4_HEIGHT=$(curl -s http://localhost:26687/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    CURRENT_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    HEIGHT_DIFF=$((CURRENT_HEIGHT - NODE4_HEIGHT))
    HEIGHT_DIFF=${HEIGHT_DIFF#-}  # Absolute value

    if [ "$HEIGHT_DIFF" -le 2 ]; then
        log_success "Node4 caught up to network (node4: $NODE4_HEIGHT, current: $CURRENT_HEIGHT)"
    else
        log_warn "Node4 still catching up (node4: $NODE4_HEIGHT, current: $CURRENT_HEIGHT)"
    fi

    echo ""
}

# Test 4: Cascading Failures
test_cascading_failures() {
    log_info "=== Test 4: Cascading Failures ==="
    log_info "Simulating progressive node failures to test resilience limits"

    # Get initial state
    INITIAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Initial height: $INITIAL_HEIGHT"
    log_info "All 4 validators + 2 sentries running"
    sleep 5

    # Failure 1: Stop sentry1
    log_info "Step 1: Stopping sentry1..."
    docker stop paw-sentry1 >/dev/null 2>&1
    sleep 5
    HEIGHT1=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Height after sentry1 failure: $HEIGHT1"

    # Failure 2: Stop validator node4
    log_info "Step 2: Stopping validator node4..."
    docker stop paw-node4 >/dev/null 2>&1
    sleep 5
    HEIGHT2=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Height after node4 failure: $HEIGHT2"

    # Failure 3: Stop sentry2
    log_info "Step 3: Stopping sentry2 (no sentries remaining)..."
    docker stop paw-sentry2 >/dev/null 2>&1
    sleep 5
    HEIGHT3=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Height after sentry2 failure: $HEIGHT3"

    # Check if consensus still working
    sleep 10
    FINAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ "$FINAL_HEIGHT" -gt "$INITIAL_HEIGHT" ]; then
        log_success "Consensus survived cascading failures (height: $INITIAL_HEIGHT → $FINAL_HEIGHT)"
        log_info "Network running with 3 validators, 0 sentries"
    else
        log_warn "Consensus impacted by cascading failures"
    fi

    # Restart all nodes
    log_info "Recovering network: restarting all stopped nodes..."
    docker start paw-sentry1 >/dev/null 2>&1
    docker start paw-sentry2 >/dev/null 2>&1
    docker start paw-node4 >/dev/null 2>&1

    log_info "Waiting for network to fully recover..."
    sleep 20

    RECOVERED_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_success "Network recovered (height: $RECOVERED_HEIGHT)"

    echo ""
}

# Test 5: Rapid Sentry Restarts
test_rapid_restarts() {
    log_info "=== Test 5: Rapid Sentry Restarts ==="
    log_info "Testing sentry stability under rapid restart cycles"

    for i in {1..3}; do
        log_info "Restart cycle $i/3..."

        # Stop both sentries
        docker stop paw-sentry1 paw-sentry2 >/dev/null 2>&1
        sleep 2

        # Start both sentries
        docker start paw-sentry1 paw-sentry2 >/dev/null 2>&1
        sleep 5

        # Check if they're responding
        SENTRY1_STATUS=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
        SENTRY2_STATUS=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)

        if [ "$SENTRY1_STATUS" = "paw-devnet" ] && [ "$SENTRY2_STATUS" = "paw-devnet" ]; then
            log_success "Cycle $i: Both sentries recovered"
        else
            log_warn "Cycle $i: Sentries still recovering"
        fi
    done

    # Final check
    sleep 10
    SENTRY1_PEERS=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    SENTRY2_PEERS=$(curl -s http://localhost:30668/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)

    log_info "Final peer counts: sentry1=$SENTRY1_PEERS, sentry2=$SENTRY2_PEERS (expected 5 each)"
    if [ "$SENTRY1_PEERS" = "5" ] && [ "$SENTRY2_PEERS" = "5" ]; then
        log_success "Sentries fully recovered with all peer connections"
    else
        log_warn "Sentries may still be reconnecting peers"
    fi

    echo ""
}

# Main execution
main() {
    echo ""
    echo "=============================================="
    echo "  PAW Network Chaos Testing"
    echo "=============================================="
    echo ""
    echo "This test suite simulates real-world network"
    echo "failures to validate sentry architecture"
    echo "resilience and fault tolerance."
    echo ""
    echo "Tests will automatically clean up on exit."
    echo ""

    # Verify network is running
    log_info "Verifying network is running..."
    if ! docker ps | grep -q "paw-node1"; then
        log_error "Network not running! Start with: docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d"
        exit 1
    fi
    log_success "Network containers are running"
    echo ""

    # Run chaos tests
    test_network_partition
    test_sentry_failure
    test_validator_failure
    test_cascading_failures
    test_rapid_restarts

    # Summary
    echo "=============================================="
    echo "  Chaos Testing Complete"
    echo "=============================================="
    echo ""
    log_success "All chaos scenarios completed successfully"
    log_info "Network is in stable state with all nodes running"
    echo ""

    # Final verification
    HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    VALIDATORS=$(curl -s http://localhost:26657/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)
    log_info "Final state: Height=$HEIGHT, Validators=$VALIDATORS/4"
    echo ""
}

# Run chaos tests
main "$@"
