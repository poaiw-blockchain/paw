# PAW Configuration Testing Report

**Generated:** 2025-12-13 12:51:37
**Test Mode:** Exhaustive
**Category Filter:** All

## Summary

- **Total Tests:** TOTAL_PLACEHOLDER
- **Passed:** PASSED_PLACEHOLDER
- **Failed:** FAILED_PLACEHOLDER
- **Skipped:** SKIPPED_PLACEHOLDER
- **Blocked:** BLOCKED_PLACEHOLDER

## Test Results

### ✅ PASSED base/moniker-valid

- **Description:** Test valid moniker change
- **Config:** `config.toml :: moniker = "test-node-custom"`

### ✅ PASSED base/moniker-empty

- **Description:** Test empty moniker (should work with default)
- **Config:** `config.toml :: moniker = ""`

### ✅ PASSED base/db-backend-goleveldb

- **Description:** Test goleveldb backend (default)
- **Config:** `config.toml :: db_backend = "goleveldb"`

### ❌ FAILED (should have failed) base/db-backend-invalid

- **Description:** Test invalid database backend
- **Config:** `config.toml :: db_backend = "invaliddb"`
- **Error:** Node started successfully when it should have failed

<details><summary>Node Logs</summary>

```

```

</details>

### ✅ PASSED base/log-level-info

- **Description:** Test log level: info
- **Config:** `config.toml :: log_level = "info"`

### ✅ PASSED base/log-level-debug

- **Description:** Test log level: debug
- **Config:** `config.toml :: log_level = "debug"`

### ✅ PASSED base/log-level-error

- **Description:** Test log level: error
- **Config:** `config.toml :: log_level = "error"`

### ❌ FAILED (should have failed) base/log-level-invalid

- **Description:** Test invalid log level
- **Config:** `config.toml :: log_level = "invalid"`
- **Error:** Node started successfully when it should have failed

<details><summary>Node Logs</summary>

```

```

</details>

### ✅ PASSED base/log-format-plain

- **Description:** Test log format: plain
- **Config:** `config.toml :: log_format = "plain"`

### ✅ PASSED base/log-format-json

- **Description:** Test log format: json
- **Config:** `config.toml :: log_format = "json"`

### ✅ PASSED base/filter-peers-true

- **Description:** Test filter_peers enabled
- **Config:** `config.toml :: filter_peers = true`

### ✅ PASSED base/filter-peers-false

- **Description:** Test filter_peers disabled
- **Config:** `config.toml :: filter_peers = false`

### ❌ FAILED rpc/laddr-default

- **Description:** Test default RPC listen address
- **Config:** `config.toml :: rpc.laddr = "tcp://127.0.0.1:26657"`
- **Error:** Validation failed: RPC port  is not listening

<details><summary>Node Logs</summary>

```

```

</details>

### ✅ PASSED rpc/laddr-custom-port

- **Description:** Test custom RPC port
- **Config:** `config.toml :: rpc.laddr = "tcp://127.0.0.1:36657"`

### ✅ PASSED rpc/laddr-all-interfaces

- **Description:** Test RPC listen on all interfaces
- **Config:** `config.toml :: rpc.laddr = "tcp://0.0.0.0:26657"`

### ❌ FAILED (should have failed) rpc/laddr-invalid-format

- **Description:** Test invalid RPC address format
- **Config:** `config.toml :: rpc.laddr = "invalid-address"`
- **Error:** Node started successfully when it should have failed

<details><summary>Node Logs</summary>

```

```

</details>

### ✅ PASSED rpc/unsafe-true

- **Description:** Test unsafe RPC enabled
- **Config:** `config.toml :: rpc.unsafe = true`

### ✅ PASSED rpc/unsafe-false

- **Description:** Test unsafe RPC disabled (default)
- **Config:** `config.toml :: rpc.unsafe = false`

### ✅ PASSED rpc/max-open-connections-default

- **Description:** Test default max open connections (900)
- **Config:** `config.toml :: rpc.max_open_connections = 900`

