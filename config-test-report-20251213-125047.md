# PAW Configuration Testing Report

**Generated:** 2025-12-13 12:50:47
**Test Mode:** Quick
**Category Filter:** All

## Summary

- **Total Tests:** TOTAL_PLACEHOLDER
- **Passed:** PASSED_PLACEHOLDER
- **Failed:** FAILED_PLACEHOLDER
- **Skipped:** SKIPPED_PLACEHOLDER
- **Blocked:** BLOCKED_PLACEHOLDER

## Test Results

### ✅ PASSED base/log-level-info

- **Description:** Test log level: info
- **Config:** `config.toml :: log_level = "info"`

### ❌ FAILED rpc/laddr-default

- **Description:** Test default RPC listen address
- **Config:** `config.toml :: rpc.laddr = "tcp://127.0.0.1:26657"`
- **Error:** Validation failed: RPC port  is not listening

<details><summary>Node Logs</summary>

```

```

</details>

