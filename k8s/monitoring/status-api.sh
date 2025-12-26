#!/bin/bash
# PAW Blockchain Status API Generator
# Queries RPC endpoint and generates JSON status for the dashboard

set -euo pipefail

RPC_ENDPOINT="${RPC_ENDPOINT:-http://paw-validator-headless:26657}"
OUTPUT_FILE="${OUTPUT_FILE:-/tmp/paw-status.json}"
NAMESPACE="${NAMESPACE:-paw-blockchain}"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >&2; }

# Fetch with timeout and error handling
fetch() {
    local url="$1"
    curl -sf --connect-timeout 5 --max-time 10 "$url" 2>/dev/null || echo "{}"
}

# Get pod status from kubectl if available
get_validator_pods() {
    if command -v kubectl &>/dev/null; then
        kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator \
            -o jsonpath='{range .items[*]}{.metadata.name}|{.status.phase}|{.status.podIP}{"\n"}{end}' 2>/dev/null || echo ""
    fi
}

# Main status collection
collect_status() {
    local status_json net_info validators blockchain

    # Fetch RPC status
    status_json=$(fetch "$RPC_ENDPOINT/status")
    net_info=$(fetch "$RPC_ENDPOINT/net_info")
    validators=$(fetch "$RPC_ENDPOINT/validators?per_page=100")

    # Extract values with jq
    local chain_id block_height block_time catching_up node_version
    chain_id=$(echo "$status_json" | jq -r '.result.node_info.network // "unknown"')
    block_height=$(echo "$status_json" | jq -r '.result.sync_info.latest_block_height // "0"')
    block_time=$(echo "$status_json" | jq -r '.result.sync_info.latest_block_time // ""')
    catching_up=$(echo "$status_json" | jq -r '.result.sync_info.catching_up // false')
    node_version=$(echo "$status_json" | jq -r '.result.node_info.version // "unknown"')

    # Network info
    local peer_count
    peer_count=$(echo "$net_info" | jq -r '.result.n_peers // "0"')

    # Validator info
    local total_validators active_validators validator_list
    total_validators=$(echo "$validators" | jq -r '.result.total // "0"')
    active_validators=$(echo "$validators" | jq '[.result.validators[]? | select((.voting_power | tonumber) > 0)] | length')
    validator_list=$(echo "$validators" | jq '[.result.validators[]? | {
        address: .address,
        voting_power: (.voting_power | tonumber),
        proposer_priority: (.proposer_priority | tonumber)
    }][:10]')

    # Get recent blocks
    local min_height max_height recent_blocks
    max_height=$((block_height))
    min_height=$((max_height - 9))
    [ $min_height -lt 1 ] && min_height=1

    blockchain=$(fetch "$RPC_ENDPOINT/blockchain?minHeight=$min_height&maxHeight=$max_height")
    recent_blocks=$(echo "$blockchain" | jq '[.result.block_metas[]? | {
        height: (.header.height | tonumber),
        hash: .block_id.hash,
        time: .header.time,
        num_txs: (.num_txs // 0 | tonumber)
    }][:10]')

    # Get validator pod status
    local pod_status="[]"
    local pod_data
    pod_data=$(get_validator_pods)
    if [ -n "$pod_data" ]; then
        pod_status=$(echo "$pod_data" | while IFS='|' read -r name phase ip; do
            [ -z "$name" ] && continue
            echo "{\"name\":\"$name\",\"phase\":\"$phase\",\"ip\":\"$ip\"}"
        done | jq -s '.')
    fi

    # Calculate health status
    local health_status="healthy"
    local health_checks='{}'

    if [ "$catching_up" = "true" ]; then
        health_status="syncing"
    fi

    if [ "$peer_count" -lt 1 ] 2>/dev/null; then
        health_status="warning"
    fi

    if [ "$block_height" = "0" ] || [ -z "$block_height" ]; then
        health_status="error"
    fi

    health_checks=$(jq -n \
        --arg consensus "$([ "$catching_up" = "false" ] && echo "healthy" || echo "syncing")" \
        --arg p2p "$([ "$peer_count" -gt 0 ] && echo "healthy" || echo "warning")" \
        --arg rpc "$([ "$block_height" != "0" ] && echo "healthy" || echo "error")" \
        --arg blocks "$([ "$block_height" != "0" ] && echo "healthy" || echo "error")" \
        '{consensus: $consensus, p2p: $p2p, rpc: $rpc, block_production: $blocks}')

    # Build final JSON
    jq -n \
        --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg chain_id "$chain_id" \
        --argjson block_height "${block_height:-0}" \
        --arg block_time "$block_time" \
        --argjson catching_up "$catching_up" \
        --arg node_version "$node_version" \
        --argjson peer_count "${peer_count:-0}" \
        --argjson total_validators "${total_validators:-0}" \
        --argjson active_validators "${active_validators:-0}" \
        --argjson validators "$validator_list" \
        --argjson recent_blocks "$recent_blocks" \
        --argjson pods "$pod_status" \
        --arg health_status "$health_status" \
        --argjson health_checks "$health_checks" \
        '{
            timestamp: $timestamp,
            chain: {
                id: $chain_id,
                version: $node_version
            },
            sync: {
                block_height: $block_height,
                block_time: $block_time,
                catching_up: $catching_up
            },
            network: {
                peer_count: $peer_count
            },
            validators: {
                total: $total_validators,
                active: $active_validators,
                list: $validators
            },
            blocks: $recent_blocks,
            pods: $pods,
            health: {
                status: $health_status,
                checks: $health_checks
            }
        }'
}

# Run and output
main() {
    log "Collecting PAW blockchain status from $RPC_ENDPOINT"

    if status_data=$(collect_status 2>/dev/null); then
        echo "$status_data" | jq '.'
        if [ -n "$OUTPUT_FILE" ]; then
            echo "$status_data" > "$OUTPUT_FILE"
            log "Status written to $OUTPUT_FILE"
        fi
    else
        log "ERROR: Failed to collect status"
        jq -n '{
            timestamp: (now | todate),
            health: {status: "error", message: "Failed to collect status"},
            chain: {id: "unknown"},
            sync: {block_height: 0}
        }'
        exit 1
    fi
}

main "$@"
