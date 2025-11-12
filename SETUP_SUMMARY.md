# PAW Blockchain - Go Project Setup Summary

## Overview

This document summarizes the core Go project structure setup for the PAW blockchain built on Cosmos SDK v0.50.3 and Tendermint v0.38.2.

## Project Structure Created

### Root Files

#### `go.mod`
- **Module**: `github.com/paw-chain/paw`
- **Go Version**: 1.21
- **Key Dependencies**:
  - `cosmos-sdk v0.50.3` - Core blockchain framework
  - `cometbft v0.38.2` - BFT consensus engine
  - `wasmd v0.50.0` - CosmWasm smart contract support
  - Standard Cosmos modules for bank, staking, governance, etc.

#### `Makefile`
Comprehensive build and development targets:
- **Build**: `make build` - Builds pawd daemon and pawcli
- **Install**: `make install` - Installs binaries to GOPATH
- **Test**: Multiple test targets (unit, integration, coverage, keeper, types)
- **Lint**: `make lint` - Runs golangci-lint
- **Format**: `make format` - Formats code with gofmt and goimports
- **Proto**: `make proto-gen` - Generates protobuf files
- **Localnet**: `make localnet-start` - Starts local testnet

#### `.gitignore`
Standard Go project ignores plus blockchain-specific exclusions:
- Build artifacts
- Chain data directories
- Generated protobuf files
- IDE files
- Tendermint/CometBFT config and data

### Command Line Interface

#### `cmd/pawd/` - Main Daemon
- **`main.go`**: Entry point for blockchain daemon
- **`cmd/root.go`**: Root command with all subcommands
  - Initializes SDK with "paw" prefix
  - Configures Tendermint with 4-second block times
  - Sets minimum gas price to 0.001upaw
  - Registers all module commands (tx, query, keys, etc.)
- **`cmd/genesis.go`**: Genesis account management commands

#### `cmd/pawcli/` - CLI Tool
- Shares the same command structure as pawd
- Used for client operations without running a full node

### Application Layer (`app/`)

#### `app.go`
Main application definition with:
- **PAWApp struct**: Extends Cosmos SDK BaseApp
- **Module Registration**:
  - Standard Cosmos modules (auth, bank, staking, gov, etc.)
  - Custom PAW modules (dex, compute, oracle)
- **Keeper Initialization**:
  - AccountKeeper, BankKeeper, StakingKeeper
  - Custom DEXKeeper, ComputeKeeper, OracleKeeper
- **Module Manager**: Orchestrates all module lifecycle hooks

#### `encoding.go`
Codec configuration:
- Protobuf codec setup
- Amino legacy codec support
- Transaction config with Ed25519 signing

#### `params.go`
Chain parameters:
- **Address Prefix**: "paw"
- **Coin Type**: 118 (Cosmos standard)
- **Bond Denom**: "upaw" (micro-PAW)
- **Default Gas Price**: 0.001upaw

#### `genesis_state.go`
Genesis state management and default genesis generation

### Custom Modules

## 1. DEX Module (`x/dex/`)

**Purpose**: Decentralized exchange with constant product AMM

### Files Created:
- **`module.go`**: Module definition and lifecycle hooks
- **`keeper/keeper.go`**: Core business logic (444 lines)
  - `CreatePool()`: Create liquidity pools
  - `Swap()`: Execute token swaps with 0.3% fee
  - `AddLiquidity()`: Add liquidity to pools
  - `RemoveLiquidity()`: Withdraw liquidity
  - `CalculateSwapAmount()`: AMM formula implementation
- **`keeper/keeper_genesis.go`**: Genesis initialization/export
- **`types/`**: Type definitions
  - `keys.go`: Store key constants
  - `genesis.go`: Genesis state
  - `params.go`: Module parameters (swap fees, slippage limits)
  - `types.go`: Pool struct and validation
  - `codec.go`: Message registration

