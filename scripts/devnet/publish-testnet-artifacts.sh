#!/usr/bin/env bash
# Wrapper that runs package-testnet-artifacts.sh and syncs results into networks/<chain-id>/.
# Usage:
#   PAW_HOME=~/.paw-testnet ./scripts/devnet/publish-testnet-artifacts.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PACKAGE_SCRIPT="${PACKAGE_SCRIPT:-${SCRIPT_DIR}/package-testnet-artifacts.sh}"

CHAIN_ID="${CHAIN_ID:-${PAW_CHAIN_ID:-paw-mvp-1}}"
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
PAWD_BIN="${PAWD_BIN:-$(command -v pawd || true)}"
NETWORK_DIR="${NETWORK_DIR:-${PROJECT_ROOT}/networks/${CHAIN_ID}}"
ARTIFACT_DIR="${ARTIFACT_DIR:-$(mktemp -d -t paw-publish-XXXXXX)}"
KEEP_ARTIFACT_DIR="${KEEP_ARTIFACT_DIR:-0}"

cleanup() {
  if [[ "${KEEP_ARTIFACT_DIR}" != "1" ]]; then
    rm -rf "${ARTIFACT_DIR}"
  else
    printf '[publish] keeping temp dir: %s\n' "${ARTIFACT_DIR}"
  fi
}
trap cleanup EXIT

log() {
  printf '[publish] %s\n' "$*"
}

if [[ -z "${PAWD_BIN}" ]]; then
  echo "[publish] pawd binary not found (set PAWD_BIN)" >&2
  exit 1
fi

if [[ ! -x "${PACKAGE_SCRIPT}" ]]; then
  echo "[publish] package script not found or not executable: ${PACKAGE_SCRIPT}" >&2
  exit 1
fi

mkdir -p "${NETWORK_DIR}"

log "packaging artifacts for ${CHAIN_ID} (PAW_HOME=${PAW_HOME})"
PAW_CHAIN_ID="${CHAIN_ID}" PAW_HOME="${PAW_HOME}" PAWD_BIN="${PAWD_BIN}" \
  "${PACKAGE_SCRIPT}" "${ARTIFACT_DIR}" >/dev/null

GENESIS_SRC="${ARTIFACT_DIR}/${CHAIN_ID}-genesis.json"
SHA_SRC="${ARTIFACT_DIR}/${CHAIN_ID}-genesis.sha256"
PEERS_SRC="${ARTIFACT_DIR}/${CHAIN_ID}-peer-metadata.txt"
MANIFEST_SRC="${ARTIFACT_DIR}/${CHAIN_ID}-manifest.json"

if [[ ! -f "${GENESIS_SRC}" || ! -f "${SHA_SRC}" ]]; then
  echo "[publish] expected files missing in ${ARTIFACT_DIR}" >&2
  exit 1
fi

log "syncing artifacts into ${NETWORK_DIR}"
cp "${GENESIS_SRC}" "${NETWORK_DIR}/genesis.json"
cp "${SHA_SRC}" "${NETWORK_DIR}/genesis.sha256"

if [[ -f "${PEERS_SRC}" ]]; then
  if [[ -f "${NETWORK_DIR}/peers.txt" ]]; then
    cp "${NETWORK_DIR}/peers.txt" "${NETWORK_DIR}/peers.txt.bak.$(date -u +%Y%m%d%H%M%S)"
  fi
  cp "${PEERS_SRC}" "${NETWORK_DIR}/peers.txt"
  log "updated peers.txt (review before publishing)"
else
  log "no peer metadata found; skipping peers.txt"
fi

if [[ -f "${MANIFEST_SRC}" ]]; then
  cp "${MANIFEST_SRC}" "${NETWORK_DIR}/${CHAIN_ID}-manifest.json"
  # Normalize the genesis filename in the manifest to the published name (genesis.json)
  if command -v jq >/dev/null 2>&1; then
    tmp_manifest="$(mktemp)"
    jq --arg file "genesis.json" '.genesis_file = $file' "${NETWORK_DIR}/${CHAIN_ID}-manifest.json" > "${tmp_manifest}" && mv "${tmp_manifest}" "${NETWORK_DIR}/${CHAIN_ID}-manifest.json"
  fi
  log "synced manifest (${CHAIN_ID}-manifest.json)"
else
  log "no manifest found; skipping manifest sync"
fi

log "artifacts ready under ${NETWORK_DIR}"
ls -1 "${NETWORK_DIR}" | sed 's/^/  - /'

log "upload ${NETWORK_DIR}/genesis.json and genesis.sha256 to your public bucket/CDN once verified."
