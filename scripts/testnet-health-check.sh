#!/bin/bash
# ============================================================================
# PAW Testnet Health Check
# ============================================================================
# Checks both PAW testnet nodes (primary on paw-testnet, secondary on
# services-testnet) and verifies they are peered and synchronized.
#
# Usage: ./testnet-health-check.sh [--json] [--quiet]
#
# Run from any machine with SSH access to testnet servers.
# ============================================================================

set -uo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Options
JSON_OUTPUT=false
QUIET=false
for arg in "$@"; do
    case $arg in
        --json) JSON_OUTPUT=true ;;
        --quiet|-q) QUIET=true ;;
    esac
done

log() {
    if [ "$QUIET" = false ] && [ "$JSON_OUTPUT" = false ]; then
        echo -e "$@"
    fi
}

# Results
PRIMARY_OK=false
SECONDARY_OK=false
PEERED=false
PRIMARY_HEIGHT=0
SECONDARY_HEIGHT=0
PRIMARY_PEERS=0
SECONDARY_PEERS=0

# ============================================================================
# Check Primary Node (paw-testnet:26657)
# ============================================================================
check_primary() {
    log "${BLUE}Primary Node (paw-testnet)${NC}"

    local status
    status=$(ssh -o ConnectTimeout=5 paw-testnet "curl -s http://127.0.0.1:26657/status" 2>/dev/null)

    if [ -z "$status" ]; then
        log "${RED}✗${NC} Not responding"
        return 1
    fi

    PRIMARY_HEIGHT=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // 0')
    local catching_up=$(echo "$status" | jq -r 'if .result.sync_info.catching_up == false then "false" else "true" end')
    local moniker=$(echo "$status" | jq -r '.result.node_info.moniker // "unknown"')

    local net_info
    net_info=$(ssh -o ConnectTimeout=5 paw-testnet "curl -s http://127.0.0.1:26657/net_info" 2>/dev/null)
    PRIMARY_PEERS=$(echo "$net_info" | jq -r '.result.n_peers // 0')
    local peer_names=$(echo "$net_info" | jq -r '[.result.peers[]?.node_info.moniker] | join(", ") // ""')

    if [ "$catching_up" = "false" ]; then
        PRIMARY_OK=true
        log "${GREEN}✓${NC} ${moniker}: height=${PRIMARY_HEIGHT}, peers=${PRIMARY_PEERS} [${peer_names}]"
    else
        log "${YELLOW}⚠${NC} ${moniker}: syncing at height ${PRIMARY_HEIGHT}"
    fi
}

# ============================================================================
# Check Secondary Node (services-testnet:27657)
# ============================================================================
check_secondary() {
    log "${BLUE}Secondary Node (services-testnet)${NC}"

    local status
    status=$(ssh -o ConnectTimeout=5 services-testnet "curl -s http://127.0.0.1:27657/status" 2>/dev/null)

    if [ -z "$status" ]; then
        log "${RED}✗${NC} Not responding"
        return 1
    fi

    SECONDARY_HEIGHT=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // 0')
    local catching_up=$(echo "$status" | jq -r 'if .result.sync_info.catching_up == false then "false" else "true" end')
    local moniker=$(echo "$status" | jq -r '.result.node_info.moniker // "unknown"')

    local net_info
    net_info=$(ssh -o ConnectTimeout=5 services-testnet "curl -s http://127.0.0.1:27657/net_info" 2>/dev/null)
    SECONDARY_PEERS=$(echo "$net_info" | jq -r '.result.n_peers // 0')
    local peer_names=$(echo "$net_info" | jq -r '[.result.peers[]?.node_info.moniker] | join(", ") // ""')

    if [ "$catching_up" = "false" ]; then
        SECONDARY_OK=true
        log "${GREEN}✓${NC} ${moniker}: height=${SECONDARY_HEIGHT}, peers=${SECONDARY_PEERS} [${peer_names}]"
    else
        log "${YELLOW}⚠${NC} ${moniker}: syncing at height ${SECONDARY_HEIGHT}"
    fi
}

# ============================================================================
# Check Peering & Sync
# ============================================================================
check_peering() {
    log ""
    log "${BLUE}Network Status${NC}"

    # Peering
    if [ "$PRIMARY_PEERS" -ge 1 ] && [ "$SECONDARY_PEERS" -ge 1 ]; then
        PEERED=true
        log "${GREEN}✓${NC} Peering: primary ↔ secondary connected"
    else
        log "${RED}✗${NC} Peering: nodes not connected (primary=${PRIMARY_PEERS}, secondary=${SECONDARY_PEERS})"
    fi

    # Height sync (allow 10 block difference)
    local height_diff=$((PRIMARY_HEIGHT - SECONDARY_HEIGHT))
    if [ ${height_diff#-} -le 10 ]; then
        log "${GREEN}✓${NC} Sync: heights match (diff=${height_diff})"
    else
        log "${YELLOW}⚠${NC} Sync: height difference=${height_diff}"
    fi
}

# ============================================================================
# JSON Output
# ============================================================================
print_json() {
    cat <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "chain": "paw",
  "primary": {
    "ok": ${PRIMARY_OK},
    "height": ${PRIMARY_HEIGHT},
    "peers": ${PRIMARY_PEERS}
  },
  "secondary": {
    "ok": ${SECONDARY_OK},
    "height": ${SECONDARY_HEIGHT},
    "peers": ${SECONDARY_PEERS}
  },
  "peered": ${PEERED}
}
EOF
}

# ============================================================================
# Main
# ============================================================================
main() {
    log ""
    log "${BOLD}PAW Testnet Health Check - $(date)${NC}"
    log ""

    check_primary
    check_secondary
    check_peering

    if [ "$JSON_OUTPUT" = true ]; then
        print_json
    else
        log ""
        log "======================================================================"
        if [ "$PRIMARY_OK" = true ] && [ "$SECONDARY_OK" = true ] && [ "$PEERED" = true ]; then
            log "${GREEN}${BOLD}PAW Testnet: HEALTHY${NC}"
            log "Height: ${PRIMARY_HEIGHT} blocks, 2 nodes peered"
            exit 0
        else
            log "${RED}${BOLD}PAW Testnet: ISSUES DETECTED${NC}"
            exit 1
        fi
    fi
}

main "$@"
