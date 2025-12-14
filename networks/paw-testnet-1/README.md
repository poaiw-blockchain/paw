# paw-testnet-1 Artifacts

This directory hosts the canonical files and metadata for the PAW public testnet (`paw-testnet-1`). Copy the latest outputs from `scripts/devnet/package-testnet-artifacts.sh` here before publishing them to operators.

## Files

| File | Description |
|------|-------------|
| `genesis.json` | Canonical network genesis file (upload + keep versioned commits). **Current file is a placeholder init (no accounts)** generated via `scripts/init-testnet.sh` for automation testing—replace with the real testnet genesis before publishing. |
| `genesis.sha256` | SHA256 checksum validators must verify before syncing |
| `peers.txt` | Seeds and persistent peers (public IP/FQDN + node IDs). **Currently empty** (local placeholder). Update with real sentry endpoints before sharing. |
| `checkpoints/` | (Optional) state sync snapshots or metadata |

## Recommended Endpoints

Fill these with live hosts once infrastructure is deployed:

| Service | Endpoint | Notes |
|---------|----------|-------|
| RPC     | `https://rpc1.paw-testnet.io` | Reverse proxy to canonical validator RPC |
| RPC     | `https://rpc2.paw-testnet.io` | Secondary RPC endpoint |
| REST    | `https://api.paw-testnet.io`  | REST gateway |
| gRPC    | `https://grpc.paw-testnet.io:443` | Envoy/ingress to gRPC port |
| Faucet  | `https://faucet.paw-testnet.io` | Backed by `scripts/faucet.sh` |
| Explorer| `https://explorer.paw-testnet.io` | Flask explorer or external service |

## Validator Bootstrapping Cheatsheet

```bash
# 1. Download artifacts
curl -L -o ~/.paw/config/genesis.json https://networks.paw.xyz/paw-testnet-1/genesis.json
curl -L https://networks.paw.xyz/paw-testnet-1/genesis.sha256
sha256sum ~/.paw/config/genesis.json  # compare to published checksum

# 2. Configure peers
curl -L -o ~/.paw/config/peers.txt https://networks.paw.xyz/paw-testnet-1/peers.txt
# Update config.toml to use the published seeds/persistent peers
```

For full onboarding instructions see `docs/guides/deployment/PUBLIC_TESTNET.md`. Track artifact readiness in `STATUS.md`.

## Maintenance Notes
-	Update `peers.txt` whenever seeds or sentry nodes change.
-	Keep `genesis.sha256` in sync with the committed `genesis.json`.
-	Record endpoint changes in this README so downstream docs can link here.
