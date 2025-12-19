#!/usr/bin/env bash
# Validate published artifacts on a remote CDN/bucket by downloading and checking integrity.
# Usage:
#   ./scripts/devnet/validate-remote-artifacts.sh https://networks.paw.xyz/paw-testnet-1 [chain-id]

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <base_url> [chain-id]" >&2
  exit 1
fi

BASE_URL="${1%/}"
CHAIN_ID="${2:-${CHAIN_ID:-${PAW_CHAIN_ID:-paw-testnet-1}}}"
NETWORK_DIR="${NETWORK_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/networks/${CHAIN_ID}}"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[remote-verify] missing dependency: $1" >&2
    exit 1
  fi
}

for bin in curl sha256sum jq; do
  require "$bin"
done

TMP_DIR="$(mktemp -d -t paw-remote-verify-XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

fetch() {
  local url=$1
  local dst=$2
  curl -fsSL "${url}" -o "${dst}"
}

echo "[remote-verify] downloading artifacts from ${BASE_URL}"
fetch "${BASE_URL}/genesis.json" "${TMP_DIR}/genesis.json"
fetch "${BASE_URL}/genesis.sha256" "${TMP_DIR}/genesis.sha256"
fetch "${BASE_URL}/${CHAIN_ID}-manifest.json" "${TMP_DIR}/manifest.json"
PEERS_PRESENT=1
if ! fetch "${BASE_URL}/peers.txt" "${TMP_DIR}/peers.txt"; then
  echo "[remote-verify] warning: peers.txt not found remotely"
  PEERS_PRESENT=0
fi

expected_hash=$(awk '{print $1; exit}' "${TMP_DIR}/genesis.sha256")
actual_hash=$(sha256sum "${TMP_DIR}/genesis.json" | awk '{print $1}')
if [[ "${expected_hash}" != "${actual_hash}" ]]; then
  echo "[remote-verify] checksum mismatch (remote)"
  echo "  expected: ${expected_hash}"
  echo "  actual:   ${actual_hash}"
  exit 1
fi

manifest_chain=$(jq -r '.chain_id // empty' "${TMP_DIR}/manifest.json")
manifest_hash=$(jq -r '.genesis_sha256 // empty' "${TMP_DIR}/manifest.json")
manifest_genesis=$(jq -r '.genesis_file // empty' "${TMP_DIR}/manifest.json")
manifest_bundle=$(jq -r '.bundle_file // empty' "${TMP_DIR}/manifest.json")
manifest_bundle_sha=$(jq -r '.bundle_sha256 // empty' "${TMP_DIR}/manifest.json")
status_page_manifest=$(jq -r '.status_page_url // empty' "${TMP_DIR}/manifest.json")
if [[ -z "${manifest_chain}" || -z "${manifest_hash}" ]]; then
  echo "[remote-verify] manifest missing required fields" >&2
  exit 1
fi
if [[ "${manifest_chain}" != "${CHAIN_ID}" ]]; then
  echo "[remote-verify] manifest chain_id mismatch (manifest=${manifest_chain}, expected=${CHAIN_ID})" >&2
  exit 1
fi
if [[ "${manifest_hash}" != "${actual_hash}" ]]; then
  echo "[remote-verify] manifest genesis hash mismatch (manifest=${manifest_hash}, actual=${actual_hash})" >&2
  exit 1
fi
if [[ -n "${manifest_genesis}" && "${manifest_genesis}" != "genesis.json" ]]; then
  echo "[remote-verify] manifest genesis filename mismatch (manifest=${manifest_genesis}, expected=genesis.json)" >&2
  exit 1
fi

# Optional bundle verification
bundle_sha_file="${CHAIN_ID}-artifacts.sha256"
explicit_bundle_sha=""
if fetch "${BASE_URL}/${bundle_sha_file}" "${TMP_DIR}/${bundle_sha_file}"; then
  explicit_bundle_sha=$(awk '{print $1; exit}' "${TMP_DIR}/${bundle_sha_file}")
fi
if [[ -n "${manifest_bundle}" && ( -n "${manifest_bundle_sha}" || -n "${explicit_bundle_sha}" ) ]]; then
  bundle_url="${BASE_URL}/${manifest_bundle}"
  bundle_path="${TMP_DIR}/bundle.tar.gz"
  echo "[remote-verify] downloading bundle ${bundle_url}"
  if fetch "${bundle_url}" "${bundle_path}"; then
    bundle_hash=$(sha256sum "${bundle_path}" | awk '{print $1}')
    expected_bundle_hash="${explicit_bundle_sha:-${manifest_bundle_sha}}"
    if [[ -n "${expected_bundle_hash}" && "${bundle_hash}" != "${expected_bundle_hash}" ]]; then
      echo "[remote-verify] bundle checksum mismatch (expected=${expected_bundle_hash}, actual=${bundle_hash})" >&2
      exit 1
    fi
    echo "[remote-verify] bundle checksum OK (${bundle_hash})"
  else
    echo "[remote-verify] warning: unable to fetch bundle from ${bundle_url}"
  fi
fi

echo "[remote-verify] remote genesis checksum OK (${actual_hash})"
echo "[remote-verify] manifest validated (${manifest_chain})"

# Optionally validate peers resolve/ports are open
if [[ "${PEERS_PRESENT}" -eq 1 ]]; then
  peers=$(grep '^persistent_peers' "${TMP_DIR}/peers.txt" | cut -d'=' -f2- | tr ',' '\n' | xargs || true)
  for entry in ${peers}; do
    node_id=$(echo "${entry}" | cut -d'@' -f1)
    hostport=$(echo "${entry}" | cut -d'@' -f2)
    host=$(echo "${hostport}" | cut -d':' -f1)
    port=$(echo "${hostport}" | cut -d':' -f2)
    if [[ -z "${host}" || -z "${port}" ]]; then
      echo "[remote-verify] warning: unable to parse peer entry: ${entry}"
      continue
    fi
    if command -v dig >/dev/null 2>&1; then
      if ! dig +short "${host}" >/dev/null; then
        echo "[remote-verify] warning: peer host not resolvable (${host})"
      fi
    fi
    if command -v nc >/dev/null 2>&1; then
      if ! nc -z -w2 "${host}" "${port}" >/dev/null 2>&1; then
        echo "[remote-verify] warning: TCP check failed for ${host}:${port} (${node_id})"
      else
        echo "[remote-verify] peer reachable: ${host}:${port} (${node_id})"
      fi
    fi
  done
fi

# Optional status page probe
STATUS_TO_PROBE="${STATUS_PAGE_URL:-${status_page_manifest}}"
if [[ -n "${STATUS_TO_PROBE}" ]]; then
  if curl -fsSL "${STATUS_TO_PROBE}/api/status" -o /dev/null; then
    echo "[remote-verify] status page reachable at ${STATUS_TO_PROBE}/api/status"
  else
    echo "[remote-verify] warning: unable to reach status page ${STATUS_TO_PROBE}/api/status"
  fi
fi

# Optionally compare to local staged copy
if [[ -f "${NETWORK_DIR}/genesis.sha256" ]]; then
  local_hash=$(sha256sum "${NETWORK_DIR}/genesis.json" | awk '{print $1}')
  if [[ "${local_hash}" != "${actual_hash}" ]]; then
    echo "[remote-verify] WARNING: local staged genesis hash differs from remote (${local_hash} vs ${actual_hash})"
  fi
fi

echo "[remote-verify] success"