### ✅ PASSED rpc/max-open-connections-low

- **Description:** Test low max open connections
- **Config:** `config.toml :: rpc.max_open_connections = 10`

### ✅ PASSED rpc/max-open-connections-high

- **Description:** Test high max open connections
- **Config:** `config.toml :: rpc.max_open_connections = 5000`

### ✅ PASSED rpc/max-open-connections-zero

- **Description:** Test unlimited connections (0)
- **Config:** `config.toml :: rpc.max_open_connections = 0`

### ✅ PASSED rpc/max-subscription-clients-default

- **Description:** Test default max subscription clients
- **Config:** `config.toml :: rpc.max_subscription_clients = 100`

### ✅ PASSED rpc/max-subscription-clients-high

- **Description:** Test high max subscription clients
- **Config:** `config.toml :: rpc.max_subscription_clients = 1000`

### ✅ PASSED rpc/timeout-broadcast-tx-commit-default

- **Description:** Test default broadcast timeout (10s)
- **Config:** `config.toml :: rpc.timeout_broadcast_tx_commit = "10s"`

### ❌ FAILED rpc/timeout-broadcast-tx-commit-long

- **Description:** Test long broadcast timeout (30s)
- **Config:** `config.toml :: rpc.timeout_broadcast_tx_commit = "30s"`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED rpc/max-body-bytes-default

- **Description:** Test default max body bytes (1MB)
- **Config:** `config.toml :: rpc.max_body_bytes = 1000000`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED rpc/max-body-bytes-large

- **Description:** Test large max body bytes (10MB)
- **Config:** `config.toml :: rpc.max_body_bytes = 10000000`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED rpc/cors-allow-all

- **Description:** Test CORS allow all origins
- **Config:** `config.toml :: rpc.cors_allowed_origins = ["*"]`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/laddr-default

- **Description:** Test default P2P listen address
- **Config:** `config.toml :: p2p.laddr = "tcp://0.0.0.0:26656"`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/laddr-custom-port

- **Description:** Test custom P2P port
- **Config:** `config.toml :: p2p.laddr = "tcp://0.0.0.0:36656"`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-inbound-peers-default

- **Description:** Test default max inbound peers (40)
- **Config:** `config.toml :: p2p.max_num_inbound_peers = 40`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-inbound-peers-low

- **Description:** Test low max inbound peers
- **Config:** `config.toml :: p2p.max_num_inbound_peers = 5`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-inbound-peers-high

- **Description:** Test high max inbound peers
- **Config:** `config.toml :: p2p.max_num_inbound_peers = 200`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-inbound-peers-zero

- **Description:** Test zero inbound peers
- **Config:** `config.toml :: p2p.max_num_inbound_peers = 0`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-outbound-peers-default

- **Description:** Test default max outbound peers (10)
- **Config:** `config.toml :: p2p.max_num_outbound_peers = 10`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/max-num-outbound-peers-high

- **Description:** Test high max outbound peers
- **Config:** `config.toml :: p2p.max_num_outbound_peers = 50`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/send-rate-default

- **Description:** Test default send rate (5120000 bytes/s)
- **Config:** `config.toml :: p2p.send_rate = 5120000`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/send-rate-low

- **Description:** Test low send rate (throttled)
- **Config:** `config.toml :: p2p.send_rate = 102400`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/send-rate-high

- **Description:** Test high send rate
- **Config:** `config.toml :: p2p.send_rate = 52428800`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/recv-rate-default

- **Description:** Test default receive rate
- **Config:** `config.toml :: p2p.recv_rate = 5120000`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED p2p/recv-rate-low

- **Description:** Test low receive rate
- **Config:** `config.toml :: p2p.recv_rate = 102400`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ✅ PASSED p2p/pex-enabled

- **Description:** Test peer exchange enabled (default)
- **Config:** `config.toml :: p2p.pex = true`

### ✅ PASSED p2p/pex-disabled

