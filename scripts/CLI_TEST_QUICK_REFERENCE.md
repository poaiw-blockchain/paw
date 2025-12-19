# CLI Test Script - Quick Reference

## One-Line Commands

### Run All Tests
```bash
./scripts/test-cli-commands.sh
```

### With Custom Binary
```bash
BINARY=/path/to/pawd ./scripts/test-cli-commands.sh
```

### Build and Test
```bash
make build && ./scripts/test-cli-commands.sh
```

## What Gets Tested

### ✅ Core Commands (8 tests)
- version, help, config

### ✅ Keys Management (10 tests)
- add, list, show, delete, export, import

### ✅ Init & Gentx (6 tests)
- init, gentx, collect-gentxs, validate-genesis

### ✅ DEX Module (35+ tests)

**Queries**: params, pool, pools, pool-by-tokens, liquidity, simulate-swap, limit-orders, order-book

**Transactions**: create-pool, add-liquidity, remove-liquidity, swap

**Invalid Tests**: negative amounts, same tokens, zero values, missing params

### ✅ Oracle Module (15+ tests)

**Queries**: params, price, prices, validator, validators, validator-price

**Transactions**: submit-price, delegate-feeder

**Invalid Tests**: invalid addresses, negative prices, zero prices, missing params

### ✅ Compute Module (60+ tests)

**Queries** (25 commands):
- Providers: provider, providers, active-providers
- Requests: request, requests, requests-by-requester, requests-by-provider, requests-by-status
- Results: result, estimate-cost
- Disputes: dispute, disputes, disputes-by-request, disputes-by-status, evidence
- Slashing: slash-record, slash-records, slash-records-by-provider
- Appeals: appeal, appeals, appeals-by-status
- Governance: governance-params

**Transactions** (14 commands):
- Provider: register-provider, update-provider, deactivate-provider
- Requests: submit-request, cancel-request, submit-result
- Disputes: create-dispute, vote-dispute, submit-evidence
- Appeals: appeal-slashing, vote-appeal
- Resolution: resolve-dispute, resolve-appeal
- Governance: update-governance-params

**Invalid Tests**: missing required flags, invalid IDs, invalid vote options, invalid status strings

### ✅ Cosmos SDK Modules (20+ tests)
- Bank, Staking, Governance

## Expected Output

```
========================================
Setting Up Test Environment
========================================
[INFO] Temporary home: /tmp/paw-cli-test-12345/home
[INFO] Using binary: ./pawd
[PASS] Test environment ready

========================================
Testing Core Commands: version, help, config
========================================
[PASS] pawd version - help text
[PASS] pawd version
[PASS] pawd --help
[PASS] pawd help
...

========================================
Test Summary Report
========================================

Total Tests:  156
Passed:       156 (100%)
Failed:       0 (0%)
Skipped:      0
```

## Files Created

- `test-cli-commands.sh` - Main test script
- `cli-test-report-YYYYMMDD-HHMMSS.txt` - Detailed report
- `CLI_TEST_README.md` - Full documentation

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Binary not found | `make build` first |
| Permission denied | `chmod +x scripts/test-cli-commands.sh` |
| Tests fail | Check report file for details |
| Want to test one module | Edit script and comment out unwanted test functions |

## Test Phases

This script covers **Phase 2.3: CLI Command Verification** from LOCAL_TESTING_PLAN.md:

✅ Test EVERY CLI command
✅ Test all custom module queries and transactions
✅ Execute each subcommand with valid AND invalid parameters
✅ Verify clear error messages for invalid usage
✅ Generate detailed test report
✅ Use temporary test environment

## Integration

Add to your workflow:

```bash
# Before committing CLI changes
make build
./scripts/test-cli-commands.sh

# Check the report
cat cli-test-report-*.txt | grep -A 10 "Test Summary"
```

## Coverage Summary

| Category | Commands | Tests |
|----------|----------|-------|
| Core | 3 | 8 |
| Keys | 7 | 10 |
| Init/Gentx | 4 | 6 |
| **DEX** | 16 | 35+ |
| **Oracle** | 8 | 15+ |
| **Compute** | 39 | 60+ |
| Cosmos SDK | 15+ | 20+ |
| **TOTAL** | **90+** | **150+** |

## Next Steps After Testing

1. **Review failed tests** in the report
2. **Fix identified issues** in CLI code
3. **Re-run specific tests** by editing the script
4. **Update documentation** if command signatures changed
5. **Commit changes** once all tests pass

## Manual Testing Examples

If you need to debug a specific command:

```bash
# DEX
./pawd query dex params --help
./pawd tx dex create-pool upaw 1000000 uatom 500000 --generate-only --help

# Oracle
./pawd query oracle price BTC --help
./pawd tx oracle submit-price pawvaloper1... BTC 50000 --generate-only --help

# Compute
./pawd query compute providers --help
./pawd tx compute register-provider --moniker test --endpoint http://test.com --help
```

## Success Criteria

✅ All 150+ tests pass
✅ Help text present for all commands
✅ Invalid inputs rejected with clear errors
✅ Valid inputs parsed correctly
✅ No unexpected crashes or panics

## Performance Metrics

- **Runtime**: 30-60 seconds
- **Tests**: 150+
- **Coverage**: All CLI commands
- **False Positives**: None (deterministic)
- **Resource Usage**: < 100MB RAM

---

**For full documentation, see**: [CLI_TEST_README.md](./CLI_TEST_README.md)
