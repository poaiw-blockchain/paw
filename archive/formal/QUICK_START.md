# Formal Verification Quick Start Guide

## 30-Second Setup

```bash
cd formal
export TLC_HOME=/path/to/TLA+Toolbox  # or let script auto-download
chmod +x verify.sh
./verify.sh
```

## What Gets Verified

### DEX Module
- ✓ Constant product `k = x * y` never decreases on swaps
- ✓ Pool reserves always positive
- ✓ No arithmetic overflow
- ✓ LP shares proportional to ownership

### Escrow Module
- ✓ NO double-spend (funds released XOR refunded)
- ✓ NO double-release
- ✓ NO double-refund
- ✓ Atomic state transitions
- ✓ Challenge period enforced

### Oracle Module
- ✓ Byzantine fault tolerance (f < n/3)
- ✓ Price within honest validator range
- ✓ Outlier detection works
- ✓ 67% vote threshold enforced
- ✓ Byzantine validators cannot manipulate

## Expected Runtime

- **DEX verification**: ~30-60 seconds
- **Escrow verification**: ~30-50 seconds
- **Oracle verification**: ~40-70 seconds

**Total**: ~2-3 minutes for all proofs

## Output Interpretation

### Success
```
✓ Verification PASSED for dex_invariant
States Generated: 1,234,567
Distinct States: 567,890
```

### Failure
```
✗ Verification FAILED for dex_invariant
Error: Invariant NoDoubleSpend is violated
```

If verification fails, review the counterexample trace in `verification_results/`.

## Common Issues

### Issue: Java not found
```bash
sudo apt install openjdk-11-jdk  # Ubuntu/Debian
brew install openjdk@11          # macOS
```

### Issue: TLC not found
Script auto-downloads TLC. Or manually:
```bash
wget https://github.com/tlaplus/tlaplus/releases/download/v1.8.0/tla2tools.jar
export TLC_HOME=$(pwd)
```

### Issue: Out of memory
Increase heap in verify.sh:
```bash
java -Xmx8G ...  # Change from 4G to 8G
```

## Next Steps

1. **Understand the proofs**: Read `README.md`
2. **Modify specs**: Edit `.tla` files
3. **Adjust model**: Edit `.cfg` files
4. **Run deeper checks**: Increase state constraints

## Key Files

- `*.tla` - Formal specifications
- `*.cfg` - Model configurations
- `verify.sh` - Verification script
- `README.md` - Full documentation
- `verification_results/` - Proof outputs

## Quick Commands

```bash
# Verify all
./verify.sh all

# Verify one module
./verify.sh dex
./verify.sh escrow
./verify.sh oracle

# Generate report
./verify.sh report

# Syntax check only
./validate_syntax.sh
```

## What This Proves

These formal proofs mathematically guarantee that:

1. **DEX**: No trader can drain pool reserves or manipulate the constant product
2. **Escrow**: No funds can be double-spent under any concurrent execution
3. **Oracle**: Byzantine validators (f < n/3) cannot manipulate the aggregated price

**Confidence Level**: Exhaustive proof over bounded state space (millions of states checked)

## Integration

Add to CI/CD:
```yaml
- name: Formal Verification
  run: |
    cd formal
    ./verify.sh all
```

## Support

- TLA+ Documentation: https://lamport.azurewebsites.net/tla/tla.html
- Issues: File on 
- Questions: Check README.md first

---

**Remember**: Formal verification proves the *specification* correct. Always ensure the *implementation* matches the specification through code review, testing, and audits.
