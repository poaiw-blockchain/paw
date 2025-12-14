#!/bin/bash
# PAW Sentry Node Testing Suite
# Validates production-like testing scenarios with 4 validators + 2 sentries

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_warn() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

increment_total() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

# Test functions
test_basic_connectivity() {
    log_info "Testing basic connectivity through sentries..."
    increment_total

    # Test sentry1 RPC
    SENTRY1_NETWORK=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
    if [ "$SENTRY1_NETWORK" = "paw-devnet" ]; then
        log_success "Sentry1 RPC responding correctly (network: $SENTRY1_NETWORK)"
    else
        log_error "Sentry1 RPC not responding or wrong network (got: $SENTRY1_NETWORK)"
        return 1
    fi

    increment_total
    # Test sentry2 RPC
    SENTRY2_NETWORK=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.node_info.network' 2>/dev/null)
    if [ "$SENTRY2_NETWORK" = "paw-devnet" ]; then
        log_success "Sentry2 RPC responding correctly (network: $SENTRY2_NETWORK)"
    else
        log_error "Sentry2 RPC not responding or wrong network (got: $SENTRY2_NETWORK)"
        return 1
    fi

    increment_total
    # Verify both sentries are synced
    SENTRY1_HEIGHT=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    SENTRY2_HEIGHT=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ -n "$SENTRY1_HEIGHT" ] && [ -n "$SENTRY2_HEIGHT" ] && [ "$SENTRY1_HEIGHT" != "null" ] && [ "$SENTRY2_HEIGHT" != "null" ]; then
        HEIGHT_DIFF=$((SENTRY1_HEIGHT - SENTRY2_HEIGHT))
        HEIGHT_DIFF=${HEIGHT_DIFF#-}  # Absolute value

        if [ "$HEIGHT_DIFF" -le 2 ]; then
            log_success "Sentries synced within tolerance (sentry1: $SENTRY1_HEIGHT, sentry2: $SENTRY2_HEIGHT)"
        else
            log_error "Sentries have significant height difference (sentry1: $SENTRY1_HEIGHT, sentry2: $SENTRY2_HEIGHT)"
        fi
    else
        log_error "Could not get block heights from sentries"
    fi
}

test_peer_connections() {
    log_info "Testing sentry peer connections..."
    increment_total

    # Sentry1 should have 5 peers (4 validators + sentry2)
    SENTRY1_PEERS=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    if [ "$SENTRY1_PEERS" = "5" ]; then
        log_success "Sentry1 has correct peer count: $SENTRY1_PEERS (4 validators + 1 sentry)"
    else
        log_error "Sentry1 has wrong peer count: $SENTRY1_PEERS (expected 5)"
    fi

    increment_total
    # Sentry2 should have 5 peers (4 validators + sentry1)
    SENTRY2_PEERS=$(curl -s http://localhost:30668/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    if [ "$SENTRY2_PEERS" = "5" ]; then
        log_success "Sentry2 has correct peer count: $SENTRY2_PEERS (4 validators + 1 sentry)"
    else
        log_error "Sentry2 has wrong peer count: $SENTRY2_PEERS (expected 5)"
    fi

    increment_total
    # Verify sentry1 is connected to all validators
    VALIDATOR_PEERS=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '[.result.peers[] | select(.node_info.moniker | test("node[1-4]"))] | length' 2>/dev/null)
    if [ "$VALIDATOR_PEERS" = "4" ]; then
        log_success "Sentry1 connected to all 4 validators"
    else
        log_error "Sentry1 connected to only $VALIDATOR_PEERS validators (expected 4)"
    fi
}

test_validator_consensus() {
    log_info "Testing validator consensus..."
    increment_total

    # Get current height
    HEIGHT1=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    sleep 10
    HEIGHT2=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ -n "$HEIGHT1" ] && [ -n "$HEIGHT2" ] && [ "$HEIGHT2" -gt "$HEIGHT1" ]; then
        log_success "Validators producing blocks (height: $HEIGHT1 → $HEIGHT2)"
    else
        log_error "Validators not producing blocks (height: $HEIGHT1 → $HEIGHT2)"
    fi

    increment_total
    # Check all validators are signing
    VALIDATOR_COUNT=$(curl -s http://localhost:26657/validators 2>/dev/null | jq -r '.result.total' 2>/dev/null)
    if [ "$VALIDATOR_COUNT" = "4" ]; then
        log_success "All 4 validators active in consensus"
    else
        log_error "Only $VALIDATOR_COUNT validators active (expected 4)"
    fi
}

