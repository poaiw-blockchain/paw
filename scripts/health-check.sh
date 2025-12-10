#!/usr/bin/env bash
# Lightweight node health checker for PAW.
# Probes /status, /net_info, /validators and fails fast on errors or stale heights.

set -euo pipefail

RPC_ENDPOINT="${RPC_ENDPOINT:-http://127.0.0.1:26657}"
MAX_LAG_BLOCKS="${MAX_LAG_BLOCKS:-3}"
TIMEOUT="${TIMEOUT:-5}"

curl_json() {
  local path="$1"
  curl -fsSL --max-time "${TIMEOUT}" "${RPC_ENDPOINT}${path}"
}

status_json=$(curl_json "/status") || { echo "status endpoint failed"; exit 1; }
net_json=$(curl_json "/net_info") || { echo "net_info endpoint failed"; exit 1; }
val_json=$(curl_json "/validators") || { echo "validators endpoint failed"; exit 1; }

latest_height=$(echo "${status_json}" | jq -r '.result.sync_info.latest_block_height')
catching_up=$(echo "${status_json}" | jq -r '.result.sync_info.catching_up')
peer_count=$(echo "${net_json}" | jq -r '.result.n_peers')
validator_set_count=$(echo "${val_json}" | jq -r '.result.validators | length')

if [[ -z "${latest_height}" || "${latest_height}" == "null" ]]; then
  echo "invalid latest_block_height"
  exit 1
fi

if [[ "${catching_up}" != "false" ]]; then
  echo "node is still catching up"
  exit 2
fi

if [[ "${latest_height}" -lt "${MAX_LAG_BLOCKS}" ]]; then
  echo "height too low: ${latest_height}"
  exit 3
fi

if [[ "${peer_count}" -eq 0 ]]; then
  echo "no connected peers"
  exit 4
fi

if [[ "${validator_set_count}" -eq 0 ]]; then
  echo "validator set empty"
  exit 5
fi

echo "ok height=${latest_height} peers=${peer_count} validators=${validator_set_count}"
