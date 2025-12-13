# Paw Project - Local Testing Plan (v4 - Definitive Edition)

This document is the definitive and most exhaustive local testing plan for the Paw project. It includes standard, advanced, and esoteric test cases to ensure maximum stability, security, and robustness. **This is the final version.**

## Phase 1: Primitives & Static Analysis

*   **[ ] 1.1: Linter and Static Analysis:** `make lint`
*   **[ ] 1.2: Unit Tests:** `make test-unit`
*   **[ ] 1.3: Integration Tests:** `make test-integration`
*   **[ ] 1.4: Simulation Tests:** `make test-simulation`
*   **[ ] 1.5: ZK-Circuit Logic:** Manually review and write specific unit tests for the logic within each `gnark` circuit (`compute`, `result`, `escrow`) to validate constraints under all edge cases.
*   **[ ] 1.6: Encoding Primitives:** Write an integration test to marshal/unmarshal all custom module types (`dex`, `oracle`, `compute`) to catch serialization bugs.

## Phase 2: Single-Node Lifecycle & Configuration

*   **[ ] 2.1: Genesis & Initialization:** Verify `pawd init`, `add-genesis-account`, `gentx`, etc., all work as expected.
*   **[ ] 2.2: Exhaustive Configuration Testing:**
    *   **Description:** Script the modification of every parameter in `config.toml` and `app.toml` to verify the node's behavior changes as expected or fails gracefully.
    *   **Action:** Pay special attention to P2P settings, timeouts, and cache sizes.
*   **[ ] 2.3: CLI Command Verification:**
    *   **Description:** Test every single CLI command provided by `pawd`, including all custom module queries and transactions.
    *   **Action:** Script the execution of every subcommand with both valid and invalid parameters (e.g., `pawd tx dex swap [VALID_PARAMS]`, `pawd tx dex swap [INVALID_PARAMS]`).
    *   **Expected Outcome:** All commands behave as documented, with clear errors for invalid usage.

## Phase 3: Multi-Node Network & Consensus

*   **[ ] 3.1: 4-Node Devnet Baseline:** `docker-compose -f compose/docker-compose.devnet.yml up -d --build`
*   **[ ] 3.2: Consensus Liveness & Halt:** Test 4-node, 3-node (live), and 2-node (halt) configurations.
*   **[ ] 3.3: Network Variable Latency/Bandwidth:** Use `tc` to simulate poor network conditions.
*   **[ ] 3.4: Malicious Peer Ejection:** Test if a node bans a peer that sends invalid consensus messages.

## Phase 4: Comprehensive Security & Attack Simulation

*   **[ ] 4.1: Validator Slashing (Double-Sign & Downtime):** Force and verify both double-sign and downtime slashing events.
*   **[ ] 4.2: 51% Re-org Attack:** Simulate a majority partition building a longer chain to test fork-choice logic.
*   **[ ] 4.3: RPC Endpoint Hardening & Fuzzing:** Fuzz test all public-facing RPC/API endpoints with malformed requests.
*   **[ ] 4.4: Governance Exploit Scenarios:** Test malicious governance proposals.
*   **[ ] 4.5: ZK-Compute - Invalid/Malformed Proof:** Fuzz the `submit-proof` transaction with garbage data, malformed proofs, and valid proofs for the wrong computation.
*   **[ ] 4.6: DEX - Economic Exploits:**
    *   **Description:** Simulate sandwich attacks, flash loan exploits, and test for rounding errors in liquidity pool calculations.
    *   **Action:** Requires custom scripts to simulate a malicious actor front-running and back-running trades.
    *   **Expected Outcome:** The DEX module is resilient, and profits cannot be extracted via trivial economic manipulation.

## Phase 5: Advanced State, Economics & Upgrades

*   **[ ] 5.1: State Snapshot & Restore:** Test a new node's ability to bootstrap from a state snapshot.
*   **[ ] 5.2: State Pruning:** Verify old state is correctly removed when pruning is enabled.
*   **[ ] 5.3: Staking & Rewards Logic:** Programmatically verify that staking and module-specific rewards (e.g., from `x/compute` or `x/dex`) are calculated correctly.
*   **[ ] 5.4: On-Chain Software Upgrade:** Test the full governance-based software upgrade process.
*   **[ ] 5.5: State Migration:** Verify custom state migration logic for all modules (`dex`, `oracle`, `compute`) during the software upgrade.

## Phase 6: Cross-Chain Interoperability

*   **[ ] 6.1: IBC (Paw <-> Aura):** Setup a relayer and test token transfers, channel creation/closing, and relayer failure/restart scenarios.
*   **[ ] 6.2: IBC - Oracle Data Transfer:** If the oracle module supports it, test sending oracle data packets over IBC.
*   **[ ] 6.3: Atomic Swaps (Paw <-> BTC):** If the module exists, test successful swaps and failed/refunded swaps.

