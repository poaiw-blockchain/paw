# PAW Production Roadmap

**Status:** Build ❌ FAILING | **Chain:** Cosmos SDK 0.50.9 + CometBFT | **Modules:** DEX, Oracle, Compute

---

## ⚠️ CRITICAL: Fix Build First (Phase 0)

Build fails due to Cosmos SDK v0.50 migration issues:
```
Error: undefined: sdkerrors.Wrapf (x/dex/keeper)
Error: undefined: types.ErrOracleDataUnavailable (x/oracle/keeper)
Error: IBC SendPacket signature mismatch
```

**All other work is blocked until build succeeds.**

---

## Existing Components (DO NOT DUPLICATE)

### Core Chain
- ✅ App wiring: `/app/app.go` (Cosmos SDK v0.50.9)
- ✅ CLI: `/cmd/pawd/`, `/cmd/pawcli/`
- ✅ Makefile: 60+ targets, 648 lines

### Custom Modules (`/x/`)
- ✅ **DEX:** `/x/dex/` - AMM pools, swaps, liquidity, IBC transfers
- ✅ **Oracle:** `/x/oracle/` - Price feeds, aggregation, slashing
- ✅ **Compute:** `/x/compute/` - Job escrow, ZK verification, IBC compute
- ⚠️ **Privacy:** `/x/privacy/` - Staging, not production ready

### Protocol Buffers (`/proto/paw/`)
- ✅ DEX, Oracle, Compute messages
- ✅ Buf configuration for code generation

### P2P Networking (`/p2p/`)
- ✅ Node discovery (DHT-based)
- ✅ Protocol handlers, reputation system
- ✅ Security with mutual TLS
- ✅ Snapshot support

### IBC (`/ibc/`)
- ✅ Relayer configuration
- ✅ Security guidelines
- ⚠️ SendPacket API needs update for v0.50

### Testing (83 test files in `/tests/`)
- ✅ Unit, integration, E2E, security, chaos
- ✅ Load tests (k6, Locust, Go benchmarks)
- ✅ Invariant, simulation, Byzantine, differential tests

### DevOps
- ✅ Docker: multi-stage Dockerfile, Docker Compose
- ✅ K8s: 19 manifests in `/k8s/`
- ✅ Scripts: 51 shell scripts in `/scripts/`
- ✅ Pre-commit hooks configured

### Monitoring (`/infra/monitoring/`)
- ✅ Prometheus config + alert rules
- ✅ Grafana dashboards
- ✅ Alertmanager, Loki, Promtail, Jaeger
- ✅ Docker Compose monitoring stack: `/compose/docker-compose.monitoring.yml`

### Documentation (`/docs/`)
- ✅ 50+ markdown files
- ✅ Technical Specification (75KB)
- ✅ Whitepaper, API docs, security guides
- ✅ Bug bounty program

### Archived (needs deployment)
- ⚠️ Faucet: `/archive/faucet/`
- ⚠️ Explorer: `/archive/explorer/`
- ⚠️ Portal: `/archive/portal/`
- ⚠️ Status: `/archive/status/`

---

## Critical Gaps

| Gap | Priority | Fix |
|-----|----------|-----|
| Build failing | 🔴 BLOCKING | Migrate sdkerrors → errors.Join() |
| Missing error types | 🔴 BLOCKING | Add ErrSlippageExceeded, ErrOracleDataUnavailable |
| IBC SendPacket | 🔴 BLOCKING | Add capability parameter |
| CosmWasm integration | 🟡 High | Complete IBC init first, then enable |
| Active testnet | 🟡 High | Deploy after build fixed |
| Security audit | 🟡 High | External audit needed |

---

## Phase 0: Critical Fixes (2-3 days)

### 0.1 Fix Build Errors

**Migrate error handling:**
- [ ] Replace `sdkerrors.Wrapf()` → `fmt.Errorf()` or `errors.Join()`
- [ ] Replace `sdkerrors.Wrap()` → module-specific error wrapping
- [ ] Files: `x/dex/keeper/ibc_aggregation.go`, `x/oracle/keeper/ibc_prices.go`, `x/oracle/keeper/ibc_timeout.go`

