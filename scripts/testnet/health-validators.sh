#!/usr/bin/env bash
# Comprehensive health check for the 4-validator PAW testnet.
# - Reads peers from networks/paw-testnet-1/peers.txt (or manifest fallback)
# - Verifies RPC health, chain-id, sync status, voting power, and peer connectivity
# - Compares heights across validators

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MANIFEST_DEFAULT="${REPO_ROOT}/networks/paw-testnet-1/paw-testnet-1-manifest.json"
PEERS_FILE_DEFAULT="${REPO_ROOT}/networks/paw-testnet-1/peers.txt"

CHAIN_ID="${CHAIN_ID:-${PAW_CHAIN_ID:-}}"
MANIFEST_PATH="${MANIFEST_PATH:-$MANIFEST_DEFAULT}"
PEERS_FILE="${PEERS_FILE:-$PEERS_FILE_DEFAULT}"
PERSISTENT_PEERS="${PERSISTENT_PEERS:-}"

RPC_SCHEME="${RPC_SCHEME:-https}"
RPC_PORT="${RPC_PORT:-}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-5}"
# Community alerting norms often flag no-new-blocks at ~2m and lag at ~100 blocks.
MAX_HEIGHT_DIFF="${MAX_HEIGHT_DIFF:-100}"
MAX_BLOCK_AGE_SECONDS="${MAX_BLOCK_AGE_SECONDS:-120}"
STRICT_PEERS=false
JSON_OUTPUT=false
QUIET=false

usage() {
  cat <<'USAGE'
Usage: health-validators.sh [options]

Options:
  --chain-id <id>          Expected chain-id (default: manifest or paw-testnet-1)
  --manifest <path>        Manifest JSON (default: networks/paw-testnet-1/paw-testnet-1-manifest.json)
  --peers-file <path>      peers.txt path (default: networks/paw-testnet-1/peers.txt)
  --peers <string>         Override persistent peers string (nodeid@host:port,...)
  --rpc-scheme <scheme>    RPC scheme (default: https)
  --rpc-port <port>        RPC port override (default: empty = 443 for https)
  --timeout <seconds>      Curl connect timeout (default: 5)
  --max-height-diff <n>    Max height difference before warning (default: 100)
  --max-block-age <sec>    Max allowed block age (default: 120)
  --strict-peers           Fail if peers are missing expected validators
  --json                   JSON output
  --quiet                  Suppress non-JSON output
  -h, --help               Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain-id) CHAIN_ID="$2"; shift 2;;
    --manifest) MANIFEST_PATH="$2"; shift 2;;
    --peers-file) PEERS_FILE="$2"; shift 2;;
    --peers) PERSISTENT_PEERS="$2"; shift 2;;
    --rpc-scheme) RPC_SCHEME="$2"; shift 2;;
    --rpc-port) RPC_PORT="$2"; shift 2;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2;;
    --max-height-diff) MAX_HEIGHT_DIFF="$2"; shift 2;;
    --max-block-age) MAX_BLOCK_AGE_SECONDS="$2"; shift 2;;
    --strict-peers) STRICT_PEERS=true; shift;;
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

if [[ -z "$CHAIN_ID" ]]; then
  if [[ -f "$MANIFEST_PATH" ]]; then
    CHAIN_ID=$(jq -r '.chain_id // empty' "$MANIFEST_PATH")
  fi
fi
if [[ -z "$CHAIN_ID" ]]; then
  CHAIN_ID="paw-testnet-1"
fi

if [[ -z "$PERSISTENT_PEERS" ]]; then
  if [[ -f "$PEERS_FILE" ]]; then
    PERSISTENT_PEERS=$(grep -E '^persistent_peers=' "$PEERS_FILE" | head -1 | cut -d= -f2-)
  fi
fi
if [[ -z "$PERSISTENT_PEERS" && -f "$MANIFEST_PATH" ]]; then
  PERSISTENT_PEERS=$(jq -r '.persistent_peers // empty' "$MANIFEST_PATH")
fi

if [[ -z "$PERSISTENT_PEERS" ]]; then
  echo "error: no peers found (set --peers or provide peers.txt/manifest)" >&2
  exit 1
fi

PERSISTENT_PEERS="${PERSISTENT_PEERS%\"}"
PERSISTENT_PEERS="${PERSISTENT_PEERS#\"}"

IFS=',' read -r -a PEER_ENTRIES <<< "$PERSISTENT_PEERS"