## Phase 7: Destructive & Long-Running Tests

*   **[ ] 7.1: Database Corruption Test:**
    *   **Description:** Intentionally corrupt a node's `application.db` while it is stopped.
    *   **Action:** Use `dd` to write garbage data into the database files.
    *   **Expected Outcome:** Upon restart, the node fails with a clear "database is corrupt" error.
*   **[ ] 7.2: Resource Constraint Test:**
    *   **Description:** Run a node with heavily restricted RAM and CPU.
    *   **Action:** Use Docker's `--memory` and `--cpus` flags.
    *   **Expected Outcome:** The node runs slower but remains stable, helping to define minimum system requirements.
*   **[ ] 7.3: Long-Running Stability (Soak Test):**
    *   **Description:** Run the 4-node testnet under a continuous, mixed load for 24-48 hours.
    *   **Action:** Use a script to continuously send a mix of DEX swaps, oracle price updates, and ZK-compute requests. Monitor with Prometheus.
    *   **Expected Outcome:** The network remains stable with no memory leaks or performance degradation.
*   **[ ] 7.4: Load Test:**
    *   **Description:** Run the makefile load test to push the system to its limits.
    *   **Command:** `make load-test`
    *   **Expected Outcome:** The network processes a high volume of transactions and remains stable.

This v4 plan represents the full scope of local testing that can be performed.
---

# Test Plan Gap Analysis - paw

Generated: Sat Dec 13 07:36:50 AM UTC 2025
Source: LOCAL_TESTING_PLAN.md

## Identified Gaps

### Missing Essential Tests

- [ ] **Crypto Primitives**: Not found in current test plan
- [ ] **Encoding/Serialization**: Not found in current test plan
- [ ] **Consensus Testing**: Not found in current test plan
- [ ] **Security Testing**: Not found in current test plan
- [ ] **Slashing Tests**: Not found in current test plan
- [ ] **RPC Endpoint Testing**: Not found in current test plan
- [ ] **State Management**: Not found in current test plan
- [ ] **Economic Testing**: Not found in current test plan
- [ ] **Upgrade Testing**: Not found in current test plan
- [ ] **Cross-Chain/IBC**: Not found in current test plan
- [ ] **Database Testing**: Not found in current test plan
- [ ] **Resource Constraints**: Not found in current test plan
- [ ] **Destructive Tests**: Not found in current test plan

### Missing Advanced Tests

- [ ] **Load Testing**: Consider adding this test category
- [ ] **Performance Profiling**: Consider adding this test category
- [ ] **Memory Leak Detection**: Consider adding this test category
- [ ] **Byzantine Behavior**: Consider adding this test category
- [ ] **Double-Spend Prevention**: Consider adding this test category
- [ ] **Timestamp Validation**: Consider adding this test category
- [ ] **Fee Market Testing**: Consider adding this test category
- [ ] **State Snapshots**: Consider adding this test category
- [ ] **Replay Protection**: Consider adding this test category
- [ ] **Nonce Management**: Consider adding this test category
- [ ] **Gas Optimization**: Consider adding this test category
- [ ] **Contract Security**: Consider adding this test category
- [ ] **Oracle Testing**: Consider adding this test category
- [ ] **DEX Testing**: Consider adding this test category
- [ ] **Governance Testing**: Consider adding this test category
- [ ] **Chain Reorganization**: Consider adding this test category
- [ ] **Orphan Blocks**: Consider adding this test category

## Recommendations

### Infrastructure Tooling

The following tools have been created to support local testing:

- `scripts/load-tests/` - Load testing with k6
- `scripts/testnet-scenarios.sh` - Multi-node test scenarios
- `scripts/snapshot-manager.sh` - State snapshot management
- `scripts/network-sim.sh` - Network condition simulation
- `scripts/profile-*.sh` - Performance profiling tools
- `scripts/db-benchmark.sh` - Database benchmarking

### Test Coverage Improvements

1. **Fuzzing**: Add property-based and fuzzing tests for critical paths
2. **Load Testing**: Implement realistic load scenarios with k6
3. **Chaos Engineering**: Use testnet-scenarios.sh for failure testing
4. **Performance Regression**: Set up automated profiling
5. **Security Scanning**: Integrate static analysis and vulnerability scanning

### Next Steps

1. Review missing essential tests and add to test plan
2. Implement infrastructure-assisted tests using new tooling
3. Set up automated test execution for CI/CD
4. Document test procedures and expected outcomes
5. Create test data generators for realistic scenarios

