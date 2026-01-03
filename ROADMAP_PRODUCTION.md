# Production Roadmap

Last updated: 2026-01-02

## Status

- Testnet infrastructure setup pending.
- Updated Discord invite links across documentation and portal assets.
- Archive node telemetry now honors configurable ports and the service is stable after rebuilding the binary.

## Testnet Infrastructure Tasks

Server: 54.39.103.49 (paw-testnet)
SSH: ssh -i .ssh_testnet_key ubuntu@54.39.103.49

### Core Infrastructure

- [x] Verify node is running and producing blocks
- [x] Configure systemd service for automatic restart
- [x] Set up RPC API endpoint with nginx reverse proxy
- [x] Set up faucet API endpoint
- [x] Set up explorer API endpoint
- [x] Configure SSL certificates with certbot
- [x] Set up DNS A records in Cloudflare pointing to server IP

### Monitoring and Security

- [x] Install and configure Prometheus for metrics collection
- [x] Install and configure Grafana with blockchain dashboard
- [x] Enable anonymous viewing in Grafana for public access
- [x] Install fail2ban for SSH and nginx protection
- [x] Configure nginx rate limiting
- [x] Set up log rotation for node logs
- [x] Set up automated daily snapshots

### High Priority - User Facing

- [x] Deploy block explorer web UI - users need visual interface to browse blocks and transactions
- [x] Deploy faucet web UI - simple HTML form for testnet token requests
- [x] Deploy documentation site - if docs exist in repo
- [x] Create public status page - uptime monitoring visible to community
- [x] Publish genesis.json - downloadable for new node operators
- [x] Publish seed node list - peer discovery for node operators
- [x] Add WebSocket subscriptions - real-time block and transaction notifications

### Medium Priority - Developer Experience

- [x] Deploy OpenAPI/Swagger UI - interactive API documentation
- [x] Publish SDKs to package registries if applicable
- [x] Create chain registry entry - standard metadata file for wallet integrations
- [x] Add GraphQL endpoint - flexible querying for dApp developers
- [x] Network stats page - display TPS, block time, validator count

### Lower Priority - Advanced Features

- [x] Run multiple validator nodes - demonstrate decentralization
- [x] Deploy archive node - full historical state queries
- [x] Set up indexer service - complex queries and event indexing
- [x] Geographic distribution - nodes in multiple regions
- [x] Load balanced RPC - multiple endpoints for reliability

> Archive node telemetry ports were previously hard-coded, but rebuilding `pawd` to honor `telemetry.metrics-port` and the new `telemetry.health-port` (plus the `PAW_TELEMETRY_*` overrides) lets the archive instance bind to `36670`/`36671` as configured. The new binary keeps `pawd-archive` healthy and ready for REST/gRPC/API traffic without conflicts.

> Multi-validator and load-balancer plans are documented in `docs/MULTI_VALIDATOR_TESTNET.md` and `docs/LOAD_BALANCER.md`. Indexer deployment instructions reside in `explorer/indexer` (see `explorer/indexer/README.md` and `explorer/indexer/cmd/main.go`). Geographic redistribution is covered in `INFRASTRUCTURE.md` and `config/GENESIS_LAUNCH_CHECKLIST.md`.

### Testnet Hardening & Professionalization

- [x] Provision dedicated RPC/sentry nodes and firewall validator P2P to sentries only
- [x] Publish snapshot artifacts with checksums + latest height metadata
- [x] Publish `addrbook.json` alongside genesis + seed list
- [x] Add external uptime monitoring and alerting (Grafana alerts)
- [x] Implement host hardening checklist (UFW allowlist, SSH hardening, unattended upgrades)
- [x] Set up encrypted offsite backups for validator keys and critical configs
- [x] Document incident response + upgrade runbook (cosmovisor upgrade steps)

#### Hardening Implementation Notes (2026-01-03)

**Sentry Architecture:**
- Validator (54.39.103.49) P2P firewalled to VPN only (10.10.0.0/24)
- Sentry (139.99.149.160:27656) public-facing with pex=true
- Persistent peers configured over WireGuard VPN (10.10.0.2 ↔ 10.10.0.4)

**Published Artifacts:**
- `https://artifacts.poaiw.org/addrbook.json`
- `https://artifacts.poaiw.org/snapshots/latest.json`

**Security Hardening:**
- UFW: P2P/RPC/REST restricted to VPN only
- SSH: Key-only, no root, MaxAuthTries=3
- fail2ban: 3 attempts, 1hr ban
- Unattended upgrades enabled

**Encrypted Backups:**
- Script: `scripts/backup/validator-backup.sh`
- GPG AES256 encrypted, uploaded to R2
- Daily cron at 3 AM

**Documentation:**
- Incident response runbook: `docs/INCIDENT_RESPONSE_RUNBOOK.md`

**Indexer Service:**
- Go binary deployed on SERVICES server (139.99.149.160:4102)
- PostgreSQL backend, Redis caching
- REST/GraphQL API available

## Details

- All testnet hardening tasks completed 2026-01-03.
