# Go 1.24.10 Installation Summary

## What Was Done

1. **Downloaded Go 1.24.10** to meet the project requirement in `go.mod`
2. **Installed to**: `$HOME/go-installs/go/`
3. **Configured environment** in multiple ways for convenience

## Installation Verification

Go 1.24.10 is successfully installed and working:

```bash
$ go version
go version go1.24.10 linux/amd64
```

## How to Use

### Quick Setup (Recommended for Development)

Before running any go commands, source the setup script:

```bash
source ./setup-go.sh
```

This configures your current shell with:
- `GOROOT=/home/decri/go-installs/go`
- `PATH` includes Go binary directory
- `GOPATH=/home/decri/go`
- `GO111MODULE=on`

### Persistent Configuration

The environment is configured in `~/.bashrc`, so **new terminal sessions** will automatically have Go 1.24.10 available.

For the current session, either:
1. Open a new terminal, OR
2. Run: `source ./setup-go.sh`

### Using direnv (Optional)

If you have [direnv](https://direnv.net/) installed:

```bash
direnv allow .
```

The `.envrc` file will automatically configure Go when you enter the PAW directory.

## Testing the Installation

Run tests to verify everything works:

```bash
# Source the environment first
source ./setup-go.sh

# Run all tests
go test ./...

# Run specific module tests
go test ./p2p/...
go test ./x/oracle/...

# Run with coverage
go test -cover ./...
```

## Building the PAW Daemon

```bash
# Source the environment
source ./setup-go.sh

# Build
go build -o pawd ./cmd/...

# Verify
./pawd version
```

## Files Created

- `setup-go.sh` - Quick setup script for development shells
- `.envrc` - Environment configuration for direnv users
- `README-GO-SETUP.md` - Detailed documentation
- `GO-INSTALLATION.md` - This summary file

## Troubleshooting

**Problem**: "go: command not found"
**Solution**: Run `source ./setup-go.sh` in your current shell

**Problem**: "package X is not in std"
**Solution**: GOROOT is not set. Run `source ./setup-go.sh`

**Problem**: Tests fail to compile
**Solution**: Ensure you've sourced the setup script and GOROOT is correct:
```bash
source ./setup-go.sh
go env GOROOT  # Should show: /home/decri/go-installs/go
```

## Environment Variables

| Variable | Value | Purpose |
|----------|-------|---------|
| GOROOT | `/home/decri/go-installs/go` | Where Go is installed |
| GOPATH | `/home/decri/go` | Go workspace |
| PATH | Includes `$GOROOT/bin` | Makes `go` command available |
| GO111MODULE | `on` | Enable Go modules |

## Why Go 1.24.10?

The `go.mod` file specifies:
```
go 1.24.0
toolchain go1.24.10
```

This version is required for the Cosmos SDK dependencies and modern Go features used in the PAW blockchain.
