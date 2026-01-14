#!/usr/bin/env bash
# Public endpoint health checks for PAW testnet.
# - Uses networks/paw-mvp-1/chain.json and paw-mvp-1-manifest.json
# - Checks RPC/REST/gRPC endpoints plus explorer/faucet/status URLs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MANIFEST_DEFAULT="${REPO_ROOT}/networks/paw-mvp-1/paw-mvp-1-manifest.json"
CHAIN_JSON_DEFAULT="${REPO_ROOT}/networks/paw-mvp-1/chain.json"

CHAIN_ID="${CHAIN_ID:-${PAW_CHAIN_ID:-}}"
MANIFEST_PATH="${MANIFEST_PATH:-$MANIFEST_DEFAULT}"
CHAIN_JSON_PATH="${CHAIN_JSON_PATH:-$CHAIN_JSON_DEFAULT}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-5}"
JSON_OUTPUT=false
QUIET=false

EXTRA_HTTP_ENDPOINTS="${EXTRA_HTTP_ENDPOINTS:-}"
EXTRA_TCP_ENDPOINTS="${EXTRA_TCP_ENDPOINTS:-}"

usage() {
  cat <<'USAGE'
Usage: health-public.sh [options]

Options:
  --chain-id <id>       Expected chain-id (default: manifest/chain.json or paw-mvp-1)
  --manifest <path>     Manifest JSON (default: networks/paw-mvp-1/paw-mvp-1-manifest.json)
  --chain-json <path>   Chain registry JSON (default: networks/paw-mvp-1/chain.json)
  --timeout <seconds>   Curl connect timeout (default: 5)
  --extra-http <list>   Comma-separated extra HTTP endpoints to probe
  --extra-tcp <list>    Comma-separated extra host:port endpoints to probe
  --json                JSON output
  --quiet               Suppress non-JSON output
  -h, --help            Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain-id) CHAIN_ID="$2"; shift 2;;
    --manifest) MANIFEST_PATH="$2"; shift 2;;
    --chain-json) CHAIN_JSON_PATH="$2"; shift 2;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2;;
    --extra-http) EXTRA_HTTP_ENDPOINTS="$2"; shift 2;;
    --extra-tcp) EXTRA_TCP_ENDPOINTS="$2"; shift 2;;
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
  if [[ -f "$CHAIN_JSON_PATH" ]]; then
    CHAIN_ID=$(jq -r '.chain_id // empty' "$CHAIN_JSON_PATH")
  fi
fi
if [[ -z "$CHAIN_ID" ]]; then
  CHAIN_ID="paw-mvp-1"
fi

log() {
  if [[ "$QUIET" == false && "$JSON_OUTPUT" == false ]]; then
    printf "%s\n" "$*"
  fi
}

collect_list() {
  local list="$1"
  local item
  for item in $list; do
    if [[ -n "$item" ]]; then
      printf "%s\n" "$item"
    fi
  done
}

add_unique() {
  local array_name="$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    return 0
  fi
  eval "local current=(\"\${${array_name}[@]}\")"
  for item in "${current[@]}"; do
    [[ "$item" == "$value" ]] && return 0
  done
  eval "${array_name}+=(\"$value\")"
}

rpc_endpoints=()
rest_endpoints=()
grpc_endpoints=()
explorer_urls=()
extra_http=()
extra_tcp=()

if [[ -f "$MANIFEST_PATH" ]]; then
  while read -r endpoint; do
    add_unique rpc_endpoints "$endpoint"
  done < <(jq -r '.rpc_endpoints[]? // empty' "$MANIFEST_PATH")
  while read -r endpoint; do
    add_unique rest_endpoints "$endpoint"
  done < <(jq -r '.rest_endpoints[]? // empty' "$MANIFEST_PATH")
  while read -r endpoint; do
    add_unique grpc_endpoints "$endpoint"
  done < <(jq -r '.grpc_endpoints[]? // empty' "$MANIFEST_PATH")

  manifest_faucet=$(jq -r '.faucet_url // empty' "$MANIFEST_PATH")
  manifest_status=$(jq -r '.status_page_url // empty' "$MANIFEST_PATH")
fi

if [[ -f "$CHAIN_JSON_PATH" ]]; then
  while read -r endpoint; do
    add_unique rpc_endpoints "$endpoint"
  done < <(jq -r '.apis.rpc[]?.address // empty' "$CHAIN_JSON_PATH")
  while read -r endpoint; do
    add_unique rest_endpoints "$endpoint"
  done < <(jq -r '.apis.rest[]?.address // empty' "$CHAIN_JSON_PATH")
  while read -r endpoint; do
    add_unique grpc_endpoints "$endpoint"
  done < <(jq -r '.apis.grpc[]?.address // empty' "$CHAIN_JSON_PATH")
  while read -r endpoint; do
    add_unique explorer_urls "$endpoint"
  done < <(jq -r '.explorers[]?.url // empty' "$CHAIN_JSON_PATH")
fi

if [[ -n "${manifest_faucet:-}" ]]; then
  add_unique extra_http "$manifest_faucet"
