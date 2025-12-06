# Go 1.24.10 Setup for PAW

## Overview

The PAW blockchain project requires **Go 1.24.10** for development and testing, as specified in `go.mod`.

## Installation Location

Go 1.24.10 has been installed to:
```
$HOME/go-installs/go
```

## Quick Start

### Option 1: Source the setup script (Recommended)
```bash
source ./setup-go.sh
```

This will configure your current shell with the correct Go environment.

### Option 2: Automatic via .bashrc
The environment is automatically configured in `~/.bashrc`, so new shells will have Go 1.24.10 available.

### Option 3: Using .envrc (for direnv users)
If you use [direnv](https://direnv.net/), simply run:
```bash
direnv allow .
```

## Verification

Check that Go is correctly configured:
```bash
go version
# Should output: go version go1.24.10 linux/amd64

go env GOROOT
# Should output: /home/decri/go-installs/go
```

## Running Tests

With the environment configured, you can run tests normally:
```bash
# Test all packages
go test ./...

# Test specific package
go test ./p2p/...
go test ./x/oracle/...

# Test with coverage
go test -cover ./...

# Test with race detection
go test -race ./...
```

## Building

Build the PAW daemon:
```bash
go build -o pawd ./cmd/...

# Or using the Makefile
make build
```

## Environment Variables

The following environment variables are configured:

| Variable | Value | Purpose |
|----------|-------|---------|
| `GOROOT` | `$HOME/go-installs/go` | Go installation directory |
| `GOPATH` | `$HOME/go` | Go workspace for dependencies |
| `GO111MODULE` | `on` | Enable Go modules |
| `PAW_HOME` | `$HOME/.paw` | PAW node data directory |

## Troubleshooting

### "package X is not in std"
This means GOROOT is not set correctly. Run:
```bash
export GOROOT="$HOME/go-installs/go"
source ./setup-go.sh
```

### "go: command not found"
The PATH is not configured. Run:
```bash
export PATH="$HOME/go-installs/go/bin:$PATH"
```

Or source the setup script:
```bash
source ./setup-go.sh
```

### Verifying the installation
```bash
# Check Go version
go version

# Check environment
go env GOROOT GOPATH

# Verify standard library
ls $GOROOT/src/
```

## Notes

- Go modules are cached in `$GOPATH/pkg/mod/`
- Compiled binaries default to `$GOPATH/bin/`
- The PAW daemon (`pawd`) can be built in the project root
