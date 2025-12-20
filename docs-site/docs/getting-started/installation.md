# Installation

Install the PAW blockchain daemon on your system.

## Prerequisites

- **Go 1.23+**: [Download Go](https://golang.org/dl/)
- **Git**: Version control
- **Make**: Build tool (usually pre-installed on Unix systems)
- **buf CLI**: Protocol buffer compiler (optional, for development)

## System Requirements

### Minimum
- CPU: 2 cores
- RAM: 4 GB
- Storage: 100 GB SSD
- Network: 10 Mbps

### Recommended
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 500 GB NVMe SSD
- Network: 100 Mbps

## Installation Methods

### Option 1: Pre-built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux x86_64
wget https://github.com/poaiw-blockchain/paw/releases/latest/download/pawd-linux-amd64
chmod +x pawd-linux-amd64
sudo mv pawd-linux-amd64 /usr/local/bin/pawd

# macOS (Intel)
wget https://github.com/poaiw-blockchain/paw/releases/latest/download/pawd-darwin-amd64
chmod +x pawd-darwin-amd64
sudo mv pawd-darwin-amd64 /usr/local/bin/pawd

# macOS (Apple Silicon)
wget https://github.com/poaiw-blockchain/paw/releases/latest/download/pawd-darwin-arm64
chmod +x pawd-darwin-arm64
sudo mv pawd-darwin-arm64 /usr/local/bin/pawd
```

Verify installation:
```bash
pawd version
```

### Option 2: Build from Source

Clone the repository and build:

```bash
# Clone repository
git clone https://github.com/poaiw-blockchain/paw.git
cd paw

# Install Go dependencies
go mod download

# Build binary
make build

# Install to system
sudo cp build/pawd /usr/local/bin/

# Verify
pawd version
```

### Option 3: Install with Go

```bash
go install github.com/poaiw-blockchain/paw/cmd/pawd@latest
```

This installs to `$GOPATH/bin/pawd` (usually `~/go/bin/pawd`). Make sure `$GOPATH/bin` is in your `PATH`.

## Docker Installation

Run PAW in a container:

```bash
docker pull ghcr.io/poaiw-blockchain/paw:latest

# Run single node
docker run -d \
  --name paw-node \
  -p 26657:26657 \
  -p 1317:1317 \
  -p 9090:9090 \
  ghcr.io/poaiw-blockchain/paw:latest
```

## Post-Installation Setup

### Initialize Node

```bash
# Initialize node with a moniker (your node's name)
pawd init my-node --chain-id paw-testnet-1

# This creates the default home directory at ~/.paw
```

### Configure Environment Variables

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
export PAW_HOME="$HOME/.paw"
export PAW_NETWORK="testnet"
export PATH="$PATH:/usr/local/bin"
```

Apply changes:
```bash
source ~/.bashrc  # or ~/.zshrc
```

## Verification

Check that everything is installed correctly:

```bash
# Check binary version
pawd version

# Check home directory was created
ls -la ~/.paw

# Test command help
pawd --help
```

Expected output:
```
PAW Blockchain CLI

Usage:
  pawd [command]

Available Commands:
  init        Initialize private validator, p2p, genesis, and application configuration files
  start       Run the full node
  ...
```

## Troubleshooting

### "command not found: pawd"

Your binary is not in PATH. Either:
- Move binary to `/usr/local/bin/`: `sudo mv pawd /usr/local/bin/`
- Or add install location to PATH: `export PATH=$PATH:/path/to/pawd`

### "permission denied"

Make the binary executable:
```bash
chmod +x pawd
```

### Go Build Errors

Ensure you have Go 1.23+:
```bash
go version
```

If version is too old, update Go from [golang.org/dl/](https://golang.org/dl/).

### Port Conflicts

Default ports:
- RPC: 26657
- API: 1317
- gRPC: 9090

If ports are in use, configure custom ports in `~/.paw/config/config.toml` and `~/.paw/config/app.toml`.

## Next Steps

Now that PAW is installed:
- [Quick Start Guide](quick-start.md) - Run your first node
- [Join Testnet](quick-start.md#join-testnet) - Connect to the live testnet
- [DEX Integration](../developers/dex-integration.md) - Start trading
