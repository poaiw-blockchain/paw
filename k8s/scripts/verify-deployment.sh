#!/bin/bash
# verify-deployment.sh - Comprehensive verification of PAW Kubernetes deployment
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-paw}"
FULL="${FULL:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_header() { echo -e "\n${CYAN}=== $1 ===${NC}"; }
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

verify_cluster() {
    log_header "Cluster Status"

    echo "Context: $(kubectl config current-context)"
    echo ""

    kubectl get nodes -o wide
}

verify_namespace() {
    log_header "Namespace: $NAMESPACE"

    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        kubectl get namespace "$NAMESPACE" -o yaml | grep -A5 "labels:" | head -10
        log_success "Namespace exists"
    else
        log_error "Namespace does not exist"
        return 1
    fi
}

verify_pods() {
    log_header "Pods"

    kubectl get pods -n "$NAMESPACE" -o wide

    echo ""

    local not_running=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | grep -v "Running\|Completed" | wc -l)
    if [ "$not_running" -gt 0 ]; then
        log_warn "$not_running pods not in Running state"
    else
        log_success "All pods running"
    fi
}

verify_statefulsets() {
    log_header "StatefulSets"

    kubectl get statefulsets -n "$NAMESPACE" -o wide

    echo ""

    local sts=$(kubectl get statefulset -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for s in $sts; do
        local ready=$(kubectl get statefulset "$s" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired=$(kubectl get statefulset "$s" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [ "$ready" = "$desired" ]; then
            log_success "$s: $ready/$desired ready"
        else
            log_warn "$s: $ready/$desired ready"
        fi
    done
}

verify_services() {
    log_header "Services"

    kubectl get services -n "$NAMESPACE" -o wide

    echo ""

    local services=$(kubectl get svc -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for svc in $services; do
        local endpoints=$(kubectl get endpoints "$svc" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)

        if [ -n "$endpoints" ]; then
            log_success "Service $svc has endpoints"
        else
            log_warn "Service $svc has no endpoints"
        fi
    done
}

verify_pvcs() {
    log_header "Persistent Volume Claims"

    kubectl get pvc -n "$NAMESPACE" -o wide

    echo ""

    local bound=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep "Bound" | wc -l)
    local total=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$bound" -eq "$total" ] && [ "$total" -gt 0 ]; then
        log_success "All $bound PVCs bound"
    elif [ "$total" -eq 0 ]; then
        log_warn "No PVCs found"
    else
        log_warn "Only $bound/$total PVCs bound"
    fi
}

verify_configmaps() {
    log_header "ConfigMaps"

    kubectl get configmaps -n "$NAMESPACE"
}

verify_secrets() {
    log_header "Secrets"

    kubectl get secrets -n "$NAMESPACE" | grep -v "default-token\|kubernetes.io"

    echo ""

    # Check external secrets
    if kubectl get crd externalsecrets.external-secrets.io &>/dev/null; then
        echo "External Secrets:"
        kubectl get externalsecrets -n "$NAMESPACE" 2>/dev/null || echo "None"
    fi
}

verify_network_policies() {
    log_header "Network Policies"

    kubectl get networkpolicies -n "$NAMESPACE"
}

verify_hpa() {
    log_header "Horizontal Pod Autoscalers"

    kubectl get hpa -n "$NAMESPACE" 2>/dev/null || echo "None configured"
}

verify_pdb() {
    log_header "Pod Disruption Budgets"

    kubectl get pdb -n "$NAMESPACE" 2>/dev/null || echo "None configured"
}

verify_vault() {
    log_header "Vault Integration"

    if kubectl get namespace vault &>/dev/null; then
        echo "Vault Pods:"
        kubectl get pods -n vault

        echo ""
        echo "ClusterSecretStore:"
        kubectl get clustersecretstore 2>/dev/null || echo "None"
    else
        log_warn "Vault namespace not found"
    fi
}

verify_monitoring() {
    log_header "Monitoring Stack"

    if kubectl get namespace monitoring &>/dev/null; then
        echo "Monitoring Pods:"
        kubectl get pods -n monitoring

        echo ""
        echo "ServiceMonitors:"
        kubectl get servicemonitors -A 2>/dev/null | grep -i paw || echo "None for PAW"
    else
        log_warn "Monitoring namespace not found"
    fi
}

verify_linkerd() {
    log_header "Linkerd Service Mesh"

    if kubectl get namespace linkerd &>/dev/null; then
        linkerd check --proxy -n "$NAMESPACE" 2>/dev/null || log_warn "Linkerd check failed"
    else
        log_warn "Linkerd not installed"
    fi
}

verify_chain_health() {
    log_header "Chain Health"

    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1)

    if [ -z "$pod" ]; then
        log_warn "No validator pod found"
        return 0
    fi

    echo "Checking RPC status..."

    local status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null || echo "{}")

    if echo "$status" | jq -e '.result' &>/dev/null; then
        local height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height')
        local catching_up=$(echo "$status" | jq -r '.result.sync_info.catching_up')
        local peers=$(echo "$status" | jq -r '.result.node_info.network')

        echo "  Block Height: $height"
        echo "  Catching Up:  $catching_up"
        echo "  Network:      $peers"

        if [ "$catching_up" = "false" ]; then
            log_success "Chain is healthy"
        else
            log_warn "Chain is still syncing"
        fi
    else
        log_warn "Cannot get chain status"
    fi
}

verify_resource_usage() {
    log_header "Resource Usage"

    echo "Top Pods by CPU:"
    kubectl top pods -n "$NAMESPACE" --sort-by=cpu 2>/dev/null | head -5 || echo "Metrics not available"

    echo ""
    echo "Top Pods by Memory:"
    kubectl top pods -n "$NAMESPACE" --sort-by=memory 2>/dev/null | head -5 || echo "Metrics not available"
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}Verification Complete${NC}"
    echo "=============================================="
    echo ""
    echo "Quick Commands:"
    echo "  Logs:    kubectl logs -f <pod> -n $NAMESPACE"
    echo "  Exec:    kubectl exec -it <pod> -n $NAMESPACE -- sh"
    echo "  Status:  kubectl exec <pod> -n $NAMESPACE -- pawd status"
    echo ""
    echo "Tests:"
    echo "  Smoke:    ./k8s/tests/smoke-tests.sh"
    echo "  Security: ./k8s/tests/security-tests.sh"
    echo "  Chaos:    ./k8s/tests/chaos-tests.sh"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Deployment Verification${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Time: $(date)"

    verify_cluster
    verify_namespace
    verify_pods
    verify_statefulsets
    verify_services
    verify_pvcs
    verify_configmaps
    verify_secrets
    verify_network_policies
    verify_hpa
    verify_pdb

    if [ "$FULL" = "true" ]; then
        verify_vault
        verify_monitoring
        verify_linkerd
        verify_chain_health
        verify_resource_usage
    fi

    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace|-n)
            NAMESPACE="$2"
            shift 2
            ;;
        --full|-f)
            FULL="true"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --namespace, -n NAME  Namespace to verify (default: paw)"
            echo "  --full, -f            Run full verification including external components"
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
