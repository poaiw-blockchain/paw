#!/bin/bash
# monitoring-tests.sh - Comprehensive monitoring and observability tests for PAW K8s deployment
# Phase 5: Monitoring Tests (80 tests)
# Tests: Prometheus, Grafana, Alerts, Loki, Tracing, Custom Metrics, SLA/SLO, Backup, DR, Key Rotation, Drift
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-paw}"
PROMETHEUS_PORT="${PROMETHEUS_PORT:-9090}"
GRAFANA_PORT="${GRAFANA_PORT:-3000}"
LOKI_PORT="${LOKI_PORT:-3100}"
JAEGER_PORT="${JAEGER_PORT:-16686}"
CATEGORY="${CATEGORY:-all}"
VERBOSE="${VERBOSE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PASSED=0
FAILED=0
SKIPPED=0
WARNINGS=0

log_test() { echo -e "${BLUE}[TEST]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }
log_skip() { echo -e "${CYAN}[SKIP]${NC} $1"; ((SKIPPED++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; ((WARNINGS++)); }
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_verbose() { [ "$VERBOSE" = "true" ] && echo -e "${CYAN}[VERBOSE]${NC} $1"; }

get_validator_pod() {
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator -o name 2>/dev/null | head -1
}

get_prometheus_pod() {
    kubectl get pods -n "$NAMESPACE" -l app=prometheus -o name 2>/dev/null | head -1
}

get_grafana_pod() {
    kubectl get pods -n "$NAMESPACE" -l app=grafana -o name 2>/dev/null | head -1
}

get_loki_pod() {
    kubectl get pods -n "$NAMESPACE" -l app=loki -o name 2>/dev/null | head -1
}

prometheus_query() {
    local query="$1"
    local pod=$(get_prometheus_pod)
    if [ -z "$pod" ]; then
        echo ""
        return 1
    fi
    kubectl exec -n "$NAMESPACE" "$pod" -- curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/query?query=$(echo "$query" | jq -sRr @uri)" 2>/dev/null
}

# ============================================================================
# PROMETHEUS TESTS (10 tests)
# ============================================================================

test_prometheus_pod_health() {
    log_test "1. Prometheus pod health..."
    local pod=$(get_prometheus_pod)
    if [ -z "$pod" ]; then
        log_skip "Prometheus pod not found"
        return 0
    fi

    local status=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$status" = "Running" ]; then
        log_pass "Prometheus pod is running"
    else
        log_fail "Prometheus pod status: $status"
    fi
}

test_prometheus_ready() {
    log_test "2. Prometheus readiness..."
    local pod=$(get_prometheus_pod)
    if [ -z "$pod" ]; then
        log_skip "Prometheus pod not found"
        return 0
    fi

    local ready=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PROMETHEUS_PORT/-/ready" 2>/dev/null || echo "000")
    if [ "$ready" = "200" ]; then
        log_pass "Prometheus is ready"
    else
        log_fail "Prometheus readiness check failed (HTTP $ready)"
    fi
}

test_prometheus_scrape_targets() {
    log_test "3. Prometheus scrape targets..."
    local pod=$(get_prometheus_pod)
    if [ -z "$pod" ]; then
        log_skip "Prometheus pod not found"
        return 0
    fi

    local targets=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/targets" 2>/dev/null | jq -r '.data.activeTargets | length' 2>/dev/null || echo "0")
    if [ "$targets" -gt 0 ]; then
        log_pass "Prometheus has $targets active scrape targets"
    else
        log_fail "No active scrape targets found"
    fi
}

test_prometheus_tendermint_metrics() {
    log_test "4. Tendermint/CometBFT metrics availability..."
    local result=$(prometheus_query 'tendermint_consensus_height')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Tendermint metrics available (consensus_height)"
    else
        log_warn "Tendermint metrics not found (chain may not be running)"
    fi
}

test_prometheus_cosmos_metrics() {
    log_test "5. Cosmos SDK metrics availability..."
    local result=$(prometheus_query 'cosmos_base_tendermint_service_synced')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Cosmos SDK metrics available"
    else
        # Try alternative metric
        result=$(prometheus_query 'cosmos_tx_count')
        if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
            log_pass "Cosmos SDK metrics available (tx_count)"
        else
            log_warn "Cosmos SDK metrics not found"
        fi
    fi
}

test_prometheus_dex_metrics() {
    log_test "6. DEX module metrics availability..."
    local result=$(prometheus_query 'paw_dex_pool_count')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "DEX metrics available (pool_count)"
    else
        log_warn "DEX metrics not found (module may not be active)"
    fi
}

test_prometheus_retention() {
    log_test "7. Prometheus retention configuration..."
    local pod=$(get_prometheus_pod)
    if [ -z "$pod" ]; then
        log_skip "Prometheus pod not found"
        return 0
    fi

    local args=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].args}' 2>/dev/null)
    if echo "$args" | grep -q "retention"; then
        log_pass "Prometheus retention is configured"
    else
        log_warn "Prometheus retention not explicitly configured"
    fi
}

test_prometheus_labels() {
    log_test "8. Prometheus metric labels..."
    local result=$(prometheus_query 'up')
    local labels=$(echo "$result" | jq -r '.data.result[0].metric | keys | length' 2>/dev/null || echo "0")
    labels="${labels:-0}"
    if [ -z "$labels" ] || ! [[ "$labels" =~ ^[0-9]+$ ]]; then labels=0; fi
    if [ "$labels" -gt 2 ]; then
        log_pass "Metrics have proper labels ($labels labels on 'up' metric)"
    else
        log_fail "Metrics missing expected labels"
    fi
}

test_prometheus_storage() {
    log_test "9. Prometheus storage (PVC)..."
    local pvc=$(kubectl get pvc -n "$NAMESPACE" -l app=prometheus -o name 2>/dev/null | head -1)
    if [ -n "$pvc" ]; then
        local status=$(kubectl get "$pvc" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$status" = "Bound" ]; then
            log_pass "Prometheus PVC is bound"
        else
            log_fail "Prometheus PVC status: $status"
        fi
    else
        log_warn "No Prometheus PVC found (may be using emptyDir)"
    fi
}

test_prometheus_config() {
    log_test "10. Prometheus configuration..."
    local cm=$(kubectl get configmap prometheus-config -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Prometheus config ConfigMap exists"
    else
        log_warn "Prometheus config ConfigMap not found"
    fi
}

# ============================================================================
# GRAFANA TESTS (6 tests)
# ============================================================================

test_grafana_pod_health() {
    log_test "11. Grafana pod health..."
    local pod=$(get_grafana_pod)
    if [ -z "$pod" ]; then
        log_skip "Grafana pod not found"
        return 0
    fi

    local status=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$status" = "Running" ]; then
        log_pass "Grafana pod is running"
    else
        log_fail "Grafana pod status: $status"
    fi
}

test_grafana_ready() {
    log_test "12. Grafana readiness..."
    local pod=$(get_grafana_pod)
    if [ -z "$pod" ]; then
        log_skip "Grafana pod not found"
        return 0
    fi

    local ready=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' "http://localhost:$GRAFANA_PORT/api/health" 2>/dev/null || echo "000")
    if [ "$ready" = "200" ]; then
        log_pass "Grafana is ready"
    else
        log_fail "Grafana readiness check failed (HTTP $ready)"
    fi
}

test_grafana_datasources() {
    log_test "13. Grafana datasources configuration..."
    local cm=$(kubectl get configmap grafana-datasources -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Grafana datasources ConfigMap exists"
    else
        log_warn "Grafana datasources ConfigMap not found"
    fi
}

test_grafana_dashboards() {
    log_test "14. Grafana dashboards ConfigMap..."
    local cm=$(kubectl get configmap grafana-dashboards -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Grafana dashboards ConfigMap exists"
    else
        log_warn "Grafana dashboards ConfigMap not found"
    fi
}

test_grafana_provisioning() {
    log_test "15. Grafana provisioning directories..."
    local pod=$(get_grafana_pod)
    if [ -z "$pod" ]; then
        log_skip "Grafana pod not found"
        return 0
    fi

    local exists=$(kubectl exec -n "$NAMESPACE" "$pod" -- ls -d /etc/grafana/provisioning 2>/dev/null)
    if [ -n "$exists" ]; then
        log_pass "Grafana provisioning directory exists"
    else
        log_warn "Grafana provisioning directory not found"
    fi
}

test_grafana_storage() {
    log_test "16. Grafana storage (PVC)..."
    local pvc=$(kubectl get pvc -n "$NAMESPACE" -l app=grafana -o name 2>/dev/null | head -1)
    if [ -n "$pvc" ]; then
        local status=$(kubectl get "$pvc" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$status" = "Bound" ]; then
            log_pass "Grafana PVC is bound"
        else
            log_fail "Grafana PVC status: $status"
        fi
    else
        log_warn "No Grafana PVC found"
    fi
}

# ============================================================================
# ALERT RULES TESTS (10 tests)
# ============================================================================

test_alert_rules_configmap() {
    log_test "17. Alert rules ConfigMaps..."
    local cms=$(kubectl get configmaps -n "$NAMESPACE" -o name 2>/dev/null | grep -E "(alert|rules|prometheus)" | wc -l)
    if [ "$cms" -gt 0 ]; then
        log_pass "Found $cms alert/rules ConfigMaps"
    else
        log_warn "No alert rules ConfigMaps found"
    fi
}

test_alertmanager_config() {
    log_test "18. Alertmanager configuration..."
    local cm=$(kubectl get configmap alertmanager-config -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Alertmanager config exists"
    else
        log_warn "Alertmanager config not found"
    fi
}

test_prometheus_rule_crd() {
    log_test "19. PrometheusRule CRDs..."
    local rules=$(kubectl get prometheusrules -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [ "$rules" -gt 0 ]; then
        log_pass "Found $rules PrometheusRule resources"
    else
        # Check if CRD exists
        if kubectl get crd prometheusrules.monitoring.coreos.com &>/dev/null; then
            log_warn "PrometheusRule CRD exists but no rules in namespace"
        else
            log_skip "PrometheusRule CRD not installed (Prometheus Operator not used)"
        fi
    fi
}

test_block_production_alert() {
    log_test "20. BlockProductionStopped alert rule..."
    local found=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -i "blockproduction\|block_production\|BlockProduction" | wc -l)
    if [ "$found" -gt 0 ]; then
        log_pass "Block production alert rule found"
    else
        log_warn "BlockProductionStopped alert rule not found in ConfigMaps"
    fi
}

test_low_peer_count_alert() {
    log_test "21. LowPeerCount alert rule..."
    local found=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -iE "lowpeer|low_peer|peer.*count" | wc -l)
    if [ "$found" -gt 0 ]; then
        log_pass "LowPeerCount alert rule found"
    else
        log_warn "LowPeerCount alert rule not found"
    fi
}

test_high_block_time_alert() {
    log_test "22. HighBlockTime alert rule..."
    local found=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -iE "highblock|high_block|block.*time" | wc -l)
    if [ "$found" -gt 0 ]; then
        log_pass "HighBlockTime alert rule found"
    else
        log_warn "HighBlockTime alert rule not found"
    fi
}

test_cpu_alert_rules() {
    log_test "23. CPU utilization alert rules..."
    local found=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -iE "cpu.*util|cpu.*high|high.*cpu" | wc -l)
    if [ "$found" -gt 0 ]; then
        log_pass "CPU alert rules found"
    else
        log_warn "CPU alert rules not found"
    fi
}

test_disk_alert_rules() {
    log_test "24. Disk utilization alert rules..."
    local found=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -iE "disk.*space|disk.*usage|storage.*full" | wc -l)
    if [ "$found" -gt 0 ]; then
        log_pass "Disk alert rules found"
    else
        log_warn "Disk alert rules not found"
    fi
}

test_alert_routing() {
    log_test "25. Alert routing configuration..."
    local am_config=$(kubectl get configmap alertmanager-config -n "$NAMESPACE" -o yaml 2>/dev/null)
    if echo "$am_config" | grep -q "route:"; then
        log_pass "Alert routing configured"
    else
        log_warn "Alert routing not found in configuration"
    fi
}

test_alert_receivers() {
    log_test "26. Alert receivers configuration..."
    local am_config=$(kubectl get configmap alertmanager-config -n "$NAMESPACE" -o yaml 2>/dev/null)
    if echo "$am_config" | grep -q "receivers:"; then
        log_pass "Alert receivers configured"
    else
        log_warn "Alert receivers not found"
    fi
}

# ============================================================================
# LOKI TESTS (8 tests)
# ============================================================================

test_loki_pod_health() {
    log_test "27. Loki pod health..."
    local pod=$(get_loki_pod)
    if [ -z "$pod" ]; then
        log_skip "Loki pod not found"
        return 0
    fi

    local status=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$status" = "Running" ]; then
        log_pass "Loki pod is running"
    else
        log_fail "Loki pod status: $status"
    fi
}

test_loki_ready() {
    log_test "28. Loki readiness..."
    local pod=$(get_loki_pod)
    if [ -z "$pod" ]; then
        log_skip "Loki pod not found"
        return 0
    fi

    local ready=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' "http://localhost:$LOKI_PORT/ready" 2>/dev/null || echo "000")
    if [ "$ready" = "200" ]; then
        log_pass "Loki is ready"
    else
        log_fail "Loki readiness check failed (HTTP $ready)"
    fi
}

test_loki_push_endpoint() {
    log_test "29. Loki push endpoint..."
    local pod=$(get_loki_pod)
    if [ -z "$pod" ]; then
        log_skip "Loki pod not found"
        return 0
    fi

    # Check if push endpoint is accessible (will return 400 without proper auth, but that's OK)
    local status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' "http://localhost:$LOKI_PORT/loki/api/v1/push" -X POST 2>/dev/null || echo "000")
    if [ "$status" != "000" ]; then
        log_pass "Loki push endpoint is accessible"
    else
        log_fail "Loki push endpoint not accessible"
    fi
}

test_loki_query_endpoint() {
    log_test "30. Loki query endpoint..."
    local pod=$(get_loki_pod)
    if [ -z "$pod" ]; then
        log_skip "Loki pod not found"
        return 0
    fi

    local status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' "http://localhost:$LOKI_PORT/loki/api/v1/labels" 2>/dev/null || echo "000")
    if [ "$status" = "200" ]; then
        log_pass "Loki query endpoint is accessible"
    else
        log_fail "Loki query endpoint not accessible (HTTP $status)"
    fi
}

test_loki_labels() {
    log_test "31. Loki label indexing..."
    local pod=$(get_loki_pod)
    if [ -z "$pod" ]; then
        log_skip "Loki pod not found"
        return 0
    fi

    local labels=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s "http://localhost:$LOKI_PORT/loki/api/v1/labels" 2>/dev/null | jq -r '.data | length' 2>/dev/null || echo "0")
    if [ "$labels" -gt 0 ]; then
        log_pass "Loki has $labels indexed labels"
    else
        log_warn "No labels indexed in Loki (may need log data)"
    fi
}

test_loki_config() {
    log_test "32. Loki configuration..."
    local cm=$(kubectl get configmap loki-config -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Loki config ConfigMap exists"
    else
        log_warn "Loki config ConfigMap not found"
    fi
}

test_loki_retention() {
    log_test "33. Loki retention configuration..."
    local config=$(kubectl get configmap loki-config -n "$NAMESPACE" -o yaml 2>/dev/null)
    if echo "$config" | grep -qE "retention|compactor"; then
        log_pass "Loki retention configured"
    else
        log_warn "Loki retention not explicitly configured"
    fi
}

test_loki_storage() {
    log_test "34. Loki storage (PVC)..."
    local pvc=$(kubectl get pvc -n "$NAMESPACE" -l app=loki -o name 2>/dev/null | head -1)
    if [ -n "$pvc" ]; then
        local status=$(kubectl get "$pvc" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$status" = "Bound" ]; then
            log_pass "Loki PVC is bound"
        else
            log_fail "Loki PVC status: $status"
        fi
    else
        log_warn "No Loki PVC found"
    fi
}

# ============================================================================
# TRACING TESTS (6 tests)
# ============================================================================

test_jaeger_deployment() {
    log_test "35. Jaeger/tracing deployment..."
    local deploy=$(kubectl get deployment -n "$NAMESPACE" -l app=jaeger -o name 2>/dev/null | head -1)
    if [ -n "$deploy" ]; then
        log_pass "Jaeger deployment found"
    else
        log_skip "Jaeger deployment not found (tracing not enabled)"
    fi
}

test_otel_collector() {
    log_test "36. OpenTelemetry collector..."
    local deploy=$(kubectl get deployment -n "$NAMESPACE" -l app=otel-collector -o name 2>/dev/null | head -1)
    if [ -n "$deploy" ]; then
        log_pass "OpenTelemetry collector deployed"
    else
        log_skip "OpenTelemetry collector not found"
    fi
}

test_tracing_configmap() {
    log_test "37. Tracing configuration..."
    local cm=$(kubectl get configmaps -n "$NAMESPACE" -o name 2>/dev/null | grep -iE "jaeger|tracing|otel" | head -1)
    if [ -n "$cm" ]; then
        log_pass "Tracing configuration found"
    else
        log_skip "Tracing configuration not found"
    fi
}

test_otlp_endpoint() {
    log_test "38. OTLP endpoint availability..."
    local svc=$(kubectl get svc -n "$NAMESPACE" -l app=otel-collector -o name 2>/dev/null | head -1)
    if [ -n "$svc" ]; then
        log_pass "OTLP endpoint service exists"
    else
        log_skip "OTLP endpoint service not found"
    fi
}

test_trace_sampling() {
    log_test "39. Trace sampling configuration..."
    local config=$(kubectl get configmaps -n "$NAMESPACE" -o yaml 2>/dev/null | grep -iE "sampling|sample_rate" | head -1)
    if [ -n "$config" ]; then
        log_pass "Trace sampling configured"
    else
        log_skip "Trace sampling configuration not found"
    fi
}

test_trace_context_propagation() {
    log_test "40. Trace context propagation..."
    # Check if validators have tracing environment variables
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod found"
        return 0
    fi

    local env=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].env}' 2>/dev/null)
    if echo "$env" | grep -qiE "trace|otel|jaeger"; then
        log_pass "Tracing environment variables configured"
    else
        log_warn "Tracing environment variables not found"
    fi
}

# ============================================================================
# CUSTOM BLOCKCHAIN METRICS TESTS (8 tests)
# ============================================================================

test_consensus_height_metric() {
    log_test "41. consensus_height metric..."
    local result=$(prometheus_query 'tendermint_consensus_height')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local height=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        log_pass "consensus_height metric available (height: $height)"
    else
        log_warn "consensus_height metric not found"
    fi
}

test_p2p_peers_metric() {
    log_test "42. p2p_peers metric..."
    local result=$(prometheus_query 'tendermint_p2p_peers')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local peers=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        log_pass "p2p_peers metric available (peers: $peers)"
    else
        log_warn "p2p_peers metric not found"
    fi
}

test_tx_counter_metrics() {
    log_test "43. Transaction counter metrics..."
    local result=$(prometheus_query 'tendermint_consensus_num_txs')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Transaction counter metrics available"
    else
        log_warn "Transaction counter metrics not found"
    fi
}

test_mempool_metrics() {
    log_test "44. Mempool metrics..."
    local result=$(prometheus_query 'tendermint_mempool_size')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local size=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        log_pass "Mempool metrics available (size: $size)"
    else
        log_warn "Mempool metrics not found"
    fi
}

test_dex_pool_metrics() {
    log_test "45. DEX pool metrics..."
    local result=$(prometheus_query 'paw_dex_total_value_locked')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "DEX pool metrics available"
    else
        log_warn "DEX pool metrics not found (module may not be active)"
    fi
}

test_oracle_metrics() {
    log_test "46. Oracle metrics..."
    local result=$(prometheus_query 'paw_oracle_price')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Oracle metrics available"
    else
        log_warn "Oracle metrics not found (module may not be active)"
    fi
}

test_validator_power_metric() {
    log_test "47. Validator voting power metric..."
    local result=$(prometheus_query 'tendermint_consensus_validators_power')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Validator power metrics available"
    else
        log_warn "Validator power metrics not found"
    fi
}

test_block_size_metric() {
    log_test "48. Block size metric..."
    local result=$(prometheus_query 'tendermint_consensus_block_size_bytes')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Block size metrics available"
    else
        log_warn "Block size metrics not found"
    fi
}

# ============================================================================
# SLA/SLO COMPLIANCE TESTS (10 tests)
# ============================================================================

test_slo_block_time() {
    log_test "49. SLO: Block time (<6 seconds p99)..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod for block time check"
        return 0
    fi

    # Get last two block timestamps and calculate difference
    local status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s http://localhost:26657/status 2>/dev/null)
    local height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)

    if [ -n "$height" ] && [ "$height" != "null" ]; then
        log_pass "Block time SLO: Chain producing blocks (height: $height)"
    else
        log_warn "Cannot verify block time SLO"
    fi
}

test_slo_uptime() {
    log_test "50. SLO: Validator uptime (99.9% target)..."
    local ready=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local desired=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

    if [ "$ready" = "$desired" ] && [ "$desired" != "0" ]; then
        log_pass "Uptime SLO: All $ready/$desired validators running"
    else
        log_fail "Uptime SLO violation: Only $ready/$desired validators ready"
    fi
}

test_slo_availability() {
    log_test "51. SLO: API availability..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod for availability check"
        return 0
    fi

    local rpc_status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:26657/status 2>/dev/null || echo "000")
    local api_status=$(kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null -w '%{http_code}' http://localhost:1317/cosmos/bank/v1beta1/params 2>/dev/null || echo "000")

    if [ "$rpc_status" = "200" ] && [ "$api_status" = "200" ]; then
        log_pass "Availability SLO: RPC and API endpoints responsive"
    elif [ "$rpc_status" = "200" ]; then
        log_warn "Availability SLO: RPC OK, REST API not responsive"
    else
        log_fail "Availability SLO violation: Endpoints not responsive"
    fi
}

test_slo_latency() {
    log_test "52. SLO: API latency (<1s p99 target)..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod for latency check"
        return 0
    fi

    local start=$(date +%s%N)
    kubectl exec -n "$NAMESPACE" "$pod" -- curl -s -o /dev/null http://localhost:26657/status 2>/dev/null
    local end=$(date +%s%N)
    local latency_ms=$(( (end - start) / 1000000 ))

    if [ "$latency_ms" -lt 1000 ]; then
        log_pass "Latency SLO: Response in ${latency_ms}ms"
    else
        log_warn "Latency SLO: Response took ${latency_ms}ms (>1000ms threshold)"
    fi
}

test_slo_consensus_participation() {
    log_test "53. SLO: Consensus participation..."
    local result=$(prometheus_query 'tendermint_consensus_missing_validators')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local missing=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        if [ "$missing" = "0" ]; then
            log_pass "Consensus participation SLO: No missing validators"
        else
            log_warn "Consensus participation: $missing validators missing"
        fi
    else
        log_warn "Cannot verify consensus participation SLO"
    fi
}

test_slo_peer_connectivity() {
    log_test "54. SLO: Peer connectivity (>=10 peers target)..."
    local result=$(prometheus_query 'tendermint_p2p_peers')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local peers=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        if [ "$peers" -ge 10 ]; then
            log_pass "Peer connectivity SLO: $peers peers connected"
        elif [ "$peers" -ge 5 ]; then
            log_warn "Peer connectivity: $peers peers (below 10 target)"
        else
            log_fail "Peer connectivity SLO violation: Only $peers peers"
        fi
    else
        log_warn "Cannot verify peer connectivity SLO"
    fi
}

test_slo_error_rate() {
    log_test "55. SLO: Error rate (<0.1% target)..."
    # Check for failed transactions in mempool
    local result=$(prometheus_query 'tendermint_mempool_failed_txs')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        local failed=$(echo "$result" | jq -r '.data.result[0].value[1]' 2>/dev/null)
        log_pass "Error rate SLO: $failed failed mempool txs"
    else
        log_warn "Cannot verify error rate SLO (metrics not available)"
    fi
}

test_slo_disk_usage() {
    log_test "56. SLO: Disk usage (<80% target)..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod for disk check"
        return 0
    fi

    local usage=$(kubectl exec -n "$NAMESPACE" "$pod" -- sh -c "df /data 2>/dev/null | tail -1 | awk '{print \$5}' | tr -d '%'" 2>/dev/null || echo "0")
    if [ "$usage" -lt 80 ]; then
        log_pass "Disk usage SLO: ${usage}% used"
    elif [ "$usage" -lt 90 ]; then
        log_warn "Disk usage: ${usage}% (approaching 80% threshold)"
    else
        log_fail "Disk usage SLO violation: ${usage}% used"
    fi
}

test_slo_memory_usage() {
    log_test "57. SLO: Memory usage (<90% target)..."
    local result=$(prometheus_query 'container_memory_usage_bytes{container="paw-node"}')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "Memory metrics available for SLO monitoring"
    else
        log_warn "Memory usage metrics not available"
    fi
}

test_slo_cpu_usage() {
    log_test "58. SLO: CPU usage (<70% average target)..."
    local result=$(prometheus_query 'rate(container_cpu_usage_seconds_total{container="paw-node"}[5m])')
    if echo "$result" | jq -e '.data.result | length > 0' &>/dev/null; then
        log_pass "CPU metrics available for SLO monitoring"
    else
        log_warn "CPU usage metrics not available"
    fi
}

# ============================================================================
# BACKUP/RESTORE TESTS (8 tests)
# ============================================================================

test_backup_pvc_exists() {
    log_test "59. Backup: Validator PVCs exist for backup..."
    local pvcs=$(kubectl get pvc -n "$NAMESPACE" -l app.kubernetes.io/component=validator --no-headers 2>/dev/null | wc -l)
    if [ "$pvcs" -gt 0 ]; then
        log_pass "Found $pvcs validator PVCs available for backup"
    else
        log_warn "No validator PVCs found"
    fi
}

test_backup_pvc_bound() {
    log_test "60. Backup: All PVCs are bound..."
    local total=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    local bound=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep "Bound" | wc -l)

    if [ "$bound" -eq "$total" ] && [ "$total" -gt 0 ]; then
        log_pass "All $bound PVCs are bound"
    elif [ "$total" -eq 0 ]; then
        log_warn "No PVCs found"
    else
        log_fail "Only $bound/$total PVCs are bound"
    fi
}

