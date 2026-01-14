#!/usr/bin/env bash
set -euo pipefail

# Local Phase D rehearsal:
# 1. Builds pawd and regenerates multi-validator genesis
# 2. Spins up the 4-validator + 2 sentry docker-compose stack
# 3. Waits for consensus, verifies metrics, and runs smoke tests
# 4. Packages artifacts into networks/<chain-id>/ ready for public distribution

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

CHAIN_ID="${CHAIN_ID:-paw-mvp-1}"
COMPOSE_FILE="${COMPOSE_FILE:-compose/docker-compose.4nodes-with-sentries.yml}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-paw-phase-d}"
COMPOSE_CMD=(docker compose -p "${COMPOSE_PROJECT_NAME}" -f "${PROJECT_ROOT}/${COMPOSE_FILE}")
NETWORK_DIR="${NETWORK_DIR:-${PROJECT_ROOT}/networks/${CHAIN_ID}}"
STATE_DIR="${SCRIPT_DIR}/.state"
PUBLISH_HOME="${PUBLISH_HOME:-${STATE_DIR}/phase-d-home}"
ARTIFACT_DIR="${ARTIFACT_DIR:-${PROJECT_ROOT}/artifacts/${CHAIN_ID}-local}"
PAWD_BIN="${PAWD_BIN:-${PROJECT_ROOT}/pawd}"
READY_HEIGHT="${READY_HEIGHT:-5}"
REBUILD_GENESIS="${REBUILD_GENESIS:-1}"
REHEARSAL_KEEP_STACK="${PAW_REHEARSAL_KEEP_STACK:-0}"

REQUIRED_BINS=(docker jq curl sha256sum)
for bin in "${REQUIRED_BINS[@]}"; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "[phase-d] missing dependency: $bin" >&2
    exit 1
  fi
done

ensure_pawd_binary "phase-d"

log() {
  echo "[phase-d] $*"
}

STACK_STARTED=0

cleanup() {
  if (( STACK_STARTED == 1 )) && [[ "${REHEARSAL_KEEP_STACK}" != "1" ]]; then
    log "tearing down docker stack"
    CHAIN_ID="${CHAIN_ID}" "${COMPOSE_CMD[@]}" down -v >/dev/null 2>&1 || true
  elif (( STACK_STARTED == 1 )); then
    log "leaving docker stack running (PAW_REHEARSAL_KEEP_STACK=1)"
  fi
}
trap cleanup EXIT

if [[ "${REBUILD_GENESIS}" == "1" ]]; then
  log "regenerating ${CHAIN_ID} multi-validator genesis"
  CHAIN_ID="${CHAIN_ID}" PAWD_BIN="${PAWD_BIN}" "${SCRIPT_DIR}/setup-multivalidators.sh"
else
  log "skipping genesis regeneration (REBUILD_GENESIS=0)"
fi

log "starting docker compose stack (${COMPOSE_FILE})"
CHAIN_ID="${CHAIN_ID}" "${COMPOSE_CMD[@]}" up -d --remove-orphans
STACK_STARTED=1

declare -A RPC_PORTS=(
  ["node1"]=26657
  ["node2"]=26667
  ["node3"]=26677
  ["node4"]=26687
  ["sentry1"]=30658
  ["sentry2"]=30668
)

wait_for_rpc() {
  local name=$1
  local port=$2
  local retries=${READY_RETRIES:-240}
  local sleep_seconds=${READY_SLEEP_SECONDS:-2}
  log "waiting for ${name} RPC on port ${port}"
  for ((attempt=1; attempt<=retries; attempt++)); do
    if curl -sf "http://localhost:${port}/status" | jq -e '.result.sync_info.catching_up == false' >/dev/null 2>&1; then
      local height
      height=$(curl -sf "http://localhost:${port}/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")
      log "${name} ready at height ${height}"
      return 0
    fi
    sleep "${sleep_seconds}"
  done
  log "ERROR: ${name} RPC never became ready"
  exit 1
}

for node in node1 node2 node3 node4 sentry1 sentry2; do
  wait_for_rpc "$node" "${RPC_PORTS[$node]}"
done

# ensure network advanced a few blocks
log "waiting for chain height >= ${READY_HEIGHT} on node1"
for ((attempt=1; attempt<=READY_HEIGHT*20; attempt++)); do
  height=$(curl -sf "http://localhost:${RPC_PORTS[node1]}/status" | jq -r '.result.sync_info.latest_block_height // "0"' 2>/dev/null || echo "0")
  if [[ -n "${height}" ]] && (( height >= READY_HEIGHT )); then
    log "chain height ${height}"
    break
  fi
  sleep 1
done

verify_metrics() {
  local container=$1
  if ! docker exec "$container" wget -qO- http://localhost:26660/metrics >/tmp/paw-metrics-"$container" 2>/dev/null; then
    log "ERROR: metrics check failed for ${container}"
    exit 1
  fi
  if ! grep -q 'tendermint_consensus_height' "/tmp/paw-metrics-${container}"; then
    log "WARNING: metrics for ${container} missing tendermint_consensus_height"
  else
    log "${container} metrics OK"
  fi
  rm -f "/tmp/paw-metrics-${container}"
}

for container in paw-node1 paw-node2 paw-node3 paw-node4 paw-sentry1 paw-sentry2; do
  verify_metrics "$container"
done

log "running smoke tests against ${CHAIN_ID}"
CHAIN_ID="${CHAIN_ID}" \
COMPOSE_FILE="${COMPOSE_FILE}" \
RPC_ENDPOINT="http://localhost:${RPC_PORTS[node1]}" \
API_ENDPOINT="http://localhost:1317" \
PAW_SMOKE_KEEP_STACK=1 \
"${SCRIPT_DIR}/smoke_tests.sh"

log "capturing node1 config for artifact packaging"
rm -rf "${PUBLISH_HOME}"
mkdir -p "${PUBLISH_HOME}/config"
docker cp paw-node1:/root/.paw/node1/config/genesis.json "${PUBLISH_HOME}/config/genesis.json" >/dev/null
docker cp paw-node1:/root/.paw/node1/config/config.toml "${PUBLISH_HOME}/config/config.toml" >/dev/null
docker cp paw-node1:/root/.paw/node1/config/node_key.json "${PUBLISH_HOME}/config/node_key.json" >/dev/null
if docker cp paw-node1:/root/.paw/node1/config/client.toml "${PUBLISH_HOME}/config/client.toml" >/dev/null 2>&1; then
  :
fi

log "publishing artifacts into ${NETWORK_DIR}"
CHAIN_ID="${CHAIN_ID}" \
PAW_HOME="${PUBLISH_HOME}" \
PAWD_BIN="${PAWD_BIN}" \
NETWORK_DIR="${NETWORK_DIR}" \
ARTIFACT_DIR="${ARTIFACT_DIR}" \
KEEP_ARTIFACT_DIR=1 \
"${SCRIPT_DIR}/publish-testnet-artifacts.sh"

CHAIN_ID="${CHAIN_ID}" NETWORK_DIR="${NETWORK_DIR}" "${SCRIPT_DIR}/verify-network-artifacts.sh"

log "Phase D rehearsal complete. Artifacts ready in ${NETWORK_DIR}."
log "Next steps: upload genesis + sha to CDN, update docs, and invite external validators."
