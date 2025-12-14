#!/bin/bash
# ============================================================================
# PAW Blockchain - Monitoring Infrastructure Verification Script
# ============================================================================
# Verifies all monitoring services are running and accessible
# Returns exit code 0 if all services are healthy, non-zero otherwise

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track overall status
OVERALL_STATUS=0

echo "================================================================"
echo "PAW Blockchain - Monitoring Infrastructure Verification"
echo "================================================================"
echo ""

# Function to check service health
check_service() {
    local service_name=$1
    local url=$2
    local expected_pattern=$3

    echo -n "Checking $service_name... "

    if response=$(curl -s -f "$url" 2>&1); then
        if [[ -n "$expected_pattern" ]]; then
            if echo "$response" | grep -q "$expected_pattern"; then
                echo -e "${GREEN}HEALTHY${NC}"
                return 0
            else
                echo -e "${YELLOW}RESPONDING (unexpected output)${NC}"
                return 1
            fi
        else
            echo -e "${GREEN}HEALTHY${NC}"
            return 0
        fi
    else
        echo -e "${RED}DOWN${NC}"
        return 1
    fi
}

# Function to check container status
check_container() {
    local container_name=$1
    echo -n "Checking container $container_name... "

    if docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
        health=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "no-healthcheck")
        if [[ "$health" == "healthy" ]] || [[ "$health" == "no-healthcheck" ]]; then
            echo -e "${GREEN}RUNNING${NC} (Health: $health)"
            return 0
        else
            echo -e "${YELLOW}RUNNING${NC} (Health: $health)"
            return 1
        fi
    else
        echo -e "${RED}NOT RUNNING${NC}"
        return 1
    fi
}

echo "=== Container Status ==="
check_container "paw-prometheus" || OVERALL_STATUS=1
check_container "paw-grafana" || OVERALL_STATUS=1
check_container "paw-grafana-db" || OVERALL_STATUS=1
check_container "paw-alertmanager" || OVERALL_STATUS=1
check_container "paw-node-exporter" || OVERALL_STATUS=1
check_container "paw-cadvisor" || OVERALL_STATUS=1
echo ""

echo "=== Service Health Checks ==="
check_service "Prometheus" "http://localhost:9091/-/healthy" "Healthy" || OVERALL_STATUS=1
check_service "Grafana" "http://localhost:11030/api/health" "ok" || OVERALL_STATUS=1
check_service "Alertmanager" "http://localhost:9093/-/healthy" "OK" || OVERALL_STATUS=1
check_service "Node Exporter" "http://localhost:9100/metrics" "go_gc_duration_seconds" || OVERALL_STATUS=1
check_service "cAdvisor" "http://localhost:11082/metrics" "cadvisor_version_info" || OVERALL_STATUS=1
echo ""

echo "=== Prometheus Targets ==="
echo "Checking scrape target health..."
if targets=$(curl -s http://localhost:9091/api/v1/targets 2>/dev/null); then
    up_count=$(echo "$targets" | jq -r '.data.activeTargets[] | select(.health=="up") | .labels.job' | wc -l)
    down_count=$(echo "$targets" | jq -r '.data.activeTargets[] | select(.health=="down") | .labels.job' | wc -l)
    unknown_count=$(echo "$targets" | jq -r '.data.activeTargets[] | select(.health=="unknown") | .labels.job' | wc -l)
    total_count=$((up_count + down_count + unknown_count))

    echo "  Total targets: $total_count"
    echo -e "  ${GREEN}UP: $up_count${NC}"
    echo -e "  ${RED}DOWN: $down_count${NC}"
    echo -e "  ${YELLOW}UNKNOWN: $unknown_count${NC}"

    echo ""
    echo "Target details:"
    echo "$targets" | jq -r '.data.activeTargets[] | "  - \(.labels.job): \(.labels.instance) - \(.health)"'

    if [[ $down_count -gt 0 ]]; then
        OVERALL_STATUS=1
    fi
else
    echo -e "${RED}Failed to fetch Prometheus targets${NC}"
    OVERALL_STATUS=1
fi
echo ""

echo "=== Grafana Dashboards ==="
echo "Checking if dashboards are provisioned..."
if dashboards=$(curl -s -u admin:grafana_secure_password http://localhost:11030/api/search?type=dash-db 2>/dev/null); then
    dashboard_count=$(echo "$dashboards" | jq '. | length')
    echo "  Provisioned dashboards: $dashboard_count"
    if [[ $dashboard_count -gt 0 ]]; then
        echo "$dashboards" | jq -r '.[] | "  - \(.title) (\(.type))"'
    else
        echo -e "  ${YELLOW}No dashboards found (may still be loading)${NC}"
    fi
else
    echo -e "${RED}Failed to fetch Grafana dashboards${NC}"
    OVERALL_STATUS=1
fi
echo ""

echo "=== Alert Rules ==="
echo "Checking Prometheus alert rules..."
if rules=$(curl -s http://localhost:9091/api/v1/rules 2>/dev/null); then
    groups_count=$(echo "$rules" | jq '.data.groups | length')
    rules_count=$(echo "$rules" | jq '[.data.groups[].rules[]] | length')

    echo "  Alert rule groups: $groups_count"
    echo "  Total alert rules: $rules_count"

    if [[ $rules_count -gt 0 ]]; then
        echo ""
        echo "  Rules by group:"
        echo "$rules" | jq -r '.data.groups[] | "  - \(.name): \(.rules | length) rules"'

        # Check for firing alerts
        firing=$(echo "$rules" | jq '[.data.groups[].rules[] | select(.state=="firing")] | length')
        if [[ $firing -gt 0 ]]; then
            echo ""
            echo -e "${YELLOW}  WARNING: $firing alert(s) currently firing${NC}"
            echo "$rules" | jq -r '.data.groups[].rules[] | select(.state=="firing") | "    - \(.name)"'
        fi
    else
        echo -e "  ${YELLOW}No alert rules loaded${NC}"
        OVERALL_STATUS=1
    fi
else
    echo -e "${RED}Failed to fetch alert rules${NC}"
    OVERALL_STATUS=1
fi
echo ""

echo "=== Access URLs ==="
echo "  Prometheus:   http://localhost:9091"
echo "  Grafana:      http://localhost:11030 (admin/grafana_secure_password)"
echo "  Alertmanager: http://localhost:9093"
echo "  Node Exporter: http://localhost:9100"
echo "  cAdvisor:     http://localhost:11082"
echo ""

echo "================================================================"
if [[ $OVERALL_STATUS -eq 0 ]]; then
    echo -e "${GREEN}All monitoring services are healthy!${NC}"
else
    echo -e "${YELLOW}Some services have issues. See details above.${NC}"
fi
echo "================================================================"

exit $OVERALL_STATUS
