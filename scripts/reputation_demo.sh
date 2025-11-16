#!/bin/bash
# PAW P2P Reputation System Demo Script
# This script demonstrates the reputation system capabilities

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
API_URL="${API_URL:-http://localhost:8080}"

echo "========================================="
echo "PAW P2P Reputation System Demo"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if API is available
check_api() {
    echo -n "Checking API availability... "
    if curl -s -f "${API_URL}/api/p2p/reputation/health" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        echo "Error: Reputation API not available at ${API_URL}"
        echo "Please ensure the PAW node is running with reputation system enabled."
        return 1
    fi
}

# Display system health
show_health() {
    echo ""
    echo "=== System Health ==="
    curl -s "${API_URL}/api/p2p/reputation/health" | jq '.'
}

# Display statistics
show_stats() {
    echo ""
    echo "=== System Statistics ==="
    curl -s "${API_URL}/api/p2p/reputation/stats" | jq '.'
}

# Display top peers
show_top_peers() {
    local count=${1:-10}
    echo ""
    echo "=== Top $count Peers ==="
    curl -s "${API_URL}/api/p2p/reputation/top?n=$count&min_score=0" | jq '.'
}

# Display all peers
show_all_peers() {
    echo ""
    echo "=== All Peers ==="
    curl -s "${API_URL}/api/p2p/reputation/peers" | jq -r '.peers[] | "\(.peer_id | .[0:16])... | Score: \(.score | tonumber | floor) | Trust: \(.trust_level) | Status: \(if .ban_status.is_banned then "BANNED" else "Active" end)"'
}

# Show specific peer
show_peer() {
    local peer_id=$1
    echo ""
    echo "=== Peer Details: $peer_id ==="
    curl -s "${API_URL}/api/p2p/reputation/peer/${peer_id}" | jq '.'
}

# Display alerts
show_alerts() {
    echo ""
    echo "=== Recent Alerts (24h) ==="
    local since=$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-24H +%Y-%m-%dT%H:%M:%SZ)
    curl -s "${API_URL}/api/p2p/reputation/alerts?since=${since}" | jq '.'
}

# Display Prometheus metrics
show_metrics() {
    echo ""
    echo "=== Prometheus Metrics ==="
    curl -s "${API_URL}/api/p2p/reputation/metrics/prometheus"
}

# Ban a peer (demo - requires peer ID)
ban_peer() {
    local peer_id=$1
    local duration=${2:-24h}
    local reason=${3:-"Demo ban"}

    echo ""
    echo "=== Banning Peer: $peer_id ==="
    curl -s -X POST "${API_URL}/api/p2p/reputation/ban" \
        -H "Content-Type: application/json" \
        -d "{\"peer_id\":\"$peer_id\",\"duration\":\"$duration\",\"reason\":\"$reason\"}" | jq '.'
}

# Unban a peer
unban_peer() {
    local peer_id=$1

    echo ""
    echo "=== Unbanning Peer: $peer_id ==="
    curl -s -X POST "${API_URL}/api/p2p/reputation/unban" \
        -H "Content-Type: application/json" \
        -d "{\"peer_id\":\"$peer_id\"}" | jq '.'
}

# Monitor mode - continuously display stats
monitor() {
    echo ""
    echo "=== Monitoring Mode (Ctrl+C to exit) ==="
    echo ""

    while true; do
        clear
        echo "PAW P2P Reputation Monitor - $(date)"
        echo "==========================================="

        # Health
        health=$(curl -s "${API_URL}/api/p2p/reputation/health")
        healthy=$(echo "$health" | jq -r '.healthy')
        total_peers=$(echo "$health" | jq -r '.total_peers')
        banned_peers=$(echo "$health" | jq -r '.banned_peers')
        avg_score=$(echo "$health" | jq -r '.avg_score')

        if [ "$healthy" = "true" ]; then
            echo -e "Status: ${GREEN}Healthy${NC}"
        else
            echo -e "Status: ${RED}Unhealthy${NC}"
        fi

        echo "Total Peers: $total_peers"
        echo "Banned Peers: $banned_peers"
        echo "Average Score: $(printf "%.1f" $avg_score)"

        # Stats
        echo ""
        echo "Score Distribution:"
        curl -s "${API_URL}/api/p2p/reputation/stats" | jq -r '.score_distribution | to_entries[] | "  \(.key): \(.value)"'

        echo ""
        echo "Trust Distribution:"
        curl -s "${API_URL}/api/p2p/reputation/stats" | jq -r '.trust_distribution | to_entries[] | "  \(.key): \(.value)"'

        sleep 5
    done
}

