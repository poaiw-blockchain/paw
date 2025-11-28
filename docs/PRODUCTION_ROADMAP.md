# PAW Blockchain Production Roadmap

This document outlines the step-by-step plan to bring the PAW blockchain from its current state to a production-ready Mainnet launch.

## Phase 1: Foundation & Stability (Current Focus)
- [ ] **Build System Repair**
    - [ ] Fix `go.mod` version mismatch (downgrade from 1.25 to system version).
    - [ ] Verify `make build` succeeds.
    - [ ] Ensure Docker builds work (`docker build .`).
- [ ] **Dependency Management**
    - [ ] Audit and pin all dependencies in `go.mod`.
    - [ ] Remove unused dependencies.
- [ ] **CI/CD Setup**
    - [ ] Configure GitHub Actions for build, lint, and test.

## Phase 2: Core Module Completion
- [ ] **x/compute** (Verifiable AI Compute)
    - [ ] Implement Job Escrow & Assignment logic.
    - [ ] Finalize Verification Hooks.
    - [ ] Add IBC Port for cross-chain compute requests.
    - [ ] Add rigorous unit and keeper tests.
- [ ] **x/dex** (Decentralized Exchange)
    - [ ] Finalize AMM Pool logic (Constant Product).
    - [ ] Implement Liquidity Accounting & LP Tokens.
    - [ ] Enable IBC Token transfers & Swaps.
    - [ ] Add simulation tests for economic security.
- [ ] **x/oracle** (Price Feeds)
    - [ ] Implement Vote Aggregation & Median calculation.
    - [ ] Add Slashing Hooks for misreporting validators.
    - [ ] Integrate with `x/dex` for TWAP if needed.

## Phase 3: Testing & Verification
- [ ] **Unit & Integration Tests**
    - [ ] Achieve >80% code coverage on all modules.
    - [ ] Fix all `golangci-lint` errors.
- [ ] **Simulation Testing**
    - [ ] Run Cosmos SDK simulations (multi-day runs) to catch non-determinism.
- [ ] **Testnet Launch**
    - [ ] Spin up a local 4-node testnet.
    - [ ] Verify genesis file generation.
    - [ ] Test validator joining/leaving.

## Phase 4: Documentation & Professionalism
- [ ] **Whitepaper**
    - [ ] Draft comprehensive `docs/WHITEPAPER.md` detailing the "Verifiable AI Compute" consensus.
- [ ] **Technical Specs**
    - [ ] Update `docs/TECHNICAL_SPECIFICATION.md` with final module architectures.
- [ ] **Developer Guides**
    - [ ] Create "Zero to Node" guide for validators.

## Phase 5: Production Hardening
- [ ] **Security Audit**
    - [ ] Run automated security tools (`gosec`, `govulncheck`).
    - [ ] Review crypto implementation (AnteHandlers, Signatures).
- [ ] **Performance Tuning**
    - [ ] Benchmark TPS and optimize heavy queries.
    - [ ] Configure state sync and pruning settings.
- [ ] **Genesis Preparation**
    - [ ] Define initial token distribution.
    - [ ] Set governance parameters (voting period, deposit).

---
*Status: Phase 1 In Progress*
