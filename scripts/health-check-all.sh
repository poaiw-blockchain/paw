#!/bin/bash
# Health Check Aggregator for PAW Testnet Services
# Run this script ON the paw-testnet server (54.39.103.49)
# Queries all service health endpoints and aggregates results
#
# MVP Testnet Port Configuration (4 validators):
#   val1: RPC 11657, gRPC 11090, REST 11317, P2P 11656
#   val2: RPC 11757, gRPC 11190, REST 11417, P2P 11756
#   val3 (services-testnet): RPC 11857, gRPC 11290, REST 11517, P2P 11856
#   val4 (services-testnet): RPC 11957, gRPC 11390, REST 11617, P2P 11956

set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Services and their health endpoints (localhost = running on paw-testnet server)
declare -A SERVICES=(
    ["val1-rpc"]="http://127.0.0.1:11657/health"
    ["val2-rpc"]="http://127.0.0.1:11757/health"
    ["explorer-api"]="http://127.0.0.1:11080/health"
    ["faucet-api"]="http://127.0.0.1:11084/health"
)

# Public endpoints to verify
declare -A PUBLIC_ENDPOINTS=(
    ["testnet-rpc.poaiw.org"]="https://testnet-rpc.poaiw.org/health"
    ["testnet-api.poaiw.org"]="https://testnet-api.poaiw.org/cosmos/base/tendermint/v1beta1/syncing"
    ["testnet-explorer.poaiw.org"]="https://testnet-explorer.poaiw.org"
    ["testnet-faucet.poaiw.org"]="https://testnet-faucet.poaiw.org"
)

# Track overall health
OVERALL_HEALTHY=true
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S UTC')

echo "======================================================================"
echo "PAW Testnet Health Check Report - $TIMESTAMP"
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
            echo -e "${GREEN}✓${NC} $name: healthy"
            return 0
            ;;
        503)
            echo -e "${YELLOW}⚠${NC} $name: not ready"
            OVERALL_HEALTHY=false
            return 1
            ;;
        000)
            echo -e "${RED}✗${NC} $name: unreachable"
            OVERALL_HEALTHY=false
            return 1
            ;;
        *)
            echo -e "${RED}✗${NC} $name: unhealthy (HTTP $response)"
            OVERALL_HEALTHY=false
            return 1
            ;;
    esac
}

# Function to check validator RPC and get height
check_validator() {
    local name=$1
    local rpc_port=$2
    local timeout=${3:-5}

    local status
    status=$(curl -s --max-time "$timeout" "http://127.0.0.1:$rpc_port/status" 2>/dev/null || echo "")

    if [[ -z "$status" ]]; then
        echo -e "${RED}✗${NC} $name (port $rpc_port): unreachable"
        OVERALL_HEALTHY=false
        return 1
    fi

    local height catching_up
    height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // "?"')
    catching_up=$(echo "$status" | jq -r '.result.sync_info.catching_up // "?"')

    if [[ "$catching_up" == "false" ]]; then
        echo -e "${GREEN}✓${NC} $name (port $rpc_port): height=$height, synced"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} $name (port $rpc_port): height=$height, catching_up"
        return 1
    fi
}

# Check local validators (on paw-testnet server)
echo "Local Validators (paw-testnet server):"
echo "----------------------------------------------------------------------"
check_validator "val1" 11657 || true
check_validator "val2" 11757 || true

echo ""
echo "Remote Validators (services-testnet via VPN 10.10.0.4):"
echo "----------------------------------------------------------------------"
# These use VPN address since this script runs on paw-testnet
val3_status=$(curl -s --max-time 5 "http://10.10.0.4:11857/status" 2>/dev/null || echo "")
if [[ -n "$val3_status" ]]; then
    val3_height=$(echo "$val3_status" | jq -r '.result.sync_info.latest_block_height // "?"')
    val3_catching=$(echo "$val3_status" | jq -r '.result.sync_info.catching_up // "?"')
    if [[ "$val3_catching" == "false" ]]; then
        echo -e "${GREEN}✓${NC} val3 (10.10.0.4:11857): height=$val3_height, synced"
    else
        echo -e "${YELLOW}⚠${NC} val3 (10.10.0.4:11857): height=$val3_height, catching_up"
    fi
else
    echo -e "${RED}✗${NC} val3 (10.10.0.4:11857): unreachable"
fi

val4_status=$(curl -s --max-time 5 "http://10.10.0.4:11957/status" 2>/dev/null || echo "")
if [[ -n "$val4_status" ]]; then
    val4_height=$(echo "$val4_status" | jq -r '.result.sync_info.latest_block_height // "?"')
    val4_catching=$(echo "$val4_status" | jq -r '.result.sync_info.catching_up // "?"')
    if [[ "$val4_catching" == "false" ]]; then
        echo -e "${GREEN}✓${NC} val4 (10.10.0.4:11957): height=$val4_height, synced"
    else
        echo -e "${YELLOW}⚠${NC} val4 (10.10.0.4:11957): height=$val4_height, catching_up"
    fi
else
    echo -e "${RED}✗${NC} val4 (10.10.0.4:11957): unreachable"
fi

echo ""
echo "======================================================================"

# Check local services
echo ""
echo "Local Services:"
echo "----------------------------------------------------------------------"
check_service "explorer-api" "http://127.0.0.1:11080/health" || true
check_service "faucet-api" "http://127.0.0.1:11084/health" || true

echo ""
echo "======================================================================"

# Check public endpoints
echo ""
echo "Public Endpoints (via nginx):"
echo "----------------------------------------------------------------------"
for endpoint in "${!PUBLIC_ENDPOINTS[@]}"; do
    check_service "$endpoint" "${PUBLIC_ENDPOINTS[$endpoint]}" || true
done

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

# Check if pawd processes are running
pawd_count=$(pgrep -c "pawd" 2>/dev/null || echo "0")
if [ "$pawd_count" -ge 2 ]; then
    echo -e "${GREEN}✓${NC} pawd processes running: $pawd_count"
elif [ "$pawd_count" -eq 1 ]; then
    echo -e "${YELLOW}⚠${NC} Only 1 pawd process running (expected 2)"
else
    echo -e "${RED}✗${NC} No pawd processes running"
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
