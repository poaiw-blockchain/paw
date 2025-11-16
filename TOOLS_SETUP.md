# PAW Blockchain Development Tools Setup

This document describes the development tools and utilities available for the PAW blockchain project.

## Overview

The PAW project includes a comprehensive set of development tools for building, testing, releasing, and maintaining the blockchain codebase.

## ✅ Installed Development Tools (Ready to Use)

All development tools have been installed globally and are ready to use for the PAW blockchain project and any other projects under `C:\Users\decri\GitClones\`.

### Go Tools (Global)

- **gosec v2.dev** - Security scanner for Go code
- **golangci-lint v1.64.8** - Comprehensive linter suite
- **goimports** - Automatic import formatting and organization
- **govulncheck v1.1.4** - Vulnerability scanner for Go dependencies

### Python Tools (Global)

- **black 25.11.0** - Opinionated code formatter
- **pylint 4.0.2** - Static code analyzer and linter
- **mypy 1.18.2** - Static type checker
- **locust 2.42.2** - Load testing framework (configured for Visual Studio 2026)
- **pre-commit** - Git hook framework (installed and configured)

### Node.js Tools (Project-level)

- **ESLint v8.57.1** - JavaScript/TypeScript linter
- **Prettier 3.6.2** - Code formatter for JS/TS/JSON/YAML/MD
- **commitlint v18.6.1** - Commit message linter
- **372 npm packages** - Complete project dependencies installed

### Git Hooks

- **Husky** - Modern Git hooks management (configured and active)
  - Replaces older pre-commit framework for cleaner hook management
  - Automatically runs linters and formatters on commit
  - Enforces commit message conventions via commitlint

### Usage Notes

- All Go tools are accessible globally via command line
- Python tools are in the global Python environment
- Node.js tools can be run via `npx` or `npm run` scripts
- Git hooks will automatically run on commits and pushes
- Tools are configured to work seamlessly with Visual Studio 2026

**Quick Commands:**

```bash
# Security scan
gosec ./...

# Lint Go code
golangci-lint run

# Format Python code
black .

# Check Python types
mypy .

# Lint JavaScript/TypeScript
npm run lint

# Format all supported files
npm run format

# Run load tests
locust -f tests/load/locustfile.py
```

## Files Created

### 1. Build & Release Tools

#### `.goreleaser.yml`

Automated release builds for multiple platforms using GoReleaser.

**Features:**

- Multi-platform builds (Windows, Linux, macOS)
- Multi-architecture support (amd64, arm64)
- Automatic changelog generation
- Docker image builds
- GitHub release automation

**Usage:**

```bash
# Test release configuration
make release-test

# Build snapshot release (no publish)
make release

# Full release (requires git tag)
goreleaser release --clean
```

#### `VERSION`

Simple version file tracking the current release version.

**Current Version:** `0.1.0-alpha`

### 2. Dependency Management

#### `renovate.json`

Automated dependency updates via Renovate bot.

**Features:**

- Auto-merge minor and patch updates
- Separate handling for major updates
- Weekly dependency scans
- Security vulnerability alerts
- Grouped updates for Go modules, GitHub Actions

**Configuration:**

- Runs on weekends and late weeknights
- Auto-merges safe updates
- Labels PRs appropriately
- Maintains lockfiles

### 3. Docker Development

#### `.dockerignore`

Optimizes Docker builds by excluding unnecessary files.

**Excluded:**

- Git files and history
- IDE configuration
- Build artifacts
- Documentation
- Test files
- Node modules
- Temporary files

#### `docker-compose.dev.yml`

Complete development environment with hot reload.

**Services:**

- `paw-node`: Main validator node
- `paw-validator-2`: Second validator for multi-node testing
- `postgres`: PostgreSQL for indexing
- `redis`: Redis for caching
- `prometheus`: Metrics collection
- `grafana`: Metrics visualization

**Ports:**

```
PAW Node 1:
  - 26657: Tendermint RPC
  - 1317: REST API
  - 9090: gRPC
  - 26660: Prometheus metrics

PAW Node 2:
  - 26667: Tendermint RPC
  - 1327: REST API
  - 9091: gRPC

Services:
  - 5432: PostgreSQL
  - 6379: Redis
  - 9090: Prometheus
  - 3000: Grafana
```

**Usage:**

```bash
# Start development environment
make dev

# Stop development environment
make dev-down

# Or use docker-compose directly
docker-compose -f docker-compose.dev.yml up
docker-compose -f docker-compose.dev.yml down -v
```

**Grafana Access:**

- URL: http://localhost:3000
- Username: admin
- Password: pawdev123

### 4. Development Scripts

#### `scripts/dev-setup.sh` (Unix/Mac/Linux)

#### `scripts/dev-setup.ps1` (Windows PowerShell)

One-command development environment setup.

**What it does:**

- Checks Go installation (requires 1.21+)
- Installs Go dependencies
- Installs development tools:
  - golangci-lint
  - goimports
  - misspell
  - buf (protobuf)
  - goreleaser
- Checks Python installation
- Checks Node.js installation
- Sets up Git hooks
- Installs pre-commit hooks
- Checks Docker installation
- Runs initial tests
- Creates necessary directories

**Usage:**

```bash
# Unix/Mac/Linux
./scripts/dev-setup.sh

# Windows PowerShell
.\scripts\dev-setup.ps1