NODE_IDS=()
NODE_HOSTS=()
NODE_P2P_PORTS=()
for peer in "${PEER_ENTRIES[@]}"; do
  peer_trimmed="${peer//[[:space:]]/}"
  [[ -z "$peer_trimmed" ]] && continue
  node_id="${peer_trimmed%%@*}"
  hostport="${peer_trimmed#*@}"
  host="${hostport%%:*}"
  port="${hostport##*:}"
  NODE_IDS+=("$node_id")
  NODE_HOSTS+=("$host")
  NODE_P2P_PORTS+=("$port")
done

NODE_COUNT=${#NODE_IDS[@]}

log() {
  if [[ "$QUIET" == false && "$JSON_OUTPUT" == false ]]; then
    printf "%s\n" "$*"
  fi
}

rpc_url_for_host() {
  local host="$1"
  if [[ -n "$RPC_PORT" ]]; then
    printf "%s://%s:%s" "$RPC_SCHEME" "$host" "$RPC_PORT"
  else
    printf "%s://%s" "$RPC_SCHEME" "$host"
  fi
}

json_nodes=()
heights=()
validator_addresses=()

overall_state="HEALTHY"

log "PAW Testnet Validator Health Check"
log "Chain ID: $CHAIN_ID"
log "Peers: $NODE_COUNT"
log ""

for i in "${!NODE_IDS[@]}"; do
  node_id_expected="${NODE_IDS[$i]}"
  host="${NODE_HOSTS[$i]}"
  rpc_url="$(rpc_url_for_host "$host")"

  node_state="OK"
  node_errors=()
  node_warnings=()

  health=""
  status=""
  net_info=""

  if ! health=$(curl -fsS --connect-timeout "$TIMEOUT_SECONDS" "$rpc_url/health" 2>/dev/null); then
    node_state="FAIL"
    node_errors+=("rpc_health_unreachable")
  fi

  if ! status=$(curl -fsS --connect-timeout "$TIMEOUT_SECONDS" "$rpc_url/status" 2>/dev/null); then
    node_state="FAIL"
    node_errors+=("rpc_status_unreachable")
  fi

  if [[ -n "$status" ]]; then
    node_id_actual=$(echo "$status" | jq -r '.result.node_info.id // empty')
    moniker=$(echo "$status" | jq -r '.result.node_info.moniker // ""')
    chain_actual=$(echo "$status" | jq -r '.result.node_info.network // empty')
    catching_up=$(echo "$status" | jq -r '.result.sync_info.catching_up // "true"')
    height=$(echo "$status" | jq -r '.result.sync_info.latest_block_height // 0')
    latest_time=$(echo "$status" | jq -r '.result.sync_info.latest_block_time // empty')
    voting_power=$(echo "$status" | jq -r '.result.validator_info.voting_power // 0')
    val_address=$(echo "$status" | jq -r '.result.validator_info.address // empty')

    validator_addresses+=("$val_address")

    if [[ "$node_id_actual" != "$node_id_expected" ]]; then
      node_state="FAIL"
      node_errors+=("node_id_mismatch")
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
          node_errors+=("stale_block_time")
        fi
      else
        node_warnings+=("block_time_parse_failed")
      fi
    else
      node_warnings+=("missing_block_time")
    fi
  fi

  if ! net_info=$(curl -fsS --connect-timeout "$TIMEOUT_SECONDS" "$rpc_url/net_info" 2>/dev/null); then
    node_state="FAIL"
    node_errors+=("rpc_net_info_unreachable")
  fi

  peer_count=0
  missing_peers=()
  if [[ -n "$net_info" ]]; then
    peer_count=$(echo "$net_info" | jq -r '.result.n_peers // 0')
    peer_ids=$(echo "$net_info" | jq -r '.result.peers[]?.node_info.id')
    if [[ "$peer_count" -eq 0 ]]; then
      node_state="FAIL"
      node_errors+=("no_peers")
    elif [[ "$peer_count" -lt 3 ]]; then
      node_warnings+=("low_peer_count")
    fi
    for expected in "${NODE_IDS[@]}"; do
      [[ "$expected" == "$node_id_expected" ]] && continue
      if ! echo "$peer_ids" | grep -F -q "$expected"; then
        missing_peers+=("$expected")
      fi
    done

    if [[ ${#missing_peers[@]} -gt 0 ]]; then
      if [[ "$STRICT_PEERS" == true ]]; then
        node_state="FAIL"
        node_errors+=("missing_peers")
      else
        node_warnings+=("missing_peers")
      fi
    fi
  fi

  if [[ "$node_state" == "FAIL" ]]; then
    overall_state="UNHEALTHY"
  elif [[ ${#node_warnings[@]} -gt 0 && "$overall_state" == "HEALTHY" ]]; then
    overall_state="DEGRADED"
  fi

  if [[ "$JSON_OUTPUT" == false ]]; then
    log "Node: $host"
    log "  RPC: $rpc_url"
    log "  Node ID: $node_id_expected"
    if [[ -n "${moniker:-}" ]]; then
      log "  Moniker: $moniker"
    fi
    if [[ -n "${height:-}" ]]; then
      log "  Height: $height"
    fi
    if [[ -n "${voting_power:-}" ]]; then
      log "  Voting Power: $voting_power"
    fi
    log "  Peers: $peer_count"
    if [[ "$node_state" == "OK" ]]; then
      log "  Status: OK"
    else
      log "  Status: FAIL"
    fi
    if [[ ${#node_errors[@]} -gt 0 ]]; then
      log "  Errors: ${node_errors[*]}"
    fi
    if [[ ${#node_warnings[@]} -gt 0 ]]; then
      log "  Warnings: ${node_warnings[*]}"
    fi
    if [[ ${#missing_peers[@]} -gt 0 ]]; then
      log "  Missing peers: ${missing_peers[*]}"
    fi
    log ""
  fi

  node_json=$(jq -n \
    --arg host "$host" \
    --arg rpc "$rpc_url" \
    --arg node_id_expected "$node_id_expected" \
    --arg node_id_actual "${node_id_actual:-}" \
    --arg moniker "${moniker:-}" \
    --arg chain_id "${chain_actual:-}" \
    --arg height "${height:-}" \
    --arg voting_power "${voting_power:-}" \
    --arg catching_up "${catching_up:-}" \
    --arg latest_time "${latest_time:-}" \
    --argjson peer_count "$peer_count" \
    --arg state "$node_state" \
    --argjson errors "$(printf '%s\n' "${node_errors[@]}" | jq -R . | jq -s '.')" \
    --argjson warnings "$(printf '%s\n' "${node_warnings[@]}" | jq -R . | jq -s '.')" \
    --argjson missing_peers "$(printf '%s\n' "${missing_peers[@]}" | jq -R . | jq -s '.')" \
    '{host: $host, rpc: $rpc, node_id_expected: $node_id_expected, node_id_actual: $node_id_actual, moniker: $moniker, chain_id: $chain_id, height: $height, voting_power: $voting_power, catching_up: $catching_up, latest_block_time: $latest_time, peer_count: $peer_count, state: $state, errors: $errors, warnings: $warnings, missing_peers: $missing_peers}'
  )
  json_nodes+=("$node_json")

done

height_diff=0
if [[ ${#heights[@]} -gt 1 ]]; then
  min_height="${heights[0]}"
  max_height="${heights[0]}"
  for h in "${heights[@]}"; do
    if (( h < min_height )); then
      min_height="$h"
    fi
    if (( h > max_height )); then
      max_height="$h"
    fi
  done
  height_diff=$((max_height - min_height))
  if (( height_diff > MAX_HEIGHT_DIFF )); then
    if [[ "$overall_state" == "HEALTHY" ]]; then
      overall_state="DEGRADED"
    fi
  fi
fi

unique_validator_count=$(printf '%s\n' "${validator_addresses[@]}" | sort -u | grep -c . || true)

if [[ "$JSON_OUTPUT" == true ]]; then
  nodes_array=$(printf '%s\n' "${json_nodes[@]}" | jq -s '.')
  jq -n \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg chain_id "$CHAIN_ID" \
    --arg state "$overall_state" \
    --argjson validator_count "$NODE_COUNT" \
    --argjson unique_validator_addresses "$unique_validator_count" \
    --argjson height_diff "$height_diff" \
    --argjson nodes "$nodes_array" \
    '{timestamp: $timestamp, chain_id: $chain_id, state: $state, expected_validators: $validator_count, unique_validator_addresses: $unique_validator_addresses, height_diff: $height_diff, nodes: $nodes}'
else
  log "Summary:"
  log "  Expected validators: $NODE_COUNT"
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
