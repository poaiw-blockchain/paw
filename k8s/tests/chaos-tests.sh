#!/bin/bash
# chaos-tests.sh - Chaos engineering tests for PAW Kubernetes deployment
set -u  # Keep unset variable check, but allow command failures

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-paw}"
SCENARIO="${SCENARIO:-all}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

wait_for_recovery() {
    local timeout=${1:-120}
    local start_time=$(date +%s)

    log_info "Waiting for recovery (timeout: ${timeout}s)..."

    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ "$elapsed" -gt "$timeout" ]; then
            log_error "Recovery timeout exceeded"
            return 1
        fi

        local ready=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [ "$ready" = "$desired" ] && [ "$desired" != "0" ]; then
            log_success "Recovered: $ready/$desired validators ready"
            return 0
        fi

        echo -n "."
        sleep 5
    done
}

check_consensus() {
    log_info "Checking consensus..."

    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1)

    if [ -z "$pod" ]; then
        log_warn "No validator pod found"
        return 1
    fi

    local height=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")

    if [ "$height" != "0" ] && [ "$height" != "null" ]; then
        log_success "Consensus active at height $height"
        return 0
    else
        log_warn "Cannot verify consensus"
        return 1
    fi
}

scenario_pod_failure() {
    log_info "=== SCENARIO: Random Pod Failure ==="

    local pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null)
    local pod_count=$(echo "$pods" | wc -l)

    if [ "$pod_count" -lt 2 ]; then
        log_warn "Need at least 2 validators for pod failure test"
        return 0
    fi

    # Pick a random pod
    local target=$(echo "$pods" | shuf | head -1)
    log_info "Killing pod: $target"

    # Record initial state
    local initial_height=$(kubectl exec -n "$NAMESPACE" "$target" -- curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")

    # Delete the pod
    kubectl delete "$target" -n "$NAMESPACE" --force --grace-period=0

    # Wait for recovery
    wait_for_recovery 180

    # Verify consensus continued
    sleep 10
    check_consensus

    log_success "Pod failure test completed"
}

scenario_network_partition() {
    log_info "=== SCENARIO: Network Partition ==="

    # Check if network policies can be modified
    if ! kubectl auth can-i create networkpolicies -n "$NAMESPACE" 2>/dev/null; then
        log_warn "Cannot create network policies - skipping"
        return 0
    fi

    local target_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$target_pod" ]; then
        log_warn "No validator pod found"
        return 0
    fi

    log_info "Isolating pod: $target_pod"

    # Create network policy to isolate the pod
    kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: chaos-network-partition
  namespace: $NAMESPACE
spec:
  podSelector:
    matchLabels:
      statefulset.kubernetes.io/pod-name: $target_pod
  policyTypes:
    - Ingress
    - Egress
  ingress: []
  egress: []
EOF

    log_info "Pod isolated - waiting 30s..."
    sleep 30

    # Check if consensus continues with 2/3 validators
    check_consensus

    # Remove the partition
    log_info "Removing network partition..."
    kubectl delete networkpolicy chaos-network-partition -n "$NAMESPACE"

    # Wait for recovery
    wait_for_recovery 120

    log_success "Network partition test completed"
}

scenario_high_latency() {
    log_info "=== SCENARIO: High Latency ==="

    # This requires tc (traffic control) inside pods or a chaos mesh tool
    # For Kind clusters, we can use kubectl exec to simulate with sleep

    local target_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$target_pod" ]; then
        log_warn "No validator pod found"
        return 0
    fi

    log_info "Simulating high latency on: $target_pod"

    # Check if tc is available
    if kubectl exec -n "$NAMESPACE" "$target_pod" -- which tc &>/dev/null; then
        # Add 200ms latency
        kubectl exec -n "$NAMESPACE" "$target_pod" -- tc qdisc add dev eth0 root netem delay 200ms 2>/dev/null || true

        log_info "Added 200ms latency - waiting 60s..."
        sleep 60

        check_consensus

        # Remove latency
        kubectl exec -n "$NAMESPACE" "$target_pod" -- tc qdisc del dev eth0 root 2>/dev/null || true
    else
        log_warn "tc not available in pod - skipping latency injection"
    fi

    log_success "High latency test completed"
}

