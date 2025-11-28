# Formal Verification Integration Guide

## Overview

PAW blockchain uses TLA+ formal verification to mathematically prove the correctness of critical modules. This guide explains how formal verification is integrated into the development workflow and CI/CD pipeline.

## What is Formal Verification?

Formal verification uses mathematical methods to prove that software behaves correctly under **all possible conditions**. Unlike testing (which checks specific scenarios), formal verification provides exhaustive guarantees.

### Why It Matters for Blockchain

- **Financial Security**: Bugs in blockchain systems can lead to loss of funds
- **Immutability**: Once deployed, smart contracts cannot be easily patched
- **Byzantine Threats**: Must handle malicious actors and network failures
- **Mathematical Proof**: Testing shows presence of bugs; verification proves their absence

## Verified Modules

### 1. DEX Module (`dex_invariant.tla`)

**Properties Proven:**
- ✓ Constant product formula k = x × y is maintained across all operations
- ✓ Reserves remain strictly positive (no zero/negative balances)
- ✓ No arithmetic overflow in any calculation
- ✓ K monotonically increases during swaps (fee accumulation)
- ✓ LP shares represent proportional ownership

**Threat Model:**
- Flash loan attacks
- MEV extraction attempts
- Arithmetic overflow exploits
- Price manipulation
- Reentrancy attacks

**Verification Time:** ~10-15 minutes
**State Space:** ~2,000,000 states

---

### 2. Escrow Module (`escrow_safety.tla`)

**Properties Proven:**
- ✓ No double-spend: funds cannot be released AND refunded
- ✓ Mutual exclusion: exactly one outcome per escrow
- ✓ No double-release (idempotency)
- ✓ No double-refund (idempotency)
- ✓ Challenge period integrity
- ✓ Balance conservation
- ✓ Nonce uniqueness for idempotent operations

**Threat Model:**
- Reentrancy attacks
- Race conditions
- Malicious compute providers
- Byzantine validators
- Expired escrow manipulation

**Verification Time:** ~15-20 minutes
**State Space:** ~1,500,000 states

---

### 3. Oracle Module (`oracle_bft.tla`)

**Properties Proven:**
- ✓ Byzantine fault tolerance: f < n/3 constraint enforced
- ✓ Validity: aggregated price within honest validator range
- ✓ Manipulation resistance against Byzantine validators
- ✓ Outlier detection effectiveness (MAD, IQR algorithms)
- ✓ Vote threshold enforcement (67%+ required)
- ✓ Data freshness (stale prices rejected)

**Threat Model:**
- Byzantine validators (up to f < n/3)
- Colluding Byzantine validators
- Extreme price manipulation
- Network partitions
- Eclipse attacks
- Sybil attacks (mitigated by stake-weighting)

**Verification Time:** ~20-30 minutes
**State Space:** ~3,500,000 states

---

## CI/CD Integration

### Automated Verification Triggers

Formal verification runs automatically on:

1. **Push to main/master/develop**
   - Verifies all specifications
   - Blocks merge if verification fails

2. **Pull Requests**
   - Validates syntax immediately
   - Runs full verification in parallel
   - Posts results as PR comment

3. **Daily Schedule (2 AM UTC)**
   - Deep verification with extended state space
   - Generates comprehensive reports
   - Alerts on failures

4. **Manual Dispatch**
   - On-demand verification via  Actions UI
   - Configurable parameters (memory, workers, spec)

### Workflow Stages

```
┌─────────────────┐
│ Syntax Check    │ (30 seconds)
│ - Fast SANY     │
│ - No model run  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Parallel TLC    │ (15-30 minutes)
│ - dex_invariant │
│ - escrow_safety │
│ - oracle_bft    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Spec Alignment  │ (1 minute)
│ - Check impl    │
│ - Verify match  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Summary Report  │
│ - Metrics       │
│ - Artifacts     │
└─────────────────┘
```

###  Actions Workflow

**File:** `hub/workflows/formal-verification.yml`

**Jobs:**
1. `syntax-check` - Fast TLA+ syntax validation
2. `verify-specs` - Parallel TLC model checking (matrix strategy)
3. `spec-alignment-check` - Implementation vs specification alignment
4. `verification-summary` - Aggregate results and generate report

