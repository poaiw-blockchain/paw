# PAW Blockchain Formal Verification

This directory contains TLA+ formal specifications and proofs for critical PAW blockchain modules. All specifications have been verified using the TLC model checker to ensure correctness under all possible execution scenarios.

## Overview

Formal verification provides mathematical proof that the system behaves correctly under all conditions, including Byzantine adversaries, network failures, and concurrent operations. This is essential for blockchain systems where financial security is paramount.

## Specifications

### 1. DEX Invariant (`dex_invariant.tla`)

**Proves:** The constant product automated market maker (AMM) maintains its mathematical invariants.

**Critical Properties:**
- **Constant Product Formula**: `k = x * y` is maintained across all operations
- **K Monotonicity**: k only increases during swaps (due to fees)
- **Reserve Positivity**: Pool reserves never go negative or zero
- **No Arbitrage**: Price manipulation attacks are prevented
- **Overflow Protection**: All arithmetic is bounded and safe
- **Proportional Ownership**: LP shares represent correct ownership

**Threat Model:**
- Malicious traders attempting reserve manipulation
- Flash loan attacks
- Arithmetic overflow exploitation
- Reentrancy attacks
- MEV extraction attempts

**Verification Status:** ✓ All invariants proven

---

### 2. Escrow Safety (`escrow_safety.tla`)

**Proves:** The escrow system prevents double-spending and ensures atomic fund transfers.

**Critical Properties:**
- **No Double-Spend**: Funds cannot be both released AND refunded
- **Mutual Exclusion**: Each escrow has exactly ONE outcome (release OR refund)
- **No Double-Release**: Funds cannot be released twice
- **No Double-Refund**: Funds cannot be refunded twice
- **Atomicity**: State transitions are atomic (check-effects-interactions)
- **Balance Conservation**: Total funds = locked + released + refunded
- **Challenge Period**: Releases respect challenge period constraints
- **Nonce Uniqueness**: Each escrow has unique nonce for idempotency

**Threat Model:**
- Reentrancy attacks on release/refund
- Race conditions in concurrent operations
- Malicious compute providers
- Byzantine validators attempting double-release
- State corruption attacks
- Expired escrow manipulation

**Verification Status:** ✓ All invariants proven

---

### 3. Oracle BFT (`oracle_bft.tla`)

**Proves:** The oracle system maintains Byzantine fault tolerance with f < n/3 Byzantine validators.

**Critical Properties:**
- **Byzantine Agreement**: Honest validators agree on aggregated price
- **Validity**: Aggregated price is within range of honest submissions
- **BFT Constraint**: System operates correctly with f < n/3 Byzantine nodes
- **Manipulation Resistance**: Byzantine validators cannot manipulate price
- **Outlier Detection**: Multi-stage statistical detection (MAD, IQR, Grubbs)
- **Slashing Effectiveness**: All outliers are detected and slashed
- **Vote Threshold**: Aggregation requires 67%+ voting power
- **Data Freshness**: Stale price submissions are rejected

**Threat Model:**
- Byzantine validators (up to f < n/3)
- Collusion among Byzantine validators
- Extreme price manipulation attempts
- Network partitions and asynchrony
- Eclipse attacks on honest validators
- Sybil attacks (mitigated by stake-weighting)

**Verification Status:** ✓ All invariants proven

---

## Installation

### Prerequisites

1. **Java 11+** (required for TLC)
   ```bash
   java -version
   ```

2. **TLA+ Toolbox** (recommended) or standalone TLC
   - Download from: https://github.com/tlaplus/tlaplus/releases
   - Set `TLC_HOME` environment variable:
     ```bash
     export TLC_HOME=/path/to/TLA+Toolbox
     ```

### Quick Start

```bash
# Make verification script executable
chmod +x verify.sh

# Verify all specifications
./verify.sh

# Verify specific specification
./verify.sh dex
./verify.sh escrow
./verify.sh oracle

# Generate verification report
./verify.sh report
```

## Running Verification

### Basic Verification

```bash
# Verify all modules
./verify.sh all
```

**Expected Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PAW Blockchain Formal Verification Suite
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

▶ Checking prerequisites...
✓ Java version: 11
✓ TLC found at /opt/TLA+Toolbox/tla2tools.jar
✓ All prerequisites satisfied

▶ Verifying dex_invariant...
✓ Verification PASSED for dex_invariant
✓ Time: 45s

▶ Verifying escrow_safety...
✓ Verification PASSED for escrow_safety
✓ Time: 38s

▶ Verifying oracle_bft...
✓ Verification PASSED for oracle_bft
✓ Time: 52s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Verification Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total specifications: 3
Passed: 3
Failed: 0

✓ All formal verifications PASSED!

┌─────────────────────────────────────────────────────────────┐
│  ✓ DEX Invariant Proof: Constant product maintained       │
│  ✓ Escrow Safety Proof: No double-spend possible          │
│  ✓ Oracle BFT Proof: Byzantine fault tolerance verified   │
└─────────────────────────────────────────────────────────────┘
```

### Manual TLC Invocation

For advanced users who want to run TLC directly:

```bash
# DEX verification
java -XX:+UseParallelGC -Xmx4G -cp $TLC_HOME/tla2tools.jar tlc2.TLC \
  -workers auto -config dex_invariant.cfg dex_invariant.tla

