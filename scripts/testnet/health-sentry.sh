#!/usr/bin/env bash
# Sentry node health check for PAW testnet
# Uses SSH to check sentry node status (designed to run from bcpc)
#
# Sentry Node Port Configuration:
#   sentry1: services-testnet (139.99.149.160) - RPC 12057, gRPC 12090, REST 12017, P2P 12056

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CHAIN_ID="${CHAIN_ID:-paw-mvp-1}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-5}"
JSON_OUTPUT=false
QUIET=false

# Sentry node definition: name|ssh_alias|rpc_port|expected_node_id
SENTRY_NODES=(
    "sentry1|services-testnet|12057|ce6afbda0a4443139ad14d2b856cca586161f00d"
)

# Validator node IDs (sentries should be connected to these)
VALIDATOR_IDS=(
    "72c594a424bfc156381860feaca3a2586173eead"
    "1780e068618ca0ffcba81574d62ab170c2ee3c8b"
    "a2b9ab78b0be7f006466131b44ede9a02fc140c4"
    "f8187d5bafe58b78b00d73b0563b65ad8c0d5fda"
)

usage() {
  cat <<'USAGE'
Usage: health-sentry.sh [options]

Options:
  --chain-id <id>          Expected chain-id (default: paw-mvp-1)
  --timeout <seconds>      Curl timeout (default: 5)
  --json                   JSON output
  --quiet                  Suppress non-JSON output
  -h, --help               Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain-id) CHAIN_ID="$2"; shift 2;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2;;
    --json) JSON_OUTPUT=true; shift;;
    --quiet|-q) QUIET=true; shift;;
    -h|--help) usage; exit 0;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1;;
  esac
done

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "error: missing dependency: $1" >&2; exit 1; }
}

require curl
require jq
require ssh

log() {
  if [[ "$QUIET" == false && "$JSON_OUTPUT" == false ]]; then
    printf "%s\n" "$*"
  fi
}

json_nodes=()
overall_state="HEALTHY"

log "PAW Testnet Sentry Node Health Check"
log "Chain ID: $CHAIN_ID"
log ""