### Protobuf Definitions:
- **`proto/paw/dex/v1/tx.proto`**: Transaction messages
  - MsgCreatePool
  - MsgAddLiquidity
  - MsgRemoveLiquidity
  - MsgSwap
- **`proto/paw/dex/v1/query.proto`**: Query service
- **`proto/paw/dex/v1/dex.proto`**: Core types (Pool, Params, GenesisState)

### Key Features:
- **Constant Product AMM**: x × y = k formula
- **Fee Structure**: 0.3% swap fee (0.25% to LPs, 0.05% to protocol)
- **Slippage Protection**: Configurable maximum slippage
- **Pool Management**: Create, add/remove liquidity
- **Token Ordering**: Consistent token pair ordering

## 2. Compute Module (`x/compute/`)

**Purpose**: AI workload verification and compute task management

### Files Created:
- **`module.go`**: Module definition
- **`keeper/keeper.go`**: State management and task verification
- **`types/`**: Type definitions
  - `keys.go`: Store keys for tasks and providers
  - `genesis.go`: Genesis state
  - `params.go`: Parameters (min stake, verification timeout, max retries)
  - `expected_keepers.go`: Bank keeper interface

### Protobuf Definitions:
- **`proto/paw/compute/v1/compute.proto`**:
  - Params (min_stake, verification_timeout, max_retries)
  - ComputeTask (id, requester, task_type, input/output, status)
  - GenesisState

### Key Features:
- **Minimum Stake**: 10,000 PAW required for compute providers
- **Verification Timeout**: 5 minutes default
- **Task Tracking**: Submitted tasks with status management
- **TEE Integration**: Designed for Trusted Execution Environment verification

## 3. Oracle Module (`x/oracle/`)

**Purpose**: Price feed aggregation and oracle services

### Files Created:
- **`module.go`**: Module definition
- **`keeper/keeper.go`**: Price feed management
- **`types/`**: Type definitions
  - `keys.go`: Store keys for price feeds
  - `genesis.go`: Genesis state
  - `params.go`: Parameters (min validators, update interval, expiry)
  - `expected_keepers.go`: Bank keeper interface

### Protobuf Definitions:
- **`proto/paw/oracle/v1/oracle.proto`**:
  - Params (min_validators, update_interval, expiry_duration)
  - PriceFeed (asset, price, timestamp, validators)
  - GenesisState

### Key Features:
- **Multi-Validator Consensus**: Minimum 3 validators per feed
- **Update Interval**: 1 minute minimum between updates
- **Price Expiry**: 5 minutes default expiry duration
- **Validator Tracking**: Records which validators submitted each price

### Scripts

#### `scripts/protocgen.sh`
Protobuf code generation script:
- Finds all .proto files
- Generates Go code with cosmos and gRPC plugins
- Moves generated files to correct locations

#### `scripts/localnet-start.sh`
Local testnet initialization:
- Cleans old data
- Initializes node with chain-id "paw-localnet-1"
- Creates validator key
- Adds genesis account with 1,000,000 PAW
- Creates genesis transaction
- Starts the blockchain

## Chain Parameters

Based on `docs/TECHNICAL_SPECIFICATION.md`:

### Consensus
- **Block Time**: 4 seconds
- **Finality**: Instant (Tendermint BFT)
- **Validators**: 25 genesis validators (max 125)
- **Byzantine Tolerance**: Up to 1/3 malicious nodes

### Timeouts
```yaml
timeoutPropose: 3000ms
timeoutPrevote: 1000ms
timeoutPrecommit: 1000ms
timeoutCommit: 0ms
```

### Block Limits
- **Max Block Size**: 2 MB
- **Max Gas Per Block**: 100,000,000 gas units
- **Throughput**: 500-2,500 TPS (depending on transaction complexity)

### Fees
- **Minimum Gas Price**: 0.001 upaw/gas
- **Fee Distribution**:
  - 50% burned (deflationary)
  - 30% to validators
  - 20% to treasury

