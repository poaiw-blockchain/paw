# PAW Blockchain Configuration Templates

This directory contains configuration templates for PAW blockchain nodes.

## Files

- `node-config.toml.template` - Full node configuration template
- `validator-config.toml.template` - Validator node configuration template (security hardened)
- `app.toml.template` - Application configuration template (gas price, pruning, API/gRPC)
- `genesis-template.json` - Genesis file template

## Baseline Values (testnet-ready)

- Chain ID: `paw-testnet-1` (set explicitly when running `pawd init`)
- Bech32 prefix: `paw`
- Base denom: `upaw` (display: `PAW`)
- Minimum gas price: `0.001upaw`
- Pruning/min-retain: `pruning = custom`, `pruning-keep-recent = 500000`, `pruning-interval = 10`, `min-retain-blocks = 500000`
- Default keyring backend: `os` (native OS keyring)
- RPC defaults: bound to `127.0.0.1:26657`, CORS disabled (`[]`), `filter_peers = true`, limited gRPC/RPC connection caps.

## Environment Variables

Create a `.env` file with the following variables:

```bash
# Chain Configuration
CHAIN_ID=paw-testnet-1
MONIKER=my-node
MINIMUM_GAS_PRICES=0.001upaw
MIN_RETAIN_BLOCKS=500000

# Network Configuration
SEEDS=seed1@seed1.paw.network:26656,seed2@seed2.paw.network:26656
PERSISTENT_PEERS=

# Node Settings
ENABLE_API=true
ENABLE_GRPC=true
LOG_LEVEL=info

# Pruning Configuration
PRUNING=custom
PRUNING_KEEP_RECENT=100
PRUNING_INTERVAL=10

# State Sync (optional)
STATE_SYNC_ENABLE=false
STATE_SYNC_RPC_SERVERS=
STATE_SYNC_TRUST_HEIGHT=0
STATE_SYNC_TRUST_HASH=
```

## Usage

### Initialize a New Node

```bash
# Initialize node
pawd init my-node --chain-id paw-testnet-1 --home ~/.paw

# Copy configuration templates
cp app.toml.template ~/.paw/config/app.toml
cp node-config.toml.template ~/.paw/config/config.toml

# Edit configuration
nano ~/.paw/config/config.toml

# Download genesis file
curl -o ~/.paw/config/genesis.json https://genesis.paw.network/genesis.json

# Start node
pawd start --home ~/.paw
```

### Initialize a Validator

```bash
# Initialize validator
pawd init my-validator --chain-id paw-testnet-1 --home ~/.paw

# Copy validator configuration template
cp validator-config.toml.template ~/.paw/config/config.toml

# Edit configuration (set moniker, seeds, etc.)
nano ~/.paw/config/config.toml

# Download genesis file
curl -o ~/.paw/config/genesis.json https://genesis.paw.network/genesis.json

# IMPORTANT: Backup validator keys
cp ~/.paw/config/priv_validator_key.json ~/validator-keys-backup/
cp ~/.paw/config/node_key.json ~/validator-keys-backup/

# Start validator
pawd start --home ~/.paw

## Deterministic Testnet Init (Scripted)

Use the helper to build `pawd`, initialize the home, apply hardened templates, and patch seeds/peers deterministically:

```bash
./scripts/init-testnet.sh <moniker>
# Optional environment overrides:
#   PAW_HOME=~/.paw-testnet   PAW_CHAIN_ID=paw-testnet-1
#   SEEDS="id1@seed1:26656,id2@seed2:26656"   PERSISTENT_PEERS="id3@p2p:26656"
#   EXTERNAL_ADDRESS="1.2.3.4:26656"          MIN_GAS_PRICES=0.001upaw
#   PRUNING_KEEP_RECENT=500000 PRUNING_INTERVAL=10 MIN_RETAIN_BLOCKS=500000
#   OVERWRITE=true   MONIKER=my-node
```

The script **never touches git** and writes only to `${PAW_HOME}` (default `~/.paw`). Do not commit any files from `${PAW_HOME}`; back up keys and data separately per `docs/DISASTER_RECOVERY.md`.

## Genesis Template

`config/genesis-template.json` is generated from `pawd init` with the hardened consensus/evidence settings above for `paw-testnet-1`. Use it as the base for testnet genesis creation (fund accounts, tweak module params) and keep the canonical format (numeric fields serialized as strings).

## Network Hardening Defaults

- RPC bound to `127.0.0.1:26657` with CORS disabled; expose via a reverse proxy if remote access is needed.
- `filter_peers = true`, `addr_book_strict = true`, conservative connection limits (gRPC/RPC caps reduced).
- Set seeds/persistent peers via `SEEDS` / `PERSISTENT_PEERS` when running `scripts/init-testnet.sh` (or edit `config.toml` manually). For private/internal testnets, leave `SEEDS` empty and set `PERSISTENT_PEERS` to the known validator ring only.
- Leave `pprof_laddr` empty unless secured behind auth.
```

