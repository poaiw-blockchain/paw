#!/usr/bin/env bash
# Package public testnet artifacts (genesis, checksums, peer metadata) for distribution.
# Usage:
#   ./scripts/devnet/package-testnet-artifacts.sh [/path/to/output]
# The script never copies private keys; it only exports public data needed by validators.

set -euo pipefail

PAW_HOME="${PAW_HOME:-$HOME/.paw}"
CHAIN_ID="${PAW_CHAIN_ID:-}"
OUTPUT_DIR="${1:-${PWD}/artifacts/paw-testnet-1}"
PAWD_BIN="${PAWD_BIN:-$(command -v pawd)}"

if [[ -z "${PAWD_BIN}" ]]; then
  echo "[package] pawd binary not found (set PAWD_BIN)" >&2
  exit 1
fi

for bin in jq sha256sum; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "[package] missing dependency: ${bin}" >&2
    exit 1
  fi
done

GENESIS_PATH="${PAW_HOME}/config/genesis.json"
CONFIG_TOML="${PAW_HOME}/config/config.toml"

if [[ ! -f "${GENESIS_PATH}" ]]; then
  echo "[package] genesis not found at ${GENESIS_PATH} (set PAW_HOME)" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"

if [[ -z "${CHAIN_ID}" ]]; then
  CHAIN_ID="$("${PAWD_BIN}" status --home "${PAW_HOME}" 2>/dev/null | jq -r '.NodeInfo.network // empty')"
fi
CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"

GENESIS_TARGET="${OUTPUT_DIR}/${CHAIN_ID}-genesis.json"
cp "${GENESIS_PATH}" "${GENESIS_TARGET}"

GENESIS_SHA=$(cd "${OUTPUT_DIR}" && sha256sum "$(basename "${GENESIS_TARGET}")" | tee "${CHAIN_ID}-genesis.sha256" | awk '{print $1}')
PAWD_SHA=$(sha256sum "${PAWD_BIN}" | awk '{print $1}')

NODE_ID="$("${PAWD_BIN}" tendermint show-node-id --home "${PAW_HOME}")"
# Prefer the P2P laddr from config.toml; fall back to the first laddr entry.
LISTEN_ADDR=$(python3 - "${CONFIG_TOML}" <<'PY'
import re, sys
cfg = open(sys.argv[1], "r", encoding="utf-8").read().splitlines()
p2p = None
in_p2p = False
fallback = None
for line in cfg:
    if re.match(r'^\s*\[p2p\]', line):
        in_p2p = True
        continue
    if re.match(r'^\s*\[.*\]', line):
        if in_p2p:
            break
        in_p2p = False
    m = re.match(r'^\s*laddr\s*=\s*"tcp://([^"]+)"', line)
    if m:
        if fallback is None:
            fallback = m.group(1)
        if in_p2p and p2p is None:
            p2p = m.group(1)
if not p2p:
    p2p = fallback or "0.0.0.0:26656"
print(p2p)
PY
)
PERSISTENT_PEERS=$(grep '^persistent_peers = ' "${CONFIG_TOML}" | head -n1 | cut -d'"' -f2)
SEEDS=$(grep '^seeds = ' "${CONFIG_TOML}" | head -n1 | cut -d'"' -f2)

# Allow operator overrides for distribution metadata
LISTEN_ADDR="${LISTEN_ADDR_OVERRIDE:-${LISTEN_ADDR}}"
PERSISTENT_PEERS="${PERSISTENT_PEERS_OVERRIDE:-${PERSISTENT_PEERS}}"
SEEDS="${SEEDS_OVERRIDE:-${SEEDS}}"

cat > "${OUTPUT_DIR}/${CHAIN_ID}-peer-metadata.txt" <<EOF
node_id=${NODE_ID}
listen_addr=${LISTEN_ADDR}
persistent_peers=${PERSISTENT_PEERS}
seeds=${SEEDS}
generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

cat > "${OUTPUT_DIR}/${CHAIN_ID}-manifest.json" <<EOF
{
  "chain_id": "${CHAIN_ID}",
  "genesis_file": "$(basename "${GENESIS_TARGET}")",
  "genesis_sha256": "${GENESIS_SHA}",
  "peers_file": "${CHAIN_ID}-peer-metadata.txt",
  "node_id": "${NODE_ID}",
  "listen_addr": "${LISTEN_ADDR}",
  "persistent_peers": "${PERSISTENT_PEERS}",
  "seeds": "${SEEDS}",
  "pawd_sha256": "${PAWD_SHA}",
  "generated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

cat <<EOF
[package] artifacts created under ${OUTPUT_DIR}
  - $(basename "${GENESIS_TARGET}")
  - ${CHAIN_ID}-genesis.sha256
  - ${CHAIN_ID}-peer-metadata.txt
  - ${CHAIN_ID}-manifest.json

Upload these files to your distribution channel (e.g., networks repo, S3, website)
and share the peer metadata so validators know which seeds/persistent peers to use.
EOF