fi
if [[ -n "${manifest_status:-}" ]]; then
  add_unique extra_http "$manifest_status"
fi

if [[ -n "$EXTRA_HTTP_ENDPOINTS" ]]; then
  IFS=',' read -r -a extra_http_list <<< "$EXTRA_HTTP_ENDPOINTS"
  for endpoint in "${extra_http_list[@]}"; do
    add_unique extra_http "$endpoint"
  done
fi

if [[ -n "$EXTRA_TCP_ENDPOINTS" ]]; then
  IFS=',' read -r -a extra_tcp_list <<< "$EXTRA_TCP_ENDPOINTS"
  for endpoint in "${extra_tcp_list[@]}"; do
    add_unique extra_tcp "$endpoint"
  done
fi

check_http() {
  local url="$1"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout "$TIMEOUT_SECONDS" "$url" 2>/dev/null || echo "000")
  if [[ "$code" =~ ^(2|3) ]]; then
    printf "OK"; return 0
  fi
  printf "FAIL"; return 1
}

check_rpc() {
  local url="$1"
  local status
  status=$(curl -fsS --connect-timeout "$TIMEOUT_SECONDS" "$url/status" 2>/dev/null || true)
  if [[ -z "$status" ]]; then
    printf "FAIL"; return 1
  fi
  local chain_actual
  chain_actual=$(echo "$status" | jq -r '.result.node_info.network // empty')
  if [[ "$chain_actual" != "$CHAIN_ID" ]]; then
    printf "FAIL"; return 1
  fi
  printf "OK"; return 0
}

check_rest() {
  local url="$1"
  local info
  info=$(curl -fsS --connect-timeout "$TIMEOUT_SECONDS" "$url/cosmos/base/tendermint/v1beta1/node_info" 2>/dev/null || true)
  if [[ -z "$info" ]]; then
    printf "FAIL"; return 1
  fi
  local chain_actual
  chain_actual=$(echo "$info" | jq -r '.default_node_info.network // empty')
  if [[ "$chain_actual" != "$CHAIN_ID" ]]; then
    printf "FAIL"; return 1
  fi
  printf "OK"; return 0
}

check_tcp() {
  local hostport="$1"
  local host="${hostport%%:*}"
  local port="${hostport##*:}"
  if timeout "$TIMEOUT_SECONDS" bash -c "cat < /dev/null > /dev/tcp/${host}/${port}" 2>/dev/null; then
    printf "OK"; return 0
  fi
  printf "FAIL"; return 1
}

json_checks=()
overall_state="HEALTHY"

log "PAW Testnet Public Endpoint Health Check"
log "Chain ID: $CHAIN_ID"
log ""

if [[ ${#rpc_endpoints[@]} -gt 0 ]]; then
  log "RPC Endpoints:"
  for endpoint in "${rpc_endpoints[@]}"; do
    result=$(check_rpc "$endpoint" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "rpc" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ ${#rest_endpoints[@]} -gt 0 ]]; then
  log "REST Endpoints:"
  for endpoint in "${rest_endpoints[@]}"; do
    result=$(check_rest "$endpoint" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "rest" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ ${#grpc_endpoints[@]} -gt 0 ]]; then
  log "gRPC Endpoints:"
  for endpoint in "${grpc_endpoints[@]}"; do
    target="$endpoint"
    if [[ "$endpoint" == http* ]]; then
      target="${endpoint#*://}"
    fi
    result=$(check_tcp "$target" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "grpc" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ ${#explorer_urls[@]} -gt 0 ]]; then
  log "Explorer Endpoints:"
  for endpoint in "${explorer_urls[@]}"; do
    result=$(check_http "$endpoint" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "explorer" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ ${#extra_http[@]} -gt 0 ]]; then
  log "Additional HTTP Endpoints:"
  for endpoint in "${extra_http[@]}"; do
    result=$(check_http "$endpoint" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "http" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ ${#extra_tcp[@]} -gt 0 ]]; then
  log "Additional TCP Endpoints:"
  for endpoint in "${extra_tcp[@]}"; do
    result=$(check_tcp "$endpoint" || true)
    if [[ "$result" == "OK" ]]; then
      log "  OK   $endpoint"
    else
      log "  FAIL $endpoint"
      overall_state="UNHEALTHY"
    fi
    json_checks+=("$(jq -n --arg type "tcp" --arg url "$endpoint" --arg state "$result" '{type:$type,url:$url,state:$state}')")
  done
  log ""
fi

if [[ "$JSON_OUTPUT" == true ]]; then
  checks_array=$(printf '%s\n' "${json_checks[@]}" | jq -s '.')
  jq -n \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg chain_id "$CHAIN_ID" \
    --arg state "$overall_state" \
    --argjson checks "$checks_array" \
    '{timestamp: $timestamp, chain_id: $chain_id, state: $state, checks: $checks}'
else
  log "Overall: $overall_state"
fi

if [[ "$overall_state" == "UNHEALTHY" ]]; then
  exit 1
fi

exit 0