**Artifacts:**
- TLC output files (30-day retention)
- Coverage reports (7-day retention)
- Verification summary (90-day retention)

---

## Local Development

### Prerequisites

```bash
# Install Java 11+
sudo apt-get install openjdk-17-jdk

# Set TLA+ tools location
export TLC_HOME=$HOME/tla-tools
mkdir -p $TLC_HOME

# Download TLC (automatic on first run)
cd formal
./verify.sh
```

### Quick Syntax Check

```bash
cd formal
./validate_syntax.sh
```

**Output:**
```
Validating TLA+ specifications...
  Checking dex_invariant... ✓
  Checking escrow_safety... ✓
  Checking oracle_bft... ✓

All specifications are syntactically valid!
```

### Verify Single Specification

```bash
cd formal
./verify.sh dex          # DEX only
./verify.sh escrow       # Escrow only
./verify.sh oracle       # Oracle only
```

### Verify All Specifications

```bash
cd formal
./verify.sh all          # Standard verification
./verify-all.sh          # Enhanced with metrics
./verify-all.sh --quick  # Fast (reduced state space)
./verify-all.sh --deep   # Thorough (may take hours)
```

### Docker-based Verification

```bash
# Build verification container
docker build -t paw-formal-verification formal/

# Run all verifications
docker run --rm paw-formal-verification

# Quick mode
docker run --rm paw-formal-verification --quick

# Mount results directory
docker run --rm -v $(pwd)/results:/formal/verification_results paw-formal-verification
```

---

## Pre-commit Hook

### Installation

```bash
# Link the hook
ln -s ../../hooks/pre-commit-formal /hooks/pre-commit

# Or copy it
cp hooks/pre-commit-formal /hooks/pre-commit
chmod +x /hooks/pre-commit
```

### What It Does

When you commit changes to `.tla` or `.cfg` files:

1. ✓ Runs fast syntax validation (~5 seconds)
2. ✓ Prevents committing broken specifications
3. ✓ Downloads TLC automatically if missing
4. ⚠ Note: Full model checking still runs in CI

### Bypassing (Not Recommended)

```bash
 commit --no-verify -m "message"
```

---

## Understanding Results

### Successful Verification

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Verification Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total specifications: 3
Passed: 3
Failed: 0

✓ All formal verifications PASSED!

┌─────────────────────────────────────────────────────────────┐
│  ✓ DEX Invariant: Constant product maintained             │
│  ✓ Escrow Safety: No double-spend possible                │
│  ✓ Oracle BFT: Byzantine fault tolerance verified         │
└─────────────────────────────────────────────────────────────┘
```

### Metrics Interpretation

```
States generated: 2,847,391    # Total states explored
Distinct states: 1,234,567     # Unique states (after deduplication)
Diameter: 18                   # Maximum depth of state graph
Duration: 45s                  # Verification time
```

**What This Means:**
- TLC explored ~2.8M state transitions
- Found ~1.2M unique states
- Maximum execution depth was 18 steps
- All invariants held in all states ✓

### Invariant Violation

If verification fails, TLC provides a **counterexample** - an exact sequence of steps that violates the invariant:

```
Error: Invariant NoDoubleSpend is violated.

State 1: <Initial State>
  escrows[1].status = "NONE"
  totalLocked = 0

State 2: LockEscrow(requestID: 1, amount: 100)
  escrows[1].status = "LOCKED"
  totalLocked = 100

State 3: InitiateRelease(requestID: 1)
  escrows[1].status = "CHALLENGED"

State 4: CompleteRelease(requestID: 1)
  escrows[1].status = "RELEASED"
  totalReleased = 100

State 5: RefundEscrow(requestID: 1)  <-- VIOLATION
  escrows[1].refundedAt = 1000
  totalRefunded = 100
  BOTH releasedAt AND refundedAt are non-zero!
