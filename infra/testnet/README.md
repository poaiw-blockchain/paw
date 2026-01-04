# PAW Testnet Server Checklist

> **Last verified**: 2026-01-03
> **Server**: paw-testnet (54.39.103.49)

This folder captures the changes applied while completing the production roadmap
for the PAW testnet server. Values below reflect the *current* deployment.

---

## Deployed Endpoints (Current)

| Service | URL | Backend | Status |
| --- | --- | --- | --- |
| RPC | https://testnet-rpc.poaiw.org | `127.0.0.1:26657` (LB + services backup) | OK |
| REST API | https://testnet-api.poaiw.org | `127.0.0.1:1317` | **Degraded** (not responding) |
| gRPC | https://testnet-grpc.poaiw.org | `127.0.0.1:9091` | OK |
| WebSocket | https://testnet-ws.poaiw.org | `127.0.0.1:11082` | OK |
| GraphQL | https://testnet-graphql.poaiw.org | `127.0.0.1:11100` | OK |
| GraphQL (via API) | https://testnet-api.poaiw.org/graphql | `127.0.0.1:11100` | OK |
| Faucet UI + API | https://testnet-faucet.poaiw.org | Static UI + `127.0.0.1:8081` | OK |
| Explorer UI + API | https://testnet-explorer.poaiw.org | Static UI + `127.0.0.1:8082` | OK |
| Status Page | https://status.poaiw.org | `127.0.0.1:11090` | OK |
| Snapshots | https://snapshots.poaiw.org | `/home/ubuntu/snapshots` | OK |
| Genesis | https://testnet-explorer.poaiw.org/genesis.json | `/home/ubuntu/.paw/config/genesis.json` | OK |
| Seed List | https://testnet-explorer.poaiw.org/peers.txt | Node ID + P2P endpoint | OK |
| Swagger | https://testnet-api.poaiw.org/swagger/ | REST API docs | **Degraded** (REST down) |
| Stats | https://testnet-explorer.poaiw.org/stats/ | Network stats UI | OK |
| Chain Registry | https://testnet-explorer.poaiw.org/chain-registry/chain.json | Chain metadata | OK |
| SDK | https://testnet-explorer.poaiw.org/sdk/ | Python SDK artifacts | OK |

---

## Server Changes Applied

- Nginx rate limiting snippet: `/etc/nginx/snippets/paw-rate-limit.conf`.
- Nginx rate limit zone config: `/etc/nginx/conf.d/paw-rate-limit.conf`.
- Updated `/etc/nginx/sites-available/poaiw.org` for faucet + explorer UIs.
- New `/etc/nginx/sites-available/status.poaiw.org` for status page.
- Updated `/etc/nginx/sites-available/poaiw.org` to proxy GraphQL on `/graphql`.
- Installed log rotation config at `/etc/logrotate.d/pawd`.
- Enabled Grafana anonymous access (`Viewer` role).
- Added daily snapshot cron job: `/usr/local/bin/paw-backup.sh`.
- Added genesis sync timer: `/etc/systemd/system/paw-genesis-sync.timer`.
- Added peers sync timer: `/etc/systemd/system/paw-peers-sync.timer`.
- Added status page service: `/etc/systemd/system/paw-status.service`.
- Added GraphQL gateway service: `/etc/systemd/system/paw-graphql.service`.
- Enabled Swagger UI in `~/.paw/config/app.toml`.
- Published Python SDK artifacts to `/var/www/paw-explorer/sdk`.
- Published chain registry JSON to `/var/www/paw-explorer/chain-registry`.
- Published stats dashboard to `/var/www/paw-explorer/stats`.

---

## Files in Repo

- Static UIs: `infra/web/faucet`, `infra/web/explorer`.
- Stats UI: `infra/web/stats`.
- Logrotate template: `infra/testnet/logrotate/pawd`.

---

## Operational Commands

```bash
# Status page
sudo systemctl status paw-status

# Genesis sync timer
sudo systemctl status paw-genesis-sync.timer

# Snapshot cron log
sudo tail -n 50 /var/log/paw-backup.log

# GraphQL service
sudo systemctl status paw-graphql
```

---

## Archive Node (work in progress)

The archive node lives under `~/.paw-archive` and is managed by
`pawd-archive.service` (`/etc/systemd/system/pawd-archive.service`). It shares
the same chain but runs on dedicated ports so it can coexist with the validator:

| Component | Binding |
| --- | --- |
| P2P | `0.0.0.0:26686` (advertised as `54.39.103.49:26686`) |
| RPC | `127.0.0.1:26687` |
| gRPC (CometBFT) | `127.0.0.1:26688` |
| gRPC (app) | `localhost:9091` |
| REST API | `tcp://localhost:1327` |
| Intended Prometheus | `127.0.0.1:36670` |
| Intended health | `127.0.0.1:36671` |

Archive RPC is **not** exposed publicly; `testnet-archive.poaiw.org` currently
proxies the primary RPC.

---

## Known Gaps (Current)

- REST API port 1317 is not responding on the primary validator.
- Archive RPC is internal-only (public host proxies primary RPC).

---

## Remaining Advanced Features (lower priority)

These items are still outstanding on the production roadmap and will require
coordination beyond the single validator we already run.

- **Run multiple validator nodes** – A multi-validator testnet is documented at `docs/MULTI_VALIDATOR_TESTNET.md`, which explains how to generate multi-node genesis files and start 2–4 validators via `compose/docker-compose.{2,3,4}nodes.yml`. Implementing this on the production IP would require additional host allocation, WireGuard peers, and cert provisioning before syncing the network as a private cluster.
- **Indexer service** – The explorer/indexer stack (`explorer/indexer`) contains the API, GraphQL, and analytics services needed for richer queries. Deployment would involve provisioning PostgreSQL/Redis, configuring `indexer/config/config.yaml` with the testnet RPC/gRPC endpoints, and running `explorer/indexer/cmd/main.go` (or Docker Compose in `explorer/docker-compose.yml`). Once live, we would update nginx to expose `/stats`, `/graphql`, etc., alongside the existing UIs.
- **Geographic distribution** – The infrastructure philosophy in `INFRASTRUCTURE.md` and `config/GENESIS_LAUNCH_CHECKLIST.md` lists geographic spreading across data centers as a goal. Reaching this stage means provisioning at least one additional region (e.g., EU or APAC), establishing VPN peering to the current WireGuard mesh, and deploying sentry/validator pairs for redundancy.
- **Load balanced RPC** – `docs/LOAD_BALANCER.md` describes how to front RPC backends with health-checked load balancers. To finish this task we need additional RPC instances (possibly using the archive node once stable), a load balancer (NGINX/Cloudflare Workers/Layer4), and DNS + cert configuration so wallets hit multiple endpoints.
