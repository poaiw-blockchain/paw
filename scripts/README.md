# PAW Scripts Directory

Utility scripts for managing the PAW blockchain and block explorer.

## Block Explorer Scripts

### start-explorer.sh

Start the PAW block explorer web interface.

**Usage:**
```bash
# Start in standalone mode (on host)
./scripts/start-explorer.sh

# Start in Docker mode
./scripts/start-explorer.sh docker
```

**What it does:**
- Checks RPC connectivity at localhost:26657
- Installs Python dependencies if needed (standalone mode)
- Starts the Flask application on port 11080
- Provides access URL and status information

**Access:** http://localhost:11080

### explorer-status.sh

Check the status of the PAW block explorer and related services.

**Usage:**
```bash
./scripts/explorer-status.sh
```

**What it checks:**
- RPC endpoint availability (localhost:26657)
- Current block height and chain ID
- Node sync status
- Explorer web server (localhost:11080)
- Docker container status
- Validator set information

**Output includes:**
- Quick access URLs for all services
- Color-coded status indicators
- Current blockchain metrics

## Environment Variables

These scripts respect the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `RPC_URL` | Cosmos RPC endpoint | `http://localhost:26657` |
| `CHAIN_ID` | Blockchain chain ID | `paw-testnet-1` |
| `FLASK_ENV` | Flask environment | `development` |

## Quick Start

```bash
# 1. Ensure PAW node is running
curl http://localhost:26657/status

# 2. Start the explorer
cd /home/hudson/blockchain-projects/paw
./scripts/start-explorer.sh

# 3. Check status
./scripts/explorer-status.sh

# 4. Open in browser
# Navigate to http://localhost:11080
```

## Docker Operations

### Start explorer in Docker
```bash
cd docker
docker compose up -d explorer
```

### Stop explorer
```bash
cd docker
docker compose stop explorer
```

### View logs
```bash
docker logs paw-explorer
docker logs paw-explorer --tail 50 -f
```

### Rebuild after changes
```bash
cd docker
docker compose build explorer
docker compose up -d explorer
```

## Troubleshooting

### Explorer shows "Unable to connect to RPC node"

1. Check if PAW node is running:
   ```bash
   curl http://localhost:26657/status
   ```

2. If node is not running, start it first

3. Verify the RPC_URL environment variable:
   ```bash
   echo $RPC_URL
   ```

### Port 11080 already in use

1. Check what's using the port:
   ```bash
   lsof -i :11080
   ```

2. Stop the conflicting service or use a different port:
   ```bash
   export PORT=11081
   python3 flask-app/app.py --port 11081
   ```

### Docker container keeps restarting

1. Check container logs:
   ```bash
   docker logs paw-explorer
   ```

2. Common issues:
   - RPC not accessible from container
   - Python dependencies missing
   - Port already in use

3. Try host network mode (already configured in docker-compose.yml)

### Pages load but show no data

1. Verify RPC is responding:
   ```bash
   curl http://localhost:26657/status
   curl http://localhost:26657/block
   ```

2. Check node sync status:
   ```bash
   ./scripts/explorer-status.sh
   ```

3. Review Flask logs for errors:
   ```bash
   docker logs paw-explorer | grep ERROR
   ```

## Port Allocation

PAW project uses the 11000-11999 port range:

| Service | Port | Description |
|---------|------|-------------|
| Explorer | 11080 | Block explorer web interface |
| Grafana | 11030 | Metrics dashboard |
| Prometheus | 11090 | Metrics collection |
| RPC | 26657 | Cosmos RPC (default) |
| P2P | 26656 | Tendermint P2P (default) |

## Service Dependencies

```
Block Explorer (11080)
    ↓
    Requires: RPC Endpoint (26657)
        ↓
        Requires: PAW Node Running
```

The explorer will start even if RPC is unavailable, but will show errors until the node is running.

## Development

### Running explorer locally (without Docker)

```bash
cd flask-app

# Install dependencies
pip3 install -r requirements.txt

# Set environment
export RPC_URL=http://localhost:26657
export CHAIN_ID=paw-testnet-1

# Run
python3 app.py --port 11080
```

### Testing changes

1. Modify code in `flask-app/`
2. If using Docker:
   ```bash
   cd docker
   docker compose build explorer
   docker compose up -d explorer
   ```
3. If running standalone, Flask auto-reloads in debug mode

### Adding new pages

1. Create template in `flask-app/templates/`
2. Add route in `flask-app/app.py`
3. Update CSS if needed in `flask-app/static/css/style.css`
4. Restart explorer

## API Endpoints

The explorer provides REST API endpoints:

| Endpoint | Description |
|----------|-------------|
| `GET /api/status` | Node status and sync info |
| `GET /api/block/<height>` | Block data by height |
| `GET /api/validators` | Current validator set |

Example:
```bash
curl http://localhost:11080/api/status | jq
curl http://localhost:11080/api/block/1000 | jq
curl http://localhost:11080/api/validators | jq
```

## Additional Resources

- [Block Explorer Documentation](../docs/BLOCK_EXPLORER.md) - Full feature documentation
- [Docker Compose Config](../docker/docker-compose.yml) - Container configuration
- [Flask Application](../flask-app/app.py) - Main application code