- **Description:** Test peer exchange disabled
- **Config:** `config.toml :: p2p.pex = false`

### ✅ PASSED p2p/seed-mode-enabled

- **Description:** Test seed mode enabled
- **Config:** `config.toml :: p2p.seed_mode = true`

### ✅ PASSED p2p/seed-mode-disabled

- **Description:** Test seed mode disabled (default)
- **Config:** `config.toml :: p2p.seed_mode = false`

### ✅ PASSED p2p/allow-duplicate-ip-true

- **Description:** Test allow duplicate IP enabled
- **Config:** `config.toml :: p2p.allow_duplicate_ip = true`

### ✅ PASSED p2p/allow-duplicate-ip-false

- **Description:** Test allow duplicate IP disabled (default)
- **Config:** `config.toml :: p2p.allow_duplicate_ip = false`

### ✅ PASSED p2p/handshake-timeout-default

- **Description:** Test default handshake timeout (20s)
- **Config:** `config.toml :: p2p.handshake_timeout = "20s"`

### ✅ PASSED p2p/handshake-timeout-short

- **Description:** Test short handshake timeout (5s)
- **Config:** `config.toml :: p2p.handshake_timeout = "5s"`

### ✅ PASSED p2p/handshake-timeout-long

- **Description:** Test long handshake timeout (60s)
- **Config:** `config.toml :: p2p.handshake_timeout = "60s"`

### ✅ PASSED p2p/dial-timeout-default

- **Description:** Test default dial timeout (3s)
- **Config:** `config.toml :: p2p.dial_timeout = "3s"`

### ✅ PASSED p2p/dial-timeout-short

- **Description:** Test short dial timeout (1s)
- **Config:** `config.toml :: p2p.dial_timeout = "1s"`

### ✅ PASSED p2p/flush-throttle-timeout-default

- **Description:** Test default flush throttle timeout (100ms)
- **Config:** `config.toml :: p2p.flush_throttle_timeout = "100ms"`

### ✅ PASSED p2p/flush-throttle-timeout-low

- **Description:** Test low flush throttle timeout (10ms)
- **Config:** `config.toml :: p2p.flush_throttle_timeout = "10ms"`

### ✅ PASSED p2p/max-packet-msg-payload-size-default

- **Description:** Test default max packet message payload size
- **Config:** `config.toml :: p2p.max_packet_msg_payload_size = 1024`

### ✅ PASSED p2p/max-packet-msg-payload-size-large

- **Description:** Test large max packet message payload size
- **Config:** `config.toml :: p2p.max_packet_msg_payload_size = 65536`

### ✅ PASSED mempool/type-flood

- **Description:** Test flood mempool type (default)
- **Config:** `config.toml :: mempool.type = "flood"`

### ✅ PASSED mempool/type-nop

- **Description:** Test nop mempool type
- **Config:** `config.toml :: mempool.type = "nop"`

### ✅ PASSED mempool/recheck-enabled

- **Description:** Test mempool recheck enabled (default)
- **Config:** `config.toml :: mempool.recheck = true`

### ✅ PASSED mempool/recheck-disabled

- **Description:** Test mempool recheck disabled
- **Config:** `config.toml :: mempool.recheck = false`

### ✅ PASSED mempool/broadcast-enabled

- **Description:** Test mempool broadcast enabled (default)
- **Config:** `config.toml :: mempool.broadcast = true`

### ✅ PASSED mempool/broadcast-disabled

- **Description:** Test mempool broadcast disabled
- **Config:** `config.toml :: mempool.broadcast = false`

### ✅ PASSED mempool/size-default

- **Description:** Test default mempool size (5000)
- **Config:** `config.toml :: mempool.size = 5000`

### ✅ PASSED mempool/size-small

- **Description:** Test small mempool size
- **Config:** `config.toml :: mempool.size = 100`

### ✅ PASSED mempool/size-large

- **Description:** Test large mempool size
- **Config:** `config.toml :: mempool.size = 50000`