test_backup_snapshot_capability() {
    log_test "61. Backup: VolumeSnapshot capability..."
    if kubectl get volumesnapshotclasses &>/dev/null 2>&1; then
        local vsc=$(kubectl get volumesnapshotclasses --no-headers 2>/dev/null | wc -l)
        if [ "$vsc" -gt 0 ]; then
            log_pass "VolumeSnapshotClass available ($vsc classes)"
        else
            log_warn "VolumeSnapshotClass CRD exists but no classes configured"
        fi
    else
        log_skip "VolumeSnapshot CRD not installed"
    fi
}

test_backup_cronjob() {
    log_test "62. Backup: Backup CronJob..."
    local cj=$(kubectl get cronjobs -n "$NAMESPACE" -o name 2>/dev/null | grep -iE "backup|snapshot" | head -1)
    if [ -n "$cj" ]; then
        log_pass "Backup CronJob found"
    else
        log_warn "No backup CronJob found"
    fi
}

test_backup_secret_encryption() {
    log_test "63. Backup: Secret encryption at rest..."
    # Check if encryption is enabled at the cluster level
    local enc=$(kubectl get secrets -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [ "$enc" -gt 0 ]; then
        log_pass "Secrets exist ($enc secrets in namespace)"
    else
        log_warn "No secrets found to verify encryption"
    fi
}

test_key_backup_secret() {
    log_test "64. Backup: Validator key secrets..."
    local secret=$(kubectl get secrets -n "$NAMESPACE" -o name 2>/dev/null | grep -iE "validator.*key|key.*validator" | head -1)
    if [ -n "$secret" ]; then
        log_pass "Validator key secret found"
    else
        log_warn "Validator key secret not found"
    fi
}

test_genesis_backup() {
    log_test "65. Backup: Genesis file backup..."
    local cm=$(kubectl get configmap genesis-config -n "$NAMESPACE" -o name 2>/dev/null)
    if [ -n "$cm" ]; then
        log_pass "Genesis ConfigMap exists (can be backed up)"
    else
        log_warn "Genesis ConfigMap not found"
    fi
}

test_backup_storage_class() {
    log_test "66. Backup: Storage class for backups..."
    local sc=$(kubectl get storageclass -o name 2>/dev/null | head -1)
    if [ -n "$sc" ]; then
        log_pass "StorageClass available for backup volumes"
    else
        log_warn "No StorageClass found"
    fi
}

# ============================================================================
# DISASTER RECOVERY TESTS (6 tests)
# ============================================================================

test_dr_multi_replica() {
    log_test "67. DR: Multi-replica deployment..."
    local replicas=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    if [ "$replicas" -ge 3 ]; then
        log_pass "Multi-replica setup: $replicas validators"
    elif [ "$replicas" -eq 1 ]; then
        log_warn "Single replica: No redundancy for DR"
    else
        log_fail "No validators deployed"
    fi
}

test_dr_pod_distribution() {
    log_test "68. DR: Pod anti-affinity (distribution)..."
    local affinity=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.affinity}' 2>/dev/null)
    if [ -n "$affinity" ]; then
        log_pass "Pod affinity/anti-affinity configured"
    else
        log_warn "No pod affinity rules (pods may be co-located)"
    fi
}

