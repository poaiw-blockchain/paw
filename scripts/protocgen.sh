#!/usr/bin/env bash

set -eo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "Generating protobuf files..."
export PATH="$(go env GOPATH)/bin:${HOME}/go/bin:${PATH}"
cd "${ROOT_DIR}/proto"
buf generate --template buf.gen.gocosmos.yaml

echo "Generating Pulsar protos..."
buf generate --template buf.gen.pulsar.yaml

cd "${ROOT_DIR}"

# Move generated files into module paths
if [ -d "${ROOT_DIR}/github.com/paw-chain/paw" ]; then
  rsync -a "${ROOT_DIR}/github.com/paw-chain/paw/" "${ROOT_DIR}/"
  rm -rf "${ROOT_DIR}/github.com"
fi

if [ -d "${ROOT_DIR}/paw" ]; then
  mkdir -p "${ROOT_DIR}/x/dex/types" "${ROOT_DIR}/x/oracle/types" "${ROOT_DIR}/x/compute/types"
  rsync -a "${ROOT_DIR}/paw/dex/v1/" "${ROOT_DIR}/x/dex/types/"
  rsync -a "${ROOT_DIR}/paw/oracle/v1/" "${ROOT_DIR}/x/oracle/types/"
  rsync -a "${ROOT_DIR}/paw/compute/v1/" "${ROOT_DIR}/x/compute/types/"
  rm -rf "${ROOT_DIR}/paw"
fi

python3 - <<'PY'
import pathlib

TAG = "//go:build pulsar\n// +build pulsar\n\n"
for path in pathlib.Path("x").rglob("*.pulsar.go"):
    text = path.read_text()
    if text.startswith("//go:build pulsar"):
        continue
    path.write_text(TAG + text)

GATEWAY_TAG = "//go:build gateway\n// +build gateway\n\n"
for path in pathlib.Path("x").rglob("*.pb.gw.go"):
    text = path.read_text()
    if text.startswith("//go:build gateway"):
        continue
    path.write_text(GATEWAY_TAG + text)
PY

echo "Protobuf generation complete!"
