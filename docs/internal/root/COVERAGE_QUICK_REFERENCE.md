# Coverage Quick Reference

**Current Coverage:** 17.6% | **Target:** 90% | **Gap:** -72.4%

## Quick Commands

```bash
# Generate coverage report
make test-coverage

# Check if coverage meets threshold
make coverage-check

# View coverage in browser
make coverage-html

# Compare against baseline
make coverage-diff

# Create new baseline
make coverage-baseline
```

## Coverage by Module

| Module | Current | Target | Status |
|--------|---------|--------|--------|
| Compute | 16.5% | 90% | ❌ -73.5% |
| DEX | 16.1% | 90% | ❌ -73.9% |
| Oracle | 20.2% | 90% | ❌ -69.8% |
| P2P | 9.1% | 80% | ❌ -70.9% |
| App | 45.2% | 90% | ❌ -44.8% |
| Ante | 76.6% | 90% | ⚠️ -13.4% |
| IBCUtil | 94.1% | 90% | ✅ +4.1% |

## Top Priority Files (Lowest Coverage)

### Critical (<20% coverage)
- `cmd/pawd/cmd/collect_gentxs.go` - 3.6%
- `cmd/pawd/cmd/add_genesis_account.go` - 9.9%
- `x/oracle/keeper/oracle_advanced.go` - 15.2%
- `p2p/discovery/address_book.go` - 16.7%
- `x/oracle/keeper/abci.go` - 17.6%

### High Priority (20-50% coverage)
- `x/dex/keeper/ibc_aggregation.go` - 22.2%
- `x/compute/keeper/verification.go` - 25.0%
- `x/oracle/keeper/slashing.go` - 25.0%
- `x/compute/keeper/ratelimit.go` - 27.3%
- `x/compute/keeper/zk_enhancements.go` - 33.3%

### No Coverage (0%)
- All of `p2p/reputation/*` (8 files)
- All of `p2p/security/*` (2 files)
- All of `p2p/snapshot/*` (2 files)

## Coverage Files

```
coverage.out              # Raw coverage data
coverage.html             # Interactive browser report
coverage_by_package.txt   # Detailed per-function stats
coverage.baseline.out     # Baseline for comparison
COVERAGE_BASELINE.md      # Full analysis and roadmap
```

## Current Test Failures

1. **IBC Tests** - `tests/ibc/dex_cross_chain_test.go:243`
   - TestMultiHopSwap type assertion failure

2. **Recovery Tests** - `tests/recovery/crash_recovery_test.go:29`
   - Validator set initialization issues

3. **Verification Tests** - `tests/verification/state_machine_test.go`
   - State machine setup problems

## Coverage Thresholds

| Category | Threshold | Enforced By |
|----------|-----------|-------------|
| Overall | ≥90% | `make coverage-check` |
| Critical Modules | ≥90% | `scripts/check-coverage.sh` |
| Infrastructure | ≥80% | `scripts/check-coverage.sh` |
| CLI | ≥70% | `scripts/check-coverage.sh` |

## Improvement Roadmap

### Phase 1: Critical Modules (3 weeks)
- Compute: 16.5% → 75%
- DEX: 16.1% → 75%
- Oracle: 20.2% → 75%

### Phase 2: Infrastructure (2 weeks)
- P2P: 9.1% → 70%
- App: 45.2% → 90%

### Phase 3: CLI & Utilities (1 week)
- CLI: 49.1% → 80%
- Utilities: Variable → 70%

### Phase 4: Polish (1 week)
- All modules: 75% → 90%+

## CI/CD Integration

**Workflow:** `.github/workflows/coverage.yml`

**Triggers:**
- Push to `main` or `develop`
- Pull requests to `main` or `develop`

**Actions:**
- Run full test suite with coverage
- Generate reports
- Check against 90% threshold
- Compare against base branch
- Comment on PRs with coverage stats
- Upload artifacts

**Status:** Ready (will activate when GitHub Actions enabled)

## Local Development Workflow

```bash
# 1. Make changes to code
vim x/compute/keeper/verification.go

# 2. Add tests
vim x/compute/keeper/verification_test.go

# 3. Run tests with coverage
make test-coverage

# 4. Check if meets threshold
make coverage-check

# 5. View detailed report
make coverage-html

# 6. If satisfied, commit
git add .
git commit -m "test(compute): Improve verification coverage to 95%"
git push
```

## Troubleshooting

**Coverage check fails:**
```bash
# See which modules are below threshold
./scripts/check-coverage.sh

# Focus on specific module
go test ./x/compute/keeper/... -coverprofile=compute_coverage.out
go tool cover -func=compute_coverage.out
```

**Tests timing out:**
```bash
# Increase timeout
go test ./... -coverprofile=coverage.out -timeout=60m
```

**Coverage data stale:**
```bash
# Clean and regenerate
make clean
make test-coverage
```

## Related Documentation

- **Full Analysis:** `COVERAGE_BASELINE.md`
- **Coverage Script:** `scripts/check-coverage.sh`
- **CI Workflow:** `.github/workflows/coverage.yml`
- **Testing Guide:** `docs/testing/` (if exists)

---

**Last Updated:** 2025-12-14
**Baseline Coverage:** 17.6%
**Next Review:** After Phase 1 completion