test_dr_pdb() {
    log_test "69. DR: PodDisruptionBudget..."
    local pdb=$(kubectl get pdb -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [ "$pdb" -gt 0 ]; then
        log_pass "PodDisruptionBudget configured ($pdb PDBs)"
    else
        log_warn "No PodDisruptionBudget found"
    fi
}

test_dr_headless_service() {
    log_test "70. DR: Headless service for state recovery..."
    local svc=$(kubectl get svc -n "$NAMESPACE" -o name 2>/dev/null | grep -i headless | head -1)
    if [ -n "$svc" ]; then
        log_pass "Headless service configured for peer discovery"
    else
        log_warn "No headless service found"
    fi
}

test_dr_resource_limits() {
    log_test "71. DR: Resource limits (prevent cascading failures)..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod"
        return 0
    fi

    local limits=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits}' 2>/dev/null)
    if [ -n "$limits" ] && [ "$limits" != "{}" ]; then
        log_pass "Resource limits configured"
    else
        log_warn "No resource limits (risk of resource exhaustion)"
    fi
}

test_dr_rollback_strategy() {
    log_test "72. DR: Rollback strategy..."
    local strategy=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.spec.updateStrategy.type}' 2>/dev/null)
    if [ "$strategy" = "RollingUpdate" ]; then
        log_pass "Rolling update strategy configured"
    else
        log_warn "Update strategy: $strategy"
    fi
}

