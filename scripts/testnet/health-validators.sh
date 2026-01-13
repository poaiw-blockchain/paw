#!/usr/bin/env bash
# Comprehensive health check for the 4-validator PAW testnet.
# Uses SSH to check validators directly (designed to run from bcpc)
#
# MVP Testnet Port Configuration:
#   val1: paw-testnet (54.39.103.49) - RPC 11657, gRPC 11090, REST 11317, P2P 11656
#   val2: paw-testnet (54.39.103.49) - RPC 11757, gRPC 11190, REST 11417, P2P 11756
#   val3: services-testnet (139.99.149.160) - RPC 11857, gRPC 11290, REST 11517, P2P 11856
#   val4: services-testnet (139.99.149.160) - RPC 11957, gRPC 11390, REST 11617, P2P 11956
#
# Sentry Node:
#   sentry1: services-testnet (139.99.149.160) - RPC 12057, gRPC 12090, REST 12017, P2P 12056
#
# For sentry-specific health checks, use: health-sentry.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-5}"
MAX_HEIGHT_DIFF="${MAX_HEIGHT_DIFF:-100}"
MAX_BLOCK_AGE_SECONDS="${MAX_BLOCK_AGE_SECONDS:-120}"
JSON_OUTPUT=false
QUIET=false

# Validator definitions: name|ssh_alias|rpc_port|expected_node_id
VALIDATORS=(
    "val1|paw-testnet|11657|72c594a424bfc156381860feaca3a2586173eead"
    "val2|paw-testnet|11757|1780e068618ca0ffcba81574d62ab170c2ee3c8b"
    "val3|services-testnet|11857|a2b9ab78b0be7f006466131b44ede9a02fc140c4"
    "val4|services-testnet|11957|f8187d5bafe58b78b00d73b0563b65ad8c0d5fda"
)

usage() {
  cat <<'USAGE'
Usage: health-validators.sh [options]

Options:
  --chain-id <id>          Expected chain-id (default: paw-testnet-1)
  --timeout <seconds>      Curl connect timeout (default: 5)
  --max-height-diff <n>    Max height difference before warning (default: 100)
  --max-block-age <sec>    Max allowed block age (default: 120)
  --json                   JSON output
  --quiet                  Suppress non-JSON output
  -h, --help               Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain-id) CHAIN_ID="$2"; shift 2;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2;;
    --max-height-diff) MAX_HEIGHT_DIFF="$2"; shift 2;;
    --max-block-age) MAX_BLOCK_AGE_SECONDS="$2"; shift 2;;
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
heights=()
validator_addresses=()
overall_state="HEALTHY"

log "PAW Testnet Validator Health Check"
log "Chain ID: $CHAIN_ID"
log "Validators: ${#VALIDATORS[@]}"
log ""