```

**Action Items:**
1. Analyze the counterexample trace
2. Identify the bug (missing guard condition, etc.)
3. Fix the implementation OR update the specification
4. Re-run verification
5. Repeat until all invariants hold

---

## Troubleshooting

### TLC Out of Memory

**Symptom:**
```
java.lang.OutOfMemoryError: Java heap space
```

**Solutions:**

1. **Increase heap size:**
   ```bash
   # Edit verify.sh
   -Xmx8G  # Change to -Xmx16G
   ```

2. **Use disk-based fingerprint set:**
   ```bash
   -Dtlc2.tool.fp.FPSet.impl=tlc2.tool.fp.OffHeapDiskFPSet
   ```

3. **Reduce state space:**
   ```bash
   # Edit .cfg file
   CONSTRAINT
       /\ reserveA <= 5000  # was 10000
       /\ blockHeight <= 100  # was 200
   ```

### Verification Too Slow

**Solutions:**

1. **Use more workers:**
   ```bash
   -workers auto  # Use all CPU cores
   ```

2. **Enable symmetry:**
   ```
   SYMMETRY
       Permutations(TRADERS)
   ```

3. **Run quick mode:**
   ```bash
   ./verify-all.sh --quick
   ```

### TLC Not Found

**Symptom:**
```
TLC not found at /opt/TLA+Toolbox/tla2tools.jar
```

**Solutions:**

1. **Set TLC_HOME:**
   ```bash
   export TLC_HOME=$HOME/tla-tools
   ```

2. **Auto-download:**
   ```bash
   ./verify.sh  # Downloads automatically
   ```

3. **Manual download:**
   ```bash
   wget https://github.com/tlaplus/tlaplus/releases/download/v1.8.0/tla2tools.jar
   ```

---

## Adding New Specifications

### 1. Create TLA+ Specification

```bash
cd formal
touch new_module.tla
```

**Template:**
```tla
--------------------------- MODULE new_module ---------------------------
EXTENDS Naturals, Sequences

CONSTANTS MAX_VALUE, USERS

VARIABLES state, counter

vars == <<state, counter>>

TypeOK ==
    /\ state \in {"INIT", "ACTIVE", "DONE"}
    /\ counter \in Nat

Init ==
    /\ state = "INIT"
    /\ counter = 0

Next == ... (* Define transitions *)

Spec == Init /\ [][Next]_vars

(* SAFETY INVARIANTS *)
CounterNonNegative == counter >= 0

=============================================================================
```

### 2. Create Configuration File

```bash
touch new_module.cfg
```

**Template:**
```
SPECIFICATION Spec

CONSTANTS
    MAX_VALUE = 1000
    USERS = {u1, u2, u3}

CONSTRAINT
    counter <= 100

INVARIANTS
    TypeOK
    CounterNonNegative

CHECK_DEADLOCK TRUE
```

### 3. Add to Verification Scripts

**verify.sh:**
```bash
declare -a specs=("dex_invariant" "escrow_safety" "oracle_bft" "new_module")
```

**validate_syntax.sh:**
```bash
validate_spec "new_module" || ((failed++))
```

### 4. Update CI/CD

**hub/workflows/formal-verification.yml:**
```yaml
matrix:
  spec:
    - name: new_module
      file: new_module.tla
      cfg: new_module.cfg
      timeout: 20
      memory: 8g
      description: "New module description"
```

### 5. Test Locally

```bash
cd formal
./validate_syntax.sh  # Syntax check
./verify.sh new_module  # Full verification
```

### 6. Commit and Push

```bash
 add formal/new_module.tla formal/new_module.cfg
 commit -m "feat: add formal verification for new module"
 push