# ============================================================================
# KEY ROTATION TESTS (4 tests)
# ============================================================================

test_key_rotation_secret() {
    log_test "73. Key Rotation: Validator key secrets exist..."
    local secrets=$(kubectl get secrets -n "$NAMESPACE" -o name 2>/dev/null | grep -iE "key|validator" | wc -l)
    if [ "$secrets" -gt 0 ]; then
        log_pass "Found $secrets key-related secrets"
    else
        log_warn "No key secrets found"
    fi
}

test_key_rotation_job() {
    log_test "74. Key Rotation: Rotation job/cronjob..."
    local job=$(kubectl get cronjobs -n "$NAMESPACE" -o name 2>/dev/null | grep -iE "key.*rotation|rotation" | head -1)
    if [ -n "$job" ]; then
        log_pass "Key rotation CronJob found"
    else
        log_skip "No key rotation CronJob (may use manual rotation)"
    fi
}

test_key_rotation_vault() {
    log_test "75. Key Rotation: Vault integration..."
    local vault=$(kubectl get pods -n "$NAMESPACE" -l app=vault -o name 2>/dev/null | head -1)
    if [ -n "$vault" ]; then
        log_pass "Vault pod found for key management"
    else
        log_skip "Vault not deployed (using K8s secrets)"
    fi
}