# Or use Makefile
make dev-setup
```

#### `scripts/clean.sh` (Unix/Mac/Linux)

#### `scripts/clean.ps1` (Windows PowerShell)

Clean build artifacts and caches.

**What it cleans:**

- Build directory
- Installed binaries (pawd, pawcli)
- Test cache
- Build cache
- Module cache (optional)
- Coverage files
- Proto-generated files (optional)
- Blockchain data (~/.paw)
- Testnet data (./data)
- Node modules
- Python cache
- Log files
- Docker volumes (optional)
- Temporary files
- Pre-commit cache
- GoReleaser dist

**Usage:**

```bash
# Unix/Mac/Linux
./scripts/clean.sh

# Windows PowerShell
.\scripts\clean.ps1

# Or use Makefile
make clean      # Basic cleanup
make clean-all  # Deep cleanup
```

#### `scripts/format-all.sh` (Unix/Mac/Linux)

#### `scripts/format-all.ps1` (Windows PowerShell)

Format all code in the repository.

**Supported Languages:**

- Go (gofmt, goimports, misspell)
- JavaScript/TypeScript (prettier)
- Python (black or autopep8)
- Protobuf (clang-format)
- YAML (yamllint)
- Markdown (prettier)
- Shell scripts (shfmt)

**Usage:**

```bash
# Unix/Mac/Linux
./scripts/format-all.sh

# Windows PowerShell
.\scripts\format-all.ps1

# Or use Makefile
make format      # Go files only
make format-all  # All languages
```

### 5. Git Configuration

#### `.gitattributes`

Git LFS and line ending configuration.

**Features:**

- Automatic line ending normalization
- Unix line endings (LF) for source code
- Windows line endings (CRLF) for Windows scripts
- Binary file detection
- Export filtering
- Linguist language detection

**Line Endings:**

- Source code: LF (Unix)
- Shell scripts: LF
- PowerShell scripts: CRLF
- Batch files: CRLF

### 6. Monitoring Configuration

#### `infra/prometheus.yml`

Prometheus metrics collection configuration.

**Monitored Targets:**

- PAW nodes (Tendermint metrics)
- Cosmos SDK metrics
- PostgreSQL (optional)
- Redis (optional)
- System metrics (optional)

#### `infra/grafana/provisioning/`

Grafana automatic provisioning.

**Included:**

- Prometheus datasource
- Dashboard configurations

## Makefile Targets

### New Targets Added

```bash
# Install all development tools
make install-tools

# Format all code (all languages)
make format-all

# Clean build artifacts (basic)
make clean

# Deep clean (everything)
make clean-all

# Start development environment (Docker)
make dev

# Stop development environment
make dev-down

# Build release binaries
make release

# Test release configuration
make release-test

# Run development setup
make dev-setup
```

### Existing Targets Enhanced

```bash
# Build project
make build

# Install binaries
make install

# Run tests
make test
make test-unit
make test-integration
make test-coverage

# Linting
make lint

# Format Go code
make format

# Protobuf
make proto-all
make proto-gen
make proto-format
make proto-lint

# Local network
make localnet-start
make localnet-stop
```

## Development Workflow

### First Time Setup

1. Clone the repository
2. Run setup script:
   ```bash
   ./scripts/dev-setup.sh  # Unix/Mac/Linux
   .\scripts\dev-setup.ps1  # Windows
   ```
3. Build the project:
   ```bash
   make build
   ```
4. Run tests:
   ```bash
   make test
   ```

### Daily Development

1. Start development environment:

   ```bash
   make dev
   ```

2. Make changes to code

3. Format code:

   ```bash
   make format-all
   ```

4. Run linter:

   ```bash
   make lint
   ```

5. Run tests:

   ```bash
   make test
   ```

6. Commit changes:
   ```bash
   git add .
   git commit -m "feat: your changes"
   ```

### Cleanup

```bash
# Basic cleanup (build artifacts)
make clean

# Deep cleanup (everything)
make clean-all

# Or use interactive script
./scripts/clean.sh
```

## Tool Requirements

### Required

- **Go 1.21+**: Core language
- **Git**: Version control
- **Make**: Build automation

### Recommended

- **golangci-lint**: Linting
- **goimports**: Import formatting
- **misspell**: Spell checking
- **buf**: Protobuf management
- **goreleaser**: Release automation

### Optional

- **Docker**: Containerized development
- **Docker Compose**: Multi-container orchestration
- **Python 3**: Wallet scripts
- **Node.js**: Frontend development
- **prettier**: JS/TS/MD formatting
- **black**: Python formatting
- **clang-format**: Protobuf formatting
- **pre-commit**: Git hook management

## Platform Compatibility

All scripts are provided in both Unix and PowerShell versions:

- **Unix/Mac/Linux**: `.sh` scripts
- **Windows**: `.ps1` scripts (PowerShell)

The Makefile works on all platforms with appropriate Make installed:

- Unix/Mac/Linux: Native make
- Windows: Use `make` via WSL, Git Bash, or install GNU Make

## Continuous Integration

The project is configured for automated CI/CD:

- **Renovate**: Automatic dependency updates
- **GoReleaser**: Automated releases on git tags
- **Docker**: Automated image builds
- **GitHub Actions**: Ready for CI/CD integration

## Troubleshooting

### Scripts not executable (Unix)

```bash
chmod +x scripts/*.sh
```

### Docker permission denied (Linux)

```bash
sudo usermod -aG docker $USER
# Log out and back in
```

### Go tools not found

```bash
# Ensure GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Or run installation again
make install-tools
```

### PowerShell execution policy (Windows)

```powershell
# Allow scripts to run
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Additional Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [Renovate Documentation](https://docs.renovatebot.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

## Version History

- **0.1.0-alpha**: Initial development tooling setup
  - GoReleaser configuration
  - Renovate integration
  - Docker development environment
  - Setup and cleanup scripts
  - Comprehensive formatting tools
  - Monitoring stack (Prometheus/Grafana)