```

CI will automatically verify on push!

---

## Best Practices

### Specification Design

1. **Start Simple**
   - Begin with small state space
   - Add complexity incrementally
   - Test each invariant separately

2. **Use Constants Wisely**
   - Parameterize for flexibility
   - Use small values initially (faster verification)
   - Scale up for thorough testing

3. **State Constraints**
   - Bound infinite state spaces
   - Balance thoroughness vs. speed
   - Document constraint rationale

4. **Symmetry Optimization**
   - Identify symmetric states
   - Use SYMMETRY to reduce state space
   - Massive speedup for symmetric models

### Invariant Design

1. **Be Specific**
   - Vague invariants are hard to debug
   - Example: `balance > 0` not just `balance >= 0`

2. **Layered Invariants**
   - Type invariants (basic correctness)
   - Safety invariants (nothing bad happens)
   - Liveness properties (good things happen)

3. **Test Invariants**
   - Temporarily weaken to see violations
   - Verify counterexamples make sense
   - Re-strengthen once confident

### Performance Optimization

1. **Incremental Verification**
   - Verify small models first
   - Gradually increase state space
   - Profile to find bottlenecks

2. **Worker Tuning**
   - Use `-workers auto` for CPU-bound
   - Reduce workers if memory-bound
   - Monitor resource usage

3. **Checkpointing**
   - Enable for long-running verification
   - Resume after crashes
   - Trade disk space for reliability

---

## Integration with Development Workflow

### Development Cycle

```
1. Write Go Code
   ↓
2. Write/Update TLA+ Spec
   ↓
3. Local Syntax Check (pre-commit)
   ↓
4. Push to 
   ↓
5. CI Runs Formal Verification
   ↓
6. Review Results
   ↓
7. Fix Issues if Any
   ↓
8. Merge when Green ✓
```

### When to Update Specifications

Update TLA+ specs when:
- ✓ Adding new module functionality
- ✓ Changing critical algorithms
- ✓ Modifying state machines
- ✓ Fixing security vulnerabilities
- ✓ Refactoring core logic

**Don't need to update for:**
- ✗ UI changes
- ✗ Logging/debugging code
- ✗ Documentation
- ✗ Test code
- ✗ Performance optimizations (that preserve semantics)

### Specification Maintenance

**Monthly:**
- Run deep verification (`--deep` mode)
- Review and update constants
- Check for new Byzantine scenarios

**Quarterly:**
- Audit specification-implementation alignment
- Add refinement mappings
- Update threat models

**Annually:**
- Comprehensive specification review
- Consider unbounded verification (TLAPS)
- Publish verification report

---

## Resources

### TLA+ Learning

- [TLA+ Homepage](https://lamport.azurewebsites.net/tla/tla.html)
- [Specifying Systems (Book)](https://lamport.azurewebsites.net/tla/book.html)
- [TLA+ Video Course](https://lamport.azurewebsites.net/video/videos.html)
- [Learn TLA+](https://learntla.com/)

### TLC Model Checker

- [TLC Documentation](https://lamport.azurewebsites.net/tla/tlc.html)
- [TLA+ ](https://github.com/tlaplus/tlaplus)
- [TLA+ Examples](https://github.com/tlaplus/Examples)

### PAW-Specific

- [formal/README.md](../../../formal/README.md) - Detailed specification docs
- [formal/VERIFICATION_SUMMARY.md](../../../formal/VERIFICATION_SUMMARY.md) - Results
- [formal/CHECKLIST.md](../../../formal/CHECKLIST.md) - Verification checklist

---

## FAQ

**Q: Do I need to understand TLA+ to contribute code?**
A: No, but you should ensure your changes don't break formal verification. CI will alert you if they do.

**Q: How long does formal verification take in CI?**
A: 15-30 minutes for all three specifications in parallel.

**Q: What if my PR fails formal verification?**
A: Review the counterexample, fix the bug (in code or spec), and push again.

**Q: Can I skip formal verification for minor changes?**
A: Verification only runs when formal/ or core modules change. Minor changes won't trigger it.

**Q: How do I know if my implementation matches the spec?**
A: Review the spec, run spec-alignment-check job, and ensure invariants hold.

**Q: What's the difference between testing and formal verification?**
A: Testing checks specific cases; verification proves correctness for ALL cases.

**Q: Can formal verification find all bugs?**
A: It proves the specification correct. Bugs can still exist in implementation details not modeled.

**Q: Is formal verification overkill for blockchain?**
A: No! Financial systems require mathematical guarantees. Multiple blockchain hacks could have been prevented by formal verification.

---

## Support

For help with formal verification:
- Check the [Troubleshooting](#troubleshooting) section
- Review [formal/README.md](../../../formal/README.md)
- Ask in #formal-verification Slack channel
- File a  issue with `[formal-verification]` tag

---

*Last Updated: 2025-11-25*
*Version: 1.0*