### Staking
- **Minimum Validator Stake**: 100,000 PAW
- **Unbonding Period**: 21 days
- **Slashing**:
  - Double sign: 5% + permanent jail
  - Downtime: 0.01% + temporary jail

## Development Workflow

### 1. Build the Project
```bash
make build
```
This creates:
- `build/pawd` - Blockchain daemon
- `build/pawcli` - CLI tool

### 2. Install Binaries
```bash
make install
```
Installs to `$GOPATH/bin`

### 3. Generate Protobuf Files
```bash
make proto-gen
```
Generates Go code from `.proto` files

### 4. Run Tests
```bash
make test          # All tests
make test-unit     # Module unit tests only
make test-keeper   # Keeper tests
make test-coverage # With coverage report
```

### 5. Lint Code
```bash
make lint
```

### 6. Format Code
```bash
make format
```

### 7. Start Local Testnet
```bash
make localnet-start
```

## Module Integration Status

### ✅ Fully Implemented
- **DEX Module**: Complete AMM implementation with pool management
  - CreatePool, AddLiquidity, RemoveLiquidity, Swap
  - Constant product formula with fee calculation
  - Event emission and error handling

### 🚧 Skeleton/TODO
- **Compute Module**: Structure complete, core logic pending
  - Task submission and verification (TODO)
  - TEE integration (TODO)
  - Provider registration (TODO)

- **Oracle Module**: Structure complete, core logic pending
  - Price feed submission (TODO)
  - Validator consensus (TODO)
  - Price aggregation (TODO)

### 🔧 Partially Complete
- **App Integration**: Module registration complete, some TODOs:
  - WASM keeper initialization (commented out)
  - Module configurator setup (pending)
  - Begin/end blocker ordering (pending)
  - Export app state (pending)

## Next Steps

### Immediate Tasks
1. **Complete WASM Integration**: Uncomment and configure WasmKeeper in app.go
2. **Implement Module Services**: Register gRPC query/msg services
3. **Add Begin/End Blockers**: Set module execution order
4. **Implement Export Logic**: Complete app state export for genesis export

### Module Development
1. **Compute Module**:
   - Implement task submission messages
   - Add verification logic
   - Create provider registration
   - Integrate TEE verification

2. **Oracle Module**:
   - Implement price feed submission
   - Add validator consensus mechanism
   - Create price aggregation logic
   - Add price query endpoints

3. **DEX Module Enhancements**:
   - Add price impact calculation
   - Implement route optimization for multi-hop swaps
   - Add liquidity mining/rewards
   - Create DEX analytics queries

### Testing
1. Write comprehensive unit tests for all keepers
2. Add integration tests for module interactions
3. Create E2E tests for full transaction flows
4. Add benchmarks for performance-critical paths

### Documentation
1. Generate Swagger/OpenAPI docs from protobuf
2. Create module-specific documentation
3. Write developer guides
4. Add deployment instructions

## File Statistics

- **Total Go Files**: 64
- **Total Protobuf Files**: 5
- **Custom Modules**: 3 (dex, compute, oracle)
- **Lines of Code**: ~3,500+ (excluding generated files)

## Key Technologies

- **Language**: Go 1.21
- **Framework**: Cosmos SDK v0.50.3
- **Consensus**: CometBFT (Tendermint) v0.38.2
- **Smart Contracts**: CosmWasm v0.50.0
- **Serialization**: Protocol Buffers (protobuf)
- **Cryptography**: Ed25519 signatures
- **State Storage**: IAVL trees
- **Network**: libp2p

## References

- **Technical Specification**: `docs/TECHNICAL_SPECIFICATION.md`
- **Cosmos SDK Docs**: https://docs.cosmos.network/
- **CometBFT Docs**: https://docs.cometbft.com/
- **CosmWasm Docs**: https://docs.cosmwasm.com/

---

**Status**: Core structure complete, ready for module implementation
**Date**: 2025-11-12
**Version**: v0.1.0-alpha
