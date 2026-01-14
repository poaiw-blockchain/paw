#!/usr/bin/env bash
# Deterministic testnet bootstrapper for PAW.
# - Builds pawd (if not present)
# - Initializes the home directory
# - Copies hardened config templates
# - Patches seeds/persistent peers/gas/pruning to known defaults
# Default home: ${PAW_HOME:-$HOME/.paw}

set -euo pipefail
if [[ "${TRACE:-0}" == "1" ]]; then
  set -x
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

PAW_HOME="${PAW_HOME:-${HOME}/.paw}"
PAW_CHAIN_ID="${PAW_CHAIN_ID:-paw-mvp-1}"
MONIKER="${MONIKER:-${1:-paw-testnet-node}}"
BUILD_DIR="${BUILD_DIR:-${REPO_ROOT}/build}"
PAWD_BIN="${PAWD_BIN:-${BUILD_DIR}/pawd}"
SEEDS="${SEEDS:-}"
PERSISTENT_PEERS="${PERSISTENT_PEERS:-}"
EXTERNAL_ADDRESS="${EXTERNAL_ADDRESS:-}"
MIN_GAS_PRICES="${MIN_GAS_PRICES:-0.001upaw}"
PRUNING_KEEP_RECENT="${PRUNING_KEEP_RECENT:-500000}"
PRUNING_INTERVAL="${PRUNING_INTERVAL:-10}"
MIN_RETAIN_BLOCKS="${MIN_RETAIN_BLOCKS:-500000}"
KEYRING_BACKEND="${KEYRING_BACKEND:-os}"
OVERWRITE="${OVERWRITE:-false}"

CONFIG_DIR="${PAW_HOME}/config"
CONFIG_TOML="${CONFIG_DIR}/config.toml"
APP_TOML="${CONFIG_DIR}/app.toml"

log() { printf "\033[1;34m[*]\033[0m %s\n" "$*"; }
fatal() { printf "\033[1;31m[!]\033[0m %s\n" "$*\n" >&2; exit 1; }

log "Using PAW_HOME=${PAW_HOME}, chain-id=${PAW_CHAIN_ID}, moniker=${MONIKER}"

if [[ -f "${CONFIG_DIR}/genesis.json" && "${OVERWRITE}" != "true" ]]; then
  fatal "Genesis already exists at ${CONFIG_DIR}/genesis.json (set OVERWRITE=true to recreate)"
fi

mkdir -p "${BUILD_DIR}"
if [[ ! -x "${PAWD_BIN}" ]]; then
  log "Building pawd -> ${PAWD_BIN}"
  (cd "${REPO_ROOT}" && go build -o "${PAWD_BIN}" ./cmd/...)
fi

mkdir -p "${PAW_HOME}"

init_args=(init "${MONIKER}" --chain-id "${PAW_CHAIN_ID}" --home "${PAW_HOME}")
if [[ "${OVERWRITE}" == "true" ]]; then
  init_args+=(--overwrite)
fi

log "Initializing node (${PAWD_BIN} ${init_args[*]})"
"${PAWD_BIN}" "${init_args[@]}"

mkdir -p "${CONFIG_DIR}"

for template in node-config.toml.template app.toml.template; do
  src="${REPO_ROOT}/config/${template}"
  dest="${CONFIG_DIR}/$(basename "${template/.template/}")"
  if [[ -f "${dest}" && "${OVERWRITE}" != "true" ]]; then
    log "Skipping copy (existing): ${dest}"
  else
    log "Copying ${src} -> ${dest}"
    cp "${src}" "${dest}"
  fi
done

log "Patching ${CONFIG_TOML}"
perl -0pi -e "s/^moniker = \".*\"/moniker = \"${MONIKER}\"/m" "${CONFIG_TOML}"
perl -0pi -e "s/^seeds = \".*\"/seeds = \"${SEEDS}\"/m" "${CONFIG_TOML}"
perl -0pi -e "s/^persistent_peers = \".*\"/persistent_peers = \"${PERSISTENT_PEERS}\"/m" "${CONFIG_TOML}"
perl -0pi -e "s/^external_address = \".*\"/external_address = \"${EXTERNAL_ADDRESS}\"/m" "${CONFIG_TOML}"

log "Patching ${APP_TOML}"
perl -0pi -e "s/^minimum-gas-prices = \".*\"/minimum-gas-prices = \"${MIN_GAS_PRICES}\"/m" "${APP_TOML}"
perl -0pi -e "s/^pruning-keep-recent = \".*\"/pruning-keep-recent = \"${PRUNING_KEEP_RECENT}\"/m" "${APP_TOML}"
perl -0pi -e "s/^pruning-interval = \".*\"/pruning-interval = \"${PRUNING_INTERVAL}\"/m" "${APP_TOML}"
perl -0pi -e "s/^min-retain-blocks = .*/min-retain-blocks = ${MIN_RETAIN_BLOCKS}/m" "${APP_TOML}"

log "Setting client keyring backend to ${KEYRING_BACKEND}"
"${PAWD_BIN}" config keyring-backend "${KEYRING_BACKEND}" --home "${PAW_HOME}"

log "Init complete."
cat <<EOF
Home:        ${PAW_HOME}
Chain ID:    ${PAW_CHAIN_ID}
Moniker:     ${MONIKER}
Seeds:       ${SEEDS:-<empty>}
Peers:       ${PERSISTENT_PEERS:-<empty>}
Gas price:   ${MIN_GAS_PRICES}
Pruning:     keep ${PRUNING_KEEP_RECENT}, interval ${PRUNING_INTERVAL}, min-retain ${MIN_RETAIN_BLOCKS}
Keyring:     ${KEYRING_BACKEND} (stored under ${PAW_HOME}/keyring-${KEYRING_BACKEND})

Next:
  ${PAWD_BIN} start --home "${PAW_HOME}"
EOF
