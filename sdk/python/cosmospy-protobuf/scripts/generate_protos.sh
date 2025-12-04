#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/proto"
OUT_DIR="${ROOT}/cosmospy_protobuf"

python -m grpc_tools.protoc \
  --proto_path="${PROTO_DIR}" \
  --python_out="${OUT_DIR}" \
  cosmos/tx/v1beta1/tx.proto \
  cosmos/tx/signing/v1beta1/signing.proto \
  cosmos/base/v1beta1/coin.proto \
  cosmos/crypto/multisig/v1beta1/multisig.proto \
  cosmos/crypto/secp256k1/keys.proto \
  amino/amino.proto \
  cosmos_proto/cosmos.proto
