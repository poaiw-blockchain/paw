# PAW Blockchain - Production Roadmap

**Version:** 2.0
**Last Updated:** November 27, 2025
**Status:** Active Development → Testnet → Mainnet
**Chain ID:** paw-mainnet-1 (target)

---

## Executive Summary

PAW is a Cosmos SDK-based Layer-1 blockchain for verifiable AI compute with integrated DEX and Oracle functionality. This roadmap provides a structured path from current development state through local testnet, cloud testnet, and production mainnet deployment.

**Current State:**
- ✅ Core blockchain structure complete (Cosmos SDK v0.50.9, CometBFT)
- ✅ Three custom modules implemented: DEX, Oracle, Compute
- ✅ Comprehensive test infrastructure (83 test files, 10 test categories)
- ✅ Monitoring stack ready (Prometheus, Grafana, Jaeger, Alertmanager)
- ✅ Kubernetes manifests and deployment scripts
- ✅ P2P networking layer with discovery, reputation, security
- ✅ IBC integration with relayer configuration
- ⚠️ Build errors in DEX and Oracle keepers (sdkerrors migration needed)
- ⚠️ CosmWasm integration blocked (IBC initialization required)
- ❌ No GitHub Actions CI/CD (intentionally disabled)

---

## Table of Contents

1. [Current Project Assessment](#1-current-project-assessment)
2. [Phase 0: Critical Fixes (IMMEDIATE)](#phase-0-critical-fixes-immediate)
3. [Phase 1: Local Testnet Readiness](#phase-1-local-testnet-readiness)
4. [Phase 2: Cloud Testnet Deployment](#phase-2-cloud-testnet-deployment)
5. [Phase 3: Security Hardening](#phase-3-security-hardening)
6. [Phase 4: Production Preparation](#phase-4-production-preparation)
7. [Phase 5: Mainnet Launch](#phase-5-mainnet-launch)
8. [Phase 6: Post-Launch Operations](#phase-6-post-launch-operations)

---

## 1. Current Project Assessment

### 1.1 Existing Components (FOUND)

#### Core Blockchain
- ✅ **Application Layer** (`app/app.go`)
  - Cosmos SDK v0.50.9 integration
  - Module manager with all standard modules
  - Ante handlers for transaction validation
  - Store keys and keepers properly configured
  - IBC capability keeper registered

- ✅ **Custom Modules** (`x/`)
  - **x/dex**: AMM pools, swaps, liquidity, IBC transfers
  - **x/oracle**: Price feeds, aggregation, slashing hooks
  - **x/compute**: Job escrow, verification, IBC compute requests
  - **x/privacy**: Staging module (not production ready)

- ✅ **Command Line Interface** (`cmd/pawd`, `cmd/pawcli`)
  - Daemon and CLI binaries
  - Transaction and query commands
  - Genesis initialization tools

#### Protocol Buffers & API
- ✅ **Protobuf Definitions** (`proto/paw/`)
  - DEX messages (tx, query, types)
  - Oracle messages (tx, query, types)
  - Compute messages with ZK proof support
  - Buf configuration for code generation

- ✅ **OpenAPI Specification** (`docs/api/openapi.yaml`)
  - REST API documentation
  - WebSocket endpoints
  - Authentication schemas

#### Networking
- ✅ **P2P Layer** (`p2p/`)
  - Node discovery (DHT-based)
  - Protocol handlers
  - Reputation system
  - Security with mutual TLS
  - Snapshot support

- ✅ **IBC Integration** (`ibc/`)
  - Relayer configuration
  - Security guidelines
  - Cross-chain messaging support

#### Testing Infrastructure
- ✅ **Comprehensive Test Suites** (`tests/`)
  - Unit tests (integration, property-based)
  - E2E tests with CometMock
  - Security tests (adversarial, crypto, injection)
  - Chaos tests (resource exhaustion)
  - Load tests (k6, Locust, Go benchmarks)
  - Invariant tests (bank, staking, DEX, oracle)
  - Simulation tests
  - Byzantine fault tests
  - Differential tests
  - Gas metering tests
  - **Total: 83 test files**

#### DevOps & Infrastructure
- ✅ **Docker Support**
  - Multi-stage Dockerfile
  - Docker Compose for dev environment
  - Monitoring stack compose (Prometheus, Grafana)
  - Development, devnet, certbot configs

- ✅ **Kubernetes Manifests** (`k8s/`)
  - 19 YAML manifests
  - Node deployments, API deployments
  - Validator StatefulSets
  - Persistent volumes and storage
  - ConfigMaps and Secrets
  - Services and Ingress
  - HPA (Horizontal Pod Autoscaling)
  - Network policies
  - Monitoring deployment
  - Genesis configuration

- ✅ **Deployment Scripts** (`scripts/deploy/`)
  - deploy-k8s.sh (comprehensive K8s deployment)
  - deploy-docker.sh
  - deploy-node.sh
  - deploy-validator.sh
  - backup-state.sh, restore-state.sh
  - setup-testnet.sh
  - upgrade-chain.sh

- ✅ **DevNet Scripts** (`scripts/devnet/`)
  - GCP deployment automation
  - Node initialization
  - Smoke tests and health checks

#### Monitoring & Observability
- ✅ **Monitoring Stack** (`infra/monitoring/`)
  - Prometheus configuration with alert rules
  - Grafana dashboards (chain metrics)
  - Alertmanager configuration
  - Loki for log aggregation
  - Promtail for log collection
  - Jaeger for distributed tracing

- ✅ **Docker Compose Monitoring** (`compose/docker-compose.monitoring.yml`)
  - Prometheus (metrics collection)
  - Grafana (visualization)
  - Node Exporter (system metrics)
  - cAdvisor (container metrics)
  - Alertmanager (alert routing)
  - All with health checks and persistent storage

#### Automation & Tooling
- ✅ **Makefile** (648 lines)
  - 60+ make targets
  - Build, test, lint, format
  - Docker operations
  - Monitoring management
  - Load testing
  - Security auditing
  - Blockchain operations (init, start, reset)

- ✅ **Scripts** (`scripts/`)
  - 51 shell scripts total
  - Bootstrap, format, security audit
  - Load testing, benchmarking
  - TLS certificate generation
  - Cleanup utilities

- ✅ **Pre-commit Hooks** (`.pre-commit-config.yaml`)
  - Go formatting, vetting, linting
  - Security scanning (gosec)
  - Secret detection
  - Markdown, YAML linting
  - Protobuf linting (buf)

#### Documentation
- ✅ **Comprehensive Docs** (`docs/`)
  - 50+ markdown files
  - Technical Specification (75KB)
  - Whitepaper
  - API documentation with examples (Go, Python, JS, curl)
  - Architecture documentation
  - Security testing recommendations
  - Bug bounty program
  - IBC implementation guides
  - Integration guides
  - Upgrade procedures
  - P2P protocol specification

- ✅ **Module Documentation**
  - DEX security guides and audit reports
  - Oracle implementation and algorithms
  - Compute cryptographic verification guides
  - Provider integration documentation

#### Configuration
- ✅ **Genesis Files**
  - Mainnet genesis template (`config/genesis-mainnet.json`)
  - Foundation and ecosystem accounts configured
  - Consensus parameters defined

- ✅ **Node Configuration**
  - `infra/node-config.yaml`
  - Prometheus exporters configured
  - App.toml and config.toml templates

### 1.2 Missing or Incomplete Components

#### Critical Build Issues
- ❌ **DEX Keeper** (`x/dex/keeper/ibc_aggregation.go`)
  - Undefined: `sdkerrors.Wrapf`, `sdkerrors.Wrap`
  - Missing: `types.ErrSlippageExceeded`
  - Incorrect struct fields in `SwapResult`
  - **Impact:** Build fails, DEX module non-functional

- ❌ **Oracle Keeper** (`x/oracle/keeper/`)
  - Undefined: `types.ErrOracleDataUnavailable`
  - Incorrect IBC SendPacket signature (missing capability)
  - Undefined: `sdk.KVStorePrefixIterator`, `sdk.ZeroDec`
  - **Impact:** Build fails, Oracle module non-functional

#### Smart Contracts
- ⚠️ **CosmWasm Integration** (BLOCKED)
  - Dependencies present (go.mod includes CosmWasm v1.0.0)
  - Store keys registered
  - Keeper initialization commented out
  - Requires IBC initialization first
  - Test infrastructure ready (`testutil/integration/contracts.go`)
  - **Impact:** No smart contract support until IBC + CosmWasm enabled
  - **Status:** Proposal documented in `docs/proposals/SMART_CONTRACT_INTEGRATION_PROPOSAL.md`

- ❌ **No WASM Contracts**
  - 0 Rust files found
  - No compiled .wasm binaries
  - No contract examples
  - **Impact:** No reference contracts for testing

#### CI/CD
- ❌ **No GitHub Actions** (`.github/` directory missing)
  - Intentionally disabled per CLAUDE.md (cost reasons)
  - All testing done locally via pre-commit hooks
  - **Impact:** No automated testing on push/PR

#### Network Deployment
- ⚠️ **No Active Testnet**
  - Scripts ready but not deployed
  - No public RPC endpoints
  - No faucet service (archived in `archive/faucet/`)
  - **Impact:** Cannot test against live network

#### Additional Tools
- ⚠️ **Block Explorer** (archived in `archive/explorer/`)
  - Code exists but not deployed
  - Indexer, API, WebSocket hub implemented
  - **Status:** Needs deployment to cloud

- ⚠️ **Developer Portal** (archived in `archive/portal/`)
  - VitePress-based documentation site
  - Built but not hosted
  - **Status:** Ready for static hosting

### 1.3 Build Status

**Current Build:** ❌ FAILING

```
Error: undefined: sdkerrors.Wrapf (x/dex/keeper)
Error: undefined: types.ErrOracleDataUnavailable (x/oracle/keeper)
Error: IBC SendPacket signature mismatch (x/oracle/keeper)
```

**Root Cause:** Cosmos SDK v0.50 migration incomplete
- Old error handling API (`sdkerrors`) deprecated
- New API uses `errors.Join()` and module-specific errors
- IBC SendPacket API changed to require capability authentication

---

## Phase 0: Critical Fixes (IMMEDIATE)

**Goal:** Achieve successful build and basic node startup
**Duration:** 2-3 days
**Priority:** CRITICAL - All other work blocked until complete

### 0.1 Fix Build Errors

**Complexity: HIGH**

- [ ] **Migrate error handling to Cosmos SDK v0.50 patterns**
  - [ ] Replace all `sdkerrors.Wrapf()` with `errors.Join()` or `fmt.Errorf()`
  - [ ] Replace `sdkerrors.Wrap()` with module-specific error wrapping
  - [ ] Update imports to remove deprecated packages
  - [ ] Files affected:
    - `x/dex/keeper/ibc_aggregation.go` (10+ errors)
    - `x/oracle/keeper/ibc_prices.go`
    - `x/oracle/keeper/ibc_timeout.go`
  - [ ] Test after changes: `make test-keeper`

- [ ] **Define missing error types in module types**
  - [ ] Add `ErrSlippageExceeded` to `x/dex/types/errors.go`
  - [ ] Add `ErrOracleDataUnavailable` to `x/oracle/types/errors.go`
  - [ ] Ensure all custom errors are registered
  - [ ] Follow pattern from existing error definitions

- [ ] **Fix SwapResult struct usage**
  - [ ] Check `x/dex/types/types.go` for correct SwapResult fields
  - [ ] Update `x/dex/keeper/ibc_aggregation.go` line 221-224
  - [ ] Ensure field names match protobuf definitions
  - [ ] Regenerate protobuf if needed: `make proto-gen`

- [ ] **Fix IBC SendPacket calls**
  - [ ] Update signature in `x/oracle/keeper/ibc_prices.go` line 481
  - [ ] Add capability parameter (from scoped keeper)
  - [ ] Follow IBC v8 API: `SendPacket(ctx, capability, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)`
  - [ ] Reference working examples in `x/compute/keeper/ibc_compute.go`

- [ ] **Fix deprecated SDK functions**
  - [ ] Replace `sdk.KVStorePrefixIterator` with `storetypes.KVStorePrefixIterator`
  - [ ] Replace `sdk.ZeroDec()` with `math.LegacyZeroDec()`
  - [ ] Update imports: `cosmossdk.io/math` for math types
  - [ ] Files: `x/oracle/keeper/ibc_prices.go`, `x/oracle/keeper/invariants.go`

- [ ] **Verify build succeeds**
  - [ ] `make build` completes without errors
  - [ ] Both `pawd` and `pawcli` binaries created
  - [ ] Run quick sanity test: `./build/pawd version`

### 0.2 Basic Node Validation

**Complexity: LOW**

- [ ] **Initialize test node**
  - [ ] `./build/pawd init test-node --chain-id paw-test-1`
  - [ ] Verify `~/.paw/` directory created
  - [ ] Check `config/genesis.json` generated
  - [ ] Review `config/app.toml` and `config/config.toml`

- [ ] **Start node in development mode**
  - [ ] `./build/pawd start --minimum-gas-prices 0.001upaw`
  - [ ] Verify node starts without panics
  - [ ] Check logs for proper module initialization
  - [ ] Stop after successful startup (Ctrl+C)

- [ ] **Run basic unit tests**
  - [ ] `make test-unit` (quick module tests)
  - [ ] Address any immediate test failures
  - [ ] Ensure core types/keeper tests pass

### 0.3 Documentation Updates

**Complexity: LOW**

- [ ] **Update README.md**
  - [ ] Verify build instructions are current
  - [ ] Update prerequisites (Go 1.21+, not 1.25)
  - [ ] Add troubleshooting section for common build errors

- [ ] **Document fixes made**
  - [ ] Add notes to CHANGELOG.md
  - [ ] Update CONTRIBUTING.md with SDK migration notes
  - [ ] Create `docs/development/SDK_V050_MIGRATION.md` guide

**Phase 0 Completion Criteria:**
- ✅ `make build` succeeds without errors
- ✅ `pawd` and `pawcli` binaries functional
- ✅ Node initializes and starts in single-node mode
- ✅ Core unit tests pass
- ✅ Documentation updated

---

## Phase 1: Local Testnet Readiness

**Goal:** Multi-node local testnet with full module functionality
**Duration:** 1-2 weeks
**Prerequisites:** Phase 0 complete

### 1.1 Module Completion & Testing

#### DEX Module

**Complexity: MEDIUM**

- [ ] **Complete AMM implementation**
  - [ ] Verify constant-product formula (x * y = k)
  - [ ] Test pool creation with initial liquidity
  - [ ] Test swap calculations with fees
  - [ ] Implement slippage protection
  - [ ] Add liquidity provider token minting/burning
  - [ ] Test: `make test-keeper` for DEX

- [ ] **IBC token transfer integration**
  - [ ] Enable ICS-20 token transfers
  - [ ] Test cross-chain swaps (simulated)
  - [ ] Implement timeout and acknowledgment handlers
  - [ ] Verify escrow account management
  - [ ] Test: `tests/ibc/transfer_test.go`

- [ ] **Economic invariants**
  - [ ] Ensure liquidity pools maintain k=constant
  - [ ] Verify no token creation/destruction in swaps
  - [ ] Check LP token supply matches pool reserves
  - [ ] Test: `tests/invariants/dex_invariants_test.go`

- [ ] **CLI commands**
  - [ ] `pawcli tx dex create-pool`
  - [ ] `pawcli tx dex add-liquidity`
  - [ ] `pawcli tx dex remove-liquidity`
  - [ ] `pawcli tx dex swap`
  - [ ] `pawcli query dex pool`
  - [ ] `pawcli query dex pools`

#### Oracle Module

**Complexity: MEDIUM**

- [ ] **Price feed aggregation**
  - [ ] Implement vote submission from validators
  - [ ] Calculate median price from votes
  - [ ] Handle outlier detection (statistical)
  - [ ] Implement TWAP (time-weighted average price)
  - [ ] Test: `tests/property/oracle_properties_test.go`

- [ ] **Slashing integration**
  - [ ] Define slashing conditions (deviation threshold)
  - [ ] Integrate with staking module slashing hooks
  - [ ] Test malicious reporter punishment
  - [ ] Verify honest reporters not affected
  - [ ] Test: `tests/security/adversarial_test.go`

- [ ] **IBC price queries**
  - [ ] Implement cross-chain price requests
  - [ ] Handle timeouts gracefully
  - [ ] Cache recent prices for availability
  - [ ] Test: `tests/ibc/compute_ibc_test.go`

- [ ] **CLI commands**
  - [ ] `pawcli tx oracle submit-price`
  - [ ] `pawcli query oracle price [asset]`
  - [ ] `pawcli query oracle prices`
  - [ ] `pawcli query oracle reporters`

#### Compute Module

**Complexity: HIGH**

- [ ] **Job escrow system**
  - [ ] Implement job posting with collateral
  - [ ] Assign jobs to providers (reputation-based)
  - [ ] Escrow release on verification
  - [ ] Refund on timeout/failure
  - [ ] Test: `tests/integration/zk_verification_test.go`

- [ ] **Verification mechanism**
  - [ ] ZK proof verification (placeholder if no ZK yet)
  - [ ] TEE attestation verification
  - [ ] Multi-provider consensus
  - [ ] Fraud proof challenges
  - [ ] Test: `x/compute/SECURITY_TESTING_GUIDE.md` scenarios

- [ ] **Provider management**
  - [ ] Provider registration with stake
  - [ ] Reputation scoring system
  - [ ] Slashing for incorrect results
  - [ ] Provider selection algorithm
  - [ ] Test: `tests/benchmarks/compute_bench_test.go`

- [ ] **IBC compute requests**
  - [ ] Cross-chain job submission
  - [ ] Result verification and return
  - [ ] Handle cross-chain timeouts
  - [ ] Test: All IBC compute test files

- [ ] **CLI commands**
  - [ ] `pawcli tx compute register-provider`
  - [ ] `pawcli tx compute submit-job`
  - [ ] `pawcli tx compute verify-result`
  - [ ] `pawcli query compute job [id]`
  - [ ] `pawcli query compute provider [address]`

### 1.2 Complete Test Coverage

**Complexity: MEDIUM**

- [ ] **Unit test coverage > 80%**
  - [ ] Run: `make test-coverage`
  - [ ] Generate report: `go tool cover -html=coverage.txt`
  - [ ] Identify gaps in coverage
  - [ ] Add tests for uncovered code paths
  - [ ] Focus on keeper methods and message handlers

- [ ] **Integration tests**
  - [ ] Multi-module workflows (DEX + Oracle pricing)
  - [ ] IBC transfer + DEX swap
  - [ ] Compute job lifecycle end-to-end
  - [ ] Test: `make test-integration`

- [ ] **Property-based tests**
  - [ ] DEX invariants always hold
  - [ ] Oracle median calculation properties
  - [ ] Compute escrow balance correctness
  - [ ] Test: `make test-properties`

- [ ] **Simulation tests**
  - [ ] Long-running multi-operation simulations
  - [ ] Non-determinism detection
  - [ ] State import/export consistency
  - [ ] Test: `make test-simulation` (may take 30+ min)

- [ ] **Security tests**
  - [ ] Adversarial behavior (malicious txs)
  - [ ] Cryptographic correctness
  - [ ] Injection attack prevention
  - [ ] Test: `make test` in `tests/security/`

### 1.3 Local Multi-Node Testnet

**Complexity: MEDIUM**

- [ ] **Initialize 4-node local testnet**
  - [ ] Use script: `./scripts/localnet-start.sh`
  - [ ] Or manual: `make init-testnet`
  - [ ] Verify 4 validators configured
  - [ ] Check genesis file has all validators
  - [ ] Ensure unique node IDs and ports

- [ ] **Start all nodes**
  - [ ] Start validator nodes (ports 26656-26659)
  - [ ] Verify consensus (blocks producing)
  - [ ] Check p2p connectivity: `curl localhost:26657/net_info`
  - [ ] Monitor logs for errors

- [ ] **Test network operations**
  - [ ] Send tokens between nodes
  - [ ] Create DEX pool and execute swap
  - [ ] Submit oracle price votes
  - [ ] Post compute job
  - [ ] Verify state consistency across nodes

- [ ] **Validator operations**
  - [ ] Add new validator dynamically
  - [ ] Remove validator (jail/unjail)
  - [ ] Test slashing for downtime
  - [ ] Verify validator set updates

- [ ] **Upgrade testing**
  - [ ] Prepare upgrade proposal
  - [ ] Test coordinated upgrade
  - [ ] Verify state migration (if any)
  - [ ] Rollback procedure

### 1.4 Developer Experience

**Complexity: LOW**

- [ ] **CLI documentation**
  - [ ] Complete: `docs/guides/CLI_QUICK_REFERENCE.md`
  - [ ] Add examples for all commands
  - [ ] Create tutorial: "First Transaction on PAW"
  - [ ] Document common workflows

- [ ] **API documentation**
  - [ ] Ensure OpenAPI spec is current
  - [ ] Test all REST endpoints
  - [ ] Document WebSocket subscriptions
  - [ ] Add rate limiting details

- [ ] **Developer guide**
  - [ ] "Setting up local testnet" tutorial
  - [ ] "Interacting with DEX" guide
  - [ ] "Running an oracle reporter" guide
  - [ ] "Becoming a compute provider" guide

- [ ] **Troubleshooting guide**
  - [ ] Common errors and solutions
  - [ ] Network connectivity issues
  - [ ] Transaction failure debugging
  - [ ] Log interpretation guide

**Phase 1 Completion Criteria:**
- ✅ All modules functionally complete with CLI
- ✅ Test coverage > 80% on critical paths
- ✅ 4-node local testnet running stably
- ✅ All basic operations tested (send, swap, oracle, compute)
- ✅ Developer documentation complete

---

## Phase 2: Cloud Testnet Deployment

**Goal:** Public testnet accessible to external validators and users
**Duration:** 2-3 weeks
**Prerequisites:** Phase 1 complete

### 2.1 Infrastructure Setup

#### Cloud Provider Selection

**Complexity: LOW**

- [ ] **Choose cloud platform**
  - [ ] Option 1: GCP (scripts exist in `scripts/devnet/gcp-*.sh`)
  - [ ] Option 2: AWS (requires new scripts)
  - [ ] Option 3: DigitalOcean (simple, cost-effective)
  - [ ] Recommended: Start with GCP (existing automation)

- [ ] **Resource planning**
  - [ ] Validator nodes: 4 vCPU, 16GB RAM, 500GB SSD (4 nodes minimum)
  - [ ] Seed nodes: 2 vCPU, 8GB RAM, 100GB SSD (2 nodes)
  - [ ] RPC endpoints: 4 vCPU, 16GB RAM, 200GB SSD (2 nodes, load-balanced)
  - [ ] Monitoring: 2 vCPU, 8GB RAM, 100GB SSD (1 node)
  - [ ] Total estimated cost: $500-800/month

#### Kubernetes Cluster

**Complexity: MEDIUM**

- [ ] **Create K8s cluster**
  - [ ] Use GKE, EKS, or managed Kubernetes
  - [ ] 3-5 worker nodes (n1-standard-4 or equivalent)
  - [ ] Enable cluster autoscaling
  - [ ] Configure network policies
  - [ ] Setup persistent storage class

- [ ] **Deploy using K8s manifests**
  - [ ] Review all manifests in `k8s/`
  - [ ] Update namespace: `paw-testnet`
  - [ ] Run: `./scripts/deploy/deploy-k8s.sh`
  - [ ] Or manual: `kubectl apply -f k8s/`

- [ ] **Configure secrets**
  - [ ] Generate JWT secrets
  - [ ] Store validator keys securely (Kubernetes secrets)
  - [ ] Configure monitoring passwords
  - [ ] Setup TLS certificates (Let's Encrypt)

- [ ] **Deploy monitoring stack**
  - [ ] Apply: `kubectl apply -f k8s/monitoring-deployment.yaml`
  - [ ] Or use: `make monitoring-start` (Docker Compose alternative)
  - [ ] Verify Prometheus scraping targets
  - [ ] Import Grafana dashboards (`infra/monitoring/grafana-dashboards/`)

#### Networking

**Complexity: MEDIUM**

- [ ] **DNS configuration**
  - [ ] Register domain (e.g., `pawtestnet.network`)
  - [ ] RPC endpoint: `rpc.pawtestnet.network`
  - [ ] REST API: `api.pawtestnet.network`
  - [ ] WebSocket: `ws.pawtestnet.network`
  - [ ] Monitoring: `monitor.pawtestnet.network`

- [ ] **Load balancing**
  - [ ] Setup L7 load balancer for RPC
  - [ ] Configure health checks
  - [ ] Enable HTTPS (TLS 1.3)
  - [ ] Rate limiting at ingress

- [ ] **Firewall rules**
  - [ ] P2P port 26656: Open to all (peer discovery)
  - [ ] RPC port 26657: Load balanced, rate limited
  - [ ] API port 1317: Load balanced, rate limited
  - [ ] Prometheus 9090: Internal only
  - [ ] Grafana 3000: Public with authentication

### 2.2 Testnet Genesis & Configuration

**Complexity: MEDIUM**

- [ ] **Create testnet genesis**
  - [ ] Start from template: `config/genesis-mainnet.json`
  - [ ] Update chain-id: `paw-testnet-1`
  - [ ] Set genesis time (future timestamp)
  - [ ] Configure initial validators (4-6)
  - [ ] Allocate tokens to foundation/faucet

- [ ] **Token distribution**
  - [ ] Foundation allocation: 30%
  - [ ] Validator allocation: 20%
  - [ ] Faucet allocation: 10%
  - [ ] Community incentives: 40%
  - [ ] Total supply: 100,000,000 PAW

- [ ] **Governance parameters**
  - [ ] Voting period: 3 days (testnet)
  - [ ] Minimum deposit: 1000 PAW
  - [ ] Quorum: 33.4%
  - [ ] Pass threshold: 50%
  - [ ] Veto threshold: 33.4%

- [ ] **Economic parameters**
  - [ ] Minimum gas price: 0.001upaw
  - [ ] Block max gas: 100,000,000
  - [ ] Inflation rate: 7% annual
  - [ ] Community tax: 2%
  - [ ] DEX swap fee: 0.3%

- [ ] **Module-specific parameters**
  - [ ] Oracle voting period: 10 blocks
  - [ ] Oracle slash fraction: 0.01 (1%)
  - [ ] Compute escrow timeout: 100 blocks
  - [ ] Provider minimum stake: 10,000 PAW

### 2.3 Validator Onboarding

**Complexity: MEDIUM**

- [ ] **Validator documentation**
  - [ ] Create: `docs/guides/VALIDATOR_SETUP_GUIDE.md`
  - [ ] Hardware requirements
  - [ ] Installation steps
  - [ ] Key management best practices
  - [ ] Monitoring setup
  - [ ] Slashing conditions

- [ ] **Validator incentive program**
  - [ ] Allocate testnet tokens for validators
  - [ ] Define uptime requirements
  - [ ] Create leaderboard/dashboard
  - [ ] Plan bug bounty for testnet issues

- [ ] **Genesis validator coordination**
  - [ ] Collect gentx from validators
  - [ ] Verify validator keys
  - [ ] Aggregate into final genesis
  - [ ] Distribute genesis file
  - [ ] Coordinate launch time (all start together)

- [ ] **Post-launch validator support**
  - [ ] Setup Discord/Telegram channel
  - [ ] Monitor validator uptime
  - [ ] Provide technical support
  - [ ] Regular network health reports

### 2.4 Public Services

#### Faucet

**Complexity: LOW**

- [ ] **Deploy faucet service**
  - [ ] Use code from: `archive/faucet/`
  - [ ] Migrate to active deployment
  - [ ] Update dependencies
  - [ ] Configure rate limiting (1 request/hour/IP)
  - [ ] Allocate faucet wallet with tokens

- [ ] **Faucet endpoints**
  - [ ] Web UI: `faucet.pawtestnet.network`
  - [ ] API: POST `/api/claim` with address
  - [ ] Discord bot integration (optional)
  - [ ] Captcha for abuse prevention

#### Block Explorer

**Complexity: MEDIUM**

- [ ] **Deploy explorer**
  - [ ] Use code from: `archive/explorer/`
  - [ ] Deploy indexer service
  - [ ] Deploy API backend
  - [ ] Deploy frontend UI
  - [ ] Configure database (PostgreSQL)

- [ ] **Explorer features**
  - [ ] Block list and details
  - [ ] Transaction search
  - [ ] Address/account details
  - [ ] Validator list and voting power
  - [ ] DEX pool analytics
  - [ ] Oracle price history
  - [ ] Network statistics

- [ ] **Explorer URL**
  - [ ] `explorer.pawtestnet.network`
  - [ ] Or integrate with existing explorers (Mintscan, Big Dipper)

#### Status Page

**Complexity: LOW**

- [ ] **Deploy status monitor**
  - [ ] Use code from: `archive/status/`
  - [ ] Monitor RPC endpoints
  - [ ] Monitor API endpoints
  - [ ] Check block production
  - [ ] Alert on issues

- [ ] **Status page URL**
  - [ ] `status.pawtestnet.network`
  - [ ] Display uptime, latency
  - [ ] Historical incident log
  - [ ] Maintenance notifications

### 2.5 Monitoring & Alerting

**Complexity: MEDIUM**

- [ ] **Prometheus configuration**
  - [ ] Deploy using: `compose/docker-compose.monitoring.yml`
  - [ ] Or K8s: `k8s/monitoring-deployment.yaml`
  - [ ] Configure scrape targets (all nodes)
  - [ ] Import alert rules: `infra/monitoring/alert_rules.yml`

- [ ] **Grafana dashboards**
  - [ ] Import: `infra/monitoring/grafana-dashboards/paw-chain.json`
  - [ ] Chain metrics (block time, tx/s, validator voting power)
  - [ ] Node metrics (CPU, memory, disk, network)
  - [ ] Module metrics (DEX volume, oracle prices, compute jobs)
  - [ ] P2P metrics (peer count, bandwidth)

- [ ] **Alerting rules**
  - [ ] Block production stopped (>1 min no new blocks)
  - [ ] Validator missing blocks (>10% missed)
  - [ ] High RPC latency (>1s)
  - [ ] High error rates (>5%)
  - [ ] Low peer count (<4 peers)
  - [ ] Disk space low (<20%)

- [ ] **Alert routing**
  - [ ] Configure Alertmanager: `infra/monitoring/alertmanager.yml`
  - [ ] Email notifications
  - [ ] Slack/Discord webhook
  - [ ] PagerDuty integration (optional)

- [ ] **Logging**
  - [ ] Deploy Loki: `infra/logging/loki-config.yaml`
  - [ ] Deploy Promtail for log collection
  - [ ] Structured JSON logging
  - [ ] Log retention: 30 days

- [ ] **Tracing**
  - [ ] Deploy Jaeger: `infra/tracing/jaeger-config.yaml`
  - [ ] Enable tracing in chain config
  - [ ] Trace RPC requests
  - [ ] Trace cross-module calls

### 2.6 IBC & Interoperability

**Complexity: HIGH**

- [ ] **IBC relayer setup**
  - [ ] Choose relayer (Hermes, rly, or ts-relayer)
  - [ ] Configure: `ibc/relayer-config.yaml`
  - [ ] Connect to Cosmos Hub testnet (theta-testnet-001)
  - [ ] Or Osmosis testnet
  - [ ] Open channels for ICS-20 transfers

- [ ] **Cross-chain testing**
  - [ ] Send tokens from Cosmos Hub to PAW
  - [ ] Send tokens from PAW to Cosmos Hub
  - [ ] Test timeout scenarios
  - [ ] Test acknowledgment failures
  - [ ] Verify IBC escrow accounting

- [ ] **IBC security**
  - [ ] Follow: `ibc/RELAYER_SECURITY.md`
  - [ ] Secure relayer key management
  - [ ] Monitor for anomalous transfers
  - [ ] Implement channel governance (allow/deny lists)

**Phase 2 Completion Criteria:**
- ✅ Public testnet running with 4+ validators
- ✅ RPC, API, WebSocket accessible publicly
- ✅ Faucet, explorer, status page operational
- ✅ Monitoring dashboard live with alerting
- ✅ IBC channels open to at least one other chain
- ✅ Validator documentation complete

---

## Phase 3: Security Hardening

**Goal:** Production-grade security posture and audit readiness
**Duration:** 3-4 weeks
**Prerequisites:** Phase 2 complete, stable testnet

### 3.1 Security Audit Preparation

**Complexity: HIGH**

- [ ] **Code freeze for audit**
  - [ ] Tag release candidate: `v1.0.0-rc1`
  - [ ] Lock dependencies (no updates during audit)
  - [ ] Create audit branch
  - [ ] Document all known issues

- [ ] **Internal security review**
  - [ ] Run: `make security-audit`
  - [ ] Fix all high/critical issues from gosec
  - [ ] Address govulncheck findings
  - [ ] Run GitLeaks: `make scan-secrets`
  - [ ] Review: `security/report-latest.txt`

- [ ] **Third-party audit selection**
  - [ ] Request proposals from:
    - [ ] Trail of Bits
    - [ ] Informal Systems (Cosmos specialists)
    - [ ] Halborn
    - [ ] CertiK
  - [ ] Budget: $50,000 - $150,000
  - [ ] Timeline: 4-6 weeks

- [ ] **Audit scope definition**
  - [ ] Core consensus (CometBFT integration)
  - [ ] Custom modules (DEX, Oracle, Compute)
  - [ ] Cryptographic implementations
  - [ ] IBC integration
  - [ ] Economic security (tokenomics, fee market)
  - [ ] P2P networking and security

### 3.2 Cryptographic Security

**Complexity: HIGH**

- [ ] **Key management**
  - [ ] Audit key generation (entropy sources)
  - [ ] Secure key storage (encrypted at rest)
  - [ ] Key rotation procedures
  - [ ] HSM integration for validator keys (optional, recommended)
  - [ ] Document in: `docs/security/KEY_MANAGEMENT.md`

- [ ] **Signature verification**
  - [ ] Ensure Ed25519 signatures properly validated
  - [ ] Test malformed signature rejection
  - [ ] Verify replay attack prevention (nonces)
  - [ ] Test cross-network replay prevention (chain-id in signatures)

- [ ] **Randomness**
  - [ ] Audit all uses of random number generation
  - [ ] Ensure cryptographically secure RNG (not math/rand)
  - [ ] Document entropy sources
  - [ ] Test randomness quality (statistical tests)

- [ ] **ZK proof verification (Compute module)**
  - [ ] Audit ZK verifier implementation
  - [ ] Test with invalid proofs (must reject)
  - [ ] Performance benchmarks
  - [ ] Document: `x/compute/CRYPTOGRAPHIC_VERIFICATION.md` (exists)

### 3.3 Module Security Hardening

#### DEX Security

**Complexity: MEDIUM**

- [ ] **Reentrancy protection**
  - [ ] Audit all state modifications
  - [ ] Ensure checks-effects-interactions pattern
  - [ ] Add reentrancy guards where needed
  - [ ] Test with malicious contracts (if CosmWasm enabled)

- [ ] **Integer overflow/underflow**
  - [ ] Replace arithmetic with safe math (sdk.Int, math.SafeXXX)
  - [ ] Test boundary conditions (max uint64, etc.)
  - [ ] Fuzz test arithmetic operations

- [ ] **Economic attacks**
  - [ ] Sandwich attack mitigation (slippage limits)
  - [ ] Front-running protection (commitment schemes?)
  - [ ] Flash loan prevention (if applicable)
  - [ ] Test: `tests/security/adversarial_test.go`

- [ ] **Access control**
  - [ ] Verify only pool creators can modify certain params
  - [ ] Ensure proper authorization checks
  - [ ] Test unauthorized access attempts

- [ ] **Follow security guides**
  - [ ] Review: `x/dex/SECURITY_IMPLEMENTATION_GUIDE.md`
  - [ ] Address all items in `x/dex/SECURITY_AUDIT_REPORT.md`

#### Oracle Security

**Complexity: MEDIUM**

- [ ] **Data manipulation attacks**
  - [ ] Test outlier detection (statistical thresholds)
  - [ ] Verify median calculation correctness
  - [ ] Test with malicious reporter data
  - [ ] Ensure slashing triggers appropriately

- [ ] **DoS resistance**
  - [ ] Rate limit price submissions
  - [ ] Prevent spam votes
  - [ ] Gas limits on vote processing
  - [ ] Test: `tests/chaos/resource_exhaustion_test.go`

- [ ] **Timestamp manipulation**
  - [ ] Verify block time used, not reporter time
  - [ ] TWAP calculations use on-chain timestamps
  - [ ] Test with future/past timestamps

- [ ] **Follow security guides**
  - [ ] Review: `x/oracle/SECURITY_AUDIT_REPORT.md`
  - [ ] Implement statistical outlier detection per: `x/oracle/STATISTICAL_OUTLIER_DETECTION.md`

#### Compute Security

**Complexity: HIGH**

- [ ] **Escrow security**
  - [ ] Verify escrow locks funds correctly
  - [ ] Test escrow release only on valid proof
  - [ ] Test refund on timeout
  - [ ] Prevent escrow draining attacks

- [ ] **Provider reputation**
  - [ ] Secure reputation scoring (prevent gaming)
  - [ ] Slashing for incorrect results
  - [ ] Test Sybil attack resistance

- [ ] **Verification integrity**
  - [ ] TEE attestation validation
  - [ ] ZK proof correctness
  - [ ] Multi-provider consensus threshold
  - [ ] Test with malicious provider colluding

- [ ] **Follow security guides**
  - [ ] Review: `x/compute/SECURITY_AUDIT_REPORT.md`
  - [ ] Follow testing in: `x/compute/SECURITY_TESTING_GUIDE.md`
  - [ ] Audit cryptography per: `x/compute/CRYPTOGRAPHIC_VERIFICATION.md`

### 3.4 Network Security

**Complexity: MEDIUM**

- [ ] **P2P security**
  - [ ] Enable mutual TLS (mTLS) between validators
  - [ ] Peer reputation system active
  - [ ] Rate limiting on P2P messages
  - [ ] DDoS protection (connection limits)
  - [ ] Test: `tests/byzantine/` test suite

- [ ] **Sybil attack resistance**
  - [ ] Stake-weighted peer selection
  - [ ] Limit connections per IP
  - [ ] Reputation decay for misbehavior

- [ ] **Eclipse attack prevention**
  - [ ] Maintain diverse peer set
  - [ ] Seed node diversity (multiple providers)
  - [ ] Monitor peer churn rates

- [ ] **Follow P2P spec**
  - [ ] Review: `docs/p2p-protocol-spec.md`
  - [ ] Implement all security recommendations

### 3.5 API & RPC Security

**Complexity: MEDIUM**

- [ ] **Rate limiting**
  - [ ] Implement at ingress (nginx/load balancer)
  - [ ] Per-IP limits: 100 req/min
  - [ ] Per-endpoint limits (expensive queries: 10 req/min)
  - [ ] Authenticated users: higher limits

- [ ] **Input validation**
  - [ ] Validate all request parameters
  - [ ] Reject malformed transactions early
  - [ ] Sanitize query inputs (SQL injection, even though using KV store)
  - [ ] Test: `tests/security/injection_test.go`

- [ ] **Authentication & Authorization**
  - [ ] API key system for privileged endpoints
  - [ ] JWT tokens with short expiry
  - [ ] Admin endpoints: require strong auth
  - [ ] Test: `tests/security/auth_test.go`

- [ ] **CORS & CSRF**
  - [ ] Strict CORS policy (whitelist origins)
  - [ ] CSRF tokens for state-changing operations
  - [ ] Secure cookie flags (HttpOnly, Secure, SameSite)

- [ ] **TLS configuration**
  - [ ] TLS 1.3 only
  - [ ] Strong cipher suites
  - [ ] Certificate pinning for validators (mTLS)
  - [ ] HSTS headers

### 3.6 Operational Security

**Complexity: MEDIUM**

- [ ] **Secrets management**
  - [ ] Never commit secrets to git (verify with gitleaks)
  - [ ] Use environment variables or secret managers (Vault, AWS Secrets Manager)
  - [ ] Rotate secrets regularly (quarterly)
  - [ ] Document in: `docs/security/SECRETS_MANAGEMENT.md`

- [ ] **Access control**
  - [ ] Principle of least privilege
  - [ ] MFA for all production access
  - [ ] Separate staging and production environments
  - [ ] Audit logs for all admin actions

- [ ] **Incident response plan**
  - [ ] Define severity levels
  - [ ] Response procedures for each level
  - [ ] Communication plan (users, validators)
  - [ ] Recovery procedures (rollback, state fork)
  - [ ] Document in: `docs/security/INCIDENT_RESPONSE.md`

- [ ] **Backup & recovery**
  - [ ] Automated state backups (daily)
  - [ ] Test restoration procedure (monthly)
  - [ ] Geographic redundancy
  - [ ] Document in: `docs/security/BACKUP_RECOVERY.md`

### 3.7 Penetration Testing

**Complexity: HIGH**

- [ ] **Internal pen test**
  - [ ] Test RPC/API for exploits
  - [ ] Attempt unauthorized transactions
  - [ ] Try to crash nodes
  - [ ] Test P2P attacks
  - [ ] Document findings

- [ ] **External pen test (optional)**
  - [ ] Hire professional pen testers
  - [ ] Full network attack simulation
  - [ ] Social engineering (phishing validators)
  - [ ] Budget: $20,000 - $50,000

- [ ] **Bug bounty program**
  - [ ] Launch on Immunefi or HackerOne
  - [ ] Rewards: $500 (low) to $50,000 (critical)
  - [ ] Scope: All chain code, APIs, infrastructure
  - [ ] Template exists: `docs/BUG_BOUNTY.md`

**Phase 3 Completion Criteria:**
- ✅ Third-party security audit complete (report received)
- ✅ All critical/high findings addressed
- ✅ Cryptographic implementations validated
- ✅ Penetration testing complete
- ✅ Bug bounty program active
- ✅ Security documentation complete

---

## Phase 4: Production Preparation

**Goal:** Mainnet-ready code, infrastructure, and community
**Duration:** 2-3 weeks
**Prerequisites:** Phase 3 complete, audit passed

### 4.1 Code Finalization

**Complexity: MEDIUM**

- [ ] **Address audit findings**
  - [ ] Implement all required fixes from security audit
  - [ ] Re-audit critical changes
  - [ ] Get auditor sign-off on fixes

- [ ] **Code freeze**
  - [ ] Tag release: `v1.0.0`
  - [ ] Lock go.mod (no dependency changes)
  - [ ] Final test suite run (all tests must pass)
  - [ ] Generate checksums for binaries

- [ ] **Binary builds**
  - [ ] Build for Linux (amd64, arm64)
  - [ ] Build for macOS (amd64, arm64)
  - [ ] Build for Windows (amd64)
  - [ ] Reproducible builds (deterministic)
  - [ ] Sign binaries (PGP or code signing cert)

- [ ] **Release notes**
  - [ ] Comprehensive changelog
  - [ ] Known issues (if any)
  - [ ] Upgrade instructions (from testnet)
  - [ ] Breaking changes highlighted

### 4.2 Genesis Preparation

**Complexity: HIGH**

- [ ] **Mainnet genesis design**
  - [ ] Start from: `config/genesis-mainnet.json`
  - [ ] Chain ID: `paw-mainnet-1`
  - [ ] Genesis time: Coordinated launch timestamp
  - [ ] Initial validators (minimum 4, target 20+)

- [ ] **Token economics**
  - [ ] Total supply: 100,000,000 PAW (or as designed)
  - [ ] Distribution:
    - [ ] Foundation: 20%
    - [ ] Team: 15% (vesting 4 years)
    - [ ] Investors: 10% (vesting 2 years)
    - [ ] Ecosystem fund: 25%
    - [ ] Community airdrop: 10%
    - [ ] Validator incentives: 10%
    - [ ] Liquidity mining: 10%
  - [ ] Vesting schedules implemented (x/auth/vesting)

- [ ] **Governance parameters (production)**
  - [ ] Voting period: 7 days
  - [ ] Minimum deposit: 10,000 PAW
  - [ ] Quorum: 40%
  - [ ] Pass threshold: 50%
  - [ ] Veto threshold: 33.4%

- [ ] **Economic parameters (production)**
  - [ ] Minimum gas price: 0.001upaw
  - [ ] Inflation: 7% annually (adjust as needed)
  - [ ] Community tax: 2%
  - [ ] DEX swap fee: 0.3%
  - [ ] Max validators: 100

- [ ] **Module parameters (production)**
  - [ ] Oracle voting period: 20 blocks
  - [ ] Oracle slash fraction: 0.05 (5%)
  - [ ] Compute provider stake: 100,000 PAW
  - [ ] Compute job timeout: 1,000 blocks

### 4.3 Infrastructure Scaling

**Complexity: HIGH**

- [ ] **Production K8s cluster**
  - [ ] 10+ worker nodes (higher capacity than testnet)
  - [ ] Multi-zone/region deployment (high availability)
  - [ ] Autoscaling configured (10-50 nodes)
  - [ ] Node pools: validators, RPC, API, monitoring

- [ ] **Validator node specs**
  - [ ] 8 vCPU, 32GB RAM, 1TB NVMe SSD
  - [ ] Dedicated nodes (not shared with RPC)
  - [ ] Low-latency networking
  - [ ] DDoS protection

- [ ] **RPC/API node specs**
  - [ ] 8 vCPU, 32GB RAM, 500GB SSD
  - [ ] Auto-scaling (based on request load)
  - [ ] CDN for static assets (if any)
  - [ ] Global load balancing (Cloudflare, Akamai)

- [ ] **Database scaling**
  - [ ] PostgreSQL for explorer (if self-hosted)
  - [ ] Managed database service (RDS, Cloud SQL)
  - [ ] Read replicas for queries
  - [ ] Backup retention: 30 days

- [ ] **Monitoring scaling**
  - [ ] Prometheus federation (multiple Prometheus instances)
  - [ ] Long-term storage (Thanos or Cortex)
  - [ ] High-availability Grafana
  - [ ] Alertmanager clustering

### 4.4 CosmWasm Integration (if required for mainnet)

**Complexity: HIGH**

- [ ] **Complete IBC initialization**
  - [ ] Initialize capability keeper
  - [ ] Initialize IBC keeper
  - [ ] Initialize transfer keeper
  - [ ] Wire all IBC modules in app.go
  - [ ] Test IBC transfers end-to-end

- [ ] **Enable CosmWasm**
  - [ ] Uncomment keeper initialization (app.go:179-180)
  - [ ] Configure production settings:
    - [ ] Upload access: AllowNobody (governance only)
    - [ ] SmartQueryGasLimit: 3,000,000
    - [ ] MemoryCacheSize: 100
    - [ ] ContractDebugMode: false
  - [ ] Register module in module manager
  - [ ] Test contract upload via governance

- [ ] **Contract deployment process**
  - [ ] Governance proposal for contract upload permission
  - [ ] Vote on proposal
  - [ ] Upload contract (if approved)
  - [ ] Instantiate via governance or permissioned actors
  - [ ] Document process in: `docs/guides/CONTRACT_DEPLOYMENT.md`

- [ ] **Reference contracts (optional)**
  - [ ] Deploy CW20 token standard
  - [ ] Deploy CW721 NFT standard
  - [ ] Deploy AMM pool contract (if using CosmWasm DEX)
  - [ ] Test all contracts thoroughly

### 4.5 Legal & Compliance

**Complexity: MEDIUM**

- [ ] **Terms of Service**
  - [ ] Draft ToS for API/RPC usage
  - [ ] Publish on website
  - [ ] Require acceptance for faucet/services

- [ ] **Privacy Policy**
  - [ ] GDPR compliance (if serving EU)
  - [ ] Data collection disclosure
  - [ ] Cookie policy

- [ ] **Disclaimers**
  - [ ] Investment disclaimer (not financial advice)
  - [ ] Experimental software warning
  - [ ] No warranty clause

- [ ] **Regulatory review (if applicable)**
  - [ ] Consult legal counsel
  - [ ] Token classification (security vs utility)
  - [ ] KYC/AML requirements (if any)

### 4.6 Community & Marketing

**Complexity: MEDIUM**

- [ ] **Website**
  - [ ] Professional landing page
  - [ ] Documentation portal (deploy `archive/portal/`)
  - [ ] Blog for announcements
  - [ ] Link to explorer, faucet, monitoring

- [ ] **Social media**
  - [ ] Twitter/X account
  - [ ] Discord server (for community + validators)
  - [ ] Telegram group (optional)
  - [ ] GitHub organization (already exists)

- [ ] **Community incentives**
  - [ ] Airdrop for early users (if planned)
  - [ ] Liquidity mining for DEX
  - [ ] Validator delegation incentives
  - [ ] Developer grants program

- [ ] **Partnerships**
  - [ ] Integrate with wallets (Keplr, Leap, Cosmostation)
  - [ ] List on DEX aggregators (Osmosis frontend, etc.)
  - [ ] Partner with other Cosmos chains for IBC

- [ ] **Launch announcement**
  - [ ] Press release
  - [ ] Blog post with roadmap
  - [ ] Twitter campaign
  - [ ] AMA (Ask Me Anything) sessions

**Phase 4 Completion Criteria:**
- ✅ Mainnet genesis.json ready and tested
- ✅ Production infrastructure deployed and load-tested
- ✅ v1.0.0 release candidate tagged and signed
- ✅ Legal documents published
- ✅ Community channels active with engaged users
- ✅ Marketing materials ready

---

## Phase 5: Mainnet Launch

**Goal:** Successful mainnet genesis and stable initial operation
**Duration:** 1 week (launch event) + 2 weeks (stabilization)
**Prerequisites:** Phase 4 complete

### 5.1 Genesis Validator Coordination

**Complexity: HIGH**

- [ ] **Validator recruitment**
  - [ ] Target: 20-30 genesis validators
  - [ ] Vet validators (reputation, infrastructure)
  - [ ] Sign agreements (if any)
  - [ ] Provide technical support

- [ ] **Gentx collection**
  - [ ] Deadline for gentx submission (1 week before launch)
  - [ ] Validate each gentx:
    - [ ] Correct chain-id
    - [ ] Valid signatures
    - [ ] Stake amounts within limits
  - [ ] Aggregate all gentx into final genesis

- [ ] **Genesis file distribution**
  - [ ] Publish final genesis.json (24 hours before launch)
  - [ ] Publish checksums (SHA256)
  - [ ] Make available via:
    - [ ] GitHub release
    - [ ] IPFS
    - [ ] Direct download from website
  - [ ] Validators verify checksums match

- [ ] **Launch dry run**
  - [ ] Simulate launch with subset of validators
  - [ ] Test coordinated start
  - [ ] Identify potential issues
  - [ ] Fix and re-test

### 5.2 Launch Day Operations

**Complexity: HIGH**

- [ ] **Pre-launch checklist (T-24 hours)**
  - [ ] All validators confirm readiness
  - [ ] Monitoring dashboards configured
  - [ ] Incident response team on standby
  - [ ] Communication channels open (Discord, Telegram)
  - [ ] Final genesis file distributed

- [ ] **Coordinated start (T-0)**
  - [ ] Genesis time reached
  - [ ] All validators start nodes simultaneously
  - [ ] Monitor for first block production
  - [ ] Target: Block 1 within 10 seconds
  - [ ] Celebrate first block! 🎉

- [ ] **Initial block monitoring (T+0 to T+1 hour)**
  - [ ] Verify block production is stable
  - [ ] Check validator voting participation (target: >66%)
  - [ ] Monitor for consensus failures
  - [ ] Watch for unexpected errors in logs
  - [ ] Verify P2P network is healthy (peer counts)

- [ ] **First transactions (T+1 to T+4 hours)**
  - [ ] Foundation sends test transactions
  - [ ] Verify transaction processing
  - [ ] Test DEX pool creation
  - [ ] Test oracle price submission
  - [ ] Test compute job posting (if providers ready)

- [ ] **IBC activation (T+4 to T+24 hours)**
  - [ ] Start IBC relayers
  - [ ] Open channels to Cosmos Hub
  - [ ] Open channels to Osmosis (if applicable)
  - [ ] Test cross-chain transfers
  - [ ] Monitor for IBC packet failures

### 5.3 Post-Launch Monitoring (Week 1)

**Complexity: MEDIUM**

- [ ] **24/7 monitoring**
  - [ ] On-call rotation for team
  - [ ] Monitor Grafana dashboards continuously
  - [ ] Watch for alerts (Alertmanager)
  - [ ] Daily health reports

- [ ] **Key metrics to watch**
  - [ ] Block time (target: 4 seconds, must not drift)
  - [ ] Validator participation (>90% expected)
  - [ ] Transaction throughput (TPS)
  - [ ] RPC/API latency (<500ms)
  - [ ] P2P peer count (>20 peers per node)
  - [ ] Disk usage (should not grow too fast)

- [ ] **Community support**
  - [ ] Active in Discord/Telegram
  - [ ] Respond to validator issues quickly
  - [ ] Publish daily status updates
  - [ ] Address user questions

- [ ] **Incident response**
  - [ ] Document any issues encountered
  - [ ] Coordinate fixes if needed
  - [ ] Communication plan for outages
  - [ ] Post-mortem for incidents

### 5.4 Service Activation

**Complexity: MEDIUM**

- [ ] **Public RPC/API (T+0)**
  - [ ] Verify endpoints accessible
  - [ ] Test rate limiting works
  - [ ] Monitor request volumes
  - [ ] Scale if needed

- [ ] **Faucet (T+1 day)**
  - [ ] Activate faucet service
  - [ ] Monitor for abuse
  - [ ] Adjust rate limits if spammed

- [ ] **Block Explorer (T+1 day)**
  - [ ] Start indexer
  - [ ] Verify data indexing correctly
  - [ ] Launch frontend
  - [ ] Announce to community

- [ ] **DEX UI (T+2 days, if applicable)**
  - [ ] Deploy frontend interface
  - [ ] Test pool creation
  - [ ] Test swaps
  - [ ] Monitor for issues

- [ ] **Wallet integrations (T+1 week)**
  - [ ] Work with Keplr to add PAW
  - [ ] Work with Leap, Cosmostation
  - [ ] Test wallet functionality
  - [ ] Announce availability

### 5.5 Governance Activation

**Complexity: LOW**

- [ ] **First governance proposal (symbolic)**
  - [ ] Proposal: "PAW Mainnet Launch Successful"
  - [ ] Tests governance voting process
  - [ ] Engages community in governance
  - [ ] Voting period: 7 days

- [ ] **Parameter adjustment proposals (as needed)**
  - [ ] Monitor network performance
  - [ ] Propose changes if needed (inflation, gas prices, etc.)
  - [ ] Community discussion before proposal
  - [ ] Follow governance process

### 5.6 Contingency Plans

**Complexity: HIGH**

- [ ] **Chain halt procedure**
  - [ ] If consensus breaks: coordinate validator stop
  - [ ] Investigate root cause
  - [ ] Prepare patch release
  - [ ] Coordinate restart with validators
  - [ ] Document in: `docs/upgrades/EMERGENCY_PROCEDURES.md`

- [ ] **State rollback procedure**
  - [ ] If critical bug found: halt chain
  - [ ] Export state before bug
  - [ ] Apply fix
  - [ ] Import state with fixed binary
  - [ ] Restart network

- [ ] **Hard fork procedure (last resort)**
  - [ ] If irrecoverable issue: plan new genesis
  - [ ] Snapshot state at specific height
  - [ ] Generate new genesis with snapshot
  - [ ] Coordinate migration
  - [ ] New chain-id: `paw-mainnet-2`

**Phase 5 Completion Criteria:**
- ✅ Mainnet producing blocks consistently for 7+ days
- ✅ Validator participation >90%
- ✅ All public services operational (RPC, API, explorer, faucet)
- ✅ IBC channels open and functional
- ✅ No critical incidents or severe bugs
- ✅ Community actively using the network

---

## Phase 6: Post-Launch Operations

**Goal:** Sustainable mainnet operation and continuous improvement
**Duration:** Ongoing
**Prerequisites:** Phase 5 complete, stable mainnet

### 6.1 Ongoing Monitoring & Maintenance

**Complexity: LOW (but ongoing effort)**

- [ ] **Daily health checks**
  - [ ] Review Grafana dashboards
  - [ ] Check alert status (Alertmanager)
  - [ ] Monitor validator uptime
  - [ ] Review transaction volumes
  - [ ] Check for anomalies

- [ ] **Weekly reports**
  - [ ] Network statistics (blocks, txs, validators)
  - [ ] DEX volume and TVL
  - [ ] Oracle price accuracy
  - [ ] Compute job completion rates
  - [ ] Publish to community (blog, Twitter)

- [ ] **Monthly reviews**
  - [ ] Security review (check for new vulnerabilities)
  - [ ] Performance review (optimize slow queries)
  - [ ] Cost review (infrastructure spending)
  - [ ] Dependency updates (Go, Cosmos SDK, libraries)

- [ ] **Quarterly audits**
  - [ ] External security audit (recommended annually at minimum)
  - [ ] Code audit (check for technical debt)
  - [ ] Compliance audit (legal/regulatory)

### 6.2 Network Upgrades

**Complexity: MEDIUM (per upgrade)**

- [ ] **Upgrade planning**
  - [ ] Identify features for next version
  - [ ] Community discussion and feedback
  - [ ] Governance proposal for upgrade
  - [ ] Set upgrade height (coordinated)

- [ ] **Testing upgrades**
  - [ ] Test on testnet first
  - [ ] Simulate upgrade with validators
  - [ ] Verify state migration (if any)
  - [ ] Document upgrade procedure

- [ ] **Coordinated upgrade execution**
  - [ ] Announce upgrade date (2+ weeks notice)
  - [ ] Provide upgrade binaries
  - [ ] Validator coordination (Discord/Telegram)
  - [ ] Execute upgrade at specified height
  - [ ] Monitor for successful completion

- [ ] **Post-upgrade validation**
  - [ ] Verify new features work
  - [ ] Check for regressions
  - [ ] Monitor performance
  - [ ] Address issues quickly

### 6.3 Community Growth

**Complexity: MEDIUM**

- [ ] **Developer engagement**
  - [ ] Hackathons (quarterly)
  - [ ] Grants program (allocate from ecosystem fund)
  - [ ] Developer documentation improvements
  - [ ] Example projects and tutorials

- [ ] **User acquisition**
  - [ ] Marketing campaigns
  - [ ] Partnerships with dApps
  - [ ] Airdrops and incentives
  - [ ] Referral programs

- [ ] **Ecosystem development**
  - [ ] Support projects building on PAW
  - [ ] Integrate with DeFi protocols
  - [ ] NFT marketplace support (if CosmWasm enabled)
  - [ ] Gaming integrations

### 6.4 Feature Development

**Complexity: VARIES**

- [ ] **Smart contract ecosystem (if CosmWasm enabled)**
  - [ ] Reference contracts (CW20, CW721)
  - [ ] DEX aggregator contracts
  - [ ] Lending protocols
  - [ ] DAO frameworks

- [ ] **DEX enhancements**
  - [ ] Concentrated liquidity (Uniswap v3 style)
  - [ ] Limit orders
  - [ ] Cross-chain swaps via IBC
  - [ ] DEX aggregation

- [ ] **Oracle improvements**
  - [ ] More asset support
  - [ ] Additional price feed sources
  - [ ] Historical price data API
  - [ ] Oracle reputation system

- [ ] **Compute network growth**
  - [ ] Onboard more compute providers
  - [ ] Support more AI/ML frameworks
  - [ ] ZK proof improvements (performance)
  - [ ] Privacy-preserving computation

- [ ] **IBC expansion**
  - [ ] Connect to more chains (all major Cosmos chains)
  - [ ] IBC middleware development
  - [ ] Cross-chain governance
  - [ ] Interchain security (consumer chain?)

### 6.5 Governance Maturation

**Complexity: LOW**

- [ ] **Decentralize governance**
  - [ ] Reduce foundation control over time
  - [ ] Increase community proposal frequency
  - [ ] Delegate decision-making to token holders

- [ ] **Governance tooling**
  - [ ] Better UI for proposals (governance dashboard)
  - [ ] Forum for discussion (Commonwealth, Discourse)
  - [ ] Voting analytics

- [ ] **Parameter optimization**
  - [ ] Regular review of economic parameters
  - [ ] Adjust based on network usage
  - [ ] Community-driven parameter proposals

### 6.6 Long-Term Sustainability

**Complexity: HIGH**

- [ ] **Economic sustainability**
  - [ ] Ensure fee revenue covers validator costs
  - [ ] Monitor inflation and adjust if needed
  - [ ] Treasury management (ecosystem fund)

- [ ] **Decentralization**
  - [ ] Reduce reliance on foundation infrastructure
  - [ ] Encourage independent RPC providers
  - [ ] Distribute validator power (no >33% concentration)

- [ ] **Ecosystem funding**
  - [ ] Allocate funds for development
  - [ ] Support critical infrastructure (explorers, wallets)
  - [ ] Long-term research and development

---

## Summary of Estimated Timelines

| Phase | Duration | Cumulative | Critical Path |
|-------|----------|------------|---------------|
| Phase 0: Critical Fixes | 2-3 days | 3 days | YES |
| Phase 1: Local Testnet | 1-2 weeks | 2.5 weeks | YES |
| Phase 2: Cloud Testnet | 2-3 weeks | 5.5 weeks | YES |
| Phase 3: Security | 3-4 weeks | 9.5 weeks | YES |
| Phase 4: Production Prep | 2-3 weeks | 12.5 weeks | YES |
| Phase 5: Mainnet Launch | 3 weeks | 15.5 weeks | YES |
| Phase 6: Post-Launch | Ongoing | - | NO |

**Total time to mainnet:** ~4 months (assuming no major blockers)

---

## Risk Assessment & Mitigation

### High-Risk Items

1. **Build Failures (Phase 0)**
   - Risk: Cannot compile due to SDK v0.50 migration issues
   - Mitigation: Prioritize error fixes, reference working Cosmos SDK v0.50 chains
   - Impact if delayed: All other work blocked

2. **Security Audit Findings (Phase 3)**
   - Risk: Critical vulnerabilities discovered requiring major refactoring
   - Mitigation: Proactive internal security review, follow best practices
   - Impact if delayed: Mainnet launch delayed 2-4 weeks per critical finding

3. **CosmWasm Integration (Phase 4)**
   - Risk: IBC + CosmWasm integration more complex than expected
   - Mitigation: Can launch mainnet without CosmWasm, add via upgrade later
   - Impact if delayed: Smart contract features unavailable at launch

4. **Validator Coordination (Phase 5)**
   - Risk: Insufficient validators or coordination failures at genesis
   - Mitigation: Recruit 2x needed validators, multiple dry runs, clear communication
   - Impact if delayed: Launch postponed until sufficient validator commitments

### Medium-Risk Items

1. **IBC Relayer Setup (Phase 2)**
   - Risk: IBC channels fail to open or experience packet timeouts
   - Mitigation: Extensive testing on testnets, use proven relayer software (Hermes)
   - Impact if delayed: Interoperability features delayed, but chain still functional

2. **Infrastructure Scaling (Phase 4)**
   - Risk: Underestimate resource requirements leading to performance issues
   - Mitigation: Load testing, over-provision initially, auto-scaling configured
   - Impact if delayed: Poor user experience, high latency, potential downtime

3. **Community Adoption (Phase 4-6)**
   - Risk: Low user/developer interest at launch
   - Mitigation: Strong marketing, partnerships, incentive programs
   - Impact if delayed: Slower growth but not critical for chain operation

### Low-Risk Items

1. **Documentation (All Phases)**
   - Risk: Incomplete or outdated documentation
   - Mitigation: Documentation is already extensive, needs updates not creation
   - Impact if delayed: Slower developer onboarding but not blocking

2. **Monitoring Dashboards (Phase 2)**
   - Risk: Dashboards not comprehensive enough
   - Mitigation: Start with existing dashboards, iterate based on needs
   - Impact if delayed: Less visibility but does not affect chain functionality

---

## Resource Requirements

### Team

**Minimum Team:**
- 2-3 Core Developers (Go, Cosmos SDK)
- 1 DevOps Engineer (Kubernetes, cloud)
- 1 Security Engineer (audits, pen testing)
- 1 Community Manager (Discord, marketing)
- 1 Technical Writer (documentation)

**Advisors/Consultants:**
- Legal counsel (token law, compliance)
- Security auditor (third-party firm)
- Cosmos ecosystem advisor (IBC, governance)

### Infrastructure Costs (Monthly)

**Testnet (Phase 2):**
- Cloud VMs: $500-800
- Monitoring: $100-200
- Networking: $100-200
- Total: ~$1,000/month

**Mainnet (Phase 5+):**
- Cloud VMs: $2,000-3,000
- Monitoring: $300-500
- Networking: $500-1,000
- Database: $200-500
- CDN: $200-500
- Total: ~$4,000-6,000/month

**Security (Phase 3):**
- Third-party audit: $50,000-150,000 (one-time)
- Bug bounty reserve: $50,000-200,000 (one-time, paid out over time)

### Development Tools

- GitHub (free for public repos)
- Docker Hub (free tier)
- Cloud provider (GCP, AWS, or DigitalOcean)
- Monitoring (self-hosted or managed Grafana/Prometheus)

---

## Success Metrics

### Phase 0 Success
- ✅ Build succeeds
- ✅ Unit tests pass

### Phase 1 Success
- ✅ 4-node local testnet running
- ✅ All modules operational (DEX, Oracle, Compute)
- ✅ Test coverage >80%

### Phase 2 Success
- ✅ Public testnet accessible
- ✅ External validators joined (4+)
- ✅ IBC channels operational

### Phase 3 Success
- ✅ Security audit passed (no critical/high findings open)
- ✅ Penetration test complete
- ✅ Bug bounty program active

### Phase 4 Success
- ✅ Mainnet genesis ready
- ✅ Infrastructure scaled and load-tested
- ✅ Community engaged (validators, users, developers)

### Phase 5 Success
- ✅ Mainnet launched successfully
- ✅ Blocks producing consistently for 7+ days
- ✅ >20 validators, >90% uptime

### Phase 6 Success (Ongoing)
- ✅ Monthly active users growing
- ✅ TVL in DEX increasing
- ✅ Governance proposals passing regularly
- ✅ No major security incidents

---

## Next Immediate Actions

**Start TODAY (Priority 1):**

1. **Fix build errors (Phase 0.1)**
   - Migrate sdkerrors to v0.50 patterns
   - Define missing error types
   - Fix IBC SendPacket signatures
   - Target: Complete in 2-3 days

2. **Run basic tests (Phase 0.2)**
   - Ensure unit tests pass after fixes
   - Validate node startup
   - Target: Complete in 1 day after build fixes

3. **Update documentation (Phase 0.3)**
   - Document SDK migration changes
   - Update README with current state
   - Target: Complete in 1 day

**Week 1 Priorities:**

4. **Complete DEX module (Phase 1.1)**
   - Test AMM math
   - Verify IBC transfers
   - CLI command testing

5. **Complete Oracle module (Phase 1.1)**
   - Median calculation
   - Slashing integration
   - CLI command testing

6. **Complete Compute module (Phase 1.1)**
   - Escrow logic
   - Verification mechanism
   - CLI command testing

**Week 2-3 Priorities:**

7. **Local testnet (Phase 1.3)**
   - 4-node deployment
   - Multi-module testing
   - Validator operations

8. **Test coverage (Phase 1.2)**
   - Achieve >80% coverage
   - Run simulation tests
   - Security test suite

---

## Appendix

### A. Reference Links

- **Cosmos SDK Documentation:** https://docs.cosmos.network
- **CometBFT Documentation:** https://docs.cometbft.com
- **IBC Protocol:** https://ibcprotocol.org
- **CosmWasm:** https://docs.cosmwasm.com
- **Tendermint Core:** https://tendermint.com/core/

### B. Related Documents

- `README.md` - Project overview and quick start
- `TECHNICAL_SPECIFICATION.md` - Detailed technical design
- `WHITEPAPER.md` - Economic and consensus model
- `CHANGELOG.md` - Version history
- `CONTRIBUTING.md` - Development guidelines
- `SECURITY.md` - Security policy and reporting
- `docs/BUG_BOUNTY.md` - Bug bounty program details

### C. Useful Commands

```bash
# Build
make build

# Test
make test                    # All tests
make test-unit              # Unit tests only
make test-integration       # Integration tests
make test-coverage          # With coverage report

# Lint and format
make lint
make format

# Security
make security-audit
make scan-secrets

# Local testnet
make init-testnet
make start-node

# Monitoring
make monitoring-start
make monitoring-stop

# Load testing
make load-test

# Kubernetes
./scripts/deploy/deploy-k8s.sh
```

### D. Contact Information

- **GitHub:** https://github.com/decristofaroj/paw
- **Discord:** [To be created]
- **Twitter:** [To be created]
- **Email:** [To be determined]

---

**Document End**

*This roadmap is a living document and will be updated as the project progresses. Last updated: November 27, 2025*
