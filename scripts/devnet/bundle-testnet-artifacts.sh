#!/usr/bin/env bash
# Create a distribution-ready tarball for public artifact publishing.
# Includes: genesis.json, genesis.sha256, peers.txt (or peers.example.txt), manifest.
#
# Usage:
#   CHAIN_ID=paw-testnet-1 ./scripts/devnet/bundle-testnet-artifacts.sh [/path/to/output-dir]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CHAIN_ID="${CHAIN_ID:-${PAW_CHAIN_ID:-paw-testnet-1}}"
NETWORK_DIR="${NETWORK_DIR:-${PROJECT_ROOT}/networks/${CHAIN_ID}}"
OUTPUT_DIR="${1:-${PROJECT_ROOT}/artifacts}"
OUTPUT_FILE="${OUTPUT_DIR}/${CHAIN_ID}-artifacts.tar.gz"

log() { printf '[bundle] %s\n' "$*"; }

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[bundle] missing dependency: $1" >&2
    exit 1
  fi
}

for bin in tar sha256sum; do
  require "$bin"
done

GENESIS="${NETWORK_DIR}/genesis.json"
SHA_FILE="${NETWORK_DIR}/genesis.sha256"
PEERS_FILE="${NETWORK_DIR}/peers.txt"
MANIFEST_FILE="${NETWORK_DIR}/${CHAIN_ID}-manifest.json"

if [[ ! -f "${GENESIS}" || ! -f "${SHA_FILE}" ]]; then
  echo "[bundle] missing genesis or checksum in ${NETWORK_DIR}" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
TMP_DIR="$(mktemp -d -t paw-bundle-XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

cp "${GENESIS}" "${TMP_DIR}/"
cp "${SHA_FILE}" "${TMP_DIR}/"
if [[ -f "${PEERS_FILE}" ]]; then
  cp "${PEERS_FILE}" "${TMP_DIR}/"
elif [[ -f "${NETWORK_DIR}/peers.example.txt" ]]; then
  cp "${NETWORK_DIR}/peers.example.txt" "${TMP_DIR}/"
fi
tar -C "${TMP_DIR}" -czf "${OUTPUT_FILE}" .

log "bundle created: ${OUTPUT_FILE}"
log "contents:"
tar -tzf "${OUTPUT_FILE}" | sed 's/^/  - /'
