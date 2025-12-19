# paw-testnet-1 Artifacts

This directory hosts the canonical files and metadata for the PAW public testnet (`paw-testnet-1`). Copy the latest outputs from `scripts/devnet/package-testnet-artifacts.sh` here before publishing them to operators.

## Files

| File | Description |
|------|-------------|
| `genesis.json` | Canonical network genesis file produced by `scripts/devnet/local-phase-d-rehearsal.sh` |
| `genesis.sha256` | SHA256 checksum validators must verify before syncing (matches the committed genesis) |
| `peers.txt` | Persistent peers from the 4-validator + 2-sentry rehearsal; seeds intentionally empty until public sentries are published (currently set to rpc1-4.paw-testnet.io:26656) |
| `paw-testnet-1-manifest.json` | Machine-readable manifest (chain_id, genesis sha, peers, pawd sha256, generated_at) |
| `artifacts/` (tarball) | Use `scripts/devnet/bundle-testnet-artifacts.sh` to produce `artifacts/paw-testnet-1-artifacts.tar.gz` for CDN upload; validate CDN copy with `scripts/devnet/validate-remote-artifacts.sh <cdn-url>` |
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
| Status  | `https://status.paw-testnet.io` | Served by `status-page/` (live RPC/REST/gRPC/Explorer/Faucet/Metrics probes) |
| Bundle  | `https://networks.paw-testnet.io/paw-testnet-1-artifacts.tar.gz` | Tarball (sha256=78b27d1c02196531b7907773874520447e0be2bee4b95b781085c9e11b6a90de; also see `paw-testnet-1-artifacts.sha256`) |

## Validator Bootstrapping Cheatsheet

```bash
# 1. Download artifacts
curl -L -o ~/.paw/config/genesis.json https://networks.paw.xyz/paw-testnet-1/genesis.json
curl -L https://networks.paw.xyz/paw-testnet-1/genesis.sha256
sha256sum ~/.paw/config/genesis.json  # compare to published checksum

# 2. Configure peers
curl -L -o ~/.paw/config/peers.txt https://networks.paw.xyz/paw-testnet-1/peers.txt
# Update config.toml to use the published seeds/persistent peers
# (seeds are intentionally blank until public sentries go live)
```

For full onboarding instructions see `docs/guides/deployment/PUBLIC_TESTNET.md`. Track artifact readiness in `STATUS.md`.

## Maintenance Notes
-	Update `peers.txt` whenever seeds or sentry nodes change.
-	Keep `genesis.sha256` in sync with the committed `genesis.json`.
-	Record endpoint changes in this README so downstream docs can link here.
