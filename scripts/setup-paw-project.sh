#!/bin/bash
# PAW Blockchain Project Setup Script
# Cosmos SDK-based blockchain
# Run after: bash ~/blockchain-projects/setup-dependencies.sh

set -e

echo "=== PAW Blockchain Project Setup ==="
echo ""

PROJECT_DIR="$HOME/blockchain-projects/paw"
cd "$PROJECT_DIR"

# Ensure Go is in PATH
export PATH=$HOME/go-sdk/bin:$PATH
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH

echo "[1/7] Verifying Go installation..."
go version

echo "[2/7] Downloading Go module dependencies..."
go mod download

echo "[3/7] Tidying Go modules..."
go mod tidy

echo "[4/7] Installing protobuf dependencies..."
go install example.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest
go install example.com/cosmos/cosmos-sdk/proto/cosmos/proto-codec-descriptors@latest || true
go install example.com/cosmos/gogoproto/protoc-gen-gogo@latest
go install example.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@latest
go install example.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo "[5/7] Building the blockchain binary..."
make build 2>/dev/null || make install

echo "[6/7] Generating protobuf files..."
make proto-gen 2>/dev/null || echo "Skipping proto-gen (may require additional setup)"

echo "[7/7] Verifying installation..."
if [ -f "$GOPATH/bin/pawd" ]; then
    echo "✓ pawd binary installed at: $GOPATH/bin/pawd"
    $GOPATH/bin/pawd version 2>/dev/null || echo "Binary built successfully"
elif [ -f "./build/pawd" ]; then
    echo "✓ pawd binary built at: ./build/pawd"
    ./build/pawd version 2>/dev/null || echo "Binary built successfully"
else
    echo "⚠ Binary location check - run 'make install' manually if needed"
fi

echo ""
echo "=== PAW Blockchain Setup Complete! ==="
echo ""
echo "Project: PAW - Cosmos SDK Blockchain"
echo "Location: ~/blockchain-projects/paw"
echo "Framework: Cosmos SDK v0.50.11"
echo "Smart Contracts: CosmWasm v0.54.0"
echo ""
echo "Common commands:"
echo "  make build          # Build the blockchain binary"
echo "  make install        # Install pawd to \$GOPATH/bin"
echo "  make proto-gen      # Generate protobuf files"
echo "  make test           # Run tests"
echo "  make start          # Start local node"
echo ""
echo "Initialize a local testnet:"
echo "  pawd init my-node --chain-id paw-local-1"
echo ""