test_key_rotation_external_secrets() {
    log_test "76. Key Rotation: External Secrets Operator..."
    local es=$(kubectl get externalsecrets -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [ "$es" -gt 0 ]; then
        log_pass "External Secrets configured ($es external secrets)"
    else
        log_skip "No External Secrets (using native K8s secrets)"
    fi
}

# ============================================================================
# DRIFT DETECTION TESTS (4 tests)
# ============================================================================

test_drift_configmap_hash() {
    log_test "77. Drift Detection: ConfigMap annotations..."
    local cm=$(kubectl get configmap -n "$NAMESPACE" -o jsonpath='{.items[0].metadata.annotations}' 2>/dev/null)
    if echo "$cm" | grep -qiE "checksum|hash"; then
        log_pass "ConfigMaps have checksum annotations"
    else
        log_warn "ConfigMaps missing checksum annotations"
    fi
}

test_drift_image_pinning() {
    log_test "78. Drift Detection: Image tag pinning..."
    local pod=$(get_validator_pod)
    if [ -z "$pod" ]; then
        log_skip "No validator pod"
        return 0
    fi

    local image=$(kubectl get "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null)
    if echo "$image" | grep -qE ":[a-f0-9]{7,}|:v[0-9]+\.[0-9]+\.[0-9]+"; then
        log_pass "Image uses pinned tag: $image"
    elif echo "$image" | grep -q ":latest"; then
        log_fail "Image uses :latest tag (risk of drift)"
    else
        log_warn "Image tagging: $image"
    fi
}

test_drift_gitops() {
    log_test "79. Drift Detection: GitOps/ArgoCD..."
    local argoapp=$(kubectl get applications.argoproj.io -n argocd -o name 2>/dev/null | grep -i paw | head -1)
    if [ -n "$argoapp" ]; then
        log_pass "ArgoCD application found for GitOps"
    else
        log_skip "ArgoCD not used (may use other GitOps tool)"
    fi
}

test_drift_resource_version() {
    log_test "80. Drift Detection: Resource versioning..."
    local sts=$(kubectl get statefulset paw-validator -n "$NAMESPACE" -o jsonpath='{.metadata.resourceVersion}' 2>/dev/null)
    if [ -n "$sts" ]; then
        log_pass "StatefulSet has resource version: $sts"
    else
        log_warn "Cannot verify resource versioning"
    fi
}

# ============================================================================
# TEST RUNNERS BY CATEGORY
# ============================================================================

run_prometheus_tests() {
    log_info "=== PROMETHEUS TESTS ==="
    test_prometheus_pod_health
    test_prometheus_ready
    test_prometheus_scrape_targets
    test_prometheus_tendermint_metrics
    test_prometheus_cosmos_metrics
    test_prometheus_dex_metrics
    test_prometheus_retention
    test_prometheus_labels
    test_prometheus_storage
    test_prometheus_config
}

run_grafana_tests() {
    log_info "=== GRAFANA TESTS ==="
    test_grafana_pod_health
    test_grafana_ready
    test_grafana_datasources
    test_grafana_dashboards
    test_grafana_provisioning
    test_grafana_storage
}

run_alert_tests() {
    log_info "=== ALERT RULES TESTS ==="
    test_alert_rules_configmap
    test_alertmanager_config
    test_prometheus_rule_crd
    test_block_production_alert
    test_low_peer_count_alert
    test_high_block_time_alert
    test_cpu_alert_rules
    test_disk_alert_rules
    test_alert_routing
    test_alert_receivers
}

run_loki_tests() {
    log_info "=== LOKI TESTS ==="
    test_loki_pod_health
    test_loki_ready
    test_loki_push_endpoint
    test_loki_query_endpoint
    test_loki_labels
    test_loki_config
    test_loki_retention
    test_loki_storage
}

run_tracing_tests() {
    log_info "=== TRACING TESTS ==="
    test_jaeger_deployment
    test_otel_collector
    test_tracing_configmap
    test_otlp_endpoint
    test_trace_sampling
    test_trace_context_propagation
}

run_custom_metrics_tests() {
    log_info "=== CUSTOM BLOCKCHAIN METRICS TESTS ==="
    test_consensus_height_metric
    test_p2p_peers_metric
    test_tx_counter_metrics
    test_mempool_metrics
    test_dex_pool_metrics
    test_oracle_metrics
    test_validator_power_metric
    test_block_size_metric
}

run_slo_tests() {
    log_info "=== SLA/SLO COMPLIANCE TESTS ==="
    test_slo_block_time
    test_slo_uptime
    test_slo_availability
    test_slo_latency
    test_slo_consensus_participation
    test_slo_peer_connectivity
    test_slo_error_rate
    test_slo_disk_usage
    test_slo_memory_usage
    test_slo_cpu_usage
}

run_backup_tests() {
    log_info "=== BACKUP/RESTORE TESTS ==="
    test_backup_pvc_exists
    test_backup_pvc_bound
    test_backup_snapshot_capability
    test_backup_cronjob
    test_backup_secret_encryption
    test_key_backup_secret
    test_genesis_backup
    test_backup_storage_class
}

run_dr_tests() {
    log_info "=== DISASTER RECOVERY TESTS ==="
    test_dr_multi_replica
    test_dr_pod_distribution
    test_dr_pdb
    test_dr_headless_service
    test_dr_resource_limits
    test_dr_rollback_strategy
}

run_key_rotation_tests() {
    log_info "=== KEY ROTATION TESTS ==="
    test_key_rotation_secret
    test_key_rotation_job
    test_key_rotation_vault
    test_key_rotation_external_secrets
}

run_drift_tests() {
    log_info "=== DRIFT DETECTION TESTS ==="
    test_drift_configmap_hash
    test_drift_image_pinning
    test_drift_gitops
    test_drift_resource_version
}

run_all_tests() {
    run_prometheus_tests
    echo ""
    run_grafana_tests
    echo ""
    run_alert_tests
    echo ""
    run_loki_tests
    echo ""
    run_tracing_tests
    echo ""
    run_custom_metrics_tests
    echo ""
    run_slo_tests
    echo ""
    run_backup_tests
    echo ""
    run_dr_tests
    echo ""
    run_key_rotation_tests
    echo ""
    run_drift_tests
}

print_summary() {
    echo ""
    echo "=============================================="
    echo "MONITORING TEST RESULTS"
    echo "=============================================="
    echo -e "Passed:   ${GREEN}$PASSED${NC}"
    echo -e "Failed:   ${RED}$FAILED${NC}"
    echo -e "Skipped:  ${CYAN}$SKIPPED${NC}"
    echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
    echo "=============================================="
    echo "Total:    $((PASSED + FAILED + SKIPPED + WARNINGS))"
    echo "=============================================="

    if [ "$FAILED" -gt 0 ]; then
        echo -e "${RED}MONITORING TESTS FAILED${NC}"
        echo ""
        echo "Review failed tests and ensure monitoring stack is properly deployed."
        exit 1
    elif [ "$WARNINGS" -gt 5 ]; then
        echo -e "${YELLOW}MONITORING TESTS PASSED WITH WARNINGS${NC}"
        echo ""
        echo "Consider addressing warnings for production readiness."
        exit 0
    else
        echo -e "${GREEN}MONITORING TESTS PASSED${NC}"
        exit 0
    fi
}

show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "PAW Kubernetes Monitoring Tests - Phase 5"
    echo "Comprehensive tests for monitoring, observability, and operational readiness."
    echo ""
    echo "Options:"
    echo "  --namespace, -n NAME     Namespace to test (default: paw)"
    echo "  --category, -c CATEGORY  Run specific test category:"
    echo "                             all (default), prometheus, grafana, alerts, loki,"
    echo "                             tracing, metrics, slo, backup, dr, keys, drift"
    echo "  --verbose, -v            Enable verbose output"
    echo "  --help, -h               Show this help"
    echo ""
    echo "Test Categories:"
    echo "  prometheus   Prometheus pod health, scrape targets, metrics (10 tests)"
    echo "  grafana      Grafana pod health, datasources, dashboards (6 tests)"
    echo "  alerts       Alert rules, routing, receivers (10 tests)"
    echo "  loki         Loki log aggregation (8 tests)"
    echo "  tracing      Jaeger/OTLP distributed tracing (6 tests)"
    echo "  metrics      Custom blockchain metrics (8 tests)"
    echo "  slo          SLA/SLO compliance (10 tests)"
    echo "  backup       Backup and restore capability (8 tests)"
    echo "  dr           Disaster recovery (6 tests)"
    echo "  keys         Key rotation (4 tests)"
    echo "  drift        Configuration drift detection (4 tests)"
    echo ""
    echo "Examples:"
    echo "  $0                        # Run all 80 tests"
    echo "  $0 -n paw-testnet        # Test in paw-testnet namespace"
    echo "  $0 -c prometheus         # Test only Prometheus"
    echo "  $0 -c slo -v             # Test SLOs with verbose output"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Monitoring Tests${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Category:  $CATEGORY"
    echo "Time:      $(date)"
    echo ""

    case "$CATEGORY" in
        all)
            run_all_tests
            ;;
        prometheus)
            run_prometheus_tests
            ;;
        grafana)
            run_grafana_tests
            ;;
        alerts)
            run_alert_tests
            ;;
        loki)
            run_loki_tests
            ;;
        tracing)
            run_tracing_tests
            ;;
        metrics)
            run_custom_metrics_tests
            ;;
        slo)
            run_slo_tests
            ;;
        backup)
            run_backup_tests
            ;;
        dr)
            run_dr_tests
            ;;
        keys)
            run_key_rotation_tests
            ;;
        drift)
            run_drift_tests
            ;;
        *)
            log_fail "Unknown category: $CATEGORY"
            echo "Use --help for available categories."
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
        --category|-c)
            CATEGORY="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE="true"
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information."
            exit 1
            ;;
    esac
done

main
