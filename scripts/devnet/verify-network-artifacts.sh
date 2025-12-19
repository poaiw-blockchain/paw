#!/usr/bin/env bash
# Verify that networks/<chain-id>/ contains a matching genesis + checksum.
# Usage:
#   ./scripts/devnet/verify-network-artifacts.sh [chain-id]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CHAIN_ID="${1:-${CHAIN_ID:-${PAW_CHAIN_ID:-paw-testnet-1}}}"
NETWORK_DIR="${NETWORK_DIR:-${PROJECT_ROOT}/networks/${CHAIN_ID}}"
GENESIS="${NETWORK_DIR}/genesis.json"
SHA_FILE="${NETWORK_DIR}/genesis.sha256"
MANIFEST_FILE="${NETWORK_DIR}/${CHAIN_ID}-manifest.json"

require_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[verify] missing dependency: $1" >&2
    exit 1
  fi
}

for bin in sha256sum jq; do
  require_bin "$bin"
done

if [[ ! -f "${GENESIS}" ]]; then
  echo "[verify] missing ${GENESIS}" >&2
  exit 1
fi

if [[ ! -f "${SHA_FILE}" ]]; then
  echo "[verify] missing ${SHA_FILE}" >&2
  exit 1
fi

expected_hash=$(awk '{print $1; exit}' "${SHA_FILE}")
actual_hash=$(sha256sum "${GENESIS}" | awk '{print $1}')

if [[ "${expected_hash}" != "${actual_hash}" ]]; then
  echo "[verify] checksum mismatch!"
  echo "  expected: ${expected_hash}"
  echo "  actual:   ${actual_hash}"
  exit 1
fi

genesis_chain_id=$(jq -r '.chain_id // empty' "${GENESIS}")
if [[ -z "${genesis_chain_id}" ]]; then
  echo "[verify] chain_id missing from genesis" >&2
  exit 1
fi

if [[ "${genesis_chain_id}" != "${CHAIN_ID}" ]]; then
  echo "[verify] chain_id mismatch (genesis=${genesis_chain_id}, expected=${CHAIN_ID})" >&2
  exit 1
fi

echo "[verify] ${CHAIN_ID} genesis checksum OK (${actual_hash})"
echo "[verify] chain_id matches (${genesis_chain_id})"

PEERS_FILE="${NETWORK_DIR}/peers.txt"
if [[ -f "${PEERS_FILE}" ]]; then
  seeds=$(grep '^seeds' "${PEERS_FILE}" | cut -d'=' -f2- | xargs || true)
  peers=$(grep '^persistent_peers' "${PEERS_FILE}" | cut -d'=' -f2- | xargs || true)
  echo "[verify] seeds: ${seeds:-<none>}"
  echo "[verify] persistent_peers: ${peers:-<none>}"
else
  echo "[verify] peers.txt not found (optional)"
fi

if [[ -f "${MANIFEST_FILE}" ]]; then
  manifest_chain=$(jq -r '.chain_id // empty' "${MANIFEST_FILE}")
  manifest_hash=$(jq -r '.genesis_sha256 // empty' "${MANIFEST_FILE}")
  manifest_genesis=$(jq -r '.genesis_file // empty' "${MANIFEST_FILE}")
  if [[ -z "${manifest_chain}" || -z "${manifest_hash}" ]]; then
    echo "[verify] manifest present but missing required fields" >&2
    exit 1
  fi
  if [[ "${manifest_chain}" != "${CHAIN_ID}" ]]; then
    echo "[verify] manifest chain_id mismatch (manifest=${manifest_chain}, expected=${CHAIN_ID})" >&2
    exit 1
  fi
  if [[ "${manifest_hash}" != "${actual_hash}" ]]; then
    echo "[verify] manifest genesis_sha256 mismatch (manifest=${manifest_hash}, actual=${actual_hash})" >&2
    exit 1
  fi
  if [[ -n "${manifest_genesis}" && "${manifest_genesis}" != "$(basename "${GENESIS}")" ]]; then
    echo "[verify] manifest genesis_file mismatch (manifest=${manifest_genesis}, expected=$(basename "${GENESIS}"))" >&2
    exit 1
  fi
  echo "[verify] manifest validated (${MANIFEST_FILE})"
else
  echo "[verify] manifest not found (optional but recommended)"
fi

echo "[verify] artifacts in ${NETWORK_DIR} look good."