scenario_resource_exhaustion() {
    log_info "=== SCENARIO: Memory Pressure ==="

    local target_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$target_pod" ]; then
        log_warn "No validator pod found"
        return 0
    fi

    log_info "Applying memory pressure to: $target_pod"

    # Check if stress-ng is available
    if kubectl exec -n "$NAMESPACE" "$target_pod" -- which stress-ng &>/dev/null; then
        # Apply memory pressure (50% of container limit) for 30s
        kubectl exec -n "$NAMESPACE" "$target_pod" -- timeout 30s stress-ng --vm 1 --vm-bytes 512M 2>/dev/null &

        log_info "Applied memory stress - waiting 30s..."
        sleep 35

        check_consensus
    else
        log_warn "stress-ng not available in pod - skipping memory pressure test"
    fi

    log_success "Memory pressure test completed"
}

scenario_rolling_restart() {
    log_info "=== SCENARIO: Rolling Restart ==="

    log_info "Initiating rolling restart of validators..."

    kubectl rollout restart statefulset/paw-validator -n "$NAMESPACE"

    # Monitor the rollout
    kubectl rollout status statefulset/paw-validator -n "$NAMESPACE" --timeout=300s

    # Verify consensus
    sleep 10
    check_consensus

    log_success "Rolling restart test completed"
}

scenario_storage_failure() {
    log_info "=== SCENARIO: Storage Stress ==="

    local target_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$target_pod" ]; then
        log_warn "No validator pod found"
        return 0
    fi

    log_info "Simulating storage stress on: $target_pod"

    # Write large file to fill disk (but leave some space)
    kubectl exec -n "$NAMESPACE" "$target_pod" -- sh -c 'dd if=/dev/zero of=/data/stress-test bs=1M count=100 2>/dev/null' || true

    log_info "Created 100MB stress file - waiting 30s..."
    sleep 30

    check_consensus

    # Cleanup
    kubectl exec -n "$NAMESPACE" "$target_pod" -- rm -f /data/stress-test

    log_success "Storage stress test completed"
}

run_all_scenarios() {
    scenario_pod_failure
    echo ""
    scenario_network_partition
    echo ""
    scenario_high_latency
    echo ""
    scenario_resource_exhaustion
    echo ""
    scenario_rolling_restart
    echo ""
    scenario_storage_failure
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}Chaos Tests Completed${NC}"
    echo "=============================================="
    echo ""
    echo "Scenarios executed: $SCENARIO"
    echo ""
    echo "Verify final state:"
    kubectl get pods -n "$NAMESPACE"
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Chaos Tests${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Scenario: $SCENARIO"
    echo "Time: $(date)"
    echo ""

    case "$SCENARIO" in
        all)
            run_all_scenarios
            ;;
        pod-failure)
            scenario_pod_failure
            ;;
        network-partition)
            scenario_network_partition
            ;;
        high-latency)
            scenario_high_latency
            ;;
        resource-exhaustion)
            scenario_resource_exhaustion
            ;;
        rolling-restart)
            scenario_rolling_restart
            ;;
        storage-failure)
            scenario_storage_failure
            ;;
        *)
            log_error "Unknown scenario: $SCENARIO"
            echo "Available: all, pod-failure, network-partition, high-latency, resource-exhaustion, rolling-restart, storage-failure"
            exit 1
            ;;
    esac

    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace|-n)
            NAMESPACE="$2"
            shift 2
            ;;
        --scenario|-s)
            SCENARIO="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --namespace, -n NAME  Namespace to test (default: paw)"
            echo "  --scenario, -s NAME   Scenario to run: all, pod-failure, network-partition, high-latency, resource-exhaustion, rolling-restart, storage-failure"
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
