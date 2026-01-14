# Local Light Client Profile

This profile is optimized for wallet-facing RPC (minimal disk, state sync, aggressive pruning) while keeping trust settings aligned with the published manifest.

## Configuration Targets
- **Pruning**: `custom`, `pruning-keep-recent = 1000`, `pruning-interval = 50`.
- **State sync**: `enable = true`, `rpc_servers = "<rpc>,<rpc>"`, `trust_height = <latest-2000>`, `trust_hash` from the same RPC, `trust_period = "168h"`.
- **Snapshots**: disabled (`snapshot-interval = 0`) because the node is a consumer, not a producer.
- **RPC**: `laddr = "tcp://0.0.0.0:26657"`, `cors_allowed_origins = ["*"]` for wallet origins behind a reverse proxy.

## Bring-Up (Testnet)
```bash
# Bootstrap and start with one line (override RPC if needed)
curl -sL https://raw.githubusercontent.com/paw-chain/paw/main/scripts/onboarding/node-onboard.sh \
  | bash -s -- --mode light --chain-id paw-mvp-1 --rpc <rpc-endpoint-from-docs/TESTNET_QUICK_REFERENCE.md> --start
```
- The script pulls `genesis.json` + `peers.txt` from `networks/paw-mvp-1`, applies pruning/state-sync, and starts `pawd`.
- Swap to `paw-mainnet-1` once mainnet artifacts are published.

## Smoke Harness
Run after the node reports `catching_up: false`:
```bash
./scripts/onboarding/light-client-smoke.sh \
  --rpc http://localhost:26657 \
  --home ~/.paw \
  --expect-pruning custom \
  --expect-state-sync true
```
Checks: `/health`, `/status` (height + catching_up), `/abci_info`, `/net_info`, pruning values, state-sync flags, and a historical block fetch to ensure the light RPC serves wallets.

## Wallet RPC Exposure
- Keep the node bound to localhost and front it with an HTTP reverse proxy (nginx/envoy) that:
  - Enforces rate limits and request body caps.
  - Adds gzip on JSON responses.
  - Restricts origins to approved wallet/web clients when moving beyond local testing.
- For ad-hoc development, SSH forward RPC: `ssh -N -L 26657:localhost:26657 <host>`.

## Maintenance
- Rotate trust parameters weekly: re-run `node-onboard.sh --mode light --rpc <fresh-rpc>` or update `trust_height`/`trust_hash` manually using the latest height - 2000 rule.
- If pruning catches up faster than wallets need, raise `pruning-keep-recent` to `5000`.
- Pair with a nearby full node (LAN or same AZ) to reduce latency on RPC queries and catch-up.