## Security Best Practices

### For Validators

1. **Restrict RPC Access**: Set `laddr = "tcp://127.0.0.1:26657"` to only allow local access
2. **Enable Peer Filtering**: Set `filter_peers = true`
3. **Use Sentry Architecture**: Configure `private_peer_ids` with sentry node IDs
4. **Backup Keys Regularly**: Store validator keys in secure, offline location
5. **Monitor Double Signs**: Enable `double_sign_check_height`
6. **Limit Connections**: Reduce `max_num_inbound_peers` and `max_num_outbound_peers`

### For Public Nodes

1. **Rate Limiting**: Configure appropriate connection limits
2. **CORS Settings**: Restrict `cors_allowed_origins` to known domains
3. **Resource Limits**: Set reasonable `max_open_connections` values
4. **Monitoring**: Enable Prometheus metrics
5. **Logging**: Use JSON format for better log aggregation

## Configuration Options

### Key Parameters

- **moniker**: Human-readable node name
- **log_level**: Logging verbosity (debug, info, warn, error)
- **db_backend**: Database backend (goleveldb, rocksdb, badgerdb)
- **timeout_commit**: Block commit timeout (affects block time)
- **max_num_inbound_peers**: Maximum incoming P2P connections
- **max_num_outbound_peers**: Maximum outgoing P2P connections
- **prometheus**: Enable Prometheus metrics endpoint

### Performance Tuning

For high-performance nodes:
```toml
[p2p]
send_rate = 10240000
recv_rate = 10240000
max_num_inbound_peers = 100
max_num_outbound_peers = 50

[mempool]
size = 10000
cache_size = 20000
```

For resource-constrained nodes:
```toml
[p2p]
send_rate = 2048000
recv_rate = 2048000
max_num_inbound_peers = 20
max_num_outbound_peers = 5

[mempool]
size = 2000
cache_size = 5000
```

## Observability

- Prometheus: enable in `config.toml` (`prometheus = true`, `prometheus_listen_addr = ":26660"`); keep behind firewall/proxy.
- Logging: use `log_format = "json"` for aggregation; apply logrotate to `pawd` output (see `docs/OBSERVABILITY.md`).
- Health checks: `RPC_ENDPOINT=http://127.0.0.1:26657 ./scripts/health-check.sh` probes `/status`, `/net_info`, `/validators` and fails fast on lag/zero peers/empty validator set.

## Troubleshooting

### Node Not Syncing

1. Check peer connections: `curl http://localhost:26657/net_info`
2. Verify seeds/persistent_peers are set correctly
3. Check if firewall allows port 26656
4. Review logs: `tail -f ~/.paw/paw.log`

### High Resource Usage

1. Reduce mempool size
2. Lower connection limits
3. Enable pruning
4. Consider state sync for initial sync

### Validator Not Signing

1. Verify validator is in active set
2. Check priv_validator_state.json is not corrupted
3. Ensure validator key matches registered key
4. Check for clock skew

## Support

For more information:
- Documentation: https://docs.paw.network
- Discord: https://discord.gg/paw
- : https://github.com/paw-chain/paw
