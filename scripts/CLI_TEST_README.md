# PAW CLI Command Verification Script

## Overview

The `test-cli-commands.sh` script provides comprehensive testing for **all** CLI commands in the PAW blockchain, covering:

- Core commands (version, help, config)
- Key management (keys add, list, show, delete, etc.)
- Initialization commands (init, gentx, collect-gentxs)
- Custom modules (DEX, Oracle, Compute)
- Standard Cosmos SDK modules (bank, staking, gov, etc.)

## Features

✅ **Comprehensive Coverage**: Tests every CLI command and subcommand
✅ **Validation Testing**: Tests both valid and invalid parameter combinations
✅ **Help Text Verification**: Ensures all commands have proper help documentation
✅ **Detailed Reporting**: Generates a timestamped report with pass/fail status
✅ **Isolated Environment**: Uses temporary directories and test keyring to avoid pollution
✅ **No Node Required**: Tests CLI parsing without needing a running blockchain node

## Prerequisites

1. **Build the binary**:
   ```bash
   cd /home/hudson/blockchain-projects/paw
   make build
   ```

2. **Verify binary location**:
   ```bash
   ./pawd version
   ```

## Usage

### Basic Usage

Run all tests with default settings:

```bash
cd /home/hudson/blockchain-projects/paw
./scripts/test-cli-commands.sh
```

### Custom Binary Location

If your binary is in a different location:

```bash
BINARY=/path/to/pawd ./scripts/test-cli-commands.sh
```

### Run from Any Directory

```bash
cd /home/hudson/blockchain-projects/paw
BINARY=$(pwd)/pawd ./scripts/test-cli-commands.sh
```

## Output

### Console Output

The script provides real-time colored output showing:
- **[PASS]** - Test passed
- **[FAIL]** - Test failed
- **[INFO]** - Informational messages
- **[WARN]** - Warnings

Example:
```
========================================
Testing DEX Query Commands
========================================
[PASS] query dex - help text
[PASS] query dex params - help text
[PASS] query dex pool - help text
[FAIL] query dex pool without ID
  Output: Error: accepts 1 arg(s), received 0
```

### Test Report

A detailed report is saved to `cli-test-report-YYYYMMDD-HHMMSS.txt` containing:

1. **Test Summary**:
   - Total tests run
   - Pass/fail counts and percentages
   - Skipped tests

2. **Failed Test Details**:
   - Test name
   - Expected vs actual result
   - Command output (truncated)

Example report:
```
========================================
Test Summary Report
========================================

Total Tests:  156
Passed:       152 (97%)
Failed:       4 (3%)
Skipped:      0

Failed Tests:

  - tx dex create-pool same tokens
    Expected: fail, Got: exit=0, Output: Error: tokens must be different
```

## Test Categories

### 1. Core Commands (8 tests)
- `pawd version`
- `pawd help`
- `pawd config`

### 2. Key Management (10 tests)
- Valid: list, show, add
- Invalid: nonexistent keys, missing parameters

### 3. Initialization (6 tests)
- Valid: init with proper parameters
- Invalid: missing moniker, missing chain-id

### 4. DEX Module (35+ tests)

**Query Commands**:
- `query dex params`
- `query dex pool [id]`
- `query dex pools`
- `query dex pool-by-tokens [token-a] [token-b]`
- `query dex liquidity [pool-id] [provider]`
- `query dex simulate-swap [pool-id] [token-in] [token-out] [amount]`
- `query dex limit-order [order-id]`
- `query dex limit-orders`
- `query dex orders-by-owner [address]`
- `query dex orders-by-pool [pool-id]`
- `query dex order-book [pool-id]`

**Transaction Commands**:
- `tx dex create-pool [token-a] [amount-a] [token-b] [amount-b]`
  - Invalid: same tokens, negative/zero amounts, invalid amounts
- `tx dex add-liquidity [pool-id] [amount-a] [amount-b]`
  - Invalid: missing/invalid pool ID, negative amounts
- `tx dex remove-liquidity [pool-id] [shares]`
  - Invalid: zero/negative shares
- `tx dex swap [pool-id] [token-in] [amount-in] [token-out] [min-amount-out]`
  - Invalid: same tokens, negative amounts

### 5. Oracle Module (15+ tests)

**Query Commands**:
- `query oracle params`
- `query oracle price [asset]`
- `query oracle prices`
- `query oracle validator [address]`
- `query oracle validators`
- `query oracle validator-price [validator] [asset]`

**Transaction Commands**:
- `tx oracle submit-price [validator] [asset] [price]`
  - Invalid: invalid validator, invalid/negative/zero price
- `tx oracle delegate-feeder [delegate-address]`
  - Invalid: invalid address

### 6. Compute Module (60+ tests)

**Query Commands** (25 commands):
- Provider queries: `provider`, `providers`, `active-providers`
- Request queries: `request`, `requests`, `requests-by-requester`, `requests-by-provider`, `requests-by-status`
- Result queries: `result`, `estimate-cost`
- Dispute queries: `dispute`, `disputes`, `disputes-by-request`, `disputes-by-status`, `evidence`
- Slash queries: `slash-record`, `slash-records`, `slash-records-by-provider`
- Appeal queries: `appeal`, `appeals`, `appeals-by-status`
- Governance: `governance-params`

