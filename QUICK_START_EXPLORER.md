# PAW Block Explorer - Quick Start Guide

## What is it?

A lightweight, Flask-based web interface for viewing the PAW blockchain data. Shows blocks, transactions, validators, and network status in a clean, responsive UI.

## Quick Start

### Option 1: Docker (Recommended)

```bash
cd /home/hudson/blockchain-projects/paw/docker
docker compose up -d explorer
```

**Access:** http://localhost:11080

### Option 2: Standalone

```bash
cd /home/hudson/blockchain-projects/paw
./scripts/start-explorer.sh
```

**Access:** http://localhost:11080

## Check Status

```bash
./scripts/explorer-status.sh
```

This shows:
- RPC connectivity
- Current block height
- Explorer web server status
- Docker container status
- Validator information
- Quick access URLs

## Features

### Web Interface

- **Homepage** (`/`): Network overview + recent 20 blocks
- **Block Detail** (`/block/<height>`): Full block information with transactions
- **Validators** (`/validators`): Current validator set with voting power
- **Search** (`/search?q=<query>`): Search by block height or tx hash
- **Transaction Detail** (`/tx/<hash>`): View transaction events and results

### REST API

```bash
# Get node status
curl http://localhost:11080/api/status | jq

# Get specific block
curl http://localhost:11080/api/block/1000 | jq

# Get validators
curl http://localhost:11080/api/validators | jq
```

## Architecture

```
Browser → Flask App (port 11080) → RPC (port 26657) → PAW Node
```

The explorer uses Cosmos RPC endpoints directly because REST API (`/cosmos/...`) is currently broken due to an IAVL bug.

## Monitoring Stack Integration

The explorer complements the existing infrastructure:

| Service | Port | Purpose |
|---------|------|---------|
| **Explorer** | 11080 | Human-readable blockchain data |
| **Grafana** | 11030 | Historical metrics & dashboards |
| **Prometheus** | 11090 | Metrics collection |

All three together provide complete observability.

## Requirements

- PAW node running with RPC on port 26657
- Python 3.11+ (for standalone mode)
- Docker (for container mode)

## Troubleshooting

### Explorer shows connection errors

1. Check if RPC is accessible:
   ```bash
   curl http://localhost:26657/status
   ```

2. If node is not running, start it first

### Docker container not starting

```bash
# Check logs
docker logs paw-explorer

# Rebuild and restart
cd docker
docker compose build explorer
docker compose up -d explorer
```

### Port already in use

```bash
# Find what's using port 11080
lsof -i :11080

# Stop the service or use different port
```

## Files & Documentation

- **Full documentation:** `/docs/BLOCK_EXPLORER.md`
- **Script reference:** `/scripts/README.md`
- **Application code:** `/flask-app/app.py`
- **Docker config:** `/docker/docker-compose.yml`

## Development

### Run locally without Docker

```bash
cd flask-app
pip3 install -r requirements.txt
export RPC_URL=http://localhost:26657
export CHAIN_ID=paw-testnet-1
python3 app.py --port 11080
```

### Make changes

1. Edit files in `flask-app/`
2. Rebuild Docker image:
   ```bash
   cd docker
   docker compose build explorer
   docker compose up -d explorer
   ```

Or if running standalone, Flask auto-reloads in debug mode.

## Limitations

1. **No transaction decoding**: Raw data shown, not decoded into human-readable format
2. **No real-time updates**: Requires manual page refresh
3. **Single validator**: Currently only shows local validator (testnet has one validator)
4. **No historical charts**: Use Grafana for time-series visualization

## Future Enhancements

Potential improvements:
- Decode protobuf messages for human-readable transactions
- WebSocket support for real-time updates
- Address pages with transaction history
- IBC channel/packet tracking
- Mempool viewer
- Governance proposal viewer

## Support

If the explorer isn't working:

1. Run status check: `./scripts/explorer-status.sh`
2. Check RPC: `curl http://localhost:26657/status`
3. View logs: `docker logs paw-explorer`
4. Review documentation: `docs/BLOCK_EXPLORER.md`

## Summary

The PAW block explorer provides a simple, effective way to view blockchain data through a web interface. It's lightweight, uses only RPC endpoints, and integrates seamlessly with the existing monitoring infrastructure.

**Start it:** `cd docker && docker compose up -d explorer`

**View it:** http://localhost:11080

**Check it:** `./scripts/explorer-status.sh`
