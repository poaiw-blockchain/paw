# paw-testnet-1 Artifacts

This directory hosts the canonical files and metadata for the PAW public testnet (`paw-testnet-1`).

## Files

| File | Description |
|------|-------------|
| `genesis.json` | Canonical network genesis file |
| `genesis.sha256` | SHA256 checksum validators must verify before syncing |
| `peers.txt` | Persistent peers for 4-validator testnet |
| `paw-testnet-1-manifest.json` | Machine-readable manifest (chain_id, genesis sha, peers, endpoints, validator details) |
| `chain.json` | Chain registry format configuration |

## Current Public Endpoints

| Service | Endpoint | Notes |
|---------|----------|-------|
| RPC     | `https://testnet-rpc.poaiw.org` | Load-balanced across val1-4 |
| REST    | `https://testnet-api.poaiw.org` | Load-balanced across val1-4 |
| gRPC    | `testnet-grpc.poaiw.org:443` | Load-balanced across val1-4 |
| WebSocket | `wss://testnet-ws.poaiw.org` | WebSocket proxy |
| Faucet  | `https://testnet-faucet.poaiw.org` | Public faucet UI + API |
| Explorer| `https://testnet-explorer.poaiw.org` | Block explorer |
| Status  | `https://status.poaiw.org` | Health monitoring |

## Validator Infrastructure

| Validator | Server | IP | P2P | RPC | gRPC | REST |
|-----------|--------|-----|-----|-----|------|------|
| val1 | paw-testnet | 54.39.103.49 | 11656 | 11657 | 11090 | 11317 |
| val2 | paw-testnet | 54.39.103.49 | 11756 | 11757 | 11190 | 11417 |
| val3 | services-testnet | 139.99.149.160 | 11856 | 11857 | 11290 | 11517 |
| val4 | services-testnet | 139.99.149.160 | 11956 | 11957 | 11390 | 11617 |

## Validator Bootstrapping

```bash
# 1. Download and verify genesis
curl -L -o ~/.paw/config/genesis.json https://testnet-explorer.poaiw.org/genesis.json
sha256sum ~/.paw/config/genesis.json  # compare to published checksum

# 2. Configure peers (from peers.txt or manifest)
# Add to config.toml:
# persistent_peers = "72c594a424bfc156381860feaca3a2586173eead@54.39.103.49:11656,..."
```

For full onboarding instructions see `docs/guides/deployment/PUBLIC_TESTNET.md`.

## Maintenance Notes
- Update `peers.txt` whenever validator node IDs change.
- Keep `genesis.sha256` in sync with the committed `genesis.json`.
- Record endpoint changes in this README so downstream docs can link here.