**Transaction Commands** (14 commands):
- Provider: `register-provider`, `update-provider`, `deactivate-provider`
- Request: `submit-request`, `cancel-request`, `submit-result`
- Dispute: `create-dispute`, `vote-dispute`, `submit-evidence`
- Appeal: `appeal-slashing`, `vote-appeal`
- Resolution: `resolve-dispute`, `resolve-appeal`
- Governance: `update-governance-params`

### 7. Standard Cosmos SDK Modules (20+ tests)
- **Bank**: balances, send
- **Staking**: validators, delegate, unbond, redelegate
- **Governance**: proposals, vote, deposit

## Test Methodology

### Help Text Verification

Every command is tested for proper help output:
```bash
pawd query dex --help
# Should output: Usage, Available Commands, Flags
```

### Valid Parameter Tests

Commands are tested with valid parameters using `--generate-only` to avoid node requirement:
```bash
pawd tx dex create-pool upaw 1000000 uatom 500000 \
  --from test-key \
  --generate-only
# Should exit with code 0
```

### Invalid Parameter Tests

Commands are tested with various invalid inputs:
- Missing required parameters
- Invalid data types (strings instead of numbers)
- Invalid ranges (negative, zero)
- Invalid addresses
- Same token in both positions
- Nonexistent IDs

## Understanding Results

### Expected Behaviors

**Pass (✓)**:
- Valid commands parse correctly
- Invalid commands fail with clear error messages
- Help text is present and properly formatted

**Fail (✗)**:
- Valid commands fail to parse
- Invalid commands succeed when they should fail
- Missing or malformed help text

### Common Failure Reasons

1. **Missing validation**: Command accepts invalid parameters
2. **Poor error messages**: Command fails but doesn't explain why
3. **Missing help text**: No documentation for the command
4. **Incorrect parsing**: Command misinterprets parameters

## Extending the Tests

### Adding New Tests

1. **Find the appropriate section** in `test-cli-commands.sh`
2. **Add help text test**:
   ```bash
   test_help "query mymodule mycommand" "$BINARY" query mymodule mycommand
   ```

3. **Add validation tests**:
   ```bash
   # Valid case (if testable without node)
   run_test "query mymodule mycommand valid" "pass" \
     "$BINARY" query mymodule mycommand valid-param --home "$TEST_HOME"

   # Invalid cases
   run_test "query mymodule mycommand missing param" "fail" \
     "$BINARY" query mymodule mycommand --home "$TEST_HOME"

   run_test "query mymodule mycommand invalid param" "fail" \
     "$BINARY" query mymodule mycommand "invalid" --home "$TEST_HOME"
   ```

### Adding New Test Sections

Create a new function following the pattern:

```bash
test_mymodule_commands() {
    section "Testing MyModule Commands"

    # Help texts
    test_help "query mymodule" "$BINARY" query mymodule
    test_help "tx mymodule" "$BINARY" tx mymodule

    # Valid tests
    run_test "..." "pass" "$BINARY" ...

    # Invalid tests
    run_test "..." "fail" "$BINARY" ...
}
```

Then call it from `main()`:
```bash
test_mymodule_commands
```

## Troubleshooting

### Binary Not Found

**Error**: `pawd binary not found`

**Solution**:
```bash
# Build the binary first
make build

# Or specify the location
BINARY=/path/to/pawd ./scripts/test-cli-commands.sh
```

### Permission Denied

**Error**: `Permission denied`

**Solution**:
```bash
chmod +x ./scripts/test-cli-commands.sh
```

### Tests Hanging

If tests hang, it usually means a command is waiting for input. Press `Ctrl+C` and check:
1. The test uses `--generate-only` for tx commands
2. The test uses `--home "$TEST_HOME"` for isolation
3. The test doesn't require user interaction

### Unexpected Failures

If tests fail unexpectedly:

1. **Check the report** for detailed output
2. **Run the command manually** to debug:
   ```bash
   ./pawd query dex pool --help
   ```
3. **Verify binary is up to date**:
   ```bash
   make build
   ```

## Integration with CI/CD

This script can be integrated into continuous integration:

```yaml
# Example GitHub Actions
- name: Test CLI Commands
  run: |
    make build
    ./scripts/test-cli-commands.sh
```

## Performance

- **Runtime**: ~30-60 seconds (depending on system)
- **Test Count**: 150+ tests
- **Resource Usage**: Minimal (no blockchain node required)

## Maintenance

### Regular Updates

Update the script when:
- New CLI commands are added
- Command signatures change
- New validation rules are implemented
- New modules are added

### Version Compatibility

This script is compatible with PAW v1.0+. For older versions, some tests may need adjustment.

## Related Documentation

- [LOCAL_TESTING_PLAN.md](../LOCAL_TESTING_PLAN.md) - Overall testing strategy
- [CLAUDE.md](../CLAUDE.md) - Development guidelines
- [docs/](../docs/) - Module-specific documentation

## Support

For issues or questions:
1. Check the test report for detailed failure information
2. Review the command's help text: `pawd <command> --help`
3. Check module documentation in `x/<module>/README.md`
4. Review CLI implementation in `x/<module>/client/cli/`

## License

This script is part of the PAW blockchain project and follows the same license.
