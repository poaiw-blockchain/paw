#!/bin/bash
# smoke-tests.sh - Quick validation of PAW Kubernetes deployment
set -euo pipefail

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
WARNINGS=0

log_test() { echo -e "${BLUE}[TEST]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; ((WARNINGS++)); }

test_namespace_exists() {
    log_test "Checking namespace exists..."

    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        log_pass "Namespace $NAMESPACE exists"
    else
        log_fail "Namespace $NAMESPACE does not exist"
        return 1
    fi
}

test_pods_running() {
    log_test "Checking pods are running..."

    local pod_count=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [ "$pod_count" -eq 0 ]; then
        log_fail "No pods found in namespace $NAMESPACE"
        return 1
    fi

    local not_running=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | grep -v "Running\|Completed" | wc -l)
    if [ "$not_running" -gt 0 ]; then
        log_warn "$not_running pods not in Running state"
        kubectl get pods -n "$NAMESPACE" --no-headers | grep -v "Running\|Completed"
    else
        log_pass "All $pod_count pods are running"
    fi
}

test_validators_ready() {
    log_test "Checking validators are ready..."

    local ready=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local desired=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

    if [ "$ready" = "$desired" ] && [ "$desired" != "0" ]; then
        log_pass "All $ready/$desired validators are ready"
    else
        log_fail "Only $ready/$desired validators are ready"
        return 1
    fi
}

test_services_have_endpoints() {
    log_test "Checking services have endpoints..."

    local services=$(kubectl get svc -n "$NAMESPACE" -o name 2>/dev/null)
    local failed=0

    for svc in $services; do
        local svc_name=$(echo "$svc" | sed 's/service\///')
        local endpoints=$(kubectl get endpoints "$svc_name" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)

        if [ -z "$endpoints" ]; then
            log_warn "Service $svc_name has no endpoints"
            ((failed++))
        fi
    done

    if [ "$failed" -eq 0 ]; then
        log_pass "All services have endpoints"
    else
        log_warn "$failed services have no endpoints"
    fi
}

test_rpc_connectivity() {
    log_test "Checking RPC connectivity..."

    # Try to connect to RPC via port-forward
    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1)

    if [ -z "$pod" ]; then
        log_warn "No validator pod found for RPC test"
        return 0
    fi

    # Execute curl inside the pod
    local status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:26657/status 2>/dev/null || echo "000")

    if [ "$status" = "200" ]; then
        log_pass "RPC endpoint is responsive (HTTP $status)"
    else
        log_fail "RPC endpoint not responsive (HTTP $status)"
        return 1
    fi
}

test_block_production() {
    log_test "Checking block production..."

    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1)

    if [ -z "$pod" ]; then
        log_warn "No validator pod found for block height test"
        return 0
    fi

    # Get block height
    local height=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")

    if [ "$height" != "0" ] && [ "$height" != "null" ]; then
        log_pass "Chain is producing blocks (height: $height)"
    else
        log_warn "Cannot verify block production (height: $height)"
    fi
}

test_health_endpoint() {
    log_test "Checking health endpoints..."

    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1)

    if [ -z "$pod" ]; then
        log_warn "No validator pod found for health test"
        return 0
    fi

    local health=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:26657/health 2>/dev/null || echo "000")

    if [ "$health" = "200" ]; then
        log_pass "Health endpoint is responsive"
    else
        log_fail "Health endpoint not responsive (HTTP $health)"
        return 1
    fi
}

test_pvcs_bound() {
    log_test "Checking PVCs are bound..."

    local total=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    local bound=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep "Bound" | wc -l)

    if [ "$total" -eq 0 ]; then
        log_warn "No PVCs found"
        return 0
    fi

    if [ "$bound" -eq "$total" ]; then
        log_pass "All $bound/$total PVCs are bound"
    else
        log_fail "Only $bound/$total PVCs are bound"
        kubectl get pvc -n "$NAMESPACE" --no-headers | grep -v "Bound"
        return 1
    fi
}

test_secrets_exist() {
    log_test "Checking secrets exist..."

    local expected_secrets=("paw-validator-keys" "paw-genesis" "paw-api-secrets")
    local missing=0

    for secret in "${expected_secrets[@]}"; do
        if ! kubectl get secret "$secret" -n "$NAMESPACE" &>/dev/null; then
            log_warn "Secret $secret not found"
            ((missing++))
        fi
    done

    if [ "$missing" -eq 0 ]; then
        log_pass "All expected secrets exist"
    else
        log_warn "$missing secrets are missing (may be using configmaps or external secrets)"
    fi
}

test_network_policies() {
    log_test "Checking network policies..."

    local policy_count=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$policy_count" -gt 0 ]; then
        log_pass "$policy_count network policies configured"
    else
        log_warn "No network policies found"
    fi
}

test_resource_quotas() {
    log_test "Checking resource quotas..."

    local quota=$(kubectl get resourcequota -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$quota" -gt 0 ]; then
        log_pass "Resource quotas configured"
    else
        log_warn "No resource quotas found"
    fi
}

print_summary() {
    echo ""
    echo "=============================================="
    echo "SMOKE TEST RESULTS"
    echo "=============================================="
    echo -e "Passed:   ${GREEN}$PASSED${NC}"
    echo -e "Failed:   ${RED}$FAILED${NC}"
    echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
    echo "=============================================="

    if [ "$FAILED" -gt 0 ]; then
        echo -e "${RED}SMOKE TESTS FAILED${NC}"
        exit 1
    elif [ "$WARNINGS" -gt 0 ]; then
        echo -e "${YELLOW}SMOKE TESTS PASSED WITH WARNINGS${NC}"
        exit 0
    else
        echo -e "${GREEN}ALL SMOKE TESTS PASSED${NC}"
        exit 0
    fi
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Smoke Tests${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Time: $(date)"
    echo ""

    test_namespace_exists
    test_pods_running
    test_validators_ready
    test_services_have_endpoints
    test_pvcs_bound
    test_secrets_exist
    test_network_policies
    test_resource_quotas
    test_rpc_connectivity
    test_health_endpoint
    test_block_production

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
