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

## Current Public Endpoints (Status as of 2026-01-03)

| Service | Endpoint | Status | Notes |
|---------|----------|--------|-------|
| RPC     | `https://testnet-rpc.poaiw.org` | OK | Reverse proxy to validator RPC |
| REST    | `https://testnet-api.poaiw.org`  | Degraded | REST port not responding on primary |
| gRPC    | `testnet-rpc.poaiw.org:9091` | Degraded | Public gRPC host currently misrouted |
| Faucet  | `https://testnet-faucet.poaiw.org` | OK | Public faucet UI + API |
| Explorer| `https://testnet-explorer.poaiw.org` | OK | Static UI + API |
| Status  | `https://status.poaiw.org` | OK | Status page (RPC/REST/gRPC/Explorer/Faucet/Metrics probes) |
| Stats  | `https://testnet-explorer.poaiw.org/stats/` | OK | Network stats dashboard |
| GraphQL | `https://testnet-graphql.poaiw.org` | OK | GraphQL gateway |
| GraphQL (via API) | `https://testnet-api.poaiw.org/graphql` | OK | GraphQL proxy |
| Chain Registry | `https://testnet-explorer.poaiw.org/chain-registry/chain.json` | OK | Chain registry JSON |
| SDK | `https://testnet-explorer.poaiw.org/sdk/` | OK | Python SDK artifacts |

## Validator Bootstrapping Cheatsheet

```bash
# 1. Download artifacts
curl -L -o ~/.paw/config/genesis.json https://testnet-explorer.poaiw.org/genesis.json
curl -L https://testnet-explorer.poaiw.org/genesis.json.sha256
sha256sum ~/.paw/config/genesis.json
sha256sum ~/.paw/config/genesis.json  # compare to published checksum

# 2. Configure peers
curl -L -o ~/.paw/config/peers.txt https://testnet-explorer.poaiw.org/peers.txt
# Update config.toml to use the published seeds/persistent peers
# (seeds are intentionally blank until public sentries go live)
```

For full onboarding instructions see `docs/guides/deployment/PUBLIC_TESTNET.md`. Track artifact readiness in `STATUS.md`.

## Maintenance Notes
-	Update `peers.txt` whenever seeds or sentry nodes change.
-	Keep `genesis.sha256` in sync with the committed `genesis.json`.
-	Record endpoint changes in this README so downstream docs can link here.
