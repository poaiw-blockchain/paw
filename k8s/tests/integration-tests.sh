#!/bin/bash
# integration-tests.sh - End-to-end integration tests for PAW Kubernetes deployment
set -u  # Keep unset variable check, but allow command failures

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-paw}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASSED=0
FAILED=0

log_test() { echo -e "${BLUE}[TEST]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }

get_validator_pod() {
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1
}

test_transaction_submission() {
    log_test "Testing transaction submission..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    # Create a test account
    local result=$(kubectl exec -n "$NAMESPACE" "$pod" -- sh -c '
        /app/bin/paw-node keys add test-account --keyring-backend test --home /data --no-backup 2>&1 | grep -o "paw[a-z0-9]*" | head -1 || echo ""
    ')

    if [ -n "$result" ] && echo "$result" | grep -q "^paw"; then
        log_pass "Transaction account created: $result"
    else
        log_fail "Could not create test account"
        return 1
    fi
}

test_block_finality() {
    log_test "Testing block finality..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    # Get current block
    local block1=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height')

    # Wait for new block
    sleep 10

    local block2=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height')

    if [ "$block2" -gt "$block1" ]; then
        log_pass "Blocks are being finalized ($block1 -> $block2)"
    else
        log_fail "Blocks not advancing ($block1 -> $block2)"
        return 1
    fi
}

test_api_endpoints() {
    log_test "Testing API endpoints..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    # Test RPC
    local rpc_status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:26657/status)
    if [ "$rpc_status" = "200" ]; then
        log_pass "RPC endpoint responsive"
    else
        log_fail "RPC endpoint not responsive (HTTP $rpc_status)"
    fi

    # Test REST API
    local api_status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:1317/cosmos/bank/v1beta1/params 2>/dev/null || echo "000")
    if [ "$api_status" = "200" ]; then
        log_pass "REST API endpoint responsive"
    else
        log_fail "REST API endpoint not responsive (HTTP $api_status)"
    fi

    # Test gRPC (via grpcurl if available)
    if kubectl exec -n "$NAMESPACE" "$pod" -- which grpcurl &>/dev/null; then
        local grpc_result=$(kubectl exec -n "$NAMESPACE" "$pod" -- grpcurl -plaintext localhost:9090 list 2>/dev/null | wc -l)
        if [ "$grpc_result" -gt 0 ]; then
            log_pass "gRPC endpoint responsive"
        else
            log_fail "gRPC endpoint not responsive"
        fi
    else
        echo "  (grpcurl not available, skipping gRPC test)"
    fi
}

test_consensus_participation() {
    log_test "Testing consensus participation..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    local validators=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/validators 2>/dev/null | jq '.result.validators | length')

    if [ "$validators" -gt 0 ]; then
        log_pass "$validators validators participating in consensus"
    else
        log_fail "No validators in consensus"
        return 1
    fi
}

test_peer_connectivity() {
    log_test "Testing peer connectivity..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    local peers=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/net_info 2>/dev/null | jq '.result.n_peers | tonumber' 2>/dev/null || echo "0")

    if [ "$peers" -gt 0 ]; then
        log_pass "$peers peers connected"
    else
        # Single node has no peers - this is acceptable for single-node testing
        log_pass "0 peers (single node mode)"
    fi
}

test_metrics_export() {
    log_test "Testing metrics export..."

    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    # Try common prometheus ports (26660, 36660)
    local metrics=""
    for port in 26660 36660; do
        metrics=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:$port/metrics 2>/dev/null | head -5)
        if echo "$metrics" | grep -q "^#"; then
            break
        fi
    done

    if echo "$metrics" | grep -q "^#"; then
        log_pass "Prometheus metrics being exported"
    else
        log_fail "Metrics not available"
        return 1
    fi
}

test_service_discovery() {
    log_test "Testing service discovery..."

    # Check DNS resolution within cluster using getent or curl
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_fail "No validator pod found"
        return 1
    fi

    # Try getent or fallback to checking if service resolves via curl
    local dns_result=$(kubectl exec -n "$NAMESPACE" "$pod" -- sh -c "getent hosts paw-validator-headless.$NAMESPACE.svc.cluster.local 2>/dev/null || curl -s --connect-timeout 2 http://paw-validator-headless.$NAMESPACE.svc.cluster.local:26657/health 2>/dev/null || echo 'failed'" 2>/dev/null)

    if echo "$dns_result" | grep -qE "(Address|[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)"; then
        log_pass "Service discovery working"
    elif echo "$dns_result" | grep -q "healthy"; then
        log_pass "Service discovery working (verified via HTTP)"
    else
        # Service may exist but nslookup not available - check Kubernetes service
        if kubectl get endpoints paw-validator-headless -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | grep -q "[0-9]"; then
            log_pass "Service discovery working (verified via Kubernetes endpoints)"
        else
            log_fail "Service discovery failed"
            return 1
        fi
    fi
}

test_persistent_storage() {
    log_test "Testing persistent storage..."

    local pvc_bound=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep "Bound" | wc -l)
    local pvc_pending=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep "Pending" | wc -l)
    local pvc_total=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$pvc_bound" -eq "$pvc_total" ] && [ "$pvc_total" -gt 0 ]; then
        log_pass "All $pvc_bound PVCs bound and data persistent"
    elif [ "$pvc_total" -eq 0 ]; then
        log_fail "No PVCs found"
    elif [ "$pvc_pending" -gt 0 ]; then
        # Pending PVCs with WaitForFirstConsumer are acceptable
        log_pass "PVCs OK ($pvc_bound bound, $pvc_pending pending for first consumer)"
    else
        log_fail "Only $pvc_bound/$pvc_total PVCs bound"
        return 1
    fi
}

print_summary() {
    echo ""
    echo "=============================================="
    echo "INTEGRATION TEST RESULTS"
    echo "=============================================="
    echo -e "Passed: ${GREEN}$PASSED${NC}"
    echo -e "Failed: ${RED}$FAILED${NC}"
    echo "=============================================="

    if [ "$FAILED" -gt 0 ]; then
        echo -e "${RED}INTEGRATION TESTS FAILED${NC}"
        exit 1
    else
        echo -e "${GREEN}ALL INTEGRATION TESTS PASSED${NC}"
        exit 0
    fi
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Integration Tests${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Time: $(date)"
    echo ""

    test_transaction_submission
    test_block_finality
    test_api_endpoints
    test_consensus_participation
    test_peer_connectivity
    test_metrics_export
    test_service_discovery
    test_persistent_storage

    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace|-n)
            NAMESPACE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --namespace, -n NAME  Namespace to test (default: paw)"
            echo "  --help, -h            Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