**Add missing error types:**
- [ ] Add `ErrSlippageExceeded` → `x/dex/types/errors.go`
- [ ] Add `ErrOracleDataUnavailable` → `x/oracle/types/errors.go`

**Fix SwapResult struct:**
- [ ] Check `x/dex/types/types.go` for correct fields
- [ ] Update `x/dex/keeper/ibc_aggregation.go:221-224`
- [ ] Regenerate protobuf if needed: `make proto-gen`

**Fix IBC SendPacket:**
- [ ] Update signature in `x/oracle/keeper/ibc_prices.go:481`
- [ ] Add capability parameter from scoped keeper
- [ ] Reference: `x/compute/keeper/ibc_compute.go` (working example)

**Fix deprecated SDK functions:**
- [ ] `sdk.KVStorePrefixIterator` → `storetypes.KVStorePrefixIterator`
- [ ] `sdk.ZeroDec()` → `math.LegacyZeroDec()`
- [ ] Update imports: `cosmossdk.io/math`

**Verify:**
- [ ] `make build` succeeds
- [ ] `./build/pawd version` runs
- [ ] `make test-unit` passes

### 0.2 Basic Node Validation
- [ ] `./build/pawd init test-node --chain-id paw-test-1`
- [ ] `./build/pawd start --minimum-gas-prices 0.001upaw`
- [ ] Verify node starts without panics

### 0.3 Documentation
- [ ] Update README.md build instructions
- [ ] Create `docs/development/SDK_V050_MIGRATION.md`

---

## Phase 1: Local Testnet (1-2 weeks)

### Module Completion

**DEX (`/x/dex/`):**
- [ ] Verify AMM formula (x * y = k)
- [ ] Test pool creation, swaps with fees
- [ ] Implement slippage protection
- [ ] Test IBC token transfers
- [ ] Verify economic invariants (no token creation/destruction)
- [ ] CLI: `pawcli tx dex create-pool/add-liquidity/swap`

**Oracle (`/x/oracle/`):**
- [ ] Implement vote submission, median calculation
- [ ] Outlier detection, TWAP calculation
- [ ] Slashing integration with staking module
- [ ] CLI: `pawcli tx oracle submit-price`, `pawcli query oracle price`

**Compute (`/x/compute/`):**
- [ ] Job escrow system
- [ ] ZK proof / TEE attestation verification
- [ ] Provider registration, reputation, slashing
- [ ] CLI: `pawcli tx compute submit-job`, `pawcli query compute job`

### Testing
- [ ] Run: `make test-coverage` (target: >80%)
- [ ] Run: `make test-integration`
- [ ] Run: `make test-simulation`
- [ ] Run security tests: `/tests/security/`

### Multi-Node Testnet
- [ ] Initialize 4-node local testnet: `./scripts/localnet-start.sh`
- [ ] Test: send tokens, create DEX pool, submit oracle prices, post compute job
- [ ] Test validator add/remove, slashing, coordinated upgrade

---

## Phase 2: Cloud Testnet (2-3 weeks)

### Infrastructure
- [ ] Provision cloud (GCP recommended - scripts exist in `/scripts/devnet/gcp-*.sh`)
- [ ] Deploy K8s: `./scripts/deploy/deploy-k8s.sh`
- [ ] Configure DNS: rpc.pawtestnet.network, api.pawtestnet.network

### Genesis
- [ ] Create testnet genesis: chain-id `paw-testnet-1`
- [ ] Use template: `/config/genesis-mainnet.json`
- [ ] Coordinate genesis ceremony with validators

### Public Services
- [ ] Deploy faucet from `/archive/faucet/`
- [ ] Deploy explorer from `/archive/explorer/`
- [ ] Deploy status page from `/archive/status/`

### Monitoring
- [ ] Deploy: `docker-compose -f compose/docker-compose.monitoring.yml up -d`
- [ ] Import dashboards from `/infra/monitoring/grafana-dashboards/`
- [ ] Configure alerts per `/infra/monitoring/alert_rules.yml`

