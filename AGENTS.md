# PAW Blockchain Project Guidelines

**Read the parent guidelines first:** `../CLAUDE.md` and `../AGENTS.md` contain general agent instructions that apply to all projects.

This file contains PAW-specific conventions and instructions.

---

## Project Overview

PAW is a Cosmos SDK-based blockchain focused on P2P networking, IBC (Inter-Blockchain Communication), and decentralized exchange functionality.

## Project Structure

```
paw/
├── app/                # Application wiring and ante handlers
├── cmd/                # CLI daemon entry point
├── x/                  # Custom modules (if any)
├── p2p/                # P2P networking code
├── ibc/                # IBC integration
├── proto/              # Protobuf definitions
├── tests/              # Test suites
├── scripts/            # Build and utility scripts
├── docs/               # Documentation
├── infra/              # Infrastructure configs
└── k8s/                # Kubernetes manifests
```

## Node Data Directory

**The node data directory is `~/.paw/` (in user's home directory), NOT in the repo.**

This is the Cosmos SDK convention. The `~/.paw/` directory contains:
- `config/` - Node configuration (app.toml, config.toml, genesis.json)
- `data/` - Blockchain state database
- `keyring-*/` - Wallet keys
- Private validator and node keys

**Do NOT:**
- Put blockchain data in the repo
- Commit private keys or node identity files
- Change the default data directory without documenting it

**To initialize a fresh node:**
```bash
go build -o pawd ./cmd/...
./pawd init <moniker> --chain-id paw-testnet-1
```

**To specify a custom data directory (if needed):**
```bash
./pawd init <moniker> --home /custom/path
# or
export PAW_HOME=/custom/path
```

## Building

```bash
# Build the daemon
go build -o pawd ./cmd/...

# Optimized build (smaller binary):
go build -ldflags="-s -w" -o pawd ./cmd/...

# Using Makefile (if available):
make build
```

**Note:** Compiled binaries are excluded from git. Always rebuild from source.

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./p2p/...
go test ./app/...

# Run with race detection
go test -race ./...
```

**Pre-commit hooks are configured.** Run `pre-commit install` to enable them.

## Protobuf Generation

When modifying `.proto` files:
```bash
make proto-gen
# or
buf generate
```

## Module Development

When adding or modifying modules:
1. Update protobuf definitions in `proto/`
2. Regenerate protobuf files
3. Implement keeper methods
4. Add genesis import/export
5. Register in `app/app.go`
6. Write comprehensive tests
7. Update documentation

## Git Workflow

- Commit frequently after completing each task
- Push to GitHub after each commit (SSH is configured - no auth prompts)
- Use clear commit messages
- GitHub Actions are DISABLED (local testing only via pre-commit hooks)

**SSH Authentication:** Remote is `git@github.com:decristofaroj/paw.git`. SSH key: `~/.ssh/id_ed25519_github`.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PAW_HOME` | Node data directory | `~/.paw` |
| `PAW_CHAIN_ID` | Chain identifier | - |

## Docker

```bash
# Build image
docker build -t paw-node .

# Run with docker-compose
docker-compose up -d
```

## Common Issues

**"pawd: command not found"**
- Build the binary first: `go build -o pawd ./cmd/...`

**"genesis.json not found"**
- Initialize the node: `./pawd init <moniker>`

**Go module issues**
- Run `go mod tidy` to clean up dependencies
