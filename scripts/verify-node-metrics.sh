#!/bin/bash
# Verify PAW node metrics configuration and collection

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_section() {
    echo -e "\n${CYAN}=== $1 ===${NC}\n"
}

# Track statistics
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARN_CHECKS=0

check() {
    local description="$1"
    local command="$2"
    local expected="${3:-0}"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected" = "0" ]; then
            log_success "$description"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            log_warn "$description (unexpected success)"
            WARN_CHECKS=$((WARN_CHECKS + 1))
            return 1
        fi
    else
        if [ "$expected" = "1" ]; then
            log_error "$description (expected to fail)"
            WARN_CHECKS=$((WARN_CHECKS + 1))
            return 1
        else
            log_error "$description"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    fi
}

check_http_endpoint() {
    local url="$1"
    local description="$2"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if curl -sf "$url" > /dev/null 2>&1; then
        log_success "$description - UP"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        log_error "$description - DOWN"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

check_metrics_present() {
    local url="$1"
    local pattern="$2"
    local description="$3"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if curl -sf "$url" 2>/dev/null | grep -q "$pattern"; then
        log_success "$description"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        log_warn "$description - Not found (may not be active yet)"
        WARN_CHECKS=$((WARN_CHECKS + 1))
        return 1
    fi
}

# Main verification
main() {
    log_section "PAW Node Metrics Verification"

    # Check if Prometheus is running
    log_section "1. Prometheus Service Status"
    check "Prometheus container running" "docker ps --filter name=paw-prometheus --format '{{.Names}}' | grep -q paw-prometheus"
    check "Prometheus health check" "curl -sf http://localhost:9091/-/healthy > /dev/null"

    # Check node configurations
    log_section "2. Node Configuration Checks"

    for node in paw-node1 paw-node2 paw-node3 paw-node4; do
        if docker ps | grep -q "$node"; then
            check "$node - CometBFT metrics enabled" \
                "docker exec $node grep -q 'prometheus = true' /root/.paw/node*/config/config.toml"
            check "$node - App telemetry enabled" \
                "docker exec $node grep -q 'enabled = true' /root/.paw/node*/config/app.toml"
        else
            log_warn "$node not running - skipping"
        fi
    done

    # Check metrics endpoints
    log_section "3. Metrics Endpoint Accessibility"

    # CometBFT metrics (internal to containers)
    for i in 1 2 3 4; do
        node="paw-node$i"
        if docker ps | grep -q "$node"; then
            check_http_endpoint \
                "http://$node:26660/metrics" \
                "Node $i - CometBFT metrics (:26660)" || \
                docker exec "$node" wget -q -O- http://localhost:26660/metrics > /dev/null 2>&1 && \
                log_success "Node $i - CometBFT metrics (:26660) - UP (internal check)"
        fi
    done

    # Check Prometheus targets
    log_section "4. Prometheus Target Status"

    if curl -sf http://localhost:9091/api/v1/targets > /dev/null 2>&1; then
        local targets_json=$(curl -s http://localhost:9091/api/v1/targets)

        # Count total targets
        local total_targets=$(echo "$targets_json" | python3 -c "import json,sys; data=json.load(sys.stdin); print(len(data['data']['activeTargets']))" 2>/dev/null || echo "0")
        log_info "Total Prometheus targets configured: $total_targets"

        # Check each blockchain target
        local blockchain_targets=(
            "paw-tendermint"
            "paw-app"
            "paw-api"
            "paw-dex"
            "paw-validators"
            "paw-oracle"
            "paw-compute"
            "paw-ibc"
            "paw-staking"
            "paw-governance"
            "paw-bank"
            "paw-distribution"
            "paw-slashing"
            "paw-evidence"
            "paw-upgrade"
        )

        for target in "${blockchain_targets[@]}"; do
            local count=$(echo "$targets_json" | python3 -c "import json,sys; data=json.load(sys.stdin); print(sum(1 for t in data['data']['activeTargets'] if t['scrapePool']=='$target'))" 2>/dev/null || echo "0")

            if [ "$count" -gt 0 ]; then
                log_success "Target '$target' - $count endpoint(s) configured"
                PASSED_CHECKS=$((PASSED_CHECKS + 1))
            else
                log_error "Target '$target' - No endpoints found"
                FAILED_CHECKS=$((FAILED_CHECKS + 1))
            fi
            TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
        done

        # Check health status of targets
        echo ""
        log_info "Target Health Summary:"
        echo "$targets_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
health_counts = {}
for target in data['data']['activeTargets']:
    health = target.get('health', 'unknown')
    health_counts[health] = health_counts.get(health, 0) + 1

for health, count in sorted(health_counts.items()):
    if health == 'up':
        print(f'  ✓ UP: {count}')
    elif health == 'down':
        print(f'  ✗ DOWN: {count}')
    else:
        print(f'  ? {health.upper()}: {count}')
" || log_warn "Could not parse target health status"

    else
        log_error "Cannot reach Prometheus API at http://localhost:9091"
    fi

    # Check for key metrics
    log_section "5. Key Metrics Validation"

    # CometBFT metrics
    check_metrics_present \
        "http://localhost:9091/api/v1/query?query=cometbft_consensus_height" \
        "cometbft_consensus_height" \
        "CometBFT consensus height metric"

    check_metrics_present \
        "http://localhost:9091/api/v1/query?query=cometbft_consensus_validators" \
        "cometbft_consensus_validators" \
        "CometBFT validators metric"

    check_metrics_present \
        "http://localhost:9091/api/v1/query?query=cometbft_p2p_peers" \
        "cometbft_p2p_peers" \
        "CometBFT P2P peers metric"

    # Check for module-specific metrics (may not exist if modules inactive)
    log_info "Checking for module-specific metrics (optional)..."

    for module in dex oracle compute; do
        if curl -sf "http://localhost:9091/api/v1/query?query=paw_${module}_" 2>/dev/null | grep -q "paw_${module}_"; then
            log_success "Module '$module' metrics found"
        else
            log_warn "Module '$module' metrics not found (may be inactive)"
        fi
    done

    # Check infrastructure metrics
    log_section "6. Infrastructure Metrics"

    check_http_endpoint "http://localhost:9100/metrics" "Node Exporter"
    check_http_endpoint "http://localhost:11082/metrics" "cAdvisor"

    # Final summary
    log_section "Verification Summary"

    echo "Total checks:  $TOTAL_CHECKS"
    echo -e "${GREEN}Passed:        $PASSED_CHECKS${NC}"
    echo -e "${YELLOW}Warnings:      $WARN_CHECKS${NC}"
    echo -e "${RED}Failed:        $FAILED_CHECKS${NC}"

    local success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    echo ""
    echo "Success Rate:  ${success_rate}%"

    if [ $FAILED_CHECKS -eq 0 ]; then
        echo -e "\n${GREEN}✓ All critical checks passed!${NC}"
        echo ""
        log_info "Prometheus UI:     http://localhost:9091"
        log_info "Grafana UI:        http://localhost:11030 (admin/grafana_secure_password)"
        log_info "Alertmanager UI:   http://localhost:9093"
        echo ""
        log_info "To view all targets: http://localhost:9091/targets"
        log_info "To query metrics:    http://localhost:9091/graph"
        return 0
    else
        echo -e "\n${RED}✗ Some checks failed. Please review the errors above.${NC}"
        echo ""
        log_info "Common issues:"
        echo "  1. Nodes may need time to start up and expose metrics"
        echo "  2. Some metrics only appear after network activity"
        echo "  3. Check docker logs: docker logs paw-node1"
        echo "  4. Verify Prometheus config: docker exec paw-prometheus cat /etc/prometheus/prometheus.yml"
        return 1
    fi
}

# Run main function
main "$@"
