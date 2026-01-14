#!/bin/bash
# PAW MVP-1 Comprehensive Health Check (runs from bcpc)
# Checks all 6 nodes across both servers

set -euo pipefail

CHAIN_ID="paw-mvp-1"
LOG_DIR="$HOME/blockchain-projects/logs/paw-mvp"
LOG_FILE="$LOG_DIR/health-$(date +%Y%m%d_%H%M%S).log"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

mkdir -p "$LOG_DIR"

log() {
    echo -e "$*" | tee -a "$LOG_FILE"
}

# Node definitions: name:server:rpc_port:type
declare -a ALL_NODES=(
    "val1:paw-testnet:11657:validator"
    "val2:paw-testnet:11757:validator"
    "val3:services-testnet:11857:validator"
    "val4:services-testnet:11957:validator"
    "sentry1:paw-testnet:12057:sentry"
    "sentry2:services-testnet:12157:sentry"
)

log "=== PAW MVP-1 Health Check (All 6 Nodes) ==="
log "Timestamp: $(date)"
log ""

# Collect data
declare -A heights
declare -A peers
declare -A catching_up
max_height=0

for node_info in "${ALL_NODES[@]}"; do
    IFS=':' read -r name server port type <<< "$node_info"

    # Get status via SSH
    status=$(ssh -o ConnectTimeout=5 "$server" "curl -s --max-time 5 http://127.0.0.1:$port/status" 2>/dev/null || echo "{}")
    net_info=$(ssh -o ConnectTimeout=5 "$server" "curl -s --max-time 5 http://127.0.0.1:$port/net_info" 2>/dev/null || echo "{}")

    height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // "0"')
    syncing=$(echo "$status" | jq -r '.result.sync_info.catching_up // "true"')
    peer_count=$(echo "$net_info" | jq -r '.result.n_peers // "0"')

    heights[$name]=$height
    peers[$name]=$peer_count
    catching_up[$name]=$syncing

    if (( height > max_height )); then
        max_height=$height
    fi
done

# Get validator set info
validator_set=$(ssh -o ConnectTimeout=5 paw-testnet "curl -s --max-time 5 http://127.0.0.1:11657/validators" 2>/dev/null || echo "{}")
validator_count=$(echo "$validator_set" | jq -r '.result.validators | length // 0')

log "=== Node Status ==="
log ""

# Display results with color coding
for node_info in "${ALL_NODES[@]}"; do
    IFS=':' read -r name server port type <<< "$node_info"

    height=${heights[$name]:-0}
    peer_count=${peers[$name]:-0}
    syncing=${catching_up[$name]:-true}

    # Determine status
    if (( height == 0 )); then
        status="${RED}DOWN${NC}"
    elif [[ "$syncing" == "true" ]]; then
        status="${YELLOW}SYNCING${NC}"
    elif (( max_height - height > 5 )); then
        status="${YELLOW}LAGGING${NC}"
    else
        status="${GREEN}HEALTHY${NC}"
    fi

    log "$(printf '%-10s %-20s Height: %-8s Peers: %-4s Sync: %-6s Status: %b' \
        "[$type]" "$name@$server" "$height" "$peer_count" "$syncing" "$status")"
done

log ""
log "=== Network Summary ==="
log "Chain ID: $CHAIN_ID"
log "Highest Block: $max_height"
log "Active Validators: $validator_count/4"
log ""

# Self-healing triggers
ISSUES_FOUND=0

for node_info in "${ALL_NODES[@]}"; do
    IFS=':' read -r name server port type <<< "$node_info"
    height=${heights[$name]:-0}
    peer_count=${peers[$name]:-0}

    if (( height == 0 )); then
        log "${RED}CRITICAL: $name is not responding!${NC}"
        ((ISSUES_FOUND++))
    elif (( max_height - height > 10 )); then
        log "${YELLOW}WARNING: $name is $((max_height - height)) blocks behind${NC}"
        ((ISSUES_FOUND++))
    fi
done

log ""
if (( ISSUES_FOUND == 0 )); then
    log "${GREEN}=== All 6 nodes healthy ===${NC}"
    exit 0
else
    log "${RED}=== $ISSUES_FOUND issues found ===${NC}"
    exit 1
fi