for sentry in "${SENTRY_NODES[@]}"; do
  IFS='|' read -r name ssh_alias rpc_port expected_node_id <<< "$sentry"

  node_state="OK"
  node_errors=()
  node_warnings=()

  status=""
  net_info=""

  # Check if sentry service is running
  service_status=$(ssh -o ConnectTimeout=3 -o BatchMode=yes "$ssh_alias" \
    "systemctl is-active pawd-sentry 2>/dev/null" || echo "inactive")

  if [[ "$service_status" != "active" ]]; then
    node_state="FAIL"
    node_errors+=("service_not_running")
    log "Node: $name ($ssh_alias:$rpc_port)"
    log "  Status: $node_state"
    log "  Errors: service not running (status: $service_status)"
    log ""

    node_json=$(jq -n \
      --arg name "$name" \
      --arg ssh_alias "$ssh_alias" \
      --arg rpc_port "$rpc_port" \
      --arg state "$node_state" \
      --argjson errors '["service_not_running"]' \
      '{name: $name, ssh_alias: $ssh_alias, rpc_port: $rpc_port, state: $state, errors: $errors}')
    json_nodes+=("$node_json")
    overall_state="UNHEALTHY"
    continue
  fi

  # Fetch status via SSH
  if ! status=$(ssh -o ConnectTimeout=3 -o BatchMode=yes "$ssh_alias" \
    "curl -s --max-time $TIMEOUT_SECONDS http://127.0.0.1:$rpc_port/status" 2>/dev/null); then
    node_state="FAIL"
    node_errors+=("rpc_unreachable")
  fi

  if [[ -n "$status" && "$status" != "" ]]; then
    node_id_actual=$(echo "$status" | jq -r '.result.node_info.id // empty')
    moniker=$(echo "$status" | jq -r '.result.node_info.moniker // ""')
    chain_actual=$(echo "$status" | jq -r '.result.node_info.network // empty')
    catching_up=$(echo "$status" | jq -r '(.result.sync_info.catching_up | tostring) // "true"')
    height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // 0')
    latest_time=$(echo "$status" | jq -r '.result.sync_info.latest_block_time // empty')

    if [[ -n "$expected_node_id" && "$node_id_actual" != "$expected_node_id" ]]; then
      node_warnings+=("node_id_mismatch")
    fi
    if [[ "$chain_actual" != "$CHAIN_ID" ]]; then
      node_state="FAIL"
      node_errors+=("chain_id_mismatch")
    fi
    if [[ "$catching_up" != "false" ]]; then
      node_state="FAIL"
      node_errors+=("node_catching_up")
    fi
  fi

  # Fetch net_info via SSH
  if ! net_info=$(ssh -o ConnectTimeout=3 -o BatchMode=yes "$ssh_alias" \
    "curl -s --max-time $TIMEOUT_SECONDS http://127.0.0.1:$rpc_port/net_info" 2>/dev/null); then
    node_warnings+=("net_info_unreachable")
  fi

  peer_count=0
  validator_connections=0
  if [[ -n "$net_info" && "$net_info" != "" ]]; then
    peer_count=$(echo "$net_info" | jq -r '.result.n_peers // 0')

    # Check connections to validators
    for val_id in "${VALIDATOR_IDS[@]}"; do
      if echo "$net_info" | jq -e ".result.peers[]? | select(.node_info.id == \"$val_id\")" >/dev/null 2>&1; then
        ((validator_connections++)) || true
      fi
    done

    if [[ "$peer_count" -eq 0 ]]; then
      node_state="FAIL"
      node_errors+=("no_peers")
    fi
    if [[ "$validator_connections" -eq 0 ]]; then
      node_state="FAIL"
      node_errors+=("no_validator_connections")
    elif [[ "$validator_connections" -lt 2 ]]; then
      node_warnings+=("low_validator_connections:$validator_connections")
    fi
  fi

  if [[ "$node_state" == "FAIL" ]]; then
    overall_state="UNHEALTHY"
  elif [[ ${#node_warnings[@]} -gt 0 && "$overall_state" == "HEALTHY" ]]; then
    overall_state="DEGRADED"
  fi

  if [[ "$JSON_OUTPUT" == false ]]; then
    log "Node: $name ($ssh_alias:$rpc_port)"
    if [[ -n "${moniker:-}" ]]; then
      log "  Moniker: $moniker"
    fi
    log "  Node ID: ${node_id_actual:-unknown}"
    if [[ -n "${height:-}" ]]; then
      log "  Height: $height"
    fi
    log "  Peers: $peer_count (validators: $validator_connections/4)"
    log "  Catching Up: $catching_up"
    log "  Status: $node_state"
    if [[ ${#node_errors[@]} -gt 0 ]]; then
      log "  Errors: ${node_errors[*]}"
    fi
    if [[ ${#node_warnings[@]} -gt 0 ]]; then
      log "  Warnings: ${node_warnings[*]}"
    fi
    log ""
  fi

  node_json=$(jq -n \
    --arg name "$name" \
    --arg ssh_alias "$ssh_alias" \
    --arg rpc_port "$rpc_port" \
    --arg node_id "${node_id_actual:-}" \
    --arg moniker "${moniker:-}" \
    --arg chain_id "${chain_actual:-}" \
    --arg height "${height:-}" \
    --arg catching_up "${catching_up:-}" \
    --argjson peer_count "$peer_count" \
    --argjson validator_connections "$validator_connections" \
    --arg state "$node_state" \
    --argjson errors "$(printf '%s\n' "${node_errors[@]:-}" | jq -R . | jq -s '.')" \
    --argjson warnings "$(printf '%s\n' "${node_warnings[@]:-}" | jq -R . | jq -s '.')" \
    '{name: $name, ssh_alias: $ssh_alias, rpc_port: $rpc_port, node_id: $node_id, moniker: $moniker, chain_id: $chain_id, height: $height, catching_up: $catching_up, peer_count: $peer_count, validator_connections: $validator_connections, state: $state, errors: $errors, warnings: $warnings}')
  json_nodes+=("$node_json")
done

if [[ "$JSON_OUTPUT" == true ]]; then
  nodes_array=$(printf '%s\n' "${json_nodes[@]}" | jq -s '.')
  jq -n \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg chain_id "$CHAIN_ID" \
    --arg state "$overall_state" \
    --argjson sentry_count "${#SENTRY_NODES[@]}" \
    --argjson nodes "$nodes_array" \
    '{timestamp: $timestamp, chain_id: $chain_id, state: $state, sentry_count: $sentry_count, nodes: $nodes}'
else
  log "Summary:"
  log "  Sentry nodes: ${#SENTRY_NODES[@]}"
  log "  Overall: $overall_state"
fi

if [[ "$overall_state" == "UNHEALTHY" ]]; then
  exit 1
fi
if [[ "$overall_state" == "DEGRADED" ]]; then
  exit 2
fi

exit 0