### ✅ PASSED mempool/max-txs-bytes-default

- **Description:** Test default max txs bytes (1GB)
- **Config:** `config.toml :: mempool.max_txs_bytes = 1073741824`

### ✅ PASSED mempool/max-txs-bytes-small

- **Description:** Test small max txs bytes (10MB)
- **Config:** `config.toml :: mempool.max_txs_bytes = 10485760`

### ✅ PASSED mempool/cache-size-default

- **Description:** Test default cache size (10000)
- **Config:** `config.toml :: mempool.cache_size = 10000`

### ✅ PASSED mempool/cache-size-small

- **Description:** Test small cache size
- **Config:** `config.toml :: mempool.cache_size = 1000`

### ✅ PASSED mempool/cache-size-large

- **Description:** Test large cache size
- **Config:** `config.toml :: mempool.cache_size = 100000`

### ✅ PASSED mempool/max-tx-bytes-default

- **Description:** Test default max tx bytes (1MB)
- **Config:** `config.toml :: mempool.max_tx_bytes = 1048576`

### ✅ PASSED mempool/max-tx-bytes-small

- **Description:** Test small max tx bytes (10KB)
- **Config:** `config.toml :: mempool.max_tx_bytes = 10240`

### ✅ PASSED consensus/timeout-propose-default

- **Description:** Test default timeout propose
- **Config:** `config.toml :: consensus.timeout_propose = "3s"`

### ✅ PASSED consensus/timeout-propose-short

- **Description:** Test short timeout propose (1s)
- **Config:** `config.toml :: consensus.timeout_propose = "1s"`

### ✅ PASSED consensus/timeout-propose-long

- **Description:** Test long timeout propose (10s)
- **Config:** `config.toml :: consensus.timeout_propose = "10s"`

### ✅ PASSED consensus/timeout-commit-default

- **Description:** Test default timeout commit (5s)
- **Config:** `config.toml :: consensus.timeout_commit = "5s"`

### ✅ PASSED consensus/timeout-commit-zero

- **Description:** Test zero timeout commit
- **Config:** `config.toml :: consensus.timeout_commit = "0s"`

### ✅ PASSED consensus/skip-timeout-commit-true

- **Description:** Test skip timeout commit enabled
- **Config:** `config.toml :: consensus.skip_timeout_commit = true`

### ✅ PASSED consensus/skip-timeout-commit-false

- **Description:** Test skip timeout commit disabled (default)
- **Config:** `config.toml :: consensus.skip_timeout_commit = false`

### ✅ PASSED consensus/create-empty-blocks-true

- **Description:** Test create empty blocks enabled (default)
- **Config:** `config.toml :: consensus.create_empty_blocks = true`

### ✅ PASSED consensus/create-empty-blocks-false

- **Description:** Test create empty blocks disabled
- **Config:** `config.toml :: consensus.create_empty_blocks = false`

### ✅ PASSED consensus/create-empty-blocks-interval-default

- **Description:** Test default create empty blocks interval (0s)
- **Config:** `config.toml :: consensus.create_empty_blocks_interval = "0s"`

### ✅ PASSED consensus/create-empty-blocks-interval-custom

- **Description:** Test custom create empty blocks interval (30s)
- **Config:** `config.toml :: consensus.create_empty_blocks_interval = "30s"`

### ✅ PASSED consensus/peer-gossip-sleep-duration-default

- **Description:** Test default peer gossip sleep duration
- **Config:** `config.toml :: consensus.peer_gossip_sleep_duration = "100ms"`

### ✅ PASSED consensus/peer-gossip-sleep-duration-short

- **Description:** Test short peer gossip sleep duration
- **Config:** `config.toml :: consensus.peer_gossip_sleep_duration = "10ms"`

### ✅ PASSED statesync/enable-false

- **Description:** Test state sync disabled (default)
- **Config:** `config.toml :: statesync.enable = false`