# Export data
export_data() {
    local output=${1:-reputation_export.json}
    echo ""
    echo "=== Exporting Reputation Data ==="
    curl -s "${API_URL}/api/p2p/reputation/peers" > "$output"
    echo "Data exported to: $output"
    echo "Total peers: $(jq '.count' "$output")"
}

# Main menu
show_menu() {
    echo ""
    echo "Available Commands:"
    echo "  health      - Show system health"
    echo "  stats       - Show statistics"
    echo "  peers       - List all peers"
    echo "  top [N]     - Show top N peers (default: 10)"
    echo "  show <id>   - Show peer details"
    echo "  alerts      - Show recent alerts"
    echo "  metrics     - Show Prometheus metrics"
    echo "  ban <id>    - Ban a peer"
    echo "  unban <id>  - Unban a peer"
    echo "  monitor     - Continuous monitoring"
    echo "  export [f]  - Export data to file"
    echo "  help        - Show this menu"
    echo "  exit        - Exit demo"
    echo ""
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}Warning: 'jq' is not installed. JSON output will not be formatted.${NC}"
    echo "Install with: apt-get install jq  (Ubuntu/Debian)"
    echo "           or: brew install jq     (macOS)"
    echo ""
fi

# Interactive mode if no arguments
if [ $# -eq 0 ]; then
    check_api || exit 1

    show_menu

    while true; do
        echo -n "> "
        read -r cmd arg1 arg2 arg3

        case "$cmd" in
            health)
                show_health
                ;;
            stats)
                show_stats
                ;;
            peers)
                show_all_peers
                ;;
            top)
                show_top_peers "${arg1:-10}"
                ;;
            show)
                if [ -z "$arg1" ]; then
                    echo "Usage: show <peer_id>"
                else
                    show_peer "$arg1"
                fi
                ;;
            alerts)
                show_alerts
                ;;
            metrics)
                show_metrics
                ;;
            ban)
                if [ -z "$arg1" ]; then
                    echo "Usage: ban <peer_id> [duration] [reason]"
                else
                    ban_peer "$arg1" "${arg2:-24h}" "${arg3:-Manual ban}"
                fi
                ;;
            unban)
                if [ -z "$arg1" ]; then
                    echo "Usage: unban <peer_id>"
                else
                    unban_peer "$arg1"
                fi
                ;;
            monitor)
                monitor
                ;;
            export)
                export_data "${arg1:-reputation_export.json}"
                ;;
            help)
                show_menu
                ;;
            exit|quit)
                echo "Goodbye!"
                exit 0
                ;;
            "")
                ;;
            *)
                echo "Unknown command: $cmd"
                show_menu
                ;;
        esac
    done
else
    # Command-line mode
    check_api || exit 1

    case "$1" in
        health)
            show_health
            ;;
        stats)
            show_stats
            ;;
        peers)
            show_all_peers
            ;;
        top)
            show_top_peers "${2:-10}"
            ;;
        show)
            if [ -z "$2" ]; then
                echo "Usage: $0 show <peer_id>"
                exit 1
            fi
            show_peer "$2"
            ;;
        alerts)
            show_alerts
            ;;
        metrics)
            show_metrics
            ;;
        ban)
            if [ -z "$2" ]; then
                echo "Usage: $0 ban <peer_id> [duration] [reason]"
                exit 1
            fi
            ban_peer "$2" "${3:-24h}" "${4:-Manual ban}"
            ;;
        unban)
            if [ -z "$2" ]; then
                echo "Usage: $0 unban <peer_id>"
                exit 1
            fi
            unban_peer "$2"
            ;;
        monitor)
            monitor
            ;;
        export)
            export_data "${2:-reputation_export.json}"
            ;;
        *)
            echo "Unknown command: $1"
            show_menu
            exit 1
            ;;
    esac
fi
