# PAW Testnet Quick Reference

One-page reference for common multi-validator testnet operations.

## Start 4-Validator Testnet

```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

## Start 4-Validator + 2-Sentry Testnet

```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
curl -s http://localhost:30658/status | jq '.result.sync_info'  # Sentry1
curl -s http://localhost:30668/status | jq '.result.sync_info'  # Sentry2
```

## Stop Testnet

```bash
# Keep data
docker compose -f compose/docker-compose.4nodes.yml down

# Remove all data
docker compose -f compose/docker-compose.4nodes.yml down -v
```

## Check Status

```bash
# Current block height
curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height'

# All validators
curl -s http://localhost:26657/validators | jq '.result.total'

# Container status
docker ps --filter "name=paw-node"
```

## Validator Counts

| Validators | Genesis | Docker Compose |
|------------|---------|----------------|
| 2 | `./scripts/devnet/setup-validators.sh 2` | `compose/docker-compose.2nodes.yml` |
| 3 | `./scripts/devnet/setup-validators.sh 3` | `compose/docker-compose.3nodes.yml` |
| 4 | `./scripts/devnet/setup-validators.sh 4` | `compose/docker-compose.4nodes.yml` |

## Node Access

### Validators

| Node | RPC | gRPC | REST |
|------|-----|------|------|
| node1 | 26657 | 39090 | 1317 |
| node2 | 26667 | 39091 | 1327 |
| node3 | 26677 | 39092 | 1337 |
| node4 | 26687 | 39093 | 1347 |

### Sentries (when using docker-compose.4nodes-with-sentries.yml)

| Node | RPC | P2P | gRPC | REST |
|------|-----|-----|------|------|
| sentry1 | 30658 | 30656 | 39094 | 1357 |
| sentry2 | 30668 | 30666 | 39095 | 1367 |

## Common Commands

```bash
# Watch blocks
watch -n 5 'curl -s http://localhost:26657/status | jq -r ".result.sync_info.latest_block_height"'

# View logs
docker logs paw-node1 -f

# Check errors
docker logs paw-node1 2>&1 | grep -i error | tail -20

# Restart network
docker compose -f compose/docker-compose.4nodes.yml restart

# Clean restart
docker compose -f compose/docker-compose.4nodes.yml down -v
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

## Critical Rules

✅ **ALWAYS** clean before generating new genesis
✅ **ALWAYS** wait 30 seconds after startup before checking
✅ **MATCH** genesis validator count with docker-compose file
✅ **USE** `-v` flag when switching configurations

❌ **NEVER** skip cleaning step
❌ **NEVER** mix validator counts (e.g., 4-validator genesis + 2-node compose)
❌ **NEVER** check status immediately after startup
❌ **NEVER** manually edit genesis after collect-gentxs

## Troubleshooting

**Problem: Genesis hash mismatch**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

**Problem: Stuck at height 0**
```bash
docker compose -f compose/docker-compose.4nodes.yml restart
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

**Problem: Port conflict**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
sudo lsof -i :26657  # Check what's using the port
```

## Full Documentation

- [MULTI_VALIDATOR_TESTNET.md](MULTI_VALIDATOR_TESTNET.md) - Complete validator guide
- [SENTRY_ARCHITECTURE.md](SENTRY_ARCHITECTURE.md) - Production-like testing with sentries
- [TESTNET_DOCUMENTATION_INDEX.md](TESTNET_DOCUMENTATION_INDEX.md) - All documentation
