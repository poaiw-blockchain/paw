# Flask Block Explorer - Deployment and Usage Guide

## Overview

The Flask Block Explorer is a lightweight web interface for viewing and exploring the PAW blockchain. It provides an intuitive dashboard for monitoring blocks, transactions, validators, and network status in real-time.

**Access URL**: http://localhost:11080

## Features

### Web Pages

1. **Dashboard** (`/`)
   - Latest block height and timestamp
   - Network status (catching up, node version)
   - Recent 20 blocks with timestamps, proposer, and transaction count
   - Chain ID and sync status

2. **Block Detail** (`/block/<height>`)
   - Complete block information
   - Block hash, parent hash, app hash
   - Proposer address
   - List of transactions with success/failure status
   - Gas usage per transaction
   - Block metadata (validators hash, consensus hash)

3. **Transaction Detail** (`/tx/<hash>`)
   - Transaction hash and block height
   - Success/failure status
   - Gas wanted vs gas used
   - Transaction events and logs
   - Transaction index in block

4. **Validators** (`/validators`)
   - Complete validator set
   - Voting power for each validator
   - Proposer priority
   - Public key information
   - Total validators and total voting power

5. **Search** (`/search`)
   - Search by block height
   - Search by transaction hash
   - Auto-redirect to block or transaction detail

### API Endpoints

All API endpoints return JSON data:

- **GET `/api/status`** - Node status, sync info, latest block
- **GET `/api/block/<height>`** - Block data by height
- **GET `/api/validators`** - Current validator set

## Deployment

### Prerequisites

- Docker and Docker Compose installed
- PAW node running on port 26657 (RPC endpoint)
- Port 11080 available

### Quick Start

```bash
# Deploy the explorer
./scripts/deploy-flask-explorer.sh

# Verify it's working
./scripts/verify-flask-explorer.sh

# Stop the explorer
./scripts/stop-flask-explorer.sh
```

### Manual Deployment

```bash
# Navigate to compose directory
cd compose/

# Start the service
docker compose -f docker-compose.flask-explorer.yml up -d --build

# Check logs
docker logs -f paw-flask-explorer

# Stop the service
docker compose -f docker-compose.flask-explorer.yml down
```

## Configuration

### Environment Variables

The Flask explorer can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RPC_URL` | `http://host.docker.internal:26657` | RPC endpoint of PAW node |
| `CHAIN_ID` | `paw-testnet-1` | Chain ID to display |
| `FLASK_ENV` | `production` | Flask environment |

### Custom Configuration

Edit `compose/docker-compose.flask-explorer.yml`:

```yaml
environment:
  - RPC_URL=http://your-node:26657
  - CHAIN_ID=your-chain-id
```

Or set environment variables before deployment:

```bash
export RPC_URL=http://192.168.1.100:26657
export CHAIN_ID=paw-mainnet-1
./scripts/deploy-flask-explorer.sh
```

## Production Deployment

### Using Nginx Reverse Proxy

For production deployment with SSL and better performance, use nginx as a reverse proxy.

1. **Copy nginx configuration**:
   ```bash
   sudo cp flask-app/nginx.conf /etc/nginx/sites-available/paw-explorer
   ```

2. **Edit configuration** to set your domain:
   ```nginx
   server_name explorer.yourdomain.com;
   ```

3. **Enable site**:
   ```bash
   sudo ln -s /etc/nginx/sites-available/paw-explorer /etc/nginx/sites-enabled/
   sudo nginx -t
   sudo systemctl reload nginx
   ```