for validator in "${VALIDATORS[@]}"; do
  IFS='|' read -r name ssh_alias rpc_port expected_node_id <<< "$validator"

  node_state="OK"
  node_errors=()
  node_warnings=()

  status=""
  net_info=""

  # Fetch status via SSH
  if ! status=$(ssh -o ConnectTimeout=3 -o BatchMode=yes "$ssh_alias" \
    "curl -s --max-time $TIMEOUT_SECONDS http://127.0.0.1:$rpc_port/status" 2>/dev/null); then
    node_state="FAIL"
    node_errors+=("rpc_status_unreachable")
  fi

  if [[ -n "$status" && "$status" != "" ]]; then
    node_id_actual=$(echo "$status" | jq -r '.result.node_info.id // empty')
    moniker=$(echo "$status" | jq -r '.result.node_info.moniker // ""')
    chain_actual=$(echo "$status" | jq -r '.result.node_info.network // empty')
    catching_up=$(echo "$status" | jq -r '(.result.sync_info.catching_up | tostring) // "true"')
    height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // 0')
    latest_time=$(echo "$status" | jq -r '.result.sync_info.latest_block_time // empty')
    voting_power=$(echo "$status" | jq -r '.result.validator_info.voting_power // 0')
    val_address=$(echo "$status" | jq -r '.result.validator_info.address // empty')

    validator_addresses+=("$val_address")

    if [[ "$node_id_actual" != "$expected_node_id" ]]; then
      node_warnings+=("node_id_mismatch:expected=$expected_node_id,got=$node_id_actual")
    fi
    if [[ "$chain_actual" != "$CHAIN_ID" ]]; then
      node_state="FAIL"
      node_errors+=("chain_id_mismatch")
    fi
    if [[ "$catching_up" != "false" ]]; then
      node_state="FAIL"
      node_errors+=("node_catching_up")
    fi
    if ! [[ "$height" =~ ^[0-9]+$ ]]; then
      node_state="FAIL"
      node_errors+=("invalid_height")
    else
      heights+=("$height")
    fi
    if ! [[ "$voting_power" =~ ^[0-9]+$ ]]; then
      node_state="FAIL"
      node_errors+=("invalid_voting_power")
    fi
    if [[ "$voting_power" -le 0 ]]; then
      node_state="FAIL"
      node_errors+=("zero_voting_power")
    fi

    if [[ -n "$latest_time" ]]; then
      block_epoch=$(date -u -d "$latest_time" +%s 2>/dev/null || echo "")
      if [[ -n "$block_epoch" ]]; then
        now_epoch=$(date -u +%s)
        age=$((now_epoch - block_epoch))
        if [[ "$age" -gt "$MAX_BLOCK_AGE_SECONDS" ]]; then
          node_state="FAIL"
          node_errors+=("stale_block_time:age=${age}s")
        fi
      fi
    fi
  fi

  # Fetch net_info via SSH
  if ! net_info=$(ssh -o ConnectTimeout=3 -o BatchMode=yes "$ssh_alias" \
    "curl -s --max-time $TIMEOUT_SECONDS http://127.0.0.1:$rpc_port/net_info" 2>/dev/null); then
    node_warnings+=("rpc_net_info_unreachable")
  fi

  peer_count=0
  if [[ -n "$net_info" && "$net_info" != "" ]]; then
    peer_count=$(echo "$net_info" | jq -r '.result.n_peers // 0')
    if [[ "$peer_count" -eq 0 ]]; then
      node_state="FAIL"
      node_errors+=("no_peers")
    elif [[ "$peer_count" -lt 2 ]]; then
      node_warnings+=("low_peer_count:$peer_count")
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
    if [[ -n "${voting_power:-}" ]]; then
      log "  Voting Power: $voting_power"
    fi
    log "  Peers: $peer_count"
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
    --arg node_id_expected "$expected_node_id" \
    --arg node_id_actual "${node_id_actual:-}" \
    --arg moniker "${moniker:-}" \
    --arg chain_id "${chain_actual:-}" \
    --arg height "${height:-}" \
    --arg voting_power "${voting_power:-}" \
    --arg catching_up "${catching_up:-}" \
    --arg latest_time "${latest_time:-}" \
    --argjson peer_count "$peer_count" \
    --arg state "$node_state" \
    --argjson errors "$(printf '%s\n' "${node_errors[@]:-}" | jq -R . | jq -s '.')" \
    --argjson warnings "$(printf '%s\n' "${node_warnings[@]:-}" | jq -R . | jq -s '.')" \
    '{name: $name, ssh_alias: $ssh_alias, rpc_port: $rpc_port, node_id_expected: $node_id_expected, node_id_actual: $node_id_actual, moniker: $moniker, chain_id: $chain_id, height: $height, voting_power: $voting_power, catching_up: $catching_up, latest_block_time: $latest_time, peer_count: $peer_count, state: $state, errors: $errors, warnings: $warnings}'
  )
  json_nodes+=("$node_json")
done

height_diff=0
if [[ ${#heights[@]} -gt 1 ]]; then
  min_height="${heights[0]}"
  max_height="${heights[0]}"
  for h in "${heights[@]}"; do
    if (( h < min_height )); then min_height="$h"; fi
    if (( h > max_height )); then max_height="$h"; fi
  done
  height_diff=$((max_height - min_height))
  if (( height_diff > MAX_HEIGHT_DIFF )); then
    if [[ "$overall_state" == "HEALTHY" ]]; then
      overall_state="DEGRADED"
    fi
  fi
fi

unique_validator_count=$(printf '%s\n' "${validator_addresses[@]}" | sort -u | grep -c . || echo "0")

if [[ "$JSON_OUTPUT" == true ]]; then
  nodes_array=$(printf '%s\n' "${json_nodes[@]}" | jq -s '.')
  jq -n \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg chain_id "$CHAIN_ID" \
    --arg state "$overall_state" \
    --argjson validator_count "${#VALIDATORS[@]}" \
    --argjson unique_validator_addresses "$unique_validator_count" \
    --argjson height_diff "$height_diff" \
    --argjson nodes "$nodes_array" \
    '{timestamp: $timestamp, chain_id: $chain_id, state: $state, expected_validators: $validator_count, unique_validator_addresses: $unique_validator_addresses, height_diff: $height_diff, nodes: $nodes}'
else
  log "Summary:"
  log "  Expected validators: ${#VALIDATORS[@]}"
  log "  Unique validator addresses: $unique_validator_count"
  log "  Height diff: $height_diff (max allowed $MAX_HEIGHT_DIFF)"
  log "  Overall: $overall_state"
fi

if [[ "$overall_state" == "UNHEALTHY" ]]; then
  exit 1
fi
if [[ "$overall_state" == "DEGRADED" ]]; then
  exit 2
fi

exit 0