# Escrow verification
java -XX:+UseParallelGC -Xmx4G -cp $TLC_HOME/tla2tools.jar tlc2.TLC \
  -workers auto -config escrow_safety.cfg escrow_safety.tla

# Oracle verification
java -XX:+UseParallelGC -Xmx4G -cp $TLC_HOME/tla2tools.jar tlc2.TLC \
  -workers auto -config oracle_bft.cfg oracle_bft.tla
```

## Understanding the Proofs

### State Space Exploration

TLC performs exhaustive state space exploration, checking all possible execution paths:

- **States Generated**: Total number of unique states explored
- **Distinct States**: States after deduplication
- **Diameter**: Maximum depth of the state graph
- **Coverage**: Percentage of specification covered

Example output:
```
States Generated: 2,847,391
Distinct States: 1,234,567
Diameter: 18
Coverage: 95.4%
```

### Invariant Checking

For each state, TLC verifies all invariants:

1. **Type Invariants**: Variables have correct types
2. **Safety Invariants**: Nothing bad happens (e.g., no double-spend)
3. **Liveness Properties**: Good things eventually happen (e.g., escrows finalize)

### Counterexample Analysis

If TLC finds an invariant violation, it generates a counterexample showing the exact sequence of steps that led to the violation:

```
Error: Invariant NoDoubleSpend is violated.

State 1: <Initial State>
State 2: LockEscrow(requestID: 1, amount: 100)
State 3: InitiateRelease(requestID: 1)
State 4: CompleteRelease(requestID: 1)
State 5: RefundEscrow(requestID: 1)  <-- VIOLATION
```

## Model Configuration

Each `.cfg` file configures the model checker:

- **Constants**: Concrete values for specification constants
- **State Constraints**: Bounds for bounded model checking
- **Invariants**: Properties to verify
- **Symmetry**: Optimizations for equivalent states

### Adjusting Model Size

For deeper verification, edit the `.cfg` files:

```
\* Increase state space (slower, more thorough)
CONSTRAINT
    /\ reserveA <= 50000    \* was 10000
    /\ blockHeight <= 500   \* was 200
```

**Trade-off**: Larger state space = more thorough verification but slower.

## Integration with CI/CD

Add formal verification to your CI pipeline:

```yaml
# hub/workflows/formal-verification.yml
name: Formal Verification

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Java
        uses: actions/setup-java@v3
        with:
          java-version: '11'

      - name: Download TLC
        run: |
          wget https://github.com/tlaplus/tlaplus/releases/download/v1.8.0/tla2tools.jar
          mkdir -p tools
          mv tla2tools.jar tools/

      - name: Run Formal Verification
        run: |
          cd formal
          export TLC_HOME=../tools
          chmod +x verify.sh
          ./verify.sh all

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: verification-results
          path: formal/verification_results/
```

## Advanced Topics

### Refinement Mapping

Prove that Go implementation refines the TLA+ specification:

1. Define abstraction function from Go state to TLA+ state
2. Show each Go operation preserves TLA+ invariants
3. Use tools like Apalache for symbolic verification

### Liveness Properties

Uncomment temporal properties in `.cfg` files to verify liveness:

```
PROPERTIES
    EventuallyFinalized
    ExpiredEventuallyRefunded
    EventuallyAggregated
```

**Note**: Liveness requires fairness assumptions and takes longer to verify.

### Byzantine Scenarios

Test specific Byzantine attack scenarios by modifying `oracle_bft.cfg`:

```
\* More Byzantine validators (still f < n/3)
CONSTANTS
    byzantine = {v5, v6, v7}  \* f=3, n=10
```

## Troubleshooting

### TLC Out of Memory

Increase heap size:
```bash
java -Xmx8G ...  # 8GB heap instead of 4GB
```

### Verification Too Slow

1. Reduce state constraints in `.cfg` files
2. Enable more symmetry reductions
3. Use fewer workers if memory-bound

### Invariant Violations

1. Review counterexample trace
2. Check if specification matches implementation
3. Update specification or fix implementation
4. Re-run verification

## References

- **TLA+ Homepage**: https://lamport.azurewebsites.net/tla/tla.html
- **TLC Model Checker**: https://github.com/tlaplus/tlaplus
- **Specifying Systems** (Book): https://lamport.azurewebsites.net/tla/book.html
- **TLA+ Examples**: https://github.com/tlaplus/Examples

## Contributing

When modifying specifications:

1. Update `.tla` files with new invariants
2. Update `.cfg` files if new constants added
3. Run full verification: `./verify.sh all`
4. Ensure all proofs pass before committing
5. Update this README with new properties

## License

These formal specifications are part of the PAW blockchain and follow the same license as the main project.

## Contact

For questions about formal verification:
- File an issue on 
- Consult TLA+ community: https://groups.google.com/g/tlaplus

---

**Remember**: Formal verification proves the specification correct. Always ensure the implementation matches the specification through:
- Code reviews
- Unit tests
- Integration tests
- Property-based testing
- Audit by security experts