4. **Configure SSL** (using Let's Encrypt):
   ```bash
   sudo certbot --nginx -d explorer.yourdomain.com
   ```

### Using Gunicorn (Standalone)

The Docker deployment already uses gunicorn with optimal settings:
- 2 worker processes
- 120-second timeout
- Bound to 0.0.0.0:11080

To run standalone:

```bash
cd flask-app/
pip install -r requirements.txt
export RPC_URL=http://localhost:26657
gunicorn --bind 0.0.0.0:11080 --workers 2 --timeout 120 app:app
```

## Health Checks

The Docker container includes health checks that verify the explorer is responding:

```bash
# Check container health
docker ps | grep paw-flask-explorer

# Manual health check
curl http://localhost:11080/
curl http://localhost:11080/api/status
```

## Monitoring and Logs

### View Logs

```bash
# Follow logs in real-time
docker logs -f paw-flask-explorer

# Last 100 lines
docker logs --tail 100 paw-flask-explorer

# Logs since 1 hour ago
docker logs --since 1h paw-flask-explorer
```

### Log Levels

The Flask explorer logs:
- RPC connection errors
- Failed requests
- Application errors

Gunicorn logs:
- HTTP requests (access logs)
- Worker status
- Startup/shutdown events

## Troubleshooting

### Explorer Not Loading

1. **Check if container is running**:
   ```bash
   docker ps | grep paw-flask-explorer
   ```

2. **Check container logs**:
   ```bash
   docker logs paw-flask-explorer
   ```

3. **Verify RPC node is accessible**:
   ```bash
   curl http://localhost:26657/status
   ```

### "Unable to Connect to RPC Node" Error

This means the Flask app cannot reach the PAW RPC endpoint.

**Solutions**:
1. Ensure PAW node is running: `curl http://localhost:26657/status`
2. Check RPC_URL environment variable in docker-compose file
3. Verify Docker networking (use `host.docker.internal` for host network)
4. Check firewall rules

### Empty or Missing Data

If pages load but show no data:

1. **Verify RPC endpoint returns data**:
   ```bash
   curl http://localhost:26657/status
   curl http://localhost:26657/block?height=1
   ```

2. **Check RPC URL configuration** in container:
   ```bash
   docker exec paw-flask-explorer env | grep RPC_URL
   ```

### Slow Performance

1. **Increase gunicorn workers** in Dockerfile:
   ```dockerfile
   CMD ["gunicorn", "--bind", "0.0.0.0:11080", "--workers", "4", "--timeout", "120", "app:app"]
   ```

2. **Use nginx caching** for static content (see nginx.conf)

3. **Optimize RPC node** - ensure it's not rate-limiting requests

## Development

### Local Development (Non-Docker)

```bash
cd flask-app/

# Install dependencies
pip install -r requirements.txt

# Set environment variables
export RPC_URL=http://localhost:26657
export CHAIN_ID=paw-testnet-1

# Run development server
python app.py --port 11080

# Or with Flask CLI
export FLASK_APP=app.py
flask run --host=0.0.0.0 --port=11080
```

### Modifying Templates

Templates are in `flask-app/templates/`:
- `base.html` - Base layout with navigation
- `index.html` - Dashboard
- `block.html` - Block detail
- `transaction.html` - Transaction detail
- `validators.html` - Validator list
- `search.html` - Search results
- `error.html` - Error page

CSS is in `flask-app/static/css/style.css`.

After modifying, rebuild the Docker image:

```bash
docker compose -f compose/docker-compose.flask-explorer.yml up -d --build
```

### Adding New Features

1. **Add route to** `flask-app/app.py`
2. **Create template** in `flask-app/templates/`
3. **Add RPC client method** if needed (in `RPCClient` class)
4. **Test locally** with `python app.py`
5. **Rebuild and deploy** Docker container

## API Usage Examples

### Get Latest Block Height

```bash
curl -s http://localhost:11080/api/status | jq '.sync_info.latest_block_height'
```

### Get Block Data

```bash
curl -s http://localhost:11080/api/block/1 | jq '.block.header'
```

### Get Validators

```bash
curl -s http://localhost:11080/api/validators | jq '.validators[] | {address, voting_power}'
```

### Monitor Sync Status

```bash
# Check if node is catching up
curl -s http://localhost:11080/api/status | jq '.sync_info.catching_up'

# Get latest block time
curl -s http://localhost:11080/api/status | jq -r '.sync_info.latest_block_time'
```

## Integration with Other Services

### Prometheus Metrics

To add Prometheus metrics support, install `prometheus-flask-exporter`:

```python
from prometheus_flask_exporter import PrometheusMetrics

app = Flask(__name__)
metrics = PrometheusMetrics(app)
```

Then scrape metrics at `http://localhost:11080/metrics`.

### GraphQL API

To add GraphQL support, install `flask-graphql` and define schema.

### WebSocket Updates

To add real-time block updates, integrate with Tendermint WebSocket:

```python
# Subscribe to new blocks
ws://localhost:26657/websocket
{"jsonrpc":"2.0","method":"subscribe","params":{"query":"tm.event='NewBlock'"},"id":1}
```

## Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────┐
│   Nginx     │ (Optional reverse proxy)
│  Port 80    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Gunicorn   │
│   Flask     │
│  Port 11080 │
└──────┬──────┘
       │ RPC (HTTP)
       ▼
┌─────────────┐
│  PAW Node   │
│  Port 26657 │
└─────────────┘
```

## Security Considerations

1. **Input Validation**: All user inputs (block heights, tx hashes) are validated
2. **XSS Protection**: Templates use Jinja2 auto-escaping
3. **CORS**: Not enabled by default (add `flask-cors` if needed)
4. **Rate Limiting**: Consider adding rate limiting for production (use `flask-limiter`)
5. **SSL**: Always use HTTPS in production (see nginx config)

### Adding Rate Limiting

```python
from flask_limiter import Limiter

limiter = Limiter(app, key_func=get_remote_address)

@app.route('/api/status')
@limiter.limit("60 per minute")
def api_status():
    # ...
```

## Performance Tuning

### Caching

Add Redis caching for frequently accessed data:

```python
from flask_caching import Cache

cache = Cache(app, config={'CACHE_TYPE': 'redis', 'CACHE_REDIS_URL': 'redis://localhost:6379'})

@app.route('/api/status')
@cache.cached(timeout=5)
def api_status():
    # ...
```

### Database

For historical data and faster queries, consider running the full indexer:
- See `explorer/indexer/` for the Golang-based indexer
- Postgres database with indexed blockchain data
- GraphQL API for complex queries

## Comparison with Full Explorer

| Feature | Flask Explorer | Full Explorer (indexer) |
|---------|----------------|-------------------------|
| Deployment | Single container | Multi-container (DB, API, frontend) |
| Storage | None (queries RPC) | PostgreSQL database |
| Performance | Direct RPC calls | Indexed database queries |
| Historical Data | Limited to RPC history | Full blockchain history |
| Setup Complexity | Minimal | Moderate |
| Resource Usage | Low | Medium-High |
| Best For | Development, testnets | Production, mainnet |

## Support

For issues or questions:
1. Check logs: `docker logs paw-flask-explorer`
2. Verify RPC connectivity: `curl http://localhost:26657/status`
3. Review this guide's Troubleshooting section
4. Check PAW documentation in `docs/`

## License

Same as PAW blockchain project.
