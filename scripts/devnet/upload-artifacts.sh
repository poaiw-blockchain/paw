#!/usr/bin/env bash
# Upload staged paw-testnet artifacts to an object store (S3-compatible).
# Requires AWS CLI configured with credentials and ARTIFACTS_DEST set to an s3 URI.
#
# Usage:
#   ARTIFACTS_DEST=s3://my-bucket/paw-mvp-1 ./scripts/devnet/upload-artifacts.sh

set -euo pipefail

DEST="${ARTIFACTS_DEST:-}"
CHAIN_ID="${CHAIN_ID:-${PAW_CHAIN_ID:-paw-mvp-1}}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NETWORK_DIR="${PROJECT_ROOT}/networks/${CHAIN_ID}"
BUNDLE="${PROJECT_ROOT}/artifacts/${CHAIN_ID}-artifacts.tar.gz"

if [[ -z "${DEST}" ]]; then
  echo "[upload] ARTIFACTS_DEST (s3 URI) is required" >&2
  exit 1
fi

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[upload] missing dependency: $1" >&2
    exit 1
  fi
}

require aws

files=(
  "${NETWORK_DIR}/genesis.json"
  "${NETWORK_DIR}/genesis.sha256"
  "${NETWORK_DIR}/peers.txt"
  "${NETWORK_DIR}/${CHAIN_ID}-manifest.json"
  "${BUNDLE}"
)

for f in "${files[@]}"; do
  if [[ ! -f "${f}" ]]; then
    echo "[upload] missing file: ${f}" >&2
    exit 1
  fi
done

echo "[upload] uploading artifacts to ${DEST}"
aws s3 cp "${NETWORK_DIR}/genesis.json" "${DEST}/genesis.json" --acl public-read
aws s3 cp "${NETWORK_DIR}/genesis.sha256" "${DEST}/genesis.sha256" --acl public-read
aws s3 cp "${NETWORK_DIR}/peers.txt" "${DEST}/peers.txt" --acl public-read
aws s3 cp "${NETWORK_DIR}/${CHAIN_ID}-manifest.json" "${DEST}/${CHAIN_ID}-manifest.json" --acl public-read
aws s3 cp "${BUNDLE}" "${DEST}/${CHAIN_ID}-artifacts.tar.gz" --acl public-read

echo "[upload] complete. Verify via: ./scripts/devnet/validate-remote-artifacts.sh ${DEST}"