### IBC
- [ ] Deploy Hermes relayer per `/ibc/relayer-config.yaml`
- [ ] Establish channel to Cosmos Hub testnet
- [ ] Test cross-chain transfers

---

## Phase 3: Security Hardening (3-4 weeks)

### Internal Review
- [ ] Run: `make security-audit`
- [ ] Run: `make scan-secrets` (GitLeaks)
- [ ] Audit DEX for reentrancy, overflow, economic attacks
- [ ] Audit Oracle for data manipulation, DoS, timestamp manipulation
- [ ] Audit Compute for escrow security, verification integrity
- [ ] Follow guides: `x/dex/SECURITY_IMPLEMENTATION_GUIDE.md`, `x/oracle/STATISTICAL_OUTLIER_DETECTION.md`, `x/compute/SECURITY_TESTING_GUIDE.md`

### External Audit
- [ ] Select firm: Trail of Bits, Informal Systems, Halborn, CertiK
- [ ] Budget: $50,000-$150,000
- [ ] Scope: consensus, custom modules, IBC, P2P
- [ ] Remediate all critical/high findings

### Penetration Testing
- [ ] Test RPC/API exploits
- [ ] Test consensus manipulation
- [ ] Launch bug bounty: `/docs/BUG_BOUNTY.md`

---

## Phase 4: Production Preparation (2-3 weeks)

### Code Finalization
- [ ] Address all audit findings
- [ ] Tag release: `v1.0.0`
- [ ] Build binaries for Linux, macOS, Windows (amd64, arm64)
- [ ] Sign binaries

### Genesis
- [ ] Finalize mainnet genesis: chain-id `paw-mainnet-1`
- [ ] Token distribution per economics model
- [ ] Governance params: 7-day voting, 40% quorum
- [ ] Collect gentx from 20+ validators

### CosmWasm (if needed)
- [ ] Complete IBC initialization in `/app/app.go`
- [ ] Uncomment CosmWasm keeper initialization (line 179-180)
- [ ] Configure: Upload=AllowNobody, SmartQueryGasLimit=3000000

---

## Phase 5: Mainnet Launch (3 weeks)

### Genesis Coordination
- [ ] Collect gentx (deadline: 1 week before launch)
- [ ] Distribute final genesis.json with SHA256 checksum
- [ ] Dry run with subset of validators

### Launch
- [ ] Coordinated start at genesis time
- [ ] Monitor first 1000 blocks
- [ ] 24/7 monitoring for first week

### Services
- [ ] Activate public RPC/API
- [ ] Launch faucet, explorer
- [ ] Enable IBC channels
- [ ] Wallet integrations (Keplr, Leap, Cosmostation)

---

## Phase 6: Post-Launch (Ongoing)

- [ ] Daily health checks via Grafana
- [ ] Weekly network reports
- [ ] Quarterly upgrades via governance
- [ ] Quarterly security audits
- [ ] Feature development: concentrated liquidity, limit orders, more oracle assets

---

## Timeline Summary

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Phase 0: Critical Fixes | 2-3 days | 3 days |
| Phase 1: Local Testnet | 1-2 weeks | 2.5 weeks |
| Phase 2: Cloud Testnet | 2-3 weeks | 5.5 weeks |
| Phase 3: Security | 3-4 weeks | 9.5 weeks |
| Phase 4: Production Prep | 2-3 weeks | 12.5 weeks |
| Phase 5: Mainnet Launch | 3 weeks | 15.5 weeks |

**Total: ~4 months to mainnet**

---

## Quick Commands

```bash
# Build
cd /home/decri/blockchain-projects/paw
make build

# Test
make test
make test-unit
make test-coverage
make security-audit

# Local testnet
make init-testnet
./scripts/localnet-start.sh

# Monitoring
make monitoring-start
# or
docker-compose -f compose/docker-compose.monitoring.yml up -d

# K8s deployment
./scripts/deploy/deploy-k8s.sh
```
