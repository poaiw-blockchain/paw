# PAW Blockchain Quick Start Guide

## First Time Setup (5 minutes)

```bash
# 1. Run development setup
./scripts/dev-setup.sh        # Unix/Mac/Linux
.\scripts\dev-setup.ps1        # Windows PowerShell

# 2. Build the project
make build

# 3. Run tests
make test
```

## Daily Development Workflow

### Option 1: Docker Development Environment (Recommended)

```bash
# Start everything (node + monitoring)
make dev

# Access services:
# - Node RPC: http://localhost:26657
# - REST API: http://localhost:1317
# - gRPC: localhost:9090
# - Grafana: http://localhost:3000 (admin/pawdev123)

# Stop everything
make dev-down
```

### Option 2: Local Development

```bash
# Build binaries
make build

# Start local testnet
make localnet-start

# Stop testnet
make localnet-stop
```

## Common Commands

### Building

```bash
make build              # Build pawd & pawcli
make install            # Install to GOPATH/bin
make build-linux        # Cross-compile for Linux
```

### Testing

```bash
make test               # All tests with coverage
make test-unit          # Unit tests only
make test-integration   # Integration tests
make test-coverage      # Generate HTML coverage report
```

### Code Quality

```bash
make lint               # Run linter
make format             # Format Go code
make format-all         # Format all code (Go, Python, JS, etc.)
```

### Cleanup

```bash
make clean              # Remove build artifacts
make clean-all          # Deep clean (interactive)
./scripts/clean.sh      # Interactive cleanup script
```

### Protobuf

```bash
make proto-gen          # Generate Go code from .proto
make proto-format       # Format .proto files
make proto-lint         # Lint .proto files
make proto-all          # Format + Lint + Generate
```

### Release

```bash
make release-test       # Test release configuration
make release            # Build snapshot release
goreleaser release      # Full release (requires tag)
```

## Development Tools

### Install All Tools

```bash
make install-tools
```

This installs:

- golangci-lint (linting)
- goimports (import formatting)
- misspell (spell checking)
- buf (protobuf)
- goreleaser (releases)

### Manual Tool Installation

```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Import formatter
go install golang.org/x/tools/cmd/goimports@latest

# Spell checker
go install github.com/client9/misspell/cmd/misspell@latest

# Protobuf
go install github.com/bufbuild/buf/cmd/buf@latest

# Release tool
go install github.com/goreleaser/goreleaser@latest
```

## Project Structure

```
paw/
├── cmd/
│   ├── pawd/           # Node daemon
│   └── pawcli/         # CLI tool
├── x/                  # Custom modules
│   ├── dex/            # DEX module
│   ├── compute/        # Compute module
│   └── oracle/         # Oracle module
├── scripts/            # Build & dev scripts
├── infra/              # Infrastructure configs
├── docs/               # Documentation
└── tests/              # E2E tests
```

## Useful File Locations

```
Config Files:
├── .goreleaser.yml           # Release configuration
├── renovate.json             # Dependency updates
├── docker-compose.dev.yml    # Dev environment
├── .dockerignore             # Docker build excludes
├── .gitattributes            # Git line endings
├── Makefile                  # Build commands
└── VERSION                   # Current version

Scripts:
├── scripts/dev-setup.sh      # Development setup
├── scripts/clean.sh          # Cleanup script
├── scripts/format-all.sh     # Code formatter
├── scripts/bootstrap-node.sh # Node initialization
└── scripts/localnet-start.sh # Start testnet

Infrastructure:
├── infra/node-config.yaml    # Node configuration
├── infra/prometheus.yml      # Metrics config
└── infra/grafana/            # Grafana dashboards
```

## Port Reference

### PAW Node 1

- **26657**: Tendermint RPC
- **1317**: REST API
- **9090**: gRPC
- **26660**: Prometheus metrics
- **26656**: P2P

### PAW Node 2 (Docker only)

- **26667**: Tendermint RPC
- **1327**: REST API
- **9091**: gRPC

### Services (Docker only)

- **5432**: PostgreSQL
- **6379**: Redis
- **9090**: Prometheus
- **3000**: Grafana

## Environment Variables

```bash
# Build options
export LEDGER_ENABLED=true    # Enable ledger support
export BUILD_TAGS="custom"    # Custom build tags
export LDFLAGS="-custom"      # Custom linker flags

# Development
export CHAIN_ID=paw-dev-1     # Chain ID
export MONIKER=my-node        # Node name
export LOG_LEVEL=info         # Log level

# API
export ENABLE_API=true        # Enable REST API
export ENABLE_GRPC=true       # Enable gRPC
export ENABLE_SWAGGER=true    # Enable Swagger docs
```

## Troubleshooting

### Build Fails

```bash
# Clean and rebuild
make clean
go mod download
make build
```

### Tests Fail

```bash
# Clear test cache
go clean -testcache
make test
```

### Docker Issues

```bash
# Rebuild containers
docker-compose -f docker-compose.dev.yml build --no-cache

# Clean volumes
docker-compose -f docker-compose.dev.yml down -v
```

### Script Permission Denied (Unix)

```bash
chmod +x scripts/*.sh
```

### PowerShell Execution Policy (Windows)

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Getting Help

1. Check documentation:
   - `TOOLS_SETUP.md` - Detailed tool documentation
   - `README.md` - Project overview
   - `PAW Extensive whitepaper .md` - Technical specifications

2. Run help commands:

   ```bash
   pawd --help
   pawcli --help
   make help  # (if implemented)
   ```

3. Check logs:

   ```bash
   # Docker logs
   docker-compose -f docker-compose.dev.yml logs -f paw-node

   # Local logs
   tail -f ~/.paw/logs/node.log
   ```

## Next Steps

1. **Read the whitepaper**: `PAW Extensive whitepaper .md`
2. **Explore modules**: Check `x/` directory
3. **Run local testnet**: `make localnet-start`
4. **Build wallet**: Review `wallet/` directory
5. **Check integration docs**: `docs/INTEGRATION_GUIDE.md`

## Useful Commands Cheat Sheet

```bash
# Setup & Build
make dev-setup          # First time setup
make install-tools      # Install development tools
make build              # Build binaries
make install            # Install binaries

# Development
make dev                # Start Docker dev environment
make localnet-start     # Start local testnet
make test               # Run all tests
make lint               # Run linter
make format-all         # Format all code

# Cleanup
make clean              # Basic cleanup
make clean-all          # Deep cleanup

# Release
make release-test       # Test release config
make release            # Build release snapshot

# Protobuf
make proto-all          # Format, lint, and generate

# Docker
docker-compose -f docker-compose.dev.yml up      # Start
docker-compose -f docker-compose.dev.yml down    # Stop
docker-compose -f docker-compose.dev.yml logs -f # Logs
```

---

**Version**: 0.1.0-alpha
**Last Updated**: 2025-11-12
