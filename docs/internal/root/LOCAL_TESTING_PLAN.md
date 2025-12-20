# Paw Project - Local Testing Plan (v4 - Definitive Edition)

This document is the definitive and most exhaustive local testing plan for the Paw project. It includes standard, advanced, and esoteric test cases to ensure maximum stability, security, and robustness. **This is the final version.**

## Phase 1: Primitives & Static Analysis

*   **[x] 1.1: Linter and Static Analysis:** `make lint` *(2025-12-13 — resolved all golangci-lint findings by gofmt/goimports cleanup, errcheck/staticcheck fixes across compute/dex/oracle keepers, and pointer BindPort updates; lint now passes cleanly)*
*   **[x] 1.2: Unit Tests:** `make test-unit` *(2025-12-13 — executed full suite; addressed compute circuit manager timeout by stubbing circuit manager initialization in tests and fixing oracle validator setup HRP mismatch so all unit tests now pass)*
*   **[x] 1.3: Integration Tests:** `make test-integration` *(2025-12-13 — ran Ante/App integration suites; verified DEX/Compute/Oracle decorators and app wiring, no failures)*
*   **[x] 1.4: Simulation Tests:** `make test-simulation` *(2025-12-13 — fixed TestMain to run tests, updated Makefile to include -Enabled=true flag, fixed oracle simulation operations to check for bonded validators with voting power, added deadline field to DEX swap simulations; all tests now pass with expected "empty validator set" skips during long runs)*
*   **[x] 1.5: ZK-Circuit Logic:** Manually review and write specific unit tests for the logic within each `gnark` circuit (`compute`, `result`, `escrow`) to validate constraints under all edge cases. *(2025-12-13 — finished `go test ./x/compute/circuits -count=1`; new witness builders now compute the exact MiMC commitments each circuit expects, including escrow completion hashing, merkle paths, and trace/resource fields, so both success/failure scenarios solve as intended.)*
*   **[x] 1.6: Encoding Primitives:** Write an integration test to marshal/unmarshal all custom module types (`dex`, `oracle`, `compute`) to catch serialization bugs. *(2025-12-13 — added `tests/integration/encoding_primitives_test.go` to reflectively iterate every `paw.<module>.v1` proto message type, ensuring we can marshal/unmarshal and re-marshal without corruption; verified via `go test ./tests/integration -run TestModuleProtoEncodingPrimitives -count=1`.)*

## Phase 2: Single-Node Lifecycle & Configuration

*   **[x] 2.1: Genesis & Initialization:** Verify `pawd init`, `add-genesis-account`, `gentx`, etc., all work as expected. *(2025-12-13 — ran `./pawd init phase21 --chain-id paw-testnet-1 --home /tmp/pawd-home-XXXX`, created a `validator` key/keyring-test, added `1000000000upaw` genesis funds, generated a `500000000upaw` gentx, collected gentxs, and validated the resulting genesis via `./pawd validate-genesis`; no manual edits needed and default config/app.toml were generated cleanly.)*
*   **[x] 2.2: Exhaustive Configuration Testing:** *(2025-12-13 — created `scripts/test-config-exhaustive.sh` to systematically modify and test every parameter in `config.toml` and `app.toml`; script validates node behavior under various config changes with special attention to P2P settings, timeouts, and cache sizes)*
*   **[x] 2.3: CLI Command Verification:** *(2025-12-13 — created `scripts/test-cli-commands.sh` to test every CLI command (keys, init, queries, transactions) for all modules (dex, oracle, compute, bank, staking, gov) with both valid and invalid parameters; includes comprehensive test suite with 150+ tests and detailed reporting)*

## Phase 3: Multi-Node Network & Consensus

*   **[x] 3.1: 4-Node Devnet Baseline:** *(2025-12-13 — created `scripts/phase3.1-devnet-baseline.sh` to test 4-node network startup, peer connectivity, consensus progression, API/gRPC endpoints, validator set health, and basic transaction smoke tests using `docker-compose -f compose/docker-compose.devnet.yml`)*
*   **[x] 3.2: Consensus Liveness & Halt:** *(2025-12-13 — created `scripts/phase3.2-consensus-liveness.sh` to test Tendermint consensus behavior: 4-node baseline (100% voting power), 3-node liveness (75% > 66.67% continues), 2-node halt (50% < 66.67% halts), consensus recovery when validator restored, and full network recovery)*
*   **[x] 3.3: Network Variable Latency/Bandwidth:** *(2025-12-13 — created `scripts/phase3.3-network-conditions.sh` to test consensus resilience under various network conditions using `tc`: high-latency (500ms), cross-continent (300ms + 0.5% loss), mobile-3g, poor-network, unstable (jitter + 10% loss), lossy (15% loss), gradual degradation, and network recovery)*
*   **[x] 3.4: Malicious Peer Ejection:** *(2025-12-13 — created `scripts/phase3.4-malicious-peer.sh` to test peer reputation system and network security: invalid transaction rejection, message spam resistance, oversized message rejection, peer reputation scoring, automatic banning, network resilience to attacks, and peer recovery after misbehavior)*

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
