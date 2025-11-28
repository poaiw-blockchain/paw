# Formal Verification CI/CD Quick Reference

## 🚀 Quick Commands

```bash
# Local syntax check (5 seconds)
cd formal && ./validate_syntax.sh

# Verify single spec (10-30 minutes)
./verify.sh dex          # DEX only
./verify.sh escrow       # Escrow only
./verify.sh oracle       # Oracle only

# Verify all specs (standard mode)
./verify.sh all

# Verify all with metrics
./verify-all.sh
./verify-all.sh --quick  # Fast mode
./verify-all.sh --deep   # Thorough mode

# Docker verification
docker build -t paw-formal .
docker run --rm paw-formal
```

---

## 📋 CI/CD Triggers

| Event | When | Duration | Specs Verified |
|-------|------|----------|----------------|
| **Push to main/master** | Every commit | 20-30 min | All 3 specs |
| **Pull Request** | On PR open/update | 20-30 min | All 3 specs |
| **Daily Schedule** | 2 AM UTC | 20-30 min | All 3 specs |
| **Manual Dispatch** | On-demand | Configurable | Selected spec(s) |

---

## 🔍 Workflow Stages

```
┌──────────────────┐
│ Syntax Check     │  30 sec   ✓ Fast SANY validation
└────────┬─────────┘
         │
         ├─────────────────────┐
         │                     │
         ▼                     ▼
┌──────────────────┐  ┌──────────────────┐
│ dex_invariant    │  │ escrow_safety    │
│ 10-15 min        │  │ 15-20 min        │
└────────┬─────────┘  └────────┬─────────┘
         │                     │
         │            ┌──────────────────┐
         │            │ oracle_bft       │
         │            │ 20-30 min        │
         │            └────────┬─────────┘
         │                     │
         └─────────┬───────────┘
                   │
                   ▼
         ┌──────────────────┐
         │ Spec Alignment   │  1 min
         └────────┬─────────┘
                  │
                  ▼
         ┌──────────────────┐
         │ Summary Report   │  1 min
         └──────────────────┘
```

---

## ✅ Success Indicators

###  Actions

```
✅ Syntax Check (30s)
✅ Verify dex_invariant (12m 34s)
✅ Verify escrow_safety (18m 12s)
✅ Verify oracle_bft (24m 56s)
✅ Spec Alignment (45s)
✅ Verification Summary (1m 2s)
```

### Local Output

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  PAW Blockchain Formal Verification Suite
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ All prerequisites satisfied

▶ Verifying dex_invariant...
✓ Verification PASSED
  States: 2,847,391 | Distinct: 1,234,567 | Diameter: 18

▶ Verifying escrow_safety...
✓ Verification PASSED
  States: 1,523,409 | Distinct: 987,654 | Diameter: 15

▶ Verifying oracle_bft...
✓ Verification PASSED
  States: 3,198,723 | Distinct: 2,105,392 | Diameter: 22

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Verification Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total: 3  Passed: 3  Failed: 0

✓ ALL FORMAL VERIFICATIONS PASSED!

┌─────────────────────────────────────────────────────────────┐
│  ✓ DEX Invariant: Constant product maintained             │
│  ✓ Escrow Safety: No double-spend possible                │
│  ✓ Oracle BFT: Byzantine fault tolerance verified         │
└─────────────────────────────────────────────────────────────┘
```

---

## ❌ Failure Handling

### Invariant Violation

```
❌ Error: Invariant NoDoubleSpend is violated

Counterexample Trace:
State 1: LockEscrow(requestID=1, amount=100)
State 2: InitiateRelease(requestID=1)
State 3: CompleteRelease(requestID=1)
State 4: RefundEscrow(requestID=1) ← VIOLATION
```

**Action:** Review counterexample, fix bug, re-run

### Syntax Error

```
❌ Syntax error in dex_invariant.tla:
   Line 245: Expected "==" but found "="
```

**Action:** Fix syntax, run `./validate_syntax.sh`

### Timeout

```
⚠ Verification exceeded 30 minute timeout
```

**Action:** Reduce state space in .cfg or increase timeout

---

## 📊 Artifacts

| Artifact | Retention | Location |
|----------|-----------|----------|
| TLC output files | 30 days | `tlc-output-{spec}` |
| Coverage reports | 7 days | `coverage-{spec}` |
| Verification summary | 90 days | `verification-summary` |
| Syntax validation | 7 days | `syntax-validation` |

---

## 🔧 Configuration

### Memory Settings

```yaml
# hub/workflows/formal-verification.yml
matrix:
  spec:
    - name: dex_invariant
      memory: 8g      # 8 GB heap
    - name: escrow_safety
      memory: 8g      # 8 GB heap
    - name: oracle_bft
      memory: 12g     # 12 GB heap (larger state space)
```

### Timeout Settings

```yaml
matrix:
  spec:
    - name: dex_invariant
      timeout: 20     # 20 minutes
    - name: escrow_safety
      timeout: 25     # 25 minutes
    - name: oracle_bft
      timeout: 30     # 30 minutes
```

### State Space Constraints

```
# formal/dex_invariant.cfg
CONSTRAINT
    /\ reserveA <= 10000
    /\ reserveB <= 10000
    /\ totalShares <= 10000
```

**Increase for deeper verification:**
```
CONSTRAINT
    /\ reserveA <= 50000   # More thorough
    /\ reserveB <= 50000
    /\ totalShares <= 50000