### ✅ PASSED statesync/discovery-time-default

- **Description:** Test default discovery time (15s)
- **Config:** `config.toml :: statesync.discovery_time = "15s"`

### ✅ PASSED statesync/discovery-time-short

- **Description:** Test short discovery time (5s)
- **Config:** `config.toml :: statesync.discovery_time = "5s"`

### ✅ PASSED statesync/chunk-request-timeout-default

- **Description:** Test default chunk request timeout (10s)
- **Config:** `config.toml :: statesync.chunk_request_timeout = "10s"`

### ✅ PASSED statesync/chunk-fetchers-default

- **Description:** Test default chunk fetchers (4)
- **Config:** `config.toml :: statesync.chunk_fetchers = "4"`

### ✅ PASSED statesync/chunk-fetchers-many

- **Description:** Test many chunk fetchers (16)
- **Config:** `config.toml :: statesync.chunk_fetchers = "16"`

### ✅ PASSED storage/discard-abci-responses-false

- **Description:** Test keep ABCI responses (default)
- **Config:** `config.toml :: storage.discard_abci_responses = false`

### ✅ PASSED storage/discard-abci-responses-true

- **Description:** Test discard ABCI responses
- **Config:** `config.toml :: storage.discard_abci_responses = true`

### ✅ PASSED tx_index/indexer-kv

- **Description:** Test kv indexer (default)
- **Config:** `config.toml :: tx_index.indexer = "kv"`

### ✅ PASSED tx_index/indexer-null

- **Description:** Test null indexer (disabled)
- **Config:** `config.toml :: tx_index.indexer = "null"`

### ✅ PASSED instrumentation/prometheus-enabled

- **Description:** Test Prometheus metrics enabled
- **Config:** `config.toml :: instrumentation.prometheus = true`

### ❌ FAILED instrumentation/prometheus-disabled

- **Description:** Test Prometheus metrics disabled (default)
- **Config:** `config.toml :: instrumentation.prometheus = false`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ❌ FAILED instrumentation/prometheus-listen-addr-default

- **Description:** Test default Prometheus listen address
- **Config:** `config.toml :: instrumentation.prometheus_listen_addr = ":26660"`
- **Error:** Node failed to start

<details><summary>Node Logs</summary>

```
      --db_dir string                                   database directory (default "data")
      --genesis_hash bytesHex                           optional SHA-256 hash of the genesis file
      --grpc-only                                       Start the node in gRPC query only mode (no CometBFT process is started)
      --grpc-web.enable                                 Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled) (default true)
      --grpc.address string                             the gRPC server address to listen on (default "localhost:9090")
      --grpc.enable                                     Define if the gRPC server should be enabled (default true)
      --halt-height uint                                Block height at which to gracefully halt the chain and shutdown the node
      --halt-time uint                                  Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node
  -h, --help                                            help for start
      --home string                                     The application home directory (default "/home/hudson/.paw")
      --iavl-disable-fastnode                           Disable fast node for IAVL tree
      --inter-block-cache                               Enable inter-block caching (default true)
      --inv-check-period uint                           Assert registered invariants every N blocks
      --mempool.max-txs int                             Sets MaxTx value for the app-side mempool (default -1)
      --min-retain-blocks uint                          Minimum block height offset during ABCI commit to prune CometBFT blocks
      --minimum-gas-prices string                       Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)
      --moniker string                                  node name (default "bcpc")
      --p2p.external-address string                     ip:port address to advertise to peers for them to dial
      --p2p.laddr string                                node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:26656")
      --p2p.persistent_peers string                     comma-delimited ID@host:port persistent peers
      --p2p.pex                                         enable/disable Peer-Exchange (default true)
      --p2p.private_peer_ids string                     comma-delimited private peer IDs
      --p2p.seed_mode                                   enable/disable seed mode
      --p2p.seeds string                                comma-delimited ID@host:port seed nodes
      --p2p.unconditional_peer_ids string               comma-delimited IDs of unconditional peers
      --priv_validator_laddr string                     socket address to listen on for connections from external priv_validator process
      --proxy_app string                                proxy app address, or one of: 'kvstore', 'persistent_kvstore' or 'noop' for local testing. (default "tcp://127.0.0.1:26658")
      --pruning string                                  Pruning strategy (default|nothing|everything|custom) (default "default")
      --pruning-interval uint                           Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')
      --pruning-keep-recent uint                        Number of recent heights to keep on disk (ignored if pruning is not 'custom')
      --query-gas-limit uint                            Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.
      --rpc.grpc_laddr string                           GRPC listen address (BroadcastTx only). Port required
      --rpc.laddr string                                RPC listen address. Port required (default "tcp://127.0.0.1:26657")
      --rpc.pprof_laddr string                          pprof listen address (https://golang.org/pkg/net/http/pprof)
      --rpc.unsafe                                      enabled unsafe rpc methods
      --shutdown-grace duration                         On Shutdown, duration to wait for resource clean up
      --state-sync.snapshot-interval uint               State sync snapshot interval
      --state-sync.snapshot-keep-recent uint32          State sync snapshot to keep (default 2)
      --trace                                           Provide full stack traces for errors in ABCI Log
      --trace-store string                              Enable KVStore tracing to an output file
      --transport string                                Transport protocol: socket, grpc (default "socket")
      --unsafe-skip-upgrades ints                       Skip a set of upgrade heights to continue the old binary
      --with-comet                                      Run abci app embedded in-process with CometBFT (default true)
      --x-crisis-skip-assert-invariants                 Skip x/crisis invariants check on startup

Global Flags:
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>') (default "info")
      --log_no_color        Disable colored logs
```