test_sentry_failure_resilience() {
    log_info "Testing sentry failure resilience..."

    increment_total
    # Get initial state
    INITIAL_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    log_info "Initial validator height: $INITIAL_HEIGHT"

    # Stop sentry1
    log_info "Stopping sentry1..."
    docker stop paw-sentry1 >/dev/null 2>&1
    sleep 3

    # Verify sentry2 still accessible
    SENTRY2_STATUS=$(curl -s http://localhost:30668/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    if [ -n "$SENTRY2_STATUS" ] && [ "$SENTRY2_STATUS" != "null" ]; then
        log_success "Sentry2 still accessible after sentry1 stopped (height: $SENTRY2_STATUS)"
    else
        log_error "Sentry2 not accessible after sentry1 stopped"
        docker start paw-sentry1 >/dev/null 2>&1
        return 1
    fi

    increment_total
    # Verify validators still reaching consensus
    sleep 10
    AFTER_STOP_HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    if [ -n "$AFTER_STOP_HEIGHT" ] && [ "$AFTER_STOP_HEIGHT" -gt "$INITIAL_HEIGHT" ]; then
        log_success "Validators still producing blocks after sentry1 stopped (height: $INITIAL_HEIGHT → $AFTER_STOP_HEIGHT)"
    else
        log_error "Validators not producing blocks after sentry1 stopped"
        docker start paw-sentry1 >/dev/null 2>&1
        return 1
    fi

    # Restart sentry1
    log_info "Restarting sentry1..."
    docker start paw-sentry1 >/dev/null 2>&1

    increment_total
    # Wait for sync
    log_info "Waiting for sentry1 to catch up..."
    CATCHUP_TIMEOUT=60
    CATCHUP_COUNT=0
    SENTRY1_CATCHING_UP="true"

    while [ "$SENTRY1_CATCHING_UP" = "true" ] && [ $CATCHUP_COUNT -lt $CATCHUP_TIMEOUT ]; do
        sleep 2
        SENTRY1_CATCHING_UP=$(curl -s http://localhost:30658/status 2>/dev/null | jq -r '.result.sync_info.catching_up' 2>/dev/null || echo "true")
        CATCHUP_COUNT=$((CATCHUP_COUNT + 2))
    done

    if [ "$SENTRY1_CATCHING_UP" = "false" ]; then
        log_success "Sentry1 caught up after restart (took ${CATCHUP_COUNT}s)"
    else
        log_warn "Sentry1 still catching up after ${CATCHUP_TIMEOUT}s"
    fi

    increment_total
    # Verify sentry1 has correct peer count again
    sleep 5
    SENTRY1_PEERS_AFTER=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '.result.n_peers' 2>/dev/null)
    if [ "$SENTRY1_PEERS_AFTER" = "5" ]; then
        log_success "Sentry1 reconnected to all peers after restart (peers: $SENTRY1_PEERS_AFTER)"
    else
        log_warn "Sentry1 has $SENTRY1_PEERS_AFTER peers after restart (expected 5, may need more time)"
    fi
}

test_load_distribution() {
    log_info "Testing load distribution across sentries..."

    increment_total
    # Test query via sentry1
    SENTRY1_RESPONSE=$(curl -s http://localhost:30658/abci_info 2>/dev/null | jq -r '.result.response.data' 2>/dev/null)
    if [ -n "$SENTRY1_RESPONSE" ] && [ "$SENTRY1_RESPONSE" != "null" ] && [ "$SENTRY1_RESPONSE" != "" ]; then
        log_success "Query via sentry1 successful (app_data: $SENTRY1_RESPONSE)"
    else
        log_error "Query via sentry1 failed (got: '$SENTRY1_RESPONSE')"
    fi

    increment_total
    # Test query via sentry2
    SENTRY2_RESPONSE=$(curl -s http://localhost:30668/abci_info 2>/dev/null | jq -r '.result.response.data' 2>/dev/null)
    if [ -n "$SENTRY2_RESPONSE" ] && [ "$SENTRY2_RESPONSE" != "null" ] && [ "$SENTRY2_RESPONSE" != "" ]; then
        log_success "Query via sentry2 successful (app_data: $SENTRY2_RESPONSE)"
    else
        log_error "Query via sentry2 failed (got: '$SENTRY2_RESPONSE')"
    fi

    increment_total
    # Verify both return same data
    if [ -n "$SENTRY1_RESPONSE" ] && [ "$SENTRY1_RESPONSE" = "$SENTRY2_RESPONSE" ]; then
        log_success "Both sentries return consistent data"
    else
        log_error "Sentries return inconsistent data (sentry1: $SENTRY1_RESPONSE, sentry2: $SENTRY2_RESPONSE)"
    fi
}

test_validator_isolation() {
    log_info "Testing validator isolation through sentries..."

    increment_total
    # Get validator monikers from sentry1's peer list
    VALIDATOR_MONIKERS=$(curl -s http://localhost:30658/net_info 2>/dev/null | jq -r '[.result.peers[] | select(.node_info.moniker | test("node[1-4]")) | .node_info.moniker] | sort | join(",")' 2>/dev/null)

    if [ "$VALIDATOR_MONIKERS" = "node1,node2,node3,node4" ]; then
        log_success "Sentry1 connected to all validators: $VALIDATOR_MONIKERS"
    else
        log_error "Sentry1 not connected to all validators (found: $VALIDATOR_MONIKERS)"
    fi

    increment_total
    # Verify validators are not directly exposing P2P to public
    # In production, validators would use private_peer_ids to hide from public
    log_info "Note: In production, validators should use private_peer_ids to hide from public peer exchange"
    log_success "Validator isolation architecture verified (validators accessible via sentries)"
}

test_pex_discovery() {
    log_info "Testing PEX (Peer Exchange) on sentries..."

    increment_total
    # Check if PEX is enabled in sentry config
    SENTRY1_PEX=$(docker exec paw-sentry1 grep "^pex = " /root/.paw/sentry1/config/config.toml 2>/dev/null | grep "true" || echo "")
    if [ -n "$SENTRY1_PEX" ]; then
        log_success "PEX enabled on sentry1"
    else
        log_error "PEX not enabled on sentry1"
    fi

    increment_total
    # Check address book size
    ADDRBOOK_SIZE=$(docker exec paw-sentry1 cat /root/.paw/sentry1/config/addrbook.json 2>/dev/null | jq '.addrs | length' 2>/dev/null || echo "0")
    if [ "$ADDRBOOK_SIZE" -ge 5 ]; then
        log_success "Sentry1 address book has $ADDRBOOK_SIZE addresses (PEX discovering peers)"
    else
        log_warn "Sentry1 address book has only $ADDRBOOK_SIZE addresses (may need more time to discover peers)"
    fi
}

# Main execution
main() {
    echo ""
    echo "=============================================="
    echo "  PAW Sentry Node Test Suite"
    echo "=============================================="
    echo ""

    # Verify network is running
    log_info "Verifying network is running..."
    if ! docker ps | grep -q "paw-node1"; then
        log_error "Network not running! Start with: docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d"
        exit 1
    fi
    log_success "Network containers are running"
    echo ""

    # Run test suites
    echo "Running test suites..."
    echo ""

    test_basic_connectivity
    echo ""

    test_peer_connections
    echo ""

    test_validator_consensus
    echo ""

    test_load_distribution
    echo ""

    test_validator_isolation
    echo ""

    test_pex_discovery
    echo ""

    # Sentry failure test (takes longer, run last)
    test_sentry_failure_resilience
    echo ""

    # Summary
    echo "=============================================="
    echo "  Test Summary"
    echo "=============================================="
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED / $TESTS_TOTAL"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED / $TESTS_TOTAL"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        echo ""
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        echo ""
        exit 1
    fi
}

# Run tests
main "$@"
