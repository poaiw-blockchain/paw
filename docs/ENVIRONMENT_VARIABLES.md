# PAW Blockchain Environment Variables

**Version:** 1.0.0
**Last Updated:** 2025-12-07
**Audience:** Operators, Developers, DevOps Engineers

---

## Table of Contents

1. [Overview](#overview)
2. [Variable Naming Convention](#variable-naming-convention)
3. [Core Configuration Variables](#core-configuration-variables)
4. [Network Configuration](#network-configuration)
5. [RPC and API Configuration](#rpc-and-api-configuration)
6. [Logging Configuration](#logging-configuration)
7. [Telemetry and Monitoring](#telemetry-and-monitoring)
8. [Security and Cryptography](#security-and-cryptography)
9. [Module-Specific Variables](#module-specific-variables)
10. [Development and Testing](#development-and-testing)
11. [Cloud Provider Configuration](#cloud-provider-configuration)
12. [Complete Reference Table](#complete-reference-table)
13. [Best Practices](#best-practices)

---

## Overview

PAW blockchain uses environment variables for runtime configuration, allowing operators to customize node behavior without modifying configuration files. This document provides a comprehensive reference of all supported environment variables.

### Configuration Hierarchy

PAW reads configuration from three sources, in order of precedence:

1. **Environment Variables** (highest priority)
2. **Configuration Files** (`config.toml`, `app.toml`, `client.toml`)
3. **Default Values** (lowest priority)

Environment variables override file-based configuration, making them ideal for:
- Containerized deployments (Docker, Kubernetes)
- CI/CD pipelines
- Multi-environment management (dev, staging, production)
- Secret management (avoiding hardcoded credentials)

### Supported Shells

All examples use POSIX-compliant syntax compatible with:
- **bash** (recommended)
- **zsh**
- **sh**

For persistent configuration, add variables to:
- `~/.bashrc` or `~/.bash_profile` (bash)
- `~/.zshrc` (zsh)
- `/etc/environment` (system-wide)

---

## Variable Naming Convention

PAW follows a standardized naming convention for environment variables:

### Naming Rules

1. **Prefix:** All PAW-specific variables use `PAW_` prefix
2. **Format:** `PAW_<COMPONENT>_<SETTING>` in UPPERCASE with underscores
3. **Components:** `HOME`, `CHAIN`, `RPC`, `API`, `P2P`, `LOG`, etc.
4. **Boolean Values:** Use `true`/`false` (lowercase)
5. **Numeric Values:** Plain integers (no units in value)

### Examples

```bash
# Correct
PAW_HOME="/data/paw"
PAW_CHAIN_ID="paw-mvp-1"
PAW_RPC_LADDR="tcp://0.0.0.0:26657"
PAW_LOG_LEVEL="info"

# Incorrect (no PAW_ prefix)
HOME="/data/paw"           # Conflicts with shell HOME
CHAIN_ID="paw-mvp-1"   # Ambiguous

# Incorrect (mixed case)
Paw_Home="/data/paw"
PAW_log_level="info"
```

---

## Core Configuration Variables

### PAW_HOME

**Description:** Node data directory containing config, data, and keyring

**Type:** String (absolute path)
**Default:** `~/.paw`
**Required:** No

**Purpose:**
- Specifies the root directory for all node data
- Contains subdirectories: `config/`, `data/`, `keyring-*/`
- Critical for running multiple nodes on the same machine

**Example:**

```bash
# Single node
export PAW_HOME="$HOME/.paw"

# Multiple nodes
export PAW_HOME="/data/paw-node-1"
export PAW_HOME="/data/paw-validator-2"

# Docker container
export PAW_HOME="/paw"
```

**Verification:**

```bash
echo $PAW_HOME
ls -la $PAW_HOME
# Should show: config/, data/, keyring-*/
```

---

### PAW_CHAIN_ID

**Description:** Blockchain network identifier

**Type:** String
**Default:** None (must be set)
**Required:** Yes (for most operations)

**Purpose:**
- Identifies which blockchain network the node belongs to
- Prevents transaction replay across different networks
- Must match genesis file and other nodes

**Common Values:**

```bash
# Mainnet (when launched)
export PAW_CHAIN_ID="paw-1"

# Testnet
export PAW_CHAIN_ID="paw-mvp-1"

# Devnet
export PAW_CHAIN_ID="paw-devnet"

# Local development
export PAW_CHAIN_ID="paw-local-$(date +%s)"
```

**Verification:**

```bash
pawd status 2>&1 | jq -r '.NodeInfo.network'
# Should output: paw-mvp-1
```

---

### PAW_MONIKER

**Description:** Human-readable node name

**Type:** String
**Default:** Hostname or random name
**Required:** No

**Purpose:**
- Identifies node in peer lists and block explorers
- Helps operators distinguish nodes in multi-node setups

**Example:**

```bash
# Validator nodes
export PAW_MONIKER="validator-us-east-1a"
export PAW_MONIKER="validator-eu-west-2b"

# Sentry nodes
export PAW_MONIKER="sentry-singapore-1"

# Personal node
export PAW_MONIKER="alice-full-node"
```

**Best Practices:**
- Use descriptive names including location/purpose
- Avoid special characters (alphanumeric and hyphens only)
- Keep under 64 characters

---

### PAW_MINIMUM_GAS_PRICE

**Description:** Minimum gas price for transaction acceptance

**Type:** String (amount with denom)
**Default:** `0upaw`
**Required:** No

**Purpose:**
- Protects against spam transactions
- Ensures economic viability for validators
- Filters transactions in mempool

**Example:**

```bash
# Mainnet (recommended)
export PAW_MINIMUM_GAS_PRICE="0.001upaw"

# Testnet (lower fees)
export PAW_MINIMUM_GAS_PRICE="0.0001upaw"

# Development (free)
export PAW_MINIMUM_GAS_PRICE="0upaw"
```

**Format:**
- Amount: Decimal number
- Denom: Token denomination (e.g., `upaw` = 10^-6 PAW)

**Verification:**

```bash
grep minimum-gas-prices $PAW_HOME/config/app.toml
```

---

## Network Configuration

### PAW_P2P_LADDR

**Description:** P2P listening address

**Type:** String (protocol://host:port)
**Default:** `tcp://0.0.0.0:26656`
**Required:** No

**Purpose:**
- Specifies address for peer-to-peer communication
- Must be reachable by other nodes for inbound connections

**Example:**

```bash
# Default (all interfaces)
export PAW_P2P_LADDR="tcp://0.0.0.0:26656"

# Specific interface
export PAW_P2P_LADDR="tcp://192.168.1.10:26656"

# Custom port (for multiple nodes)
export PAW_P2P_LADDR="tcp://0.0.0.0:26756"

# IPv6
export PAW_P2P_LADDR="tcp://[::]:26656"
```

**Firewall Configuration:**

```bash
# Allow inbound P2P connections
sudo ufw allow 26656/tcp comment 'PAW P2P'
```

---

### PAW_P2P_EXTERNAL_ADDRESS

**Description:** Externally routable P2P address advertised to peers

**Type:** String (host:port)
**Default:** Auto-detected or empty
**Required:** No (required for NAT/cloud)

**Purpose:**
- Advertises public IP to peers behind NAT
- Enables inbound connections from internet

**Example:**

```bash
# Cloud instance
export PAW_P2P_EXTERNAL_ADDRESS="203.0.113.42:26656"

# Dynamic DNS
export PAW_P2P_EXTERNAL_ADDRESS="validator.example.com:26656"

# Multiple addresses (comma-separated)
export PAW_P2P_EXTERNAL_ADDRESS="203.0.113.42:26656,validator.example.com:26656"
```

**Auto-Detection:**

```bash
# Get public IP
PUBLIC_IP=$(curl -s ifconfig.me)
export PAW_P2P_EXTERNAL_ADDRESS="${PUBLIC_IP}:26656"
```

---

### PAW_P2P_SEEDS

**Description:** Seed nodes for initial peer discovery

**Type:** String (comma-separated node IDs with addresses)
**Default:** Empty
**Required:** No (recommended for new nodes)

**Purpose:**
- Bootstrap peer discovery for new nodes
- Seed nodes provide peer addresses then disconnect

**Example:**

```bash
# Testnet seeds
export PAW_P2P_SEEDS="node-id-1@seed-1.paw-chain.io:26656,node-id-2@seed-2.paw-chain.io:26656"

# Multiple regions
export PAW_P2P_SEEDS="\
ff5d34e3f2c8f3a9b@seed-us.paw-chain.io:26656,\
8a7d9e2f1b4c5a6d@seed-eu.paw-chain.io:26656,\
3c2e1a9f7b8d5e4c@seed-asia.paw-chain.io:26656"
```

**Format:** `<node-id>@<host>:<port>`

**Get Node ID:**

```bash
pawd tendermint show-node-id
# Output: ff5d34e3f2c8f3a9b1c7d2e8f4a6b9c5d
```

---

### PAW_P2P_PERSISTENT_PEERS

**Description:** Persistent peer connections maintained continuously

**Type:** String (comma-separated node IDs with addresses)
**Default:** Empty
**Required:** No (recommended for validators)

**Purpose:**
- Maintains permanent connections to specific peers
- Critical for validator sentry architecture

**Example:**

```bash
# Sentry nodes for validator
export PAW_P2P_PERSISTENT_PEERS="\
sentry-1-id@10.0.1.10:26656,\
sentry-2-id@10.0.2.10:26656"

# Trusted peer network
export PAW_P2P_PERSISTENT_PEERS="\
peer-1@peer1.example.com:26656,\
peer-2@peer2.example.com:26656"
```

**Validator Security:**
- Validators should ONLY connect to their sentry nodes
- Set `PAW_P2P_PEX=false` to disable peer exchange

---

### PAW_P2P_PEX

**Description:** Enable Peer Exchange (PEX) reactor

**Type:** Boolean
**Default:** `true`
**Required:** No

**Purpose:**
- Enables automatic peer discovery via peer exchange
- Validators should disable for security (use persistent peers only)

**Example:**

```bash
# Full nodes and sentries (enable peer discovery)
export PAW_P2P_PEX=true

# Validators (disable for security)
export PAW_P2P_PEX=false
```

---

### PAW_P2P_MAX_NUM_INBOUND_PEERS

**Description:** Maximum inbound peer connections

**Type:** Integer
**Default:** `40`
**Required:** No

**Example:**

```bash
# Default
export PAW_P2P_MAX_NUM_INBOUND_PEERS=40

# High-capacity sentry node
export PAW_P2P_MAX_NUM_INBOUND_PEERS=100

# Resource-constrained node
export PAW_P2P_MAX_NUM_INBOUND_PEERS=20
```

---

### PAW_P2P_MAX_NUM_OUTBOUND_PEERS

**Description:** Maximum outbound peer connections

**Type:** Integer
**Default:** `10`
**Required:** No

**Example:**

```bash
# Default
export PAW_P2P_MAX_NUM_OUTBOUND_PEERS=10

# Validator (only connect to sentries)
export PAW_P2P_MAX_NUM_OUTBOUND_PEERS=2
```

---

## RPC and API Configuration

### PAW_RPC_LADDR

**Description:** RPC server listening address

**Type:** String (protocol://host:port)
**Default:** `tcp://127.0.0.1:26657`
**Required:** No

**Purpose:**
- CometBFT RPC for queries and transaction broadcast
- Default binds to localhost (secure)

**Example:**

```bash
# Localhost only (secure default)
export PAW_RPC_LADDR="tcp://127.0.0.1:26657"

# All interfaces (public RPC node)
export PAW_RPC_LADDR="tcp://0.0.0.0:26657"

# Unix socket (highest security)
export PAW_RPC_LADDR="unix:///var/run/paw/rpc.sock"
```

**Security Warning:**
- Exposing RPC publicly allows anyone to query blockchain data
- Use firewall rules or reverse proxy for public exposure
- Validators should NOT expose RPC publicly

---

### PAW_RPC_ENDPOINT

**Description:** RPC endpoint URL for client operations

**Type:** String (URL)
**Default:** `http://localhost:26657`
**Required:** No (for CLI operations)

**Purpose:**
- Used by CLI to connect to node for queries
- Can point to remote node

**Example:**

```bash
# Local node
export PAW_RPC_ENDPOINT="http://localhost:26657"

# Remote node
export PAW_RPC_ENDPOINT="https://rpc.paw-chain.io:443"

# Load balancer
export PAW_RPC_ENDPOINT="https://rpc-lb.paw-chain.io"
```

**Client Usage:**

```bash
pawd query bank balances paw1... --node $PAW_RPC_ENDPOINT
```

---

### PAW_API_ADDRESS

**Description:** Cosmos SDK REST API listening address

**Type:** String (host:port)
**Default:** `tcp://0.0.0.0:1317`
**Required:** No

**Purpose:**
- REST API for queries (alternative to RPC)
- Swagger documentation available

**Example:**

```bash
# Default
export PAW_API_ADDRESS="tcp://0.0.0.0:1317"

# Localhost only
export PAW_API_ADDRESS="tcp://127.0.0.1:1317"

# Custom port
export PAW_API_ADDRESS="tcp://0.0.0.0:8080"
```

---

### PAW_API_ENABLE

**Description:** Enable REST API server

**Type:** Boolean
**Default:** `true`
**Required:** No

**Example:**

```bash
# Enable API (for public RPC nodes)
export PAW_API_ENABLE=true

# Disable API (for validators, saves resources)
export PAW_API_ENABLE=false
```

---

### PAW_GRPC_ADDRESS

**Description:** gRPC server listening address

**Type:** String (host:port)
**Default:** `0.0.0.0:9090`
**Required:** No

**Purpose:**
- gRPC API for advanced clients and services
- Used by Cosmos SDK clients and block explorers

**Example:**

```bash
# Default
export PAW_GRPC_ADDRESS="0.0.0.0:9090"

# Custom port
export PAW_GRPC_ADDRESS="0.0.0.0:9091"
```

---

### PAW_GRPC_WEB_ADDRESS

**Description:** gRPC-Web server listening address

**Type:** String (host:port)
**Default:** `0.0.0.0:9091`
**Required:** No

**Purpose:**
- Enables gRPC from web browsers
- Used by web-based wallets and dApps

**Example:**

```bash
export PAW_GRPC_WEB_ADDRESS="0.0.0.0:9091"
```

---

## Logging Configuration

### PAW_LOG_LEVEL

**Description:** Global logging level

**Type:** String (enum)
**Default:** `info`
**Required:** No

**Valid Values:**
- `debug`: Verbose debug information
- `info`: General informational messages
- `warn`: Warning messages only
- `error`: Error messages only
- `panic`: Panic-level errors only

**Example:**

```bash
# Production (recommended)
export PAW_LOG_LEVEL="info"

# Development/debugging
export PAW_LOG_LEVEL="debug"

# Minimal logging
export PAW_LOG_LEVEL="error"
```

---

### PAW_LOG_FORMAT

**Description:** Log output format

**Type:** String (enum)
**Default:** `plain`
**Required:** No

**Valid Values:**
- `plain`: Human-readable text
- `json`: Structured JSON (for log aggregation)

**Example:**

```bash
# Human-readable (development)
export PAW_LOG_FORMAT="plain"

# Structured logging (production, for Loki/ELK)
export PAW_LOG_FORMAT="json"
```

**JSON Example Output:**

```json
{"level":"info","ts":"2025-12-07T12:34:56.789Z","caller":"node/node.go:123","msg":"Starting node","chain_id":"paw-mvp-1","moniker":"validator-1"}
```

---

### PAW_LOG_MODULE_LEVELS

**Description:** Per-module log levels (overrides global level)

**Type:** String (comma-separated module:level pairs)
**Default:** Empty
**Required:** No

**Purpose:**
- Fine-grained logging control
- Debug specific modules without excessive logs

**Example:**

```bash
# Debug DEX module only
export PAW_LOG_MODULE_LEVELS="dex:debug,oracle:debug"

# Silence noisy modules
export PAW_LOG_MODULE_LEVELS="p2p:error,consensus:warn"

# Complex configuration
export PAW_LOG_MODULE_LEVELS="dex:debug,oracle:info,compute:warn,state:error"
```

**Available Modules:**
- `state`, `consensus`, `mempool`, `p2p`, `rpc`
- `bank`, `staking`, `distribution`, `gov`, `slashing`
- `dex`, `oracle`, `compute` (PAW custom)

---

## Telemetry and Monitoring

### PAW_TELEMETRY_ENABLED

**Description:** Enable telemetry and metrics collection

**Type:** Boolean
**Default:** `false`
**Required:** No

**Example:**

```bash
# Enable metrics (production)
export PAW_TELEMETRY_ENABLED=true

# Disable metrics (saves resources)
export PAW_TELEMETRY_ENABLED=false
```

---

### PAW_TELEMETRY_PROMETHEUS_RETENTION

**Description:** Prometheus metrics retention time in seconds

**Type:** Integer
**Default:** `600` (10 minutes)
**Required:** No

**Example:**

```bash
# 10 minutes (default)
export PAW_TELEMETRY_PROMETHEUS_RETENTION=600

# 1 hour
export PAW_TELEMETRY_PROMETHEUS_RETENTION=3600

# No retention (scrape immediately)
export PAW_TELEMETRY_PROMETHEUS_RETENTION=0
```

---

### PAW_TELEMETRY_GLOBAL_LABELS

**Description:** Global labels added to all metrics

**Type:** String (JSON array of [key, value] pairs)
**Default:** `[]`
**Required:** No

**Example:**

```bash
# Single label
export PAW_TELEMETRY_GLOBAL_LABELS='[["chain_id","paw-mvp-1"]]'

# Multiple labels
export PAW_TELEMETRY_GLOBAL_LABELS='[["chain_id","paw-mvp-1"],["network","testnet"],["region","us-east-1"]]'

# Kubernetes environment
export PAW_TELEMETRY_GLOBAL_LABELS='[["chain_id","paw-mvp-1"],["pod","'$HOSTNAME'"],["namespace","paw-blockchain"]]'
```

---

### PAW_PROMETHEUS_LISTEN_ADDR

**Description:** Prometheus metrics endpoint address

**Type:** String (host:port)
**Default:** `:26660`
**Required:** No

**Purpose:**
- Exposes `/metrics` endpoint for Prometheus scraping

**Example:**

```bash
# Default (all interfaces)
export PAW_PROMETHEUS_LISTEN_ADDR=":26660"

# Localhost only (security)
export PAW_PROMETHEUS_LISTEN_ADDR="127.0.0.1:26660"

# Custom port
export PAW_PROMETHEUS_LISTEN_ADDR=":9091"
```

---

### PAW_TELEMETRY_HEALTH_PORT

**Description:** Port for the bundled health check endpoints

**Type:** Integer
**Default:** `36661`
**Required:** No

**Purpose:**
- Hosts `/health`, `/health/ready`, `/health/detailed`, and `/health/startup`
- Allows validator and archive instances to use unique telemetry ports

**Example:**

```bash
# Run health server on 36671 to avoid collisions
export PAW_TELEMETRY_HEALTH_PORT=36671
```

---

## Security and Cryptography

### PAW_KEYRING_BACKEND

**Description:** Keyring storage backend

**Type:** String (enum)
**Default:** `os`
**Required:** No

**Valid Values:**
- `os`: Operating system keyring (secure, recommended)
- `file`: Encrypted file (password required)
- `test`: Unencrypted (DEVELOPMENT ONLY)
- `kwallet`: KDE Wallet (Linux with KDE)
- `pass`: Password manager (Linux)

**Example:**

```bash
# Production (secure OS keyring)
export PAW_KEYRING_BACKEND="os"

# Development (no password)
export PAW_KEYRING_BACKEND="test"

# File-based (portable, requires password)
export PAW_KEYRING_BACKEND="file"
```

**Security Warning:**
- NEVER use `test` backend in production
- Compromised keys can result in total loss of funds

---

### PAW_KEYRING_DIR

**Description:** Keyring storage directory

**Type:** String (absolute path)
**Default:** `$PAW_HOME` (uses subdirectory `keyring-<backend>`)
**Required:** No

**Example:**

```bash
# Default (inside PAW_HOME)
# Keys stored in: ~/.paw/keyring-os/

# Custom directory
export PAW_KEYRING_DIR="/secure/keys/paw"

# Separate volume (Kubernetes)
export PAW_KEYRING_DIR="/mnt/secrets/paw-keys"
```

---

### PAW_UNSAFE_SKIP_UPGRADES

**Description:** Block heights to skip during upgrades

**Type:** String (comma-separated integers)
**Default:** Empty
**Required:** No (used for emergency recovery)

**Purpose:**
- Emergency recovery from failed upgrades
- Skip problematic upgrade heights

**Example:**

```bash
# Skip upgrade at height 1000000
export PAW_UNSAFE_SKIP_UPGRADES="1000000"

# Skip multiple upgrades
export PAW_UNSAFE_SKIP_UPGRADES="1000000,2000000,3000000"
```

**Warning:** Only use under guidance from core team

---

## Module-Specific Variables

### DEX Module

#### PAW_DEX_ENABLED

**Description:** Enable DEX module functionality

**Type:** Boolean
**Default:** `true`
**Required:** No

```bash
export PAW_DEX_ENABLED=true
```

#### PAW_DEX_FEE_COLLECTOR

**Description:** Address receiving DEX swap fees

**Type:** String (bech32 address)
**Default:** Module account
**Required:** No

```bash
export PAW_DEX_FEE_COLLECTOR="paw1dexfees..."
```

---

### Oracle Module

#### PAW_ORACLE_ENABLED

**Description:** Enable Oracle module functionality

**Type:** Boolean
**Default:** `true`
**Required:** No

```bash
export PAW_ORACLE_ENABLED=true
```

#### PAW_ORACLE_PRICE_SOURCES

**Description:** External price feed URLs

**Type:** String (comma-separated URLs)
**Default:** Empty (rely on validators)
**Required:** No

```bash
export PAW_ORACLE_PRICE_SOURCES="https://api.coingecko.com,https://api.binance.com"
```

---

### Compute Module

#### PAW_COMPUTE_ENABLED

**Description:** Enable Compute module functionality

**Type:** Boolean
**Default:** `true`
**Required:** No

```bash
export PAW_COMPUTE_ENABLED=true
```

#### PAW_COMPUTE_MAX_PROVIDERS

**Description:** Maximum number of registered compute providers

**Type:** Integer
**Default:** `1000`
**Required:** No

```bash
export PAW_COMPUTE_MAX_PROVIDERS=1000
```

---

## Development and Testing

### USE_COMETMOCK

**Description:** Use CometMock for deterministic testing

**Type:** Boolean
**Default:** `false`
**Required:** No (testing only)

**Example:**

```bash
# Enable CometMock for E2E tests
export USE_COMETMOCK=true
```

---

### PAW_SMOKE_PHASES

**Description:** Smoke test phases to run

**Type:** String (comma-separated phase names)
**Default:** `setup,bank,dex,swap,summary`
**Required:** No (testing only)

**Example:**

```bash
# Run all phases
export PAW_SMOKE_PHASES="setup,bank,dex,swap,summary"

# Run specific phases
export PAW_SMOKE_PHASES="setup,bank"
```

---

### PAW_SMOKE_KEEP_STACK

**Description:** Keep Docker stack after smoke tests

**Type:** Boolean
**Default:** `false`
**Required:** No (testing only)

**Example:**

```bash
# Keep stack for debugging
export PAW_SMOKE_KEEP_STACK=1
```

---

## Cloud Provider Configuration

### AWS

```bash
# S3 state sync snapshots
export AWS_ACCESS_KEY_ID="AKIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_DEFAULT_REGION="us-east-1"
export PAW_STATESYNC_SNAPSHOT_BUCKET="paw-snapshots"
```

### Google Cloud Platform

```bash
# GCS state sync snapshots
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export PAW_STATESYNC_SNAPSHOT_BUCKET="gs://paw-snapshots"
```

### Azure

```bash
# Azure Blob Storage
export AZURE_STORAGE_ACCOUNT="pawstorage"
export AZURE_STORAGE_KEY="..."
export PAW_STATESYNC_SNAPSHOT_CONTAINER="snapshots"
```

---

## Complete Reference Table

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| **Core** |
| `PAW_HOME` | string | `~/.paw` | Node data directory |
| `PAW_CHAIN_ID` | string | - | Chain identifier |
| `PAW_MONIKER` | string | hostname | Node name |
| `PAW_MINIMUM_GAS_PRICE` | string | `0upaw` | Minimum gas price |
| **Network** |
| `PAW_P2P_LADDR` | string | `tcp://0.0.0.0:26656` | P2P listen address |
| `PAW_P2P_EXTERNAL_ADDRESS` | string | - | External P2P address |
| `PAW_P2P_SEEDS` | string | - | Seed nodes |
| `PAW_P2P_PERSISTENT_PEERS` | string | - | Persistent peers |
| `PAW_P2P_PEX` | boolean | `true` | Enable peer exchange |
| `PAW_P2P_MAX_NUM_INBOUND_PEERS` | integer | `40` | Max inbound peers |
| `PAW_P2P_MAX_NUM_OUTBOUND_PEERS` | integer | `10` | Max outbound peers |
| **RPC/API** |
| `PAW_RPC_LADDR` | string | `tcp://127.0.0.1:26657` | RPC listen address |
| `PAW_RPC_ENDPOINT` | string | `http://localhost:26657` | RPC endpoint URL |
| `PAW_API_ADDRESS` | string | `tcp://0.0.0.0:1317` | REST API address |
| `PAW_API_ENABLE` | boolean | `true` | Enable REST API |
| `PAW_GRPC_ADDRESS` | string | `0.0.0.0:9090` | gRPC address |
| `PAW_GRPC_WEB_ADDRESS` | string | `0.0.0.0:9091` | gRPC-Web address |
| **Logging** |
| `PAW_LOG_LEVEL` | string | `info` | Global log level |
| `PAW_LOG_FORMAT` | string | `plain` | Log format |
| `PAW_LOG_MODULE_LEVELS` | string | - | Per-module levels |
| **Telemetry** |
| `PAW_TELEMETRY_ENABLED` | boolean | `false` | Enable telemetry |
| `PAW_TELEMETRY_PROMETHEUS_RETENTION` | integer | `600` | Metrics retention (seconds) |
| `PAW_TELEMETRY_GLOBAL_LABELS` | string | `[]` | Global metric labels |
| `PAW_PROMETHEUS_LISTEN_ADDR` | string | `:26660` | Prometheus endpoint |
| `PAW_TELEMETRY_HEALTH_PORT` | integer | `36661` | Health check endpoint port |
| **Security** |
| `PAW_KEYRING_BACKEND` | string | `os` | Keyring backend |
| `PAW_KEYRING_DIR` | string | `$PAW_HOME` | Keyring directory |
| `PAW_UNSAFE_SKIP_UPGRADES` | string | - | Skip upgrade heights |
| **Modules** |
| `PAW_DEX_ENABLED` | boolean | `true` | Enable DEX module |
| `PAW_ORACLE_ENABLED` | boolean | `true` | Enable Oracle module |
| `PAW_COMPUTE_ENABLED` | boolean | `true` | Enable Compute module |

---

## Best Practices

### 1. Use Environment-Specific Configuration

```bash
# .env.production
PAW_CHAIN_ID="paw-1"
PAW_LOG_LEVEL="info"
PAW_TELEMETRY_ENABLED=true
PAW_P2P_PEX=false  # Validator

# .env.development
PAW_CHAIN_ID="paw-local"
PAW_LOG_LEVEL="debug"
PAW_KEYRING_BACKEND="test"
PAW_TELEMETRY_ENABLED=false
```

Load with: `source .env.production`

### 2. Never Commit Secrets

```bash
# .gitignore
.env
.env.*
*.key
keyring-*

# Use secret management
export PAW_VALIDATOR_KEY=$(kubectl get secret paw-validator-key -o jsonpath='{.data.key}' | base64 -d)
```

### 3. Validate Variables

```bash
#!/bin/bash
# validate-env.sh

required_vars=(
  "PAW_HOME"
  "PAW_CHAIN_ID"
  "PAW_MONIKER"
)

for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "ERROR: $var is not set"
    exit 1
  fi
done

echo "Environment validation passed"
```

### 4. Use Configuration Management

**Docker Compose:**

```yaml
version: '3.8'
services:
  paw-node:
    image: paw-node:latest
    environment:
      PAW_HOME: /paw
      PAW_CHAIN_ID: ${CHAIN_ID}
      PAW_LOG_LEVEL: ${LOG_LEVEL:-info}
    env_file:
      - .env
```

**Kubernetes:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: paw-config
data:
  PAW_CHAIN_ID: "paw-mvp-1"
  PAW_LOG_LEVEL: "info"
  PAW_TELEMETRY_ENABLED: "true"

---
apiVersion: v1
kind: Secret
metadata:
  name: paw-secrets
type: Opaque
data:
  PAW_KEYRING_PASSWORD: <base64-encoded>
```

### 5. Document Custom Variables

If adding custom variables to your deployment:

```bash
# Custom variables for our deployment
export PAW_BACKUP_ENABLED=true           # Enable automatic backups
export PAW_BACKUP_INTERVAL=3600          # Backup every hour
export PAW_BACKUP_S3_BUCKET="paw-backups"  # S3 bucket for backups
```

Add to this documentation or maintain separate `CUSTOM_ENV.md`.

---

## Troubleshooting

### Variable Not Taking Effect

**Check precedence:**

```bash
# 1. Check if variable is set
echo $PAW_HOME

# 2. Check configuration file
grep "^home" ~/.paw/config/client.toml

# 3. Check default value
pawd config home

# 4. Force override
PAW_HOME=/custom/path pawd start
```

### Boolean Values Not Working

**Use lowercase:**

```bash
# Correct
export PAW_TELEMETRY_ENABLED=true

# Incorrect
export PAW_TELEMETRY_ENABLED=TRUE
export PAW_TELEMETRY_ENABLED=1
```

### Path Variables Not Expanding

**Use absolute paths:**

```bash
# Correct
export PAW_HOME="/home/user/.paw"

# Incorrect (~ may not expand in all contexts)
export PAW_HOME="~/.paw"

# Solution: Expand in shell
export PAW_HOME="$HOME/.paw"
```

---

## References

- [Cosmos SDK Configuration](https://docs.cosmos.network/main/run-node/run-node#configuring-the-node-using-app-toml-and-config-toml)
- [CometBFT Configuration](https://docs.cometbft.com/v0.38/core/configuration)
- [12-Factor App Methodology](https://12factor.net/config)

---

**For additional support, see:**
- [DEPLOYMENT_QUICKSTART.md](/docs/DEPLOYMENT_QUICKSTART.md)
- [VALIDATOR_OPERATOR_GUIDE.md](/docs/VALIDATOR_OPERATOR_GUIDE.md)
- [TROUBLESHOOTING.md](/docs/TROUBLESHOOTING.md)