</details>

### ✅ PASSED instrumentation/prometheus-listen-addr-custom

- **Description:** Test custom Prometheus listen address
- **Config:** `config.toml :: instrumentation.prometheus_listen_addr = ":9090"`

### ✅ PASSED app-base/minimum-gas-prices-default

- **Description:** Test default minimum gas prices
- **Config:** `app.toml :: minimum-gas-prices = "0.001upaw"`

### ✅ PASSED app-base/minimum-gas-prices-zero

- **Description:** Test zero minimum gas prices
- **Config:** `app.toml :: minimum-gas-prices = "0upaw"`

### ✅ PASSED app-base/minimum-gas-prices-high

- **Description:** Test high minimum gas prices
- **Config:** `app.toml :: minimum-gas-prices = "1.0upaw"`

### ✅ PASSED app-base/pruning-default

- **Description:** Test default pruning strategy
- **Config:** `app.toml :: pruning = "default"`

### ✅ PASSED app-base/pruning-nothing

- **Description:** Test pruning nothing (archive node)
- **Config:** `app.toml :: pruning = "nothing"`

### ✅ PASSED app-base/pruning-everything

- **Description:** Test pruning everything
- **Config:** `app.toml :: pruning = "everything"`

### ✅ PASSED app-base/pruning-custom

- **Description:** Test custom pruning strategy
- **Config:** `app.toml :: pruning = "custom"`

### ✅ PASSED app-base/halt-height-zero

- **Description:** Test halt height zero (disabled)
- **Config:** `app.toml :: halt-height = 0`

### ✅ PASSED app-base/halt-height-custom

- **Description:** Test custom halt height
- **Config:** `app.toml :: halt-height = 1000`

### ✅ PASSED app-base/inter-block-cache-enabled

- **Description:** Test inter-block cache enabled (default)
- **Config:** `app.toml :: inter-block-cache = true`

### ✅ PASSED app-base/inter-block-cache-disabled

- **Description:** Test inter-block cache disabled
- **Config:** `app.toml :: inter-block-cache = false`

### ✅ PASSED app-base/iavl-cache-size-default

- **Description:** Test default IAVL cache size
- **Config:** `app.toml :: iavl-cache-size = 781250`

### ✅ PASSED app-base/iavl-cache-size-small

