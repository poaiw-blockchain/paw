# paw-testnet-1 Artifact Status

**Last updated:** 2025-12-12T23:03:34Z

## Current Repository Contents
- `genesis.json`: Placeholder generated via `scripts/init-testnet.sh` (empty accounts, not suitable for launch)
- `genesis.sha256`: Hash of the placeholder genesis (replace once canonical file exists)
- `peers.txt`: Placeholder fields awaiting real seed/persistent peer entries

## Action Items Before Public Release
1. Run the validated deploy pipeline (`CHAIN_ID=paw-testnet-1 NODES_SPEC=<name:ip,...> ./scripts/devnet/gcp-deploy.sh`) to bring up the canonical validator set and capture their node IDs / public endpoints.
2. Generate the real paw-testnet-1 genesis (funded accounts, params) using the hardened config pipeline (`config/genesis-template.json` + onboarding scripts).
3. Re-run `scripts/devnet/publish-testnet-artifacts.sh` (with `PAW_HOME` pointing at the canonical node) to sync the real `genesis.json`/`genesis.sha256`/`peers.txt` into this folder.
4. Verify the staged files with `scripts/devnet/verify-network-artifacts.sh paw-testnet-1`.
5. Upload the verified artifacts to the public distribution channel (`https://networks.paw.xyz/paw-testnet-1/` or CDN) and update DNS for RPC/gRPC/REST endpoints.
6. Remove this placeholder note once the real artifacts are in place.

Refer to `docs/guides/deployment/PUBLIC_TESTNET.md` for the full deployment and onboarding runbook.
