#!/bin/bash
# Health Check Aggregator for PAW Services
# Queries all service health endpoints and aggregates results

set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Services and their health endpoints
declare -A SERVICES=(
    ["pawd"]="http://localhost:36661/health"
    ["pawd-detailed"]="http://localhost:36661/health/detailed"
    ["explorer"]="http://localhost:11080/health"
    ["explorer-ready"]="http://localhost:11080/health/ready"
    ["prometheus"]="http://localhost:9091/-/healthy"
    ["grafana"]="http://localhost:11030/api/health"
    ["alertmanager"]="http://localhost:9093/-/healthy"
)

# Track overall health
OVERALL_HEALTHY=true
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo "======================================================================"
echo "PAW Health Check Report - $TIMESTAMP"
echo "======================================================================"
echo ""

# Function to check a single service
check_service() {
    local name=$1
    local url=$2
    local timeout=${3:-5}

    # Try to fetch health endpoint
    response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout "$timeout" "$url" 2>/dev/null || echo "000")

    case "$response" in
        200)
            echo -e "${GREEN}✓${NC} $name: healthy ($url)"
            return 0
            ;;
        503)
            echo -e "${YELLOW}⚠${NC} $name: not ready ($url)"
            OVERALL_HEALTHY=false
            return 1
            ;;
        000)
            echo -e "${RED}✗${NC} $name: unreachable ($url)"
            OVERALL_HEALTHY=false
            return 1
            ;;
        *)
            echo -e "${RED}✗${NC} $name: unhealthy (HTTP $response) ($url)"
            OVERALL_HEALTHY=false
            return 1
            ;;
    esac
}

# Function to get detailed health info
get_detailed_health() {
    local url=$1
    local response

    response=$(curl -s --connect-timeout 5 "$url" 2>/dev/null || echo "{}")

    if [ -n "$response" ] && [ "$response" != "{}" ]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    fi
}

# Check all services
echo "Service Health Status:"
echo "----------------------------------------------------------------------"

for service in "${!SERVICES[@]}"; do
    check_service "$service" "${SERVICES[$service]}"
done

echo ""
echo "======================================================================"

# Get detailed health for main daemon
echo ""
echo "Detailed Health Information (pawd):"
echo "----------------------------------------------------------------------"
detailed=$(get_detailed_health "http://localhost:36661/health/detailed")
if [ -n "$detailed" ]; then
    echo "$detailed"
else
    echo "No detailed health information available"
fi

echo ""
echo "======================================================================"

# Check for specific issues
echo ""
echo "System Checks:"
echo "----------------------------------------------------------------------"

# Check disk space
disk_usage=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$disk_usage" -gt 90 ]; then
    echo -e "${RED}✗${NC} Disk usage critical: ${disk_usage}%"
    OVERALL_HEALTHY=false
elif [ "$disk_usage" -gt 80 ]; then
    echo -e "${YELLOW}⚠${NC} Disk usage high: ${disk_usage}%"
else
    echo -e "${GREEN}✓${NC} Disk usage normal: ${disk_usage}%"
fi

# Check memory usage
mem_usage=$(free | awk '/^Mem:/ {printf "%.0f", $3/$2 * 100}')
if [ "$mem_usage" -gt 90 ]; then
    echo -e "${RED}✗${NC} Memory usage critical: ${mem_usage}%"
    OVERALL_HEALTHY=false
elif [ "$mem_usage" -gt 80 ]; then
    echo -e "${YELLOW}⚠${NC} Memory usage high: ${mem_usage}%"
else
    echo -e "${GREEN}✓${NC} Memory usage normal: ${mem_usage}%"
fi

# Check if pawd process is running
if pgrep -x "pawd" > /dev/null; then
    echo -e "${GREEN}✓${NC} pawd process is running"
else
    echo -e "${RED}✗${NC} pawd process is not running"
    OVERALL_HEALTHY=false
fi

echo ""
echo "======================================================================"
echo ""

# Print overall status
if [ "$OVERALL_HEALTHY" = true ]; then
    echo -e "${GREEN}Overall Status: HEALTHY${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}Overall Status: UNHEALTHY${NC}"
    echo ""
    echo "Please review the failures above and take corrective action."
    exit 1
fi
