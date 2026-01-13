# Lightweight & Full Node Onboarding (Testnet/Mainnet)

Purpose-built instructions for joining PAW as either a lightweight state-sync node or a full archival node. These steps cover artifact retrieval, pruning/state-sync settings, metrics, and faucet readiness with one-line bootstrap commands.

## Quick Start (One-Liners)
- **Full node (testnet)**: `curl -sL https://raw.githubusercontent.com/paw-chain/paw/main/scripts/onboarding/node-onboard.sh | bash -s -- --mode full --chain-id paw-testnet-1 --start`
- **Light node (testnet, wallet RPC)**: `curl -sL https://raw.githubusercontent.com/paw-chain/paw/main/scripts/onboarding/node-onboard.sh | bash -s -- --mode light --chain-id paw-testnet-1 --rpc <rpc-endpoint-from-docs/TESTNET_QUICK_REFERENCE.md> --start`
- Swap `paw-testnet-1` with `paw-mainnet-1` when mainnet artifacts are published.

## Artifact Sources
- Manifest: `networks/paw-testnet-1/paw-testnet-1-manifest.json` (chain_id, checksums, peers, endpoints).
- Genesis: `networks/paw-testnet-1/genesis.json` (verify against `genesis.sha256`).
- Peers: `networks/paw-testnet-1/peers.txt` - **Connect to sentry node** (`ce6afbda0a4443139ad14d2b856cca586161f00d@139.99.149.160:12056`), NOT directly to validators.
- Bundle: `networks/paw-testnet-1/paw-testnet-1-artifacts.tar.gz` (checksum in `paw-testnet-1-artifacts.sha256`).

**Important:** External nodes should connect to the sentry node for DDoS protection. Direct validator connections are for internal use only.

## Manual Install (Full or Light)
1. **Install binary**: `GO111MODULE=on go install github.com/paw-chain/paw/cmd/pawd@main` (or `make build && mv build/pawd ~/go/bin/`).
2. **Init home**: `pawd init <moniker> --chain-id paw-testnet-1 --home ~/.paw`.
3. **Fetch artifacts**:
   ```bash
   BASE=networks/paw-testnet-1
   cp $BASE/paw-testnet-1-manifest.json /tmp/paw-manifest.json
   cp $BASE/genesis.json ~/.paw/config/genesis.json
   sha256sum ~/.paw/config/genesis.json && cat $BASE/genesis.sha256
   cp $BASE/peers.txt ~/.paw/config/peers.txt
   ```
4. **Wire peers**:
   ```bash
   PEERS=$(jq -r '.persistent_peers' /tmp/paw-manifest.json)
   SEEDS=$(jq -r '.seeds' /tmp/paw-manifest.json)
   sed -i "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
   sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml
   ```
5. **Pruning & snapshots**:
   - Full node: `pruning = "default"`, `snapshot-interval = 1000` (keeps recent history + snapshots for downstream state sync).
   - Light node: `pruning = "custom"`, `pruning-keep-recent = "1000"`, `pruning-interval = "50"`, `snapshot-interval = 0`.
6. **State sync (light profile)**:
   ```bash
   RPC="https://<rpc-endpoint-from-docs/TESTNET_QUICK_REFERENCE.md>"
   LATEST=$(curl -fsSL ${RPC}/status | jq -r '.result.sync_info.latest_block_height')
   TRUST_HEIGHT=$((LATEST-2000)); ((TRUST_HEIGHT>1)) || TRUST_HEIGHT=1
   TRUST_HASH=$(curl -fsSL "${RPC}/block?height=$TRUST_HEIGHT" | jq -r '.result.block_id.hash')
   sed -i '/^\[statesync\]/,/^\[/{s/^enable = .*/enable = true/}' ~/.paw/config/config.toml
   sed -i "/^\[statesync\]/,/^\[/{s|^rpc_servers = .*|rpc_servers = \"$RPC,$RPC\"|}" ~/.paw/config/config.toml
   sed -i "/^\[statesync\]/,/^\[/{s/^trust_height = .*/trust_height = $TRUST_HEIGHT/}" ~/.paw/config/config.toml
   sed -i "/^\[statesync\]/,/^\[/{s/^trust_hash = .*/trust_hash = \"$TRUST_HASH\"/}" ~/.paw/config/config.toml
   sed -i "/^\[statesync\]/,/^\[/{s/^trust_period = .*/trust_period = \"168h\"/}" ~/.paw/config/config.toml
   ```
7. **Gas price & APIs**:
   ```bash
   sed -i 's/^minimum-gas-prices =.*/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml
   sed -i '/^\[api\]/,/^\[/{s/^enable = .*/enable = true/}' ~/.paw/config/app.toml
   sed -i '/^\[rpc\]/,/^\[/{s/^laddr = .*/laddr = "tcp:\\/\\/0.0.0.0:26657"/}' ~/.paw/config/config.toml
   ```
8. **Start**: `pawd start --home ~/.paw` (or use the systemd/Compose pack below). Validate with `scripts/onboarding/light-client-smoke.sh --rpc http://localhost:26657`.

## Metrics, Snapshots, Faucet
- Enable Prometheus + telemetry: `./scripts/enable-node-metrics.sh` then restart the node.
- Snapshot serving: set `snapshot-interval = 1000`, `snapshot-keep-recent = 10` (full nodes only) to feed state sync peers.
- Faucet: use RPC endpoints from `docs/TESTNET_QUICK_REFERENCE.md` when running `./scripts/faucet.sh --check <rpc-endpoint>`; CLI requests use `pawd tx bank send ... --fees 5000upaw`.

## Operational Notes
- Connect to the **sentry node** (`ce6afbda0a4443139ad14d2b856cca586161f00d@139.99.149.160:12056`) for external access. Never connect directly to validators.
- Mainnet will mirror the same layout; only swap chain ID and base URLs. Keep trust period at 168h for Tendermint light clients.
- For RPC endpoints serving wallets, ensure `cors_allowed_origins = ["*"]` (defaulted by the bootstrap script) and place a reverse proxy with rate limits in front of `26657`.
