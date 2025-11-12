#!/usr/bin/env bash

set -eo pipefail

echo "Generating protobuf files..."

proto_dirs=$(find ./proto -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  echo "Generating protos from $dir"
  buf protoc \
    -I "proto" \
    -I "third_party/proto" \
    --gocosmos_out=plugins=interfacetype+grpc,Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:. \
    $(find "${dir}" -maxdepth 1 -name '*.proto')
done

# Generate Pulsar protos
echo "Generating Pulsar protos..."
buf generate --template proto/buf.gen.pulsar.yaml

# Move generated files to the right places
cp -r github.com/paw-chain/paw/* ./
rm -rf github.com

echo "Protobuf generation complete!"
