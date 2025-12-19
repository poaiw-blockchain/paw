# PAW Configuration Testing Report

**Generated:** 2025-12-13 12:50:59
**Test Mode:** Quick
**Category Filter:** All

## Summary

- **Total Tests:** 13
- **Passed:** 11
- **Failed:** 2
- **Skipped:** 0
- **Blocked:** 0

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

### ✅ PASSED rpc/max-open-connections-default

- **Description:** Test default max open connections
- **Config:** `config.toml :: rpc.max_open_connections = 900`

### ❌ FAILED p2p/laddr-default

- **Description:** Test default P2P listen address
- **Config:** `config.toml :: p2p.laddr = "tcp://0.0.0.0:26656"`
- **Error:** Validation failed: P2P port  is not listening

<details><summary>Node Logs</summary>

```

```

</details>

### ✅ PASSED p2p/max-num-inbound-peers-default

- **Description:** Test default max inbound peers
- **Config:** `config.toml :: p2p.max_num_inbound_peers = 40`

### ✅ PASSED p2p/handshake-timeout-default

- **Description:** Test default handshake timeout
- **Config:** `config.toml :: p2p.handshake_timeout = "20s"`

### ✅ PASSED mempool/size-default

- **Description:** Test default mempool size
- **Config:** `config.toml :: mempool.size = 5000`

### ✅ PASSED mempool/cache-size-default

- **Description:** Test default cache size
- **Config:** `config.toml :: mempool.cache_size = 10000`

### ✅ PASSED consensus/timeout-propose-default

- **Description:** Test default timeout propose
- **Config:** `config.toml :: consensus.timeout_propose = "3s"`

### ✅ PASSED consensus/timeout-commit-default

- **Description:** Test default timeout commit
- **Config:** `config.toml :: consensus.timeout_commit = "5s"`

### ✅ PASSED app-base/minimum-gas-prices-default

- **Description:** Test default minimum gas prices
- **Config:** `app.toml :: minimum-gas-prices = "0.001upaw"`

### ✅ PASSED app-base/pruning-default

- **Description:** Test default pruning strategy
- **Config:** `app.toml :: pruning = "default"`

### ✅ PASSED app-base/iavl-cache-size-default

- **Description:** Test default IAVL cache size
- **Config:** `app.toml :: iavl-cache-size = 781250`