```

---

## 🔐 Proven Properties

### DEX Module
- ✓ k = x × y maintained (constant product)
- ✓ Reserves > 0 always
- ✓ No overflow
- ✓ K increases on swaps (fees)
- ✓ LP shares = proportional ownership

### Escrow Module
- ✓ No double-spend (release ⊕ refund)
- ✓ Exactly one outcome
- ✓ No double-release
- ✓ No double-refund
- ✓ Balance conservation
- ✓ Challenge period integrity

### Oracle Module
- ✓ f < n/3 Byzantine tolerance
- ✓ Price ∈ [min_honest, max_honest]
- ✓ Manipulation resistance
- ✓ Outlier detection (MAD, IQR)
- ✓ 67%+ vote threshold
- ✓ Freshness guaranteed

---

## 🐛 Troubleshooting

### Issue: TLC Out of Memory

**Symptoms:**
```
java.lang.OutOfMemoryError: Java heap space
```

**Solutions:**
1. Increase heap: `-Xmx16G`
2. Reduce state space in .cfg
3. Use disk-based fingerprints (already enabled)

### Issue: Verification Too Slow

**Solutions:**
1. Use `--quick` mode for development
2. Reduce CONSTRAINT bounds in .cfg
3. Enable more SYMMETRY optimizations
4. Use more workers: `-workers auto`

### Issue: False Positive Invariant Violations

**Symptoms:**
```
Invariant violated but behavior seems correct
```

**Solutions:**
1. Review specification vs implementation
2. Check if invariant is too strict
3. Update specification if needed
4. Re-verify after changes

---

## 📝 Pre-commit Hook

**Installation:**
```bash
ln -s ../../hooks/pre-commit-formal /hooks/pre-commit
```

**What it does:**
- ✓ Syntax validation (5 seconds)
- ✓ Prevents broken commits
- ✓ Auto-downloads TLC
- ⚠ Full verification still in CI

**Bypass (not recommended):**
```bash
 commit --no-verify
```

---

## 🎯 When to Update Specs

**Update specifications when:**
- ✅ Adding new module functionality
- ✅ Changing state machine logic
- ✅ Modifying critical algorithms
- ✅ Fixing security bugs
- ✅ Refactoring core components

**No update needed for:**
- ❌ UI changes
- ❌ Logging/debugging
- ❌ Documentation
- ❌ Test code
- ❌ Performance optimizations (semantic-preserving)

---

## 📈 Metrics Guide

```
Model checking completed. No error has been found.
  Estimates of the probability that TLC did not check all reachable states
  because two distinct states had the same fingerprint:
  calculated (optimistic):  val = 1.2E-11
  based on the actual fingerprints:  val = 3.4E-12
2847391 states generated, 1234567 distinct states found, 0 states left on queue.
The depth of the complete state graph search is 18.
Finished in 12min 34s at (2025-11-25 20:42:00)
```

**Key Metrics:**
- **States Generated:** Total transitions explored (2.8M)
- **Distinct States:** Unique states found (1.2M)
- **Diameter:** Max depth of state graph (18 steps)
- **Fingerprint Collision:** Probability of missed states (negligible)
- **Duration:** Total verification time (12m 34s)

---

## 🔗 Quick Links

| Resource | Link |
|----------|------|
| **Full Guide** | [FORMAL_VERIFICATION_GUIDE.md](../docs/implementation/testing/FORMAL_VERIFICATION_GUIDE.md) |
| **Specifications** | [formal/](.) |
| **Workflow** | [hub/workflows/formal-verification.yml](../hub/workflows/formal-verification.yml) |
| **TLA+ Homepage** | https://lamport.azurewebsites.net/tla/tla.html |
| **TLC Docs** | https://lamport.azurewebsites.net/tla/tlc.html |

---

## 💡 Pro Tips

1. **Run quick mode during development:**
   ```bash
   ./verify-all.sh --quick
   ```

2. **Use Docker for reproducibility:**
   ```bash
   docker run --rm paw-formal
   ```

3. **Check syntax before full verification:**
   ```bash
   ./validate_syntax.sh && ./verify.sh all
   ```

4. **Review counterexamples carefully:**
   - They show EXACT sequence to violation
   - Usually reveal missing guard conditions
   - Help identify edge cases

5. **Monitor CI artifacts:**
   - Download TLC output for detailed analysis
   - Check coverage reports for missed states
   - Review summary for trends

6. **Parallel local verification:**
   ```bash
   ./verify-all.sh --parallel  # Experimental
   ```

---

## 🎓 Learning Path

1. **Week 1:** Read specifications, understand invariants
2. **Week 2:** Run local verification, interpret results
3. **Week 3:** Modify .cfg files, experiment with constraints
4. **Week 4:** Write simple spec, verify it works
5. **Month 2:** Add new invariants to existing specs
6. **Month 3:** Create specification for new module

---

## 📞 Support

**Quick Help:**
- Syntax errors → Check [TLA+ syntax guide](https://lamport.azurewebsites.net/tla/summary.pdf)
- Verification failures → Review counterexample
- Performance issues → Reduce state space
- CI failures → Check  Actions logs

**Need More Help?**
- File issue: `[formal-verification]` tag
- Slack: #formal-verification
- Email: formal-verification@paw-chain.org

---

*Quick Reference v1.0 | Last Updated: 2025-11-25*
