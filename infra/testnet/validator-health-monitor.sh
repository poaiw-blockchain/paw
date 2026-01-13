#!/bin/bash
#
# PAW Testnet Validator Health Monitor
# Mainnet-ready monitoring script for 4-validator deployment
#
# Usage: ./validator-health-monitor.sh [--continuous] [--alert-webhook URL]
#
# Run continuously with: ./validator-health-monitor.sh --continuous
# Results saved to: /var/log/paw-health.log (or stdout)

set -euo pipefail

# Configuration
VALIDATORS=(
    "val1:paw-testnet:11657"
    "val2:paw-testnet:11757"
    "val3:services-testnet:11857"
    "val4:services-testnet:11957"
)

BLOCK_TIME_TARGET=5
BLOCK_TIME_WARNING=8
BLOCK_TIME_CRITICAL=15
HEIGHT_DIFF_WARNING=2
HEIGHT_DIFF_CRITICAL=5
PEER_COUNT_WARNING=2
PEER_COUNT_CRITICAL=1

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Parse arguments
CONTINUOUS=false
ALERT_WEBHOOK=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --continuous) CONTINUOUS=true; shift ;;
        --alert-webhook) ALERT_WEBHOOK="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

alert() {
    local level=$1
    local message=$2
    log "[$level] $message"

    if [[ -n "$ALERT_WEBHOOK" ]]; then
        curl -s -X POST "$ALERT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"level\":\"$level\",\"message\":\"$message\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" \
            >/dev/null 2>&1 || true
    fi
}

get_status() {
    local server=$1
    local port=$2
    ssh -o ConnectTimeout=5 "$server" "curl -s http://127.0.0.1:$port/status 2>/dev/null" || echo "{}"
}

get_net_info() {
    local server=$1
    local port=$2
    ssh -o ConnectTimeout=5 "$server" "curl -s http://127.0.0.1:$port/net_info 2>/dev/null" || echo "{}"
}

check_validator() {
    local name=$1
    local server=$2
    local port=$3

    local status=$(get_status "$server" "$port")

    if [[ -z "$status" || "$status" == "{}" ]]; then
        alert "CRITICAL" "$name: Unable to connect to RPC"
        return 1
    fi

    local height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // "0"')
    local catching_up=$(echo "$status" | jq -r '.result.sync_info.catching_up // "true"')
    local block_time=$(echo "$status" | jq -r '.result.sync_info.latest_block_time // ""')
    local voting_power=$(echo "$status" | jq -r '.result.validator_info.voting_power // "0"')

    # Check if catching up
    if [[ "$catching_up" == "true" ]]; then
        alert "WARNING" "$name: Node is catching up (height: $height)"
    fi

    # Check voting power
    if [[ "$voting_power" == "0" ]]; then
        alert "WARNING" "$name: Zero voting power - validator may be jailed"
    fi

    # Get peer count
    local net_info=$(get_net_info "$server" "$port")
    local peer_count=$(echo "$net_info" | jq -r '.result.n_peers // "0"')

    if (( peer_count <= PEER_COUNT_CRITICAL )); then
        alert "CRITICAL" "$name: Low peer count ($peer_count)"
    elif (( peer_count <= PEER_COUNT_WARNING )); then
        alert "WARNING" "$name: Low peer count ($peer_count)"
    fi

    echo "$name:$height:$peer_count:$catching_up:$voting_power"
}

check_consensus() {
    local heights=()
    local results=()

    for validator in "${VALIDATORS[@]}"; do
        IFS=':' read -r name server port <<< "$validator"
        local result=$(check_validator "$name" "$server" "$port")
        results+=("$result")

        local height=$(echo "$result" | cut -d: -f2)
        heights+=("$height")
    done

    # Check height consistency (use base 10 to avoid octal interpretation)
    local max_height=$(printf '%s\n' "${heights[@]}" | sort -rn | head -1)
    local min_height=$(printf '%s\n' "${heights[@]}" | sort -n | head -1)
    local height_diff=$((10#$max_height - 10#$min_height))

    if (( height_diff >= HEIGHT_DIFF_CRITICAL )); then
        alert "CRITICAL" "Height divergence: $height_diff blocks (max: $max_height, min: $min_height)"
    elif (( height_diff >= HEIGHT_DIFF_WARNING )); then
        alert "WARNING" "Height divergence: $height_diff blocks"
    fi

    # Print summary
    echo ""
    echo "=== PAW Testnet Health Report ==="
    echo "Time: $(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo ""
    printf "%-8s %-10s %-8s %-12s %-12s\n" "Node" "Height" "Peers" "Synced" "VotingPower"
    echo "------------------------------------------------------"
    for result in "${results[@]}"; do
        IFS=':' read -r name height peers catching_up voting <<< "$result"
        local synced="Yes"
        [[ "$catching_up" == "true" ]] && synced="No"
        printf "%-8s %-10s %-8s %-12s %-12s\n" "$name" "$height" "$peers" "$synced" "$voting"
    done
    echo ""
    echo "Height Diff: $height_diff blocks"
    echo "Max Height: $max_height"

    # Overall status
    if (( height_diff < HEIGHT_DIFF_WARNING )); then
        echo -e "Status: ${GREEN}HEALTHY${NC}"
    elif (( height_diff < HEIGHT_DIFF_CRITICAL )); then
        echo -e "Status: ${YELLOW}WARNING${NC}"
    else
        echo -e "Status: ${RED}CRITICAL${NC}"
    fi
    echo ""
}

# Main
if [[ "$CONTINUOUS" == "true" ]]; then
    log "Starting continuous monitoring (Ctrl+C to stop)"
    while true; do
        check_consensus
        sleep 60
    done
else
    check_consensus
fi
