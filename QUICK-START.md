# Quick Start: Using Go 1.24.10 in PAW

## ✅ Installation Complete

Go 1.24.10 is installed at: `~/go-installs/go/`

## 🚀 Three Ways to Use Go

### Method 1: Wrapper Script (Easiest for Single Commands)

Use the wrapper script for one-off commands:

```bash
./run-with-go.sh go test ./...
./run-with-go.sh go build -o pawd ./cmd/...
./run-with-go.sh go version
```

### Method 2: Source Setup Script (Best for Development Sessions)

For active development, source the setup script once per terminal session:

```bash
source ./setup-go.sh
```

Then use go commands normally:
```bash
go test ./...
go build ./cmd/...
go version
```

### Method 3: New Terminal (Automatic)

Open a **new terminal** - Go will be automatically available thanks to `~/.bashrc` configuration.

```bash
# In a NEW terminal (not your current one):
cd /home/decri/blockchain-projects/paw
go version  # Works automatically!
go test ./...
```

## ⚡ Quick Test

Verify the installation works:

```bash
# Using wrapper:
./run-with-go.sh go test ./p2p/discovery/...

# Or source first:
source ./setup-go.sh
go test ./p2p/discovery/...
```

## 📋 Common Commands

```bash
# Run all tests
./run-with-go.sh go test ./...

# Run specific module
./run-with-go.sh go test ./x/oracle/...

# Build the daemon
./run-with-go.sh go build -o pawd ./cmd/...

# With coverage
./run-with-go.sh go test -cover ./...

# With race detection
./run-with-go.sh go test -race ./...
```

## 🔧 For Your Current Shell

If you're in the middle of work and don't want to open a new terminal:

```bash
source ./setup-go.sh
```

This activates Go in your current session.

## 📝 Files Created

- `run-with-go.sh` - Wrapper script for single commands
- `setup-go.sh` - Environment setup for development sessions
- `.envrc` - For direnv users
- `README-GO-SETUP.md` - Detailed documentation
- `GO-INSTALLATION.md` - Installation summary

## ❓ Troubleshooting

**"go: command not found"**
- Solution 1: Use wrapper: `./run-with-go.sh go <command>`
- Solution 2: Source setup: `source ./setup-go.sh`
- Solution 3: Open a new terminal

**"package X is not in std"**
- Your GOROOT isn't set. Run: `source ./setup-go.sh`

## ✨ Confirmed Working

```bash
✓ Go 1.24.10 installed
✓ Standard library available
✓ Testing framework works
✓ Dependencies resolve correctly
✓ p2p/discovery tests pass
```

You're all set! 🎉
