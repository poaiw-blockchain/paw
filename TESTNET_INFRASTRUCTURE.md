# PAW Blockchain Testnet Infrastructure

> **Last Verified**: 2026-01-03
> **Maintainer**: Jeff DeCristofaro <info@poaiw.org>
> **License**: Apache 2.0

---

## Quick Reference

| Item | Value |
|------|-------|
| **Primary Server** | 54.39.103.49 (`ssh paw-testnet`) |
| **Secondary Server** | 139.99.149.160 (`ssh services-testnet`) |
| **VPN IP (primary)** | 10.10.0.2 |
| **VPN IP (secondary)** | 10.10.0.4 |
| **Chain ID** | paw-testnet-1 |
| **Denom** | upaw |
| **Binary** | `~/.paw/cosmovisor/genesis/bin/pawd` |
| **Home Dir** | `~/.paw` |
| **RPC (primary)** | 127.0.0.1:26657 |
| **P2P** | 0.0.0.0:26656 |
| **gRPC** | 127.0.0.1:9091 |
| **REST API** | 127.0.0.1:1317 (**not responding**) |

---

## Public Endpoints (Status as of 2026-01-03)

| Service | URL | Status | Notes |
| --- | --- | --- | --- |
| RPC | https://testnet-rpc.poaiw.org | **OK** | Nginx LB (primary + services backup) |
| REST API | https://testnet-api.poaiw.org | **Degraded** | REST port not listening on primary |
| gRPC | https://testnet-grpc.poaiw.org | **OK** | gRPC on 9091 |
| WebSocket | https://testnet-ws.poaiw.org | **OK** | WS proxy on 11082 |
| GraphQL | https://testnet-graphql.poaiw.org | **OK** | GraphQL gateway on 11100 |
| Explorer | https://testnet-explorer.poaiw.org | **OK** | UI + API |
| Faucet | https://testnet-faucet.poaiw.org | **OK** | UI + API |
| Status | https://status.poaiw.org | **OK** | Status page |
| Archive RPC | https://testnet-archive.poaiw.org | **Degraded** | Proxies primary RPC (archive is internal) |
| Docs | https://testnet-docs.poaiw.org | **OK** | Static docs |
| Snapshots | https://snapshots.poaiw.org | **OK** | Snapshot directory |
| Monitoring | https://monitoring.poaiw.org | **OK** | Grafana |

---

## Directory Structure

```
~/.paw/
├── config/
│   ├── app.toml          # Application configuration
│   ├── client.toml       # Client configuration
│   ├── config.toml       # CometBFT configuration
│   ├── genesis.json      # Chain genesis
│   ├── node_key.json     # Node identity key
│   └── priv_validator_key.json  # Validator signing key
├── cosmovisor/
│   ├── current -> genesis
│   └── genesis/
│       └── bin/
│           └── pawd      # Main binary
├── data/
│   ├── application.db/   # Application state (IAVL)
│   ├── blockstore.db/    # Block storage
│   ├── state.db/         # CometBFT state
│   ├── tx_index.db/      # Transaction index
│   ├── evidence.db/      # Evidence storage
│   ├── snapshots/        # State sync snapshots
│   └── priv_validator_state.json
└── logs/
    └── node.log          # Node output log

~/paw/                    # Source code repository
```

---

## Configuration (Key Settings)

### app.toml
```toml
[api]
enable = true
address = "tcp://localhost:1317"

[grpc]
enable = true
address = "localhost:9091"
```

### config.toml
```toml
[rpc]
laddr = "tcp://127.0.0.1:26657"

[p2p]
laddr = "tcp://0.0.0.0:26656"
```

---

## Services (Primary)

```
- pawd.service
- pawd-archive.service
- paw-explorer.service
- paw-faucet.service
- paw-graphql.service
- paw-websocket-proxy.service
- paw-status.service
```

---

## Archive Node (Primary Host)

| Component | Binding |
| --- | --- |
| P2P | 0.0.0.0:26686 |
| RPC | 127.0.0.1:26687 |

Archive RPC is **not** exposed publicly; `testnet-archive.poaiw.org` currently
proxies the primary RPC.

---

## Secondary Node (services-testnet)

| Component | Binding |
| --- | --- |
| RPC | 0.0.0.0:27657 |
| P2P | 0.0.0.0:27656 |
| REST | 0.0.0.0:1327 |
| gRPC | 0.0.0.0:9091 |

---

## Common Commands

```bash
# Node status
pawd status --home ~/.paw

# Query modules
pawd query compute params --home ~/.paw
pawd query dex pools --home ~/.paw
pawd query oracle prices --home ~/.paw
```

---

## Status

- **Network**: Active (public endpoints partially degraded)
- **Last Updated**: 2026-01-03
