#!/bin/bash

# ============================================================================
# PAW Blockchain - Log Query Helper
# Query Loki logs via CLI for debugging and troubleshooting
# ============================================================================

set -euo pipefail

LOKI_URL="${LOKI_URL:-http://localhost:11101}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ----------------------------------------------------------------------------
# Color codes for output
# ----------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ----------------------------------------------------------------------------
# Helper functions
# ----------------------------------------------------------------------------
print_usage() {
    cat <<EOF
Usage: $0 <command> [options]

Commands:
    errors [HOURS]              Find all errors in last N hours (default: 1)
    warnings [HOURS]            Find all warnings in last N hours (default: 1)
    tx TXHASH                   Find logs for specific transaction hash
    service SERVICE [HOURS]     Find logs for specific service (default: 1 hour)
    consensus [HOURS]           Find consensus-related logs (default: 1)
    dex [HOURS]                 Find DEX transaction logs (default: 1)
    oracle [HOURS]              Find oracle price feed logs (default: 1)
    query LOGQL                 Execute custom LogQL query
    labels                      List all available log labels
    values LABEL                List all values for a specific label
    export SERVICE HOURS FILE   Export logs to file

Examples:
    $0 errors 24                # Errors in last 24 hours
    $0 tx A1B2C3D4...           # Logs for specific transaction
    $0 service paw-node1 2      # Logs from paw-node1 for last 2 hours
    $0 consensus 1              # Consensus logs for last hour
    $0 export paw-node1 24 node1-logs.txt

EOF
}

query_loki() {
    local query="$1"
    local limit="${2:-1000}"

    curl -sG "${LOKI_URL}/loki/api/v1/query_range" \
        --data-urlencode "query=${query}" \
        --data-urlencode "limit=${limit}" \
        | jq -r '.data.result[] | .values[] | .[1]' 2>/dev/null || echo "No results found"
}

instant_query() {
    local query="$1"

    curl -sG "${LOKI_URL}/loki/api/v1/query" \
        --data-urlencode "query=${query}" \
        | jq '.data.result' 2>/dev/null
}

# ----------------------------------------------------------------------------
# Command implementations
# ----------------------------------------------------------------------------
cmd_errors() {
    local hours="${1:-1}"
    echo -e "${RED}Errors in last ${hours} hour(s):${NC}"
    query_loki "{container_name=~\"paw-.*\"} |~ \"(?i)(error|fail|exception|fatal|panic)\"" 5000
}

cmd_warnings() {
    local hours="${1:-1}"
    echo -e "${YELLOW}Warnings in last ${hours} hour(s):${NC}"
    query_loki "{container_name=~\"paw-.*\"} |~ \"(?i)warn\"" 5000
}

cmd_tx() {
    local txhash="$1"

    if [[ -z "$txhash" ]]; then
        echo -e "${RED}Error: Transaction hash required${NC}"
        exit 1
    fi

    echo -e "${BLUE}Logs for transaction ${txhash}:${NC}"
    query_loki "{container_name=~\"paw-.*\"} |~ \"${txhash}\"" 1000
}

cmd_service() {
    local service="$1"
    local hours="${2:-1}"

    if [[ -z "$service" ]]; then
        echo -e "${RED}Error: Service name required${NC}"
        exit 1
    fi

    echo -e "${GREEN}Logs from ${service} in last ${hours} hour(s):${NC}"
    query_loki "{container_name=\"${service}\"}" 5000
}

cmd_consensus() {
    local hours="${1:-1}"
    echo -e "${BLUE}Consensus logs in last ${hours} hour(s):${NC}"
    query_loki "{container_name=~\"paw-node.*\"} |~ \"(?i)consensus\"" 5000
}

cmd_dex() {
    local hours="${1:-1}"
    echo -e "${GREEN}DEX transaction logs in last ${hours} hour(s):${NC}"
    query_loki "{container_name=~\"paw-node.*\"} |~ \"(?i)dex\"" 5000
}

cmd_oracle() {
    local hours="${1:-1}"
    echo -e "${YELLOW}Oracle price feed logs in last ${hours} hour(s):${NC}"
    query_loki "{container_name=~\"paw-node.*\"} |~ \"(?i)oracle\"" 5000
}

cmd_query() {
    local logql="$1"

    if [[ -z "$logql" ]]; then
        echo -e "${RED}Error: LogQL query required${NC}"
        exit 1
    fi

    echo -e "${BLUE}Query results:${NC}"
    query_loki "$logql" 5000
}

cmd_labels() {
    echo -e "${GREEN}Available log labels:${NC}"
    curl -s "${LOKI_URL}/loki/api/v1/labels" | jq -r '.data[]'
}

cmd_values() {
    local label="$1"

    if [[ -z "$label" ]]; then
        echo -e "${RED}Error: Label name required${NC}"
        exit 1
    fi

    echo -e "${GREEN}Values for label '${label}':${NC}"
    curl -s "${LOKI_URL}/loki/api/v1/label/${label}/values" | jq -r '.data[]'
}

cmd_export() {
    local service="$1"
    local hours="${2:-24}"
    local output_file="${3:-logs-export.txt}"

    if [[ -z "$service" ]]; then
        echo -e "${RED}Error: Service name required${NC}"
        exit 1
    fi

    echo -e "${BLUE}Exporting logs from ${service} (last ${hours} hours) to ${output_file}...${NC}"
    query_loki "{container_name=\"${service}\"}" 100000 > "$output_file"
    echo -e "${GREEN}Logs exported to: ${output_file}${NC}"
    echo -e "${BLUE}Total lines: $(wc -l < "$output_file")${NC}"
}

# ----------------------------------------------------------------------------
# Main
# ----------------------------------------------------------------------------
main() {
    if [[ $# -lt 1 ]]; then
        print_usage
        exit 1
    fi

    local command="$1"
    shift

    case "$command" in
        errors)
            cmd_errors "$@"
            ;;
        warnings)
            cmd_warnings "$@"
            ;;
        tx)
            cmd_tx "$@"
            ;;
        service)
            cmd_service "$@"
            ;;
        consensus)
            cmd_consensus "$@"
            ;;
        dex)
            cmd_dex "$@"
            ;;
        oracle)
            cmd_oracle "$@"
            ;;
        query)
            cmd_query "$@"
            ;;
        labels)
            cmd_labels
            ;;
        values)
            cmd_values "$@"
            ;;
        export)
            cmd_export "$@"
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            print_usage
            exit 1
            ;;
    esac
}

main "$@"
