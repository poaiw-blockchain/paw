# Public Testnet Deployment & Onboarding

This runbook completes Production Roadmap **Phase D** deliverables: launching a public multi-node testnet, publishing artifacts (genesis, peers, endpoints), and documenting validator onboarding. It ties together the existing automation (`scripts/devnet`) with operational guidance.

---

## 0. Local Phase D Rehearsal (Required)

Before touching cloud infrastructure, run the new local rehearsal script to validate the complete workflow (multi-validator network + sentries + artifact packaging) on your workstation:

```bash
cd /home/hudson/blockchain-projects/paw
CHAIN_ID=paw-testnet-1 ./scripts/devnet/local-phase-d-rehearsal.sh
```

This builds `pawd`, regenerates the four-validator genesis (with faucet funding), boots `compose/docker-compose.4nodes-with-sentries.yml`, waits for consensus, executes the full smoke suite, verifies Prometheus metrics on every container, and syncs the resulting `genesis.json`, `genesis.sha256`, and `peers.txt` into `networks/${CHAIN_ID}/`. The script is idempotent and exposes a few useful overrides:

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | `paw-testnet-1` | Chain ID used for genesis + docker stack |
| `REBUILD_GENESIS` | `1` | Set to `0` to skip `setup-multivalidators.sh` (use existing `.state/`) |
| `PAW_REHEARSAL_KEEP_STACK` | `0` | Set to `1` to keep the docker stack running after the script exits |
| `COMPOSE_FILE` | `compose/docker-compose.4nodes-with-sentries.yml` | Alternate topology if desired |

Once the script completes, `networks/${CHAIN_ID}/` contains the canonical artifacts ready for publication, and `scripts/devnet/.state/` stores the validator + faucet key material for later use. Only after this local rehearsal passes should you proceed to the cloud deployment below.

## 1. Launch Canonical Validator Nodes

Use the hardened GCP automation to deploy three validators + supporting accounts:

```bash
export PROJECT_ID="aixn-node-1"            # Override with your GCP project
export ZONE="us-central1-a"
export CHAIN_ID="paw-testnet-1"
export NODES_SPEC="xai-testnode-1:34.29.163.145,xai-testnode-2:108.59.86.86,xai-testnode-3:35.184.167.38"
cd /home/hudson/blockchain-projects/paw
./scripts/devnet/gcp-deploy.sh
```

`scripts/devnet/gcp-deploy.sh` will:
- Build a fresh `pawd` binary and copy it to each node
- Initialize node1, generate validator/faucet/trader keys, and propagate genesis + configs to the other nodes
- Create and start systemd services for every instance
- Sync the resulting `genesis.json`, `genesis.sha256`, and `peers.txt` into `networks/paw-testnet-1/` (set `PUBLISH_ARTIFACTS=0` to skip) and run `scripts/devnet/verify-network-artifacts.sh` automatically

Environment overrides:

| Variable | Purpose | Default |
|----------|---------|---------|
| `PROJECT_ID` | GCP project containing the validator instances | `aixn-node-1` |
| `ZONE` | GCP zone for the nodes | `us-central1-a` |
| `CHAIN_ID` | Chain ID used during init (should be `paw-testnet-1`) | `paw-testnet-1` |
| `NODES_SPEC` | Comma-separated `name:ip` list for the target VMs | script defaults |
| `NETWORK_DIR` | Local directory for artifact sync | `networks/$CHAIN_ID` |
| `PUBLISH_ARTIFACTS` | Set to `0` to skip local artifact sync | `1` |

The script:
1. Builds a fresh `pawd` binary locally
2. Installs dependencies on `xai-testnode-1..3` (edit hostnames in the script if needed)
3. Initializes node1, creates validator + faucet accounts, and captures node IDs
4. Distributes the canonical `genesis.json` and config patches to node2/node3
5. Configures `minimum-gas-prices`, gRPC/REST exposure, and persistent peers pointing at node1

> ⚠️ **Security:** Review the host list, SSH project, and chain ID before running the script. It does not copy private validator keys out of the machines.

**Recommended public endpoints (update DNS after deployment):**

| Role  | Endpoint placeholder | Notes                        |
|-------|----------------------|------------------------------|
| RPC   | `<rpc-endpoint-1>`   | Reverse proxy to node1:26657 |
| RPC   | `<rpc-endpoint-2>`   | Reverse proxy to node2:26657 |
| gRPC  | `<grpc-endpoint>`    | Proxy to node1:9090          |
| REST  | `<api-endpoint>`     | Proxy to node1:1317          |
| Faucet| `<faucet-endpoint>`  | Wraps `scripts/faucet.sh`     |

Expose the endpoints via load balancers/Cloud Armor as appropriate before inviting external validators.

---

## 2. Package Genesis & Peer Metadata

Once node1 is live and funded, export the distribution artifacts:

```bash
PAW_HOME=/root/.paw/node1 \
PAWD_BIN=$(pwd)/build/pawd \
./scripts/devnet/package-testnet-artifacts.sh ./artifacts/public-testnet
# Or create a CDN-ready bundle (genesis, sha, peers, manifest) in ./artifacts/
./scripts/devnet/bundle-testnet-artifacts.sh
# Validate remote CDN copy after upload:
# ./scripts/devnet/validate-remote-artifacts.sh https://networks.paw.xyz/paw-testnet-1
# Optional helper to upload to S3-compatible storage:
# ARTIFACTS_DEST=s3://<bucket>/paw-testnet-1 ./scripts/devnet/upload-artifacts.sh
```

Outputs inside `./artifacts/public-testnet/`:
- `paw-testnet-1-genesis.json`
- `paw-testnet-1-genesis.sha256`
- `paw-testnet-1-peer-metadata.txt` (node ID, listen address, seeds/persistent peers, timestamp)

Copy these files into `networks/paw-testnet-1/` before publishing them to your CDN/bucket so the repository always carries the latest canonical snapshot. You can automate this by running:

```bash
PAW_HOME=/root/.paw/node1 ./scripts/devnet/publish-testnet-artifacts.sh
```

Upload the files to the public artifacts bucket or `networks/` repository and share the SHA256 checksum alongside the download URL.
Validate the repo copy before distribution:

```bash
./scripts/devnet/verify-network-artifacts.sh paw-testnet-1
```

### Seed & Peer List

Populate `paw-testnet-1-peer-metadata.txt` with the canonical set:

```
seeds = "seed1@<seed-host-1>:26656,seed2@<seed-host-2>:26656"
persistent_peers = "node1id@<rpc-endpoint-1>:26656,node2id@<rpc-endpoint-2>:26656"
```

Keep this file updated when adding new sentry nodes so validators can bootstrap quickly.

---

## 3. Publish Public Documentation

Share the following links (update hosts if you use a different domain):

| Artifact | Location |
|----------|----------|
| Genesis JSON | `networks/paw-testnet-1/genesis.json` (host via your CDN) |
| Genesis SHA | `networks/paw-testnet-1/genesis.sha256` |
| Seed/Peer List | `networks/paw-testnet-1/peers.txt` |
| Faucet | `<faucet-endpoint>` |
| Explorer | `<explorer-endpoint>` |
| Docs | `docs/TESTNET_QUICK_REFERENCE.md` |
| Support | https://github.com/paw-chain/paw/issues |

---

## 4. External Validator Onboarding

Operators can follow these steps (also link to `docs/guides/VALIDATOR_QUICKSTART.md`):

1. **Install the binary**
   ```bash
   git clone https://github.com/paw-chain/paw
   cd paw && make build
   ```
2. **Initialize the node**
   ```bash
   pawd init <moniker> --chain-id paw-testnet-1 --home ~/.paw
   ```
3. **Download & verify genesis**
   ```bash
   curl -o ~/.paw/config/genesis.json https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.json
   curl -o /tmp/genesis.sha256 https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.sha256
   (cd ~/.paw/config && sha256sum -c /tmp/genesis.sha256)
   ```
4. **Configure seeds/persistent peers**
   ```bash
   PEERS="node1id@<rpc-endpoint-1>:26656,node2id@<rpc-endpoint-2>:26656"
   SEEDS="seed1@<seed-host-1>:26656,seed2@<seed-host-2>:26656"
   sed -i "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
   sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml
   ```
5. **Set minimum gas price**
   ```bash
   sed -i 's/^minimum-gas-prices =.*/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml
   ```
6. **Start the node**
   ```bash
   pawd start --home ~/.paw
   ```
7. **Request faucet funds** (CLI or faucet UI) and create validator:
   ```bash
   pawd keys add validator --keyring-backend os --home ~/.paw
   pawd tx staking create-validator \
     --amount 250000000upaw \
     --pubkey $(pawd tendermint show-validator --home ~/.paw) \
     --moniker "<moniker>" \
     --chain-id paw-testnet-1 \
     --commission-rate 0.10 \
     --min-self-delegation 1 \
     --fees 5000upaw \
     --from validator \
     --home ~/.paw \
     --yes
   ```

> Encourage validators to set up monitoring (Prometheus/Grafana stack in `infra/monitoring/`) and alert on slashing metrics.

---

## 5. Governance & Status Tracking

- Track public nodes in `docs/NETWORK_PORTS.md`
- Maintain a `SEEDS.md` file or update `config/node-config.toml.template` comments when seeds change
- Announce upgrades via `docs/upgrades/` playbooks; always test proposals on `paw-testnet-1` before mainnet

---

## 6. Operations Checklist

| Task | Frequency | Source |
|------|-----------|--------|
| Snapshot & upload state sync chunks | Daily | `scripts/verify-iavl-data.sh` |
| Verify RPC/REST/gRPC health | Hourly | `scripts/health-check.sh` |
| Rotate logs & backups | Daily | `docs/OBSERVABILITY.md` |
| Update peer file and republish | On new validator additions | This guide |
| Refresh faucet funds | Weekly | `scripts/faucet.sh` |

Completion of the steps above satisfies Phase D requirements for public testnet deployment, artifact publication, and validator onboarding guidance. Update `roadmap_production.md` as milestones are achieved (deployments live, artifacts published, documentation linked).
