#!/usr/bin/env bash
set -euo pipefail

# Lightweight test node runner (no real consensus engine yet).
#
BASE="$(pwd)/infra"
DATA="$BASE/node/data"
CONFIG="$BASE/node-config.yaml"

if [[ ! -f "$BASE/node-config.yaml" ]]; then
  echo "Missing node-config.yaml. Run scripts/bootstrap-node.sh first."
  exit 1
fi

echo "Starting PAW test node (placeholder)"
echo "Config: $CONFIG"
echo "Data dir: $DATA"

cat <<'INFO'
This script simulates starting the controller node by printing the genesis snapshot.
To run a full validator, replace this block with your Goose (Go/Golang) or Rust binary invocation
that reads from infra/genesis.json, listens on port 26656, and serves the REST/API endpoints.
INFO

jq . "$BASE/genesis.json"