- **Description:** Test small IAVL cache size
- **Config:** `app.toml :: iavl-cache-size = 10000`

### ✅ PASSED app-base/iavl-cache-size-large

- **Description:** Test large IAVL cache size
- **Config:** `app.toml :: iavl-cache-size = 5000000`

### ✅ PASSED app-base/iavl-disable-fastnode-false

- **Description:** Test IAVL fast node enabled (default)
- **Config:** `app.toml :: iavl-disable-fastnode = false`

### ✅ PASSED app-base/iavl-disable-fastnode-true

- **Description:** Test IAVL fast node disabled
- **Config:** `app.toml :: iavl-disable-fastnode = true`

### ✅ PASSED telemetry/enabled-true

- **Description:** Test telemetry enabled
- **Config:** `app.toml :: telemetry.enabled = true`

### ✅ PASSED telemetry/enabled-false

- **Description:** Test telemetry disabled (default)
- **Config:** `app.toml :: telemetry.enabled = false`

### ✅ PASSED telemetry/enable-hostname-true

- **Description:** Test enable hostname in telemetry
- **Config:** `app.toml :: telemetry.enable-hostname = true`

### ✅ PASSED telemetry/enable-hostname-label-true

- **Description:** Test enable hostname label in telemetry
- **Config:** `app.toml :: telemetry.enable-hostname-label = true`

### ✅ PASSED api/enable-false

- **Description:** Test API disabled (default)
- **Config:** `app.toml :: api.enable = false`

### ✅ PASSED api/enable-true

- **Description:** Test API enabled
- **Config:** `app.toml :: api.enable = true`

### ✅ PASSED api/swagger-enabled

- **Description:** Test Swagger enabled
- **Config:** `app.toml :: api.swagger = true`

### ✅ PASSED api/max-open-connections-default

- **Description:** Test default API max open connections
- **Config:** `app.toml :: api.max-open-connections = 1000`

### ✅ PASSED api/max-open-connections-low

- **Description:** Test low API max open connections
- **Config:** `app.toml :: api.max-open-connections = 100`

### ✅ PASSED api/rpc-read-timeout-default

- **Description:** Test default RPC read timeout (10s)
- **Config:** `app.toml :: api.rpc-read-timeout = 10`

### ✅ PASSED api/rpc-read-timeout-long

- **Description:** Test long RPC read timeout (60s)
- **Config:** `app.toml :: api.rpc-read-timeout = 60`

### ✅ PASSED grpc/enable-true

- **Description:** Test gRPC enabled (default)
- **Config:** `app.toml :: grpc.enable = true`

### ✅ PASSED grpc/enable-false

- **Description:** Test gRPC disabled
- **Config:** `app.toml :: grpc.enable = false`

### ✅ PASSED grpc/max-recv-msg-size-default

- **Description:** Test default gRPC max receive message size
- **Config:** `app.toml :: grpc.max-recv-msg-size = "10485760"`

### ✅ PASSED grpc/max-recv-msg-size-large

- **Description:** Test large gRPC max receive message size
- **Config:** `app.toml :: grpc.max-recv-msg-size = "52428800"`

### ✅ PASSED state-sync/snapshot-interval-zero

- **Description:** Test state sync snapshots disabled (default)
- **Config:** `app.toml :: state-sync.snapshot-interval = 0`

### ✅ PASSED state-sync/snapshot-interval-enabled

- **Description:** Test state sync snapshots enabled (every 1000 blocks)
- **Config:** `app.toml :: state-sync.snapshot-interval = 1000`

### ✅ PASSED state-sync/snapshot-keep-recent-default

- **Description:** Test default snapshot keep recent (2)
- **Config:** `app.toml :: state-sync.snapshot-keep-recent = 2`

### ✅ PASSED state-sync/snapshot-keep-recent-many

- **Description:** Test keep many recent snapshots (10)
- **Config:** `app.toml :: state-sync.snapshot-keep-recent = 10`

