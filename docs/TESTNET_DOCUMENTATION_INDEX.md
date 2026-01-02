# PAW Testnet Documentation Index

Use this index to navigate public testnet documentation.

## Primary references
- [TESTNET_QUICK_REFERENCE.md](TESTNET_QUICK_REFERENCE.md) — one-page commands, health checks, troubleshooting
- [MULTI_VALIDATOR_TESTNET.md](MULTI_VALIDATOR_TESTNET.md) — full setup for 2–4 validators
- [TESTNET_DEPLOYMENT_GUIDE.md](TESTNET_DEPLOYMENT_GUIDE.md) — step-by-step runbook with smoke tests
- [../VALIDATOR_QUICK_START.md](../VALIDATOR_QUICK_START.md) — validator bring-up checklist
- [../scripts/devnet/README.md](../scripts/devnet/README.md) — genesis generation and helper scripts
- [SENTRY_ARCHITECTURE.md](SENTRY_ARCHITECTURE.md) and [SENTRY_TESTING_GUIDE.md](SENTRY_TESTING_GUIDE.md) — production-style sentry topology and validation tests
- [DASHBOARDS_GUIDE.md](DASHBOARDS_GUIDE.md) — operational dashboards
- Network artifacts: `../networks/paw-testnet-1/` (`genesis.json`, `genesis.sha256`, `peers.txt`)

## Common scenarios

**Run a 4-validator localnet**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

**Bring up the sentry topology**
```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
curl -s http://localhost:30658/status | jq '.result.sync_info'  # Sentry1
curl -s http://localhost:30668/status | jq '.result.sync_info'  # Sentry2
```

**Troubleshoot quickly**
- Check logs: `docker logs paw-node1 2>&1 | grep -i error`
- Verify peers configured in `scripts/devnet/.state/node*/config/config.toml`
- Confirm height is moving: `curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'`
